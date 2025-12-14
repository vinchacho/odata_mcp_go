# Development Workflow: SDD + RPI Methodology

*This document defines the mandatory development workflow for all changes to this project.*

---

## Overview

All development follows a **strict hybrid workflow**:
- **SDD** (Spec-Driven Development)
- **RPI** (Research → Plan → Implement)

This ensures quality, traceability, and prevents scope creep.

---

## Operating Principles

1. **Work in explicit phases** with human approval gates
2. **No coding until the Implement phase** (no code blocks, no diffs)
3. **Stop at gates** for approval before proceeding
4. **Never guess silently** - ask for missing inputs or mark as ASSUMPTION with risk
5. **Every change maps to a requirement and acceptance criterion**
6. **Prefer small, testable, PR-sized increments**

---

## Complexity Modes

Choose one mode at the start of each task:

| Mode | Criteria | Documentation Level |
|------|----------|---------------------|
| **LITE** | ≤2 files, no schema changes, minimal tests | Abbreviated |
| **STANDARD** | Multiple modules, tests required | Full |
| **HEAVY** | Cross-cutting changes, migrations, performance/security critical | Full + extra review |

---

## The Three Turns

### Turn 1: ARCHITECT (SDD + Research)

**Purpose:** Understand the problem, define requirements, research the codebase.

**Required Output:**

#### Section 0: Intake
- Restate the request
- Goals / Non-goals
- Constraints
- Open questions (only if blocking; otherwise assumptions)
- **Complexity Check:** Declare mode (LITE / STANDARD / HEAVY)

#### Section 1: SPEC (SDD)
- **Functional Requirements (FR-1, FR-2...):** Numbered, testable
- **Non-Functional Requirements (NFR-1...):** Only if applicable
- **Edge cases & failure modes**
- **Acceptance Criteria (AC-1, AC-2...):** Objectively verifiable
- **Out of scope**

#### Section 2: RESEARCH (RPI)
- What you can infer from provided info
- What files you need to inspect (paths/patterns)
- What repo conventions you must confirm
- Risks/constraints discovered

#### Devil's Advocate
- Potential incorrect assumptions
- Missing edge cases
- Mismatched repo conventions
- Risks of the proposed approach

**🛑 STOP** - Ask: "Does this Spec + Research align with your understanding?"

---

### Turn 2: PLANNER (Plan only)

**Purpose:** Design the implementation approach without writing code.

**Required Output:**

#### Section 3: PLAN (RPI)
- **Approach:** Brief description
- **Atomic task checklist**, each with:
  - What to change (file-level)
  - Why (maps to FR-x / AC-x)
  - How to validate (tests/commands)
- **Test plan:** Unit/integration/negative cases as applicable
- **Rollout/migration plan:** If applicable
- **Plan risks & mitigations**

#### Devil's Advocate
- What could be wrong with this plan?
- Alternative approaches not considered
- Potential integration issues

**🛑 STOP** - Ask: "Shall I proceed with Implementation?"

---

### Turn 3: BUILDER (Implement only)

**Purpose:** Execute the approved plan with full traceability.

**Required Output:**

#### Section 4: IMPLEMENT (RPI)
- **Code changes:** Unified diff OR file-by-file blocks with filenames
- **Tests:** As planned
- **Validation commands:** Exact commands to run and expected outcomes
- **Traceability Matrix:**

| Requirement | File(s) Changed | Test(s) |
|-------------|-----------------|---------|
| FR-1 / AC-1 | path/to/file.go | TestXxx |
| FR-2 / AC-2 | path/to/other.go | TestYyy |

- **Notes for reviewer:** Tradeoffs, follow-ups, known limitations

**Constraints:**
- Follow repo conventions
- No unrelated refactors
- No reformatting of unchanged code

---

## Brownfield Rules

When working in existing codebases:

1. **Avoid unrelated refactors/renames/reformatting**
2. **Match existing patterns** - check similar files first
3. **Preserve existing test structure**
4. **Don't change files outside the task scope**
5. **If refactoring is needed**, make it a separate PR

---

## Gate Checklist

Before approving each gate, verify:

### After Turn 1 (Spec + Research)
- [ ] Requirements are clear and testable
- [ ] Scope is appropriate (not too broad)
- [ ] Edge cases identified
- [ ] No blocking open questions remain

### After Turn 2 (Plan)
- [ ] Plan addresses all requirements
- [ ] Changes are minimal and focused
- [ ] Test plan covers happy path and edge cases
- [ ] No unrelated changes included

### After Turn 3 (Implement)
- [ ] Code matches the plan
- [ ] Tests pass
- [ ] Traceability matrix is complete
- [ ] No scope creep

---

## Example Workflow

```
Human: "Add retry logic to the HTTP client"

Claude (Turn 1):
  - Section 0: Intake + Complexity Check → STANDARD
  - Section 1: SPEC with FR-1, FR-2, AC-1, AC-2
  - Section 2: RESEARCH findings
  - Devil's Advocate notes
  - STOP: "Does this Spec + Research align with your understanding?"