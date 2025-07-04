# Advanced SAML Authentication Methods

OData MCP now supports advanced SAML authentication methods that can automatically handle the complex redirect flows:

## WebView2 Authentication (Windows Only)

Uses Microsoft Edge WebView2 to handle SAML authentication automatically.

### Prerequisites
- Windows 10/11
- [Microsoft Edge WebView2 Runtime](https://go.microsoft.com/fwlink/p/?LinkId=2124703)

### Usage
```bash
# Interactive authentication with visible browser window
odata-mcp --auth-webview2 --service <service-url>

# Test authentication only
odata-mcp --auth-webview2 --test-auth --service <service-url>

# Save cookies after successful auth
odata-mcp --auth-webview2 --test-auth --cookies cookies.txt --service <service-url>
```

### How it Works
1. Opens an embedded Edge browser window
2. Navigates to your SAP service URL
3. Handles all SAML redirects automatically
4. Extracts cookies after successful authentication
5. Closes the window and continues with MCP

### Advantages
- Fully automated - no manual steps
- Handles complex SAML flows
- Native Windows integration
- Supports all identity providers

## Chrome Automation

Uses Chrome DevTools Protocol to automate authentication.

### Prerequisites
- Google Chrome or Chromium installed
- Chrome must be in PATH or standard location

### Usage

#### Visible Browser Mode
```bash
# Opens Chrome window for authentication
odata-mcp --auth-chrome --service <service-url>

# Test mode with cookie saving
odata-mcp --auth-chrome --test-auth --cookies cookies.txt --service <service-url>
```

#### Headless Mode
```bash
# Runs Chrome in headless mode (no visible window)
odata-mcp --auth-chrome-headless --service <service-url>

# Note: Some SAML providers may detect and block headless browsers
```

### How it Works
1. Launches Chrome with automation enabled
2. Navigates to SAP service
3. Monitors for SAML redirects
4. Waits for MYSAPSSO2 cookie
5. Extracts all cookies
6. Closes Chrome

### Advantages
- Cross-platform (Windows, macOS, Linux)
- Can run headless for automation
- Full browser capabilities
- Handles JavaScript-heavy SAML flows

## Comparison of Methods

| Method | Platform | Automation | Visibility | Reliability |
|--------|----------|------------|------------|-------------|
| `--auth-saml-browser` | All | Manual | Browser opens | High |
| `--auth-windows` | Windows | Automatic | No UI | Low (timeouts) |
| `--auth-webview2` | Windows | Automatic | Embedded window | High |
| `--auth-chrome` | All | Automatic | Chrome window | High |
| `--auth-chrome-headless` | All | Automatic | No UI | Medium |

## Troubleshooting

### WebView2 Issues
- **"WebView2 runtime not found"**: Install the runtime from the link provided
- **Window doesn't appear**: Check if antivirus is blocking WebView2
- **Authentication loops**: Clear Edge cookies and try again

### Chrome Issues
- **"Chrome not found"**: Ensure Chrome is installed and in PATH
- **Timeouts**: Increase timeout or use visible mode to debug
- **Headless blocked**: Some SAML providers block headless browsers

## Security Considerations

1. **Cookies are sensitive**: The extracted MYSAPSSO2 token grants access to your SAP system
2. **Store securely**: Use appropriate file permissions for saved cookie files
3. **Limited lifetime**: SAP cookies typically expire after 8-12 hours
4. **Automated tools**: Be cautious when using automation in production environments

## Examples

### Complete Workflow with WebView2
```bash
# 1. Test authentication and save cookies
odata-mcp --auth-webview2 --test-auth --cookies ~/.odata-mcp/sap-cookies.txt \
  --service http://sapserver:8000/sap/opu/odata/sap/SERVICE/

# 2. Use saved cookies for MCP
odata-mcp --cookies ~/.odata-mcp/sap-cookies.txt \
  --service http://sapserver:8000/sap/opu/odata/sap/SERVICE/
```

### Automation Script with Chrome
```bash
#!/bin/bash
# Automated SAP data extraction

COOKIE_FILE="/tmp/sap-cookies-$$.txt"
SERVICE_URL="http://sapserver:8000/sap/opu/odata/sap/SERVICE/"

# Authenticate and save cookies
odata-mcp --auth-chrome-headless --test-auth --cookies "$COOKIE_FILE" \
  --service "$SERVICE_URL"

# Use cookies with MCP
odata-mcp --cookies "$COOKIE_FILE" --service "$SERVICE_URL"

# Clean up
rm -f "$COOKIE_FILE"
```

## Future Enhancements

- Automatic cookie refresh before expiration
- Support for custom identity provider configurations
- Integration with OS credential managers
- Support for other browsers (Firefox, Safari)