# Skill 1: Repo Scout

Build a mental model of the repository structure, identify entry points, and document how to build/test/run the project.

## When to Use

- First step in any codebase analysis
- Onboarding to a new project
- Before making significant changes

## Inputs

- Repository path
- Optional: Focus area (e.g., "transport layer", "authentication")

## Prompt

```
You are analyzing a codebase. Your goal is to build a comprehensive map of how it works.

## Instructions

1. **Directory Structure**: Glob for all directories and major file types. Create an annotated tree showing what each area is responsible for.

2. **Entry Points**: Find all main() functions, init() functions, and HTTP handlers. Document:
   - CLI entry points
   - Server/daemon modes
   - Test entry points

3. **Build & Run**: Read README.md, Makefile, package.json, go.mod, or equivalent. Document:
   - How to build
   - How to run tests (unit, integration, e2e)
   - Common development commands
   - Required environment variables

4. **Dependencies**: Identify:
   - External services (databases, APIs, queues)
   - Key libraries and their purposes
   - Version constraints

5. **Architecture**: Summarize:
   - Data flow (how requests/data move through the system)
   - Key abstractions and interfaces
   - Extension points

## Output Format

```markdown
## Repo Map

### Directory Structure
[annotated tree]

### Key Files (>100 lines, high importance)
| File | Lines | Responsibility |
|------|-------|----------------|

## Entry Points

### CLI
- `cmd/*/main.go` — [description]

### Server Modes
- [mode]: [how to start]

### Test Entry Points
- [test command]

## Build & Run

### Build
```bash
[commands]
```

### Test
```bash
[commands]
```

### Common Operations
| Task | Command |
|------|---------|

### Environment Variables
| Var | Required | Description |
|-----|----------|-------------|

## Dependencies

### External Services
| Service | Purpose | Required |
|---------|---------|----------|

### Key Libraries
| Library | Version | Purpose |
|---------|---------|---------|

## Architecture Notes

### Data Flow
[description or diagram]

### Key Abstractions
- [abstraction]: [purpose]

### Extension Points
- [where and how to extend]
```

Focus on accuracy over completeness. Mark anything uncertain as "NEEDS_VERIFICATION".
```

## Expected Output

Structured Markdown document with:
- Annotated directory tree
- Build/test commands verified to work
- Clear entry point documentation
- Architecture summary

## Tips

- Start with README and Makefile
- Look for CI/CD configs for accurate build commands
- Check for docker-compose.yml for service dependencies
- Review go.mod/package.json for key libraries
