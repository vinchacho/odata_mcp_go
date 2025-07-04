// +build !windows

package auth

import (
	"context"
	"fmt"
	"runtime"
)

// PowerShellAuth is not available on non-Windows platforms
type PowerShellAuth struct {
	serviceURL string
	verbose    bool
}

// NewPowerShellAuth returns an error on non-Windows platforms
func NewPowerShellAuth(serviceURL string, verbose bool) *PowerShellAuth {
	return &PowerShellAuth{
		serviceURL: serviceURL,
		verbose:    verbose,
	}
}

// Authenticate returns an error on non-Windows platforms
func (p *PowerShellAuth) Authenticate(ctx context.Context) (map[string]string, error) {
	return nil, fmt.Errorf("Windows authentication is only available on Windows (current OS: %s)", runtime.GOOS)
}

// GetPowerShellAuthScript returns an error message on non-Windows platforms
func GetPowerShellAuthScript(serviceURL string) string {
	return fmt.Sprintf("# Windows authentication is only available on Windows (current OS: %s)", runtime.GOOS)
}