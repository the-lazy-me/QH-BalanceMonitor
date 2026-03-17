package router

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"balance-monitor/internal/services"

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
		{"POST", "/api/balance/consumption/history/jobs"},
		{"GET", "/api/balance/consumption/history/jobs/active"},
		{"GET", "/api/balance/consumption/history/jobs/1"},
		{"POST", "/api/balance/consumption/history/jobs/1/cancel"},
		{"POST", "/api/balance/consumption/history/backfill-day"},
		{"POST", "/api/providers/1/test"},
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
			price_ratio REAL DEFAULT 1.0 NOT NULL,
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
		CREATE TABLE provider_daily_consumption_history (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			provider_id INTEGER NOT NULL,
			date TEXT NOT NULL,
			quota_consumed INTEGER NOT NULL DEFAULT 0,
			request_count INTEGER NOT NULL DEFAULT 0,
			token_used INTEGER NOT NULL DEFAULT 0,
			fetched_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (provider_id) REFERENCES providers(id) ON DELETE CASCADE,
			UNIQUE(provider_id, date)
		);
		CREATE TABLE consumption_backfill_jobs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			status TEXT NOT NULL,
			days_limit INTEGER NOT NULL,
			created_by INTEGER NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			started_at DATETIME,
			finished_at DATETIME,
			FOREIGN KEY (created_by) REFERENCES users(id) ON DELETE RESTRICT
		);
		CREATE TABLE consumption_backfill_job_providers (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			job_id INTEGER NOT NULL,
			provider_id INTEGER NOT NULL,
			status TEXT NOT NULL,
			current_date TEXT,
			inserted_days INTEGER NOT NULL DEFAULT 0,
			skipped_days INTEGER NOT NULL DEFAULT 0,
			error_message TEXT,
			started_at DATETIME,
			finished_at DATETIME,
			FOREIGN KEY (job_id) REFERENCES consumption_backfill_jobs(id) ON DELETE CASCADE,
			FOREIGN KEY (provider_id) REFERENCES providers(id) ON DELETE CASCADE,
			UNIQUE(job_id, provider_id)
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

func authHeaderForTest(t *testing.T, db *sql.DB) string {
	t.Helper()

	if err := services.InitializeSystem(db, "admin", "password123"); err != nil && err.Error() != "系统已初始化" {
		t.Fatalf("initialize system failed: %v", err)
	}

	token, err := services.Login(db, "admin", "password123")
	if err != nil {
		t.Fatalf("login failed: %v", err)
	}

	return "Bearer " + token
}

func TestBackfillHistoryRejectsDisabledAndCustomProviders(t *testing.T) {
	db := setupRouterTestDB(t)
	defer db.Close()

	router := Setup(db)
	authHeader := authHeaderForTest(t, db)

	cases := []struct {
		name      string
		enabled   bool
		queryMode string
	}{
		{name: "disabled provider", enabled: false, queryMode: "default"},
		{name: "custom provider", enabled: true, queryMode: "custom"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := db.Exec(`
				INSERT INTO providers (name, upstream_url, authorization, new_api_user, enabled, query_mode, price_ratio)
				VALUES (?, ?, ?, ?, ?, ?, ?)
			`, tc.name, "https://example.com", "Bearer test", "1", tc.enabled, tc.queryMode, 1.0)
			if err != nil {
				t.Fatalf("insert provider failed: %v", err)
			}
			providerID, _ := result.LastInsertId()

			body, _ := json.Marshal(map[string]interface{}{
				"provider_id": providerID,
				"date":        "2026-03-17",
			})
			req := httptest.NewRequest(http.MethodPost, "/api/balance/consumption/history/backfill-day", bytes.NewReader(body))
			req.Header.Set("Authorization", authHeader)
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != http.StatusBadRequest {
				t.Fatalf("expected 400, got %d with body %s", w.Code, w.Body.String())
			}
		})
	}
}

func TestGetBalanceConsumptionRMBPrefersExternalHistory(t *testing.T) {
	db := setupRouterTestDB(t)
	defer db.Close()

	router := Setup(db)
	authHeader := authHeaderForTest(t, db)
	testDate := services.ShanghaiStartDate(1)

	result, err := db.Exec(`
		INSERT INTO providers (name, upstream_url, authorization, new_api_user, enabled, query_mode, price_ratio)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, "provider-a", "https://example.com", "Bearer test", "1", true, "default", 2.0)
	if err != nil {
		t.Fatalf("insert provider failed: %v", err)
	}
	providerID, _ := result.LastInsertId()

	_, err = db.Exec(`
		INSERT INTO provider_daily_consumption_history (provider_id, date, quota_consumed, request_count, token_used)
		VALUES (?, ?, ?, ?, ?)
	`, providerID, testDate, 500000, 10, 1000)
	if err != nil {
		t.Fatalf("insert external history failed: %v", err)
	}

	_, err = db.Exec(`
		INSERT INTO balance_history (provider_id, quota, balance, checked_at)
		VALUES
			(?, ?, ?, ?),
			(?, ?, ?, ?)
	`,
		providerID, 0, 100.0, testDate+"T00:00:00Z",
		providerID, 0, 95.0, testDate+"T12:00:00Z",
	)
	if err != nil {
		t.Fatalf("insert fallback history failed: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/balance/consumption/rmb?days=30", nil)
	req.Header.Set("Authorization", authHeader)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d with body %s", w.Code, w.Body.String())
	}

	var payload []struct {
		ProviderID  int     `json:"provider_id"`
		Consumption float64 `json:"consumption"`
		Date        string  `json:"date"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal response failed: %v", err)
	}

	if len(payload) != 1 {
		t.Fatalf("expected 1 row, got %d: %s", len(payload), w.Body.String())
	}
	if payload[0].ProviderID != int(providerID) || payload[0].Date != testDate {
		t.Fatalf("unexpected payload row: %+v", payload[0])
	}
	if payload[0].Consumption != 2.0 {
		t.Fatalf("expected external consumption 2.0, got %f", payload[0].Consumption)
	}
}

func TestGetBalanceConsumptionRMBFiltersByDateRange(t *testing.T) {
	db := setupRouterTestDB(t)
	defer db.Close()

	router := Setup(db)
	authHeader := authHeaderForTest(t, db)

	result, err := db.Exec(`
		INSERT INTO providers (name, upstream_url, authorization, new_api_user, enabled, query_mode, price_ratio)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, "provider-a", "https://example.com", "Bearer test", "1", true, "default", 1.0)
	if err != nil {
		t.Fatalf("insert provider failed: %v", err)
	}
	providerID, _ := result.LastInsertId()

	_, err = db.Exec(`
		INSERT INTO provider_daily_consumption_history (provider_id, date, quota_consumed, request_count, token_used)
		VALUES
			(?, ?, ?, ?, ?),
			(?, ?, ?, ?, ?),
			(?, ?, ?, ?, ?)
	`,
		providerID, "2026-03-10", 500000, 10, 1000,
		providerID, "2026-03-11", 600000, 11, 1100,
		providerID, "2026-03-12", 700000, 12, 1200,
	)
	if err != nil {
		t.Fatalf("insert external history failed: %v", err)
	}

	req := httptest.NewRequest(
		http.MethodGet,
		"/api/balance/consumption/rmb?start_date=2026-03-11&end_date=2026-03-12",
		nil,
	)
	req.Header.Set("Authorization", authHeader)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d with body %s", w.Code, w.Body.String())
	}

	var payload []struct {
		Date string `json:"date"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal response failed: %v", err)
	}

	if len(payload) != 2 {
		t.Fatalf("expected 2 rows, got %d: %s", len(payload), w.Body.String())
	}
	if payload[0].Date != "2026-03-11" || payload[1].Date != "2026-03-12" {
		t.Fatalf("unexpected payload rows: %+v", payload)
	}
}

func TestGetBalanceConsumptionRMBRejectsInvalidDateRange(t *testing.T) {
	db := setupRouterTestDB(t)
	defer db.Close()

	router := Setup(db)
	authHeader := authHeaderForTest(t, db)

	req := httptest.NewRequest(
		http.MethodGet,
		"/api/balance/consumption/rmb?start_date=2026-03-12&end_date=2026-03-11",
		nil,
	)
	req.Header.Set("Authorization", authHeader)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d with body %s", w.Code, w.Body.String())
	}
}

func TestCreateConsumptionBackfillJobReturnsConflictWhenActiveJobExists(t *testing.T) {
	db := setupRouterTestDB(t)
	defer db.Close()

	authHeader := authHeaderForTest(t, db)
	router := Setup(db)

	result, err := db.Exec(`
		INSERT INTO providers (name, upstream_url, authorization, new_api_user, enabled, query_mode, price_ratio)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, "provider-a", "https://example.com", "Bearer test", "1", true, "default", 1.0)
	if err != nil {
		t.Fatalf("insert provider failed: %v", err)
	}
	providerID, _ := result.LastInsertId()

	_, err = db.Exec(`
		INSERT INTO consumption_backfill_jobs (status, days_limit, created_by, created_at, started_at)
		VALUES (?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
	`, services.ConsumptionBackfillJobStatusRunning, services.DefaultConsumptionBackfillDaysLimit, 1)
	if err != nil {
		t.Fatalf("insert job failed: %v", err)
	}

	var jobID int
	if err := db.QueryRow(`SELECT id FROM consumption_backfill_jobs LIMIT 1`).Scan(&jobID); err != nil {
		t.Fatalf("query job id failed: %v", err)
	}

	_, err = db.Exec(`
		INSERT INTO consumption_backfill_job_providers (job_id, provider_id, status, current_date, inserted_days, skipped_days)
		VALUES (?, ?, ?, ?, ?, ?)
	`, jobID, providerID, services.ConsumptionBackfillProviderStatusRunning, services.ShanghaiStartDate(1), 3, 1)
	if err != nil {
		t.Fatalf("insert job provider failed: %v", err)
	}

	body, _ := json.Marshal(map[string]interface{}{
		"provider_ids": []int{int(providerID)},
	})
	req := httptest.NewRequest(http.MethodPost, "/api/balance/consumption/history/jobs", bytes.NewReader(body))
	req.Header.Set("Authorization", authHeader)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d with body %s", w.Code, w.Body.String())
	}
}

func TestGetActiveConsumptionBackfillJobAndJobDetail(t *testing.T) {
	db := setupRouterTestDB(t)
	defer db.Close()

	authHeader := authHeaderForTest(t, db)
	router := Setup(db)

	result, err := db.Exec(`
		INSERT INTO providers (name, upstream_url, authorization, new_api_user, enabled, query_mode, price_ratio)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, "provider-a", "https://example.com", "Bearer test", "1", true, "default", 1.0)
	if err != nil {
		t.Fatalf("insert provider failed: %v", err)
	}
	providerID, _ := result.LastInsertId()

	_, err = db.Exec(`
		INSERT INTO consumption_backfill_jobs (status, days_limit, created_by, created_at, started_at)
		VALUES (?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
	`, services.ConsumptionBackfillJobStatusRunning, services.DefaultConsumptionBackfillDaysLimit, 1)
	if err != nil {
		t.Fatalf("insert job failed: %v", err)
	}

	var jobID int
	if err := db.QueryRow(`SELECT id FROM consumption_backfill_jobs LIMIT 1`).Scan(&jobID); err != nil {
		t.Fatalf("query job id failed: %v", err)
	}

	_, err = db.Exec(`
		INSERT INTO consumption_backfill_job_providers (job_id, provider_id, status, current_date, inserted_days, skipped_days)
		VALUES (?, ?, ?, ?, ?, ?)
	`, jobID, providerID, services.ConsumptionBackfillProviderStatusRunning, services.ShanghaiStartDate(1), 5, 2)
	if err != nil {
		t.Fatalf("insert job provider failed: %v", err)
	}

	activeReq := httptest.NewRequest(http.MethodGet, "/api/balance/consumption/history/jobs/active", nil)
	activeReq.Header.Set("Authorization", authHeader)
	activeRes := httptest.NewRecorder()
	router.ServeHTTP(activeRes, activeReq)

	if activeRes.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d with body %s", activeRes.Code, activeRes.Body.String())
	}

	var activePayload struct {
		Job *struct {
			JobID        int `json:"job_id"`
			InsertedDays int `json:"inserted_days"`
			SkippedDays  int `json:"skipped_days"`
			Providers    []struct {
				ProviderID int    `json:"provider_id"`
				Status     string `json:"status"`
			} `json:"providers"`
		} `json:"job"`
	}
	if err := json.Unmarshal(activeRes.Body.Bytes(), &activePayload); err != nil {
		t.Fatalf("decode active payload failed: %v", err)
	}
	if activePayload.Job == nil {
		t.Fatalf("expected active job payload, got %s", activeRes.Body.String())
	}
	if activePayload.Job.JobID != jobID || activePayload.Job.InsertedDays != 5 || activePayload.Job.SkippedDays != 2 {
		t.Fatalf("unexpected active payload: %+v", activePayload.Job)
	}

	detailReq := httptest.NewRequest(http.MethodGet, "/api/balance/consumption/history/jobs/"+strconv.Itoa(jobID), nil)
	detailReq.Header.Set("Authorization", authHeader)
	detailRes := httptest.NewRecorder()
	router.ServeHTTP(detailRes, detailReq)

	if detailRes.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d with body %s", detailRes.Code, detailRes.Body.String())
	}

	var detailPayload struct {
		JobID     int `json:"job_id"`
		Providers []struct {
			ProviderID int    `json:"provider_id"`
			Status     string `json:"status"`
		} `json:"providers"`
	}
	if err := json.Unmarshal(detailRes.Body.Bytes(), &detailPayload); err != nil {
		t.Fatalf("decode detail payload failed: %v", err)
	}
	if detailPayload.JobID != jobID || len(detailPayload.Providers) != 1 {
		t.Fatalf("unexpected detail payload: %+v", detailPayload)
	}
}

func TestTestProviderEndpointDefaultMode(t *testing.T) {
	db := setupRouterTestDB(t)
	defer db.Close()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/user/self" {
			http.NotFound(w, r)
			return
		}

		if got := r.Header.Get("Authorization"); got != "Bearer test" {
			t.Fatalf("unexpected authorization header: %s", got)
		}
		if got := r.Header.Get("New-API-User"); got != "7" {
			t.Fatalf("unexpected New-API-User header: %s", got)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success":true,"message":"ok","data":{"quota":750000}}`))
	}))
	defer server.Close()

	result, err := db.Exec(`
		INSERT INTO providers (name, upstream_url, authorization, new_api_user, enabled, query_mode, price_ratio)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, "default-provider", server.URL, "Bearer test", "7", false, "default", 1.0)
	if err != nil {
		t.Fatalf("insert provider failed: %v", err)
	}
	providerID, _ := result.LastInsertId()

	router := Setup(db)
	authHeader := authHeaderForTest(t, db)
	req := httptest.NewRequest(http.MethodPost, "/api/providers/"+strconv.FormatInt(providerID, 10)+"/test", nil)
	req.Header.Set("Authorization", authHeader)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d with body %s", w.Code, w.Body.String())
	}

	var payload struct {
		Success      bool    `json:"success"`
		ProviderID   int64   `json:"provider_id"`
		ProviderName string  `json:"provider_name"`
		QueryMode    string  `json:"query_mode"`
		Quota        int64   `json:"quota"`
		Balance      float64 `json:"balance"`
		CheckedAt    string  `json:"checked_at"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}

	if !payload.Success {
		t.Fatalf("expected success response, got %+v", payload)
	}
	if payload.ProviderID != providerID || payload.ProviderName != "default-provider" {
		t.Fatalf("unexpected provider info: %+v", payload)
	}
	if payload.QueryMode != "default" || payload.Quota != 750000 || payload.Balance != 1.5 {
		t.Fatalf("unexpected balance payload: %+v", payload)
	}
	if payload.CheckedAt == "" {
		t.Fatalf("expected checked_at to be set")
	}
}

func TestTestProviderEndpointCustomMode(t *testing.T) {
	db := setupRouterTestDB(t)
	defer db.Close()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/wallet" {
			http.NotFound(w, r)
			return
		}

		if got := r.Header.Get("X-Token"); got != "abc" {
			t.Fatalf("unexpected X-Token header: %s", got)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"balance":23.5}}`))
	}))
	defer server.Close()

	result, err := db.Exec(`
		INSERT INTO providers (name, upstream_url, authorization, new_api_user, enabled, query_mode, price_ratio)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, "custom-provider", "https://unused.example.com", "Bearer test", "1", true, "custom", 1.0)
	if err != nil {
		t.Fatalf("insert provider failed: %v", err)
	}
	providerID, _ := result.LastInsertId()

	_, err = db.Exec(`
		INSERT INTO custom_queries (provider_id, method, url, headers, balance_field_path)
		VALUES (?, ?, ?, ?, ?)
	`, providerID, "GET", server.URL+"/wallet", `{"X-Token":"abc"}`, "data.balance")
	if err != nil {
		t.Fatalf("insert custom query failed: %v", err)
	}

	router := Setup(db)
	authHeader := authHeaderForTest(t, db)
	req := httptest.NewRequest(http.MethodPost, "/api/providers/"+strconv.FormatInt(providerID, 10)+"/test", nil)
	req.Header.Set("Authorization", authHeader)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d with body %s", w.Code, w.Body.String())
	}

	var payload struct {
		Success   bool    `json:"success"`
		QueryMode string  `json:"query_mode"`
		Quota     int64   `json:"quota"`
		Balance   float64 `json:"balance"`
		CheckedAt string  `json:"checked_at"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}

	if !payload.Success {
		t.Fatalf("expected success response, got %+v", payload)
	}
	if payload.QueryMode != "custom" || payload.Quota != 0 || payload.Balance != 23.5 {
		t.Fatalf("unexpected custom payload: %+v", payload)
	}
	if payload.CheckedAt == "" {
		t.Fatalf("expected checked_at to be set")
	}
}
