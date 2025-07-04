# Windows Integrated Authentication

This guide explains how to use Windows integrated authentication with the OData MCP bridge.

## Overview

The `--auth-windows` flag enables automatic authentication using your Windows credentials. This is particularly useful for:
- Corporate SAP systems that use SAML with Windows SSO
- Internal services that support Windows integrated authentication
- Domain-joined Windows machines

## How It Works

1. **PowerShell Integration**: Uses PowerShell's `Invoke-WebRequest` with `-UseDefaultCredentials`
2. **Automatic SAML Flow**: Handles all redirects and SAML assertions automatically
3. **Cookie Extraction**: Captures authentication cookies after successful login
4. **Seamless Experience**: No manual steps required

## Usage

### Command Line

```bash
# Basic usage
odata-mcp --auth-windows --service https://sap.company.com/odata/

# With verbose output to see the authentication process
odata-mcp --auth-windows --verbose --service https://sap.company.com/odata/
```

### MCP Configuration

For Claude Desktop or other MCP clients:

```json
{
    "mcpServers": {
        "sap-production": {
            "command": "C:/bin/odata-mcp.exe",
            "args": [
                "--auth-windows",
                "--service",
                "https://sap.company.com/odata/"
            ]
        }
    }
}
```

## Requirements

- **Windows OS**: This feature only works on Windows
- **PowerShell**: PowerShell 5.1 or later (included in Windows 10/11)
- **Domain Joined**: Works best on domain-joined machines
- **Network Access**: Must be able to reach the authentication servers

## Authentication Flow

1. PowerShell creates a web session with Windows credentials
2. Requests the OData service URL
3. Follows all redirects (SAML, ADFS, etc.) automatically
4. Windows credentials are used via NTLM/Kerberos
5. Cookies are captured from the session
6. Cookies are passed to the OData client

## Troubleshooting

### "PowerShell authentication failed"

1. **Check PowerShell execution policy**:
   ```powershell
   Get-ExecutionPolicy
   # Should not be "Restricted"
   ```

2. **Verify you can access the service in browser**:
   - Open the OData URL in Edge or Chrome
   - Ensure it authenticates automatically

3. **Run with verbose mode**:
   ```bash
   odata-mcp --auth-windows --verbose --service <url>
   ```

### "No cookies captured"

This might happen if:
- The service doesn't set cookies (uses different auth mechanism)
- Authentication failed silently
- Cookies are set on a different domain

Try the manual PowerShell script approach:
```powershell
.\scripts\sap_auth.ps1 -ServiceUrl "https://sap.company.com/odata/"
```

### Non-Domain Machines

If your machine isn't domain-joined:
1. Use `--auth-saml-browser` instead
2. Or manually extract cookies and use `--cookies`

## Security Notes

- Credentials are never stored or logged
- Uses Windows secure credential handling
- Cookies are only kept in memory during the session
- No passwords are ever written to disk

## Comparison with Other Methods

| Method | Automatic | Windows Only | Domain Required | Manual Steps |
|--------|-----------|--------------|-----------------|--------------|
| `--auth-windows` | ✓ | ✓ | Recommended | None |
| `--auth-aad` | ✓ | ✗ | ✗ | Device code |
| `--auth-saml-browser` | ✗ | ✗ | ✗ | Cookie extraction |
| `--cookies` | ✗ | ✗ | ✗ | Get cookies first |

## Examples

### Corporate SAP Example
```bash
odata-mcp --auth-windows --service http://sap-server.example.com:8000/sap/opu/odata/sap/SERVICE_NAME/
```

### Corporate SAP with Custom Domain
```bash
odata-mcp --auth-windows --verbose --service https://sap.contoso.com/sap/opu/odata/sap/MY_SERVICE/
```

## PowerShell Script Details

The authentication uses a PowerShell script that:
- Creates a web session
- Uses `-UseDefaultCredentials` for Windows auth
- Handles up to 20 redirects
- Extracts cookies from the session
- Returns them as JSON

You can see the generated script by running:
```bash
odata-mcp --auth-windows --verbose --service <url>
```

The verbose output will show the authentication process and any issues.