# OData MCP Bridge (Go) — Specification

**Version**: 1.1
**Status**: Approved
**Applies to**: odata-mcp binary v1.6.x
**Last Updated**: 2025-01-14

---

## 1. Overview

The OData MCP Bridge is a Go binary that:

1. **Discovers** an OData service's structure via `$metadata`
2. **Generates** MCP tools for entity sets and function imports
3. **Serves** those tools over one of three transports: stdio, HTTP/SSE, or Streamable HTTP
4. **Proxies** tool invocations to the OData service, handling auth, CSRF, and response formatting

### OData Version Support

| Version | Metadata Format | Query Differences | Date Format |
|---------|----------------|-------------------|-------------|
| v2 | CSDL (Atom) | `$inlinecount=allpages`, `/Date()/` | Epoch `/Date(...)/` |
| v4 | CSDL (JSON/XML) | `$count=true`, ISO dates | ISO 8601 |

The client auto-detects version from metadata and translates query parameters accordingly.

---

## 2. Terminology

| Term | Definition |
|------|------------|
| **OData service** | HTTP endpoint exposing entities/functions via OData protocol |
| **Metadata** | XML/JSON schema from `$metadata` endpoint |
| **Entity Set** | Named collection of entities (e.g., `Products`) |
| **Entity Type** | Schema for entities (properties, keys, nav props) |
| **Function Import** | Callable operation defined in metadata (v2) |
| **Action** | POST-based callable (v4, treated as function) |
| **MCP tool** | JSON-RPC callable exposed to MCP clients |
| **Transport** | Communication layer (stdio, HTTP/SSE, Streamable HTTP) |
| **Protocol version** | MCP handshake version (`2024-11-05`, `2025-06-18`) |
| **Hint** | Service-specific guidance injected into `odata_service_info` |
| **CSRF token** | SAP-specific anti-forgery header for mutations |

---

## 3. Supported Deployment Modes

### 3.1 STDIO (Default)
- Primary use case: Claude Desktop
- Communication: JSON-RPC over stdin/stdout
- No network exposure

### 3.2 HTTP/SSE (Legacy)
- Endpoints: `/health` (GET), `/sse` (GET), `/rpc` (POST)
- Default address: `localhost:8080`
- **No authentication** — localhost-only unless `--i-am-security-expert-i-know-what-i-am-doing` flag

### 3.3 Streamable HTTP (Modern)
- Endpoints: `/mcp` (POST, with optional SSE upgrade), `/health` (GET), `/sse` (GET, legacy)
- Session management with stream contexts
- Same security posture as HTTP/SSE

### 3.4 Security Constraints
- Non-localhost bind requires explicit expert flag
- HTTP transports have **NO AUTHENTICATION** — MCP protocol lacks auth
- Credentials (user/password) are NOT logged

---

## 4. External Interfaces / Contracts

### 4.1 MCP Contract

#### Initialization
```json
{"jsonrpc":"2.0","method":"initialize","params":{...},"id":0}
```
Response includes:
- `capabilities.tools.listChanged: true`
- `protocolVersion`: configurable (default `2024-11-05`, AI Foundry uses `2025-06-18`)
- `serverInfo.name`: `odata-mcp-bridge`

#### Tool Listing
`tools/list` returns all generated tools in insertion order (or alphabetical if `--sort-tools`).

#### Tool Invocation
`tools/call` with `{"name": "<tool>", "arguments": {...}}`.
Response: `{"content": [{"type":"text","text":"<JSON result>"}]}`.

#### Notifications
- `initialized`: No response (handled silently)

#### Error Codes
| MCP Code | Mapped From |
|----------|------------|
| -32602 | HTTP 400, 404, 422, invalid params |
| -32603 | HTTP 401, 403, 409, 429, 5xx, CSRF, timeout, network |
| -32600 | Invalid JSON-RPC |
| -32601 | Unknown method |

### 4.2 OData Contract

#### Metadata Discovery
- GET `{service}/$metadata` with `Accept: application/xml`
- Parses CSDL to extract entity sets, types, function imports
- Falls back to service document if metadata fails

#### Assumed Endpoints
| Endpoint | Method | Purpose |
|----------|--------|---------|
| `$metadata` | GET | Schema discovery |
| `{EntitySet}` | GET | List entities |
| `{EntitySet}({key})` | GET | Single entity |
| `{EntitySet}` | POST | Create |
| `{EntitySet}({key})` | PUT/PATCH/MERGE | Update |
| `{EntitySet}({key})` | DELETE | Delete |
| `{FunctionImport}` | GET/POST | Call function |

#### Query Options Supported
`$filter`, `$select`, `$expand`, `$orderby`, `$top`, `$skip`, `$count`/`$inlinecount`, `$search`, `$format`

### 4.3 HTTP Endpoints (Transports)

#### Streamable HTTP
| Path | Method | Status Codes | Description |
|------|--------|--------------|-------------|
| `/mcp` | POST | 200, 400, 403, 405 | Main MCP endpoint |
| `/health` | GET | 200 | `{"status":"ok","transport":"streamable-http"}` |
| `/sse` | GET | 200, 400 | Legacy SSE (redirects to /mcp) |

#### HTTP/SSE
| Path | Method | Status Codes | Description |
|------|--------|--------------|-------------|
| `/health` | GET | 200 | Health check |
| `/sse` | GET | 200 | SSE stream |
| `/rpc` | POST | 200 | JSON-RPC endpoint |

---

## 5. CLI Contract

### 5.1 Service & Auth

| Flag | Env Var | Default | Description |
|------|---------|---------|-------------|
| `--service` | `ODATA_SERVICE_URL`, `ODATA_URL` | (required) | OData service URL (positional argument also accepted) |
| `-u, --user` | `ODATA_USERNAME`, `ODATA_USER` | | Basic auth username |
| `-p, --password` | `ODATA_PASSWORD`, `ODATA_PASS` | | Basic auth password |
| `--pass` | | | Basic auth password (alias for `--password`) |
| `--cookie-file` | `ODATA_COOKIE_FILE` | | Netscape cookie file |
| `--cookie-string` | `ODATA_COOKIE_STRING` | | Inline cookies |

**Constraint**: Only one auth method allowed at a time.

### 5.2 Transport & Network

| Flag | Default | Description |
|------|---------|-------------|
| `--transport` | `stdio` | `stdio`, `http`, `streamable-http` |
| `--http-addr` | `localhost:8080` | Bind address |
| `--i-am-security-expert...` | false | Allow non-localhost HTTP |

### 5.3 Protocol & Version

| Flag | Default | Description |
|------|---------|-------------|
| `--protocol-version` | `2024-11-05` | MCP protocol version for `initialize` |

### 5.4 Tool Generation & Naming

| Flag | Default | Description |
|------|---------|-------------|
| `--tool-prefix` | | Custom prefix |
| `--tool-postfix` | auto-derived | Custom postfix |
| `--no-postfix` | false | Use prefix mode |
| `--tool-shrink` | false | Shortened operation names (upd, del) |
| `--sort-tools` | true | Alphabetical tool order |
| `--entities` | (all) | Comma-separated filter (wildcards: `*`) |
| `--functions` | (all) | Comma-separated filter (wildcards: `*`) |
| `-c, --claude-code-friendly` | false | Remove `$` prefix from params |

### 5.5 Safety & Read-Only

| Flag | Description |
|------|-------------|
| `--read-only, -ro` | Hide create, update, delete, and functions |
| `--read-only-but-functions, -robf` | Hide CUD but allow functions |
| `--enable` | Whitelist operations (C,S,F,G,U,D,A,R) |
| `--disable` | Blacklist operations |

**Constraint**: `--enable` and `--disable` are mutually exclusive.

### 5.6 Output Limits & Pagination

| Flag | Default | Description |
|------|---------|-------------|
| `--max-items` | 100 | Max entities returned (ceiling: 10000) |
| `--max-response-size` | 5MB | Max response bytes |
| `--pagination-hints` | false | Add `has_more`, `suggested_next_call` |
| `--response-metadata` | false | Include `__metadata` blocks |

### 5.7 Date & SAP Conversions

| Flag | Default | Description |
|------|---------|-------------|
| `--legacy-dates` | true | Convert `/Date()/` to ISO |
| `--no-legacy-dates` | false | Disable date conversion |

### 5.8 Timeouts

| Flag | Default | Description |
|------|---------|-------------|
| `--http-timeout` | 30 | HTTP request timeout in seconds |
| `--metadata-timeout` | 60 | Metadata fetch timeout in seconds |

### 5.9 Tracing & Debugging

| Flag | Description |
|------|-------------|
| `-v, --verbose` | Verbose stderr logging |
| `--debug` | Alias for verbose |
| `--trace` | Print tools and exit |
| `--trace-mcp` | Log MCP messages to temp file |
| `--verbose-errors` | Include query options in errors |

### 5.10 Hints

| Flag | Default | Description |
|------|---------|-------------|
| `--hints-file` | `hints.json` (binary dir) | Custom hints file |
| `--hint` | | Inline hint JSON/text |

---

## 6. Dynamic Tool Generation Rules

### 6.1 Tools per Entity Set

| Tool | Condition | Operation Letter |
|------|-----------|-----------------|
| `filter_{EntitySet}` | Always | F |
| `count_{EntitySet}` | Always | F |
| `search_{EntitySet}` | `Searchable=true` | S |
| `get_{EntitySet}` | Always | G |
| `create_{EntitySet}` | `Creatable=true` + not read-only | C |
| `update_{EntitySet}` | `Updatable=true` + not read-only | U |
| `delete_{EntitySet}` | `Deletable=true` + not read-only | D |

### 6.2 Function Import Tools

- One tool per function import
- Skipped if actions disabled (`--disable a`) or read-only mode
- HTTP method from metadata (default: GET)
- POST functions treated as actions (modifying)

### 6.3 Service Info Tool

Always generated: `odata_service_info` — returns metadata summary and matched hints.

### 6.4 Naming Convention

**Default (postfix)**:
`{operation}_{EntitySet}_for_{ServiceID}`
Example: `filter_Products_for_NorthwSvc`

**Prefix mode** (`--no-postfix`):
`{ServiceID}_{EntitySet}_{operation}`

**Tool shrink**:
`update` → `upd`, `delete` → `del`

### 6.5 Lazy Metadata Mode

When enabled, the bridge generates a fixed set of 10 generic tools instead of per-entity tools to reduce token usage for large services.

| Flag | Default | Description |
|------|---------|-------------|
| `--lazy-metadata` | false | Enable lazy metadata mode |
| `--lazy-threshold` | 0 | Auto-enable lazy mode if estimated tool count exceeds threshold |

### 6.6 Tool Schema

```json
{
  "name": "filter_Products_for_NorthwSvc",
  "description": "List/filter Products entities with OData query options",
  "inputSchema": {
    "type": "object",
    "properties": {
      "$filter": {"type": "string", "description": "OData filter expression"},
      "$select": {"type": "string"},
      "$expand": {"type": "string"},
      "$orderby": {"type": "string"},
      "$top": {"type": "integer"},
      "$skip": {"type": "integer"},
      "$count": {"type": "boolean"}
    }
  }
}
```

---

## 7. Auth & Security Requirements

### 7.1 Supported Auth Methods
1. **Basic Auth**: `-u`/`-p` flags or env vars
2. **Cookie Auth**: `--cookie-file` (Netscape format) or `--cookie-string`
3. **Anonymous**: No auth configured

### 7.2 SAP CSRF Handling
- CSRF token fetched before **every** modifying operation (POST/PUT/PATCH/MERGE/DELETE)
- On 403 + "CSRF validation failed": automatic refetch + retry (once)
- Session cookies captured and reused

### 7.3 HTTP Transport Security
- **No built-in authentication** for MCP clients
- Default bind: localhost only
- Non-localhost requires `--i-am-security-expert-i-know-what-i-am-doing`
- Security headers: `X-Content-Type-Options: nosniff`, `X-Frame-Options: DENY`

### 7.4 Credential Handling
- Credentials never logged (even in verbose mode)
- Token preview limited to 20 chars

---

## 8. Data Handling & Serialization Rules

### 8.1 Response Limiting
1. Apply `--max-items` truncation
2. Apply `--max-response-size` truncation
3. Add `metadata.truncated`, `metadata.warning` if truncated

### 8.2 Response Metadata
- `__metadata` blocks stripped unless `--response-metadata`

### 8.3 Date Conversion
- **Legacy dates** (default on): `/Date(1234567890000)/` → ISO 8601
- Bidirectional: ISO → legacy for create/update payloads

### 8.4 SAP GUID Conversion
- For SAP services: auto-transform `'uuid'` → `guid'uuid'` in filters
- Detection: URL contains `sap`, schema namespace, or SAP-specific annotations

### 8.5 Numeric Conversion
- Numbers converted to strings for SAP OData v2 compatibility (prevents "Failed to read property" errors)

---

## 9. Error Handling Semantics

### 9.1 OData Error Mapping
```
HTTP 4xx/5xx → parsed OData error → MCP error response
```

Error message includes: tool name, OData code, message, target, details.

### 9.2 Timeout Behavior
- Default HTTP client timeout: 30 seconds
- Maps to MCP error code -32603

### 9.3 Retry Policy

- CSRF 403: Single automatic retry with fresh token
- HTTP 5xx errors: Configurable retry with exponential backoff (v1.6.0+)
  - `--retry-max-attempts` (default: 3)
  - `--retry-initial-backoff-ms` (default: 100)
  - `--retry-max-backoff-ms` (default: 10000)
  - `--retry-backoff-multiplier` (default: 2.0)
- Retryable errors: 500, 502, 503, 504, connection reset, timeout
- Non-retryable: 4xx errors (except CSRF 403)

### 9.4 Credential Masking (v1.6.0+)

- Passwords masked in verbose output as `***`
- CSRF tokens truncated to first 20 characters
- Cookie values masked in logs
- Authorization headers redacted

---

## 10. Observability & Diagnostics

### 10.1 Verbose Mode (`-v`)
Logs to stderr:
- HTTP requests/responses
- Auth mode detection
- CSRF token lifecycle
- Filter transformations
- Date conversions

### 10.2 Trace Mode (`--trace`)
- Initializes bridge, prints tool list as JSON, exits
- Does not start transport

### 10.3 MCP Trace (`--trace-mcp`)
- Logs all JSON-RPC messages to temp file
- Location: `/tmp/mcp_trace_*.log` (Linux) or `%TEMP%\mcp_trace_*.log` (Windows)

---

## 11. Edge Cases & Failure Modes

### 11.1 Large Metadata / Tool Explosion
- Mitigate with `--entities`, `--functions` filters
- Wildcards: `Product*`, `*Order`

### 11.2 Transport Disconnects
- HTTP/SSE: Stale streams cleaned up after 5 minutes inactivity
- STDIO: Binary exits on EOF

### 11.3 Auth Expiry
- Cookie expiry: Not auto-refreshed (user must restart)
- Basic auth: Valid until service rejects

### 11.4 CSRF Token Expiry
- Refetched on 403 (once per request)

### 11.5 Partial Metadata
- Falls back to service document (minimal entity set list)
- Missing entity types: Logged, tools skipped

### 11.6 Protocol Version Mismatch
- Server echoes whatever `--protocol-version` is set to
- AI Foundry requires `2025-06-18`

### 11.7 OData v2/v4 Query Mismatches

- `$inlinecount` auto-translated to `$count` for v4
- Spaces encoded as `%20` (not `+`)

### 11.8 Thread Safety (v1.6.3+)

- SSE stream contexts use `sync.Once` to prevent double-close panics
- ResponseWriter writes protected by mutex (not goroutine-safe by default)
- Composite entity keys sorted alphabetically for deterministic URL generation
- MCP tool handlers receive HTTP request context for proper cancellation

---

## 12. Acceptance Criteria

| ID | Criterion | Verification |
|----|-----------|--------------|
| AC-1 | Binary starts and responds to `initialize` | `echo '{"jsonrpc":"2.0","method":"initialize","params":{},"id":0}' \| ./odata-mcp --service <url>` |
| AC-2 | `tools/list` returns non-empty array | Manual or test script |
| AC-3 | `--read-only` hides create/update/delete tools | `./odata-mcp --trace --read-only --service <url>` |
| AC-4 | HTTP transport refuses non-localhost without expert flag | `./odata-mcp --transport http --http-addr 0.0.0.0:8080 --service <url>` → error |
| AC-5 | CSRF retry succeeds on SAP service | Integration test with SAP endpoint |
| AC-6 | `--max-items 10` truncates response | Filter returning >10 entities shows truncation metadata |
| AC-7 | `--protocol-version 2025-06-18` echoed in initialize | JSON response check |
| AC-8 | `--entities "Product*"` filters tools | `--trace` output shows only matching tools |
| AC-9 | Legacy date conversion works | Response contains ISO dates, not `/Date()/` |
| AC-10 | `go test ./...` passes | `make test` |
| AC-11 | Retry on 5xx errors (v1.6.0+) | `--max-retries 2` retries on 503 |
| AC-12 | Credentials masked in logs (v1.6.0+) | `--verbose` shows `***` for passwords |
| AC-13 | Composite keys deterministic (v1.6.3+) | Same key map always generates same URL |
| AC-14 | SSE streams don't panic on close (v1.6.3+) | Concurrent stream cleanup doesn't crash |

---

## 13. Out of Scope / Non-Goals

1. **OAuth / API Key auth for MCP clients** — MCP protocol limitation
2. **Batching ($batch)** — Not implemented
3. **Deep insert / deep update** — Single entity ops only
4. **Custom actions with complex types** — Primitives only
5. **Streaming large responses** — Truncation used instead
6. **Automatic pagination traversal** — Hints provided, not auto-followed
7. **Schema validation of tool inputs** — Delegated to OData service
8. **Caching** — No local cache of metadata or responses

---

## Appendix A: Transport × Protocol Compatibility Matrix

| Transport | Claude Desktop | Claude Code CLI | AI Foundry | Custom MCP Client |
|-----------|---------------|-----------------|------------|-------------------|
| stdio | Yes (default) | Yes | No | Yes |
| http/sse | No | No | No | Yes |
| streamable-http | No | No | Yes (with --protocol-version) | Yes |

---

## Appendix B: Operation Type Codes

| Code | Operation | Included in "R" |
|------|-----------|-----------------|
| C | Create | No |
| S | Search | Yes |
| F | Filter/List | Yes |
| G | Get (single) | Yes |
| U | Update | No |
| D | Delete | No |
| A | Actions/Functions | No |
| R | Read (S+F+G) | — |
