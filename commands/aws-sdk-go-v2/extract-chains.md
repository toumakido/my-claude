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

### Phase 1: SDK Function Discovery

1. **Discover SDK functions using agent**

   Use Task tool with subagent_type: general-purpose

   Task prompt: "Find all AWS SDK v2 operation calls in the current branch changes.

   **Execution policy:** See EXECUTION POLICY FOR TASK TOOL INVOCATIONS section

   Identify changes that include github.com/aws/aws-sdk-go-v2 imports.
   Use the most appropriate method to discover SDK operation calls.

   **Target operations only:**
   - Data operations: PutItem, GetItem, Query, Scan, UpdateItem, DeleteItem, etc.
   - Service operations: Invoke, Publish, SendMessage, PutObject, GetObject, etc.
   - Exclude client initialization: NewFromConfig, New*Client, LoadDefaultConfig, etc.

   If no SDK v2 operation calls found:
   - Output: 'このブランチはAWS SDK Go v2関連の変更を含んでいません'
   - Exit

   For each SDK operation call found:
   - Extract the enclosing Go function name
   - Record file path and line number

   Return list with format: file:line FunctionName

   Example output:
   - internal/service/datastore.go:79 GetByIndex
   - internal/service/counter.go:37 GetNext
   - internal/service/storage.go:235 Insert

   Total: N functions"

### Phase 2: Parallel Call Chain Tracing

2. **Trace call chains using go-call-chain-tracer agent**

   **IMPORTANT:** Use parallel execution to minimize processing time.

   For each SDK function discovered in Phase 1, launch a Task with subagent_type: go-call-chain-tracer

   **Parallel execution strategy:**
   - If SDK functions ≤ 5: Launch all agents in parallel (single message with multiple Task tool calls)
   - If SDK functions > 5: Launch in batches of 5, wait for completion, then launch next batch

   **Task prompt for each SDK function:**
   ```
   Trace the function "[FunctionName]" at "[filepath:line]".

   Focus on production entry points only:
   - API endpoints (HTTP handlers)
   - Task entry points (cmd/*/main.go)
   - CLI commands (cli.Command)

   Exclude test files, mocks, and internal/repository/mocks/.

   **IMPORTANT for API endpoints:**
   - Start call chain from HTTP handler function (e.g., HandleGetEntities)
   - DO NOT include main.go or router setup in call chain
   - Record endpoint information separately (method, path, handler location)

   **Output format:**
   Your response MUST be valid JSON only (no markdown, no explanations).
   Return the complete call chain from each entry point to this function in the JSON format specified in your agent definition.
   ```

   **Example Task invocations:**
   ```
   Task 1: "Trace GetByIndex at internal/service/datastore.go:79"
   Task 2: "Trace GetNext at internal/service/counter.go:37"
   Task 3: "Trace Insert at internal/service/storage.go:235"
   ```

3. **Integrate agent results (JSON format)**

   **IMPORTANT:** go-call-chain-tracer agent returns structured JSON output. Parse the JSON to extract chain information.

   For each agent result:

   a. **Parse JSON output**
      - Agent returns JSON with structure:
        ```json
        {
          "target_function": {...},
          "call_chains": [...],
          "statistics": {...}
        }
        ```
      - Extract `call_chains` array from JSON

   b. **Process each call chain from JSON**
      - Extract `entry_point_type`, `entry_point_identifier`, `entry_point_location`
      - Extract `chain` array (each element has `file`, `line`, `function`)
      - Extract `endpoint` object (for API type only)
      - Extract `sdk_operations` array (with `service`, `operation`, `type`)
      - Extract `depth` value

   c. **Transform for .migration-chains.json format**
      - Convert JSON structure to .migration-chains.json format:
        - `entry_point_type` → `type`
        - `entry_point_identifier` → `identifier`
        - `endpoint` → `endpoint` (keep as-is for API)
        - `chain` → `call_chain` (keep file:line:function structure)
        - `sdk_operations` → `sdk_operations` (keep as-is)

   d. **Validation**
      - Mark chains without entry points as "SKIP - No entry point"
      - Verify all required fields are present
      - Count SDK operations per entry point

### Phase 3: Deduplication

4. **Deduplicate chains using coverage-based selection**

   Use Task tool with subagent_type: general-purpose

   Task prompt: "Select minimum chains covering all unique AWS SDK operations.

   **Execution policy:** See EXECUTION POLICY FOR TASK TOOL INVOCATIONS section

   **Objective**: Select minimum chains covering all unique AWS SDK operations. Prioritize chains with multiple SDK operations.

   **Priority order**:
   - Integrated chains (4+ SDK ops) > Medium chains (2-3 ops) > Single-op chains (1 op)
   - Within same priority: Shorter chains preferred

   **Selection rule**: For each chain, select it if it covers new SDK operations not yet covered. Skip chains with "SKIP - No entry point".

   **Operation comparison**: Service + Operation name must match (e.g., "DynamoDB PutItem"). Ignore parameters like table names.

   Return: Selected chains (optimal combination), skipped chains with reasons"

### Phase 4: Output

5. **Validate and save chains**

   A. Validation:

   For each chain in optimal combination:
   - Verify entry point format matches requirements:
     - Task: "Task [name]" with cmd/[name]/main.go reference
     - API: "API [METHOD] [path]" with handler file:line reference
     - CLI: "CLI [command]" with cmd/cli/*/main.go reference
   - Remove chains with invalid entry points

   If all chains fail validation:
   - Output: "有効なチェーンが見つかりませんでした。全てのチェーンでエントリーポイントが特定できませんでした。"
   - Exit

   B. Save to JSON:

   Create `.migration-chains.json` in repository root using Write tool
   - Include: chains with call_chain/sdk_operations, summary with totals, skipped chains with reasons

   C. Display summary:
   ```
   === 抽出完了 ===

   選択されたチェーン: N個
   - 統合チェーン: X個（複数SDK操作）
   - 単一操作チェーン: Y個

   カバーするSDK操作: M個
   - Create: A個
   - Read: B個
   - Update: C個
   - Delete: D個

   スキップされたチェーン: Z個（重複によりカバー済み）

   保存先: .migration-chains.json

   次のステップ: /prepare-tests コマンドでテスト準備と検証手順書を生成
   ```

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

**"有効なチェーンが見つかりませんでした"**
- All chains failed entry point validation
- Solution: Check if SDK operations are reachable from API/Task/CLI entry points

**go-call-chain-tracer agent fails**
- Function not found or unreachable
- Solution: Verify code compiles and function is called from production entry points

**Parallel agent execution timeout**
- Too many SDK functions to trace
- Solution: Process in smaller batches or increase timeout

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
    "endpoint": {
      "method": "POST",
      "path": "/v1/entities",
      "handler": "internal/api/handler.go:50:CreateEntity"
    },
    "call_chain": [
      {"file": "internal/api/handler.go", "line": 50, "function": "CreateEntity"},
      {"file": "internal/service/entity.go", "line": 80, "function": "Save"}
    ],
    "sdk_operations": [{"service": "DynamoDB", "operation": "PutItem", "type": "Create"}]
  }],
  "summary": {"total_chains": 1, "total_sdk_ops": 1}
}
```

## Next Steps

1. Review `.migration-chains.json`
2. Run `/aws-sdk-go-v2:prepare-tests` command to:
   - Modify code for testing
   - Generate AWS verification procedures
3. Deploy to test environment and execute verification procedures
