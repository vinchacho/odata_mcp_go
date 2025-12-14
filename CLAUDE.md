# CLAUDE.md

This file provides guidance to Claude Code when working with this repository.

## Project Overview

**OData MCP Bridge (Go)** — A Go binary that bridges OData v2/v4 services to the Model Context Protocol (MCP). It dynamically generates MCP tools from OData `$metadata` and serves them via stdio, HTTP/SSE, or Streamable HTTP transports.

## Architecture

```
cmd/odata-mcp/main.go    # CLI entrypoint, flag parsing, transport selection
internal/
  bridge/bridge.go       # Core: tool generation, OData handlers
  mcp/server.go          # MCP protocol: initialize, tools/list, tools/call
  client/client.go       # OData HTTP client, CSRF handling
  transport/             # Transport implementations (stdio, http, streamable)
  metadata/              # OData metadata parsing (v2 + v4)
  config/config.go       # Configuration struct
  models/models.go       # Data structures
  constants/             # Magic strings, defaults
  hint/hint.go           # Service hints system
  utils/                 # Date/numeric conversions
```

## Build & Test Commands

```bash
# Build for current platform
make build

# Build for all platforms
make build-all

# Run tests
make test
# or
go test ./...

# Development build + test
make dev

# Check version
make version

# See all make targets
make help
```

## Key Conventions

### Code Style
- Standard Go formatting (`gofmt`)
- Error wrapping with `fmt.Errorf("context: %w", err)`
- Verbose logging to stderr with `[VERBOSE]` prefix
- No log output to stdout (contaminates MCP stdio)

### CLI Flags
- Defined in `cmd/odata-mcp/main.go` using Cobra
- Environment variable support via Viper with `ODATA_` prefix
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

## Important Files

| File | Purpose |
|------|---------|
| `SPEC.md` | Behavioral specification (contracts, requirements) |
| `README.md` | User documentation |
| `docs/ROADMAP.md` | Roadmap, backlog, and improvement tracking |
| `hints.json` | Default service hints (SAP workarounds) |
| `CHANGELOG.md` | Version history |
| `AI_FOUNDRY_COMPATIBILITY.md` | Protocol version guide |

## Security Notes

- HTTP transports have NO authentication (MCP limitation)
- Default: localhost-only binding
- Credentials never logged
- CSRF tokens truncated in verbose output

## Development Workflow (MANDATORY)

**All development MUST follow the SDD + RPI methodology.** See [docs/DEVELOPMENT_WORKFLOW.md](docs/DEVELOPMENT_WORKFLOW.md) for full details.

### Quick Reference

```text
Turn 1: ARCHITECT (Spec + Research) → STOP for approval
Turn 2: PLANNER (Plan only)         → STOP for approval
Turn 3: BUILDER (Implement)         → Deliver with traceability
```

### Hard Rules

1. **NO CODING** until the Implement phase (Turn 3)
2. **STOP at gates** - always ask for approval before proceeding
3. **Never guess** - ask for missing inputs or mark as ASSUMPTION
4. **Every change** must map to a requirement and acceptance criterion
5. **Include Devil's Advocate** section at each gate
6. **Brownfield rules** - no unrelated refactors/renames/reformatting

### Complexity Modes

| Mode | When to Use |
|------|-------------|
| LITE | ≤2 files, no schema changes, minimal tests |
| STANDARD | Multiple modules, tests required |
| HEAVY | Cross-cutting, migrations, security/perf critical |

### Gate Questions

- After Turn 1: "Does this Spec + Research align with your understanding?"
- After Turn 2: "Shall I proceed with Implementation?"
