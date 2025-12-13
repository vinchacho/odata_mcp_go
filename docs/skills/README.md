# Analysis Skills Suite

Reusable prompts for systematic codebase analysis with Claude.

## Overview

This suite contains 6 structured analysis skills that can be run sequentially or in parallel to assess a codebase for quality, reliability, and improvement opportunities.

## Skills

| # | Skill | Purpose | Time |
|---|-------|---------|------|
| 1 | [Repo Scout](01-repo-scout.md) | Architecture + how-to-run | 5-10 min |
| 2 | [Test Auditor](02-test-auditor.md) | Coverage gaps + test quality | 10-15 min |
| 3 | [Debt Collector](03-debt-collector.md) | TODOs + incomplete features | 5-10 min |
| 4 | [Reliability Reviewer](04-reliability-reviewer.md) | Errors + concurrency + security | 15-20 min |
| 5 | [DX Reviewer](05-dx-reviewer.md) | Logging + config + tooling | 10-15 min |
| 6 | [Roadmap Builder](06-roadmap-builder.md) | Prioritization + PR slicing | 10-15 min |

## Pipeline

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

## Usage

### Full Analysis

Run skills in order, passing outputs as context:

```
1. Run Repo Scout → save output
2. Run Test Auditor, Debt Collector, Reliability Reviewer, DX Reviewer (parallel)
3. Run Roadmap Builder with all outputs
```

### Targeted Analysis

Run individual skills for specific concerns:
- **Coverage issues?** → Test Auditor
- **Technical debt?** → Debt Collector
- **Production readiness?** → Reliability Reviewer
- **Onboarding friction?** → DX Reviewer

## Output Format

Each skill produces structured Markdown with:
- Tables for scannable findings
- File:line references for traceability
- Severity/priority ratings
- Actionable recommendations

## Customization

Skills can be modified for specific needs:
- Add language-specific patterns
- Adjust severity thresholds
- Include/exclude sections
- Change output format

## Coverage

The suite covers these minimum capabilities:
- Repo reconnaissance + "how to run"
- Test gap analysis
- TODO/incomplete feature harvesting
- Enhancement opportunities (reliability/perf/security/DX)
- Roadmap / prioritization with PR-sized slicing
