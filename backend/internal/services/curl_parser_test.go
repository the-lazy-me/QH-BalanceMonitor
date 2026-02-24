package services

import (
	"strings"
	"testing"

	"pgregory.net/rapid"
)

// --- Unit Tests (Task 2.3) ---

func TestParseCurl_StandardCurl(t *testing.T) {
	input := `curl -X POST https://api.example.com/balance -H 'Authorization: Bearer token123' -H 'Content-Type: application/json'`
	parsed, err := ParseCurl(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if parsed.Method != "POST" {
		t.Errorf("expected method POST, got %s", parsed.Method)
	}
	if parsed.URL != "https://api.example.com/balance" {
		t.Errorf("expected URL https://api.example.com/balance, got %s", parsed.URL)
	}
	if parsed.Headers["Authorization"] != "Bearer token123" {
		t.Errorf("expected Authorization header 'Bearer token123', got %q", parsed.Headers["Authorization"])
	}
	if parsed.Headers["Content-Type"] != "application/json" {
		t.Errorf("expected Content-Type header 'application/json', got %q", parsed.Headers["Content-Type"])
	}
}

func TestParseCurl_URLOnly(t *testing.T) {
	input := `curl https://example.com`
	parsed, err := ParseCurl(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if parsed.Method != "GET" {
		t.Errorf("expected default method GET, got %s", parsed.Method)
	}
	if parsed.URL != "https://example.com" {
		t.Errorf("expected URL https://example.com, got %s", parsed.URL)
	}
	if len(parsed.Headers) != 0 {
		t.Errorf("expected no headers, got %v", parsed.Headers)
	}
}

func TestParseCurl_MultipleHeaders(t *testing.T) {
	input := `curl -X GET https://api.test.com -H "Accept: text/html" -H "X-Custom: value1" -H "X-Another: value2"`
	parsed, err := ParseCurl(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(parsed.Headers) != 3 {
		t.Errorf("expected 3 headers, got %d", len(parsed.Headers))
	}
	if parsed.Headers["Accept"] != "text/html" {
		t.Errorf("expected Accept header 'text/html', got %q", parsed.Headers["Accept"])
	}
	if parsed.Headers["X-Custom"] != "value1" {
		t.Errorf("expected X-Custom header 'value1', got %q", parsed.Headers["X-Custom"])
	}
	if parsed.Headers["X-Another"] != "value2" {
		t.Errorf("expected X-Another header 'value2', got %q", parsed.Headers["X-Another"])
	}
}

func TestParseCurl_MissingURL(t *testing.T) {
	input := `curl -X POST -H 'Authorization: Bearer token'`
	_, err := ParseCurl(input)
	if err == nil {
		t.Fatal("expected error for missing URL, got nil")
	}
	if !strings.Contains(err.Error(), "URL") {
		t.Errorf("expected error message to mention URL, got: %s", err.Error())
	}
}

func TestParseCurl_DefaultGETMethod(t *testing.T) {
	input := `curl https://example.com/api -H 'Accept: application/json'`
	parsed, err := ParseCurl(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if parsed.Method != "GET" {
		t.Errorf("expected default method GET, got %s", parsed.Method)
	}
}

func TestParseCurl_QuotedURLAndHeaders(t *testing.T) {
	input := `curl -X GET "https://example.com/path?key=value" -H "Authorization: Bearer abc123"`
	parsed, err := ParseCurl(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if parsed.URL != "https://example.com/path?key=value" {
		t.Errorf("expected URL with query params, got %s", parsed.URL)
	}
	if parsed.Headers["Authorization"] != "Bearer abc123" {
		t.Errorf("expected Authorization header, got %q", parsed.Headers["Authorization"])
	}
}

func TestParseCurl_LongFormFlags(t *testing.T) {
	input := `curl --request PUT --url https://api.example.com/update --header "Content-Type: application/json"`
	parsed, err := ParseCurl(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if parsed.Method != "PUT" {
		t.Errorf("expected method PUT, got %s", parsed.Method)
	}
	if parsed.URL != "https://api.example.com/update" {
		t.Errorf("expected URL https://api.example.com/update, got %s", parsed.URL)
	}
	if parsed.Headers["Content-Type"] != "application/json" {
		t.Errorf("expected Content-Type header, got %q", parsed.Headers["Content-Type"])
	}
}

func TestSerializeCurl_Basic(t *testing.T) {
	parsed := &ParsedCurl{
		Method:  "POST",
		URL:     "https://api.example.com/balance",
		Headers: map[string]string{"Authorization": "Bearer token"},
	}
	result := SerializeCurl(parsed)
	if !strings.Contains(result, "curl") {
		t.Error("expected result to contain 'curl'")
	}
	if !strings.Contains(result, "-X POST") {
		t.Error("expected result to contain '-X POST'")
	}
	if !strings.Contains(result, "https://api.example.com/balance") {
		t.Error("expected result to contain the URL")
	}
	if !strings.Contains(result, "Authorization: Bearer token") {
		t.Error("expected result to contain the Authorization header")
	}
}

// --- Property-Based Test (Task 2.4) ---

// Feature: custom-curl-query, Property 1: Curl 解析/序列化 Round-Trip
// **Validates: Requirements 1.1, 1.2, 1.3, 1.4, 1.8**
func TestPropertyCurlRoundTrip(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate a random valid ParsedCurl
		methods := []string{"GET", "POST", "PUT", "DELETE", "PATCH", "HEAD", "OPTIONS"}
		method := rapid.SampledFrom(methods).Draw(t, "method")

		// Generate a valid URL (scheme + host, no single quotes to avoid shell quoting issues)
		schemes := []string{"https://", "http://"}
		scheme := rapid.SampledFrom(schemes).Draw(t, "scheme")
		// Host: alphanumeric with dots, at least 1 char
		host := rapid.StringMatching(`[a-z][a-z0-9]{0,15}(\.[a-z][a-z0-9]{0,7}){1,3}`).Draw(t, "host")
		// Optional path
		path := rapid.StringMatching(`(/[a-zA-Z0-9_-]{1,10}){0,3}`).Draw(t, "path")
		url := scheme + host + path

		// Generate random headers (0-5 headers, keys and values without colons or single quotes)
		numHeaders := rapid.IntRange(0, 5).Draw(t, "numHeaders")
		headers := make(map[string]string)
		for i := 0; i < numHeaders; i++ {
			key := rapid.StringMatching(`[A-Z][a-zA-Z0-9-]{0,15}`).Draw(t, "headerKey")
			// Generate values with at least one non-space char to ensure round-trip stability
			value := rapid.StringMatching(`[a-zA-Z0-9_./-][a-zA-Z0-9 _./-]{0,29}`).Draw(t, "headerValue")
			headers[key] = value
		}

		original := &ParsedCurl{
			Method:  method,
			URL:     url,
			Headers: headers,
		}

		// Serialize then parse back
		serialized := SerializeCurl(original)
		roundTripped, err := ParseCurl(serialized)
		if err != nil {
			t.Fatalf("round-trip parse failed: %v\nserialized: %s", err, serialized)
		}

		// Verify semantic equivalence
		if roundTripped.Method != original.Method {
			t.Fatalf("method mismatch: original=%s, roundTripped=%s", original.Method, roundTripped.Method)
		}
		if roundTripped.URL != original.URL {
			t.Fatalf("URL mismatch: original=%s, roundTripped=%s", original.URL, roundTripped.URL)
		}
		if len(roundTripped.Headers) != len(original.Headers) {
			t.Fatalf("headers count mismatch: original=%d, roundTripped=%d", len(original.Headers), len(roundTripped.Headers))
		}
		for k, v := range original.Headers {
			// TrimSpace is expected: HTTP header values have insignificant leading/trailing whitespace
			expected := strings.TrimSpace(v)
			if roundTripped.Headers[k] != expected {
				t.Fatalf("header %q mismatch: expected=%q, roundTripped=%q", k, expected, roundTripped.Headers[k])
			}
		}
	})
}
