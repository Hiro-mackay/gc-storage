# フロントエンドAPI連携仕様書

## メタ情報

| 項目 | 値 |
|------|-----|
| ステータス | Ready |
| 優先度 | High |
| 関連仕様 | fe-foundation, fe-auth-pages, fe-file-browser |
| 依存する仕様 | auth-identity, storage-folder, storage-file |

---

## 1. 概要

本ドキュメントでは、フロントエンドとバックエンドAPI間の連携仕様を定義します。認証方式、APIエンドポイント、リクエスト/レスポンス形式、エラーハンドリングを含みます。

---

## 2. 認証方式

### 2.1 セッションベース認証

| 項目 | 値 |
|------|-----|
| 認証方式 | セッションID（Cookie） |
| Cookie名 | `session_id` |
| HttpOnly | true（JavaScriptからアクセス不可） |
| Secure | true（HTTPS必須） |
| SameSite | Lax（OAuthリダイレクト対応） |
| 有効期限 | 7日（スライディングウィンドウ） |

### 2.2 フロントエンドの責務

- APIリクエストに `credentials: 'include'` を設定
- セッションIDの保存・取得はブラウザが自動処理
- フロントエンドはセッションIDに直接アクセスしない

### 2.3 認証状態

| 状態 | 説明 | 判定方法 |
|------|------|---------|
| initializing | アプリ起動時、認証状態を確認中 | GET /api/v1/me 呼び出し中 |
| authenticated | ログイン済み | GET /api/v1/me が成功 |
| unauthenticated | 未ログイン | GET /api/v1/me が401エラー |

### 2.4 認証フロー

```
アプリ起動
    │
    ▼
GET /api/v1/me (Cookie自動送信)
    │
    ├─ 200 OK ──▶ authenticated（ユーザー情報を保持）
    │
    └─ 401 Unauthorized ──▶ unauthenticated
```

---

## 3. API基本設定

### 3.1 ベースURL

| 環境 | URL |
|------|-----|
| 開発 | `/api/v1`（Viteプロキシ経由） |
| 本番 | `https://api.gc-storage.example.com/api/v1` |

### 3.2 共通ヘッダー

| ヘッダー | 値 | 用途 |
|---------|-----|------|
| Content-Type | application/json | リクエストボディ形式 |
| Accept | application/json | レスポンス形式 |

### 3.3 タイムアウト

- デフォルト: 30秒
- ファイルアップロード: 5分

---

## 型安全なAPI呼び出し

API呼び出しはOpenAPIスキーマから自動生成された型定義を使用し、E2Eの型安全性を確保します。

詳細は [fe-openapi-typegen.md](./fe-openapi-typegen.md) を参照。

### API クライアント

```typescript
import { api } from "@/lib/api/client";

// パス・パラメータ・レスポンスがすべて型推論される
const { data, error } = await api.POST("/auth/login", {
  body: { email, password },
});
```

---

## 4. 認証API

### 4.1 ユーザー登録

**エンドポイント**: `POST /api/v1/auth/register`

**認証**: 不要

**リクエスト**:
```json
{
  "email": "user@example.com",
  "password": "Password123",
  "name": "User Name"
}
```

| フィールド | 型 | 必須 | バリデーション |
|-----------|-----|------|--------------|
| email | string | Yes | 有効なメール形式、最大255文字 |
| password | string | Yes | 8文字以上、大文字1つ以上、数字1つ以上 |
| name | string | Yes | 1-100文字 |

**レスポンス** (201 Created):
```json
{
  "data": {
    "user_id": "550e8400-e29b-41d4-a716-446655440000",
    "message": "Registration successful. Please check your email to verify your account."
  }
}
```

**動作**:
- 登録成功時、確認メールを送信
- セッションは作成しない（メール確認後にログイン）

---

### 4.2 ログイン

**エンドポイント**: `POST /api/v1/auth/login`

**認証**: 不要

**リクエスト**:
```json
{
  "email": "user@example.com",
  "password": "Password123"
}
```

**レスポンス** (200 OK):
```json
{
  "data": {
    "user": {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "email": "user@example.com",
      "name": "User Name",
      "status": "active",
      "email_verified": true,
      "created_at": "2024-01-01T00:00:00Z",
      "updated_at": "2024-01-01T00:00:00Z"
    }
  }
}
```

**動作**:
- 成功時、`session_id` Cookieが設定される
- フロントエンドはユーザー情報をメモリに保持

---

### 4.3 ログアウト

**エンドポイント**: `POST /api/v1/auth/logout`

**認証**: 必須

**リクエスト**: なし

**レスポンス** (200 OK):
```json
{
  "data": {
    "message": "logged out successfully"
  }
}
```

**動作**:
- サーバー側でセッションを削除
- `session_id` Cookieを削除
- フロントエンドはユーザー情報をクリア

---

### 4.4 現在のユーザー取得

**エンドポイント**: `GET /api/v1/me`

**認証**: 必須

**レスポンス** (200 OK):
```json
{
  "data": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "email": "user@example.com",
    "name": "User Name",
    "status": "active",
    "email_verified": true,
    "created_at": "2024-01-01T00:00:00Z",
    "updated_at": "2024-01-01T00:00:00Z"
  }
}
```

**用途**:
- アプリ起動時の認証状態確認
- ページリロード時のユーザー情報再取得

---

### 4.5 OAuth認証

**エンドポイント**: `POST /api/v1/auth/oauth/:provider`

**認証**: 不要

**パスパラメータ**:
- `provider`: `google` | `github`

**リクエスト**:
```json
{
  "code": "authorization_code_from_provider"
}
```

**レスポンス** (200 OK):
```json
{
  "data": {
    "user": {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "email": "user@example.com",
      "name": "User Name",
      "status": "active",
      "email_verified": true,
      "created_at": "2024-01-01T00:00:00Z",
      "updated_at": "2024-01-01T00:00:00Z"
    },
    "is_new_user": false
  }
}
```

**OAuthフロー**:
1. フロントエンドからOAuthプロバイダーの認証URLへリダイレクト
2. ユーザーが認証を許可
3. プロバイダーからコールバックURLにリダイレクト（codeパラメータ付き）
4. フロントエンドがcodeをバックエンドに送信
5. バックエンドがトークン交換・ユーザー情報取得
6. セッション作成・Cookie設定

---

### 4.6 メール確認

**エンドポイント**: `POST /api/v1/auth/email/verify?token=xxx`

**認証**: 不要

**クエリパラメータ**:
- `token`: メール確認トークン

**レスポンス** (200 OK):
```json
{
  "data": {
    "message": "Email verified successfully"
  }
}
```

---

### 4.7 確認メール再送

**エンドポイント**: `POST /api/v1/auth/email/resend`

**認証**: 不要

**リクエスト**:
```json
{
  "email": "user@example.com"
}
```

**レスポンス** (200 OK):
```json
{
  "data": {
    "message": "Verification email sent"
  }
}
```

---

### 4.8 パスワードリセット要求

**エンドポイント**: `POST /api/v1/auth/password/forgot`

**認証**: 不要

**リクエスト**:
```json
{
  "email": "user@example.com"
}
```

**レスポンス** (200 OK):
```json
{
  "data": {
    "message": "Password reset email sent"
  }
}
```

**セキュリティ**:
- メールが存在しない場合も同じレスポンスを返す

---

### 4.9 パスワードリセット

**エンドポイント**: `POST /api/v1/auth/password/reset`

**認証**: 不要

**リクエスト**:
```json
{
  "token": "reset_token_from_email",
  "password": "NewPassword123"
}
```

**レスポンス** (200 OK):
```json
{
  "data": {
    "message": "Password reset successfully"
  }
}
```

---

## 5. フォルダAPI

### 5.1 ルートフォルダ内容取得

**エンドポイント**: `GET /api/v1/folders/root/contents`

**認証**: 必須

**レスポンス** (200 OK):
```json
{
  "data": {
    "folders": [
      {
        "id": "550e8400-e29b-41d4-a716-446655440001",
        "name": "Documents",
        "parentId": null,
        "ownerId": "550e8400-e29b-41d4-a716-446655440000",
        "depth": 1,
        "status": "active",
        "createdAt": "2024-01-01T00:00:00Z",
        "updatedAt": "2024-01-01T00:00:00Z"
      }
    ],
    "files": [
      {
        "id": "550e8400-e29b-41d4-a716-446655440002",
        "name": "document.pdf",
        "mimeType": "application/pdf",
        "size": 1024000,
        "folderId": "550e8400-e29b-41d4-a716-446655440000",
        "ownerId": "550e8400-e29b-41d4-a716-446655440000",
        "currentVersion": 1,
        "status": "active",
        "createdAt": "2024-01-01T00:00:00Z",
        "updatedAt": "2024-01-01T00:00:00Z"
      }
    ]
  }
}
```

---

### 5.2 フォルダ内容取得

**エンドポイント**: `GET /api/v1/folders/:id/contents`

**認証**: 必須

**パスパラメータ**:
- `id`: フォルダID (UUID)

**レスポンス** (200 OK):
```json
{
  "data": {
    "folder": {
      "id": "550e8400-e29b-41d4-a716-446655440001",
      "name": "Documents",
      "parentId": null,
      "ownerId": "550e8400-e29b-41d4-a716-446655440000",
      "depth": 1,
      "status": "active",
      "createdAt": "2024-01-01T00:00:00Z",
      "updatedAt": "2024-01-01T00:00:00Z"
    },
    "folders": [...],
    "files": [...]
  }
}
```

---

### 5.3 パンくずリスト取得

**エンドポイント**: `GET /api/v1/folders/:id/ancestors`

**認証**: 必須

**レスポンス** (200 OK):
```json
{
  "data": {
    "items": [
      { "id": "550e8400-e29b-41d4-a716-446655440001", "name": "Documents" },
      { "id": "550e8400-e29b-41d4-a716-446655440003", "name": "Projects" },
      { "id": "550e8400-e29b-41d4-a716-446655440004", "name": "2024" }
    ]
  }
}
```

**順序**: ルートから現在のフォルダまで昇順

---

### 5.4 フォルダ作成

**エンドポイント**: `POST /api/v1/folders`

**認証**: 必須

**リクエスト**:
```json
{
  "name": "New Folder",
  "parentId": "550e8400-e29b-41d4-a716-446655440001"
}
```

| フィールド | 型 | 必須 | バリデーション |
|-----------|-----|------|--------------|
| name | string | Yes | 1-255文字 |
| parentId | string | No | UUID、nullでルート |

**レスポンス** (201 Created):
```json
{
  "data": {
    "id": "550e8400-e29b-41d4-a716-446655440005",
    "name": "New Folder",
    "parentId": "550e8400-e29b-41d4-a716-446655440001",
    "ownerId": "550e8400-e29b-41d4-a716-446655440000",
    "depth": 2,
    "status": "active",
    "createdAt": "2024-01-01T00:00:00Z",
    "updatedAt": "2024-01-01T00:00:00Z"
  }
}
```

---

### 5.5 フォルダ名変更

**エンドポイント**: `PATCH /api/v1/folders/:id/rename`

**認証**: 必須

**リクエスト**:
```json
{
  "name": "Renamed Folder"
}
```

**レスポンス** (200 OK): フォルダオブジェクト

---

### 5.6 フォルダ削除

**エンドポイント**: `DELETE /api/v1/folders/:id`

**認証**: 必須

**レスポンス**: 204 No Content

---

## 6. ファイルAPI

### 6.1 アップロード開始

**エンドポイント**: `POST /api/v1/files/upload`

**認証**: 必須

**リクエスト**:
```json
{
  "folderId": "550e8400-e29b-41d4-a716-446655440001",
  "fileName": "document.pdf",
  "mimeType": "application/pdf",
  "size": 1024000
}
```

**レスポンス** (200 OK):
```json
{
  "data": {
    "sessionId": "550e8400-e29b-41d4-a716-446655440010",
    "fileId": "550e8400-e29b-41d4-a716-446655440011",
    "isMultipart": false,
    "uploadUrls": [
      {
        "partNumber": 1,
        "url": "https://minio.example.com/presigned-put-url",
        "expiresAt": "2024-01-01T01:00:00Z"
      }
    ],
    "expiresAt": "2024-01-01T01:00:00Z"
  }
}
```

**アップロードフロー**:
1. フロントエンドがアップロード開始APIを呼び出し
2. Presigned URLを取得
3. フロントエンドがPresigned URLに直接ファイルをPUT
4. MinIOがWebhookでバックエンドに完了通知
5. ファイル一覧が自動更新される

---

### 6.2 ダウンロードURL取得

**エンドポイント**: `GET /api/v1/files/:id/download`

**認証**: 必須

**クエリパラメータ**:
- `version`: バージョン番号（オプション、未指定時は最新）

**レスポンス** (200 OK):
```json
{
  "data": {
    "fileId": "550e8400-e29b-41d4-a716-446655440011",
    "fileName": "document.pdf",
    "mimeType": "application/pdf",
    "size": 1024000,
    "versionNumber": 1,
    "downloadUrl": "https://minio.example.com/presigned-get-url",
    "expiresAt": "2024-01-01T02:00:00Z"
  }
}
```

**ダウンロードフロー**:
1. フロントエンドがダウンロードURL取得APIを呼び出し
2. Presigned URLを取得
3. ブラウザがPresigned URLからファイルをダウンロード

---

### 6.3 ファイル名変更

**エンドポイント**: `PATCH /api/v1/files/:id/rename`

**認証**: 必須

**リクエスト**:
```json
{
  "name": "new-name.pdf"
}
```

---

### 6.4 ファイル移動

**エンドポイント**: `PATCH /api/v1/files/:id/move`

**認証**: 必須

**リクエスト**:
```json
{
  "newFolderId": "550e8400-e29b-41d4-a716-446655440002"
}
```

---

### 6.5 ゴミ箱へ移動

**エンドポイント**: `POST /api/v1/files/:id/trash`

**認証**: 必須

---

### 6.6 ゴミ箱一覧

**エンドポイント**: `GET /api/v1/trash`

**認証**: 必須

**レスポンス** (200 OK):
```json
{
  "data": {
    "items": [
      {
        "id": "550e8400-e29b-41d4-a716-446655440020",
        "originalFileId": "550e8400-e29b-41d4-a716-446655440011",
        "originalFolderId": "550e8400-e29b-41d4-a716-446655440001",
        "originalPath": "/Documents/document.pdf",
        "name": "document.pdf",
        "mimeType": "application/pdf",
        "size": 1024000,
        "archivedAt": "2024-01-01T00:00:00Z",
        "expiresAt": "2024-02-01T00:00:00Z",
        "daysUntilExpiry": 31
      }
    ]
  }
}
```

---

### 6.7 ゴミ箱から復元

**エンドポイント**: `POST /api/v1/trash/:id/restore`

**認証**: 必須

**リクエスト**:
```json
{
  "restoreFolderId": "550e8400-e29b-41d4-a716-446655440001"
}
```

| フィールド | 型 | 必須 | 説明 |
|-----------|-----|------|------|
| restoreFolderId | string | No | 復元先フォルダID、nullで元の場所 |

---

## 7. エラーレスポンス

### 7.1 形式

```json
{
  "error": {
    "code": "ERROR_CODE",
    "message": "Human-readable error message",
    "details": [
      {
        "field": "email",
        "message": "Please enter a valid email address"
      }
    ]
  }
}
```

### 7.2 エラーコード一覧

| HTTPステータス | code | 説明 | フロントエンド動作 |
|---------------|------|------|------------------|
| 400 | VALIDATION_ERROR | バリデーションエラー | フィールドエラー表示 |
| 400 | INVALID_REQUEST | 不正なリクエスト | エラーメッセージ表示 |
| 401 | UNAUTHORIZED | 認証が必要 | ログインページへリダイレクト |
| 403 | FORBIDDEN | アクセス権限なし | 権限エラー表示 |
| 404 | NOT_FOUND | リソースが存在しない | 404ページ表示 |
| 409 | CONFLICT | リソース競合（名前重複等） | エラーメッセージ表示 |
| 429 | RATE_LIMIT_EXCEEDED | レート制限超過 | リトライ案内表示 |
| 500 | INTERNAL_ERROR | サーバー内部エラー | エラーページ表示 |

### 7.3 フロントエンドのエラーハンドリング

| エラー種別 | 処理 |
|-----------|------|
| 401 Unauthorized | 認証状態をクリア、ログインページへリダイレクト |
| バリデーションエラー | フォームフィールドにエラーメッセージを表示 |
| ネットワークエラー | 「インターネット接続を確認してください」を表示 |
| 500系エラー | 「サーバーでエラーが発生しました。しばらく待ってから再試行してください」を表示 |

---

## 8. データ型定義

### 8.1 User

| フィールド | 型 | 説明 |
|-----------|-----|------|
| id | string (UUID) | ユーザーID |
| email | string | メールアドレス |
| name | string | 表示名 |
| status | string | active / pending / suspended / deactivated |
| email_verified | boolean | メール確認済み |
| created_at | string (ISO 8601) | 作成日時 |
| updated_at | string (ISO 8601) | 更新日時 |

### 8.2 Folder

| フィールド | 型 | 説明 |
|-----------|-----|------|
| id | string (UUID) | フォルダID |
| name | string | フォルダ名 |
| parentId | string (UUID) / null | 親フォルダID |
| ownerId | string (UUID) | オーナーID |
| depth | number | 階層深さ |
| status | string | active / trashed |
| createdAt | string (ISO 8601) | 作成日時 |
| updatedAt | string (ISO 8601) | 更新日時 |

### 8.3 File

| フィールド | 型 | 説明 |
|-----------|-----|------|
| id | string (UUID) | ファイルID |
| name | string | ファイル名 |
| mimeType | string | MIMEタイプ |
| size | number | サイズ（バイト） |
| folderId | string (UUID) | 所属フォルダID |
| ownerId | string (UUID) | オーナーID |
| currentVersion | number | 現在のバージョン番号 |
| status | string | active / trashed |
| createdAt | string (ISO 8601) | 作成日時 |
| updatedAt | string (ISO 8601) | 更新日時 |

---

## 9. 受け入れ基準

### 9.1 API通信

- [ ] すべてのAPIリクエストに `credentials: 'include'` が設定される
- [ ] APIエラーレスポンスからエラーコード・メッセージを取得できる
- [ ] 401エラー時にログイン画面にリダイレクトされる
- [ ] ネットワークエラー時に適切なエラーメッセージが表示される
- [ ] タイムアウト時に適切なエラーメッセージが表示される

### 9.2 認証フロー

- [ ] アプリ起動時に認証状態が確認される
- [ ] ログイン成功後、ユーザー情報がメモリに保持される
- [ ] ログアウト後、ユーザー情報がクリアされる
- [ ] ページリロード時、APIからユーザー情報を再取得する

### 9.3 データ整合性

- [ ] APIレスポンスの型がフロントエンドの型定義と一致する
- [ ] 日付フィールドがISO 8601形式で処理される
- [ ] UUIDフィールドが文字列として処理される

---

## 関連ドキュメント

- [fe-foundation.md](./fe-foundation.md) - フロントエンド基盤仕様
- [fe-auth-pages.md](./fe-auth-pages.md) - 認証画面仕様
- [fe-file-browser.md](./fe-file-browser.md) - ファイルブラウザ仕様
- [auth-identity.md](./auth-identity.md) - 認証バックエンド仕様
- [API.md](../02-architecture/API.md) - API設計
