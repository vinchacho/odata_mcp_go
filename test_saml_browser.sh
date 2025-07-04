#!/bin/bash

# Test SAML browser authentication
echo "Testing SAML browser authentication flow..."
echo "=========================================="

# This will open browser and show instructions for manual cookie extraction
./odata-mcp --verbose \
    --auth-saml-browser \
    http://sap-server.example.com:8000/sap/opu/odata/sap/SERVICE_NAME/