# Prepare SDK Code for Testing

This command prepares AWS SDK v2 migrated code for connection testing by modifying code temporarily.

**IMPORTANT**: This command modifies production code. All changes are reviewable via `git diff` and can be reverted with `git checkout .`.

## Usage

```
/prepare-sdk-tests
```

Run this command from the repository root directory after running `/extract-sdk-chains`.

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

### Phase 1: Comment Out Unrelated Code

For each chain in `.migration-chains.json`:

1. **Read chain configuration**
   - Load chain ID, entry point, call chain, SDK operations from `.migration-chains.json`
   - Display: "=== Processing Chain N/M: [type] [identifier] ==="

2. **Analyze and comment out** (use Read/Edit/Bash tools directly, no Task tool)

   For each function in `call_chain` array:

   a. **Read function source** (Read tool)
      - Load complete function from file:line

   b. **Classify code blocks**

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

   c. **Apply comment-out** (Edit tool)
      - For each COMMENT block: Replace with `// ` prefixed version
      - Example:
        ```go
        // Old:
        result, err := repo.GetUser(ctx, id)

        // New:
        // result, err := repo.GetUser(ctx, id)
        ```

   d. **Add dummy values if needed** (Edit tool)
      - If commented code returns values: Add type-appropriate dummies
      - Strings: `"test-value"`
      - Integers: `1`
      - UUIDs: `uuid.MustParse("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee")`

3. **Verify compilation after all functions in chain** (Bash tool)
   - Run: `go build -o /tmp/test-build ./...`
   - If fails: Fix errors (unused vars, undefined refs) by adding dummies, retry until success

4. **Display progress**
   ```
   完了 (N/M): コメントアウト処理
   Chain: [type] [identifier]
   - 関数数: X個
   - コメントアウトブロック: Y個
   - コンパイル: 成功
   ```

### Phase 2: Generate Test Data

For each SDK operation in `.migration-chains.json`:

1. **Determine operation type**
   - Create: PutItem, PutObject, SendEmail, Publish → No pre-insert
   - Update: UpdateItem, TransactWriteItems → Pre-insert required
   - Read: Query, GetItem, Scan, GetObject → Pre-insert required
   - Delete: DeleteItem, DeleteObject → Pre-insert required

2. **Generate pre-insert code** (use Read/Edit/Bash tools directly, no Task tool)

   For Update/Read/Delete operations:

   a. **Read SDK function** (Read tool)
      - Load function source from `file` and `line` fields in SDK operation
      - Extract SDK call parameters: table name, key fields, bucket name

   b. **Generate pre-insert code**
      - Create 1-2 minimal test records
      - Use Go pointer types: `aws.String()`, `aws.Int64()`
      - Example for DynamoDB:
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

   c. **Insert pre-insert code** (Edit tool)
      - Find line before SDK call
      - Insert pre-insert code with proper indentation
      - Add required imports if missing

3. **Verify compilation after all operations** (Bash tool)
   - Run: `go build -o /tmp/test-build ./...`
   - If fails: Fix errors (missing imports, type mismatches), retry until success

4. **Display progress**
   ```
   完了 (N/M): テストデータ準備
   SDK function: [file:line] [function_name]
   - 操作種別: Create/Update/Read/Delete
   - Pre-insert: 追加済み / 不要
   - コンパイル: 成功
   ```

### Phase 3: Final Verification

1. **Verify all pre-inserts**
   - Grep for Update/Read/Delete operations in `.migration-chains.json`
   - For each operation: Check preceding 20 lines for pre-insert comment pattern
   - If any missing: ERROR and exit

2. **Display completion**
   ```
   === Phase 1-3 完了 ===
   処理済みチェーン: N個
   コメントアウトブロック: X個
   Pre-insert生成: Y個
   最終コンパイル: 成功

   Next: Review git diff, then deploy to test environment
   ```

## Output

Creates temporary modifications to production code:
- Comments out business logic unrelated to SDK connections
- Adds pre-insert test data for Update/Read/Delete operations
- Replaces commented values with type-appropriate dummies
- All changes visible via `git diff`

## Validation

After completion, verify:
- [ ] All chains processed without compilation errors
- [ ] `git diff` shows commented code and pre-insert additions
- [ ] `go build` succeeds for all modified packages
- [ ] Update/Read/Delete operations have pre-insert code
- [ ] Create operations have no pre-insert code

## Error Handling

**".migration-chains.json not found"**
- Solution: Run `/extract-sdk-chains` command first

**Compilation fails after comment-out**
- Cause: Commented code returns values used elsewhere
- Solution: Command auto-adds dummy values during Phase 1 step 3

**Compilation fails after pre-insert**
- Cause: Import missing or type mismatch
- Solution: Check pre-insert code uses correct types (`aws.String()`, etc.)

**"Phase 1-3 incomplete - pre-insert missing"**
- Cause: Update/Read/Delete operation without pre-insert
- Solution: Check operation type classification in `.migration-chains.json`

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

**No sub-agent usage:**
- Uses Read/Edit/Bash tools directly (no Task tool)
- Original `/verify-migration-connections`: 4 sub-agent calls per chain

**Benefits:**
- Faster execution (seconds vs minutes per chain)
- No context accumulation
- Predictable behavior

## Next Steps

1. Review: `git diff`
2. Verify: `go build` succeeds
3. Deploy to AWS test environment
4. Run `/generate-verification` command for verification procedures
5. After testing: `git checkout .` to revert changes
