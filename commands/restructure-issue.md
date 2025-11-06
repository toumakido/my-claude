既存の GitHub issue $ARGUMENTS を読み取り、問題定義を明確化してください。

## 目的
曖昧または不完全な issue を、`/fix-github-issue` が効果的に動作する形式に変換する。実装の詳細は fix-github-issue が担当するため、問題の明確化に集中する。

## 実行手順

### 1. 既存 Issue の取得
- `gh issue view $ARGUMENTS --json title,body,labels,comments` で取得
- タイトル、本文、コメントを確認
- issue が存在しない場合はエラー終了

### 2. 不足要素の特定
以下が明確に記述されているかチェック:
- **問題定義**: 何が問題か
- **再現手順**: ステップバイステップ（バグの場合）
- **期待/実際の動作**: 両方が明確か（バグの場合）
- **背景・目的**: なぜ必要か（機能要望の場合）
- **環境情報**: 該当する場合

コメントも確認して追加情報を収集

### 3. 不足情報の質問
**AskUserQuestion で不足情報を質問**:
- 再現手順の詳細
- 期待される動作
- 環境情報
- エラーメッセージ
- 背景や目的

### 4. 構造化された本文の作成

#### バグ報告:
```markdown
## 概要
[問題の簡潔な説明]

## 再現手順
1. [ステップ1]
2. [ステップ2]
3. [ステップ3]

## 期待される動作
[何が起こるべきか]

## 実際の動作
[実際に何が起こるか]

## エラーメッセージ
```
[エラーログ]
```

## 環境情報
- OS: [例: macOS 14.0]
- ブラウザ/Node.js: [該当するもの]
- バージョン: [v1.2.3]

---
<details>
<summary>元の issue 内容</summary>

[元の本文]
</details>
```

#### 機能要望:
```markdown
## 概要
[機能の簡潔な説明]

## 背景・目的
[なぜこの機能が必要か、どんな問題を解決するか]

## 期待される動作
[この機能がどう動作すべきか]

## 補足情報
[UI/UX の期待、参考例など]

---
<details>
<summary>元の issue 内容</summary>

[元の本文]
</details>
```

#### 改善提案:
```markdown
## 概要
[改善内容の簡潔な説明]

## 現状の問題点
[現在何が問題か、なぜ改善が必要か]

## 期待される改善結果
[改善後どうなるべきか]

---
<details>
<summary>元の issue 内容</summary>

[元の本文]
</details>
```

### 5. タイトルの改善
曖昧なタイトルを明確化:
- Before: `バグがある` → After: `[Bug] ログイン時に認証エラー`
- Before: `機能追加` → After: `[Feature] ダークモード対応`

### 6. ラベルの最適化
基本的なラベルのみ:
- 種類: `bug`, `enhancement`, `feature`
- 優先度: `priority:high`, `priority:medium`, `priority:low`

### 7. ユーザー確認
再構成内容をユーザーに提示して確認

### 8. Issue の更新
```bash
gh issue edit $ARGUMENTS --title "新しいタイトル"
gh issue edit $ARGUMENTS --body "$(cat <<'EOF'
[新しい本文]
EOF
)"
gh issue edit $ARGUMENTS --add-label "label1,label2"
```

### 9. フォローアップ
- issue URL を提示
- `/fix-github-issue $ARGUMENTS` で修正を開始できることを案内

## 注意事項
- **問題定義に集中**: 実装方法や技術的詳細は含めない
- **元の内容を保持**: 折りたたみセクションで必ず保存
- **意図を尊重**: 元の作成者の意図を改変しない
- 議論中の issue は、コメントで提案してから更新
