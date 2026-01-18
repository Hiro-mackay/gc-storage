# 機能仕様マップ (Spec Map)

> このドキュメントは、03-domains/ のドメイン定義に基づく機能仕様の全体像と詳細設計の進め方を示します。

---

## 概要

GC Storage の機能仕様を **6つのフェーズ** に分類し、インフラ基盤から順次実装を進めます。

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                         GC Storage 機能仕様マップ                              │
└─────────────────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────────────────┐
│  Phase 0: Infrastructure Layer（基盤）                                        │
│  ┌─────────────┐ ┌─────────────┐ ┌─────────────┐ ┌─────────────┐           │
│  │ PostgreSQL  │ │    Redis    │ │    MinIO    │ │    SMTP     │           │
│  │  Database   │ │   Cache     │ │   Storage   │ │   Email     │           │
│  └─────────────┘ └─────────────┘ └─────────────┘ └─────────────┘           │
│  ┌─────────────┐ ┌─────────────┐ ┌─────────────┐                           │
│  │  API Base   │ │   Logging   │ │  Workers    │                           │
│  │  Framework  │ │  Monitoring │ │   Batch     │                           │
│  └─────────────┘ └─────────────┘ └─────────────┘                           │
└─────────────────────────────────────────────────────────────────────────────┘
                                       │
                                       ▼
                              ┌─────────────────┐
                              │    Identity     │  Phase 1
                              │    Context      │  認証・ユーザー管理
                              └────────┬────────┘
                                       │
                    ┌──────────────────┼──────────────────┐
                    │                  │                  │
                    ▼                  ▼                  ▼
          ┌─────────────────┐ ┌─────────────────┐ ┌─────────────────┐
          │   Storage       │ │ Collaboration   │ │ Authorization   │
          │   Context       │ │   Context       │ │   Context       │
          │                 │ │                 │ │                 │
          │  Phase 2        │ │  Phase 3        │ │  Phase 2-3      │
          │  ファイル・フォルダ │ │  グループ・招待   │ │  権限管理        │
          └────────┬────────┘ └────────┬────────┘ └────────┬────────┘
                   │                   │                   │
                   └───────────────────┼───────────────────┘
                                       │
                                       ▼
                              ┌─────────────────┐
                              │    Sharing      │  Phase 4
                              │    Context      │  共有リンク
                              └─────────────────┘
```

---

## Phase 0: Infrastructure Layer（インフラ基盤）

> **依存関係**: なし（全フェーズの基盤）
> **目的**: アプリケーション全体で使用する基盤サービスの統合

### 0A: データベース（PostgreSQL）

| # | 仕様ID | 機能名 | 優先度 | 状態 | 依存 |
|---|--------|--------|--------|------|------|
| 0A.1 | `db-connection` | DB接続・コネクションプール | High | Draft | - |
| 0A.2 | `db-migration` | マイグレーション管理 | High | Done | 0A.1 |
| 0A.3 | `db-transaction` | トランザクション管理 | High | Draft | 0A.1 |
| 0A.4 | `db-repository-base` | リポジトリ基底実装 | High | Draft | 0A.1 |

### 0B: キャッシュ（Redis）

| # | 仕様ID | 機能名 | 優先度 | 状態 | 依存 |
|---|--------|--------|--------|------|------|
| 0B.1 | `redis-connection` | Redis接続管理 | High | Draft | - |
| 0B.2 | `redis-session-store` | セッションストア | High | Draft | 0B.1 |
| 0B.3 | `redis-rate-limit` | レート制限 | Medium | Draft | 0B.1 |
| 0B.4 | `redis-cache` | 汎用キャッシュ | Medium | Draft | 0B.1 |

### 0C: オブジェクトストレージ（MinIO）

| # | 仕様ID | 機能名 | 優先度 | 状態 | 依存 |
|---|--------|--------|--------|------|------|
| 0C.1 | `minio-connection` | MinIO接続・クライアント初期化 | High | Draft | - |
| 0C.2 | `minio-bucket-setup` | バケット作成・設定 | High | Draft | 0C.1 |
| 0C.3 | `minio-presigned-put` | Presigned PUT URL生成 | High | Draft | 0C.1 |
| 0C.4 | `minio-presigned-get` | Presigned GET URL生成 | High | Draft | 0C.1 |
| 0C.5 | `minio-multipart-init` | マルチパートアップロード開始 | High | Draft | 0C.1 |
| 0C.6 | `minio-multipart-complete` | マルチパートアップロード完了 | High | Draft | 0C.5 |
| 0C.7 | `minio-multipart-abort` | マルチパートアップロード中断 | Medium | Draft | 0C.5 |
| 0C.8 | `minio-object-info` | オブジェクト情報取得 | High | Draft | 0C.1 |
| 0C.9 | `minio-object-delete` | オブジェクト削除 | High | Draft | 0C.1 |
| 0C.10 | `minio-object-copy` | オブジェクトコピー | Medium | Draft | 0C.1 |
| 0C.11 | `minio-lifecycle` | ライフサイクルポリシー設定 | Low | Draft | 0C.2 |
| 0C.12 | `minio-cleanup` | 孤立オブジェクトクリーンアップ | Low | Draft | 0C.1 |

#### MinIO 設計詳細

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                         MinIO Storage Architecture                           │
└─────────────────────────────────────────────────────────────────────────────┘

【バケット構成】
gc-storage (メインバケット)
├── users/{user_id}/{file_id}/v{version}     # ユーザー所有ファイル
├── groups/{group_id}/{file_id}/v{version}   # グループ所有ファイル
└── temp/{upload_session_id}                  # 一時アップロード領域

【アップロードフロー】
┌─────────┐      ┌─────────┐      ┌─────────┐      ┌─────────┐
│ Client  │─────▶│   API   │─────▶│  MinIO  │─────▶│   API   │
│         │      │         │      │         │      │         │
│ 1.要求  │      │2.Presign│      │3.Upload │      │4.Complete│
│         │      │  URL    │      │  Direct │      │  通知    │
└─────────┘      └─────────┘      └─────────┘      └─────────┘

【Presigned URL】
• PUT URL: 15分有効、クライアントから直接MinIOへアップロード
• GET URL: 1時間有効、クライアントから直接MinIOからダウンロード

【マルチパートアップロード】
• 5MB以上のファイルに推奨
• 最大10,000パート
• パートサイズ: 5MB〜5GB
• 各パートに個別のPresigned URL発行

【StorageKey形式】
{owner_type}/{owner_id}/{file_id}/v{version}
例: users/550e8400-e29b-41d4-a716-446655440000/file-123/v1
```

### 0D: メール送信（SMTP）

| # | 仕様ID | 機能名 | 優先度 | 状態 | 依存 |
|---|--------|--------|--------|------|------|
| 0D.1 | `smtp-connection` | SMTP接続設定 | High | Draft | - |
| 0D.2 | `email-template` | メールテンプレート管理 | High | Draft | 0D.1 |
| 0D.3 | `email-verification` | メール確認メール送信 | High | Draft | 0D.2 |
| 0D.4 | `email-password-reset` | パスワードリセットメール | High | Draft | 0D.2 |
| 0D.5 | `email-invitation` | グループ招待メール | Medium | Draft | 0D.2 |
| 0D.6 | `email-notification` | 通知メール（共有など） | Low | Draft | 0D.2 |

### 0E: API基盤

| # | 仕様ID | 機能名 | 優先度 | 状態 | 依存 |
|---|--------|--------|--------|------|------|
| 0E.1 | `api-router` | ルーティング設定 | High | Draft | - |
| 0E.2 | `api-middleware-auth` | 認証ミドルウェア | High | Draft | 0E.1 |
| 0E.3 | `api-middleware-cors` | CORSミドルウェア | High | Done | 0E.1 |
| 0E.4 | `api-error-response` | エラーレスポンス標準化 | High | Draft | 0E.1 |
| 0E.5 | `api-validation` | リクエストバリデーション | High | Draft | 0E.1 |
| 0E.6 | `api-pagination` | ページネーション共通化 | Medium | Draft | 0E.1 |
| 0E.7 | `api-rate-limit` | APIレート制限 | Medium | Draft | 0B.3 |
| 0E.8 | `api-versioning` | APIバージョニング | Low | Draft | 0E.1 |

### 0F: ロギング・監視

| # | 仕様ID | 機能名 | 優先度 | 状態 | 依存 |
|---|--------|--------|--------|------|------|
| 0F.1 | `log-structured` | 構造化ログ出力 | High | Draft | - |
| 0F.2 | `log-request` | リクエストログ | High | Draft | 0F.1 |
| 0F.3 | `log-audit` | 監査ログ | Medium | Draft | 0F.1 |
| 0F.4 | `metrics-basic` | 基本メトリクス | Medium | Draft | - |
| 0F.5 | `health-check` | ヘルスチェック | High | Done | - |
| 0F.6 | `health-readiness` | Readiness Probe | Medium | Draft | 0F.5 |

### 0G: バックグラウンドジョブ

| # | 仕様ID | 機能名 | 優先度 | 状態 | 依存 |
|---|--------|--------|--------|------|------|
| 0G.1 | `worker-framework` | ワーカー基盤 | Medium | Draft | - |
| 0G.2 | `job-session-cleanup` | 期限切れセッション削除 | Medium | Draft | 0G.1 |
| 0G.3 | `job-trash-cleanup` | ゴミ箱自動削除（30日） | Medium | Draft | 0G.1 |
| 0G.4 | `job-upload-cleanup` | 未完了アップロード削除 | Medium | Draft | 0G.1, 0C.7 |
| 0G.5 | `job-invitation-expire` | 期限切れ招待処理 | Low | Draft | 0G.1 |
| 0G.6 | `job-share-link-expire` | 期限切れ共有リンク処理 | Low | Draft | 0G.1 |
| 0G.7 | `job-storage-cleanup` | 孤立ストレージ削除 | Low | Draft | 0G.1, 0C.12 |

### Phase 0 実装順序

```
1. 0A.1 → 0A.2 → 0A.3 → 0A.4  (Database)
2. 0B.1 → 0B.2                  (Redis Session)
3. 0C.1 → 0C.2 → 0C.3 → 0C.4  (MinIO Basic)
4. 0E.1 → 0E.2 → 0E.4 → 0E.5  (API Base)
5. 0F.1 → 0F.2 → 0F.5          (Logging)
6. 0D.1 → 0D.2                  (Email Base)
7. 残りを順次実装
```

---

## Phase 1: Identity Context（認証・ユーザー管理）

> **依存関係**: なし（基盤）
> **ドメインファイル**: `03-domains/user.md`

### 仕様一覧

| # | 仕様ID | 機能名 | 優先度 | 状態 | 依存 |
|---|--------|--------|--------|------|------|
| 1.1 | `auth-register` | ユーザー登録 | High | Draft | - |
| 1.2 | `auth-login` | メール/パスワードログイン | High | Draft | 1.1 |
| 1.3 | `auth-oauth` | OAuth認証（Google/GitHub） | High | Draft | 1.1 |
| 1.4 | `auth-logout` | ログアウト | High | Draft | 1.2 |
| 1.5 | `auth-refresh` | トークンリフレッシュ | High | Draft | 1.2 |
| 1.6 | `auth-email-verify` | メールアドレス確認 | Medium | Draft | 1.1 |
| 1.7 | `auth-password-reset` | パスワードリセット | Medium | Draft | 1.1 |
| 1.8 | `auth-password-change` | パスワード変更 | Medium | Draft | 1.2 |
| 1.9 | `user-profile` | プロフィール管理 | Medium | Draft | 1.2 |
| 1.10 | `user-session-list` | セッション一覧・管理 | Low | Draft | 1.2 |
| 1.11 | `user-oauth-link` | OAuth連携管理 | Low | Draft | 1.3 |
| 1.12 | `user-deactivate` | アカウント無効化 | Low | Draft | 1.2 |

### 主要エンティティ

```
User ─┬─ OAuthAccount (1:N)
      ├─ Session (1:N, max 10)
      └─ UserProfile (1:1)
```

### 重要なビジネスルール

- email は一意
- パスワード or OAuth の少なくとも1つが必須
- セッションは最大10個（古いものから自動失効）
- status: pending → active → suspended/deactivated

---

## Phase 2: Storage Context（ファイル・フォルダ管理）

> **依存関係**: Identity Context
> **ドメインファイル**: `03-domains/file.md`, `03-domains/folder.md`

### 2A: フォルダ管理

| # | 仕様ID | 機能名 | 優先度 | 状態 | 依存 |
|---|--------|--------|--------|------|------|
| 2A.1 | `folder-create` | フォルダ作成 | High | Draft | 1.2 |
| 2A.2 | `folder-list` | フォルダ一覧・ナビゲーション | High | Draft | 2A.1 |
| 2A.3 | `folder-rename` | フォルダ名変更 | Medium | Draft | 2A.1 |
| 2A.4 | `folder-move` | フォルダ移動 | Medium | Draft | 2A.1 |
| 2A.5 | `folder-trash` | フォルダをゴミ箱へ | Medium | Draft | 2A.1 |
| 2A.6 | `folder-restore` | ゴミ箱から復元 | Medium | Draft | 2A.5 |
| 2A.7 | `folder-delete` | 完全削除 | Low | Draft | 2A.5 |

### 2B: ファイル管理

| # | 仕様ID | 機能名 | 優先度 | 状態 | 依存 |
|---|--------|--------|--------|------|------|
| 2B.1 | `file-upload` | ファイルアップロード（単一） | High | Draft | 2A.1 |
| 2B.2 | `file-upload-multipart` | 大容量ファイルアップロード | High | Draft | 2B.1 |
| 2B.3 | `file-download` | ファイルダウンロード | High | Draft | 2B.1 |
| 2B.4 | `file-list` | ファイル一覧 | High | Draft | 2A.2 |
| 2B.5 | `file-preview` | ファイルプレビュー | Medium | Draft | 2B.1 |
| 2B.6 | `file-rename` | ファイル名変更 | Medium | Draft | 2B.1 |
| 2B.7 | `file-move` | ファイル移動 | Medium | Draft | 2B.1 |
| 2B.8 | `file-copy` | ファイルコピー | Medium | Draft | 2B.1 |
| 2B.9 | `file-trash` | ファイルをゴミ箱へ | Medium | Draft | 2B.1 |
| 2B.10 | `file-restore` | ゴミ箱から復元 | Medium | Draft | 2B.9 |
| 2B.11 | `file-delete` | 完全削除 | Low | Draft | 2B.9 |
| 2B.12 | `file-version-list` | バージョン一覧 | Medium | Draft | 2B.1 |
| 2B.13 | `file-version-restore` | バージョン復元 | Medium | Draft | 2B.12 |
| 2B.14 | `file-search` | ファイル検索 | Medium | Draft | 2B.4 |
| 2B.15 | `file-metadata` | メタデータ取得 | Low | Draft | 2B.1 |

### 2C: ゴミ箱管理

| # | 仕様ID | 機能名 | 優先度 | 状態 | 依存 |
|---|--------|--------|--------|------|------|
| 2C.1 | `trash-list` | ゴミ箱一覧 | Medium | Draft | 2A.5, 2B.9 |
| 2C.2 | `trash-empty` | ゴミ箱を空にする | Low | Draft | 2C.1 |
| 2C.3 | `trash-auto-cleanup` | 自動クリーンアップ（30日） | Low | Draft | 2C.1 |

### 主要エンティティ

```
Folder ─┬─ parent_id → Folder (自己参照)
        └─ owner_id → User/Group

File ─┬─ folder_id → Folder
      ├─ FileVersion (1:N)
      ├─ FileMetadata (1:1)
      └─ UploadSession (1:N)
```

### 重要なビジネスルール

- 同一フォルダ内でファイル名・フォルダ名は一意
- フォルダの最大深さは20
- 循環参照防止（フォルダ移動時）
- ファイルステータス: pending → active → trashed → deleted
- バージョンは連番、最新バージョンは削除不可

---

## Phase 3: Collaboration Context（グループ・チーム協調）

> **依存関係**: Identity Context, Storage Context
> **ドメインファイル**: `03-domains/group.md`

### 3A: グループ管理

| # | 仕様ID | 機能名 | 優先度 | 状態 | 依存 |
|---|--------|--------|--------|------|------|
| 3A.1 | `group-create` | グループ作成 | High | Draft | 1.2 |
| 3A.2 | `group-list` | 所属グループ一覧 | High | Draft | 3A.1 |
| 3A.3 | `group-detail` | グループ詳細表示 | High | Draft | 3A.1 |
| 3A.4 | `group-update` | グループ情報更新 | Medium | Draft | 3A.1 |
| 3A.5 | `group-delete` | グループ削除 | Low | Draft | 3A.1 |
| 3A.6 | `group-ownership-transfer` | オーナー権限譲渡 | Low | Draft | 3A.1 |

### 3B: メンバー管理

| # | 仕様ID | 機能名 | 優先度 | 状態 | 依存 |
|---|--------|--------|--------|------|------|
| 3B.1 | `member-invite` | メンバー招待 | High | Draft | 3A.1 |
| 3B.2 | `member-accept` | 招待承諾 | High | Draft | 3B.1 |
| 3B.3 | `member-decline` | 招待辞退 | Medium | Draft | 3B.1 |
| 3B.4 | `member-list` | メンバー一覧 | High | Draft | 3A.1 |
| 3B.5 | `member-remove` | メンバー削除 | Medium | Draft | 3B.4 |
| 3B.6 | `member-leave` | グループ脱退 | Medium | Draft | 3B.4 |
| 3B.7 | `member-role-change` | ロール変更 | Medium | Draft | 3B.4 |
| 3B.8 | `invitation-list` | 招待一覧 | Low | Draft | 3B.1 |
| 3B.9 | `invitation-cancel` | 招待取消 | Low | Draft | 3B.8 |

### 主要エンティティ

```
Group ─┬─ owner_id → User
       ├─ Membership (1:N)
       └─ Invitation (1:N)

Membership ─── user_id → User
               role: member | admin | owner

Invitation ─── email, token, expires_at
               status: pending | accepted | declined | expired
```

### 重要なビジネスルール

- グループには必ず1人の owner が存在
- owner はグループから脱退不可（譲渡が必要）
- owner ロールでの招待は不可
- 同一ユーザーは同一グループに1つの Membership のみ
- 招待トークンは7日間有効

---

## Phase 2-3: Authorization Context（権限管理）

> **依存関係**: Identity Context, Storage Context, Collaboration Context
> **ドメインファイル**: `03-domains/permission.md`

### 権限管理

| # | 仕様ID | 機能名 | 優先度 | 状態 | 依存 |
|---|--------|--------|--------|------|------|
| P.1 | `permission-grant` | 権限付与 | High | Draft | 2A.1, 2B.1 |
| P.2 | `permission-revoke` | 権限取消 | High | Draft | P.1 |
| P.3 | `permission-list` | 権限一覧表示 | Medium | Draft | P.1 |
| P.4 | `permission-check` | アクセス権限判定 | High | Draft | P.1 |
| P.5 | `permission-inherit` | 権限継承解決 | High | Draft | P.1, 2A.4 |

### 認可モデル

```
┌─────────────────────────────────────────────────────────────────┐
│                     PBAC + ReBAC Hybrid Model                    │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  Permission = {resource}:{action}                               │
│  例: file:read, folder:write, group:invite                      │
│                                                                  │
│  Role = Permission の集合                                        │
│  • viewer: file:read, folder:read                               │
│  • editor: viewer + file:write, folder:write                    │
│  • manager: editor + permission:grant, file:share               │
│                                                                  │
│  Relationship Tuple (ReBAC)                                      │
│  (subject_type, subject_id, relation, object_type, object_id)   │
│  例: (user, u1, owner, folder, f1)                              │
│      (group, g1, viewer, file, f2)                              │
│      (folder, f1, parent, folder, f2)                           │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

### 権限解決フロー

```
1. ユーザーがリソースに操作を要求
2. 以下の経路から Permission を収集:
   a. 直接付与された Permission
   b. Role 経由の Permission
   c. 親フォルダからの継承 Permission
   d. グループメンバーシップ経由の Permission
3. 要求された Permission が収集セットに含まれるか判定
4. 許可 or 拒否
```

---

## Phase 4: Sharing Context（共有機能）

> **依存関係**: Storage Context, Authorization Context
> **ドメインファイル**: `03-domains/sharing.md`

### 共有リンク管理

| # | 仕様ID | 機能名 | 優先度 | 状態 | 依存 |
|---|--------|--------|--------|------|------|
| 4.1 | `share-link-create` | 共有リンク作成 | High | Draft | 2B.1, P.1 |
| 4.2 | `share-link-access` | 共有リンクアクセス | High | Draft | 4.1 |
| 4.3 | `share-link-list` | 共有リンク一覧 | Medium | Draft | 4.1 |
| 4.4 | `share-link-update` | 共有リンク設定変更 | Medium | Draft | 4.1 |
| 4.5 | `share-link-revoke` | 共有リンク無効化 | Medium | Draft | 4.1 |
| 4.6 | `share-link-password` | パスワード保護設定 | Medium | Draft | 4.1 |
| 4.7 | `share-link-expiry` | 有効期限設定 | Medium | Draft | 4.1 |
| 4.8 | `share-link-access-log` | アクセス履歴 | Low | Draft | 4.2 |

### 主要エンティティ

```
ShareLink ─┬─ resource_id → File/Folder
           ├─ created_by → User
           └─ ShareLinkAccess (1:N, アクセスログ)

ShareLink.permission: read | write
ShareLink.status: active | revoked | expired
```

### 重要なビジネスルール

- token は URL-safe、32文字以上
- expires_at 到達でアクセス不可
- max_access_count 到達でアクセス不可
- パスワード保護はオプション
- アクセス履歴は監査目的で記録

---

## Phase 5: フロントエンド実装

> **依存関係**: Phase 0〜4 の API 完成後
> **目的**: React SPA の実装

### 5A: 共通コンポーネント・基盤

| # | 仕様ID | 機能名 | 優先度 | 状態 | 依存 |
|---|--------|--------|--------|------|------|
| 5A.1 | `fe-api-client` | APIクライアント設定 | High | Draft | 0E |
| 5A.2 | `fe-auth-context` | 認証コンテキスト | High | Draft | 5A.1 |
| 5A.3 | `fe-router-setup` | TanStack Router設定 | High | Draft | - |
| 5A.4 | `fe-ui-components` | 共通UIコンポーネント | High | Draft | - |
| 5A.5 | `fe-error-boundary` | エラーバウンダリ | Medium | Draft | 5A.4 |
| 5A.6 | `fe-loading-states` | ローディング状態管理 | Medium | Draft | 5A.4 |

### 5B: 認証画面

| # | 仕様ID | 機能名 | 優先度 | 状態 | 依存 |
|---|--------|--------|--------|------|------|
| 5B.1 | `fe-login-page` | ログインページ | High | Draft | 5A.2 |
| 5B.2 | `fe-register-page` | 新規登録ページ | High | Draft | 5A.2 |
| 5B.3 | `fe-oauth-callback` | OAuthコールバック | High | Draft | 5B.1 |
| 5B.4 | `fe-password-reset` | パスワードリセット画面 | Medium | Draft | 5A.2 |

### 5C: ファイル管理画面

| # | 仕様ID | 機能名 | 優先度 | 状態 | 依存 |
|---|--------|--------|--------|------|------|
| 5C.1 | `fe-file-browser` | ファイルブラウザ | High | Draft | 5A.3 |
| 5C.2 | `fe-file-upload` | アップロードUI | High | Draft | 5C.1 |
| 5C.3 | `fe-file-preview` | プレビューモーダル | Medium | Draft | 5C.1 |
| 5C.4 | `fe-file-context-menu` | コンテキストメニュー | Medium | Draft | 5C.1 |
| 5C.5 | `fe-drag-drop` | ドラッグ&ドロップ | Medium | Draft | 5C.1 |
| 5C.6 | `fe-trash-view` | ゴミ箱ビュー | Low | Draft | 5C.1 |

### 5D: グループ・共有画面

| # | 仕様ID | 機能名 | 優先度 | 状態 | 依存 |
|---|--------|--------|--------|------|------|
| 5D.1 | `fe-group-list` | グループ一覧 | Medium | Draft | 5A.3 |
| 5D.2 | `fe-group-detail` | グループ詳細 | Medium | Draft | 5D.1 |
| 5D.3 | `fe-member-manage` | メンバー管理UI | Medium | Draft | 5D.2 |
| 5D.4 | `fe-share-dialog` | 共有ダイアログ | Medium | Draft | 5C.1 |
| 5D.5 | `fe-share-link-view` | 共有リンクアクセス画面 | Medium | Draft | - |

---

## 仕様サマリー

### フェーズ別仕様数

| フェーズ | カテゴリ数 | 仕様数 | 状態 |
|---------|----------|--------|------|
| Phase 0 | 7 | 47 | Draft |
| Phase 1 | 1 | 12 | Draft |
| Phase 2 | 3 | 25 | Draft |
| Phase 3 | 2 | 15 | Draft |
| Phase 2-3 | 1 | 5 | Draft |
| Phase 4 | 1 | 8 | Draft |
| Phase 5 | 4 | 21 | Draft |
| **合計** | **19** | **133** | - |

---

## 実装優先度マトリクス

```
                        重要度
                    高          低
               ┌─────────┬─────────┐
           高  │  P0     │   P1    │
    緊急度     │ 基盤    │  認証   │
               ├─────────┼─────────┤
           低  │  P2     │   P3    │
               │ Storage │ Share   │
               └─────────┴─────────┘

P0 (基盤):     db-connection, minio-connection, redis-connection, api-router
P1 (認証):     auth-register, auth-login, minio-presigned-*, email-verification
P2 (Storage):  folder-create, file-upload, file-download, permission-check
P3 (Share):    group-create, member-invite, share-link-create
```

### クリティカルパス

```
db-connection → db-repository-base ─┐
                                    │
redis-connection → redis-session ───┼─→ api-middleware-auth → auth-register
                                    │
minio-connection → minio-bucket ────┘
         │
         └─→ minio-presigned-put ─→ file-upload
         └─→ minio-presigned-get ─→ file-download
```

---

## 仕様書作成の進め方

### 1. 仕様書テンプレート

各仕様書は `04-specs/README.md` のテンプレートに従って作成します。

### 2. 命名規則

```
{category}-{feature}.md

例:
- db-connection.md
- minio-presigned-put.md
- auth-register.md
- file-upload.md
- fe-file-browser.md
```

### 3. 作成順序（推奨）

```
Phase 0（基盤）
├─ 0A: db-connection → db-transaction → db-repository-base
├─ 0B: redis-connection → redis-session-store
├─ 0C: minio-connection → minio-bucket-setup → minio-presigned-put/get
├─ 0D: smtp-connection → email-template
├─ 0E: api-router → api-middleware-auth → api-error-response
└─ 0F: log-structured → log-request

Phase 1（認証）
├─ auth-register → auth-email-verify
├─ auth-login → auth-refresh → auth-logout
└─ auth-oauth

Phase 2（ストレージ）
├─ folder-create → folder-list
├─ file-upload → file-upload-multipart → file-download
└─ file-list → file-preview

Phase 3（グループ）
├─ group-create → group-list
└─ member-invite → member-accept

Phase 4（共有）
└─ share-link-create → share-link-access

Phase 5（フロントエンド）
├─ fe-api-client → fe-auth-context
├─ fe-login-page → fe-register-page
└─ fe-file-browser → fe-file-upload
```

### 4. レビュープロセス

```
Draft → Review → Ready → In Progress → Done
  │        │       │          │         │
  └─ 作成中  └─ レビュー  └─ 実装可能  └─ 実装中  └─ 完了
```

### 5. 依存関係の確認

仕様書作成前に、依存する仕様が Ready または Done であることを確認してください。

---

## クイックリンク

### ドメイン参照

| コンテキスト | ドメインファイル | イベント定義 |
|-------------|-----------------|-------------|
| Identity | [user.md](../03-domains/user.md) | EVENT_STORMING §2.1 |
| Storage | [file.md](../03-domains/file.md), [folder.md](../03-domains/folder.md) | EVENT_STORMING §2.2 |
| Authorization | [permission.md](../03-domains/permission.md) | EVENT_STORMING §2.3 |
| Sharing | [sharing.md](../03-domains/sharing.md) | EVENT_STORMING §2.4 |
| Collaboration | [group.md](../03-domains/group.md) | EVENT_STORMING §2.5 |

### アーキテクチャ参照

| カテゴリ | ドキュメント | 参照ポイント |
|---------|------------|-------------|
| システム全体 | [SYSTEM.md](../02-architecture/SYSTEM.md) | データフロー、コンポーネント構成 |
| バックエンド | [BACKEND.md](../02-architecture/BACKEND.md) | Clean Architecture、ディレクトリ構成 |
| フロントエンド | [FRONTEND.md](../02-architecture/FRONTEND.md) | React設計、状態管理 |
| データベース | [DATABASE.md](../02-architecture/DATABASE.md) | スキーマ設計、インデックス |
| API | [API.md](../02-architecture/API.md) | エンドポイント規約、レスポンス形式 |
| セキュリティ | [SECURITY.md](../02-architecture/SECURITY.md) | 認証・認可、JWT設計 |

### 技術スタック参照

| レイヤー | 技術 | 公式ドキュメント |
|---------|------|-----------------|
| Database | PostgreSQL 16 | https://www.postgresql.org/docs/16/ |
| Cache | Redis 7 | https://redis.io/docs/ |
| Storage | MinIO | https://min.io/docs/minio/linux/developers/go/API.html |
| Backend | Echo v4 | https://echo.labstack.com/docs |
| Frontend | React 19 | https://react.dev/ |
| Router | TanStack Router | https://tanstack.com/router/latest |
| Query | TanStack Query | https://tanstack.com/query/latest |

---

## 次のステップ

1. **Phase 0** から順次仕様書を作成
2. 各仕様書は実装可能な粒度で設計（1仕様 ≒ 1PR）
3. 依存する仕様が Ready になってから着手
4. 実装前にレビュー・承認を得る
5. 実装完了後、Done ステータスに更新

### 推奨開始ポイント

最初に作成すべき仕様書:
1. `db-connection.md` - データベース接続基盤
2. `minio-connection.md` - MinIO接続基盤
3. `redis-connection.md` - Redis接続基盤
4. `api-router.md` - APIルーティング基盤

---

## 関連ドキュメント

- [仕様書テンプレート](./README.md) - 仕様書の書き方
- [ドメイン定義](../03-domains/README.md) - ビジネスルール
- [イベントストーミング](../03-domains/EVENT_STORMING.md) - ドメインイベント
- [アーキテクチャ設計](../02-architecture/) - 技術設計
- [コーディング規約](../01-policies/CODING_STANDARDS.md) - 命名規則
- [テスト方針](../01-policies/TESTING.md) - テスト戦略
