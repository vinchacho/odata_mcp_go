// +build !windows

package auth

import (
	"context"
	"fmt"
)

// SimpleWebView2Auth is not available on non-Windows platforms
type SimpleWebView2Auth struct {
	serviceURL string
	verbose    bool
}

// NewSimpleWebView2Auth creates a new simple WebView2 authenticator (non-Windows stub)
func NewSimpleWebView2Auth(serviceURL string, verbose bool) *SimpleWebView2Auth {
	return &SimpleWebView2Auth{
		serviceURL: serviceURL,
		verbose:    verbose,
	}
}

// Authenticate returns an error on non-Windows platforms
func (s *SimpleWebView2Auth) Authenticate(ctx context.Context) (map[string]string, error) {
	return nil, fmt.Errorf("WebView2 authentication is only available on Windows")
}