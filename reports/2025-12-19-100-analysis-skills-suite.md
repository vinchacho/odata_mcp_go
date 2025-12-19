# Analysis Skills Suite - Research & Documentation Report

**Date:** 2025-12-19
**Report ID:** 100
**Category:** Analysis & Research
**Status:** Complete

## Summary

The Analysis Skills Suite provides 6 structured Claude prompts for systematic codebase analysis. These skills can be run sequentially or in parallel to assess quality, reliability, and improvement opportunities.

## Skills Overview

| # | Skill | Purpose | Time | Inputs |
|---|-------|---------|------|--------|
| 1 | Repo Scout | Architecture & how-to-run | 5-10 min | Repository path |
| 2 | Test Auditor | Coverage gaps & test quality | 10-15 min | Repo Scout output |
| 3 | Debt Collector | TODOs & incomplete features | 5-10 min | Repository path |
| 4 | Reliability Reviewer | Errors, concurrency, security | 15-20 min | Repo Scout output |
| 5 | DX Reviewer | Logging, config, tooling | 10-15 min | Repository path |
| 6 | Roadmap Builder | Prioritization & PR slicing | 10-15 min | All skill outputs |

## Pipeline Architecture

```
                    ┌─────────────────┐
                    │   Repo Scout    │
                    └────────┬────────┘
                             │
         ┌───────────────────┼───────────────────┐
         │                   │                   │
         ▼                   ▼                   ▼
┌─────────────────┐ ┌─────────────────┐ ┌─────────────────┐
│  Test Auditor   │ │ Debt Collector  │ │    Reliability  │
│                 │ │                 │ │    Reviewer     │
└────────┬────────┘ └────────┬────────┘ └────────┬────────┘
         │                   │                   │
         │          ┌────────┴────────┐          │
         │          │   DX Reviewer   │          │
         │          └────────┬────────┘          │
         │                   │                   │
         └───────────────────┼───────────────────┘
                             │
                             ▼
                    ┌─────────────────┐
                    │ Roadmap Builder │
                    └─────────────────┘
```

## Skill Details

### 1. Repo Scout (`01-repo-scout.md`)

**Purpose:** Build a mental model of the repository structure.

**Outputs:**
- Annotated directory tree
- Entry points (CLI, server, test)
- Build & test commands
- External dependencies
- Architecture notes

**Key Searches:**
- `main()` and `init()` functions
- README.md, Makefile, go.mod
- docker-compose.yml
- CI/CD configurations

### 2. Test Auditor (`02-test-auditor.md`)

**Purpose:** Identify test coverage gaps and quality issues.

**Outputs:**
- Coverage matrix by package
- Untested functions (prioritized)
- Test quality issues
- Anti-patterns (network in unit tests, flaky tests)
- Test type distribution (unit/integration/e2e)

**Key Searches:**
- `*_test.go`, `*.test.js`, `*_test.py`
- `t.Skip`, `@skip`, `pytest.skip`
- `time.Sleep`, hardcoded delays
- Missing error case tests

### 3. Debt Collector (`03-debt-collector.md`)

**Purpose:** Harvest technical debt markers.

**Outputs:**
- TODO/FIXME/HACK inventory
- Incomplete implementations
- Deprecated patterns
- Theme clustering
- Quick wins (<30 min fixes)

**Key Searches:**
- `TODO`, `FIXME`, `HACK`, `XXX`, `TEMP`
- `panic("not implemented")`
- `io/ioutil` (deprecated in Go)
- Empty catch blocks

### 4. Reliability Reviewer (`04-reliability-reviewer.md`)

**Purpose:** Find reliability, concurrency, and security issues.

**Outputs:**
- Error handling issues (swallowed, missing, generic)
- Retry & timeout gaps
- Race conditions
- Goroutine/thread leaks
- Security issues (credential exposure, injection)

**Key Searches:**
- `_, _ :=` (swallowed errors in Go)
- `go func()` without context
- password/secret/token in logs
- String concatenation in SQL

**Severity Levels:**
- Critical: Security breach, data loss
- High: Service outage
- Medium: Degraded service
- Low: Minor impact

### 5. DX Reviewer (`05-dx-reviewer.md`)

**Purpose:** Identify developer experience friction.

**Outputs:**
- DX score card (5-point scale)
- Logging consistency issues
- Configuration hardcoding
- Documentation gaps
- Tooling assessment
- Onboarding friction points

**Areas Assessed:**
| Area | Score Range | Criteria |
|------|-------------|----------|
| Logging | 0-5 | Consistency, levels, output streams |
| Configuration | 0-5 | Env vars, flags, documentation |
| Documentation | 0-5 | README, API docs, examples |
| Tooling | 0-5 | Linting, formatting, CI/CD |
| Onboarding | 0-5 | Time to first build, gotchas |

### 6. Roadmap Builder (`06-roadmap-builder.md`)

**Purpose:** Synthesize findings into actionable roadmap.

**Outputs:**
- Priority matrix (Impact/Effort/Risk)
- Quick wins (<1 day)
- Short-term items (1-2 weeks)
- Medium-term items (1 month+)
- PR specifications with acceptance criteria
- Dependency graph
- Implementation timeline

**Priority Formula:**
```
Priority = (Impact / Effort) × Risk_Adjustment
```

**Effort Levels:**
| Level | Time |
|-------|------|
| S | < 2 hours |
| M | < 1 day |
| L | < 1 week |
| XL | > 1 week |

## Usage Patterns

### Full Analysis (Recommended)

1. Run **Repo Scout** → save output
2. Run in parallel:
   - **Test Auditor** (needs Repo Scout)
   - **Debt Collector**
   - **Reliability Reviewer** (needs Repo Scout)
   - **DX Reviewer**
3. Run **Roadmap Builder** with all outputs

**Total time:** 45-75 minutes

### Targeted Analysis

| Concern | Skill |
|---------|-------|
| Test coverage issues | Test Auditor |
| Technical debt | Debt Collector |
| Production readiness | Reliability Reviewer |
| Onboarding friction | DX Reviewer |
| Planning sprint | Roadmap Builder |

## Output Format

All skills produce structured Markdown with:
- Tables for scannable findings
- `file:line` references for traceability
- Severity/priority ratings
- Actionable recommendations

## Customization

Skills can be modified for specific needs:
- Add language-specific patterns (e.g., Java, Python)
- Adjust severity thresholds
- Include/exclude sections
- Change output format for ticketing systems

## File Locations

```
docs/skills/
├── README.md                    # Overview and pipeline
├── 01-repo-scout.md             # Architecture analysis
├── 02-test-auditor.md           # Test coverage
├── 03-debt-collector.md         # Technical debt
├── 04-reliability-reviewer.md   # Reliability/security
├── 05-dx-reviewer.md            # Developer experience
└── 06-roadmap-builder.md        # Prioritized roadmap
```

## Integration with Development Workflow

The skills complement the SDD + RPI methodology:

| Phase | Relevant Skills |
|-------|-----------------|
| Research (Turn 1) | Repo Scout, Reliability Reviewer |
| Planning (Turn 2) | Roadmap Builder, DX Reviewer |
| Pre-merge | Test Auditor, Debt Collector |

## Best Practices

1. **Run Repo Scout first** - provides context for other skills
2. **Parallelize middle layer** - Test Auditor, Debt Collector, Reliability, DX can run concurrently
3. **Keep PR specs small** - <400 lines changed, reviewable in <30 minutes
4. **Track metrics** - Use Roadmap Builder's metrics table to measure progress
5. **Iterate** - Re-run skills after major changes to track improvement
