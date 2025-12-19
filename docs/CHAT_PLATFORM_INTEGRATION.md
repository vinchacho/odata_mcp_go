# Chat Platform Integration Guide

This guide covers integrating the OData MCP Bridge with chat-based AI platforms. These platforms may require different transports than IDE integrations.

## GitHub Copilot

**Transport:** stdio
**Status:** ✅ Stable

GitHub Copilot supports MCP through agent mode, allowing it to use MCP servers as tools.

### Prerequisites

- GitHub Copilot subscription (with agent/MCP features)
- VS Code with GitHub Copilot extension
- OData MCP Bridge binary

### Configuration

Configure MCP servers in VS Code settings or `.vscode/mcp.json`:

```json
{
  "mcpServers": {
    "odata": {
      "command": "/path/to/odata-mcp",
      "args": [
        "--url", "https://services.odata.org/V4/Northwind/Northwind.svc/",
        "--lazy-metadata"
      ]
    }
  }
}
```

### Using with Copilot

Once configured, use the `@mcp` agent or invoke tools directly in Copilot Chat:

```
@mcp list all products from the OData service
```

Or in agent mode:

```
Use the odata tools to get customers from Germany
```

### Common Use Cases

1. **Data exploration**: "What entities are available in the OData service?"
2. **Query building**: "Get all orders placed in the last month"
3. **Code generation**: "Write a function that uses this OData service to fetch products"

### Platform-Specific Notes

- MCP support requires recent Copilot versions with agent capabilities
- Check GitHub Copilot documentation for current MCP setup instructions
- Uses stdio transport like IDE integrations

### Troubleshooting

| Problem | Solution |
|---------|----------|
| @mcp not recognized | Ensure Copilot agent mode is enabled |
| Tools not available | Verify MCP config in VS Code settings |
| Permission errors | Check binary path and permissions |

---

## ChatGPT

**Transport:** http (required)
**Status:** ✅ Stable

ChatGPT requires HTTP transport since it cannot spawn local processes. You'll need to expose the bridge as an HTTP endpoint, either locally (for testing) or via a public URL (for production).

### Prerequisites

- ChatGPT Plus subscription (for Custom GPTs)
- OData MCP Bridge running with HTTP transport
- Network accessibility (localhost for testing, public URL for production)

### Step 1: Start the Bridge with HTTP Transport

```bash
# Local testing (accessible only from your machine)
./odata-mcp \
  --url "https://services.odata.org/V4/Northwind/Northwind.svc/" \
  --transport http \
  --http-addr "127.0.0.1:8080" \
  --lazy-metadata

# Or with streamable HTTP (modern MCP protocol)
./odata-mcp \
  --url "https://your-odata-service.com/odata" \
  --transport streamable-http \
  --http-addr "0.0.0.0:8080" \
  --protocol-version "2025-06-18"
```

### Step 2: Expose Publicly (Production)

For ChatGPT to access your bridge, it needs a public URL. Options:

**Option A: ngrok (Quick Testing)**
```bash
ngrok http 8080
# Gives you: https://abc123.ngrok.io
```

**Option B: Cloud Deployment**
- Deploy to cloud provider (AWS, GCP, Azure)
- Use Docker: `docker run -p 8080:8080 odata-mcp --transport http ...`
- Add HTTPS via reverse proxy (nginx, Caddy)

### Step 3: Create Custom GPT

1. Go to [ChatGPT](https://chat.openai.com) → Explore GPTs → Create
2. In the **Configure** tab, scroll to **Actions**
3. Click **Create new action**
4. Import the OpenAPI schema or define actions manually:

```yaml
openapi: 3.0.0
info:
  title: OData MCP Bridge
  version: 1.0.0
servers:
  - url: https://your-public-url.com
paths:
  /mcp:
    post:
      operationId: mcpCall
      summary: Call MCP tool
      requestBody:
        required: true
        content:
          application/json:
            schema:
              type: object
      responses:
        '200':
          description: MCP response
```

5. Save and test

### Common Use Cases

1. **Natural language queries**: "Show me all customers from the OData service"
2. **Data analysis**: "Summarize the product categories available"
3. **Report generation**: "Create a report of orders by region"

### Security Considerations

**Local/Development:**
- Use `127.0.0.1` binding (not `0.0.0.0`) for local testing
- ngrok URLs are temporary and change on restart

**Production:**
- Use HTTPS (required by ChatGPT)
- Implement authentication if exposing sensitive data
- Consider rate limiting
- Use `--read-only` to prevent modifications
- Monitor access logs

### Platform-Specific Notes

- ChatGPT Custom GPT creation process may change - refer to OpenAI docs
- Use `--protocol-version "2025-06-18"` for AI Foundry/OpenAI compatibility
- HTTP transport has no built-in authentication - add at network layer

### Troubleshooting

| Problem | Solution |
|---------|----------|
| "Failed to connect" | Check URL is publicly accessible, HTTPS required |
| CORS errors | Add CORS headers if using browser-based access |
| Timeout | Large services - use `--lazy-metadata` |
| "Invalid response" | Check `--protocol-version` matches client expectations |

---

## Transport Comparison

| Transport | Use Case | Command |
|-----------|----------|---------|
| `stdio` | Local IDE/desktop apps | (default) |
| `http` | Legacy HTTP + SSE | `--transport http` |
| `streamable-http` | Modern MCP protocol | `--transport streamable-http` |

## Security Best Practices

When exposing the bridge over HTTP:

1. **Use HTTPS** - Required for ChatGPT, recommended always
2. **Bind locally when possible** - Use `127.0.0.1` not `0.0.0.0`
3. **Use read-only mode** - `--read-only` prevents data modifications
4. **Add authentication** - Via reverse proxy (nginx, Caddy) if needed
5. **Monitor logs** - Enable `--verbose` and watch for unusual patterns
6. **Limit exposure** - Only expose what's needed, use `--entities` to restrict

## Protocol Version Reference

| Client | Protocol Version | Flag |
|--------|-----------------|------|
| Claude Desktop | 2024-11-05 | (default) |
| AI Foundry / OpenAI | 2025-06-18 | `--protocol-version "2025-06-18"` |

See [AI Foundry Compatibility Guide](../AI_FOUNDRY_COMPATIBILITY.md) for details.
