// Copyright (c) 2024 OData MCP Contributors
// SPDX-License-Identifier: MIT

package client

import (
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/zmcp/odata-mcp/internal/constants"
	"github.com/zmcp/odata-mcp/internal/debug"
)

// Context key for HTTP headers passed from MCP server
type contextKey string

const HTTPHeadersContextKey contextKey = "mcp-http-headers"

// ODataClient handles HTTP communication with OData services
type ODataClient struct {
	baseURL        string
	httpClient     *http.Client
	cookies        map[string]string
	username       string
	password       string
	csrfToken      string
	verbose        bool
	sessionCookies []*http.Cookie // Track session cookies from server
	isV4           bool           // Whether the service is OData v4
	retryConfig    *RetryConfig   // Retry configuration for failed requests
	mu             sync.RWMutex   // Guards mutable fields: csrfToken, sessionCookies, cookies
}

// NewODataClient creates a new OData client
func NewODataClient(baseURL string, verbose bool) *ODataClient {
	// Ensure base URL ends with /
	if !strings.HasSuffix(baseURL, "/") {
		baseURL += "/"
	}

	return &ODataClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: time.Duration(constants.DefaultTimeout) * time.Second,
		},
		verbose:     verbose,
		isV4:        false,                // Will be determined when fetching metadata
		retryConfig: DefaultRetryConfig(), // Use default retry configuration
	}
}

// SetBasicAuth configures basic authentication
func (c *ODataClient) SetBasicAuth(username, password string) {
	c.username = username
	c.password = password
}

// SetCookies configures cookie authentication
func (c *ODataClient) SetCookies(cookies map[string]string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.cookies = cookies
}

// SetRetryConfig configures retry behavior for failed requests
func (c *ODataClient) SetRetryConfig(cfg *RetryConfig) {
	if cfg != nil {
		c.retryConfig = cfg
	}
}

// ConfigureRetry configures retry behavior from individual parameters.
// Convenience for wiring CLI flags directly.
func (c *ODataClient) ConfigureRetry(maxAttempts, initialBackoffMs, maxBackoffMs int, backoffMultiplier float64) {
	c.retryConfig = &RetryConfig{
		MaxRetries:        maxAttempts,
		InitialBackoff:    time.Duration(initialBackoffMs) * time.Millisecond,
		MaxBackoff:        time.Duration(maxBackoffMs) * time.Millisecond,
		BackoffMultiplier: backoffMultiplier,
		JitterFraction:    0.1,
		RetryableStatuses: []int{429, 500, 502, 503, 504},
	}
}

// SetTimeout sets the HTTP client timeout for all requests
func (c *ODataClient) SetTimeout(timeout time.Duration) {
	c.httpClient.Timeout = timeout
}

// SetMetadataTimeout temporarily sets a longer timeout for metadata operations.
// Returns a function to restore the original timeout.
func (c *ODataClient) SetMetadataTimeout(timeout time.Duration) func() {
	original := c.httpClient.Timeout
	c.httpClient.Timeout = timeout
	return func() { c.httpClient.Timeout = original }
}

// maskHeaders creates a string representation of headers with sensitive values masked.
// Used by verbose logging in http.go to avoid leaking auth tokens.
func maskHeaders(headers http.Header) string {
	var parts []string
	for name, values := range headers {
		for _, value := range values {
			maskedValue := debug.MaskHeader(name, value)
			parts = append(parts, fmt.Sprintf("%s: %s", name, maskedValue))
		}
	}
	return strings.Join(parts, ", ")
}

// Helper function for min
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// shouldForwardHeader determines if a header should be forwarded from MCP to OData service
func shouldForwardHeader(headerName string) bool {
	lower := strings.ToLower(headerName)

	// Allow authentication headers
	if lower == "authorization" || lower == "cookie" {
		return true
	}

	// Allow custom headers (X- prefix)
	if strings.HasPrefix(lower, "x-") {
		return true
	}

	// Block hop-by-hop headers and other problematic headers
	blockedHeaders := []string{
		"host",
		"connection",
		"keep-alive",
		"transfer-encoding",
		"upgrade",
		"proxy-authenticate",
		"proxy-authorization",
		"te",
		"trailer",
		"content-length", // Will be set by http.Client
		"content-type",   // Set by specific methods
		"accept",         // Set by buildRequest
		"user-agent",     // Set by buildRequest
	}

	for _, blocked := range blockedHeaders {
		if lower == blocked {
			return false
		}
	}

	// Allow other headers by default
	return true
}
