# CLAUDE.md

Claude Code がこのリポジトリで作業する際のガイドです。

---

## プロジェクト概要

**GC Storage** はクラウドストレージシステムです。ファイル管理、共有、チームコラボレーション機能を提供します。

### 技術スタック

| レイヤー | 技術 |
|---------|------|
| Backend | Go 1.22+ / Echo v4 / Clean Architecture |
| Frontend | React 19 / TanStack Router & Query / Zustand / Tailwind CSS |
| Database | PostgreSQL 16 |
| Cache | Redis 7 |
| Storage | MinIO (S3互換) |
| Auth | JWT + OAuth 2.0 (Google, GitHub) |
| Task Runner | Taskfile |

### 境界づけられたコンテキスト

```
┌─────────────────┐     ┌─────────────────┐
│ Identity        │     │ Storage         │
│ - User          │────▶│ - File          │
│ - Session       │     │ - Folder        │
│ - OAuth         │     │ - Version       │
└─────────────────┘     └─────────────────┘
         │                      │
         ▼                      ▼
┌─────────────────┐     ┌─────────────────┐
│ Collaboration   │     │ Authorization   │
│ - Group         │────▶│ - Permission    │
│ - Membership    │     │ - Relationship  │
│ - Invitation    │     │ (PBAC + ReBAC)  │
└─────────────────┘     └─────────────────┘
                               │
                               ▼
                        ┌─────────────────┐
                        │ Sharing         │
                        │ - ShareLink     │
                        │ - AccessLog     │
                        └─────────────────┘
```

---

## クイックスタート

```bash
# 前提: Task がインストールされていること
# brew install go-task

# ワンコマンドで全環境起動
task dev
```

これで以下が起動します:
- PostgreSQL (localhost:5432)
- Redis (localhost:6379)
- MinIO (localhost:9000, Console: localhost:9001)
- MailHog (localhost:8025)
- Backend API (localhost:8080)
- Frontend (localhost:3000)

---

## プロジェクト構造

```
gc-storage/
├── backend/                  # Go バックエンド
│   ├── cmd/api/main.go       # エントリーポイント
│   ├── internal/
│   │   ├── domain/           # ドメイン層（エンティティ、リポジトリIF）
│   │   ├── usecase/          # ユースケース層（ビジネスロジック）
│   │   ├── interface/        # インターフェース層（ハンドラ、DTO）
│   │   └── infrastructure/   # インフラ層（リポジトリ実装）
│   │       └── database/
│   │           ├── migrations/   # DBマイグレーション (golang-migrate)
│   │           ├── queries/      # SQLC クエリ定義
│   │           └── sqlcgen/      # SQLC 生成コード
│   ├── tests/integration/    # 統合テスト
│   ├── pkg/                  # 共有パッケージ
│   ├── go.mod
│   └── .air.toml             # Hot Reload 設定
│
├── frontend/                 # React フロントエンド
│   ├── src/
│   │   ├── app/routes/       # TanStack Router
│   │   ├── components/       # UIコンポーネント
│   │   ├── features/         # 機能モジュール
│   │   ├── stores/           # Zustand ストア
│   │   └── lib/              # ユーティリティ、APIクライアント
│   ├── package.json
│   └── vite.config.ts
│
├── docs/                     # ドキュメント
│   ├── 01-policies/          # 開発ポリシー
│   ├── 02-architecture/      # アーキテクチャ設計
│   ├── 03-domains/           # ドメイン定義
│   ├── 04-specs/             # 機能仕様
│   └── 05-operations/        # 運用ドキュメント
│
├── docker-compose.yml        # ローカルインフラ
├── Taskfile.yml              # タスクランナー
├── .env.local                # ローカル環境変数（git管理）
└── .env.sample               # 本番用テンプレート
```

---

## タスクコマンド

### クイックリファレンス

```bash
task                     # 利用可能なタスク一覧
task dev                 # 全環境をワンコマンド起動
task check               # lint + test (クイック検証)
task doctor              # ツールのインストール確認
```

### 開発環境

```bash
task dev                 # 全環境起動 (infra + backend + frontend)
task dev:backend         # バックエンドのみ起動 (infra + backend)
task dev:frontend        # フロントエンドのみ起動
task infra:up            # インフラのみ起動
task infra:down          # インフラ停止（データ保持）
task infra:destroy       # インフラ停止・ボリューム削除（データ削除）
task infra:logs          # ログ確認
task infra:status        # ステータス確認
```

### Backend (Go)

```bash
task backend:dev         # Air で Hot Reload 起動
task backend:run         # Hot Reload なしで起動
task backend:build       # バイナリビルド
task backend:test        # ユニットテスト実行
task backend:test-integration  # 統合テスト実行
task backend:test-coverage     # カバレッジレポート生成
task backend:lint        # golangci-lint
task backend:lint-fix    # golangci-lint (自動修正)
task backend:fmt         # コードフォーマット
task backend:sqlc        # SQL → Go コード生成
task backend:mocks       # モック生成
```

### Frontend (React)

```bash
task frontend:dev        # Vite dev server
task frontend:build      # プロダクションビルド
task frontend:preview    # ビルドプレビュー
task frontend:test       # Vitest 実行
task frontend:test-watch # ウォッチモード
task frontend:test-coverage    # カバレッジ
task frontend:lint       # ESLint
task frontend:fmt        # Prettier フォーマット
```

### テスト

```bash
task test                # 全テスト (unit + integration)
task test:unit           # ユニットテストのみ
task test:integration    # 統合テストのみ
task check               # lint + test (クイック検証)
task ci                  # CI パイプライン (lint + test + build)
task ci:full             # フル CI (統合テスト含む)
```

### Database

```bash
task migrate:up          # マイグレーション適用
task migrate:down        # ロールバック（1つ）
task migrate:reset       # 全ロールバック（危険）
task migrate:create NAME=xxx  # 新規マイグレーション作成
task migrate:version     # 現在のバージョン確認
task db:connect          # psql でDB接続
task db:reset            # DB リセット（drop + create + migrate）
```

### セットアップ

```bash
task setup               # ツール + 依存関係インストール
task setup:tools         # Go ツールインストール
task setup:deps          # 依存関係インストール
task doctor              # ツールインストール確認
task clean               # ビルド成果物削除
```

### ブラウザで開く

```bash
task open:frontend       # フロントエンド
task open:api            # API
task open:minio          # MinIO コンソール
task open:mailhog        # MailHog UI
```

---

## 環境変数

| ファイル | 用途 | Git管理 |
|---------|------|---------|
| `.env.local` | ローカル開発用（固定値） | ✅ |
| `.env.sample` | 本番用テンプレート | ✅ |
| `.env` | 本番/ステージング用 | ❌ |

ローカル開発では `.env.local` が Taskfile により自動で読み込まれます。

---

## ドキュメントガイド

### 読むべきドキュメント

| タスク | 参照ドキュメント |
|-------|-----------------|
| プロジェクト理解 | `docs/03-domains/EVENT_STORMING.md` |
| コード実装 | `docs/01-policies/CODING_STANDARDS.md` → `docs/02-architecture/BACKEND.md` |
| 新機能開発 | `docs/01-policies/TDD_WORKFLOW.md` → `docs/03-domains/*.md` → `docs/02-architecture/*.md` |
| テスト作成 | `docs/01-policies/TDD_WORKFLOW.md` → `docs/01-policies/TESTING.md` |
| テストケース設計 | `docs/04-specs/templates/TEST_SPEC_TEMPLATE.md` |
| 環境構築 | `docs/01-policies/SETUP.md` |

### ドメイン定義（03-domains/）

| ファイル | 内容 |
|---------|------|
| EVENT_STORMING.md | イベントストーミング結果、コンテキストマップ |
| user.md | User, OAuthAccount, Session |
| group.md | Group, Membership, Invitation |
| folder.md | Folder, FolderPath |
| file.md | File, FileVersion, UploadSession |
| permission.md | PermissionGrant, Relationship (Zanzibar) |
| sharing.md | ShareLink, ShareLinkAccess |

---

## コーディング規約

### Go

```go
// ファイル名: snake_case.go
// パッケージ名: lowercase, singular
// 構造体/メソッド: PascalCase
// 変数: camelCase
// 最初の引数: ctx context.Context

// テスト命名
func TestFunctionName_Scenario_ExpectedBehavior(t *testing.T) {}
```

### TypeScript

```typescript
// ファイル名: kebab-case.tsx
// コンポーネント: PascalCase
// フック: useXxx
// イベントハンドラ: handleXxxYyy
```

### SQL

```sql
-- テーブル名: 複数形 snake_case (users, file_versions)
-- インデックス: idx_{table}_{columns}
-- 外部キー: fk_{table}_{ref_table}
```

---

## アーキテクチャ

### Clean Architecture (4層)

```
HTTP Request
     │
     ▼
┌─────────────────┐
│   Interface     │  Handler, Middleware, DTO
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│   UseCase       │  Command (書込), Query (読取) - CQRS
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│   Domain        │  Entity, Repository IF, Domain Service
└────────▲────────┘
         │ Interface経由
┌────────┴────────┐
│ Infrastructure  │  Repository実装, 外部サービス
└─────────────────┘
```

**依存関係のルール:**
- 依存は常に内側（Domain）に向かう
- Domain 層は外部依存を持たない
- Infrastructure は Domain のインターフェースを実装

### UseCase層 CQRS パターン

UseCase層はCQRS（Command Query Responsibility Segregation）パターンを採用:

```
usecase/
├── auth/
│   ├── command/           # 書き込み操作
│   │   ├── register.go    # RegisterCommand
│   │   ├── login.go       # LoginCommand
│   │   └── logout.go      # LogoutCommand
│   └── query/             # 読み取り操作
│       └── get_user.go    # GetUserQuery
└── storage/
    ├── command/
    │   └── create_folder.go
    └── query/
        └── list_folder_contents.go
```

**命名規則:**

| 要素 | パターン | 例 |
|------|---------|-----|
| Command構造体 | `{Action}Command` | `LoginCommand` |
| Query構造体 | `{Action}Query` | `GetUserQuery` |
| コンストラクタ | `New{StructName}` | `NewLoginCommand` |
| Input/Output | `{Action}Input/Output` | `LoginInput` |
| メソッド | `Execute` | すべて統一 |

**分類基準:**
- **Command**: 副作用（状態変更）がある操作（Create, Update, Delete, Login, Logout）
- **Query**: 副作用がない操作（Read, List, Get）

### 状態管理（Frontend）

| 状態の種類 | 解決策 |
|-----------|--------|
| サーバー状態 | TanStack Query |
| URL状態 | TanStack Router |
| グローバルUI状態 | Zustand |
| ローカルUI状態 | useState |

---

## AI コーディングガイドライン

1. **型安全性** - `any`, `interface{}` を避け、明示的な型を使用
2. **副作用の最小化** - 純粋関数を優先、副作用は分離
3. **テスタビリティ** - DI、インターフェース抽象化
4. **YAGNI** - 過剰設計を避け、現在のタスクに必要な最小限の複雑さ
5. **ドキュメント整合性** - 実装は設計ドキュメントに従う

### 実装時のチェックリスト

- [ ] TDDワークフロー（`docs/01-policies/TDD_WORKFLOW.md`）に従っているか
- [ ] テストケースを先に設計・作成したか（RED）
- [ ] ドメイン定義（`docs/03-domains/*.md`）を確認したか
- [ ] 既存のコードパターンに従っているか
- [ ] 全テストがPASSすることを確認したか（GREEN）
- [ ] エラーハンドリングは適切か
- [ ] セキュリティ（入力検証、認可）を考慮したか
- [ ] リファクタリング後も全テストがPASSするか（REFACTOR）

### TDDワークフロー（必須）

新機能開発では、TDD（テスト駆動開発）アプローチを採用します。

```
RED → GREEN → REFACTOR
```

1. **RED**: テストケース設計 → テストコード作成 → 全テストFAIL確認
2. **GREEN**: 最小実装 → 全テストPASS確認
3. **REFACTOR**: コード改善 → 全テストPASS維持

詳細は `docs/01-policies/TDD_WORKFLOW.md` を参照。
テストケース設計には `docs/04-specs/templates/TEST_SPEC_TEMPLATE.md` を使用。
