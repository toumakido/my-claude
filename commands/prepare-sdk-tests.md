# Prepare SDK Code for Testing

This command prepares AWS SDK v2 migrated code for connection testing by modifying code temporarily.

Output language: Japanese, formal business tone

**IMPORTANT**: This command modifies production code. All changes are reviewable via `git diff` and can be reverted with `git checkout .`.

## When to Use This Command

Use this command when:
- After running `/extract-sdk-chains` and approving chains
- Need to isolate SDK connection code from business logic
- Want to test AWS SDK v2 connections without external dependencies
- Before deploying to test environment for verification

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
      5. Otherwise → COMMENT

      KEEP (SDK-related):
      - SDK client init: `dynamodb.New()`, `s3.NewFromConfig()`
      - SDK input construction: `&dynamodb.PutItemInput{...}`
      - Data for SDK input fields
      - Context, minimal error check

      COMMENT (unrelated to connection):
      - Logging/metrics
      - External calls (HTTP, gRPC)
      - Business logic after SDK call
      - Response parsing: parseAttributes, loops over resp.Items
      - Entity transformation: ToEntity
      - Detailed error wrapping

      Record: List of code blocks to comment out (file:line, old_string)

   c. **Identify dummy value requirements**
      - If commented code returns values: Record type and insertion point
      - Strings: `"test-value"`
      - Integers: `1`
      - UUIDs: `uuid.MustParse("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee")`

      Record: List of dummy values to add (file:line, type, new_string)

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

4. **Display analysis summary**
   ```
   完了 (N/M): 分析
   Chain: [type] [identifier]
   - 関数数: X個
   - コメントアウト対象: Y個ブロック
   - ダミー値必要: Z個
   - 実行フロー問題: W個
   - Pre-insert必要: P個 (Update/Read/Delete operations)
   ```

5. **After analyzing all chains, display Phase 1 summary**
   ```
   === Phase 1: 分析完了 ===
   処理済みチェーン: N個
   コメントアウト対象: 合計Y個ブロック
   ダミー値必要: 合計Z個
   実行フロー問題: 合計W個
   Pre-insert生成: 合計P個

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

   For each recorded dummy value requirement from Phase 1:
   - Apply Edit: Insert dummy value after commented code
   - Use type-appropriate values:
     - Strings: `"test-value"`
     - Integers: `1`
     - UUIDs: `uuid.MustParse("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee")`
   - Display progress: "Adding dummy: [file:line] ([type])"

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

6. **Display Phase 2 summary**
   ```
   === Phase 2: 変更適用完了 ===
   コメントアウト適用: Y個ブロック
   ダミー値追加: Z個
   実行フロー修正: W個
   Pre-insert追加: P個

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

4. **Display Phase 3 summary**
   ```
   === Phase 3: 検証完了 ===
   - コンパイル: 成功
   - 外部サービス呼び出し: すべてコメント済み
   - レスポンス処理: 最小化済み
   - Pre-insert: すべて生成済み (N operations)
   ```

5. **Display final completion**
   ```
   === Phase 1-3 完了 ===
   処理済みチェーン: N個
   コメントアウトブロック: X個
   Pre-insert生成: Y個
   実行フロー修正: W個
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

**Execution flow issues**
- Cause: Dummy values don't satisfy conditional branches
- Solution: Phase 1 identifies issues, Phase 2 applies fixes automatically

## Example

Input (.migration-chains.json):
```json
{
  "chains": [{
    "sdk_operations": [{
      "file": "internal/service/entity.go",
      "line": 92,
      "operation": "Query",
      "type": "Read"
    }]
  }]
}
```

Before:
```go
func (s *Service) GetEntities(ctx context.Context) ([]Entity, error) {
    // Business logic
    user := s.userRepo.GetUser(ctx)
    date := s.dateService.GetBusinessDate()

    // SDK operation
    result, err := s.db.Query(ctx, &dynamodb.QueryInput{
        TableName: aws.String("entities"),
        KeyConditionExpression: aws.String("date = :date"),
    })

    // Response processing
    entities := parseEntities(result.Items)
    return enrichEntities(entities, user), nil
}
```

After:
```go
func (s *Service) GetEntities(ctx context.Context) ([]Entity, error) {
    // // Business logic
    // user := s.userRepo.GetUser(ctx)
    // date := s.dateService.GetBusinessDate()
    user := User{ID: "test-user"}  // Dummy
    date := "20250101"  // Dummy

    // pre-insert test data for DynamoDB Query
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
    })

    // // Response processing
    // entities := parseEntities(result.Items)
    // return enrichEntities(entities, user), nil
    return []Entity{}, nil  // Dummy
}
```

## Key Advantages

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

## Next Steps

1. Review: `git diff`
2. Verify: `go build` succeeds
3. Deploy to AWS test environment
4. Run `/generate-verification` command for verification procedures
5. After testing: `git checkout .` to revert changes
