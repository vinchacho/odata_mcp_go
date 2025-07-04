@echo off
REM Test script for Windows authentication methods

set SERVICE_URL=http://sap-server.example.com:8000/sap/opu/odata/sap/SERVICE_NAME/
set COOKIE_FILE=test_cookies.txt

echo Windows Authentication Test
echo ==========================
echo Service URL: %SERVICE_URL%
echo.

echo Available methods:
echo 1. Chrome (visible browser)
echo 2. Chrome Headless  
echo 3. WebView2 (Edge)
echo 4. PowerShell (Windows integrated)
echo.

set /p METHOD="Select method (1-4): "

if "%METHOD%"=="1" (
    echo.
    echo Testing Chrome authentication...
    odata-mcp.exe --auth-chrome --test-auth --cookies "%COOKIE_FILE%" --service "%SERVICE_URL%" --verbose
) else if "%METHOD%"=="2" (
    echo.
    echo Testing Chrome headless authentication...
    odata-mcp.exe --auth-chrome-headless --test-auth --cookies "%COOKIE_FILE%" --service "%SERVICE_URL%" --verbose
) else if "%METHOD%"=="3" (
    echo.
    echo Testing WebView2 authentication...
    odata-mcp.exe --auth-webview2 --test-auth --cookies "%COOKIE_FILE%" --service "%SERVICE_URL%" --verbose
) else if "%METHOD%"=="4" (
    echo.
    echo Testing PowerShell Windows integrated authentication...
    odata-mcp.exe --auth-windows --test-auth --cookies "%COOKIE_FILE%" --service "%SERVICE_URL%" --verbose
) else (
    echo Invalid selection
    exit /b 1
)

REM Check if cookies were saved
if exist "%COOKIE_FILE%" (
    echo.
    echo Cookies saved to %COOKIE_FILE%:
    echo ==============================
    type "%COOKIE_FILE%"
    echo.
    echo You can now use:
    echo odata-mcp.exe --cookies "%COOKIE_FILE%" --service "%SERVICE_URL%"
) else (
    echo.
    echo No cookies were saved.
)

pause