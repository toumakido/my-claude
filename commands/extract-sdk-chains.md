# Extract AWS SDK v2 Call Chains

This command extracts AWS SDK v2 migration call chains from git diff and deduplicates them for verification testing.

## Usage

```
/extract-sdk-chains
```

Run this command from the repository root directory.

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

   **Step 1: Find entry points**
   Execute 3 Grep searches in parallel (independent):
   1. `pattern: "router\\.(POST|GET|PUT|DELETE|PATCH)"`, `output_mode: "content"`, `-C: 3`
   2. `pattern: "func main"`, `path: "cmd/"`, `output_mode: "content"`, `-C: 5`
   3. `pattern: "cli\\.(Command|App)"`, `output_mode: "content"`, `-C: 3`

   From results, extract: entry type (API/Task/CLI), identifier, file:line

   **Step 2: Trace call chains**
   For each entry point:
   - Read entry function source with Read tool
   - Extract all function calls from source
   - For each call: Use Grep to find function definition → Read source → Verify call relationship exists
   - Recurse until finding SDK operations matching: `client.(PutItem|GetObject|Query|UpdateItem|DeleteItem|PutObject|DeleteObject|SendEmail|Publish|TransactWriteItems|BatchWriteItem)`
   - Record complete chain with file:line for each hop: `[file:line] Func1 → [file:line] Func2 → ... → [file:line] SDKFunc`
   - Chain breaks if: (a) function definition not found via Grep, OR (b) function not called from any entry point
   - If chain breaks: Mark as "SKIP - BROKEN CHAIN"

   **Step 3: Classify by SDK operation count**
   - Integrated chains (4+ SDK ops): priority=highest
   - Medium chains (2-3 SDK ops): priority=medium
   - Single-op chains (1 SDK op): priority=lowest

   **Step 4: Validate entry points**
   For chains without clear entry (not from cmd/*/main.go or API handler):
   - Use Grep to search function usage in handler layer (pattern: function name)
   - Use Grep to find router registration (pattern: router method calls)
   - If no usage found in handlers and no router registration: Mark as "SKIP - No entry point"

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

   Return: All chains sorted by priority (Integrated → Medium → Single-op), skipped functions list with reasons."

3. **Deduplicate chains**

   Use Task tool with subagent_type: general-purpose

   Task prompt: "Deduplicate call chains using coverage-based selection:

   **Objective**: Select minimum chains covering all unique AWS SDK operations.

   **Algorithm**:
   1. Sort chains by: SDK operation count (descending) → chain length (ascending)
   2. Initialize empty set: covered_operations
   3. For each chain in sorted order:
      - Extract operations as "Service Operation" pairs (e.g., "DynamoDB PutItem")
      - Determine new_operations = chain operations NOT in covered_operations
      - If new_operations is not empty:
        * SELECT this chain
        * Add new_operations to covered_operations
        * Mark chain with [+N similar] where N = count of operations already covered
      - If new_operations is empty:
        * SKIP this chain (all operations already covered)
   4. Filter out chains marked "SKIP - No entry point" or "SKIP - BROKEN CHAIN"

   **Operation comparison rule**: Match by Service name + Operation name only (ignore table names, region, parameters).

   Return:
   - List of selected chains (optimal combination for full coverage)
   - List of skipped chains with skip reasons"

### Phase 2: Review and Approval

4. **Display summary and request approval**

   For each selected chain, display:
   - Entry point (API/Task/CLI with identifier)
   - Complete call chain with file:line for each hop
   - SDK operations (single or multiple with service and operation names)
   - External dependencies (if any)
   - Complexity estimate (number of hops)

   Display summary:
   ```
   Total SDK functions: N
   - Create operations: A
   - Update operations: B (Pre-insert required)
   - Read operations: C (Pre-insert required)
   - Delete operations: D (Pre-insert required)
   - Integrated chains: E
   ```

5. **Request batch approval**

   Use AskUserQuestion tool:
   - Question: "この組み合わせで{N}個のSDK関数をバッチ処理しますか？"
   - Options:
     * "はい" (description: "Proceed to save chains to .migration-chains.json")
     * "いいえ" (description: "Exit without saving")

6. **Save chains to JSON**

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
