# my-claude

個人用 Claude Code 設定

## 構成

- `CLAUDE.md` - Claude Code全体のルール設定（約45行に最適化）
- `skills/` - カスタムスキル（メインの拡張機能）
- `agents/` - カスタムエージェント定義
- `commands/` - レガシーコマンド（aws-sdk-go-v2関連のみ）

### skills/ (推奨)

GitHub issue/PR ワークフロー用のカスタムスキル:
- `structure-issue` - GitHub issue作成
- `restructure-issue` - 既存issue整理
- `answer-issue` - issue調査・回答
- `work-on-issue` - issue対応（実装・PR作成）
- `review-pr` - PR一次レビュー
- `verify-pr` - PR動作確認手順生成
- `impl-review` - PRレビューコメント実装
- `list-unresolved-pr-comments` - PR未解決コメント一覧
- `improve-command` - スキル改善提案作成（手動呼び出し専用）
- `optimize-ai-efficiency` - スキル最適化PR作成（手動呼び出し専用）
- `pr-creator` - PR作成ガイド

### commands/ (レガシー)

後方互換性のため保持。新しいワークフローは `skills/` に作成してください。

現在保持されているコマンド:
- `aws-sdk-go-v2/migrate` - AWS SDK v1→v2移行
- `aws-sdk-go-v2/extract-chains` - SDK呼び出しチェーン抽出
- `aws-sdk-go-v2/prepare-tests` - テストコード準備

## シンボリックリンク設定

このリポジトリの以下のファイル/ディレクトリは `~/.claude/` とシンボリックリンクで同期されています：

- `./CLAUDE.md` ↔ `~/.claude/CLAUDE.md`
- `./skills/` ↔ `~/.claude/skills/`
- `./agents/` ↔ `~/.claude/agents/`
- `./commands/` ↔ `~/.claude/commands/` (aws-sdk-go-v2用)

**重要**: Claude Code に関する設定変更を依頼する場合は、必ずこのリポジトリ内のファイル（`./CLAUDE.md`, `./skills/`, `./agents/`, `./commands/`）を操作してください。`~/.claude/` 配下を直接操作しないでください。

## 備考

- [公式ドキュメント](https://code.claude.com/docs/en/skills)によると、commandsはskillsに統合されました
- 同名のskillとcommandがある場合、skillが優先されます
- 既存のcommands/は後方互換性のため動作を継続しますが、新規作成はskills/を使用してください
