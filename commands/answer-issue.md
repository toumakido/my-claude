Efficiently answer question about GitHub issue $ARGUMENTS

Output language: Japanese, formal business tone

## Prerequisites

- gh CLI installed and authenticated
- $ARGUMENTS format: `<issue_number> <user_prompt>`
  - Example: `123 この問題の原因を調査して`
  - Example: `456 どのファイルが関連しているか教えて`

## Purpose

Efficiently investigate and answer questions about structured GitHub issues without necessarily implementing fixes. Optimized for issues created by `/structure-issue` or `/restructure-issue`.

## Process

1. Parse arguments: extract issue number and user prompt
2. Fetch issue: `gh issue view <issue_number> --json title,body,labels,comments`
3. Parse structured issue content:
   - Bug: 概要、再現手順、期待される動作、実際の動作、エラーメッセージ、環境情報
   - Feature: 概要、背景・目的、期待される動作
   - Enhancement: 概要、現状の問題点、期待される改善結果
4. Based on user prompt, determine investigation type:
   - Root cause analysis: "原因を調査"
   - Related files search: "関連ファイル", "どこを修正"
   - Implementation approach: "実装方法", "どうやって直す"
   - Impact assessment: "影響範囲", "副作用"
   - General questions: その他の質問
5. Use appropriate tools efficiently:
   - For file/code location: Task tool with subagent_type=Explore (NOT direct Grep/Glob)
   - For code reading: Read tool
   - For log analysis: Grep tool with specific patterns
6. Provide structured answer:
   ```
   ## 調査結果

   [direct answer to user prompt]

   ## 関連情報

   [supporting details, file locations with line numbers, code snippets]

   ## 推奨される次のステップ

   [actionable recommendations]
   ```
7. If fix is needed, suggest: `/work-on-issue <issue_number>`

## Efficiency Guidelines

- **DO NOT** implement fixes unless explicitly requested
- **DO** use Task tool for codebase exploration (avoid direct Grep/Glob for open-ended searches)
- **DO** provide file:line references for easy navigation
- **DO** focus on answering the specific prompt efficiently
- **DO NOT** run unnecessary tests or build commands
- **DO** read only relevant files (use context from structured issue)
- **DO** leverage structured issue format to narrow search scope

## Example Scenarios

### Scenario 1: Root cause investigation
```
User: /answer-issue 123 エラーの原因を特定して
→ Read error message from issue → Search for error pattern → Analyze code → Report cause
```

### Scenario 2: Related files search
```
User: /answer-issue 456 どのファイルを修正すればいい？
→ Parse feature request → Use Task (Explore) to find relevant files → Report with file:line
```

### Scenario 3: Impact assessment
```
User: /answer-issue 789 この修正の影響範囲は？
→ Identify proposed change → Search for dependencies → Report affected areas
```

## Output Format

Always structure response as:

```markdown
## Issue #<number>: <title>

### 質問
<user prompt>

### 調査結果
<direct answer with specific details>

### 関連箇所
- `file/path.go:123` - [description]
- `file/other.go:456` - [description]

### 推奨される次のステップ
1. [actionable step]
2. [if fix needed] `/work-on-issue <number>` を実行
```

## Notes

- Optimize for speed: don't over-investigate
- Trust structured issue format: it contains key information
- Use Task (Explore) for codebase searches, not direct tools
- Provide actionable, specific answers with code references
- Stay focused on user's prompt, don't diverge into implementation unless asked
