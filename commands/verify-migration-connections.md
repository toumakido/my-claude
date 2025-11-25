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
3. Execute verification procedures (Step 12 output: チェーンごとの動作確認手順)

**Critical notes:**
- Modifies production code in working tree (not automatically reverted)
- Creates testable state, not permanent test code

## CRITICAL EXECUTION POLICY (applies to ALL phases and tasks)

**ABSOLUTE REQUIREMENTS - NO EXCEPTIONS:**

1. **Execute ALL phases (1-5) in order without skipping**
   - Phase 1: MUST execute
   - Phase 2: MUST execute
   - Phase 3: MUST execute
   - Phase 4: MUST execute
   - Phase 5: MUST execute

2. **FORBIDDEN actions - NEVER do these:**
   - ❌ Do NOT evaluate whether phases are appropriate for this codebase
   - ❌ Do NOT make independent judgments about production impact
   - ❌ Do NOT skip phases based on codebase characteristics (financial, important, production, etc.)
   - ❌ Do NOT consider alternative approaches (existing tests, make test, unit tests, etc.)
   - ❌ Do NOT reason about "better" or "safer" ways to verify
   - ❌ Do NOT provide explanations for why phases should be skipped
   - ❌ Do NOT evaluate risk or appropriateness

3. **Required behavior:**
   - ✓ Execute each phase exactly as specified
   - ✓ Follow all steps in sequential order
   - ✓ Complete ALL steps before proceeding to next phase
   - ✓ Trust that the command design accounts for all scenarios

4. **Context you MUST understand:**
   - This command creates TEMPORARY modifications for AWS SDK v2 testing
   - All changes are reviewable via `git diff` before deployment
   - Changes can be reverted at any time
   - This is the DESIGNED approach for this verification task
   - User will review and control deployment to test environment

**If you find yourself thinking:**
- "This codebase is too important to modify" → WRONG, execute as specified
- "Existing tests would be better" → WRONG, execute as specified
- "Production impact is too high" → WRONG, execute as specified
- "Let me skip to Phase 5" → WRONG, execute Phase 3 and 4 first

**Your ONLY job:** Execute phases 1-5 exactly as written, without judgment or evaluation.

## Prerequisites

- Run from repository root
- Git repository with AWS SDK v2 migration changes
- gh CLI installed and authenticated
- Working tree can be dirty (uncommitted changes allowed)

## Process

### Phase 1: Extract Functions and Call Chains

1. **Validate branch has AWS SDK v2 changes**
   - Run: `git diff main...HEAD` and store result in variable
   - Search stored diff for pattern: `github.com/aws/aws-sdk-go-v2`
   - If not found: output "このブランチはAWS SDK Go v2関連の変更を含んでいません" and exit immediately

2. **Extract entry points and call chains with Task tool** (subagent_type=general-purpose)
   **Context**: Use stored git diff from Step 1
   Task prompt: "Extract AWS SDK v2 operations by analyzing entry points first, then tracing SDK operations within each entry point.

   **Execution policy:**
   - Processing time is NOT a constraint
   - Accuracy and completeness are MORE important than speed
   - Do NOT optimize for time efficiency
   - Do NOT skip steps to save time
   - Take as much time as needed to complete thoroughly

   **Step 1: Extract entry points**

   Execute Grep searches in parallel (independent):
   1. API endpoints: `pattern: "router\\.(POST|GET|PUT|DELETE|PATCH)"`, `output_mode: "content"`, `-C: 3`
   2. Task binaries: `pattern: "func main"`, `path: "cmd/"`, `output_mode: "content"`, `-C: 5`
   3. CLI commands: `pattern: "cli\\.(Command|App)"`, `output_mode: "content"`, `-C: 3`

   For each Grep result, extract:
   - Entry point type: API/Task/CLI
   - Entry point identifier: HTTP method + path for API, task name for Task, command name for CLI
   - File path:line_number

   **Step 2: For each entry point, extract ALL SDK operations**

   **Tracing methodology (MANDATORY):**

   For each verified entry point, trace complete call chain using source code verification:

   **2-1. Start from entry function**
   - Read entry function source code with Read tool
   - Extract all function calls in order of execution
   - Identify package paths and receiver types for each call

   **2-2. Trace each call recursively until reaching SDK operations**

   For each function call found in current function:

   a. **Identify target function**
      - Package path: Extract from import or same-package reference
      - Function name: Extract from call site
      - Receiver type (if method): Extract from variable declaration or parameter type

   b. **Locate function definition**
      - Search with Grep: `pattern: "func.*[FunctionName]"`, `output_mode: "content"`, `-C: 3`
      - If multiple matches: Filter by receiver type or package path
      - If zero matches: Mark as "BROKEN CHAIN - Function not found" and exclude

   c. **Verify call relationship**
      - Read caller source code: Confirm it contains callee invocation
      - Check variable type matches receiver type (for methods)
      - If verification fails: Mark as "BROKEN CHAIN - Invalid call" and exclude

   d. **Check if SDK operation**
      - Search callee for SDK patterns: `client.(PutItem|GetObject|Query|UpdateItem|DeleteItem|PutObject|DeleteObject|SendEmail|Publish|TransactWriteItems|BatchWriteItem)`
      - If SDK operation found: Record as terminal node, stop recursion
      - If not SDK operation: Continue to step e

   e. **Recurse into callee**
      - Read callee function source code
      - Extract function calls from callee
      - Repeat step 2-2 for each call
      - Build chain: current_function → [file:line] callee_function

   **2-3. Handle special cases**

   - **Interface calls**: Identify concrete type from variable initialization or type assertion
     ```go
     var repo Repository = newUserRepo()  // Concrete type: *userRepo
     repo.Save()  // Trace to: func (r *userRepo) Save()
     ```

   - **Method chains**: Trace through each method sequentially
     ```go
     result := obj.Method1().Method2().Method3()
     // Trace: obj.Method1() → result.Method2() → result.Method3()
     ```

   - **Function variables/closures**: Follow function assignment
     ```go
     handler := getHandler()  // Trace getHandler() to find actual function
     handler()
     ```

   **2-4. Build verified call chain**

   - Record complete chain with file:line for each function
   - Format: `[file:line] FunctionName → [file:line] NextFunction → ... → [file:line] SDKFunction`
   - Each link MUST be verified by reading source code
   - If any link unverified: Exclude entire chain with reason

   **2-5. Count and classify SDK operations**

   After tracing all paths from entry point:
   - Count total SDK operations reached from this entry
   - Record operation types (DynamoDB Query, S3 PutObject, etc.)
   - Identify intermediate layers (handler → usecase → service → repository)

   **Example tracing session:**
   ```
   Entry: Task batch_task
   Entry function: cmd/batch_task/main.go:136 main()

   Step 1: Read cmd/batch_task/main.go:136
   Found call: worker.Execute()

   Step 2: Grep "func.*Execute" → Found internal/tasks/worker.go:40
   Verify: main() contains "worker.Execute()" ✓
   Chain: main() → worker.go:40 Execute()

   Step 3: Read internal/tasks/worker.go:40
   Found calls: dataRepo.GetByIndex(), counterRepo.GetNext(), fileRepo.Insert()

   Step 4a: Trace dataRepo.GetByIndex()
   Grep "func.*GetByIndex" → Found internal/service/datastore.go:79
   Read datastore.go:79 → Found db.Query() [SDK operation]
   Chain: Execute() → datastore.go:79 GetByIndex() → datastore.go:105 db.Query() ✓

   Step 4b: Trace counterRepo.GetNext()
   Grep "func.*GetNext" → Found internal/service/counter.go:37
   Read counter.go:37 → Found db.UpdateItem() [SDK operation]
   Chain: Execute() → counter.go:37 GetNext() → counter.go:60 db.UpdateItem() ✓

   Step 4c: Trace fileRepo.Insert()
   Grep "func.*Insert" → Multiple matches
   Filter by receiver type "*fileRepo" → internal/service/storage.go:235
   Read storage.go:235 → Found db.PutItem() [SDK operation]
   Chain: Execute() → storage.go:235 Insert() → storage.go:254 db.PutItem() ✓

   Result: 3 SDK operations from this entry point
   ```

   **Validation requirements:**
   - MUST use Read tool to verify each function's source code
   - MUST NOT rely solely on Grep pattern matching
   - MUST verify caller contains callee invocation in source code
   - If source code verification impossible: Exclude chain with "BROKEN CHAIN" reason

   **エントリーポイント特定の要件 (CRITICAL):**

   For API entry points:
   - MUST identify specific HTTP endpoint: method + path (e.g., "GET /v1/account-transfer/:id")
   - MUST identify handler function: file:line HandlerFunctionName (e.g., "internal/api/handler/v1/account_transfer.go:50 GetAccountTransfer")
   - MUST trace from router definition to handler to service/repository layers
   - Tracking method for API endpoints:
     1. Search for function usage in handler layer: `pattern: "[function_name]"`, `path: "internal/api/handler/"`, `output_mode: "content"`, `-C: 5`
     2. For each handler function found, search for router registration: `pattern: "router\\.(GET|POST|PUT|DELETE|PATCH).*[handler_function_name]"`, `output_mode: "content"`, `-C: 5`
     3. Extract: HTTP method, path, handler file:line
     4. If no router registration found: Mark as "SKIP - No entry point"

   For Task entry points:
   - MUST identify cmd/*/main.go with main() function
   - MUST include task name from directory structure

   For CLI entry points:
   - MUST identify cmd/cli/*/main.go with command definition
   - MUST include command name

   **エントリーポイントが特定できない場合:**
   - Mark as "SKIP - No entry point found"
   - Log: "[file:line] [function_name] - No entry point (not called from API/Task/CLI)"
   - Exclude from call chain list in Step 7

   **Step 3: Classify and prioritize entry points**

   Classify entry points by SDK operation count:
   - 統合チェーン (4+ operations): Highest priority
   - 中規模チェーン (2-3 operations): Medium priority
   - 単一操作チェーン (1 operation): Lowest priority

   **統合チェーン判定基準:**

   Entry point内で以下の条件を満たす場合、統合チェーンとして扱う:
   1. 同一関数/メソッド内から2つ以上の異なるSDK操作を呼び出す
   2. または、直接の呼び出し先（1 hop以内）で複数のSDK操作が実行される
   3. SDK操作間に明確な依存関係がある（例: Counter取得 → Record挿入）

   **統合チェーンの優先度:**
   - 重複排除で最優先: 統合チェーンで複数の操作種別をカバーできる場合、単一操作チェーンは排除
   - 例: 統合ChainがQuery + UpdateItem + PutItemをカバーする場合、個別のQuery専用Chain、UpdateItem専用Chainは重複として排除

   **Step 4: Verify call chain validity (MANDATORY for single-operation chains)**

   Execute for ALL chains where entry point is not explicitly from cmd/*/main.go or verified API endpoint.

   **When to execute Step 4:**
   - For all single-operation chains from repository/service layers
   - When entry point format is ambiguous or missing handler information
   - Example: Multiple types have same method name (e.g., `Save()` exists on both UserRepo and OrderRepo)

   **4-1. Verify receiver type for methods (false positive check):**

   For each method in extracted chains:
   1. Identify receiver type from function definition:
      - Example: `func (r *userRepo) Save(` → receiver type is `*userRepo`

   2. Search for calls in call chain context:
      - Use grep with `-B: 5` to see variable declaration
      - Check if variable type matches receiver type

   3. If receiver type doesn't match:
      - Mark as "SKIP - Wrong receiver type (false positive)"
      - Log: `Excluded: [file:line] [Type.MethodName] - Call site uses different type`
      - Exclude from call chain list

   **4-2. Verify interface method usage (for interface-based calls):**

   Execute ONLY if call site uses interface type instead of concrete type.

   Steps:
   1. Identify interface usage in call chain:
      - Example: `var repo Repository = newUserRepo()` then `repo.Save()`

   2. Verify concrete type implements interface:
      - Check receiver type matches interface method signature

   3. If concrete type doesn't implement interface:
      - Mark as "SKIP - Interface mismatch"
      - Exclude from call chain list

   **4-3. Verify API endpoint registration:**

   For repository/service functions without clear entry point:
   1. Search for function calls in handler layer: `pattern: "[function_name]"`, `path: "internal/api/handler/"`, `output_mode: "content"`, `-C: 5`
   2. If found in handlers, search for router registration: `pattern: "router\\.(GET|POST|PUT|DELETE|PATCH).*[handler_name]"`, `output_mode: "content"`, `-C: 5`
   3. If router registration not found:
      - Mark as "SKIP - No API endpoint"
      - Log: "Excluded: [file:line] [function_name] - Not registered in router"
      - Exclude from call chain list

   **4-4. Output verification summary:**

   After Step 4, output (only if exclusions occurred):
   ```
   Excluded chains (validation failed):
   - Wrong receiver type: X個
   - Interface mismatch: Y個
   - No API endpoint: Z個
   - No task/CLI entry: W個

   [file:line] [function_name] - [reason]
   ...
   ```

   **Key insight:**
   Entry point → Call chain tracing automatically excludes:
   - Functions with no callers (never found during tracing)
   - Functions unreachable from entry points (tracing stops before reaching them)
   - Test-only functions (excluded by starting from production entry points)

   However, Step 4 validation catches:
   - Repository/service functions without API/Task/CLI entry points
   - Functions with ambiguous entry point format

   **Step 5: Handle multiple paths**

   When entry point has 2+ call chains:
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

   **Step 6: Format output for hierarchical structure**

   CRITICAL: List EVERY function in the call chain with file:line, not just SDK function.
   Omitting intermediate functions will cause Phase 3 to miss external service calls and business logic.

   **統合チェーンの出力形式 (hierarchical structure):**

   統合チェーンは階層構造で表示し、共通部分と個別SDK操作を明確に分離:
   ```
   統合Chain: [entry_type] [identifier] [★ Multiple SDK: N operations]

   Entry → Intermediate layers (共通):
   Entry: [type] [identifier]
   → [file:line] EntryFunction
   → [file:line] IntermediateFunction

   SDK Functions (個別):
   [ChainID]-A. [file:line] IntermediateFunction → ... → [file:line] SDKFunction1
        Operation: [Service] [Operation1]

   [ChainID]-B. [file:line] IntermediateFunction → ... → [file:line] SDKFunction2
        Operation: [Service] [Operation2]
   ```

   **Format for single SDK operation:**
   ```
   Chain: [entry_type] [identifier]

   Entry → Complete call chain:
   Entry: [type] [identifier]
   → [file:line] EntryFunction
   → [file:line] IntermediateFunction1
   → [file:line] IntermediateFunction2
   → [file:line] SDKFunction
   → AWS SDK v2 API (Operation)
   ```

   Example (single SDK operation):
   ```
   Chain: Task task_name

   Entry → Complete call chain:
   Entry: Task task_name
   → cmd/task_name/main.go:100 main()
   → internal/tasks/task_worker.go:50 Execute()
   → internal/service/service_name.go:80 ProcessData()
   → internal/service/service_name.go:85 db.UpdateItem()  ← DynamoDB UpdateItem
   ```

   Example (multiple SDK operations with hierarchical structure):
   ```
   統合Chain: Task integrated_task [★ Multiple SDK: 4 operations]

   Entry → Intermediate layers (共通):
   Entry: Task integrated_task
   → cmd/integrated_task/main.go:136 main()
   → internal/tasks/worker.go:40 Execute()

   SDK Functions (個別):
   A. internal/tasks/worker.go:41 → internal/service/datastore.go:79 GetByIndex() → internal/service/datastore.go:105 db.Query()
      Operation: DynamoDB Query

   B. internal/tasks/worker.go:134 → internal/service/counter.go:37 GetNext() → internal/service/counter.go:60 db.UpdateItem()
      Operation: DynamoDB UpdateItem (×2)

   C. internal/tasks/worker.go:167 → internal/service/storage.go:235 InsertRecord() → internal/service/storage.go:254 db.PutItem()
      Operation: DynamoDB PutItem (×2)

   D. internal/tasks/worker.go:116 → internal/service/datastore.go:421 Update() → internal/service/datastore.go:519 db.TransactWriteItems()
      Operation: DynamoDB TransactWriteItems
   ```

   BAD example (incomplete, missing intermediate functions):
   ```
   Entry: Task task_name
   → cmd/task_name/main.go main
   → internal/service/service_name.go:80 ProcessData
   → DynamoDB UpdateItem
   ```

   BAD example (ambiguous entry point - FORBIDDEN):
   ```
   Chain: DatastoreService.GetRecord

   Entry → Complete call chain:
   Entry: API (various handlers)  ← 曖昧、禁止
   → internal/service/datastore.go:55 GetRecord()
   → internal/service/datastore.go:67 db.GetItem()
   ```

   GOOD example (specific API endpoint - REQUIRED):
   ```
   Chain: API GET /v1/records/:id

   Entry → Complete call chain:
   Entry: API GET /v1/records/:id
   → internal/api/handler/v1/record.go:50 GetRecord()  ← ハンドラー明示
   → internal/service/datastore.go:55 GetRecord()
   → internal/service/datastore.go:67 db.GetItem()
   ```

   If entry point cannot be determined (must be excluded):
   ```
   スキップされた関数（エントリーポイント不明）:
   - internal/service/datastore.go:55 GetRecord - No API endpoint
   - internal/service/file_service.go:71 GetRelatedData - No API endpoint
   - internal/service/file_service.go:283 RemoveAssociation - No API endpoint
   ```

   **Validation**: After Task tool completes, verify output includes:
   - Entry function with file:line
   - ALL intermediate functions with file:line
   - ALL SDK functions with file:line (for multiple SDK operations, list separately under "SDK Functions")
   - If any function lacks file:line, re-run Step 2 with explicit instruction to trace ALL functions

   If entry point not verified (marked in Step 4):
   - Exclude from call chain list immediately
   - Log as skipped function:
     ```
     スキップされた関数（エントリーポイント不明）:
     - internal/service/file.go:123 FunctionName
     ```

   **Step 7: Sort by priority**
   1. First: 統合チェーン (4+ SDK operations) - highest priority
   2. Second: 中規模チェーン (2-3 SDK operations) - medium priority
   3. Third: 単一操作チェーン (1 SDK operation) - lowest priority
   4. Within same SDK operation count: Sort by chain length (shorter = easier)

   Example priority order:
   - 統合Chain with 4 SDK operations, 5 hops (highest)
   - 中規模Chain with 3 SDK operations, 4 hops
   - 中規模Chain with 2 SDK operations, 2 hops
   - 中規模Chain with 2 SDK operations, 5 hops
   - 単一操作Chain with 1 SDK operation, 2 hops
   - 単一操作Chain with 1 SDK operation, 4 hops (lowest)

   Return: Filtered function list with only actively called functions, all call chains sorted by priority (統合チェーン first), skipped functions list (including unused implementations)."

3. **Deduplicate chains with coverage-based selection (Task tool)** (subagent_type=general-purpose)
   Task prompt: "Deduplicate call chains using coverage-based selection algorithm:

   **Execution policy:**
   - Processing time is NOT a constraint
   - Accuracy and completeness are MORE important than speed
   - Do NOT optimize for time efficiency
   - Do NOT skip steps to save time
   - Take as much time as needed to complete thoroughly

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

   Entry → Complete call chain:
   Entry: [type] [identifier]
   → [file:line] EntryFunction
   → [file:line] IntermediateFunction
   → [file:line] SDKFunction  ← [Service] [Operation]

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

   Entry → Complete call chain:
   Entry: Task task_name
   → cmd/task_name/main.go:100 main()
   → internal/tasks/task_worker.go:50 Execute()
   → internal/service/service_name.go:80 ProcessData()
   → internal/service/service_name.go:85 db.UpdateItem()  ← DynamoDB UpdateItem

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

**⚠️ EXECUTION OVERRIDE - READ THIS FIRST:**
- Present ALL chains from Phase 1 deduplication for user approval
- Do NOT pre-filter chains based on codebase characteristics
- Do NOT exclude chains based on "appropriateness" judgment
- User will approve which chains to process
- See "CRITICAL EXECUTION POLICY" at top of this command

5. **Present optimal combination with complete details and get batch approval**

   **Pre-approval validation:**

   For each chain in optimal combination:
   - Verify entry point format matches requirements:
     - Task: "Task [name]" with cmd/[name]/main.go reference
     - API: "API [METHOD] [path]" with handler file:line reference
     - CLI: "CLI [command]" with cmd/cli/*/main.go reference

   - If entry point is ambiguous (e.g., "API (various handlers)", missing handler, no router definition):
     - ERROR: "Invalid entry point format for chain: [chain_description]"
     - Log: "Chain excluded from batch approval: [file:line] [function_name] - [reason]"
     - Remove from optimal combination
     - Continue validation with remaining chains

   - If all chains fail validation:
     - Output: "バッチ承認可能なチェーンがありません。全てのチェーンでエントリーポイントが特定できませんでした。"
     - Exit command

   A. Display complete details for each chain before approval:

   **For each chain, display:**
   1. Entry point type and identifier (API/Task/CLI)
   2. Complete call chain with file:line for ALL functions
   3. External dependencies (HTTP clients, other services)
   4. SDK operation details:
      - Operation type (Create/Update/Read/Delete)
      - Resource (table name, bucket name)
      - Pre-insert requirements

   **Display format for single SDK operation:**
   ```
   [N]. Chain: [entry_type] [identifier]

   Entry → Complete call chain:
   Entry: [type] [identifier]
   → [file:line] EntryFunction                           [External: None]
   → [file:line] IntermediateFunction1                   [External: HTTP Client]
   → [file:line] IntermediateFunction2                   [External: None]
   → [file:line] SDKFunction                             ← [Service] [Operation]

   SDK Operation:
   - Type: [Create/Update/Read/Delete]
   - Resource: [table_name/bucket_name]
   - Pre-insert required: [Yes/No]

   External Dependencies: [count] ([list])
   Estimated complexity: [Low/Medium/High] ([N] operations, [M] external deps)
   ```

   **Display format for multiple SDK operations:**
   ```
   [N]. Chain: [entry_type] [identifier] [★ Multiple SDK: M operations]

   Entry → Complete call chain:
   Entry: [type] [identifier]
   → [file:line] EntryFunction                           [External: None]
   → [file:line] IntermediateFunction                    [External: None]

   → [file:line] BranchFunction1                         [External: None]
   → [file:line] SDKFunction1                            ← [Service] [Operation1]

   → [file:line] BranchFunction2                         [External: HTTP Client]
   → [file:line] SDKFunction2                            ← [Service] [Operation2]

   → [file:line] BranchFunction3                         [External: None]
   → [file:line] SDKFunction3                            ← [Service] [Operation3]

   SDK Operations Summary:
   - [Service] [Operation1] ([Type]): Pre-insert [required/not needed]
   - [Service] [Operation2] ([Type]): Pre-insert [required/not needed]
   - [Service] [Operation3] ([Type]): Pre-insert [required/not needed]

   External Dependencies: [count] ([list])
   Estimated complexity: [Low/Medium/High] ([M] operations, [N] external deps)
   ```

   **Example (multiple SDK operations with complete details):**
   ```
   3. Chain: Task integrated_task [★ Multiple SDK: 4 operations]

   Entry → Complete call chain:
   Entry: Task integrated_task
   → cmd/integrated_task/main.go:136 main()
   → internal/tasks/worker.go:40 Execute()
   → internal/tasks/worker.go:41 dataRepo.GetByIndex()     [External: None]
   → internal/service/datastore.go:79 GetByIndex()
   → internal/service/datastore.go:105 db.Query()         ← DynamoDB Query

   → internal/tasks/worker.go:134 counterRepo.GetNext()   [External: None]
   → internal/service/counter.go:37 GetNext()
   → internal/service/counter.go:60 db.UpdateItem()              ← DynamoDB UpdateItem (×2)

   → internal/tasks/worker.go:167 fileRepo.Insert()       [External: None]
   → internal/service/storage.go:235 InsertRecord()
   → internal/service/storage.go:254 db.PutItem()          ← DynamoDB PutItem (×2)

   → internal/tasks/worker.go:116 dataRepo.Update()       [External: None]
   → internal/service/datastore.go:421 Update()
   → internal/service/datastore.go:519 db.TransactWriteItems() ← DynamoDB TransactWriteItems

   SDK Operations Summary:
   - DynamoDB Query (Read): Requires Pre-insert
   - DynamoDB UpdateItem (Update): Requires Pre-insert
   - DynamoDB PutItem (Create): No Pre-insert needed
   - DynamoDB TransactWriteItems (Update): Requires Pre-insert

   External Dependencies: None (all operations are AWS SDK only)
   Estimated complexity: Medium (4 operations, 0 external deps)
   ```

   B. Display summary of optimal combination:
   ```
   === バッチ処理する組み合わせサマリー ===

   合計SDK関数数: N個
   - Create操作: A個
   - Update操作: B個 (Pre-insert必要)
   - Read操作: X個 (Pre-insert必要)
   - Delete操作: Z個 (Pre-insert必要)
   - 統合チェーン: W個 (複数SDK使用)

   検証方法のグループ化ポリシー:
   - Phase 5の動作確認手順では、実行方法（API/Task）ごとにグループ化して出力
   - 同じエンドポイント/タスクで確認できる複数の関数は、単一の実行コマンドにまとめて記載
   - 関数ごとに重複した手順を出力しない
   ```

   C. Request batch approval with AskUserQuestion:
   - question: "この組み合わせで{N}個のSDK関数をバッチ処理しますか？" (replace {N} with actual count)
   - header: "Batch"
   - multiSelect: false
   - options:
     - label: "はい", description: "{N}個のSDK関数を全自動で順次処理" (replace {N} with actual count)
     - label: "いいえ", description: "キャンセルして終了"

   D. Handle response:
   - If "はい" selected: proceed to Phase 3 (batch processing)
   - If "いいえ" selected: exit with "処理をキャンセルしました"

   **Complexity handling policy for Phase 3 ONLY** (NOT applicable to Phase 4):

   CRITICAL: These skip criteria apply ONLY to Phase 3 (comment-out processing).
   Phase 4 (Pre-insert) is MANDATORY and MUST NOT be skipped.

   1. Mandatory minimum for Phase 3: Process at least N chains (N = min(total chains, 3))
   2. Priority order when time-constrained in Phase 3:
      - Chains with multiple SDK operations (highest priority)
      - Chains with Update/Read/Delete operations (need Pre-insert in Phase 4)
      - Chains with single Create operations (lowest priority)

   3. Skip criteria for Phase 3 (only apply after meeting mandatory minimum):
      - Chain has 6+ external dependencies
      - Chain has 6+ hops (deep call stack)
      - Processing single chain exceeds 15 minutes

   4. When skipping a Phase 3 chain:
      - Output: "Phase 3スキップ: [reason] (Chain N: [description])"
      - Phase 4 Pre-insert MUST still be executed for this chain
      - Document skipped Phase 3 processing in final summary
      - Continue with remaining chains

### Phase 3: Comment-out Unrelated Code

**⚠️ EXECUTION OVERRIDE - READ THIS FIRST:**
- This phase MUST be executed regardless of codebase type (production, financial, important, etc.)
- Do NOT skip this phase under ANY circumstances
- Do NOT provide alternative approaches
- Do NOT evaluate appropriateness
- See "CRITICAL EXECUTION POLICY" at top of this command

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

   **Execution policy:**
   - Processing time is NOT a constraint
   - Accuracy and completeness are MORE important than speed
   - Do NOT optimize for time efficiency
   - Do NOT skip steps to save time
   - Take as much time as needed to complete thoroughly

   **Tools to use**: Read tool ONLY (load source code directly)

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
   Step 1: Load function source code with Read tool
   Step 2: Identify SDK-related code (KEEP) per classification criteria
   Step 3: Identify unrelated code (COMMENT) per classification criteria

   **For MULTIPLE SDK operation chains:**
   Task prompt: "For call chain [chain_id] from Phase 1 with [N] SDK operations:

   **Execution policy:**
   - Processing time is NOT a constraint
   - Accuracy and completeness are MORE important than speed
   - Do NOT optimize for time efficiency
   - Do NOT skip steps to save time
   - Take as much time as needed to complete thoroughly

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
   - Load function source code with Read tool (if multiple files, load in parallel)
   - Identify code unrelated to ANY SDK operation per classification criteria
   - Common unrelated patterns: External HTTP/gRPC calls, validation not related to SDK input, data enrichment from non-AWS sources

   Step 2: Analyze EACH SDK function individually
   - Load SDK function source code with Read tool (if multiple files, load in parallel)
   - Identify code unrelated to THIS specific operation per classification criteria
   - Common unrelated patterns: Response parsing (parseAttributes, loops over resp.Items), entity transformation (ToEntity), pagination logic, detailed error wrapping

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
   log.Printf("processing")
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
          - Format: Comment out all lines with `//`

   Example Edit tool usage:
   ```go
   old_string:
   userData, err := h.userRepo.GetUser(ctx, userID)
   if err != nil {
       return err
   }

   new_string:
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

**⚠️ EXECUTION OVERRIDE - READ THIS FIRST:**
- This phase MUST be executed regardless of codebase type (production, financial, important, etc.)
- Do NOT skip this phase under ANY circumstances
- Do NOT provide alternative approaches (existing tests, make test, unit tests)
- Do NOT evaluate appropriateness or risk
- See "CRITICAL EXECUTION POLICY" at top of this command

**CRITICAL: This phase is MANDATORY and MUST NOT be skipped under any circumstances.**

**Processing policy:**
- MUST process ALL chains from Phase 1 deduplication result
- Phase 3 completion time does NOT affect Phase 4 execution
- If time constraints exist, reduce Phase 3 scope instead of skipping Phase 4
- **Processing time is NOT a constraint - accuracy is the priority**
- **Do NOT skip or simplify any steps to save time**
- **Each step (8, 9, 10, 10.5) must be executed completely for every chain**

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

   B. Execute steps 8-10.5 for current chain

8. **Analyze AWS SDK operation with Task tool** (subagent_type=general-purpose)
   Task prompt: "For function [function_name] at [file_path:line_number]:

   **Execution policy:**
   - Processing time is NOT a constraint
   - Accuracy and completeness are MORE important than speed
   - Do NOT optimize for time efficiency
   - Do NOT skip steps to save time
   - Take as much time as needed to complete thoroughly

   **Tools**: Read for source code, Grep for pattern searches (execute independent Grep searches in parallel)

   **Context from Phase 1**:
   - SDK operation: [operation_name] (e.g., DynamoDB PutItem)
   - Operation type classification: Create (PutItem, PutObject, SendEmail, Publish), Update (UpdateItem, TransactWriteItems), Read (Query, GetItem, Scan, GetObject), Delete (DeleteItem, DeleteObject)

   **Extract AWS settings** (execute Grep searches in parallel):
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

   **Execution policy:**
   - Processing time is NOT a constraint
   - Accuracy and completeness are MORE important than speed
   - Do NOT optimize for time efficiency
   - Do NOT skip steps to save time
   - Take as much time as needed to complete thoroughly

   **Tools**: Read to extract SDK call parameters

   **Context from Phase 1**:
   - Operations in chain: [list]
   - Operation type classification (from Step 8): Create/Update/Read/Delete

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

10.5. **Verify call chain execution flow with test data**

   Execute execution flow analysis for current chain.

   A. **Analyze execution flow with Task tool** (subagent_type=general-purpose)

   Task prompt: "For current call chain being processed:

   **Execution policy:**
   - Processing time is NOT a constraint
   - Accuracy and completeness are MORE important than speed
   - Do NOT optimize for time efficiency
   - Do NOT skip steps to save time
   - Take as much time as needed to complete thoroughly

   **Tools**: Read tool to load function source code

   **Target chain**: [Copy chain details from loop context]
   - Entry: [type] [identifier]
   - Call chain: [file:line] function list from entry to SDK function
   - SDK operation: [Service] [Operation]

   **Analysis objective**: Verify test data enables expected execution flow

   **Analysis procedure**:

   Step 1: Load all functions in call chain
   - Entry function at [file:line]
   - ALL intermediate functions at [file:line]
   - SDK function at [file:line]

   Step 2: Trace execution flow with test data context
   - Identify conditional branches (if/switch/for)
   - Check each branch condition against test data
   - Verify expected path is taken

   Step 3: Identify potential runtime issues
   - Error handling that may trigger prematurely
   - Uninitialized variables accessed in execution path
   - Type mismatches with dummy values
   - Nil pointer dereferences

   Step 4: Classify findings by severity
   - Critical: Causes compilation error or runtime panic
   - Warning: May cause unexpected path or early return
   - Info: Optimization opportunity

   **Return format**:
   ```
   Chain: [entry_type] [identifier]
   SDK function: [file:line] [function_name]

   Analysis results:
   - Conditional branches: [N branches analyzed]
     - Expected path: [OK / ISSUE]
   - Error handling: [M error checks analyzed]
     - Premature errors: [None / List]
   - Variable initialization: [OK / ISSUE]
   - Type compatibility: [OK / ISSUE]

   Issues found (if any):
   [Severity] [file:line] [description]
   - Suggested fix: [description]

   Overall: [PASS / NEEDS_FIX]
   ```"

   B. **Fix issues if found**

   If Step A returned "Overall: NEEDS_FIX":

   1. For each issue in priority order (Critical → Warning → Info):
      - Apply fix using Edit tool
      - Output: "修正適用: [file:line] [description]"

   2. Verify compilation after ALL fixes:
      - Run: `go build -o /tmp/test-build 2>&1`
      - If fails: Analyze error, apply additional fixes, retry
      - Repeat until success
      - Output: "コンパイル成功: 実行フロー修正完了"

   If Step A returned "Overall: PASS":
      - Skip to Step C

   C. **Display verification result**

   For single SDK operation:
   ```
   === 実行フロー検証 (i/N) ===
   Chain: [entry_type] [identifier]
   SDK function: [file:line] [function_name]

   分析結果:
   - 条件分岐: [OK / 修正適用]
   - エラーハンドリング: [OK / 修正適用]
   - 変数初期化: [OK / 修正適用]
   - 型互換性: [OK / 修正適用]

   修正内容: (修正があった場合のみ)
   - [file:line] [description]

   検証完了 (i/N): 実行フロー
   ```

   For multiple SDK operations:
   ```
   === 実行フロー検証 (i/N, SDK function [N]-A) ===
   Chain: [entry_type] [identifier] [★ Multiple SDK]
   SDK function: [N]-A. [file:line] [function_name]

   分析結果:
   - 条件分岐: [OK / 修正適用]
   - エラーハンドリング: [OK / 修正適用]
   - 変数初期化: [OK / 修正適用]
   - 型互換性: [OK / 修正適用]

   修正内容: (修正があった場合のみ)
   - [file:line] [description]

   検証完了 (i/N, SDK function [N]-A): 実行フロー
   ```

11. **Automatic progression**
   - If i < N: continue to next chain (repeat from step 7.A)
   - If i = N:
     - Verify Phase 4 completion: ALL chains must have Pre-insert analysis (even if "No Pre-insert needed")
     - Count processed chains: Must equal N (total chains from Phase 1)
     - If any chain missing Phase 4 processing: ERROR "Phase 4 incomplete: [N - processed] chains not processed" and HALT
     - If all N chains processed: proceed to step 12

12. **Verify Pre-insert code completeness**

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

   4. After verification: proceed to Phase 3.5

### Phase 3.5: Verify Comment-out Completeness

After Phase 3, verify all unrelated code is commented out by executing Grep searches in parallel (independent):

1. Verify external service calls are commented:
   - Grep: `pattern: "http\\.(Get|Post|Client)|grpc\\.(Dial|NewClient)"`, `output_mode: "content"`, `-C: 5`, `glob: "!(*_test.go)"`
   - For each match in modified files, check if line starts with `//`
   - If uncommented in modified chain functions: ERROR - re-run Phase 3

2. Verify response processing is minimized:
   - Grep: `pattern: "parseAttributes|ToEntity|for.*resp\\.(Items|Records)"`, `output_mode: "content"`, `glob: "!(*_test.go)"`
   - For each match in modified files, check if commented or replaced with log.Printf
   - If complex processing remains in modified chain functions: ERROR - re-run Phase 3

### Phase 5: AWS Verification Procedures

12. **Generate AWS verification procedures with complete call chain details**

    Output AWS-specific verification procedures for each call chain.

    **For EACH chain from Phase 1 deduplication result:**

    Display format for single SDK operation:
    ```markdown
    ## Chain [N]: [entry_type] [identifier]

    ### コールチェーン
    Entry: [type] [identifier]
    → [file:line] EntryFunction
    → [file:line] IntermediateFunction1
    → [file:line] IntermediateFunction2
    → [file:line] SDKFunction  ← [Service] [Operation]

    ### 実行コマンド
    ```bash
    # For API endpoint
    curl -X [METHOD] https://[host]/[path] \
      -H "Content-Type: application/json" \
      -d '{"key":"value"}'

    # For Task
    aws ecs run-task \
      --cluster [cluster-name] \
      --task-definition [task-name]
    ```

    ### X-Ray確認ポイント
    - [Service] [Operation] × N回
    - Response: [Expected behavior]
    ```

    Display format for multiple SDK operations:
    ```markdown
    ## Chain [N]: [entry_type] [identifier] [★ Multiple SDK: M operations]

    ### コールチェーン
    Entry → Intermediate layers (共通):
    Entry: [type] [identifier]
    → [file:line] EntryFunction
    → [file:line] IntermediateFunction

    SDK Functions (個別):
    [N]-A. [file:line] IntermediateFunction → ... → [file:line] SDKFunction1
           Operation: [Service] [Operation1]

    [N]-B. [file:line] IntermediateFunction → ... → [file:line] SDKFunction2
           Operation: [Service] [Operation2]

    ### 実行コマンド
    ```bash
    # Single execution command covers all SDK operations
    [command based on entry point type]
    ```

    ### X-Ray確認ポイント
    - [Service] [Operation1] × N回
    - [Service] [Operation2] × M回
    - Data flow: Operation1 → Operation2 → ...
    ```

    **Complete example:**
    ```markdown
    ## Chain 1: Task batch_task [★ Multiple SDK: 4 operations]

    ### コールチェーン
    Entry → Intermediate layers (共通):
    Entry: Task batch_task
    → cmd/batch_task/main.go:136 main()
    → internal/tasks/batch_worker.go:40 Execute()

    SDK Functions (個別):
    A. internal/tasks/batch_worker.go:41 → internal/service/entity_datastore.go:79 GetByIndex()
       → internal/service/entity_datastore.go:105 db.Query()
       Operation: DynamoDB Query

    B. internal/tasks/batch_worker.go:134 → internal/service/counter.go:37 GetNext()
       → internal/service/counter.go:60 db.UpdateItem()
       Operation: DynamoDB UpdateItem (×2)

    C. internal/tasks/batch_worker.go:167 → internal/service/file_storage.go:235 insertRecord()
       → internal/service/file_storage.go:254 db.PutItem()
       Operation: DynamoDB PutItem (×2)

    D. internal/tasks/batch_worker.go:116 → internal/service/entity_datastore.go:421 Update()
       → internal/service/entity_datastore.go:519 db.TransactWriteItems()
       Operation: DynamoDB TransactWriteItems

    ### 実行コマンド
    ```bash
    aws ecs run-task \
      --cluster production-cluster \
      --task-definition batch_task:latest \
      --launch-type FARGATE
    ```

    ### X-Ray確認ポイント
    - DynamoDB Query × 1回 (entities table)
    - DynamoDB UpdateItem × 2回 (counter table)
    - DynamoDB PutItem × 2回 (files table)
    - DynamoDB TransactWriteItems × 1回 (entities table)
    - Data flow: Query entities → Update counter → Put files → TransactWrite entities
    ```

    **DO NOT include:**
    - Summary statistics (processed SDK functions count, etc.)
    - File modification details (commented files, edited files)
    - Compilation results
    - Next steps or actions

## Output Format

### Optimal Combination (Phase 1)
```
=== 重複排除と最適化後の組み合わせ ===

合計SDK関数数: 7個 (重複排除前: 10個のチェーン)
選択されたチェーン数: 4個

[Sorted by priority: 統合チェーン (4+ operations) first, then 中規模チェーン (2-3 operations), then 単一操作チェーン]

1. 統合Chain: Task integrated_task_a [★ Multiple SDK: 3 operations]

   Entry → Intermediate layers (共通):
   Entry: Task integrated_task_a
   → cmd/integrated_task_a/main.go:100 main
   → internal/tasks/worker.go:50 Execute
   → internal/service/processor.go:80 ProcessData

   SDK Functions (個別):
   1-A. internal/service/processor.go:120 CreateRecord()
        → internal/service/processor.go:125 db.PutItem()
        Operation: DynamoDB PutItem

   1-B. internal/service/processor.go:150 StoreFile()
        → internal/storage/s3_client.go:45 client.PutObject()
        Operation: S3 PutObject

   1-C. internal/service/processor.go:180 SendNotification()
        → internal/notification/email.go:30 client.SendEmail()
        Operation: SES SendEmail

   (3 SDK operations, 5 hops) Active callers: 2箇所

2. 中規模Chain: POST /v1/resources/process [★ Multiple SDK: 2 operations] [+1 other chain]

   Entry → Intermediate layers (共通):
   Entry: API POST /v1/resources/process
   → internal/api/handler/v1/resource.go:80 HandleProcess
   → internal/gateway/resource_gateway.go:89 ProcessResource

   SDK Functions (個別):
   2-A. internal/gateway/resource_gateway.go:120 FetchData()
        → internal/storage/s3_gateway.go:67 client.GetObject()
        Operation: S3 GetObject

   2-B. internal/gateway/resource_gateway.go:200 SaveBatch()
        → internal/repository/batch_repo.go:123 client.BatchWriteItem()
        Operation: DynamoDB BatchWriteItem

   (2 SDK operations, 4 hops) Active callers: 1箇所

3. 単一操作Chain: Task simple_task_a [+2 other chains]

   Entry → Complete call chain:
   Entry: Task simple_task_a
   → cmd/simple_task_a/main.go:50 main()
   → internal/usecase/processor.go:30 Execute()
   → internal/repository/data_repo.go:45 Save()
   → internal/repository/data_repo.go:48 db.PutItem()  ← DynamoDB PutItem

   (1 SDK operation, 4 hops) Active callers: 3箇所

4. 単一操作Chain: API GET /v1/resources/:id

   Entry → Complete call chain:
   Entry: API GET /v1/resources/:id
   → internal/api/handler/v1/resource.go:100 GetResource()
   → internal/service/resource_service.go:50 Fetch()
   → internal/repository/resource_repo.go:89 Get()
   → internal/repository/resource_repo.go:92 db.GetItem()  ← DynamoDB GetItem

   (1 SDK operation, 4 hops) Active callers: 2箇所
```

### Batch Approval Summary (Phase 2)
```
=== バッチ処理する組み合わせ（詳細版） ===

合計SDK関数数: 6個
- Create操作: 3個
- Update操作: 0個
- Read操作: 2個 (Pre-insert必要)
- Delete操作: 0個
- 統合チェーン: 2個 (複数SDK使用)

処理対象のチェーン:

1. 統合Chain: Task integrated_task_a [★ Multiple SDK: 3 operations]

   Entry → Complete call chain:
   Entry: Task integrated_task_a
   → cmd/integrated_task_a/main.go:100 main()
   → internal/tasks/worker.go:50 Execute()
   → internal/service/processor.go:80 ProcessData()

   → internal/service/processor.go:120 CreateRecord()        [External: None]
   → internal/service/processor.go:125 db.PutItem()          ← DynamoDB PutItem

   → internal/service/processor.go:150 StoreFile()           [External: None]
   → internal/storage/s3_client.go:45 client.PutObject()        ← S3 PutObject

   → internal/service/processor.go:180 SendNotification()         [External: None]
   → internal/notification/email.go:30 client.SendEmail()       ← SES SendEmail

   SDK Operations Summary:
   - DynamoDB PutItem (Create): No Pre-insert needed
   - S3 PutObject (Create): No Pre-insert needed
   - SES SendEmail (Create): No Pre-insert needed

   External Dependencies: None
   Estimated complexity: Low (3 operations, 0 external deps)

2. 中規模Chain: POST /v1/resources/process [★ Multiple SDK: 2 operations]

   Entry → Complete call chain:
   Entry: API POST /v1/resources/process
   → internal/api/handler/v1/resource.go:80 HandleProcess()
   → internal/gateway/resource_gateway.go:89 ProcessResource()

   → internal/gateway/resource_gateway.go:120 FetchData()           [External: None]
   → internal/storage/s3_gateway.go:67 client.GetObject()       ← S3 GetObject

   → internal/gateway/resource_gateway.go:200 SaveBatch()           [External: None]
   → internal/repository/batch_repo.go:123 client.BatchWriteItem() ← DynamoDB BatchWriteItem

   SDK Operations Summary:
   - S3 GetObject (Read): Requires Pre-insert
   - DynamoDB BatchWriteItem (Create): No Pre-insert needed

   External Dependencies: None
   Estimated complexity: Medium (2 operations, 0 external deps)

3. 単一操作Chain: Task simple_task_a

   Entry → Complete call chain:
   Entry: Task simple_task_a
   → cmd/simple_task_a/main.go:50 main()
   → internal/usecase/processor.go:30 Execute()              [External: None]
   → internal/repository/data_repo.go:45 Save()
   → internal/repository/data_repo.go:48 db.PutItem()     ← DynamoDB PutItem

   SDK Operation:
   - Type: Create
   - Resource: table_a
   - Pre-insert required: No

   External Dependencies: None
   Estimated complexity: Low (1 operation, 0 external deps)

4. 単一操作Chain: API GET /v1/resources/:id

   Entry → Complete call chain:
   Entry: API GET /v1/resources/:id
   → internal/api/handler/v1/resource.go:100 GetResource()
   → internal/service/resource_service.go:50 Fetch()                [External: None]
   → internal/repository/resource_repo.go:89 Get()
   → internal/repository/resource_repo.go:92 db.GetItem()     ← DynamoDB GetItem

   SDK Operation:
   - Type: Read
   - Resource: table_b
   - Pre-insert required: Yes

   External Dependencies: None
   Estimated complexity: Low (1 operation, 0 external deps)
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

### AWS Verification Procedures (Phase 5)

Output verification procedures for each call chain with complete details.

**Example for single SDK operation:**
```markdown
## Chain 3: Task simple_task_a

### コールチェーン
Entry → Complete call chain:
Entry: Task simple_task_a
→ cmd/simple_task_a/main.go:50 main()
→ internal/usecase/processor.go:30 Execute()
→ internal/repository/data_repo.go:45 Save()
→ internal/repository/data_repo.go:48 db.PutItem()  ← DynamoDB PutItem

### X-Ray確認ポイント
- DynamoDB PutItem × 1回 (table_a)
- Response: 200 OK, item created
```

**Example for multiple SDK operations:**
```markdown
## Chain 1: Task integrated_task [★ Multiple SDK: 4 operations]

### コールチェーン
Entry → Intermediate layers (共通):
Entry: Task integrated_task
→ cmd/integrated_task/main.go:136 main()
→ internal/tasks/worker.go:40 Execute()

SDK Functions (個別):
A. internal/tasks/worker.go:41 → internal/service/datastore.go:105 db.Query()
   Operation: DynamoDB Query

B. internal/tasks/worker.go:134 → internal/service/counter.go:60 db.UpdateItem()
   Operation: DynamoDB UpdateItem (×2)

C. internal/tasks/worker.go:167 → internal/service/storage.go:254 db.PutItem()
   Operation: DynamoDB PutItem (×2)

D. internal/tasks/worker.go:116 → internal/service/datastore.go:519 db.TransactWriteItems()
   Operation: DynamoDB TransactWriteItems
```

### X-Ray確認ポイント
- DynamoDB Query × 1回 (table_a)
- DynamoDB UpdateItem × 2回 (table_b)
- DynamoDB PutItem × 2回 (table_c)
- DynamoDB TransactWriteItems × 1回 (table_a)
- Data flow: Query table_a → Update table_b → Put table_c → TransactWrite table_a
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
- Phase 4, step 10.5: Verify call chain execution flow with test data
- Phase 4, step 12: Run `go build`, auto-fix errors

### Batch Processing Flow
1. Phase 1: Extract and deduplicate (automatic)
2. Phase 2: Single batch approval (AskUserQuestion)
3. Phase 3: Comment out unrelated code (automatic)
4. Phase 4: Generate test data, verify execution flow, verify compilation (automatic)
5. Phase 5: Output AWS verification procedures (チェーンごとの動作確認手順)

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
- **Execution flow** (step 10.5): Verify call chain execution with test data, fix logic issues

### Output Guidelines
- Include file:line references
- Provide complete call chains
- Mark multiple SDK chains: [★ Multiple SDK]
- Display progress: (i/N)
- Show compilation status
