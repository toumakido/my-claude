# my-claude

個人用 Claude Code 設定

## 構成

- `CLAUDE.md` - Claude Code全体のルール設定
- `commands/` - カスタムコマンド
- `agents/` - カスタムエージェント定義
- `sample/` - サンプルコード

### commands/ ディレクトリの設計原則

このディレクトリのマークダウンファイルはClaude CodeがAIとして読み取り実行するためのものです。編集時は以下を優先してください：

- **AI判断の効率性 > 人間の可読性**
- ルールベース構造（if X then Y、パターンマッチング）
- 判断フローを順序付きルールで明確化
- 絵文字不使用（代わりにテキストマーカー: Correct:/Wrong:）

## シンボリックリンク設定

このリポジトリの以下のファイル/ディレクトリは `~/.claude/` とシンボリックリンクで同期されています：

- `./CLAUDE.md` ↔ `~/.claude/CLAUDE.md`
- `./commands/` ↔ `~/.claude/commands/`
- `./agents/` ↔ `~/.claude/agents/`

**重要**: Claude Code に関する設定変更を依頼する場合は、必ずこのリポジトリ内のファイル（`./CLAUDE.md`, `./commands/`, `./agents/`）を操作してください。`~/.claude/` 配下を直接操作しないでください。
