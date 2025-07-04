# SAML Authentication Guide for OData MCP

Since Windows integrated authentication (`--auth-windows`) doesn't work with the SAP SAML flow (it times out and doesn't get the MYSAPSSO2 cookie), use the browser-based SAML authentication instead.

## Quick Start

```bash
odata-mcp --auth-saml-browser --service http://sap-server.example.com:8000/sap/opu/odata/sap/SERVICE_NAME/
```

## What Happens

1. **Browser Opens**: Two browser tabs will open:
   - Instructions page on localhost
   - SAP service URL for authentication

2. **Login**: Complete the SAML login in the SAP tab using your Microsoft credentials

3. **Confirm**: Click "I've Successfully Authenticated" in the instructions tab

4. **Extract Cookies**: Follow the terminal instructions to manually extract cookies using F12 Developer Tools

## After Getting Cookies

Once you have the MYSAPSSO2 cookie value, you can use it directly:

```bash
# Method 1: Cookie string
odata-mcp --cookie-string "MYSAPSSO2=<your-cookie-value>" --service <service-url>

# Method 2: Cookie file
echo "MYSAPSSO2=<your-cookie-value>" > cookies.txt
odata-mcp --cookie-file cookies.txt --service <service-url>
```

## Why Manual Extraction?

- SAML authentication involves complex redirects through identity providers
- Browser security prevents automatic cookie extraction from cross-origin SAML flows
- The MYSAPSSO2 cookie is httpOnly and secure, requiring manual extraction

## Cookie Persistence

The MYSAPSSO2 cookie typically lasts 8-12 hours. Save it to a file for reuse:

```bash
# Save to default location
mkdir -p ~/.odata-mcp
echo "MYSAPSSO2=<your-cookie-value>" > ~/.odata-mcp/cookies.txt

# Use saved cookies
odata-mcp --cookie-file ~/.odata-mcp/cookies.txt --service <service-url>
```

## Troubleshooting

If authentication fails:
1. Clear browser cookies for the SAP domain
2. Try in an incognito/private window
3. Ensure you're on the corporate network or VPN
4. Check that the service URL is accessible

## Alternative: Use Existing Browser Session

If you're already logged into SAP in your browser:
1. Navigate to the OData service URL
2. Extract cookies using F12 → Application → Cookies
3. Use the extracted MYSAPSSO2 value with `--cookie-string`