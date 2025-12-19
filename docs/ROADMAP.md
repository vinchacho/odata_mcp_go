# OData MCP Bridge — Roadmap & Backlog

*Last Updated: December 2025*

This document is the single source of truth for planned improvements, active development phases, and future backlog items.

---

## Quick Reference

| Version | Status | Theme |
|---------|--------|-------|
| v1.6.0 | ✅ Complete | Foundation (Credential Masking, Retry) |
| v1.6.1 | ✅ Complete | Quick Wins (Code Quality) |
| v1.6.2 | ✅ Complete | Critical Bug Fixes (Concurrency, Parser, SSE) |
| v1.6.3 | ✅ Complete | Stability & Thread Safety |
| v1.6.4 | Parked | Test Coverage |
| v1.6.5 | ✅ Complete | Reliability & DX Quick Fixes |
| v1.6.6 | Next | Feature Polish |
| v1.7.0 | ✅ Complete | Token-Optimized Discovery |
| v1.8.0 | Planned | Skill Generator |
| v1.9.0 | Planned | Advanced Features |
| v2.0.0 | Backlog | Multi-LLM Platform Support |

---

## Active Phases

### v1.6.0 — Foundation ✅ COMPLETE

**Theme**: Security hygiene + reliability basics

#### Credential Masking in Logs

Prevents accidental credential exposure in verbose output.

**Files**: `internal/debug/masking.go`, `internal/debug/masking_test.go`

**Features**:
- Passwords completely masked as `***`
- CSRF tokens show only last 8 characters (`****abcd1234`)
- Authorization headers show type but mask credentials (`Basic ****`)
- URLs mask passwords in userinfo and sensitive query parameters
- Cookie values masked in debug output

#### Exponential Backoff Retry

Retries transient failures with jitter to prevent thundering herd.

**Files**: `internal/client/retry.go`, `internal/client/retry_test.go`, `internal/client/retry_integration_test.go`

**CLI Flags**:
| Flag | Default | Description |
|------|---------|-------------|
| `--retry-max-attempts` | 3 | Maximum retry attempts |
| `--retry-initial-backoff-ms` | 100 | Initial backoff delay |
| `--retry-max-backoff-ms` | 10000 | Maximum backoff delay |
| `--retry-backoff-multiplier` | 2.0 | Backoff multiplier |

**Retryable Status Codes**: 429, 500, 502, 503, 504

**Milestone Checklist**:
- [x] Implement credential masking
- [x] Write masking unit tests
- [x] Implement exponential backoff
- [x] Write retry unit tests
- [x] Add CLI flags for retry configuration
- [x] Update documentation (README, CHANGELOG)

---

### v1.6.1 — Quick Wins ✅ COMPLETE

**Theme**: Low-effort code quality fixes

These are 5-15 minute fixes that should be done before larger features.

#### CQ-1: Replace Deprecated io/ioutil

**Effort**: 5 min | **Risk**: Low

```go
// Current (internal/mcp/server.go:7)
import "io/ioutil"
log.SetOutput(ioutil.Discard)

// Target
import "io"
log.SetOutput(io.Discard)
```

**Validation**: `grep -r "io/ioutil" internal/ && go build ./...`

---

#### CQ-2: Fix Swallowed JSON Marshal Error

**Effort**: 10 min | **Risk**: Low

```go
// Current (internal/debug/trace.go:70)
jsonData, _ := json.Marshal(entry)

// Target
jsonData, err := json.Marshal(entry)
if err != nil {
    fmt.Fprintf(os.Stderr, "[TRACE ERROR] Failed to marshal entry: %v\n", err)
    return
}
```

---

#### CQ-3: Fix Swallowed Body Read Error

**Effort**: 10 min | **Risk**: Low

In CSRF retry path (`internal/client/client.go`), handle body read error instead of ignoring it.

---

#### CQ-4: Fix Constant Inconsistencies

**Effort**: 15 min | **Risk**: Low

Align `constants.go` defaults with CLI defaults:

| Constant | constants.go | main.go | Action |
|----------|-------------|---------|--------|
| MaxResponseSize | 10MB | 5MB | Align to 5MB |
| MaxItems | 1000 | 100 | Align to 100 |

---

#### Milestone Checklist

- [x] CQ-1: Replace `io/ioutil` with `io`
- [x] CQ-2: Handle JSON marshal error in trace.go
- [x] CQ-3: Handle body read error in CSRF path (already fixed)
- [x] CQ-4: Align constant defaults

---

### v1.6.3 — Stability & Thread Safety ✅ COMPLETE

**Theme**: Concurrency safety and deterministic behavior

#### SS-1: Composite Key Determinism

**Problem**: Map iteration in Go is non-deterministic, causing composite entity keys to generate different URLs on different runs.

**Solution**: Sort key names alphabetically before building URL predicate.

**File**: `internal/client/client.go`

```go
// Composite key - iterate deterministically
keys := make([]string, 0, len(key))
for k := range key {
    keys = append(keys, k)
}
sort.Strings(keys)
```

---

#### SS-2: SSE Double-Close Panic

**Problem**: `close(stream.done)` called multiple times in concurrent cleanup scenarios.

**Solution**: Guard close with `sync.Once`.

**File**: `internal/transport/http/streamable.go`

```go
type streamContext struct {
    // ...
    closeOnce sync.Once // Ensures done channel is closed exactly once
}

// Usage
stream.closeOnce.Do(func() { close(stream.done) })
```

---

#### SS-3: Concurrent ResponseWriter Writes

**Problem**: `http.ResponseWriter` is not goroutine-safe; concurrent writes cause data corruption.

**Solution**: Protect writes with mutex.

**File**: `internal/transport/http/streamable.go`

```go
type streamContext struct {
    // ...
    writeMu sync.Mutex // Guards writes to ResponseWriter
}
```

---

#### SS-4: Context Propagation in MCP Handlers

**Problem**: Tool handlers used server context instead of HTTP request context, preventing proper cancellation.

**Solution**: Pass request context through handler chain.

**File**: `internal/mcp/server.go`

```go
func (s *Server) handleToolsCallV2(ctx context.Context, req *Request) (*transport.Message, error) {
    result, err := handler(ctx, params)  // Uses passed context, not s.ctx
}
```

---

#### SS-5: MCP Mock Server for Tests

**Problem**: Protocol tests used unreliable mock causing test failures.

**Solution**: Implement proper `httptest.Server` based mock with correct JSON-RPC handling.

**File**: `internal/test/mcp_protocol_test.go`

---

#### Milestone Checklist

- [x] SS-1: Composite key deterministic ordering
- [x] SS-2: SSE double-close panic fix
- [x] SS-3: Concurrent ResponseWriter mutex
- [x] SS-4: Context propagation in MCP handlers
- [x] SS-5: MCP mock server for tests

---

### v1.6.4 — Test Coverage (Parked)

**Theme**: Fill critical unit test gaps

Current test coverage is heavily skewed toward integration tests (91% integration, 9% unit). These unit tests fill critical gaps.

**Target**: >70% coverage for core packages

#### TC-1: Config Package Tests

**Effort**: 1-2 hours | **Current Coverage**: 0%

**Functions to Test**:
| Function | Test Cases |
|----------|------------|
| `HasBasicAuth()` | empty, partial, complete credentials |
| `HasCookieAuth()` | empty, populated |
| `UsePostfix()` | `NoPostfix` true/false |
| `IsReadOnly()` | none, `--read-only`, `--read-only-but-functions` |
| `IsOperationEnabled(rune)` | enable/disable combos, R expansion |

**File**: `internal/config/config_test.go` (NEW)

---

#### TC-2: Hint Manager Tests

**Effort**: 1-2 hours | **Current Coverage**: 0%

**Functions to Test**:
| Function | Test Cases |
|----------|------------|
| `LoadFromFile(path)` | valid file, missing file, invalid JSON |
| `SetCLIHint(hint)` | JSON string, plain text, invalid |
| `GetHints(url)` | exact match, wildcard, no match, priority merge |

**File**: `internal/hint/hint_test.go` (NEW)

---

#### TC-3: Handler Unit Tests

**Effort**: 4-6 hours | **Current Coverage**: 0% (indirect only)

Mock the OData client interface and test each handler in `internal/bridge/bridge.go`:

1. `handleServiceInfo`
2. `handleEntityFilter`
3. `handleEntityCount`
4. `handleEntitySearch`
5. `handleEntityGet`
6. `handleEntityCreate`
7. `handleEntityUpdate`
8. `handleEntityDelete`
9. `handleFunctionCall`

**File**: `internal/bridge/bridge_handler_test.go` (NEW)

---

#### TC-4: MCP Server Unit Tests

**Effort**: 2-3 hours | **Current Coverage**: 0%

Test message routing and tool registration in `internal/mcp/server.go`.

**File**: `internal/mcp/server_test.go` (NEW)

---

#### TC-5: Transport Layer Tests

**Effort**: 4-6 hours | **Current Coverage**: 0%

| Component | File | Key Methods |
|-----------|------|-------------|
| STDIO | `stdio/stdio.go` | `Start`, `ReadMessage`, `WriteMessage` |
| HTTP/SSE | `http/sse.go` | `Start`, `handleSSE`, `handleRPC` |
| Streamable | `http/streamable.go` | `Start`, `handleMCP`, `upgradeToSSE` |

**Files**: `internal/transport/*/..._test.go` (NEW)

---

#### Milestone Checklist

- [ ] TC-1: Config package tests
- [ ] TC-2: Hint manager tests
- [ ] TC-3: Handler unit tests
- [ ] TC-4: MCP server tests
- [ ] TC-5: Transport layer tests

---

### v1.7.0 — Token-Optimized Discovery ✅ COMPLETE

**Theme**: ~90% token reduction for large SAP services

**Design Document**: [docs/plans/2025-12-17-lazy-metadata-design.md](plans/2025-12-17-lazy-metadata-design.md)

**Problem**: For large OData services (especially SAP), eager tool generation creates significant token overhead:
- 100+ entity sets × ~5 tools each = 500+ tools
- Each tool definition includes name, description, and full input schema
- Estimated 200-500 tokens per tool = **100K-250K tokens** in `tools/list` response

**Solution**: Hybrid lazy mode — instead of per-entity tools, generate 10 generic tools that accept `entity_set` as a parameter.

#### Lazy Mode Tool Set

| Tool | Parameters | Description |
|------|------------|-------------|
| `odata_service_info` | `include_metadata?` | Service overview, entity list, function list |
| `list_entities` | `entity_set`, `filter?`, `select?`, etc. | Generic query tool |
| `count_entities` | `entity_set`, `filter?` | Count entities with optional filter |
| `get_entity` | `entity_set`, `key` | Get single entity by key |
| `get_entity_schema` | `entity_set` | Return schema (fields, types, keys, nav props) |
| `create_entity` | `entity_set`, `data` | Create entity |
| `update_entity` | `entity_set`, `key`, `data` | Update entity |
| `delete_entity` | `entity_set`, `key` | Delete entity |
| `list_functions` | (none) | List available function imports |
| `call_function` | `function_name`, `params` | Call any function import |

**Total: 10 tools** regardless of service size (vs 500+ in eager mode).

#### CLI Flags

```bash
# Explicit opt-in
odata-mcp --service https://... --lazy-metadata

# Auto-enable if tool count exceeds threshold
odata-mcp --service https://... --lazy-threshold 100

# Environment variables
ODATA_LAZY_METADATA=true
ODATA_LAZY_THRESHOLD=100
```

#### Token Math

| Mode | Calculation | Total |
|------|-------------|-------|
| **Eager** | 100 entities × 5 tools × 300 tokens | ~150,000 tokens |
| **Lazy** | 10 tools × 200 tokens | ~2,000 tokens |

**Savings: ~99%**

#### Milestone Checklist

- [x] Phase 1: Add `LazyMetadata`, `LazyThreshold` to config + CLI flags
- [x] Phase 2: Create `internal/bridge/lazy_tools.go` (generic tool generation)
- [x] Phase 3: Create `internal/bridge/lazy_handlers.go` (entity_set resolution)
- [x] Phase 4: Add `shouldUseLazyMode()` to bridge.go, split tool generation paths
- [x] Phase 5: Unit tests (`lazy_tools_test.go`)
- [x] Phase 6: Integration tests (`lazy_mode_test.go`)
- [x] Phase 7: Update README, CHANGELOG

---

### v1.8.0 — Skill Generator (Planned)

**Theme**: AI-native documentation from OData metadata

> **Note**: The existing `docs/skills/` directory contains **Analysis Skills** — reusable prompts for systematic codebase review (Repo Scout, Test Auditor, etc.). Phase 3's **OData Skills** are different: they are auto-generated guides for using OData MCP tools. The skill generator output will go to a configurable directory (default: `./skills/`), NOT to `docs/skills/`.

**Goal**: Auto-generate Claude Code Skills from OData service metadata.

Skills are markdown files that serve as intelligent usage guides for MCP tools:
- **Context**: What the service/entity is for
- **Workflows**: Multi-step procedures combining multiple tools
- **Best Practices**: Recommended patterns, common filters, pagination hints
- **Domain Knowledge**: Business terminology and relationships

#### CLI Flags

```
--generate-skills      Generate Claude Code Skills from OData metadata
--skills-output        Output directory (default: ./skills)
--skills-hints         Optional hints file with domain-specific context
```

#### Output Structure

```
skills/
├── {service-id}/
│   ├── README.md           # Service overview
│   ├── entities/
│   │   ├── {EntitySet}.md  # Per-entity skills
│   │   └── ...
│   └── workflows/
│       ├── common-queries.md
│       └── {domain-workflow}.md
└── ...
```

#### Example Use Case: SAP Solution Manager BPA

SAP Business Process Analytics (BPA) provides OData services for process step monitoring, exception analysis, and volume analytics.

See [SOLMAN_HUB_ARCHITECTURE.md](SOLMAN_HUB_ARCHITECTURE.md) for the broader Solution Manager integration strategy.

#### Milestone Checklist

- [ ] Create skill generator module (`internal/skills/generator.go`)
- [ ] Write generator unit tests
- [ ] Create Go templates for skill markdown files
- [ ] Add CLI flags
- [ ] Implement hints file loading and merging
- [ ] Generate service README from metadata
- [ ] Generate per-entity skill files
- [ ] Generate common workflow templates
- [ ] Test with SAP BPA service metadata
- [ ] Update documentation (README, CHANGELOG)

---

### v1.6.5 — Reliability & DX Quick Fixes ✅ COMPLETE

**Theme**: Low-effort reliability and developer experience improvements

#### RL-2: Fix Dropped SSE Messages

**Effort**: 30 min | **Risk**: Low

Messages silently dropped when client buffer full. Add logging and metrics.

```go
// Current (internal/transport/http/sse.go)
select {
case client.events <- data:
default:
    // Client buffer full, skip
}

// Target: Log and count dropped messages
select {
case client.events <- data:
default:
    if c.verbose {
        fmt.Fprintf(os.Stderr, "[SSE] Dropped message for client %s: buffer full\n", client.id)
    }
    atomic.AddInt64(&c.droppedMessages, 1)
}
```

---

#### RL-3: Add Configurable HTTP Timeout

**Effort**: 1 hour | **Risk**: Low

Current: Hardcoded 30s timeout.

```
--http-timeout seconds       HTTP request timeout (default: 30)
--metadata-timeout seconds   Metadata fetch timeout (default: 60)
```

---

#### DX-1: Add Linting Configuration

**Effort**: 1 hour | **Risk**: Low

Add `.golangci.yml` with sensible defaults and `make lint` target.

```yaml
linters:
  enable:
    - errcheck
    - gosimple
    - govet
    - ineffassign
    - staticcheck
    - unused
    - gofmt
    - goimports
```

---

#### v1.6.5 Milestone Checklist

- [x] RL-2: Fix dropped SSE messages with logging
- [x] RL-3: Add `--http-timeout` and `--metadata-timeout` flags
- [x] DX-1: Add `.golangci.yml` and `make lint` target
- [x] Fix retry configuration not being applied to HTTP client

---

### v1.6.6 — Feature Polish (Planned)

**Theme**: User-facing feature improvements

#### FE-1: Pretty-Print Output

**Effort**: 2 hours | **Risk**: Low

TODO at `main.go:568`: Implement pretty printing like the Python version.

```
--format string    Output format: json (default), pretty, csv
```

---

#### FE-3: Additional Service Hints

**Effort**: 1-2 hours | **Risk**: Low

Add patterns for:
- Microsoft Dataverse (`*dataverse.api.dynamics.com*`)
- Dynamics 365 Business Central (`*api.businesscentral.dynamics.com*`)
- SAP SuccessFactors (`*successfactors.com/odata/*`)

---

#### PF-1: Connection Pooling Configuration

**Effort**: 1 hour | **Risk**: Low

Expose HTTP transport settings:

```
--http-max-idle-conns       Max idle connections (default: 100)
--http-max-conns-per-host   Max connections per host (default: 10)
--http-idle-timeout         Idle connection timeout (default: 90s)
```

---

#### v1.6.6 Milestone Checklist

- [ ] FE-1: Implement `--format` flag (json, pretty, csv)
- [ ] FE-3: Add Microsoft Dataverse, D365 BC, SuccessFactors hints
- [ ] PF-1: Add connection pooling CLI flags

---

### v2.0.0 — Multi-LLM Platform Support (Backlog)

**Theme**: Documentation and testing for the broader MCP ecosystem

**Context**: As of December 2025, MCP has become the industry standard for AI tool integration. The protocol was donated to the Linux Foundation's Agentic AI Foundation (AAIF), co-founded by Anthropic, OpenAI, and Block, with support from Google, Microsoft, AWS, Cloudflare, and Bloomberg.

**Key Insight**: The OData MCP Bridge already supports all required transports (stdio, HTTP/SSE, Streamable HTTP). This phase is **90% documentation** — the code already works.

#### MCP Ecosystem Compatibility Matrix

| Platform | MCP Support | Transport Required | Current Status |
|----------|-------------|-------------------|----------------|
| **Claude Desktop** | ✅ Native | stdio | ✅ Verified |
| **Cline** (VS Code) | ✅ Native | stdio | ✅ Should work |
| **Roo Code** (VS Code) | ✅ Native | stdio, SSE, Streamable HTTP | ✅ Should work |
| **Cursor** | ✅ Native | stdio, SSE, Streamable HTTP | ✅ Should work |
| **Windsurf** | ✅ Native | stdio | ✅ Should work |
| **GitHub Copilot** (VS Code) | ✅ Native (July 2025) | stdio + remote HTTP | ✅ Should work |
| **GitHub Copilot** (JetBrains/Eclipse/Xcode) | ✅ Native (Aug 2025) | stdio + remote HTTP | ✅ Should work |
| **ChatGPT** (Plus/Pro/Business) | ✅ Native (Sept 2025) | HTTP/SSE, Streamable HTTP | ⚠️ Requires remote hosting |
| **OpenAI Agents SDK** | ✅ Native | HTTP | ✅ Should work |
| **Microsoft Copilot Studio** | ✅ Native | HTTP | Untested |

#### ML-1: IDE Integration Guides (stdio-based)

**Effort**: 3-4 hours | **Risk**: Low

Documentation for local IDE-based MCP clients that use stdio transport.

**Platforms to cover**:
- Cline (VS Code) — configuration in VS Code settings
- Roo Code (VS Code) — `mcp_settings.json` or `.roo/mcp.json`
- Cursor — Cursor settings, one-click install options
- Windsurf — Similar to Cursor
- GitHub Copilot (local) — `.vscode/mcp.json`

**File**: `docs/IDE_INTEGRATION.md` (NEW)

**References**:
- [Roo Code MCP Docs](https://docs.roocode.com/features/mcp/overview)
- [Cline GitHub](https://github.com/cline/cline)
- [Cursor MCP Docs](https://docs.cursor.com/context/model-context-protocol)

---

#### ML-2: ChatGPT Integration Guide

**Effort**: 2-3 hours | **Risk**: Low

ChatGPT's MCP support requires remote HTTP endpoints — cannot connect to localhost.

**Documentation needed**:
- Remote deployment options (cloud VMs, containers)
- Tunneling setup (ngrok, Cloudflare Tunnel) for local development
- OAuth configuration (ChatGPT supports OAuth for MCP)
- Step-by-step connection walkthrough
- Troubleshooting common issues

**File**: `docs/CHATGPT_INTEGRATION.md` (NEW)

**References**:
- [OpenAI MCP Support Announcement](https://www.infoq.com/news/2025/10/chat-gpt-mcp/)
- [ChatGPT Release Notes](https://help.openai.com/en/articles/6825453-chatgpt-release-notes)

---

#### ML-3: GitHub Copilot Integration Guide

**Effort**: 2-3 hours | **Risk**: Low

GitHub Copilot supports both local (stdio) and remote MCP servers.

**Documentation needed**:
- VS Code configuration
- JetBrains/Eclipse/Xcode setup
- Organization policy requirements (MCP disabled by default for orgs)
- Remote server configuration for Copilot coding agent

**File**: `docs/GITHUB_COPILOT_INTEGRATION.md` (NEW)

**References**:
- [MCP support in VS Code GA](https://github.blog/changelog/2025-07-14-model-context-protocol-mcp-support-in-vs-code-is-generally-available/)
- [Extending Copilot with MCP](https://docs.github.com/copilot/customizing-copilot/using-model-context-protocol/extending-copilot-chat-with-mcp)

---

#### ML-4: Remote Deployment Guide

**Effort**: 3-4 hours | **Risk**: Low

For ChatGPT and remote Copilot scenarios requiring HTTP endpoints.

**Documentation needed**:
- Docker deployment (already have Dockerfile)
- Cloud deployment (AWS, GCP, Azure)
- Tunneling for local development (ngrok, Cloudflare Tunnel)
- Reverse proxy configuration (nginx, Caddy)
- TLS/HTTPS setup (required for production)
- Security considerations for remote exposure

**File**: `docs/REMOTE_DEPLOYMENT.md` (NEW)

---

#### ML-5: Authentication for Remote Transports (Optional)

**Effort**: 4-6 hours | **Risk**: Medium

Current HTTP transports have NO authentication. For production remote scenarios, this may be needed.

**Options to evaluate**:
- Bearer token authentication (`--http-auth-token`)
- OAuth 2.0 support (ChatGPT uses this)
- API key validation

**Note**: Only implement if user demand warrants it. Many scenarios work with network-level security (VPN, private networks).

**Files**: `internal/transport/http/auth.go` (NEW)

---

#### ML-6: Platform Compatibility Testing

**Effort**: 4-6 hours | **Risk**: Low

Verify actual compatibility with each platform.

**Test matrix**:
| Platform | Transport | Test Status |
|----------|-----------|-------------|
| Cline | stdio | TODO |
| Roo Code | stdio | TODO |
| Cursor | stdio | TODO |
| GitHub Copilot (VS Code) | stdio | TODO |
| ChatGPT (via ngrok) | Streamable HTTP | TODO |
| OpenAI Agents SDK | HTTP/SSE | TODO |

**Deliverable**: Verified compatibility matrix in README

---

#### v2.0.0 Milestone Checklist

- [ ] ML-1: IDE integration guide (Cline, Roo Code, Cursor, Windsurf)
- [ ] ML-2: ChatGPT integration guide
- [ ] ML-3: GitHub Copilot integration guide
- [ ] ML-4: Remote deployment guide
- [ ] ML-5: Authentication for remote transports (if needed)
- [ ] ML-6: Platform compatibility testing
- [ ] Update README with "Supported Platforms" section and badges

---

### v1.9.0 — Advanced Features (Planned)

**Theme**: Complex features requiring more architectural work

#### FE-2: Metadata Caching

**Effort**: 3-4 hours | **Risk**: Medium

Large metadata files (SAP services) can take 5-10 seconds to parse. Caching improves startup time.

```
--cache-metadata           Enable metadata caching (default: false)
--metadata-cache-ttl       Cache TTL (default: 1h)
--metadata-cache-dir       Cache directory (default: ~/.odata-mcp/cache)
```

**Cache Key**: SHA256 of (ServiceURL + credentials hash)

---

#### DX-2: Improve Logging Consistency

**Effort**: 2-3 hours | **Risk**: Low

Current: Mix of `fmt.Fprintf(os.Stderr, ...)` and `fmt.Printf(...)`.

Target: Consistent logger abstraction with ERROR, VERBOSE, TRACE levels.

```go
// internal/logger/logger.go
type Logger struct {
    verbose bool
    writer  io.Writer
}

func (l *Logger) Verbose(format string, args ...interface{}) {
    if l.verbose {
        fmt.Fprintf(l.writer, "[VERBOSE] "+format+"\n", args...)
    }
}
```

---

#### v1.9.0 Milestone Checklist

- [ ] FE-2: Implement metadata caching with TTL
- [ ] DX-2: Create logger abstraction, migrate all logging

---

## Future Exploration (Parked Ideas)

This section captures ideas that emerged during brainstorming but don't have a clear problem to solve yet. Revisit when user demand or pain points emerge.

### FX-1: Tool Registry / Service Catalog

**Concept**: Persist discovered tool definitions to disk for offline browsing and cross-session reuse.

```text
~/.odata-mcp/registry/
  ├── northwind/
  │   ├── tools.json        # Generated tool definitions
  │   ├── schemas.json      # Entity schemas
  │   └── metadata.json     # Service metadata snapshot
  └── sap-s4hana-prod/
      └── ...
```

**Potential benefits**:
- Offline discovery: Browse tools without live OData connection
- Cross-session learning: Agent remembers what worked before
- Team sharing: Export/import service configurations
- Schema versioning: Track changes over time

**Open questions**:
- Is this solving agent efficiency, team collaboration, or offline capability?
- How does cache invalidation work when backend schema changes?
- Does v1.8.0 Skill Generator already address this via generated markdown?

---

### FX-2: Semantic Tool Enrichment

**Concept**: Enhance tool descriptions with business context beyond technical OData names.

**Current**: "List/filter Products entities with OData query options"

**Enhanced**: "Query product catalog including pricing, inventory levels, and supplier relationships. Common filters: CategoryID, Discontinued, UnitsInStock."

**Sources for enrichment**:
- SAP annotations in metadata (`sap:label`, `sap:quickinfo`)
- User-provided hints (existing hint system)
- AI-generated descriptions from schema analysis

**Note**: Could be integrated into v1.8.0 Skill Generator rather than a separate feature.

---

### FX-3: Usage Analytics / Query Patterns

**Concept**: Track what queries agents actually run to surface patterns and suggestions.

**Potential insights**:
- Which entities are accessed most?
- What filters are commonly used?
- Which queries fail and why?
- "Users often filter Products by CategoryID"
- "This entity has 1M+ rows, always use $top"

**Complexity**: High — requires persistent storage, privacy considerations.

---

### FX-4: Multi-Service Federation

**Concept**: Unified discovery and querying across multiple OData services.

**Current**: Run separate bridge instances per service.

**Future possibility**:
- Single `tools/list` spanning multiple services
- Cross-service joins: "Get orders from SAP, match with CRM contacts"
- Smart routing: Agent says "find customer", bridge knows which service has Customers

**Complexity**: High — requires service registry, conflict resolution for entity names.

---

### FX-5: Recipe / Workflow Library

**Concept**: Predefined multi-step patterns for common tasks.

```yaml
# recipes/get-order-details.yaml
name: Get Full Order Details
steps:
  1. get_Orders { key: order_id }
  2. list_Order_Details { filter: "OrderID eq {order_id}" }
  3. For each detail: get_Products { key: product_id }
```

**Note**: v1.8.0 Skill Generator is the seed of this — generates persistent documentation/workflows from metadata.

---

### FX-6: Header Forwarding / API Key Auth

**Concept**: Pass custom headers (X-API-Key, X-Tenant) from MCP client to OData backend.

**Use case**: OData services that use header-based authentication instead of basic/cookie auth.

**Implementation sketch**:
- `--header "X-API-Key=@ENV_VAR"` for static headers
- `--forward-header "X-Tenant"` for pass-through from MCP HTTP requests

**From consolidated review** — documented in detail there. Consider for v1.7.x or v1.8.x if user demand emerges.

---

## Completed

Historical record of completed work.

### v1.6.3 (December 2025)

- ✅ Composite key deterministic ordering (sort keys alphabetically)
- ✅ SSE stream double-close panic fix (sync.Once guard)
- ✅ Concurrent ResponseWriter write protection (mutex)
- ✅ Context propagation in MCP tool handlers
- ✅ MCP mock server for protocol tests (httptest.Server)

### v1.6.2 (December 2025)

- ✅ Fix race condition in ODataClient with mutex guards for concurrent access
- ✅ Handle multiple EDMX schemas (parser now processes all `<Schema>` blocks)
- ✅ Relax SSE Accept header checks (allow combined headers)
- ✅ Propagate metadata parse failures with meaningful error messages

### v1.6.1 (December 2025)

- ✅ CQ-1: Replace deprecated `io/ioutil` with `io`
- ✅ CQ-2: Handle JSON marshal error in trace.go
- ✅ CQ-3: Body read error handling (already fixed)
- ✅ CQ-4: Align constant defaults with CLI

### v1.6.0 (December 2025)

- ✅ Credential masking in verbose output
- ✅ Exponential backoff retry with jitter
- ✅ Streamable HTTP transport (`--transport streamable-http`)

---

## Risk Assessment

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| Lazy loading breaks existing workflows | Medium | High | Make it opt-in, default to full mode |
| Retry storms on service outages | Low | Medium | Cap max retries, jitter prevents thundering herd |
| Cache invalidation issues | Medium | Low | Short default TTL, manual invalidation option |
| Generated skills lack domain context | Medium | Medium | Hints file system for domain enrichment |

---

## Success Metrics

| Metric | Target | Measurement |
|--------|--------|-------------|
| Token reduction (lazy mode) | >80% | Compare tool list response size |
| Retry success rate | >95% | Integration tests with flaky mock |
| Credential exposure | 0 instances | Audit verbose output in tests |
| Test coverage | >70% | `go test -cover` for core packages |
| Skill generation coverage | 100% entities | All entity sets get skill files |

---

## Appendix: File Index

Key files and their scheduled improvements:

| File | Issues | Target |
|------|--------|--------|
| `cmd/odata-mcp/main.go` | Pretty print (FE-1) | v1.6.5 |
| `internal/bridge/bridge.go` | Handler test coverage (TC-3) | v1.6.4 |
| `internal/client/client.go` | ~~Race condition~~ ✅, ~~Composite key~~ ✅, HTTP timeout (RL-3) | v1.6.2 ✅, v1.6.3 ✅, v1.6.5 |
| `internal/mcp/server.go` | ~~Deprecated import (CQ-1)~~ ✅ | v1.6.1 |
| `internal/config/config.go` | Test coverage (TC-1) | v1.6.4 |
| `internal/hint/hint.go` | Test coverage (TC-2), new hints (FE-3) | v1.6.4, v1.6.6 |
| `internal/debug/trace.go` | ~~Swallowed error (CQ-2)~~ ✅ | v1.6.1 |
| `internal/metadata/parser.go` | ~~Multiple schemas~~ ✅ | v1.6.2 |
| `internal/transport/http/sse.go` | ~~Accept header check~~ ✅, Dropped messages (RL-2), tests (TC-5) | v1.6.2 ✅, v1.6.5, v1.6.4 |
| `internal/transport/http/streamable.go` | ~~Accept header check~~ ✅, ~~Double-close~~ ✅, ~~Concurrent writes~~ ✅ | v1.6.2 ✅, v1.6.3 ✅ |
| `internal/constants/constants.go` | ~~Inconsistent defaults (CQ-4)~~ ✅ | v1.6.1 |

---

*End of document*
