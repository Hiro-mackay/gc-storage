# OpenAPI型生成パイプライン仕様書

## メタ情報

| 項目 | 値 |
|------|-----|
| ステータス | Ready |
| 優先度 | High |
| 関連ドメイン | 全ドメイン（横断） |
| 依存する仕様 | infra-api |

---

## 1. 概要

バックエンドGoハンドラーのSwaggerアノテーションからOpenAPI仕様書を自動生成し、フロントエンドのTypeScript型定義を自動生成するパイプラインを定義します。

**目的:** バックエンドAPIとフロントエンド間のE2E型安全性を確保する。

---

## 2. パイプラインフロー

```
Go Handler Annotations
        │
        ▼
  swag init (swaggo/swag)
        │
        ▼
  swagger.json (backend/docs/)   ← Swagger 2.0
        │
        ▼
  swagger2openapi
        │
        ▼
  openapi.json (frontend/src/lib/api/)  ← OpenAPI 3.0
        │
        ▼
  openapi-typescript
        │
        ▼
  schema.d.ts (frontend/src/lib/api/)
        │
        ▼
  openapi-fetch client
```

---

## 3. ツール構成

### 3.1 バックエンド: swaggo/swag

| 項目 | 値 |
|------|-----|
| パッケージ | `github.com/swaggo/swag` |
| バージョン | v1.16.x |
| 役割 | Goコメントアノテーション → OpenAPI 2.0 (Swagger) JSON |

**生成コマンド:**

```bash
swag init -g cmd/api/main.go -o docs --parseDependency --parseInternal
```

**出力ファイル:**

| ファイル | 説明 |
|---------|------|
| `backend/docs/swagger.json` | OpenAPI仕様（JSON） |
| `backend/docs/swagger.yaml` | OpenAPI仕様（YAML） |
| `backend/docs/docs.go` | Goパッケージ（echo-swaggerで利用） |

### 3.2 フロントエンド: swagger2openapi

| 項目 | 値 |
|------|-----|
| パッケージ | `swagger2openapi` |
| 役割 | Swagger 2.0 → OpenAPI 3.0 変換 |

**変換コマンド:**

```bash
pnpm exec swagger2openapi ../backend/docs/swagger.json -o src/lib/api/openapi.json
```

### 3.3 フロントエンド: openapi-typescript

| 項目 | 値 |
|------|-----|
| パッケージ | `openapi-typescript` |
| バージョン | 7.x |
| 役割 | OpenAPI JSON → TypeScript型定義 |

**生成コマンド:**

```bash
pnpm exec openapi-typescript src/lib/api/openapi.json -o src/lib/api/schema.d.ts
```

### 3.4 フロントエンド: openapi-fetch

| 項目 | 値 |
|------|-----|
| パッケージ | `openapi-fetch` |
| バージョン | 0.x |
| 役割 | 生成された型を使った型安全なfetch wrapper |

---

## 4. レスポンスエンベロープ仕様

### 4.1 問題

バックエンドの `presenter.Response` は `Data` と `Meta` フィールドが `interface{}` 型であり、swaggoは `interface{}` から具体的なOpenAPIスキーマを生成できない。

### 4.2 解決策

`handler/swagger_models.go` にSwagger専用のラッパー構造体を定義し、`interface{}` を具体型に置き換える。

```go
// handler/swagger_models.go
type SwaggerLoginResponse struct {
    Data response.LoginResponse `json:"data"`
    Meta *presenter.Meta        `json:"meta"`
}
```

ハンドラーのアノテーションでこのラッパー型を参照:

```go
// @Success 200 {object} handler.SwaggerLoginResponse
```

---

## 5. APIクライアント

### 5.1 クライアント設定

```typescript
// frontend/src/lib/api/client.ts
import createClient from "openapi-fetch";
import type { paths } from "./schema";

export const api = createClient<paths>({
  baseUrl: "/api/v1",
  credentials: "include", // Cookie自動送信
});
```

### 5.2 使用例

```typescript
// 型安全なAPI呼び出し
const { data, error } = await api.POST("/auth/login", {
  body: { email: "user@example.com", password: "password" },
});
// data は LoginResponse | undefined として型推論される
```

---

## 6. Taskfileコマンド

| コマンド | 説明 |
|---------|------|
| `task backend:swagger` | swagger.json を再生成 |
| `task frontend:generate-types` | openapi.json + schema.d.ts を再生成 |
| `task api:generate` | swagger.json + schema.d.ts を一括再生成 |

---

## 7. 受け入れ基準

- [ ] `task backend:swagger` がエラーなく実行される
- [ ] `task frontend:generate-types` で `schema.d.ts` が生成される
- [ ] `api.POST("/auth/login", { body: { email, password } })` で型推論が効く
- [ ] `api.GET("/folders/{id}/contents", { params: { path: { id } } })` で型推論が効く
- [ ] `task backend:test` がPASS
- [ ] `task frontend:build` がPASS
