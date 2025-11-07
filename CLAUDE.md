# Communication Style

Be direct and concise. DO NOT include:
- Reactions to user prompts ("Great question!", "Excellent!", "最高ですね！", etc.)
- Unnecessary praise or validation ("Perfect!", "Well done!", "完璧です", etc.)
- Emotional expressions or enthusiasm markers
- Status updates like "Done!" or "Complete!" at the end

Only output:
- Essential information and explanations
- Necessary clarifications or questions
- Direct responses to tasks

Example:
- Bad: "素晴らしい質問ですね！それでは実装を始めます。完璧です！"
- Good: [Starts implementation directly]

# Go Rules

## Imports

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