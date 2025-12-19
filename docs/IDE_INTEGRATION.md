# IDE Integration Guide

This guide covers configuring the OData MCP Bridge with IDE-based MCP clients. All platforms use stdio transport.

## Claude Desktop

**Transport:** stdio
**Status:** ✅ Stable (Reference Implementation)

### Prerequisites

- Claude Desktop installed
- OData MCP Bridge binary built or downloaded

### Configuration

Edit `~/Library/Application Support/Claude/claude_desktop_config.json` (macOS) or `%APPDATA%\Claude\claude_desktop_config.json` (Windows):

```json
{
  "mcpServers": {
    "odata-northwind": {
      "command": "/path/to/odata-mcp",
      "args": [
        "--url", "https://services.odata.org/V4/Northwind/Northwind.svc/"
      ]
    }
  }
}
```

### Common Use Cases

1. **Query products**: "List all products with price over $20"
2. **Explore schema**: "What entities are available in this OData service?"
3. **Filter data**: "Show orders from customer ALFKI"

### Platform-Specific Notes

- Restart Claude Desktop after config changes
- Check `~/Library/Logs/Claude/` for MCP logs on macOS
- Protocol version 2024-11-05 (default) works correctly

---

## Cline

**Transport:** stdio
**Status:** ✅ Stable

### Prerequisites

- VS Code with Cline extension installed
- OData MCP Bridge binary in accessible location

### Configuration

Create or edit `.cline/mcp.json` in your workspace root (workspace-specific) or `~/.cline/mcp.json` (global):

```json
{
  "mcpServers": {
    "odata-sap": {
      "command": "/path/to/odata-mcp",
      "args": [
        "--url", "https://your-sap-server.com/sap/opu/odata/sap/API_BUSINESS_PARTNER",
        "--user", "your-user",
        "--password", "your-password",
        "--lazy-metadata"
      ]
    }
  }
}
```

### Common Use Cases

1. **SAP integration**: "Get all business partners from SAP"
2. **Code generation**: "Create a function that fetches customer data using this OData service"
3. **Data exploration**: "What fields are available on the Products entity?"

### Platform-Specific Notes

- Workspace config (`.cline/mcp.json`) takes precedence over global
- Use `--lazy-metadata` for large SAP services to reduce tool count
- Cline shows tool list in sidebar - verify tools appear after config

### Troubleshooting

| Problem | Solution |
|---------|----------|
| Tools not appearing | Check config JSON syntax, restart VS Code |
| "Command not found" | Use absolute path to binary |
| Too many tools displayed | Add `--lazy-metadata` flag |

---

## Roo Code

**Transport:** stdio
**Status:** ✅ Stable

### Prerequisites

- VS Code with Roo Code extension installed
- OData MCP Bridge binary

### Configuration

Similar to Cline, create `.roo/mcp.json` in workspace or `~/.roo/mcp.json` globally:

```json
{
  "mcpServers": {
    "odata": {
      "command": "/path/to/odata-mcp",
      "args": [
        "--url", "https://services.odata.org/V4/Northwind/Northwind.svc/",
        "--read-only"
      ]
    }
  }
}
```

### Common Use Cases

1. **Data queries**: "Count all orders in the database"
2. **Schema exploration**: "Describe the Customer entity structure"
3. **Filtered retrieval**: "Get products in category 'Beverages'"

### Platform-Specific Notes

- Config structure matches Cline
- Supports all OData MCP Bridge flags
- Use `--read-only` for safe exploration

### Troubleshooting

| Problem | Solution |
|---------|----------|
| Config not recognized | Ensure `.roo/mcp.json` path is correct |
| Binary permission denied | Run `chmod +x /path/to/odata-mcp` |
| Connection timeout | Check `--url` accessibility, add `--verbose` |

---

## Cursor

**Transport:** stdio
**Status:** ✅ Stable

### Prerequisites

- Cursor IDE installed (with MCP support)
- OData MCP Bridge binary

### Configuration

Edit `~/.cursor/mcp.json` or use Cursor's Settings UI:

```json
{
  "mcpServers": {
    "odata": {
      "command": "/path/to/odata-mcp",
      "args": [
        "--url", "https://your-odata-service.com/odata",
        "--lazy-threshold", "100"
      ]
    }
  }
}
```

### Common Use Cases

1. **Code assistance**: "Write a query to get all active customers"
2. **API exploration**: "What operations can I perform on Orders?"
3. **Data retrieval**: "Fetch the first 10 products sorted by name"

### Platform-Specific Notes

- MCP support added in recent Cursor versions
- Check Cursor settings for MCP configuration UI option
- `--lazy-threshold 100` auto-enables lazy mode for large services

### Troubleshooting

| Problem | Solution |
|---------|----------|
| MCP not available | Update Cursor to latest version |
| Config location unclear | Check Cursor docs or use Settings UI |
| Tools not loading | Verify binary path, check Cursor logs |

---

## Windsurf

**Transport:** stdio
**Status:** ✅ Stable

### Prerequisites

- Windsurf IDE installed
- OData MCP Bridge binary

### Configuration

Create `~/.windsurf/mcp.json` or check Windsurf documentation for current config location:

```json
{
  "mcpServers": {
    "odata": {
      "command": "/path/to/odata-mcp",
      "args": [
        "--url", "https://your-odata-service.com/odata"
      ]
    }
  }
}
```

### Common Use Cases

1. **Quick queries**: "Show me the schema of this OData service"
2. **Data access**: "Get customer details for ID 'ALFKI'"
3. **Filtering**: "List all products with stock below 10"

### Platform-Specific Notes

- Config location may vary by Windsurf version
- Standard MCP stdio protocol supported
- Test with simple query after configuration

### Troubleshooting

| Problem | Solution |
|---------|----------|
| Config not found | Check Windsurf docs for current config path |
| Connection issues | Add `--verbose` to debug, check network |
| Slow startup | Large metadata; use `--lazy-metadata` |

---

## Common Troubleshooting (All Platforms)

| Problem | Cause | Solution |
|---------|-------|----------|
| No tools appear | Config syntax error | Validate JSON, check commas |
| "Binary not found" | Wrong path | Use absolute path: `/full/path/to/odata-mcp` |
| CSRF 403 errors | SAP token expired | Automatic retry handles this; check credentials |
| Timeout on startup | Large metadata | Use `--lazy-metadata` or increase timeout |
| Too many tools | Large OData service | Use `--lazy-metadata` (10 generic tools) |
| "Method not allowed" | Read-only service | Use `--read-only` to hide write operations |

## Environment Variables

All platforms support environment variable configuration:

```bash
export ODATA_URL="https://your-service.com/odata"
export ODATA_USER="username"
export ODATA_PASSWORD="password"
export ODATA_LAZY_METADATA="true"
```

Then simplify your config:

```json
{
  "mcpServers": {
    "odata": {
      "command": "/path/to/odata-mcp"
    }
  }
}
```
