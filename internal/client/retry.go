// Copyright (c) 2024 OData MCP Contributors
// SPDX-License-Identifier: MIT

package client

import (
	"math"
	"math/rand"
	"net/http"
	"strings"
	"time"
)

// RetryConfig defines retry behavior for HTTP requests
type RetryConfig struct {
	MaxRetries        int           // Maximum number of retry attempts (0 = no retries)
	InitialBackoff    time.Duration // Initial delay before first retry
	MaxBackoff        time.Duration // Maximum delay between retries
	BackoffMultiplier float64       // Multiplier for exponential backoff
	JitterFraction    float64       // Random jitter fraction (0.0-1.0)
	RetryableStatuses []int         // HTTP status codes that trigger retry
}

// DefaultRetryConfig returns sensible defaults for retry behavior
func DefaultRetryConfig() *RetryConfig {
	return &RetryConfig{
		MaxRetries:        3,
		InitialBackoff:    100 * time.Millisecond,
		MaxBackoff:        10 * time.Second,
		BackoffMultiplier: 2.0,
		JitterFraction:    0.1,
		RetryableStatuses: []int{429, 500, 502, 503, 504},
	}
}

// CalculateBackoff returns the delay for a given attempt (0-indexed)
// attempt 0 returns InitialBackoff, subsequent attempts grow exponentially
func (c *RetryConfig) CalculateBackoff(attempt int) time.Duration {
	if attempt <= 0 {
		return c.InitialBackoff
	}

	// Exponential backoff: initial * multiplier^attempt
	backoff := float64(c.InitialBackoff) * math.Pow(c.BackoffMultiplier, float64(attempt))

	// Cap at max backoff
	if backoff > float64(c.MaxBackoff) {
		backoff = float64(c.MaxBackoff)
	}

	// Apply jitter to prevent thundering herd
	if c.JitterFraction > 0 {
		// jitter is ±(backoff * jitterFraction)
		jitterRange := backoff * c.JitterFraction
		jitter := (rand.Float64()*2 - 1) * jitterRange
		backoff += jitter

		// Ensure we don't go negative
		if backoff < 0 {
			backoff = 0
		}
	}

	return time.Duration(backoff)
}

// ShouldRetry determines if a request should be retried based on status code and attempt count
func (c *RetryConfig) ShouldRetry(statusCode int, attempt int) bool {
	if attempt >= c.MaxRetries {
		return false
	}

	for _, code := range c.RetryableStatuses {
		if statusCode == code {
			return true
		}
	}

	return false
}

// IsRetryableStatus checks if a status code is in the retryable list
func (c *RetryConfig) IsRetryableStatus(statusCode int) bool {
	for _, code := range c.RetryableStatuses {
		if statusCode == code {
			return true
		}
	}
	return false
}

// IsCSRFFailure checks if the response indicates a CSRF token validation failure
// This is specific to SAP systems which return 403 with CSRF-related error messages
func IsCSRFFailure(resp *http.Response, body []byte) bool {
	if resp == nil || resp.StatusCode != http.StatusForbidden {
		return false
	}

	// Check response header for CSRF requirement
	if strings.EqualFold(resp.Header.Get("x-csrf-token"), "required") {
		return true
	}

	// Check response body for CSRF-related error messages
	bodyStr := string(body)
	return strings.Contains(bodyStr, "CSRF token validation failed") ||
		strings.Contains(strings.ToLower(bodyStr), "csrf")
}
