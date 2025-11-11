# my-claude

個人用 Claude Code 設定

## 構成

- `CLAUDE.md` - Claude Code全体のルール設定
- `commands/` - カスタムコマンド
- `sample/` - サンプルコード

## シンボリックリンク設定

このリポジトリの以下のファイル/ディレクトリは `~/.claude/` とシンボリックリンクで同期されています：

- `./CLAUDE.md` ↔ `~/.claude/CLAUDE.md`
- `./commands/` ↔ `~/.claude/commands/`

**重要**: Claude Code に関する設定変更を依頼する場合は、必ずこのリポジトリ内のファイル（`./CLAUDE.md`, `./commands/`）を操作してください。`~/.claude/` 配下を直接操作しないでください。
