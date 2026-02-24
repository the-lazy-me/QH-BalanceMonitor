package router

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"pgregory.net/rapid"
)

// Feature: custom-curl-query, Property 7: 认证强制执行
// **Validates: Requirements 6.4, 7.1, 7.2**
func TestPropertyAuthEnforcement(t *testing.T) {
	db := setupRouterTestDB(t)
	defer db.Close()

	router := Setup(db)

	// All custom query endpoints that require auth
	endpoints := []struct {
		method string
		path   string
	}{
		{"POST", "/api/curl/parse"},
		{"POST", "/api/custom-query/test"},
		{"GET", "/api/providers/1/custom-query"},
		{"PUT", "/api/providers/1/custom-query"},
		{"DELETE", "/api/providers/1/custom-query"},
	}

	rapid.Check(t, func(t *rapid.T) {
		// Pick a random endpoint
		idx := rapid.IntRange(0, len(endpoints)-1).Draw(t, "endpointIdx")
		ep := endpoints[idx]

		// Create request WITHOUT auth token
		req, err := http.NewRequest(ep.method, ep.path, nil)
		if err != nil {
			t.Fatalf("create request failed: %v", err)
		}

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Should return 401
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401 for %s %s without auth, got %d", ep.method, ep.path, w.Code)
		}
	})
}

func setupRouterTestDB(t *testing.T) *sql.DB {
	db, err := sql.Open("sqlite3", ":memory:?_foreign_keys=ON")
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`
		CREATE TABLE users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			username TEXT NOT NULL UNIQUE,
			password_hash TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);
		CREATE TABLE providers (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			upstream_url TEXT NOT NULL,
			authorization TEXT NOT NULL,
			new_api_user TEXT NOT NULL,
			enabled BOOLEAN DEFAULT 1,
			alert_threshold REAL DEFAULT 10.0,
			query_mode TEXT DEFAULT 'default' NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);
		CREATE TABLE custom_queries (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			provider_id INTEGER NOT NULL UNIQUE,
			method TEXT NOT NULL DEFAULT 'GET',
			url TEXT NOT NULL,
			headers TEXT DEFAULT '{}',
			balance_field_path TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (provider_id) REFERENCES providers(id) ON DELETE CASCADE
		);
		CREATE TABLE balance_history (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			provider_id INTEGER NOT NULL,
			quota INTEGER NOT NULL,
			balance REAL NOT NULL,
			checked_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (provider_id) REFERENCES providers(id) ON DELETE CASCADE
		);
		CREATE TABLE smtp_settings (
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
		CREATE TABLE notifications (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			provider_id INTEGER NOT NULL,
			balance REAL NOT NULL,
			message TEXT NOT NULL,
			sent_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (provider_id) REFERENCES providers(id) ON DELETE CASCADE
		);
		CREATE TABLE monitor_settings (
			id INTEGER PRIMARY KEY CHECK (id = 1),
			cron_expression TEXT DEFAULT '@hourly',
			alert_count INTEGER DEFAULT 3,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);
		CREATE TABLE accounting_records (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			provider_id INTEGER NOT NULL,
			amount REAL NOT NULL CHECK (amount > 0),
			date TEXT NOT NULL,
			note TEXT DEFAULT '',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (provider_id) REFERENCES providers(id) ON DELETE CASCADE
		);
		INSERT OR IGNORE INTO smtp_settings (id, enabled) VALUES (1, 0);
		INSERT OR IGNORE INTO monitor_settings (id, cron_expression, alert_count) VALUES (1, '@hourly', 3);
	`)
	if err != nil {
		t.Fatal(err)
	}
	return db
}
