package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/pkg/browser"
)

// BrowserCookieExtractor handles browser-based authentication with cookie extraction
type BrowserCookieExtractor struct {
	serviceURL string
	verbose    bool
	server     *http.Server
	cookies    chan map[string]string
	authError  chan error
}

// NewBrowserCookieExtractor creates a new browser cookie extractor
func NewBrowserCookieExtractor(serviceURL string, verbose bool) *BrowserCookieExtractor {
	return &BrowserCookieExtractor{
		serviceURL: serviceURL,
		verbose:    verbose,
		cookies:    make(chan map[string]string, 1),
		authError:  make(chan error, 1),
	}
}

// ExtractCookies opens browser for authentication and extracts cookies
func (b *BrowserCookieExtractor) ExtractCookies(ctx context.Context) (map[string]string, error) {
	// Parse service URL to get the host
	serviceURL, err := url.Parse(b.serviceURL)
	if err != nil {
		return nil, fmt.Errorf("invalid service URL: %w", err)
	}

	// Set up local HTTP server for cookie capture
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("failed to start local server: %w", err)
	}
	defer listener.Close()

	port := listener.Addr().(*net.TCPAddr).Port
	captureURL := fmt.Sprintf("http://localhost:%d/capture", port)

	// Create HTTP server to capture cookies
	mux := http.NewServeMux()
	
	// Serve the cookie capture page
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		html := fmt.Sprintf(cookieCaptureHTML, b.serviceURL, serviceURL.Host, captureURL)
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, html)
	})

	// Handle cookie submission
	mux.HandleFunc("/capture", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Parse the JSON body
		var data struct {
			Cookies []struct {
				Name   string `json:"name"`
				Value  string `json:"value"`
				Domain string `json:"domain"`
			} `json:"cookies"`
		}

		if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
			b.authError <- fmt.Errorf("failed to parse cookies: %w", err)
			http.Error(w, "Invalid data", http.StatusBadRequest)
			return
		}

		// Convert to map
		cookies := make(map[string]string)
		for _, cookie := range data.Cookies {
			cookies[cookie.Name] = cookie.Value
			if b.verbose {
				fmt.Fprintf(os.Stderr, "[VERBOSE] Captured cookie: %s (domain: %s)\n", cookie.Name, cookie.Domain)
			}
		}

		// Send success response
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "success"})

		// Send cookies through channel
		b.cookies <- cookies
	})

	b.server = &http.Server{Handler: mux}

	// Start server in background
	go func() {
		if err := b.server.Serve(listener); err != http.ErrServerClosed {
			b.authError <- err
		}
	}()

	// Open browser to the capture page
	capturePageURL := fmt.Sprintf("http://localhost:%d/", port)
	
	fmt.Println("\n=== Browser Authentication Required ===")
	fmt.Printf("Opening browser for authentication...\n")
	fmt.Printf("If the browser doesn't open, visit:\n%s\n", capturePageURL)
	fmt.Println("=======================================\n")

	if err := browser.OpenURL(capturePageURL); err != nil {
		fmt.Printf("Failed to open browser: %v\n", err)
	}

	// Wait for cookies or error
	select {
	case cookies := <-b.cookies:
		b.server.Close()
		
		// Check for important SAP cookies
		important := []string{"MYSAPSSO2", "SAP_SESSIONID", "sap-usercontext"}
		found := false
		for _, name := range important {
			if _, ok := cookies[name]; ok {
				found = true
				break
			}
		}
		
		if !found && b.verbose {
			fmt.Fprintf(os.Stderr, "[VERBOSE] Warning: No SAP authentication cookies found\n")
		}
		
		fmt.Println("Authentication successful! Cookies captured.")
		return cookies, nil
		
	case err := <-b.authError:
		b.server.Close()
		return nil, err
		
	case <-ctx.Done():
		b.server.Close()
		return nil, ctx.Err()
		
	case <-time.After(10 * time.Minute):
		b.server.Close()
		return nil, fmt.Errorf("authentication timeout")
	}
}

// HTML template for the cookie capture page
const cookieCaptureHTML = `<!DOCTYPE html>
<html>
<head>
    <title>OData MCP - Browser Authentication</title>
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
        .step {
            margin: 20px 0;
            padding: 15px;
            background: #f8f9fa;
            border-left: 4px solid #0078d4;
            border-radius: 4px;
        }
        .button {
            display: inline-block;
            padding: 10px 20px;
            background: #0078d4;
            color: white;
            text-decoration: none;
            border-radius: 4px;
            border: none;
            cursor: pointer;
            font-size: 16px;
            margin: 10px 0;
        }
        .button:hover { background: #106ebe; }
        .button:disabled {
            background: #ccc;
            cursor: not-allowed;
        }
        #status {
            margin-top: 20px;
            padding: 15px;
            border-radius: 4px;
            display: none;
        }
        #status.success {
            background: #d4edda;
            color: #155724;
            border: 1px solid #c3e6cb;
        }
        #status.error {
            background: #f8d7da;
            color: #721c24;
            border: 1px solid #f5c6cb;
        }
        .code {
            background: #e9ecef;
            padding: 2px 6px;
            border-radius: 3px;
            font-family: monospace;
        }
        #cookieList {
            max-height: 200px;
            overflow-y: auto;
            font-family: monospace;
            font-size: 12px;
            background: #f8f9fa;
            padding: 10px;
            border-radius: 4px;
            margin: 10px 0;
        }
        .cookie-item {
            margin: 2px 0;
            word-break: break-all;
        }
        .important-cookie {
            font-weight: bold;
            color: #0078d4;
        }
    </style>
</head>
<body>
    <div class="container">
        <h1>🔐 OData MCP - Browser Authentication</h1>
        
        <p>This page will help you authenticate with the OData service and capture the necessary cookies.</p>
        
        <div class="step">
            <h3>Step 1: Authenticate with the Service</h3>
            <p>Click the button below to open the OData service in a new tab. Log in with your credentials when prompted.</p>
            <p>Service URL: <span class="code">%s</span></p>
            <a href="%s" target="_blank" class="button" onclick="enableCaptureButton()">
                Open Service Login Page →
            </a>
        </div>
        
        <div class="step">
            <h3>Step 2: Capture Authentication Cookies</h3>
            <p>After successfully logging in, click the button below to capture the authentication cookies.</p>
            <button id="captureBtn" class="button" onclick="captureCookies()" disabled>
                Capture Cookies
            </button>
            <div id="cookieList" style="display: none;"></div>
        </div>
        
        <div id="status"></div>
    </div>

    <script>
        const targetDomain = '%s';
        const captureUrl = '%s';
        const importantCookies = ['MYSAPSSO2', 'SAP_SESSIONID', 'sap-usercontext'];
        
        function enableCaptureButton() {
            // Enable the capture button after a short delay
            setTimeout(() => {
                document.getElementById('captureBtn').disabled = false;
            }, 2000);
        }
        
        async function captureCookies() {
            const status = document.getElementById('status');
            const captureBtn = document.getElementById('captureBtn');
            const cookieList = document.getElementById('cookieList');
            
            captureBtn.disabled = true;
            status.style.display = 'block';
            status.className = '';
            status.textContent = 'Attempting to capture cookies...';
            
            try {
                // Get all cookies
                const allCookies = await getCookies();
                
                // Filter cookies for the target domain
                const relevantCookies = allCookies.filter(cookie => 
                    cookie.domain.includes(targetDomain.split(':')[0])
                );
                
                if (relevantCookies.length === 0) {
                    throw new Error('No cookies found for the service domain. Please ensure you have logged in successfully.');
                }
                
                // Display cookies
                cookieList.style.display = 'block';
                cookieList.innerHTML = '<strong>Captured cookies:</strong><br>';
                relevantCookies.forEach(cookie => {
                    const isImportant = importantCookies.includes(cookie.name);
                    const className = isImportant ? 'cookie-item important-cookie' : 'cookie-item';
                    cookieList.innerHTML += '<div class="' + className + '">' + 
                        cookie.name + ' = ' + cookie.value.substring(0, 20) + '...' +
                        ' (domain: ' + cookie.domain + ')</div>';
                });
                
                // Send cookies to the server
                const response = await fetch(captureUrl, {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json',
                    },
                    body: JSON.stringify({ cookies: relevantCookies })
                });
                
                if (!response.ok) {
                    throw new Error('Failed to send cookies to the server');
                }
                
                status.className = 'success';
                status.textContent = '✓ Cookies captured successfully! You can close this window.';
                
                // Auto-close after success
                setTimeout(() => {
                    window.close();
                }, 3000);
                
            } catch (error) {
                status.className = 'error';
                status.textContent = '✗ Error: ' + error.message;
                captureBtn.disabled = false;
            }
        }
        
        async function getCookies() {
            // Try Chrome extension API first (if available)
            if (typeof chrome !== 'undefined' && chrome.cookies) {
                return new Promise((resolve) => {
                    chrome.cookies.getAll({}, (cookies) => {
                        resolve(cookies);
                    });
                });
            }
            
            // Fallback: instruct user to use developer tools
            const status = document.getElementById('status');
            status.className = 'error';
            status.innerHTML = '<strong>Manual cookie extraction required:</strong><br>' +
                '1. Open Developer Tools (F12)<br>' +
                '2. Go to Application/Storage &rarr; Cookies<br>' +
                '3. Find cookies for domain: <span class="code">' + targetDomain + '</span><br>' +
                '4. Look for: MYSAPSSO2, SAP_SESSIONID, sap-usercontext<br>' +
                '5. Use the --cookie-string option with the values<br>' +
                '<br>' +
                'Example:<br>' +
                '<span class="code">--cookie-string "MYSAPSSO2=value1; SAP_SESSIONID=value2"</span>';
            throw new Error('Automatic cookie extraction not available');
        }
    </script>
</body>
</html>`