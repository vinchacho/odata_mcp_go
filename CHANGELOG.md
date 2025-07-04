# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- **Azure AD (AAD) authentication support**
  - Device code flow for CLI-friendly authentication
  - Browser-based authentication flow with `--aad-browser`
  - Automatic token refresh and caching
  - Support for federated SAP systems
  - Multi-factor authentication (MFA) support
  - Secure token storage with file-based cache
  - Authentication tracing with `--aad-trace` for debugging
  - `--auth-aad` flag to enable AAD authentication
  - `--aad-tenant`, `--aad-client-id`, `--aad-scopes`, `--aad-cache` configuration options
- **SAML browser authentication support**
  - `--auth-saml-browser` flag for SAML-based systems
  - Browser-assisted authentication with manual cookie extraction
  - Step-by-step instructions for cookie extraction
  - Support for MYSAPSSO2 and other SAP cookies
- **Windows integrated authentication**
  - `--auth-windows` flag for automatic Windows authentication
  - Uses PowerShell with Windows credentials
  - Handles SAML/ADFS redirects automatically
  - No manual steps required on domain-joined machines
- **Advanced SAML authentication methods**
  - `--auth-webview2` flag for WebView2 (Edge) authentication on Windows
  - `--auth-chrome` flag for Chrome automation (cross-platform)
  - `--auth-chrome-headless` flag for headless Chrome authentication
  - Fully automated SAML handling with cookie extraction
  - No manual browser steps required
  - Seamless authentication for corporate environments
- **Read-only mode flags** (`--read-only`/`-ro` and `--read-only-but-functions`/`-robf`)
  - Hide all modifying operations (create, update, delete) in read-only mode
  - Allow function imports in read-only-but-functions mode
- **MCP trace logging** (`--trace-mcp`) for debugging protocol communication
  - Captures all incoming/outgoing MCP messages
  - Saves detailed trace logs to `/tmp/mcp_trace_*.log` (Linux/WSL) or `%TEMP%\mcp_trace_*.log` (Windows)
  - Helps diagnose client compatibility issues
- **Flexible hint system** for service-specific guidance
  - JSON-based hint configuration with wildcard pattern matching
  - `--hints-file` flag to load custom hint files
  - `--hint` flag for direct CLI hint injection
  - Priority-based hint merging for multiple matching patterns
  - Default hints for SAP OData services including HTTP 501 workarounds
  - Hints appear in `odata_service_info` tool response
- **Full MCP protocol compliance**
  - Added missing `resources/list` and `prompts/list` handlers
  - Proper capability declarations in initialize response
  - Strict JSON-RPC 2.0 validation
- **Enhanced error handling**
  - Better null ID handling for Claude Desktop compatibility
  - Proper JSON-RPC error responses
  - Detailed error categorization
- **HTTP/SSE transport support** (in addition to stdio)
  - Support for Server-Sent Events transport with `--transport http`
  - Configurable HTTP server address with `--http-addr`
- **Legacy date format support** for SAP compatibility
  - Automatic conversion of SAP date formats
  - `--no-legacy-dates` flag to disable conversion
- **Enhanced response features**
  - Response size limits with `--max-response-size`
  - Item count limits with `--max-items`
  - Pagination hints with `--pagination-hints`
  - Response metadata inclusion with `--response-metadata`
  - Date conversion options with `--convert-dates-from-sap`
- OData v4 support with automatic version detection
- Query parameter translation ($inlinecount to $count for v4)
- Automatic versioning based on git tags and commit history
- GitHub Actions workflows for automated releases
- WSL-specific build targets
- Comprehensive test suite for v4 functionality

### Changed
- **Improved MCP protocol implementation**
  - Initialize response now includes all required capabilities (tools, resources, prompts)
  - Better compatibility with different MCP clients (Claude Desktop, RooCode, GitHub Copilot)
  - Stricter validation to prevent client-side errors
- **ID handling improvements**
  - Null IDs are converted to 0 for better client compatibility
  - Proper handling of different ID types (string, number, null)
- Improved response parsing for both v2 and v4 formats
- Enhanced error handling with detailed OData error messages
- Makefile now uses dynamic versioning instead of hardcoded version
- Added `--cookies` as an alias for `--cookie-file` for backward compatibility

### Fixed
- **Claude Desktop Zod validation errors**
  - Missing capability declarations that caused validation failures
  - Null ID handling that triggered type errors
  - Missing method handlers for resources and prompts
- **MCP client compatibility issues**
  - Fixed issues preventing tools from appearing in RooCode
  - Resolved connection problems with various MCP clients
  - Better error response formatting
- Multiple main function declarations in test files
- Type assertion panics in response parser
- Count value parsing for v2 string responses

## [0.1.0] - 2024-06-30

### Added
- Initial Go implementation of OData MCP Bridge
- Support for OData v2 services
- Dynamic tool generation based on metadata
- Basic auth and cookie authentication
- SAP OData extensions with CSRF token support
- Comprehensive CRUD operations
- Advanced query support with OData query options
- Function import support
- Cross-platform builds for Linux, Windows, and macOS

### Notes
- This is a Go port of the Python OData-MCP bridge
- Maintains CLI compatibility with the original implementation

[Unreleased]: https://github.com/odata-mcp/go/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/odata-mcp/go/releases/tag/v0.1.0