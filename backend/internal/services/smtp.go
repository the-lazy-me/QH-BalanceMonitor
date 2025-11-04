package services

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"balance-monitor/internal/database"
	"gopkg.in/gomail.v2"
)

func GetSMTPSettings(db *sql.DB) (*database.SMTPSettings, error) {
	var settings database.SMTPSettings
	
	err := db.QueryRow(`
		SELECT id, 
		       COALESCE(enabled, 0), 
		       COALESCE(smtp_host, ''), 
		       COALESCE(smtp_port, 587), 
		       COALESCE(smtp_username, ''), 
		       COALESCE(smtp_password, ''), 
		       COALESCE(from_email, ''), 
		       COALESCE(to_emails, '')
		FROM smtp_settings WHERE id = 1
	`).Scan(
		&settings.ID,
		&settings.Enabled,
		&settings.SMTPHost,
		&settings.SMTPPort,
		&settings.SMTPUsername,
		&settings.SMTPPassword,
		&settings.FromEmail,
		&settings.ToEmails,
	)
	if err != nil {
		return nil, err
	}
	return &settings, nil
}

func SendEmail(settings *database.SMTPSettings, subject, body string) error {
	m := gomail.NewMessage()
	m.SetHeader("From", settings.FromEmail)
	
	// 解析收件人列表
	recipients := strings.Split(settings.ToEmails, ",")
	for i, recipient := range recipients {
		recipients[i] = strings.TrimSpace(recipient)
	}
	m.SetHeader("To", recipients...)
	m.SetHeader("Subject", subject)
	m.SetBody("text/html", body)

	d := gomail.NewDialer(settings.SMTPHost, settings.SMTPPort, settings.SMTPUsername, settings.SMTPPassword)
	
	// 根据端口号配置 SSL/TLS
	// 端口 465 使用 SSL，端口 587 使用 STARTTLS
	if settings.SMTPPort == 465 {
		d.SSL = true
	}

	if err := d.DialAndSend(m); err != nil {
		return fmt.Errorf("发送邮件失败: %v", err)
	}

	return nil
}

func CheckAndSendAlert(db *sql.DB, provider *database.Provider, balance float64) error {
	// 获取 SMTP 设置
	settings, err := GetSMTPSettings(db)
	if err != nil || !settings.Enabled {
		return nil // SMTP 未启用，不发送告警
	}

	// 检查余额是否低于供应商设置的阈值
	if balance >= provider.AlertThreshold {
		return nil // 余额充足，不需要告警
	}

	// 获取监控设置中的告警次数
	var alertCount int
	err = db.QueryRow("SELECT alert_count FROM monitor_settings WHERE id = 1").Scan(&alertCount)
	if err != nil {
		alertCount = 3 // 默认 3 次
	}

	// 检查最近的告警次数
	var recentAlertCount int
	err = db.QueryRow(`
		SELECT COUNT(*) FROM notifications 
		WHERE provider_id = ? 
		AND sent_at > datetime('now', '-24 hours')
	`, provider.ID).Scan(&recentAlertCount)
	
	if err == nil && recentAlertCount >= alertCount {
		return nil // 24 小时内已达到告警次数上限
	}

	// 构建邮件内容
	subject := fmt.Sprintf("⚠️ 余额告警：%s", provider.Name)
	body := fmt.Sprintf(`
		<h2>余额告警通知</h2>
		<p><strong>供应商：</strong>%s</p>
		<p><strong>当前余额：</strong>%.2f</p>
		<p><strong>告警阈值：</strong>%.2f</p>
		<p><strong>检查时间：</strong>%s</p>
		<p><strong>24小时内告警次数：</strong>%d/%d</p>
		<hr>
		<p style="color: red;">余额已低于设定的告警阈值，请及时充值！</p>
	`, provider.Name, balance, provider.AlertThreshold, time.Now().Format("2006-01-02 15:04:05"), recentAlertCount+1, alertCount)

	// 发送邮件
	if err := SendEmail(settings, subject, body); err != nil {
		return err
	}

	// 记录通知
	message := fmt.Sprintf("余额 %.2f 低于阈值 %.2f", balance, provider.AlertThreshold)
	_, err = db.Exec(`
		INSERT INTO notifications (provider_id, balance, message)
		VALUES (?, ?, ?)
	`, provider.ID, balance, message)

	return err
}

