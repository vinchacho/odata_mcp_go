// +build windows

package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// PowerShellAuth uses PowerShell for Windows integrated authentication
type PowerShellAuth struct {
	serviceURL string
	verbose    bool
}

// NewPowerShellAuth creates a new PowerShell authenticator
func NewPowerShellAuth(serviceURL string, verbose bool) *PowerShellAuth {
	return &PowerShellAuth{
		serviceURL: serviceURL,
		verbose:    verbose,
	}
}

// Authenticate uses PowerShell to authenticate and get cookies
func (p *PowerShellAuth) Authenticate(ctx context.Context) (map[string]string, error) {
	if p.verbose {
		fmt.Fprintf(os.Stderr, "[VERBOSE] Starting Windows integrated authentication via PowerShell\n")
	}
	
	// PowerShell script that handles authentication
	script := fmt.Sprintf(`
# Use Windows integrated authentication
$ErrorActionPreference = 'Continue'
$serviceUrl = '%s'
$uri = [System.Uri]$serviceUrl
$baseUrl = $uri.Scheme + "://" + $uri.Host

# Create session
$session = New-Object Microsoft.PowerShell.Commands.WebRequestSession

# First request will trigger auth flow
try {
    Write-Output "Authenticating to: $serviceUrl"
    $response = Invoke-WebRequest -Uri $serviceUrl -UseDefaultCredentials -SessionVariable session -MaximumRedirection 20 -TimeoutSec 30 -ErrorAction Stop
    Write-Output "Authentication successful"
} catch {
    # Check if we got cookies even with an error
    $statusCode = $_.Exception.Response.StatusCode.value__
    Write-Output "Response status: $statusCode"
    
    # Some SAP systems return 401 even after setting cookies
    if ($session.Cookies.Count -eq 0) {
        Write-Error "Authentication failed: $_"
        exit 1
    }
}

# Extract all cookies from the session
$cookies = @{}
$allCookies = $session.Cookies.GetCookies($baseUrl)

Write-Output "Found $($allCookies.Count) cookies"
foreach ($cookie in $allCookies) {
    $cookies[$cookie.Name] = $cookie.Value
    Write-Output "  - $($cookie.Name)"
}

# Also check for cookies on the specific path
$pathCookies = $session.Cookies.GetCookies($serviceUrl)
foreach ($cookie in $pathCookies) {
    if (-not $cookies.ContainsKey($cookie.Name)) {
        $cookies[$cookie.Name] = $cookie.Value
        Write-Output "  - $($cookie.Name) (path-specific)"
    }
}

# Output as JSON
$cookies | ConvertTo-Json -Compress
`, p.serviceURL)

	// Create temp script file
	tempDir := os.TempDir()
	scriptFile := filepath.Join(tempDir, "odata_auth.ps1")
	if err := os.WriteFile(scriptFile, []byte(script), 0600); err != nil {
		return nil, err
	}
	defer os.Remove(scriptFile)

	// Execute PowerShell with timeout
	timeoutCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	
	cmd := exec.CommandContext(timeoutCtx, "powershell.exe",
		"-NoProfile",
		"-NonInteractive", 
		"-ExecutionPolicy", "Bypass",
		"-File", scriptFile)

	output, err := cmd.CombinedOutput()
	if err != nil {
		if p.verbose {
			fmt.Fprintf(os.Stderr, "[VERBOSE] PowerShell output:\n%s\n", string(output))
		}
		return nil, fmt.Errorf("PowerShell authentication failed: %w", err)
	}

	// Find JSON in output (PowerShell might output other text)
	outputStr := string(output)
	jsonStart := strings.LastIndex(outputStr, "{")
	if jsonStart == -1 {
		if p.verbose {
			fmt.Fprintf(os.Stderr, "[VERBOSE] PowerShell output:\n%s\n", outputStr)
		}
		return nil, fmt.Errorf("no JSON output from PowerShell")
	}
	
	jsonStr := outputStr[jsonStart:]
	
	// Parse JSON output
	var cookies map[string]string
	if err := json.Unmarshal([]byte(jsonStr), &cookies); err != nil {
		if p.verbose {
			fmt.Fprintf(os.Stderr, "[VERBOSE] Failed to parse JSON: %s\n", jsonStr)
		}
		return nil, fmt.Errorf("failed to parse cookies: %w", err)
	}

	if len(cookies) == 0 {
		return nil, fmt.Errorf("no cookies captured")
	}

	if p.verbose {
		fmt.Fprintf(os.Stderr, "[VERBOSE] PowerShell extracted %d cookies\n", len(cookies))
	}

	return cookies, nil
}

// GetAuthScript returns a standalone PowerShell script for manual use
func GetPowerShellAuthScript(serviceURL string) string {
	return fmt.Sprintf(`
# PowerShell script to authenticate and save cookies
# Run this script as: powershell.exe -ExecutionPolicy Bypass -File auth_sap.ps1

$serviceUrl = '%s'
$cookieFile = Join-Path $env:USERPROFILE '.odata-mcp\cookies.txt'

Write-Host "Authenticating to: $serviceUrl"

# Create session with Windows credentials
$session = New-Object Microsoft.PowerShell.Commands.WebRequestSession

try {
    # Invoke request with integrated auth
    $response = Invoke-WebRequest -Uri $serviceUrl -UseDefaultCredentials -SessionVariable session -MaximumRedirection 10
    
    Write-Host "✓ Authentication successful!"
} catch {
    Write-Host "✗ Authentication failed: $_"
    exit 1
}

# Extract cookies
$cookieLines = @()
$domain = ([System.Uri]$serviceUrl).Host

foreach ($cookie in $session.Cookies.GetCookies($serviceUrl)) {
    # Netscape cookie format
    $line = "$domain" + [char]9 + "TRUE" + [char]9 + "/" + [char]9 + "FALSE" + [char]9 + "0" + [char]9 + $cookie.Name + [char]9 + $cookie.Value
    $cookieLines += $line
    Write-Host "  Found cookie: $($cookie.Name)"
}

# Save to file
$cookieDir = Split-Path $cookieFile -Parent
if (!(Test-Path $cookieDir)) {
    New-Item -ItemType Directory -Path $cookieDir -Force | Out-Null
}

# Write Netscape format header
"# Netscape HTTP Cookie File" | Out-File $cookieFile -Encoding UTF8
"# Generated by odata-mcp PowerShell auth" | Out-File $cookieFile -Append -Encoding UTF8
$cookieLines | Out-File $cookieFile -Append -Encoding UTF8

Write-Host ""
Write-Host "✓ Cookies saved to: $cookieFile"
Write-Host ""
Write-Host "Use with: odata-mcp --cookies '$cookieFile' --service '$serviceUrl'"
`, serviceURL)
}