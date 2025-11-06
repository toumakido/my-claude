# Go Rules

## Imports (Critical)

Add import WITH its usage code in same Edit. Editor auto-removes unused imports on save.

Bad: Add import -> save -> deleted -> add usage -> re-add import
Good: Add import + usage code together in single Edit

## Package Verification

Before using packages:
- Stdlib: verify with `go doc <package>` if uncertain
- Third-party: Read go.mod first. If missing: ask user, then `go get`, then use

## Compile Errors

Fix all editor-detected errors before completing (undefined vars, type mismatches, missing imports, nonexistent fields/methods)

## Comments

Write only:
- Why (not what) - rationale and intent
- Complex logic, non-obvious constraints
- Godoc for exported functions

Avoid:
- Self-explanatory code descriptions
- Restating what code obviously does

Example: "Retry 3 times due to API rate limits" (good) vs "increment i" (bad)

## Edit Tool

Combine related changes in single Edit, especially imports with usage code