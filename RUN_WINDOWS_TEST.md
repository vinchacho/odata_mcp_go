# Windows Authentication Test Instructions

## Where to Run

You can run this test from either **PowerShell** or **Command Prompt (CMD)**. Here's how:

### Option 1: From PowerShell (Recommended)
1. Open PowerShell as a regular user (no admin needed)
   - Press `Win + X` and select "Windows PowerShell"
   - Or press `Win + R`, type `powershell`, and press Enter

2. Navigate to your directory:
   ```powershell
   cd C:\Users\avinogradova\dev\odata_mcp_go
   ```

3. Run the test:
   ```powershell
   .\windows_auth_test.ps1
   ```

### Option 2: From Command Prompt (CMD)
1. Open Command Prompt
   - Press `Win + R`, type `cmd`, and press Enter

2. Navigate to your directory:
   ```cmd
   cd C:\Users\avinogradova\dev\odata_mcp_go
   ```

3. Run PowerShell script from CMD:
   ```cmd
   powershell -ExecutionPolicy Bypass -File windows_auth_test.ps1
   ```

## Can They Be Interchanged?

**Yes**, PowerShell scripts (`.ps1` files) can be run from both:
- **PowerShell**: Direct execution with `.\script.ps1`
- **CMD**: Using `powershell -File script.ps1`

The results should be identical since both use the same PowerShell engine.

## What the Test Does

1. **Basic Connectivity**: Checks if the SAP server is reachable
2. **Initial Request**: Makes a request without following redirects to see the SAML flow
3. **Full Authentication**: Attempts complete authentication with redirect following
4. **Cookie Analysis**: Shows all cookies collected during authentication
5. **SAP Cookie Check**: Specifically looks for MYSAPSSO2 and other SAP cookies
6. **Alternative Method**: Tries HttpWebRequest as an alternative approach

## Expected Output

The test will show:
- Whether connection to SAP server works
- What redirects occur (SAML flow)
- Which cookies are set
- Whether MYSAPSSO2 token is obtained
- Time taken for each operation

## Quick Copy-Paste Commands

For PowerShell:
```powershell
cd C:\Users\avinogradova\dev\odata_mcp_go
.\windows_auth_test.ps1
```

For CMD:
```cmd
cd C:\Users\avinogradova\dev\odata_mcp_go
powershell -ExecutionPolicy Bypass -File windows_auth_test.ps1
```

## Troubleshooting

If you get an execution policy error in PowerShell:
```powershell
Set-ExecutionPolicy -ExecutionPolicy Bypass -Scope Process
.\windows_auth_test.ps1
```