package auth

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
)

// ChromeManualAuth uses Chrome but waits for manual confirmation
type ChromeManualAuth struct {
	serviceURL string
	verbose    bool
}

// NewChromeManualAuth creates a new manual Chrome authenticator
func NewChromeManualAuth(serviceURL string, verbose bool) *ChromeManualAuth {
	return &ChromeManualAuth{
		serviceURL: serviceURL,
		verbose:    verbose,
	}
}

// Authenticate opens Chrome and waits for user to complete auth
func (c *ChromeManualAuth) Authenticate(ctx context.Context) (map[string]string, error) {
	if c.verbose {
		fmt.Fprintf(os.Stderr, "[VERBOSE] Starting Chrome manual authentication\n")
	}

	// Chrome options - visible window
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.NoFirstRun,
		chromedp.NoDefaultBrowserCheck,
		chromedp.Flag("headless", false),
		chromedp.WindowSize(1024, 768),
	)

	allocCtx, cancel := chromedp.NewExecAllocator(ctx, opts...)
	defer cancel()

	chromeCtx, cancel := chromedp.NewContext(allocCtx)
	defer cancel()

	// Enable network events
	if err := chromedp.Run(chromeCtx, network.Enable()); err != nil {
		return nil, fmt.Errorf("failed to enable network events: %w", err)
	}

	// Navigate to service URL
	fmt.Println("\n=== Chrome Manual Authentication ===")
	fmt.Println("1. A Chrome window will open")
	fmt.Println("2. Complete your SAML login")
	fmt.Println("3. Once you see the SAP service page, press Enter here")
	fmt.Println("=====================================\n")

	if err := chromedp.Run(chromeCtx, chromedp.Navigate(c.serviceURL)); err != nil {
		return nil, fmt.Errorf("failed to navigate: %w", err)
	}

	// Wait for user to press Enter
	fmt.Print("Press Enter when authentication is complete...")
	fmt.Scanln()

	// Extract cookies
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
		return nil, fmt.Errorf("failed to extract cookies: %w", err)
	}

	if c.verbose {
		fmt.Fprintf(os.Stderr, "[VERBOSE] Current URL: %s\n", currentURL)
		fmt.Fprintf(os.Stderr, "[VERBOSE] Found %d cookies\n", len(cookies))
	}

	// Convert cookies to map
	cookieMap := make(map[string]string)
	serviceHost := strings.Split(strings.Replace(c.serviceURL, "http://", "", 1), "/")[0]
	serviceHost = strings.Split(strings.Replace(serviceHost, "https://", "", 1), "/")[0]
	
	for _, cookie := range cookies {
		// Include cookies for our domain
		if strings.Contains(serviceHost, cookie.Domain) || 
		   strings.Contains(cookie.Domain, serviceHost) ||
		   cookie.Domain == "" {
			cookieMap[cookie.Name] = cookie.Value
			if c.verbose {
				fmt.Fprintf(os.Stderr, "[VERBOSE] Captured cookie: %s (domain: %s)\n", 
					cookie.Name, cookie.Domain)
			}
		}
	}

	// Give user option to keep Chrome open
	fmt.Print("\nClose Chrome window? (y/n): ")
	var response string
	fmt.Scanln(&response)
	
	if strings.ToLower(response) == "y" {
		// Close Chrome
		cancel()
		time.Sleep(1 * time.Second)
	} else {
		fmt.Println("Chrome window will remain open. Close it manually when done.")
	}

	if len(cookieMap) == 0 {
		return nil, fmt.Errorf("no cookies captured")
	}

	return cookieMap, nil
}