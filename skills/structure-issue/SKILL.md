---
name: structure-issue
description: Create clear GitHub issue from user prompt
---

Create clear GitHub issue from user prompt: $ARGUMENTS

## Process

1. Analyze prompt, determine type (bug/feature/enhancement)
2. (Feature/Enhancement only) If prompt mentions existing tables, APIs, or models: search codebase for relevant schema/struct definitions and use findings to inform question selection and draft accuracy
3. Use AskUserQuestion for minimal required info
4. Create title format: `[Bug] ログイン時に無限ループが発生` or `[Feature] ダークモード対応`
5. Create body using templates below
6. Show to user for confirmation
7. Create with `gh issue create`

## AskUserQuestion Guidelines

- For features adding new optional parameters: ask about default values
  - Example: 「新パラメータのデフォルト値はありますか？（例: true / false / null）」

## Templates

- Bug: 概要, 再現手順, 期待される動作, 実際の動作, 環境情報
- Feature: 概要, 背景・目的, 期待される動作
  - 期待される動作には「何ができるようになるか（ユーザー・オペレーター視点）」を記載する
  - APIパラメータ名・型・内部処理フローなど実装詳細は含めない
- Enhancement: 概要, 現状の問題点, 期待される改善結果

## Follow-up

Show issue number/URL and suggest `/work-on-issue [number]`

Focus on problem definition only, not implementation details
