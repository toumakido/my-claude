Interactively analyze AWS SDK migration function by function

Output language: Japanese, formal business tone

## Command Purpose

This command **actually tests** AWS connections after AWS SDK Go v1→v2 migration by automatically executing:

1. Extract and analyze migrated functions
2. Generate test data and pre-insert to DynamoDB/S3
3. Mock external APIs (data source replacement)
4. **Apply actual code modifications** (automatic rewrite with Edit tool)
5. Execute AWS SDK v2 API tests

**Important**: This command does not just analyze - it **actually modifies code**. Changes can be reverted with `git restore` if under Git management.

## Prerequisites

- Run from repository root
- Current branch must contain AWS SDK Go v2 migration changes
- Working tree can be dirty (uncommitted changes allowed)

## Process

### Phase 1: Extract Functions and Call Chains

1. **Fetch and validate branch diff**
   - Run: `git diff main...HEAD`
   - If diff does not contain `github.com/aws/aws-sdk-go-v2` imports: output "このブランチはAWS SDK Go v2関連の変更を含んでいません" and stop

2. **Extract functions and call chains with Task tool** (subagent_type=general-purpose)
   Task prompt: "Parse git diff and extract all functions/methods using AWS SDK v2 with their call chains.

   Step 1: Extract functions. Search patterns:
   - Import: `github.com/aws/aws-sdk-go-v2/service/*`
   - Client calls: `client.PutItem`, `client.GetObject`, etc.
   - Context parameter: functions with `context.Context` calling AWS clients

   For each match, extract:
   - File path:line_number from diff headers
   - Function/method name from signature
   - AWS service from import path (dynamodb, s3, ses, etc.)
   - Operation from client method name (PutItem, GetObject, etc.)

   Step 2: For each function, trace ALL call chains using Grep:
   - Find entry points: `main\(`, `handler\(`, `ServeHTTP`, `Handle`
   - Search function references to trace call paths
   - Identify intermediate layers (usecase/service/repository/gateway)
   - Build complete chains: entry → intermediate → SDK function
   - For each chain, count all AWS SDK v2 method calls within the chain
   - Mark chains with multiple SDK methods as high priority
   - Execute Grep searches in parallel for independent functions

   Step 3: Sort call chains by priority:
   1. First: Chains with multiple AWS SDK methods (higher priority)
   2. Within same SDK method count: Sort by chain length (shorter = easier)

   Example priority order:
   - Chain with 3 SDK methods, 4 hops (highest)
   - Chain with 2 SDK methods, 2 hops
   - Chain with 2 SDK methods, 5 hops
   - Chain with 1 SDK method, 2 hops
   - Chain with 1 SDK method, 4 hops (lowest)

   Return: function list with all call chains sorted by priority."

3. **Group and deduplicate chains with Task tool** (subagent_type=general-purpose)
   Task prompt: "Group and deduplicate call chains to create optimal combination:

   Step 1: Group by SDK operation type (動作確認の観点)
   - Key: AWS_service + SDK_operation
   - Value: list of call chains using same SDK operation
   - Example: all chains using 'DynamoDB PutItem' are grouped together
   - Ignore: file path, line number, function name, region, endpoint, table/bucket names, filters, and all parameters
   - Rationale: From operation verification perspective, only AWS service type and SDK operation matter

   Examples of grouping:
   ```
   Same group (same S3 GetObject):
   - s3.go:45 | DownloadFromBucketA | S3 GetObject (bucket: bucket-a)
   - s3.go:89 | DownloadFromBucketB | S3 GetObject (bucket: bucket-b)
   → Only verify one

   Same group (same DynamoDB Query):
   - user.go:30 | GetByStatus | DynamoDB Query (filter: status)
   - user.go:60 | GetByAge | DynamoDB Query (filter: age)
   → Only verify one
   ```

   Step 2: Select representative chain from each group
   For each group with multiple chains:
   - Priority 1: Shortest chain (fewest hops)
   - Priority 2: Entry point is 'main' function
   - Priority 3: First in list (tie-breaker)
   - Mark selected chain with: [+N other operations] where N = group size - 1

   For groups with single chain:
   - Select the only chain
   - No marker needed

   Step 3: Create optimal combination
   - Combine all selected representative chains
   - Maintain original priority sorting (from step 2)
   - Result: minimal set covering all unique SDK operation types

   Return: optimal combination (deduplicated chains), each with group info"

4. **Format and cache optimal combination**
   Store Task result in variable for batch processing.

   Output:
   ```
   === 重複排除と最適化後の組み合わせ ===

   合計SDK関数数: N個 (重複排除前: M個のチェーン)
   選択されたチェーン数: N個

   [Sorted by priority: multiple SDK methods first, then by chain length]

   各チェーンに [+X other chains] マーカーを表示（重複がある場合）
   ```

### Phase 2: Batch Approval

5. **Present optimal combination and get batch approval**

   A. Display summary of optimal combination:
   ```
   === バッチ処理する組み合わせ ===

   合計SDK関数数: N個
   - Read操作: X個
   - Write操作: Y個
   - Delete操作: Z個
   - 複数SDK使用: W個

   処理対象のチェーン:
   1. [file:line] | [function] | [operations] [markers]
   2. [file:line] | [function] | [operations] [markers]
   ...
   ```

   B. Request batch approval with AskUserQuestion:
   - question: "この組み合わせでN個のSDK関数をバッチ処理しますか？"
   - header: "Batch"
   - multiSelect: false
   - options:
     - label: "はい", description: "N個のSDK関数を全自動で順次処理"
     - label: "いいえ", description: "キャンセルして終了"

   C. Handle response:
   - If "はい" selected: proceed to Phase 3 (batch processing)
   - If "いいえ" selected: exit with "処理をキャンセルしました"

### Phase 3: Batch Processing

6. **Process all chains in optimal combination sequentially**

   For each chain in optimal combination (index i from 1 to N):

   A. Display progress:
   ```
   === 処理中 (i/N) ===
   関数: [file_path:line_number] | [function_name] | [operations]
   ```

   B. Execute steps 7-14 for current chain

7. **Analyze selected function with Task tool** (subagent_type=general-purpose)
   Task prompt: "For function [function_name] at [file_path:line_number] with call chain [selected_chain]:

   1. Identify AWS SDK operation type using Read:
      - Find AWS SDK v2 client method calls: `client\.GetItem`, `client\.PutItem`, etc.
      - Classify operation type:
        - Read operations: GetItem, GetObject, DescribeTable, etc.
        - Delete operations: DeleteItem, DeleteObject, etc.
        - Write operations: PutItem, PutObject, UpdateItem, etc.
      - Extract operation name and line number

   2. Extract AWS settings from [function_name] using Read:
      - Region: look for `WithRegion\|AWS_REGION`
      - Resource: table name, bucket name from client call parameters
      - Endpoint: look for `WithEndpointResolver\|endpoint`

   3. Document v1 → v2 changes from git diff:
      - Client init: session.New vs config.LoadDefaultConfig
      - API call: old vs new method signature
      - Type changes: aws.String vs direct string usage

   Return: AWS operation type (Read/Delete/Write), operation name, line number, AWS settings, migration summary. Use [selected_chain] as call chain (do not re-trace)."

7.5. **Analyze filter conditions and determine test data strategy with Task tool** (subagent_type=general-purpose)
   Task prompt: "Analyze FilterExpression in [function_name] at [file_path:line_number] and determine test data strategy:

   Purpose: Avoid functional duplication in test data across multiple handlers

   1. Extract FilterExpression from AWS SDK Read operation:
      - Use Read to search for `FilterExpression:` within [function_name] scope
      - Extract complete filter string and parameter values from Scan/Query input parameters
      - Example: `FilterExpression: \"(attribute_not_exists(#OP) OR attribute_type(#OP, :null)) AND (#RT = :rt)\"`

   2. Categorize filter complexity:
      - Simple filter: Single attribute check (e.g., `#FIELD = :value`)
      - Complex filter: Multiple conditions with OR/AND (e.g., `(attribute_not_exists(#F) OR #F = :empty) AND #T <= :lte`)
      - No filter: GetItem or operations without FilterExpression

   3. Identify filter pattern:
      - Extract filter structure (ignore parameter values)
      - Example pattern: `(attribute_not_exists(X) OR attribute_type(X, :null)) AND (Y = :value)`
      - Compare with filters from other functions in optimal combination

   4. Determine test data strategy:
      - If this is the FIRST occurrence of this filter pattern: **Comprehensive testing**
        - Generate both matching and non-matching test records
        - Test all filter condition branches (OR, AND, attribute_not_exists, etc.)
      - If this filter pattern already tested by another function: **Minimal or skip**
        - Skip Pre-insert (rely on other function's filter testing)
        - Or generate minimal records for function-specific processing only
      - If complex filter (3+ conditions): **Always test comprehensively**
        - Regardless of other functions
      - If no filter: **Basic SDK operation testing**
        - Generate 1-2 matching records only

   5. Document decision:
      - Return filter pattern, complexity, strategy decision, and rationale
      - Example: \"Complex filter with attribute_not_exists + parameter check. First occurrence. Strategy: Comprehensive testing with 2 match + 2 non-match records.\"

   Return: Filter pattern, complexity level, test data strategy (comprehensive/minimal/skip), match record count, non-match record count, rationale."

8. **Identify data source access with Task tool** (subagent_type=general-purpose)
   Task prompt: "In function [function_name] at [file_path:line_number], identify ALL data source access BEFORE AWS SDK calls using Read:

   Search patterns:
   - Repository/gateway calls: `repo\.\|gateway\.\|[A-Z][a-z]*Repository\|[A-Z][a-z]*Gateway`
   - Database: `db\.Query\|db\.Exec\|\.Scan\|\.QueryRow`
   - HTTP: `client\.Get\|client\.Post\|http\.Do`
   - File: `os\.ReadFile\|ioutil\.ReadFile\|os\.Open`
   - Cache: `cache\.Get\|redis\.Get`

   For each match, extract:
   - Line number
   - Method signature or function call
   - Variable name storing result
   - Return type from function declaration or type inference

   Return: list with line numbers, calls, variables, types."

8.5. **Analyze downstream processing with Task tool** (subagent_type=general-purpose)
   Task prompt: "Analyze ALL code after AWS SDK calls in [function_name] at [file_path:line_number] to identify required fields:

   Purpose: Ensure test data is complete enough to avoid runtime errors in business logic

   1. Identify AWS SDK call boundaries using Read:
      - Find AWS SDK method calls: client.GetItem, client.Scan, client.PutItem, etc.
      - Mark line numbers where AWS SDK operations complete
      - All code AFTER these lines (in the same function) is \"downstream processing\"

   2. Analyze downstream field usage patterns for ALL variables from data sources (from step 8):

      **Pattern A: Nil pointer dereference**
      - Direct dereference: *field, field.Method()
      - Without nil check: if no `field != nil` guard before usage
      - Severity: CRITICAL (causes panic)
      - Example: logger.Infof(\"Value: %s\", *acc.BranchCode)

      **Pattern B: Validation function calls**
      - validator.Validate(obj), obj.Validate()
      - Custom validation: ValidateXXX(obj), obj.IsValid()
      - If validation function found: use Grep + Read to locate and read validator definition
      - Severity: HIGH (causes error return)
      - Example: if err := validator.Validate(acc); err != nil

      **Pattern C: Struct tag validation**
      - Use struct definition from step 9
      - Extract validation tags: `validate:\"required\"`, `validate:\"min=1,max=100\"`, etc.
      - Extract json tags: `json:\"field,omitempty\"` → optional, `json:\"field\"` → required
      - Severity: HIGH (validation will fail)

      **Pattern D: Length/empty checks**
      - len(field) == 0, len(field) > 0
      - field == \"\", field != \"\"
      - strings.TrimSpace(field) == \"\"
      - Severity: MEDIUM (causes conditional error)
      - Example: if len(acc.Email) == 0 { return errors.New(...) }

      **Pattern E: Numeric range checks**
      - field > max, field < min
      - field >= 0, field <= limit
      - Severity: MEDIUM (causes conditional error)
      - Example: if *acc.Balance < 0 { return errors.New(...) }

      **Pattern F: Field used in function calls**
      - Function arguments: doSomething(obj.Field1, obj.Field2)
      - Method calls on fields: obj.Field.Method()
      - External API calls: api.Call(obj.Field)
      - Severity: depends on function (check if function handles nil/zero values)
      - Example: score := calculateCredit(acc.CreditHistory, acc.TransactionCount)

      **Pattern G: Loop/iteration on slices**
      - for _, item := range obj.Items
      - obj.Items[0], obj.Items[i]
      - Severity: CRITICAL if slice is nil (panic on nil slice iteration or index access)

   3. For each field usage, determine requirement level:
      - CRITICAL: Must be non-nil and valid (no nil check, direct dereference, or used in loops)
      - HIGH: Must satisfy validation rules (validator, struct tags)
      - MEDIUM: Must pass business logic checks (length, range)
      - LOW: Used but has nil check or default handling
      - OPTIONAL: Has nil check and safe fallback

   4. Build field requirement map:
      - Field name → Requirement level → Reason → Line number(s)
      - Include both pointer and non-pointer fields
      - Include nested struct fields if accessed (e.g., obj.Nested.Field)
      - Include slice fields if iterated or indexed
      - Example:
        {
          \"BranchCode\": \"CRITICAL - Direct dereference at line 45 without nil check\",
          \"Email\": \"HIGH - Length check at line 52, must be non-empty\",
          \"Balance\": \"MEDIUM - Range check at line 58, must be >= 0 and <= 10000000\",
          \"CreditScore\": \"CRITICAL - Used in function call at line 65, function does not handle nil\",
          \"Transactions\": \"CRITICAL - Iterated in for loop at line 72, must be non-nil slice\",
        }

   5. Identify validation requirements:
      - Extract validator rules from struct tags (from step 9)
      - Find Validate() method implementation (use Grep + Read if exists)
      - Document expected value formats/ranges
      - Example:
        {
          \"Email\": \"validate:\\\"required,email\\\" - must be valid email format\",
          \"PhoneNumber\": \"validate:\\\"required,len=13\\\" - must be exactly 13 characters\",
          \"Age\": \"validate:\\\"min=0,max=150\\\" - must be between 0 and 150\",
        }

   Return: Field requirement map with levels, reasons, line numbers, and validation rules."

9. **Validate and extract type information with Task tool** (subagent_type=general-purpose)
   Task prompt: "Before generating test data, extract and validate type information from model definitions:

   Purpose: Prevent compilation errors from incorrect type assumptions

   1. Find and read model definition files using Glob + Read:
      - Search for `type.*struct` in `model.go`, `models.go`, `types.go`, `entity.go`, `dto.go`
      - Search in same directory as [file_path] and parent directories (use parallel Glob searches)
      - Check return types in function signatures
      - After identifying all model files: READ each file before proceeding to step 2
      - **Critical**: Never assume field names or types - always read actual file contents

   2. Extract type information for each data source return type (from step 8):
      - Struct fields and their types (including pointer types: `*string`, `*int64`, `*bool`)
      - Slice element types (distinguish `[]*Type` vs `[]Type`)
      - Map key/value types
      - Exported vs unexported fields
      - Nested struct types
      - Struct tags: `json:\"fieldName\"`, `dynamodbav:\"AttributeName\"`

   3. Verify pointer type usage patterns:
      - For pointer fields (`Field *string`), test data must allocate pointer:
        ```go
        // Correct for *string fields
        value := \"example\"
        entity := &Entity{Field: &value}

        // Or use aws.String helper
        entity := &Entity{Field: aws.String(\"example\")}

        // Incorrect: direct string literal (compilation error)
        entity := &Entity{Field: \"example\"}
        ```
      - Document which fields require pointer allocation

   4. Check required imports for test data:
      - AWS SDK v2 packages: `github.com/aws/aws-sdk-go-v2/aws` (for `aws.String`, `aws.Int64`, etc.)
      - AWS SDK v2 packages: `github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue` (for `MarshalMap`, `UnmarshalMap`)
      - AWS SDK v2 service types: `github.com/aws/aws-sdk-go-v2/service/*/types` (for service-specific types)
      - Standard library packages for test data construction

   5. For each struct type, document with examples:
      - Full type name (e.g., `Account`, `*Account`, `[]Account`, `[]*Account`)
      - All field names with exact casing (e.g., `CustomerCode` not `AccountID`)
      - Field types with pointer indicators (e.g., `*string`, `int64`, `*bool`)
      - Correct initialization example:
        ```go
        // Example: type Account struct { BranchCode string; CustomerCode string; Balance *int64 }
        testAccount := &Account{
            BranchCode:   \"100\",            // Non-pointer field
            CustomerCode: \"123456\",         // Non-pointer field
            Balance:      aws.Int64(1000000), // Pointer field - use aws.Int64
        }
        ```

   Return:
   - Type information map with exact field names, types, pointer indicators
   - Pointer allocation patterns for each field
   - Required imports list with full package paths
   - Initialization examples for each struct type"

10. **Generate test data and pre-insert code with Task tool** (subagent_type=general-purpose)
   Task prompt: "Generate test data and code modifications using validated type information from step 9 and field requirements from step 8.5:

   Part A - Data source access replacement (from step 8):
   For each data source access:
   1. Generate COMPLETE test data (COMPLETE = satisfying all 5 priority levels below) prioritizing by field requirements:

      **Priority 1: AWS SDK parameter fields** (from step 7)
      - Fields used in AWS SDK call parameters (e.g., Key, FilterExpression values)
      - Must be set with realistic values

      **Priority 2: CRITICAL fields** (from step 8.5)
      - Nil pointer dereference without guards
      - Fields used in function calls where nil causes panic
      - Slice fields that are iterated or indexed
      - Must be non-nil with realistic values

      **Priority 3: HIGH fields** (from step 8.5)
      - Fields with validation requirements (struct tags, Validate() methods)
      - Must satisfy all validation rules:
        - `validate:\"required\"` → non-nil, non-empty
        - `validate:\"email\"` → valid email format
        - `validate:\"min=X,max=Y\"` → within range
        - `validate:\"len=N\"` → exactly N characters/elements

      **Priority 4: MEDIUM fields** (from step 8.5)
      - Fields with business logic checks (length, range, format)
      - Must satisfy conditions to avoid error returns

      **Priority 5: OPTIONAL fields**
      - Fields with safe nil handling or default fallbacks
      - Set for completeness but not critical

   2. Use exact type information from step 9:
      - Match struct field names exactly (e.g., `CustomerCode` not `AccountID`)
      - Use pointer types where required: `aws.String(\"value\")`, `aws.Int64(123)`, `aws.Bool(true)`
      - Initialize slices as non-nil: `[]Type{}` or `[]*Type{{...}}`
      - Match slice element types: `[]*Type` vs `[]Type`
      - Example for Account struct:
        ```go
        // From step 9: type Account struct {
        //   BranchCode *string `validate:\"required\"`
        //   CustomerCode string `validate:\"required,len=6\"`
        //   Email string `validate:\"required,email\"`
        //   Balance *int64 `validate:\"min=0,max=10000000\"`
        //   Transactions []*Transaction
        // }

        testAccount := &Account{
            // Priority 1: AWS SDK parameter
            CustomerCode: \"123456\", // Used in DynamoDB Key

            // Priority 2: CRITICAL fields
            BranchCode: aws.String(\"100\"), // Dereferenced at line 45
            Transactions: []*Transaction{   // Iterated at line 72
                {ID: \"tx-001\", Amount: aws.Int64(5000)},
            },

            // Priority 3: HIGH fields (validation)
            Email: \"test@example.com\", // validate:\"required,email\"

            // Priority 4: MEDIUM fields (business logic)
            Balance: aws.Int64(1000000), // Range check at line 58: 0 <= x <= 10000000

            // Priority 5: OPTIONAL fields
            // (none in this example)
        }
        ```

   3. Add comment documenting field requirements:
      ```go
      // Test data for [function_name]
      // AWS SDK fields: CustomerCode
      // CRITICAL fields: BranchCode (deref:45), Transactions (loop:72)
      // Validation fields: Email (required,email)
      // Business logic: Balance (range:0-10000000)
      ```

   4. Include all required imports in new_string:
      - Add missing imports from step 9 to import block
      - Example: `\"github.com/aws/aws-sdk-go-v2/aws\"`, `\"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue\"`

   5. Generate code modification:
      - Comment out original call with `//`
      - Assign test data to same variable
      - Preserve all downstream logic

   Part B - AWS SDK pre-insert code (from step 7 and step 7.5):
   If AWS operation type is Read or Delete:

   **Use test data strategy from step 7.5 to determine record counts**

   1. Generate BOTH matching and non-matching test records:

      **Matching records** (should be retrieved by filter):
      - Generate N records based on step 7.5 strategy (e.g., 1-2 for minimal, 2-3 for comprehensive)
      - Ensure all FilterExpression conditions are satisfied
      - Example for filter `(attribute_not_exists(#OP) OR #OP = :empty) AND (#RT = :rt)`:
        ```go
        matchRecords := []*Entity{
            {ID: \"match-001\", RequestType: aws.Int64(1), OpStatus: nil},        // attribute_not_exists
            {ID: \"match-002\", RequestType: aws.Int64(1), OpStatus: aws.String(\"\")}, // OR #OP = :empty
        }
        ```

      **Non-matching records** (should be excluded by filter):
      - Generate M records based on step 7.5 strategy (e.g., 1 for minimal, 2-3 for comprehensive)
      - Violate different FilterExpression conditions to test filter correctness
      - Example for same filter:
        ```go
        nonMatchRecords := []*Entity{
            {ID: \"nomatch-001\", RequestType: aws.Int64(1), OpStatus: aws.String(\"PROCESSED\")}, // Non-null OpStatus
            {ID: \"nomatch-002\", RequestType: aws.Int64(999), OpStatus: nil},                    // Wrong RequestType
        }
        ```

      **Special cases**:
      - Empty strings: Include if filter checks `attribute_type` or `= :empty`
      - Timestamps: Include out-of-range values if filter uses `<=`, `>=`, `BETWEEN`
      - NULL types: Consider SDK v1 migration scenarios where NULL handling may differ

   2. Generate Pre-insert code for both record sets:
      - For GetItem/Scan/Query: generate PutItem for each test record (matching + non-matching)
      - For GetObject: generate PutObject with same key
      - For DeleteItem: generate PutItem with same key
      - Use same resource (table/bucket) from step 7
      - Use realistic test data values with correct types from step 9
      - Add comment documenting expected behavior:
        ```go
        // Pre-insert: test data for Scan operation
        // Matching records: 2 (should be retrieved)
        // Non-matching records: 2 (should be excluded)
        ```

   3. Identify insertion point:
      - Line number just before AWS SDK Read/Delete operation
      - Preserve indentation

   Return:
   - Data source replacements: [original code (commented), test assignment code with imports]
   - Pre-insert code (if AWS operation is Read/Delete): [pre-insert code with match/non-match records, match count, non-match count, insertion line number]"

11. **Apply code modifications with Edit tool**

   **Important**: This step applies actual code changes. This is not just analysis.

   Part A - Replace data source access (from step 10 Part A):
   For each data source access identified in step 8:
   - Use Edit tool to replace original data source call with test data
   - old_string: exact original code from function
   - new_string: test data assignment preserving downstream logic
   - If multiple data sources: apply edits sequentially
   - Output: "データソース書き換え完了: [file_path:line_number]"

   Part B - Insert pre-insert code (from step 10 Part B):
   If AWS operation type is Read or Delete:
   - Use Edit tool to insert pre-insert code before AWS SDK operation
   - Identify the line before AWS SDK call using line number from step 10
   - old_string: line before AWS SDK operation (preserve exact indentation)
   - new_string: line before AWS SDK operation + "\n" + pre-insert code (with proper indentation)
   - Output: "Pre-insertコード追加: [file_path:line_number]"
   - Add comment above pre-insert code: "// Pre-insert: test data for [operation_name]"

   Part C - Add verification logging (from step 9 and step 10):
   For Read operations (Scan, Query, GetItem, GetObject):
   - Use Edit tool to insert logging code after AWS SDK Read operation
   - Extract key fields from type information (step 9)
   - Use expected record counts from step 10 (match count, non-match count)
   - Log format with filter verification:
     ```go
     // Verify Pre-insert: log retrieved records
     // Expected: N matching records (inserted: N match + M non-match)
     logger.Infof(\"Test records inserted: %d match (should be retrieved), %d non-match (should be excluded)\", matchCount, nonMatchCount)
     logger.Infof(\"[function_name] returned %d records (expected: %d)\", len(result), matchCount)

     if len(result) != matchCount {
         logger.Warnf(\"Filter verification failed: expected %d records but got %d\", matchCount, len(result))
     }

     // Log details of retrieved records for debugging
     for i, record := range result {
         logger.Infof(\"  [%d] Key1=%v, Key2=%v, ...\", i, record.Key1, record.Key2)
     }
     ```
   - Insert after Read operation, before any result length check
   - Include matchCount and nonMatchCount as constants in the code (from step 10)
   - Output: "検証ログ追加: [file_path:line_number]"

12. **Verify compilation and runtime safety with Bash tool**

   A. Compile check:
   - Run: `go build -o /tmp/test-build 2>&1`
   - If compilation fails:
     - Analyze error messages
     - Fix missing imports, type mismatches, undefined fields
     - Retry Edit tool with corrections
     - Repeat until compilation succeeds
   - Output: "コンパイル成功: [file_path]"

   B. Static analysis (if tools available):
   - Run go vet and staticcheck in parallel (independent checks):
     - `go vet ./... 2>&1` (check for common mistakes including nil pointer issues)
     - `staticcheck ./... 2>&1` if installed (advanced checks)
   - If go vet reports issues in modified files:
     - Analyze warnings (especially nil pointer dereferences, unreachable code)
     - Output: "警告: go vet detected issues: [summary]"
   - If staticcheck reports issues in modified files:
     - Output: "警告: staticcheck detected issues: [summary]"
   - Note: Only report issues in files modified by this command, ignore pre-existing issues

   C. Document potential runtime risks:
   - Review field requirement map from step 8.5
   - Identify fields marked as CRITICAL or HIGH that may still cause issues:
     - Fields used in external function calls (depends on function implementation)
     - Complex validation logic that cannot be fully analyzed
     - Dynamic field access (reflection, map lookups)
   - If potential risks exist, output warning:
     ```
     潜在的なランタイムリスク:
     - [function_name] at line X: [field_name] used in external call [function_call]
     - Recommendation: Verify [function_call] handles nil/zero values correctly
     ```
   - Suggest additional manual testing if needed

13. **Output detailed report**
    Generate report for selected function with:
    - File path and function name
    - Complete call chain
    - AWS operation type (Read/Delete/Write) and operation name
    - AWS service and resource details
    - Migration changes summary
    - Applied code modifications:
      - Data source replacements (if any)
      - Pre-insert code (if AWS operation is Read/Delete)
      - Verification logging (if AWS operation is Read)
    - Compilation status
    - AWS console verification steps
    - Git diff summary showing changes

14. **Output brief summary for current chain**
    ```
    完了 (i/N): [file_path]
    - AWS操作: [operation_type] ([operation_name])
    - データソースモック: X個
    - Pre-insertコード: 追加済み / 不要
    - 検証ログ: 追加済み / 不要
    - コンパイル: 成功 / 失敗
    ```

   C. Proceed to next chain automatically (no user interaction)
      - If i < N: continue to next chain (repeat from step 6.A)
      - If i = N: proceed to final summary

### Phase 4: Final Summary

15. **Output final summary report**
    After processing all chains, display comprehensive summary:

    ```
    === 処理完了サマリー ===

    処理したSDK関数: N個
    - Read操作: X個 (Pre-insertコード追加済み、検証ログ追加済み)
    - Write操作: Y個
    - Delete操作: Z個 (Pre-insertコード追加済み)
    - 複数SDK使用: W個

    書き換えたファイル: (unique list)
    - [file_path_1]
    - [file_path_2]
    ...

    データソースモック: 合計M個
    検証ログ: 合計L個
    コンパイル: 成功P個 / 失敗Q個

    次のステップ:
    1. git diff で変更内容を確認
    2. コンパイルエラーがある場合は修正
    3. アプリケーションを実行してAWS接続をテスト
    4. ログ出力から取得レコード数と内容を確認
    5. CloudWatchログで各API呼び出しを確認
    6. 必要に応じてテストデータを調整

    すべての処理が完了しました。
    ```

## Output Format

### Optimal Combination (Phase 1)
```
=== 重複排除と最適化後の組み合わせ ===

合計SDK関数数: 5個 (重複排除前: 8個のチェーン)
選択されたチェーン数: 5個

[Sorted by priority: multiple SDK methods first, then by chain length]

1. internal/service/order.go:120 | (*OrderService).Process | DynamoDB + S3 + SES [★ Multiple SDK]
   Chain: main → OrderService.Process → DynamoDB PutItem → S3 PutObject → SES SendEmail (3 SDK methods, 4 hops)

2. internal/gateway/data.go:89 | (*DataGateway).Import | S3 + DynamoDB [★ Multiple SDK] [+1 other chain]
   Chain: handler → ImportService.Run → S3 GetObject → DynamoDB BatchWriteItem (2 SDK methods, 5 hops)

3. internal/repository/user.go:45 | (*UserRepository).Save | DynamoDB PutItem [+2 other chains]
   Chain: main → UserUsecase.Create → UserRepository.Save (1 SDK method, 2 hops)

4. internal/gateway/s3.go:120 | (*S3Gateway).Upload | S3 PutObject
   Chain: main → FileService.Process → S3Gateway.Upload (1 SDK method, 2 hops)

5. internal/repository/user.go:89 | (*UserRepository).Get | DynamoDB GetItem
   Chain: handler → UserService.Fetch → UserRepository.Get (1 SDK method, 3 hops)
```

### Batch Approval Summary (Phase 2)
```
=== バッチ処理する組み合わせ ===

合計SDK関数数: 5個
- Read操作: 1個
- Write操作: 3個
- Delete操作: 0個
- 複数SDK使用: 2個

処理対象のチェーン:
1. order.go:120 | Process | DynamoDB + S3 + SES [★]
2. data.go:89 | Import | S3 + DynamoDB [★] [+1]
3. user.go:45 | Save | DynamoDB [+2]
4. s3.go:120 | Upload | S3
5. user.go:89 | Get | DynamoDB
```

### Detailed Report for Selected Function (Phase 3)
```markdown
## 選択した関数の接続検証情報

### ファイル: [file_path:line_number]
#### 関数/メソッド: [function_name]

**AWS操作タイプ**: [Read/Delete/Write]
**AWS操作名**: [operation_name] (例: GetItem, PutItem, DeleteObject)

**呼び出しチェーン**:
```
[entry_point] (例: cmd/main.go:main())
  → [usecase/service_layer] (例: internal/usecase/user.go:(*UserUsecase).MigrateUser())
  → [repository/gateway_layer] (例: internal/repository/user.go:(*UserRepository).FetchByID())
  → AWS SDK v2 API呼び出し (例: DynamoDB GetItem)
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

**適用したコード変更**:

**A. データソースのモック** (該当する場合):
<details>
<summary>元のコード</summary>

```go
// Original data source access code
```
</details>

<details>
<summary>テスト用コード</summary>

```go
// Test data assignment code
```
</details>

**B. Pre-insertコード** (AWS操作がRead/Deleteの場合):
<details>
<summary>追加されたPre-insertコード</summary>

```go
// Pre-insert: test data for [operation_name]
// [pre-insert code that populates test data]
// Example for GetItem:
_, err := client.PutItem(ctx, &dynamodb.PutItemInput{
    TableName: aws.String("users"),
    Item: map[string]types.AttributeValue{
        "id": &types.AttributeValueMemberS{Value: "test-123"},
        "name": &types.AttributeValueMemberS{Value: "Test User"},
    },
})
```
</details>

**動作確認観点**:
- AWSコンソール: [確認するサービス/リソース]
- CloudWatchログ: [確認すべきAPIコール]
- 設定確認: [region/endpoint/認証情報など]
```

### Brief Summary (Phase 3, per chain)
```
完了 (3/5): internal/repository/user.go
- AWS操作: Write (PutItem)
- データソースモック: 2個
- Pre-insertコード: 不要
```

### Final Summary (Phase 4)
```
=== 処理完了サマリー ===

処理したSDK関数: 5個
- Read操作: 1個 (Pre-insertコード追加済み)
- Write操作: 3個
- Delete操作: 0個
- 複数SDK使用: 2個

書き換えたファイル:
- internal/service/order.go
- internal/gateway/data.go
- internal/repository/user.go
- internal/gateway/s3.go

データソースモック: 合計8個

次のステップ:
1. git diff で変更内容を確認
2. アプリケーションを実行してAWS接続をテスト
3. CloudWatchログで各API呼び出しを確認
4. 必要に応じてテストデータを調整

すべての処理が完了しました。
```

## Analysis Requirements

### General
- Focus on production AWS connections (exclude localhost/test endpoints)
- Extract resource names (table names, bucket names, queue URLs)
- Identify region configuration (explicit config or AWS_REGION env var)
- Summarize v1 → v2 migration patterns clearly
- Provide actionable AWS console verification steps

### Critical Process Steps
Detailed instructions are in Process section above. Key requirements:
- **Type validation (step 9)**: READ actual model files, never assume field names/types
- **Downstream analysis (step 8.5)**: Identify CRITICAL/HIGH/MEDIUM/OPTIONAL field requirements
- **Filter analysis (step 7.5)**: Categorize complexity and determine test data strategy
- **Test data generation (step 10)**: Generate COMPLETE data satisfying all 5 priority levels
- **Test records (step 10 Part B)**: Generate BOTH matching and non-matching records for Read/Delete
- **Static analysis (step 12)**: Run go vet and staticcheck in parallel, document runtime risks

### Complex Chain Handling
- **Multiple AWS SDK calls**: Process all SDK calls within the same function
  - Step 2 identifies chains with multiple SDK methods and prioritizes them
  - Step 7 identifies all AWS SDK operations within the function
  - Steps 8-11 process each SDK operation independently
  - Generate separate Pre-insert code for each Read/Delete operation
  - Apply data source mocks once (shared across all SDK operations in the function)
- **Complex detection criteria** (for reference):
  - AWS SDK calls: 3 or more operations
  - Data source access: 4 or more calls
  - Call chain depth: 6 or more hops
- **Processing approach**: Attempt automatic processing for all chains
  - Only suggest manual intervention when Task tool encounters unrecoverable errors
  - Provide clear error context and suggested fixes when manual intervention needed

### Batch Processing UX
- Phase 1: Extract and deduplicate automatically
- Phase 2: Single batch approval via AskUserQuestion
- Phase 3: Process all chains automatically without user interaction
- Phase 4: Display final summary
- No interactive loop - fully automated after approval

## Notes

### Tool Usage
- Stop immediately if branch diff does not contain `aws-sdk-go-v2` imports
- Use Task tool for code analysis (steps 2, 3, 7, 7.5, 8, 8.5, 9, 10)
- Use Edit tool to automatically apply code modifications (step 11)
- Use Bash tool for compilation and static analysis (step 12)

### Key Process Steps
- **Deduplication** (step 3): Group by AWS_service + SDK_operation, ignore parameters. Select shortest chain from each group.
- **Filter analysis** (step 7.5): Categorize filter complexity and determine test data strategy (comprehensive/minimal/skip) to avoid duplication.
- **Downstream processing analysis** (step 8.5): Analyze code AFTER AWS SDK calls to identify field requirements (CRITICAL/HIGH/MEDIUM/OPTIONAL) and prevent runtime errors.
- **Type validation** (step 9): Actually READ model files. Extract exact field names, pointer types, slice types, struct tags, validation rules.
- **Test data generation** (step 10): Generate COMPLETE test data satisfying all field requirements (AWS SDK + downstream processing). Generate BOTH matching and non-matching records for Read/Delete operations.
- **Code modifications** (step 11): Replace data sources, insert pre-insert code, add verification logging.
- **Compilation and safety verification** (step 12): Run `go build`, fix errors automatically. Run `go vet` and `staticcheck` for static analysis. Document potential runtime risks.

### Output Guidelines
- Include file:line references for navigation
- Provide complete call chains for traceability
- Mark chains with multiple SDK methods with [★ Multiple SDK] indicator
- Display progress (i/N) during batch processing
- Show final summary with compilation status

For detailed requirements, see Analysis Requirements section above.
