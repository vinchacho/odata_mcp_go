# Skill 6: Roadmap Builder

Synthesize all findings into a prioritized, actionable roadmap with PR-sized work items.

## When to Use

- After running other analysis skills
- Planning improvement sprints
- Creating actionable backlogs

## Inputs

- All prior skill outputs (Scout, Test, Debt, Reliability, DX)
- Optional: Team size, timeline constraints

## Prompt

```
You are synthesizing analysis findings into an actionable improvement roadmap.

## Instructions

### Aggregate Findings

1. **Collect All Issues**: From all prior analyses, list:
   - Test coverage gaps
   - Technical debt items
   - Reliability issues
   - DX problems

2. **Deduplicate**: Merge overlapping findings

3. **Categorize**: Group by type:
   - Code Quality
   - Test Coverage
   - Reliability
   - Security
   - Performance
   - Developer Experience

### Score Each Item

For each item, assess:

1. **Impact** (1-5):
   - 5: Critical for production/security
   - 4: Blocks important features
   - 3: Improves reliability/DX significantly
   - 2: Nice improvement
   - 1: Minor cleanup

2. **Effort** (S/M/L/XL):
   - S: < 2 hours
   - M: < 1 day
   - L: < 1 week
   - XL: > 1 week

3. **Risk** (Low/Med/High):
   - High: Could break things, needs careful review
   - Med: Some risk, needs testing
   - Low: Safe change

4. **Dependencies**: What must be done first?

### Prioritize

1. **Priority Formula**: Impact / Effort, adjusted for risk
2. **Group into buckets**:
   - Quick Wins: High impact, low effort
   - Strategic: High impact, high effort
   - Tactical: Medium impact, low effort
   - Backlog: Lower priority

### Create PR Specs

For each work item, define:
- PR title
- Files to modify
- Acceptance criteria
- Test plan
- Risks and mitigations

## Output Format

```markdown
## Summary

| Category | Items | Critical | High | Medium | Low |
|----------|-------|----------|------|--------|-----|
| Code Quality | | | | | |
| Test Coverage | | | | | |
| Reliability | | | | | |
| Security | | | | | |
| Performance | | | | | |
| DX | | | | | |
| **Total** | | | | | |

## Priority Matrix

| Priority | ID | Title | Category | Impact | Effort | Risk | Deps |
|----------|-----|-------|----------|--------|--------|------|------|

## Quick Wins (< 1 day total)

| ID | Task | Files | Effort | Acceptance Criteria |
|----|------|-------|--------|---------------------|

## Short-Term (1-2 weeks)

| ID | Task | Scope | Effort | PR Breakdown |
|----|------|-------|--------|--------------|

## Medium-Term (1 month+)

| ID | Task | Scope | Effort | Milestones |
|----|------|-------|--------|------------|

## PR Specifications

### PR-001: [Title]

**Category**: [category]
**Priority**: [P0/P1/P2/P3]
**Effort**: [S/M/L/XL]

**Description**:
[What and why]

**Files to Modify**:
- `path/to/file.go` — [change description]

**Acceptance Criteria**:
- [ ] [criterion 1]
- [ ] [criterion 2]

**Test Plan**:
- [ ] [test 1]
- [ ] [test 2]

**Risks**:
- [risk]: [mitigation]

---

[Repeat for each PR]

## Dependency Graph

```
PR-001 (Quick Win)
    │
    ├── PR-002 (depends on 001)
    │       │
    │       └── PR-004 (depends on 002)
    │
    └── PR-003 (depends on 001)
```

## Implementation Order

1. **Week 1**: PR-001, PR-002 (quick wins)
2. **Week 2**: PR-003, PR-004 (foundation)
3. **Week 3-4**: PR-005, PR-006 (features)

## Metrics to Track

| Metric | Current | Target | How to Measure |
|--------|---------|--------|----------------|
```

Focus on actionable, PR-sized work items. Each PR should be reviewable in < 30 minutes.
```

## Expected Output

- Complete priority matrix
- PR-ready work item specs
- Dependency graph
- Implementation timeline

## Tips

- Keep PRs small (< 400 lines changed)
- Group related changes
- Put quick wins first for momentum
- Include acceptance criteria for every PR
- Consider reviewer burden
