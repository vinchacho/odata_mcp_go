# Lazy Metadata Design (v1.7.0)

**Status**: Approved
**Author**: Claude + Vincent
**Date**: 2025-12-17
**Target Version**: v1.7.0

---

## Problem Statement

For large OData services (especially SAP), the current eager tool generation creates significant token overhead:

- 100+ entity sets × ~5 tools each = 500+ tools
- Each tool definition includes name, description, and full input schema
- Estimated 200-500 tokens per tool = **100K-250K tokens** in `tools/list` response
- AI clients must process all tool definitions before doing useful work

**Primary goal**: Reduce token cost of tool discovery by ~90%.

**Secondary goals**: Reduce startup latency, improve tool discoverability.

---

## Solution: Hybrid Lazy Mode

Instead of generating per-entity tools (`filter_Products`, `get_Products`, etc.), generate generic tools that accept `entity_set` as a parameter.

### Behavior Comparison

| Mode | `tools/list` payload | Schema loading | Tool count |
|------|---------------------|----------------|------------|
| **Eager** (current) | Full schemas embedded | Upfront | ~500 (entities × operations) |
| **Lazy** (new) | Entity names only | On-demand | 10 generic tools |

### Trigger Logic

```
if --lazy-metadata OR (--lazy-threshold > 0 AND estimated_tools > threshold):
    use lazy mode
else:
    use eager mode (current behavior)
```

**Default**: Eager mode (backwards compatible). Lazy is opt-in.

---

## Lazy Mode Tool Set

| Tool | Parameters | Description |
|------|------------|-------------|
| `odata_service_info` | `include_metadata?` | Service overview, entity list, function list |
| `list_entities` | `entity_set`, `filter?`, `select?`, `expand?`, `orderby?`, `top?`, `skip?`, `count?` | Generic query tool |
| `count_entities` | `entity_set`, `filter?` | Count entities with optional filter |
| `get_entity` | `entity_set`, `key` | Get single entity by key |
| `get_entity_schema` | `entity_set` | Return schema (fields, types, keys, nav props) |
| `create_entity` | `entity_set`, `data` | Create entity |
| `update_entity` | `entity_set`, `key`, `data` | Update entity |
| `delete_entity` | `entity_set`, `key` | Delete entity |
| `list_functions` | (none) | List available function imports |
| `call_function` | `function_name`, `params` | Call any function import |

**Total: 10 tools** regardless of service size.

---

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                     LAZY LOADING FLOW                           │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  Startup (same as today):                                       │
│    1. Fetch $metadata from OData service (HTTP call)            │
│    2. Parse into b.metadata (stored in memory)                  │
│    3. Generate 10 generic tools (vs 500+ eager tools)           │
│                                                                 │
│  AI calls list_entities { entity_set: "Products", ... }         │
│    → Handler extracts entity_set                                │
│    → Lookup schema from b.metadata (already in memory)          │
│    → Execute OData query using existing client logic            │
│                                                                 │
│  AI calls get_entity_schema { entity_set: "Products" }          │
│    → Return schema from b.metadata                              │
│    → No additional HTTP calls                                   │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

**Key insight**: Metadata is fetched ONCE at startup (same as today). The "lazy" part is about what we send to the MCP client, not about fetching data.

---

## CLI Interface

```bash
# Explicit opt-in
odata-mcp --service https://... --lazy-metadata

# Auto-enable if tool count exceeds threshold
odata-mcp --service https://... --lazy-threshold 100

# Environment variables
ODATA_LAZY_METADATA=true
ODATA_LAZY_THRESHOLD=100
```

**Defaults**:

- `--lazy-metadata`: false
- `--lazy-threshold`: 0 (disabled)

---

## Implementation Plan

### Phase 1: Config & CLI

**Files**:

- `internal/config/config.go` — Add `LazyMetadata`, `LazyThreshold` fields
- `cmd/odata-mcp/main.go` — Add CLI flags, env bindings

### Phase 2: Lazy Tool Generators

**Files**:

- `internal/bridge/lazy_tools.go` (NEW) — Generate 10 generic tools
- `internal/bridge/lazy_handlers.go` (NEW) — Handlers that resolve entity_set → schema

### Phase 3: Bridge Integration

**Files**:

- `internal/bridge/bridge.go` — Add `shouldUseLazyMode()`, split `generateTools()` into eager/lazy paths

### Phase 4: Testing

**Files**:

- `internal/bridge/lazy_tools_test.go` (NEW) — Unit tests
- `internal/test/lazy_mode_test.go` (NEW) — Integration tests

### Phase 5: Documentation

**Files**:

- `README.md` — Lazy mode section
- `CHANGELOG.md` — v1.7.0 entry

---

## Backwards Compatibility

- **Default behavior unchanged**: Eager mode remains default
- **Opt-in lazy mode**: Explicit `--lazy-metadata` flag
- **Read-only modes respected**: `--read-only` still works in lazy mode
- **Operation filtering respected**: `-o` flag still works

---

## Risk Assessment

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| Lazy handlers duplicate logic | Low | Medium | Delegate to existing handlers after schema resolution |
| Key handling differs per entity | Low | Low | Reuse existing `handleEntityGet` which handles composite/GUID keys |
| AI needs entity names first | Medium | Low | `odata_service_info` returns entity list; AI calls that first |

---

## Success Metrics

| Metric | Target | Measurement |
|--------|--------|-------------|
| Token reduction | >90% | Compare `tools/list` response size |
| Startup time | No regression | Benchmark eager vs lazy |
| Test coverage | >80% | `go test -cover` for lazy_tools.go |

---

## Open Questions (Resolved)

**Q: Where's the efficiency if schema is loaded on every call?**

A: Schema is NOT loaded on every call. Metadata is fetched once at startup and stored in `b.metadata`. Lazy mode just avoids embedding full schemas in tool definitions.

**Q: What about multi-user scenarios?**

A: Same as today — single bridge instance serves all requests with shared in-memory metadata.

---

## Appendix: Token Math

### Eager Mode (100 entity sets)

```
100 entities × 5 tools/entity = 500 tools
500 tools × ~300 tokens/tool = 150,000 tokens
```

### Lazy Mode

```
10 generic tools × ~200 tokens/tool = 2,000 tokens
```

**Savings: ~99%**
