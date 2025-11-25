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
   Task prompt: "Extract all functions/methods using AWS SDK v2 from git diff and trace their complete call chains.

   **Step 1: Extract SDK functions using Grep**

   Execute Grep searches in this exact order (run sequentially to build on previous results):
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

   **Step 3: Verify active callers**

   **Parallel execution**: Execute in parallel with Step 2 (independent operations)

   For each extracted function from Step 1:
   1. Use Grep: `pattern: "<function_name>\("`, `glob: "!(*_test.go|mocks/*.go)"`, `output_mode: "files_with_matches"`
   2. Count result files (active call sites in production code)
   3. If count = 0:
      - Mark as "SKIP - No active callers"
      - Exclude from call chain list

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

**Keep (required for connection testing)**:
- SDK client initialization
- SDK input construction
- SDK call
- Minimal error check (if err != nil)

**Comment out (unrelated to connection)**:
- Response processing
- Business logic
- External service calls
- Detailed error handling

6. **Comment out unrelated code in call chain functions**

   For each chain from Phase 1 deduplication result (index i from 1 to N):

   A. Display progress:

   For single SDK operation:
   ```
   === コメントアウト処理中 (i/N) ===
   Chain: [entry_type] [identifier]
   SDK operation: [Service] [Operation]
   ```

   For multiple SDK operations:
   ```
   === コメントアウト処理中 (i/N) ===
   Chain: [entry_type] [identifier] [★ Multiple SDK: M operations]
   SDK operations: [N]-A [Operation1], [N]-B [Operation2], [N]-C [Operation3]
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

   **Analysis approach differs by SDK operation count:**

   **For SINGLE SDK operation chains:**
   Task prompt: "For call chain [chain_id] from Phase 1:

   **Tools to use**: Read tool for loading source code, no Grep/Glob needed

   **Context from Phase 1** (copy complete chain):
   ```
   Entry: [type] [identifier]
   → [file:line] EntryFunction
   → [file:line] IntermediateFunction
   → [file:line] SDKFunction
   → [Service] [Operation]
   ```

   **Functions to analyze** (MUST analyze ALL):
   Function 1: [file:line] EntryFunction (entry)
   Function 2: [file:line] IntermediateFunction (intermediate)
   Function 3: [file:line] SDKFunction (sdk)

   For EACH function above:
   Step 1: Load function source code
   Step 2: Identify SDK-related code (KEEP)
   Step 3: Identify unrelated code (COMMENT)

   **For MULTIPLE SDK operation chains:**
   Task prompt: "For call chain [chain_id] from Phase 1 with [N] SDK operations:

   **Tools to use**: Read tool only (load source code directly)

   **Context from Phase 1** (copy hierarchical structure):
   ```
   Entry → Intermediate layers:
   Entry: [type] [identifier]
   → [file:line] EntryFunction
   → [file:line] IntermediateFunction

   SDK Functions:
   [N]-A. [file:line] SDKFunction1
          Operation: [Service] [Operation1]

   [N]-B. [file:line] SDKFunction2
          Operation: [Service] [Operation2]
   ```

   **Analysis procedure (optimize for shared layers):**

   Step 1: Analyze Entry/Intermediate layers ONCE
   - Load function source code with Read tool
   - Identify code unrelated to ANY SDK operation
   - Common patterns to comment out:
     - External HTTP/gRPC calls
     - Validation not related to SDK input
     - Data enrichment from non-AWS sources

   Step 2: Analyze EACH SDK function individually
   - Load SDK function source code with Read tool
   - Identify code unrelated to THIS specific operation
   - Common patterns to comment out:
     - Response parsing (parseAttributes, loops over resp.Items)
     - Entity transformation (ToEntity)
     - Pagination logic
     - Detailed error wrapping

   **Objective**: Classify code into SDK-related (KEEP) and unrelated (COMMENT).

   **Classification criteria**:

   SDK-related code (KEEP):
   - SDK client initialization: `dynamodb.New()`, `s3.NewFromConfig()`
   - SDK input construction: `&dynamodb.PutItemInput{...}`
   - Data transformation for SDK input: variables used in input fields
   - Context handling: `ctx` parameter
   - Minimal error check: `if err != nil { log; return }`

   Unrelated code (COMMENT):
   - Logging/metrics/monitoring
   - External service calls (HTTP, gRPC)
   - Validation not related to SDK input
   - Business logic after SDK operation
   - Cache operations
   - Response parsing: parseAttributes, loops over resp.Items
   - Entity transformation: ToEntity, domain model conversion
   - Pagination logic: ExclusiveStartKey handling
   - Detailed error wrapping: apperrors.Wrap, utils.GetFunctionName

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

   Return format for SINGLE SDK operation:
   ```
   Chain [N]: [entry_type] [identifier]
   SDK operation: [Service] [Operation]

   Function 1: [entry_function] at [file:line] (entry)
   SDK-related code (KEEP):
   - Lines X-Y: [description]

   Unrelated code (COMMENT):
   - Lines A-B: [description]

   Function 2: [intermediate_function] at [file:line] (intermediate)
   Unrelated code (COMMENT):
   - Lines P-Q: [description]

   Function 3: [sdk_function] at [file:line] (sdk)
   SDK-related code (KEEP):
   - Lines M-N: [description]

   Unrelated code (COMMENT):
   - Lines S-T: [description]
   ```

   Return format for MULTIPLE SDK operations:
   ```
   Chain [N]: [entry_type] [identifier] [★ Multiple SDK: M operations]

   Entry/Intermediate layers (analyzed once):
   Function 1: [entry_function] at [file:line] (entry)
   Unrelated code (COMMENT):
   - Lines A-B: [description]

   Function 2: [intermediate_function] at [file:line] (intermediate)
   Unrelated code (COMMENT):
   - Lines C-D: External service call
   - Lines E-F: Validation logic

   SDK Functions (analyzed individually):
   [N]-A. [sdk_function_1] at [file:line]
   Operation: [Service] [Operation1]
   SDK-related code (KEEP):
   - Lines X-Y: [description]
   Unrelated code (COMMENT):
   - Lines P-Q: Response parsing

   [N]-B. [sdk_function_2] at [file:line]
   Operation: [Service] [Operation2]
   SDK-related code (KEEP):
   - Lines M-N: [description]
   Unrelated code (COMMENT):
   - Lines S-T: Entity transformation
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

   For single SDK operation:
   ```
   完了 (i/N): コメントアウト処理
   Chain: [entry_type] [identifier]
   - Entry/Intermediate関数: X個
   - SDK関数: 1個
   - コメントアウトしたブロック数: Y個
   - コンパイル: 成功
   ```

   For multiple SDK operations:
   ```
   完了 (i/N): コメントアウト処理
   Chain: [entry_type] [identifier] [★ Multiple SDK: M operations]
   - Entry/Intermediate関数: X個 (1回のみ分析)
   - SDK関数: M個 (個別に分析)
   - コメントアウトしたブロック数: Y個
   - コンパイル: 成功
   ```

### Phase 4: Simplified Test Data Preparation

7. **Process all chains sequentially**

   **Processing strategy:**
   - Single SDK operation: Process once per chain
   - Multiple SDK operations: Process EACH SDK function separately (chain with 3 operations = 3 processing runs)

   For each chain from Phase 1 deduplication result (index i from 1 to N):

   A. Display progress:

   For single SDK operation:
   ```
   === テストデータ準備中 (i/N) ===
   Chain: [entry_type] [identifier]
   SDK function: [file:line] [function_name]
   Operation: [Service] [Operation]
   ```

   For multiple SDK operations (each SDK function processed separately):
   ```
   === テストデータ準備中 (i/N, SDK function [N]-A/B/C) ===
   Chain: [entry_type] [identifier] [★ Multiple SDK]
   SDK function: [N]-A. [file:line] [function_name]
   Operation: [Service] [Operation]
   ```

   B. Execute steps 8-10 for current chain

8. **Analyze AWS SDK operation with Task tool** (subagent_type=general-purpose)
   Task prompt: "For function [function_name] at [file_path:line_number]:

   **Tools**: Read for source code, Grep for pattern searches

   **Context from Phase 1**:
   - SDK operation: [operation_name] (e.g., DynamoDB PutItem)
   - Operation type (classify by name):
     - Create: PutItem, PutObject, SendEmail, Publish
     - Update: UpdateItem, TransactWriteItems
     - Read: Query, GetItem, Scan, GetObject
     - Delete: DeleteItem, DeleteObject

   **Extract AWS settings**:
   1. Region: Grep `pattern: "WithRegion|AWS_REGION"`, `output_mode: "content"`, `-C: 5`
   2. Resource: Read SDK call parameters (table name, bucket name)
   3. Endpoint: Grep `pattern: "WithEndpointResolver|endpoint"`, `output_mode: "content"`, `-C: 5`

   **Document v1 → v2 migration changes**:
   - Client init: session.New → config.LoadDefaultConfig
   - API call: method signature changes
   - Type changes: aws.String usage patterns

   Return: AWS settings, migration summary."

9. **Generate minimal test data with Task tool** (subagent_type=general-purpose)

   Task prompt: "For function [function_name] with AWS SDK operation [operation_name]:

   **Tools**: Read to extract SDK call parameters

   **Context from Phase 1**:
   - Operations in chain: [list]
   - Operation types (classify by name):
     - Create: PutItem, PutObject, SendEmail, Publish
     - Update: UpdateItem, TransactWriteItems
     - Read: Query, GetItem, Scan, GetObject
     - Delete: DeleteItem, DeleteObject

   **Pre-insert requirements**:
   - Create → No Pre-insert (creates new data)
   - Update → Generate Pre-insert (requires existing data)
   - Read → Generate Pre-insert (requires existing data)
   - Delete → Generate Pre-insert (requires existing data)

   **If operation is Update/Read/Delete**:
   1. Read SDK call parameters (table name, key, bucket)
   2. Generate 1-2 minimal test records
   3. Use Go pointer types: `aws.String()`, `aws.Int64()`
   4. Generate Pre-insert code (PutItem/PutObject)

   Return:
   - Pre-insert code snippet with indentation
   - Insertion line number (before SDK call)
   - Required imports

   **If operation is Create**:
   Return: \"No Pre-insert needed\""

10. **Apply Pre-insert code with Edit tool**

   If operation type is Update/Read/Delete:

   A. Insert Pre-insert code from step 9:
      - Use Edit: insert before AWS SDK operation
      - old_string: line before SDK call
      - new_string: line + "\n" + Pre-insert code
      - Output: "Pre-insertコード追加: [file:line]"

   B. Verify compilation:
      - Run: `go build -o /tmp/test-build 2>&1`
      - If fails: Fix errors (imports, types), retry Edit
      - Repeat until success
      - Output: "コンパイル成功: [file_path]"

   C. Display completion:

      For single SDK operation:
      ```
      完了 (i/N): テストデータ準備
      Chain: [entry_type] [identifier]
      SDK function: [file:line] [function_name]
      - Pre-insertコード: 追加済み / 不要
      - コンパイル: 成功
      ```

      For multiple SDK operations (per SDK function):
      ```
      完了 (i/N, SDK function [N]-A): テストデータ準備
      Chain: [entry_type] [identifier] [★ Multiple SDK]
      SDK function: [N]-A. [file:line] [function_name]
      - Pre-insertコード: 追加済み / 不要
      - コンパイル: 成功
      ```

   D. **Verify Pre-insert code completeness**

   After processing all chains, verify Pre-insert code for Update/Read/Delete operations:

   1. Grep for Update/Read/Delete operations:
      - `pattern: "client\.(UpdateItem|TransactWriteItems|Query|GetItem|GetObject|Scan|DeleteItem|DeleteObject)"`
      - `output_mode: "content"`, `-B: 20`
      - `path: [processed files]`

   2. For each operation:
      - Check preceding 20 lines for Pre-insert patterns: `// Pre-insert test data`, `PutItem.*Input`, `PutObject.*Input`

   3. Results:
      - All have Pre-insert → "検証完了: Pre-insertコード生成済み (N operations)"
      - Missing Pre-insert → ERROR: "Phase 4 incomplete - Pre-insert missing", HALT

11. **Automatic progression**
   - If i < N: continue to next chain (repeat from step 7.A)
   - If i = N: proceed to Phase 3.5

### Phase 3.5: Verify Comment-out Completeness

After Phase 3, verify all unrelated code is commented out:

1. Check entry/intermediate functions analyzed:
   - Grep: `pattern: "Commented out for testing"`, `glob: "cmd/**/*.go"`, `output_mode: "files_with_matches"`
   - Grep: `pattern: "Commented out for testing"`, `glob: "internal/tasks/**/*.go"`, `output_mode: "files_with_matches"`
   - If no matches: WARNING

2. Verify external service calls commented:
   - Grep: `pattern: "http\\.(Get|Post|Client)|grpc\\.(Dial|NewClient)"`, `output_mode: "content"`, `-C: 5`, `glob: "!(*_test.go)"`
   - Check each match preceded by "Commented out for testing"
   - If uncommented: ERROR - re-run Phase 3

3. Verify response processing minimized:
   - Grep: `pattern: "parseAttributes|ToEntity|for.*resp\\.(Items|Records)"`, `output_mode: "content"`, `glob: "!(*_test.go)"`
   - Check each match commented or replaced with log.Printf
   - If complex processing remains: ERROR - re-run Phase 3

### Phase 5: Final Summary

12. **Output final summary report**

    ```
    === 処理完了サマリー ===

    処理したSDK関数: N個
    - Create: A個
    - Update: B個 (Pre-insert追加済み)
    - Read: X個 (Pre-insert追加済み)
    - Delete: Z個 (Pre-insert追加済み)
    - 複数SDK使用: W個

    コメントアウト (Phase 3):
    - [file_path_1] (X blocks)
    - [file_path_2] (Y blocks)

    書き換え (Phase 4):
    - [file_path_1] (Pre-insert追加)
    - [file_path_2] (Pre-insert追加)

    コンパイル: 成功P個 / 失敗Q個
    ```

13. **Generate AWS verification procedures**

    Output AWS-specific verification procedures grouped by execution method:

    ```markdown
    ## AWS環境での動作確認方法

    ### 1. [Execution Method] (例: POST /v1/entities)
    **検証対象関数**:
    - [file:line] FunctionName1 | [Operation1]
    - [file:line] FunctionName2 | [Operation2]

    **実行コマンド**:
    ```bash
    curl -X POST https://api.example.com/v1/entities \
      -H "Content-Type: application/json" \
      -d '{"param":"value"}'
    ```

    **X-Ray確認ポイント**:
    - [Service] [Operation1] × N回
    - [Service] [Operation2] × M回
    - Data flow: Operation1 → Operation2
    ```

    ### Grouping Policy

    Group by execution method (not by function):

    Good:
    ```
    ### POST /v1/entities
    Verifies: FunctionA (PutItem), FunctionB (Query)
    curl -X POST ...
    ```

    Bad (duplicates commands):
    ```
    ### FunctionA
    curl -X POST ...
    ### FunctionB
    curl -X POST ...
    ```

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

### Focus
- Production AWS connections (exclude localhost/test)
- Resource names (table names, bucket names)
- Region configuration (explicit or AWS_REGION)
- v1 → v2 migration patterns

### Output Scope

**Include**:
- AWS API endpoints
- ECS task run commands (aws-cli)
- X-Ray trace points
- Expected SDK call sequences

**Exclude**:
- Local development setup
- Environment configuration
- Authentication procedures
- Dependency installation

### Key Process Steps
- Phase 3, step 6: Comment out code unrelated to SDK operation
- Phase 4, step 9: Generate minimal test data
- Phase 4, step 11: Run `go build`, auto-fix errors

### Batch Processing Flow
1. Phase 1: Extract and deduplicate (automatic)
2. Phase 2: Single batch approval (AskUserQuestion)
3. Phase 3: Comment out unrelated code (automatic)
4. Phase 4: Generate test data, verify compilation (automatic)
5. Phase 5: Display final summary

## Notes

### Tool Usage
- Bash: Verify branch contains `aws-sdk-go-v2` imports, exit if not
- Task (subagent_type=general-purpose): Code analysis
- Edit: Apply code modifications
- Bash (`go build`): Compilation verification

### Process Details
- **Deduplication** (step 3): Group by AWS_service + SDK_operation, select shortest chain
- **Comment-out** (step 6): Comment unrelated code blocks, verify compilation
- **Test data** (step 9): Generate minimal test data (1-2 records) for Update/Read/Delete
- **Pre-insert** (step 10): Insert test data, verify compilation

### Output Guidelines
- Include file:line references
- Provide complete call chains
- Mark multiple SDK chains: [★ Multiple SDK]
- Display progress: (i/N)
- Show compilation status
