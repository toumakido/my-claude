Restructure existing GitHub issue $ARGUMENTS to clarify problem definition

Output language: Japanese, formal business tone

## Purpose

Transform vague/incomplete issues into format that `/fix-github-issue` can work with effectively. Focus on problem clarity, not implementation.

## Process

1. Fetch: `gh issue view $ARGUMENTS --json title,body,labels,comments`
2. Check completeness: problem definition, reproduction steps (bug), expected/actual behavior (bug), background/purpose (feature), environment info
3. Use AskUserQuestion for missing info
4. Create structured body using templates below
5. Improve title: `バグがある` → `[Bug] ログイン時に認証エラー`, `機能追加` → `[Feature] ダークモード対応`
6. Optimize labels: bug/enhancement/feature, priority:high/medium/low
7. Show to user for confirmation
8. Update issue:
```bash
gh issue edit $ARGUMENTS --title "new title"
gh issue edit $ARGUMENTS --body "$(cat <<'EOF'
[new body]
EOF
)"
gh issue edit $ARGUMENTS --add-label "label1,label2"
```
9. Show issue URL and suggest `/fix-github-issue $ARGUMENTS`

## Templates

Bug:
```
## 概要
[brief description]

## 再現手順
1. [step]

## 期待される動作
[expected]

## 実際の動作
[actual]

## エラーメッセージ
[error log]

## 環境情報
- OS: [e.g. macOS 14.0]

---
<details>
<summary>元の issue 内容</summary>
[original]
</details>
```

Feature:
```
## 概要
[brief]

## 背景・目的
[why needed]

## 期待される動作
[how it should work]

---
<details>
<summary>元の issue 内容</summary>
[original]
</details>
```

Enhancement:
```
## 概要
[brief]

## 現状の問題点
[current issue]

## 期待される改善結果
[expected improvement]

---
<details>
<summary>元の issue 内容</summary>
[original]
</details>
```

## Notes

- Focus on problem definition, not implementation
- Preserve original content in collapsed section
- Respect original author's intent
- For issues under discussion, suggest changes in comment first
