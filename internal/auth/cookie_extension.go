package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

// CookieExtensionHelper helps with cookie extraction via browser extension
type CookieExtensionHelper struct {
	verbose bool
	server  *http.Server
	cookies chan map[string]string
	error   chan error
}

// NewCookieExtensionHelper creates a new cookie extension helper
func NewCookieExtensionHelper(verbose bool) *CookieExtensionHelper {
	return &CookieExtensionHelper{
		verbose: verbose,
		cookies: make(chan map[string]string, 1),
		error:   make(chan error, 1),
	}
}

// GenerateBookmarklet generates a JavaScript bookmarklet for cookie extraction
func GenerateBookmarklet(serviceURL string, callbackURL string) string {
	// JavaScript code that extracts cookies and sends them to our server
	jsCode := fmt.Sprintf(`
(function() {
    const targetDomain = '%s';
    const callbackURL = '%s';
    
    // Get all cookies for the current domain
    const cookies = document.cookie.split(';').map(c => {
        const [name, value] = c.trim().split('=');
        return { name, value, domain: window.location.hostname };
    }).filter(c => c.name && c.value);
    
    // Send to our local server
    fetch(callbackURL, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ cookies })
    }).then(() => {
        alert('Cookies captured successfully!');
    }).catch(err => {
        alert('Failed to capture cookies: ' + err);
    });
})();
`, serviceURL, callbackURL)

	// Convert to bookmarklet format
	return "javascript:" + jsCode
}

// StartCookieServer starts a local server to receive cookies
func (c *CookieExtensionHelper) StartCookieServer(ctx context.Context) (string, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", fmt.Errorf("failed to start cookie server: %w", err)
	}

	port := listener.Addr().(*net.TCPAddr).Port
	callbackURL := fmt.Sprintf("http://localhost:%d/cookies", port)

	mux := http.NewServeMux()
	
	// CORS headers for browser requests
	mux.HandleFunc("/cookies", func(w http.ResponseWriter, r *http.Request) {
		// Set CORS headers
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		
		if r.Method != "POST" {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Parse cookies from request
		var data struct {
			Cookies []struct {
				Name   string `json:"name"`
				Value  string `json:"value"`
				Domain string `json:"domain"`
			} `json:"cookies"`
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			c.error <- fmt.Errorf("failed to read request body: %w", err)
			http.Error(w, "Failed to read body", http.StatusBadRequest)
			return
		}

		if err := json.Unmarshal(body, &data); err != nil {
			c.error <- fmt.Errorf("failed to parse cookies: %w", err)
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		// Convert to map
		cookies := make(map[string]string)
		for _, cookie := range data.Cookies {
			cookies[cookie.Name] = cookie.Value
			if c.verbose {
				fmt.Fprintf(os.Stderr, "[VERBOSE] Received cookie: %s\n", cookie.Name)
			}
		}

		// Send success response
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]bool{"success": true})

		// Send cookies through channel
		c.cookies <- cookies
	})

	c.server = &http.Server{Handler: mux}

	// Start server in background
	go func() {
		if err := c.server.Serve(listener); err != http.ErrServerClosed {
			c.error <- err
		}
	}()

	return callbackURL, nil
}

// WaitForCookies waits for cookies to be received
func (c *CookieExtensionHelper) WaitForCookies(ctx context.Context, timeout time.Duration) (map[string]string, error) {
	defer c.server.Close()

	select {
	case cookies := <-c.cookies:
		return cookies, nil
	case err := <-c.error:
		return nil, err
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-time.After(timeout):
		return nil, fmt.Errorf("timeout waiting for cookies")
	}
}

// GeneratePowerShellScript generates a PowerShell script for Windows cookie extraction
func GeneratePowerShellScript(serviceURL string) string {
	return fmt.Sprintf(`
# PowerShell script to extract cookies from Edge/Chrome
# Run this in PowerShell after logging into the service

$serviceUrl = "%s"
$domain = ([System.Uri]$serviceUrl).Host

# Try Edge first
$edgeCookiesPath = "$env:LOCALAPPDATA\Microsoft\Edge\User Data\Default\Network\Cookies"
$chromeCookiesPath = "$env:LOCALAPPDATA\Google\Chrome\User Data\Default\Cookies"

if (Test-Path $edgeCookiesPath) {
    Write-Host "Found Edge cookies database"
    # Note: Actual extraction requires decryption of the cookies database
    Write-Host "Please use the browser developer tools to extract cookies manually:"
    Write-Host "1. Press F12 in Edge"
    Write-Host "2. Go to Application > Cookies"
    Write-Host "3. Find cookies for domain: $domain"
    Write-Host "4. Copy MYSAPSSO2 and other SAP cookies"
} elseif (Test-Path $chromeCookiesPath) {
    Write-Host "Found Chrome cookies database"
    Write-Host "Please use the browser developer tools to extract cookies manually:"
    Write-Host "1. Press F12 in Chrome"
    Write-Host "2. Go to Application > Cookies"
    Write-Host "3. Find cookies for domain: $domain"
    Write-Host "4. Copy MYSAPSSO2 and other SAP cookies"
} else {
    Write-Host "No supported browser cookies found"
}
`, serviceURL)
}

// SaveCookiesToFile saves cookies in Netscape format
func SaveCookiesToFile(cookies map[string]string, domain string, filename string) error {
	// Ensure directory exists
	dir := filepath.Dir(filename)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create cookie file: %w", err)
	}
	defer file.Close()

	// Write Netscape format header
	fmt.Fprintln(file, "# Netscape HTTP Cookie File")
	fmt.Fprintln(file, "# This file was generated by odata-mcp")
	fmt.Fprintln(file, "# https://github.com/zmcp/odata-mcp")

	// Write cookies in Netscape format
	// domain, flag, path, secure, expiration, name, value
	expiration := time.Now().Add(24 * time.Hour).Unix()
	for name, value := range cookies {
		fmt.Fprintf(file, "%s\tTRUE\t/\tFALSE\t%d\t%s\t%s\n",
			domain, expiration, name, value)
	}

	return nil
}