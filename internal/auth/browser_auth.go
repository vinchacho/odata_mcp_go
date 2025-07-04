package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/pkg/browser"
)

// BrowserAuth implements OAuth2 authorization code flow with PKCE
type BrowserAuth struct {
	config      *AADConfig
	redirectURI string
	server      *http.Server
	authCode    chan string
	authError   chan error
}

// AuthenticateBrowser performs browser-based authentication
func (a *AADAuthProvider) AuthenticateBrowser(ctx context.Context, scopes []string) (*AADToken, error) {
	// Log to tracer if available
	if a.tracer != nil {
		a.tracer.Log("Starting browser-based authentication")
		a.tracer.Log("Tenant: %s", a.config.TenantID)
		a.tracer.Log("Client ID: %s", a.config.ClientID)
		a.tracer.Log("Scopes: %v", scopes)
	}
	// Check cache first
	cacheKey := strings.Join(scopes, " ")
	if cached, ok := a.cachedTokens[cacheKey]; ok && cached.ExpiresAt.After(time.Now()) {
		if a.verbose {
			fmt.Fprintf(os.Stderr, "[VERBOSE] Using cached AAD token (expires: %s)\n", cached.ExpiresAt.Format(time.RFC3339))
		}
		return cached, nil
	}

	// Set up local HTTP server for redirect
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("failed to start local server: %w", err)
	}
	defer listener.Close()

	port := listener.Addr().(*net.TCPAddr).Port
	redirectURI := fmt.Sprintf("http://localhost:%d/callback", port)

	// Generate PKCE challenge
	codeVerifier := generateCodeVerifier()
	codeChallenge := generateCodeChallenge(codeVerifier)

	// Build auth URL
	authURL := buildAuthURL(a.config, redirectURI, scopes, codeChallenge)
	
	if a.verbose {
		fmt.Fprintf(os.Stderr, "[VERBOSE] Authorization URL: %s\n", authURL)
	}
	
	if a.tracer != nil {
		a.tracer.Log("PKCE Code Verifier: %s", codeVerifier)
		a.tracer.Log("PKCE Code Challenge: %s", codeChallenge)
		a.tracer.Log("Redirect URI: %s", redirectURI)
		a.tracer.Log("Authorization URL: %s", authURL)
	}

	// Set up HTTP server to handle callback
	authCode := make(chan string, 1)
	authError := make(chan error, 1)

	mux := http.NewServeMux()
	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		if a.tracer != nil {
			a.tracer.Log("Received callback request")
			a.tracer.Log("Query params: %s", r.URL.RawQuery)
		}
		
		code := r.URL.Query().Get("code")
		if code == "" {
			errMsg := r.URL.Query().Get("error_description")
			if errMsg == "" {
				errMsg = r.URL.Query().Get("error")
			}
			if a.tracer != nil {
				a.tracer.Log("Authentication error: %s", errMsg)
			}
			authError <- fmt.Errorf("authentication failed: %s", errMsg)
			fmt.Fprintf(w, htmlErrorPage, errMsg)
			return
		}

		if a.tracer != nil {
			a.tracer.Log("Received authorization code: %s...%s", code[:10], code[len(code)-10:])
		}
		authCode <- code
		fmt.Fprintf(w, htmlSuccessPage)
	})

	server := &http.Server{Handler: mux}
	
	// Start server in background
	go func() {
		if err := server.Serve(listener); err != http.ErrServerClosed {
			authError <- err
		}
	}()

	// Open browser
	if a.verbose {
		fmt.Fprintf(os.Stderr, "[VERBOSE] Opening browser for authentication...\n")
	}
	
	fmt.Println("\n=== Azure AD Authentication ===")
	fmt.Println("Opening your browser for authentication...")
	fmt.Printf("If the browser doesn't open, visit:\n%s\n", authURL)
	
	if err := browser.OpenURL(authURL); err != nil {
		fmt.Printf("Failed to open browser: %v\n", err)
	}

	// Wait for callback
	var authorizationCode string
	select {
	case authorizationCode = <-authCode:
		if a.verbose {
			fmt.Fprintf(os.Stderr, "[VERBOSE] Received authorization code\n")
		}
	case err := <-authError:
		server.Close()
		return nil, err
	case <-ctx.Done():
		server.Close()
		return nil, ctx.Err()
	case <-time.After(5 * time.Minute):
		server.Close()
		return nil, fmt.Errorf("authentication timeout")
	}

	// Shutdown server
	server.Close()

	// Exchange code for token
	token, err := a.exchangeCodeForToken(ctx, authorizationCode, redirectURI, codeVerifier, scopes)
	if err != nil {
		return nil, err
	}

	// Cache the token
	a.cachedTokens[cacheKey] = token

	if a.tracer != nil {
		a.tracer.Log("Browser authentication completed successfully")
		a.tracer.Log("Token cached for scopes: %v", scopes)
		a.tracer.Log("Token expires at: %s", token.ExpiresAt.Format(time.RFC3339))
	}

	fmt.Println("Authentication successful!")
	return token, nil
}

// exchangeCodeForToken exchanges authorization code for access token
func (a *AADAuthProvider) exchangeCodeForToken(ctx context.Context, code, redirectURI, codeVerifier string, scopes []string) (*AADToken, error) {
	tokenURL := fmt.Sprintf("https://login.microsoftonline.com/%s/oauth2/v2.0/token", a.config.TenantID)
	
	if a.verbose {
		fmt.Fprintf(os.Stderr, "[VERBOSE] Exchanging authorization code for token\n")
		fmt.Fprintf(os.Stderr, "[VERBOSE] Token URL: %s\n", tokenURL)
		fmt.Fprintf(os.Stderr, "[VERBOSE] Client ID: %s\n", a.config.ClientID)
		fmt.Fprintf(os.Stderr, "[VERBOSE] Redirect URI: %s\n", redirectURI)
		fmt.Fprintf(os.Stderr, "[VERBOSE] Scopes: %v\n", scopes)
	}
	
	data := url.Values{
		"grant_type":    {"authorization_code"},
		"client_id":     {a.config.ClientID},
		"code":          {code},
		"redirect_uri":  {redirectURI},
		"code_verifier": {codeVerifier},
		"scope":         {strings.Join(scopes, " ")},
	}

	resp, err := a.httpClient.PostForm(tokenURL, data)
	if err != nil {
		return nil, fmt.Errorf("token exchange failed: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read token response: %w", err)
	}
	
	if a.verbose {
		fmt.Fprintf(os.Stderr, "[VERBOSE] Token response status: %d\n", resp.StatusCode)
		fmt.Fprintf(os.Stderr, "[VERBOSE] Token response body: %s\n", string(body))
	}
	
	// Check for error response
	if resp.StatusCode != http.StatusOK {
		var errorResp struct {
			Error            string `json:"error"`
			ErrorDescription string `json:"error_description"`
		}
		if err := json.Unmarshal(body, &errorResp); err == nil && errorResp.Error != "" {
			return nil, fmt.Errorf("token exchange error: %s - %s", errorResp.Error, errorResp.ErrorDescription)
		}
		return nil, fmt.Errorf("token exchange failed with status %d: %s", resp.StatusCode, string(body))
	}
	
	// Parse successful response
	var tokenResp struct {
		AccessToken  string `json:"access_token"`
		TokenType    string `json:"token_type"`
		ExpiresIn    int    `json:"expires_in"`
		Scope        string `json:"scope"`
		RefreshToken string `json:"refresh_token,omitempty"`
		IDToken      string `json:"id_token,omitempty"`
	}
	
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return nil, fmt.Errorf("failed to parse token response: %w", err)
	}
	
	if tokenResp.AccessToken == "" {
		return nil, fmt.Errorf("no access token in response")
	}
	
	// Calculate expiration time
	expiresAt := time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)
	
	return &AADToken{
		AccessToken:  tokenResp.AccessToken,
		RefreshToken: tokenResp.RefreshToken,
		ExpiresAt:    expiresAt,
		Scopes:       scopes,
	}, nil
}

func buildAuthURL(config *AADConfig, redirectURI string, scopes []string, codeChallenge string) string {
	params := url.Values{
		"client_id":             {config.ClientID},
		"response_type":         {"code"},
		"redirect_uri":          {redirectURI},
		"response_mode":         {"query"},
		"scope":                 {strings.Join(scopes, " ")},
		"code_challenge":        {codeChallenge},
		"code_challenge_method": {"S256"},
	}

	return fmt.Sprintf("https://login.microsoftonline.com/%s/oauth2/v2.0/authorize?%s",
		config.TenantID, params.Encode())
}

func generateCodeVerifier() string {
	b := make([]byte, 32)
	rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}

func generateCodeChallenge(verifier string) string {
	h := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(h[:])
}

const htmlSuccessPage = `<!DOCTYPE html>
<html>
<head>
    <title>Authentication Successful</title>
    <style>
        body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif; 
               display: flex; align-items: center; justify-content: center; height: 100vh; 
               margin: 0; background: #f0f2f5; }
        .message { text-align: center; padding: 2em; background: white; border-radius: 8px; 
                   box-shadow: 0 2px 4px rgba(0,0,0,0.1); }
        h1 { color: #0078d4; }
        p { color: #323130; }
    </style>
</head>
<body>
    <div class="message">
        <h1>✓ Authentication Successful</h1>
        <p>You can close this window and return to the terminal.</p>
        <script>setTimeout(function() { window.close(); }, 2000);</script>
    </div>
</body>
</html>`

const htmlErrorPage = `<!DOCTYPE html>
<html>
<head>
    <title>Authentication Failed</title>
    <style>
        body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif; 
               display: flex; align-items: center; justify-content: center; height: 100vh; 
               margin: 0; background: #f0f2f5; }
        .message { text-align: center; padding: 2em; background: white; border-radius: 8px; 
                   box-shadow: 0 2px 4px rgba(0,0,0,0.1); max-width: 500px; }
        h1 { color: #d13438; }
        p { color: #323130; }
        .error { background: #fde7e9; padding: 1em; border-radius: 4px; margin-top: 1em; }
    </style>
</head>
<body>
    <div class="message">
        <h1>✗ Authentication Failed</h1>
        <p>There was an error during authentication.</p>
        <div class="error">%s</div>
    </div>
</body>
</html>`