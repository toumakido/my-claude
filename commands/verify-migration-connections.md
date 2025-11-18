Analyze AWS SDK migration PR and provide connection verification information: $ARGUMENTS

Output language: Japanese, formal business tone

## Prerequisites

- gh CLI installed and authenticated
- $ARGUMENTS: PR number
- Run from repository root
- PR must contain AWS SDK Go migration changes

## Process

1. **Fetch and validate PR**
   - Run: `gh pr diff $ARGUMENTS`
   - Check if diff contains `github.com/aws/aws-sdk-go` imports/changes
   - If not AWS SDK related, output: "このPRはAWS SDK Go関連の変更を含んでいません" and stop

2. **Analyze AWS SDK migration with Task tool** (subagent_type=general-purpose)
   - Identify changed files with AWS SDK usage
   - Extract function/method names that use AWS SDK v2
   - Identify AWS service types (DynamoDB, S3, SES, etc.)
   - Extract AWS connection settings (region, endpoint, resource names)
   - Document v1 → v2 migration patterns

3. **Trace call chains**
   For each function with AWS SDK v2 calls:
   - Find entry point (main.go, Lambda handler, HTTP handler)
   - Trace complete call chain from entry point to AWS SDK call
   - Identify intermediate layers (service, usecase, repository, gateway)

4. **Identify data source access for mocking**
   For each function containing AWS SDK v2 calls:
   - Scan function body from start to AWS SDK call
   - Identify all data source access operations:
     - Repository/gateway method calls
     - Database queries (SQL, etc.)
     - External API calls
     - File system reads
   - Extract return type for each data source access (from diff or function signatures)
   - Note variable names used to store retrieved data

5. **Generate test data and code modifications**
   For each identified data source access:
   - Create test data matching return type structure
   - Generate code modification showing:
     - Original data source call (commented out)
     - Test data assignment
     - Preserved downstream logic
   - For Get/Delete AWS operations: include pre-insert setup code

6. **Compile output**
   Generate complete report with:
   - File paths and function names
   - Complete call chains
   - AWS service and resource details
   - Migration changes summary
   - Test-ready code modifications
   - AWS console verification steps

## Output Format

```markdown
## AWS SDK接続先変更サマリー

### ファイル: [file_path:line_number]
#### 関数/メソッド: [function_name]

**呼び出しチェーン**:
```
[entry_point] (例: cmd/main.go:main())
  → [usecase/service_layer] (例: internal/usecase/user.go:(*UserUsecase).MigrateUser())
  → [repository/gateway_layer] (例: internal/repository/user.go:(*UserRepository).FetchByID())
  → AWS SDK v2 API呼び出し (例: DynamoDB PutItem)
```

**使用サービス**: [AWS Service Name (DynamoDB, S3, SES, etc.)]

**AWS接続先情報**:
- リージョン: [region or "デフォルト設定"]
- リソース名: [table name / bucket name / queue URL / etc.]
- エンドポイント: [カスタムエンドポイント or "デフォルト"]

**v1 → v2 変更内容**:
- クライアント初期化: `[v1 code]` → `[v2 code]`
- API呼び出し: `[v1 method]` → `[v2 method]`
- コンテキスト: [context propagation changes]
- 型変更: [type changes if any]

**動作確認観点**:
- AWSコンソール: [確認するサービス/リソース]
- CloudWatchログ: [確認すべきAPIコール]
- 設定確認: [region/endpoint/認証情報など]

---

[Repeat for each AWS SDK usage location]
```

## Analysis Guidelines

### General Analysis
- Focus on production AWS connections (exclude localhost/test endpoints)
- Trace complete call chain from entry point to AWS SDK v2 call
- Extract AWS resource names (table names, bucket names, queue URLs)
- Identify region configuration (explicit or AWS_REGION env)
- Summarize v1 → v2 migration patterns clearly
- Provide actionable AWS console verification steps

### Data Source Identification
- Look for these patterns BEFORE AWS SDK calls in function body:
  - `repo.Method()` / `gateway.Method()` calls
  - Direct database queries (`db.Query`, `db.Exec`)
  - HTTP client calls (`client.Get`, `http.Do`)
  - File reads (`os.ReadFile`, `ioutil.ReadFile`)
  - Cache access (`cache.Get`)
- Extract function signatures from PR diff or declaration
- Note variable names that store retrieved data

### Test Data Generation
- Match Go types exactly:
  - Structs: `&StructName{Field: value, ...}`
  - Slices: `[]Type{elem1, elem2}`
  - Maps: `map[KeyType]ValueType{key: value}`
  - Primitives: use realistic values
- Include all fields used in downstream logic
- For pointer types, use address operator: `&Type{}`
- For error returns: comment out error handling

### Code Modification Instructions
- Show original code with `//` comments (preserve indentation)
- Show test data assignment (match variable name exactly)
- Indicate preserved logic: validation, transformation, AWS SDK call
- For AWS Get/Delete: provide separate pre-insert code block
- Use `<details>` tags for better readability

## Notes

### Command Behavior
- Stop immediately if PR is not AWS SDK Go related
- Use Task tool for code analysis (avoid manual grep/read)
- Group output by file and function
- Include file:line references for easy navigation

### Output Quality
- Provide complete call chains for traceability
- Focus on connection configuration (client, endpoints, regions)
- Include specific AWS console verification steps
- Make test code modifications copy-paste ready
- Show realistic test data that matches actual types

### Testing Approach
- Mock only data source access (repository, DB, API, file)
- Keep AWS SDK v2 calls active (they will use test data)
- Preserve all business logic between data fetch and AWS call
- For Get/Delete: provide both pre-insert and execution code
