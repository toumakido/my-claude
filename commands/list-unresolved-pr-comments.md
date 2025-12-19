---
description: 指定したPRの未解決レビューコメントを一覧表示
args:
  pr_number:
    description: PR番号（省略時は現在のブランチに紐づくPR）
    required: false
---


指定されたPR {{pr_number}} の未解決レビューコメントを取得して**日本語で**表示してください。

## 処理手順

1. **PR番号とリポジトリ情報の取得**

   以下を並行実行:
   - PR番号: {{pr_number}} が指定されている場合はその番号、未指定の場合は `gh pr view --json number -q .number` で取得
   - リポジトリ情報: `gh pr view --json url -q .url` から owner/repo を抽出（例: `https://github.com/owner/repo/pull/123` → `owner`, `repo`）

   PR番号が取得できない場合はエラー終了。

2. **GraphQL クエリの実行**

   以下のコマンドで未解決コメントを取得（`owner`, `repo`, `pr_number` を手順1の値に置換）:
   ```bash
   gh api graphql -F pr={pr_number} -f query='
     query($pr: Int!, $cursor: String) {
       repository(owner: "{owner}", name: "{repo}") {
         pullRequest(number: $pr) {
           reviewThreads(first: 100, after: $cursor) {
             pageInfo {
               hasNextPage
               endCursor
             }
             nodes {
               isResolved
               isOutdated
               line
               path
               comments(first: 1) {
                 nodes {
                   databaseId
                   body
                   author { login }
                 }
               }
             }
           }
         }
       }
     }'
   ```

   ページネーション: `hasNextPage` が true の場合、`-F cursor={endCursor}` を追加して再実行。全ページ結合。

3. **結果の処理**

   - `isResolved: false` のスレッドのみ抽出
   - ソート: `jq` で `sort_by(.path, .line // 0)` を使用
   - 各スレッドの最初のコメントについて、本文をそのまま表示するのではなく、**内容を要約してわかりやすく説明する**

## 出力フォーマット

```
PR #{pr_number} の未解決コメント: {total}件

{path}:{line}
著者: {author.login} | Outdated: {isOutdated}
要約: {コメント本文の内容をわかりやすく要約した説明}
→ https://github.com/{owner}/{repo}/pull/{pr_number}#discussion_r{databaseId}

(繰り返し)
```

## エラーハンドリング

- gh CLI が利用不可の場合: "エラー: gh CLI がインストールされていないか、認証されていません。'gh auth login' を実行してください"
- PR番号が未指定かつ現在のブランチにPRがない場合: "エラー: 現在のブランチにPRが見つかりません。PR番号を指定してください"
- PR が見つからない場合: "エラー: PR #{pr_number} が見つかりません"
- 未解決コメントが0件の場合: "✓ PR #{pr_number} のコメントはすべて解決済みです"
- GraphQL APIエラーの場合: エラーメッセージを日本語で表示

## 注意事項

- パスが null の場合は "全般的なコメント" と表示
- 行番号が null の場合は行番号を省略
- すべての出力は日本語で行う
