#!/bin/bash

echo "Testing Windows PowerShell invocation from WSL..."
echo "=============================================="

# Test 1: Check if Windows PowerShell exists
echo -e "\n1. Checking if Windows PowerShell is accessible:"
if [ -f "/mnt/c/Windows/System32/WindowsPowerShell/v1.0/powershell.exe" ]; then
    echo "✓ PowerShell executable found"
else
    echo "✗ PowerShell executable NOT found"
    exit 1
fi

# Test 2: Try to get PowerShell version
echo -e "\n2. Getting PowerShell version:"
/mnt/c/Windows/System32/WindowsPowerShell/v1.0/powershell.exe -NoProfile -Command '$PSVersionTable.PSVersion'

# Test 3: Simple echo test
echo -e "\n3. Simple echo test:"
/mnt/c/Windows/System32/WindowsPowerShell/v1.0/powershell.exe -NoProfile -Command 'Write-Host "Hello from Windows PowerShell via WSL!"'

# Test 4: Get current user
echo -e "\n4. Getting current Windows user:"
/mnt/c/Windows/System32/WindowsPowerShell/v1.0/powershell.exe -NoProfile -Command '[System.Security.Principal.WindowsIdentity]::GetCurrent().Name'

# Test 5: Test with a simple .ps1 file
echo -e "\n5. Testing .ps1 file execution:"
TEMP_PS1="/tmp/test_simple.ps1"
cat > "$TEMP_PS1" << 'EOF'
Write-Host "Script executed successfully!"
Write-Host "Current directory: $PWD"
Write-Host "Computer name: $env:COMPUTERNAME"
EOF

# Convert to Windows path
WIN_PS1=$(wslpath -w "$TEMP_PS1")
/mnt/c/Windows/System32/WindowsPowerShell/v1.0/powershell.exe -NoProfile -ExecutionPolicy Bypass -File "$WIN_PS1"

# Cleanup
rm -f "$TEMP_PS1"

echo -e "\n✓ All tests completed!"