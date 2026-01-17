# 技術スタック

> プロジェクトで使用する技術・バージョン・選定理由を定義します。

---

## 1. 言語・ランタイム

| 技術 | バージョン | 用途 |
|------|-----------|------|
| Go | 1.22+ | バックエンドAPI |
| Node.js | 20 LTS | フロントエンドビルド |
| TypeScript | 5.x | フロントエンド開発 |

---

## 2. バックエンド

| カテゴリ | 技術 | バージョン | 用途 |
|---------|------|-----------|------|
| Web Framework | Echo | v4 | HTTPルーティング・ミドルウェア |
| DB Driver | pgx | v5 | PostgreSQL接続 |
| SQL Generator | sqlc | latest | 型安全なSQLクエリ生成 |
| Validation | go-playground/validator | v10 | リクエストバリデーション |
| JWT | golang-jwt | v5 | トークン生成・検証 |
| Logging | slog | stdlib | 構造化ログ |
| Testing | testify | v1 | アサーション・モック |

### 使用しないライブラリ

| ライブラリ | 理由 |
|-----------|------|
| GORM | sqlcで型安全なクエリ生成を行うため |
| gin | Echoを標準として採用 |
| logrus | Go 1.21+標準のslogを使用 |

---

## 3. フロントエンド

| カテゴリ | 技術 | バージョン | 用途 |
|---------|------|-----------|------|
| Framework | React | 18.x | UIライブラリ |
| Build Tool | Vite | 5.x | ビルド・開発サーバー |
| Routing | TanStack Router | 1.x | 型安全なルーティング |
| Data Fetching | TanStack Query | 5.x | サーバー状態管理 |
| State Management | Zustand | 4.x | クライアント状態管理 |
| UI Components | shadcn/ui | latest | コンポーネントライブラリ |
| Styling | Tailwind CSS | 3.x | ユーティリティCSS |
| Form Validation | Zod | latest | スキーマバリデーション |
| HTTP Client | ky | 1.x | APIクライアント |
| Testing | Vitest | latest | ユニットテスト |
| Testing | Testing Library | latest | コンポーネントテスト |
| E2E Testing | Playwright | latest | E2Eテスト |

### 使用しないライブラリ

| ライブラリ | 理由 |
|-----------|------|
| React Hook Form | shadcn/ui Fieldに準拠するため |
| Redux | TanStack Query + Zustandで十分なため |
| React Context (グローバル状態) | グローバル状態管理にはZustandを使用 |
| axios | kyを標準として採用 |

---

## 4. データベース・ストレージ

| 技術 | バージョン | 用途 |
|------|-----------|------|
| PostgreSQL | 16 | メインデータベース |
| Redis | 7 | キャッシュ・セッション |
| MinIO | latest | オブジェクトストレージ（S3互換） |

---

## 5. インフラストラクチャ

| 技術 | バージョン | 用途 |
|------|-----------|------|
| Docker | 24+ | コンテナ化 |
| Kubernetes | 1.28+ | コンテナオーケストレーション |
| NGINX Ingress | latest | Ingressコントローラー |
| Prometheus | latest | メトリクス収集 |
| Grafana | latest | 可視化・ダッシュボード |
| Fluentd | latest | ログ集約 |

---

## 6. 開発ツール

| ツール | 用途 |
|--------|------|
| Air | Go Hot Reload |
| golang-migrate | DBマイグレーション |
| oapi-codegen | OpenAPI → Goコード生成 |
| pnpm | パッケージマネージャ |
| ESLint | JavaScript/TypeScript Linter |
| Prettier | コードフォーマッター |
| golangci-lint | Go Linter |

---

## 7. バージョン管理方針

### セマンティックバージョニング

```
MAJOR.MINOR.PATCH

MAJOR: 破壊的変更
MINOR: 後方互換の機能追加
PATCH: 後方互換のバグ修正
```

### 依存関係の更新

| 種類 | 更新頻度 | 方針 |
|------|---------|------|
| セキュリティパッチ | 即時 | 最優先で適用 |
| マイナーアップデート | 月次 | テスト後に適用 |
| メジャーアップデート | 四半期 | 影響評価後に計画的に適用 |

---

## 8. 技術選定の理由

詳細な選定理由は以下のADRを参照：

- [ADR-0001: Go言語の採用](./adr/0001-go-language.md)
- [ADR-0002: React + TanStack構成](./adr/0002-react-tanstack.md)
- [ADR-0003: PBAC + ReBAC認可モデル](./adr/0003-pbac-rebac.md)

---

## 関連ドキュメント

- [CODING_STANDARDS.md](./CODING_STANDARDS.md) - コーディング規約
- [SETUP.md](./SETUP.md) - 開発環境セットアップ
