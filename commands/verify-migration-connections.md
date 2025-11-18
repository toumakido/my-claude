Interactively analyze AWS SDK migration function by function

Output language: Japanese, formal business tone

## Command Purpose

This command **actually tests** AWS connections after AWS SDK Go v1→v2 migration by automatically executing:

1. Extract and analyze migrated functions
2. Generate test data and pre-insert to DynamoDB/S3
3. Mock external APIs (data source replacement)
4. **Apply actual code modifications** (automatic rewrite with Edit tool)
5. Execute AWS SDK v2 API tests

**Important**: This command does not just analyze - it **actually modifies code**. Changes can be reverted with `git restore` if under Git management.

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
   - For each chain, count all AWS SDK v2 method calls within the chain
   - Mark chains with multiple SDK methods as high priority

   Step 3: Sort call chains by priority:
   1. First: Chains with multiple AWS SDK methods (higher priority)
   2. Within same SDK method count: Sort by chain length (shorter = easier)

   Example priority order:
   - Chain with 3 SDK methods, 4 hops (highest)
   - Chain with 2 SDK methods, 2 hops
   - Chain with 2 SDK methods, 5 hops
   - Chain with 1 SDK method, 2 hops
   - Chain with 1 SDK method, 4 hops (lowest)

   Return: function list with all call chains sorted by priority."

3. **Group and deduplicate chains with Task tool** (subagent_type=general-purpose)
   Task prompt: "Group and deduplicate call chains to create optimal combination:

   Step 1: Group by unique SDK function
   - Key: file_path:line_number + function_name + SDK_operation
   - Value: list of call chains targeting same SDK function
   - Example: all chains to 'internal/repository/user.go:45 | Save | DynamoDB PutItem' are grouped

   Step 2: Select representative chain from each group
   For each group with multiple chains:
   - Priority 1: Shortest chain (fewest hops)
   - Priority 2: Entry point is 'main' function
   - Priority 3: First in list (tie-breaker)
   - Mark selected chain with: [+N other chains] where N = group size - 1

   For groups with single chain:
   - Select the only chain
   - No marker needed

   Step 3: Create optimal combination
   - Combine all selected representative chains
   - Maintain original priority sorting (from step 2)
   - Result: minimal set covering all unique SDK functions

   Return: optimal combination (deduplicated chains), each with group info"

4. **Format and cache optimal combination**
   Store Task result in variable for batch processing.

   Output:
   ```
   === 重複排除と最適化後の組み合わせ ===

   合計SDK関数数: N個 (重複排除前: M個のチェーン)
   選択されたチェーン数: N個

   [Sorted by priority: multiple SDK methods first, then by chain length]

   各チェーンに [+X other chains] マーカーを表示（重複がある場合）
   ```

### Phase 2: Batch Approval

5. **Present optimal combination and get batch approval**

   A. Display summary of optimal combination:
   ```
   === バッチ処理する組み合わせ ===

   合計SDK関数数: N個
   - Read操作: X個
   - Write操作: Y個
   - Delete操作: Z個
   - 複数SDK使用: W個

   処理対象のチェーン:
   1. [file:line] | [function] | [operations] [markers]
   2. [file:line] | [function] | [operations] [markers]
   ...
   ```

   B. Request batch approval with AskUserQuestion:
   - question: "この組み合わせでN個のSDK関数をバッチ処理しますか？"
   - header: "Batch"
   - multiSelect: false
   - options:
     - label: "はい", description: "N個のSDK関数を全自動で順次処理"
     - label: "いいえ", description: "キャンセルして終了"

   C. Handle response:
   - If "はい" selected: proceed to Phase 3 (batch processing)
   - If "いいえ" selected: exit with "処理をキャンセルしました"

### Phase 3: Batch Processing

6. **Process all chains in optimal combination sequentially**

   For each chain in optimal combination (index i from 1 to N):

   A. Display progress:
   ```
   === 処理中 (i/N) ===
   関数: [file_path:line_number] | [function_name] | [operations]
   ```

   B. Execute steps 7-11 for current chain

7. **Analyze selected function with Task tool** (subagent_type=general-purpose)
   Task prompt: "For function [function_name] at [file_path:line_number] with call chain [selected_chain]:

   1. Identify AWS SDK operation type using Read:
      - Find AWS SDK v2 client method calls: `client\.GetItem`, `client\.PutItem`, etc.
      - Classify operation type:
        - Read operations: GetItem, GetObject, DescribeTable, etc.
        - Delete operations: DeleteItem, DeleteObject, etc.
        - Write operations: PutItem, PutObject, UpdateItem, etc.
      - Extract operation name and line number

   2. Extract AWS settings from [function_name] using Read:
      - Region: look for `WithRegion\|AWS_REGION`
      - Resource: table name, bucket name from client call parameters
      - Endpoint: look for `WithEndpointResolver\|endpoint`

   3. Document v1 → v2 changes from git diff:
      - Client init: session.New vs config.LoadDefaultConfig
      - API call: old vs new method signature
      - Type changes: aws.String vs direct string usage

   Return: AWS operation type (Read/Delete/Write), operation name, line number, AWS settings, migration summary. Use [selected_chain] as call chain (do not re-trace)."

8. **Identify data source access with Task tool** (subagent_type=general-purpose)
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

9. **Validate and extract type information with Task tool** (subagent_type=general-purpose)
   Task prompt: "Before generating test data, extract and validate type information:

   1. Find model definitions using Glob:
      - Search for `type.*struct` in `model.go`, `models.go`, `types.go`, `entity.go`, `dto.go`
      - Search in same directory as [file_path] and parent directories
      - Check return types in function signatures

   2. Extract type information for each data source return type (from step 8):
      - Struct fields and their types (including pointer types: `*string`, `*int64`, `*bool`)
      - Slice element types (distinguish `[]*Type` vs `[]Type`)
      - Map key/value types
      - Exported vs unexported fields
      - Nested struct types

   3. Check required imports for test data:
      - AWS SDK v2 packages: `github.com/aws/aws-sdk-go-v2/aws`, `github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue`, `github.com/aws/aws-sdk-go-v2/service/*/types`
      - Standard library packages for test data construction

   4. For each struct type, document:
      - Full type name (e.g., `Account`, `*Account`, `[]Account`, `[]*Account`)
      - All field names with exact casing (e.g., `CustomerCode` not `AccountID`)
      - Field types with pointer indicators (e.g., `*string`, `int64`, `*bool`)
      - Example: `type Account struct { BranchCode string; CustomerCode string; Balance *int64 }`

   Return: Type information map with exact field names, types, pointer indicators, and required imports list."

10. **Generate test data and pre-insert code with Task tool** (subagent_type=general-purpose)
   Task prompt: "Generate test data and code modifications using validated type information from step 9:

   Part A - Data source access replacement (from step 8):
   For each data source access:
   1. Use exact type information from step 9:
      - Match struct field names exactly (e.g., `CustomerCode` not `AccountID`)
      - Use pointer types where required: `aws.String(\"value\")`, `aws.Int64(123)`
      - Match slice element types: `[]*Type` vs `[]Type`
      - Example for Account struct:
        ```go
        // From step 9: type Account struct { BranchCode string; CustomerCode string; Balance *int64 }
        testAccount := &Account{
            BranchCode:   \"100\",
            CustomerCode: \"123456\", // Not AccountID
            Balance:      aws.Int64(1000000),
        }
        ```

   2. Include all required imports in new_string:
      - Add missing imports from step 9 to import block
      - Example: `\"github.com/aws/aws-sdk-go-v2/aws\"`, `\"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue\"`

   3. Generate code modification:
      - Comment out original call with `//`
      - Assign test data to same variable
      - Preserve all downstream logic

   Part B - AWS SDK pre-insert code (from step 7):
   If AWS operation type is Read or Delete:
   1. Generate pre-insert code to populate test data:
      - For GetItem/Scan/Query: generate PutItem with same key
      - For GetObject: generate PutObject with same key
      - For DeleteItem: generate PutItem with same key
      - Use same resource (table/bucket) from step 7
      - Use realistic test data values with correct types from step 9

   2. Identify insertion point:
      - Line number just before AWS SDK Read/Delete operation
      - Preserve indentation

   Return:
   - Data source replacements: [original code (commented), test assignment code with imports]
   - Pre-insert code (if AWS operation is Read/Delete): [pre-insert code, insertion line number]"

11. **Apply code modifications with Edit tool**

   **Important**: This step applies actual code changes. This is not just analysis.

   Part A - Replace data source access (from step 10 Part A):
   For each data source access identified in step 8:
   - Use Edit tool to replace original data source call with test data
   - old_string: exact original code from function
   - new_string: test data assignment preserving downstream logic
   - If multiple data sources: apply edits sequentially
   - Output: "データソース書き換え完了: [file_path:line_number]"

   Part B - Insert pre-insert code (from step 10 Part B):
   If AWS operation type is Read or Delete:
   - Use Edit tool to insert pre-insert code before AWS SDK operation
   - Identify the line before AWS SDK call using line number from step 10
   - old_string: line before AWS SDK operation (preserve exact indentation)
   - new_string: line before AWS SDK operation + "\n" + pre-insert code (with proper indentation)
   - Output: "Pre-insertコード追加: [file_path:line_number]"
   - Add comment above pre-insert code: "// Pre-insert: test data for [operation_name]"

   Part C - Add verification logging (from step 9):
   For Read operations (Scan, Query, GetItem, GetObject):
   - Use Edit tool to insert logging code after AWS SDK Read operation
   - Extract key fields from type information (step 9)
   - Log format:
     ```go
     // Verify Pre-insert: log retrieved records
     logger.Infof(\"[function_name] returned %d records\", len(result))
     for i, record := range result {
         logger.Infof(\"  [%d] Key1=%v, Key2=%v, ...\", i, record.Key1, record.Key2)
     }
     ```
   - Insert after Read operation, before any result length check
   - Output: "検証ログ追加: [file_path:line_number]"

12. **Verify compilation with Bash tool**
   - Run: `go build -o /tmp/test-build 2>&1`
   - If compilation fails:
     - Analyze error messages
     - Fix missing imports, type mismatches, undefined fields
     - Retry Edit tool with corrections
     - Repeat until compilation succeeds
   - Output: "コンパイル成功: [file_path]"

13. **Output detailed report**
    Generate report for selected function with:
    - File path and function name
    - Complete call chain
    - AWS operation type (Read/Delete/Write) and operation name
    - AWS service and resource details
    - Migration changes summary
    - Applied code modifications:
      - Data source replacements (if any)
      - Pre-insert code (if AWS operation is Read/Delete)
      - Verification logging (if AWS operation is Read)
    - Compilation status
    - AWS console verification steps
    - Git diff summary showing changes

14. **Output brief summary for current chain**
    ```
    完了 (i/N): [file_path]
    - AWS操作: [operation_type] ([operation_name])
    - データソースモック: X個
    - Pre-insertコード: 追加済み / 不要
    - 検証ログ: 追加済み / 不要
    - コンパイル: 成功 / 失敗
    ```

   C. Proceed to next chain automatically (no user interaction)
      - If i < N: continue to next chain (repeat from step 6.A)
      - If i = N: proceed to final summary

### Phase 4: Final Summary

15. **Output final summary report**
    After processing all chains, display comprehensive summary:

    ```
    === 処理完了サマリー ===

    処理したSDK関数: N個
    - Read操作: X個 (Pre-insertコード追加済み、検証ログ追加済み)
    - Write操作: Y個
    - Delete操作: Z個 (Pre-insertコード追加済み)
    - 複数SDK使用: W個

    書き換えたファイル: (unique list)
    - [file_path_1]
    - [file_path_2]
    ...

    データソースモック: 合計M個
    検証ログ: 合計L個
    コンパイル: 成功P個 / 失敗Q個

    次のステップ:
    1. git diff で変更内容を確認
    2. コンパイルエラーがある場合は修正
    3. アプリケーションを実行してAWS接続をテスト
    4. ログ出力から取得レコード数と内容を確認
    5. CloudWatchログで各API呼び出しを確認
    6. 必要に応じてテストデータを調整

    すべての処理が完了しました。
    ```

## Output Format

### Optimal Combination (Phase 1)
```
=== 重複排除と最適化後の組み合わせ ===

合計SDK関数数: 5個 (重複排除前: 8個のチェーン)
選択されたチェーン数: 5個

[Sorted by priority: multiple SDK methods first, then by chain length]

1. internal/service/order.go:120 | (*OrderService).Process | DynamoDB + S3 + SES [★ Multiple SDK]
   Chain: main → OrderService.Process → DynamoDB PutItem → S3 PutObject → SES SendEmail (3 SDK methods, 4 hops)

2. internal/gateway/data.go:89 | (*DataGateway).Import | S3 + DynamoDB [★ Multiple SDK] [+1 other chain]
   Chain: handler → ImportService.Run → S3 GetObject → DynamoDB BatchWriteItem (2 SDK methods, 5 hops)

3. internal/repository/user.go:45 | (*UserRepository).Save | DynamoDB PutItem [+2 other chains]
   Chain: main → UserUsecase.Create → UserRepository.Save (1 SDK method, 2 hops)

4. internal/gateway/s3.go:120 | (*S3Gateway).Upload | S3 PutObject
   Chain: main → FileService.Process → S3Gateway.Upload (1 SDK method, 2 hops)

5. internal/repository/user.go:89 | (*UserRepository).Get | DynamoDB GetItem
   Chain: handler → UserService.Fetch → UserRepository.Get (1 SDK method, 3 hops)
```

### Batch Approval Summary (Phase 2)
```
=== バッチ処理する組み合わせ ===

合計SDK関数数: 5個
- Read操作: 1個
- Write操作: 3個
- Delete操作: 0個
- 複数SDK使用: 2個

処理対象のチェーン:
1. order.go:120 | Process | DynamoDB + S3 + SES [★]
2. data.go:89 | Import | S3 + DynamoDB [★] [+1]
3. user.go:45 | Save | DynamoDB [+2]
4. s3.go:120 | Upload | S3
5. user.go:89 | Get | DynamoDB
```

### Detailed Report for Selected Function (Phase 3)
```markdown
## 選択した関数の接続検証情報

### ファイル: [file_path:line_number]
#### 関数/メソッド: [function_name]

**AWS操作タイプ**: [Read/Delete/Write]
**AWS操作名**: [operation_name] (例: GetItem, PutItem, DeleteObject)

**呼び出しチェーン**:
```
[entry_point] (例: cmd/main.go:main())
  → [usecase/service_layer] (例: internal/usecase/user.go:(*UserUsecase).MigrateUser())
  → [repository/gateway_layer] (例: internal/repository/user.go:(*UserRepository).FetchByID())
  → AWS SDK v2 API呼び出し (例: DynamoDB GetItem)
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

**適用したコード変更**:

**A. データソースのモック** (該当する場合):
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

**B. Pre-insertコード** (AWS操作がRead/Deleteの場合):
<details>
<summary>追加されたPre-insertコード</summary>

```go
// Pre-insert: test data for [operation_name]
// [pre-insert code that populates test data]
// Example for GetItem:
_, err := client.PutItem(ctx, &dynamodb.PutItemInput{
    TableName: aws.String("users"),
    Item: map[string]types.AttributeValue{
        "id": &types.AttributeValueMemberS{Value: "test-123"},
        "name": &types.AttributeValueMemberS{Value: "Test User"},
    },
})
```
</details>

**動作確認観点**:
- AWSコンソール: [確認するサービス/リソース]
- CloudWatchログ: [確認すべきAPIコール]
- 設定確認: [region/endpoint/認証情報など]
```

### Brief Summary (Phase 3, per chain)
```
完了 (3/5): internal/repository/user.go
- AWS操作: Write (PutItem)
- データソースモック: 2個
- Pre-insertコード: 不要
```

### Final Summary (Phase 4)
```
=== 処理完了サマリー ===

処理したSDK関数: 5個
- Read操作: 1個 (Pre-insertコード追加済み)
- Write操作: 3個
- Delete操作: 0個
- 複数SDK使用: 2個

書き換えたファイル:
- internal/service/order.go
- internal/gateway/data.go
- internal/repository/user.go
- internal/gateway/s3.go

データソースモック: 合計8個

次のステップ:
1. git diff で変更内容を確認
2. アプリケーションを実行してAWS接続をテスト
3. CloudWatchログで各API呼び出しを確認
4. 必要に応じてテストデータを調整

すべての処理が完了しました。
```

## Analysis Requirements

### General
- Focus on production AWS connections (exclude localhost/test endpoints)
- Extract resource names (table names, bucket names, queue URLs)
- Identify region configuration (explicit config or AWS_REGION env var)
- Summarize v1 → v2 migration patterns clearly
- Provide actionable AWS console verification steps

### Test Data and Code Modifications
- **Type validation (step 9)**: Extract exact type information before generating test data
  - Match struct field names exactly (e.g., `CustomerCode` not `AccountID`)
  - Distinguish pointer types: `*string`, `*int64`, `*bool`
  - Distinguish slice element types: `[]*Type` vs `[]Type`
  - Include all required imports: `aws`, `attributevalue`, service-specific `types`
- Match Go types exactly (structs, slices, maps, primitives, pointers)
- Include all fields used in downstream logic
- Comment out error handling for test data
- Preserve indentation in code blocks
- Match variable names exactly from original code
- For AWS Read/Delete operations:
  - Generate pre-insert code to populate test data (e.g., PutItem before GetItem/Scan/Query)
  - Automatically insert pre-insert code before AWS SDK operation
  - Add comment: "// Pre-insert: test data for [operation_name]"
  - Add verification logging after Read operation to confirm data retrieval
- Use `<details>` tags for readability

### Complex Chain Handling
- **Multiple AWS SDK calls**: Process all SDK calls within the same function
  - Step 2 identifies chains with multiple SDK methods and prioritizes them
  - Step 7 identifies all AWS SDK operations within the function
  - Steps 8-11 process each SDK operation independently
  - Generate separate Pre-insert code for each Read/Delete operation
  - Apply data source mocks once (shared across all SDK operations in the function)
- **Complex detection criteria** (for reference):
  - AWS SDK calls: 3 or more operations
  - Data source access: 4 or more calls
  - Call chain depth: 6 or more hops
- **Processing approach**: Attempt automatic processing for all chains
  - Only suggest manual intervention when Task tool encounters unrecoverable errors
  - Provide clear error context and suggested fixes when manual intervention needed

### Batch Processing UX
- Phase 1: Extract and deduplicate automatically
- Phase 2: Single batch approval via AskUserQuestion
- Phase 3: Process all chains automatically without user interaction
- Phase 4: Display final summary
- No interactive loop - fully automated after approval

## Notes

- Stop immediately if branch diff does not contain `aws-sdk-go-v2` imports
- Use Task tool for code analysis (steps 2, 3, 7, 8, 9, 10)
- Use Edit tool to automatically apply code modifications (step 11)
- Use Bash tool for compilation verification (step 12)
- **Deduplication** (step 3):
  - Group chains by unique SDK function (file:line + function + operation)
  - Select representative chain from each group (shortest hops, main entry point)
  - Mark with [+N other chains] indicator
- **Batch processing** (Phase 3):
  - Process all chains in optimal combination sequentially
  - Display progress for each chain (i/N)
  - No user interaction during processing
- Sort call chains by priority:
  1. Chains with multiple AWS SDK methods (higher priority/重要度高)
  2. Within same SDK method count: shorter chains first (easier execution)
- Present call chains with SDK method count and hop counts
- Mark chains with multiple SDK methods with [★ Multiple SDK] indicator
- Include file:line references in all outputs for navigation
- Provide complete call chains for traceability
- Identify AWS operation type (Read/Delete/Write) in step 7
- Focus on connection configuration (client, endpoints, regions)
- **Type validation and test data generation**:
  - Extract exact type information in step 9 (field names, pointer types, slice types)
  - Generate test data with correct types in step 10 Part A
  - Include required imports (aws, attributevalue, types)
- Automatically replace data source access with test data (step 11 Part A)
- Mock only data source access (repository, DB, API, file)
- For AWS Read/Delete operations:
  - Generate pre-insert code to populate test data (step 10 Part B)
  - Automatically insert pre-insert code before AWS SDK operation (step 11 Part B)
  - Add verification logging after Read operation (step 11 Part C)
  - Enables testing of read/delete operations with pre-populated data
- **Compilation verification** (step 12):
  - Run `go build` after code modifications
  - Fix compilation errors automatically (imports, types, fields)
  - Retry until compilation succeeds
- Keep AWS SDK v2 calls active to test against real AWS
- Preserve all business logic between data fetch and AWS call
- Show git diff after modifications to verify changes
- Display final summary with statistics after all processing complete
  - Include compilation status (success/failure counts)
