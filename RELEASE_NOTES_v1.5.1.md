# OData MCP Bridge v1.5.1

## 🎉 Release Highlights

This release adds **AI Foundry compatibility** through configurable MCP protocol versions, allowing seamless integration with AI Foundry agents and other MCP clients that use different protocol versions.

## ✨ New Features

### AI Foundry Compatibility
- **Configurable Protocol Version**: New `--protocol-version` flag to override the MCP protocol version
- **Support for Protocol Version 2025-06-18**: Full compatibility with AI Foundry's MCP client
- **Reordered Response Fields**: JSON response fields now match AI Foundry's expected ordering
- **Environment Variable Support**: Can set `ODATA_PROTOCOL_VERSION` for configuration

### Usage Examples

#### For AI Foundry Integration
```bash
./odata-mcp --service "https://your-service.com/odata" --protocol-version "2025-06-18"
```

#### Default (Claude, etc.)
```bash
./odata-mcp --service "https://your-service.com/odata"
# Uses default protocol version 2024-11-05
```

## 🐛 Bug Fixes
- Fixed protocol version negotiation issues with AI Foundry
- Corrected JSON field ordering in initialization responses

## 📚 Documentation
- Added comprehensive AI Foundry compatibility guide (`AI_FOUNDRY_COMPATIBILITY.md`)
- Includes troubleshooting steps and version compatibility matrix

## 🔄 Backwards Compatibility
- Default behavior remains unchanged for existing Claude users
- Protocol version 2024-11-05 remains the default

## 💾 Binary Downloads

Pre-built binaries are available for:
- **macOS**: Intel (amd64) and Apple Silicon (arm64)
- **Linux**: amd64, arm64, and arm
- **Windows**: amd64 (64-bit), 386 (32-bit), and arm64

## 🔧 Installation

### macOS/Linux
```bash
# Download the appropriate binary for your platform
tar -xzf odata-mcp-v1.5.1-<platform>-<arch>.tar.gz
chmod +x odata-mcp-v1.5.1-<platform>-<arch>
mv odata-mcp-v1.5.1-<platform>-<arch> /usr/local/bin/odata-mcp
```

### Windows
```powershell
# Extract the ZIP file and add to PATH
# Or run directly: .\odata-mcp-v1.5.1-windows-amd64.exe
```

## 🙏 Acknowledgments

Special thanks to **Padmaja Tandra** and **Shayak** from the AI Foundry team for reporting the compatibility issue and providing detailed debugging information!

## 📝 Changelog Since v1.5.0

- c53c387 feat: Add AI Foundry compatibility with protocol version flexibility
- 6803749 feat: Add automatic GUID formatting for SAP OData services

## Version Compatibility Matrix

| Client | Protocol Version | Command Line Option |
|--------|-----------------|-------------------|
| AI Foundry | 2025-06-18 | `--protocol-version "2025-06-18"` |
| Claude | 2024-11-05 | (default, no option needed) |
| Custom | Any | `--protocol-version "YOUR-VERSION"` |