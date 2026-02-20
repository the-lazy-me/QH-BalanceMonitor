package models

type QuotaResponse struct {
	Success bool `json:"success"`
	Message string `json:"message"`
	Data    struct {
		Quota int64 `json:"quota"`
	} `json:"data"`
}

type BalanceInfo struct {
	ProviderID   int     `json:"provider_id"`
	ProviderName string  `json:"provider_name"`
	Quota        int64   `json:"quota"`
	Balance      float64 `json:"balance"`
	LastChecked  string  `json:"last_checked"`
}


// AccountingRecord 记账记录
type AccountingRecord struct {
	ID           int     `json:"id"`
	ProviderID   int     `json:"provider_id"`
	ProviderName string  `json:"provider_name,omitempty"` // 查询时 JOIN 填充
	Amount       float64 `json:"amount"`
	Date         string  `json:"date"`       // 格式: YYYY-MM-DD
	Note         string  `json:"note"`
	CreatedAt    string  `json:"created_at"`
	UpdatedAt    string  `json:"updated_at"`
}

// AccountingCreateRequest 新增记账请求
type AccountingCreateRequest struct {
	ProviderID int     `json:"provider_id" binding:"required"`
	Amount     float64 `json:"amount" binding:"required"`
	Date       string  `json:"date" binding:"required"`
	Note       string  `json:"note"`
}

// AccountingUpdateRequest 编辑记账请求（指针类型支持部分更新）
type AccountingUpdateRequest struct {
	Amount *float64 `json:"amount"`
	Date   *string  `json:"date"`
	Note   *string  `json:"note"`
}

// ProviderCostSummary 按供应商成本汇总
type ProviderCostSummary struct {
	ProviderID   int     `json:"provider_id"`
	ProviderName string  `json:"provider_name"`
	TotalAmount  float64 `json:"total_amount"`
	RecordCount  int     `json:"record_count"`
}

// MonthlyCostSummary 按月成本汇总
type MonthlyCostSummary struct {
	Month       string  `json:"month"` // 格式: YYYY-MM
	TotalAmount float64 `json:"total_amount"`
	RecordCount int     `json:"record_count"`
}
