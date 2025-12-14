# OData MCP Bridge — Roadmap & Backlog

*Last Updated: December 2025*

This document is the single source of truth for planned improvements, active development phases, and future backlog items.

---

## Quick Reference

| Version | Status | Theme |
|---------|--------|-------|
| v1.6.0 | ✅ Complete | Foundation (Credential Masking, Retry) |
| v1.6.1 | ✅ Complete | Quick Wins (Code Quality) |
| v1.6.2 | Pending | Test Coverage |
| v1.7.0 | Planned | Token-Optimized Discovery |
| v1.8.0 | Planned | Skill Generator |

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

### v1.6.2 — Test Coverage (Pending)

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

### v1.7.0 — Token-Optimized Discovery (Planned)

**Theme**: ~90% token reduction for large SAP services

**Goal**: Reduce token consumption through lazy metadata loading.

Current behavior loads full metadata (potentially 100+ entity sets with all properties) upfront. For large SAP services, this consumes significant tokens before the AI even starts working.

#### New Tools

| Tool | Purpose |
|------|---------|
| `list_entities` | Return entity set names only (no schema) |
| `get_entity_schema` | Load schema for specific entity on-demand |

#### CLI Flags

```
--lazy-metadata        Enable lazy metadata loading (default: false)
--metadata-cache-ttl   Cache TTL for loaded schemas (default: 5m)
```

#### Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                     LAZY LOADING FLOW                           │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  AI Request          Bridge                    OData Service    │
│                                                                 │
│  list_entities ──────► Return entity names ◄── Quick regex     │
│                        from $metadata           parse           │
│                                                                 │
│  get_entity_schema ──► Return full schema  ◄── Parse specific  │
│  "Products"            for Products            EntityType       │
│                                                                 │
│  query_Products ─────► Execute query       ◄── Normal OData    │
│                        (schema cached)         request          │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

#### Milestone Checklist

- [ ] Implement metadata cache (`internal/cache/metadata_cache.go`)
- [ ] Write cache unit tests
- [ ] Implement lazy loader (`internal/bridge/lazy_loader.go`)
- [ ] Write lazy loader unit tests
- [ ] Add `GetEntitySetNames()` to client
- [ ] Modify bridge to support lazy mode
- [ ] Add lazy mode tools (list_entities, get_entity_schema)
- [ ] Add CLI flags for lazy loading
- [ ] Write integration tests with Northwind
- [ ] Update documentation (README, CHANGELOG)
- [ ] Performance benchmarks

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

## Backlog

Items not yet scheduled for a specific version.

### Reliability

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
```

---

#### RL-3: Add Configurable HTTP Timeout

**Effort**: 1 hour | **Risk**: Low

Current: Hardcoded 30s timeout.

```
--http-timeout duration    HTTP request timeout (default: 30s)
--metadata-timeout duration  Metadata fetch timeout (default: 10s)
```

---

### Features

#### FE-1: Pretty-Print Output

**Effort**: 2 hours | **Risk**: Low

TODO at `main.go:568`: Implement pretty printing like the Python version.

```
--format string    Output format: json (default), pretty, csv
```

---

#### FE-2: Metadata Caching

**Effort**: 3-4 hours | **Risk**: Medium

Large metadata files (SAP services) can take 5-10 seconds to parse. Caching improves startup time.

```
--cache-metadata           Enable metadata caching (default: false)
--metadata-cache-ttl       Cache TTL (default: 1h)
--metadata-cache-dir       Cache directory (default: ~/.odata-mcp/cache)
```

---

#### FE-3: Additional Service Hints

**Effort**: 1-2 hours | **Risk**: Low

Add patterns for:
- Microsoft Dataverse (`*dataverse.api.dynamics.com*`)
- Dynamics 365 Business Central (`*api.businesscentral.dynamics.com*`)
- SAP SuccessFactors (`*successfactors.com/odata/*`)

---

### Performance

#### PF-1: Connection Pooling Configuration

**Effort**: 1 hour | **Risk**: Low

Expose HTTP transport settings:

```
--http-max-idle-conns       Max idle connections (default: 100)
--http-max-conns-per-host   Max connections per host (default: 10)
--http-idle-timeout         Idle connection timeout (default: 90s)
```

---

### Developer Experience

#### DX-1: Add Linting Configuration

**Effort**: 1 hour | **Risk**: Low

Add `.golangci.yml` with sensible defaults and `make lint` target.

---

#### DX-2: Improve Logging Consistency

**Effort**: 2-3 hours | **Risk**: Low

Current: Mix of `fmt.Fprintf(os.Stderr, ...)` and `fmt.Printf(...)`.

Target: Consistent logger abstraction with ERROR, VERBOSE, TRACE levels.

---

## Completed

Historical record of completed work.

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

Key files and their improvement opportunities:

| File | Issues |
|------|--------|
| `cmd/odata-mcp/main.go` | TODO (pretty print) |
| `internal/bridge/bridge.go` | 0% handler test coverage |
| `internal/client/client.go` | — |
| `internal/mcp/server.go` | Deprecated import (CQ-1) |
| `internal/config/config.go` | 0% test coverage (TC-1) |
| `internal/hint/hint.go` | 0% test coverage (TC-2) |
| `internal/debug/trace.go` | Swallowed error (CQ-2) |
| `internal/transport/http/sse.go` | Dropped messages (RL-2), 0% coverage (TC-5) |
| `internal/constants/constants.go` | Inconsistent defaults (CQ-4) |

---

*End of document*
