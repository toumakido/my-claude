Analyze command usage and create improvement proposal issue

Output language: Japanese, formal business tone

## Prerequisites

- gh CLI installed and authenticated
- Must be executed in same conversation session after using target command
- Working repository: Can be any repository where the command was used
- Issue target repository: https://github.com/toumakido/my-claude (determined dynamically or confirmed with user)

## Parameters

- `$ARGUMENTS`: Target command name to improve (e.g., `migrate-aws-sdk`, `review-pr`, `improve-command`)
  - Must correspond to a file in `commands/` directory (without `.md` extension)
  - Can be used recursively to improve improve-command itself

## Usage

```
/improve-command migrate-aws-sdk
/improve-command improve-command  # Recursive usage
```

## Process

1. Validate target command:
   - Check if `commands/<command-name>.md` exists
   - Read and understand command specification
2. Analyze conversation history:
   - Identify when target command was executed
   - Extract problems encountered during/after execution
   - Identify missing patterns or guidance
   - Note additional steps required beyond command specification
   - Find workarounds or fixes applied
3. Analyze improvement opportunities from conversation history:
   - User corrections or指摘 (e.g., "周りのフォーマットに合わせて")
   - Multiple Edit rejections indicating unclear requirements
   - Workarounds or manual interventions required
   - Additional questions needed during execution

   Optional (only if needed for reference):
   - Get PR number for reference: `gh pr list --limit 5`
   - Check if repository is public: `gh repo view --json isPrivate`
   - Note: Repository must be public to include PR links in issue
4. Extract improvement opportunities:
   - Missing patterns or examples
   - Insufficient guidance or documentation
   - Additional checks needed
   - Service/library-specific knowledge gaps
   - More efficient approaches
4.5. Evaluate severity and confirm with user:
   - Severity levels:
     - Critical/Important: コマンド仕様の明確な不足、複数回の指摘 → Proceed to issue creation
     - Nice-to-have/None: 軽微な改善、1回のみの指摘、改善点なし → Use AskUserQuestion to confirm before proceeding
5. Generate structured issue content (if confirmed):
   - Title: `<command-name>.md の改善提案: [主要な改善点の要約]`
   - Body sections:
     - 概要: Brief summary of improvements
     - 実際に発生した問題: Concrete problems encountered
     - 改善提案: Specific proposals with code samples
     - 期待される効果: Expected benefits
     - 参考: Links to relevant PRs/commits (only if repository is public)
6. Create GitHub issue:
   - Determine target repository:
     - Check current repository: `gh repo view --json nameWithOwner -q .nameWithOwner`
     - If current repository is toumakido/my-claude: Create issue here
     - Otherwise: Confirm with user or default to https://github.com/toumakido/my-claude
   - Use `gh issue create --repo toumakido/my-claude` with generated content
7. Display created issue URL

## Issue Template Format

```markdown
## 概要

`commands/<command-name>.md` コマンドを実際のプロジェクトで使用した結果、いくつかの重要な改善点が判明しました。

## 実際に発生した問題

### 1. [問題カテゴリ]
[具体的な問題の説明]

### 2. [問題カテゴリ]
[具体的な問題の説明]

## 改善提案

### 1. [改善項目]
[具体的な改善内容とコードサンプル]

```markdown
[追加すべきセクションやコード例]
```

### 2. [改善項目]
[具体的な改善内容とコードサンプル]

## 期待される効果

- [効果1]
- [効果2]
- [効果3]

## 参考

(このセクションは作業対象リポジトリがpublicの場合のみ含める)

- PR: [PR URL]
- Commit: [commit hash/URL]
```

## Analysis Guidelines

### Identifying Problems
- Look for multiple commits on same topic (indicates iterative fixes)
- Check PR review comments for gaps in initial implementation
- Identify patterns where manual intervention was required
- Note deviations from command specification

### Extracting Improvements
- Compare command specification with actual implementation steps
- Identify missing error handling patterns
- Note service-specific details not covered
- Consider validation steps that could prevent issues

### Prioritizing Suggestions
- Critical: Issues that cause incorrect implementation
- Important: Missing patterns that require manual fixes
- Nice-to-have: Efficiency improvements or better documentation

## Error Handling

- Command file not found: Display error and list available commands
- No recent usage found: Ask user to confirm command was used in current session
- No git history: Warn user but continue with conversation analysis only
- gh CLI error: Display error and ask user to check authentication

## Code Example Generalization

When working repository is private, generalize code examples to avoid exposing confidential information:

1. Extract general patterns from actual code
2. Replace specific values with generic placeholders:
   - Repository names → `example-repo`
   - Variable names → descriptive generic names (`userId` → `entityId`)
   - Function names → pattern-based names (`GetActiveUsers` → `FetchEntities`)
   - Company/product-specific terms → generic terms
3. Add explanatory comments to clarify intent

Example transformation:
```go
// Actual code (private repository)
func (repo *UserRepository) GetActiveUsers(ctx context.Context, companyId string) ([]*User, error) {
    return repo.dynamoDB.QueryByCompany(ctx, companyId, "active")
}

// Generalized code (for issue)
func (repo *EntityRepository) FetchEntities(ctx context.Context, filterKey string) ([]*Entity, error) {
    return repo.store.QueryByFilter(ctx, filterKey, filterValue)
}
```

## Notes

- This command works best when executed immediately after target command usage
- Longer conversation history provides more context for analysis
- Include specific code examples in proposals for clarity
- For public repositories: Include PR/commit links as evidence
- For private repositories:
  - Omit PR/commit links from issue (not accessible to public)
  - Generalize code examples to remove confidential information
- Consider both technical gaps and documentation improvements
