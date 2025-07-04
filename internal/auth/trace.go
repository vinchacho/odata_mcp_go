package auth

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// AuthTracer provides detailed tracing of authentication flows
type AuthTracer struct {
	enabled  bool
	logFile  *os.File
	verbose  bool
}

// NewAuthTracer creates a new authentication tracer
func NewAuthTracer(enabled bool) (*AuthTracer, error) {
	if !enabled {
		return &AuthTracer{enabled: false}, nil
	}
	
	// Create trace directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	
	traceDir := filepath.Join(homeDir, ".odata-mcp", "traces")
	if err := os.MkdirAll(traceDir, 0755); err != nil {
		return nil, err
	}
	
	// Create trace file with timestamp
	timestamp := time.Now().Format("20060102_150405")
	filename := filepath.Join(traceDir, fmt.Sprintf("auth_trace_%s.log", timestamp))
	
	file, err := os.Create(filename)
	if err != nil {
		return nil, err
	}
	
	fmt.Fprintf(os.Stderr, "[TRACE] Authentication trace file: %s\n", filename)
	
	return &AuthTracer{
		enabled: true,
		logFile: file,
		verbose: true,
	}, nil
}

// Close closes the trace file
func (t *AuthTracer) Close() error {
	if t.logFile != nil {
		return t.logFile.Close()
	}
	return nil
}

// Log writes a log entry
func (t *AuthTracer) Log(format string, args ...interface{}) {
	if !t.enabled || t.logFile == nil {
		return
	}
	
	timestamp := time.Now().Format("2006-01-02 15:04:05.000")
	msg := fmt.Sprintf(format, args...)
	fmt.Fprintf(t.logFile, "[%s] %s\n", timestamp, msg)
	t.logFile.Sync()
	
	if t.verbose {
		fmt.Fprintf(os.Stderr, "[TRACE] %s\n", msg)
	}
}

// LogRequest logs an HTTP request
func (t *AuthTracer) LogRequest(req *http.Request) {
	if !t.enabled {
		return
	}
	
	t.Log("=== HTTP Request ===")
	t.Log("Method: %s", req.Method)
	t.Log("URL: %s", req.URL.String())
	
	// Log headers (redact sensitive ones)
	t.Log("Headers:")
	for key, values := range req.Header {
		value := strings.Join(values, ", ")
		if strings.Contains(strings.ToLower(key), "authorization") {
			value = redactToken(value)
		}
		t.Log("  %s: %s", key, value)
	}
	
	// Log body if present
	if req.Body != nil {
		body, _ := io.ReadAll(req.Body)
		req.Body = io.NopCloser(bytes.NewReader(body))
		if len(body) > 0 {
			t.Log("Body: %s", redactSensitive(string(body)))
		}
	}
	
	t.Log("===================")
}

// LogResponse logs an HTTP response
func (t *AuthTracer) LogResponse(resp *http.Response) {
	if !t.enabled {
		return
	}
	
	t.Log("=== HTTP Response ===")
	t.Log("Status: %s", resp.Status)
	
	// Log headers
	t.Log("Headers:")
	for key, values := range resp.Header {
		value := strings.Join(values, ", ")
		t.Log("  %s: %s", key, value)
	}
	
	// Log cookies
	if cookies := resp.Cookies(); len(cookies) > 0 {
		t.Log("Cookies:")
		for _, cookie := range cookies {
			t.Log("  %s: %s (domain=%s, path=%s)", cookie.Name, redactCookieValue(cookie.Value), cookie.Domain, cookie.Path)
		}
	}
	
	// Log body preview
	if resp.Body != nil {
		body, _ := io.ReadAll(resp.Body)
		resp.Body = io.NopCloser(bytes.NewReader(body))
		if len(body) > 0 {
			preview := string(body)
			if len(preview) > 1000 {
				preview = preview[:1000] + "... (truncated)"
			}
			t.Log("Body: %s", redactSensitive(preview))
		}
	}
	
	t.Log("====================")
}

// TraceRoundTripper wraps an http.RoundTripper to add tracing
type TraceRoundTripper struct {
	Transport http.RoundTripper
	Tracer    *AuthTracer
}

// RoundTrip implements http.RoundTripper with tracing
func (t *TraceRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	t.Tracer.LogRequest(req)
	
	// Clone request for dumping
	if t.Tracer.verbose {
		dump, _ := httputil.DumpRequestOut(req, true)
		t.Tracer.Log("Full Request Dump:\n%s", redactSensitive(string(dump)))
	}
	
	resp, err := t.Transport.RoundTrip(req)
	if err != nil {
		t.Tracer.Log("Request Error: %v", err)
		return nil, err
	}
	
	t.Tracer.LogResponse(resp)
	
	// Clone response for dumping
	if t.Tracer.verbose {
		dump, _ := httputil.DumpResponse(resp, true)
		t.Tracer.Log("Full Response Dump:\n%s", redactSensitive(string(dump)))
	}
	
	return resp, nil
}

// redactToken redacts sensitive parts of authorization tokens
func redactToken(value string) string {
	if strings.HasPrefix(value, "Bearer ") {
		token := value[7:]
		if len(token) > 20 {
			return fmt.Sprintf("Bearer %s...%s", token[:10], token[len(token)-10:])
		}
	}
	return value
}

// redactCookieValue redacts sensitive cookie values
func redactCookieValue(value string) string {
	if len(value) > 20 {
		return fmt.Sprintf("%s...%s", value[:8], value[len(value)-8:])
	}
	return value
}

// redactSensitive redacts sensitive information from strings
func redactSensitive(text string) string {
	// Redact access tokens
	if strings.Contains(text, "access_token") {
		var data map[string]interface{}
		if err := json.Unmarshal([]byte(text), &data); err == nil {
			if token, ok := data["access_token"].(string); ok && len(token) > 20 {
				data["access_token"] = fmt.Sprintf("%s...%s", token[:10], token[len(token)-10:])
			}
			if token, ok := data["refresh_token"].(string); ok && len(token) > 20 {
				data["refresh_token"] = fmt.Sprintf("%s...%s", token[:10], token[len(token)-10:])
			}
			if redacted, err := json.Marshal(data); err == nil {
				return string(redacted)
			}
		}
	}
	
	// Redact client secrets
	text = redactPattern(text, "client_secret=", "&")
	text = redactPattern(text, "code=", "&")
	text = redactPattern(text, "code_verifier=", "&")
	
	return text
}

// redactPattern redacts values between start and end patterns
func redactPattern(text, start, end string) string {
	idx := strings.Index(text, start)
	if idx >= 0 {
		startIdx := idx + len(start)
		endIdx := strings.Index(text[startIdx:], end)
		if endIdx < 0 {
			endIdx = len(text) - startIdx
		}
		value := text[startIdx : startIdx+endIdx]
		if len(value) > 10 {
			redacted := fmt.Sprintf("%s...%s", value[:4], value[len(value)-4:])
			text = text[:startIdx] + redacted + text[startIdx+endIdx:]
		}
	}
	return text
}