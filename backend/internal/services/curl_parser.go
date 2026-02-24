package services

import (
	"fmt"
	"sort"
	"strings"
)

// ParsedCurl represents a parsed curl command.
type ParsedCurl struct {
	Method  string            `json:"method"`
	URL     string            `json:"url"`
	Headers map[string]string `json:"headers"`
}

// ParseCurl parses a curl command string into a ParsedCurl struct.
// It supports -X/--request, -H/--header, --url, and bare URLs.
func ParseCurl(curlCommand string) (*ParsedCurl, error) {
	tokens, err := tokenize(curlCommand)
	if err != nil {
		return nil, fmt.Errorf("curl 命令解析失败: %w", err)
	}

	// Strip leading "curl" if present
	if len(tokens) > 0 && strings.ToLower(tokens[0]) == "curl" {
		tokens = tokens[1:]
	}

	parsed := &ParsedCurl{
		Headers: make(map[string]string),
	}

	for i := 0; i < len(tokens); i++ {
		tok := tokens[i]

		switch {
		case tok == "-X" || tok == "--request":
			if i+1 >= len(tokens) {
				return nil, fmt.Errorf("curl 命令解析失败: %s 缺少参数值", tok)
			}
			i++
			parsed.Method = strings.ToUpper(tokens[i])

		case tok == "-H" || tok == "--header":
			if i+1 >= len(tokens) {
				return nil, fmt.Errorf("curl 命令解析失败: %s 缺少参数值", tok)
			}
			i++
			key, value, err := parseHeader(tokens[i])
			if err != nil {
				return nil, err
			}
			parsed.Headers[key] = value

		case tok == "--url":
			if i+1 >= len(tokens) {
				return nil, fmt.Errorf("curl 命令解析失败: --url 缺少参数值")
			}
			i++
			parsed.URL = tokens[i]

		case isURL(tok):
			if parsed.URL == "" {
				parsed.URL = tok
			}
		}
	}

	if parsed.URL == "" {
		return nil, fmt.Errorf("curl 命令缺少 URL")
	}

	if parsed.Method == "" {
		parsed.Method = "GET"
	}

	return parsed, nil
}

// SerializeCurl produces a curl command string from a ParsedCurl struct.
// Headers are sorted alphabetically for deterministic output.
func SerializeCurl(parsed *ParsedCurl) string {
	var parts []string
	parts = append(parts, "curl")

	method := parsed.Method
	if method == "" {
		method = "GET"
	}
	parts = append(parts, "-X", method)

	parts = append(parts, quote(parsed.URL))

	// Sort header keys for deterministic output
	keys := make([]string, 0, len(parsed.Headers))
	for k := range parsed.Headers {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		parts = append(parts, "-H", quote(k+": "+parsed.Headers[k]))
	}

	return strings.Join(parts, " ")
}

// tokenize splits a curl command string into tokens, handling quoted strings
// and escape characters.
func tokenize(input string) ([]string, error) {
	var tokens []string
	var current strings.Builder
	inSingle := false
	inDouble := false
	escaped := false

	for i := 0; i < len(input); i++ {
		ch := input[i]

		if escaped {
			current.WriteByte(ch)
			escaped = false
			continue
		}

		if ch == '\\' && !inSingle {
			escaped = true
			continue
		}

		if ch == '\'' && !inDouble {
			inSingle = !inSingle
			continue
		}

		if ch == '"' && !inSingle {
			inDouble = !inDouble
			continue
		}

		if (ch == ' ' || ch == '\t' || ch == '\n' || ch == '\r') && !inSingle && !inDouble {
			if current.Len() > 0 {
				tokens = append(tokens, current.String())
				current.Reset()
			}
			continue
		}

		current.WriteByte(ch)
	}

	if inSingle || inDouble {
		return nil, fmt.Errorf("引号不匹配")
	}

	if current.Len() > 0 {
		tokens = append(tokens, current.String())
	}

	return tokens, nil
}

// parseHeader parses a header string in "Key: Value" format.
func parseHeader(header string) (string, string, error) {
	idx := strings.Index(header, ":")
	if idx < 0 {
		return "", "", fmt.Errorf("请求头格式无效: %q (应为 'Key: Value')", header)
	}
	key := strings.TrimSpace(header[:idx])
	value := strings.TrimSpace(header[idx+1:])
	return key, value, nil
}

// isURL checks if a token looks like a URL (starts with http:// or https://).
func isURL(s string) bool {
	lower := strings.ToLower(s)
	return strings.HasPrefix(lower, "http://") || strings.HasPrefix(lower, "https://")
}

// quote wraps a string in single quotes for curl command output.
func quote(s string) string {
	escaped := strings.ReplaceAll(s, "'", "'\\''")
	return "'" + escaped + "'"
}
