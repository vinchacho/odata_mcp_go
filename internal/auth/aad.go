package auth

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/AzureAD/microsoft-authentication-library-for-go/apps/public"
)

// AADAuthProvider handles Azure AD authentication
type AADAuthProvider struct {
	config       *AADConfig
	msalClient   public.Client
	httpClient   *http.Client
	cachedTokens map[string]*AADToken // Cache by scope
	verbose      bool
	tracer       *AuthTracer
}

// AADToken represents an AAD access token with metadata
type AADToken struct {
	AccessToken  string
	RefreshToken string
	ExpiresAt    time.Time
	Scopes       []string
}

// SAPCookies represents the cookies needed for SAP authentication
type SAPCookies struct {
	MYSAPSSO2      string
	SAPSessionID   string
	SAPUserContext string
	CSRFToken      string
	Cookies        map[string]string // All cookies for the session
}

// NewAADAuthProvider creates a new AAD authentication provider
func NewAADAuthProvider(config *AADConfig, verbose bool) (*AADAuthProvider, error) {
	// Create MSAL public client
	clientOptions := []public.Option{
		public.WithAuthority(fmt.Sprintf("https://login.microsoftonline.com/%s", config.TenantID)),
	}
	
	// Add cache if specified
	if config.CacheLocation != "" {
		// TODO: Implement persistent cache
		// For now, we'll use in-memory cache
	}
	
	client, err := public.New(config.ClientID, clientOptions...)
	if err != nil {
		return nil, fmt.Errorf("failed to create MSAL client: %w", err)
	}
	
	provider := &AADAuthProvider{
		config:       config,
		msalClient:   client,
		httpClient:   &http.Client{Timeout: 30 * time.Second},
		cachedTokens: make(map[string]*AADToken),
		verbose:      verbose,
	}
	
	return provider, nil
}

// AuthenticateDeviceCode performs device code flow authentication
func (a *AADAuthProvider) AuthenticateDeviceCode(ctx context.Context, scopes []string) (*AADToken, error) {
	// Check cache first
	cacheKey := strings.Join(scopes, " ")
	if cached, ok := a.cachedTokens[cacheKey]; ok && cached.ExpiresAt.After(time.Now()) {
		if a.verbose {
			fmt.Fprintf(os.Stderr, "[VERBOSE] Using cached AAD token (expires: %s)\n", cached.ExpiresAt.Format(time.RFC3339))
		}
		return cached, nil
	}
	
	// Try silent authentication first (uses MSAL cache)
	accounts, err := a.msalClient.Accounts(ctx)
	if err == nil && len(accounts) > 0 {
		result, err := a.msalClient.AcquireTokenSilent(ctx, scopes, public.WithSilentAccount(accounts[0]))
		if err == nil {
			token := &AADToken{
				AccessToken: result.AccessToken,
				ExpiresAt:   result.ExpiresOn,
				Scopes:      scopes,
			}
			a.cachedTokens[cacheKey] = token
			if a.verbose {
				fmt.Fprintf(os.Stderr, "[VERBOSE] Acquired AAD token silently\n")
			}
			return token, nil
		}
	}
	
	// Perform device code flow
	deviceCode, err := a.msalClient.AcquireTokenByDeviceCode(ctx, scopes)
	if err != nil {
		return nil, fmt.Errorf("failed to initiate device code flow: %w", err)
	}
	
	// Display device code to user
	fmt.Println("\n=== Azure AD Authentication Required ===")
	fmt.Printf("To sign in, use a web browser to open the page %s\n", deviceCode.Result.VerificationURL)
	fmt.Printf("Enter the code: %s\n", deviceCode.Result.UserCode)
	fmt.Println("Waiting for authentication...")
	fmt.Println("=======================================\n")
	
	// Wait for user to authenticate
	result, err := deviceCode.AuthenticationResult(ctx)
	if err != nil {
		return nil, fmt.Errorf("device code authentication failed: %w", err)
	}
	
	token := &AADToken{
		AccessToken: result.AccessToken,
		ExpiresAt:   result.ExpiresOn,
		Scopes:      scopes,
	}
	
	// Cache the token
	a.cachedTokens[cacheKey] = token
	
	if a.verbose {
		fmt.Fprintf(os.Stderr, "[VERBOSE] AAD authentication successful. Token expires: %s\n", result.ExpiresOn.Format(time.RFC3339))
	}
	
	return token, nil
}

// ExchangeTokenForSAPCookies exchanges an AAD token for SAP cookies
func (a *AADAuthProvider) ExchangeTokenForSAPCookies(ctx context.Context, token *AADToken, serviceURL string) (*SAPCookies, error) {
	// This is a complex process that depends on how the SAP system is configured
	// for AAD federation. Common approaches:
	
	// 1. SAML assertion endpoint
	// 2. OAuth2 token exchange
	// 3. Custom federation endpoint
	
	// For now, we'll implement a generic approach that should work with
	// SAP systems configured for AAD federation via SAML
	
	if a.verbose {
		fmt.Fprintf(os.Stderr, "[VERBOSE] Exchanging AAD token for SAP cookies...\n")
	}
	
	// Try to access the service with the Bearer token
	// SAP might redirect us through SAML flow and set cookies
	req, err := http.NewRequestWithContext(ctx, "GET", serviceURL, nil)
	if err != nil {
		return nil, err
	}
	
	// Add the AAD token as Bearer token
	req.Header.Set("Authorization", "Bearer "+token.AccessToken)
	req.Header.Set("Accept", "application/json")
	
	// Create a cookie jar to capture cookies
	cookieJar := make(map[string]string)
	
	// Custom transport to capture all cookies
	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
	}
	
	// Wrap with tracing if available
	var clientTransport http.RoundTripper = transport
	if a.tracer != nil {
		clientTransport = &TraceRoundTripper{
			Transport: transport,
			Tracer:    a.tracer,
		}
	}
	
	client := &http.Client{
		Transport: clientTransport,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			// Capture cookies from redirects
			if len(via) > 0 {
				for _, cookie := range via[len(via)-1].Response.Cookies() {
					cookieJar[cookie.Name] = cookie.Value
					if a.verbose {
						fmt.Fprintf(os.Stderr, "[VERBOSE] Captured cookie from redirect: %s\n", cookie.Name)
					}
				}
			}
			return nil
		},
		Timeout: 30 * time.Second,
	}
	
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange token: %w", err)
	}
	defer resp.Body.Close()
	
	// Capture cookies from final response
	for _, cookie := range resp.Cookies() {
		cookieJar[cookie.Name] = cookie.Value
		if a.verbose {
			fmt.Fprintf(os.Stderr, "[VERBOSE] Captured cookie: %s\n", cookie.Name)
		}
	}
	
	// Also check for CSRF token in headers
	csrfToken := resp.Header.Get("X-CSRF-Token")
	if csrfToken == "" {
		csrfToken = resp.Header.Get("x-csrf-token")
	}
	
	// Extract SAP-specific cookies
	sapCookies := &SAPCookies{
		Cookies: cookieJar,
	}
	
	// Look for specific SAP cookies
	if mysapsso2, ok := cookieJar["MYSAPSSO2"]; ok {
		sapCookies.MYSAPSSO2 = mysapsso2
	}
	
	// Look for SAP session ID (might have different patterns)
	for name, value := range cookieJar {
		if strings.Contains(name, "SAP_SESSIONID") {
			sapCookies.SAPSessionID = value
		}
		if strings.Contains(name, "sap-usercontext") {
			sapCookies.SAPUserContext = value
		}
	}
	
	if csrfToken != "" {
		sapCookies.CSRFToken = csrfToken
	}
	
	// If we didn't get MYSAPSSO2, we might need to follow a different flow
	if sapCookies.MYSAPSSO2 == "" {
		// Try alternate approach - check if response indicates SAML redirect
		if resp.StatusCode == 302 || resp.StatusCode == 303 {
			location := resp.Header.Get("Location")
			if strings.Contains(location, "saml") || strings.Contains(location, "adfs") {
				return nil, fmt.Errorf("SAP system requires SAML authentication flow - not yet implemented")
			}
		}
		
		if a.verbose {
			fmt.Fprintf(os.Stderr, "[VERBOSE] Warning: MYSAPSSO2 cookie not found. Authentication may not be complete.\n")
		}
	}
	
	if a.verbose {
		fmt.Fprintf(os.Stderr, "[VERBOSE] Token exchange complete. Captured %d cookies\n", len(cookieJar))
	}
	
	return sapCookies, nil
}

// RefreshToken refreshes an AAD token if needed
func (a *AADAuthProvider) RefreshToken(ctx context.Context, token *AADToken) (*AADToken, error) {
	// Check if token needs refresh (5 minutes before expiry)
	if token.ExpiresAt.After(time.Now().Add(5 * time.Minute)) {
		return token, nil
	}
	
	if a.verbose {
		fmt.Fprintf(os.Stderr, "[VERBOSE] Token expired or expiring soon, refreshing...\n")
	}
	
	// Use MSAL's silent token acquisition which handles refresh
	accounts, err := a.msalClient.Accounts(ctx)
	if err != nil || len(accounts) == 0 {
		// No cached account, need to re-authenticate
		return a.AuthenticateDeviceCode(ctx, token.Scopes)
	}
	
	result, err := a.msalClient.AcquireTokenSilent(ctx, token.Scopes, public.WithSilentAccount(accounts[0]))
	if err != nil {
		// Silent refresh failed, need to re-authenticate
		return a.AuthenticateDeviceCode(ctx, token.Scopes)
	}
	
	newToken := &AADToken{
		AccessToken: result.AccessToken,
		ExpiresAt:   result.ExpiresOn,
		Scopes:      token.Scopes,
	}
	
	// Update cache
	cacheKey := strings.Join(token.Scopes, " ")
	a.cachedTokens[cacheKey] = newToken
	
	return newToken, nil
}

// ClearCache clears all cached tokens
func (a *AADAuthProvider) ClearCache() error {
	a.cachedTokens = make(map[string]*AADToken)
	
	// Clear MSAL cache
	accounts, err := a.msalClient.Accounts(context.Background())
	if err != nil {
		return err
	}
	
	for _, account := range accounts {
		err := a.msalClient.RemoveAccount(context.Background(), account)
		if err != nil && a.verbose {
			fmt.Fprintf(os.Stderr, "[VERBOSE] Failed to remove account from cache: %v\n", err)
		}
	}
	
	return nil
}

// EnableTracing enables authentication tracing
func (a *AADAuthProvider) EnableTracing() error {
	if a.tracer != nil && a.tracer.enabled {
		return nil // Already enabled
	}
	
	tracer, err := NewAuthTracer(true)
	if err != nil {
		return fmt.Errorf("failed to create auth tracer: %w", err)
	}
	
	a.tracer = tracer
	
	// Also wrap the http client with tracing
	if a.httpClient.Transport == nil {
		a.httpClient.Transport = http.DefaultTransport
	}
	
	a.httpClient.Transport = &TraceRoundTripper{
		Transport: a.httpClient.Transport,
		Tracer:    tracer,
	}
	
	if a.verbose {
		fmt.Fprintf(os.Stderr, "[VERBOSE] Authentication tracing enabled\n")
	}
	
	return nil
}

// DisableTracing disables authentication tracing
func (a *AADAuthProvider) DisableTracing() error {
	if a.tracer != nil {
		if err := a.tracer.Close(); err != nil {
			return err
		}
		a.tracer = nil
	}
	
	// Unwrap the tracing transport
	if rt, ok := a.httpClient.Transport.(*TraceRoundTripper); ok {
		a.httpClient.Transport = rt.Transport
	}
	
	return nil
}