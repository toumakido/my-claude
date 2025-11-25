Verify AWS SDK v2 migration by temporarily modifying code for focused testing

Output language: Japanese, formal business tone

## Command Purpose

Prepares code for AWS SDK v2 connection testing by temporarily modifying migrated code:

1. Extract migrated functions from current branch (Phase 1)
2. Comment out unrelated code to isolate AWS SDK operations (Phase 3)
3. Generate minimal test data (Phase 4)
4. Output AWS environment verification procedures (Phase 5)

**Post-execution steps:**
1. Review `git diff`
2. Deploy to AWS test environment
3. Execute verification procedures (Step 13 output)

**Critical notes:**
- Modifies production code in working tree (not automatically reverted)
- Creates testable state, not permanent test code

## Prerequisites

- Run from repository root
- Working tree can be dirty (uncommitted changes allowed)
- gh CLI installed and authenticated
- Git repository with AWS SDK v2 migration changes

## Process

### Phase 1: Extract Functions and Call Chains

1. **Validate branch has AWS SDK v2 changes**
   - Run: `git diff main...HEAD` and store result in variable
   - Search stored diff for pattern: `github.com/aws/aws-sdk-go-v2`
   - If not found: output "このブランチはAWS SDK Go v2関連の変更を含んでいません" and exit immediately

2. **Extract functions and call chains with Task tool** (subagent_type=general-purpose)
   **Context**: Use stored git diff from Step 1
   Task prompt: "Parse git diff and extract all functions/methods using AWS SDK v2 with their call chains.

   **Step 1: Extract SDK functions using Grep**

   Execute Grep searches in this exact order:
   1. `pattern: "github\.com/aws/aws-sdk-go-v2/service/"`, `output_mode: "files_with_matches"`
   2. `pattern: "client\.(PutItem|GetObject|Query|UpdateItem|DeleteItem|PutObject|DeleteObject|SendEmail|Publish)"`, `output_mode: "content"`, `-C: 10`
   3. Filter results: keep only functions with `context.Context` parameter in signature

   For each match, extract:
   - File path:line_number (from Grep output)
   - Function/method name (from function signature line)
   - AWS service (from import path: dynamodb, s3, ses, sns)
   - Operation (from client method call: PutItem, GetObject, etc.)

   **Step 2: Trace call chains**

   Prerequisites: Entry point must exist and be verified before tracing

   For each function from Step 1, trace COMPLETE call chains including entry point using Grep:

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
   - Trace from verified entry points using Grep: `pattern: "main\(|handler\(|ServeHTTP|Handle)"`, `output_mode: "content"`
   - Search function references: `pattern: "<function_name>\("`, `output_mode: "content"`
   - Identify intermediate layers (usecase/service/repository/gateway)
   - Build complete chains: verified_entry → intermediate → SDK function
   - Count all AWS SDK v2 method calls in chain
   - Record all SDK operations for grouped verification
   - **Parallel execution**: Execute Grep searches in single tool call for functions that:
     1. Have no shared intermediate layers (different service/repository files)
     2. Belong to different API endpoints or tasks
     3. Use different AWS services (e.g., DynamoDB vs S3)

   **Step 3: Verify active callers (execute in parallel with Step 2)**

   For each extracted function from Step 1:
   1. Use Grep: `pattern: "<function_name>\("`, `glob: "!(*_test.go|mocks/*.go)"`, `output_mode: "files_with_matches"`
   2. Count result files (active call sites in production code)
   3. If count = 0:
      - Mark as "SKIP - No active callers"
      - Exclude from call chain list

   **Parallel execution note**: Step 3 depends only on Step 1 results (function names). Execute Grep searches for caller verification in parallel with Step 2 call chain tracing to improve performance.

   **Step 4: Handle multiple paths**

   When function has 2+ call chains:
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

   **Step 5: Format output**

   CRITICAL: List EVERY function in the call chain with file:line, not just SDK function.
   Omitting intermediate functions will cause Phase 3 to miss external service calls and business logic.

   **Format for single SDK operation:**
   ```
   Entry: [type] [identifier]
   → [file:line] EntryFunction
   → [file:line] IntermediateFunction1
   → [file:line] IntermediateFunction2
   → [file:line] SDKFunction
   → AWS SDK v2 API (Operation)
   ```

   **Format for multiple SDK operations (hierarchical structure):**
   ```
   Chain: [entry_type] [identifier] [★ Multiple SDK: N operations]

   Entry → Intermediate layers:
   Entry: [type] [identifier]
   → [file:line] EntryFunction
   → [file:line] IntermediateFunction1
   → [file:line] IntermediateFunction2

   SDK Functions (Phase 3 targets):
   [ChainID]-A. [file:line] SDKFunction1
        Operation: [Service] [Operation1]

   [ChainID]-B. [file:line] SDKFunction2
        Operation: [Service] [Operation2]
   ```

   Example (single SDK operation):
   ```
   Entry: Task task_name
   → cmd/task_name/main.go:100 main
   → internal/tasks/task_worker.go:50 Execute
   → internal/service/service_name.go:80 ProcessData
   → DynamoDB UpdateItem
   ```

   Example (multiple SDK operations):
   ```
   Chain: Task task_name [★ Multiple SDK: 3 operations]

   Entry → Intermediate layers:
   Entry: Task task_name
   → cmd/task_name/main.go:100 main
   → internal/tasks/task_worker.go:50 Execute
   → internal/service/service_name.go:80 ProcessData

   SDK Functions (Phase 3 targets):
   1-A. internal/service/service_name.go:120 createRecord
        Operation: DynamoDB PutItem

   1-B. internal/service/service_name.go:150 updateRecord
        Operation: DynamoDB UpdateItem

   1-C. internal/service/service_name.go:180 processTransaction
        Operation: DynamoDB TransactWriteItems
   ```

   BAD example (incomplete, missing intermediate functions):
   ```
   Entry: Task task_name
   → cmd/task_name/main.go main
   → internal/service/service_name.go:80 ProcessData
   → DynamoDB UpdateItem
   ```

   **Validation**: After Task tool completes, verify output includes:
   - Entry function with file:line
   - ALL intermediate functions with file:line
   - ALL SDK functions with file:line (for multiple SDK operations, list separately under "SDK Functions")
   - If any function lacks file:line, re-run Step 2 with explicit instruction to trace ALL functions

   If entry point not verified (marked in Step 2):
   - Exclude from call chain list immediately
   - Log as skipped function:
     ```
     スキップされた関数（エントリーポイント不明）:
     - internal/service/file.go:123 FunctionName
     ```

   **Step 6: Sort by priority**
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
   Task prompt: "Deduplicate call chains using coverage-based selection algorithm:

   **Objective**: Select minimum chains covering all unique AWS SDK operations. Prioritize chains with multiple SDK operations.

   **Algorithm**:
   ```
   covered_operations = {} (empty set)
   selected_chains = []

   1. Sort chains by:
      - Primary: SDK operation count (descending)
      - Secondary: Chain length (ascending)

   2. For each chain in sorted order:
      operations = extract_all_sdk_operations(chain)  // Format: "Service Operation" (e.g., "DynamoDB PutItem")
      new_operations = operations - covered_operations

      if new_operations is NOT empty:
         SELECT chain
         selected_chains.append(chain)
         covered_operations.update(operations)
         mark_duplicate_count(chain)  // Add [+N similar chains] marker
      else:
         SKIP chain  // All operations already covered

   3. Filter selected chains:
      For each chain in selected_chains:
         if chain.entry_point == "SKIP - No entry point":
            REMOVE chain from selected_chains

   4. Return:
      - selected_chains (optimal combination)
      - skipped_chains with reasons
   ```

   **Operation comparison**: Treat operations as identical if Service + Operation name match. Ignore parameters (table names, filters) as they don't affect SDK v2 migration verification.

   Return: optimal combination (deduplicated chains with verified entry points), skipped chains list with skip reasons"

4. **Format and cache optimal combination**
   Store Task result in variable for batch processing.

   **Output format for single SDK operation:**
   ```
   [N]. Chain: [entry_type] [identifier]

   Entry: [type] [identifier]
   → [file:line] EntryFunction
   → [file:line] IntermediateFunction
   → [file:line] SDKFunction
   → [Service] [Operation]

   (1 SDK operation, [hop_count] hops) Active callers: [count]箇所
   ```

   **Output format for multiple SDK operations:**
   ```
   [N]. Chain: [entry_type] [identifier] [★ Multiple SDK: N operations]

   Entry → Intermediate layers:
   Entry: [type] [identifier]
   → [file:line] EntryFunction
   → [file:line] IntermediateFunction

   SDK Functions (Phase 3 targets):
   [N]-A. [file:line] SDKFunction1
          Operation: [Service] [Operation1]

   [N]-B. [file:line] SDKFunction2
          Operation: [Service] [Operation2]

   ([N] SDK operations, [hop_count] hops) Active callers: [count]箇所
   ```

   Example (single SDK operation):
   ```
   1. Chain: Task task_name

   Entry: Task task_name
   → cmd/task_name/main.go:100 main
   → internal/tasks/task_worker.go:50 Execute
   → internal/service/service_name.go:80 ProcessData
   → DynamoDB UpdateItem

   (1 SDK operation, 4 hops) Active callers: 3箇所
   ```

   Example (multiple SDK operations):
   ```
   2. Chain: Task task_name [★ Multiple SDK: 3 operations]

   Entry → Intermediate layers:
   Entry: Task task_name
   → cmd/task_name/main.go:100 main
   → internal/tasks/task_worker.go:50 Execute
   → internal/service/service_name.go:80 ProcessData

   SDK Functions (Phase 3 targets):
   2-A. internal/service/service_name.go:120 createRecord
        Operation: DynamoDB PutItem

   2-B. internal/service/service_name.go:150 updateRecord
        Operation: DynamoDB UpdateItem

   2-C. internal/service/service_name.go:180 processTransaction
        Operation: DynamoDB TransactWriteItems

   (3 SDK operations, 5 hops) Active callers: 2箇所
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
   1. [Entry Point]
      → [Handler/Task file:line] HandlerMethod
      → [Service file:line] ServiceMethod
      → [Target file:line] TargetFunction
      → AWS SDK v2 API (Operation)
   2. [Next Entry Point]
      → [file:line] Method
      → ...
      → AWS SDK v2 API (Operation)

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
      - Chains with Update/Read/Delete operations (need Pre-insert)
      - Chains with single Create operations (lowest priority)

   3. Skip criteria (only apply after meeting mandatory minimum):
      - Chain has 6+ external dependencies
      - Chain has 6+ hops (deep call stack)
      - Processing single chain exceeds 15 minutes

   4. When skipping a chain:
      - Output: "スキップ: [reason] (Chain N: [description])"
      - Document skipped chains in final summary
      - Continue with remaining chains

### Phase 3: Comment-out Unrelated Code

**Objective**: Minimize code to focus ONLY on AWS SDK connection verification.
Keep only: SDK client init, SDK input construction, SDK call, minimal error check.
Comment out: Response processing, business logic, external calls, detailed error handling.

**What to verify in connection testing:**
- SDK call succeeds (no error)
- Connects to correct resource (table name, bucket name)
- X-Ray trace is recorded

**What NOT to verify (comment out these):**
- Response data accuracy
- Business logic correctness
- Entity transformation correctness

**Approach**:
Keep only SDK-related code, comment out everything else:
1. Identify SDK operation and its input variables (e.g., `dynamoDB.PutItem(ctx, &input)`)
2. Identify SDK-related code: code that builds SDK input
3. Comment out unrelated code: code NOT used by SDK operation
4. Comment out response processing beyond connection verification
5. Replace detailed error handling with simple logging

6. **Comment out unrelated code in call chain functions**

   For each chain in optimal combination (index i from 1 to N):

   A. Display progress:
   ```
   === コメントアウト処理中 (i/N) ===
   関数: [file_path:line_number] | [function_name] | [operations]
   ```

   B. **Identify SDK-related code and unrelated code** (Task tool: subagent_type=general-purpose)

   CRITICAL: Analyze ALL functions in call chain from Phase 1, including:
   1. Entry function (main.go, task entry)
   2. ALL intermediate functions (handler, usecase, task worker methods)
   3. SDK operation function (repository, gateway)

   DO NOT analyze only the SDK function. Intermediate layers often contain:
   - External service calls (HTTP/gRPC) ← MUST BE COMMENTED OUT
   - Business logic unrelated to SDK
   - Data preparation from non-AWS sources

   Task prompt: "For call chain [entry_point → ... → target_function] with AWS SDK operations [operation_names]:

   **Tools to use**: Read tool for loading source code, no Grep/Glob needed

   **Context from Phase 1**:
   - Complete call chain (copy from Phase 1 output with ALL functions listed)
   - SDK operation: [operation_name]

   **Functions to analyze** (MUST analyze ALL, not just SDK function):

   Function 1: [entry_function] at [file:line]
   Function 2: [intermediate_function_1] at [file:line]
   Function 3: [intermediate_function_2] at [file:line]
   Function 4: [sdk_function] at [file:line]

   For EACH function above:
   Step 1: Load function source code
   Step 2: Identify SDK-related code (KEEP)
   Step 3: Identify unrelated code (COMMENT)

   Common unrelated code in intermediate layers:
   - External HTTP/gRPC calls (e.g., externalServiceRepo.GetEntitiesByID)
   - Validation not related to SDK input
   - Data enrichment from non-AWS sources
   - Concurrent processing logic (goroutines, waitgroups) unrelated to SDK call

   **Objective**: Classify code into SDK-related (KEEP) and unrelated (COMMENT).

   **Classification criteria**:

   SDK-related code (KEEP) - Code that directly contributes to SDK operation:
   1. SDK client initialization (e.g., `dynamodb.New()`, `s3.NewFromConfig()`)
   2. SDK input struct construction (e.g., `&dynamodb.PutItemInput{...}`)
   3. Data transformation for SDK input (variables used in input fields)
   4. Context handling for SDK calls (e.g., `ctx` parameter)
   5. MINIMAL error check (if err != nil { log; return })
   6. SUCCESS confirmation logging (log.Printf("Operation succeeded"))

   DO NOT KEEP (must comment out):
   - Detailed error wrapping (apperrors.Wrap, utils.GetFunctionName)
   - Response parsing (parseAttributes, for-loops over resp.Items)
   - Entity transformation (ToEntity, domain model conversion)
   - Pagination logic (ExclusiveStartKey handling)
   - Business logic using response data

   Unrelated code (COMMENT):
   1. Logging statements
   2. Metrics/monitoring
   3. External service calls
   4. Validation logic not related to SDK input
   5. Business logic after SDK operation completes
   6. Cache operations
   7. Response parsing into domain entities (NEW)
   8. Detailed error wrapping (keep only basic error check) (NEW)
   9. Entity transformation (parseAttributes, ToEntity, etc.) (NEW)
   10. Pagination loops (NEW)

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

   C. **Apply comment-out modifications with Edit tool**

   Check Step B result and apply modifications:

   1. If Step B identified zero code blocks to comment out:
      - Output: "スキップ: コメントアウトするコードなし"
      - Proceed to Step F (compilation verification)

   2. If Step B identified one or more code blocks to comment out:
      - For each function in call chain (entry → target):
        - For each code block marked as COMMENT:
          - Use Edit tool: Replace block with commented version
          - Format: Prefix marker line + comment out all lines with `//`
          - Marker: `// Commented out for testing: Unrelated to SDK operation`

   Example Edit tool usage:
   ```go
   old_string:
   userData, err := h.userRepo.GetUser(ctx, userID)
   if err != nil {
       return err
   }

   new_string:
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

   D. **Replace commented-out code with dummy values if needed**

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

   E. **Verify compilation after modifications**
      - Run: `go build -o /tmp/test-build 2>&1`
      - If compilation fails:
        - Analyze error: unused variables, undefined references
        - Fix by commenting out dependent code or adding stubs
        - Retry until compilation succeeds
      - Output: "コンパイル成功: [file_path]"

   F. Display completion and proceed to next chain:
   ```
   完了 (i/N): コメントアウト処理
   - 処理した関数数: X個
   - コメントアウトしたブロック数: Y個
   - コンパイル: 成功
   ```

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

   **Tools to use**: Read tool for source code, Grep tool for searching patterns

   **Context from Phase 1**:
   - SDK operation: [operation_name from Phase 1] (e.g., DynamoDB PutItem)
   - Operation type: Classify from operation name:
     - Create: PutItem, PutObject, SendEmail, Publish
     - Update: UpdateItem, TransactWriteItems
     - Read: Query, GetItem, Scan, GetObject
     - Delete: DeleteItem, DeleteObject
   - Function location: [file:line from Phase 1]

   1. Extract AWS settings from [function_name] using Read and Grep:
      - Region: Use Grep with `pattern: "WithRegion|AWS_REGION"`, `output_mode: "content"`, `-C: 5`
      - Resource: Read SDK call parameters for table name, bucket name
      - Endpoint: Use Grep with `pattern: "WithEndpointResolver|endpoint"`, `output_mode: "content"`, `-C: 5`

   2. Document v1 → v2 changes from git diff:
      - Client init: session.New vs config.LoadDefaultConfig
      - API call: old vs new method signature
      - Type changes: aws.String vs direct string usage

   Return: AWS settings, migration summary. Use [selected_chain] as call chain (do not re-trace)."

9. **Generate minimal test data with Task tool** (subagent_type=general-purpose)

   Task prompt: "For target function [function_name] with AWS SDK operation [operation_name]:

   **Tools to use**: Read tool to extract SDK call parameters

   **Context from Phase 1**:
   - Operations in chain: [list from Phase 1] (e.g., [Query, UpdateItem, TransactWriteItems])
   - Operation types: Classify each operation:
     - Create: PutItem, PutObject, SendEmail, Publish
     - Update: UpdateItem, TransactWriteItems
     - Read: Query, GetItem, Scan, GetObject
     - Delete: DeleteItem, DeleteObject

   **Pre-insert requirement** (based on Phase 1 operation types):
   - Create operations → No Pre-insert needed (creates new data)
   - Update operations → Generate Pre-insert for EACH Update (requires existing data)
   - Read operations → Generate Pre-insert for EACH Read (requires existing data)
   - Delete operations → Generate Pre-insert for EACH Delete (requires existing data)

   Example:
   ```
   Operations from Phase 1: [Query (Read), UpdateItem (Update), PutItem (Create)]
   Result: Generate Pre-insert for Query and UpdateItem operations
   ```

   If operation type is Update, Read, or Delete (from Phase 1):
   1. Use Read to identify AWS SDK call parameters (table name, key, bucket, etc.)
   2. Generate 1-2 minimal test records matching those parameters
   3. Use correct Go types (pointer allocation with `aws.String`, `aws.Int64`, etc.)
   4. Generate Pre-insert code (PutItem/PutObject) before AWS SDK call

   Return:
   - Pre-insert code snippet with proper indentation
   - Insertion line number (line before AWS SDK call)
   - Required imports (aws SDK packages)

   If operation type is Create (from Phase 1):
   - Return: \"No Pre-insert needed for Create operation\""

10. **Apply Pre-insert code with Edit tool**

   If operation type is Update, Read, or Delete (from Phase 1):

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

   C. Display completion and proceed to next chain:
      ```
      完了 (i/N): テストデータ準備
      - Pre-insertコード: 追加済み / 不要
      - コンパイル: 成功
      ```

   D. **Verify Pre-insert code for all Update/Read/Delete operations**

   Verify Pre-insert code was generated for operations requiring existing data:

   1. Use Grep to search for Update/Read/Delete operations:
      - `pattern: "client\.(UpdateItem|TransactWriteItems|Query|GetItem|GetObject|Scan|DeleteItem|DeleteObject)"`, `output_mode: "content"`, `-B: 20`
      - `path: [processed chain file paths]`

   2. For each Update/Read/Delete operation in results:
      - Check preceding 20 lines (from Grep -B output)
      - Search for Pre-insert patterns: `// Pre-insert test data`, `PutItem.*Input`, `PutObject.*Input`

   3. Verification results:
      - All operations have Pre-insert → Output: "検証完了: Pre-insertコード生成済み (N operations)"
      - Missing Pre-insert → ERROR: "Phase 4 incomplete - Pre-insert code missing", HALT processing

   Example output:
   ```
   検証実行中: Pre-insertコードの生成チェック
   - UpdateItem at repository.go:100 - Pre-insert: FOUND
   - Query at repository.go:123 - Pre-insert: NOT FOUND
   - GetItem at gateway.go:234 - Pre-insert: FOUND
   ERROR: Phase 4 incomplete - 1 operation missing Pre-insert code
   ```

11. **Automatic progression**
   - If i < N: continue to next chain (repeat from step 7.A)
   - If i = N: proceed to Phase 3.5

### Phase 3.5: Verify Comment-out Completeness

After completing Phase 3 for all chains, verify:

1. Check entry/intermediate functions were analyzed:
   - Use Grep: `pattern: "Commented out for testing"`, `glob: "cmd/**/*.go"`, `output_mode: "files_with_matches"`
   - Use Grep: `pattern: "Commented out for testing"`, `glob: "internal/tasks/**/*.go"`, `output_mode: "files_with_matches"`
   - If no matches: WARNING - intermediate functions may not have been analyzed

2. Verify external service calls are commented:
   - Use Grep: `pattern: "http\\.(Get|Post|Client)|grpc\\.(Dial|NewClient)"`, `output_mode: "content"`, `-C: 5`, `glob: "!(*_test.go)"`
   - For each match in analyzed files: Check if preceded by "Commented out for testing"
   - If uncommented external calls found: ERROR - re-run Phase 3

3. Verify response processing is minimal:
   - Use Grep: `pattern: "parseAttributes|ToEntity|for.*resp\\.(Items|Records)"`, `output_mode: "content"`, `glob: "!(*_test.go)"`
   - For each match: Check if commented out or replaced with log.Printf
   - If complex processing remains: ERROR - re-run Phase 3

Output:
```
=== Phase 3検証結果 ===

✓ Entry/intermediate functions analyzed: 3 files
✓ External service calls commented: 2箇所
✓ Response processing minimized: 4箇所

全てのコードが接続確認に最適化されました
```

### Phase 5: Final Summary

12. **Output final summary report**
    After processing all chains, display comprehensive summary:

    ```
    === 処理完了サマリー ===

    処理したSDK関数: N個
    - Create操作: A個
    - Update操作: B個 (Pre-insertコード追加済み)
    - Read操作: X個 (Pre-insertコード追加済み)
    - Delete操作: Z個 (Pre-insertコード追加済み)
    - 複数SDK使用: W個

    コメントアウトしたファイル: (Phase 3で処理)
    - [file_path_1] (X blocks)
    - [file_path_2] (Y blocks)
    ...

    書き換えたファイル: (Phase 4で処理)
    - [file_path_1] (Pre-insert追加)
    - [file_path_2] (Pre-insert追加)
    ...

    コンパイル: 成功P個 / 失敗Q個

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

合計SDK関数数: 7個 (重複排除前: 10個のチェーン)
選択されたチェーン数: 4個

[Sorted by priority: multiple SDK methods first, then by chain length]

1. Chain: Task task_name_1 [★ Multiple SDK: 3 operations]

   Entry → Intermediate layers:
   Entry: Task task_name_1
   → cmd/task_name_1/main.go:100 main
   → internal/tasks/task_worker.go:50 Execute
   → internal/service/service_name.go:80 ProcessData

   SDK Functions (Phase 3 targets):
   1-A. internal/service/service_name.go:120 createRecord
        Operation: DynamoDB PutItem

   1-B. internal/service/service_name.go:150 storeFile
        Operation: S3 PutObject

   1-C. internal/service/service_name.go:180 sendMessage
        Operation: SES SendEmail

   (3 SDK operations, 5 hops) Active callers: 2箇所

2. Chain: POST /v1/resource/action [★ Multiple SDK: 2 operations] [+1 other chain]

   Entry → Intermediate layers:
   Entry: API POST /v1/resource/action
   → internal/api/handler/v1/handler_name.go:80 HandleAction
   → internal/gateway/gateway_name.go:89 ProcessAction

   SDK Functions (Phase 3 targets):
   2-A. internal/gateway/gateway_name.go:120 fetchData
        Operation: S3 GetObject

   2-B. internal/gateway/gateway_name.go:200 saveBatch
        Operation: DynamoDB BatchWriteItem

   (2 SDK operations, 4 hops) Active callers: 1箇所

3. Chain: Task task_name_2 [+2 other chains]

   Entry: Task task_name_2
   → cmd/task_name_2/main.go:50 main
   → internal/usecase/usecase_name.go:30 Execute
   → internal/repository/repository_name.go:45 Save
   → DynamoDB PutItem

   (1 SDK operation, 3 hops) Active callers: 3箇所

4. Chain: API GET /v1/resource/:id

   Entry: API GET /v1/resource/:id
   → internal/api/handler/v1/handler_name.go:100 GetResource
   → internal/service/service_name.go:50 Fetch
   → internal/repository/repository_name.go:89 Get
   → DynamoDB GetItem

   (1 SDK operation, 4 hops) Active callers: 2箇所
```

### Batch Approval Summary (Phase 2)
```
=== バッチ処理する組み合わせ ===

合計SDK関数数: 7個
- Create操作: 4個
- Update操作: 0個
- Read操作: 2個
- Delete操作: 0個
- 複数SDK使用: 2個

処理対象のチェーン:

1. Chain: Task task_name_1 [★ Multiple SDK: 3 operations]
   Entry → Intermediate layers:
   Entry: Task task_name_1
   → cmd/task_name_1/main.go:100 main
   → internal/tasks/task_worker.go:50 Execute
   → internal/service/service_name.go:80 ProcessData

   SDK Functions:
   1-A. internal/service/service_name.go:120 createRecord | DynamoDB PutItem
   1-B. internal/service/service_name.go:150 storeFile | S3 PutObject
   1-C. internal/service/service_name.go:180 sendMessage | SES SendEmail

2. Chain: POST /v1/resource/action [★ Multiple SDK: 2 operations]
   Entry → Intermediate layers:
   Entry: API POST /v1/resource/action
   → internal/api/handler/v1/handler_name.go:80 HandleAction
   → internal/gateway/gateway_name.go:89 ProcessAction

   SDK Functions:
   2-A. internal/gateway/gateway_name.go:120 fetchData | S3 GetObject
   2-B. internal/gateway/gateway_name.go:200 saveBatch | DynamoDB BatchWriteItem

3. Chain: Task task_name_2
   Entry: Task task_name_2
   → cmd/task_name_2/main.go:50 main
   → internal/usecase/usecase_name.go:30 Execute
   → internal/repository/repository_name.go:45 Save
   → DynamoDB PutItem

4. Chain: API GET /v1/resource/:id
   Entry: API GET /v1/resource/:id
   → internal/api/handler/v1/handler_name.go:100 GetResource
   → internal/service/service_name.go:50 Fetch
   → internal/repository/repository_name.go:89 Get
   → DynamoDB GetItem
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
- AWS操作: [Create/Update/Read/Delete] ([operation_name])
- Pre-insertコード: 追加済み / 不要
- コンパイル: 成功
```

### Final Summary (Phase 5)
```
=== 処理完了サマリー ===

処理したSDK関数: 5個
- Create操作: 3個
- Update操作: 0個
- Read操作: 1個 (Pre-insertコード追加済み)
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
- Bash: Stop immediately if branch diff does not contain `aws-sdk-go-v2` imports
- Task (subagent_type=general-purpose): Code analysis (steps 2, 3, 6, 8, 9)
- Edit: Automatically apply code modifications (steps 6, 10)
- Bash (`go build`): Compilation verification (steps 6, 10)

### Key Process Steps
- **Deduplication** (step 3): Group by AWS_service + SDK_operation, ignore parameters. Select shortest chain from each group.
- **Comment-out unrelated code** (step 6): Identify code blocks unrelated to target AWS SDK operation, comment out with reason, verify compilation.
- **Simplified test data generation** (step 9): Skip detailed analysis, generate minimal test data (1-2 records) with correct Go types for Update/Read/Delete operations.
- **Apply Pre-insert code** (step 10): Insert test data preparation code, verify compilation, automatically proceed to next chain.

### Output Guidelines
- Include file:line references for navigation
- Provide complete call chains for traceability
- Mark chains with multiple SDK methods with [★ Multiple SDK] indicator
- Display progress (i/N) during batch processing
- Show final summary with compilation status

For detailed requirements, see Analysis Requirements section above.
