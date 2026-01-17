# GC Storage 運用ガイド

## 概要

本ドキュメントでは、GC Storageの本番環境における運用作業、監視、バックアップ、スケーリングについて説明します。

---

## 1. 監視ダッシュボード

### 1.1 Grafana ダッシュボード構成

| ダッシュボード | 内容 |
|--------------|------|
| Overview | システム全体のヘルス状態 |
| API Metrics | リクエスト数、レイテンシ、エラー率 |
| Database | PostgreSQL の接続数、クエリ性能 |
| Storage | MinIO の使用量、I/O |
| Kubernetes | Pod、Node のリソース使用状況 |

### 1.2 主要メトリクス

#### アプリケーションメトリクス

| メトリクス | 説明 | 閾値 |
|-----------|------|------|
| `http_requests_total` | HTTPリクエスト総数 | - |
| `http_request_duration_seconds` | リクエストレイテンシ | P95 < 500ms |
| `http_requests_in_flight` | 同時処理中のリクエスト数 | < 1000 |
| `http_errors_total` | HTTPエラー総数 | エラー率 < 1% |

#### インフラメトリクス

| メトリクス | 説明 | 閾値 |
|-----------|------|------|
| `container_cpu_usage_seconds_total` | CPU使用率 | < 80% |
| `container_memory_usage_bytes` | メモリ使用量 | < 80% |
| `pg_stat_activity_count` | DB接続数 | < max_connections * 0.8 |
| `redis_connected_clients` | Redis接続数 | < 10000 |

### 1.3 アラートルール

```yaml
# prometheus/alerts/gc-storage.yaml
groups:
  - name: gc-storage-alerts
    rules:
      # API 高エラー率
      - alert: HighErrorRate
        expr: |
          sum(rate(http_requests_total{status=~"5.."}[5m]))
          / sum(rate(http_requests_total[5m])) > 0.01
        for: 5m
        labels:
          severity: critical
        annotations:
          summary: "High error rate detected"
          description: "Error rate is above 1% for 5 minutes"

      # API 高レイテンシ
      - alert: HighLatency
        expr: |
          histogram_quantile(0.95,
            sum(rate(http_request_duration_seconds_bucket[5m])) by (le)
          ) > 1
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "High latency detected"
          description: "P95 latency is above 1 second"

      # Pod 再起動
      - alert: PodRestarting
        expr: |
          increase(kube_pod_container_status_restarts_total{namespace="gc-storage"}[1h]) > 3
        labels:
          severity: warning
        annotations:
          summary: "Pod is restarting frequently"

      # ディスク使用率
      - alert: HighDiskUsage
        expr: |
          (kubelet_volume_stats_used_bytes / kubelet_volume_stats_capacity_bytes) > 0.8
        for: 10m
        labels:
          severity: warning
        annotations:
          summary: "High disk usage"
          description: "Disk usage is above 80%"

      # PostgreSQL 接続数
      - alert: HighDatabaseConnections
        expr: |
          pg_stat_activity_count > 150
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "High database connection count"
```

---

## 2. バックアップ

### 2.1 バックアップ戦略

| データ種別 | 方式 | 頻度 | 保持期間 |
|-----------|------|------|---------|
| PostgreSQL | WAL + pg_dump | 継続的 + 日次 | 30日 |
| MinIO | オブジェクト複製 | リアルタイム | 永続 |
| Redis | RDB + AOF | 5分ごと | 7日 |
| Kubernetes | etcd snapshot | 日次 | 14日 |

### 2.2 PostgreSQL バックアップ

#### 自動バックアップ（CronJob）

```yaml
# k8s/backup/postgres-backup-cronjob.yaml
apiVersion: batch/v1
kind: CronJob
metadata:
  name: postgres-backup
  namespace: gc-storage
spec:
  schedule: "0 2 * * *"  # 毎日 02:00
  concurrencyPolicy: Forbid
  jobTemplate:
    spec:
      template:
        spec:
          containers:
            - name: backup
              image: postgres:16-alpine
              command:
                - /bin/sh
                - -c
                - |
                  BACKUP_FILE="gc-storage-$(date +%Y%m%d-%H%M%S).sql.gz"
                  pg_dump -h $DB_HOST -U $DB_USER -d $DB_NAME | gzip > /backup/$BACKUP_FILE
                  # S3 にアップロード
                  aws s3 cp /backup/$BACKUP_FILE s3://gc-storage-backup/postgres/
                  # 30日以上古いバックアップを削除
                  aws s3 ls s3://gc-storage-backup/postgres/ | \
                    awk '{print $4}' | \
                    while read file; do
                      date=$(echo $file | grep -oP '\d{8}')
                      if [ $(date -d $date +%s) -lt $(date -d '30 days ago' +%s) ]; then
                        aws s3 rm s3://gc-storage-backup/postgres/$file
                      fi
                    done
              env:
                - name: DB_HOST
                  value: postgres-primary
                - name: DB_USER
                  valueFrom:
                    secretKeyRef:
                      name: db-secret
                      key: username
                - name: PGPASSWORD
                  valueFrom:
                    secretKeyRef:
                      name: db-secret
                      key: password
                - name: DB_NAME
                  value: gc_storage
              volumeMounts:
                - name: backup-volume
                  mountPath: /backup
          volumes:
            - name: backup-volume
              emptyDir: {}
          restartPolicy: OnFailure
```

#### 手動バックアップ

```bash
# フルバックアップ
kubectl exec -n gc-storage deployment/postgres -- \
  pg_dump -U gc_storage gc_storage | gzip > backup-$(date +%Y%m%d).sql.gz

# 特定テーブルのみ
kubectl exec -n gc-storage deployment/postgres -- \
  pg_dump -U gc_storage -t files -t folders gc_storage > tables-backup.sql
```

### 2.3 復元手順

```bash
# 1. 現在のサービスを停止（必要に応じて）
kubectl scale deployment/api --replicas=0 -n gc-storage

# 2. バックアップファイルのダウンロード
aws s3 cp s3://gc-storage-backup/postgres/gc-storage-20240115.sql.gz ./

# 3. データベースの復元
gunzip -c gc-storage-20240115.sql.gz | \
  kubectl exec -i -n gc-storage deployment/postgres -- \
  psql -U gc_storage gc_storage

# 4. サービスの再開
kubectl scale deployment/api --replicas=3 -n gc-storage
```

### 2.4 ポイントインタイムリカバリ（PITR）

WAL アーカイブを使用した特定時点への復元:

```bash
# 1. リカバリ対象時刻を指定
RECOVERY_TARGET_TIME="2024-01-15 10:30:00"

# 2. PostgreSQL の recovery.conf を設定
restore_command = 'aws s3 cp s3://gc-storage-backup/wal/%f %p'
recovery_target_time = '$RECOVERY_TARGET_TIME'
recovery_target_action = 'promote'

# 3. PostgreSQL を起動してリカバリ実行
```

---

## 3. スケーリング

### 3.1 自動スケーリング（HPA）

```yaml
# k8s/api/hpa.yaml
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: api-hpa
  namespace: gc-storage
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: api
  minReplicas: 3
  maxReplicas: 10
  metrics:
    - type: Resource
      resource:
        name: cpu
        target:
          type: Utilization
          averageUtilization: 70
    - type: Resource
      resource:
        name: memory
        target:
          type: Utilization
          averageUtilization: 80
```

### 3.2 スケーリング判断基準

| 指標 | スケールアウト | スケールイン |
|------|--------------|-------------|
| CPU使用率 | > 70% が5分継続 | < 30% が15分継続 |
| メモリ使用率 | > 80% が5分継続 | < 40% が15分継続 |
| リクエストキュー | > 100 が3分継続 | - |

### 3.3 手動スケーリング

```bash
# レプリカ数の変更
kubectl scale deployment/api --replicas=5 -n gc-storage

# HPA の一時無効化
kubectl patch hpa api-hpa -n gc-storage -p '{"spec":{"minReplicas":5,"maxReplicas":5}}'

# HPA の復元
kubectl apply -f k8s/api/hpa.yaml
```

### 3.4 データベーススケーリング

#### 読み取りレプリカの追加

```bash
# レプリカ StatefulSet のスケール
kubectl scale statefulset/postgres-replica --replicas=2 -n gc-storage
```

#### 接続プール（PgBouncer）設定

```ini
# pgbouncer.ini
[databases]
gc_storage = host=postgres-primary port=5432 dbname=gc_storage

[pgbouncer]
listen_port = 6432
listen_addr = 0.0.0.0
auth_type = md5
pool_mode = transaction
max_client_conn = 1000
default_pool_size = 25
```

---

## 4. 日常運用タスク

### 4.1 ヘルスチェック

```bash
# API ヘルスチェック
curl https://api.gc-storage.example.com/health

# 詳細ヘルスチェック
curl https://api.gc-storage.example.com/health/ready

# Pod の状態確認
kubectl get pods -n gc-storage

# リソース使用状況
kubectl top pods -n gc-storage
kubectl top nodes
```

### 4.2 ログ確認

```bash
# API ログ
kubectl logs -f deployment/api -n gc-storage

# 特定 Pod のログ
kubectl logs -f <pod-name> -n gc-storage

# 過去のログ（前のコンテナ）
kubectl logs <pod-name> -n gc-storage --previous

# ログ検索（Loki/Grafana経由）
# Label: {namespace="gc-storage", container="api"}
# Query: |= "error"
```

### 4.3 定期メンテナンス

#### 週次タスク

```bash
# データベースの VACUUM ANALYZE
kubectl exec -n gc-storage deployment/postgres -- \
  psql -U gc_storage -c "VACUUM ANALYZE;"

# 古い監査ログの削除（90日以上）
kubectl exec -n gc-storage deployment/postgres -- \
  psql -U gc_storage -c "DELETE FROM audit_logs WHERE created_at < NOW() - INTERVAL '90 days';"

# Redis キャッシュの確認
kubectl exec -n gc-storage deployment/redis -- redis-cli info memory
```

#### 月次タスク

```bash
# インデックスの再作成（必要に応じて）
kubectl exec -n gc-storage deployment/postgres -- \
  psql -U gc_storage -c "REINDEX DATABASE gc_storage;"

# テーブル統計の更新
kubectl exec -n gc-storage deployment/postgres -- \
  psql -U gc_storage -c "ANALYZE;"

# バックアップの復元テスト
# （ステージング環境で実施）
```

---

## 5. 証明書管理

### 5.1 TLS証明書の更新

cert-manager による自動更新:

```yaml
# k8s/cert-manager/certificate.yaml
apiVersion: cert-manager.io/v1
kind: Certificate
metadata:
  name: gc-storage-tls
  namespace: gc-storage
spec:
  secretName: gc-storage-tls
  issuerRef:
    name: letsencrypt-prod
    kind: ClusterIssuer
  dnsNames:
    - gc-storage.example.com
    - api.gc-storage.example.com
```

### 5.2 証明書の確認

```bash
# 証明書の有効期限確認
kubectl get certificate -n gc-storage
kubectl describe certificate gc-storage-tls -n gc-storage

# Secret の確認
kubectl get secret gc-storage-tls -n gc-storage -o jsonpath='{.data.tls\.crt}' | \
  base64 -d | openssl x509 -noout -dates
```

---

## 6. シークレットローテーション

### 6.1 データベースパスワードの変更

```bash
# 1. 新しいシークレットを作成
kubectl create secret generic db-secret-new \
  --from-literal=username=gc_storage \
  --from-literal=password=new-secure-password \
  -n gc-storage

# 2. PostgreSQL でパスワードを変更
kubectl exec -n gc-storage deployment/postgres -- \
  psql -U postgres -c "ALTER USER gc_storage WITH PASSWORD 'new-secure-password';"

# 3. Deployment を更新して新しいシークレットを参照
kubectl patch deployment api -n gc-storage \
  --patch '{"spec":{"template":{"spec":{"containers":[{"name":"api","env":[{"name":"DB_PASSWORD","valueFrom":{"secretKeyRef":{"name":"db-secret-new","key":"password"}}}]}]}}}}'

# 4. 古いシークレットを削除
kubectl delete secret db-secret -n gc-storage
kubectl patch secret db-secret-new -n gc-storage -p '{"metadata":{"name":"db-secret"}}'
```

### 6.2 JWT シークレットの変更

JWT シークレットの変更は慎重に行う必要があります（既存セッションが無効化されるため）:

1. メンテナンスウィンドウを設定
2. 新しいシークレットでデプロイ
3. ユーザーに再ログインを促す

---

## 7. 容量管理

### 7.1 ストレージ使用量の監視

```bash
# PVC の使用状況
kubectl get pvc -n gc-storage
kubectl exec -n gc-storage deployment/postgres -- df -h /var/lib/postgresql/data

# MinIO の使用状況
mc admin info local
```

### 7.2 容量拡張

```bash
# PVC の拡張（StorageClass が allowVolumeExpansion: true の場合）
kubectl patch pvc postgres-data -n gc-storage -p '{"spec":{"resources":{"requests":{"storage":"200Gi"}}}}'
```

### 7.3 データのクリーンアップ

```bash
# 削除済みファイルの物理削除（ゴミ箱の完全削除）
kubectl exec -n gc-storage deployment/api -- \
  /app/cli cleanup-trash --older-than 30d

# 孤立したストレージオブジェクトの削除
kubectl exec -n gc-storage deployment/api -- \
  /app/cli cleanup-orphans --dry-run
```

---

## 8. 運用チェックリスト

### 8.1 日次チェック

```
□ Grafana ダッシュボードでシステム状態確認
□ アラート履歴の確認
□ エラーログの確認
□ バックアップ成功の確認
```

### 8.2 週次チェック

```
□ リソース使用率のトレンド確認
□ セキュリティアラートの確認
□ 依存パッケージの脆弱性確認
□ バックアップからの復元テスト（ステージング）
```

### 8.3 月次チェック

```
□ 容量計画の見直し
□ パフォーマンス分析
□ コスト分析
□ インシデントの振り返り
□ ドキュメントの更新
```

---

## 関連ドキュメント

- [DEPLOYMENT.md](./DEPLOYMENT.md) - デプロイメント手順
- [INCIDENT_RESPONSE.md](./INCIDENT_RESPONSE.md) - 障害対応
- [INFRASTRUCTURE.md](./INFRASTRUCTURE.md) - インフラストラクチャ設計
- [TROUBLESHOOTING.md](./TROUBLESHOOTING.md) - トラブルシューティング

---

## 更新履歴

| 日付 | バージョン | 内容 |
|------|-----------|------|
| 2026-01-17 | 1.0.0 | 初版作成 |
