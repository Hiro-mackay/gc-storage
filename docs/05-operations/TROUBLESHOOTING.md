# GC Storage トラブルシューティングガイド

## 概要

本ドキュメントでは、GC Storageで発生しやすい問題とその解決方法を説明します。

---

## 1. 開発環境の問題

### 1.1 Docker Compose が起動しない

**症状**: `docker compose up` がエラーで失敗する

**原因と解決策**:

```bash
# ポートの競合を確認
lsof -i :5432  # PostgreSQL
lsof -i :6379  # Redis
lsof -i :9000  # MinIO

# 競合するプロセスを停止
kill -9 <PID>

# Docker リソースの問題
docker system prune -f  # 不要なリソースを削除
docker volume prune -f  # 不要なボリュームを削除

# Docker Desktop を再起動
```

### 1.2 データベースに接続できない

**症状**: "connection refused" または "role does not exist"

**解決策**:

```bash
# PostgreSQL コンテナの状態確認
docker compose ps postgres
docker compose logs postgres

# データベースの初期化を確認
docker compose exec postgres psql -U gc_storage -d gc_storage -c '\dt'

# コンテナを再作成
docker compose down -v
docker compose up -d postgres
```

### 1.3 マイグレーションエラー

**症状**: "dirty database version" または "migration failed"

**解決策**:

```bash
# 現在のバージョン確認
migrate -path migrations -database "${DATABASE_URL}" version

# dirty 状態をリセット
migrate -path migrations -database "${DATABASE_URL}" force <VERSION>

# マイグレーションを再実行
migrate -path migrations -database "${DATABASE_URL}" up
```

### 1.4 Go モジュールの問題

**症状**: "module not found" または "checksum mismatch"

**解決策**:

```bash
# モジュールキャッシュのクリア
go clean -modcache

# go.sum の再生成
rm go.sum
go mod tidy

# プライベートモジュールの設定（必要な場合）
go env -w GOPRIVATE=github.com/your-org/*
```

### 1.5 フロントエンドのビルドエラー

**症状**: "Module not found" または "Type error"

**解決策**:

```bash
# node_modules の再インストール
rm -rf node_modules
rm pnpm-lock.yaml
pnpm install

# TypeScript のキャッシュクリア
rm -rf node_modules/.cache
rm -rf .next  # Next.js の場合

# 型定義の再生成
pnpm run generate-types
```

---

## 2. 認証の問題

### 2.1 ログインできない

**症状**: 正しい認証情報でもログインが失敗する

**チェック項目**:

```bash
# 1. ユーザーの存在確認
docker compose exec postgres psql -U gc_storage -c \
  "SELECT id, email FROM users WHERE email = 'user@example.com';"

# 2. パスワードハッシュの確認
# パスワードを再設定（開発環境のみ）
docker compose exec postgres psql -U gc_storage -c \
  "UPDATE users SET password_hash = '<new_hash>' WHERE email = 'user@example.com';"

# 3. Redis の接続確認（セッション関連）
docker compose exec redis redis-cli ping
```

### 2.2 トークンの期限切れ

**症状**: "Token expired" エラー

**解決策**:

```bash
# フロントエンドでリフレッシュトークンを使用してトークンを更新
# 自動更新が動作しない場合は再ログインが必要

# 開発環境でのトークン有効期限の延長（.envを編集）
JWT_ACCESS_EXPIRATION=24h
JWT_REFRESH_EXPIRATION=30d
```

### 2.3 OAuth ログインエラー

**症状**: OAuth コールバックでエラー

**チェック項目**:

```bash
# 1. 環境変数の確認
echo $GOOGLE_CLIENT_ID
echo $GOOGLE_CLIENT_SECRET

# 2. コールバックURLの設定確認
# Google Cloud Console / GitHub Developer Settings で
# リダイレクトURLが正しく設定されているか確認

# 3. ログでエラー詳細を確認
docker compose logs api | grep -i oauth
```

---

## 3. ファイル操作の問題

### 3.1 アップロードが失敗する

**症状**: ファイルアップロードが途中で失敗

**チェック項目**:

```bash
# 1. MinIO の状態確認
curl http://localhost:9000/minio/health/live
curl http://localhost:9000/minio/health/ready

# 2. バケットの存在確認
docker compose exec minio mc ls local/gc-storage-dev

# 3. ディスク容量の確認
df -h
docker system df

# 4. ファイルサイズ制限の確認（nginx/ingress）
# proxy-body-size の設定を確認
```

**解決策**:

```bash
# MinIO バケットの作成
docker compose exec minio mc mb local/gc-storage-dev

# Presigned URL の有効期限延長（大きなファイル用）
# 環境変数で設定: PRESIGNED_URL_EXPIRY=1h
```

### 3.2 ダウンロードが遅い

**症状**: ファイルダウンロードが非常に遅い

**チェック項目**:

```bash
# 1. MinIO のネットワーク設定確認
docker compose exec minio mc admin info local

# 2. バンドウィズの確認
# 開発環境：Docker Desktop のリソース設定を確認

# 3. Presigned URL の直接テスト
curl -I "http://localhost:9000/gc-storage-dev/test-file"
```

### 3.3 ファイルが見つからない

**症状**: アップロードしたはずのファイルが表示されない

**チェック項目**:

```bash
# 1. データベースでファイルレコードを確認
docker compose exec postgres psql -U gc_storage -c \
  "SELECT id, name, status, storage_key FROM files WHERE name LIKE '%filename%';"

# 2. MinIO でオブジェクトを確認
docker compose exec minio mc ls local/gc-storage-dev/

# 3. ファイルのステータスを確認
# status が 'pending' のままになっていないか
# アップロード完了通知が送信されているか
```

---

## 4. パフォーマンスの問題

### 4.1 API レスポンスが遅い

**症状**: API の応答が数秒かかる

**診断**:

```bash
# 1. スロークエリの確認
docker compose exec postgres psql -U gc_storage -c \
  "SELECT query, calls, mean_time FROM pg_stat_statements ORDER BY mean_time DESC LIMIT 10;"

# 2. コネクションプールの状態
docker compose exec postgres psql -U gc_storage -c \
  "SELECT count(*) FROM pg_stat_activity;"

# 3. Redis の状態
docker compose exec redis redis-cli info stats
```

**解決策**:

```bash
# インデックスの確認・作成
docker compose exec postgres psql -U gc_storage -c \
  "EXPLAIN ANALYZE SELECT * FROM files WHERE folder_id = 'xxx';"

# 統計情報の更新
docker compose exec postgres psql -U gc_storage -c "ANALYZE;"

# キャッシュのクリア
docker compose exec redis redis-cli FLUSHDB
```

### 4.2 メモリ使用量が高い

**症状**: APIサーバーのメモリ使用量が増加し続ける

**診断**:

```bash
# Go のメモリプロファイリング
curl http://localhost:8080/debug/pprof/heap > heap.prof
go tool pprof heap.prof

# コンテナのメモリ使用状況
docker stats
```

**解決策**:

```bash
# GOGC の調整（環境変数）
GOGC=50  # デフォルトは100、小さくするとGCが頻繁に

# メモリリークの疑いがある場合はプロファイリングで調査
```

---

## 5. 権限の問題

### 5.1 アクセス拒否される

**症状**: ファイルにアクセスしようとすると 403 Forbidden

**チェック項目**:

```bash
# 1. ユーザーの権限を確認
docker compose exec postgres psql -U gc_storage -c \
  "SELECT * FROM permission_grants WHERE resource_id = 'xxx';"

# 2. グループメンバーシップを確認
docker compose exec postgres psql -U gc_storage -c \
  "SELECT * FROM relationships WHERE subject_id = 'user_id' AND relation = 'member';"

# 3. フォルダ階層の権限継承を確認
docker compose exec postgres psql -U gc_storage -c \
  "SELECT * FROM relationships WHERE object_id = 'folder_id';"
```

### 5.2 権限が反映されない

**症状**: 権限を設定しても反映されない

**解決策**:

```bash
# 権限キャッシュのクリア
docker compose exec redis redis-cli keys "permission:*" | xargs redis-cli del

# APIサーバーの再起動
docker compose restart api
```

---

## 6. ログの解釈

### 6.1 エラーログの読み方

```json
{
  "level": "error",
  "time": "2024-01-15T10:30:00Z",
  "caller": "handler/file_handler.go:123",
  "msg": "failed to upload file",
  "error": "context deadline exceeded",
  "request_id": "abc-123",
  "user_id": "user-456"
}
```

| フィールド | 説明 |
|-----------|------|
| level | ログレベル（error, warn, info, debug） |
| time | 発生時刻（UTC） |
| caller | エラー発生箇所 |
| msg | エラーメッセージ |
| error | 詳細なエラー内容 |
| request_id | リクエスト追跡ID |
| user_id | 関連ユーザー |

### 6.2 よくあるエラーメッセージ

| エラー | 原因 | 対処 |
|--------|------|------|
| `context deadline exceeded` | タイムアウト | タイムアウト値の調整、処理の最適化 |
| `connection refused` | 接続先サービスがダウン | サービスの起動確認 |
| `too many connections` | コネクション数超過 | プール設定の調整 |
| `permission denied` | 権限不足 | 権限設定の確認 |
| `no such file or directory` | ファイルが存在しない | パス・存在確認 |

---

## 7. 便利なデバッグコマンド

### 7.1 データベース

```bash
# テーブル一覧
\dt

# テーブル構造
\d+ files

# 実行中のクエリ
SELECT * FROM pg_stat_activity WHERE state = 'active';

# ロック状況
SELECT * FROM pg_locks WHERE NOT granted;
```

### 7.2 Redis

```bash
# キー一覧
KEYS *

# キーの内容確認
GET session:xxx
HGETALL cache:xxx

# メモリ使用状況
INFO memory
```

### 7.3 MinIO

```bash
# バケット一覧
mc ls local/

# オブジェクト一覧
mc ls local/gc-storage-dev/

# オブジェクト情報
mc stat local/gc-storage-dev/path/to/file
```

---

## 8. サポートへの問い合わせ

問題が解決しない場合は、以下の情報を添えて Issue を作成してください:

1. **環境情報**: OS、Docker バージョン、Go/Node.js バージョン
2. **再現手順**: 問題が発生する具体的な手順
3. **エラーメッセージ**: 完全なエラーログ
4. **期待する動作**: 本来どうなるべきか
5. **試した対処**: これまでに試した解決策

---

## 関連ドキュメント

- [SETUP.md](./SETUP.md) - 開発環境セットアップ
- [INCIDENT_RESPONSE.md](./INCIDENT_RESPONSE.md) - 障害対応
- [OPERATIONS.md](./OPERATIONS.md) - 運用ガイド

---

## 更新履歴

| 日付 | バージョン | 内容 |
|------|-----------|------|
| 2026-01-17 | 1.0.0 | 初版作成 |
