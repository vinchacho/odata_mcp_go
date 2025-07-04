package auth

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/pkg/browser"
)

// SAMLBrowserAuth handles SAML authentication via browser
type SAMLBrowserAuth struct {
	serviceURL string
	verbose    bool
	tracer     *AuthTracer
}

// NewSAMLBrowserAuth creates a new SAML browser authenticator
func NewSAMLBrowserAuth(serviceURL string, verbose bool) *SAMLBrowserAuth {
	return &SAMLBrowserAuth{
		serviceURL: serviceURL,
		verbose:    verbose,
	}
}

// AuthenticateAndExtractCookies performs SAML auth and extracts cookies
func (s *SAMLBrowserAuth) AuthenticateAndExtractCookies(ctx context.Context) (map[string]string, error) {
	if s.verbose {
		fmt.Fprintf(os.Stderr, "[VERBOSE] Starting SAML browser authentication for: %s\n", s.serviceURL)
	}

	// Start local server for cookie capture instructions
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("failed to start local server: %w", err)
	}
	defer listener.Close()

	port := listener.Addr().(*net.TCPAddr).Port
	instructionURL := fmt.Sprintf("http://localhost:%d/", port)

	// Channel to signal when user is ready
	ready := make(chan bool, 1)
	
	// Create HTTP server with instructions
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, s.generateInstructionPage())
	})
	
	mux.HandleFunc("/ready", func(w http.ResponseWriter, r *http.Request) {
		ready <- true
		fmt.Fprint(w, `<html><body><h1>✓ Thank you!</h1><p>You can close this window and return to the terminal.</p></body></html>`)
	})

	server := &http.Server{Handler: mux}
	go server.Serve(listener)
	defer server.Close()

	// Open instruction page
	fmt.Println("\n=== SAML Authentication Required ===")
	fmt.Printf("Opening browser with instructions...\n")
	fmt.Printf("If the browser doesn't open, visit:\n%s\n", instructionURL)
	fmt.Println("=====================================\n")

	if err := browser.OpenURL(instructionURL); err != nil {
		fmt.Printf("Failed to open browser: %v\n", err)
	}

	// Also open the service URL in a new tab
	time.Sleep(2 * time.Second)
	if err := browser.OpenURL(s.serviceURL); err != nil {
		fmt.Printf("Failed to open service URL: %v\n", err)
	}

	// Wait for user to indicate they're ready
	select {
	case <-ready:
		// User clicked "I'm authenticated"
	case <-time.After(5 * time.Minute):
		return nil, fmt.Errorf("authentication timeout")
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	// Provide instructions for manual cookie extraction
	fmt.Println("\n=== Cookie Extraction Instructions ===")
	fmt.Println("Please follow these steps to extract cookies:")
	fmt.Println("1. In the browser tab with the OData service, press F12 to open Developer Tools")
	fmt.Println("2. Go to the 'Application' or 'Storage' tab")
	fmt.Println("3. Find 'Cookies' in the left sidebar and click on it")
	fmt.Println("4. Look for cookies from the service domain")
	fmt.Println("5. Find and copy these cookie values:")
	fmt.Println("   - MYSAPSSO2")
	fmt.Println("   - SAP_SESSIONID (if present)")
	fmt.Println("   - sap-usercontext (if present)")
	fmt.Println("\nThen use one of these methods:")
	fmt.Println("\nMethod 1 - Cookie String:")
	fmt.Printf("odata-mcp --cookie-string \"MYSAPSSO2=<value>; SAP_SESSIONID=<value>\" %s\n", s.serviceURL)
	fmt.Println("\nMethod 2 - Save to file and use:")
	fmt.Println("Save the cookies to a file (one per line as name=value) and run:")
	fmt.Printf("odata-mcp --cookie-file cookies.txt %s\n", s.serviceURL)
	fmt.Println("=====================================")

	// For now, return empty map as manual extraction is required
	return map[string]string{}, nil
}

func (s *SAMLBrowserAuth) generateInstructionPage() string {
	return fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
    <title>SAML Authentication - OData MCP</title>
    <style>
        body {
            font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;
            max-width: 800px;
            margin: 50px auto;
            padding: 20px;
            background: #f0f2f5;
        }
        .container {
            background: white;
            border-radius: 8px;
            box-shadow: 0 2px 4px rgba(0,0,0,0.1);
            padding: 30px;
        }
        h1 { color: #0078d4; }
        .info {
            background: #e3f2fd;
            border-left: 4px solid #0078d4;
            padding: 15px;
            margin: 20px 0;
            border-radius: 4px;
        }
        .step {
            margin: 15px 0;
            padding: 10px;
            background: #f8f9fa;
            border-radius: 4px;
        }
        .button {
            display: inline-block;
            padding: 12px 24px;
            background: #0078d4;
            color: white;
            text-decoration: none;
            border-radius: 4px;
            border: none;
            cursor: pointer;
            font-size: 16px;
            margin: 20px 0;
        }
        .button:hover { background: #106ebe; }
        code {
            background: #e9ecef;
            padding: 2px 6px;
            border-radius: 3px;
            font-family: monospace;
        }
        .warning {
            background: #fff3cd;
            border-left: 4px solid #ffc107;
            padding: 15px;
            margin: 20px 0;
            border-radius: 4px;
        }
    </style>
</head>
<body>
    <div class="container">
        <h1>🔐 SAML Authentication Required</h1>
        
        <div class="info">
            <strong>Service URL:</strong> <code>%s</code>
        </div>
        
        <p>This SAP system uses SAML authentication through your organization's identity provider. 
        A new browser tab will open with the service login page.</p>
        
        <h2>📋 Instructions:</h2>
        
        <div class="step">
            <strong>Step 1:</strong> A new tab should open with the OData service. If not, click 
            <a href="%s" target="_blank">here to open it</a>.
        </div>
        
        <div class="step">
            <strong>Step 2:</strong> Log in with your organizational credentials when prompted.
        </div>
        
        <div class="step">
            <strong>Step 3:</strong> After successful login, you should see either:
            <ul>
                <li>An XML document (the OData service document)</li>
                <li>A JSON response</li>
                <li>An error saying "No handler for Data" (this is normal)</li>
            </ul>
        </div>
        
        <div class="step">
            <strong>Step 4:</strong> Once authenticated, click the button below:
        </div>
        
        <a href="/ready" class="button">✓ I've Successfully Authenticated</a>
        
        <div class="warning">
            <strong>⚠️ Important:</strong> After clicking the button above, you'll need to manually 
            extract cookies from the browser. Instructions will be provided in the terminal.
        </div>
        
        <h3>🔧 Why Manual Cookie Extraction?</h3>
        <p>Due to browser security restrictions, we cannot automatically extract cookies from 
        SAML-authenticated sessions. This is a security feature that protects your credentials.</p>
    </div>
</body>
</html>`, strings.ReplaceAll(s.serviceURL, `"`, `\"`), s.serviceURL)
}

// EnableTracing enables authentication tracing
func (s *SAMLBrowserAuth) EnableTracing() error {
	if s.tracer != nil && s.tracer.enabled {
		return nil
	}
	
	tracer, err := NewAuthTracer(true)
	if err != nil {
		return fmt.Errorf("failed to create tracer: %w", err)
	}
	
	s.tracer = tracer
	return nil
}

// DisableTracing disables authentication tracing
func (s *SAMLBrowserAuth) DisableTracing() error {
	if s.tracer != nil {
		if err := s.tracer.Close(); err != nil {
			return err
		}
		s.tracer = nil
	}
	return nil
}