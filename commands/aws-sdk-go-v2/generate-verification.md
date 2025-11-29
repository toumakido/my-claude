# Generate AWS Verification Procedures

Generates AWS environment verification procedures for testing SDK v2 migration connections.

## Prerequisites

- `.migration-chains.json` exists (created by `/extract-sdk-chains`)
- Code prepared by `/prepare-sdk-tests`
- Deployed to AWS test environment
- X-Ray tracing enabled in test environment

## Process

1. **Read `.migration-chains.json`** using Read tool

2. **For each chain, extract**:
   - Entry point type (API/Task/CLI)
   - Identifier (endpoint path, task name, command name)
   - SDK operations list with file:line references

3. **Generate execution command**:

   For API endpoints:
   ```bash
   curl -X [METHOD] https://[host]/[path] \
     -H "Content-Type: application/json" \
     -H "Authorization: Bearer $TOKEN" \
     -d '{"key":"value"}'
   ```

   For ECS Tasks:
   ```bash
   aws ecs run-task \
     --cluster [cluster-name] \
     --task-definition [task-name]:latest \
     --launch-type FARGATE \
     --network-configuration "awsvpcConfiguration={subnets=[subnet-xxx],securityGroups=[sg-xxx],assignPublicIp=ENABLED}"
   ```

   For CLI commands:
   ```bash
   ./bin/[command] [args]
   ```

4. **Generate X-Ray verification points** listing expected SDK operations with count and resource

5. **Write output** to `aws-verification-procedures.md` using Write tool

6. **Output structure**

   **Single SDK operation**:
   ```markdown
   ## Chain N: [type] [identifier]

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

   **Multiple SDK operations**:
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

   **Complete example**:
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

7. **Generate document header**:

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

   [Individual chain procedures from step 6]
   ```

## Output

File: `aws-verification-procedures.md`

Contents:
- Summary statistics (chains, operations, entry point types)
- Per-chain verification procedures with execution commands and X-Ray trace points

## Validation

After generating `aws-verification-procedures.md`, verify:
- [ ] Each chain has executable command
- [ ] X-Ray trace points match SDK operations in `.migration-chains.json`
- [ ] All SDK operations include service and operation name
- [ ] Entry point identifiers are specific (not generic)

## Error Handling

**".migration-chains.json not found"**
- Run `/extract-sdk-chains` first

**"Cannot generate command for entry type"**
- Entry point type not recognized (not API/Task/CLI)
- Check entry point format in `.migration-chains.json`

**X-Ray traces not appearing in test**
- X-Ray not enabled or daemon not running
- Verify IAM permissions for X-Ray

**Execution command fails in test**
- AWS resource names incorrect (cluster, task definition, endpoint)
- Verify resource names match test environment

## Example Output

```markdown
# AWS SDK v2 Migration Verification Procedures

## Summary
- Total chains: 4
- Total SDK operations: 7
- API endpoints: 2
- ECS tasks: 2
- CLI commands: 0

## Chain 1: Task batch_task [★ Multiple SDK: 4 operations]

### コールチェーン
Entry: Task batch_task
→ cmd/batch_task/main.go:136 main()
→ internal/tasks/worker.go:40 Execute()

SDK Functions:
A. internal/service/datastore.go:105 ← DynamoDB Query
B. internal/service/counter.go:60 ← DynamoDB UpdateItem
C. internal/service/storage.go:254 ← DynamoDB PutItem
D. internal/service/datastore.go:519 ← DynamoDB TransactWriteItems

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
- Data flow: Query → UpdateItem → PutItem → TransactWriteItems
```

## Next Steps

1. Execute commands in test environment
2. Check X-Ray traces in AWS Console
3. Document results
4. If tests pass: Revert temporary changes (`git checkout .`)
5. Merge to production
