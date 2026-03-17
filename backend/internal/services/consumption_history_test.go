package services

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

func TestBackfillProviderDailyConsumptionHistoryAggregatesAndSkips(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/data/self" {
			http.NotFound(w, r)
			return
		}

		if err := json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"message": "",
			"data": []map[string]interface{}{
				{"quota": 500000, "count": 3, "token_used": 1000},
				{"quota": 250000, "count": 2, "token_used": 400},
			},
		}); err != nil {
			t.Fatalf("encode response failed: %v", err)
		}
	}))
	defer server.Close()

	db := setupConsumptionHistoryTestDB(t)
	defer db.Close()

	providerID := insertBackfillProvider(t, db, "provider-a", server.URL, true, "default", 2.0)

	result, err := BackfillProviderDailyConsumptionHistory(db, providerID, "2026-03-17")
	if err != nil {
		t.Fatalf("backfill failed: %v", err)
	}

	if result.Status != "inserted" {
		t.Fatalf("expected inserted status, got %s", result.Status)
	}
	if result.QuotaConsumed != 750000 || result.RequestCount != 5 || result.TokenUsed != 1400 {
		t.Fatalf("unexpected aggregation result: %+v", result)
	}
	if result.RMBConsumed != 3.0 {
		t.Fatalf("expected rmb consumption 3.0, got %f", result.RMBConsumed)
	}

	var quotaConsumed, requestCount, tokenUsed int64
	err = db.QueryRow(`
		SELECT quota_consumed, request_count, token_used
		FROM provider_daily_consumption_history
		WHERE provider_id = ? AND date = ?
	`, providerID, "2026-03-17").Scan(&quotaConsumed, &requestCount, &tokenUsed)
	if err != nil {
		t.Fatalf("query stored history failed: %v", err)
	}
	if quotaConsumed != 750000 || requestCount != 5 || tokenUsed != 1400 {
		t.Fatalf("unexpected stored history: quota=%d count=%d token=%d", quotaConsumed, requestCount, tokenUsed)
	}

	skipped, err := BackfillProviderDailyConsumptionHistory(db, providerID, "2026-03-17")
	if err != nil {
		t.Fatalf("second backfill failed: %v", err)
	}
	if skipped.Status != "skipped" {
		t.Fatalf("expected skipped status, got %s", skipped.Status)
	}
	if skipped.QuotaConsumed != 750000 || skipped.RequestCount != 5 || skipped.TokenUsed != 1400 {
		t.Fatalf("unexpected skipped result: %+v", skipped)
	}
}

func TestBackfillProviderDailyConsumptionHistoryStoresEmptyDays(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"message": "",
			"data":    []map[string]interface{}{},
		}); err != nil {
			t.Fatalf("encode response failed: %v", err)
		}
	}))
	defer server.Close()

	db := setupConsumptionHistoryTestDB(t)
	defer db.Close()

	providerID := insertBackfillProvider(t, db, "provider-b", server.URL, true, "default", 1.5)

	result, err := BackfillProviderDailyConsumptionHistory(db, providerID, "2026-03-16")
	if err != nil {
		t.Fatalf("backfill failed: %v", err)
	}

	if result.Status != "inserted" {
		t.Fatalf("expected inserted status, got %s", result.Status)
	}
	if result.QuotaConsumed != 0 || result.RequestCount != 0 || result.TokenUsed != 0 || result.RMBConsumed != 0 {
		t.Fatalf("expected empty-day zero values, got %+v", result)
	}
}

func setupConsumptionHistoryTestDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := sql.Open("sqlite3", ":memory:?_foreign_keys=ON")
	if err != nil {
		t.Fatal(err)
	}

	_, err = db.Exec(`
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
	`)
	if err != nil {
		t.Fatal(err)
	}

	return db
}

func insertBackfillProvider(t *testing.T, db *sql.DB, name, upstreamURL string, enabled bool, queryMode string, priceRatio float64) int {
	t.Helper()

	result, err := db.Exec(`
		INSERT INTO providers (name, upstream_url, authorization, new_api_user, enabled, query_mode, price_ratio)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, name, upstreamURL, "Bearer test", "1", enabled, queryMode, priceRatio)
	if err != nil {
		t.Fatalf("insert provider failed: %v", err)
	}

	providerID, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("get provider id failed: %v", err)
	}

	return int(providerID)
}
