#!/bin/bash

# Test browser authentication with public service first
echo "Testing browser auth implementation with Northwind service..."
echo "============================================================"

# This should work without authentication
./odata-mcp --verbose --trace https://services.odata.org/V2/Northwind/Northwind.svc/

echo -e "\n\nNow testing AAD browser flow (will fail but shows flow)..."
echo "==========================================================="

# This will fail but shows the browser auth flow is working
./odata-mcp --verbose \
    --auth-aad \
    --aad-tenant common \
    --aad-browser \
    --aad-trace \
    --trace \
    https://services.odata.org/V2/Northwind/Northwind.svc/ 2>&1 | head -50