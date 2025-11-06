# Claude Code プロジェクトルール

このファイルは Claude Code がコードを生成・編集する際に従うべきプロジェクト固有のルールを定義します。

## Go 言語開発のルール

### Import 文の扱い

**重要**: Go の import 文を追加する際は、必ず以下のルールに従うこと：

1. **Import 文だけを先に書かない**
   - エディタの自動補完により、使用されていない import が削除されるため無駄になる
   - Import を追加する場合は、そのパッケージを実際に使用するコードまで一気に実装する

2. **段階的実装の正しい手順**
   ```
   ❌ 悪い例:
   1. import 文を追加
   2. 保存（未使用のimportが削除される）
   3. 実際にパッケージを使うコードを書く
   4. 再度 import を追加...（無駄な繰り返し）

   ✅ 良い例:
   1. 必要なパッケージを特定
   2. Import 文とそれを使用するコード実装を同じ Edit で行う
   3. 保存時には既に使用されているため削除されない
   ```

3. **具体例**
   ```go
   // ❌ 悪い実装順序
   // Step 1: import だけ追加
   import (
       "fmt"
       "encoding/json"  // この時点で使われていない
   )
   // → 保存時に encoding/json が削除される

   // ✅ 良い実装順序
   // import と使用コードを同時に追加
   import (
       "fmt"
       "encoding/json"
   )

   func processData(data interface{}) error {
       bytes, err := json.Marshal(data)  // 同じ Edit で実装
       if err != nil {
           return err
       }
       fmt.Println(string(bytes))
       return nil
   }
   ```

### パッケージの存在確認

**重要**: 存在しないパッケージを使用しないための手順：

1. **標準ライブラリの確認**
   - 不確かな場合は、`go doc <package>` で存在を確認
   - 例: `go doc encoding/json`

2. **サードパーティパッケージの確認**
   - `go.mod` ファイルを必ず確認
   - ファイル内に記載されているパッケージのみ使用可能
   - 新しいパッケージが必要な場合:
     1. ユーザーに確認を取る
     2. `go get <package>` でインストール
     3. `go.mod` が更新されたことを確認
     4. その後にコードで使用

3. **パッケージ使用前のチェックリスト**
   - [ ] 標準ライブラリか？ → `go doc` で確認
   - [ ] サードパーティか？ → `go.mod` に存在するか確認
   - [ ] 新規追加が必要か？ → ユーザーに確認後、`go get` でインストール

4. **よくある間違い**
   ```go
   // ❌ 存在しないパッケージ
   import "github.com/user/nonexistent"  // go.mod に存在しない

   // ✅ 使用前に確認
   // 1. go.mod を Read ツールで確認
   // 2. 存在しない場合はユーザーに質問
   // 3. go get でインストール後に使用
   ```

### Go コーディングのベストプラクティス

1. **エラーハンドリング**
   - すべての error を適切に処理する
   - `if err != nil` を省略しない

2. **nil チェック**
   - ポインタや interface を使用する前に nil チェックを行う

3. **defer の活用**
   - リソースのクリーンアップには defer を使用
   - ファイル、接続、ロックなど

4. **go fmt 準拠**
   - Go の標準フォーマットに従う
   - インデントはタブ文字

5. **命名規則**
   - パッケージ名: 小文字、短く、簡潔
   - エクスポート: PascalCase (大文字始まり)
   - プライベート: camelCase (小文字始まり)

## コード編集の一般ルール

### Edit ツールの効果的な使用

1. **原子的な変更**
   - 関連する変更は1つの Edit にまとめる
   - Import とその使用コードは同時に追加

2. **段階的な実装を避ける場合**
   - Import 追加時
   - 相互に依存する関数の実装時
   - 型定義とそのメソッド実装時

### ファイル操作

1. **Read before Edit**
   - 常にファイルを読んでから編集する
   - 既存のコードスタイルに合わせる

2. **Write vs Edit**
   - 既存ファイルは必ず Edit を使用
   - Write は新規ファイルのみ

## プロジェクト固有の設定

### ディレクトリ構造
```
.
├── cmd/           # メインアプリケーション
├── pkg/           # ライブラリコード
├── internal/      # プライベートコード
├── api/           # API定義
├── configs/       # 設定ファイル
└── scripts/       # ビルドスクリプト
```

### テスト

1. **テストファイルの命名**
   - `*_test.go` の形式

2. **テストの実行**
   - `go test ./...` で全テスト実行
   - `go test -v ./...` で詳細出力

3. **テーブル駆動テスト**
   - 複数のテストケースは構造体スライスで定義

### ビルドとデプロイ

1. **ビルドコマンド**
   - `go build -o bin/app cmd/app/main.go`

2. **依存関係の管理**
   - `go mod tidy` で整理
   - `go mod verify` で検証

## トラブルシューティング

### Import が削除される問題
- **原因**: 使用されていない import が自動削除される
- **解決**: Import とその使用コードを同時に実装

### パッケージが見つからない
- **原因**: `go.mod` に存在しないパッケージを使用
- **解決**: `go.mod` を確認し、必要なら `go get` でインストール

### ビルドエラー
- **確認事項**:
  1. `go mod tidy` を実行
  2. すべての import が正しいか確認
  3. `go build` でエラー詳細を確認

## 参考リソース

- [Effective Go](https://go.dev/doc/effective_go)
- [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- [Standard Package Layout](https://github.com/golang-standards/project-layout)
