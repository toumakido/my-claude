Create clear GitHub issue from user prompt: $ARGUMENTS

Output language: Japanese, formal business tone

## Process

1. Analyze prompt, determine type (bug/feature/enhancement)
2. Check duplicates: `gh issue list --search "$KEYWORDS"`
3. Use AskUserQuestion for minimal required info
4. Create title format: `[Bug] ログイン時に無限ループが発生` or `[Feature] ダークモード対応`
5. Create body using templates below
6. Show to user for confirmation
7. Create with `gh issue create`

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

## 環境情報
- OS: [e.g. macOS 14.0]
```

Feature:
```
## 概要
[brief description]

## 背景・目的
[why needed]

## 期待される動作
[how it should work]
```

Enhancement:
```
## 概要
[brief description]

## 現状の問題点
[current issue]

## 期待される改善結果
[expected improvement]
```

## Labels

Type: bug, enhancement, feature
Priority: priority:high, priority:medium, priority:low

## Follow-up

Show issue number/URL and suggest `/work-on-issue [number]`

Focus on problem definition only, not implementation details
