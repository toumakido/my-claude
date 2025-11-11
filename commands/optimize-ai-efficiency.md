Review and optimize commands and CLAUDE.md for AI efficiency

Output language: Japanese, formal business tone

## Prerequisites

- Must run from toumakido/my-claude repository root
- This command only works for toumakido/my-claude repository
- gh CLI installed and authenticated
- Git working tree clean

## Process

1. Read and analyze all files in this repository:
   - `commands/*.md` (all command files)
   - `CLAUDE.md` (global configuration)
   - `.claude/*` (if exists)
2. Identify optimization opportunities based on check criteria
3. Apply optimizations using Edit tool
4. Create branch: `optimize/ai-efficiency-YYYYMMDD`
5. Commit with summary
6. Create PR: `gh pr create --repo toumakido/my-claude`
7. Display PR URL and summary

## Check Criteria

### 1. AI Efficiency

Eliminate ambiguity, clarify conditions, specify priorities.

Bad: Run tests if needed
Good: If changes affect core logic: run `npm test`

Bad: Update the file
Good: Update `commands/example.md` using Edit tool

Bad: Consider parallel execution
Good: Execute these commands in parallel (independent)

### 2. Redundancy

Remove duplicates and self-evident descriptions.

Bad:
```
## Prerequisites
- git installed
## Process
1. Ensure git installed
2. Run git command
```

Good:
```
## Prerequisites
- git installed
## Process
1. Run git command
```

### 3. Structure

Organize logically, clarify dependencies.

Bad:
```
1. Do A
3. Prerequisites: Install X
2. Do B
```

Good:
```
## Prerequisites
- Install X
## Process
1. Do A
2. Do B
```

### 4. Execution Efficiency

Mark parallel execution, simplify error handling, remove redundant checks.

Bad: Run A, then run B (B is independent)
Good: Run A and B in parallel (independent)

Bad: Check file exists, then read file
Good: Read file (handle error if not found)

### 5. Consistency

Unify terminology, formatting, naming.

Bad: "repository root" in file A, "repo root" in file B
Good: "repository root" consistently

## Optimization Priority

Critical: Ambiguous logic, missing information
Important: Redundancy, inconsistent terminology, inefficient patterns
Nice-to-have: Minor formatting, additional examples

## PR Format

```markdown
## Summary

### Files Modified
- commands/example.md: [changes]
- CLAUDE.md: [changes]

### Improvements
1. AI Efficiency: Clarified N conditionals, added M explicit examples
2. Redundancy: Removed N duplicates
3. Structure: Reorganized N files
4. Execution: Marked N parallel opportunities
5. Consistency: Unified terminology

### Before/After (2-3 examples)
[Examples]

### Impact
- Improved interpretation accuracy
- Reduced ambiguity
- Maintained readability
```

## Notes

- Preserve human readability
- Do not change command functionality
- Prefer explicit over implicit
- If no optimizations: report and exit
- If breaking changes: confirm with user
