package services

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestConsumptionBackfillJobRunsProvidersInParallelButEachProviderSerial(t *testing.T) {
	db := setupConsumptionHistoryTestDB(t)
	defer db.Close()

	userID := insertBackfillUser(t, db, "job-user")

	var (
		mu                sync.Mutex
		currentByProvider = make(map[string]int)
		maxByProvider     = make(map[string]int)
		currentGlobal     int
		maxGlobal         int
	)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		providerKey := r.Header.Get("New-API-User")

		mu.Lock()
		currentByProvider[providerKey]++
		if currentByProvider[providerKey] > maxByProvider[providerKey] {
			maxByProvider[providerKey] = currentByProvider[providerKey]
		}
		currentGlobal++
		if currentGlobal > maxGlobal {
			maxGlobal = currentGlobal
		}
		mu.Unlock()

		time.Sleep(40 * time.Millisecond)

		mu.Lock()
		currentByProvider[providerKey]--
		currentGlobal--
		mu.Unlock()

		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"message": "",
			"data": []map[string]interface{}{
				{"quota": 100000, "count": 1, "token_used": 10},
			},
		})
	}))
	defer server.Close()

	providerOneID := insertBackfillProviderWithUserID(t, db, "provider-one", server.URL, "provider-one", true, "default", 1.0)
	providerTwoID := insertBackfillProviderWithUserID(t, db, "provider-two", server.URL, "provider-two", true, "default", 1.0)

	job, created, err := CreateConsumptionBackfillJob(db, userID, []int{providerOneID, providerTwoID}, 2)
	if err != nil {
		t.Fatalf("create backfill job failed: %v", err)
	}
	if !created {
		t.Fatalf("expected job to be created")
	}

	finalJob := waitForBackfillJobTerminal(t, db, job.JobID, 5*time.Second)
	if finalJob.Status != ConsumptionBackfillJobStatusCompleted {
		t.Fatalf("expected completed job, got %+v", finalJob)
	}
	if finalJob.InsertedDays != 4 || finalJob.SkippedDays != 0 {
		t.Fatalf("unexpected job day counts: %+v", finalJob)
	}

	mu.Lock()
	defer mu.Unlock()
	if maxGlobal < 2 {
		t.Fatalf("expected providers to run in parallel, max global concurrency = %d", maxGlobal)
	}
	if maxByProvider["provider-one"] != 1 || maxByProvider["provider-two"] != 1 {
		t.Fatalf("expected each provider to run serially, got provider-one=%d provider-two=%d", maxByProvider["provider-one"], maxByProvider["provider-two"])
	}
}

func TestConsumptionBackfillJobMarksSingleProviderFailureWithoutStoppingOthers(t *testing.T) {
	db := setupConsumptionHistoryTestDB(t)
	defer db.Close()

	userID := insertBackfillUser(t, db, "job-user")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("New-API-User") == "fail-provider" {
			w.WriteHeader(http.StatusBadGateway)
			_, _ = w.Write([]byte(`{"success":false,"message":"upstream error"}`))
			return
		}

		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"message": "",
			"data": []map[string]interface{}{
				{"quota": 100000, "count": 1, "token_used": 10},
			},
		})
	}))
	defer server.Close()

	okProviderID := insertBackfillProviderWithUserID(t, db, "ok-provider", server.URL, "ok-provider", true, "default", 1.0)
	failProviderID := insertBackfillProviderWithUserID(t, db, "fail-provider", server.URL, "fail-provider", true, "default", 1.0)

	job, created, err := CreateConsumptionBackfillJob(db, userID, []int{okProviderID, failProviderID}, 1)
	if err != nil {
		t.Fatalf("create backfill job failed: %v", err)
	}
	if !created {
		t.Fatalf("expected job to be created")
	}

	finalJob := waitForBackfillJobTerminal(t, db, job.JobID, 5*time.Second)
	if finalJob.Status != ConsumptionBackfillJobStatusCompletedWithErrors {
		t.Fatalf("expected completed_with_errors, got %+v", finalJob)
	}
	if finalJob.CompletedProviders != 1 || finalJob.FailedProviders != 1 {
		t.Fatalf("unexpected provider summary: %+v", finalJob)
	}
}

func TestConsumptionBackfillJobCancelsAfterCurrentDayCompletes(t *testing.T) {
	db := setupConsumptionHistoryTestDB(t)
	defer db.Close()

	userID := insertBackfillUser(t, db, "job-user")

	var requestCount int32
	started := make(chan struct{})
	release := make(chan struct{})
	var once sync.Once

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&requestCount, 1)
		once.Do(func() {
			close(started)
		})
		<-release

		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"message": "",
			"data": []map[string]interface{}{
				{"quota": 100000, "count": 1, "token_used": 10},
			},
		})
	}))
	defer server.Close()

	providerID := insertBackfillProviderWithUserID(t, db, "provider-cancel", server.URL, "provider-cancel", true, "default", 1.0)

	job, created, err := CreateConsumptionBackfillJob(db, userID, []int{providerID}, 3)
	if err != nil {
		t.Fatalf("create backfill job failed: %v", err)
	}
	if !created {
		t.Fatalf("expected job to be created")
	}

	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatalf("timed out waiting for provider request to start")
	}

	if err := RequestCancelConsumptionBackfillJob(db, job.JobID); err != nil {
		t.Fatalf("cancel backfill job failed: %v", err)
	}
	close(release)

	finalJob := waitForBackfillJobTerminal(t, db, job.JobID, 5*time.Second)
	if finalJob.Status != ConsumptionBackfillJobStatusCancelled {
		t.Fatalf("expected cancelled job, got %+v", finalJob)
	}
	if len(finalJob.Providers) != 1 || finalJob.Providers[0].Status != ConsumptionBackfillProviderStatusCancelled {
		t.Fatalf("unexpected provider payload: %+v", finalJob.Providers)
	}
	if finalJob.InsertedDays+finalJob.SkippedDays != 1 {
		t.Fatalf("expected one processed day before cancellation, got %+v", finalJob)
	}
	if atomic.LoadInt32(&requestCount) != 1 {
		t.Fatalf("expected only one request before cancellation, got %d", requestCount)
	}
}

func TestMarkInterruptedConsumptionBackfillJobs(t *testing.T) {
	db := setupConsumptionHistoryTestDB(t)
	defer db.Close()

	userID := insertBackfillUser(t, db, "job-user")
	providerID := insertBackfillProviderWithUserID(t, db, "provider-interrupted", "https://example.com", "provider-interrupted", true, "default", 1.0)

	_, err := db.Exec(`
		INSERT INTO consumption_backfill_jobs (status, days_limit, created_by)
		VALUES
			(?, ?, ?),
			(?, ?, ?)
	`,
		ConsumptionBackfillJobStatusRunning, DefaultConsumptionBackfillDaysLimit, userID,
		ConsumptionBackfillJobStatusCancelling, DefaultConsumptionBackfillDaysLimit, userID,
	)
	if err != nil {
		t.Fatalf("insert backfill jobs failed: %v", err)
	}

	rows, err := db.Query(`SELECT id FROM consumption_backfill_jobs ORDER BY id ASC`)
	if err != nil {
		t.Fatalf("query job ids failed: %v", err)
	}
	defer rows.Close()

	var jobIDs []int
	for rows.Next() {
		var jobID int
		if err := rows.Scan(&jobID); err != nil {
			t.Fatalf("scan job id failed: %v", err)
		}
		jobIDs = append(jobIDs, jobID)
	}

	_, err = db.Exec(`
		INSERT INTO consumption_backfill_job_providers (job_id, provider_id, status)
		VALUES
			(?, ?, ?),
			(?, ?, ?)
	`,
		jobIDs[0], providerID, ConsumptionBackfillProviderStatusRunning,
		jobIDs[1], providerID, ConsumptionBackfillProviderStatusPending,
	)
	if err != nil {
		t.Fatalf("insert job providers failed: %v", err)
	}

	if err := MarkInterruptedConsumptionBackfillJobs(db); err != nil {
		t.Fatalf("mark interrupted failed: %v", err)
	}

	var interruptedJobs int
	if err := db.QueryRow(`
		SELECT COUNT(*) FROM consumption_backfill_jobs WHERE status = ?
	`, ConsumptionBackfillJobStatusInterrupted).Scan(&interruptedJobs); err != nil {
		t.Fatalf("count interrupted jobs failed: %v", err)
	}
	if interruptedJobs != 2 {
		t.Fatalf("expected 2 interrupted jobs, got %d", interruptedJobs)
	}

	var interruptedProviders int
	if err := db.QueryRow(`
		SELECT COUNT(*) FROM consumption_backfill_job_providers WHERE status = ?
	`, ConsumptionBackfillProviderStatusInterrupted).Scan(&interruptedProviders); err != nil {
		t.Fatalf("count interrupted provider jobs failed: %v", err)
	}
	if interruptedProviders != 2 {
		t.Fatalf("expected 2 interrupted provider tasks, got %d", interruptedProviders)
	}
}

func waitForBackfillJobTerminal(t *testing.T, db *sql.DB, jobID int, timeout time.Duration) *ConsumptionBackfillJobDetails {
	t.Helper()

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		job, err := GetConsumptionBackfillJob(db, jobID)
		if err != nil {
			t.Fatalf("get backfill job failed: %v", err)
		}
		if job == nil {
			t.Fatalf("expected job %d to exist", jobID)
		}
		if !IsActiveConsumptionBackfillJobStatus(job.Status) {
			return job
		}
		time.Sleep(20 * time.Millisecond)
	}

	t.Fatalf("timed out waiting for backfill job %d to finish", jobID)
	return nil
}
