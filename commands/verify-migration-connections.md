Interactively analyze AWS SDK migration PR function by function: $ARGUMENTS

Output language: Japanese, formal business tone

## Prerequisites

- gh CLI installed and authenticated
- peco installed for interactive selection
- $ARGUMENTS: PR number
- Run from repository root
- PR must contain AWS SDK Go migration changes

## Process

### Phase 1: Extract Functions

1. **Fetch and validate PR**
   - Run: `gh pr diff $ARGUMENTS`
   - If diff does not contain `github.com/aws/aws-sdk-go-v2` imports: output "このPRはAWS SDK Go関連の変更を含んでいません" and stop

2. **Extract function list with Task tool** (subagent_type=general-purpose)
   Task prompt: "Parse PR diff and extract all functions/methods using AWS SDK v2. Search patterns:
   - Import: `github.com/aws/aws-sdk-go-v2/service/*`
   - Client calls: `client.PutItem`, `client.GetObject`, etc.
   - Context parameter: functions with `context.Context` calling AWS clients

   For each match, extract:
   - File path:line_number from diff headers
   - Function/method name from signature
   - AWS service from import path (dynamodb, s3, ses, etc.)
   - Operation from client method name (PutItem, GetObject, etc.)

   Return formatted list only, no analysis."

3. **Format and cache function list**
   Store Task result in variable for reuse across loop iterations.

   Format entries as:
   ```
   <file_path>:<line_number> | <function_name> | <AWS_Service> <Operation>
   ```
   Example:
   ```
   internal/repository/user.go:45 | (*UserRepository).Save | DynamoDB PutItem
   internal/gateway/s3.go:120 | (*S3Gateway).Upload | S3 PutObject
   ```

   Output:
   ```
   === AWS SDK v2を使用している関数一覧 ===

   検出された関数数: N個

   [function list]
   ```

### Phase 2: Interactive Selection Loop

4. **Present selection UI**
   ```bash
   echo "関数を選択してください (Ctrl-C で終了):"
   echo "<formatted_list>" | peco --prompt "関数を選択> "
   ```

5. **Handle selection**
   - If user cancels (Ctrl-C): exit with "終了しました"
   - If function selected: extract file path and function name
   - Proceed to Phase 3

### Phase 3: Detailed Analysis for Selected Function

6. **Analyze selected function with Task tool** (subagent_type=general-purpose)
   Task prompt: "For function [function_name] at [file_path:line_number]:

   1. Find entry point using Grep:
      - Search `main\(` in cmd/main.go or main.go
      - Search `handler\(` for Lambda handlers
      - Search `ServeHTTP\|Handle` for HTTP handlers

   2. Trace call chain using Grep from entry point to [function_name]:
      - Search [function_name] references
      - Identify intermediate layers (usecase/service/repository/gateway)

   3. Extract AWS settings from [function_name] using Read:
      - Region: look for `WithRegion\|AWS_REGION`
      - Resource: table name, bucket name from client call parameters
      - Endpoint: look for `WithEndpointResolver\|endpoint`

   4. Document v1 → v2 changes from PR diff:
      - Client init: session.New vs config.LoadDefaultConfig
      - API call: old vs new method signature
      - Type changes: aws.String vs direct string usage

   Return: call chain, AWS settings, migration summary."

7. **Identify data source access with Task tool** (subagent_type=general-purpose)
   Task prompt: "In function [function_name] at [file_path:line_number], identify ALL data source access BEFORE AWS SDK calls using Read:

   Search patterns:
   - Repository/gateway calls: `repo\.\|gateway\.\|[A-Z][a-z]*Repository\|[A-Z][a-z]*Gateway`
   - Database: `db\.Query\|db\.Exec\|\.Scan\|\.QueryRow`
   - HTTP: `client\.Get\|client\.Post\|http\.Do`
   - File: `os\.ReadFile\|ioutil\.ReadFile\|os\.Open`
   - Cache: `cache\.Get\|redis\.Get`

   For each match, extract:
   - Line number
   - Method signature or function call
   - Variable name storing result
   - Return type from function declaration or type inference

   Return: list with line numbers, calls, variables, types."

8. **Generate test data with Task tool** (subagent_type=general-purpose)
   Task prompt: "For each data source access from step 7, generate test data and code modification:

   For each access:
   1. Create test data matching return type:
      - Struct: `&StructName{Field: \"value\", ...}` (use realistic values)
      - Slice: `[]Type{elem1, elem2}`
      - Map: `map[KeyType]ValueType{\"key\": value}`
      - Primitive: use realistic value
      - Pointer: use `&Type{...}`

   2. Generate code modification:
      - Comment out original call with `//`
      - Assign test data to same variable
      - Preserve all downstream logic
      - If AWS operation is Get/Delete: provide pre-insert code block

   Return: original code (commented), test assignment code, pre-insert code (if needed)."

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

## Analysis Requirements

### General
- Focus on production AWS connections (exclude localhost/test endpoints)
- Extract resource names (table names, bucket names, queue URLs)
- Identify region configuration (explicit config or AWS_REGION env var)
- Summarize v1 → v2 migration patterns clearly
- Provide actionable AWS console verification steps

### Test Data and Code Modifications
- Match Go types exactly (structs, slices, maps, primitives, pointers)
- Include all fields used in downstream logic
- Comment out error handling for test data
- Preserve indentation in code blocks
- Match variable names exactly from original code
- For AWS Get/Delete: provide separate pre-insert code block
- Use `<details>` tags for readability

### Interactive UX
- Use peco for function selection
- Handle Ctrl-C gracefully with "終了しました"
- Allow multiple function analysis in single session
- Confirm before looping: "別の関数を確認しますか？ (y/n):"
- If peco not installed: output "pecoがインストールされていません。brew install pecoを実行してください" and stop

## Notes

- Stop immediately if PR does not contain `aws-sdk-go-v2` imports
- Use Task tool for all code analysis (steps 2, 6, 7, 8)
- Cache function list from step 3 for reuse in Phase 4 loop
- Include file:line references in all outputs for navigation
- Provide complete call chains for traceability
- Focus on connection configuration (client, endpoints, regions)
- Make test code modifications copy-paste ready with realistic data
- Mock only data source access (repository, DB, API, file)
- Keep AWS SDK v2 calls active to test against real AWS
- Preserve all business logic between data fetch and AWS call
- For Get/Delete operations: provide both pre-insert and execution code
