# CLAUDE.md

This file provides guidance to Claude Code when working with this repository.

## Interaction Principles (MANDATORY)

**Be rigorous and constructively direct.** Apply these principles in all interactions:

### Critical Thinking
- **Stress-test ideas**: Identify flaws, edge cases, assumptions, and alternatives—explain why they matter to outcomes
- **Challenge premature abstraction**: Call out when domain modeling is premature; suggest a walking skeleton when uncertainty is high
- **Ground in constraints**: Pull from abstractions to concrete constraints (performance, UX, ops)
- **Question simplicity**: Ask "simplest thing that could work?" and "what must be true for this to fail?"

### Technical Rigor
- **Default to proven solutions**: Demand justification for novelty
- **Flag unknowns**: Identify unknowns being glossed over
- **Use analogies carefully**: Use them briefly, then show where they break; teach transferable meta-patterns
- **Identify thinking patterns**: When correcting, identify the thinking pattern and propose guardrails

### Proactive Guidance
- **Introduce adjacent concepts**: Proactively surface related ideas, contrarian views
- **Occasionally flip the question**: Challenge the framing itself
- **Adapt to mode**:
  - Debugging → concrete, specific
  - Learning → wide exploration
  - Shipping → ruthless scope
  - Architecture → test extremes

### Meta-Rules
- **Match pace**: Adapt communication density to the task
- **Call out misuse**: Tell the user when they're using Claude wrong
- **Override when needed**: Any rule can be overridden when context demands
- **Optimize for**: Learning velocity AND shipping value

## HARD STOP: Before ANY Implementation

**Before writing ANY code or documentation:**

1. [ ] Did I check for applicable skills? (`superpowers:*`)
2. [ ] Did I use `superpowers:brainstorming` for non-trivial tasks?
3. [ ] Did I create a design doc and STOP for approval?

**If ANY answer is NO → STOP and use the skill now.**

### Workflow Trigger

**Use superpowers workflow when:**

- Task involves >1 file
- Task would take >30 minutes
- Multi-file documentation changes
- New guides/tutorials

**No exceptions.** "Simple" tasks that skip workflow become complex problems.

### Documentation = Implementation

Treat documentation creation the same as code:

- Multi-file docs → `superpowers:brainstorming` first
- New guides → design doc + approval gate
- Doc refactoring → `superpowers:writing-plans`

## Project Overview

**OData MCP Bridge (Go)** — A Go binary that bridges OData v2/v4 services to the Model Context Protocol (MCP). It dynamically generates MCP tools from OData `$metadata` and serves them via stdio, HTTP/SSE, or Streamable HTTP transports.

## Quick Reference

### Build & Test
```bash
go build -o odata-mcp ./cmd/odata-mcp    # Build binary
go test ./...                             # Run unit tests
INTEGRATION_TESTS=true go test ./...      # Include integration tests
make dev                                  # Build + test
make help                                 # See all targets
```

### Run Examples
```bash
# Basic usage
./odata-mcp https://services.odata.org/V4/Northwind/Northwind.svc/

# Lazy mode (10 generic tools instead of 500+)
./odata-mcp --service $SERVICE_URL --lazy-metadata

# Read-only mode
./odata-mcp --service $SERVICE_URL --read-only

# Restrict to specific entities
./odata-mcp --service $SERVICE_URL --entities Products,Orders
```

### Configuration Priority
**CLI flags > positional service URL > environment variables (service/auth only) > defaults**

## Configuration Reference

| Flag | Env Var | Default | Description |
|------|---------|---------|-------------|
| `--service` | `ODATA_URL`, `ODATA_SERVICE_URL` | - | OData service URL (positional argument also accepted) |
| `--lazy-metadata` | — | `false` | Use 10 generic tools |
| `--lazy-threshold` | — | `0` | Auto-enable lazy if tools > N |
| `--read-only` | — | `false` | Block mutating operations |
| `--entities` | — | - | Restrict to entity sets (comma-sep) |
| `--enable` | — | - | Enable only these ops (C,S,F,G,U,D,A,R) |
| `--disable` | — | - | Disable these ops |
| `--transport` | — | `stdio` | Transport: stdio, http, streamable-http |
| `--http-addr` | — | `localhost:8080` | HTTP transport address |
| `--protocol-version` | — | `2024-11-05` | MCP protocol version |
| `--verbose` | — | `false` | Enable verbose logging |
| `--hints-file` | — | `hints.json` (binary dir) | Service hints file path |

### Operation Codes
| Code | Operation | Description |
|------|-----------|-------------|
| `C` | Create | POST new entities |
| `S` | Search | Full-text search |
| `F` | Filter | List/query entities |
| `G` | Get | Read single entity |
| `U` | Update | PATCH/PUT entities |
| `D` | Delete | DELETE entities |
| `A` | Actions | Function imports |
| `R` | Read | Read-only ops (S+F+G) |

## Architecture

```
cmd/odata-mcp/main.go       # CLI entrypoint, flag parsing, transport selection
internal/
  bridge/
    bridge.go               # Core: tool generation, OData handlers
    lazy_tools.go           # Lazy mode: 10 generic tools
    lazy_handlers.go        # Lazy mode: runtime entity validation
  mcp/server.go             # MCP protocol: initialize, tools/list, tools/call
  client/client.go          # OData HTTP client, CSRF handling
  transport/                # Transport implementations (stdio, http, streamable-http)
  metadata/                 # OData metadata parsing (v2 + v4)
  config/config.go          # Configuration struct
  models/models.go          # Data structures
  constants/                # Magic strings, defaults
  hint/hint.go              # Service hints system
  utils/                    # Date/numeric conversions
```

## Key Files for Common Tasks

| Task | Files |
|------|-------|
| Add CLI flag | `internal/config/config.go`, `cmd/odata-mcp/main.go` |
| Add tool handler | `internal/bridge/bridge.go` |
| Add lazy mode handler | `internal/bridge/lazy_handlers.go` |
| Add OData operation | `internal/client/client.go` |
| Parse new metadata | `internal/metadata/parser.go` (v2), `parser_v4.go` (v4) |
| Add transport | `internal/transport/`, `cmd/odata-mcp/main.go` |
| Add unit test | `internal/bridge/*_test.go`, `internal/test/*_test.go` |
| Add integration test | `internal/test/*_test.go` (use `INTEGRATION_TESTS=true`) |

## Code Patterns

### Tool Handler Pattern

```go
func (b *ODataMCPBridge) handleEntityFilter(ctx context.Context, entitySet string, args map[string]interface{}) (interface{}, error) {
    // Build query from args
    query := b.buildFilterQuery(args)

    // Execute OData request
    result, err := b.client.Get(ctx, entitySet, query)
    if err != nil {
        return nil, fmt.Errorf("failed to filter %s: %w", entitySet, err)
    }

    return result, nil
}
```

### Lazy Handler Pattern

```go
func (b *ODataMCPBridge) handleLazySomething(ctx context.Context, args map[string]interface{}) (interface{}, error) {
    // Extract entity_set parameter
    entitySet, ok := args["entity_set"].(string)
    if !ok || entitySet == "" {
        return nil, fmt.Errorf("missing required parameter: entity_set")
    }

    // Validate entity set exists AND is allowed by filters
    _, _, err := b.validateEntitySet(entitySet)
    if err != nil {
        return nil, err
    }

    // Delegate to existing handler...
}
```

### CSRF Mutation Pattern

```go
func (c *ODataClient) Post(ctx context.Context, path string, body interface{}) (*Response, error) {
    // Fetch CSRF token before mutation
    token, err := c.fetchCSRFToken(ctx)
    if err != nil {
        return nil, fmt.Errorf("failed to fetch CSRF token: %w", err)
    }

    // Execute with token, auto-retry on 403
    return c.doWithCSRF(ctx, "POST", path, body, token)
}
```

## Key Conventions

### Code Style
- Standard Go formatting (`gofmt`)
- Error wrapping with `fmt.Errorf("context: %w", err)`
- Verbose logging to stderr with `[VERBOSE]` prefix
- No log output to stdout (contaminates MCP stdio)

### CLI Flags
- Defined in `cmd/odata-mcp/main.go` using Cobra
- Environment variable support for service/auth via Viper with `ODATA_` prefix
- Mutually exclusive flags validated in `runBridge()`

### Tool Naming
- Default pattern: `{operation}_{EntitySet}_for_{ServiceID}`
- Service ID extracted by `constants.FormatServiceID()`
- Shrink mode: `update` → `upd`, `delete` → `del`

### OData Version Handling
- Version auto-detected from `$metadata`
- v2: `$inlinecount=allpages`, `/Date()/` format
- v4: `$count=true`, ISO dates
- Translation happens in `client.go`

### SAP Quirks
- CSRF token fetched before every mutation
- Auto-retry on 403 + "CSRF validation failed"
- GUID values auto-transformed: `'uuid'` → `guid'uuid'`
- Detection via URL patterns and metadata annotations

### MCP Protocol
- JSON-RPC 2.0 over transport
- Protocol version configurable (`--protocol-version`)
- Claude default: `2024-11-05`
- AI Foundry: `2025-06-18`

## Testing

- Unit tests: `internal/test/*_test.go`
- Integration tests require network (Northwind, SAP services)
- Run specific test: `go test -run TestName ./internal/test/`
- MCP compliance: `./simple_compliance_test.sh`

## Common Tasks

### Adding a New CLI Flag
1. Add field to `internal/config/config.go`
2. Define flag in `cmd/odata-mcp/main.go` `init()`
3. Process in `runBridge()` if needed
4. Update README.md

### Adding a New Transport
1. Implement `transport.Transport` interface
2. Add case in `runBridge()` switch
3. Add `--transport` option documentation

### Supporting a New OData Feature
1. Update `internal/metadata/parser.go` (v2) or `parser_v4.go`
2. Add to `internal/models/models.go` if new types needed
3. Update tool generation in `internal/bridge/bridge.go`
4. Add query handling in `internal/client/client.go`

## Documentation Hierarchy

| Question | Consult |
|----------|---------|
| "How should this behave?" | [SPEC.md](SPEC.md) — Behavioral contracts, error handling, acceptance criteria |
| "How do I use this tool?" | [README.md](README.md) — User guide, examples, installation |
| "What's planned/shipped?" | [docs/ROADMAP.md](docs/ROADMAP.md) — Version history, backlog, future ideas |
| "How should I develop features?" | [docs/DEVELOPMENT_WORKFLOW.md](docs/DEVELOPMENT_WORKFLOW.md) — SDD + RPI methodology |
| "How do I work on this codebase?" | This file (CLAUDE.md) — Patterns, conventions, common tasks |

## Important Files

| File | Purpose |
|------|---------|
| `SPEC.md` | Behavioral contracts, error handling, acceptance criteria |
| `README.md` | User guide, examples, installation |
| `docs/ROADMAP.md` | Version history, backlog, future ideas |
| `docs/DEVELOPMENT_WORKFLOW.md` | SDD + RPI methodology |
| `hints.json` | Default service hints (SAP workarounds) |
| `CHANGELOG.md` | Version history |
| `AI_FOUNDRY_COMPATIBILITY.md` | Protocol version guide |

## Common Issues

| Problem | Cause | Solution |
|---------|-------|----------|
| CSRF 403 errors | Token expired or missing | Auto-retried; check SAP session timeout |
| Empty tool list | Metadata fetch failed | Check `--service`, credentials, network |
| Tools > 500 | Large OData service | Use `--lazy-metadata` or `--lazy-threshold` |
| Entity not found | Typo or filter mismatch | Check `--entities` flag (case-sensitive) |
| Operation not allowed | Filtered out | Check `--enable`/`--disable` flags |
| Protocol mismatch | Wrong MCP version | Use `--protocol-version 2025-06-18` for AI Foundry |
| Stdout contamination | Log output to stdout | All logs must go to stderr with `[VERBOSE]` |

## Security Notes

- HTTP transports have NO authentication (MCP limitation)
- Default: localhost-only binding
- Credentials never logged
- CSRF tokens truncated in verbose output
- **Never commit** `.env`, credentials, or API keys
- **Verify logs** don't contain secrets before sharing
- Review `git log --all -p` before pushing sensitive repos

## Development Workflow (MANDATORY)

**Superpowers skills ARE the enforcement mechanism.** This project uses `superpowers:brainstorming`, `superpowers:writing-plans`, and `superpowers:executing-plans` with SDD+RPI enhancements.

### Project-Specific Skill Enhancements

When using superpowers skills in this project, apply these additions:

| Skill | Enhancement |
|-------|-------------|
| `brainstorming` | Declare **Complexity Mode** (LITE/STANDARD/HEAVY) at start |
| `brainstorming` | Include **Devil's Advocate** section before gate |
| `brainstorming` | Gate: "Does this Spec + Research align with your understanding?" |
| `writing-plans` | Include **Traceability Matrix** (requirement → files → tests) |
| `writing-plans` | Gate: "Shall I proceed with Implementation?" |
| `executing-plans` | Follow **Brownfield Rules**: no unrelated refactors, match existing patterns |

### Complexity Modes

| Mode | Criteria |
|------|----------|
| LITE | ≤2 files, no schema changes, minimal tests |
| STANDARD | Multiple modules, tests required |
| HEAVY | Cross-cutting, migrations, security/perf critical |

**Full methodology:** [docs/DEVELOPMENT_WORKFLOW.md](docs/DEVELOPMENT_WORKFLOW.md)

## Documentation Sync Requirements (MANDATORY)

**All code changes MUST include corresponding documentation updates.** This ensures the codebase remains self-documenting and reduces knowledge decay.

### Tier 1: Core Documentation (Update with every release)

| File | Purpose | Update Trigger |
|------|---------|----------------|
| `CHANGELOG.md` | Version history | Every change merged |
| `README.md` | User documentation | Features, flags, examples |
| `SPEC.md` | Behavioral specification | API contracts, error codes |
| `docs/ROADMAP.md` | Development roadmap | Completed/planned work |
| `CLAUDE.md` | AI assistant guidance | Conventions, commands |

### Tier 2: Feature-Specific (Update when feature changes)

| File | Purpose | Update Trigger |
|------|---------|----------------|
| `AI_FOUNDRY_COMPATIBILITY.md` | Protocol version guide | Protocol changes |
| `HINTS.md` | Hints system documentation | Hint format changes |
| `QUICK_REFERENCE.md` | CLI cheat sheet | Flag additions/changes |
| `TROUBLESHOOTING.md` | Common issues/solutions | New error patterns |
| `SECURITY.md` | Security guidance | Auth/transport changes |
| `RELEASING.md` | Release process | CI/CD changes |
| `VERSIONING.md` | Version policy | Process changes |

### Tier 3: Implementation Guides (Update on major refactors)

| File | Purpose |
|------|---------|
| `SAP_DATE_HANDLING.md` | SAP date conversion logic |
| `SAP_NUMERIC_HANDLING.md` | SAP numeric quirks |
| `ODATA_V4_IMPLEMENTATION.md` | v4 support details |
| `CSRF_COMPARISON.md` | CSRF handling analysis |

### Configuration Files (Update when behavior changes)

| File | Purpose | Update Trigger |
|------|---------|----------------|
| `hints.json` | Default service hints | New SAP patterns |
| `Makefile` | Build commands | Build process changes |
| `.github/workflows/*.yml` | CI/CD pipelines | Test/release changes |

### Sync Checklist

Before completing any PR or commit:

1. **New CLI flag?** → Update README.md, QUICK_REFERENCE.md, SPEC.md
2. **New feature?** → Update CHANGELOG.md, README.md, relevant guides
3. **Bug fix?** → Update CHANGELOG.md, TROUBLESHOOTING.md if applicable
4. **API change?** → Update SPEC.md, AI_FOUNDRY_COMPATIBILITY.md if protocol
5. **Security change?** → Update SECURITY.md, SPEC.md
6. **Version bump?** → Update CHANGELOG.md, README.md "What's New" section

### Release Documentation Requirements (MANDATORY)

**Every version release MUST have these artifacts:**

| Artifact | Location | Purpose | When to Create |
|----------|----------|---------|----------------|
| **Design Document** | `docs/plans/YYYY-MM-DD-vX.Y.Z-*.md` | Pre-implementation spec | Before coding starts |
| **Implementation Report** | `reports/YYYY-MM-DD-05#-*.md` | Post-implementation summary | After release |
| **CHANGELOG Entry** | `CHANGELOG.md` | Version history | With release |

**Design Document Template** (`docs/plans/`):
- Problem Statement
- Solution Architecture
- CLI Interface (if applicable)
- Files Changed
- Acceptance Criteria
- Risk Assessment
- Success Metrics

**Implementation Report Template** (`reports/` - category 050-099):
- Version and date
- Features implemented
- Files changed with line counts
- Test coverage
- Known issues (if any)

**Failure to create these documents = incomplete release.**

### Documentation Naming Conventions

```
docs/plans/
  YYYY-MM-DD-vX.Y.Z-feature-name-design.md    # Design docs

reports/
  YYYY-MM-DD-001-design-doc-name.md           # Design (001-049)
  YYYY-MM-DD-050-vX.Y.Z-implementation.md     # Implementation (050-099)
  YYYY-MM-DD-100-analysis-name.md             # Analysis (100-149)
  YYYY-MM-DD-150-troubleshooting-name.md      # Troubleshooting (150-199)
```

## Feature Status

| Feature | Status |
|---------|--------|
| OData v2 Support | ✅ Complete |
| OData v4 Support | ✅ Complete |
| Lazy Metadata Mode | ✅ Complete |
| Multi-LLM Platform Guides | ✅ Complete |
| SAP CSRF Handling | ✅ Complete |
| HTTP/SSE Transport | ✅ Complete |
| Streamable HTTP | ✅ Complete |
| Service Hints | ✅ Complete |
| Operation Filters | ✅ Complete |
| Entity Filters | ✅ Complete |
| MCP Protocol Versions | ✅ Complete |
| AI Foundry Compatibility | ✅ Complete |

For version roadmap, planned features, and future exploration ideas, see [docs/ROADMAP.md](docs/ROADMAP.md).

## Reports and Documentation

### Report Naming Convention

Format: `./reports/{YYYY-MM-DD-###-title}.md`

Examples:

- `2025-12-17-001-lazy-metadata-design.md`
- `2025-12-19-001-v1.7.0-implementation-complete.md`

### Report Categories

| Range | Category | Purpose |
|-------|----------|---------|
| 001-049 | Design Documents | Pre-implementation specs |
| 050-099 | Implementation Reports | Post-implementation summaries |
| 100-149 | Analysis & Research | Investigations, deep dives |
| 150-199 | Troubleshooting | Issue resolution docs |

### Current Reports

#### Implementation Reports (050-099)

| File | Purpose |
|------|---------|
| `reports/2025-12-19-050-v1.7.0-lazy-metadata-implementation.md` | v1.7.0 Lazy Metadata Mode |
| `reports/2025-12-17-054-v1.6.5-timeouts-sse.md` | v1.6.5 Timeouts/SSE |
| `reports/2025-12-16-053-v1.6.3-bug-fixes.md` | v1.6.3 Bug Fixes |
| `reports/2025-12-14-052-v1.6.0-major-features.md` | v1.6.0 Major Features |
| `reports/2024-06-30-051-v0.1.0-initial-implementation.md` | v0.1.0 Foundation |

#### Analysis Reports (100-149)

| File | Purpose |
|------|---------|
| `reports/2025-12-19-100-analysis-skills-suite.md` | Analysis Skills Suite |
| `reports/2025-09-01-101-hints-system-blog-post.md` | Hints System Blog Post |
| `reports/2025-12-01-102-e2e-strategic-guide.md` | E2E Strategic Guide |
| `reports/2025-12-01-103-competitive-analysis.md` | Competitive Analysis |
| `reports/2025-12-01-104-solman-hub-architecture.md` | SolMan Hub Architecture |

### Current Documentation

#### Strategic & Marketing

| File | Purpose |
|------|---------|
| `docs/002-odata-mcp-september-2025-update.md` | Hints System Blog Post (engaging intro) |
| `docs/003-odata-mcp-e2e-documentation.md` | Strategic Guide for SAP Leaders |
| `docs/COMPETITIVE_ANALYSIS.md` | SAP MCP Ecosystem Analysis |
| `docs/SOLMAN_HUB_ARCHITECTURE.md` | SolMan Hub Architecture Proposal |

#### Development & Process

| File | Purpose |
|------|---------|
| `docs/DEVELOPMENT_WORKFLOW.md` | SDD + RPI Methodology |
| `docs/ROADMAP.md` | Roadmap and Backlog |

#### Platform Integration Guides

| File | Purpose |
|------|---------|
| `docs/LLM_COMPATIBILITY.md` | Master overview, compatibility matrix |
| `docs/IDE_INTEGRATION.md` | Claude Desktop, Cline, Roo Code, Cursor, Windsurf |
| `docs/CHAT_PLATFORM_INTEGRATION.md` | ChatGPT, GitHub Copilot |

#### Design Documents (December 2025)

| File | Purpose |
|------|---------|
| `docs/plans/2025-12-14-v1.6.0-major-features-design.md` | v1.6.0 Major Features Design |
| `docs/plans/2025-12-16-v1.6.3-bug-fixes-design.md` | v1.6.3 Bug Fixes Design |
| `docs/plans/2025-12-17-v1.6.5-timeouts-sse-design.md` | v1.6.5 Timeouts/SSE Design |
| `docs/plans/2025-12-17-lazy-metadata-design.md` | v1.7.0 Lazy Metadata Design |
| `docs/plans/2025-12-19-llm-compatibility-docs-design.md` | LLM Compatibility Docs Design |

#### Analysis Skills Suite

| File | Purpose |
|------|---------|
| `docs/skills/README.md` | Skills Overview & Pipeline |
| `docs/skills/01-repo-scout.md` | Architecture Analysis |
| `docs/skills/02-test-auditor.md` | Test Coverage Audit |
| `docs/skills/03-debt-collector.md` | Technical Debt Harvest |
| `docs/skills/04-reliability-reviewer.md` | Reliability/Security Review |
| `docs/skills/05-dx-reviewer.md` | Developer Experience Review |
| `docs/skills/06-roadmap-builder.md` | Prioritized Roadmap Builder |

## Analysis Skills Usage

```bash
# Full analysis pipeline (45-75 min total)
1. Run Repo Scout first (provides context)
2. Run Test Auditor, Debt Collector, Reliability, DX in parallel
3. Run Roadmap Builder with all outputs
```

See [reports/2025-12-19-100-analysis-skills-suite.md](reports/2025-12-19-100-analysis-skills-suite.md) for detailed documentation.

## Legacy Documentation (Root Level)

Older docs from before the current structure. Reference as needed.

| File | Purpose |
|------|---------|
| `IMPLEMENTATION_GUIDE.md` | Original implementation walkthrough |
| `IMPLEMENTATION_SUMMARY.md` | Implementation overview |
| `ODATA_V4_IMPLEMENTATION.md` | OData v4 support details |
| `SAP_DATE_HANDLING.md` | SAP date conversion logic |
| `SAP_NUMERIC_HANDLING.md` | SAP numeric quirks |
| `CSRF_COMPARISON.md` | CSRF handling analysis |
| `MCP_COMPLIANCE_REPORT.md` | MCP protocol compliance |
| `mcp_protocol_analysis.md` | Protocol analysis |
| `FIXES_SUMMARY.md` | Historical fixes summary |
| `ISSUE_FIXES.md` | Issue resolutions |
| `BUG_FIX_REPORT_AAP-GO-001.md` | AAP-GO-001 bug fix |
| `issue_9_response.md` | GitHub issue #9 response |
| `RELEASE_NOTES.md` | General release notes |
| `RELEASE_NOTES_v1.5.1.md` | v1.5.1 specific notes |
| `WINDOWS_TRACE_GUIDE.md` | Windows debugging guide |
| `WSL_TRACE_GUIDE.md` | WSL debugging guide |
