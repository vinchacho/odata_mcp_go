# Skill 5: DX Reviewer

Identify developer experience issues that slow down development and onboarding.

## When to Use

- Improving team productivity
- Onboarding new developers
- Reducing development friction

## Inputs

- Repository path

## Prompt

```
You are reviewing developer experience (DX) in a codebase.

## Instructions

### Logging

1. **Inconsistent Patterns**: Find:
   - Mix of fmt.Printf, log.*, custom loggers
   - Different log formats
   - Inconsistent log levels

2. **Wrong Output Stream**: Find:
   - Debug output to stdout (should be stderr)
   - Logs mixed with program output

3. **Missing Log Levels**: Find:
   - No way to enable/disable debug logging
   - Missing verbose mode
   - No log level configuration

### Configuration

1. **Hardcoded Values**: Find:
   - Magic numbers
   - Hardcoded URLs, ports, timeouts
   - Values that should be configurable

2. **Missing Env Vars**: Find:
   - Values that should come from environment
   - Secrets without env var option
   - Missing .env.example

3. **Undocumented Flags**: Find:
   - CLI flags without help text
   - Flags not in README
   - Hidden configuration options

### Documentation

1. **Missing Docs**: Check for:
   - README completeness
   - API documentation
   - Architecture docs
   - Contributing guide

2. **Outdated Docs**: Find:
   - Docs that don't match code
   - Deprecated instructions
   - Missing new features

3. **Missing Examples**: Check for:
   - Example directory
   - Usage examples in README
   - Sample configurations

### Tooling

1. **Linting**: Check for:
   - .golangci.yml / .eslintrc / etc.
   - Lint errors when run
   - Pre-commit hooks

2. **Formatting**: Check for:
   - Formatter configuration
   - Format on save setup
   - CI formatting check

3. **CI/CD**: Check for:
   - Build automation
   - Test automation
   - Release automation

### Onboarding

1. **Time to First Build**: Estimate:
   - Steps from clone to build
   - Required tool installations
   - Configuration needed

2. **Common Gotchas**: Identify:
   - Surprising behaviors
   - Non-obvious requirements
   - Undocumented dependencies

## Output Format

```markdown
## DX Score

| Area | Score | Notes |
|------|-------|-------|
| Logging | /5 | |
| Configuration | /5 | |
| Documentation | /5 | |
| Tooling | /5 | |
| Onboarding | /5 | |
| **Overall** | /25 | |

## Logging Issues

| Issue | File:Line | Impact | Fix |
|-------|-----------|--------|-----|

## Configuration Issues

### Hardcoded Values
| Value | File:Line | Should Be |
|-------|-----------|-----------|

### Missing Env Vars
| Config | Current | Recommended Env Var |
|--------|---------|---------------------|

### Undocumented Flags
| Flag | File | Missing |
|------|------|---------|

## Documentation Gaps

| Gap | Location | Priority | Action |
|-----|----------|----------|--------|

## Tooling Gaps

| Tool | Status | Recommendation |
|------|--------|----------------|
| Linting | | |
| Formatting | | |
| CI/CD | | |
| Pre-commit | | |

## Onboarding Friction

### Steps to First Build
1. [step]
2. [step]

### Required Tools
| Tool | Version | Installation |
|------|---------|--------------|

### Gotchas
| Issue | Where | Workaround |
|-------|-------|------------|

## Recommendations

### Quick Wins
1. [improvement]

### High Impact
1. [improvement]

### Nice to Have
1. [improvement]
```

Score: 5 = Excellent, 4 = Good, 3 = Adequate, 2 = Needs Work, 1 = Poor.
```

## Expected Output

- DX score card
- Specific issues with fixes
- Onboarding friction points
- Prioritized improvements

## Tips

- Try building from scratch to find friction
- Check for .github/CONTRIBUTING.md
- Look for Makefile help target
- Check if tests run cleanly out of the box
