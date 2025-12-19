# LLM & Platform Compatibility Guide

The OData MCP Bridge works with any MCP-compatible client. This guide covers supported platforms and their configuration requirements.

## Compatibility Matrix

| Platform | Transport | Status | Guide |
|----------|-----------|--------|-------|
| Claude Desktop | stdio | ✅ Stable | [IDE Integration](IDE_INTEGRATION.md#claude-desktop) |
| Cline | stdio | ✅ Stable | [IDE Integration](IDE_INTEGRATION.md#cline) |
| Roo Code | stdio | ✅ Stable | [IDE Integration](IDE_INTEGRATION.md#roo-code) |
| Cursor | stdio | ✅ Stable | [IDE Integration](IDE_INTEGRATION.md#cursor) |
| Windsurf | stdio | ✅ Stable | [IDE Integration](IDE_INTEGRATION.md#windsurf) |
| GitHub Copilot | stdio | ✅ Stable | [Chat Platforms](CHAT_PLATFORM_INTEGRATION.md#github-copilot) |
| ChatGPT | http | ✅ Stable | [Chat Platforms](CHAT_PLATFORM_INTEGRATION.md#chatgpt) |

## Which Guide Do I Need?

```
Are you using an IDE extension (Cline, Roo Code, Cursor, Windsurf)?
├── Yes → IDE_INTEGRATION.md
└── No
    ├── Claude Desktop? → IDE_INTEGRATION.md (stdio)
    ├── GitHub Copilot? → CHAT_PLATFORM_INTEGRATION.md (stdio)
    └── ChatGPT? → CHAT_PLATFORM_INTEGRATION.md (requires HTTP)
```

## Transport Overview

**Stdio (Standard I/O):** The bridge runs as a local process. The client spawns it and communicates via stdin/stdout. This is the most common and simplest setup.

**HTTP/SSE:** The bridge runs as a web server. Required for ChatGPT Custom GPTs and remote access scenarios. Use `--transport http` or `--transport streamable-http`.

## Quick Start

Most platforms use stdio transport with identical configuration:

```json
{
  "mcpServers": {
    "odata": {
      "command": "/path/to/odata-mcp",
      "args": ["--url", "https://your-odata-service.com/odata"]
    }
  }
}
```

See platform-specific guides for config file locations and additional options.

## Recommended Flags by Use Case

| Use Case | Recommended Flags |
|----------|-------------------|
| Large SAP service (500+ entities) | `--lazy-metadata` or `--lazy-threshold 100` |
| Read-only access | `--read-only` |
| Debugging connection issues | `--verbose` |
| AI Foundry / OpenAI MCP | `--protocol-version "2025-06-18"` |

## Future: Remote Deployment

For exposing the bridge over a network (Docker, cloud hosting, reverse proxy), see the roadmap. Current focus is local stdio-based usage.

## Related Documentation

- [IDE Integration Guide](IDE_INTEGRATION.md) - Claude Desktop, Cline, Roo Code, Cursor, Windsurf
- [Chat Platform Integration](CHAT_PLATFORM_INTEGRATION.md) - ChatGPT, GitHub Copilot
- [AI Foundry Compatibility](../AI_FOUNDRY_COMPATIBILITY.md) - Protocol version configuration
- [README](../README.md) - Full CLI reference
