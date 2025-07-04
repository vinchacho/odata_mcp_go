# Azure AD Browser Authentication

This document describes the browser-based authentication flow for Azure AD in the OData MCP bridge.

## Overview

The browser authentication flow is designed to work seamlessly with MCP servers by:
1. Opening a browser window for user authentication
2. Capturing the authorization code via local callback
3. Exchanging the code for an access token
4. Using the token to access the OData service

## Usage

```bash
# Basic browser authentication
odata-mcp --auth-aad --aad-browser <service-url>

# With specific tenant
odata-mcp --auth-aad --aad-browser --aad-tenant mycompany.com <service-url>

# With custom client ID
odata-mcp --auth-aad --aad-browser --aad-client-id <client-id> <service-url>

# With authentication tracing for debugging
odata-mcp --auth-aad --aad-browser --aad-trace <service-url>
```

## Authentication Flow

1. **Initialization**: The bridge starts a local HTTP server on a random port
2. **Browser Launch**: Opens the Azure AD login page in the default browser
3. **User Login**: User authenticates with their Azure AD credentials
4. **Authorization Code**: Azure AD redirects back to the local server with an authorization code
5. **Token Exchange**: The code is exchanged for an access token
6. **Service Access**: The token is used to authenticate with the OData service

## Debugging

Enable authentication tracing to debug issues:

```bash
odata-mcp --auth-aad --aad-browser --aad-trace --verbose <service-url>
```

This will create a detailed trace file in `~/.odata-mcp/traces/` containing:
- All HTTP requests and responses
- Authentication flow steps
- Token information (redacted)
- Error details

## Security Features

- Uses PKCE (Proof Key for Code Exchange) for enhanced security
- Tokens are cached in memory during the session
- Sensitive information is redacted in logs
- Local callback server only accepts connections from localhost

## Common Issues

### AADSTS500011 Error
This error means the resource principal (OData service) was not found in the tenant. This typically happens when:
- The service requires a specific app registration
- The service URL is not registered as a valid resource
- The tenant doesn't have access to the service

### Browser Doesn't Open
If the browser doesn't open automatically:
1. Check the console output for the authentication URL
2. Copy and paste the URL into your browser manually
3. Complete the authentication flow

### Token Exchange Fails
If the token exchange fails:
1. Enable tracing with `--aad-trace`
2. Check the trace file for detailed error messages
3. Verify the client ID and tenant are correct
4. Ensure the redirect URI matches what's registered in Azure AD

## Integration with MCP

The browser authentication flow is designed to work with MCP servers:
- Authentication happens during server startup
- Tokens are cached for the session duration
- Automatic token refresh is handled transparently
- No user interaction required after initial authentication