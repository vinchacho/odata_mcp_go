package auth

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// TokenCache provides file-based token caching
type TokenCache struct {
	Tokens map[string]CachedToken `json:"tokens"`
}

// CachedToken represents a cached AAD token
type CachedToken struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token,omitempty"`
	ExpiresAt    time.Time `json:"expires_at"`
	Scopes       []string  `json:"scopes"`
	TenantID     string    `json:"tenant_id"`
	ClientID     string    `json:"client_id"`
}

// LoadTokenCache loads tokens from cache file
func LoadTokenCache(cacheFile string) (*TokenCache, error) {
	if cacheFile == "" {
		return &TokenCache{Tokens: make(map[string]CachedToken)}, nil
	}
	
	data, err := os.ReadFile(cacheFile)
	if err != nil {
		if os.IsNotExist(err) {
			// Create new cache if file doesn't exist
			return &TokenCache{Tokens: make(map[string]CachedToken)}, nil
		}
		return nil, fmt.Errorf("failed to read token cache: %w", err)
	}
	
	var cache TokenCache
	if err := json.Unmarshal(data, &cache); err != nil {
		return nil, fmt.Errorf("failed to parse token cache: %w", err)
	}
	
	if cache.Tokens == nil {
		cache.Tokens = make(map[string]CachedToken)
	}
	
	return &cache, nil
}

// Save saves the token cache to file
func (tc *TokenCache) Save(cacheFile string) error {
	if cacheFile == "" {
		return nil
	}
	
	// Ensure directory exists
	dir := filepath.Dir(cacheFile)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed to create cache directory: %w", err)
	}
	
	data, err := json.MarshalIndent(tc, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal token cache: %w", err)
	}
	
	// Write with restricted permissions
	if err := os.WriteFile(cacheFile, data, 0600); err != nil {
		return fmt.Errorf("failed to write token cache: %w", err)
	}
	
	return nil
}

// GetToken retrieves a token from cache
func (tc *TokenCache) GetToken(key string) (*CachedToken, bool) {
	token, ok := tc.Tokens[key]
	if !ok {
		return nil, false
	}
	
	// Check if token is expired
	if token.ExpiresAt.Before(time.Now()) {
		delete(tc.Tokens, key)
		return nil, false
	}
	
	return &token, true
}

// SetToken stores a token in cache
func (tc *TokenCache) SetToken(key string, token CachedToken) {
	tc.Tokens[key] = token
}

// Clear removes all tokens from cache
func (tc *TokenCache) Clear() {
	tc.Tokens = make(map[string]CachedToken)
}

// GetDefaultCacheLocation returns the default cache file location
func GetDefaultCacheLocation() string {
	// Get user's home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	
	// Use XDG cache directory on Linux/macOS, AppData on Windows
	if os.Getenv("XDG_CACHE_HOME") != "" {
		return filepath.Join(os.Getenv("XDG_CACHE_HOME"), "odata-mcp", "tokens.json")
	}
	
	switch {
	case os.Getenv("APPDATA") != "": // Windows
		return filepath.Join(os.Getenv("APPDATA"), "odata-mcp", "tokens.json")
	default: // Linux/macOS
		return filepath.Join(homeDir, ".cache", "odata-mcp", "tokens.json")
	}
}