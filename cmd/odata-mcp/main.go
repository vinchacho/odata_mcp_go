package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/zmcp/odata-mcp/internal/auth"
	"github.com/zmcp/odata-mcp/internal/bridge"
	"github.com/zmcp/odata-mcp/internal/config"
	"github.com/zmcp/odata-mcp/internal/debug"
	"github.com/zmcp/odata-mcp/internal/transport"
	"github.com/zmcp/odata-mcp/internal/transport/http"
	"github.com/zmcp/odata-mcp/internal/transport/stdio"
)

var cfg *config.Config

var rootCmd = &cobra.Command{
	Use:   "odata-mcp [service-url]",
	Short: "OData to MCP Bridge - Universal OData v2 to Model Context Protocol bridge",
	Long: `OData to MCP Bridge - Universal OData v2 to Model Context Protocol bridge.

This tool creates a bridge between OData v2 services and the Model Context Protocol
(MCP), dynamically generating MCP tools based on OData metadata.

Examples:
  odata-mcp https://services.odata.org/V2/Northwind/Northwind.svc/
  odata-mcp --service https://my-sap-service.com/sap/opu/odata/sap/SERVICE_NAME/
  odata-mcp --user admin --password secret https://my-service.com/odata/
  odata-mcp --cookie-file cookies.txt https://my-service.com/odata/`,
	Args: cobra.MaximumNArgs(1),
	RunE: runBridge,
}

func init() {
	// Load .env file if it exists
	godotenv.Load()

	// Initialize config
	cfg = &config.Config{}

	// Service URL
	rootCmd.Flags().StringVar(&cfg.ServiceURL, "service", "", "URL of the OData service (overrides positional argument and ODATA_SERVICE_URL env var)")

	// Authentication flags (mutually exclusive handled in validation)
	rootCmd.Flags().StringVarP(&cfg.Username, "user", "u", "", "Username for basic authentication (overrides ODATA_USERNAME env var)")
	rootCmd.Flags().StringVarP(&cfg.Password, "password", "p", "", "Password for basic authentication (overrides ODATA_PASSWORD env var)")
	rootCmd.Flags().StringVar(&cfg.Password, "pass", "", "Password for basic authentication (alias for --password)")
	rootCmd.Flags().StringVar(&cfg.CookieFile, "cookie-file", "", "Path to cookie file in Netscape format")
	rootCmd.Flags().StringVar(&cfg.CookieFile, "cookies", "", "Path to cookie file in Netscape format (alias for --cookie-file)")
	rootCmd.Flags().StringVar(&cfg.CookieString, "cookie-string", "", "Cookie string (key1=val1; key2=val2)")

	// Tool naming options
	rootCmd.Flags().StringVar(&cfg.ToolPrefix, "tool-prefix", "", "Custom prefix for tool names (use with --no-postfix)")
	rootCmd.Flags().StringVar(&cfg.ToolPostfix, "tool-postfix", "", "Custom postfix for tool names (default: _for_<service_id>)")
	rootCmd.Flags().BoolVar(&cfg.NoPostfix, "no-postfix", false, "Use prefix instead of postfix for tool naming")
	rootCmd.Flags().BoolVar(&cfg.ToolShrink, "tool-shrink", false, "Use shortened tool names (create_, get_, upd_, del_, search_, filter_)")

	// Entity and function filtering
	rootCmd.Flags().StringVar(&cfg.Entities, "entities", "", "Comma-separated list of entities to generate tools for (e.g., 'Products,Categories,Orders'). Supports wildcards: 'Product*,Order*'")
	rootCmd.Flags().StringVar(&cfg.Functions, "functions", "", "Comma-separated list of function imports to generate tools for (e.g., 'GetProducts,CreateOrder'). Supports wildcards: 'Get*,Create*'")

	// Output and debugging options
	rootCmd.Flags().BoolVarP(&cfg.Verbose, "verbose", "v", false, "Enable verbose output to stderr")
	rootCmd.Flags().BoolVar(&cfg.Debug, "debug", false, "Alias for --verbose")
	rootCmd.Flags().BoolVar(&cfg.SortTools, "sort-tools", true, "Sort tools alphabetically in the output")
	rootCmd.Flags().BoolVar(&cfg.Trace, "trace", false, "Initialize MCP service and print all tools and parameters, then exit (useful for debugging)")
	
	// Response enhancement options
	rootCmd.Flags().BoolVar(&cfg.PaginationHints, "pagination-hints", false, "Add pagination support with suggested_next_call and has_more indicators")
	rootCmd.Flags().BoolVar(&cfg.LegacyDates, "legacy-dates", true, "Support epoch timestamp format (/Date(1234567890000)/) - enabled by default for SAP")
	rootCmd.Flags().BoolVar(&cfg.NoLegacyDates, "no-legacy-dates", false, "Disable legacy date format conversion")
	rootCmd.Flags().BoolVar(&cfg.VerboseErrors, "verbose-errors", false, "Provide detailed error context and debugging information")
	rootCmd.Flags().BoolVar(&cfg.ResponseMetadata, "response-metadata", false, "Include detailed __metadata blocks in entity responses")
	
	// Response size limits
	rootCmd.Flags().IntVar(&cfg.MaxResponseSize, "max-response-size", 5*1024*1024, "Maximum response size in bytes (default: 5MB)")
	rootCmd.Flags().IntVar(&cfg.MaxItems, "max-items", 100, "Maximum number of items in response (default: 100)")
	
	// Read-only mode flags
	rootCmd.Flags().BoolVar(&cfg.ReadOnly, "read-only", false, "Read-only mode: hide all modifying operations (create, update, delete, and functions)")
	rootCmd.Flags().BoolVar(&cfg.ReadOnly, "ro", false, "Read-only mode (shorthand for --read-only)")
	rootCmd.Flags().BoolVar(&cfg.ReadOnlyButFunctions, "read-only-but-functions", false, "Read-only mode but allow function imports")
	rootCmd.Flags().BoolVar(&cfg.ReadOnlyButFunctions, "robf", false, "Read-only but functions (shorthand for --read-only-but-functions)")
	
	// Transport options
	rootCmd.Flags().String("transport", "stdio", "Transport type: 'stdio' or 'http' (SSE)")
	rootCmd.Flags().String("http-addr", ":8080", "HTTP server address (used with --transport http)")
	
	// Debug options
	rootCmd.Flags().Bool("trace-mcp", false, "Enable trace logging to debug MCP communication")
	rootCmd.Flags().BoolVar(&cfg.TestAuth, "test-auth", false, "Test authentication only (exit after auth)")
	
	// Hint options
	rootCmd.Flags().StringVar(&cfg.HintsFile, "hints-file", "", "Path to hints JSON file (defaults to hints.json in same directory as binary)")
	rootCmd.Flags().StringVar(&cfg.Hint, "hint", "", "Direct hint JSON or text to inject into service info")
	
	// AAD authentication options
	rootCmd.Flags().BoolVar(&cfg.AuthAAD, "auth-aad", false, "Use Azure AD authentication")
	rootCmd.Flags().StringVar(&cfg.AADTenant, "aad-tenant", "common", "Azure AD tenant ID (default: common)")
	rootCmd.Flags().StringVar(&cfg.AADClientID, "aad-client-id", "", "Azure AD application (client) ID")
	rootCmd.Flags().StringVar(&cfg.AADScopes, "aad-scopes", "", "Comma-separated OAuth2 scopes (default: service URL + /.default)")
	rootCmd.Flags().StringVar(&cfg.AADCache, "aad-cache", "", "Token cache location (default: OS secure storage)")
	rootCmd.Flags().BoolVar(&cfg.AADBrowser, "aad-browser", false, "Use browser-based authentication instead of device code")
	rootCmd.Flags().BoolVar(&cfg.AADTrace, "aad-trace", false, "Enable detailed authentication tracing for debugging")
	rootCmd.Flags().BoolVar(&cfg.AuthSAMLBrowser, "auth-saml-browser", false, "Use browser for SAML authentication (shows manual cookie extraction steps)")
	rootCmd.Flags().BoolVar(&cfg.AuthWindows, "auth-windows", false, "Use Windows integrated authentication with PowerShell (Windows only)")
	rootCmd.Flags().BoolVar(&cfg.AuthWebView2, "auth-webview2", false, "Use WebView2 (Edge) for SAML authentication (Windows only)")
	rootCmd.Flags().BoolVar(&cfg.AuthChrome, "auth-chrome", false, "Use Chrome automation for SAML authentication")
	rootCmd.Flags().BoolVar(&cfg.AuthChromeHeadless, "auth-chrome-headless", false, "Use headless Chrome for SAML authentication")
	rootCmd.Flags().BoolVar(&cfg.AuthChromeManual, "auth-chrome-manual", false, "Use Chrome with manual confirmation for SAML authentication")

	// Bind flags to viper for environment variable support
	viper.BindPFlag("service", rootCmd.Flags().Lookup("service"))
	viper.BindPFlag("username", rootCmd.Flags().Lookup("user"))
	viper.BindPFlag("password", rootCmd.Flags().Lookup("password"))
	viper.BindPFlag("verbose", rootCmd.Flags().Lookup("verbose"))

	// Set up environment variable mapping
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	viper.AutomaticEnv()
	viper.SetEnvPrefix("ODATA")
}

func runBridge(cmd *cobra.Command, args []string) error {
	// Handle --debug as alias for --verbose
	if cfg.Debug {
		cfg.Verbose = true
	}
	
	// Handle legacy dates flags
	if cfg.NoLegacyDates {
		cfg.LegacyDates = false
		if cfg.Verbose {
			fmt.Fprintf(os.Stderr, "[VERBOSE] Legacy date format conversion disabled.\n")
		}
	} else if !cmd.Flags().Changed("legacy-dates") {
		// Default to legacy dates for SAP compatibility
		cfg.LegacyDates = true
		if cfg.Verbose {
			fmt.Fprintf(os.Stderr, "[VERBOSE] Legacy date format enabled by default for SAP compatibility. Use --no-legacy-dates to disable.\n")
		}
	}
	
	// Handle read-only mode flags
	if cfg.ReadOnly && cfg.ReadOnlyButFunctions {
		return fmt.Errorf("cannot use both --read-only and --read-only-but-functions flags at the same time")
	}
	
	if cfg.IsReadOnly() {
		if cfg.Verbose {
			if cfg.ReadOnly {
				fmt.Fprintf(os.Stderr, "[VERBOSE] Read-only mode enabled. All modifying operations (create, update, delete, and functions) will be hidden.\n")
			} else if cfg.ReadOnlyButFunctions {
				fmt.Fprintf(os.Stderr, "[VERBOSE] Read-only mode enabled with function exception. Create, update, and delete operations will be hidden, but function imports will be available.\n")
			}
		}
	}

	// Determine service URL with priority: --service flag > positional arg > env vars
	if cfg.ServiceURL == "" && len(args) > 0 {
		cfg.ServiceURL = args[0]
		if cfg.Verbose {
			fmt.Fprintf(os.Stderr, "[VERBOSE] Using OData service URL from positional argument.\n")
		}
	}

	if cfg.ServiceURL == "" {
		cfg.ServiceURL = viper.GetString("URL")
		if cfg.ServiceURL == "" {
			cfg.ServiceURL = viper.GetString("SERVICE_URL")
		}
		if cfg.ServiceURL != "" && cfg.Verbose {
			fmt.Fprintf(os.Stderr, "[VERBOSE] Using ODATA_URL from environment.\n")
		}
	}

	if cfg.ServiceURL == "" {
		return fmt.Errorf("OData service URL not provided. Use --service flag, positional argument, or ODATA_URL environment variable")
	}

	// Validate and process authentication
	if err := processAuthentication(cfg); err != nil {
		return err
	}

	// Parse entity and function filters
	if cfg.Entities != "" {
		cfg.AllowedEntities = parseCommaSeparated(cfg.Entities)
		if cfg.Verbose {
			fmt.Fprintf(os.Stderr, "[VERBOSE] Filtering tools to only these entities: %v\n", cfg.AllowedEntities)
		}
	}

	if cfg.Functions != "" {
		cfg.AllowedFunctions = parseCommaSeparated(cfg.Functions)
		if cfg.Verbose {
			fmt.Fprintf(os.Stderr, "[VERBOSE] Filtering tools to only these functions: %v\n", cfg.AllowedFunctions)
		}
	}

	// Set up signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Create and initialize bridge
	odataBridge, err := bridge.NewODataMCPBridge(cfg)
	if err != nil {
		return fmt.Errorf("failed to create OData MCP bridge: %w", err)
	}

	// Handle trace mode
	if cfg.Trace {
		return printTraceInfo(odataBridge)
	}
	
	// Set up transport based on flag
	transportType, _ := cmd.Flags().GetString("transport")
	
	// Get the MCP server from the bridge
	mcpServer := odataBridge.GetServer()
	if mcpServer == nil {
		return fmt.Errorf("failed to get MCP server from bridge")
	}
	
	// Set up tracing if requested
	enableTrace, _ := cmd.Flags().GetBool("trace-mcp")
	var tracer *debug.TraceLogger
	if enableTrace {
		tracer, err = debug.NewTraceLogger(true)
		if err != nil {
			fmt.Fprintf(os.Stderr, "[ERROR] Failed to create trace logger: %v\n", err)
		} else {
			defer tracer.Close()
			fmt.Fprintf(os.Stderr, "[TRACE] Trace logging enabled. Output file: %s\n", tracer.GetFilename())
		}
	}
	
	// Create handler function that delegates to the MCP server
	handler := func(ctx context.Context, msg *transport.Message) (*transport.Message, error) {
		return mcpServer.HandleMessage(ctx, msg)
	}
	
	var trans transport.Transport
	switch transportType {
	case "http", "sse":
		httpAddr, _ := cmd.Flags().GetString("http-addr")
		if cfg.Verbose {
			fmt.Fprintf(os.Stderr, "[VERBOSE] Starting HTTP/SSE transport on %s\n", httpAddr)
		}
		trans = http.NewSSE(httpAddr, handler)
	case "stdio":
		fallthrough
	default:
		if cfg.Verbose {
			fmt.Fprintf(os.Stderr, "[VERBOSE] Using stdio transport\n")
		}
		stdioTrans := stdio.New(handler)
		if tracer != nil {
			stdioTrans.SetTracer(tracer)
		}
		trans = stdioTrans
	}
	
	// Set transport on the MCP server
	mcpServer.SetTransport(trans)

	// Start bridge in a goroutine
	errChan := make(chan error, 1)
	go func() {
		errChan <- odataBridge.Run()
	}()

	// Wait for signal or error
	select {
	case sig := <-sigChan:
		fmt.Fprintf(os.Stderr, "\n%s received, shutting down server...\n", sig)
		odataBridge.Stop()
		return nil
	case err := <-errChan:
		return err
	}
}

func processAuthentication(cfg *config.Config) error {
	// Check for mutually exclusive authentication options
	authMethods := 0
	if cfg.CookieFile != "" {
		authMethods++
	}
	if cfg.CookieString != "" {
		authMethods++
	}
	if cfg.Username != "" {
		authMethods++
	}
	if cfg.AuthAAD {
		authMethods++
	}
	if cfg.AuthSAMLBrowser {
		authMethods++
	}
	if cfg.AuthWindows {
		authMethods++
	}
	if cfg.AuthWebView2 {
		authMethods++
	}
	if cfg.AuthChrome {
		authMethods++
	}
	if cfg.AuthChromeHeadless {
		authMethods++
	}
	if cfg.AuthChromeManual {
		authMethods++
	}

	if authMethods > 1 {
		return fmt.Errorf("only one authentication method can be used at a time")
	}
	
	// Handle AAD authentication
	if cfg.AuthAAD {
		return processAADAuthentication(cfg)
	}
	
	// Handle SAML browser authentication
	if cfg.AuthSAMLBrowser {
		return processSAMLBrowserAuthentication(cfg)
	}
	
	// Handle Windows integrated authentication
	if cfg.AuthWindows {
		return processWindowsAuthentication(cfg)
	}
	
	// Handle WebView2 authentication
	if cfg.AuthWebView2 {
		return processWebView2Authentication(cfg)
	}
	
	// Handle Chrome authentication
	if cfg.AuthChrome || cfg.AuthChromeHeadless || cfg.AuthChromeManual {
		return processChromeAuthentication(cfg)
	}

	// Process cookie file authentication
	if cfg.CookieFile != "" {
		if _, err := os.Stat(cfg.CookieFile); os.IsNotExist(err) {
			return fmt.Errorf("cookie file not found: %s", cfg.CookieFile)
		}

		cookies, err := loadCookiesFromFile(cfg.CookieFile)
		if err != nil {
			return fmt.Errorf("failed to load cookies from file: %w", err)
		}

		cfg.Cookies = cookies
		if cfg.Verbose {
			fmt.Fprintf(os.Stderr, "[VERBOSE] Loaded %d cookies from file: %s\n", len(cookies), cfg.CookieFile)
		}
	} else if cfg.CookieString != "" {
		// Process cookie string authentication
		cookies := parseCookieString(cfg.CookieString)
		if len(cookies) == 0 {
			return fmt.Errorf("failed to parse cookie string")
		}

		cfg.Cookies = cookies
		if cfg.Verbose {
			fmt.Fprintf(os.Stderr, "[VERBOSE] Parsed %d cookies from string\n", len(cookies))
		}
	} else {
		// Handle basic authentication from environment if not provided via flags
		if cfg.Username == "" {
			cfg.Username = viper.GetString("USER")
			if cfg.Username == "" {
				cfg.Username = viper.GetString("USERNAME")
			}
		}

		if cfg.Password == "" {
			cfg.Password = viper.GetString("PASS")
			if cfg.Password == "" {
				cfg.Password = viper.GetString("PASSWORD")
			}
		}

		// Check for cookie environment variables if no auth is configured
		if cfg.Username == "" {
			envCookieFile := viper.GetString("COOKIE_FILE")
			envCookieString := viper.GetString("COOKIE_STRING")

			if envCookieFile != "" {
				if _, err := os.Stat(envCookieFile); err == nil {
					cookies, err := loadCookiesFromFile(envCookieFile)
					if err == nil {
						cfg.Cookies = cookies
						if cfg.Verbose {
							fmt.Fprintf(os.Stderr, "[VERBOSE] Loaded %d cookies from environment ODATA_COOKIE_FILE\n", len(cookies))
						}
					}
				}
			} else if envCookieString != "" {
				cookies := parseCookieString(envCookieString)
				if len(cookies) > 0 {
					cfg.Cookies = cookies
					if cfg.Verbose {
						fmt.Fprintf(os.Stderr, "[VERBOSE] Parsed %d cookies from environment ODATA_COOKIE_STRING\n", len(cookies))
					}
				}
			}
		}

		// Set up basic auth if credentials are available
		if cfg.Username != "" && cfg.Password != "" {
			if cfg.Verbose {
				fmt.Fprintf(os.Stderr, "[VERBOSE] Using basic authentication for user: %s\n", cfg.Username)
			}
		} else if cfg.Verbose && len(cfg.Cookies) == 0 {
			fmt.Fprintf(os.Stderr, "[VERBOSE] No authentication provided or configured. Attempting anonymous access.\n")
		}
	}

	return nil
}

func loadCookiesFromFile(cookieFile string) (map[string]string, error) {
	cookies := make(map[string]string)

	file, err := os.Open(cookieFile)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Parse Netscape format (7 fields separated by tabs)
		parts := strings.Split(line, "\t")
		if len(parts) >= 7 {
			// domain, flag, path, secure, expiration, name, value
			name := parts[5]
			value := parts[6]
			cookies[name] = value
		} else if strings.Contains(line, "=") {
			// Simple key=value format fallback
			kv := strings.SplitN(line, "=", 2)
			if len(kv) == 2 {
				cookies[strings.TrimSpace(kv[0])] = strings.TrimSpace(kv[1])
			}
		}
	}

	return cookies, scanner.Err()
}

func parseCookieString(cookieString string) map[string]string {
	cookies := make(map[string]string)
	for _, cookie := range strings.Split(cookieString, ";") {
		cookie = strings.TrimSpace(cookie)
		if strings.Contains(cookie, "=") {
			kv := strings.SplitN(cookie, "=", 2)
			if len(kv) == 2 {
				cookies[strings.TrimSpace(kv[0])] = strings.TrimSpace(kv[1])
			}
		}
	}
	return cookies
}

func parseCommaSeparated(input string) []string {
	var result []string
	for _, item := range strings.Split(input, ",") {
		item = strings.TrimSpace(item)
		if item != "" {
			result = append(result, item)
		}
	}
	return result
}

func printTraceInfo(bridge *bridge.ODataMCPBridge) error {
	fmt.Println(strings.Repeat("=", 80))
	fmt.Println("🔍 OData MCP Bridge Trace Information")
	fmt.Println(strings.Repeat("=", 80))

	info, err := bridge.GetTraceInfo()
	if err != nil {
		return fmt.Errorf("failed to get trace info: %w", err)
	}

	// Print trace information as JSON for now
	// TODO: Implement pretty printing like the Python version
	data, err := json.MarshalIndent(info, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal trace info: %w", err)
	}

	fmt.Println(string(data))

	fmt.Println("\n" + strings.Repeat("=", 80))
	fmt.Println("✅ Trace complete - MCP bridge initialized successfully but not started")
	fmt.Println("💡 Use without --trace to start the actual MCP server")
	fmt.Println(strings.Repeat("=", 80))

	return nil
}

func processAADAuthentication(cfg *config.Config) error {
	// Set default client ID if not provided
	if cfg.AADClientID == "" {
		// This is a well-known client ID for Azure CLI / development
		// In production, users should register their own app
		cfg.AADClientID = "04b07795-8ddb-461a-bbee-02f9e1bf7b46" // Azure CLI client ID
		if cfg.Verbose {
			fmt.Fprintf(os.Stderr, "[VERBOSE] Using default Azure CLI client ID for AAD authentication\n")
		}
	}
	
	// Create AAD config
	aadConfig := &auth.AADConfig{
		TenantID:      cfg.AADTenant,
		ClientID:      cfg.AADClientID,
		CacheLocation: cfg.AADCache,
	}
	
	// Validate AAD config
	if err := aadConfig.Validate(); err != nil {
		return fmt.Errorf("invalid AAD configuration: %w", err)
	}
	
	// Set scopes - use provided or default to service URL
	scopes := cfg.GetAADScopes()
	if len(scopes) == 0 {
		scopes = aadConfig.GetDefaultScopes(cfg.ServiceURL)
		if cfg.Verbose {
			fmt.Fprintf(os.Stderr, "[VERBOSE] Using default AAD scope: %v\n", scopes)
		}
	}
	aadConfig.Scopes = scopes
	
	// Create AAD auth provider
	authProvider, err := auth.NewAADAuthProvider(aadConfig, cfg.Verbose)
	if err != nil {
		return fmt.Errorf("failed to create AAD auth provider: %w", err)
	}
	
	// Enable tracing if requested
	if cfg.AADTrace {
		if err := authProvider.EnableTracing(); err != nil {
			return fmt.Errorf("failed to enable auth tracing: %w", err)
		}
		defer authProvider.DisableTracing()
	}
	
	// Perform authentication
	ctx := context.Background()
	var token *auth.AADToken
	
	if cfg.AADBrowser {
		if cfg.Verbose {
			fmt.Fprintf(os.Stderr, "[VERBOSE] Using browser-based authentication flow\n")
		}
		token, err = authProvider.AuthenticateBrowser(ctx, scopes)
	} else {
		if cfg.Verbose {
			fmt.Fprintf(os.Stderr, "[VERBOSE] Using device code authentication flow\n")
		}
		token, err = authProvider.AuthenticateDeviceCode(ctx, scopes)
	}
	
	if err != nil {
		return fmt.Errorf("AAD authentication failed: %w", err)
	}
	
	// Exchange token for SAP cookies
	sapCookies, err := authProvider.ExchangeTokenForSAPCookies(ctx, token, cfg.ServiceURL)
	if err != nil {
		return fmt.Errorf("failed to exchange AAD token for SAP cookies: %w", err)
	}
	
	// Set cookies in config
	cfg.Cookies = sapCookies.Cookies
	
	if cfg.Verbose {
		fmt.Fprintf(os.Stderr, "[VERBOSE] AAD authentication successful. Acquired %d cookies\n", len(cfg.Cookies))
		if sapCookies.MYSAPSSO2 != "" {
			fmt.Fprintf(os.Stderr, "[VERBOSE] MYSAPSSO2 cookie acquired\n")
		}
	}
	
	return nil
}

func processSAMLBrowserAuthentication(cfg *config.Config) error {
	if cfg.Verbose {
		fmt.Fprintf(os.Stderr, "[VERBOSE] Using SAML browser authentication\n")
	}
	
	// Create SAML browser authenticator
	samlAuth := auth.NewSAMLBrowserAuth(cfg.ServiceURL, cfg.Verbose)
	
	// Enable tracing if requested
	if cfg.AADTrace {
		if err := samlAuth.EnableTracing(); err != nil {
			return fmt.Errorf("failed to enable tracing: %w", err)
		}
		defer samlAuth.DisableTracing()
	}
	
	// Perform authentication and get instructions
	ctx := context.Background()
	_, err := samlAuth.AuthenticateAndExtractCookies(ctx)
	if err != nil {
		return fmt.Errorf("SAML browser authentication failed: %w", err)
	}
	
	// Exit after showing instructions
	fmt.Println("\nPlease run odata-mcp again with the extracted cookies.")
	os.Exit(0)
	return nil
}

func processWindowsAuthentication(cfg *config.Config) error {
	if cfg.Verbose {
		fmt.Fprintf(os.Stderr, "[VERBOSE] Using Windows integrated authentication\n")
	}
	
	// Check if we're in test-auth mode
	if cfg.TestAuth {
		// Perform authentication immediately for testing
		fmt.Fprintf(os.Stderr, "[INFO] Running authentication test...\n")
		
		psAuth := auth.NewPowerShellAuth(cfg.ServiceURL, cfg.Verbose)
		ctx := context.Background()
		
		startTime := time.Now()
		cookies, err := psAuth.Authenticate(ctx)
		duration := time.Since(startTime)
		
		if err != nil {
			fmt.Fprintf(os.Stderr, "\n[ERROR] Authentication failed after %v: %v\n", duration, err)
			return err
		}
		
		fmt.Fprintf(os.Stderr, "\n[SUCCESS] Authentication completed in %v\n", duration)
		fmt.Fprintf(os.Stderr, "Acquired %d cookies:\n", len(cookies))
		for name, value := range cookies {
			// Show first and last 10 chars of cookie value for security
			displayValue := value
			if len(value) > 20 {
				displayValue = value[:10] + "..." + value[len(value)-10:]
			}
			fmt.Fprintf(os.Stderr, "  - %s: %s\n", name, displayValue)
		}
		
		// Optionally save cookies
		if cfg.CookieFile != "" {
			fmt.Fprintf(os.Stderr, "\nSaving cookies to: %s\n", cfg.CookieFile)
			if err := auth.SaveCookiesToFile(cookies, cfg.ServiceURL, cfg.CookieFile); err != nil {
				fmt.Fprintf(os.Stderr, "[ERROR] Failed to save cookies: %v\n", err)
			} else {
				fmt.Fprintf(os.Stderr, "[SUCCESS] Cookies saved successfully\n")
				fmt.Fprintf(os.Stderr, "\nYou can now use:\n")
				fmt.Fprintf(os.Stderr, "odata-mcp --cookies \"%s\" --service \"%s\"\n", cfg.CookieFile, cfg.ServiceURL)
			}
		} else {
			fmt.Fprintf(os.Stderr, "\nTo save cookies, use: --test-auth --cookies <file>\n")
		}
		
		// Exit after test
		os.Exit(0)
	}
	
	// For MCP compatibility, we need to handle auth AFTER initialization
	// So we'll just validate here and do actual auth when first tool is called
	fmt.Fprintf(os.Stderr, "[INFO] Windows authentication will be performed on first request\n")
	
	// Mark that we need Windows auth
	cfg.DeferredWindowsAuth = true
	
	return nil
}

func processWebView2Authentication(cfg *config.Config) error {
	if cfg.Verbose {
		fmt.Fprintf(os.Stderr, "[VERBOSE] Using WebView2 (Edge) authentication\n")
	}
	
	
	// Try simple WebView2 implementation
	webviewAuth := auth.NewSimpleWebView2Auth(cfg.ServiceURL, cfg.Verbose)
	
	// Perform authentication
	ctx := context.Background()
	cookies, err := webviewAuth.Authenticate(ctx)
	if err != nil {
		return fmt.Errorf("WebView2 authentication failed: %w", err)
	}
	
	// Set cookies in config
	cfg.Cookies = cookies
	
	if cfg.Verbose {
		fmt.Fprintf(os.Stderr, "[VERBOSE] WebView2 authentication successful. Acquired %d cookies\n", len(cookies))
		if _, ok := cookies["MYSAPSSO2"]; ok {
			fmt.Fprintf(os.Stderr, "[VERBOSE] MYSAPSSO2 cookie acquired\n")
		}
	}
	
	// Check if we're in test mode
	if cfg.TestAuth {
		fmt.Fprintf(os.Stderr, "\n[SUCCESS] Authentication test completed\n")
		fmt.Fprintf(os.Stderr, "Acquired %d cookies:\n", len(cookies))
		for name, value := range cookies {
			displayValue := value
			if len(value) > 20 {
				displayValue = value[:10] + "..." + value[len(value)-10:]
			}
			fmt.Fprintf(os.Stderr, "  - %s: %s\n", name, displayValue)
		}
		os.Exit(0)
	}
	
	return nil
}

func processChromeAuthentication(cfg *config.Config) error {
	if cfg.AuthChromeManual {
		if cfg.Verbose {
			fmt.Fprintf(os.Stderr, "[VERBOSE] Using Chrome manual authentication\n")
		}
		
		// Create manual Chrome authenticator
		chromeAuth := auth.NewChromeManualAuth(cfg.ServiceURL, cfg.Verbose)
		
		// Perform authentication
		ctx := context.Background()
		cookies, err := chromeAuth.Authenticate(ctx)
		if err != nil {
			return fmt.Errorf("Chrome manual authentication failed: %w", err)
		}
		
		// Set cookies in config
		cfg.Cookies = cookies
		
		if cfg.Verbose {
			fmt.Fprintf(os.Stderr, "[VERBOSE] Chrome manual authentication successful. Acquired %d cookies\n", len(cookies))
			if _, ok := cookies["MYSAPSSO2"]; ok {
				fmt.Fprintf(os.Stderr, "[VERBOSE] MYSAPSSO2 cookie acquired\n")
			}
		}
		
		// Handle test mode
		if cfg.TestAuth {
			handleTestAuthSuccess(cfg, cookies)
		}
		
		return nil
	}
	
	// Regular Chrome automation
	headless := cfg.AuthChromeHeadless
	
	if cfg.Verbose {
		if headless {
			fmt.Fprintf(os.Stderr, "[VERBOSE] Using headless Chrome authentication\n")
		} else {
			fmt.Fprintf(os.Stderr, "[VERBOSE] Using Chrome authentication (visible browser)\n")
		}
	}
	
	// Create ChromeDP authenticator
	chromeAuth := auth.NewChromeDPAuth(cfg.ServiceURL, cfg.Verbose, headless)
	
	// Perform authentication
	ctx := context.Background()
	cookies, err := chromeAuth.Authenticate(ctx)
	if err != nil {
		return fmt.Errorf("Chrome authentication failed: %w", err)
	}
	
	// Set cookies in config
	cfg.Cookies = cookies
	
	if cfg.Verbose {
		fmt.Fprintf(os.Stderr, "[VERBOSE] Chrome authentication successful. Acquired %d cookies\n", len(cookies))
		if _, ok := cookies["MYSAPSSO2"]; ok {
			fmt.Fprintf(os.Stderr, "[VERBOSE] MYSAPSSO2 cookie acquired\n")
		}
	}
	
	// Check if we're in test mode
	if cfg.TestAuth {
		handleTestAuthSuccess(cfg, cookies)
	}
	
	return nil
}

func handleTestAuthSuccess(cfg *config.Config, cookies map[string]string) {
	fmt.Fprintf(os.Stderr, "\n[SUCCESS] Authentication test completed\n")
	fmt.Fprintf(os.Stderr, "Acquired %d cookies:\n", len(cookies))
	for name, value := range cookies {
		displayValue := value
		if len(value) > 20 {
			displayValue = value[:10] + "..." + value[len(value)-10:]
		}
		fmt.Fprintf(os.Stderr, "  - %s: %s\n", name, displayValue)
	}
	
	// Optionally save cookies
	if cfg.CookieFile != "" {
		fmt.Fprintf(os.Stderr, "\nSaving cookies to: %s\n", cfg.CookieFile)
		// Get domain from service URL
		domain := cfg.ServiceURL
		if u, err := url.Parse(cfg.ServiceURL); err == nil {
			domain = u.Host
		}
		if err := auth.SaveCookiesToFile(cookies, domain, cfg.CookieFile); err != nil {
			fmt.Fprintf(os.Stderr, "[ERROR] Failed to save cookies: %v\n", err)
		} else {
			fmt.Fprintf(os.Stderr, "[SUCCESS] Cookies saved successfully\n")
			fmt.Fprintf(os.Stderr, "\nYou can now use:\n")
			fmt.Fprintf(os.Stderr, "odata-mcp --cookies \"%s\" --service \"%s\"\n", cfg.CookieFile, cfg.ServiceURL)
		}
	}
	
	os.Exit(0)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "\n--- FATAL ERROR ---\n")
		fmt.Fprintf(os.Stderr, "An unexpected error occurred: %v\n", err)
		fmt.Fprintf(os.Stderr, "-------------------\n")
		os.Exit(1)
	}
}