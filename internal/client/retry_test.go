// Copyright (c) 2024 OData MCP Contributors
// SPDX-License-Identifier: MIT

package client

import (
	"net/http"
	"testing"
	"time"
)

func TestDefaultRetryConfig(t *testing.T) {
	cfg := DefaultRetryConfig()

	if cfg.MaxRetries != 3 {
		t.Errorf("MaxRetries = %d, want 3", cfg.MaxRetries)
	}
	if cfg.InitialBackoff != 100*time.Millisecond {
		t.Errorf("InitialBackoff = %v, want 100ms", cfg.InitialBackoff)
	}
	if cfg.MaxBackoff != 10*time.Second {
		t.Errorf("MaxBackoff = %v, want 10s", cfg.MaxBackoff)
	}
	if cfg.BackoffMultiplier != 2.0 {
		t.Errorf("BackoffMultiplier = %v, want 2.0", cfg.BackoffMultiplier)
	}
	if cfg.JitterFraction != 0.1 {
		t.Errorf("JitterFraction = %v, want 0.1", cfg.JitterFraction)
	}

	expectedStatuses := []int{429, 500, 502, 503, 504}
	if len(cfg.RetryableStatuses) != len(expectedStatuses) {
		t.Errorf("RetryableStatuses length = %d, want %d", len(cfg.RetryableStatuses), len(expectedStatuses))
	}
	for i, status := range expectedStatuses {
		if cfg.RetryableStatuses[i] != status {
			t.Errorf("RetryableStatuses[%d] = %d, want %d", i, cfg.RetryableStatuses[i], status)
		}
	}
}

func TestCalculateBackoff(t *testing.T) {
	cfg := &RetryConfig{
		InitialBackoff:    100 * time.Millisecond,
		MaxBackoff:        10 * time.Second,
		BackoffMultiplier: 2.0,
		JitterFraction:    0, // Disable jitter for predictable tests
	}

	tests := []struct {
		attempt  int
		expected time.Duration
	}{
		{0, 100 * time.Millisecond},
		{1, 200 * time.Millisecond},
		{2, 400 * time.Millisecond},
		{3, 800 * time.Millisecond},
		{4, 1600 * time.Millisecond},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			result := cfg.CalculateBackoff(tt.attempt)
			if result != tt.expected {
				t.Errorf("CalculateBackoff(%d) = %v, want %v", tt.attempt, result, tt.expected)
			}
		})
	}
}

func TestCalculateBackoffMaxCap(t *testing.T) {
	cfg := &RetryConfig{
		InitialBackoff:    100 * time.Millisecond,
		MaxBackoff:        500 * time.Millisecond,
		BackoffMultiplier: 2.0,
		JitterFraction:    0, // Disable jitter for predictable tests
	}

	// After several attempts, backoff should cap at MaxBackoff
	tests := []struct {
		attempt  int
		expected time.Duration
	}{
		{0, 100 * time.Millisecond},
		{1, 200 * time.Millisecond},
		{2, 400 * time.Millisecond},
		{3, 500 * time.Millisecond}, // Capped
		{4, 500 * time.Millisecond}, // Still capped
		{10, 500 * time.Millisecond}, // Still capped
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			result := cfg.CalculateBackoff(tt.attempt)
			if result != tt.expected {
				t.Errorf("CalculateBackoff(%d) = %v, want %v", tt.attempt, result, tt.expected)
			}
		})
	}
}

func TestCalculateBackoffWithJitter(t *testing.T) {
	cfg := &RetryConfig{
		InitialBackoff:    100 * time.Millisecond,
		MaxBackoff:        10 * time.Second,
		BackoffMultiplier: 2.0,
		JitterFraction:    0.1, // ±10% jitter
	}

	// With jitter, we can't test exact values, but we can test the range
	baseBackoff := 200 * time.Millisecond // attempt 1
	minExpected := time.Duration(float64(baseBackoff) * 0.9)
	maxExpected := time.Duration(float64(baseBackoff) * 1.1)

	// Run multiple times to verify jitter is applied
	for i := 0; i < 10; i++ {
		result := cfg.CalculateBackoff(1)
		if result < minExpected || result > maxExpected {
			t.Errorf("CalculateBackoff(1) with jitter = %v, want between %v and %v",
				result, minExpected, maxExpected)
		}
	}
}

func TestShouldRetry(t *testing.T) {
	cfg := DefaultRetryConfig()

	tests := []struct {
		name       string
		statusCode int
		attempt    int
		expected   bool
	}{
		// Retryable status codes within attempt limit
		{"503 first attempt", 503, 0, true},
		{"503 second attempt", 503, 1, true},
		{"503 third attempt", 503, 2, true},
		{"429 first attempt", 429, 0, true},
		{"500 first attempt", 500, 0, true},
		{"502 first attempt", 502, 0, true},
		{"504 first attempt", 504, 0, true},

		// Max retries exceeded
		{"503 fourth attempt", 503, 3, false},
		{"503 fifth attempt", 503, 4, false},

		// Non-retryable status codes
		{"200 OK", 200, 0, false},
		{"201 Created", 201, 0, false},
		{"400 Bad Request", 400, 0, false},
		{"401 Unauthorized", 401, 0, false},
		{"403 Forbidden", 403, 0, false}, // CSRF handled separately
		{"404 Not Found", 404, 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cfg.ShouldRetry(tt.statusCode, tt.attempt)
			if result != tt.expected {
				t.Errorf("ShouldRetry(%d, %d) = %v, want %v",
					tt.statusCode, tt.attempt, result, tt.expected)
			}
		})
	}
}

func TestIsRetryableStatus(t *testing.T) {
	cfg := DefaultRetryConfig()

	tests := []struct {
		statusCode int
		expected   bool
	}{
		{429, true},
		{500, true},
		{502, true},
		{503, true},
		{504, true},
		{200, false},
		{400, false},
		{401, false},
		{403, false},
		{404, false},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			result := cfg.IsRetryableStatus(tt.statusCode)
			if result != tt.expected {
				t.Errorf("IsRetryableStatus(%d) = %v, want %v", tt.statusCode, result, tt.expected)
			}
		})
	}
}

func TestIsCSRFFailure(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		headers    map[string]string
		body       string
		expected   bool
	}{
		{
			name:       "403 with CSRF message in body",
			statusCode: 403,
			body:       `{"error": {"message": "CSRF token validation failed"}}`,
			expected:   true,
		},
		{
			name:       "403 with lowercase csrf in body",
			statusCode: 403,
			body:       `{"error": {"message": "csrf error occurred"}}`,
			expected:   true,
		},
		{
			name:       "403 with x-csrf-token required header",
			statusCode: 403,
			headers:    map[string]string{"x-csrf-token": "required"},
			body:       `{"error": {"message": "Access denied"}}`,
			expected:   true,
		},
		{
			name:       "403 without CSRF indicators",
			statusCode: 403,
			body:       `{"error": {"message": "Access denied"}}`,
			expected:   false,
		},
		{
			name:       "401 with CSRF message (wrong status)",
			statusCode: 401,
			body:       `{"error": {"message": "CSRF token validation failed"}}`,
			expected:   false,
		},
		{
			name:       "200 OK",
			statusCode: 200,
			body:       `{"d": {"results": []}}`,
			expected:   false,
		},
		{
			name:       "nil response",
			statusCode: 0,
			expected:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var resp *http.Response
			if tt.statusCode > 0 {
				resp = &http.Response{
					StatusCode: tt.statusCode,
					Header:     make(http.Header),
				}
				for k, v := range tt.headers {
					resp.Header.Set(k, v)
				}
			}

			result := IsCSRFFailure(resp, []byte(tt.body))
			if result != tt.expected {
				t.Errorf("IsCSRFFailure() = %v, want %v", result, tt.expected)
			}
		})
	}
}
