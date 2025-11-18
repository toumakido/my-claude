Interactively analyze AWS SDK migration PR function by function: $ARGUMENTS

Output language: Japanese, formal business tone

## Prerequisites

- gh CLI installed and authenticated
- fzf installed for interactive selection
- $ARGUMENTS: PR number
- Run from repository root
- PR must contain AWS SDK Go migration changes

## Process

### Phase 1: Extract Functions

1. **Fetch and validate PR**
   - Run: `gh pr diff $ARGUMENTS`
   - Check if diff contains `github.com/aws/aws-sdk-go` imports/changes
   - If not AWS SDK related, output: "このPRはAWS SDK Go関連の変更を含んでいません" and stop

2. **Extract function list with Task tool** (subagent_type=general-purpose)
   - Identify all functions/methods that use AWS SDK v2 APIs
   - For each function, extract:
     - File path and line number
     - Function/method name
     - AWS service type (DynamoDB, S3, SES, etc.)
     - Brief AWS operation description (PutItem, GetObject, etc.)

3. **Format function list for fzf**
   Create entries in format:
   ```
   <file_path>:<line_number> | <function_name> | <AWS_Service> <Operation>
   ```
   Example:
   ```
   internal/repository/user.go:45 | (*UserRepository).Save | DynamoDB PutItem
   internal/gateway/s3.go:120 | (*S3Gateway).Upload | S3 PutObject
   ```

   Output initial summary:
   ```
   === AWS SDK v2を使用している関数一覧 ===

   検出された関数数: N個

   [function list]
   ```

### Phase 2: Interactive Selection Loop

4. **Present selection UI**
   ```bash
   echo "関数を選択してください (Ctrl-C で終了):"
   echo "<formatted_list>" | fzf --prompt="関数を選択> " --height=40% --layout=reverse --border
   ```

5. **Handle selection**
   - If user cancels (Ctrl-C): exit with "終了しました"
   - If function selected: extract file path and function name
   - Proceed to Phase 3

### Phase 3: Detailed Analysis for Selected Function

6. **Analyze selected function with Task tool** (subagent_type=general-purpose)
   For the selected function:
   - Find entry point (main.go, Lambda handler, HTTP handler)
   - Trace complete call chain from entry point to AWS SDK call
   - Identify intermediate layers (service, usecase, repository, gateway)
   - Extract AWS connection settings (region, endpoint, resource names)
   - Document v1 → v2 migration patterns

7. **Identify data source access for mocking**
   For the selected function:
   - Scan function body from start to AWS SDK call
   - Identify all data source access operations:
     - Repository/gateway method calls
     - Database queries (SQL, etc.)
     - External API calls
     - File system reads
   - Extract return type for each data source access
   - Note variable names used to store retrieved data

8. **Generate test data and code modifications**
   For each identified data source access:
   - Create test data matching return type structure
   - Generate code modification showing:
     - Original data source call (commented out)
     - Test data assignment
     - Preserved downstream logic
   - For Get/Delete AWS operations: include pre-insert setup code

9. **Output detailed report**
   Generate report for selected function with:
   - File path and function name
   - Complete call chain
   - AWS service and resource details
   - Migration changes summary
   - Test-ready code modifications
   - AWS console verification steps

### Phase 4: Loop Back

10. **Prompt for next action**
    After displaying report, ask:
    ```
    別の関数を確認しますか？ (y/n):
    ```
    - If 'y': return to step 4 (Phase 2)
    - If 'n': exit with "検証情報の出力を完了しました"

## Output Format

### Initial Function List (Phase 1)
```
=== AWS SDK v2を使用している関数一覧 ===

検出された関数数: N個

internal/repository/user.go:45 | (*UserRepository).Save | DynamoDB PutItem
internal/repository/user.go:89 | (*UserRepository).Get | DynamoDB GetItem
internal/gateway/s3.go:120 | (*S3Gateway).Upload | S3 PutObject
...
```

### Detailed Report for Selected Function (Phase 3)
```markdown
## 選択した関数の接続検証情報

### ファイル: [file_path:line_number]
#### 関数/メソッド: [function_name]

**呼び出しチェーン**:
```
[entry_point] (例: cmd/main.go:main())
  → [usecase/service_layer] (例: internal/usecase/user.go:(*UserUsecase).MigrateUser())
  → [repository/gateway_layer] (例: internal/repository/user.go:(*UserRepository).FetchByID())
  → AWS SDK v2 API呼び出し (例: DynamoDB PutItem)
```

**使用サービス**: [AWS Service Name (DynamoDB, S3, SES, etc.)]

**AWS接続先情報**:
- リージョン: [region or "デフォルト設定"]
- リソース名: [table name / bucket name / queue URL / etc.]
- エンドポイント: [カスタムエンドポイント or "デフォルト"]

**v1 → v2 変更内容**:
- クライアント初期化: `[v1 code]` → `[v2 code]`
- API呼び出し: `[v1 method]` → `[v2 method]`
- コンテキスト: [context propagation changes]
- 型変更: [type changes if any]

**データソースのモック方法**:
<details>
<summary>元のコード</summary>

```go
// Original data source access code
```
</details>

<details>
<summary>テスト用コード</summary>

```go
// Test data assignment code
```
</details>

**動作確認観点**:
- AWSコンソール: [確認するサービス/リソース]
- CloudWatchログ: [確認すべきAPIコール]
- 設定確認: [region/endpoint/認証情報など]
```

## Analysis Guidelines

### Function Extraction (Phase 1)
- Search for these patterns in PR diff:
  - `context.Context` parameter with AWS SDK v2 client calls
  - Package imports: `github.com/aws/aws-sdk-go-v2/service/*`
  - Client method calls: `client.PutItem`, `client.GetObject`, etc.
- Group by file and extract function signatures
- Identify AWS service from import path or client type
- Format for easy scanning in fzf selection

### Single Function Analysis (Phase 3)
- Focus on production AWS connections (exclude localhost/test endpoints)
- Trace complete call chain from entry point to AWS SDK v2 call
- Extract AWS resource names (table names, bucket names, queue URLs)
- Identify region configuration (explicit or AWS_REGION env)
- Summarize v1 → v2 migration patterns clearly
- Provide actionable AWS console verification steps

### Data Source Identification (Phase 3)
- For selected function only, look for these patterns BEFORE AWS SDK calls:
  - `repo.Method()` / `gateway.Method()` calls
  - Direct database queries (`db.Query`, `db.Exec`)
  - HTTP client calls (`client.Get`, `http.Do`)
  - File reads (`os.ReadFile`, `ioutil.ReadFile`)
  - Cache access (`cache.Get`)
- Extract function signatures from PR diff or declaration
- Note variable names that store retrieved data

### Test Data Generation (Phase 3)
- Match Go types exactly:
  - Structs: `&StructName{Field: value, ...}`
  - Slices: `[]Type{elem1, elem2}`
  - Maps: `map[KeyType]ValueType{key: value}`
  - Primitives: use realistic values
- Include all fields used in downstream logic
- For pointer types, use address operator: `&Type{}`
- For error returns: comment out error handling

### Code Modification Instructions (Phase 3)
- Show original code with `//` comments (preserve indentation)
- Show test data assignment (match variable name exactly)
- Indicate preserved logic: validation, transformation, AWS SDK call
- For AWS Get/Delete: provide separate pre-insert code block
- Use `<details>` tags for better readability

### Interactive UX (Phase 2 & 4)
- Use fzf for smooth selection experience
- Clear prompts in Japanese
- Handle Ctrl-C gracefully
- Allow easy navigation between multiple functions
- Confirm before looping back

## Notes

### Command Behavior
- Stop immediately if PR is not AWS SDK Go related
- Use Task tool for all code analysis (avoid manual grep/read)
- Cache initial function list to avoid re-analyzing on each loop iteration
- Include file:line references for easy navigation
- If fzf is not installed, provide clear error with installation instructions
- Handle edge cases: no functions found, invalid selection, etc.

### Interactive Flow
- Phase 1: Extract all functions once (cached for loop)
- Phase 2: User selects function via fzf
- Phase 3: Detailed analysis for selected function only
- Phase 4: Loop back or exit
- Keep analysis context between iterations for performance

### Output Quality
- Provide complete call chains for traceability
- Focus on connection configuration (client, endpoints, regions)
- Include specific AWS console verification steps
- Make test code modifications copy-paste ready
- Show realistic test data that matches actual types

### Testing Approach
- Mock only data source access (repository, DB, API, file)
- Keep AWS SDK v2 calls active (they will use test data)
- Preserve all business logic between data fetch and AWS call
- For Get/Delete: provide both pre-insert and execution code
