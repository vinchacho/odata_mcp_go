#!/bin/bash

# Test browser-based AAD authentication with tracing
echo "Testing browser-based AAD authentication flow with tracing..."
echo "=================================================="

# Test with Microsoft internal SAP system using browser flow and tracing
./odata-mcp --verbose \
    --auth-aad \
    --aad-tenant microsoft.com \
    --aad-browser \
    --aad-trace \
    --trace \
    http://sap-server.example.com:8000/sap/opu/odata/sap/SERVICE_NAME/