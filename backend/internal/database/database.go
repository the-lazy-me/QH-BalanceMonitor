package database

import (
	"database/sql"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

func Init(dbPath string) (*sql.DB, error) {
	// 启用 WAL 模式和其他优化参数
	// _journal_mode=WAL: 使用 Write-Ahead Logging 提高并发性能
	// _busy_timeout=5000: 在锁定时等待 5 秒
	// _synchronous=NORMAL: 平衡性能和安全性
	// _cache_size=1000: 缓存大小
	// _foreign_keys=ON: 启用外键约束
	dsn := dbPath + "?_journal_mode=WAL&_busy_timeout=5000&_synchronous=NORMAL&_cache_size=1000&_foreign_keys=ON"
	
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, err
	}

	// 设置连接池参数
	// 限制最大打开连接数，避免过多并发
	db.SetMaxOpenConns(1)
	// 设置最大空闲连接数
	db.SetMaxIdleConns(1)
	// 设置连接的最大生命周期
	db.SetConnMaxLifetime(time.Hour)

	if err := createTables(db); err != nil {
		return nil, err
	}

	return db, nil
}

func createTables(db *sql.DB) error {
	schema := `
	CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		username TEXT NOT NULL UNIQUE,
		password_hash TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS providers (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		upstream_url TEXT NOT NULL,
		authorization TEXT NOT NULL,
		new_api_user TEXT NOT NULL,
		enabled BOOLEAN DEFAULT 1,
		alert_threshold REAL DEFAULT 10.0,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS balance_history (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		provider_id INTEGER NOT NULL,
		quota INTEGER NOT NULL,
		balance REAL NOT NULL,
		checked_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (provider_id) REFERENCES providers(id) ON DELETE CASCADE
	);

	CREATE TABLE IF NOT EXISTS smtp_settings (
		id INTEGER PRIMARY KEY CHECK (id = 1),
		enabled BOOLEAN DEFAULT 0,
		smtp_host TEXT,
		smtp_port INTEGER,
		smtp_username TEXT,
		smtp_password TEXT,
		from_email TEXT,
		to_emails TEXT,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS notifications (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		provider_id INTEGER NOT NULL,
		balance REAL NOT NULL,
		message TEXT NOT NULL,
		sent_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (provider_id) REFERENCES providers(id) ON DELETE CASCADE
	);

	CREATE TABLE IF NOT EXISTS monitor_settings (
		id INTEGER PRIMARY KEY CHECK (id = 1),
		cron_expression TEXT DEFAULT '@hourly',
		alert_count INTEGER DEFAULT 3,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	-- 插入默认 SMTP 设置
	INSERT OR IGNORE INTO smtp_settings (id, enabled) VALUES (1, 0);
	
	-- 插入默认监控设置
	INSERT OR IGNORE INTO monitor_settings (id, cron_expression, alert_count) VALUES (1, '@hourly', 3);
	`

	_, err := db.Exec(schema)
	return err
}

type Provider struct {
	ID             int       `json:"id"`
	Name           string    `json:"name"`
	UpstreamURL    string    `json:"upstream_url"`
	Authorization  string    `json:"authorization"`
	NewAPIUser     string    `json:"new_api_user"`
	Enabled        bool      `json:"enabled"`
	AlertThreshold float64   `json:"alert_threshold"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type BalanceHistory struct {
	ID         int       `json:"id"`
	ProviderID int       `json:"provider_id"`
	Quota      int64     `json:"quota"`
	Balance    float64   `json:"balance"`
	CheckedAt  time.Time `json:"checked_at"`
}

type SMTPSettings struct {
	ID           int    `json:"id"`
	Enabled      bool   `json:"enabled"`
	SMTPHost     string `json:"smtp_host"`
	SMTPPort     int    `json:"smtp_port"`
	SMTPUsername string `json:"smtp_username"`
	SMTPPassword string `json:"smtp_password"`
	FromEmail    string `json:"from_email"`
	ToEmails     string `json:"to_emails"`
}

type Notification struct {
	ID         int       `json:"id"`
	ProviderID int       `json:"provider_id"`
	Balance    float64   `json:"balance"`
	Message    string    `json:"message"`
	SentAt     time.Time `json:"sent_at"`
}

type MonitorSettings struct {
	ID             int    `json:"id"`
	CronExpression string `json:"cron_expression"`
	AlertCount     int    `json:"alert_count"`
}

type User struct {
	ID           int       `json:"id"`
	Username     string    `json:"username"`
	PasswordHash string    `json:"-"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

