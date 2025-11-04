package router

import (
	"database/sql"
	"net/http"
	"time"

	"balance-monitor/internal/database"
	"balance-monitor/internal/middleware"
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
			SELECT id, name, upstream_url, authorization, new_api_user, enabled, alert_threshold, created_at, updated_at
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
			if err := rows.Scan(&p.ID, &p.Name, &p.UpstreamURL, &p.Authorization, &p.NewAPIUser, &p.Enabled, &p.AlertThreshold, &p.CreatedAt, &p.UpdatedAt); err != nil {
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

		result, err := db.Exec(`
			INSERT INTO providers (name, upstream_url, authorization, new_api_user, enabled, alert_threshold)
			VALUES (?, ?, ?, ?, ?, ?)
		`, provider.Name, provider.UpstreamURL, provider.Authorization, provider.NewAPIUser, provider.Enabled, provider.AlertThreshold)

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

		_, err := db.Exec(`
			UPDATE providers 
			SET name = ?, upstream_url = ?, authorization = ?, new_api_user = ?, enabled = ?, alert_threshold = ?, updated_at = CURRENT_TIMESTAMP
			WHERE id = ?
		`, provider.Name, provider.UpstreamURL, provider.Authorization, provider.NewAPIUser, provider.Enabled, provider.AlertThreshold, id)

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
			SELECT p.id, p.name, bh.quota, bh.balance, bh.checked_at
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
			var quota sql.NullInt64
			var balance sql.NullFloat64
			var checkedAt sql.NullTime

			if err := rows.Scan(&id, &name, &quota, &balance, &checkedAt); err != nil {
				continue
			}

			item := map[string]interface{}{
				"provider_id":   id,
				"provider_name": name,
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

		query := `
			SELECT bh.id, bh.provider_id, p.name, bh.quota, bh.balance, bh.checked_at
			FROM balance_history bh
			JOIN providers p ON bh.provider_id = p.id
		`

		var rows *sql.Rows
		var err error

		if providerID != "" {
			query += " WHERE bh.provider_id = ? ORDER BY bh.checked_at DESC LIMIT ?"
			rows, err = db.Query(query, providerID, limit)
		} else {
			query += " ORDER BY bh.checked_at DESC LIMIT ?"
			rows, err = db.Query(query, limit)
		}

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

