# Sample Todo API Server

Go 言語で実装したシンプルな TODO 管理 Web サーバーのサンプルアプリケーションです。

## 特徴

- RESTful API による CRUD 操作
- インメモリデータストア（データベース不要）
- 標準ライブラリのみで実装
- ローカル環境で簡単に動作確認可能

## ディレクトリ構造

```
sample/
├── cmd/
│   └── server/
│       └── main.go          # エントリーポイント
├── internal/
│   ├── model/
│   │   └── todo.go          # データモデル
│   ├── store/
│   │   └── memory.go        # インメモリストア
│   └── handler/
│       └── todo.go          # HTTP ハンドラー
├── go.mod
└── README.md
```

## 実行方法

### 1. サーバーの起動

```bash
cd sample
go run cmd/server/main.go
```

サーバーが起動すると、以下のメッセージが表示されます：

```
Server starting on http://localhost:8080
Try: curl http://localhost:8080/
```

### 2. ビルドして実行

```bash
cd sample
go build -o bin/server cmd/server/main.go
./bin/server
```

## API エンドポイント

### ルートエンドポイント

```bash
GET /
```

API の情報を取得します。

```bash
curl http://localhost:8080/
```

### TODO 一覧の取得

```bash
GET /todos
```

すべての TODO を取得します。

```bash
curl http://localhost:8080/todos
```

### TODO の作成

```bash
POST /todos
Content-Type: application/json

{
  "title": "タスクのタイトル"
}
```

新しい TODO を作成します。

```bash
curl -X POST http://localhost:8080/todos \
  -H "Content-Type: application/json" \
  -d '{"title":"買い物に行く"}'
```

### 特定の TODO の取得

```bash
GET /todos/:id
```

ID を指定して TODO を取得します。

```bash
curl http://localhost:8080/todos/1
```

### TODO の更新

```bash
PUT /todos/:id
Content-Type: application/json

{
  "title": "更新後のタイトル",
  "completed": true
}
```

既存の TODO を更新します。

```bash
curl -X PUT http://localhost:8080/todos/1 \
  -H "Content-Type: application/json" \
  -d '{"title":"買い物に行く（完了）","completed":true}'
```

### TODO の削除

```bash
DELETE /todos/:id
```

TODO を削除します。

```bash
curl -X DELETE http://localhost:8080/todos/1
```

## 使用例

以下は一連の操作例です：

```bash
# 1. サーバーを起動（別のターミナルで）
cd sample
go run cmd/server/main.go

# 2. TODO を作成
curl -X POST http://localhost:8080/todos \
  -H "Content-Type: application/json" \
  -d '{"title":"Go の勉強"}'

curl -X POST http://localhost:8080/todos \
  -H "Content-Type: application/json" \
  -d '{"title":"買い物"}'

# 3. TODO 一覧を取得
curl http://localhost:8080/todos

# 4. TODO を完了にする
curl -X PUT http://localhost:8080/todos/1 \
  -H "Content-Type: application/json" \
  -d '{"title":"Go の勉強","completed":true}'

# 5. TODO を削除
curl -X DELETE http://localhost:8080/todos/2

# 6. 残りの TODO を確認
curl http://localhost:8080/todos
```

## レスポンス形式

### 成功時

```json
{
  "id": 1,
  "title": "タスクのタイトル",
  "completed": false,
  "created_at": "2025-11-06T18:00:00Z",
  "updated_at": "2025-11-06T18:00:00Z"
}
```

### エラー時

```json
{
  "error": "エラーメッセージ"
}
```

## ステータスコード

- `200 OK` - 成功（取得、更新）
- `201 Created` - 作成成功
- `204 No Content` - 削除成功
- `400 Bad Request` - リクエストが不正
- `404 Not Found` - リソースが見つからない
- `500 Internal Server Error` - サーバーエラー

## 技術スタック

- Go 1.x
- 標準ライブラリ:
  - `net/http` - HTTP サーバー
  - `encoding/json` - JSON エンコード/デコード
  - `sync` - 並行処理の同期

## 注意事項

- データはメモリ上にのみ保存されるため、サーバーを停止するとデータは失われます
- 本番環境での使用は想定していません
- 学習・開発用のサンプル実装です
