# SAP MCP Server Competitive Analysis

*Generated: December 2025*

## Overview

This document analyzes the SAP-related MCP (Model Context Protocol) server ecosystem and positions the OData MCP Bridge (Go) project within it.

---

## Category 1: OData Data Access Servers (Direct Competitors)

| Feature | **odata_mcp_go (This Project)** | **lemaiwo/btp-sap-odata-to-mcp-server** | **GutjahrAI/sap-odata-mcp-server** | **one-kash/sap-odata-explorer** |
|---------|--------------------------------|----------------------------------------|-----------------------------------|--------------------------------|
| **Language** | Go (compiled) | Node.js/TypeScript | Node.js/TypeScript | TypeScript |
| **Runtime** | None (single binary) | Node.js + BTP | Node.js | Node.js |
| **OData Versions** | v2 + v4 (auto-detect) | Not specified | v2 + v4 implied | Not specified |
| **Deployment** | Binary download | BTP CloudFoundry | npm install | Claude Code skill |
| **Transports** | stdio, HTTP/SSE, Streamable HTTP | Streamable HTTP, stdio | HTTP | Claude Code native |
| **SAP Quirks** | CSRF, GUID, Date, Hints | BTP Destination Service | CSRF handling | Retry + exponential backoff |
| **Token Optimization** | No | Yes (3-level, ~90% reduction) | No | No |
| **Unique Feature** | Service hints system | BTP-native integration | Modular TypeScript | Secure logging with masking |

### Repository Links
- [lemaiwo/btp-sap-odata-to-mcp-server](https://github.com/lemaiwo/btp-sap-odata-to-mcp-server)
- [GutjahrAI/sap-odata-mcp-server](https://github.com/GutjahrAI/sap-odata-mcp-server)
- [one-kash/sap-odata-explorer](https://github.com/one-kash/sap-odata-explorer)

---

## Category 2: SAP System Access Servers (Complementary)

| Server | Purpose | Language | What It Accesses |
|--------|---------|----------|------------------|
| **mario-andreschak/mcp-abap-adt** | ABAP development | TypeScript | Programs, Classes, Functions, Tables, Packages via ADT API |
| **vadimklimov/cpi-mcp-server** | Integration Suite | Go | Integration packages, flows, mappings, scripts, runtime artifacts |
| **mario-andreschak/mcp-sap-gui** | GUI automation | TypeScript | Any SAP GUI transaction via simulated clicks/keyboard |

### Repository Links
- [mario-andreschak/mcp-abap-adt](https://github.com/mario-andreschak/mcp-abap-adt)
- [vadimklimov/cpi-mcp-server](https://github.com/vadimklimov/cpi-mcp-server)
- [mario-andreschak/mcp-sap-gui](https://github.com/mario-andreschak/mcp-sap-gui)

**Note:** The cpi-mcp-server is also written in Go and follows similar patterns (stdio + HTTP transports, OAuth2 auth). It's a sibling project for CPI rather than OData.

---

## Category 3: SAP Knowledge/Documentation Servers

| Server | Purpose | What It Provides |
|--------|---------|------------------|
| **marianfoo/mcp-sap-docs** | Documentation access | SAP Help, UI5 APIs, SAP Community content (offline + real-time) |
| **marianfoo/mcp-sap-notes** | SAP Notes access | SAP Note search & retrieval via SAP Passport authentication |

### Repository Links
- [marianfoo/mcp-sap-docs](https://github.com/marianfoo/mcp-sap-docs)
- [marianfoo/mcp-sap-notes](https://github.com/marianfoo/mcp-sap-notes)

These are **information retrieval** servers, not data access servers. They help AI understand SAP concepts rather than query SAP data.

---

## Category 4: Official SAP Development Servers

Per [SAP Build's announcement](https://community.sap.com/t5/technology-blog-posts-by-sap/sap-build-introduces-new-mcp-servers-to-enable-agentic-development-for/ba-p/14205602), SAP released 4 official MCP servers:

| Server | Purpose | License |
|--------|---------|---------|
| **CAP MCP Server** | Cloud Application Programming Model development | Apache-2.0 |
| **SAP Fiori Elements MCP** | Fiori Elements development | Apache-2.0 |
| **UI5 MCP Server** | SAPUI5 development (templates, linting, API docs) | Apache-2.0 |
| **MDK MCP Server** | Mobile Development Kit | Apache-2.0 |

### Repository Links
- [UI5/mcp-server](https://github.com/UI5/mcp-server)
- [SAP/mdk-mcp-server](https://github.com/SAP/mdk-mcp-server)

These are **code generation assistants** — they help write SAP apps, not access SAP data.

---

## Ecosystem Map

```
┌─────────────────────────────────────────────────────────────────────────┐
│                         SAP MCP SERVER ECOSYSTEM                         │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                          │
│  ┌──────────────────────────────────────────────────────────────────┐   │
│  │                     DATA ACCESS LAYER                             │   │
│  │                                                                   │   │
│  │   ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐  │   │
│  │   │  odata_mcp_go   │  │ btp-sap-odata   │  │  sap-odata-mcp  │  │   │
│  │   │  (THIS PROJECT) │  │ (lemaiwo)       │  │  (GutjahrAI)    │  │   │
│  │   │                 │  │                 │  │                 │  │   │
│  │   │  • Go binary    │  │  • BTP-native   │  │  • TypeScript   │  │   │
│  │   │  • v2 + v4      │  │  • Token-opt    │  │  • Modular      │  │   │
│  │   │  • SAP hints    │  │  • 3-level      │  │  • Basic        │  │   │
│  │   └─────────────────┘  └─────────────────┘  └─────────────────┘  │   │
│  └──────────────────────────────────────────────────────────────────┘   │
│                                                                          │
│  ┌──────────────────────────────────────────────────────────────────┐   │
│  │                     SYSTEM ACCESS LAYER                           │   │
│  │                                                                   │   │
│  │   ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐  │   │
│  │   │   mcp-abap-adt  │  │ cpi-mcp-server  │  │   mcp-sap-gui   │  │   │
│  │   │                 │  │                 │  │                 │  │   │
│  │   │  ABAP objects   │  │  CPI artifacts  │  │  GUI screens    │  │   │
│  │   │  via ADT API    │  │  via REST API   │  │  via automation │  │   │
│  │   └─────────────────┘  └─────────────────┘  └─────────────────┘  │   │
│  └──────────────────────────────────────────────────────────────────┘   │
│                                                                          │
│  ┌──────────────────────────────────────────────────────────────────┐   │
│  │                    KNOWLEDGE ACCESS LAYER                         │   │
│  │                                                                   │   │
│  │   ┌─────────────────┐  ┌─────────────────┐                       │   │
│  │   │  mcp-sap-docs   │  │  mcp-sap-notes  │                       │   │
│  │   │                 │  │                 │                       │   │
│  │   │  Help, UI5 API  │  │  SAP Notes via  │                       │   │
│  │   │  Community      │  │  SAP Passport   │                       │   │
│  │   └─────────────────┘  └─────────────────┘                       │   │
│  └──────────────────────────────────────────────────────────────────┘   │
│                                                                          │
│  ┌──────────────────────────────────────────────────────────────────┐   │
│  │                   DEVELOPMENT ASSIST LAYER (Official SAP)         │   │
│  │                                                                   │   │
│  │   ┌────────┐  ┌────────┐  ┌────────┐  ┌────────┐                 │   │
│  │   │  CAP   │  │ Fiori  │  │  UI5   │  │  MDK   │                 │   │
│  │   │  MCP   │  │  MCP   │  │  MCP   │  │  MCP   │                 │   │
│  │   └────────┘  └────────┘  └────────┘  └────────┘                 │   │
│  └──────────────────────────────────────────────────────────────────┘   │
│                                                                          │
└─────────────────────────────────────────────────────────────────────────┘
```

---

## Competitive Position Summary

| Dimension | This Project's Position |
|-----------|------------------------|
| **Deployment Simplicity** | **#1** — single binary, no runtime, no cloud required |
| **OData Version Support** | **#1** — explicit v2 + v4 with auto-detection |
| **SAP Quirks Handling** | **#1** — CSRF retry, GUID formatting, date conversion, hints system |
| **Cross-Platform** | **#1** — native binaries for Win/Mac/Linux |
| **BTP Integration** | #3 — no BTP Destination Service support (lemaiwo wins) |
| **Token Optimization** | #3 — no multi-level discovery (lemaiwo wins) |
| **Enterprise Auth** | #2 — basic/cookie only, no OAuth2 for OData (CPI server has OAuth2) |

---

## Unique Differentiators

1. **The only Go-based OData MCP server** — vadimklimov's CPI server is also Go, but for Integration Suite, not OData
2. **Service hints system** — no other OData server has pattern-matching guidance for known service issues
3. **AI Foundry compatibility** — configurable protocol version (`--protocol-version 2025-06-18`)
4. **Claude Code friendly mode** — `--claude-code-friendly` flag for parameter naming compatibility
5. **Operation filtering** — fine-grained `--enable`/`--disable` flags (CSFGUDA operations)
6. **Read-only modes** — `--read-only` and `--read-only-but-functions` for safety

---

## Target Users

- Developers who want OData access without BTP infrastructure
- Users on Windows/macOS/Linux who want a single binary
- SAP consultants who need quick OData exploration
- Teams using AI Foundry or other MCP clients beyond Claude Desktop

---

## Feature Overlap Analysis

### Features We Share With Competitors

| Feature | Us | lemaiwo | GutjahrAI | one-kash |
|---------|-----|---------|-----------|----------|
| Dynamic tool generation from $metadata | ✅ | ✅ | ✅ | ✅ |
| CRUD operations | ✅ | ✅ | ✅ | ✅ |
| CSRF token handling | ✅ | ✅ | ✅ | ❌ |
| $filter, $select, $expand support | ✅ | ✅ | ✅ | ✅ |
| stdio transport | ✅ | ✅ | ❌ | ❌ |
| HTTP transport | ✅ | ✅ | ✅ | ❌ |
| Basic authentication | ✅ | ❌ (BTP) | ✅ | ✅ |
| Function imports | ✅ | ❓ | ✅ | ❓ |

### Features Others Have That We Lack

| Feature | Who Has It | Benefit | Priority |
|---------|-----------|---------|----------|
| **3-Level Discovery** | lemaiwo | ~90% token reduction for large schemas | HIGH |
| **BTP Destination Service** | lemaiwo | Enterprise SSO, principal propagation | MEDIUM |
| **OAuth2 Client Credentials** | cpi-mcp-server | Modern auth for cloud services | MEDIUM |
| **Session Management** | lemaiwo | Multi-service connections with cleanup | LOW |
| **Secure Logging with Masking** | one-kash | Credential protection in logs | LOW |
| **Exponential Backoff Retry** | one-kash | Better resilience for flaky services | LOW |

---

## Improvement Opportunities

### High Priority

#### 1. Token-Optimized Discovery (from lemaiwo)
**Problem:** Large SAP services can have 100+ entity sets. Loading full metadata consumes significant tokens.

**Solution:** Implement 3-level discovery:
- Level 1: Lightweight list of services/entities (names only)
- Level 2: Full schema on-demand for selected entities
- Level 3: Execute operations using Level 2 metadata

**Implementation:**
- Add `--discovery-mode` flag with values: `full` (default), `lazy`
- In lazy mode, `odata_service_info` returns entity names only
- Add `get_entity_metadata_{EntitySet}` tool for on-demand schema loading
- Cache metadata per entity set

**Estimated Benefit:** 80-90% reduction in initial context consumption

#### 2. OAuth2 Client Credentials Flow (from cpi-mcp-server)
**Problem:** Cloud services increasingly use OAuth2 instead of basic auth.

**Solution:** Add OAuth2 client credentials support:
```bash
./odata-mcp --oauth2-client-id "xxx" --oauth2-client-secret "yyy" \
            --oauth2-token-url "https://auth.server/oauth/token" \
            https://my-cloud-service.com/odata/
```

**Implementation:**
- Add `internal/auth/oauth2.go` with token refresh logic
- Store tokens with TTL, auto-refresh before expiry
- Support `ODATA_OAUTH2_*` environment variables

### Medium Priority

#### 3. BTP Destination Service Integration
**Problem:** Enterprise SAP deployments use BTP Destination Service for centralized credential management.

**Solution:** Add BTP destination lookup:
```bash
./odata-mcp --btp-destination "MY_DESTINATION" \
            --btp-subaccount "xxx" \
            --btp-client-id "yyy" \
            --btp-client-secret "zzz"
```

**Note:** This is complex and may be better as a separate wrapper tool.

#### 4. Exponential Backoff Retry (from one-kash)
**Problem:** SAP services can be flaky, especially during high load.

**Current:** We retry on CSRF failure only.

**Solution:** Add configurable retry with exponential backoff:
```bash
./odata-mcp --max-retries 3 --retry-backoff-ms 1000 https://...
```

**Implementation:**
- Add retry logic in `internal/client/client.go`
- Configurable max retries, initial backoff, max backoff
- Retry on 429, 503, 504, connection errors

### Low Priority

#### 5. Secure Logging with Credential Masking (from one-kash)
**Problem:** Verbose logs might accidentally expose credentials.

**Current:** We truncate CSRF tokens but don't systematically mask credentials.

**Solution:** Add credential masking in verbose output:
- Mask `Authorization` header values
- Mask URL credentials (`user:pass@host`)
- Add `--log-mask-fields` for custom field masking

#### 6. Multi-Service Session Management (from lemaiwo)
**Problem:** Users might want to connect to multiple OData services in one session.

**Solution:** This would require significant architecture changes. Consider for v2.0.

---

## Recommendations

### Short-term (v1.6)
1. **Implement lazy discovery mode** — biggest impact for enterprise users
2. **Add exponential backoff retry** — low effort, high resilience benefit

### Medium-term (v1.7)
1. **Add OAuth2 client credentials** — required for modern cloud services
2. **Enhance credential masking** — security hygiene

### Long-term (v2.0)
1. **Consider BTP integration** — either native or as wrapper
2. **Evaluate multi-service support** — significant architecture change

---

## References

- [SAP Community: Universal OData MCP Bridge](https://community.sap.com/t5/technology-blog-posts-by-members/universal-odata-mcp-bridge-or-how-i-accidentally-made-15-000-enterprise/ba-p/14134696)
- [SAP Community: MCP Comprehensive Guide](https://community.sap.com/t5/technology-blog-posts-by-sap/mcp-a-comprehensive-guide/ba-p/14238053)
- [SAP Build MCP Servers Announcement](https://community.sap.com/t5/technology-blog-posts-by-sap/sap-build-introduces-new-mcp-servers-to-enable-agentic-development-for/ba-p/14205602)
- [SAP on Azure Podcast: The MCP Bridge](https://www.saponazurepodcast.de/episode248/)
