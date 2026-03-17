package services

import (
	"database/sql"
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync"
	"time"

	"balance-monitor/internal/database"
)

const DefaultConsumptionBackfillDaysLimit = 365

const (
	ConsumptionBackfillJobStatusPending             = "pending"
	ConsumptionBackfillJobStatusRunning             = "running"
	ConsumptionBackfillJobStatusCancelling          = "cancelling"
	ConsumptionBackfillJobStatusCancelled           = "cancelled"
	ConsumptionBackfillJobStatusCompleted           = "completed"
	ConsumptionBackfillJobStatusCompletedWithErrors = "completed_with_errors"
	ConsumptionBackfillJobStatusInterrupted         = "interrupted"
)

const (
	ConsumptionBackfillProviderStatusPending     = "pending"
	ConsumptionBackfillProviderStatusRunning     = "running"
	ConsumptionBackfillProviderStatusCompleted   = "completed"
	ConsumptionBackfillProviderStatusFailed      = "failed"
	ConsumptionBackfillProviderStatusCancelled   = "cancelled"
	ConsumptionBackfillProviderStatusInterrupted = "interrupted"
)

var consumptionBackfillJobMu sync.Mutex

type ConsumptionBackfillJobProviderView struct {
	ProviderID   int     `json:"provider_id"`
	ProviderName string  `json:"provider_name"`
	Status       string  `json:"status"`
	CurrentDate  *string `json:"current_date"`
	InsertedDays int     `json:"inserted_days"`
	SkippedDays  int     `json:"skipped_days"`
	ErrorMessage *string `json:"error_message"`
	StartedAt    *string `json:"started_at"`
	FinishedAt   *string `json:"finished_at"`
}

type ConsumptionBackfillJobDetails struct {
	JobID                int                                  `json:"job_id"`
	Status               string                               `json:"status"`
	DaysLimit            int                                  `json:"days_limit"`
	CreatedBy            int                                  `json:"created_by"`
	CreatedAt            string                               `json:"created_at"`
	StartedAt            *string                              `json:"started_at"`
	FinishedAt           *string                              `json:"finished_at"`
	TotalProviders       int                                  `json:"total_providers"`
	PendingProviders     int                                  `json:"pending_providers"`
	RunningProviders     int                                  `json:"running_providers"`
	CompletedProviders   int                                  `json:"completed_providers"`
	FailedProviders      int                                  `json:"failed_providers"`
	CancelledProviders   int                                  `json:"cancelled_providers"`
	InterruptedProviders int                                  `json:"interrupted_providers"`
	InsertedDays         int                                  `json:"inserted_days"`
	SkippedDays          int                                  `json:"skipped_days"`
	Providers            []ConsumptionBackfillJobProviderView `json:"providers"`
}

func MarkInterruptedConsumptionBackfillJobs(db *sql.DB) error {
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("开始中断修复事务失败: %w", err)
	}
	defer tx.Rollback()

	_, err = tx.Exec(`
		UPDATE consumption_backfill_jobs
		SET status = ?, finished_at = CURRENT_TIMESTAMP
		WHERE status IN (?, ?, ?)
	`,
		ConsumptionBackfillJobStatusInterrupted,
		ConsumptionBackfillJobStatusPending,
		ConsumptionBackfillJobStatusRunning,
		ConsumptionBackfillJobStatusCancelling,
	)
	if err != nil {
		return fmt.Errorf("更新任务状态失败: %w", err)
	}

	_, err = tx.Exec(`
		UPDATE consumption_backfill_job_providers
		SET status = ?, finished_at = CURRENT_TIMESTAMP
		WHERE job_id IN (
			SELECT id FROM consumption_backfill_jobs WHERE status = ?
		)
		AND status IN (?, ?)
	`,
		ConsumptionBackfillProviderStatusInterrupted,
		ConsumptionBackfillJobStatusInterrupted,
		ConsumptionBackfillProviderStatusPending,
		ConsumptionBackfillProviderStatusRunning,
	)
	if err != nil {
		return fmt.Errorf("更新供应商任务状态失败: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("提交中断修复事务失败: %w", err)
	}

	return nil
}

func CreateConsumptionBackfillJob(db *sql.DB, createdBy int, providerIDs []int, daysLimit int) (*ConsumptionBackfillJobDetails, bool, error) {
	if daysLimit <= 0 {
		daysLimit = DefaultConsumptionBackfillDaysLimit
	}

	normalizedProviderIDs := normalizeProviderIDs(providerIDs)
	if len(normalizedProviderIDs) == 0 {
		return nil, false, fmt.Errorf("至少选择一个供应商")
	}

	consumptionBackfillJobMu.Lock()
	defer consumptionBackfillJobMu.Unlock()

	activeJob, err := GetActiveConsumptionBackfillJob(db)
	if err != nil {
		return nil, false, err
	}
	if activeJob != nil {
		return activeJob, false, nil
	}

	providers, err := getBackfillProviders(db, normalizedProviderIDs)
	if err != nil {
		return nil, false, err
	}

	tx, err := db.Begin()
	if err != nil {
		return nil, false, fmt.Errorf("开始创建任务事务失败: %w", err)
	}
	defer tx.Rollback()

	result, err := tx.Exec(`
		INSERT INTO consumption_backfill_jobs (status, days_limit, created_by, created_at)
		VALUES (?, ?, ?, CURRENT_TIMESTAMP)
	`, ConsumptionBackfillJobStatusPending, daysLimit, createdBy)
	if err != nil {
		return nil, false, fmt.Errorf("创建回补任务失败: %w", err)
	}

	jobID64, err := result.LastInsertId()
	if err != nil {
		return nil, false, fmt.Errorf("读取任务 ID 失败: %w", err)
	}
	jobID := int(jobID64)

	stmt, err := tx.Prepare(`
		INSERT INTO consumption_backfill_job_providers
			(job_id, provider_id, status, inserted_days, skipped_days)
		VALUES (?, ?, ?, 0, 0)
	`)
	if err != nil {
		return nil, false, fmt.Errorf("准备供应商任务写入失败: %w", err)
	}
	defer stmt.Close()

	for _, provider := range providers {
		if _, err := stmt.Exec(jobID, provider.ID, ConsumptionBackfillProviderStatusPending); err != nil {
			return nil, false, fmt.Errorf("创建供应商回补任务失败: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, false, fmt.Errorf("提交回补任务失败: %w", err)
	}

	job, err := GetConsumptionBackfillJob(db, jobID)
	if err != nil {
		return nil, false, err
	}

	go runConsumptionBackfillJob(db, jobID, providers, daysLimit)

	return job, true, nil
}

func GetActiveConsumptionBackfillJob(db *sql.DB) (*ConsumptionBackfillJobDetails, error) {
	var jobID int
	err := db.QueryRow(`
		SELECT id
		FROM consumption_backfill_jobs
		WHERE status IN (?, ?, ?)
		ORDER BY created_at DESC, id DESC
		LIMIT 1
	`,
		ConsumptionBackfillJobStatusPending,
		ConsumptionBackfillJobStatusRunning,
		ConsumptionBackfillJobStatusCancelling,
	).Scan(&jobID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("查询活跃回补任务失败: %w", err)
	}

	return GetConsumptionBackfillJob(db, jobID)
}

func GetConsumptionBackfillJob(db *sql.DB, jobID int) (*ConsumptionBackfillJobDetails, error) {
	var (
		job        ConsumptionBackfillJobDetails
		startedAt  sql.NullTime
		finishedAt sql.NullTime
		createdAt  time.Time
	)

	err := db.QueryRow(`
		SELECT id, status, days_limit, created_by, created_at, started_at, finished_at
		FROM consumption_backfill_jobs
		WHERE id = ?
	`, jobID).Scan(&job.JobID, &job.Status, &job.DaysLimit, &job.CreatedBy, &createdAt, &startedAt, &finishedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("查询回补任务失败: %w", err)
	}

	job.CreatedAt = createdAt.Format(time.RFC3339)
	job.StartedAt = nullableTimeToStringPtr(startedAt)
	job.FinishedAt = nullableTimeToStringPtr(finishedAt)

	rows, err := db.Query(`
		SELECT jp.provider_id, p.name, jp.status, jp.current_date, jp.inserted_days, jp.skipped_days,
			jp.error_message, jp.started_at, jp.finished_at
		FROM consumption_backfill_job_providers jp
		JOIN providers p ON jp.provider_id = p.id
		WHERE jp.job_id = ?
		ORDER BY p.name ASC, jp.provider_id ASC
	`, jobID)
	if err != nil {
		return nil, fmt.Errorf("查询回补任务供应商进度失败: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var (
			provider           ConsumptionBackfillJobProviderView
			currentDate        sql.NullString
			errorMessage       sql.NullString
			providerStartedAt  sql.NullTime
			providerFinishedAt sql.NullTime
		)

		if err := rows.Scan(
			&provider.ProviderID,
			&provider.ProviderName,
			&provider.Status,
			&currentDate,
			&provider.InsertedDays,
			&provider.SkippedDays,
			&errorMessage,
			&providerStartedAt,
			&providerFinishedAt,
		); err != nil {
			return nil, fmt.Errorf("读取供应商任务进度失败: %w", err)
		}

		provider.CurrentDate = nullableStringToPtr(currentDate)
		provider.ErrorMessage = nullableStringToPtr(errorMessage)
		provider.StartedAt = nullableTimeToStringPtr(providerStartedAt)
		provider.FinishedAt = nullableTimeToStringPtr(providerFinishedAt)

		job.TotalProviders++
		job.InsertedDays += provider.InsertedDays
		job.SkippedDays += provider.SkippedDays

		switch provider.Status {
		case ConsumptionBackfillProviderStatusPending:
			job.PendingProviders++
		case ConsumptionBackfillProviderStatusRunning:
			job.RunningProviders++
		case ConsumptionBackfillProviderStatusCompleted:
			job.CompletedProviders++
		case ConsumptionBackfillProviderStatusFailed:
			job.FailedProviders++
		case ConsumptionBackfillProviderStatusCancelled:
			job.CancelledProviders++
		case ConsumptionBackfillProviderStatusInterrupted:
			job.InterruptedProviders++
		}

		job.Providers = append(job.Providers, provider)
	}

	return &job, nil
}

func RequestCancelConsumptionBackfillJob(db *sql.DB, jobID int) error {
	var status string
	err := db.QueryRow(`SELECT status FROM consumption_backfill_jobs WHERE id = ?`, jobID).Scan(&status)
	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("回补任务不存在")
		}
		return fmt.Errorf("查询回补任务失败: %w", err)
	}

	switch status {
	case ConsumptionBackfillJobStatusPending, ConsumptionBackfillJobStatusRunning:
		_, err = db.Exec(`
			UPDATE consumption_backfill_jobs
			SET status = ?
			WHERE id = ? AND status IN (?, ?)
		`,
			ConsumptionBackfillJobStatusCancelling,
			jobID,
			ConsumptionBackfillJobStatusPending,
			ConsumptionBackfillJobStatusRunning,
		)
		if err != nil {
			return fmt.Errorf("更新回补任务取消状态失败: %w", err)
		}
		return nil
	case ConsumptionBackfillJobStatusCancelling:
		return nil
	default:
		return fmt.Errorf("任务已结束，无法取消")
	}
}

func runConsumptionBackfillJob(db *sql.DB, jobID int, providers []database.Provider, daysLimit int) {
	if err := markConsumptionBackfillJobStarted(db, jobID); err != nil {
		log.Printf("标记回补任务 %d 启动失败: %v", jobID, err)
		return
	}

	var waitGroup sync.WaitGroup
	for _, provider := range providers {
		provider := provider
		waitGroup.Add(1)
		go func() {
			defer waitGroup.Done()
			runConsumptionBackfillProvider(db, jobID, provider, daysLimit)
		}()
	}

	waitGroup.Wait()

	if err := finalizeConsumptionBackfillJob(db, jobID); err != nil {
		log.Printf("回补任务 %d 收尾失败: %v", jobID, err)
	}
}

func runConsumptionBackfillProvider(db *sql.DB, jobID int, provider database.Provider, daysLimit int) {
	defer func() {
		if recovered := recover(); recovered != nil {
			_ = markConsumptionBackfillProviderFailed(db, jobID, provider.ID, "", fmt.Sprintf("panic: %v", recovered))
		}
	}()

	if err := markConsumptionBackfillProviderRunning(db, jobID, provider.ID); err != nil {
		log.Printf("标记供应商回补任务启动失败: job=%d provider=%d err=%v", jobID, provider.ID, err)
		return
	}

	today := time.Now().In(shanghaiLocation)
	today = time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, shanghaiLocation)

	for offset := 0; offset < daysLimit; offset++ {
		if cancelling, err := isConsumptionBackfillJobCancelling(db, jobID); err != nil {
			_ = markConsumptionBackfillProviderFailed(db, jobID, provider.ID, "", err.Error())
			return
		} else if cancelling {
			_ = markConsumptionBackfillProviderCancelled(db, jobID, provider.ID)
			return
		}

		currentDate := today.AddDate(0, 0, -offset).Format("2006-01-02")
		if err := updateConsumptionBackfillProviderCurrentDate(db, jobID, provider.ID, currentDate); err != nil {
			_ = markConsumptionBackfillProviderFailed(db, jobID, provider.ID, currentDate, err.Error())
			return
		}

		result, err := backfillProviderDailyConsumptionHistory(db, &provider, currentDate)
		if err != nil {
			_ = markConsumptionBackfillProviderFailed(db, jobID, provider.ID, currentDate, err.Error())
			return
		}

		insertedDays := 0
		skippedDays := 0
		if result.Status == "inserted" {
			insertedDays = 1
		} else {
			skippedDays = 1
		}

		if err := updateConsumptionBackfillProviderProgress(db, jobID, provider.ID, currentDate, insertedDays, skippedDays); err != nil {
			_ = markConsumptionBackfillProviderFailed(db, jobID, provider.ID, currentDate, err.Error())
			return
		}
	}

	if err := markConsumptionBackfillProviderCompleted(db, jobID, provider.ID); err != nil {
		log.Printf("标记供应商回补完成失败: job=%d provider=%d err=%v", jobID, provider.ID, err)
	}
}

func finalizeConsumptionBackfillJob(db *sql.DB, jobID int) error {
	rows, err := db.Query(`
		SELECT status
		FROM consumption_backfill_job_providers
		WHERE job_id = ?
	`, jobID)
	if err != nil {
		return fmt.Errorf("查询回补任务供应商状态失败: %w", err)
	}
	defer rows.Close()

	finalStatus := ConsumptionBackfillJobStatusCompleted
	hasFailed := false
	hasCancelled := false

	for rows.Next() {
		var status string
		if err := rows.Scan(&status); err != nil {
			return fmt.Errorf("读取回补任务供应商状态失败: %w", err)
		}

		switch status {
		case ConsumptionBackfillProviderStatusFailed:
			hasFailed = true
		case ConsumptionBackfillProviderStatusCancelled:
			hasCancelled = true
		case ConsumptionBackfillProviderStatusInterrupted:
			finalStatus = ConsumptionBackfillJobStatusInterrupted
		}
	}

	if finalStatus != ConsumptionBackfillJobStatusInterrupted {
		switch {
		case hasFailed:
			finalStatus = ConsumptionBackfillJobStatusCompletedWithErrors
		case hasCancelled:
			finalStatus = ConsumptionBackfillJobStatusCancelled
		default:
			finalStatus = ConsumptionBackfillJobStatusCompleted
		}
	}

	_, err = db.Exec(`
		UPDATE consumption_backfill_jobs
		SET status = ?, finished_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, finalStatus, jobID)
	if err != nil {
		return fmt.Errorf("更新回补任务最终状态失败: %w", err)
	}

	return nil
}

func getBackfillProviders(db *sql.DB, providerIDs []int) ([]database.Provider, error) {
	providers := make([]database.Provider, 0, len(providerIDs))
	for _, providerID := range providerIDs {
		provider, err := getBackfillProvider(db, providerID)
		if err != nil {
			return nil, err
		}
		providers = append(providers, *provider)
	}

	return providers, nil
}

func markConsumptionBackfillJobStarted(db *sql.DB, jobID int) error {
	_, err := db.Exec(`
		UPDATE consumption_backfill_jobs
		SET started_at = COALESCE(started_at, CURRENT_TIMESTAMP),
			status = CASE
				WHEN status = ? THEN ?
				ELSE status
			END
		WHERE id = ? AND status IN (?, ?)
	`,
		ConsumptionBackfillJobStatusPending,
		ConsumptionBackfillJobStatusRunning,
		jobID,
		ConsumptionBackfillJobStatusPending,
		ConsumptionBackfillJobStatusCancelling,
	)
	if err != nil {
		return fmt.Errorf("更新任务启动状态失败: %w", err)
	}

	return nil
}

func markConsumptionBackfillProviderRunning(db *sql.DB, jobID int, providerID int) error {
	_, err := db.Exec(`
		UPDATE consumption_backfill_job_providers
		SET status = ?, started_at = COALESCE(started_at, CURRENT_TIMESTAMP)
		WHERE job_id = ? AND provider_id = ? AND status = ?
	`,
		ConsumptionBackfillProviderStatusRunning,
		jobID,
		providerID,
		ConsumptionBackfillProviderStatusPending,
	)
	if err != nil {
		return fmt.Errorf("更新供应商任务启动状态失败: %w", err)
	}

	return nil
}

func updateConsumptionBackfillProviderCurrentDate(db *sql.DB, jobID int, providerID int, currentDate string) error {
	_, err := db.Exec(`
		UPDATE consumption_backfill_job_providers
		SET current_date = ?
		WHERE job_id = ? AND provider_id = ?
	`, currentDate, jobID, providerID)
	if err != nil {
		return fmt.Errorf("更新供应商任务当前日期失败: %w", err)
	}

	return nil
}

func updateConsumptionBackfillProviderProgress(db *sql.DB, jobID int, providerID int, currentDate string, insertedDays int, skippedDays int) error {
	_, err := db.Exec(`
		UPDATE consumption_backfill_job_providers
		SET current_date = ?, inserted_days = inserted_days + ?, skipped_days = skipped_days + ?
		WHERE job_id = ? AND provider_id = ?
	`, currentDate, insertedDays, skippedDays, jobID, providerID)
	if err != nil {
		return fmt.Errorf("更新供应商任务进度失败: %w", err)
	}

	return nil
}

func markConsumptionBackfillProviderCompleted(db *sql.DB, jobID int, providerID int) error {
	_, err := db.Exec(`
		UPDATE consumption_backfill_job_providers
		SET status = ?, finished_at = CURRENT_TIMESTAMP
		WHERE job_id = ? AND provider_id = ?
	`,
		ConsumptionBackfillProviderStatusCompleted,
		jobID,
		providerID,
	)
	if err != nil {
		return fmt.Errorf("更新供应商任务完成状态失败: %w", err)
	}

	return nil
}

func markConsumptionBackfillProviderCancelled(db *sql.DB, jobID int, providerID int) error {
	_, err := db.Exec(`
		UPDATE consumption_backfill_job_providers
		SET status = ?, finished_at = CURRENT_TIMESTAMP
		WHERE job_id = ? AND provider_id = ? AND status IN (?, ?)
	`,
		ConsumptionBackfillProviderStatusCancelled,
		jobID,
		providerID,
		ConsumptionBackfillProviderStatusPending,
		ConsumptionBackfillProviderStatusRunning,
	)
	if err != nil {
		return fmt.Errorf("更新供应商任务取消状态失败: %w", err)
	}

	return nil
}

func markConsumptionBackfillProviderFailed(db *sql.DB, jobID int, providerID int, currentDate, errorMessage string) error {
	if strings.TrimSpace(errorMessage) == "" {
		errorMessage = "未知错误"
	}

	_, err := db.Exec(`
		UPDATE consumption_backfill_job_providers
		SET status = ?, current_date = ?, error_message = ?, finished_at = CURRENT_TIMESTAMP
		WHERE job_id = ? AND provider_id = ?
	`,
		ConsumptionBackfillProviderStatusFailed,
		nullToEmptyString(currentDate),
		errorMessage,
		jobID,
		providerID,
	)
	if err != nil {
		return fmt.Errorf("更新供应商任务失败状态失败: %w", err)
	}

	return nil
}

func isConsumptionBackfillJobCancelling(db *sql.DB, jobID int) (bool, error) {
	var status string
	err := db.QueryRow(`SELECT status FROM consumption_backfill_jobs WHERE id = ?`, jobID).Scan(&status)
	if err != nil {
		return false, fmt.Errorf("查询任务状态失败: %w", err)
	}

	return status == ConsumptionBackfillJobStatusCancelling, nil
}

func normalizeProviderIDs(providerIDs []int) []int {
	seen := make(map[int]struct{}, len(providerIDs))
	normalized := make([]int, 0, len(providerIDs))
	for _, providerID := range providerIDs {
		if providerID <= 0 {
			continue
		}
		if _, exists := seen[providerID]; exists {
			continue
		}
		seen[providerID] = struct{}{}
		normalized = append(normalized, providerID)
	}

	return normalized
}

func nullableTimeToStringPtr(value sql.NullTime) *string {
	if !value.Valid {
		return nil
	}

	formatted := value.Time.Format(time.RFC3339)
	return &formatted
}

func nullableStringToPtr(value sql.NullString) *string {
	if !value.Valid {
		return nil
	}

	trimmed := strings.TrimSpace(value.String)
	if trimmed == "" {
		return nil
	}

	return &trimmed
}

func nullToEmptyString(value string) string {
	return strings.TrimSpace(value)
}

func IsActiveConsumptionBackfillJobStatus(status string) bool {
	switch status {
	case ConsumptionBackfillJobStatusPending, ConsumptionBackfillJobStatusRunning, ConsumptionBackfillJobStatusCancelling:
		return true
	default:
		return false
	}
}

func FormatConsumptionBackfillStatusLabel(status string) string {
	switch status {
	case ConsumptionBackfillJobStatusPending:
		return "pending"
	case ConsumptionBackfillJobStatusRunning:
		return "running"
	case ConsumptionBackfillJobStatusCancelling:
		return "cancelling"
	case ConsumptionBackfillJobStatusCancelled:
		return "cancelled"
	case ConsumptionBackfillJobStatusCompleted:
		return "completed"
	case ConsumptionBackfillJobStatusCompletedWithErrors:
		return "completed_with_errors"
	case ConsumptionBackfillJobStatusInterrupted:
		return "interrupted"
	default:
		return strings.TrimSpace(status)
	}
}

func FormatProviderBackfillStatusLabel(status string) string {
	switch status {
	case ConsumptionBackfillProviderStatusPending,
		ConsumptionBackfillProviderStatusRunning,
		ConsumptionBackfillProviderStatusCompleted,
		ConsumptionBackfillProviderStatusFailed,
		ConsumptionBackfillProviderStatusCancelled,
		ConsumptionBackfillProviderStatusInterrupted:
		return status
	default:
		return strings.TrimSpace(status)
	}
}

func parseJobID(jobID string) (int, error) {
	parsed, err := strconv.Atoi(strings.TrimSpace(jobID))
	if err != nil || parsed <= 0 {
		return 0, fmt.Errorf("无效的任务 ID")
	}

	return parsed, nil
}
