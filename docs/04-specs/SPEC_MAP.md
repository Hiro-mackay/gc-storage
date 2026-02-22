# 機能仕様マップ (Spec Map)

> Feature-based で再編された仕様マップ。1つの機能をフルサイクル（Domain -> API -> UI -> Test）で開発できる構造。

---

## 概要

GC Storage の機能仕様を **5つの Tier** で分類し、依存関係に従って実装を進めます。

```
+---------------------------------------------------------------+
|  Infra Layer (cross-cutting)                                   |
|  database, redis, minio, email, api-base                       |
+---------------------------------------------------------------+
         |
         v
+---------------------------------------------------------------+
|  Platform Layer                                                |
|  openapi-typegen, frontend-foundation                          |
+---------------------------------------------------------------+
         |
         v
+------------------+  +------------------+  +------------------+
| Tier 1: Auth     |  | Tier 2: Storage  |  | Tier 3: Secondary|
| registration     |->| folder-mgmt      |  | password         |
| login            |  | file-upload      |  | user-profile     |
+------------------+  | file-mgmt        |  +------------------+
                      | trash            |
                      +------------------+
                               |
         +---------------------+---------------------+
         |                                           |
         v                                           v
+------------------+                      +------------------+
| Tier 4: Collab   |                      | Tier 5: Cross    |
| group-mgmt       |--------------------->| permission-mgmt  |
| group-invitation |                      | sharing          |
+------------------+                      +------------------+
```

---

## ディレクトリ構造

```
docs/04-specs/
  SPEC_MAP.md                          # 本ファイル
  README.md                            # 仕様書ガイドライン

  infra/                               # インフラ基盤 (cross-cutting)
    database.md                        # PostgreSQL
    redis.md                           # Redis
    minio.md                           # MinIO
    email.md                           # SMTP
    api-base.md                        # API routing, middleware

  platform/                            # プラットフォーム共通
    openapi-typegen.md                 # OpenAPI型生成パイプライン
    frontend-foundation.md             # FE基盤 (API通信, 認証, routing, 状態管理, レイアウト)

  features/                            # Feature specs (フルスタック)
    auth-registration.md               # 登録 + メール確認
    auth-login.md                      # ログイン (email/OAuth) + ログアウト
    auth-password.md                   # パスワードリセット + 変更
    user-profile.md                    # プロフィール + 設定
    folder-management.md               # フォルダ CRUD + 階層ナビ
    file-upload.md                     # アップロード (single + multipart)
    file-management.md                 # ダウンロード, rename, move
    trash.md                           # ゴミ箱ライフサイクル
    group-management.md                # グループ CRUD + メンバー管理
    group-invitation.md                # 招待フロー
    permission-management.md           # PBAC+ReBAC + 権限UI
    sharing.md                         # 共有リンク + 公開アクセス

  templates/
    FEATURE_SPEC_TEMPLATE.md           # Feature spec テンプレート
    TEST_SPEC_TEMPLATE.md              # テストケース設計テンプレート

  archive/                             # 旧specファイル (参照用)
```

---

## Feature Spec 一覧

### Tier 1: Authentication（認証基盤）

> 依存: Infra Layer, Platform Layer
> ドメイン: `03-domains/user.md`

| Feature Spec | 説明 | 旧Spec ID | Status |
|-------------|------|-----------|--------|
| [auth-registration.md](./features/auth-registration.md) | ユーザー登録 + メール確認 + Personal Folder自動作成 | 1.1, 1.6 | Draft |
| [auth-login.md](./features/auth-login.md) | Email/Password + OAuth ログイン + ログアウト | 1.2-1.5 | Draft |

### Tier 2: Storage（ファイル・フォルダ管理）

> 依存: Tier 1
> ドメイン: `03-domains/folder.md`, `03-domains/file.md`

| Feature Spec | 説明 | 旧Spec ID | Status |
|-------------|------|-----------|--------|
| [folder-management.md](./features/folder-management.md) | フォルダ CRUD + Closure Table階層 + パンくず | 2A.1-2A.4 | Draft |
| [file-upload.md](./features/file-upload.md) | Single/Multipart Upload (Webhook駆動) | 2B.1-2B.2 | Draft |
| [file-management.md](./features/file-management.md) | ダウンロード + リネーム + 移動 + 一覧 | 2B.3-2B.7 | Draft |
| [trash.md](./features/trash.md) | ゴミ箱ライフサイクル (Archive Table方式) | 2A.5-7, 2B.9-11, 2C | Draft |

### Tier 3: Secondary Features（セカンダリ機能）

> 依存: Tier 1
> ドメイン: `03-domains/user.md`

| Feature Spec | 説明 | 旧Spec ID | Status |
|-------------|------|-----------|--------|
| [auth-password.md](./features/auth-password.md) | パスワードリセット + パスワード変更 | 1.7-1.8 | Draft |
| [user-profile.md](./features/user-profile.md) | プロフィール管理 + 設定 | 1.9 | Draft |

### Tier 4: Collaboration（グループ・チーム協調）

> 依存: Tier 1, Tier 2
> ドメイン: `03-domains/group.md`

| Feature Spec | 説明 | 旧Spec ID | Status |
|-------------|------|-----------|--------|
| [group-management.md](./features/group-management.md) | グループ CRUD + メンバー管理 | 3A, 3B.4-7 | Draft |
| [group-invitation.md](./features/group-invitation.md) | 招待フロー (送信/承諾/辞退/取消) | 3B.1-3, 3B.8-9 | Draft |

### Tier 5: Cross-cutting Features（横断機能）

> 依存: Tier 2, Tier 4
> ドメイン: `03-domains/permission.md`, `03-domains/sharing.md`

| Feature Spec | 説明 | 旧Spec ID | Status |
|-------------|------|-----------|--------|
| [permission-management.md](./features/permission-management.md) | PBAC+ReBAC ハイブリッド認可 + 権限UI | P.1-P.5 | Draft |
| [sharing.md](./features/sharing.md) | 共有リンク + パスワード保護 + 公開アクセス | 4.1-4.8 | Draft |

---

## Infra / Platform Specs

### Infrastructure Layer

| Spec | 説明 | Status |
|------|------|--------|
| [infra/database.md](./infra/database.md) | PostgreSQL接続・トランザクション管理 | Done |
| [infra/redis.md](./infra/redis.md) | Redis接続・セッション・キャッシュ | Done |
| [infra/minio.md](./infra/minio.md) | MinIOストレージ・Presigned URL・Webhook | Done |
| [infra/email.md](./infra/email.md) | SMTP・メールテンプレート | Done |
| [infra/api-base.md](./infra/api-base.md) | APIルーティング・ミドルウェア・エラー標準化 | Done |

### Platform Layer

| Spec | 説明 | Status |
|------|------|--------|
| [platform/openapi-typegen.md](./platform/openapi-typegen.md) | OpenAPI型生成パイプライン | Ready |
| [platform/frontend-foundation.md](./platform/frontend-foundation.md) | FE基盤（API通信, 認証, routing, 状態管理, レイアウト） | Ready |

---

## 依存関係グラフ

```
infra/database ─┐
infra/redis ────┼── infra/api-base ── platform/* ── auth-registration
infra/minio ────┘                                        |
infra/email ────────────────────────────────────────────-+
                                                         |
                                                    auth-login
                                                    /    |    \
                                                   /     |     \
                                  folder-management  auth-password  group-management
                                       |                |               |
                                  file-upload     user-profile    group-invitation
                                       |                               |
                                  file-management                      |
                                       |                               |
                                     trash         permission-management
                                       |                    |
                                       +--------------------+
                                                |
                                             sharing
```

---

## 実装推奨順序

| 順序 | Feature Spec | 理由 |
|------|-------------|------|
| 1 | auth-registration | ユーザー登録がすべての基盤 |
| 2 | auth-login | 認証が他の全機能の前提 |
| 3 | folder-management | Storage機能の基盤（Personal Folder含む） |
| 4 | file-upload | コアStorage機能 |
| 5 | file-management | ダウンロード・操作はアップロード後 |
| 6 | trash | ゴミ箱はファイル/フォルダ操作の延長 |
| 7 | auth-password | 認証の補完機能 |
| 8 | user-profile | 設定画面 |
| 9 | group-management | コラボレーション基盤 |
| 10 | group-invitation | グループ機能の拡張 |
| 11 | permission-management | 権限管理（グループ/Storage依存） |
| 12 | sharing | 共有リンク（最後、全機能を前提） |

---

## ドメイン検証レポート（Step 0）

実装コードと `03-domains/` の突き合わせ結果:

| Domain | 整合性 | 注記 |
|--------|--------|------|
| user.md | 95%+ | `name` vs `display_name` の命名差異のみ。Session Redis実装、personal_folder_id、UserProfile すべて実装済み |
| file.md | 95%+ | ArchivedFile/ArchivedFileVersion テーブル実装済み。Webhook駆動アップロード実装済み。UploadSession/UploadPart 完全実装 |
| folder.md | 95%+ | Closure table は `folder_paths` 名で実装（domain docでは `FolderClosure`）。MaxDepth=20 実装済み |
| group.md | 100% | Group/Membership/Invitation すべて実装。GroupRole: viewer/contributor/owner。InvitationExpiry: 7日 |
| permission.md | 95%+ | PermissionGrant/Relationship テーブル実装済み。Relation CHECK制約は owner/member/parent のみ（ロール別リレーションはpermission_grantsで管理） |
| sharing.md | 100% | ShareLink/ShareLinkAccess 完全実装。SharePermission: read/write。パスワード保護・有効期限・アクセス回数すべて対応 |

**結論**: 実装とドメイン定義の整合性は極めて高い。重大な不整合なし。

---

## クイックリンク

### ドメイン参照

| コンテキスト | ドメインファイル |
|-------------|-----------------|
| Identity | [user.md](../03-domains/user.md) |
| Storage | [file.md](../03-domains/file.md), [folder.md](../03-domains/folder.md) |
| Authorization | [permission.md](../03-domains/permission.md) |
| Sharing | [sharing.md](../03-domains/sharing.md) |
| Collaboration | [group.md](../03-domains/group.md) |

### アーキテクチャ参照

| カテゴリ | ドキュメント |
|---------|------------|
| システム全体 | [SYSTEM.md](../02-architecture/SYSTEM.md) |
| バックエンド | [BACKEND.md](../02-architecture/BACKEND.md) |
| フロントエンド | [FRONTEND.md](../02-architecture/FRONTEND.md) |
| データベース | [DATABASE.md](../02-architecture/DATABASE.md) |
| API | [API.md](../02-architecture/API.md) |
| セキュリティ | [SECURITY.md](../02-architecture/SECURITY.md) |

### テンプレート

| テンプレート | 用途 |
|------------|------|
| [FEATURE_SPEC_TEMPLATE.md](./templates/FEATURE_SPEC_TEMPLATE.md) | Feature spec作成時のテンプレート |
| [TEST_SPEC_TEMPLATE.md](./templates/TEST_SPEC_TEMPLATE.md) | テストケース設計テンプレート |
