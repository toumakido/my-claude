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

## Notes

- Large changes: suggest split into smaller PRs
- Breaking changes: explicitly confirm with user
- Never push directly to main/master