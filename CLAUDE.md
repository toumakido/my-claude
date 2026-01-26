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

## Package Verification

Before using packages:
- Stdlib: verify with `go doc <package>` if uncertain
- Third-party: Read go.mod first. If missing: ask user, then `go get`, then use

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

# Git Commit

DO NOT include these in commit messages:
- "🤖 Generated with [Claude Code](https://claude.com/claude-code)"
- "Co-Authored-By: Claude <noreply@anthropic.com>"

Commit format:
```
<type>: <brief> (#<issue_number>)

- details
- scope

Fixes #<issue_number>
```

# Skills

## Global Defaults

All skills in this project assume:
- Output language: Japanese, formal business tone
- Prerequisites: gh CLI installed and authenticated