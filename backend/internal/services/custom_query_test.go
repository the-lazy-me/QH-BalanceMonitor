package services

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"pgregory.net/rapid"
)

// --- Helper Functions ---

func setupTestDB(t *testing.T) *sql.DB {
	db, err := sql.Open("sqlite3", ":memory:?_foreign_keys=ON")
	if err != nil {
		t.Fatal(err)
	}
	// Create tables
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

func createTestProvider(t *testing.T, db *sql.DB) int {
	result, err := db.Exec(`INSERT INTO providers (name, upstream_url, authorization, new_api_user) VALUES ('test', 'https://test.com', 'Bearer test', '1')`)
	if err != nil {
		t.Fatal(err)
	}
	id, _ := result.LastInsertId()
	return int(id)
}

// --- Task 3.4: ExtractBalance Unit Tests ---

func TestExtractBalance_Normal(t *testing.T) {
	data := []byte(`{"balance": 100.50}`)
	val, err := ExtractBalance(data, "balance")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != 100.50 {
		t.Errorf("expected 100.50, got %f", val)
	}
}

func TestExtractBalance_NestedPath(t *testing.T) {
	data := []byte(`{"data": {"balance": 200.75}}`)
	val, err := ExtractBalance(data, "data.balance")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != 200.75 {
		t.Errorf("expected 200.75, got %f", val)
	}
}

func TestExtractBalance_StringNumber(t *testing.T) {
	data := []byte(`{"credit": "150.25"}`)
	val, err := ExtractBalance(data, "credit")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != 150.25 {
		t.Errorf("expected 150.25, got %f", val)
	}
}

func TestExtractBalance_PathNotFound(t *testing.T) {
	data := []byte(`{"data": {"other": 100}}`)
	_, err := ExtractBalance(data, "data.balance")
	if err == nil {
		t.Fatal("expected error for path not found, got nil")
	}
	if !strings.Contains(err.Error(), "不存在") {
		t.Errorf("expected error about path not existing, got: %s", err.Error())
	}
}

func TestExtractBalance_ValueNotNumber(t *testing.T) {
	data := []byte(`{"balance": "not-a-number"}`)
	_, err := ExtractBalance(data, "balance")
	if err == nil {
		t.Fatal("expected error for non-numeric value, got nil")
	}
	if !strings.Contains(err.Error(), "数字") {
		t.Errorf("expected error about number conversion, got: %s", err.Error())
	}
}

func TestExtractBalance_EmptyJSON(t *testing.T) {
	_, err := ExtractBalance([]byte{}, "balance")
	if err == nil {
		t.Fatal("expected error for empty JSON, got nil")
	}
}

// --- Task 3.5: [PBT] Property 2 - JSON Balance Field Extraction ---

// Feature: custom-curl-query, Property 2: JSON 余额字段提取
// **Validates: Requirements 3.3, 3.4, 3.5**
func TestPropertyJSONBalanceExtraction(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate a random numeric value
		value := rapid.Float64Range(-1e9, 1e9).Draw(t, "value")
		// Avoid NaN/Inf
		if math.IsNaN(value) || math.IsInf(value, 0) {
			value = 42.0
		}

		// Generate random nesting depth (1-4 levels)
		depth := rapid.IntRange(1, 4).Draw(t, "depth")
		segments := make([]string, depth)
		for i := 0; i < depth; i++ {
			segments[i] = rapid.StringMatching(`[a-z][a-z0-9_]{0,7}`).Draw(t, fmt.Sprintf("seg%d", i))
		}
		fieldPath := strings.Join(segments, ".")

		// Build nested JSON with the numeric value
		jsonObj := buildNestedJSON(segments, value)
		data, err := json.Marshal(jsonObj)
		if err != nil {
			t.Fatalf("failed to marshal JSON: %v", err)
		}

		// Extract and verify
		extracted, err := ExtractBalance(data, fieldPath)
		if err != nil {
			t.Fatalf("ExtractBalance failed: %v\nJSON: %s\nPath: %s", err, string(data), fieldPath)
		}

		if math.Abs(extracted-value) > 1e-6 {
			t.Fatalf("value mismatch: expected %f, got %f", value, extracted)
		}

		// Also test with string representation of the number
		jsonObjStr := buildNestedJSON(segments, fmt.Sprintf("%g", value))
		dataStr, err := json.Marshal(jsonObjStr)
		if err != nil {
			t.Fatalf("failed to marshal string JSON: %v", err)
		}

		extractedStr, err := ExtractBalance(dataStr, fieldPath)
		if err != nil {
			t.Fatalf("ExtractBalance (string) failed: %v\nJSON: %s\nPath: %s", err, string(dataStr), fieldPath)
		}

		if math.Abs(extractedStr-value) > 1e-6 {
			t.Fatalf("string value mismatch: expected %f, got %f", value, extractedStr)
		}
	})
}

// buildNestedJSON creates a nested map with the given value at the leaf.
func buildNestedJSON(segments []string, value interface{}) map[string]interface{} {
	if len(segments) == 1 {
		return map[string]interface{}{segments[0]: value}
	}
	return map[string]interface{}{segments[0]: buildNestedJSON(segments[1:], value)}
}

// --- Task 3.6: [PBT] Property 3 - Custom Query Config Persistence Round-Trip ---

// Feature: custom-curl-query, Property 3: 自定义查询配置持久化 Round-Trip
// **Validates: Requirements 2.4, 2.5, 2.6, 6.1**
func TestPropertyCustomQueryPersistenceRoundTrip(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	providerID := createTestProvider(t, db)

	rapid.Check(t, func(t *rapid.T) {
		methods := []string{"GET", "POST", "PUT", "DELETE", "PATCH"}
		method := rapid.SampledFrom(methods).Draw(t, "method")

		url := "https://" + rapid.StringMatching(`[a-z][a-z0-9]{1,10}\.[a-z]{2,4}`).Draw(t, "host") +
			rapid.StringMatching(`(/[a-z0-9]{1,8}){0,3}`).Draw(t, "path")

		// Generate random headers (0-3)
		numHeaders := rapid.IntRange(0, 3).Draw(t, "numHeaders")
		headers := make(map[string]string)
		for i := 0; i < numHeaders; i++ {
			key := rapid.StringMatching(`[A-Z][a-zA-Z-]{1,12}`).Draw(t, fmt.Sprintf("hKey%d", i))
			val := rapid.StringMatching(`[a-zA-Z0-9 ._/-]{1,20}`).Draw(t, fmt.Sprintf("hVal%d", i))
			headers[key] = val
		}

		balanceFieldPath := rapid.StringMatching(`[a-z][a-z0-9_]{0,5}(\.[a-z][a-z0-9_]{0,5}){0,3}`).Draw(t, "fieldPath")

		// Save
		err := SaveCustomQuery(db, providerID, method, url, headers, balanceFieldPath)
		if err != nil {
			t.Fatalf("SaveCustomQuery failed: %v", err)
		}

		// Read back
		cq, err := GetCustomQuery(db, providerID)
		if err != nil {
			t.Fatalf("GetCustomQuery failed: %v", err)
		}
		if cq == nil {
			t.Fatal("GetCustomQuery returned nil")
		}

		// Verify fields match
		if cq.Method != method {
			t.Fatalf("method mismatch: expected %s, got %s", method, cq.Method)
		}
		if cq.URL != url {
			t.Fatalf("URL mismatch: expected %s, got %s", url, cq.URL)
		}
		if cq.BalanceFieldPath != balanceFieldPath {
			t.Fatalf("balanceFieldPath mismatch: expected %s, got %s", balanceFieldPath, cq.BalanceFieldPath)
		}

		// Verify headers (stored as JSON string)
		var storedHeaders map[string]string
		if err := json.Unmarshal([]byte(cq.Headers), &storedHeaders); err != nil {
			t.Fatalf("failed to unmarshal stored headers: %v", err)
		}
		if len(storedHeaders) != len(headers) {
			t.Fatalf("headers count mismatch: expected %d, got %d", len(headers), len(storedHeaders))
		}
		for k, v := range headers {
			if storedHeaders[k] != v {
				t.Fatalf("header %q mismatch: expected %q, got %q", k, v, storedHeaders[k])
			}
		}
	})
}

// --- Task 3.7: [PBT] Property 4 - Required Field Validation ---

// Feature: custom-curl-query, Property 4: 必填字段验证
// **Validates: Requirements 2.3, 2.8**
func TestPropertyRequiredFieldValidation(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	providerID := createTestProvider(t, db)

	rapid.Check(t, func(t *rapid.T) {
		method := rapid.SampledFrom([]string{"GET", "POST", "PUT", "DELETE"}).Draw(t, "method")
		headers := map[string]string{}

		// Decide which field to make empty: 0 = URL empty, 1 = balanceFieldPath empty, 2 = both empty
		emptyChoice := rapid.IntRange(0, 2).Draw(t, "emptyChoice")

		var url, balanceFieldPath string
		switch emptyChoice {
		case 0:
			url = rapid.SampledFrom([]string{"", "  ", "\t"}).Draw(t, "emptyURL")
			balanceFieldPath = rapid.StringMatching(`[a-z][a-z0-9_]{1,8}`).Draw(t, "validPath")
		case 1:
			url = "https://" + rapid.StringMatching(`[a-z]{3,8}\.[a-z]{2,3}`).Draw(t, "validURL")
			balanceFieldPath = rapid.SampledFrom([]string{"", "  ", "\t"}).Draw(t, "emptyPath")
		case 2:
			url = rapid.SampledFrom([]string{"", "  "}).Draw(t, "emptyURL")
			balanceFieldPath = rapid.SampledFrom([]string{"", "  "}).Draw(t, "emptyPath")
		}

		err := SaveCustomQuery(db, providerID, method, url, headers, balanceFieldPath)
		if err == nil {
			t.Fatalf("expected error for empty required field (choice=%d, url=%q, path=%q), got nil",
				emptyChoice, url, balanceFieldPath)
		}
	})
}

// --- Task 3.8: [PBT] Property 6 - Provider-CustomQuery One-to-One Constraint ---

// Feature: custom-curl-query, Property 6: Provider-CustomQuery 一对一约束
// **Validates: Requirements 6.2**
func TestPropertyOneToOneConstraint(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	providerID := createTestProvider(t, db)

	rapid.Check(t, func(t *rapid.T) {
		// Save multiple different configs for the same provider
		numSaves := rapid.IntRange(2, 5).Draw(t, "numSaves")

		var lastMethod, lastURL, lastPath string
		for i := 0; i < numSaves; i++ {
			lastMethod = rapid.SampledFrom([]string{"GET", "POST", "PUT", "DELETE"}).Draw(t, fmt.Sprintf("method%d", i))
			lastURL = "https://" + rapid.StringMatching(`[a-z]{3,8}\.[a-z]{2,3}`).Draw(t, fmt.Sprintf("url%d", i)) +
				rapid.StringMatching(`(/[a-z]{1,5}){0,2}`).Draw(t, fmt.Sprintf("path%d", i))
			lastPath = rapid.StringMatching(`[a-z][a-z0-9_]{0,5}`).Draw(t, fmt.Sprintf("fieldPath%d", i))

			err := SaveCustomQuery(db, providerID, lastMethod, lastURL, map[string]string{}, lastPath)
			if err != nil {
				t.Fatalf("SaveCustomQuery #%d failed: %v", i, err)
			}
		}

		// Verify only one record exists for this provider
		var count int
		err := db.QueryRow("SELECT COUNT(*) FROM custom_queries WHERE provider_id = ?", providerID).Scan(&count)
		if err != nil {
			t.Fatalf("count query failed: %v", err)
		}
		if count != 1 {
			t.Fatalf("expected 1 record for provider %d, got %d", providerID, count)
		}

		// Verify it's the latest config
		cq, err := GetCustomQuery(db, providerID)
		if err != nil {
			t.Fatalf("GetCustomQuery failed: %v", err)
		}
		if cq.Method != lastMethod {
			t.Fatalf("expected last method %s, got %s", lastMethod, cq.Method)
		}
		if cq.URL != lastURL {
			t.Fatalf("expected last URL %s, got %s", lastURL, cq.URL)
		}
		if cq.BalanceFieldPath != lastPath {
			t.Fatalf("expected last path %s, got %s", lastPath, cq.BalanceFieldPath)
		}
	})
}

// --- Task 3.9: Cascade Delete and Mode Switch Tests ---

func TestCascadeDelete(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	providerID := createTestProvider(t, db)

	// Save a custom query
	err := SaveCustomQuery(db, providerID, "GET", "https://api.example.com/balance", map[string]string{"Authorization": "Bearer token"}, "data.balance")
	if err != nil {
		t.Fatalf("SaveCustomQuery failed: %v", err)
	}

	// Verify it exists
	cq, err := GetCustomQuery(db, providerID)
	if err != nil {
		t.Fatalf("GetCustomQuery failed: %v", err)
	}
	if cq == nil {
		t.Fatal("expected custom query to exist")
	}

	// Delete the provider
	_, err = db.Exec("DELETE FROM providers WHERE id = ?", providerID)
	if err != nil {
		t.Fatalf("delete provider failed: %v", err)
	}

	// Verify custom query is also deleted (cascade)
	cq, err = GetCustomQuery(db, providerID)
	if err != nil {
		t.Fatalf("GetCustomQuery after cascade failed: %v", err)
	}
	if cq != nil {
		t.Fatal("expected custom query to be deleted after provider deletion (cascade)")
	}
}

func TestModeSwitchPreservesData(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	providerID := createTestProvider(t, db)

	// Set query_mode to 'custom' and save a custom query
	_, err := db.Exec("UPDATE providers SET query_mode = 'custom' WHERE id = ?", providerID)
	if err != nil {
		t.Fatalf("update query_mode failed: %v", err)
	}

	err = SaveCustomQuery(db, providerID, "POST", "https://api.example.com/credit", map[string]string{"X-Key": "abc"}, "credit.amount")
	if err != nil {
		t.Fatalf("SaveCustomQuery failed: %v", err)
	}

	// Switch query_mode back to 'default'
	_, err = db.Exec("UPDATE providers SET query_mode = 'default' WHERE id = ?", providerID)
	if err != nil {
		t.Fatalf("update query_mode to default failed: %v", err)
	}

	// Verify custom query config still exists in DB
	cq, err := GetCustomQuery(db, providerID)
	if err != nil {
		t.Fatalf("GetCustomQuery after mode switch failed: %v", err)
	}
	if cq == nil {
		t.Fatal("expected custom query to still exist after switching to default mode")
	}
	if cq.Method != "POST" {
		t.Errorf("expected method POST, got %s", cq.Method)
	}
	if cq.URL != "https://api.example.com/credit" {
		t.Errorf("expected URL https://api.example.com/credit, got %s", cq.URL)
	}
	if cq.BalanceFieldPath != "credit.amount" {
		t.Errorf("expected balance_field_path credit.amount, got %s", cq.BalanceFieldPath)
	}
}
