package services

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"balance-monitor/internal/database"
)

const quotaPerBalanceUnit = 500000.0

var shanghaiLocation = func() *time.Location {
	location, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		return time.FixedZone("CST", 8*60*60)
	}
	return location
}()

type DailyConsumptionBackfillResult struct {
	ProviderID    int     `json:"provider_id"`
	Date          string  `json:"date"`
	Status        string  `json:"status"`
	QuotaConsumed int64   `json:"quota_consumed"`
	RequestCount  int64   `json:"request_count"`
	TokenUsed     int64   `json:"token_used"`
	RMBConsumed   float64 `json:"rmb_consumed"`
}

type dailyConsumptionHistoryRecord struct {
	QuotaConsumed int64
	RequestCount  int64
	TokenUsed     int64
}

type dataSelfResponse struct {
	Data []struct {
		Quota     int64 `json:"quota"`
		Count     int64 `json:"count"`
		TokenUsed int64 `json:"token_used"`
	} `json:"data"`
	Message string `json:"message"`
	Success bool   `json:"success"`
}

func BackfillProviderDailyConsumptionHistory(db *sql.DB, providerID int, date string) (*DailyConsumptionBackfillResult, error) {
	provider, err := getBackfillProvider(db, providerID)
	if err != nil {
		return nil, err
	}

	return backfillProviderDailyConsumptionHistory(db, provider, date)
}

func ShanghaiStartDate(days int) string {
	if days <= 0 {
		days = 1
	}

	now := time.Now().In(shanghaiLocation)
	start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, shanghaiLocation).
		AddDate(0, 0, -(days - 1))
	return start.Format("2006-01-02")
}

func getBackfillProvider(db *sql.DB, providerID int) (*database.Provider, error) {
	var provider database.Provider
	err := db.QueryRow(`
		SELECT id, name, upstream_url, authorization, new_api_user, enabled, query_mode, price_ratio
		FROM providers
		WHERE id = ?
	`, providerID).Scan(
		&provider.ID,
		&provider.Name,
		&provider.UpstreamURL,
		&provider.Authorization,
		&provider.NewAPIUser,
		&provider.Enabled,
		&provider.QueryMode,
		&provider.PriceRatio,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("供应商不存在")
		}
		return nil, fmt.Errorf("查询供应商失败: %w", err)
	}

	if !provider.Enabled {
		return nil, fmt.Errorf("供应商未启用")
	}
	if provider.QueryMode != "default" {
		return nil, fmt.Errorf("仅支持默认模式供应商回补历史数据")
	}
	if provider.PriceRatio <= 0 {
		provider.PriceRatio = 1.0
	}

	return &provider, nil
}

func getExistingDailyConsumptionRecord(db *sql.DB, providerID int, date string) (*dailyConsumptionHistoryRecord, bool, error) {
	var record dailyConsumptionHistoryRecord
	err := db.QueryRow(`
		SELECT quota_consumed, request_count, token_used
		FROM provider_daily_consumption_history
		WHERE provider_id = ? AND date = ?
	`, providerID, date).Scan(&record.QuotaConsumed, &record.RequestCount, &record.TokenUsed)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, false, nil
		}
		return nil, false, err
	}

	return &record, true, nil
}

func backfillProviderDailyConsumptionHistory(db *sql.DB, provider *database.Provider, date string) (*DailyConsumptionBackfillResult, error) {
	normalizedDate, err := normalizeShanghaiDate(date)
	if err != nil {
		return nil, err
	}

	existing, found, err := getExistingDailyConsumptionRecord(db, provider.ID, normalizedDate)
	if err != nil {
		return nil, fmt.Errorf("查询历史消耗记录失败: %w", err)
	}
	if found {
		return buildDailyConsumptionBackfillResult(provider, normalizedDate, "skipped", existing.QuotaConsumed, existing.RequestCount, existing.TokenUsed), nil
	}

	quotaConsumed, requestCount, tokenUsed, err := fetchDailyConsumptionFromUpstream(provider, normalizedDate)
	if err != nil {
		return nil, err
	}

	_, err = db.Exec(`
		INSERT INTO provider_daily_consumption_history
			(provider_id, date, quota_consumed, request_count, token_used, fetched_at)
		VALUES (?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
	`, provider.ID, normalizedDate, quotaConsumed, requestCount, tokenUsed)
	if err != nil {
		return nil, fmt.Errorf("保存历史消耗记录失败: %w", err)
	}

	return buildDailyConsumptionBackfillResult(provider, normalizedDate, "inserted", quotaConsumed, requestCount, tokenUsed), nil
}

func fetchDailyConsumptionFromUpstream(provider *database.Provider, date string) (int64, int64, int64, error) {
	startTime, endTime, err := shanghaiDayRange(date)
	if err != nil {
		return 0, 0, 0, err
	}

	query := url.Values{}
	query.Set("start_timestamp", strconv.FormatInt(startTime.Unix(), 10))
	query.Set("end_timestamp", strconv.FormatInt(endTime.Unix(), 10))
	query.Set("default_time", "day")

	requestURL := strings.TrimRight(provider.UpstreamURL, "/") + "/api/data/self?" + query.Encode()

	client := &http.Client{Timeout: 30 * time.Second}
	req, err := http.NewRequest(http.MethodGet, requestURL, nil)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("创建历史数据请求失败: %w", err)
	}

	req.Header.Set("Authorization", provider.Authorization)
	req.Header.Set("New-API-User", provider.NewAPIUser)

	resp, err := client.Do(req)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("请求历史数据失败: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("读取历史数据响应失败: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return 0, 0, 0, fmt.Errorf("历史数据请求失败，状态码: %d, 响应: %s", resp.StatusCode, string(body))
	}

	var payload dataSelfResponse
	if err := json.Unmarshal(body, &payload); err != nil {
		return 0, 0, 0, fmt.Errorf("解析历史数据响应失败: %w", err)
	}
	if !payload.Success {
		if strings.TrimSpace(payload.Message) == "" {
			payload.Message = "上游返回 success=false"
		}
		return 0, 0, 0, fmt.Errorf("历史数据请求失败: %s", payload.Message)
	}

	var quotaConsumed int64
	var requestCount int64
	var tokenUsed int64
	for _, item := range payload.Data {
		quotaConsumed += item.Quota
		requestCount += item.Count
		tokenUsed += item.TokenUsed
	}

	return quotaConsumed, requestCount, tokenUsed, nil
}

func normalizeShanghaiDate(date string) (string, error) {
	startTime, _, err := shanghaiDayRange(date)
	if err != nil {
		return "", err
	}
	return startTime.Format("2006-01-02"), nil
}

func shanghaiDayRange(date string) (time.Time, time.Time, error) {
	startTime, err := time.ParseInLocation("2006-01-02", strings.TrimSpace(date), shanghaiLocation)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("无效的日期格式，应为 YYYY-MM-DD")
	}
	return startTime, startTime.Add(24 * time.Hour), nil
}

func quotaToRMB(quotaConsumed int64, priceRatio float64) float64 {
	if priceRatio <= 0 {
		priceRatio = 1.0
	}
	return (float64(quotaConsumed) / quotaPerBalanceUnit) * priceRatio
}

func buildDailyConsumptionBackfillResult(provider *database.Provider, date, status string, quotaConsumed, requestCount, tokenUsed int64) *DailyConsumptionBackfillResult {
	return &DailyConsumptionBackfillResult{
		ProviderID:    provider.ID,
		Date:          date,
		Status:        status,
		QuotaConsumed: quotaConsumed,
		RequestCount:  requestCount,
		TokenUsed:     tokenUsed,
		RMBConsumed:   quotaToRMB(quotaConsumed, provider.PriceRatio),
	}
}
