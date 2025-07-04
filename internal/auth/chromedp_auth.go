package auth

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
)

// ChromeDPAuth handles SAML authentication using headless Chrome
type ChromeDPAuth struct {
	serviceURL string
	verbose    bool
	headless   bool
}

// NewChromeDPAuth creates a new ChromeDP authenticator
func NewChromeDPAuth(serviceURL string, verbose, headless bool) *ChromeDPAuth {
	return &ChromeDPAuth{
		serviceURL: serviceURL,
		verbose:    verbose,
		headless:   headless,
	}
}

// Authenticate performs SAML authentication using ChromeDP
func (c *ChromeDPAuth) Authenticate(ctx context.Context) (map[string]string, error) {
	if c.verbose {
		fmt.Fprintf(os.Stderr, "[VERBOSE] Starting ChromeDP SAML authentication (headless: %v)\n", c.headless)
	}

	// Chrome options
	opts := []chromedp.ExecAllocatorOption{
		chromedp.NoFirstRun,
		chromedp.NoDefaultBrowserCheck,
		chromedp.UserAgent("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"),
	}

	if c.headless {
		opts = append(opts, chromedp.Headless)
	} else {
		// Show browser window
		opts = append(opts, 
			chromedp.Flag("headless", false),
			chromedp.Flag("hide-scrollbars", false),
			chromedp.Flag("mute-audio", false),
			chromedp.WindowSize(1024, 768),
		)
	}

	// Create Chrome instance
	allocCtx, cancel := chromedp.NewExecAllocator(ctx, opts...)
	defer cancel()

	// Create Chrome context
	chromeCtx, cancel := chromedp.NewContext(allocCtx, chromedp.WithLogf(func(format string, args ...interface{}) {
		if c.verbose {
			fmt.Fprintf(os.Stderr, "[CHROME] "+format+"\n", args...)
		}
	}))
	defer cancel()

	// Enable network events to capture cookies
	if err := chromedp.Run(chromeCtx, network.Enable()); err != nil {
		return nil, fmt.Errorf("failed to enable network events: %w", err)
	}

	// Channel to receive cookies
	cookieChan := make(chan map[string]string, 1)
	errorChan := make(chan error, 1)

	// Listen for cookies
	chromedp.ListenTarget(chromeCtx, func(ev interface{}) {
		switch ev := ev.(type) {
		case *network.EventResponseReceived:
			// Check if we got a response from the SAP server
			if strings.Contains(ev.Response.URL, c.serviceURL) {
				if c.verbose {
					fmt.Fprintf(os.Stderr, "[VERBOSE] Response from SAP: %s (status: %d)\n", 
						ev.Response.URL, ev.Response.Status)
				}
			}
		}
	})

	// Navigate and wait for authentication
	go func() {
		if c.verbose {
			fmt.Fprintf(os.Stderr, "[VERBOSE] Navigating to: %s\n", c.serviceURL)
		}
		
		err := chromedp.Run(chromeCtx,
			// Navigate to service URL
			chromedp.Navigate(c.serviceURL),
		)
		if err != nil {
			if c.verbose {
				fmt.Fprintf(os.Stderr, "[VERBOSE] Navigation error: %v\n", err)
			}
			errorChan <- err
			return
		}
		
		// Poll for cookies in a loop
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()
		timeout := time.After(5 * time.Minute)
		
		for {
			select {
			case <-ticker.C:
				var cookies []*network.Cookie
				var currentURL string
				
				err := chromedp.Run(chromeCtx,
					chromedp.Location(&currentURL),
					chromedp.ActionFunc(func(ctx context.Context) error {
						var err error
						cookies, err = network.GetCookies().Do(ctx)
						return err
					}),
				)
				
				if err != nil {
					if c.verbose {
						fmt.Fprintf(os.Stderr, "[VERBOSE] Error getting cookies: %v\n", err)
					}
					continue
				}
				
				// Check if we have MYSAPSSO2 cookie
				foundSAP := false
				for _, cookie := range cookies {
					if cookie.Name == "MYSAPSSO2" {
						foundSAP = true
						if c.verbose {
							fmt.Fprintf(os.Stderr, "[VERBOSE] Found MYSAPSSO2 cookie\n")
						}
						break
					}
				}
				
				// Log current state
				if c.verbose && len(cookies) > 0 {
					fmt.Fprintf(os.Stderr, "[VERBOSE] Current URL: %s\n", currentURL)
					fmt.Fprintf(os.Stderr, "[VERBOSE] Cookies found: %d\n", len(cookies))
					for _, cookie := range cookies {
						fmt.Fprintf(os.Stderr, "[VERBOSE]   - %s (domain: %s)\n", cookie.Name, cookie.Domain)
					}
				}
				
				// Check if authentication is complete
				if foundSAP || (strings.Contains(currentURL, strings.Split(c.serviceURL, "?")[0]) && len(cookies) > 5) {
					if c.verbose {
						fmt.Fprintf(os.Stderr, "[VERBOSE] Authentication appears complete with %d cookies\n", len(cookies))
					}
					
					// Wait a bit more to ensure all cookies are set
					time.Sleep(2 * time.Second)
					
					// Get final cookies
					chromedp.Run(chromeCtx,
						chromedp.ActionFunc(func(ctx context.Context) error {
							cookies, err = network.GetCookies().Do(ctx)
							return err
						}),
					)
					
					// Convert cookies to map
					cookieMap := make(map[string]string)
					serviceHost := strings.Split(strings.Replace(c.serviceURL, "http://", "", 1), "/")[0]
					
					for _, cookie := range cookies {
						// Include cookies for our domain
						if strings.Contains(serviceHost, cookie.Domain) || strings.Contains(cookie.Domain, serviceHost) {
							cookieMap[cookie.Name] = cookie.Value
							if c.verbose {
								fmt.Fprintf(os.Stderr, "[VERBOSE] Captured cookie: %s (domain: %s)\n", 
									cookie.Name, cookie.Domain)
							}
						}
					}
					
					cookieChan <- cookieMap
					return
				}
				
			case <-timeout:
				errorChan <- fmt.Errorf("authentication timeout")
				return
			}
		}
	}()

	// Show instructions if not headless
	if !c.headless {
		fmt.Println("\n=== Chrome Browser Authentication ===")
		fmt.Println("A Chrome window has opened for SAML authentication.")
		fmt.Println("Please log in with your credentials.")
		fmt.Println("The window will close automatically after successful authentication.")
		fmt.Println("=====================================\n")
	}

	// Monitor Chrome context
	go func() {
		<-chromeCtx.Done()
		if c.verbose {
			fmt.Fprintf(os.Stderr, "[VERBOSE] Chrome context closed: %v\n", chromeCtx.Err())
		}
		errorChan <- fmt.Errorf("Chrome window was closed")
	}()

	// Wait for result
	select {
	case cookies := <-cookieChan:
		if len(cookies) == 0 {
			return nil, fmt.Errorf("no cookies captured")
		}
		
		// Check for important cookies
		if _, ok := cookies["MYSAPSSO2"]; !ok && c.verbose {
			fmt.Fprintf(os.Stderr, "[VERBOSE] Warning: MYSAPSSO2 cookie not found\n")
			fmt.Fprintf(os.Stderr, "[VERBOSE] Captured cookies:\n")
			for name := range cookies {
				fmt.Fprintf(os.Stderr, "[VERBOSE]   - %s\n", name)
			}
		}
		
		return cookies, nil
		
	case err := <-errorChan:
		return nil, fmt.Errorf("chrome automation failed: %w", err)
		
	case <-ctx.Done():
		return nil, ctx.Err()
		
	case <-time.After(5 * time.Minute):
		return nil, fmt.Errorf("authentication timeout")
	}
}

// ExtractCookiesFromBrowser extracts cookies from an existing browser session
func (c *ChromeDPAuth) ExtractCookiesFromBrowser(ctx context.Context) (map[string]string, error) {
	if c.verbose {
		fmt.Fprintf(os.Stderr, "[VERBOSE] Extracting cookies from existing browser session\n")
	}

	// Use debugging port to connect to existing Chrome
	opts := []chromedp.ExecAllocatorOption{
		chromedp.Flag("remote-debugging-port", "9222"),
		chromedp.Flag("disable-gpu", true),
	}

	allocCtx, cancel := chromedp.NewExecAllocator(ctx, opts...)
	defer cancel()

	chromeCtx, cancel := chromedp.NewContext(allocCtx)
	defer cancel()

	var cookies []*network.Cookie
	err := chromedp.Run(chromeCtx,
		// Navigate to the service URL
		chromedp.Navigate(c.serviceURL),
		
		// Get all cookies
		chromedp.ActionFunc(func(ctx context.Context) error {
			var err error
			cookies, err = network.GetCookies().Do(ctx)
			return err
		}),
	)

	if err != nil {
		return nil, fmt.Errorf("failed to extract cookies: %w", err)
	}

	// Convert to map
	cookieMap := make(map[string]string)
	serviceHost := strings.Split(strings.Replace(c.serviceURL, "http://", "", 1), "/")[0]
	
	for _, cookie := range cookies {
		if strings.Contains(serviceHost, cookie.Domain) || strings.Contains(cookie.Domain, serviceHost) {
			cookieMap[cookie.Name] = cookie.Value
			if c.verbose {
				fmt.Fprintf(os.Stderr, "[VERBOSE] Extracted cookie: %s\n", cookie.Name)
			}
		}
	}

	return cookieMap, nil
}

// ConvertCookiesToHTTP converts ChromeDP cookies to http.Cookie format
func ConvertCookiesToHTTP(cookies []*network.Cookie) []*http.Cookie {
	httpCookies := make([]*http.Cookie, 0, len(cookies))
	
	for _, c := range cookies {
		httpCookie := &http.Cookie{
			Name:     c.Name,
			Value:    c.Value,
			Path:     c.Path,
			Domain:   c.Domain,
			Secure:   c.Secure,
			HttpOnly: c.HTTPOnly,
		}
		
		if c.Expires > 0 {
			httpCookie.Expires = time.Unix(int64(c.Expires), 0)
		}
		
		httpCookies = append(httpCookies, httpCookie)
	}
	
	return httpCookies
}