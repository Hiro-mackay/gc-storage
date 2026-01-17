# GC Storage アーキテクチャ設計書

## 概要

GC Storageは、クラウドベースのファイルストレージシステムです。本ドキュメントでは、システム全体のアーキテクチャ、コンポーネント構成、データフローについて説明します。

## 技術スタック

| レイヤー | 技術 |
|---------|------|
| Backend | Go 1.22+ |
| Frontend | React 18 + TanStack (Query, Router) |
| UI Library | shadcn/ui + Tailwind CSS |
| Database | PostgreSQL 16 |
| Cache | Redis 7 |
| Object Storage | MinIO (S3互換) |
| Container Orchestration | Kubernetes |
| 認証 | JWT + OAuth2.0 (Google, GitHub) |

---

## 1. システムアーキテクチャ

### 1.1 全体構成図

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              Kubernetes Cluster                             │
│  ┌────────────────────────────────────────────────────────────────────────┐ │
│  │                           Ingress Controller                           │ │
│  │                        (NGINX / Traefik)                               │ │
│  └────────────────────────────────────────────────────────────────────────┘ │
│                    │                              │                         │
│                    ▼                              ▼                         │
│  ┌─────────────────────────────┐  ┌──────────────────────────────────────┐  │
│  │      Frontend Service       │  │         API Service                  │  │
│  │  ┌───────────────────────┐  │  │  ┌───────────────────────────────┐   │  │
│  │  │    React SPA          │  │  │  │      Go API Server            │   │  │
│  │  │    (Nginx静的配信)     │  │  │  │      (3 replicas)             │   │  │
│  │  └───────────────────────┘  │  │  └───────────────────────────────┘   │  │
│  └─────────────────────────────┘  └──────────────────────────────────────┘  │
│                                                  │                          │
│                    ┌────────────────────────────┼─────────────────────┐     │
│                    ▼                            ▼                     ▼     │
│  ┌──────────────────────┐  ┌──────────────────────┐  ┌────────────────────┐ │
│  │   PostgreSQL         │  │      Redis           │  │     MinIO          │ │
│  │   (Primary/Replica)  │  │   (Cluster Mode)     │  │   (Distributed)    │ │
│  │                      │  │                      │  │                    │ │
│  │   PVC: 100Gi         │  │   PVC: 10Gi          │  │   PVC: 1Ti         │ │
│  └──────────────────────┘  └──────────────────────┘  └────────────────────┘ │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
                                      │
                    ┌─────────────────┼─────────────────┐
                    ▼                 ▼                 ▼
            ┌─────────────┐  ┌─────────────────┐  ┌──────────────┐
            │   Google    │  │     GitHub      │  │   External   │
            │   OAuth     │  │     OAuth       │  │   Clients    │
            └─────────────┘  └─────────────────┘  └──────────────┘
```

### 1.2 コンポーネント概要

| コンポーネント | 説明 | スケーリング戦略 |
|---------------|------|-----------------|
| Ingress | 外部トラフィックのルーティング、TLS終端 | 冗長化 |
| Frontend | React SPAの静的ファイル配信 | HPA (CPU 70%) |
| API Server | RESTful API、ビジネスロジック | HPA (CPU 70%, Memory 80%) |
| PostgreSQL | メタデータ、ユーザー情報の永続化 | Primary-Replica構成 |
| Redis | セッション、キャッシュ、レート制限 | Cluster Mode |
| MinIO | ファイル実体の保存（S3互換） | Distributed Mode |

---

## 2. データフロー

### 2.1 ファイルアップロードフロー

```
┌────────┐     ┌─────────┐     ┌──────────┐     ┌───────────┐     ┌───────┐
│ Client │────▶│ Ingress │────▶│ API      │────▶│ PostgreSQL│     │ MinIO │
└────────┘     └─────────┘     │ Server   │     └───────────┘     └───────┘
                               └──────────┘                           ▲
                                    │                                 │
                                    └─────────────────────────────────┘
                                         Presigned URL / Direct Upload

【詳細フロー】
1. クライアント → API: アップロードリクエスト（ファイルメタデータ）
2. API → PostgreSQL: ファイルメタデータ登録（status: pending）
3. API → MinIO: Presigned PUT URL生成
4. API → クライアント: Presigned URL返却
5. クライアント → MinIO: ファイル本体を直接アップロード
6. クライアント → API: アップロード完了通知
7. API → PostgreSQL: ステータス更新（status: active）
```

### 2.2 ファイルダウンロードフロー

```
┌────────┐     ┌─────────┐     ┌──────────┐     ┌──────────┐
│ Client │────▶│ Ingress │────▶│ API      │────▶│ PostgreSQL│
└────────┘     └─────────┘     │ Server   │     └──────────┘
    │                          └──────────┘
    │                               │
    │                               ▼
    │                          ┌───────┐
    │◀─────────────────────────│ MinIO │
    │    Presigned GET URL     └───────┘

【詳細フロー】
1. クライアント → API: ダウンロードリクエスト
2. API → PostgreSQL: 権限チェック
3. API → MinIO: Presigned GET URL生成（有効期限: 15分）
4. API → クライアント: Presigned URL返却
5. クライアント → MinIO: ファイルを直接ダウンロード
```

### 2.3 マルチパートアップロードフロー（大容量ファイル）

```
【初期化フェーズ】
1. クライアント → API: マルチパートアップロード開始リクエスト
2. API → MinIO: CreateMultipartUpload
3. API → PostgreSQL: アップロードセッション作成
4. API → クライアント: uploadId返却

【アップロードフェーズ】
5. クライアント → API: 各パートのPresigned URL要求
6. API → MinIO: Presigned URL生成（各パート用）
7. クライアント → MinIO: パート並列アップロード（最大5並列）

【完了フェーズ】
8. クライアント → API: 完了リクエスト（ETag一覧）
9. API → MinIO: CompleteMultipartUpload
10. API → PostgreSQL: ファイルステータス更新
```

---

## 3. レイヤー構成

### 3.1 アプリケーションレイヤー

```
┌─────────────────────────────────────────────────────────────┐
│                      Presentation Layer                     │
│  ┌────────────────────────────────────────────────────────┐ │
│  │  HTTP Handlers │ Middleware │ Request/Response DTOs    │ │
│  └────────────────────────────────────────────────────────┘ │
├─────────────────────────────────────────────────────────────┤
│                       Application Layer                     │
│  ┌────────────────────────────────────────────────────────┐ │
│  │  Use Cases │ Application Services │ DTOs/Commands      │ │
│  └────────────────────────────────────────────────────────┘ │
├─────────────────────────────────────────────────────────────┤
│                        Domain Layer                         │
│  ┌────────────────────────────────────────────────────────┐ │
│  │  Entities │ Value Objects │ Domain Services │ Events   │ │
│  └────────────────────────────────────────────────────────┘ │
├─────────────────────────────────────────────────────────────┤
│                     Infrastructure Layer                    │
│  ┌────────────────────────────────────────────────────────┐ │
│  │  Repository Impl │ External Services │ DB │ Storage    │ │
│  └────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────┘
```

### 3.2 依存関係の方向

```
HTTP Request
    │
    ▼
┌────────────────────┐
│   Handler          │  ─────┐
└────────────────────┘       │
    │                        │
    ▼                        │  依存関係は
┌────────────────────┐       │  常に内側に向かう
│   UseCase          │  ─────┤
└────────────────────┘       │
    │                        │
    ▼                        │
┌────────────────────┐       │
│   Domain           │  ◀────┘
│   (Entity/Service) │
└────────────────────┘
    ▲
    │ Interface経由
    │
┌────────────────────┐
│   Infrastructure   │
│   (Repository等)   │
└────────────────────┘
```

---

## 4. 通信プロトコル

### 4.1 外部通信

| 通信経路 | プロトコル | 認証方式 |
|---------|-----------|---------|
| Client ↔ Ingress | HTTPS (TLS 1.3) | JWT Bearer Token |
| Client ↔ MinIO | HTTPS | Presigned URL |
| API ↔ OAuth Provider | HTTPS | OAuth 2.0 |

### 4.2 内部通信

| 通信経路 | プロトコル | 認証方式 |
|---------|-----------|---------|
| API ↔ PostgreSQL | TCP (5432) | Username/Password |
| API ↔ Redis | TCP (6379) | Password |
| API ↔ MinIO | HTTP/HTTPS (9000) | Access Key/Secret Key |

---

## 5. 可用性設計

### 5.1 冗長化構成

| コンポーネント | 構成 | 可用性目標 |
|---------------|------|-----------|
| API Server | 3 replicas, Rolling Update | 99.9% |
| PostgreSQL | Primary + 2 Replicas | 99.95% |
| Redis | 6 nodes (3 masters, 3 replicas) | 99.9% |
| MinIO | 4 nodes (Erasure Coding) | 99.99% |

### 5.2 障害復旧

```
【PostgreSQL障害時】
- Replica自動昇格（Patroni/pgpool経由）
- フェイルオーバー時間: < 30秒

【Redis障害時】
- Cluster自動再構成
- Sentinel経由の自動フェイルオーバー

【MinIO障害時】
- Erasure Codingにより2ノード障害まで許容
- 自動リバランシング
```

---

## 6. 監視・ログ

### 6.1 メトリクス収集

```
┌──────────────┐     ┌────────────────┐     ┌─────────────┐
│  API Server  │────▶│   Prometheus   │────▶│   Grafana   │
│  /metrics    │     │                │     │             │
└──────────────┘     └────────────────┘     └─────────────┘

収集メトリクス:
- HTTP Request Rate/Latency/Error Rate
- Database Connection Pool
- Redis Hit/Miss Rate
- MinIO Operations/Bandwidth
- Go Runtime (Goroutines, Memory, GC)
```

### 6.2 ログ集約

```
┌──────────────┐     ┌────────────────┐     ┌─────────────┐
│  API Server  │────▶│   Fluentd/     │────▶│   Elastic   │
│  (stdout)    │     │   Fluent Bit   │     │   Search    │
└──────────────┘     └────────────────┘     └─────────────┘

ログフォーマット: JSON (構造化ログ)
ログレベル: DEBUG, INFO, WARN, ERROR
```

---

## 7. 開発環境

### 7.1 ローカル開発構成

```yaml
# docker-compose.yml
services:
  api:
    build: ./backend
    ports:
      - "8080:8080"
    environment:
      - DATABASE_URL=postgres://...
      - REDIS_URL=redis://...
      - MINIO_ENDPOINT=minio:9000

  frontend:
    build: ./frontend
    ports:
      - "3000:3000"

  postgres:
    image: postgres:16
    ports:
      - "5432:5432"

  redis:
    image: redis:7
    ports:
      - "6379:6379"

  minio:
    image: minio/minio
    ports:
      - "9000:9000"
      - "9001:9001"
```

### 7.2 開発ツール

| ツール | 用途 |
|-------|------|
| Air | Go Hot Reload |
| Vite | Frontend Dev Server |
| golang-migrate | Database Migration |
| sqlc | SQL → Go Code Generation |
| oapi-codegen | OpenAPI → Go Code Generation |

---

## 関連ドキュメント

- [バックエンド設計](./BACKEND.md)
- [データベース設計](./DATABASE.md)
- [API設計](./API.md)
- [フロントエンド設計](./FRONTEND.md)
- [インフラストラクチャ設計](./INFRASTRUCTURE.md)
- [セキュリティ設計](./SECURITY.md)
