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

### Phase 1: Analysis (One-pass for all chains)

Analyze all chains in `.migration-chains.json` in a single pass to collect modification requirements.

For each chain in `.migration-chains.json`:

1. **Read chain configuration**
   - Load chain ID, entry point, call chain, SDK operations from `.migration-chains.json`
   - Display: "=== Analyzing Chain N/M: [type] [identifier] ==="

2. **Analyze all functions in call chain** (use Read tool only, no modifications yet)

   For each function in `call_chain` array:

   a. **Read function source** (Read tool)
      - Load complete function from file:line
      - Analyze function content (Phase 2 will re-read if needed for Edit tool context)

   b. **Identify comment-out targets**

      Decision flow:
      1. Does line initialize SDK client? → KEEP
      2. Does line construct SDK input struct? → KEEP
      3. Does line assign data to SDK input fields? → KEEP
      4. Is line ctx/basic error check? → KEEP
      5. Is line a function call in current chain's `call_chain` array? → KEEP
      6. Otherwise (everything else) → COMMENT

      KEEP (SDK-related and call chain):
      - SDK client init: `dynamodb.New()`, `s3.NewFromConfig()`
      - SDK input construction: `&dynamodb.PutItemInput{...}`
      - Data for SDK input fields
      - Context, minimal error check
      - **Function calls in `call_chain` array from `.migration-chains.json`**

      COMMENT (everything else):
      - **Function calls NOT in `call_chain` array**
      - Everything else not explicitly kept

      Note: `call_chain` functions are kept to provide actual values for SDK inputs, reducing dummy value requirements.

      Record: List of code blocks to comment out (file:line, old_string)

   c. **Identify dummy value requirements**

      Note: With `call_chain` criteria, dummy values are rarely needed since functions in `call_chain` execute and return actual values.

      Dummy values are only needed for:
      - **Non-function initializations**: `time.Now()`, `uuid.New()`, arithmetic operations
      - **Call chain external values**: Values from functions NOT in `call_chain` but used in SDK inputs

      If commented code returns values used in SDK inputs:
      - Strings: `"test-value"`
      - Integers: `1`
      - UUIDs: `uuid.MustParse("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee")`

      Record: List of dummy values to add (file:line, type, new_string, reason)

   d. **Analyze execution flow**
      - Identify conditional branches (if/switch/for)
      - Check each condition works with dummy/test data
      - Identify error handling that may trigger prematurely
      - Check for uninitialized variables
      - Verify type compatibility with dummy values

      Record: List of execution flow issues (file:line, issue_type, fix_action)

3. **Analyze SDK operations for pre-insert requirements**

   For each SDK operation in chain:

   a. **Determine operation type**
      - Create: PutItem, PutObject, SendEmail, Publish → No pre-insert
      - Update: UpdateItem, TransactWriteItems → Pre-insert required
      - Read: Query, GetItem, Scan, GetObject → Pre-insert required
      - Delete: DeleteItem, DeleteObject → Pre-insert required

   b. **Extract pre-insert parameters** (for Update/Read/Delete only)
      - Load SDK function source (re-read with Read tool if not already loaded in step 2a)
      - Extract: table name, key fields, bucket name
      - Generate pre-insert code structure

      Record: Pre-insert code to add (file:line, pre_insert_code)

   c. **Identify SDK operation log positions**
      - For each SDK operation, locate the operation call and error check
      - Identify insertion point: after error check, before response processing
      - Determine appropriate log message based on operation type:
        - Query/Scan: `log.Printf("[SDK Test] %s succeeded: %%d items", operation, len(resp.Items))`
        - GetItem: `log.Printf("[SDK Test] %s succeeded: item found=%%v", operation, resp.Item != nil)`
        - PutItem/UpdateItem: `log.Printf("[SDK Test] %s succeeded", operation)`
        - DeleteItem: `log.Printf("[SDK Test] %s succeeded", operation)`
        - TransactWriteItems: `log.Printf("[SDK Test] %s succeeded: %%d items", operation, len(input.TransactItems))`
        - GetObject (S3): `log.Printf("[SDK Test] %s succeeded: size=%%d", operation, resp.ContentLength)`
        - PutObject (S3): `log.Printf("[SDK Test] %s succeeded", operation)`
        - SendEmail (SES): `log.Printf("[SDK Test] %s succeeded: MessageId=%%s", operation, *resp.MessageId)`
        - Publish (SNS): `log.Printf("[SDK Test] %s succeeded: MessageId=%%s", operation, *resp.MessageId)`

      Record: Log insertion points (file:line, operation, log_code, response_var)

4. **Display analysis summary**
   ```
   完了 (N/M): 分析
   Chain: [type] [identifier]
   - 関数数: X個
   - コメントアウト対象: Y個ブロック
   - ダミー値必要: Z個
   - 実行フロー問題: W個
   - Pre-insert必要: P個 (Update/Read/Delete operations)
   - SDK操作ログ追加: L個
   ```

5. **After analyzing all chains, display Phase 1 summary**
   ```
   === Phase 1: 分析完了 ===
   処理済みチェーン: N個
   コメントアウト対象: 合計Y個ブロック
   ダミー値必要: 合計Z個
   実行フロー問題: 合計W個
   Pre-insert生成: 合計P個
   SDK操作ログ追加: 合計L個

   Phase 2で一括適用します
   ```

### Phase 2: Application (Batch modifications for all chains)

Apply all modifications collected in Phase 1 in a single batch.

1. **Group modifications by file**
   - Merge all modifications (comment-outs, dummies, pre-inserts, flow fixes) targeting same file
   - Sort by line number (descending, highest line first) to avoid line offset issues
     - Reason: Modifying line 100 before line 50 prevents line 50's offset from changing

2. **Apply comment-out modifications** (Edit tool)

   For each recorded comment-out target from Phase 1:
   - Apply Edit: Replace old_string with `// ` prefixed version
   - Display progress: "Commenting out: [file:line]"

   Example:
   ```go
   // Old:
   result, err := repo.GetUser(ctx, id)

   // New:
   // result, err := repo.GetUser(ctx, id)
   ```

3. **Apply dummy value additions** (Edit tool)

   Note: With `call_chain` criteria, this step is rarely needed since functions in call_chain execute and return actual values.

   For each recorded dummy value requirement from Phase 1:
   - Apply Edit: Insert dummy value after commented code
   - Use type-appropriate values:
     - Strings: `"test-value"`
     - Integers: `1`
     - UUIDs: `uuid.MustParse("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee")`
   - Display progress: "Adding dummy: [file:line] ([type]) - [reason]"

4. **Apply execution flow fixes** (Edit tool)

   For each recorded execution flow issue from Phase 1:
   - Apply Edit based on issue_type:
     - Conditional branch: Adjust dummy values to pass conditions
     - Premature error: Comment out error check
     - Uninitialized variable: Add initialization
     - Type mismatch: Fix dummy value type
   - Display progress: "Fixing flow: [file:line] ([issue_type])"

5. **Apply pre-insert code additions** (Edit tool)

   For each recorded pre-insert requirement from Phase 1:
   - Apply Edit: Insert pre-insert code before SDK call
   - Use proper indentation matching surrounding code
   - Add required imports if missing
   - Display progress: "Adding pre-insert: [file:line] ([operation])"

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

6. **Apply SDK operation log additions** (Edit tool)

   For each recorded log insertion point from Phase 1:
   - Apply Edit: Insert log statement after SDK call and error check
   - Use proper indentation matching surrounding code
   - Add `log` package import if missing
   - Display progress: "Adding log: [file:line] ([operation])"

   Example for DynamoDB Query:
   ```go
   // Before
   result, err := s.db.Query(ctx, &dynamodb.QueryInput{...})
   if err != nil {
       return nil, err
   }
   // // Response processing
   // entities := parseEntities(result.Items)

   // After
   result, err := s.db.Query(ctx, &dynamodb.QueryInput{...})
   if err != nil {
       return nil, err
   }
   log.Printf("[SDK Test] Query succeeded: %d items returned", len(result.Items))
   // // Response processing
   // entities := parseEntities(result.Items)
   ```

   Example for S3 GetObject:
   ```go
   // Before
   resp, err := s.s3.GetObject(ctx, &s3.GetObjectInput{...})
   if err != nil {
       return nil, err
   }
   // // Process object
   // data := readObject(resp.Body)

   // After
   resp, err := s.s3.GetObject(ctx, &s3.GetObjectInput{...})
   if err != nil {
       return nil, err
   }
   log.Printf("[SDK Test] GetObject succeeded: size=%d bytes", *resp.ContentLength)
   // // Process object
   // data := readObject(resp.Body)
   ```

7. **Display Phase 2 summary**
   ```
   === Phase 2: 変更適用完了 ===
   コメントアウト適用: Y個ブロック
   ダミー値追加: Z個
   実行フロー修正: W個
   Pre-insert追加: P個
   SDK操作ログ追加: L個

   Phase 3で検証します
   ```

### Phase 3: Verification (Single compilation and completeness checks)

Verify all modifications in a single phase with one compilation check.

1. **Compile modified code** (Bash tool)
   - Run: `go build -o /tmp/test-build ./...`
   - If fails:
     - Analyze error type: unused vars, undefined refs, type mismatches, missing imports
     - Apply fixes using Edit tool
     - Retry compilation
     - Repeat until success
   - Display: "コンパイル成功"

2. **Verify comment-out completeness** (Grep tool)

   a. **Verify external service calls are commented**
      - Pattern: `pattern: "http\\.(Get|Post|Client)|grpc\\.(Dial|NewClient)"`
      - Options: `output_mode: "content"`, `-C: 5`, `glob: "!(*_test.go)"`
      - For each match in modified files (from `.migration-chains.json`):
        - Check if line starts with `//`
        - If uncommented in modified chain functions: ERROR with list of uncommented lines

   b. **Verify response processing is minimized**
      - Pattern: `pattern: "parseAttributes|ToEntity|for.*resp\\.(Items|Records)"`
      - Options: `output_mode: "content"`, `glob: "!(*_test.go)"`
      - For each match in modified files:
        - Check if commented or replaced with simple log
        - If complex processing remains in modified chain functions: ERROR with list of remaining processing

3. **Verify pre-insert completeness** (Grep tool)
   - Load Update/Read/Delete operations from `.migration-chains.json`
   - For each operation:
     - Grep for operation in SDK function file with `-B: 20`
     - Check preceding 20 lines for pre-insert comment pattern: `// Pre-insert test data`
     - If missing: ERROR with list of missing pre-inserts

4. **Verify execution flow and SDK operation coverage** (Read tool, static analysis)

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

5. **Display Phase 3 summary**
   ```
   === Phase 3: 検証完了 ===
   - コンパイル: 成功
   - 外部サービス呼び出し: すべてコメント済み
   - レスポンス処理: 最小化済み
   - Pre-insert: すべて生成済み (N operations)
   - 実行フロー: 修正完了 (M chains with fixes)
   - SDK操作カバレッジ: X/X 操作が到達可能 (100%)
   ```

6. **Display final completion**
   ```
   === Phase 1-3 完了 ===
   処理済みチェーン: N個
   コメントアウトブロック: X個
   Pre-insert生成: Y個
   SDK操作ログ追加: L個
   実行フロー修正: W個
   SDK操作カバレッジ: Z/Z 操作 (100%)
   最終コンパイル: 成功

   Next: See "Next Steps" section below
   ```

## Error Handling

**".migration-chains.json not found"**
- Solution: Run `/extract-sdk-chains` command first

**Phase 1 analysis errors**
- Cause: Unable to read function source from file:line
- Solution: Verify `.migration-chains.json` contains correct file paths and line numbers

**Phase 2 application errors**
- Cause: Edit tool fails to find old_string
- Solution: Check Phase 1 recorded correct code blocks; file may have changed since Phase 1

**Phase 3 compilation fails**
- Cause: Unused vars, undefined refs, type mismatches, missing imports
- Solution: Command auto-fixes during Phase 3 step 1; applies Edit and retries compilation

**Phase 3 comment-out verification failed**
- Cause: HTTP/gRPC calls remain uncommented in modified functions
- Solution: Review Phase 1 analysis; may need to adjust KEEP/COMMENT classification logic

**Phase 3 response processing verification failed**
- Cause: Complex response parsing (parseAttributes, ToEntity) remains
- Solution: Review Phase 1 analysis; response processing should be marked as COMMENT

**Phase 3 pre-insert verification failed**
- Cause: Update/Read/Delete operation without pre-insert code
- Solution: Check Phase 1 operation type classification and Phase 2 application logs

**Phase 3 execution flow verification failed**
- Cause: SDK operations unreachable due to empty return values
- Solution: Phase 3 step 4d auto-fixes by adding dummy data; re-verification runs automatically (max 3 iterations)

**Phase 3 SDK operation coverage incomplete**
- Cause: Not all SDK operations in chain are reachable from entry point
- Solution: Phase 3 step 4 detects and fixes automatically; check modified functions for correct dummy data structure

**Execution flow issues**
- Cause: Dummy values don't satisfy conditional branches
- Solution: Phase 1 identifies issues, Phase 2 applies fixes automatically; Phase 3 step 4 provides additional verification and auto-fix

## Example

### Example 1: Functions NOT in call_chain (commented out)

Input (.migration-chains.json):
```json
{
  "chains": [{
    "call_chain": [
      {"file": "internal/service/entity.go", "line": 50, "function": "GetEntities"}
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
      {"file": "internal/service/entity.go", "line": 50, "function": "GetEntities"},
      {"file": "internal/repository/user.go", "line": 20, "function": "GetUser"},
      {"file": "internal/service/date.go", "line": 10, "function": "GetBusinessDate"}
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
      {"file": "handler.go", "line": 50, "function": "ProcessEntities"},
      {"file": "service.go", "line": 100, "function": "GetEntitiesByStatus"}
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

**Optimized processing flow:**
- Phase 1: Analyze all chains in one pass (single file read per function)
- Phase 2: Apply all modifications in batch (grouped by file)
- Phase 3: Single compilation check (was N×3 compilations before)

**Performance improvements:**
- File I/O: 1 read per function (was 3× before)
- Compilation: 1 time total (was 3N times before, N = chain count)
- No sub-agent usage: Direct tool calls only (no Task tool)
- Example: 10 chains with 3 functions each
  - Before: 90 file reads, 30+ compilations
  - After: 30 file reads, 1 compilation

**Benefits:**
- Faster execution (seconds vs minutes per chain)
- No context accumulation
- Predictable behavior
- Clear separation: Analysis → Application → Verification

**Execution flow verification (Phase 3 step 4):**
- Complete SDK operation testing: All SDK operations in chain are reachable and testable
- Automatic correction: Flow issues detected and fixed without manual intervention
- Improved verification reliability: Ensures not just compilation but actual execution paths
- Reduced debugging time: Prevents discovering untested SDK operations after deployment
- Coverage guarantee: All operations in `.migration-chains.json` verified as reachable

**SDK operation logging (Phase 2 step 6):**
- Runtime verification: Confirms SDK operations actually execute and return responses
- Operation-specific output: Logs relevant data (item counts, sizes, message IDs) for each operation type
- Easy debugging: Clear `[SDK Test]` prefix identifies test-related logs in application output
- No manual intervention: Logs automatically added after all SDK operations during Phase 2
- Production-ready format: Uses standard `log.Printf` compatible with existing logging infrastructure

### Phase 4: Verification Document Generation

Generate execution procedures for testing in AWS environment.

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
