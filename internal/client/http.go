// Copyright (c) 2024 OData MCP Contributors
// SPDX-License-Identifier: MIT

package client

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/zmcp/odata-mcp/internal/constants"
	"github.com/zmcp/odata-mcp/internal/debug"
)

// buildRequest creates an HTTP request with proper headers and authentication
func (c *ODataClient) buildRequest(ctx context.Context, method, endpoint string, body io.Reader) (*http.Request, error) {
	fullURL := c.baseURL + strings.TrimPrefix(endpoint, "/")

	req, err := http.NewRequestWithContext(ctx, method, fullURL, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set standard headers
	req.Header.Set(constants.UserAgent, constants.DefaultUserAgent)
	if c.isV4 {
		req.Header.Set(constants.Accept, constants.ContentTypeODataJSONV4)
	} else {
		req.Header.Set(constants.Accept, constants.ContentTypeJSON)
	}

	// Apply headers from context (from MCP HTTP connection) - do this BEFORE config-based auth
	// This allows dynamic headers from MCP connection to be used
	if headers, ok := ctx.Value(HTTPHeadersContextKey).(http.Header); ok {
		var headerCount int
		for key, values := range headers {
			// Skip headers that shouldn't be forwarded
			if shouldForwardHeader(key) {
				for _, value := range values {
					req.Header.Add(key, value)
					headerCount++
				}
			}
		}
		if c.verbose && headerCount > 0 {
			fmt.Fprintf(os.Stderr, "[VERBOSE] Applied %d header(s) from MCP context\n", headerCount)
		}
	}

	// Set authentication from configuration (will override context headers if both exist)
	// This maintains backward compatibility with existing config-based auth
	if c.username != "" && c.password != "" {
		req.SetBasicAuth(c.username, c.password)
		if c.verbose {
			fmt.Fprintf(os.Stderr, "[VERBOSE] Using configured basic auth\n")
		}
	}

	// Lock for reading mutable fields: cookies, sessionCookies, csrfToken
	c.mu.RLock()
	defer c.mu.RUnlock()

	// Set cookies
	for name, value := range c.cookies {
		req.AddCookie(&http.Cookie{
			Name:  name,
			Value: value,
		})
	}

	// Add session cookies received from server
	for _, cookie := range c.sessionCookies {
		req.AddCookie(cookie)
	}

	// Set CSRF token if available
	if c.csrfToken != "" {
		req.Header.Set(constants.CSRFTokenHeader, c.csrfToken)
		if c.verbose {
			fmt.Fprintf(os.Stderr, "[VERBOSE] Adding CSRF token to request: %s\n", debug.MaskToken(c.csrfToken))
		}
	}

	return req, nil
}

// doRequest executes an HTTP request with exponential backoff retry,
// CSRF retry, and common error handling.
func (c *ODataClient) doRequest(req *http.Request) (*http.Response, error) {
	// For requests with body, we need to save it for potential retry
	var bodyBytes []byte
	if req.Body != nil && req.ContentLength > 0 {
		var err error
		bodyBytes, err = io.ReadAll(req.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to read request body: %w", err)
		}
		req.Body = io.NopCloser(bytes.NewReader(bodyBytes))
	}

	return c.doRequestWithBackoff(req, bodyBytes)
}

// doRequestWithBackoff wraps doRequestWithRetry with exponential backoff
// for transient HTTP failures (429, 5xx) and network errors.
func (c *ODataClient) doRequestWithBackoff(req *http.Request, bodyBytes []byte) (*http.Response, error) {
	var lastErr error
	var lastResp *http.Response

	for attempt := 0; attempt <= c.retryConfig.MaxRetries; attempt++ {
		// Wait before retry (skip first attempt)
		if attempt > 0 {
			backoff := c.retryConfig.CalculateBackoff(attempt - 1)
			if c.verbose {
				fmt.Fprintf(os.Stderr, "[VERBOSE] Retry attempt %d/%d after %v\n",
					attempt, c.retryConfig.MaxRetries, backoff)
			}
			select {
			case <-req.Context().Done():
				return nil, req.Context().Err()
			case <-time.After(backoff):
			}
		}

		// Reset body for retry
		if len(bodyBytes) > 0 {
			req.Body = io.NopCloser(bytes.NewReader(bodyBytes))
			req.ContentLength = int64(len(bodyBytes))
		}

		resp, err := c.doRequestWithRetry(req, bodyBytes, false)
		if err != nil {
			lastErr = err
			if c.verbose {
				fmt.Fprintf(os.Stderr, "[VERBOSE] Request failed: %v\n", err)
			}
			continue
		}

		// Check if we should retry based on status code
		if c.retryConfig.ShouldRetry(resp.StatusCode, attempt) {
			if c.verbose {
				fmt.Fprintf(os.Stderr, "[VERBOSE] Received status %d, will retry\n", resp.StatusCode)
			}
			// Drain and close body before retrying so the connection can be reused
			respBody, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			lastResp = resp
			lastResp.Body = io.NopCloser(bytes.NewReader(respBody))
			continue
		}

		// Success or non-retryable error
		return resp, nil
	}

	// All retries exhausted
	if lastResp != nil {
		return lastResp, nil
	}
	return nil, fmt.Errorf("all %d retries failed: %w", c.retryConfig.MaxRetries, lastErr)
}

// doRequestWithRetry executes an HTTP request with CSRF retry logic.
// This is called by doRequestWithBackoff for each backoff attempt.
func (c *ODataClient) doRequestWithRetry(req *http.Request, bodyBytes []byte, isRetry bool) (*http.Response, error) {
	if c.verbose && !isRetry {
		fmt.Fprintf(os.Stderr, "[VERBOSE] %s %s\n", req.Method, debug.MaskURL(req.URL.String()))
	}

	// Reset body if we have it (for retry scenarios)
	if len(bodyBytes) > 0 {
		req.Body = io.NopCloser(bytes.NewReader(bodyBytes))
		req.ContentLength = int64(len(bodyBytes))
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}

	// Check if this is a modifying operation
	modifyingMethods := []string{"POST", "PUT", "MERGE", "PATCH", "DELETE"}
	isModifying := false
	for _, m := range modifyingMethods {
		if req.Method == m {
			isModifying = true
			break
		}
	}

	// Handle CSRF token validation failure (Python-style)
	if resp.StatusCode == http.StatusForbidden && isModifying && !isRetry {
		// Read response body to check for CSRF-related errors
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		bodyStr := string(body)

		csrfFailed := strings.Contains(bodyStr, "CSRF token validation failed") ||
			strings.Contains(strings.ToLower(bodyStr), "csrf") ||
			strings.EqualFold(resp.Header.Get("x-csrf-token"), "required")

		if csrfFailed {
			if c.verbose {
				fmt.Fprintf(os.Stderr, "[VERBOSE] CSRF token validation failed, attempting to refetch...\n")
			}

			// Clear the invalid token
			c.mu.Lock()
			c.csrfToken = ""
			c.mu.Unlock()

			// Try to fetch new CSRF token
			if err := c.fetchCSRFToken(req.Context()); err != nil {
				// Return original error with CSRF context
				return nil, fmt.Errorf("CSRF token required but refetch failed. Status: %d. Response: %s", resp.StatusCode, bodyStr)
			}

			// Retry original request with new CSRF token
			c.mu.RLock()
			req.Header.Set(constants.CSRFTokenHeader, c.csrfToken)
			c.mu.RUnlock()
			if c.verbose {
				fmt.Fprintf(os.Stderr, "[VERBOSE] Retrying request with new CSRF token...\n")
			}
			return c.doRequestWithRetry(req, bodyBytes, true)
		}

		// Not a CSRF error, recreate response with body
		resp.Body = io.NopCloser(bytes.NewReader(body))
	}

	return resp, nil
}

// fetchCSRFToken fetches a CSRF token from the service
func (c *ODataClient) fetchCSRFToken(ctx context.Context) error {
	if c.verbose {
		fmt.Fprintf(os.Stderr, "[VERBOSE] Fetching CSRF token...\n")
	}

	// Clear any existing CSRF token (Python behavior)
	c.mu.Lock()
	c.csrfToken = ""
	c.mu.Unlock()

	// Use service root for CSRF token fetching (more reliable than empty string)
	req, err := c.buildRequest(ctx, constants.GET, "", nil)
	if err != nil {
		return err
	}

	req.Header.Set(constants.CSRFTokenHeader, constants.CSRFTokenFetch)

	if c.verbose {
		fmt.Fprintf(os.Stderr, "[VERBOSE] Token fetch request: %s %s\n", req.Method, debug.MaskURL(req.URL.String()))
		fmt.Fprintf(os.Stderr, "[VERBOSE] Token fetch headers: %s\n", maskHeaders(req.Header))
	}

	// Don't use doRequest here to avoid retry loops - fetch token requests shouldn't retry
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("CSRF token request failed: %w", err)
	}
	defer resp.Body.Close()

	// Store any session cookies from the response
	if cookies := resp.Cookies(); len(cookies) > 0 {
		c.mu.Lock()
		c.sessionCookies = append(c.sessionCookies, cookies...)
		c.mu.Unlock()
		if c.verbose {
			fmt.Fprintf(os.Stderr, "[VERBOSE] Received %d session cookies during token fetch\n", len(cookies))
			for _, cookie := range cookies {
				fmt.Fprintf(os.Stderr, "[VERBOSE] Cookie: %s=%s (Path=%s)\n", cookie.Name, debug.MaskToken(cookie.Value), cookie.Path)
			}
		}
	}

	if c.verbose {
		fmt.Fprintf(os.Stderr, "[VERBOSE] Token fetch response status: %d\n", resp.StatusCode)
		fmt.Fprintf(os.Stderr, "[VERBOSE] Token fetch response headers: %s\n", maskHeaders(resp.Header))
	}

	// Check both possible header names (case variations)
	token := resp.Header.Get(constants.CSRFTokenHeader)
	if token == "" {
		token = resp.Header.Get(constants.CSRFTokenHeaderLower)
	}

	// Additional header variations that some SAP systems use
	if token == "" {
		token = resp.Header.Get("x-csrf-token")
	}
	if token == "" {
		token = resp.Header.Get("X-Csrf-Token")
	}

	if token == "" || token == constants.CSRFTokenFetch {
		return fmt.Errorf("CSRF token not found in response headers")
	}

	c.mu.Lock()
	c.csrfToken = token
	c.mu.Unlock()
	if c.verbose {
		fmt.Fprintf(os.Stderr, "[VERBOSE] CSRF token fetched successfully: %s\n", debug.MaskToken(token))
	}

	return nil
}
