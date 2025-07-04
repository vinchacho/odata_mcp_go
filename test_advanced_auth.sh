#!/bin/bash

# Test script for advanced SAML authentication methods

SERVICE_URL="${1:-http://sap-server.example.com:8000/sap/opu/odata/sap/SERVICE_NAME/}"
COOKIE_FILE="test_cookies.txt"

echo "Advanced SAML Authentication Test"
echo "================================="
echo "Service URL: $SERVICE_URL"
echo ""

# Check platform
if [[ "$OSTYPE" == "msys" || "$OSTYPE" == "win32" ]]; then
    echo "Platform: Windows"
    WINDOWS=true
else
    echo "Platform: Linux/Mac"
    WINDOWS=false
fi

echo ""
echo "Available authentication methods:"
echo "1. Chrome (visible browser)"
echo "2. Chrome Headless"
if [ "$WINDOWS" = true ]; then
    echo "3. WebView2 (Edge)"
fi
echo ""

read -p "Select method (1-3): " METHOD

case $METHOD in
    1)
        echo "Testing Chrome authentication..."
        ./odata-mcp --auth-chrome --test-auth --cookies "$COOKIE_FILE" --service "$SERVICE_URL" --verbose
        ;;
    2)
        echo "Testing Chrome headless authentication..."
        ./odata-mcp --auth-chrome-headless --test-auth --cookies "$COOKIE_FILE" --service "$SERVICE_URL" --verbose
        ;;
    3)
        if [ "$WINDOWS" = true ]; then
            echo "Testing WebView2 authentication..."
            ./odata-mcp --auth-webview2 --test-auth --cookies "$COOKIE_FILE" --service "$SERVICE_URL" --verbose
        else
            echo "WebView2 is only available on Windows"
            exit 1
        fi
        ;;
    *)
        echo "Invalid selection"
        exit 1
        ;;
esac

# Check if cookies were saved
if [ -f "$COOKIE_FILE" ]; then
    echo ""
    echo "Cookies saved to $COOKIE_FILE:"
    echo "=============================="
    cat "$COOKIE_FILE"
    echo ""
    echo "You can now use:"
    echo "./odata-mcp --cookies \"$COOKIE_FILE\" --service \"$SERVICE_URL\""
else
    echo ""
    echo "No cookies were saved."
fi