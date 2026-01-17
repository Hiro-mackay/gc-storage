# GC Storage マイグレーションガイド

## 概要

本ドキュメントでは、GC Storageのバージョンアップ時に必要な移行手順、Breaking Changesへの対応方法について説明します。

---

## 1. バージョンアップの基本手順

### 1.1 標準的なアップグレードフロー

```
┌─────────────────┐
│  1. リリースノート │
│     の確認        │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│  2. ステージング  │
│     でテスト      │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│  3. バックアップ  │
│     の取得        │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│  4. マイグレーション │
│     の実行        │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│  5. アプリケーション │
│     のデプロイ     │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│  6. 動作確認     │
│                 │
└─────────────────┘
```

### 1.2 チェックリスト

```
□ リリースノートで Breaking Changes を確認
□ 依存関係の更新が必要か確認
□ データベースマイグレーションの有無を確認
□ API の変更点を確認
□ ステージング環境でテスト
□ バックアップを取得
□ メンテナンスウィンドウを設定（必要な場合）
□ マイグレーションを実行
□ アプリケーションをデプロイ
□ 動作確認
□ モニタリングで異常がないか確認
```

---

## 2. バージョン別移行ガイド

### v1.0.x → v1.1.0

#### 変更点

| カテゴリ | 変更内容 |
|---------|---------|
| 機能追加 | ゴミ箱機能、お気に入り機能 |
| データベース | 新規テーブル追加 |
| API | 新規エンドポイント追加（後方互換） |

#### マイグレーション手順

```bash
# 1. バックアップ
pg_dump -U gc_storage gc_storage > backup-before-1.1.0.sql

# 2. マイグレーション実行
migrate -path migrations -database "$DATABASE_URL" up

# 3. アプリケーション更新
kubectl set image deployment/api api=gc-storage/api:v1.1.0
kubectl set image deployment/frontend frontend=gc-storage/frontend:v1.1.0

# 4. 動作確認
curl https://api.example.com/health
```

#### 注意事項

- ゴミ箱機能により、削除操作が論理削除に変更されます
- 物理削除は30日後に自動実行、または管理者による手動実行

---

### v1.1.x → v1.2.0

#### 変更点

| カテゴリ | 変更内容 |
|---------|---------|
| 機能追加 | ファイルコメント、通知機能 |
| データベース | 新規テーブル追加、既存テーブル変更 |
| API | 新規エンドポイント追加 |
| 依存関係 | 通知サービス（オプション） |

#### マイグレーション手順

```bash
# 1. バックアップ
pg_dump -U gc_storage gc_storage > backup-before-1.2.0.sql

# 2. 環境変数の追加（通知機能用）
export NOTIFICATION_SERVICE_URL=https://notifications.example.com
export EMAIL_SMTP_HOST=smtp.example.com

# 3. マイグレーション実行
migrate -path migrations -database "$DATABASE_URL" up

# 4. アプリケーション更新
kubectl apply -k environments/production/
```

---

### v1.x → v2.0.0 (Breaking Changes)

#### 重要な変更

| 変更 | 影響 | 対応 |
|------|------|------|
| API v2 | エンドポイントURLの変更 | クライアントの更新が必要 |
| 認証フロー | トークン形式の変更 | 全ユーザーが再ログイン必要 |
| レスポンス形式 | 一部フィールド名の変更 | クライアントコードの修正 |

#### 段階的移行手順

**Phase 1: 準備（2週間前）**

```bash
# 1. 現在のバージョンでデータのエクスポート
# ユーザーデータ、ファイルメタデータのバックアップ

# 2. ステージング環境でv2.0.0をテスト
kubectl apply -k environments/staging/

# 3. API v1 → v2 の移行ガイドをチーム/ユーザーに通知
```

**Phase 2: 並行運用（1週間）**

```bash
# v1とv2を並行で運用
# /api/v1/* と /api/v2/* の両方を提供

# Ingress設定（両方のパスをルーティング）
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: api-ingress
spec:
  rules:
    - host: api.example.com
      http:
        paths:
          - path: /api/v1
            backend:
              service:
                name: api-v1
                port: 80
          - path: /api/v2
            backend:
              service:
                name: api-v2
                port: 80
```

**Phase 3: v1 の廃止**

```bash
# v1 エンドポイントに Deprecation ヘッダーを追加
# 1ヶ月後に v1 を完全に停止
```

#### API 変更の詳細

**レスポンス形式の変更:**

```json
// v1
{
  "file_id": "xxx",
  "file_name": "document.pdf"
}

// v2
{
  "id": "xxx",
  "name": "document.pdf"
}
```

**エンドポイントの変更:**

| v1 | v2 | 備考 |
|-----|-----|------|
| `GET /api/v1/files` | `GET /api/v2/files` | レスポンス形式変更 |
| `POST /api/v1/files/upload` | `POST /api/v2/uploads/initiate` | パス変更 |
| `DELETE /api/v1/files/:id` | `DELETE /api/v2/files/:id` | 同じ |

---

## 3. データベースマイグレーション

### 3.1 マイグレーションの種類

| 種類 | 説明 | ダウンタイム |
|------|------|------------|
| 追加のみ | 新テーブル、新カラム（NULL許可）| なし |
| 変更あり | カラム型変更、NOT NULL追加 | 必要な場合あり |
| 削除あり | テーブル削除、カラム削除 | 段階的に実行 |

### 3.2 ゼロダウンタイムマイグレーション

破壊的な変更を伴うマイグレーションは段階的に実行:

```
Phase 1: 新カラムを追加（NULL許可）
         ↓
Phase 2: アプリで新旧両方に書き込み
         ↓
Phase 3: データ移行（バッチ処理）
         ↓
Phase 4: アプリで新カラムのみ使用
         ↓
Phase 5: 旧カラムを削除
```

**例: カラム名の変更**

```sql
-- Phase 1: 新カラム追加
ALTER TABLE files ADD COLUMN new_name VARCHAR(255);

-- Phase 3: データ移行
UPDATE files SET new_name = old_name WHERE new_name IS NULL;

-- Phase 5: 旧カラム削除（アプリ更新後）
ALTER TABLE files DROP COLUMN old_name;
ALTER TABLE files RENAME COLUMN new_name TO name;
```

### 3.3 マイグレーションのロールバック

```bash
# 直前のマイグレーションを元に戻す
migrate -path migrations -database "$DATABASE_URL" down 1

# 特定バージョンまで戻す
migrate -path migrations -database "$DATABASE_URL" goto 5

# 注意: データを削除するマイグレーションは元に戻せない場合がある
```

---

## 4. クライアントの更新

### 4.1 フロントエンドの更新

```bash
# 1. 依存関係の更新
pnpm update @gc-storage/client

# 2. 型定義の再生成（OpenAPI使用時）
pnpm run generate-types

# 3. コードの修正（Breaking Changes対応）
# 変更点に応じてコードを修正

# 4. テスト実行
pnpm test

# 5. ビルド・デプロイ
pnpm build
```

### 4.2 API クライアントの更新

**TypeScript クライアント:**

```typescript
// v1
const file = await client.getFile({ file_id: id });
console.log(file.file_name);

// v2
const file = await client.getFile({ id });
console.log(file.name);
```

---

## 5. 設定の移行

### 5.1 環境変数の変更

バージョンアップに伴う環境変数の変更:

| バージョン | 追加 | 変更 | 削除 |
|-----------|------|------|------|
| v1.1.0 | `TRASH_RETENTION_DAYS` | - | - |
| v1.2.0 | `NOTIFICATION_*` | - | - |
| v2.0.0 | `JWT_ALGORITHM` | `JWT_SECRET` → `JWT_ACCESS_SECRET` | `LEGACY_*` |

### 5.2 Kubernetes 設定の更新

```bash
# ConfigMap の更新
kubectl apply -f k8s/configmaps/

# Secret の更新（必要な場合）
kubectl apply -f k8s/secrets/

# Deployment の更新
kubectl apply -f k8s/deployments/
```

---

## 6. トラブルシューティング

### 6.1 マイグレーション失敗時

```bash
# 1. エラー内容を確認
migrate -path migrations -database "$DATABASE_URL" version

# 2. dirty 状態をリセット
migrate -path migrations -database "$DATABASE_URL" force <VERSION>

# 3. 問題を修正して再実行
migrate -path migrations -database "$DATABASE_URL" up
```

### 6.2 ロールバックが必要な場合

```bash
# 1. アプリケーションを旧バージョンに戻す
kubectl rollout undo deployment/api

# 2. マイグレーションをロールバック
migrate -path migrations -database "$DATABASE_URL" down 1

# 3. 動作確認
curl https://api.example.com/health
```

---

## 7. 非推奨機能

### 7.1 非推奨予定

| 機能/API | 非推奨バージョン | 削除予定 | 代替 |
|---------|----------------|---------|------|
| `/api/v1/*` | v2.0.0 | v3.0.0 | `/api/v2/*` |
| `file_id` フィールド | v2.0.0 | v3.0.0 | `id` |

### 7.2 非推奨機能の使用検出

```bash
# 非推奨APIの使用をログで確認
grep "DEPRECATED" /var/log/api.log

# メトリクスで確認（Prometheus）
http_requests_total{path=~"/api/v1.*"}
```

---

## 関連ドキュメント

- [CHANGELOG.md](./CHANGELOG.md) - リリース履歴
- [DEPLOYMENT.md](./DEPLOYMENT.md) - デプロイメント手順
- [API.md](./API.md) - API仕様

---

## 更新履歴

| 日付 | バージョン | 内容 |
|------|-----------|------|
| 2026-01-17 | 1.0.0 | 初版作成 |
