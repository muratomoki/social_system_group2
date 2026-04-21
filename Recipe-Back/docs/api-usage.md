# Recipe-Back API 使用ガイド

## セットアップ

### 1. 環境変数の設定

`.env.example` をコピーして `.env` を作成し、値を設定します。

```bash
cp .env.example .env
```

必須の環境変数:

| 変数名 | 説明 |
|---|---|
| `ANTHROPIC_API_KEY` | Anthropic APIキー（[console.anthropic.com](https://console.anthropic.com)から取得） |
| `FRIDGE_API_BASE_URL` | 冷蔵庫在庫管理システムのベースURL |

任意の環境変数:

| 変数名 | デフォルト値 | 説明 |
|---|---|---|
| `PORT` | `8080` | HTTPサーバーのポート番号 |
| `USER_PREFS_FILE_PATH` | `./data/preferences.json` | ユーザー設定の保存先 |
| `CLAUDE_MODEL` | `claude-opus-4-7` | 使用するClaudeモデル |
| `AI_CACHE_ENABLED` | `true` | AIレスポンスのインメモリキャッシュ（TTL: 10分） |

### 2. ビルドと起動

```bash
cd Recipe-Back

# 依存関係の解決
go mod tidy

# サーバー起動
go run src/main.go

# または バイナリビルド後に起動
go build -o recipe-back ./src/...
./recipe-back
```

> **冷蔵庫システム未接続の場合**: `FRIDGE_API_BASE_URL` に接続できなくても、レシピ提案は動作します。冷蔵庫の食材なしで、常備調味料と個人の趣向のみをもとにAIが提案します。

---

## エンドポイント一覧

| Method | Path | 説明 |
|---|---|---|
| `GET` | `/api/v1/health` | ヘルスチェック |
| `POST` | `/api/v1/recipes/suggest` | レシピ提案 |
| `GET` | `/api/v1/preferences` | ユーザー設定取得 |
| `PUT` | `/api/v1/preferences` | ユーザー設定更新 |

---

## GET /api/v1/health

サーバーの稼働確認。

```bash
curl http://localhost:8080/api/v1/health
```

**レスポンス 200:**
```json
{"status": "ok"}
```

---

## POST /api/v1/recipes/suggest

冷蔵庫の食材・常備調味料・個人の趣向をもとに、AIがレシピを3件提案します。

### リクエスト

```bash
# 追加リクエストなし
curl -X POST http://localhost:8080/api/v1/recipes/suggest \
  -H "Content-Type: application/json" \
  -d '{}'

# 追加リクエストあり
curl -X POST http://localhost:8080/api/v1/recipes/suggest \
  -H "Content-Type: application/json" \
  -d '{"additionalNotes": "30分以内で作れるもの。辛い料理は避けたい。"}'
```

| フィールド | 型 | 必須 | 最大長 | 説明 |
|---|---|---|---|---|
| `additionalNotes` | string | 任意 | 1000文字 | 追加の要望・制約 |

### レスポンス 200

```json
{
  "recipes": [
    {
      "name": "鶏むね肉と小松菜の和風炒め",
      "url": "https://cookpad.com/recipe/1234567",
      "description": "冷蔵庫の食材で手軽に作れる和風炒め物。ご飯が進む一品。",
      "matchScore": "高",
      "ingredients": [
        { "name": "鶏むね肉", "amount": "200g", "isInFridge": true },
        { "name": "小松菜", "amount": "1束", "isInFridge": true },
        { "name": "醤油", "amount": "大さじ2", "isInFridge": false },
        { "name": "みりん", "amount": "大さじ1", "isInFridge": false }
      ],
      "steps": [
        { "order": 1, "description": "鶏むね肉を一口大に切る" },
        { "order": 2, "description": "フライパンに油を熱し、鶏肉を中火で炒める" },
        { "order": 3, "description": "小松菜を加えてさらに炒め、醤油・みりんで味付けする" }
      ]
    }
  ]
}
```

| フィールド | 説明 |
|---|---|
| `matchScore` | 冷蔵庫食材との一致度: `"高"` / `"中"` / `"低"` |
| `isInFridge` | `true` = 冷蔵庫にある食材 |

### エラーレスポンス

**400 BAD_REQUEST** — リクエストボディが不正なJSON:
```json
{ "error": "Request body is invalid JSON", "code": "BAD_REQUEST" }
```

**422 VALIDATION_ERROR** — バリデーション失敗:
```json
{
  "error": "Validation failed",
  "code": "VALIDATION_ERROR",
  "details": "additionalNotes exceeds maximum length of 1000 characters"
}
```

**502 AI_ERROR** — AI APIエラー:
```json
{ "error": "Failed to get recipe suggestions from AI", "code": "AI_ERROR", "details": "..." }
```

---

## GET /api/v1/preferences

現在のユーザー設定（常備調味料・個人の趣向）を取得します。

```bash
curl http://localhost:8080/api/v1/preferences
```

**レスポンス 200:**
```json
{
  "condiments": ["醤油", "みりん", "砂糖", "塩", "胡椒", "ごま油"],
  "personalNotes": "魚料理が好き。辛い食べ物は苦手。週3回以上料理する。"
}
```

---

## PUT /api/v1/preferences

ユーザー設定を更新します。設定はサーバー上のJSONファイルに永続化されます。

```bash
curl -X PUT http://localhost:8080/api/v1/preferences \
  -H "Content-Type: application/json" \
  -d '{
    "condiments": ["醤油", "みりん", "砂糖", "塩", "胡椒", "ごま油", "酢", "酒"],
    "personalNotes": "魚料理が好き。辛い食べ物は苦手。できるだけヘルシーなレシピを希望。"
  }'
```

| フィールド | 型 | 必須 | 制約 | 説明 |
|---|---|---|---|---|
| `condiments` | string[] | 任意 | 最大50件、各50文字以内 | 常備調味料リスト |
| `personalNotes` | string | 任意 | 最大2000文字 | 個人の趣向・好み・制約 |

**レスポンス 200:** 更新後の設定をそのまま返す（GET と同じ形式）

**422 VALIDATION_ERROR:**
```json
{
  "error": "Validation failed",
  "code": "VALIDATION_ERROR",
  "details": "condiments: max 50 items allowed"
}
```

---

## エラーコード一覧

| HTTPステータス | コード | 発生条件 |
|---|---|---|
| 400 | `BAD_REQUEST` | リクエストボディが不正なJSON |
| 422 | `VALIDATION_ERROR` | バリデーションルール違反 |
| 500 | `INTERNAL_ERROR` | サーバー内部エラー |
| 502 | `AI_ERROR` | Anthropic APIエラー |

---

## 冷蔵庫システム連携

冷蔵庫在庫管理システムが稼働していない場合、Recipe-Backはエラーにならず空の食材リストでレシピを提案します。

冷蔵庫システムが実装された際は、以下のエンドポイントを提供してください:

```
GET {FRIDGE_API_BASE_URL}/inventory
```

**期待するレスポンス形式:**
```json
{
  "items": [
    {
      "name": "鶏むね肉",
      "amount": "300g",
      "expiryDate": "2026-04-25T00:00:00Z",
      "category": "肉"
    }
  ],
  "fetchedAt": "2026-04-21T10:00:00Z"
}
```
