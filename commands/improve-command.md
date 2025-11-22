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
   - Check if `commands/<command-name>.md` exists using Read tool
   - If not found: Display error and list available commands (see Error Handling)
2. Analyze conversation history:
   - Identify when target command was executed (search for `/command-name` in conversation)
   - Extract problems encountered during/after execution
   - Identify missing patterns or guidance
   - Note additional steps required beyond command specification
   - Find workarounds or fixes applied
3. Analyze improvement opportunities from conversation history:
   - User corrections or指摘 (e.g., "周りのフォーマットに合わせて") - indicates missing style guidelines
   - Multiple Edit rejections indicating unclear requirements - count rejections with same file
   - Workarounds or manual interventions required - user ran commands directly
   - Additional questions needed during execution - count AskUserQuestion tool calls during command execution
   - Post-command fixes by user - commits/edits after command completion

   **Check repository type** (for reference links in issue):
   - Run: `gh repo view --json isPrivate -q .isPrivate`
   - If repository is public: Collect PR/commit references for issue body using `gh pr list --limit 5`
   - If repository is private: Skip reference collection (will use generalized examples only)
4. Extract improvement opportunities:
   - Missing patterns or examples
   - Insufficient guidance or documentation
   - Additional checks needed
   - Service/library-specific knowledge gaps
   - More efficient approaches
4.5. Categorize issues by improvement scope:

   **Purpose**: Distinguish between command specification issues and implementation-phase issues

   **Categories**:

   1. **Command specification issues** (Should be included in improvement proposal):
      - Missing or unclear steps in command process
      - Insufficient guidance or examples in command documentation
      - Lack of error handling patterns in command specification
      - Missing validation steps
      - Unclear decision criteria (e.g., when to apply certain patterns)

      Examples:
      - "Command doesn't specify how to handle duplicate test data"
      - "No guidance on determining test data distribution strategy"
      - "Missing step for validating type information"

   2. **Implementation-phase issues** (Should be excluded from command specification):
      - Technology-specific breaking changes (e.g., SDK version differences)
      - Library-specific behavior patterns
      - Domain-specific business logic
      - Project-specific code patterns

      Examples:
      - "SDK version X handles empty strings differently than version Y"
      - "This API requires specific authentication headers"
      - "Business rule X requires validation Y"

   **Action**:
   - Review all identified issues and categorize them
   - Only include command specification issues in improvement proposal
   - Document excluded implementation-phase issues separately (if requested by user)

4.6. Evaluate severity and confirm with user:
   - Severity levels:
     - Critical/Important: コマンド仕様の明確な不足、複数回の指摘 → Proceed to step 5
     - Nice-to-have/None: 軽微な改善、1回のみの指摘、改善点なし → Use AskUserQuestion to confirm before proceeding to step 5
5. Generate structured issue content (if confirmed):
   - Title: `<command-name>.md の改善提案: [主要な改善点の要約]`
   - Body sections:
     - 概要: Brief summary of improvements
     - 実際に発生した問題: Concrete problems encountered
     - 改善提案: Specific proposals with code samples
     - 期待される効果: Expected benefits
     - 参考: Links to relevant PRs/commits (only if repository is public)
5.5. Present improvement proposal summary to user for confirmation:

   **Purpose**: Ensure proposal aligns with user's intent before creating issue

   **Actions**:
   1. Generate concise summary of improvement proposals (3-5 bullet points)
   2. Present to user using text output (NOT AskUserQuestion, NOT issue creation yet)
   3. Format:
      ```
      ## 改善提案サマリー

      以下の改善をissueとして提案します：

      1. [改善項目1]: [1行の説明]
      2. [改善項目2]: [1行の説明]
      3. [改善項目3]: [1行の説明]

      この内容でissueを作成してよろしいですか？
      追加・削除・修正したい項目があれば教えてください。
      ```
   4. Wait for user confirmation or feedback
   5. Adjust proposal based on feedback
   6. Proceed to Step 6 only after user approval

   **Example**:
   ```
   ## 改善提案サマリー

   以下の改善をissueとして提案します：

   1. テストデータ設計方針の明記: 重複を避ける戦略を追加
   2. フィルター検証用データ生成の必須化: match/non-matchの両方を生成
   3. 型情報検証ステップの追加: 事前検証でコンパイルエラー防止

   この内容でissueを作成してよろしいですか？
   追加・削除・修正したい項目があれば教えてください。
   ```

5.6. Apply generalization for private repositories (if applicable):

   **Purpose**: Automatically generalize code examples and identifiers when working repository is private

   **Actions**:
   1. Check if working repository is private (already done in Step 3)
   2. If private, apply generalization patterns to issue body:

      **Automatic pattern matching and replacement**:

      a. Function/method names:
      ```
      Pattern: CamelCase words ending in common suffixes
      - *Repository → EntityRepository, DataRepository
      - *Service → ProcessService, RecordService
      - *Handler → RequestHandler, EventHandler
      - *Manager → ResourceManager, StateManager
      - Get*/Fetch*/Post*/Put*/Delete* → Generic verb prefixes
      ```

      b. Repository/Gateway/Client names:
      ```
      Pattern: Variable names ending in Repository/Gateway/Client
      - [a-z]+Repository → entityRepo, dataRepo, recordRepo
      - [a-z]+Gateway → apiGateway, serviceGateway
      - [a-z]+Client → httpClient, grpcClient
      ```

      c. Method names with business domain terms:
      ```
      Pattern: Domain-specific verbs + nouns
      - Register* → RegisterEntity, RegisterItem
      - Process* → ProcessRecord, ProcessData
      - Calculate* → CalculateValue, CalculateTotal
      - Validate* → ValidateInput, ValidateData
      ```

      d. Preserve technical patterns:
      ```
      Keep as-is:
      - AWS service names: DynamoDB, S3, Lambda, etc.
      - Standard methods: GetItem, PutItem, Query, Scan
      - Error handling: error, err, context.Context, ctx
      - Common patterns: TransactWriteItems, FilterExpression
      ```

   3. Generate generalization mapping (example patterns only, not shown to user):
      ```
      Original → Generalized (maintain consistency)
      - UserAccountRepository → EntityRepository
      - FetchCustomerDetails → GetEntityDetails
      - paymentGatewayClient → externalServiceClient
      - calculateMonthlyFee → calculateValue
      ```

   4. Apply replacements to issue body using regex patterns
   5. Maintain consistency: same term → same generic name throughout issue body
   6. Track generalization count (functions, repositories, domain terms) for Step 5.7

   **Implementation guideline**:
   - Use regex patterns to detect domain-specific terms
   - Preserve code structure and technical accuracy
   - Do not generalize AWS service names or standard library functions
   - Refer to "Code Example Generalization" section (lines 296-356) for detailed guidelines:
     - **Level 1 (MUST generalize)**: Function names, variable names, type names, table names
     - **Level 2 (SHOULD generalize)**: Business terms, service names, company-specific terms
     - **Level 3 (MAY keep specific)**: FilterExpression structure, query patterns, AWS service names
     - **Level 4 (MUST keep specific)**: Language keywords, standard library, common patterns

5.7. Confirm generalized content with user (for private repositories):

   **Purpose**: Ensure generalization is appropriate before creating public issue

   **Actions**:
   1. Display generalization summary:
      ```
      ## 一般化サマリー

      privateリポジトリのため、以下の情報を一般化しました:

      関数名: N箇所
      Repository名: M箇所
      ドメイン用語: K箇所

      一般化後のissue本文を確認しますか？
      ```

   2. Use AskUserQuestion:
      - question: "一般化されたissue内容を確認しますか？"
      - header: "Generalization"
      - multiSelect: false
      - options:
        - label: "確認する", description: "一般化後の全文を表示"
        - label: "このまま作成", description: "一般化を信頼してissue作成"
        - label: "キャンセル", description: "issue作成を中止"

   3. Handle response:
      - "確認する": Display generalized issue body in full, then ask "この内容でissueを作成してよろしいですか？" and wait for user approval
      - "このまま作成": Proceed to Step 6
      - "キャンセル": Exit with message "Issue作成をキャンセルしました"

   **Example output**:
   ```
   ## 一般化サマリー

   privateリポジトリのため、以下の情報を一般化しました:

   関数名: 8箇所 (CreateEntity, UpdateEntity, etc.)
   Repository名: 5箇所 (entityRepository, dataRepository, etc.)
   ドメイン用語: 3箇所 (generic business terms)

   一般化後のissue本文を確認しますか？
   ```

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

When working repository is private, generalize code examples to avoid exposing confidential information.

### Generalization Levels

Apply generalization from most specific to most generic:

**Level 1: Identifiers (MUST generalize)**
- Function/handler names: Use generic names like `handlerA`, `processEntity`, `fetchData`
- Variable names: Use generic names like `entityId`, `recordKey`, `value`
- Type names: Use generic names like `Entity`, `Record`, `Item`
- Database table names: Use generic names like `EntityTable`, `RecordStore`

**Level 2: Domain concepts (SHOULD generalize)**
- Business-specific terms: Replace with generic equivalents
- Product/service names: Use placeholders like `ServiceA`, `ComponentX`
- Company-specific terminology: Remove or replace with generic terms

**Level 3: Technical patterns (MAY keep specific)**
- FilterExpression structure: Can show actual syntax patterns
- Query patterns: Can show structure without specific field names
- AWS service names: DynamoDB, S3, Lambda (public knowledge)
- Standard error patterns: Can show generic error handling

**Level 4: Generic constructs (MUST keep specific)**
- Language keywords: Keep as-is
- Standard library functions: Keep as-is
- Common patterns: `ctx context.Context`, `err error`

### Generalization Guidelines

1. **Avoid concrete examples**: Don't include specific struct definitions, actual field names, or real parameter values
2. **Use abstract descriptions**: Describe patterns in prose rather than showing actual code
3. **Focus on structure**: Show FilterExpression patterns, not actual field names
4. **When in doubt, ask**: Use AskUserQuestion to confirm if specific terms should be generalized

**Good example** (generalized):
```
FilterExpression with multiple conditions using OR and AND operators,
checking for attribute existence and type
```

**Bad example** (too specific):
```go
FilterExpression: "(attribute_not_exists(#OpStatus) OR attribute_type(#OpStatus, :null)) AND (#RequestType = :rt)"
```

### Example Transformation

```go
// Actual code (private repository) - DO NOT include this in issue
func (repo *UserRepository) GetActiveUsers(ctx context.Context, companyId string) ([]*User, error) {
    return repo.dynamoDB.QueryByCompany(ctx, companyId, "active")
}

// Generalized code (for issue) - Use this level of abstraction
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
