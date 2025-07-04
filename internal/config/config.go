package config

import "strings"

// Config holds all configuration options for the OData MCP bridge
type Config struct {
	// Service configuration
	ServiceURL string `mapstructure:"service_url"`

	// Authentication
	Username     string            `mapstructure:"username"`
	Password     string            `mapstructure:"password"`
	CookieFile   string            `mapstructure:"cookie_file"`
	CookieString string            `mapstructure:"cookie_string"`
	Cookies      map[string]string // Parsed cookies

	// Tool naming options
	ToolPrefix  string `mapstructure:"tool_prefix"`
	ToolPostfix string `mapstructure:"tool_postfix"`
	NoPostfix   bool   `mapstructure:"no_postfix"`
	ToolShrink  bool   `mapstructure:"tool_shrink"`

	// Entity and function filtering
	Entities         string   `mapstructure:"entities"`
	Functions        string   `mapstructure:"functions"`
	AllowedEntities  []string // Parsed from Entities
	AllowedFunctions []string // Parsed from Functions

	// Output and debugging
	Verbose   bool `mapstructure:"verbose"`
	Debug     bool `mapstructure:"debug"`
	SortTools bool `mapstructure:"sort_tools"`
	Trace     bool `mapstructure:"trace"`
	
	// Response enhancement options
	PaginationHints  bool `mapstructure:"pagination_hints"`   // Add pagination support with hints
	LegacyDates      bool `mapstructure:"legacy_dates"`       // Support epoch timestamp format
	NoLegacyDates    bool `mapstructure:"no_legacy_dates"`    // Disable legacy date format
	VerboseErrors    bool `mapstructure:"verbose_errors"`     // Detailed error context
	ResponseMetadata bool `mapstructure:"response_metadata"`  // Include __metadata in responses
	
	// Response size limits
	MaxResponseSize int `mapstructure:"max_response_size"` // Maximum response size in bytes
	MaxItems        int `mapstructure:"max_items"`         // Maximum number of items in response
	
	// Read-only mode flags
	ReadOnly             bool `mapstructure:"read_only"`               // Read-only mode: hide all modifying operations
	ReadOnlyButFunctions bool `mapstructure:"read_only_but_functions"` // Read-only mode but allow function imports
	
	// Hint configuration
	HintsFile string `mapstructure:"hints_file"` // Path to hints JSON file
	Hint      string `mapstructure:"hint"`       // Direct hint JSON from CLI
	
	// AAD authentication
	AuthAAD      bool   `mapstructure:"auth_aad"`       // Enable AAD authentication
	AADTenant    string `mapstructure:"aad_tenant"`     // Azure AD tenant ID
	AADClientID  string `mapstructure:"aad_client_id"`  // AAD app client ID
	AADScopes    string `mapstructure:"aad_scopes"`     // Comma-separated scopes
	AADCache     string `mapstructure:"aad_cache"`      // Token cache location
	AADBrowser   bool   `mapstructure:"aad_browser"`    // Use browser-based auth flow
	AADTrace     bool   `mapstructure:"aad_trace"`      // Enable auth tracing
	
	// SAML authentication
	AuthSAMLBrowser bool `mapstructure:"auth_saml_browser"` // Use browser for SAML auth
	
	// Windows authentication
	AuthWindows bool `mapstructure:"auth_windows"` // Use Windows integrated auth
	DeferredWindowsAuth bool // Internal flag for deferred auth
	TestAuth bool `mapstructure:"test_auth"` // Test authentication only
	
	// Advanced SAML authentication methods
	AuthWebView2       bool `mapstructure:"auth_webview2"`        // Use WebView2 for SAML
	AuthChrome         bool `mapstructure:"auth_chrome"`          // Use Chrome automation
	AuthChromeHeadless bool `mapstructure:"auth_chrome_headless"` // Use headless Chrome
	AuthChromeManual   bool `mapstructure:"auth_chrome_manual"`   // Use Chrome with manual confirmation
}

// HasBasicAuth returns true if username and password are configured
func (c *Config) HasBasicAuth() bool {
	return c.Username != "" && c.Password != ""
}

// HasCookieAuth returns true if cookies are configured
func (c *Config) HasCookieAuth() bool {
	return len(c.Cookies) > 0
}

// UsePostfix returns true if tool postfix should be used instead of prefix
func (c *Config) UsePostfix() bool {
	return !c.NoPostfix
}

// IsReadOnly returns true if read-only mode is enabled
func (c *Config) IsReadOnly() bool {
	return c.ReadOnly || c.ReadOnlyButFunctions
}

// AllowModifyingFunctions returns true if modifying function imports are allowed
func (c *Config) AllowModifyingFunctions() bool {
	return !c.ReadOnly
}

// HasAADAuth returns true if AAD authentication is configured
func (c *Config) HasAADAuth() bool {
	return c.AuthAAD
}

// GetAADScopes returns the parsed AAD scopes
func (c *Config) GetAADScopes() []string {
	if c.AADScopes == "" {
		return []string{}
	}
	
	var scopes []string
	for _, scope := range strings.Split(c.AADScopes, ",") {
		scope = strings.TrimSpace(scope)
		if scope != "" {
			scopes = append(scopes, scope)
		}
	}
	return scopes
}