#!/bin/bash

# Test Windows integrated authentication
echo "Testing Windows integrated authentication..."
echo "========================================="

# This will use PowerShell to authenticate with Windows credentials
./odata-mcp --verbose \
    --auth-windows \
    --service http://sap-server.example.com:8000/sap/opu/odata/sap/SERVICE_NAME/