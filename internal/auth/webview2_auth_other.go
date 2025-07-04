// +build !windows

package auth

import (
	"context"
	"fmt"
)

// WebView2Auth is not available on non-Windows platforms
type WebView2Auth struct {
	serviceURL string
	verbose    bool
}

// NewWebView2Auth creates a new WebView2 authenticator (non-Windows stub)
func NewWebView2Auth(serviceURL string, verbose bool) *WebView2Auth {
	return &WebView2Auth{
		serviceURL: serviceURL,
		verbose:    verbose,
	}
}

// Authenticate returns an error on non-Windows platforms
func (w *WebView2Auth) Authenticate(ctx context.Context) (map[string]string, error) {
	return nil, fmt.Errorf("WebView2 authentication is only available on Windows")
}

