# Skill 4: Reliability Reviewer

Identify reliability, concurrency, and security issues that could cause production failures.

## When to Use

- Pre-production readiness review
- Investigating stability issues
- Security assessment

## Inputs

- Repository path
- Repo Scout output (for architecture context)

## Prompt

```
You are reviewing a codebase for reliability, concurrency, and security issues.

## Instructions

### Error Handling

1. **Swallowed Errors**: Find patterns like:
   - `_, _ := function()` (Go)
   - `err != nil { return }` without logging
   - Empty catch blocks
   - Ignoring returned errors

2. **Missing Error Checks**: Find:
   - Unchecked return values
   - Missing nil checks before dereference
   - Missing bounds checks

3. **Generic Errors**: Find:
   - `return errors.New("error")` without context
   - Logging without including error details

### Retry & Timeout

1. **Missing Retries**: Find operations that should retry:
   - Network calls without retry
   - Database operations
   - External API calls

2. **Hardcoded Timeouts**: Find:
   - Magic numbers for timeouts
   - Timeouts that should be configurable
   - Missing timeouts entirely

3. **No Backoff**: Find retry loops without:
   - Exponential backoff
   - Jitter
   - Max retry limits

### Concurrency (if applicable)

1. **Race Conditions**: Find:
   - Shared state without synchronization
   - Check-then-act patterns
   - Non-atomic operations on shared data

2. **Goroutine/Thread Leaks**: Find:
   - `go func()` without cancellation
   - Missing context propagation
   - Unbounded goroutine creation

3. **Channel Issues**: Find:
   - Unbuffered channels that could block
   - Missing channel closes
   - Select without default or timeout

### Security

1. **Credential Exposure**: Find:
   - Logging passwords/tokens
   - Credentials in error messages
   - Hardcoded secrets

2. **Missing Auth Checks**: Find:
   - Endpoints without authentication
   - Missing authorization checks
   - Privilege escalation paths

3. **Injection Risks**: Find:
   - String concatenation for queries
   - Unsanitized user input
   - Command injection patterns

## Output Format

```markdown
## Summary

| Category | Critical | High | Medium | Low |
|----------|----------|------|--------|-----|
| Error Handling | | | | |
| Retry/Timeout | | | | |
| Concurrency | | | | |
| Security | | | | |

## Error Handling Issues

### Swallowed Errors
| ID | File:Line | Pattern | Severity | Fix |
|----|-----------|---------|----------|-----|

### Missing Error Checks
| ID | File:Line | Issue | Severity | Fix |
|----|-----------|-------|----------|-----|

### Generic Errors
| ID | File:Line | Current | Suggested |
|----|-----------|---------|-----------|

## Retry & Timeout Issues

### Missing Retries
| ID | File:Line | Operation | Impact | Recommendation |
|----|-----------|-----------|--------|----------------|

### Timeout Issues
| ID | File:Line | Issue | Current | Recommended |
|----|-----------|-------|---------|-------------|

## Concurrency Issues

### Race Conditions
| ID | File:Line | Pattern | Risk | Fix |
|----|-----------|---------|------|-----|

### Goroutine/Thread Leaks
| ID | File:Line | Pattern | Risk | Fix |
|----|-----------|---------|------|-----|

### Channel Issues
| ID | File:Line | Issue | Risk | Fix |
|----|-----------|-------|------|-----|

## Security Issues

### Credential Exposure
| ID | File:Line | Issue | Severity | Fix |
|----|-----------|-------|----------|-----|

### Auth Issues
| ID | File:Line | Issue | Severity | Fix |
|----|-----------|-------|----------|-----|

### Injection Risks
| ID | File:Line | Pattern | Severity | Fix |
|----|-----------|---------|----------|-----|

## Recommendations

### Critical (Fix Immediately)
1. [issue with impact]

### High (Fix This Sprint)
1. [issue with impact]

### Medium (Plan to Fix)
1. [issue with impact]

### Low (Monitor)
1. [issue]
```

Severity: Critical (security breach, data loss), High (service outage), Medium (degraded service), Low (minor impact).
```

## Expected Output

- Categorized reliability issues
- Severity ratings with justification
- Specific fix recommendations
- Priority ordering

## Tips

- Use `grep -n "_, _ :="` for swallowed errors in Go
- Check for `go func()` without context
- Look for password/secret/token in log statements
- Check HTTP handlers for auth middleware
