# GC Storage 開発環境セットアップガイド

## 概要

本ドキュメントでは、GC Storageの開発環境をローカルマシンにセットアップする手順を説明します。

---

## 1. 前提条件

### 1.1 必須ソフトウェア

| ソフトウェア | バージョン | 用途 |
|-------------|-----------|------|
| Go | 1.22+ | バックエンド開発 |
| Node.js | 20.x LTS | フロントエンド開発 |
| pnpm | 8.x+ | パッケージマネージャー |
| Docker | 24.x+ | コンテナ実行環境 |
| Docker Compose | 2.x+ | ローカル開発環境構築 |
| Task | 3.x+ | タスクランナー（Makefile代替） |
| Git | 2.x+ | バージョン管理 |

### 1.2 推奨ツール

| ツール | 用途 |
|--------|------|
| VS Code | エディタ（推奨拡張機能は後述） |
| Go拡張機能 | Go開発支援 |
| ESLint | TypeScript/JavaScript Linting |
| Prettier | コードフォーマット |
| TablePlus / DBeaver | データベースクライアント |
| Postman / Bruno | API テスト |

### 1.3 VS Code 推奨拡張機能

```json
{
  "recommendations": [
    "golang.go",
    "bradlc.vscode-tailwindcss",
    "dbaeumer.vscode-eslint",
    "esbenp.prettier-vscode",
    "ms-azuretools.vscode-docker",
    "redhat.vscode-yaml"
  ]
}
```

---

## 2. リポジトリのクローン

```bash
# HTTPS
git clone https://github.com/Hiro-mackay/gc-storage.git

# SSH
git clone git@github.com:Hiro-mackay/gc-storage.git

cd gc-storage
```

---

## 3. Docker Compose による環境構築

### 3.1 Docker Compose 構成

開発環境では以下のサービスが起動します:

| サービス | ポート | 説明 |
|---------|--------|------|
| postgres | 5432 | PostgreSQL データベース |
| redis | 6379 | Redis キャッシュ |
| minio | 9000, 9001 | MinIO オブジェクトストレージ |
| mailhog | 1025, 8025 | メールテスト用SMTPサーバー |

### 3.2 環境変数の設定

本プロジェクトでは環境変数を以下のように管理します:

| ファイル | 用途 | Git管理 |
|---------|------|---------|
| `.env.local` | ローカル開発用（固定値） | ✅ 管理対象 |
| `.env.sample` | 本番用テンプレート | ✅ 管理対象 |
| `.env` | 本番/ステージング用 | ❌ 管理対象外 |

**ローカル開発の場合:**

`.env.local` がリポジトリに含まれているため、追加の設定は不要です。
Taskfile が自動的に `.env.local` を読み込みます。

```bash
# .env.local の内容（リポジトリに含まれる）
DATABASE_URL=postgres://postgres:postgres@localhost:5432/gc_storage
REDIS_URL=redis://localhost:6379
MINIO_ENDPOINT=localhost:9000
MINIO_ACCESS_KEY=minioadmin
MINIO_SECRET_KEY=minioadmin
MINIO_BUCKET=gc-storage
SMTP_HOST=localhost
SMTP_PORT=1025
JWT_SECRET=local-development-secret-key-do-not-use-in-production
# OAuth (ローカル用ダミー)
GOOGLE_CLIENT_ID=dummy-google-client-id
GOOGLE_CLIENT_SECRET=dummy-google-client-secret
GITHUB_CLIENT_ID=dummy-github-client-id
GITHUB_CLIENT_SECRET=dummy-github-client-secret
```

**本番環境の場合:**

`.env.sample` をコピーして `.env` を作成し、実際の値を設定:

```bash
cp .env.sample .env
# .env を編集して本番用の値を設定
```

### 3.3 Docker Compose の起動

```bash
# インフラサービスの起動（バックグラウンド）
docker compose up -d

# ログの確認
docker compose logs -f

# 特定サービスのログ
docker compose logs -f postgres
```

### 3.4 サービスの起動確認

```bash
# PostgreSQL
docker compose exec postgres pg_isready -U gc_storage

# Redis
docker compose exec redis redis-cli ping

# MinIO
curl http://localhost:9000/minio/health/live
```

### 3.5 MinIO の初期設定

1. MinIO Console にアクセス: http://localhost:9001
2. ログイン（Access Key / Secret Key）
3. バケット `gc-storage-dev` を作成

または CLI で作成:

```bash
# MinIO Client のインストール
brew install minio/stable/mc  # macOS
# または
docker run --rm -it --entrypoint /bin/sh minio/mc

# エイリアス設定
mc alias set local http://localhost:9000 minio_dev_access_key minio_dev_secret_key

# バケット作成
mc mb local/gc-storage-dev
```

---

## 4. バックエンドのセットアップ

### 4.1 依存パッケージのインストール

```bash
cd backend

# Go モジュールのダウンロード
go mod download

# 開発ツールのインストール
go install github.com/air-verse/air@latest
go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
```

### 4.2 データベースマイグレーション

```bash
# マイグレーションの実行
task migrate:up

# または直接実行
migrate -path migrations -database "$DATABASE_URL" up
```

### 4.3 sqlc コード生成

```bash
# SQLからGoコードを生成
task backend:sqlc

# または直接実行
sqlc generate
```

### 4.4 開発サーバーの起動

```bash
# Air によるホットリロード
task backend:dev

# または直接実行
air

# 通常起動（ホットリロードなし）
go run cmd/api/main.go
```

サーバーが起動したら http://localhost:8080/health で確認できます。

---

## 5. フロントエンドのセットアップ

### 5.1 依存パッケージのインストール

```bash
cd frontend

# pnpm のインストール（未インストールの場合）
npm install -g pnpm

# 依存パッケージのインストール
pnpm install
```

### 5.2 環境変数の設定

フロントエンドの環境変数は `.env.local` で管理されています（Git管理対象）。

```bash
# frontend/.env.local の内容
VITE_API_BASE_URL=http://localhost:8080/api/v1
VITE_MINIO_ENDPOINT=http://localhost:9000
```

### 5.3 開発サーバーの起動

```bash
# 開発サーバー起動
pnpm dev
```

フロントエンドは http://localhost:3000 でアクセスできます。

---

## 6. 統合開発環境

### 6.1 ワンコマンド起動（推奨）

プロジェクトルートの Taskfile.yml を使用:

```bash
# Task のインストール（未インストールの場合）
brew install go-task  # macOS
# または https://taskfile.dev/installation/ を参照

# 全サービスをワンコマンドで起動
task dev
```

`task dev` で以下が自動的に実行されます:
1. Docker Compose でインフラ起動（PostgreSQL, Redis, MinIO, MailHog）
2. データベースマイグレーション適用
3. バックエンド（Air でホットリロード）
4. フロントエンド（Vite dev server）

### 6.2 個別起動

```bash
# インフラのみ
task infra:up

# バックエンドのみ
task backend:dev

# フロントエンドのみ
task frontend:dev

# インフラ停止
task infra:down
```

### 6.3 開発時のワークフロー（手動）

```bash
# 1. インフラの起動
task infra:up

# 2. マイグレーション確認
task migrate:up

# 3. バックエンド起動（ターミナル1）
task backend:dev

# 4. フロントエンド起動（ターミナル2）
task frontend:dev
```

---

## 7. よく使うコマンド

### 7.1 タスク一覧を確認

```bash
# 利用可能なタスク一覧
task --list
```

### 7.2 データベース操作

```bash
# マイグレーション
task migrate:up              # 適用
task migrate:down            # ロールバック（1つ）
task migrate:create NAME=xxx # 新規作成

# sqlc
task backend:sqlc            # コード生成

# データベース接続
docker compose exec postgres psql -U postgres -d gc_storage
```

### 7.3 テスト

```bash
# バックエンド
task backend:test            # 全テスト
task backend:test-coverage   # カバレッジ付き

# フロントエンド
task frontend:test           # 全テスト
task frontend:test-watch     # ウォッチモード
```

### 7.4 Lint & Format

```bash
# バックエンド
task backend:lint            # golangci-lint
task backend:fmt             # go fmt

# フロントエンド
task frontend:lint           # ESLint
task frontend:format         # Prettier
```

### 7.5 ビルド

```bash
# バックエンド
task backend:build           # バイナリビルド

# フロントエンド
task frontend:build          # プロダクションビルド
```

---

## 8. Docker Compose 詳細設定

### 8.1 docker-compose.yml

```yaml
services:
  postgres:
    image: postgres:16-alpine
    container_name: gc-storage-postgres
    environment:
      POSTGRES_USER: postgres
      POSTGRES_PASSWORD: postgres
      POSTGRES_DB: gc_storage
    ports:
      - "5432:5432"
    volumes:
      - postgres_data:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U postgres"]
      interval: 10s
      timeout: 5s
      retries: 5

  redis:
    image: redis:7-alpine
    container_name: gc-storage-redis
    ports:
      - "6379:6379"
    volumes:
      - redis_data:/data
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 10s
      timeout: 5s
      retries: 5

  minio:
    image: minio/minio:latest
    container_name: gc-storage-minio
    command: server /data --console-address ":9001"
    environment:
      MINIO_ROOT_USER: minioadmin
      MINIO_ROOT_PASSWORD: minioadmin
    ports:
      - "9000:9000"
      - "9001:9001"
    volumes:
      - minio_data:/data
    healthcheck:
      test: ["CMD", "mc", "ready", "local"]
      interval: 30s
      timeout: 20s
      retries: 3

  mailhog:
    image: mailhog/mailhog:latest
    container_name: gc-storage-mailhog
    ports:
      - "1025:1025"
      - "8025:8025"

volumes:
  postgres_data:
  redis_data:
  minio_data:
```

### 8.2 データの永続化とリセット

```bash
# ボリュームの確認
docker volume ls | grep gc-storage

# データの完全リセット
docker compose down -v

# 特定サービスのみリセット
docker compose rm -s -v postgres
docker volume rm gc-storage_postgres_data
```

---

## 9. トラブルシューティング

### 9.1 ポートの競合

```bash
# 使用中のポートを確認
lsof -i :5432
lsof -i :6379
lsof -i :9000

# プロセスを終了（PIDを指定）
kill -9 <PID>
```

### 9.2 Docker の問題

```bash
# コンテナの状態確認
docker compose ps

# コンテナの再作成
docker compose up -d --force-recreate

# イメージの再ビルド
docker compose build --no-cache
```

### 9.3 Go モジュールの問題

```bash
# キャッシュのクリア
go clean -modcache

# 依存関係の再取得
go mod tidy
go mod download
```

### 9.4 フロントエンドの問題

```bash
# node_modules の削除と再インストール
rm -rf node_modules
pnpm install

# キャッシュのクリア
pnpm store prune
```

### 9.5 マイグレーションエラー

```bash
# dirty 状態の解消
migrate -path migrations -database "${DATABASE_URL}" force <version>

# バージョン確認
migrate -path migrations -database "${DATABASE_URL}" version
```

---

## 10. 次のステップ

- [CONTRIBUTING.md](./CONTRIBUTING.md) - 開発ガイドライン
- [TESTING.md](./TESTING.md) - テスト戦略
- [BACKEND.md](./BACKEND.md) - バックエンド設計
- [FRONTEND.md](./FRONTEND.md) - フロントエンド設計
- [API.md](./API.md) - API設計

---

## 更新履歴

| 日付 | バージョン | 内容 |
|------|-----------|------|
| 2026-01-17 | 1.0.0 | 初版作成 |
