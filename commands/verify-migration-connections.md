Analyze AWS SDK migration PR and provide connection verification information: $ARGUMENTS

Output language: Japanese, formal business tone

## Prerequisites

- gh CLI installed and authenticated
- $ARGUMENTS: PR number
- Run from repository root
- PR must contain AWS SDK Go migration changes

## Process

1. Fetch PR diff: `gh pr diff $ARGUMENTS`
2. Validate PR is AWS SDK Go related:
   - Check if diff contains `github.com/aws/aws-sdk-go` imports/changes
   - If not related, output: "このPRはAWS SDK Go関連の変更を含んでいません" and stop
3. Use Task tool (subagent_type=general-purpose) to analyze:
   - Changed files with AWS SDK usage
   - Function/method names that call AWS SDK APIs
   - Call chain from entry point (main/handler) to AWS SDK call
   - AWS service types (S3, DynamoDB, SES, etc.)
   - AWS connection settings (region, endpoint, table/bucket names)
   - Migration changes (v1 → v2 patterns)
4. For each AWS SDK usage location:
   - Extract entry point (main.go, Lambda handler, Echo handler)
   - Trace call chain through service/repository layers
   - Identify AWS resource names (table names, bucket names, etc.)
   - Extract region configuration
   - Summarize v1 → v2 changes
5. Generate verification information with:
   - File path and function/method name
   - Complete call chain from entry point
   - AWS service and resource details
   - Connection configuration changes
   - Verification points for AWS console

## Output Format

```markdown
## AWS SDK接続先変更サマリー

### ファイル: [file_path]
#### 関数/メソッド: [function_name]

**呼び出しチェーン**:
```
[entry_point] (例: main.go:main() or handler.go:HandleRequest())
  → [service_layer] (例: service/user_service.go:(*UserService).GetUser())
  → [repository_layer] (例: repository/user_repo.go:(*UserRepository).FetchByID())
  → AWS SDK API呼び出し
```

**使用サービス**: [AWS Service Name (e.g., DynamoDB, S3, SES)]

**AWS接続先情報**:
- リージョン: [region or "デフォルト設定"]
- リソース名: [table name, bucket name, queue URL, etc.]
- エンドポイント: [カスタムエンドポイントがあれば記載、なければ"デフォルト"]

**v1 → v2 変更内容**:
- クライアント初期化: [before] → [after]
- API呼び出し: [before] → [after]
- コンテキスト伝搬: [context propagation changes]
- その他の変更: [type changes, parameter changes, etc.]

**動作確認観点**:
- [AWSコンソールでの確認方法]
- [確認すべきAPIコールやログ]
- [注意すべき設定変更]

---

[Repeat for each AWS SDK usage location]
```

## Analysis Guidelines

- Focus on production AWS connections (exclude localhost endpoints)
- Trace complete call chain from application entry point
- Extract resource names from code (table names, bucket names, etc.)
- Identify region configuration (explicit or default)
- Summarize migration patterns clearly
- Provide actionable verification steps for AWS console

## Notes

- If PR is not AWS SDK Go related, stop immediately with clear message
- Group by file and function for clear organization
- Include full call chain for traceability
- Focus on connection-related changes (client init, endpoints, regions)
- Provide specific AWS console verification steps
