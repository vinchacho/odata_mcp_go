# Skill 2: Test Auditor

Identify test coverage gaps, test quality issues, and testing anti-patterns.

## When to Use

- Assessing test health before major changes
- Prioritizing test improvements
- Understanding what's actually tested

## Inputs

- Repository path
- Repo Scout output (for file index)

## Prompt

```
You are auditing the test coverage and quality of a codebase.

## Instructions

1. **Find All Tests**: Glob for test files (*_test.go, *.test.js, *_test.py, etc.)

2. **Coverage Analysis**: For each package/module:
   - List implementation files
   - List corresponding test files
   - Identify functions/methods WITHOUT tests
   - Estimate coverage status (None, Partial, Good)

3. **Test Quality**: Look for:
   - Happy-path only tests (no error cases)
   - Missing edge cases (empty inputs, nulls, boundaries)
   - Skipped tests (t.Skip, @skip, pytest.skip)
   - Flaky indicators (time.Sleep, fixed delays, race-prone patterns)

4. **Anti-Patterns**: Identify:
   - Network calls in unit tests
   - Time-dependent tests
   - Shared mutable state between tests
   - Tests that don't assert anything
   - Overly broad assertions

5. **Test Types**: Categorize tests as:
   - Unit (isolated, fast)
   - Integration (multiple components)
   - E2E (full system)

## Output Format

```markdown
## Coverage Matrix

| Package | Impl Files | Test Files | Coverage | Status |
|---------|------------|------------|----------|--------|

## Untested Functions

### Critical (High Risk)
| Function | File:Line | Why Critical |
|----------|-----------|--------------|

### Important (Medium Risk)
| Function | File:Line | Reason |
|----------|-----------|--------|

### Lower Priority
| Function | File:Line | Notes |
|----------|-----------|-------|

## Test Quality Issues

### Missing Negative Tests
| Test | File:Line | Missing Cases |
|------|-----------|---------------|

### Missing Edge Cases
| Test | File:Line | Suggested Cases |
|------|-----------|-----------------|

### Skipped Tests
| Test | File:Line | Reason |
|------|-----------|--------|

## Test Anti-Patterns

### Network Calls in Unit Tests
| Test | File:Line | Impact |
|------|-----------|--------|

### Time-Dependent Tests
| Test | File:Line | Issue |
|------|-----------|-------|

### Shared State
| Test | File:Line | Risk |
|------|-----------|------|

## Test Type Distribution

| Type | Count | % |
|------|-------|---|
| Unit | | |
| Integration | | |
| E2E | | |

## Recommendations

### Priority 1 (Do First)
1. [recommendation]

### Priority 2 (Do Soon)
1. [recommendation]

### Priority 3 (Nice to Have)
1. [recommendation]
```

Be specific with file:line references. Focus on tests that matter most (core logic, error handling, edge cases).
```

## Expected Output

- Complete coverage matrix by package
- Prioritized list of untested functions
- Quality issues with evidence
- Actionable recommendations

## Tips

- Use `go test -cover` or equivalent for initial coverage data
- Check for t.Helper() usage (test quality indicator)
- Look for table-driven tests (good pattern)
- Check CI for test commands and coverage thresholds
