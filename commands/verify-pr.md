Analyze PR implementation and provide verification instructions for $ARGUMENTS

Output language: Japanese, formal business tone

## Prerequisites

- gh CLI installed and authenticated
- $ARGUMENTS: PR number
- Run from repo root

## Process

1. Fetch PR diff: `gh pr diff $ARGUMENTS`
2. Use Task tool (subagent_type=Plan) to analyze:
   - Added/modified route definitions (e.g., router.GET, router.POST, http.HandleFunc)
   - Handler implementations
   - Authentication/middleware usage (JWT, API keys, session)
   - Environment variable references (os.Getenv, config files)
   - Database operations (migrations, seeds)
   - External service dependencies (Redis, S3, etc.)
3. Generate verification instructions with:
   - Prerequisites (server startup, env vars, DB setup, test data)
   - curl examples for each endpoint (method, URL, headers, body)
   - Authentication details (token acquisition, header format)
   - Expected responses (status codes, response body examples)
4. Output structured verification guide

## Output Format

```markdown
## 動作確認方法

### 前提条件
- [server startup command]
- [environment variables]
- [database initialization]
- [test data setup]

### APIエンドポイント確認

#### [METHOD] [PATH] - [Description]
```bash
curl -X [METHOD] [URL] \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer ${TOKEN}" \
  -d '[REQUEST_BODY]'
```

**期待されるレスポンス**:
- Status: [CODE]
- Body: `[RESPONSE_EXAMPLE]`

[Repeat for each endpoint]
```

## Analysis Guidelines

- Focus on user-facing changes (new/modified endpoints)
- Extract authentication requirements from middleware/handlers
- Identify environment-dependent configuration
- Include realistic request/response examples
- Note any breaking changes or migration requirements

## Notes

- If no API changes detected, report: "このPRにはAPI変更が含まれていません"
- For non-API changes (refactoring, internal), suggest relevant verification method
- Include warnings for breaking changes or required migrations
