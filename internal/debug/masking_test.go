// Copyright (c) 2024 OData MCP Contributors
// SPDX-License-Identifier: MIT

package debug

import (
	"strings"
	"testing"
)

func TestMaskPassword(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"empty password", "", ""},
		{"short password", "abc", "***"},
		{"normal password", "secret123", "***"},
		{"long password", "verylongpassword123!@#", "***"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MaskPassword(tt.input)
			if result != tt.expected {
				t.Errorf("MaskPassword(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestMaskToken(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"empty token", "", ""},
		{"very short token", "abc", "****"},
		{"exactly 8 chars", "12345678", "****"},
		{"9 chars", "123456789", "****23456789"},
		{"long token", "verylongtokenabcd1234", "****abcd1234"},
		{"realistic CSRF token", "abc123def456ghi789jkl", "****hi789jkl"}, // Last 8 chars: "hi789jkl"
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MaskToken(tt.input)
			if result != tt.expected {
				t.Errorf("MaskToken(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestMaskValue(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		showLastChars int
		expected      string
	}{
		{"empty value", "", 4, ""},
		{"shorter than show", "abc", 4, "***"},
		{"equal to show", "abcd", 4, "****"},
		{"longer than show", "abcdefgh", 4, "****efgh"},
		{"show zero chars", "secret", 0, "******"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MaskValue(tt.input, tt.showLastChars)
			if result != tt.expected {
				t.Errorf("MaskValue(%q, %d) = %q, want %q", tt.input, tt.showLastChars, result, tt.expected)
			}
		})
	}
}

func TestMaskURL(t *testing.T) {
	tests := []struct {
		name             string
		input            string
		shouldNotContain []string
		shouldContain    []string
	}{
		{
			name:             "URL without sensitive data",
			input:            "https://example.com/path",
			shouldNotContain: nil,
			shouldContain:    []string{"https://example.com/path"},
		},
		{
			name:             "URL with password in userinfo",
			input:            "https://user:secretpass@example.com/path",
			shouldNotContain: []string{"secretpass"},
			shouldContain:    []string{"user", "example.com"}, // URL encoding may encode ***
		},
		{
			name:             "URL with token query param",
			input:            "https://example.com/path?token=abc123secret",
			shouldNotContain: []string{"abc123secret"},
			shouldContain:    []string{"token="}, // Value is masked (may be URL encoded)
		},
		{
			name:             "URL with password query param",
			input:            "https://example.com/api?password=mysecret&user=admin",
			shouldNotContain: []string{"mysecret"},
			shouldContain:    []string{"password=", "user=admin"}, // Value is masked
		},
		{
			name:             "URL with api_key query param",
			input:            "https://example.com/api?api_key=key123&format=json",
			shouldNotContain: []string{"key123"},
			shouldContain:    []string{"api_key=", "format=json"}, // Value is masked
		},
		{
			name:             "Invalid URL returns as-is",
			input:            "not-a-valid-url://[invalid",
			shouldNotContain: nil,
			shouldContain:    []string{"not-a-valid-url://[invalid"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MaskURL(tt.input)

			for _, s := range tt.shouldNotContain {
				if strings.Contains(result, s) {
					t.Errorf("MaskURL(%q) = %q, should not contain %q", tt.input, result, s)
				}
			}

			for _, s := range tt.shouldContain {
				if !strings.Contains(result, s) {
					t.Errorf("MaskURL(%q) = %q, should contain %q", tt.input, result, s)
				}
			}
		})
	}
}

func TestMaskHeader(t *testing.T) {
	tests := []struct {
		name            string
		headerName      string
		headerValue     string
		shouldNotContain string
		shouldContain   string
	}{
		{
			name:            "empty value",
			headerName:      "Authorization",
			headerValue:     "",
			shouldContain:   "",
		},
		{
			name:             "Basic auth header",
			headerName:       "Authorization",
			headerValue:      "Basic dXNlcjpwYXNzd29yZA==",
			shouldNotContain: "dXNlcjpwYXNzd29yZA==",
			shouldContain:    "Basic ****",
		},
		{
			name:             "Bearer token header",
			headerName:       "Authorization",
			headerValue:      "Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9",
			shouldNotContain: "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9",
			shouldContain:    "Bearer ****",
		},
		{
			name:             "CSRF token header",
			headerName:       "X-CSRF-Token",
			headerValue:      "abc123def456ghi789",
			shouldNotContain: "abc123def456",
			shouldContain:    "****",
		},
		{
			name:          "Normal header unchanged",
			headerName:    "Content-Type",
			headerValue:   "application/json",
			shouldContain: "application/json",
		},
		{
			name:          "Accept header unchanged",
			headerName:    "Accept",
			headerValue:   "text/html",
			shouldContain: "text/html",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MaskHeader(tt.headerName, tt.headerValue)

			if tt.shouldNotContain != "" && strings.Contains(result, tt.shouldNotContain) {
				t.Errorf("MaskHeader(%q, %q) = %q, should not contain %q",
					tt.headerName, tt.headerValue, result, tt.shouldNotContain)
			}

			if tt.shouldContain != "" && !strings.Contains(result, tt.shouldContain) {
				t.Errorf("MaskHeader(%q, %q) = %q, should contain %q",
					tt.headerName, tt.headerValue, result, tt.shouldContain)
			}
		})
	}
}

func TestIsSensitiveKey(t *testing.T) {
	tests := []struct {
		key      string
		expected bool
	}{
		// Positive cases
		{"password", true},
		{"PASSWORD", true},
		{"user_password", true},
		{"passwd", true},
		{"pwd", true},
		{"secret", true},
		{"client_secret", true},
		{"token", true},
		{"access_token", true},
		{"api_key", true},
		{"apikey", true},
		{"api-key", true},
		{"authorization", true},
		{"auth", true},
		{"auth_token", true},
		{"credential", true},
		{"credentials", true},
		{"x-csrf-token", true},
		{"csrf", true},
		{"csrf_token", true},

		// Negative cases
		{"username", false},
		{"user", false},
		{"email", false},
		{"content-type", false},
		{"accept", false},
		{"host", false},
		{"path", false},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			result := IsSensitiveKey(tt.key)
			if result != tt.expected {
				t.Errorf("IsSensitiveKey(%q) = %v, want %v", tt.key, result, tt.expected)
			}
		})
	}
}
