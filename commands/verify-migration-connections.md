Interactively analyze AWS SDK migration function by function

Output language: Japanese, formal business tone

## Prerequisites

- Run from repository root
- Current branch must contain AWS SDK Go v2 migration changes
- Working tree can be dirty (uncommitted changes allowed)

## Process

### Phase 1: Extract Functions and Call Chains

1. **Fetch and validate branch diff**
   - Run: `git diff main...HEAD`
   - If diff does not contain `github.com/aws/aws-sdk-go-v2` imports: output "このブランチはAWS SDK Go v2関連の変更を含んでいません" and stop

2. **Extract functions and call chains with Task tool** (subagent_type=general-purpose)
   Task prompt: "Parse git diff and extract all functions/methods using AWS SDK v2 with their call chains.

   Step 1: Extract functions. Search patterns:
   - Import: `github.com/aws/aws-sdk-go-v2/service/*`
   - Client calls: `client.PutItem`, `client.GetObject`, etc.
   - Context parameter: functions with `context.Context` calling AWS clients

   For each match, extract:
   - File path:line_number from diff headers
   - Function/method name from signature
   - AWS service from import path (dynamodb, s3, ses, etc.)
   - Operation from client method name (PutItem, GetObject, etc.)

   Step 2: For each function, trace ALL call chains using Grep:
   - Find entry points: `main\(`, `handler\(`, `ServeHTTP`, `Handle`
   - Search function references to trace call paths
   - Identify intermediate layers (usecase/service/repository/gateway)
   - Build complete chains: entry → intermediate → SDK function

   Step 3: If function has multiple call chains:
   - Sort by chain length (shorter = easier to execute)
   - Prioritize shorter chains in output

   Return: function list with all call chains sorted by preference."

3. **Format and cache results**
   Store Task result in variable for reuse across loop iterations.

   Output:
   ```
   === AWS SDK v2を使用している関数一覧 ===

   検出された関数数: N個
   検出された呼び出しパターン数: M個

   [Sorted by execution ease]
   ```

### Phase 2: Interactive Selection Loop

4. **Present selection UI with AskUserQuestion**
   - Take up to 4 call chains from cached results (sorted by execution ease)
   - Format each option:
     - label: `[Function] file:line`
     - description: Complete call chain with → separators

   Example options:
   ```
   label: "[Save] internal/repository/user.go:45"
   description: "main → UserUsecase.Create → UserRepository.Save → DynamoDB PutItem (3 hops)"

   label: "[Save] internal/repository/user.go:45"
   description: "handler → AdminService.Import → UserUsecase.Migrate → UserRepository.Save → DynamoDB PutItem (4 hops)"
   ```

   AskUserQuestion parameters:
   - question: "検証する関数と呼び出しチェーンを選択してください"
   - header: "Function"
   - multiSelect: false
   - Include "次の4件を表示" option if more than 4 chains remain

5. **Handle selection**
   - If "次の4件を表示" selected: show next 4 chains, repeat step 4
   - If chain selected: extract file path, function name, and chain
   - Proceed to Phase 3

### Phase 3: Detailed Analysis for Selected Function

6. **Analyze selected function with Task tool** (subagent_type=general-purpose)
   Task prompt: "For function [function_name] at [file_path:line_number] with call chain [selected_chain]:

   1. Extract AWS settings from [function_name] using Read:
      - Region: look for `WithRegion\|AWS_REGION`
      - Resource: table name, bucket name from client call parameters
      - Endpoint: look for `WithEndpointResolver\|endpoint`

   2. Document v1 → v2 changes from git diff:
      - Client init: session.New vs config.LoadDefaultConfig
      - API call: old vs new method signature
      - Type changes: aws.String vs direct string usage

   Return: AWS settings, migration summary. Use [selected_chain] as call chain (do not re-trace)."

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

9. **Apply code modifications with Edit tool**
   For each data source access identified in step 7:
   - Use Edit tool to replace original data source call with test data
   - old_string: exact original code from function
   - new_string: test data assignment preserving downstream logic
   - If multiple data sources: apply edits sequentially
   - Output: "書き換え完了: [file_path:line_number]"

10. **Output detailed report**
    Generate report for selected function with:
    - File path and function name
    - Complete call chain
    - AWS service and resource details
    - Migration changes summary
    - Applied code modifications (show what was changed)
    - AWS console verification steps
    - Git diff summary showing changes

### Phase 4: Loop Back

11. **Prompt for next action with AskUserQuestion**
    After displaying report, use AskUserQuestion:
    - question: "別の関数を確認しますか？"
    - header: "Next"
    - multiSelect: false
    - options:
      - label: "はい", description: "別の関数を選択して検証を続ける"
      - label: "いいえ", description: "検証を終了する"

    - If "はい" selected: return to step 4 (Phase 2)
    - If "いいえ" selected: exit with "検証情報の出力を完了しました"

## Output Format

### Initial Function List (Phase 1)
```
=== AWS SDK v2を使用している関数一覧 ===

検出された関数数: N個
検出された呼び出しパターン数: M個

[Sorted by execution ease - shorter chains first]

1. internal/repository/user.go:45 | (*UserRepository).Save | DynamoDB PutItem
   Chain: main → UserUsecase.Create → UserRepository.Save (2 hops)

2. internal/repository/user.go:45 | (*UserRepository).Save | DynamoDB PutItem
   Chain: handler → AdminService.Import → UserUsecase.Migrate → UserRepository.Save (3 hops)

3. internal/gateway/s3.go:120 | (*S3Gateway).Upload | S3 PutObject
   Chain: main → FileService.Process → S3Gateway.Upload (2 hops)
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
- Use AskUserQuestion for all user interactions
- Present up to 4 call chains per selection (sorted by execution ease)
- Include pagination option if more than 4 chains available
- Allow multiple function analysis in single session
- Confirm before looping with AskUserQuestion

## Notes

- Stop immediately if branch diff does not contain `aws-sdk-go-v2` imports
- Use Task tool for code analysis (steps 2, 6, 7, 8)
- Use Edit tool to automatically apply code modifications (step 9)
- Cache function list and call chains from step 3 for reuse in Phase 4 loop
- Sort call chains by length (shorter chains = easier execution)
- Present call chains with hop counts for easy comparison
- Include file:line references in all outputs for navigation
- Provide complete call chains for traceability
- Focus on connection configuration (client, endpoints, regions)
- Automatically replace data source access with test data
- Mock only data source access (repository, DB, API, file)
- Keep AWS SDK v2 calls active to test against real AWS
- Preserve all business logic between data fetch and AWS call
- For Get/Delete operations: provide both pre-insert and execution code
- Show git diff after modifications to verify changes
