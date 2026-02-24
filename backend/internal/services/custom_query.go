package services

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"balance-monitor/internal/database"
)

// SaveCustomQuery saves or updates a custom query config for a provider (upsert).
// Validates that url and balanceFieldPath are not empty.
func SaveCustomQuery(db *sql.DB, providerID int, method, url string, headers map[string]string, balanceFieldPath string) error {
	if strings.TrimSpace(url) == "" {
		return fmt.Errorf("请求 URL 不能为空")
	}
	if strings.TrimSpace(balanceFieldPath) == "" {
		return fmt.Errorf("余额字段路径不能为空")
	}

	headersJSON, err := json.Marshal(headers)
	if err != nil {
		return fmt.Errorf("序列化请求头失败: %v", err)
	}

	if method == "" {
		method = "GET"
	}

	_, err = db.Exec(
		`INSERT OR REPLACE INTO custom_queries (provider_id, method, url, headers, balance_field_path, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, COALESCE((SELECT created_at FROM custom_queries WHERE provider_id = ?), CURRENT_TIMESTAMP), CURRENT_TIMESTAMP)`,
		providerID, method, url, string(headersJSON), balanceFieldPath, providerID,
	)
	if err != nil {
		return fmt.Errorf("保存自定义查询配置失败: %v", err)
	}

	return nil
}

// GetCustomQuery retrieves the custom query config for a provider.
// Returns nil, nil if not found.
func GetCustomQuery(db *sql.DB, providerID int) (*database.CustomQuery, error) {
	var cq database.CustomQuery
	err := db.QueryRow(
		`SELECT id, provider_id, method, url, headers, balance_field_path, created_at, updated_at
		 FROM custom_queries WHERE provider_id = ?`, providerID,
	).Scan(&cq.ID, &cq.ProviderID, &cq.Method, &cq.URL, &cq.Headers, &cq.BalanceFieldPath, &cq.CreatedAt, &cq.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("查询自定义查询配置失败: %v", err)
	}

	return &cq, nil
}

// DeleteCustomQuery deletes the custom query config for a provider.
func DeleteCustomQuery(db *sql.DB, providerID int) error {
	_, err := db.Exec("DELETE FROM custom_queries WHERE provider_id = ?", providerID)
	if err != nil {
		return fmt.Errorf("删除自定义查询配置失败: %v", err)
	}
	return nil
}

// ExtractBalance extracts a balance value from JSON data using a dot-separated field path.
// Supports nested paths (e.g. "data.balance"), and handles float64, string, and json.Number types.
func ExtractBalance(data []byte, fieldPath string) (float64, error) {
	if len(data) == 0 {
		return 0, fmt.Errorf("JSON 数据为空")
	}

	var obj map[string]interface{}
	decoder := json.NewDecoder(strings.NewReader(string(data)))
	decoder.UseNumber()
	if err := decoder.Decode(&obj); err != nil {
		return 0, fmt.Errorf("解析 JSON 失败: %v", err)
	}

	segments := strings.Split(fieldPath, ".")
	var current interface{} = obj

	for i, seg := range segments {
		m, ok := current.(map[string]interface{})
		if !ok {
			return 0, fmt.Errorf("字段路径 '%s' 在第 %d 层不是对象", fieldPath, i)
		}
		val, exists := m[seg]
		if !exists {
			return 0, fmt.Errorf("字段路径 '%s' 在响应中不存在", fieldPath)
		}
		current = val
	}

	// Handle the final value
	switch v := current.(type) {
	case json.Number:
		f, err := v.Float64()
		if err != nil {
			return 0, fmt.Errorf("字段值 '%v' 无法转换为数字: %v", v, err)
		}
		return f, nil
	case float64:
		return v, nil
	case string:
		f, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return 0, fmt.Errorf("字段值 '%s' 无法转换为数字: %v", v, err)
		}
		return f, nil
	default:
		return 0, fmt.Errorf("字段值类型 '%T' 无法转换为数字", current)
	}
}

// ExecuteCustomQuery executes a custom HTTP query and extracts the balance.
// Returns the extracted balance, the raw response body, and any error.
func ExecuteCustomQuery(method, url string, headers map[string]string, balanceFieldPath string) (float64, json.RawMessage, error) {
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		return 0, nil, fmt.Errorf("创建请求失败: %v", err)
	}

	for k, v := range headers {
		req.Header.Set(k, v)
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return 0, nil, fmt.Errorf("请求失败: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, nil, fmt.Errorf("读取响应失败: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		return 0, json.RawMessage(body), fmt.Errorf("HTTP 请求失败，状态码: %d", resp.StatusCode)
	}

	balance, err := ExtractBalance(body, balanceFieldPath)
	if err != nil {
		return 0, json.RawMessage(body), err
	}

	return balance, json.RawMessage(body), nil
}
