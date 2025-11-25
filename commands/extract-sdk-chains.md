# Extract AWS SDK v2 Call Chains

This command extracts AWS SDK v2 migration call chains from git diff and deduplicates them for verification testing.

Output language: Japanese, formal business tone

## EXECUTION POLICY FOR TASK TOOL INVOCATIONS

**This policy applies to ALL Task tool invocations throughout this command:**
- Processing time is NOT a constraint
- Accuracy and completeness are MORE important than speed
- Do NOT optimize for time efficiency
- Do NOT skip steps to save time
- Take as much time as needed to complete thoroughly

**Reference:** When you see "**Execution policy:** See EXECUTION POLICY section" in any Task prompt, apply the policy stated above.

## When to Use This Command

Use this command when:
- Starting AWS SDK v2 migration verification workflow
- Need to identify which SDK operations are being migrated
- Want to understand call chains from entry points to SDK calls
- Need optimal chain selection for comprehensive testing

## Prerequisites

- Repository root directory
- Current branch contains AWS SDK v2 changes (`github.com/aws/aws-sdk-go-v2` imports)
- Go development environment
- gh CLI (optional, for PR context)

## Process

### Phase 1: Chain Extraction and Tracing

1. **Validate branch for SDK v2 changes**
   - Run: `git diff main...HEAD`
   - Search output for: `github.com/aws/aws-sdk-go-v2`
   - If not found: Output "このブランチはAWS SDK Go v2関連の変更を含んでいません" and exit
   - If found: Proceed to step 2

2. **Extract entry points and call chains**

   Use Task tool with subagent_type: general-purpose

   Task prompt: "Extract AWS SDK v2 operations by tracing from entry points to SDK calls.

   **Execution policy:** See EXECUTION POLICY FOR TASK TOOL INVOCATIONS section

   **Step 1: Find entry points**
   Execute 3 Grep searches in parallel (independent):
   1. `pattern: "router\\.(POST|GET|PUT|DELETE|PATCH)"`, `output_mode: "content"`, `-C: 3`
   2. `pattern: "func main"`, `path: "cmd/"`, `output_mode: "content"`, `-C: 5`
   3. `pattern: "cli\\.(Command|App)"`, `output_mode: "content"`, `-C: 3`

   From results, extract: entry type (API/Task/CLI), identifier, file:line

   **Step 2: Trace call chains with source code verification**

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
      - Search with Grep tool: `pattern: "func.*[FunctionName]"`, `output_mode: "content"`, `-C: 3`
      - If multiple matches: Use Read tool to load each match, compare receiver type (for methods) or package path (for functions) with caller context, select the matching definition
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
   Verify: main() contains "worker.Execute()" [VERIFIED]
   Chain: main() → worker.go:40 Execute()

   Step 3: Read internal/tasks/worker.go:40
   Found calls: dataRepo.GetByIndex(), counterRepo.GetNext(), fileRepo.Insert()

   Step 4a: Trace dataRepo.GetByIndex()
   Grep "func.*GetByIndex" → Found internal/service/datastore.go:79
   Read datastore.go:79 → Found db.Query() [SDK operation]
   Chain: Execute() → datastore.go:79 GetByIndex() → datastore.go:105 db.Query() [VERIFIED]

   Step 4b: Trace counterRepo.GetNext()
   Grep "func.*GetNext" → Found internal/service/counter.go:37
   Read counter.go:37 → Found db.UpdateItem() [SDK operation]
   Chain: Execute() → counter.go:37 GetNext() → counter.go:60 db.UpdateItem() [VERIFIED]

   Step 4c: Trace fileRepo.Insert()
   Grep "func.*Insert" → Multiple matches
   Filter by receiver type "*fileRepo" → internal/service/storage.go:235
   Read storage.go:235 → Found db.PutItem() [SDK operation]
   Chain: Execute() → storage.go:235 Insert() → storage.go:254 db.PutItem() [VERIFIED]

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
   - Exclude from call chain list

   **Step 3: Classify by SDK operation count**
   - Integrated chains (4+ SDK ops): priority=highest
   - Medium chains (2-3 SDK ops): priority=medium
   - Single-op chains (1 SDK op): priority=lowest

   **Integrated chain criteria:**
   Entry point qualifies as integrated chain if it meets these conditions:
   1. Same function/method calls 2+ different SDK operations directly
   2. OR direct callees (within 1 hop) execute multiple SDK operations
   3. SDK operations have clear dependencies (e.g., Get counter → Insert record)

   **Integrated chain priority in deduplication:**
   - Highest priority: Integrated chains covering multiple operation types eliminate need for single-operation chains
   - Example: If integrated chain covers Query + UpdateItem + PutItem, then individual Query-only chain and UpdateItem-only chain are redundant and excluded
   - Rationale: One integrated chain test verifies multiple SDK operations simultaneously, reducing total test overhead

   **Priority order example:**
   1. Integrated chain with 4 SDK operations, 5 hops (highest)
   2. Medium chain with 3 SDK operations, 4 hops
   3. Medium chain with 2 SDK operations, 2 hops
   4. Medium chain with 2 SDK operations, 5 hops
   5. Single-op chain with 1 SDK operation, 2 hops
   6. Single-op chain with 1 SDK operation, 4 hops (lowest)

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
   1. Search for function usage in handler layer using Grep tool: `pattern: "\\b[function_name]\\b"`, `path: "internal/api/handler/"`, `output_mode: "content"`, `-C: 5`
      Note: Use word boundaries (\\b) to avoid false matches in comments or string literals
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
   Omitting intermediate functions will cause downstream processing to miss external service calls and business logic.

   **Output format (single SDK):**
   ```
   Chain: [type] [identifier]
   Entry → Complete call chain:
   Entry: [type] [identifier]
   → [file:line] EntryFunc
   → [file:line] IntermediateFunc
   → [file:line] SDKFunc ← [Service] [Operation]
   ```

   **Output format (multiple SDK):**
   ```
   Integrated Chain: [type] [identifier] [★ Multiple SDK: N operations]
   Entry → Intermediate layers:
   Entry: [type] [identifier]
   → [file:line] EntryFunc
   → [file:line] IntermediateFunc

   SDK Functions:
   A. [file:line] → ... → [file:line] SDKFunc1
      Operation: [Service] [Op1]
   B. [file:line] → ... → [file:line] SDKFunc2
      Operation: [Service] [Op2]
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

3. **Deduplicate chains**

   Use Task tool with subagent_type: general-purpose

   Task prompt: "Deduplicate call chains using coverage-based selection algorithm:

   **Execution policy:** See EXECUTION POLICY FOR TASK TOOL INVOCATIONS section

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
         if chain.entry_point == "SKIP - No entry point" or "SKIP - BROKEN CHAIN":
            REMOVE chain from selected_chains

   4. Return:
      - selected_chains (optimal combination)
      - skipped_chains with reasons
   ```

   **Operation comparison**: Treat operations as identical if Service + Operation name match. Ignore parameters (table names, filters) as they don't affect SDK v2 migration verification.

   Return: optimal combination (deduplicated chains with verified entry points), skipped chains list with skip reasons"

### Phase 2: Review and Approval

4. **Present optimal combination with complete details and get batch approval**

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
   - If "はい" selected: proceed to save chains to .migration-chains.json
   - If "いいえ" selected: exit with "処理をキャンセルしました"

5. **Save chains to JSON** (only if user approved)

   If user selected "はい":
   - Create `.migration-chains.json` in repository root using Write tool
   - File structure:
     ```json
     {
       "chains": [
         {
           "id": 1,
           "type": "Task",
           "identifier": "batch_task",
           "entry_point": "cmd/batch_task/main.go:136",
           "call_chain": [
             {"file": "cmd/batch_task/main.go", "line": 136, "function": "main"},
             {"file": "internal/tasks/worker.go", "line": 40, "function": "Execute"}
           ],
           "sdk_operations": [
             {
               "id": "1-A",
               "file": "internal/service/datastore.go",
               "line": 105,
               "function": "db.Query",
               "service": "DynamoDB",
               "operation": "Query",
               "type": "Read"
             }
           ],
           "sdk_count": 4,
           "hops": 5,
           "priority": "integrated"
         }
       ],
       "summary": {
         "total_chains": 4,
         "total_sdk_ops": 7,
         "create_ops": 3,
         "update_ops": 2,
         "read_ops": 1,
         "delete_ops": 1
       },
       "skipped": [
         {
           "function": "internal/service/unused.go:123 UnusedFunc",
           "reason": "No entry point"
         }
       ]
     }
     ```
   - Output: "Chains saved to .migration-chains.json (N chains, M SDK operations)"
   - If user selected "いいえ": Exit without creating file

## Output

Creates `.migration-chains.json` containing:
- Selected call chains with complete file:line references
- SDK operations per chain (service, operation type, classification)
- Entry point information (API/Task/CLI)
- Summary statistics
- Skipped chains with reasons

## Validation

After completion, verify:
- [ ] `.migration-chains.json` exists and is valid JSON
- [ ] All selected chains have verified entry points
- [ ] Each chain includes complete file:line references
- [ ] SDK operation types are classified (Create/Update/Read/Delete)
- [ ] Summary totals match chain details

## Error Handling

**"このブランチはAWS SDK Go v2関連の変更を含んでいません"**
- Branch doesn't contain AWS SDK v2 imports
- Solution: Checkout correct branch or make SDK v2 changes

**"バッチ承認可能なチェーンがありません"**
- All chains failed entry point validation
- Solution: Check if SDK operations are reachable from API/Task/CLI entry points

**Task tool fails with "BROKEN CHAIN"**
- Function definition not found during tracing
- Solution: Verify code compiles and function names match

## Example

Input (git diff):
```go
// API endpoint
router.POST("/v1/entities", handler.CreateEntity)

// Handler
func (h *Handler) CreateEntity(c *gin.Context) {
    entity := parseRequest(c)
    err := h.service.Save(ctx, entity)
    // ...
}

// Service (SDK operation)
func (s *Service) Save(ctx context.Context, e Entity) error {
    _, err := s.db.PutItem(ctx, &dynamodb.PutItemInput{
        TableName: aws.String("entities"),
        Item: buildItem(e),
    })
    return err
}
```

Output (.migration-chains.json):
```json
{
  "chains": [{
    "id": 1,
    "type": "API",
    "identifier": "POST /v1/entities",
    "entry_point": "internal/api/handler.go:50",
    "call_chain": [
      {"file": "internal/api/handler.go", "line": 50, "function": "CreateEntity"},
      {"file": "internal/service/entity.go", "line": 80, "function": "Save"},
      {"file": "internal/service/entity.go", "line": 92, "function": "db.PutItem"}
    ],
    "sdk_operations": [{
      "service": "DynamoDB",
      "operation": "PutItem",
      "type": "Create"
    }]
  }]
}
```

## Next Steps

1. Review `.migration-chains.json`
2. Run `/prepare-sdk-tests` command to modify code for testing
3. Deploy to test environment
4. Run `/generate-verification` command for AWS verification procedures
