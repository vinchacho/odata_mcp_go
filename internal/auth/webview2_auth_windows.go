// +build windows

package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/jchv/go-webview2"
)

// WebView2Auth handles SAML authentication using Edge WebView2
type WebView2Auth struct {
	serviceURL string
	verbose    bool
	cookies    map[string]*http.Cookie
	mu         sync.Mutex
	done       chan bool
	error      error
}

// NewWebView2Auth creates a new WebView2 authenticator
func NewWebView2Auth(serviceURL string, verbose bool) *WebView2Auth {
	return &WebView2Auth{
		serviceURL: serviceURL,
		verbose:    verbose,
		cookies:    make(map[string]*http.Cookie),
		done:       make(chan bool, 1),
	}
}

// Authenticate performs SAML authentication using WebView2
func (w *WebView2Auth) Authenticate(ctx context.Context) (map[string]string, error) {
	if w.verbose {
		fmt.Fprintf(os.Stderr, "[VERBOSE] Starting WebView2 SAML authentication\n")
	}

	// Create WebView2 instance
	webView := webview2.New(w.verbose)
	defer webView.Destroy()

	// Set window title and size
	webView.SetTitle("OData MCP - SAML Authentication")
	webView.SetSize(1024, 768, webview2.HintNone)

	// Navigate to service URL
	if w.verbose {
		fmt.Fprintf(os.Stderr, "[VERBOSE] Navigating to: %s\n", w.serviceURL)
	}
	webView.Navigate(w.serviceURL)

	// Inject JavaScript to monitor authentication
	webView.Init(`
		// Monitor for successful authentication
		(function() {
			let checkInterval = setInterval(function() {
				// Check if we're back at the service URL or have SAP cookies
				if (window.location.href.includes('` + strings.Split(w.serviceURL, "?")[0] + `') || 
					document.cookie.includes('MYSAPSSO2')) {
					
					// Extract all cookies
					let cookies = {};
					document.cookie.split(';').forEach(function(cookie) {
						let parts = cookie.trim().split('=');
						if (parts.length >= 2) {
							cookies[parts[0]] = parts.slice(1).join('=');
						}
					});
					
					// Send cookies to Go
					window.sendCookies(JSON.stringify(cookies));
					clearInterval(checkInterval);
				}
			}, 1000);
			
			// Also monitor navigation events
			let originalPushState = history.pushState;
			let originalReplaceState = history.replaceState;
			
			history.pushState = function() {
				originalPushState.apply(history, arguments);
				window.checkAuth();
			};
			
			history.replaceState = function() {
				originalReplaceState.apply(history, arguments);
				window.checkAuth();
			};
			
			window.addEventListener('popstate', window.checkAuth);
		})();
	`)

	// Bind JavaScript functions
	webView.Bind("sendCookies", func(cookiesJSON string) {
		var cookies map[string]string
		if err := json.Unmarshal([]byte(cookiesJSON), &cookies); err == nil {
			w.mu.Lock()
			for name, value := range cookies {
				w.cookies[name] = &http.Cookie{
					Name:  name,
					Value: value,
				}
			}
			w.mu.Unlock()
			
			// Check for MYSAPSSO2
			if _, ok := cookies["MYSAPSSO2"]; ok {
				if w.verbose {
					fmt.Fprintf(os.Stderr, "[VERBOSE] Found MYSAPSSO2 cookie, authentication successful\n")
				}
				w.done <- true
			}
		}
	})

	webView.Bind("checkAuth", func() {
		webView.Eval(`
			if (document.cookie.includes('MYSAPSSO2')) {
				let cookies = {};
				document.cookie.split(';').forEach(function(cookie) {
					let parts = cookie.trim().split('=');
					if (parts.length >= 2) {
						cookies[parts[0]] = parts.slice(1).join('=');
					}
				});
				window.sendCookies(JSON.stringify(cookies));
			}
		`)
	})

	// Run WebView2 in a goroutine
	go func() {
		webView.Run()
	}()

	// Wait for authentication or timeout
	select {
	case <-w.done:
		webView.Terminate()
		
		// Convert cookies to map
		result := make(map[string]string)
		w.mu.Lock()
		for name, cookie := range w.cookies {
			result[name] = cookie.Value
			if w.verbose {
				fmt.Fprintf(os.Stderr, "[VERBOSE] Extracted cookie: %s\n", name)
			}
		}
		w.mu.Unlock()
		
		if len(result) == 0 {
			return nil, fmt.Errorf("no cookies captured")
		}
		
		return result, nil
		
	case <-ctx.Done():
		webView.Terminate()
		return nil, ctx.Err()
		
	case <-time.After(5 * time.Minute):
		webView.Terminate()
		return nil, fmt.Errorf("authentication timeout")
	}
}

