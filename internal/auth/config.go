package auth

import (
	"fmt"
	"strings"
)

// AADConfig holds Azure AD authentication configuration
type AADConfig struct {
	// TenantID is the Azure AD tenant ID (e.g., "contoso.onmicrosoft.com" or GUID)
	// Use "common" for multi-tenant applications
	TenantID string
	
	// ClientID is the application (client) ID from app registration
	ClientID string
	
	// Scopes are the permissions requested (e.g., ["https://sapserver.com/.default"])
	Scopes []string
	
	// CacheLocation is the path for token cache storage (optional)
	CacheLocation string
	
	// Authority URL (optional, defaults to public cloud)
	Authority string
}

// DefaultAADConfig returns a default AAD configuration
func DefaultAADConfig() *AADConfig {
	return &AADConfig{
		TenantID: "common",
		ClientID: "", // Must be provided
		Scopes:   []string{},
	}
}

// Validate checks if the AAD configuration is valid
func (c *AADConfig) Validate() error {
	if c.TenantID == "" {
		return fmt.Errorf("tenant ID is required")
	}
	
	if c.ClientID == "" {
		return fmt.Errorf("client ID is required")
	}
	
	// Validate client ID format (basic check for GUID)
	if !isValidGUID(c.ClientID) && !strings.Contains(c.ClientID, "-") {
		return fmt.Errorf("client ID must be a valid GUID")
	}
	
	return nil
}

// GetAuthority returns the authority URL for the tenant
func (c *AADConfig) GetAuthority() string {
	if c.Authority != "" {
		return c.Authority
	}
	
	// Default to public cloud
	return fmt.Sprintf("https://login.microsoftonline.com/%s", c.TenantID)
}

// GetDefaultScopes returns default scopes for the service URL
func (c *AADConfig) GetDefaultScopes(serviceURL string) []string {
	if len(c.Scopes) > 0 {
		return c.Scopes
	}
	
	// Extract the base URL for the default scope
	// For SAP systems, this often includes the system ID
	baseURL := extractBaseURL(serviceURL)
	
	// Default scope for resource
	return []string{baseURL + "/.default"}
}

// Helper function to validate GUID format
func isValidGUID(s string) bool {
	// Basic GUID validation - 8-4-4-4-12 format
	parts := strings.Split(s, "-")
	if len(parts) != 5 {
		return false
	}
	
	expectedLengths := []int{8, 4, 4, 4, 12}
	for i, part := range parts {
		if len(part) != expectedLengths[i] {
			return false
		}
		// Check if all characters are hexadecimal
		for _, c := range part {
			if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
				return false
			}
		}
	}
	
	return true
}

// Helper function to extract base URL
func extractBaseURL(serviceURL string) string {
	// Remove protocol
	url := strings.TrimPrefix(serviceURL, "https://")
	url = strings.TrimPrefix(url, "http://")
	
	// Get just the host and port
	if idx := strings.Index(url, "/"); idx > 0 {
		url = url[:idx]
	}
	
	return "https://" + url
}