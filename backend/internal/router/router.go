package router

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"balance-monitor/internal/database"
	"balance-monitor/internal/middleware"
	"balance-monitor/internal/models"
	"balance-monitor/internal/services"

	"github.com/gin-gonic/gin"
)

func Setup(db *sql.DB) *gin.Engine {
	r := gin.Default()

	// CORS 中间件
	r.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	})

	api := r.Group("/api")
	{
		// 认证相关（公开端点）
		auth := api.Group("/auth")
		{
			auth.GET("/status", checkInitStatus(db))
			auth.POST("/initialize", initializeSystem(db))
			auth.POST("/login", login(db))
		}

		// 需要认证的端点
		protected := api.Group("")
		protected.Use(middleware.AuthMiddleware())
		{
			// 用户相关
			protected.POST("/auth/change-password", changePassword(db))

			// 供应商管理
			protected.GET("/providers", getProviders(db))
			protected.POST("/providers", createProvider(db))
			protected.PUT("/providers/:id", updateProvider(db))
			protected.DELETE("/providers/:id", deleteProvider(db))

			// 余额相关
			protected.GET("/balance", getCurrentBalance(db))
			protected.GET("/balance/history", getBalanceHistory(db))
			protected.GET("/balance/history/rmb", getBalanceHistoryRMB(db))
			protected.GET("/balance/consumption/rmb", getBalanceConsumptionRMB(db))
			protected.POST("/balance/check", manualCheckBalance(db))

			// SMTP 设置
			protected.GET("/smtp", getSMTPSettings(db))
			protected.PUT("/smtp", updateSMTPSettings(db))
			protected.POST("/smtp/test", testSMTP(db))

			// 监控设置
			protected.GET("/monitor", getMonitorSettings(db))
			protected.PUT("/monitor", updateMonitorSettings(db))

			// 通知历史
			protected.GET("/notifications", getNotifications(db))

			// 记账管理
			protected.GET("/accounting/stats/provider", getProviderCostStats(db))
			protected.GET("/accounting/stats/monthly", getMonthlyCostStats(db))
			protected.POST("/accounting", createAccountingRecord(db))
			protected.GET("/accounting", getAccountingRecords(db))
			protected.PUT("/accounting/:id", updateAccountingRecord(db))
			protected.DELETE("/accounting/:id", deleteAccountingRecord(db))

			// Curl 解析
			protected.POST("/curl/parse", parseCurlHandler())

			// 自定义查询测试
			protected.POST("/custom-query/test", testCustomQueryHandler())

			// 自定义查询配置 CRUD
			protected.GET("/providers/:id/custom-query", getCustomQueryHandler(db))
			protected.PUT("/providers/:id/custom-query", saveCustomQueryHandler(db))
			protected.DELETE("/providers/:id/custom-query", deleteCustomQueryHandler(db))
		}
	}

	return r
}

// 检查系统初始化状态
func checkInitStatus(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		initialized, err := services.IsSystemInitialized(db)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"initialized": initialized})
	}
}

// 初始化系统
func initializeSystem(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Username string `json:"username" binding:"required"`
			Password string `json:"password" binding:"required"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "用户名和密码不能为空"})
			return
		}

		if err := services.InitializeSystem(db, req.Username, req.Password); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "系统初始化成功"})
	}
}

// 登录
func login(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Username string `json:"username" binding:"required"`
			Password string `json:"password" binding:"required"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "用户名和密码不能为空"})
			return
		}

		token, err := services.Login(db, req.Username, req.Password)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"token": token,
			"message": "登录成功",
		})
	}
}

// 修改密码
func changePassword(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			OldPassword string `json:"old_password" binding:"required"`
			NewPassword string `json:"new_password" binding:"required"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "旧密码和新密码不能为空"})
			return
		}

		userID, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "未授权"})
			return
		}

		if err := services.ChangePassword(db, userID.(int), req.OldPassword, req.NewPassword); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "密码修改成功"})
	}
}

func getProviders(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		rows, err := db.Query(`
			SELECT id, name, upstream_url, authorization, new_api_user, enabled, alert_threshold, query_mode, price_ratio, created_at, updated_at
			FROM providers ORDER BY created_at DESC
		`)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		defer rows.Close()

		var providers []database.Provider
		for rows.Next() {
			var p database.Provider
			if err := rows.Scan(&p.ID, &p.Name, &p.UpstreamURL, &p.Authorization, &p.NewAPIUser, &p.Enabled, &p.AlertThreshold, &p.QueryMode, &p.PriceRatio, &p.CreatedAt, &p.UpdatedAt); err != nil {
				continue
			}
			providers = append(providers, p)
		}

		c.JSON(http.StatusOK, providers)
	}
}

func createProvider(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var provider database.Provider
		if err := c.ShouldBindJSON(&provider); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// 设置默认阈值
		if provider.AlertThreshold == 0 {
			provider.AlertThreshold = 10.0
		}

		// 设置默认价格比例
		if provider.PriceRatio == 0 {
			provider.PriceRatio = 1.0
		}

		// 校验价格比例必须为正数
		if provider.PriceRatio <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "价格比例必须为正数"})
			return
		}

		// 设置默认查询模式
		if provider.QueryMode == "" {
			provider.QueryMode = "default"
		}

		result, err := db.Exec(`
			INSERT INTO providers (name, upstream_url, authorization, new_api_user, enabled, alert_threshold, price_ratio, query_mode)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		`, provider.Name, provider.UpstreamURL, provider.Authorization, provider.NewAPIUser, provider.Enabled, provider.AlertThreshold, provider.PriceRatio, provider.QueryMode)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		id, _ := result.LastInsertId()
		provider.ID = int(id)
		provider.CreatedAt = time.Now()
		provider.UpdatedAt = time.Now()

		c.JSON(http.StatusCreated, provider)
	}
}

func updateProvider(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		var provider database.Provider
		if err := c.ShouldBindJSON(&provider); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// 校验价格比例必须为正数
		if provider.PriceRatio <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "价格比例必须为正数"})
			return
		}

		// 设置默认查询模式
		if provider.QueryMode == "" {
			provider.QueryMode = "default"
		}

		_, err := db.Exec(`
			UPDATE providers 
			SET name = ?, upstream_url = ?, authorization = ?, new_api_user = ?, enabled = ?, alert_threshold = ?, price_ratio = ?, query_mode = ?, updated_at = CURRENT_TIMESTAMP
			WHERE id = ?
		`, provider.Name, provider.UpstreamURL, provider.Authorization, provider.NewAPIUser, provider.Enabled, provider.AlertThreshold, provider.PriceRatio, provider.QueryMode, id)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "更新成功"})
	}
}

func deleteProvider(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		_, err := db.Exec("DELETE FROM providers WHERE id = ?", id)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "删除成功"})
	}
}

func getCurrentBalance(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		rows, err := db.Query(`
			SELECT p.id, p.name, p.price_ratio, bh.quota, bh.balance, bh.checked_at
			FROM providers p
			LEFT JOIN (
				SELECT provider_id, quota, balance, checked_at,
					ROW_NUMBER() OVER (PARTITION BY provider_id ORDER BY checked_at DESC) as rn
				FROM balance_history
			) bh ON p.id = bh.provider_id AND bh.rn = 1
			WHERE p.enabled = 1
			ORDER BY p.name
		`)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		defer rows.Close()

		var balances []map[string]interface{}
		for rows.Next() {
			var id int
			var name string
			var priceRatio sql.NullFloat64
			var quota sql.NullInt64
			var balance sql.NullFloat64
			var checkedAt sql.NullTime

			if err := rows.Scan(&id, &name, &priceRatio, &quota, &balance, &checkedAt); err != nil {
				continue
			}

			// 若 price_ratio 为 NULL 或无效值，返回默认值 1.0
			priceRatioValue := 1.0
			if priceRatio.Valid && priceRatio.Float64 > 0 {
				priceRatioValue = priceRatio.Float64
			}

			item := map[string]interface{}{
				"provider_id":   id,
				"provider_name": name,
				"price_ratio":   priceRatioValue,
			}

			if quota.Valid {
				item["quota"] = quota.Int64
				item["balance"] = balance.Float64
				item["last_checked"] = checkedAt.Time.Format(time.RFC3339)
			} else {
				item["quota"] = nil
				item["balance"] = nil
				item["last_checked"] = nil
			}

			balances = append(balances, item)
		}

		c.JSON(http.StatusOK, balances)
	}
}

func getBalanceHistory(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		providerID := c.Query("provider_id")
		limit := c.DefaultQuery("limit", "100")
		days := c.Query("days")

		query := `
			SELECT bh.id, bh.provider_id, p.name, bh.quota, bh.balance, bh.checked_at
			FROM balance_history bh
			JOIN providers p ON bh.provider_id = p.id
		`

		var rows *sql.Rows
		var err error
		var conditions []string
		var args []interface{}

		// 添加供应商筛选条件
		if providerID != "" {
			conditions = append(conditions, "bh.provider_id = ?")
			args = append(args, providerID)
		}

		// 添加天数筛选条件
		if days != "" {
			conditions = append(conditions, "bh.checked_at >= datetime('now', '-' || ? || ' days')")
			args = append(args, days)
		}

		// 组合WHERE条件
		if len(conditions) > 0 {
			query += " WHERE "
			for i, condition := range conditions {
				if i > 0 {
					query += " AND "
				}
				query += condition
			}
		}

		// 添加排序和限制
		query += " ORDER BY bh.checked_at DESC"
		
		// 只有在没有指定天数时才应用limit限制
		if days == "" {
			query += " LIMIT ?"
			args = append(args, limit)
		}

		rows, err = db.Query(query, args...)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		defer rows.Close()

		var history []map[string]interface{}
		for rows.Next() {
			var id, providerID int
			var providerName string
			var quota int64
			var balance float64
			var checkedAt time.Time

			if err := rows.Scan(&id, &providerID, &providerName, &quota, &balance, &checkedAt); err != nil {
				continue
			}

			history = append(history, map[string]interface{}{
				"id":            id,
				"provider_id":   providerID,
				"provider_name": providerName,
				"quota":         quota,
				"balance":       balance,
				"checked_at":    checkedAt.Format(time.RFC3339),
			})
		}

		c.JSON(http.StatusOK, history)
	}
}

func getBalanceHistoryRMB(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		providerID := c.Query("provider_id")
		days := c.DefaultQuery("days", "7")

		query := `
			SELECT bh.provider_id, p.name,
				bh.balance * COALESCE(NULLIF(p.price_ratio, 0), 1.0) AS rmb_balance,
				bh.checked_at
			FROM balance_history bh
			JOIN providers p ON bh.provider_id = p.id
			WHERE bh.checked_at >= datetime('now', '-' || ? || ' days')
		`
		args := []interface{}{days}

		if providerID != "" {
			query += " AND bh.provider_id = ?"
			args = append(args, providerID)
		}

		query += " ORDER BY bh.checked_at ASC"

		rows, err := db.Query(query, args...)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		defer rows.Close()

		var history []map[string]interface{}
		for rows.Next() {
			var pid int
			var name string
			var rmbBalance float64
			var checkedAt time.Time

			if err := rows.Scan(&pid, &name, &rmbBalance, &checkedAt); err != nil {
				continue
			}

			history = append(history, map[string]interface{}{
				"provider_id":   pid,
				"provider_name": name,
				"rmb_balance":   rmbBalance,
				"checked_at":    checkedAt.Format(time.RFC3339),
			})
		}

		c.JSON(http.StatusOK, history)
	}
}


func getBalanceConsumptionRMB(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		providerID := c.Query("provider_id")
		days := c.DefaultQuery("days", "7")

		// 每天每个供应商取最早和最晚的余额，差值为当天消耗
		query := `
			SELECT sub.provider_id, sub.provider_name, sub.date,
				sub.first_balance - sub.last_balance AS consumption
			FROM (
				SELECT bh.provider_id, p.name AS provider_name,
					DATE(bh.checked_at) AS date,
					-- first_value / last_value over the day window
					FIRST_VALUE(bh.balance * COALESCE(NULLIF(p.price_ratio, 0), 1.0))
						OVER (PARTITION BY bh.provider_id, DATE(bh.checked_at) ORDER BY bh.checked_at ASC) AS first_balance,
					FIRST_VALUE(bh.balance * COALESCE(NULLIF(p.price_ratio, 0), 1.0))
						OVER (PARTITION BY bh.provider_id, DATE(bh.checked_at) ORDER BY bh.checked_at DESC) AS last_balance,
					ROW_NUMBER() OVER (PARTITION BY bh.provider_id, DATE(bh.checked_at) ORDER BY bh.checked_at ASC) AS rn
				FROM balance_history bh
				JOIN providers p ON bh.provider_id = p.id
				WHERE bh.checked_at >= datetime('now', '-' || ? || ' days')
		`
		args := []interface{}{days}

		if providerID != "" {
			query += " AND bh.provider_id = ?"
			args = append(args, providerID)
		}

		query += `
			) sub
			WHERE sub.rn = 1
			ORDER BY sub.date ASC, sub.provider_name ASC
		`

		rows, err := db.Query(query, args...)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		defer rows.Close()

		var result []map[string]interface{}
		for rows.Next() {
			var pid int
			var name, date string
			var consumption float64
			if err := rows.Scan(&pid, &name, &date, &consumption); err != nil {
				continue
			}
			// 负值表示充值，不计入消耗
			if consumption < 0 {
				consumption = 0
			}
			result = append(result, map[string]interface{}{
				"provider_id":   pid,
				"provider_name": name,
				"consumption":   consumption,
				"date":          date,
			})
		}

		c.JSON(http.StatusOK, result)
	}
}




func manualCheckBalance(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		results, err := services.CheckAllProviders(db)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, results)
	}
}

func getSMTPSettings(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		settings, err := services.GetSMTPSettings(db)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, settings)
	}
}

func updateSMTPSettings(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var settings database.SMTPSettings
		if err := c.ShouldBindJSON(&settings); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		_, err := db.Exec(`
			UPDATE smtp_settings 
			SET enabled = ?, smtp_host = ?, smtp_port = ?, smtp_username = ?, 
			    smtp_password = ?, from_email = ?, to_emails = ?,
			    updated_at = CURRENT_TIMESTAMP
			WHERE id = 1
		`, settings.Enabled, settings.SMTPHost, settings.SMTPPort, settings.SMTPUsername,
			settings.SMTPPassword, settings.FromEmail, settings.ToEmails)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "SMTP 设置已更新"})
	}
}

func testSMTP(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		settings, err := services.GetSMTPSettings(db)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		if !settings.Enabled {
			c.JSON(http.StatusBadRequest, gin.H{"error": "SMTP 未启用"})
			return
		}

		subject := "余额监控系统 - 测试邮件"
		body := "<h2>这是一封测试邮件</h2><p>如果您收到此邮件，说明 SMTP 配置正确。</p>"

		if err := services.SendEmail(settings, subject, body); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "测试邮件已发送"})
	}
}

func getMonitorSettings(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var settings database.MonitorSettings
		
		err := db.QueryRow(`
			SELECT id, cron_expression, alert_count
			FROM monitor_settings WHERE id = 1
		`).Scan(&settings.ID, &settings.CronExpression, &settings.AlertCount)
		
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, settings)
	}
}

func updateMonitorSettings(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var settings database.MonitorSettings
		if err := c.ShouldBindJSON(&settings); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		_, err := db.Exec(`
			UPDATE monitor_settings 
			SET cron_expression = ?, alert_count = ?, updated_at = CURRENT_TIMESTAMP
			WHERE id = 1
		`, settings.CronExpression, settings.AlertCount)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "监控设置已更新"})
	}
}

func getNotifications(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		limit := c.DefaultQuery("limit", "50")
		
		rows, err := db.Query(`
			SELECT n.id, n.provider_id, p.name, n.balance, n.message, n.sent_at
			FROM notifications n
			JOIN providers p ON n.provider_id = p.id
			ORDER BY n.sent_at DESC
			LIMIT ?
		`, limit)
		
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		defer rows.Close()

		var notifications []map[string]interface{}
		for rows.Next() {
			var id, providerID int
			var providerName, message string
			var balance float64
			var sentAt time.Time

			if err := rows.Scan(&id, &providerID, &providerName, &balance, &message, &sentAt); err != nil {
				continue
			}

			notifications = append(notifications, map[string]interface{}{
				"id":            id,
				"provider_id":   providerID,
				"provider_name": providerName,
				"balance":       balance,
				"message":       message,
				"sent_at":       sentAt.Format(time.RFC3339),
			})
		}

		c.JSON(http.StatusOK, notifications)
	}
}


// 记账相关 handler

func createAccountingRecord(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req models.AccountingCreateRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		record, err := services.CreateAccountingRecord(db, req)
		if err != nil {
			switch err.Error() {
			case "金额必须为正数", "供应商不存在":
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			default:
				c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器内部错误"})
			}
			return
		}

		c.JSON(http.StatusCreated, record)
	}
}

func getAccountingRecords(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		providerID := 0
		if pidStr := c.Query("provider_id"); pidStr != "" {
			pid, err := strconv.Atoi(pidStr)
			if err == nil {
				providerID = pid
			}
		}
		startDate := c.Query("start_date")
		endDate := c.Query("end_date")

		records, err := services.GetAccountingRecords(db, providerID, startDate, endDate)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器内部错误"})
			return
		}

		c.JSON(http.StatusOK, records)
	}
}

func updateAccountingRecord(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "无效的记录 ID"})
			return
		}

		var req models.AccountingUpdateRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		if err := services.UpdateAccountingRecord(db, id, req); err != nil {
			switch err.Error() {
			case "记账记录不存在":
				c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			case "金额必须为正数":
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			default:
				c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器内部错误"})
			}
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "更新成功"})
	}
}

func deleteAccountingRecord(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "无效的记录 ID"})
			return
		}

		if err := services.DeleteAccountingRecord(db, id); err != nil {
			switch err.Error() {
			case "记账记录不存在":
				c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			default:
				c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器内部错误"})
			}
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "删除成功"})
	}
}

func getProviderCostStats(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		startDate := c.Query("start_date")
		endDate := c.Query("end_date")

		summaries, totalAmount, err := services.GetProviderCostStats(db, startDate, endDate)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器内部错误"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"summaries":    summaries,
			"total_amount": totalAmount,
		})
	}
}

func getMonthlyCostStats(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		providerID := 0
		if pidStr := c.Query("provider_id"); pidStr != "" {
			pid, err := strconv.Atoi(pidStr)
			if err == nil {
				providerID = pid
			}
		}

		summaries, err := services.GetMonthlyCostStats(db, providerID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器内部错误"})
			return
		}

		c.JSON(http.StatusOK, summaries)
	}
}


// parseCurlHandler 解析 curl 命令
func parseCurlHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			CurlCommand string `json:"curl_command" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "curl 命令不能为空"})
			return
		}
		parsed, err := services.ParseCurl(req.CurlCommand)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, parsed)
	}
}

// testCustomQueryHandler 测试自定义查询
func testCustomQueryHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Method           string            `json:"method"`
			URL              string            `json:"url" binding:"required"`
			Headers          map[string]string `json:"headers"`
			BalanceFieldPath string            `json:"balance_field_path" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if req.Method == "" {
			req.Method = "GET"
		}
		if req.Headers == nil {
			req.Headers = make(map[string]string)
		}
		balance, rawResp, err := services.ExecuteCustomQuery(req.Method, req.URL, req.Headers, req.BalanceFieldPath)
		if err != nil {
			c.JSON(http.StatusOK, gin.H{
				"success":      false,
				"error":        err.Error(),
				"raw_response": rawResp,
			})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"success":           true,
			"extracted_balance": balance,
			"raw_response":      rawResp,
		})
	}
}

// getCustomQueryHandler 获取供应商的自定义查询配置
func getCustomQueryHandler(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "无效的供应商 ID"})
			return
		}
		cq, err := services.GetCustomQuery(db, id)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if cq == nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "自定义查询配置不存在"})
			return
		}
		// Parse headers JSON for response
		var headers map[string]string
		if err := json.Unmarshal([]byte(cq.Headers), &headers); err != nil {
			headers = make(map[string]string)
		}
		c.JSON(http.StatusOK, gin.H{
			"method":             cq.Method,
			"url":                cq.URL,
			"headers":            headers,
			"balance_field_path": cq.BalanceFieldPath,
		})
	}
}

// saveCustomQueryHandler 保存/更新供应商的自定义查询配置
func saveCustomQueryHandler(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "无效的供应商 ID"})
			return
		}
		var req struct {
			Method           string            `json:"method"`
			URL              string            `json:"url"`
			Headers          map[string]string `json:"headers"`
			BalanceFieldPath string            `json:"balance_field_path"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if req.Headers == nil {
			req.Headers = make(map[string]string)
		}
		if err := services.SaveCustomQuery(db, id, req.Method, req.URL, req.Headers, req.BalanceFieldPath); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "自定义查询配置已保存"})
	}
}

// deleteCustomQueryHandler 删除供应商的自定义查询配置
func deleteCustomQueryHandler(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "无效的供应商 ID"})
			return
		}
		if err := services.DeleteCustomQuery(db, id); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "自定义查询配置已删除"})
	}
}
