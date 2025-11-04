package services

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"balance-monitor/internal/database"
	"balance-monitor/internal/models"
)

// checkResult 存储单个供应商的检查结果
type checkResult struct {
	provider database.Provider
	quota    int64
	balance  float64
	err      error
}

// maskString 用于遮蔽敏感信息
func maskString(s string) string {
	if len(s) <= 10 {
		return "****"
	}
	return s[:5] + "****" + s[len(s)-5:]
}

func CheckAllProviders(db *sql.DB) ([]models.BalanceInfo, error) {
	// 获取所有启用的供应商
	rows, err := db.Query("SELECT id, name, upstream_url, authorization, new_api_user, alert_threshold FROM providers WHERE enabled = 1")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var providers []database.Provider
	for rows.Next() {
		var provider database.Provider
		if err := rows.Scan(&provider.ID, &provider.Name, &provider.UpstreamURL, &provider.Authorization, &provider.NewAPIUser, &provider.AlertThreshold); err != nil {
			fmt.Printf("扫描供应商失败: %v\n", err)
			continue
		}
		providers = append(providers, provider)
	}

	if len(providers) == 0 {
		return nil, fmt.Errorf("没有可用的启用供应商")
	}

	// fmt.Printf("\n开始检查 %d 个供应商的余额...\n", len(providers))
	// fmt.Println("====================================================")
	
	var results []models.BalanceInfo
	var errors []string

	// 逐个检查供应商（不再在 CheckBalance 中保存，而是收集结果后批量保存）
	var checkResults []checkResult

	for _, provider := range providers {
		// fmt.Printf("\n正在检查供应商: %s (ID: %d, URL: %s)\n", provider.Name, provider.ID, provider.UpstreamURL)
		
		quota, balance, err := fetchBalance(&provider)
		checkResults = append(checkResults, checkResult{
			provider: provider,
			quota:    quota,
			balance:  balance,
			err:      err,
		})
		
		if err != nil {
			errMsg := fmt.Sprintf("检查供应商 %s 失败: %v", provider.Name, err)
			// fmt.Printf("✗ %s\n", errMsg)
			errors = append(errors, errMsg)
		} else {
			// fmt.Printf("✓ 供应商 %s 检查成功，余额: %.2f\n", provider.Name, balance)
		}
	}

	// 批量保存到数据库（使用事务）
	// fmt.Println("\n正在保存检查结果到数据库...")
	if err := saveBalanceResults(db, checkResults); err != nil {
		return nil, fmt.Errorf("保存检查结果失败: %v", err)
	}

	// 构建返回结果并发送告警
	for _, result := range checkResults {
		if result.err == nil {
			balanceInfo := models.BalanceInfo{
				ProviderID:   result.provider.ID,
				ProviderName: result.provider.Name,
				Quota:        result.quota,
				Balance:      result.balance,
				LastChecked:  time.Now().Format(time.RFC3339),
			}
			results = append(results, balanceInfo)

			// 检查是否需要发送告警
			CheckAndSendAlert(db, &result.provider, result.balance)
		}
	}

	// fmt.Println("====================================================")
	fmt.Printf("检查完成: 成功 %d 个，失败 %d 个\n\n", len(results), len(errors))

	// 如果有错误，返回错误信息
	if len(errors) > 0 {
		// 如果所有供应商都失败了，返回错误
		if len(results) == 0 {
			return nil, fmt.Errorf("所有供应商检查失败: %v", errors)
		}
		// 如果部分成功，打印警告但返回成功的结果
		// fmt.Printf("警告: 部分供应商检查失败: %v\n", errors)
	}

	return results, nil
}

// fetchBalance 从上游获取余额信息（不保存到数据库）
func fetchBalance(provider *database.Provider) (int64, float64, error) {
	requestURL := provider.UpstreamURL + "/api/user/self"
	// fmt.Printf("  → 请求 URL: %s\n", requestURL)
	
	client := &http.Client{Timeout: 30 * time.Second}
	req, err := http.NewRequest("GET", requestURL, nil)
	if err != nil {
		return 0, 0, fmt.Errorf("创建请求失败: %v", err)
	}

	// 设置请求头
	req.Header.Set("Authorization", provider.Authorization)
	req.Header.Set("New-API-User", provider.NewAPIUser)
	// fmt.Printf("  → 请求头: Authorization=%s, New-API-User=%s\n", maskString(provider.Authorization), provider.NewAPIUser)

	// 发送请求
	// fmt.Println("  → 发送 HTTP 请求...")
	resp, err := client.Do(req)
	if err != nil {
		return 0, 0, fmt.Errorf("请求失败: %v", err)
	}
	defer resp.Body.Close()

	// fmt.Printf("  → 收到响应，状态码: %d\n", resp.StatusCode)

	// 读取响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, 0, fmt.Errorf("读取响应失败: %v", err)
	}

	// fmt.Printf("  → 响应内容: %s\n", string(body))

	// 检查 HTTP 状态码
	if resp.StatusCode != http.StatusOK {
		return 0, 0, fmt.Errorf("请求失败，状态码: %d, 响应: %s", resp.StatusCode, string(body))
	}

	// 解析响应
	var quotaResp models.QuotaResponse
	if err := json.Unmarshal(body, &quotaResp); err != nil {
		return 0, 0, fmt.Errorf("解析响应失败: %v, 响应内容: %s", err, string(body))
	}

	// 计算余额
	quota := quotaResp.Data.Quota
	balance := float64(quota) / 500000.0
	// fmt.Printf("  → 解析成功: quota=%d, balance=%.2f\n", quota, balance)

	return quota, balance, nil
}

// saveBalanceResults 批量保存检查结果到数据库（使用事务）
func saveBalanceResults(db *sql.DB, results []checkResult) error {
	// 开始事务
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("开始事务失败: %v", err)
	}
	defer tx.Rollback() // 如果没有 commit，回滚事务

	// 准备批量插入语句
	stmt, err := tx.Prepare("INSERT INTO balance_history (provider_id, quota, balance) VALUES (?, ?, ?)")
	if err != nil {
		return fmt.Errorf("准备语句失败: %v", err)
	}
	defer stmt.Close()

	// 批量插入成功的检查结果
	successCount := 0
	for _, result := range results {
		if result.err == nil {
			_, err := stmt.Exec(result.provider.ID, result.quota, result.balance)
			if err != nil {
				return fmt.Errorf("插入供应商 %s 的数据失败: %v", result.provider.Name, err)
			}
			successCount++
		}
	}

	// 提交事务
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("提交事务失败: %v", err)
	}

	// fmt.Printf("✓ 已成功保存 %d 条记录到数据库\n", successCount)
	return nil
}

