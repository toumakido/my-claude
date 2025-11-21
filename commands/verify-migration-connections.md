Verify AWS SDK v2 migration by temporarily modifying code for focused testing

Output language: Japanese, formal business tone

## Command Purpose

This command **prepares code for AWS SDK v2 connection testing** by **temporarily modifying migrated code** to enable focused verification:

**What this command does:**
1. **Extract and analyze** migrated functions from current branch (already migrated to AWS SDK v2)
2. **Comment out unrelated code** in call chains to isolate target AWS SDK operations (Phase 3)
   - Removes other AWS SDK calls, external APIs, logging, metrics, etc.
   - Keeps only the data flow directly related to target AWS SDK operation
   - **Why necessary**: Prevents unrelated code from interfering with targeted AWS SDK testing
3. **Generate minimal test data** and pre-insert code to DynamoDB/S3 (Phase 4)
4. **Output verification procedures** for AWS environment testing (Phase 5)

**What you need to do after this command:**
1. Review modified files in git diff
2. Deploy modified code to AWS test environment
3. Execute verification procedures from Step 14 output
4. **Manually run `git restore .` to revert all changes** after verification completes

**Important**:
- This command **modifies production code** in your local working tree
- Changes are **NOT automatically reverted** - you must run `git restore .` manually
- The goal is to create a testable state, not permanent test code
- Phase 3 comment-out is **required** to isolate AWS SDK operations for testing

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

   Step 2: For each function, trace COMPLETE call chains including entry point using Grep:

   Entry point identification:
   - API handlers: Extract HTTP method and route path from router definition
     - Search for route registration: router.POST, router.GET, http.HandleFunc, etc.
     - Extract full path: /v1/resources, /api/v2/items, etc.
   - Task entry points: Extract command/binary name from cmd/ directory
     - Example: cmd/process_task/main.go → process_task command
   - CLI commands: Extract subcommand name and arguments

   Call chain tracing:
   - Find entry points: `main\(`, `handler\(`, `ServeHTTP`, `Handle`
   - Search function references to trace call paths
   - Identify intermediate layers (usecase/service/repository/gateway)
   - Build complete chains: entry → intermediate → SDK function
   - For each chain, count all AWS SDK v2 method calls within the chain
   - Mark chains with multiple SDK methods as high priority
   - Execute Grep searches in parallel for independent functions

   Step 2.5: Verify active callers (exclude unused implementations)

   For each extracted function:
   1. Use Grep to search for function calls (exclude *_test.go, mocks/*.go)
   2. Count active call sites in production code (handlers, tasks, services)
   3. If active callers = 0:
      - Mark as "SKIP - No active callers (implementation only)"
      - Exclude from call chain list
      - Log for reference

   Example verification:
   ```bash
   # Search for function calls
   grep -r "FunctionName" --include="*.go" --exclude="*_test.go" --exclude-dir="mocks"

   # Verify call sites are in production code
   grep -r "FunctionName" internal/api/handler internal/tasks cmd/
   ```

   **Multiple path handling**:
   When a function has multiple call chains, select simplest path:
   1. Selection priority (choose first match):
      - Fewest external dependencies (< 4 data sources preferred)
      - Shortest chain length (< 4 hops preferred)
      - Direct entry point (main function or simple handler)
   2. Exclude complex paths:
      - 6+ external dependencies (too many mocks needed)
      - 6+ chain hops (too deep call stack)
      - Multiple validation layers (complex to satisfy)
   3. Document path selection in output:
      ```
      Selected: POST /v1/entities (2 dependencies, 3 hops)
      Skipped: Background job path (7 dependencies, 5 hops) - too complex
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

   If call chain cannot be traced to entry point:
   - Mark function as "SKIP - No entry point found"
   - Exclude from optimal combination
   - Do NOT include in verification output
   - Log skipped functions for reference:
     ```
     スキップされた関数（呼び出し元不明）:
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

3. **Group and deduplicate chains with Task tool** (subagent_type=general-purpose)
   Task prompt: "Group and deduplicate call chains to create optimal combination:

   Step 1: Group by SDK operation type (動作確認の観点)
   - Key: AWS_service + SDK_operation
   - Value: list of call chains using same SDK operation
   - Example: all chains using 'DynamoDB PutItem' are grouped together
   - Ignore: file path, line number, function name, region, endpoint, table/bucket names, filters, and all parameters
   - Rationale: From operation verification perspective, only AWS service type and SDK operation matter

   Examples of grouping:
   ```
   Same group (same S3 GetObject):
   - s3.go:45 | DownloadFromBucketA | S3 GetObject (bucket: bucket-a)
   - s3.go:89 | DownloadFromBucketB | S3 GetObject (bucket: bucket-b)
   → Only verify one

   Same group (same DynamoDB Query):
   - user.go:30 | GetByStatus | DynamoDB Query (filter: status)
   - user.go:60 | GetByAge | DynamoDB Query (filter: age)
   → Only verify one
   ```

   Step 2: Select representative chain from each group
   For each group with multiple chains:
   - Priority 1: Shortest chain (fewest hops)
   - Priority 2: Entry point is 'main' function
   - Priority 3: First in list (tie-breaker)
   - Mark selected chain with: [+N other operations] where N = group size - 1

   For groups with single chain:
   - Select the only chain
   - No marker needed

   Step 3: Create optimal combination
   - Combine all selected representative chains
   - Maintain original priority sorting (from step 2)
   - Result: minimal set covering all unique SDK operation types

   Step 4: Verify entry points before finalizing
   For each selected representative chain:
   1. Confirm entry point is traceable and executable:
      - API handler: Verify route registration exists using Grep
      - Task command: Verify binary in cmd/ directory using Glob
      - CLI command: Verify subcommand definition using Grep
   2. If entry point cannot be confirmed:
      - Use Grep to search for function references in codebase
      - If no active references found: Mark as \"SKIP - No active callers\"
      - Remove from optimal combination
      - Document in skipped functions list
   3. Output skipped functions separately:
      ```
      以下の関数はエントリーポイントが不明なためスキップします:
      - [file:line] FunctionName | [Operation] | Reason: No route registration found
      ```

   Return: optimal combination (deduplicated chains with verified entry points), skipped functions list"

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

### Phase 3: Comment-out Unrelated Code

**IMPORTANT**: This phase MUST be executed for ALL chains. Do NOT skip this phase.
- Purpose: Isolate target AWS SDK operation by removing unrelated code that would interfere with testing
- This ensures only the target AWS SDK call executes during verification

6. **Comment out unrelated code in call chain functions (strict mode)**

   For each chain in optimal combination (index i from 1 to N):

   **CRITICAL**: Execute this step for EVERY chain. Do NOT skip even if you think the code is production-ready.

   A. Display progress:
   ```
   === コメントアウト処理中 (i/N) ===
   関数: [file_path:line_number] | [function_name] | [operations]
   ```

   B. **Identify unrelated code with Task tool** (subagent_type=general-purpose)
   Task prompt: "For call chain [entry_point → ... → target_function] with target AWS SDK operation [operation_name]:

   **CRITICAL**: You MUST identify and return code blocks to comment out. Do NOT skip this analysis.
   - Even if the code appears production-ready, you must analyze and identify unrelated blocks
   - If you cannot find any unrelated code, explicitly state 'No unrelated code found' with reasoning
   - Do NOT make assumptions about whether this step should be skipped

   Purpose: Comment out ALL code unrelated to target AWS SDK operation to enable focused testing

   **Strict mode**: Only keep code DIRECTLY related to target AWS SDK operation data flow

   For EACH function in call chain (entry → intermediate → target):

   1. Use Read to load function source code

   2. Identify target data flow path:
      - Target function: Identify variables/parameters used in AWS SDK call
      - Intermediate functions: Trace these variables backwards through function calls
      - Entry point: Trace to function parameters or immediate values

   3. Classify ALL code blocks as KEEP or COMMENT:

      **KEEP (directly related to target AWS SDK operation)**:
      - Variable declarations used in target data flow
      - Assignments to target data flow variables
      - Function calls that return values used in target data flow
      - Control flow (if/for/switch) that affects target data flow
      - Error returns after target data flow operations

      **COMMENT (unrelated to target AWS SDK operation)**:
      - Other AWS SDK service calls (different service or independent operation)
      - External API calls (HTTP clients, gRPC, etc.)
      - Database operations not in target data flow
      - Logging statements
      - Metrics collection
      - Validation not affecting target data flow
      - Business logic on independent variables
      - Side effects (notifications, cache updates, etc.)

   4. For each COMMENT block, document reason:
      ```go
      // Commented out for testing: [reason]
      // [original code]
      ```

   Return format for entire call chain:
   ```
   Call chain: [entry] → [intermediate] → [target]

   Function 1: [entry_function] at [file:line]
   Code blocks to comment out:
   - Lines X-Y: [reason] (e.g., \"DynamoDB call unrelated to target S3 operation\")
   - Lines A-B: [reason] (e.g., \"HTTP API call for external notification\")

   Function 2: [intermediate_function] at [file:line]
   Code blocks to comment out:
   - Lines M-N: [reason]

   Function 3: [target_function] at [file:line]
   Code blocks to comment out:
   - Lines P-Q: [reason]
   ```"

   C. **Apply comment-out modifications with Edit tool**

   **CRITICAL**: Apply ALL modifications identified in step B. Do NOT skip this step.
   - If step B returned 'No unrelated code found', output: "スキップ: コメントアウトするコードなし"
   - Otherwise, apply ALL comment-out modifications as specified below

   For each function in call chain (process in order: entry → target):

   1. For each code block identified as COMMENT:
      - Use Edit tool to comment out the block
      - old_string: original code block
      - new_string: commented code with reason
      - Use simple `//` line comments
      - Add reason comment: `// Commented out for testing: [reason]`

   Example:
   ```go
   // Original code:
   userData, err := h.userRepo.GetUser(ctx, userID)
   if err != nil {
       return err
   }

   // Modified code:
   // Commented out for testing: User data not used in target DynamoDB SaveEntity operation
   // userData, err := h.userRepo.GetUser(ctx, userID)
   // if err != nil {
   //     return err
   // }
   ```

   2. Output after each edit:
      ```
      コメントアウト完了: [function_name] [file:line] (N blocks commented)
      ```

   D. **Verify compilation after comment-out**
      - Run: `go build -o /tmp/test-build 2>&1`
      - If compilation fails:
        - Analyze error: unused variables, undefined references
        - Fix by commenting out dependent code or adding stubs
        - Retry until compilation succeeds
      - Output: "コンパイル成功: [file_path]"

   E. Display completion:
   ```
   完了 (i/N): コメントアウト処理
   - 処理した関数数: X個
   - コメントアウトしたブロック数: Y個
   - コンパイル: 成功
   ```

   F. Proceed to next chain automatically

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

9. **Generate minimal test data with Task tool** (subagent_type=general-purpose)

   Task prompt: "For target function [function_name] with AWS SDK operation [operation_name]:

   **Simplified approach**: Skip detailed analysis, generate minimal test data

   If AWS operation is Read or Delete:
   1. Use Read to identify AWS SDK call parameters (table name, key, bucket, etc.)
   2. Generate 1-2 minimal test records matching those parameters
   3. Use correct Go types (pointer allocation with `aws.String`, `aws.Int64`, etc.)
   4. Generate Pre-insert code (PutItem/PutObject) before AWS SDK call

   Return:
   - Pre-insert code snippet with proper indentation
   - Insertion line number (line before AWS SDK call)
   - Required imports (aws SDK packages)

   If AWS operation is Write:
   - Return: \"No Pre-insert needed for Write operation\"

   Keep it simple - no complex validation, no downstream analysis, no field requirement mapping."

10. **Apply Pre-insert code with Edit tool**

   If AWS operation is Read or Delete:

   A. Insert Pre-insert code from step 9:
      - Use Edit tool to insert before AWS SDK operation
      - old_string: line before AWS SDK call
      - new_string: line + "\n" + Pre-insert code
      - Output: "Pre-insertコード追加: [file:line]"

   B. Verify compilation:
      - Run: `go build -o /tmp/test-build 2>&1`
      - If fails: Fix imports/types and retry
      - Output: "コンパイル成功"

   C. Display completion:
      ```
      完了 (i/N): テストデータ準備
      - Pre-insertコード: 追加済み / 不要
      - コンパイル: 成功
      ```

   D. Proceed to next chain automatically

11. **Verify compilation with Bash tool**

   - Run: `go build -o /tmp/test-build 2>&1`
   - If compilation fails:
     - Analyze error messages
     - Fix missing imports, type mismatches, undefined fields
     - Retry Edit tool with corrections
     - Repeat until compilation succeeds
   - Output: "コンパイル成功: [file_path]"

12. **Output brief summary for current chain**
    ```
    完了 (i/N): [file_path]
    - AWS操作: [operation_type] ([operation_name])
    - Pre-insertコード: 追加済み / 不要
    - コンパイル: 成功 / 失敗
    ```

   Proceed to next chain automatically (no user interaction):
   - If i < N: continue to next chain (repeat from step 6.A)
   - If i = N: proceed to final summary

### Phase 5: Final Summary

13. **Output final summary report**
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
    1. Step 14でAWS環境での動作確認手順を出力
    2. git diffで変更内容を確認
    3. AWS環境で検証実行
    4. 完了後に `git restore .` で変更を戻す
    ```

14. **Generate AWS verification procedures section**

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
    - [Service] [Operation2] × M回
    - FilterExpressionが正しく動作（該当する場合）
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

次: AWS環境での動作確認手順を参照（Step 14の出力）
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
