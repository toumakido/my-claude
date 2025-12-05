# Prepare SDK Code for Testing

This command prepares AWS SDK v2 migrated code for connection testing by modifying code temporarily and generating verification procedures.

Output language: Japanese, formal business tone

**IMPORTANT**: This command modifies production code. All changes are reviewable via `git diff` and can be reverted with `git checkout .`.

## When to Use This Command

Use this command when:
- After running `/extract-chains` and approving chains
- Need to isolate SDK connection code from business logic
- Want to test AWS SDK v2 connections without external dependencies
- Before deploying to test environment for verification

This command performs:
- Code modification (comment out business logic, add pre-insert, add logs)
- Compilation verification
- SDK operation coverage verification
- AWS verification procedure document generation

## Prerequisites

- `.migration-chains.json` exists (created by `/extract-sdk-chains` command)
- Repository root directory
- Go development environment configured
- Clean working directory recommended (commit or stash changes)

## Process

### Phase 1: Comment Out Non-Chain Code (Parallel)

Launch subagents to comment out non-essential code in parallel.

1. **Read `.migration-chains.json`**
   - Load all chains
   - Display: "総チェーン数: N個"

2. **Launch comment-out-non-chain-code subagents in parallel**

   **IMPORTANT**: Use a single message with multiple Task tool calls to run subagents in parallel.

   For each chain:
   - Prepare JSON input:
     ```json
     {
       "chain_id": "chain-1",
       "call_chain": [
         {"file": "handler.go", "line": 50, "function": "HandleGetEntities", "caller": null},
         {"file": "service.go", "line": 100, "function": "GetEntities", "caller": "HandleGetEntities"}
       ]
     }
     ```
   - Launch Task tool with:
     - `subagent_type: "comment-out-non-chain-code"`
     - `prompt: "Process this chain and comment out non-essential code:\n\n{JSON input}"`

3. **Wait for all subagents to complete**
   - Each subagent will:
     - Comment out non-chain code
     - Add dummy values as needed
     - Ensure compilation succeeds
   - No output from subagents (they complete silently on success)

4. **Display Phase 1 summary**
   ```
   === Phase 1: コメントアウト完了 ===
   処理済みチェーン: N個 (並列実行)

   Phase 2でPre-insertを追加します
   ```

### Phase 2: Add Pre-Insert Code

Add test data insertion before Update/Read/Delete SDK operations.

1. **Analyze SDK operations for pre-insert requirements**

   For each chain in `.migration-chains.json`:

   a. **Determine operation type**
      - Create: PutItem, PutObject, SendEmail, Publish → No pre-insert
      - Update: UpdateItem, TransactWriteItems → Pre-insert required
      - Read: Query, GetItem, Scan, GetObject → Pre-insert required
      - Delete: DeleteItem, DeleteObject → Pre-insert required

   b. **Extract pre-insert parameters** (for Update/Read/Delete only)
      - Load SDK function source from `sdk_operations[].file:line`
      - Extract: table name, key fields, bucket name
      - Generate pre-insert code

2. **Apply pre-insert code** (Edit tool)

   For each operation requiring pre-insert:
   - Insert pre-insert code before SDK call
   - Use proper indentation matching surrounding code
   - Add required imports if missing

   Example for DynamoDB:
   ```go
   // Pre-insert test data for DynamoDB Query
   preInsertInput := &dynamodb.PutItemInput{
       TableName: aws.String("entities"),
       Item: map[string]types.AttributeValue{
           "id":   &types.AttributeValueMemberS{Value: "test-id-001"},
           "date": &types.AttributeValueMemberS{Value: "20250101"},
       },
   }
   _, err := db.PutItem(ctx, preInsertInput)
   if err != nil {
       log.Printf("Pre-insert failed: %v", err)
       return err
   }
   ```

3. **Display Phase 2 summary**
   ```
   === Phase 2: Pre-insert追加完了 ===
   Pre-insert追加: P個

   Phase 3でSDK操作ログを追加します
   ```

### Phase 3: Add SDK Operation Logs

Add logging after SDK operations for runtime verification.

1. **Identify SDK operation log positions**

   For each SDK operation in `.migration-chains.json`:
   - Locate operation call and error check
   - Identify insertion point: after error check, before response processing
   - Determine log message based on operation type:
     - Query/Scan: `log.Printf("[SDK Test] %s succeeded: %%d items", operation, len(resp.Items))`
     - GetItem: `log.Printf("[SDK Test] %s succeeded: item found=%%v", operation, resp.Item != nil)`
     - PutItem/UpdateItem: `log.Printf("[SDK Test] %s succeeded", operation)`
     - DeleteItem: `log.Printf("[SDK Test] %s succeeded", operation)`
     - TransactWriteItems: `log.Printf("[SDK Test] %s succeeded: %%d items", operation, len(input.TransactItems))`
     - GetObject (S3): `log.Printf("[SDK Test] %s succeeded: size=%%d", operation, resp.ContentLength)`
     - PutObject (S3): `log.Printf("[SDK Test] %s succeeded", operation)`
     - SendEmail (SES): `log.Printf("[SDK Test] %s succeeded: MessageId=%%s", operation, *resp.MessageId)`
     - Publish (SNS): `log.Printf("[SDK Test] %s succeeded: MessageId=%%s", operation, *resp.MessageId)`

2. **Apply SDK operation logs** (Edit tool)

   For each SDK operation:
   - Insert log statement after error check
   - Add `log` package import if missing

   Example for DynamoDB Query:
   ```go
   // After
   result, err := s.db.Query(ctx, &dynamodb.QueryInput{...})
   if err != nil {
       return nil, err
   }
   log.Printf("[SDK Test] Query succeeded: %d items returned", len(result.Items))
   // // Response processing
   // entities := parseEntities(result.Items)
   ```

3. **Display Phase 3 summary**
   ```
   === Phase 3: SDK操作ログ追加完了 ===
   SDK操作ログ追加: L個

   Phase 4で検証します
   ```

### Phase 4: Verification (Compilation and Coverage)

Verify compilation and SDK operation coverage.

1. **Compile modified code** (Bash tool)
   - Run: `go build -o /tmp/test-build ./...`
   - If fails:
     - Analyze error type: unused vars, undefined refs, type mismatches, missing imports
     - Apply fixes using Edit tool
     - Retry compilation
     - Repeat until success
   - Display: "コンパイル成功"

2. **Verify pre-insert completeness** (Grep tool)
   - Load Update/Read/Delete operations from `.migration-chains.json`
   - For each operation:
     - Grep for operation in SDK function file with `-B: 20`
     - Check preceding 20 lines for pre-insert comment pattern: `// Pre-insert test data`
     - If missing: ERROR with list of missing pre-inserts

3. **Verify execution flow and SDK operation coverage** (Read tool, static analysis)

   For each chain in `.migration-chains.json`:

   a. **Analyze entry point function execution flow**
      - Read entry point function from `call_chain[0]` (file:line)
      - Track all SDK operation calls in the function body
      - For each SDK operation in chain's `sdk_operations`:
        - Check if operation is directly called in entry point
        - If not direct, trace through function calls in `call_chain`
        - Identify path from entry point to SDK operation

   b. **Detect return value flow issues**
      - For functions with commented response processing:
        - Identify return statement and its value type
        - Check if caller uses return value in control flow:
          - `for _, x := range returnValue` patterns
          - `switch returnValue.Field` patterns
          - `if returnValue != nil/empty` patterns
        - If empty value causes path to skip:
          - Record issue: function name, line, affected SDK operations
          - Determine fix: dummy data structure needed

   c. **Verify SDK operation coverage**
      - Count reachable SDK operations from entry point
      - Compare with total operations in chain's `sdk_operations` array
      - If mismatch:
        - Display warning:
          ```
          [WARNING] 実行フロー問題検出 (Chain N)
          関数: GetEntitiesByStatusAndDate (file.go:100)
          問題: 戻り値が空のため、後続の3個のSDK操作が実行されません
          - 影響を受けるSDK操作: [UpdateItem, PutItem, TransactWriteItems]
          ```
        - Prepare auto-fix: dummy data to enable path execution

   d. **Apply execution flow fixes** (Edit tool)
      - For each detected flow issue:
        - Determine appropriate dummy data:
          - For slice: `[]Type{{Field: value}}` with values satisfying downstream conditions
          - For struct: `Type{Field: value}` matching switch/if conditions
          - Analyze downstream switch/if to pick correct values
        - Apply Edit: Replace empty return with populated dummy
        - Display: "実行フロー修正: [function] にダミーデータ追加"

   e. **Re-verify after fixes**
      - If fixes applied:
        - Re-run compilation (step 1)
        - Re-check SDK operation coverage
        - Repeat until all operations reachable (max 3 iterations)
        - Display: "[OK] 実行フロー修正完了: SDK操作カバレッジ X/X (100%)"

   f. **Display SDK coverage summary**
      ```
      === SDK操作カバレッジ ===
      Chain 1: 4/4 操作が到達可能 [OK]
      Chain 2: 2/2 操作が到達可能 [OK]
      Chain 3: 1/1 操作が到達可能 [OK]
      ```

      Or if issues detected before fixes:
      ```
      === SDK操作カバレッジ ===
      Chain 1: 1/4 操作が到達可能 [WARNING]
        到達不可能:
        - UpdateItem (counter.go:60) - 原因: ループがスキップ
        - PutItem (dir_file.go:254) - 原因: ループがスキップ
        - TransactWriteItems (datastore.go:519) - 原因: ループがスキップ

      修正中...
      ```

4. **Display Phase 4 summary**
   ```
   === Phase 4: 検証完了 ===
   - コンパイル: 成功
   - Pre-insert: すべて生成済み (P operations)
   - 実行フロー: 修正完了 (M chains with fixes)
   - SDK操作カバレッジ: X/X 操作が到達可能 (100%)

   Phase 5で検証手順書を生成します
   ```

### Phase 5: Verification Document Generation

Generate execution procedures for testing in AWS environment.

1. **Load chain data from `.migration-chains.json`** (Read tool)

2. **For each chain, generate execution command**:

   **API endpoints (use endpoint object from chain):**
   - Extract method, path from chain.endpoint
   - Generate curl command:
   ```bash
   curl -X [endpoint.method] https://[host][endpoint.path] \
     -H "Content-Type: application/json" \
     -H "Authorization: Bearer $TOKEN" \
     -d '{"key":"value"}'
   ```

   **ECS Tasks:**
   ```bash
   aws ecs run-task \
     --cluster [cluster-name] \
     --task-definition [task-name]:latest \
     --launch-type FARGATE \
     --network-configuration "awsvpcConfiguration={subnets=[subnet-xxx],securityGroups=[sg-xxx],assignPublicIp=ENABLED}"
   ```

   **CLI commands:**
   ```bash
   ./bin/[command] [args]
   ```

3. **Generate X-Ray verification points**

   For each chain's SDK operations:
   - Extract service, operation, and resource from `sdk_operations` array
   - List expected operations with counts:
     - Query/Scan: Include item count
     - GetItem: Include found status
     - PutItem/UpdateItem/DeleteItem: Basic success
     - TransactWriteItems: Include transaction count
     - S3 operations: Include size/object info
     - SES/SNS: Include MessageId

4. **Write `aws-verification-procedures.md`** (Write tool)

   Document structure:
   ```markdown
   # AWS SDK v2 Migration Verification Procedures

   ## Summary
   - Total chains: N
   - Total SDK operations: M
   - API endpoints: A
   - ECS tasks: B
   - CLI commands: C

   ## Verification Steps
   1. For each chain below:
      - Execute command
      - Check X-Ray traces in AWS Console
      - Verify expected SDK operations appear
      - Confirm no errors in CloudWatch Logs
   2. Document results

   ## Chain 1: [type] [identifier]

   ### コールチェーン
   Entry: [type] [identifier]
   → [file:line] EntryFunc
   → [file:line] IntermediateFunc
   → [file:line] SDKFunc ← [Service] [Operation]

   ### 実行コマンド
   ```bash
   [execution command]
   ```

   ### X-Ray確認ポイント
   - [Service] [Operation] × N回 ([resource])
   ```

   For chains with multiple SDK operations:
   ```markdown
   ## Chain N: [type] [identifier] [★ Multiple SDK: M operations]

   ### コールチェーン
   Entry: [type] [identifier]
   → [file:line] EntryFunc
   → [file:line] IntermediateFunc

   SDK Functions:
   A. [file:line] SDKFunc1 ← [Service] [Op1]
   B. [file:line] SDKFunc2 ← [Service] [Op2]

   ### 実行コマンド
   ```bash
   [execution command]
   ```

   ### X-Ray確認ポイント
   - [Service] [Op1] × N回
   - [Service] [Op2] × M回
   - Data flow: Op1 → Op2
   ```

5. **Display Phase 5 summary**
   ```
   === Phase 5: 検証手順書生成完了 ===
   出力ファイル: aws-verification-procedures.md
   - 総チェーン数: N個
   - 総SDK操作数: M個
   - API: A個, Task: B個, CLI: C個
   ```

6. **Display final completion**
   ```
   === 全Phase完了 ===
   Phase 1: コメントアウト (並列実行) - N chains
   Phase 2: Pre-insert追加 - P operations
   Phase 3: SDK操作ログ追加 - L logs
   Phase 4: 検証 - コンパイル成功, SDK操作カバレッジ 100%
   Phase 5: 検証手順書生成 - aws-verification-procedures.md

   Next: See "Next Steps" section below
   ```

## Error Handling

**".migration-chains.json not found"**
- Solution: Run `/extract-sdk-chains` command first

**Phase 1: Subagent fails**
- Cause: comment-out-non-chain-code subagent encounters errors
- Solution: Check subagent error output; may need to fix file paths in `.migration-chains.json` or resolve compilation issues manually

**Phase 2: Pre-insert generation fails**
- Cause: Unable to extract table name or key fields from SDK operation
- Solution: Manually inspect SDK operation code; verify Input struct contains required fields

**Phase 3: SDK log insertion fails**
- Cause: Unable to locate SDK operation call or error check
- Solution: Verify SDK operation exists at file:line specified in `.migration-chains.json`

**Phase 4: Compilation fails**
- Cause: Unused vars, undefined refs, type mismatches, missing imports
- Solution: Command auto-fixes; applies Edit and retries compilation

**Phase 4: Pre-insert verification failed**
- Cause: Update/Read/Delete operation without pre-insert code
- Solution: Check Phase 2 operation type classification and application logs

**Phase 4: Execution flow verification failed**
- Cause: SDK operations unreachable due to empty return values
- Solution: Phase 4 step 3d auto-fixes by adding dummy data; re-verification runs automatically (max 3 iterations)

**Phase 4: SDK operation coverage incomplete**
- Cause: Not all SDK operations in chain are reachable from entry point
- Solution: Phase 4 step 3 detects and fixes automatically; check modified functions for correct dummy data structure

## Example

### Example 1: Functions NOT in call_chain (commented out)

Input (.migration-chains.json):
```json
{
  "chains": [{
    "call_chain": [
      {"file": "internal/service/entity.go", "line": 50, "function": "GetEntities", "caller": null}
    ],
    "sdk_operations": [{
      "file": "internal/service/entity.go",
      "line": 92,
      "operation": "Query",
      "type": "Read"
    }]
  }]
}
```

Note: `GetUser` and `GetBusinessDate` are NOT in `call_chain`, so they will be commented out.

Before:
```go
func (s *Service) GetEntities(ctx context.Context) ([]Entity, error) {
    // Business logic - these functions NOT in call_chain
    user := s.userRepo.GetUser(ctx)
    date := s.dateService.GetBusinessDate()

    // SDK operation
    result, err := s.db.Query(ctx, &dynamodb.QueryInput{
        TableName: aws.String("entities"),
        KeyConditionExpression: aws.String("date = :date"),
        ExpressionAttributeValues: map[string]types.AttributeValue{
            ":date": &types.AttributeValueMemberS{Value: date},
        },
    })
    if err != nil {
        return nil, err
    }

    // Response processing
    entities := parseEntities(result.Items)
    return enrichEntities(entities, user), nil
}
```

After:
```go
func (s *Service) GetEntities(ctx context.Context) ([]Entity, error) {
    // // Business logic - these functions NOT in call_chain
    // user := s.userRepo.GetUser(ctx)
    // date := s.dateService.GetBusinessDate()
    // Dummy values needed since functions are commented
    user := User{ID: "test-user"}
    date := "20250101"

    // Pre-insert test data for DynamoDB Query
    preInsertInput := &dynamodb.PutItemInput{
        TableName: aws.String("entities"),
        Item: map[string]types.AttributeValue{
            "date": &types.AttributeValueMemberS{Value: "20250101"},
            "id":   &types.AttributeValueMemberS{Value: "test-001"},
        },
    }
    _, _ = s.db.PutItem(ctx, preInsertInput)

    // SDK operation
    result, err := s.db.Query(ctx, &dynamodb.QueryInput{
        TableName: aws.String("entities"),
        KeyConditionExpression: aws.String("date = :date"),
        ExpressionAttributeValues: map[string]types.AttributeValue{
            ":date": &types.AttributeValueMemberS{Value: date},
        },
    })
    if err != nil {
        return nil, err
    }
    log.Printf("[SDK Test] Query succeeded: %d items returned", len(result.Items))

    // // Response processing
    // entities := parseEntities(result.Items)
    // return enrichEntities(entities, user), nil
    return []Entity{}, nil  // Dummy
}
```

### Example 2: Functions IN call_chain (kept and executed)

Input (.migration-chains.json):
```json
{
  "chains": [{
    "call_chain": [
      {"file": "internal/service/entity.go", "line": 50, "function": "GetEntities", "caller": null},
      {"file": "internal/repository/user.go", "line": 20, "function": "GetUser", "caller": "GetEntities"},
      {"file": "internal/service/date.go", "line": 10, "function": "GetBusinessDate", "caller": "GetEntities"}
    ],
    "sdk_operations": [{
      "file": "internal/service/entity.go",
      "line": 92,
      "operation": "Query",
      "type": "Read"
    }]
  }]
}
```

Note: `GetUser` and `GetBusinessDate` ARE in `call_chain`, so they will execute normally.

Before:
```go
func (s *Service) GetEntities(ctx context.Context) ([]Entity, error) {
    // Business logic - these functions IN call_chain
    user := s.userRepo.GetUser(ctx)
    date := s.dateService.GetBusinessDate()

    // SDK operation
    result, err := s.db.Query(ctx, &dynamodb.QueryInput{
        TableName: aws.String("entities"),
        KeyConditionExpression: aws.String("date = :date"),
        ExpressionAttributeValues: map[string]types.AttributeValue{
            ":date": &types.AttributeValueMemberS{Value: date},
        },
    })
    if err != nil {
        return nil, err
    }

    // Response processing
    entities := parseEntities(result.Items)
    return enrichEntities(entities, user), nil
}
```

After:
```go
func (s *Service) GetEntities(ctx context.Context) ([]Entity, error) {
    // Business logic - these functions IN call_chain → KEPT
    user := s.userRepo.GetUser(ctx)  // Executes normally, returns actual value
    date := s.dateService.GetBusinessDate()  // Executes normally, returns actual value
    // No dummy values needed!

    // Pre-insert test data for DynamoDB Query
    preInsertInput := &dynamodb.PutItemInput{
        TableName: aws.String("entities"),
        Item: map[string]types.AttributeValue{
            "date": &types.AttributeValueMemberS{Value: date},  // Uses actual date
            "id":   &types.AttributeValueMemberS{Value: "test-001"},
        },
    }
    _, _ = s.db.PutItem(ctx, preInsertInput)

    // SDK operation
    result, err := s.db.Query(ctx, &dynamodb.QueryInput{
        TableName: aws.String("entities"),
        KeyConditionExpression: aws.String("date = :date"),
        ExpressionAttributeValues: map[string]types.AttributeValue{
            ":date": &types.AttributeValueMemberS{Value: date},  // Uses actual date
        },
    })
    if err != nil {
        return nil, err
    }
    log.Printf("[SDK Test] Query succeeded: %d items returned", len(result.Items))

    // // Response processing
    // entities := parseEntities(result.Items)
    // return enrichEntities(entities, user), nil
    return []Entity{}, nil  // Dummy
}
```

### Example: Execution Flow Issue (Multiple SDK Operations)

This example demonstrates the issue reported in #96 where empty return values prevent subsequent SDK operations from being executed.

Input (.migration-chains.json):
```json
{
  "chains": [{
    "call_chain": [
      {"file": "handler.go", "line": 50, "function": "ProcessEntities", "caller": null},
      {"file": "service.go", "line": 100, "function": "GetEntitiesByStatus", "caller": "ProcessEntities"}
    ],
    "sdk_operations": [
      {"file": "service.go", "line": 110, "operation": "Query", "type": "Read"},
      {"file": "handler.go", "line": 60, "operation": "UpdateItem", "type": "Update"},
      {"file": "handler.go", "line": 65, "operation": "PutItem", "type": "Create"}
    ]
  }]
}
```

Before Phase 3 step 4 (compilation succeeds but flow broken):
```go
// service.go
func (s *Service) GetEntitiesByStatus(ctx context.Context) ([]Entity, error) {
    result, err := s.db.Query(ctx, &dynamodb.QueryInput{...})
    if err != nil {
        return nil, err
    }
    log.Printf("[SDK Test] Query succeeded: %d items returned", len(result.Items))
    // // Response processing commented out
    // for _, item := range result.Items {
    //     entity := parseEntity(item)
    //     entities = append(entities, entity)
    // }
    return []Entity{}, nil  // Empty slice returned
}

// handler.go
func (h *Handler) ProcessEntities(ctx context.Context) error {
    entities, err := h.svc.GetEntitiesByStatus(ctx)
    if err != nil {
        return err
    }
    // This loop never executes due to empty slice
    for _, entity := range entities {
        // UpdateItem never reached
        _, err := h.db.UpdateItem(ctx, &dynamodb.UpdateItemInput{...})
        if err != nil {
            return err
        }
        log.Printf("[SDK Test] UpdateItem succeeded")
        // PutItem never reached
        _, err = h.db.PutItem(ctx, &dynamodb.PutItemInput{...})
        if err != nil {
            return err
        }
        log.Printf("[SDK Test] PutItem succeeded")
    }
    return nil
}
```

Phase 3 step 4 detects issue:
```
=== SDK操作カバレッジ ===
Chain 1: 1/3 操作が到達可能 [WARNING]
  到達不可能:
  - UpdateItem (handler.go:60) - 原因: ループがスキップ
  - PutItem (handler.go:65) - 原因: ループがスキップ

修正中...
```

After Phase 3 step 4d auto-fix:
```go
// service.go - Phase 3 step 4d adds dummy data
func (s *Service) GetEntitiesByStatus(ctx context.Context) ([]Entity, error) {
    result, err := s.db.Query(ctx, &dynamodb.QueryInput{...})
    if err != nil {
        return nil, err
    }
    log.Printf("[SDK Test] Query succeeded: %d items returned", len(result.Items))
    // // Response processing commented out
    // for _, item := range result.Items {
    //     entity := parseEntity(item)
    //     entities = append(entities, entity)
    // }
    // Dummy data added to enable downstream execution
    return []Entity{{ID: "test-id", Status: "active"}}, nil
}

// handler.go - now loop executes, all SDK operations reachable
func (h *Handler) ProcessEntities(ctx context.Context) error {
    entities, err := h.svc.GetEntitiesByStatus(ctx)
    if err != nil {
        return err
    }
    // Loop now executes with dummy data
    for _, entity := range entities {
        // UpdateItem now reachable
        _, err := h.db.UpdateItem(ctx, &dynamodb.UpdateItemInput{...})
        if err != nil {
            return err
        }
        log.Printf("[SDK Test] UpdateItem succeeded")
        // PutItem now reachable
        _, err = h.db.PutItem(ctx, &dynamodb.PutItemInput{...})
        if err != nil {
            return err
        }
        log.Printf("[SDK Test] PutItem succeeded")
    }
    return nil
}
```

Phase 3 step 4f confirms fix:
```
[OK] 実行フロー修正完了: SDK操作カバレッジ 3/3 (100%)

=== SDK操作カバレッジ ===
Chain 1: 3/3 操作が到達可能 [OK]
```

## Key Advantages

**`call_chain` criteria benefits:**
- Clear, mechanical decision: Check if function is in `.migration-chains.json` call_chain array
- Consistent behavior: No ambiguous "business logic" judgments
- Reduced dummy values: Functions in call_chain execute and return actual values
- Better test realism: SDK inputs use actual values instead of dummies
- Traceability: Easy to understand why code was kept or commented

**Parallel subagent execution:**
- Phase 1: All chains processed simultaneously via comment-out-non-chain-code subagent
- Dramatic speed improvement: 10 chains processed in parallel vs sequential
- Independent compilation: Each subagent ensures its chain compiles
- Fault isolation: One chain's failure doesn't block others

**Optimized processing flow:**
- Phase 1: Parallel comment-out (N subagents running simultaneously)
- Phase 2: Pre-insert addition (sequential, depends on Phase 1 completion)
- Phase 3: SDK log addition (sequential)
- Phase 4: Verification (single compilation check)
- Phase 5: Document generation

**Performance improvements:**
- Parallel execution: N chains processed simultaneously
- Modular subagents: Reusable comment-out logic
- Example: 10 chains with 3 functions each
  - Sequential: ~10 minutes (1 min per chain)
  - Parallel: ~2 minutes (all chains simultaneously + final verification)

**Execution flow verification (Phase 4 step 3):**
- Complete SDK operation testing: All SDK operations in chain are reachable and testable
- Automatic correction: Flow issues detected and fixed without manual intervention
- Improved verification reliability: Ensures not just compilation but actual execution paths
- Reduced debugging time: Prevents discovering untested SDK operations after deployment
- Coverage guarantee: All operations in `.migration-chains.json` verified as reachable

**SDK operation logging (Phase 3):**
- Runtime verification: Confirms SDK operations actually execute and return responses
- Operation-specific output: Logs relevant data (item counts, sizes, message IDs) for each operation type
- Easy debugging: Clear `[SDK Test]` prefix identifies test-related logs in application output
- No manual intervention: Logs automatically added after all SDK operations
- Production-ready format: Uses standard `log.Printf` compatible with existing logging infrastructure

1. **Load chain data from `.migration-chains.json`** (Read tool)

2. **For each chain, generate execution command**:

   **API endpoints:**
   ```bash
   curl -X [METHOD] https://[host]/[path] \
     -H "Content-Type: application/json" \
     -H "Authorization: Bearer $TOKEN" \
     -d '{"key":"value"}'
   ```

   **ECS Tasks:**
   ```bash
   aws ecs run-task \
     --cluster [cluster-name] \
     --task-definition [task-name]:latest \
     --launch-type FARGATE \
     --network-configuration "awsvpcConfiguration={subnets=[subnet-xxx],securityGroups=[sg-xxx],assignPublicIp=ENABLED}"
   ```

   **CLI commands:**
   ```bash
   ./bin/[command] [args]
   ```

3. **Generate X-Ray verification points**

   For each chain's SDK operations:
   - Extract service, operation, and resource from `sdk_operations` array
   - List expected operations with counts:
     - Query/Scan: Include item count
     - GetItem: Include found status
     - PutItem/UpdateItem/DeleteItem: Basic success
     - TransactWriteItems: Include transaction count
     - S3 operations: Include size/object info
     - SES/SNS: Include MessageId

4. **Write `aws-verification-procedures.md`** (Write tool)

   Document structure:
   ```markdown
   # AWS SDK v2 Migration Verification Procedures

   ## Summary
   - Total chains: N
   - Total SDK operations: M
   - API endpoints: A
   - ECS tasks: B
   - CLI commands: C

   ## Verification Steps
   1. For each chain below:
      - Execute command
      - Check X-Ray traces in AWS Console
      - Verify expected SDK operations appear
      - Confirm no errors in CloudWatch Logs
   2. Document results

   ## Chain 1: [type] [identifier]

   ### コールチェーン
   Entry: [type] [identifier]
   → [file:line] EntryFunc
   → [file:line] IntermediateFunc
   → [file:line] SDKFunc ← [Service] [Operation]

   ### 実行コマンド
   ```bash
   [execution command]
   ```

   ### X-Ray確認ポイント
   - [Service] [Operation] × N回 ([resource])
   ```

   For chains with multiple SDK operations:
   ```markdown
   ## Chain N: [type] [identifier] [★ Multiple SDK: M operations]

   ### コールチェーン
   Entry: [type] [identifier]
   → [file:line] EntryFunc
   → [file:line] IntermediateFunc

   SDK Functions:
   A. [file:line] SDKFunc1 ← [Service] [Op1]
   B. [file:line] SDKFunc2 ← [Service] [Op2]

   ### 実行コマンド
   ```bash
   [execution command]
   ```

   ### X-Ray確認ポイント
   - [Service] [Op1] × N回
   - [Service] [Op2] × M回
   - Data flow: Op1 → Op2
   ```

5. **Display Phase 4 summary**
   ```
   === Phase 4: 検証手順書生成完了 ===
   出力ファイル: aws-verification-procedures.md
   - 総チェーン数: N個
   - 総SDK操作数: M個
   - API: A個, Task: B個, CLI: C個
   ```

6. **Display final completion**
   ```
   === 全Phase完了 ===
   Phase 1: 分析 - N chains
   Phase 2: 変更適用 - X modifications
   Phase 3: 検証 - コンパイル成功, SDK操作カバレッジ 100%
   Phase 4: 検証手順書生成 - aws-verification-procedures.md

   Next: See "Next Steps" section below
   ```

## Next Steps

1. Review: `git diff`
2. Verify: `go build` succeeds
3. Review: `aws-verification-procedures.md`
4. Deploy to AWS test environment
5. Execute verification procedures
6. After testing: `git checkout .` to revert changes
