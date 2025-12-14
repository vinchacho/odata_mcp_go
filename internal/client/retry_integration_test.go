// Copyright (c) 2024 OData MCP Contributors
// SPDX-License-Identifier: MIT

package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestRetryWithMockServer(t *testing.T) {
	tests := []struct {
		name              string
		maxRetries        int
		serverResponses   []int // Status codes to return in sequence
		expectedAttempts  int   // Expected number of requests to the server
		expectError       bool
		expectedStatus    int // Expected final status code
	}{
		{
			name:             "success on first try",
			maxRetries:       3,
			serverResponses:  []int{200},
			expectedAttempts: 1,
			expectError:      false,
			expectedStatus:   200,
		},
		{
			name:             "success after one retry",
			maxRetries:       3,
			serverResponses:  []int{503, 200},
			expectedAttempts: 2,
			expectError:      false,
			expectedStatus:   200,
		},
		{
			name:             "success after two retries",
			maxRetries:       3,
			serverResponses:  []int{503, 502, 200},
			expectedAttempts: 3,
			expectError:      false,
			expectedStatus:   200,
		},
		{
			name:             "exhausts all retries",
			maxRetries:       3,
			serverResponses:  []int{503, 503, 503, 503},
			expectedAttempts: 4, // Initial + 3 retries
			expectError:      false,
			expectedStatus:   503, // Returns last response
		},
		{
			name:             "429 rate limit with retry",
			maxRetries:       2,
			serverResponses:  []int{429, 429, 200},
			expectedAttempts: 3,
			expectError:      false,
			expectedStatus:   200,
		},
		{
			name:             "non-retryable 400 error",
			maxRetries:       3,
			serverResponses:  []int{400},
			expectedAttempts: 1,
			expectError:      false,
			expectedStatus:   400,
		},
		{
			name:             "non-retryable 404 error",
			maxRetries:       3,
			serverResponses:  []int{404},
			expectedAttempts: 1,
			expectError:      false,
			expectedStatus:   404,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var attemptCount int32

			// Create test server that returns different status codes
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				attempt := int(atomic.AddInt32(&attemptCount, 1)) - 1
				if attempt < len(tt.serverResponses) {
					w.WriteHeader(tt.serverResponses[attempt])
				} else {
					w.WriteHeader(200) // Default to success if more attempts than expected
				}
				w.Write([]byte(`{"d":{"results":[]}}`))
			}))
			defer server.Close()

			// Create client with fast backoff for testing
			client := NewODataClient(server.URL, false)
			client.retryConfig = &RetryConfig{
				MaxRetries:        tt.maxRetries,
				InitialBackoff:    1 * time.Millisecond, // Fast for testing
				MaxBackoff:        10 * time.Millisecond,
				BackoffMultiplier: 2.0,
				JitterFraction:    0,
				RetryableStatuses: []int{429, 500, 502, 503, 504},
			}

			// Make request
			ctx := context.Background()
			req, err := http.NewRequestWithContext(ctx, "GET", server.URL+"/test", nil)
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}

			resp, err := client.doRequestWithRetry(req, nil)

			// Check error expectation
			if tt.expectError && err == nil {
				t.Error("Expected error but got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			// Check attempt count
			actualAttempts := int(atomic.LoadInt32(&attemptCount))
			if actualAttempts != tt.expectedAttempts {
				t.Errorf("Expected %d attempts, got %d", tt.expectedAttempts, actualAttempts)
			}

			// Check final status
			if resp != nil && resp.StatusCode != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, resp.StatusCode)
			}
		})
	}
}

func TestRetryCSRFIntegration(t *testing.T) {
	var attemptCount int32
	csrfToken := "test-csrf-token-12345678"

	// Create test server that simulates CSRF failure then success
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&attemptCount, 1)

		// Token fetch request (GET with X-CSRF-Token: Fetch)
		if r.Method == "GET" && strings.EqualFold(r.Header.Get("X-CSRF-Token"), "Fetch") {
			w.Header().Set("X-CSRF-Token", csrfToken)
			w.WriteHeader(200)
			w.Write([]byte(`{}`))
			return
		}

		// POST request handling
		if r.Method == "POST" {
			token := r.Header.Get("X-CSRF-Token")
			if token == "" || token != csrfToken {
				// No token or wrong token - fail with CSRF error
				w.Header().Set("X-CSRF-Token", "required")
				w.WriteHeader(403)
				w.Write([]byte(`{"error":{"message":"CSRF token validation failed"}}`))
				return
			}
			// Valid token - success
			w.WriteHeader(200)
			w.Write([]byte(`{"d":{"results":[]}}`))
			return
		}

		// Default response
		w.WriteHeader(200)
		w.Write([]byte(`{"d":{"results":[]}}`))
	}))
	defer server.Close()

	// Create client
	client := NewODataClient(server.URL+"/", false)
	client.retryConfig = &RetryConfig{
		MaxRetries:        3,
		InitialBackoff:    1 * time.Millisecond,
		MaxBackoff:        10 * time.Millisecond,
		BackoffMultiplier: 2.0,
		JitterFraction:    0,
		RetryableStatuses: []int{429, 500, 502, 503, 504},
	}

	// Make POST request (should trigger CSRF flow)
	ctx := context.Background()
	req, err := http.NewRequestWithContext(ctx, "POST", server.URL+"/test", nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	resp, err := client.doRequestWithRetry(req, nil)

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if resp == nil {
		t.Fatal("Expected response but got nil")
	}

	// The flow should be:
	// 1. POST without token -> 403 CSRF
	// 2. GET token fetch -> get token
	// 3. POST with token -> 200
	// Total: at least 3 requests

	actualAttempts := int(atomic.LoadInt32(&attemptCount))
	if actualAttempts < 3 {
		t.Errorf("Expected at least 3 attempts for CSRF flow, got %d", actualAttempts)
	}

	if resp.StatusCode != 200 {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
}

func TestRetryBackoffTiming(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping timing test in short mode")
	}

	var timestamps []time.Time

	// Create test server that tracks request times
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		timestamps = append(timestamps, time.Now())
		if len(timestamps) < 3 {
			w.WriteHeader(503)
		} else {
			w.WriteHeader(200)
		}
		w.Write([]byte(`{}`))
	}))
	defer server.Close()

	// Create client with measurable backoff
	client := NewODataClient(server.URL, false)
	client.retryConfig = &RetryConfig{
		MaxRetries:        5,
		InitialBackoff:    50 * time.Millisecond,
		MaxBackoff:        500 * time.Millisecond,
		BackoffMultiplier: 2.0,
		JitterFraction:    0, // No jitter for predictable timing
		RetryableStatuses: []int{503},
	}

	ctx := context.Background()
	req, err := http.NewRequestWithContext(ctx, "GET", server.URL+"/test", nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	_, err = client.doRequestWithRetry(req, nil)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Verify we got 3 requests
	if len(timestamps) != 3 {
		t.Fatalf("Expected 3 requests, got %d", len(timestamps))
	}

	// Check timing between requests (with some tolerance)
	// First retry after ~50ms
	delay1 := timestamps[1].Sub(timestamps[0])
	if delay1 < 40*time.Millisecond || delay1 > 70*time.Millisecond {
		t.Errorf("First retry delay %v not in expected range [40ms, 70ms]", delay1)
	}

	// Second retry after ~100ms
	delay2 := timestamps[2].Sub(timestamps[1])
	if delay2 < 90*time.Millisecond || delay2 > 120*time.Millisecond {
		t.Errorf("Second retry delay %v not in expected range [90ms, 120ms]", delay2)
	}
}

func TestRetryContextCancellation(t *testing.T) {
	var attemptCount int32

	// Create test server that always returns retryable error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&attemptCount, 1)
		w.WriteHeader(503)
		w.Write([]byte(`{}`))
	}))
	defer server.Close()

	// Create client with slow backoff
	client := NewODataClient(server.URL, false)
	client.retryConfig = &RetryConfig{
		MaxRetries:        10, // Many retries
		InitialBackoff:    500 * time.Millisecond,
		MaxBackoff:        5 * time.Second,
		BackoffMultiplier: 2.0,
		JitterFraction:    0,
		RetryableStatuses: []int{503},
	}

	// Create cancellable context
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", server.URL+"/test", nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	_, err = client.doRequestWithRetry(req, nil)

	// Should get context error
	if err == nil {
		t.Error("Expected error due to context cancellation")
	}

	// Should have made at least 1 request but not all 10
	actualAttempts := int(atomic.LoadInt32(&attemptCount))
	if actualAttempts == 0 {
		t.Error("Expected at least 1 attempt")
	}
	if actualAttempts >= 5 {
		t.Errorf("Expected fewer attempts due to cancellation, got %d", actualAttempts)
	}
}
