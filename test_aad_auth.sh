#\!/bin/bash

# Test AAD authentication with Microsoft internal SAP system
echo "Testing AAD authentication flow..."

# First, let's see what happens when we try to access the service
./odata-mcp --verbose --auth-aad --aad-tenant microsoft.com \
    https://sap-server.example.com/sap/opu/odata/sap/SERVICE_NAME/ 2>&1  < /dev/null |  head -30
