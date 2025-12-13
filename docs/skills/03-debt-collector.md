# Skill 3: Debt Collector

Harvest TODO/FIXME comments, incomplete implementations, and deprecated patterns.

## When to Use

- Planning technical debt sprints
- Understanding codebase health
- Finding quick wins

## Inputs

- Repository path

## Prompt

```
You are collecting technical debt markers from a codebase.

## Instructions

1. **TODO/FIXME Search**: Grep for:
   - TODO, FIXME, HACK, XXX, TEMP, DEPRECATED
   - BUG, OPTIMIZE, REFACTOR
   - "not implemented", "stub", "placeholder", "workaround"

2. **Incomplete Implementations**: Find:
   - Functions that return placeholder values
   - Empty method bodies
   - panic("not implemented") / throw new NotImplementedError()
   - Functions with only TODO comments

3. **Deprecated Patterns**: Identify:
   - Deprecated stdlib imports (e.g., io/ioutil in Go)
   - Deprecated APIs or methods
   - Legacy patterns that should be modernized

4. **Theme Clustering**: Group findings by theme:
   - Missing features
   - Performance improvements
   - Error handling gaps
   - Refactoring needs
   - Documentation gaps

5. **Quick Wins**: Highlight items fixable in <30 minutes

## Output Format

```markdown
## Summary

| Type | Count |
|------|-------|
| TODO | |
| FIXME | |
| HACK | |
| Incomplete | |
| Deprecated | |
| **Total** | |

## TODO/FIXME Items

### By Theme

#### Missing Features
| ID | File:Line | Description | Effort |
|----|-----------|-------------|--------|

#### Error Handling
| ID | File:Line | Description | Effort |
|----|-----------|-------------|--------|

#### Performance
| ID | File:Line | Description | Effort |
|----|-----------|-------------|--------|

#### Refactoring
| ID | File:Line | Description | Effort |
|----|-----------|-------------|--------|

#### Documentation
| ID | File:Line | Description | Effort |
|----|-----------|-------------|--------|

## Incomplete Implementations

| ID | Pattern | File:Line | Evidence |
|----|---------|-----------|----------|

## Deprecated Patterns

| ID | Pattern | File:Line | Replacement |
|----|---------|-----------|-------------|

## Quick Wins (<30 min)

| ID | Item | File:Line | Fix |
|----|------|-----------|-----|

## Recommendations

### Address Now
- [items that block other work or cause bugs]

### Address Soon
- [items that accumulate interest]

### Address Eventually
- [nice-to-have cleanups]
```

Include exact text of TODO comments. Effort estimates: S (< 30 min), M (< 2 hours), L (< 1 day), XL (> 1 day).
```

## Expected Output

- Complete inventory of debt markers
- Themed groupings for planning
- Quick wins highlighted
- Effort estimates

## Tips

- Search case-insensitively
- Check git blame for TODO age
- Look in comments AND code (some patterns are in code)
- Check for TODO in test files too
