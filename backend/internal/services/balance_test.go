package services

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"balance-monitor/internal/database"

	_ "github.com/mattn/go-sqlite3"
	"pgregory.net/rapid"
)

// Feature: custom-curl-query, Property 5: 查询模式分发
// **Validates: Requirements 3.1, 3.2**
func TestPropertyQueryModeDispatch(t *testing.T) {
	// Create test HTTP server with different endpoints for default vs custom mode
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/user/self":
			// Default mode response (New API format)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": true,
				"data": map[string]interface{}{
					"quota": 5000000,
				},
			})
		case "/custom/balance":
			// Custom mode response
			json.NewEncoder(w).Encode(map[string]interface{}{
				"credit": 99.99,
			})
		default:
			w.WriteHeader(404)
		}
	}))
	defer server.Close()

	rapid.Check(t, func(rt *rapid.T) {
		db := setupBalanceTestDB(t)
		defer db.Close()

		queryMode := rapid.SampledFrom([]string{"default", "custom"}).Draw(rt, "queryMode")

		// Create provider pointing to the test server
		result, err := db.Exec(
			`INSERT INTO providers (name, upstream_url, authorization, new_api_user, query_mode) VALUES (?, ?, ?, ?, ?)`,
			"test-provider", server.URL, "Bearer test", "1", queryMode,
		)
		if err != nil {
			rt.Fatalf("create provider failed: %v", err)
		}
		providerID, _ := result.LastInsertId()

		if queryMode == "custom" {
			// Create custom query config pointing to the test server's custom endpoint
			err = SaveCustomQuery(db, int(providerID), "GET", server.URL+"/custom/balance", map[string]string{}, "credit")
			if err != nil {
				rt.Fatalf("save custom query failed: %v", err)
			}
		}

		// Build provider struct
		provider := database.Provider{
			ID:            int(providerID),
			Name:          "test-provider",
			UpstreamURL:   server.URL,
			Authorization: "Bearer test",
			NewAPIUser:    "1",
			QueryMode:     queryMode,
		}

		// Call fetchBalance and verify dispatch behavior
		quota, balance, err := fetchBalance(db, &provider)
		if err != nil {
			rt.Fatalf("fetchBalance failed for mode %s: %v", queryMode, err)
		}

		if queryMode == "default" {
			// Default mode: quota should be 5000000, balance = 5000000/500000 = 10.0
			if quota != 5000000 {
				rt.Fatalf("default mode: expected quota 5000000, got %d", quota)
			}
			if balance != 10.0 {
				rt.Fatalf("default mode: expected balance 10.0, got %f", balance)
			}
		} else {
			// Custom mode: quota should be 0, balance should be 99.99
			if quota != 0 {
				rt.Fatalf("custom mode: expected quota 0, got %d", quota)
			}
			if balance != 99.99 {
				rt.Fatalf("custom mode: expected balance 99.99, got %f", balance)
			}
		}
	})
}

// setupBalanceTestDB creates an in-memory SQLite database with providers and custom_queries tables.
func setupBalanceTestDB(t testing.TB) *sql.DB {
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
	`)
	if err != nil {
		t.Fatal(err)
	}
	return db
}
