Analyze and fix GitHub issue $ARGUMENTS

Output language: Japanese, formal business tone

## Prerequisites

- gh CLI installed and authenticated
- $ARGUMENTS: issue number (123) or URL
- Run from repo root

## Process

1. Fetch issue: `gh issue view $ARGUMENTS`
2. Create TodoWrite plan: issue review, analysis, find files, implement, test, commit, create PR
3. Analyze problem, confirm understanding with user via AskUserQuestion, propose fix approach
4. Create branch: `git checkout -b fix/issue-$ISSUE_NUMBER`
5. Find related files using Task tool (subagent_type=Explore)
6. Implement fix with Edit tool, avoid security issues (XSS, SQL injection), mark todos complete

   **Before editing any file:**
   - Read the target file (or similar files in same directory) to understand:
     - Language (Japanese/English/Mixed)
     - Code comment style
     - Formatting conventions (indentation, line length, etc.)
     - Section structure and naming
   - Match the existing style in your edits

   **When using Edit/Write tools:**
   - Include brief explanation of changes:
     - What is being changed/added
     - Why this change addresses the issue
     - Confirmation that existing style is matched
   - For format-sensitive files (markdown, config), verify consistency
7. Run tests (unit, integration, e2e), add new tests if needed
8. Commit with format:
```
fix: <brief> (#$ISSUE_NUMBER)

- details
- scope

Fixes #$ISSUE_NUMBER
```
9. Create PR with `gh pr create`, include: problem summary, fix details, test method, screenshots (if UI), Fixes #$ISSUE_NUMBER
10. Output only: `https://github.com/user/repo/pull/123`

## Error Handling

- No gh CLI: guide installation
- No issue: verify number
- Test failure: analyze and continue fixing
- PR creation failure: guide manual creation

## File Edit Guidelines

For ALL file edits (new creation or modification):

1. **Understand existing style:**
   - New file in existing directory: Read 2-3 similar files for reference
   - Existing file modification: Read target file to understand style
   - Pay attention to: language, comments, formatting, structure

2. **Match the style:**
   - If file/directory uses English comments → use English
   - If file/directory uses Japanese comments → use Japanese
   - If file uses specific formatting → follow same format
   - If file has particular section structure → maintain consistency

3. **Exceptions:**
   - Only deviate from existing style if issue explicitly requires it
   - Explain deviation reason to user if necessary

## Examples

### Good: Matching existing style
```go
// Existing file has English comments
func foo() error {
    // existing comment
}

// Your addition - also in English
func bar() error {
    // process items
}
```

### Bad: Not matching existing style
```go
// Existing file has English comments
func foo() error {
    // existing comment
}

// Your addition - in Japanese (inconsistent!)
func bar() error {
    // アイテムを処理
}
```

## Notes

- Large changes: suggest split into smaller PRs
- Breaking changes: explicitly confirm with user
- Never push directly to main/master