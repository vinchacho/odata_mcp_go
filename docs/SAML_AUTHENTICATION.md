# SAML Authentication Guide

This guide explains how to authenticate with SAP systems that use SAML-based authentication through Azure AD or other identity providers.

## Understanding the Authentication Flow

When you access a SAML-protected SAP OData service in a browser:

1. **Initial Request**: Browser requests the OData service
2. **SAML Redirect**: SAP redirects to its SAML Identity Provider (IdP)
3. **IdP Authentication**: The IdP (e.g., Azure AD) authenticates the user
4. **SAML Assertion**: IdP sends a SAML assertion back to SAP
5. **Session Creation**: SAP validates the assertion and creates a session
6. **Cookie Setting**: SAP sets authentication cookies (MYSAPSSO2, etc.)

## Why Direct OAuth2 Doesn't Work

The error `AADSTS500011: The resource principal named https://sapsbx00... was not found` indicates that:
- The SAP system is not registered as an OAuth2 resource in Azure AD
- It only accepts SAML assertions, not OAuth2 tokens
- Authentication must go through the SAML flow

## Authentication Methods

### Method 1: Browser-Assisted Cookie Extraction (Recommended)

Use the `--auth-saml-browser` flag to get step-by-step instructions:

```bash
odata-mcp --auth-saml-browser <service-url>
```

This will:
1. Open a browser window with instructions
2. Open the OData service in another tab
3. Guide you through the authentication process
4. Show you how to extract the cookies

### Method 2: Manual Cookie Extraction

1. **Open the OData service in your browser**:
   ```
   http://sapsbx00.redmond.corp.microsoft.com:8031/sap/opu/odata/sap/SRA020_PO_TRACKING_SRV/
   ```

2. **Authenticate when prompted**:
   - You'll be redirected to your organization's login page
   - Enter your credentials
   - Complete any MFA requirements

3. **Extract cookies using Developer Tools**:
   - Press F12 to open Developer Tools
   - Go to Application → Storage → Cookies
   - Find cookies for the SAP domain
   - Look for these important cookies:
     - `MYSAPSSO2` - Primary SAP authentication token
     - `SAP_SESSIONID_*` - Session identifier
     - `sap-usercontext` - User context token

4. **Use the cookies with odata-mcp**:

   **Option A - Cookie String**:
   ```bash
   odata-mcp --cookie-string "MYSAPSSO2=AjQx...xyz; SAP_SESSIONID_SX0_300=ABC...123" \
     <service-url>
   ```

   **Option B - Cookie File**:
   Create a file `cookies.txt`:
   ```
   MYSAPSSO2=AjQx...xyz
   SAP_SESSIONID_SX0_300=ABC...123
   sap-usercontext=sap-client=300
   ```
   
   Then use:
   ```bash
   odata-mcp --cookie-file cookies.txt <service-url>
   # Or use the shorter alias:
   odata-mcp --cookies cookies.txt <service-url>
   ```

### Method 3: Browser Extension (Future)

We're working on a browser extension that can automatically extract cookies after SAML authentication.

## Cookie File Format

The cookie file supports two formats:

### Netscape Format (Recommended)
```
# Netscape HTTP Cookie File
sap-server.example.com	TRUE	/	FALSE	1234567890	MYSAPSSO2	AjQx...
sap-server.example.com	TRUE	/	FALSE	1234567890	SAP_SESSIONID_SX0_300	ABC...
```

### Simple Format
```
MYSAPSSO2=AjQx...xyz
SAP_SESSIONID_SX0_300=ABC...123
sap-usercontext=sap-client=300
```

## Troubleshooting

### "Resource principal not found" Error
This error means the SAP system doesn't support direct OAuth2. Use the SAML browser method instead.

### Cookies Expire Quickly
SAP sessions typically expire after inactivity. You may need to re-authenticate periodically.

### No MYSAPSSO2 Cookie
Ensure you're fully authenticated. Some SAP systems require accessing specific paths to generate the SSO token.

### MCP Integration
For MCP server usage, you'll need to:
1. Authenticate manually first
2. Extract cookies
3. Configure the MCP server with the cookie file:
   ```json
   {
     "command": "odata-mcp",
     "args": ["--cookies", "/path/to/cookies.txt", "<service-url>"]
   }
   ```
   
   Or for Windows:
   ```json
   {
     "command": "C:/bin/odata-mcp.exe",
     "args": ["--cookies", "C:/path/to/cookies.txt", "<service-url>"]
   }
   ```

## Security Considerations

- **Cookie Security**: SAP authentication cookies are sensitive. Store them securely.
- **Expiration**: Cookies expire. Plan for re-authentication.
- **Rotation**: In production, implement automated cookie refresh if possible.

## Future Improvements

We're working on:
1. Automated SAML flow handling
2. Browser extension for automatic cookie extraction
3. Cookie refresh mechanisms
4. Windows integrated authentication support