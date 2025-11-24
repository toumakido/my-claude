Verify AWS SDK v2 migration by temporarily modifying code for focused testing

Output language: Japanese, formal business tone

## Command Purpose

Prepares code for AWS SDK v2 connection testing by temporarily modifying migrated code:

1. Extract migrated functions from current branch
2. Comment out unrelated code to isolate AWS SDK operations (Phase 3)
3. Generate minimal test data (Phase 4)
4. Output AWS environment verification procedures (Phase 5)

**Post-execution steps:**
1. Review `git diff`
2. Deploy to AWS test environment
3. Execute verification procedures (Step 14 output)
4. Run `git restore .` to revert changes

**Critical notes:**
- Modifies production code in working tree (not automatically reverted)
- Phase 3 isolation is mandatory to prevent interference from unrelated code
- Creates testable state, not permanent test code

## Prerequisites

- Run from repository root
- Working tree can be dirty (uncommitted changes allowed)

## Process

### Phase 1: Extract Functions and Call Chains

1. **Validate branch has AWS SDK v2 changes**
   - Run: `git diff main...HEAD`
   - Search for pattern: `github.com/aws/aws-sdk-go-v2`
   - If not found: output "このブランチはAWS SDK Go v2関連の変更を含んでいません" and exit immediately

2. **Extract functions and call chains with Task tool** (subagent_type=general-purpose)
   Task prompt: "Parse git diff and extract all functions/methods using AWS SDK v2 with their call chains.

   Step 1: Extract functions using Grep in this order:
   1. Search for imports: `github.com/aws/aws-sdk-go-v2/service/*`
   2. Search for client calls: `client\.(PutItem|GetObject|Query|UpdateItem|DeleteItem|PutObject|GetObject|DeleteObject|SendEmail|Publish)` pattern
   3. Filter functions with `context.Context` parameter

   For each match, extract:
   - File path:line_number from diff headers
   - Function/method name from signature
   - AWS service from import path (dynamodb, s3, ses, etc.)
   - Operation from client method name (PutItem, GetObject, etc.)

   Step 2: For each function from Step 1, trace COMPLETE call chains including entry point using Grep:

   Entry point identification and verification:
   1. Identify entry point type:
      - API handlers: Search for route registration (router.POST, router.GET, http.HandleFunc)
      - Task entry points: Search cmd/ directory for binary definitions
      - CLI commands: Search for subcommand definitions
   2. Extract entry point details:
      - API: HTTP method + full path (/v1/resources, /api/v2/items)
      - Task: Binary name (cmd/process_task/main.go → process_task)
      - CLI: Subcommand name + arguments
   3. Verify entry point exists using Grep/Glob before tracing
      - If not found: mark function as "SKIP - No entry point"

   Call chain tracing (execute after entry point verification):
   - Trace from verified entry points: `main\(`, `handler\(`, `ServeHTTP`, `Handle`
   - Search function references to build call paths
   - Identify intermediate layers (usecase/service/repository/gateway)
   - Build complete chains: verified_entry → intermediate → SDK function
   - Count all AWS SDK v2 method calls in chain
   - Record all SDK operations for grouped verification
   - **Parallel execution**: Execute Grep searches in parallel for functions with independent call chains (no shared intermediate layers)

   Step 2.5: Verify active callers (exclude unused implementations)

   For each extracted function:
   1. Use Grep to search for function calls (exclude *_test.go, mocks/*.go)
   2. Count active call sites in production code (handlers, tasks, services)
   3. If active callers = 0:
      - Mark as "SKIP - No active callers"
      - Exclude from call chain list

   **Multiple path handling** (when function has 2+ call chains):
   1. Apply exclusion criteria first (eliminate complex paths):
      - Exclude if 6+ external dependencies (too many mocks needed)
      - Exclude if 6+ chain hops (too deep call stack)
      - Exclude if multiple validation layers (complex to satisfy)
   2. From remaining paths, select by priority (choose first match):
      - Fewest external dependencies (< 4 preferred)
      - Shortest chain length (< 4 hops preferred)
      - Direct entry point (main function or simple handler)
   3. Document selection in output:
      ```
      Selected: POST /v1/entities (2 dependencies, 3 hops)
      Excluded: Background job path (7 dependencies, 5 hops) - too complex
      ```

   Call chain format:
   ```
   [Entry Point]
   → [Handler/Task file:line] HandlerMethod
   → [Service file:line] ServiceMethod
   → [Target file:line] TargetFunction
   → AWS SDK v2 API (Operation)
   ```

   Example:
   ```
   POST /v1/entities
   → internal/api/handler/v1/entity_handler.go:45 PostEntities
   → internal/service/entity_service.go:123 CreateEntity
   → internal/repository/entity_repo.go:78 SaveEntity
   → DynamoDB PutItem
   ```

   If entry point not verified (marked in Step 2):
   - Exclude from call chain list immediately
   - Log as skipped function:
     ```
     スキップされた関数（エントリーポイント不明）:
     - internal/service/file.go:123 FunctionName
     ```

   Step 3: Sort call chains by priority:
   1. First: Chains with multiple AWS SDK methods (higher priority)
   2. Within same SDK method count: Sort by chain length (shorter = easier)

   Example priority order:
   - Chain with 3 SDK methods, 4 hops (highest)
   - Chain with 2 SDK methods, 2 hops
   - Chain with 2 SDK methods, 5 hops
   - Chain with 1 SDK method, 2 hops
   - Chain with 1 SDK method, 4 hops (lowest)

   Return: Filtered function list with only actively called functions, all call chains sorted by priority, skipped functions list (including unused implementations)."

3. **Deduplicate chains with coverage-based selection (Task tool)** (subagent_type=general-purpose)
   Task prompt: "Deduplicate call chains using coverage-based selection to minimize redundant verification:

   **Strategy**: Prioritize chains with multiple SDK operations, track covered operations, skip chains that don't add new coverage.

   Step 1: Sort chains by SDK operation count (descending)
   - Primary sort: Number of SDK operations (more operations = higher priority)
   - Secondary sort: Chain length (fewer hops = easier to verify)
   - Rationale: Chains with multiple SDK operations can verify several migrations in one execution

   Example sorted order:
   ```
   1. ChainB: DynamoDB PutItem + S3 PutObject + SES SendEmail (3 ops, 5 hops)
   2. ChainD: DynamoDB Query + SNS Publish (2 ops, 3 hops)
   3. ChainA: DynamoDB PutItem (1 op, 2 hops)
   4. ChainC: S3 PutObject (1 op, 2 hops)
   5. ChainE: SES SendEmail (1 op, 4 hops)
   ```

   Step 2: Select chains with coverage tracking
   Initialize: covered_operations = {} (empty set)

   For each chain in sorted order:
   1. Extract all SDK operations in chain
      - Format: "AWS_service + SDK_operation"
      - Example: ["DynamoDB PutItem", "S3 PutObject", "SES SendEmail"]

   2. Check for new coverage:
      - new_operations = chain operations NOT in covered_operations
      - If new_operations is empty: SKIP this chain (all operations already covered)
      - If new_operations is not empty: SELECT this chain

   3. Update covered operations:
      - Add all chain operations to covered_operations
      - Mark chain with: [+N similar chains] where N = number of skipped chains covering same operations

   Example execution:
   ```
   ChainB (DynamoDB + S3 + SES):
     new_operations = {DynamoDB PutItem, S3 PutObject, SES SendEmail}
     → SELECT ✓
     covered = {DynamoDB PutItem, S3 PutObject, SES SendEmail}

   ChainD (DynamoDB Query + SNS):
     new_operations = {DynamoDB Query, SNS Publish}
     → SELECT ✓
     covered = {DynamoDB PutItem, S3 PutObject, SES SendEmail, DynamoDB Query, SNS Publish}

   ChainA (DynamoDB PutItem):
     new_operations = {} (already in covered)
     → SKIP (covered by ChainB)

   ChainC (S3 PutObject):
     new_operations = {} (already in covered)
     → SKIP (covered by ChainB)

   ChainE (SES SendEmail):
     new_operations = {} (already in covered)
     → SKIP (covered by ChainB)

   Result: ChainB [+2 similar chains], ChainD
   ```

   Step 3: Verify entry points (already verified in Phase 1 Step 2)
   For each selected chain:
   1. Confirm entry point was verified in Phase 1
   2. If entry point marked as "SKIP - No entry point" in Phase 1:
      - Remove from optimal combination
      - Add skipped operations back to uncovered set
      - Continue selection from remaining chains

   Step 4: Handle same-operation chains with different parameters
   When multiple chains use same SDK operation but with different parameters:
   - Example: DynamoDB Query with different FilterExpressions
   - Group by exact SDK operation signature if parameters affect behavior
   - Otherwise treat as same operation (parameters like table names, filters don't affect SDK v2 migration verification)

   Return: optimal combination (deduplicated chains with verified entry points), skipped chains list with skip reasons"

4. **Format and cache optimal combination**
   Store Task result in variable for batch processing.

   Output format (simple chain format):
   ```
   [N]. [file:line] | [function] | [operations] [markers]
   Chain: [entry point] → [intermediate layers] → AWS SDK API ([SDK method count], [hop count])
   Active callers: [count]箇所 ([locations])
   ```

   Example:
   ```
   1. internal/service/datastore.go:306 | CreateRecord | DynamoDB TransactWriteItems [+3 other operations]
   Chain: POST /v1/records → handler.CreateRecords → service.CreateRecord → DynamoDB TransactWriteItems (3 SDK methods, 4 hops)
   Active callers: 5箇所 (handlers)
   ```

   DO NOT output verbose format with full code signatures unless explicitly requested.

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

   検証方法のグループ化ポリシー:
   - Phase 4の動作確認手順では、実行方法（API/Task）ごとにグループ化して出力
   - 同じエンドポイント/タスクで確認できる複数の関数は、単一の実行コマンドにまとめて記載
   - 関数ごとに重複した手順を出力しない
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

   **Complexity handling policy** (if processing time exceeds reasonable limits):

   1. Mandatory minimum: Process at least N chains (N = min(total chains, 3))
   2. Priority order when time-constrained:
      - Chains with multiple SDK operations (highest priority)
      - Chains with Read/Delete operations (need Pre-insert)
      - Chains with single Write operations (lowest priority)

   3. Skip criteria (only apply after meeting mandatory minimum):
      - Chain has 6+ external dependencies
      - Chain has 6+ hops (deep call stack)
      - Processing single chain exceeds 15 minutes

   4. When skipping a chain:
      - Output: "スキップ: [reason] (Chain N: [description])"
      - Document skipped chains in final summary
      - Continue with remaining chains

### Phase 3: Comment-out Unrelated Code

**CRITICAL - MUST NOT SKIP**: This phase MUST be executed for ALL chains without exception.

**Approach**:
Keep only SDK-related code, comment out everything else:
1. Identify SDK operation and its input variables (e.g., `dynamoDB.PutItem(ctx, &input)`)
2. Identify SDK-related code: code that builds SDK input
3. Comment out unrelated code: code NOT used by SDK operation

**Why this is required**:
- Isolates target AWS SDK operations by removing unrelated code
- Enables focused testing of SDK v2 migration
- Without this step, verification tests too many operations simultaneously

6. **Comment out unrelated code in call chain functions (strict mode)**

   For each chain in optimal combination (index i from 1 to N):

   A. Display progress:
   ```
   === コメントアウト処理中 (i/N) ===
   関数: [file_path:line_number] | [function_name] | [operations]
   ```

   B. **Identify SDK-related code and unrelated code** (Task tool: subagent_type=general-purpose)
   Task prompt: "For call chain [entry_point → ... → target_function] with AWS SDK operations [operation_names]:

   **Context from Phase 1**:
   - Call chain: [chain from Phase 1]
   - SDK operation: [operation_name from Phase 1] (e.g., DynamoDB PutItem)
   - Functions: [file:line for each function in chain]

   **Strategy**: Classify code into SDK-related (keep) and unrelated (comment out).

   **For EACH function in call chain** (entry → intermediate → target):

   Step 1: Load function source code using Read
   - File: [function file path from Phase 1]
   - Read entire function body

   Step 2: Identify SDK-related code (KEEP)
   - All code that defines, constructs, or provides data to SDK operation
   - Variable declarations, assignments, function calls used by SDK input
   - Example: `input := &dynamodb.PutItemInput{...}`, `item := buildItem(entity)`

   Step 3: Identify unrelated code (COMMENT)
   - All code NOT used by SDK operation

   **Example**:
   ```go
   // SDK operation (already known from Phase 1: DynamoDB PutItem)
   result, err := dynamoDB.PutItem(ctx, &dynamodb.PutItemInput{
       TableName: aws.String("entities"),
       Item: item,
   })

   // SDK-related (KEEP) - builds SDK input
   item := buildItem(entity)
   entity := req.ToEntity()
   req := parseRequest(r)

   // Unrelated (COMMENT) - not used by SDK operation
   userData := userRepo.Get(ctx)
   logger.Info("processing")
   metrics.Increment("calls")
   ```

   Return format:
   ```
   Call chain: [entry] → [intermediate] → [target]
   SDK operations: [list]

   Function 1: [entry_function] at [file:line]
   SDK-related code (KEEP):
   - Lines X-Y: [description]

   Unrelated code (COMMENT):
   - Lines A-B: [description]
   - Lines C-D: [description]

   Function 2: [intermediate_function] at [file:line]
   SDK-related code (KEEP):
   - Lines M-N: [description]

   Unrelated code (COMMENT):
   - Lines P-Q: [description]
   ```"

   C. **Verify Step B identified code blocks (optional sanity check)**

   Quick verification that Step B produced actionable results:

   1. Check Step B output:
      - If "Code blocks to COMMENT" sections are empty for ALL functions: Output "検証: コメントアウトするコードなし"
      - If at least one "Code blocks to COMMENT" section has entries: Output "検証完了: コメントアウト対象 N個"

   2. Proceed to Step D (no halting condition)

   D. **Apply comment-out modifications with Edit tool**

   Check Step B result and apply modifications:

   1. If Step B identified zero code blocks to comment out:
      - Output: "スキップ: コメントアウトするコードなし"
      - Proceed to Step F (compilation verification)

   2. If Step B identified one or more code blocks to comment out:
      - For each function in call chain (entry → target):
        - For each code block marked as COMMENT:
          - Use Edit tool to comment out the block
          - Add simple comment: `// Commented out for testing: Unrelated to SDK operation`
          - Use `//` line comments for all lines

   Example:
   ```go
   // Original:
   userData, err := h.userRepo.GetUser(ctx, userID)
   if err != nil {
       return err
   }

   // Modified:
   // Commented out for testing: Unrelated to SDK operation
   // userData, err := h.userRepo.GetUser(ctx, userID)
   // if err != nil {
   //     return err
   // }
   ```

   3. Output after each edit:
      ```
      コメントアウト完了: [function_name] [file:line] (N blocks)
      ```

   E. **Replace commented-out code with dummy values if needed**

   Only if commented code returns values that cause compilation errors:

   Apply type-appropriate dummy values:
   - Strings: `"test-value"`
   - Integers: `1`
   - Dates: `"20250101"` or `time.Now()`
   - UUIDs: `uuid.MustParse("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee")`
   - Structs: Minimal initialization with required fields

   Example:
   ```go
   // Commented out for testing: Unrelated to SDK operation
   // businessDate := dateRepo.GetBusinessDate(ctx)
   businessDate := "20250101" // Dummy for testing
   ```

   Skip if commented code has no return values or values are unused

   F. **Verify compilation after modifications**
      - Run: `go build -o /tmp/test-build 2>&1`
      - If compilation fails:
        - Analyze error: unused variables, undefined references
        - Fix by commenting out dependent code or adding stubs
        - Retry until compilation succeeds
      - Output: "コンパイル成功: [file_path]"

   G. Display completion:
   ```
   完了 (i/N): コメントアウト処理
   - 処理した関数数: X個
   - コメントアウトしたブロック数: Y個
   - コンパイル: 成功
   ```

   H. Proceed to next chain automatically

### Phase 4: Simplified Test Data Preparation

7. **Process all chains in optimal combination sequentially**

   For each chain in optimal combination (index i from 1 to N):

   A. Display progress:
   ```
   === テストデータ準備中 (i/N) ===
   関数: [file_path:line_number] | [function_name] | [operations]
   ```

   B. Execute steps 8-10 for current chain

8. **Analyze AWS SDK operation with Task tool** (subagent_type=general-purpose)
   Task prompt: "For function [function_name] at [file_path:line_number] with call chain [selected_chain]:

   **Context from Phase 1**:
   - SDK operation: [operation_name from Phase 1] (e.g., DynamoDB PutItem)
   - Operation type: [Read/Write/Delete from operation name]
   - Function location: [file:line from Phase 1]

   1. Extract AWS settings from [function_name] using Read:
      - Region: look for `WithRegion\|AWS_REGION`
      - Resource: table name, bucket name from client call parameters
      - Endpoint: look for `WithEndpointResolver\|endpoint`

   2. Document v1 → v2 changes from git diff:
      - Client init: session.New vs config.LoadDefaultConfig
      - API call: old vs new method signature
      - Type changes: aws.String vs direct string usage

   Return: AWS settings, migration summary. Use [selected_chain] as call chain (do not re-trace)."

9. **Generate minimal test data with Task tool** (subagent_type=general-purpose)

   Task prompt: "For target function [function_name] with AWS SDK operation [operation_name]:

   **Context from Phase 1**:
   - Operations in chain: [list from Phase 1] (e.g., [Query, UpdateItem, TransactWriteItems])
   - Operation types: [Read/Write/Delete for each operation]

   **Pre-insert requirement** (based on Phase 1 operation types):
   - ANY Read operation → Generate Pre-insert for EACH Read
   - ANY Delete operation → Generate Pre-insert for EACH Delete target
   - Only Write operations → No Pre-insert needed

   Example:
   ```
   Operations from Phase 1: [Query (Read), UpdateItem (Write), TransactWriteItems (Write)]
   Result: Generate Pre-insert for Query operation only
   ```

   If operation type is Read or Delete (from Phase 1):
   1. Use Read to identify AWS SDK call parameters (table name, key, bucket, etc.)
   2. Generate 1-2 minimal test records matching those parameters
   3. Use correct Go types (pointer allocation with `aws.String`, `aws.Int64`, etc.)
   4. Generate Pre-insert code (PutItem/PutObject) before AWS SDK call

   Return:
   - Pre-insert code snippet with proper indentation
   - Insertion line number (line before AWS SDK call)
   - Required imports (aws SDK packages)

   If operation type is Write (from Phase 1):
   - Return: \"No Pre-insert needed for Write operation\""

10. **Apply Pre-insert code with Edit tool**

   If operation type is Read or Delete (from Phase 1):

   A. Insert Pre-insert code from step 9:
      - Use Edit tool to insert before AWS SDK operation
      - old_string: line before AWS SDK call
      - new_string: line + "\n" + Pre-insert code
      - Output: "Pre-insertコード追加: [file:line]"

   B. Verify compilation:
      - Run: `go build -o /tmp/test-build 2>&1`
      - If fails:
        - Analyze error messages
        - Fix missing imports, type mismatches, undefined fields
        - Retry Edit tool with corrections
        - Repeat until compilation succeeds
      - Output: "コンパイル成功: [file_path]"

   C. **Verify Pre-insert code for all Read/Delete operations (MANDATORY)**

   Run the following verification for the processed chain:

   1. Search for Read/Delete operations using Grep:
      - Pattern: `client\.(Query|GetItem|GetObject|Scan|DeleteItem|DeleteObject)`
      - In files: processed chain files only

   2. For each Read/Delete operation found:
      - Check lines before operation (within 20 lines)
      - Search for Pre-insert code patterns:
        - Comment: `// Pre-insert test data`
        - Code: `PutItem.*Input` or `PutObject.*Input`

   3. If Pre-insert missing for any operation:
      - List operations without Pre-insert
      - ERROR: "Phase 4 incomplete - Pre-insert code missing"
      - HALT processing for this chain

   4. If all operations have Pre-insert:
      - Output: "検証完了: Pre-insertコード生成済み (N operations)"
      - Proceed to next chain

   Example output:
   ```
   検証実行中: Pre-insertコードの生成チェック
   - Query at repository.go:123 - Pre-insert: NOT FOUND
   - GetItem at gateway.go:234 - Pre-insert: FOUND
   ERROR: Phase 4 incomplete - 1 operation missing Pre-insert code
   ```

   D. Display completion:
      ```
      完了 (i/N): テストデータ準備
      - Pre-insertコード: 追加済み / 不要
      - コンパイル: 成功
      ```

   E. Proceed to next chain automatically

11. **Proceed to next chain automatically**
   - If i < N: continue to next chain (repeat from step 6.A)
   - If i = N: proceed to Phase 5

### Phase 5: Final Summary

12. **Output final summary report**
    After processing all chains, display comprehensive summary:

    ```
    === 処理完了サマリー ===

    処理したSDK関数: N個
    - Read操作: X個 (Pre-insertコード追加済み)
    - Write操作: Y個
    - Delete操作: Z個 (Pre-insertコード追加済み)
    - 複数SDK使用: W個

    コメントアウトしたファイル: (Phase 3で処理)
    - [file_path_1] (X blocks)
    - [file_path_2] (Y blocks)
    ...

    書き換えたファイル: (Phase 4で処理)
    - [file_path_1]
    - [file_path_2]
    ...

    コンパイル: 成功P個 / 失敗Q個

    次のステップ:
    1. Step 13でAWS環境での動作確認手順を出力
    2. git diffで変更内容を確認
    3. AWS環境で検証実行
    4. 完了後に `git restore .` で変更を戻す
    ```

13. **Generate AWS verification procedures section**

    After final summary, output detailed AWS-specific verification procedures grouped by execution method:

    ```markdown
    ## AWS環境での動作確認方法

    ### 検証ログあり（優先確認）

    #### 1. [Execution Method] (例: POST /v1/entities, aws ecs run-task --task-definition process-data)
    **検証内容**: [Summary of what is being verified]
    **検証対象関数**:
    - [file:line] FunctionName1 | [Operation1]
    - [file:line] FunctionName2 | [Operation2]

    **呼び出しチェーン**:
    ```
    [Entry Point]
    → [Handler file:line] HandlerMethod
    → [Service file:line] ServiceMethod
    → [Target file:line] FunctionName1
    → AWS SDK API (Operation1)
    ```

    **AWS環境での確認方法**:
    ```bash
    # Example: API call
    curl -X POST https://api.example.com/v1/entities \
      -H "Content-Type: application/json" \
      -H "Bearer: jwt=<token>" \
      -d '{"param":"value"}'

    # Or: ECS task
    aws ecs run-task \
      --cluster production-cluster \
      --task-definition process-data:latest \
      --launch-type FARGATE
    ```

    **期待されるログ**:
    ```
    [INFO] Test records inserted: N match (should be retrieved), M non-match (should be excluded)
    [INFO] FunctionName1 returned N records (expected: N)
    ```

    **X-Ray確認ポイント**:
    - [Service] [Operation1] × N回
    - [Service] [Operation2] × M回 (connected to Operation1)
    - [Service] [Operation3] × M回 (connected flow)
    - Data flow: Operation1 result → Operation2 input → Operation3
    - FilterExpressionが正しく動作（該当する場合）

    **連続実行の確認**:
    - 複数SDK操作が順次実行され、すべて成功することを確認
    - データが正しく連携していることをログで確認
    - 独立したSDK操作（コメントアウト済み）は実行されないことを確認
    ```

    ### Verification Method Grouping Policy

    Group verification methods by execution method, not by function:

    **Good (execution method-based)**:
    ```
    ### API Method: POST /v1/entities
    Verifies: FunctionA (PutItem), FunctionB (Query)

    curl -X POST https://api.example.com/v1/entities ...
    ```

    **Bad (function-based, duplicates commands)**:
    ```
    ### FunctionA
    curl -X POST https://api.example.com/v1/entities ...

    ### FunctionB
    curl -X POST https://api.example.com/v1/entities ...
    ```

    When multiple functions share the same API/task:
    1. Group them under single execution method
    2. List all verified functions with their operations
    3. Provide single execution command
    4. Document expected outcomes for each function

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

### Comment-out Summary (Phase 3)
```
=== コメントアウト完了 (i/N) ===

関数: [file_path:line_number] | [function_name]
- コメントアウトしたブロック数: X個
- 処理した関数数: Y個 (entry → intermediate → target)
- コンパイル: 成功
```

### Test Data Preparation Summary (Phase 4)
```
=== テストデータ準備完了 (i/N) ===

関数: [file_path:line_number] | [function_name]
- AWS操作: [Read/Write/Delete] ([operation_name])
- Pre-insertコード: 追加済み / 不要
- コンパイル: 成功
```

### Final Summary (Phase 5)
```
=== 処理完了サマリー ===

処理したSDK関数: 5個
- Read操作: 1個 (Pre-insertコード追加済み)
- Write操作: 3個
- Delete操作: 0個
- 複数SDK使用: 2個

コメントアウトしたファイル: (Phase 3で処理)
- internal/service/order.go (3 blocks)
- internal/gateway/data.go (2 blocks)
- internal/repository/user.go (1 block)
- internal/gateway/s3.go (1 block)

書き換えたファイル: (Phase 4で処理)
- internal/repository/user.go (Pre-insert追加)

コンパイル: 成功5個 / 失敗0個

次: AWS環境での動作確認手順を参照（Step 13の出力）
```

## Analysis Requirements

### General
- Focus on production AWS connections (exclude localhost/test endpoints)
- Extract resource names (table names, bucket names, queue URLs)
- Identify region configuration (explicit config or AWS_REGION env var)
- Summarize v1 → v2 migration patterns clearly
- Provide actionable AWS console verification steps

### Output Focus Guidelines

**Include (AWS-specific verification)**:
- AWS API endpoints for verification
- ECS task run commands with aws-cli
- X-Ray trace points
- Expected AWS SDK call sequences

**Exclude (non-AWS or environment setup)**:
- Local development setup (Docker Compose, DynamoDB Local)
- Environment variable configuration (.env files)
- General prerequisites (Go version, make commands)
- Authentication setup procedures
- Repository cloning or dependency installation

### Critical Process Steps
Detailed instructions are in Process section above. Key requirements:
- **Comment-out unrelated code (Phase 3, step 6)**: Identify and comment out code blocks unrelated to target AWS SDK operation
- **Simplified test data generation (Phase 4, step 9)**: Generate minimal test data without complex analysis
- **Compilation verification (Phase 4, step 11)**: Run `go build`, fix errors automatically

### Batch Processing UX
- Phase 1: Extract and deduplicate automatically
- Phase 2: Single batch approval via AskUserQuestion
- Phase 3: Comment out unrelated code automatically for all chains
- Phase 4: Generate test data and verify compilation automatically for all chains
- Phase 5: Display final summary
- No interactive loop - fully automated after approval

## Notes

### Tool Usage
- Stop immediately if branch diff does not contain `aws-sdk-go-v2` imports
- Use Task tool for code analysis (steps 2, 3, 6, 8, 9)
- Use Edit tool to automatically apply code modifications (steps 6, 10)
- Use Bash tool for compilation verification (steps 6, 10)

### Key Process Steps
- **Deduplication** (step 3): Group by AWS_service + SDK_operation, ignore parameters. Select shortest chain from each group.
- **Comment-out unrelated code** (step 6): Identify code blocks unrelated to target AWS SDK operation, comment out with reason, verify compilation.
- **Simplified test data generation** (step 9): Skip detailed analysis, generate minimal test data (1-2 records) with correct Go types for Read/Delete operations only.
- **Apply Pre-insert code** (step 10): Insert test data preparation code, verify compilation, automatically proceed to next chain.

### Output Guidelines
- Include file:line references for navigation
- Provide complete call chains for traceability
- Mark chains with multiple SDK methods with [★ Multiple SDK] indicator
- Display progress (i/N) during batch processing
- Show final summary with compilation status

For detailed requirements, see Analysis Requirements section above.
