// +build windows

package auth

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/jchv/go-webview2"
)

// SimpleWebView2Auth provides a simpler WebView2 implementation
type SimpleWebView2Auth struct {
	serviceURL string
	verbose    bool
	cookies    map[string]string
	mu         sync.Mutex
	done       chan bool
}

// NewSimpleWebView2Auth creates a new simple WebView2 authenticator
func NewSimpleWebView2Auth(serviceURL string, verbose bool) *SimpleWebView2Auth {
	return &SimpleWebView2Auth{
		serviceURL: serviceURL,
		verbose:    verbose,
		cookies:    make(map[string]string),
		done:       make(chan bool, 1),
	}
}

// Authenticate performs SAML authentication using WebView2
func (s *SimpleWebView2Auth) Authenticate(ctx context.Context) (map[string]string, error) {
	if s.verbose {
		fmt.Fprintf(os.Stderr, "[VERBOSE] Starting simple WebView2 authentication\n")
		fmt.Fprintf(os.Stderr, "[VERBOSE] Note: Please complete authentication in the popup window\n")
	}

	// Create a simple webview
	w := webview2.New(false)
	if w == nil {
		return nil, fmt.Errorf("failed to create WebView2 instance. Please ensure Edge WebView2 Runtime is installed")
	}
	defer w.Destroy()

	// Set basic properties
	w.SetTitle("SAP Authentication - OData MCP")
	w.SetSize(1024, 768, webview2.HintNone)

	// Bind a function to receive cookies from JavaScript
	w.Bind("notifyCookies", func(cookieString string) {
		if s.verbose {
			fmt.Fprintf(os.Stderr, "[VERBOSE] Received cookies from JavaScript\n")
		}
		
		// Parse cookies
		s.mu.Lock()
		for _, cookie := range strings.Split(cookieString, ";") {
			cookie = strings.TrimSpace(cookie)
			if cookie == "" {
				continue
			}
			parts := strings.SplitN(cookie, "=", 2)
			if len(parts) == 2 {
				name := strings.TrimSpace(parts[0])
				value := strings.TrimSpace(parts[1])
				s.cookies[name] = value
				if s.verbose {
					fmt.Fprintf(os.Stderr, "[VERBOSE] Cookie: %s\n", name)
				}
			}
		}
		s.mu.Unlock()
		
		// Check for MYSAPSSO2
		if _, ok := s.cookies["MYSAPSSO2"]; ok {
			s.done <- true
		}
	})

	// Inject cookie monitoring script
	w.Init(`
		console.log('Cookie monitor initialized');
		
		// Function to get all cookies
		function getAllCookies() {
			return document.cookie;
		}
		
		// Monitor for changes
		let lastCookies = '';
		setInterval(function() {
			let currentCookies = getAllCookies();
			if (currentCookies !== lastCookies) {
				console.log('Cookies changed:', currentCookies);
				lastCookies = currentCookies;
				
				// Send to Go
				if (window.notifyCookies) {
					window.notifyCookies(currentCookies);
				}
				
				// Check for MYSAPSSO2
				if (currentCookies.includes('MYSAPSSO2')) {
					console.log('MYSAPSSO2 cookie detected!');
				}
			}
		}, 1000);
		
		// Also check on page load
		window.addEventListener('load', function() {
			let cookies = getAllCookies();
			if (cookies && window.notifyCookies) {
				window.notifyCookies(cookies);
			}
		});
	`)

	// Navigate to the service URL
	w.Navigate(s.serviceURL)

	// Run in a separate goroutine
	go func() {
		w.Run()
	}()

	// Wait for authentication or timeout
	select {
	case <-s.done:
		w.Terminate()
		
		s.mu.Lock()
		result := make(map[string]string)
		for k, v := range s.cookies {
			result[k] = v
		}
		s.mu.Unlock()
		
		if len(result) == 0 {
			return nil, fmt.Errorf("no cookies captured")
		}
		
		return result, nil
		
	case <-ctx.Done():
		w.Terminate()
		return nil, ctx.Err()
		
	case <-time.After(5 * time.Minute):
		w.Terminate()
		return nil, fmt.Errorf("authentication timeout")
	}
}