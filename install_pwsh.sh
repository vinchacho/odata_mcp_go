#!/bin/bash

# Install PowerShell on Ubuntu/Debian WSL
echo "Installing PowerShell Core on WSL..."

# Update package list
sudo apt-get update

# Install prerequisites
sudo apt-get install -y wget apt-transport-https software-properties-common

# Get Microsoft signing key
wget -q "https://packages.microsoft.com/config/ubuntu/$(lsb_release -rs)/packages-microsoft-prod.deb"

# Register Microsoft repository
sudo dpkg -i packages-microsoft-prod.deb

# Update package list with Microsoft repository
sudo apt-get update

# Install PowerShell
sudo apt-get install -y powershell

echo "PowerShell installed. Run 'pwsh' to start PowerShell Core"