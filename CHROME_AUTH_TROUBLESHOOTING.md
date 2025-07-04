# Chrome Authentication Troubleshooting

If the automated Chrome authentication is not working, here are some troubleshooting steps:

## Common Issues

### 1. Chrome Closes Without Capturing Cookies
This can happen if:
- The SAML flow completes but cookies are set on a different domain
- Chrome security policies prevent cookie access
- The authentication redirects to a different URL than expected

**Solution**: Use `--verbose` flag to see what's happening:
```cmd
odata-mcp.exe --auth-chrome --verbose --test-auth --service <url>
```

### 2. Chrome Window Not Responding
Some SAML providers detect automated browsers and block them.

**Solution**: Try the manual browser approach:
```cmd
odata-mcp.exe --auth-saml-browser --service <url>
```

### 3. WebView2 Not Working
WebView2 requires Edge WebView2 Runtime to be installed.

**Solution**: Download and install from:
https://go.microsoft.com/fwlink/p/?LinkId=2124703

## Alternative: Semi-Automated Approach

If full automation isn't working, use this semi-automated approach:

### Step 1: Start Cookie Capture Server
```cmd
odata-mcp.exe --auth-saml-browser --service <url>
```
This will open a browser and start a local server waiting for cookies.

### Step 2: Complete Authentication
1. Log in to the SAP system in the browser
2. Once logged in, press F12 to open Developer Tools
3. Go to Console tab
4. Paste this code:

```javascript
// Extract and send cookies to OData MCP
(function() {
    const cookies = document.cookie.split(';').map(c => {
        const [name, value] = c.trim().split('=');
        return { name, value, domain: window.location.hostname };
    });
    
    // Copy to clipboard
    const cookieString = cookies.map(c => `${c.name}=${c.value}`).join('; ');
    navigator.clipboard.writeText(cookieString).then(() => {
        console.log('Cookies copied to clipboard!');
        console.log('Use with: --cookie-string "' + cookieString + '"');
    });
})();
```

### Step 3: Use Extracted Cookies
```cmd
odata-mcp.exe --cookie-string "<paste-cookies-here>" --service <url>
```

## Manual Cookie Extraction

If all else fails, manually extract cookies:

1. Open Chrome and navigate to your SAP service
2. Complete the SAML login
3. Press F12 → Application → Cookies
4. Find these cookies:
   - MYSAPSSO2
   - SAP_SESSIONID (if present)
   - sap-usercontext (if present)
5. Create a cookie string:
   ```
   MYSAPSSO2=<value>; SAP_SESSIONID=<value>; sap-usercontext=<value>
   ```
6. Use with:
   ```cmd
   odata-mcp.exe --cookie-string "<cookie-string>" --service <url>
   ```

## Debug Mode

To understand what's happening, run with maximum verbosity:
```cmd
set ODATA_VERBOSE=1
odata-mcp.exe --auth-chrome --verbose --test-auth --service <url> 2> auth_debug.log
```

Then check `auth_debug.log` for details about:
- Which URLs are being visited
- What cookies are being found
- Where the process is failing