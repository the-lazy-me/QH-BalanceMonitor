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

