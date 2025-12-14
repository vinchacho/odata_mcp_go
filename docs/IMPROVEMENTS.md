# OData MCP Bridge (Go) — Improvement Opportunities

**Document Version**: 1.0
**Created**: 2024-12-14
**Status**: Backlog — Ready for implementation

---

## Executive Summary

This document catalogs improvement opportunities for the odata_mcp_go repository, identified through systematic codebase analysis. Each opportunity includes detailed implementation notes, acceptance criteria, and validation steps.

**Repository Purpose**: A Go binary bridging OData v2/v4 services to MCP, dynamically generating tools from `$metadata`.

**Key Metrics**:
- 14 test files, ~31 test functions
- 91% integration tests, 9% unit tests
- 6 packages with 0% direct test coverage
- 2 TODO comments, 3 swallowed errors, 1 deprecated import

---

## Table of Contents

1. [Code Quality Quick Wins](#1-code-quality-quick-wins)
2. [Test Coverage Gaps](#2-test-coverage-gaps)
3. [Reliability Improvements](#3-reliability-improvements)
4. [Feature Enhancements](#4-feature-enhancements)
5. [Performance Optimizations](#5-performance-optimizations)
6. [Developer Experience](#6-developer-experience)

---

## 1. Code Quality Quick Wins

### 1.1 Replace Deprecated io/ioutil

**ID**: CQ-1
**Effort**: 5 minutes
**Risk**: Low

**Current State**:
```go
// internal/mcp/server.go:7
import "io/ioutil"

// internal/mcp/server.go:52
log.SetOutput(ioutil.Discard)
```

**Target State**:
```go
import "io"

log.SetOutput(io.Discard)
```

**Rationale**: `io/ioutil` deprecated in Go 1.16. The codebase uses Go 1.21+ (per go.mod).

**Acceptance Criteria**:
- [ ] No imports of `io/ioutil` in codebase
- [ ] `go build` succeeds
- [ ] `go vet` passes

**Validation**:
```bash
grep -r "io/ioutil" internal/
go build ./...
go vet ./...
```

---

### 1.2 Fix Swallowed JSON Marshal Error

**ID**: CQ-2
**Effort**: 10 minutes
**Risk**: Low

**Current State**:
```go
// internal/debug/trace.go:70
jsonData, _ := json.Marshal(entry)
```

**Issue**: Error from `json.Marshal()` silently ignored. If entry contains unmarshalable types (channels, functions), tracing fails silently.

**Target State**:
```go
jsonData, err := json.Marshal(entry)
if err != nil {
    // Log error to stderr (trace is already debug output)
    fmt.Fprintf(os.Stderr, "[TRACE ERROR] Failed to marshal entry: %v\n", err)
    return
}
```

**Acceptance Criteria**:
- [ ] Error is handled (logged or returned)
- [ ] Tracing doesn't panic on bad input
- [ ] Existing trace functionality unchanged

**Validation**:
```bash
go test ./internal/debug/...
# Manual: trigger trace with problematic data
```

---

### 1.3 Fix Swallowed Body Read Error

**ID**: CQ-3
**Effort**: 10 minutes
**Risk**: Low

**Current State**:
```go
// internal/client/client.go:166-167
body, _ := io.ReadAll(resp.Body)
resp.Body.Close()
```

**Issue**: In CSRF retry path, body read error ignored. Could mask network issues during CSRF detection.

**Target State**:
```go
body, err := io.ReadAll(resp.Body)
resp.Body.Close()
if err != nil {
    return nil, fmt.Errorf("failed to read CSRF error response: %w", err)
}
```

**Acceptance Criteria**:
- [ ] Error is propagated or logged
- [ ] CSRF retry logic still functions
- [ ] No regression in existing CSRF tests

**Validation**:
```bash
go test ./internal/test/csrf_*.go -v
```

---

### 1.4 Fix Constant Inconsistencies

**ID**: CQ-4
**Effort**: 15 minutes
**Risk**: Low

**Current State**:
| Constant | constants.go | main.go (CLI default) |
|----------|-------------|----------------------|
| MaxResponseSize | 10MB | 5MB |
| MaxItems | 1000 | 100 |

**Issue**: Documentation and constants disagree with actual CLI defaults.

**Target State**: Align `constants.go` with CLI defaults (5MB, 100) OR document why they differ.

**Acceptance Criteria**:
- [ ] `DefaultMaxResponseSize` = 5MB
- [ ] `DefaultMaxItems` = 100
- [ ] OR: Add comments explaining the difference

**Validation**:
```bash
grep -n "MaxResponseSize\|MaxItems" internal/constants/constants.go cmd/odata-mcp/main.go
```

---

## 2. Test Coverage Gaps

### 2.1 Config Package Tests

**ID**: TC-1
**Effort**: 1-2 hours
**Risk**: Low
**Priority**: High

**Current Coverage**: 0%

**Functions to Test**:
| Function | Purpose | Test Cases |
|----------|---------|------------|
| `HasBasicAuth()` | Check if username+password set | empty, partial, complete |
| `HasCookieAuth()` | Check if cookies set | empty, populated |
| `UsePostfix()` | Determine naming mode | `NoPostfix` true/false |
| `IsReadOnly()` | Check read-only state | none, `--read-only`, `--read-only-but-functions` |
| `AllowModifyingFunctions()` | Check function permission | read-only vs robf |
| `IsOperationEnabled(rune)` | Check operation flags | enable/disable combos, R expansion |

**Test File**: `internal/config/config_test.go` (NEW)

**Test Cases for `IsOperationEnabled`**:
```go
// Table-driven tests
tests := []struct {
    name       string
    enableOps  string
    disableOps string
    op         rune
    want       bool
}{
    {"default allows all", "", "", 'C', true},
    {"enable R expands to SFG", "R", "", 'S', true},
    {"enable R expands to SFG", "R", "", 'F', true},
    {"enable R expands to SFG", "R", "", 'G', true},
    {"enable R does not include C", "R", "", 'C', false},
    {"disable CUD", "", "CUD", 'C', false},
    {"disable CUD allows G", "", "CUD", 'G', true},
    {"case insensitive", "r", "", 'S', true},
}
```

**Acceptance Criteria**:
- [ ] 100% coverage of config helper methods
- [ ] Table-driven tests for `IsOperationEnabled`
- [ ] Edge cases: empty strings, mixed case, invalid chars

**Validation**:
```bash
go test -v -cover ./internal/config/...
```

---

### 2.2 Hint Manager Tests

**ID**: TC-2
**Effort**: 1-2 hours
**Risk**: Low

**Current Coverage**: 0%

**Functions to Test**:
| Function | Purpose | Test Cases |
|----------|---------|------------|
| `NewManager()` | Constructor | Returns valid manager |
| `LoadFromFile(path)` | Load hints JSON | valid file, missing file, invalid JSON |
| `SetCLIHint(hint)` | Parse CLI hint | JSON string, plain text, invalid |
| `GetHints(url)` | Match URL to hints | exact match, wildcard, no match, priority merge |

**Test File**: `internal/hint/hint_test.go` (NEW)

**Pattern Matching Test Cases**:
```go
tests := []struct {
    pattern string
    url     string
    want    bool
}{
    {"*/sap/opu/odata/*", "https://example.com/sap/opu/odata/sap/SVC/", true},
    {"*Northwind*", "https://services.odata.org/V2/Northwind/Northwind.svc/", true},
    {"*Northwind*", "https://example.com/other/", false},
    {"exact/match", "exact/match", true},
    {"exact/match", "exact/match/extra", false},
}
```

**Acceptance Criteria**:
- [ ] All public methods tested
- [ ] Wildcard pattern matching verified
- [ ] Priority-based hint merging tested
- [ ] Error paths tested (missing file, bad JSON)

**Validation**:
```bash
go test -v -cover ./internal/hint/...
```

---

### 2.3 Handler Unit Tests

**ID**: TC-3
**Effort**: 4-6 hours
**Risk**: Low
**Priority**: High

**Current Coverage**: 0% (indirect via integration tests only)

**Handlers to Test** (in `internal/bridge/bridge.go`):
1. `handleServiceInfo` (line 973)
2. `handleEntityFilter` (line 1116)
3. `handleEntityCount` (line 1370)
4. `handleEntitySearch` (line 1407)
5. `handleEntityGet` (line 1454)
6. `handleEntityCreate` (line 1496)
7. `handleEntityUpdate` (line 1533)
8. `handleEntityDelete` (line 1597)
9. `handleFunctionCall` (line 1618)

**Approach**: Mock the OData client interface

```go
// Mock interface
type MockODataClient interface {
    GetEntitySet(ctx, entitySet, options) (*models.ODataResponse, error)
    GetEntity(ctx, entitySet, key, options) (*models.ODataResponse, error)
    CreateEntity(ctx, entitySet, data) (*models.ODataResponse, error)
    UpdateEntity(ctx, entitySet, key, data, method) (*models.ODataResponse, error)
    DeleteEntity(ctx, entitySet, key) (*models.ODataResponse, error)
    CallFunction(ctx, name, params, method) (*models.ODataResponse, error)
}
```

**Test Cases per Handler**:
| Handler | Happy Path | Error Path | Edge Cases |
|---------|-----------|------------|------------|
| handleEntityFilter | Return entities | OData error | Empty result, truncation |
| handleEntityGet | Return entity | 404 Not Found | Missing key params |
| handleEntityCreate | Return created | 400 Validation | Empty body |
| handleEntityUpdate | Return updated | 409 Conflict | PUT vs PATCH vs MERGE |
| handleEntityDelete | Return success | 404 Not Found | — |
| handleFunctionCall | Return result | 501 Not Implemented | GET vs POST |

**Acceptance Criteria**:
- [ ] All 9 handlers have unit tests
- [ ] Happy path + at least 1 error path per handler
- [ ] Mocked OData client (no network calls)
- [ ] Coverage >80% for bridge.go

**Validation**:
```bash
go test -v -cover ./internal/bridge/...
```

---

### 2.4 MCP Server Unit Tests

**ID**: TC-4
**Effort**: 2-3 hours
**Risk**: Low

**Current Coverage**: 0% (indirect only)

**Functions to Test** (in `internal/mcp/server.go`):
| Function | Purpose | Test Cases |
|----------|---------|------------|
| `NewServer(name, version)` | Constructor | Returns valid server |
| `SetProtocolVersion(v)` | Set protocol | Valid version, empty |
| `AddTool(tool, handler)` | Register tool | New tool, duplicate |
| `RemoveTool(name)` | Unregister | Existing, non-existing |
| `GetTools()` | List tools | Empty, populated, order preserved |
| `HandleMessage(ctx, msg)` | Route messages | initialize, tools/list, tools/call, unknown |

**Test File**: `internal/mcp/server_test.go` (NEW)

**Message Handling Test Cases**:
```go
tests := []struct {
    name    string
    method  string
    wantErr bool
    errCode int
}{
    {"initialize succeeds", "initialize", false, 0},
    {"tools/list succeeds", "tools/list", false, 0},
    {"tools/call with valid tool", "tools/call", false, 0},
    {"tools/call with unknown tool", "tools/call", true, -32602},
    {"unknown method", "foo/bar", true, -32601},
    {"invalid jsonrpc version", "", true, -32600},
}
```

**Acceptance Criteria**:
- [ ] All public methods tested
- [ ] Message routing verified
- [ ] Error codes match MCP spec
- [ ] Protocol version propagation verified

**Validation**:
```bash
go test -v -cover ./internal/mcp/...
```

---

### 2.5 Transport Layer Tests

**ID**: TC-5
**Effort**: 4-6 hours
**Risk**: Medium

**Current Coverage**: 0%

**Components**:
| Component | File | Key Methods |
|-----------|------|-------------|
| STDIO | `stdio/stdio.go` | `Start`, `ReadMessage`, `WriteMessage`, `Close` |
| HTTP/SSE | `http/sse.go` | `Start`, `handleSSE`, `handleRPC`, `BroadcastMessage` |
| Streamable | `http/streamable.go` | `Start`, `handleMCP`, `upgradeToSSE`, `Close` |

**Approach**: Use `httptest` for HTTP transports, mock stdin/stdout for stdio.

**Acceptance Criteria**:
- [ ] STDIO reads/writes JSON-RPC correctly
- [ ] HTTP/SSE endpoints return correct status codes
- [ ] Streamable HTTP upgrades to SSE when requested
- [ ] Graceful shutdown works

**Validation**:
```bash
go test -v -cover ./internal/transport/...
```

---

## 3. Reliability Improvements

### 3.1 Add Retry with Exponential Backoff ✅ DONE (v1.6.0)

**ID**: RL-1
**Effort**: 2-3 hours
**Risk**: Medium
**Priority**: High
**Status**: ✅ **IMPLEMENTED** in v1.6.0 — See [IMPLEMENTATION_PLAN.md](IMPLEMENTATION_PLAN.md) Phase 1

**Implementation Summary**:

- `internal/client/retry.go` - RetryConfig with exponential backoff + jitter
- `internal/client/retry_test.go` - Unit tests
- `internal/client/retry_integration_test.go` - Integration tests with mock server
- CLI flags: `--retry-max-attempts`, `--retry-initial-backoff-ms`, `--retry-max-backoff-ms`, `--retry-backoff-multiplier`
- Env vars: `ODATA_RETRY_*`
- Retryable: 429, 500, 502, 503, 504
- CSRF refresh integrated without counting toward retry limit

**Original Spec** (preserved for reference):

**Current State**: Only CSRF 403 retried (once). No retry on transient errors.

**Target State**: Retry on:
- 408 Request Timeout
- 429 Too Many Requests
- 503 Service Unavailable
- 504 Gateway Timeout
- Connection errors (dial timeout, connection reset)

**Implementation**:
```go
// New config fields
type Config struct {
    // ...existing...
    RetryMaxAttempts   int           // default: 3
    RetryInitialDelay  time.Duration // default: 100ms
    RetryMaxDelay      time.Duration // default: 5s
    RetryMultiplier    float64       // default: 2.0
}

// New function in client.go
func (c *ODataClient) doRequestWithBackoff(req *http.Request, bodyBytes []byte) (*http.Response, error) {
    var lastErr error
    delay := c.config.RetryInitialDelay

    for attempt := 0; attempt <= c.config.RetryMaxAttempts; attempt++ {
        resp, err := c.doRequestInternal(req, bodyBytes)
        if err == nil && !isRetryableStatus(resp.StatusCode) {
            return resp, nil
        }

        lastErr = err
        if resp != nil {
            lastErr = fmt.Errorf("HTTP %d", resp.StatusCode)
        }

        // Only retry GET or explicitly idempotent methods
        if req.Method != "GET" && !c.isIdempotent(req) {
            break
        }

        // Add jitter: ±10%
        jitter := delay / 10
        sleepTime := delay + time.Duration(rand.Int63n(int64(jitter*2))) - jitter
        time.Sleep(sleepTime)

        delay = time.Duration(float64(delay) * c.config.RetryMultiplier)
        if delay > c.config.RetryMaxDelay {
            delay = c.config.RetryMaxDelay
        }
    }

    return nil, fmt.Errorf("max retries exceeded: %w", lastErr)
}
```

**CLI Flags**:
```
--retry-max-attempts     Maximum retry attempts (default: 3)
--retry-initial-delay    Initial delay between retries (default: 100ms)
--retry-max-delay        Maximum delay between retries (default: 5s)
--no-retry               Disable retry logic
```

**Acceptance Criteria**:
- [ ] Retries on 429, 503, 504 status codes
- [ ] Exponential backoff with jitter
- [ ] Does NOT retry mutations (POST/PUT/DELETE) unless idempotent
- [ ] Respects `Retry-After` header if present
- [ ] Configurable via CLI flags
- [ ] Logged in verbose mode

**Test Cases**:
```go
tests := []struct {
    name           string
    statusCodes    []int  // sequence of responses
    method         string
    expectRetries  int
    expectSuccess  bool
}{
    {"success first try", []int{200}, "GET", 0, true},
    {"503 then success", []int{503, 200}, "GET", 1, true},
    {"429 then 503 then success", []int{429, 503, 200}, "GET", 2, true},
    {"max retries exceeded", []int{503, 503, 503, 503}, "GET", 3, false},
    {"no retry on POST", []int{503, 200}, "POST", 0, false},
    {"no retry on 400", []int{400}, "GET", 0, false},
}
```

**Validation**:
```bash
go test -v ./internal/client/... -run TestRetry
# Integration: test against httpbin.org/status/503
```

---

### 3.2 Fix Dropped SSE Messages

**ID**: RL-2
**Effort**: 30 minutes
**Risk**: Low

**Current State**:
```go
// internal/transport/http/sse.go:211-216
select {
case client.events <- data:
default:
    // Client buffer full, skip
}
```

**Issue**: Messages silently dropped when client buffer full. No visibility into data loss.

**Target State**:
```go
select {
case client.events <- data:
default:
    // Client buffer full - log and track
    if c.verbose {
        fmt.Fprintf(os.Stderr, "[SSE] Dropped message for client %s: buffer full\n", client.id)
    }
    atomic.AddInt64(&c.droppedMessages, 1)
}
```

**Acceptance Criteria**:
- [ ] Dropped messages logged in verbose mode
- [ ] Counter available for metrics
- [ ] No behavior change in normal operation

---

### 3.3 Add Configurable HTTP Timeout

**ID**: RL-3
**Effort**: 1 hour
**Risk**: Low

**Current State**: Hardcoded 30s timeout in `client.go:50-52`

**Target State**:
```go
// CLI flag
--http-timeout duration    HTTP request timeout (default: 30s)
--metadata-timeout duration  Metadata fetch timeout (default: 10s)

// Config struct
HTTPTimeout      time.Duration
MetadataTimeout  time.Duration

// Client initialization
httpClient: &http.Client{
    Timeout: cfg.HTTPTimeout,
},
```

**Acceptance Criteria**:
- [ ] `--http-timeout` flag works
- [ ] Default unchanged (30s)
- [ ] Metadata can have separate timeout
- [ ] Timeout error messages are clear

**Validation**:
```bash
# Test with very short timeout
./odata-mcp --http-timeout 1ms --service https://example.com/odata/
# Should fail fast with timeout error
```

---

## 4. Feature Enhancements

### 4.1 Pretty-Print Output

**ID**: FE-1
**Effort**: 2 hours
**Risk**: Low

**Current State**: TODO at `main.go:568`:
```go
// TODO: Implement pretty printing like the Python version
```

**Target State**:
```
--format string    Output format: json (default), pretty, csv
```

**Implementation**:
```go
func printTraceInfo(bridge *bridge.ODataMCPBridge, format string) error {
    info, err := bridge.GetTraceInfo()
    if err != nil {
        return err
    }

    switch format {
    case "pretty":
        return printTracePretty(info)
    case "csv":
        return printTraceCSV(info)
    default:
        return printTraceJSON(info)
    }
}

func printTracePretty(info *models.TraceInfo) error {
    fmt.Println("╔════════════════════════════════════════╗")
    fmt.Println("║     OData MCP Bridge - Trace Info      ║")
    fmt.Println("╠════════════════════════════════════════╣")
    fmt.Printf("║ Service: %-30s ║\n", truncate(info.ServiceURL, 30))
    fmt.Printf("║ Tools:   %-30d ║\n", info.TotalTools)
    // ... etc
}
```

**Acceptance Criteria**:
- [ ] `--format pretty` produces human-readable output
- [ ] `--format json` unchanged (default)
- [ ] `--format csv` produces valid CSV
- [ ] Works with `--trace` mode

---

### 4.2 Metadata Caching

**ID**: FE-2
**Effort**: 3-4 hours
**Risk**: Medium

**Spec Reference**: Section 13 lists "Caching" as out of scope.

**Rationale**: Large metadata files (SAP services) can take 5-10 seconds to parse. Caching improves startup time for repeated invocations.

**Target State**:
```
--cache-metadata           Enable metadata caching (default: false)
--metadata-cache-ttl       Cache TTL (default: 1h)
--metadata-cache-dir       Cache directory (default: ~/.odata-mcp/cache)
```

**Implementation**:
```go
func (c *ODataClient) GetMetadata(ctx context.Context) (*models.ODataMetadata, error) {
    if c.config.CacheMetadata {
        if cached := c.loadCachedMetadata(); cached != nil {
            if c.verbose {
                fmt.Fprintf(os.Stderr, "[VERBOSE] Using cached metadata\n")
            }
            return cached, nil
        }
    }

    // Fetch from service
    metadata, err := c.fetchMetadata(ctx)
    if err != nil {
        return nil, err
    }

    if c.config.CacheMetadata {
        c.saveCachedMetadata(metadata)
    }

    return metadata, nil
}
```

**Cache Key**: SHA256 of (ServiceURL + credentials hash)

**Acceptance Criteria**:
- [ ] Cache hit skips network fetch
- [ ] Cache miss fetches and stores
- [ ] TTL expiration works
- [ ] `--no-cache` forces fresh fetch
- [ ] Cache invalidated on credential change

**Risks**:
- Stale metadata if service changes
- Mitigation: Short default TTL (1h), `--no-cache` flag

---

### 4.3 Additional Service Hints

**ID**: FE-3
**Effort**: 1-2 hours
**Risk**: Low

**Current Hints**: 3 patterns (SAP generic, SAP PO Tracking, Northwind)

**Proposed Additions**:

```json
{
  "pattern": "*dataverse.api.dynamics.com*",
  "priority": 10,
  "service_type": "Microsoft Dataverse",
  "known_issues": [
    "Requires Azure AD authentication",
    "Some entities require special permissions",
    "DateTime fields use UTC"
  ],
  "notes": [
    "Use $expand for related entities",
    "Check entity permissions in Power Platform admin"
  ]
},
{
  "pattern": "*api.businesscentral.dynamics.com*",
  "priority": 10,
  "service_type": "Microsoft Dynamics 365 Business Central",
  "known_issues": [
    "API versioning required in URL",
    "Company-scoped endpoints"
  ]
},
{
  "pattern": "*successfactors.com/odata/*",
  "priority": 10,
  "service_type": "SAP SuccessFactors",
  "known_issues": [
    "OAuth 2.0 SAML bearer required",
    "Some fields require specific permissions"
  ]
}
```

**Acceptance Criteria**:
- [ ] New patterns added to `hints.json`
- [ ] Patterns match correctly (tested)
- [ ] Notes are accurate and helpful

---

## 5. Performance Optimizations

### 5.1 Connection Pooling Configuration

**ID**: PF-1
**Effort**: 1 hour
**Risk**: Low

**Current State**: Default `http.Client` settings

**Target State**:
```
--http-max-idle-conns       Max idle connections (default: 100)
--http-max-conns-per-host   Max connections per host (default: 10)
--http-idle-timeout         Idle connection timeout (default: 90s)
```

**Implementation**:
```go
transport := &http.Transport{
    MaxIdleConns:        cfg.MaxIdleConns,
    MaxConnsPerHost:     cfg.MaxConnsPerHost,
    IdleConnTimeout:     cfg.IdleTimeout,
    DisableCompression:  false,
    DisableKeepAlives:   false,
}

httpClient := &http.Client{
    Transport: transport,
    Timeout:   cfg.HTTPTimeout,
}
```

**Acceptance Criteria**:
- [ ] Flags exposed in CLI
- [ ] Connection reuse verified (verbose logging)
- [ ] No regression in existing behavior

---

## 6. Developer Experience

### 6.1 Add Linting Configuration

**ID**: DX-1
**Effort**: 1 hour
**Risk**: Low

**Current State**: No linting configured

**Target State**: `.golangci.yml` with sensible defaults

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

linters-settings:
  errcheck:
    check-type-assertions: true
  govet:
    check-shadowing: true

issues:
  exclude-rules:
    - path: _test\.go
      linters:
        - errcheck
```

**Makefile Addition**:
```makefile
.PHONY: lint
lint:
	golangci-lint run ./...
```

**Acceptance Criteria**:
- [ ] `make lint` works
- [ ] No lint errors in clean codebase
- [ ] CI runs linting (if CI exists)

---

### 6.2 Improve Logging Consistency

**ID**: DX-2
**Effort**: 2-3 hours
**Risk**: Low

**Current State**: Mix of:
- `fmt.Fprintf(os.Stderr, "[VERBOSE] ...")`
- `fmt.Printf("[VERBOSE] ...")` (wrong: goes to stdout)
- `log.SetOutput(ioutil.Discard)` (suppresses all logging)

**Target State**: Consistent logger abstraction

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

func (l *Logger) Error(format string, args ...interface{}) {
    fmt.Fprintf(l.writer, "[ERROR] "+format+"\n", args...)
}

func (l *Logger) Trace(format string, args ...interface{}) {
    if l.traceEnabled {
        fmt.Fprintf(l.writer, "[TRACE] "+format+"\n", args...)
    }
}
```

**Acceptance Criteria**:
- [ ] All logging uses consistent interface
- [ ] No output to stdout (reserved for MCP)
- [ ] Verbose mode toggles correctly
- [ ] Log levels: ERROR, VERBOSE, TRACE

---

## Appendix: File Index

| File | Lines | Issues |
|------|-------|--------|
| `cmd/odata-mcp/main.go` | 591 | TODO (pretty print) |
| `internal/bridge/bridge.go` | 1651 | 0% handler test coverage |
| `internal/client/client.go` | 853 | TODO (optimize), swallowed error |
| `internal/mcp/server.go` | 452 | Deprecated import |
| `internal/config/config.go` | 130 | 0% test coverage |
| `internal/hint/hint.go` | ~150 | 0% test coverage |
| `internal/debug/trace.go` | ~80 | Swallowed error |
| `internal/transport/http/sse.go` | ~280 | Dropped messages, 0% coverage |
| `internal/transport/http/streamable.go` | ~370 | 0% coverage |
| `internal/transport/stdio/stdio.go` | ~100 | 0% coverage |
| `internal/constants/constants.go` | 246 | Inconsistent defaults |

---

## Implementation Priority Matrix

| Priority | ID | Title | Effort | Impact |
|----------|-----|-------|--------|--------|
| P0 | CQ-1 | Replace io/ioutil | 5m | Low |
| P0 | CQ-2 | Fix trace.go error | 10m | Med |
| P0 | CQ-3 | Fix client.go error | 10m | Med |
| P0 | CQ-4 | Fix constant mismatch | 15m | Low |
| P1 | TC-1 | Config tests | 2h | High |
| P1 | TC-2 | Hint tests | 2h | Med |
| P1 | RL-3 | HTTP timeout flag | 1h | Med |
| P2 | TC-3 | Handler tests | 6h | High |
| P2 | RL-1 | Retry backoff | 3h | High |
| P2 | TC-4 | MCP server tests | 3h | High |
| P3 | FE-1 | Pretty print | 2h | Low |
| P3 | FE-3 | More hints | 2h | Med |
| P3 | DX-1 | Linting | 1h | Med |
| P4 | FE-2 | Metadata cache | 4h | High |
| P4 | TC-5 | Transport tests | 6h | Med |
| P4 | DX-2 | Logging | 3h | Med |

---

*End of document*
