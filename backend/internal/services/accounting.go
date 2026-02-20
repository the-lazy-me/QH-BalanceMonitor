package services

import (
	"database/sql"
	"errors"
	"strings"

	"balance-monitor/internal/models"
)

// CreateAccountingRecord 创建记账记录
// 校验供应商存在性、金额为正数，插入记录并返回包含 ID 的完整记录
func CreateAccountingRecord(db *sql.DB, req models.AccountingCreateRequest) (*models.AccountingRecord, error) {
	// 校验金额为正数
	if req.Amount <= 0 {
		return nil, errors.New("金额必须为正数")
	}

	// 校验供应商存在性
	var providerExists bool
	err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM providers WHERE id = ?)", req.ProviderID).Scan(&providerExists)
	if err != nil {
		return nil, err
	}
	if !providerExists {
		return nil, errors.New("供应商不存在")
	}

	// 插入记录
	result, err := db.Exec(
		`INSERT INTO accounting_records (provider_id, amount, date, note) VALUES (?, ?, ?, ?)`,
		req.ProviderID, req.Amount, req.Date, req.Note,
	)
	if err != nil {
		return nil, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}

	// 查询完整记录返回
	var record models.AccountingRecord
	err = db.QueryRow(
		`SELECT id, provider_id, amount, date, note, created_at, updated_at
		 FROM accounting_records WHERE id = ?`, id,
	).Scan(&record.ID, &record.ProviderID, &record.Amount, &record.Date, &record.Note, &record.CreatedAt, &record.UpdatedAt)
	if err != nil {
		return nil, err
	}

	return &record, nil
}

// GetAccountingRecords 按条件查询记账记录
// JOIN providers 表获取供应商名称，支持 provider_id、start_date、end_date 可选筛选，按日期倒序排列
func GetAccountingRecords(db *sql.DB, providerID int, startDate, endDate string) ([]models.AccountingRecord, error) {
	query := `SELECT ar.id, ar.provider_id, p.name AS provider_name, ar.amount, ar.date, ar.note, ar.created_at, ar.updated_at
		FROM accounting_records ar
		JOIN providers p ON ar.provider_id = p.id`

	var conditions []string
	var args []interface{}

	if providerID > 0 {
		conditions = append(conditions, "ar.provider_id = ?")
		args = append(args, providerID)
	}
	if startDate != "" {
		conditions = append(conditions, "ar.date >= ?")
		args = append(args, startDate)
	}
	if endDate != "" {
		conditions = append(conditions, "ar.date <= ?")
		args = append(args, endDate)
	}

	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	query += " ORDER BY ar.date DESC"

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []models.AccountingRecord
	for rows.Next() {
		var r models.AccountingRecord
		if err := rows.Scan(&r.ID, &r.ProviderID, &r.ProviderName, &r.Amount, &r.Date, &r.Note, &r.CreatedAt, &r.UpdatedAt); err != nil {
			return nil, err
		}
		records = append(records, r)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return records, nil
}

// UpdateAccountingRecord 更新记账记录
// 校验记录存在性，对提交的字段执行与创建相同的校验规则，支持部分字段更新
func UpdateAccountingRecord(db *sql.DB, id int, req models.AccountingUpdateRequest) error {
	// 校验记录存在性
	var exists bool
	err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM accounting_records WHERE id = ?)", id).Scan(&exists)
	if err != nil {
		return err
	}
	if !exists {
		return errors.New("记账记录不存在")
	}

	// 校验金额为正数（仅当提交了金额字段时）
	if req.Amount != nil && *req.Amount <= 0 {
		return errors.New("金额必须为正数")
	}

	// 动态构建 SQL 更新语句（仅更新非 nil 字段）
	var setClauses []string
	var args []interface{}

	if req.Amount != nil {
		setClauses = append(setClauses, "amount = ?")
		args = append(args, *req.Amount)
	}
	if req.Date != nil {
		setClauses = append(setClauses, "date = ?")
		args = append(args, *req.Date)
	}
	if req.Note != nil {
		setClauses = append(setClauses, "note = ?")
		args = append(args, *req.Note)
	}

	// 无字段需要更新时直接返回
	if len(setClauses) == 0 {
		return nil
	}

	// 始终更新 updated_at
	setClauses = append(setClauses, "updated_at = CURRENT_TIMESTAMP")
	args = append(args, id)

	query := "UPDATE accounting_records SET " + strings.Join(setClauses, ", ") + " WHERE id = ?"
	_, err = db.Exec(query, args...)
	return err
}

// DeleteAccountingRecord 删除记账记录
// 校验记录存在性，删除记录
func DeleteAccountingRecord(db *sql.DB, id int) error {
	// 校验记录存在性
	var exists bool
	err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM accounting_records WHERE id = ?)", id).Scan(&exists)
	if err != nil {
		return err
	}
	if !exists {
		return errors.New("记账记录不存在")
	}

	_, err = db.Exec("DELETE FROM accounting_records WHERE id = ?", id)
	return err
}




// GetProviderCostStats 按供应商分组统计成本
// 支持 start_date、end_date 可选日期范围筛选，返回每个供应商的汇总和总金额合计
func GetProviderCostStats(db *sql.DB, startDate, endDate string) ([]models.ProviderCostSummary, float64, error) {
	query := `SELECT ar.provider_id, p.name AS provider_name,
		SUM(ar.amount) AS total_amount, COUNT(*) AS record_count
		FROM accounting_records ar
		JOIN providers p ON ar.provider_id = p.id`

	var conditions []string
	var args []interface{}

	if startDate != "" {
		conditions = append(conditions, "ar.date >= ?")
		args = append(args, startDate)
	}
	if endDate != "" {
		conditions = append(conditions, "ar.date <= ?")
		args = append(args, endDate)
	}

	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	query += " GROUP BY ar.provider_id, p.name"

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var summaries []models.ProviderCostSummary
	var totalAmount float64

	for rows.Next() {
		var s models.ProviderCostSummary
		if err := rows.Scan(&s.ProviderID, &s.ProviderName, &s.TotalAmount, &s.RecordCount); err != nil {
			return nil, 0, err
		}
		summaries = append(summaries, s)
		totalAmount += s.TotalAmount
	}

	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	return summaries, totalAmount, nil
}

// GetMonthlyCostStats 按月分组统计成本
// 支持 provider_id 可选筛选（0 表示不筛选），按月份正序排列
func GetMonthlyCostStats(db *sql.DB, providerID int) ([]models.MonthlyCostSummary, error) {
	query := `SELECT strftime('%Y-%m', ar.date) AS month,
		SUM(ar.amount) AS total_amount, COUNT(*) AS record_count
		FROM accounting_records ar`

	var args []interface{}

	if providerID > 0 {
		query += " WHERE ar.provider_id = ?"
		args = append(args, providerID)
	}

	query += " GROUP BY strftime('%Y-%m', ar.date) ORDER BY month ASC"

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var summaries []models.MonthlyCostSummary
	for rows.Next() {
		var s models.MonthlyCostSummary
		if err := rows.Scan(&s.Month, &s.TotalAmount, &s.RecordCount); err != nil {
			return nil, err
		}
		summaries = append(summaries, s)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return summaries, nil
}

