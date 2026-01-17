# GC Storage API設計書

## 概要

本ドキュメントでは、GC StorageのRESTful APIに関する**設計原則、ポリシー、スケーリング戦略**について説明します。個別のエンドポイント仕様はOpenAPI定義ファイルを参照してください。

---

## 1. 設計原則

### 1.1 基本方針

| 原則 | 説明 |
|------|------|
| RESTful | リソース指向のURL設計 |
| JSON | リクエスト/レスポンスボディはJSON形式 |
| ステートレス | サーバー側でセッション状態を保持しない |
| 冪等性 | PUT/DELETEは冪等に設計 |
| 一貫性 | 命名規則、エラーレスポンスの統一 |

### 1.2 URL設計規則

**構造:**
```
/{version}/{resource}/{id}/{sub-resource}
```

**ルール:**

| ルール | 例 |
|--------|-----|
| リソース名は複数形 | `/users`, `/files`, `/folders` |
| 小文字ケバブケース | `/share-links`, `/audit-logs` |
| 動詞ではなく名詞 | `GET /files` (× `GET /getFiles`) |
| ネストは2階層まで | `/groups/{id}/members` |
| アクションはサブパス | `/files/{id}/download`, `/files/{id}/move` |

### 1.3 HTTPメソッド使用規則

| メソッド | 用途 | 冪等性 | ボディ |
|---------|------|--------|--------|
| GET | リソース取得 | Yes | No |
| POST | リソース作成、アクション実行 | No | Yes |
| PUT | リソース全体更新 | Yes | Yes |
| PATCH | リソース部分更新 | No | Yes |
| DELETE | リソース削除 | Yes | Optional |

**使い分け指針:**
- 新規作成 → POST
- 全体置換 → PUT
- 一部変更 → PATCH
- カスタムアクション → POST `/resource/{id}/action`

---

## 2. バージョニング戦略

### 2.1 方式

URLパスにバージョンを含める方式を採用:

```
/api/v1/files
/api/v2/files
```

**理由:**
- キャッシュとの相性が良い
- デバッグ時に明示的
- ロードバランサーでの振り分けが容易

### 2.2 バージョンアップ方針

| 変更種別 | バージョン対応 |
|---------|---------------|
| 後方互換（フィールド追加等） | 同一バージョン内 |
| 破壊的変更（フィールド削除、型変更等） | メジャーバージョンアップ |

### 2.3 非推奨化フロー

1. 新バージョンリリース
2. 旧バージョンにDeprecationヘッダー追加
3. 移行期間（最低6ヶ月）
4. 旧バージョン廃止

```http
Deprecation: true
Sunset: Sat, 01 Jan 2025 00:00:00 GMT
Link: </api/v2/files>; rel="successor-version"
```

---

## 3. 統一レスポンス形式

すべてのAPIレスポンスは同一のエンベロープ構造を持ちます。これにより:
- クライアント側の共通処理が容易になる
- APIの予測可能性が高まる
- バリデーション処理を最小化できる

### 3.1 基本構造

```typescript
// 成功時
{
  "data": T | T[],           // リソースデータ（単一またはコレクション）
  "meta": { ... } | null     // メタ情報（ページネーション等）
}

// 失敗時
{
  "error": {
    "code": string,
    "message": string,
    "details": [...] | null
  },
  "meta": { ... } | null
}
```

### 3.2 成功レスポンス

**単一リソース取得（GET /files/:id）:**
```json
{
  "data": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "name": "document.pdf",
    "size": 1048576,
    "mime_type": "application/pdf",
    "created_at": "2024-01-15T10:30:00Z",
    "updated_at": "2024-01-15T10:30:00Z"
  },
  "meta": null
}
```

**コレクション取得（GET /files）:**
```json
{
  "data": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "name": "document.pdf",
      "size": 1048576,
      "created_at": "2024-01-15T10:30:00Z"
    },
    {
      "id": "550e8400-e29b-41d4-a716-446655440001",
      "name": "image.png",
      "size": 2048000,
      "created_at": "2024-01-15T11:00:00Z"
    }
  ],
  "meta": {
    "pagination": {
      "page": 1,
      "per_page": 50,
      "total_items": 250,
      "total_pages": 5,
      "has_next": true,
      "has_prev": false
    }
  }
}
```

**リソース作成（POST /files）:**
```json
{
  "data": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "name": "document.pdf",
    "status": "pending",
    "upload_url": "https://minio.example.com/...",
    "created_at": "2024-01-15T10:30:00Z"
  },
  "meta": null
}
```

**アクション実行（POST /files/:id/move）:**
```json
{
  "data": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "folder_id": "550e8400-e29b-41d4-a716-446655440001",
    "updated_at": "2024-01-15T12:00:00Z"
  },
  "meta": {
    "message": "ファイルを移動しました"
  }
}
```

**削除（DELETE /files/:id）:**
```json
{
  "data": null,
  "meta": {
    "message": "ファイルを削除しました"
  }
}
```

### 3.3 エラーレスポンス

```json
{
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "リクエストの検証に失敗しました",
    "details": [
      {
        "field": "email",
        "message": "有効なメールアドレスを入力してください"
      },
      {
        "field": "password",
        "message": "パスワードは8文字以上必要です"
      }
    ]
  },
  "meta": null
}
```

**詳細なしのエラー:**
```json
{
  "error": {
    "code": "NOT_FOUND",
    "message": "ファイルが見つかりません",
    "details": null
  },
  "meta": null
}
```

### 3.4 クライアント側の処理例

```typescript
// 統一されたレスポンス型
interface ApiResponse<T> {
  data: T | null;
  meta: Record<string, unknown> | null;
  error: {
    code: string;
    message: string;
    details: Array<{ field: string; message: string }> | null;
  } | null;
}

// 共通処理
async function apiCall<T>(url: string): Promise<T> {
  const response = await fetch(url);
  const json: ApiResponse<T> = await response.json();

  if (!json.success) {
    throw new ApiError(json.error!);
  }

  return json.data!;
}
```

### 3.5 エラーコード体系

| HTTPステータス | エラーコード | 用途 |
|---------------|-------------|------|
| 400 | VALIDATION_ERROR | バリデーションエラー |
| 400 | INVALID_REQUEST | 不正なリクエスト形式 |
| 401 | UNAUTHORIZED | 認証が必要 |
| 401 | TOKEN_EXPIRED | トークン期限切れ |
| 403 | FORBIDDEN | アクセス権限なし |
| 403 | QUOTA_EXCEEDED | クォータ超過 |
| 404 | NOT_FOUND | リソースが存在しない |
| 409 | CONFLICT | リソース競合（名前重複等） |
| 429 | RATE_LIMIT_EXCEEDED | レート制限超過 |
| 500 | INTERNAL_ERROR | サーバー内部エラー |
| 503 | SERVICE_UNAVAILABLE | サービス利用不可 |

---

## 4. 共通ヘッダー

### 4.1 リクエストヘッダー

| ヘッダー | 必須 | 説明 |
|---------|------|------|
| Content-Type | Yes (POST/PUT/PATCH) | `application/json` |
| Authorization | Yes (認証API以外) | `Bearer {token}` |
| X-Request-ID | No | トレーシング用（未指定時は自動生成） |
| Accept-Language | No | レスポンス言語（デフォルト: ja） |

### 4.2 レスポンスヘッダー

| ヘッダー | 説明 |
|---------|------|
| Content-Type | `application/json` |
| X-Request-ID | リクエスト追跡ID |
| X-RateLimit-Limit | レート制限上限 |
| X-RateLimit-Remaining | 残りリクエスト数 |
| X-RateLimit-Reset | 制限リセット時刻（Unix timestamp） |

---

## 5. レート制限ポリシー

### 5.1 制限値

| カテゴリ | 制限 | 対象 |
|---------|------|------|
| 認証API | 10 req/min | IP単位 |
| 一般API | 1000 req/min | ユーザー単位 |
| アップロードAPI | 100 req/min | ユーザー単位 |
| 検索API | 30 req/min | ユーザー単位 |
| 公開共有API | 60 req/min | IP単位 |

### 5.2 レスポンスヘッダー

```http
X-RateLimit-Limit: 1000
X-RateLimit-Remaining: 950
X-RateLimit-Reset: 1640000060
```

### 5.3 制限超過時

```http
HTTP/1.1 429 Too Many Requests
Retry-After: 60
```

```json
{
  "error": {
    "code": "RATE_LIMIT_EXCEEDED",
    "message": "リクエスト数の上限に達しました。60秒後に再試行してください。",
    "details": null
  },
  "meta": null
}
```

### 5.4 レート制限実装方針

- Token Bucket アルゴリズムを使用
- Redis を使用した分散レート制限
- ユーザーID（認証済み）またはIPアドレス（未認証）で識別

---

## 6. ページネーション規則

### 6.1 クエリパラメータ

| パラメータ | デフォルト | 最大値 | 説明 |
|-----------|-----------|--------|------|
| page | 1 | - | ページ番号（1始まり） |
| per_page | 50 | 100 | 1ページあたりの件数 |
| sort | リソースによる | - | ソートキー |
| order | asc | - | ソート順（asc/desc） |

### 6.2 レスポンス形式

```json
{
  "data": [...],
  "meta": {
    "pagination": {
      "page": 2,
      "per_page": 50,
      "total_items": 250,
      "total_pages": 5,
      "has_next": true,
      "has_prev": true
    }
  }
}
```

### 6.3 大量データ対応

- 10,000件以上のデータはカーソルベースページネーションを検討
- `total_items`の取得はオプション化（クエリパラメータで制御）
- 検索結果は最大1,000件に制限

---

## 7. 認証・認可ポリシー

### 7.1 認証方式

JWT (JSON Web Token) を使用したBearer認証:

| トークン | 有効期限 | 用途 |
|---------|---------|------|
| Access Token | 1時間 | API認証 |
| Refresh Token | 7日 | Access Token再発行 |

### 7.2 トークン管理

- Access Tokenはメモリ/ローカルストレージに保存
- Refresh TokenはHttpOnly Cookieに保存推奨
- トークンローテーション: Refresh使用時に新しいペアを発行

### 7.3 認可モデル

リソースベースのアクセス制御:

| 権限レベル | できること |
|-----------|-----------|
| view | 閲覧のみ |
| edit | 閲覧 + 編集 |
| manage | 閲覧 + 編集 + 権限管理 + 削除 |

---

## 8. スケーリング考慮

### 8.1 ステートレス設計

- セッション状態をサーバーに保持しない
- 認証状態はJWTで管理
- アップロードセッションはDBで管理

### 8.2 キャッシュ戦略

| 対象 | キャッシュ方式 | TTL |
|------|--------------|-----|
| ユーザー情報 | Redis | 5分 |
| ファイルメタデータ | Redis | 1分 |
| フォルダ構造 | Redis | 1分 |
| Presigned URL | 生成しない（都度生成） | - |

**キャッシュ無効化:**
- 更新時に該当キーを削除
- パターンマッチによる一括削除

### 8.3 水平スケーリング対応

| コンポーネント | スケール方式 |
|--------------|-------------|
| APIサーバー | ロードバランサー配下で複数台 |
| Redis | Cluster または Sentinel |
| PostgreSQL | Read Replica + PgBouncer |
| MinIO | 分散モード |

### 8.4 非同期処理

長時間処理は非同期化:

| 処理 | 方式 |
|------|------|
| 大容量ファイルアップロード | Presigned URL + 完了通知 |
| ファイル削除（物理） | バックグラウンドジョブ |
| サムネイル生成 | メッセージキュー |

---

## 9. API仕様管理

### 9.1 OpenAPI (Swagger)

**ファイル構成:**
```
api/
├── openapi.yaml         # メイン定義
├── components/
│   ├── schemas/         # スキーマ定義
│   ├── parameters/      # パラメータ定義
│   ├── responses/       # レスポンス定義
│   └── securitySchemes/ # 認証定義
└── paths/               # パス定義
    ├── auth.yaml
    ├── files.yaml
    └── ...
```

### 9.2 仕様ファーストアプローチ

1. OpenAPI定義を先に作成
2. 定義からクライアントコード生成
3. 定義からバリデーション生成
4. 定義とE2Eテストで整合性確認

### 9.3 ドキュメント公開

- Swagger UI を `/docs` で公開（開発環境のみ）
- 本番環境ではアクセス制限または非公開

---

## 10. セキュリティ考慮

### 10.1 入力バリデーション

- すべての入力を検証
- 最大長、形式、許可値をチェック
- SQLインジェクション、XSS対策

### 10.2 出力サニタイズ

- JSONエスケープは自動
- ファイル名等はサニタイズ

### 10.3 CORS設定

```
Access-Control-Allow-Origin: https://app.gc-storage.example.com
Access-Control-Allow-Methods: GET, POST, PUT, PATCH, DELETE, OPTIONS
Access-Control-Allow-Headers: Content-Type, Authorization, X-Request-ID
Access-Control-Max-Age: 86400
```

### 10.4 その他のセキュリティヘッダー

```
X-Content-Type-Options: nosniff
X-Frame-Options: DENY
Strict-Transport-Security: max-age=31536000; includeSubDomains
```

---

## 関連ドキュメント

- [バックエンド設計](./BACKEND.md)
- [データベース設計](./DATABASE.md)
- [フロントエンド設計](./FRONTEND.md)
