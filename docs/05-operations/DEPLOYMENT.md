# GC Storage デプロイメントガイド

## 概要

本ドキュメントでは、GC Storageの各環境へのデプロイ手順、マイグレーション実行、およびロールバック手順について説明します。

---

## 1. 環境構成

### 1.1 環境一覧

| 環境 | 用途 | デプロイトリガー |
|------|------|----------------|
| Development | 開発・テスト | 手動 |
| Staging | リリース前検証 | `develop` ブランチへのマージ |
| Production | 本番環境 | タグ作成（`v*.*.*`） |

### 1.2 環境別設定

```
environments/
├── development/
│   ├── kustomization.yaml
│   └── patches/
├── staging/
│   ├── kustomization.yaml
│   └── patches/
└── production/
    ├── kustomization.yaml
    └── patches/
```

### 1.3 環境別リソース設定

| リソース | Development | Staging | Production |
|---------|-------------|---------|------------|
| API Replicas | 1 | 2 | 3+ |
| Frontend Replicas | 1 | 2 | 2+ |
| PostgreSQL | Single | Single | Primary + Replica |
| Redis | Single | Single | Cluster |
| MinIO | Single | Single | Distributed |

---

## 2. CI/CD パイプライン

### 2.1 パイプライン概要

```
┌─────────────┐    ┌─────────────┐    ┌─────────────┐    ┌─────────────┐
│    Build    │───▶│    Test     │───▶│    Push     │───▶│   Deploy    │
│             │    │             │    │   Image     │    │             │
└─────────────┘    └─────────────┘    └─────────────┘    └─────────────┘
      │                  │                  │                  │
      ▼                  ▼                  ▼                  ▼
 Go/Node Build      Unit/E2E Test    Container Registry   Kubernetes
```

### 2.2 GitHub Actions ワークフロー

```yaml
# .github/workflows/deploy.yml
name: Deploy

on:
  push:
    branches:
      - main
      - develop
    tags:
      - 'v*.*.*'

env:
  REGISTRY: ghcr.io
  IMAGE_NAME: ${{ github.repository }}

jobs:
  build-and-push:
    runs-on: ubuntu-latest
    permissions:
      contents: read
      packages: write

    steps:
      - name: Checkout
        uses: actions/checkout@v4

      - name: Set up Docker Buildx
        uses: docker/setup-buildx-action@v3

      - name: Login to Container Registry
        uses: docker/login-action@v3
        with:
          registry: ${{ env.REGISTRY }}
          username: ${{ github.actor }}
          password: ${{ secrets.GITHUB_TOKEN }}

      - name: Extract metadata
        id: meta
        uses: docker/metadata-action@v5
        with:
          images: ${{ env.REGISTRY }}/${{ env.IMAGE_NAME }}
          tags: |
            type=ref,event=branch
            type=semver,pattern={{version}}
            type=semver,pattern={{major}}.{{minor}}
            type=sha,prefix=

      # Backend Image
      - name: Build and push backend
        uses: docker/build-push-action@v5
        with:
          context: ./backend
          push: true
          tags: ${{ env.REGISTRY }}/${{ env.IMAGE_NAME }}/api:${{ steps.meta.outputs.version }}
          cache-from: type=gha
          cache-to: type=gha,mode=max

      # Frontend Image
      - name: Build and push frontend
        uses: docker/build-push-action@v5
        with:
          context: ./frontend
          push: true
          tags: ${{ env.REGISTRY }}/${{ env.IMAGE_NAME }}/frontend:${{ steps.meta.outputs.version }}
          cache-from: type=gha
          cache-to: type=gha,mode=max

  deploy-staging:
    needs: build-and-push
    if: github.ref == 'refs/heads/develop'
    runs-on: ubuntu-latest
    environment: staging

    steps:
      - name: Checkout
        uses: actions/checkout@v4

      - name: Set up kubectl
        uses: azure/setup-kubectl@v3

      - name: Configure kubeconfig
        run: |
          mkdir -p ~/.kube
          echo "${{ secrets.KUBECONFIG_STAGING }}" | base64 -d > ~/.kube/config

      - name: Deploy to staging
        run: |
          kubectl apply -k environments/staging/
          kubectl rollout status deployment/api -n gc-storage-staging
          kubectl rollout status deployment/frontend -n gc-storage-staging

  deploy-production:
    needs: build-and-push
    if: startsWith(github.ref, 'refs/tags/v')
    runs-on: ubuntu-latest
    environment: production

    steps:
      - name: Checkout
        uses: actions/checkout@v4

      - name: Set up kubectl
        uses: azure/setup-kubectl@v3

      - name: Configure kubeconfig
        run: |
          mkdir -p ~/.kube
          echo "${{ secrets.KUBECONFIG_PRODUCTION }}" | base64 -d > ~/.kube/config

      - name: Run database migrations
        run: |
          kubectl exec -n gc-storage deployment/api -- \
            /app/migrate -path /app/migrations -database "$DATABASE_URL" up

      - name: Deploy to production
        run: |
          kubectl apply -k environments/production/
          kubectl rollout status deployment/api -n gc-storage --timeout=5m
          kubectl rollout status deployment/frontend -n gc-storage --timeout=5m

      - name: Verify deployment
        run: |
          kubectl get pods -n gc-storage
          curl -f https://api.gc-storage.example.com/health || exit 1
```

---

## 3. 手動デプロイ手順

### 3.1 前提条件

```bash
# 必要なツール
kubectl version --client
kustomize version
helm version  # オプション

# クラスターへの接続確認
kubectl cluster-info
kubectl get nodes
```

### 3.2 イメージのビルドとプッシュ

```bash
# バックエンド
cd backend
docker build -t ghcr.io/hiro-mackay/gc-storage/api:v1.0.0 .
docker push ghcr.io/hiro-mackay/gc-storage/api:v1.0.0

# フロントエンド
cd frontend
docker build -t ghcr.io/hiro-mackay/gc-storage/frontend:v1.0.0 .
docker push ghcr.io/hiro-mackay/gc-storage/frontend:v1.0.0
```

### 3.3 Kustomize によるデプロイ

```bash
# Staging
kubectl apply -k environments/staging/

# Production
kubectl apply -k environments/production/

# デプロイ状況確認
kubectl rollout status deployment/api -n gc-storage
kubectl rollout status deployment/frontend -n gc-storage
```

### 3.4 イメージタグの更新

```bash
# kustomization.yaml でイメージタグを更新
cd environments/production/
kustomize edit set image ghcr.io/hiro-mackay/gc-storage/api:v1.0.1

# または直接編集
# environments/production/kustomization.yaml
images:
  - name: ghcr.io/hiro-mackay/gc-storage/api
    newTag: v1.0.1
  - name: ghcr.io/hiro-mackay/gc-storage/frontend
    newTag: v1.0.1
```

---

## 4. データベースマイグレーション

### 4.1 マイグレーション実行（デプロイ前）

```bash
# Job でマイグレーションを実行
kubectl apply -f - <<EOF
apiVersion: batch/v1
kind: Job
metadata:
  name: db-migrate-$(date +%Y%m%d%H%M%S)
  namespace: gc-storage
spec:
  template:
    spec:
      containers:
        - name: migrate
          image: ghcr.io/hiro-mackay/gc-storage/api:v1.0.0
          command:
            - /app/migrate
            - -path
            - /app/migrations
            - -database
            - $(DATABASE_URL)
            - up
          env:
            - name: DATABASE_URL
              valueFrom:
                secretKeyRef:
                  name: db-secret
                  key: url
      restartPolicy: Never
  backoffLimit: 3
EOF

# ジョブの完了を待機
kubectl wait --for=condition=complete job/db-migrate-* -n gc-storage --timeout=300s
```

### 4.2 マイグレーション状況確認

```bash
# 現在のバージョン確認
kubectl exec -n gc-storage deployment/api -- \
  /app/migrate -path /app/migrations -database "$DATABASE_URL" version

# マイグレーション履歴
kubectl exec -n gc-storage -it deployment/api -- \
  psql "$DATABASE_URL" -c "SELECT * FROM schema_migrations ORDER BY version DESC LIMIT 10;"
```

### 4.3 マイグレーションのロールバック

```bash
# 1つ前のバージョンに戻す
kubectl exec -n gc-storage deployment/api -- \
  /app/migrate -path /app/migrations -database "$DATABASE_URL" down 1

# 特定バージョンまで戻す
kubectl exec -n gc-storage deployment/api -- \
  /app/migrate -path /app/migrations -database "$DATABASE_URL" goto <version>

# 強制バージョン設定（dirty状態の解消）
kubectl exec -n gc-storage deployment/api -- \
  /app/migrate -path /app/migrations -database "$DATABASE_URL" force <version>
```

### 4.4 破壊的マイグレーションの注意点

破壊的な変更（カラム削除、型変更等）を行う場合:

1. **事前準備**
   - バックアップの取得
   - 影響範囲の確認
   - メンテナンスウィンドウの設定

2. **段階的な移行**
   ```
   v1: 新カラム追加（NULL許可）
   v2: データ移行、アプリケーション対応
   v3: 旧カラム削除
   ```

3. **バックアップからの復元手順を事前にテスト**

---

## 5. ロールバック手順

### 5.1 Kubernetes Deploymentのロールバック

```bash
# ロールアウト履歴確認
kubectl rollout history deployment/api -n gc-storage

# 直前のバージョンに戻す
kubectl rollout undo deployment/api -n gc-storage

# 特定リビジョンに戻す
kubectl rollout undo deployment/api -n gc-storage --to-revision=2

# ロールバック状況確認
kubectl rollout status deployment/api -n gc-storage
```

### 5.2 完全ロールバックの手順

```bash
# 1. 新しいデプロイを停止
kubectl scale deployment/api --replicas=0 -n gc-storage

# 2. データベースのロールバック（必要な場合）
kubectl exec -n gc-storage deployment/api -- \
  /app/migrate -path /app/migrations -database "$DATABASE_URL" down 1

# 3. 前バージョンのイメージでデプロイ
kustomize edit set image ghcr.io/hiro-mackay/gc-storage/api:v0.9.9
kubectl apply -k environments/production/

# 4. ヘルスチェック
kubectl rollout status deployment/api -n gc-storage
curl -f https://api.gc-storage.example.com/health
```

### 5.3 緊急ロールバックチェックリスト

```
□ 1. インシデント発生を確認・記録
□ 2. 影響範囲を特定
□ 3. ロールバック判断（エスカレーション）
□ 4. メンテナンス通知（必要に応じて）
□ 5. トラフィック遮断（必要に応じて）
□ 6. DBロールバック実行
□ 7. アプリケーションロールバック実行
□ 8. 動作確認
□ 9. トラフィック復旧
□ 10. ポストモーテム作成
```

---

## 6. Blue-Green デプロイ

### 6.1 概要

```
         ┌─────────────────────┐
         │    Load Balancer    │
         │      (Ingress)      │
         └──────────┬──────────┘
                    │
        ┌───────────┴───────────┐
        │                       │
        ▼                       ▼
┌───────────────┐       ┌───────────────┐
│   Blue (v1)   │       │  Green (v2)   │
│   (Current)   │       │    (New)      │
└───────────────┘       └───────────────┘
```

### 6.2 Blue-Green 切り替え手順

```bash
# 1. Green環境にデプロイ
kubectl apply -f k8s/green-deployment.yaml

# 2. Green環境のヘルスチェック
kubectl rollout status deployment/api-green -n gc-storage
curl -f http://api-green.gc-storage.svc.cluster.local/health

# 3. Ingressの切り替え
kubectl patch ingress gc-storage -n gc-storage --type='json' \
  -p='[{"op": "replace", "path": "/spec/rules/0/http/paths/0/backend/service/name", "value": "api-green"}]'

# 4. 動作確認
curl -f https://api.gc-storage.example.com/health

# 5. Blue環境の削除（確認後）
kubectl delete deployment api-blue -n gc-storage
```

---

## 7. Canary デプロイ

### 7.1 Canary の概念

```
                    100% Traffic
                         │
         ┌───────────────┴───────────────┐
         │                               │
         ▼ 90%                           ▼ 10%
┌─────────────────┐             ┌─────────────────┐
│   Stable (v1)   │             │   Canary (v2)   │
│   3 replicas    │             │   1 replica     │
└─────────────────┘             └─────────────────┘
```

### 7.2 Canary デプロイ手順（Ingress-NGINX）

```yaml
# k8s/canary/canary-ingress.yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: gc-storage-canary
  namespace: gc-storage
  annotations:
    nginx.ingress.kubernetes.io/canary: "true"
    nginx.ingress.kubernetes.io/canary-weight: "10"
spec:
  rules:
    - host: api.gc-storage.example.com
      http:
        paths:
          - path: /
            pathType: Prefix
            backend:
              service:
                name: api-canary
                port:
                  number: 80
```

```bash
# Canaryデプロイを作成
kubectl apply -f k8s/canary/

# トラフィック割合を段階的に増加
kubectl annotate ingress gc-storage-canary \
  nginx.ingress.kubernetes.io/canary-weight="25" --overwrite

# 50%
kubectl annotate ingress gc-storage-canary \
  nginx.ingress.kubernetes.io/canary-weight="50" --overwrite

# 100%（本番昇格）
kubectl annotate ingress gc-storage-canary \
  nginx.ingress.kubernetes.io/canary-weight="100" --overwrite

# 安定版を更新してCanaryを削除
kubectl apply -k environments/production/
kubectl delete -f k8s/canary/
```

---

## 8. シークレット管理

### 8.1 Sealed Secrets

```bash
# kubeseal のインストール
brew install kubeseal

# Sealed Secret の作成
kubectl create secret generic db-secret \
  --from-literal=username=gc_storage \
  --from-literal=password=secure-password \
  --dry-run=client -o yaml | \
  kubeseal --format yaml > k8s/secrets/db-secret-sealed.yaml

# 適用
kubectl apply -f k8s/secrets/db-secret-sealed.yaml
```

### 8.2 External Secrets Operator

```yaml
# k8s/external-secrets/db-external-secret.yaml
apiVersion: external-secrets.io/v1beta1
kind: ExternalSecret
metadata:
  name: db-secret
  namespace: gc-storage
spec:
  refreshInterval: 1h
  secretStoreRef:
    kind: ClusterSecretStore
    name: aws-secrets-manager
  target:
    name: db-secret
  data:
    - secretKey: username
      remoteRef:
        key: gc-storage/database
        property: username
    - secretKey: password
      remoteRef:
        key: gc-storage/database
        property: password
```

---

## 9. 環境別 Kustomization

### 9.1 Base 構成

```yaml
# k8s/base/kustomization.yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization

resources:
  - namespace.yaml
  - api/deployment.yaml
  - api/service.yaml
  - frontend/deployment.yaml
  - frontend/service.yaml
  - ingress.yaml

configMapGenerator:
  - name: api-config
    literals:
      - LOG_LEVEL=info
      - LOG_FORMAT=json
```

### 9.2 Production オーバーレイ

```yaml
# environments/production/kustomization.yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization

namespace: gc-storage

resources:
  - ../../k8s/base

images:
  - name: ghcr.io/hiro-mackay/gc-storage/api
    newTag: v1.0.0
  - name: ghcr.io/hiro-mackay/gc-storage/frontend
    newTag: v1.0.0

replicas:
  - name: api
    count: 3
  - name: frontend
    count: 2

patches:
  - path: patches/api-resources.yaml
  - path: patches/hpa.yaml

configMapGenerator:
  - name: api-config
    behavior: merge
    literals:
      - LOG_LEVEL=warn
      - CORS_ALLOWED_ORIGINS=https://gc-storage.example.com
```

```yaml
# environments/production/patches/api-resources.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: api
spec:
  template:
    spec:
      containers:
        - name: api
          resources:
            requests:
              cpu: 200m
              memory: 512Mi
            limits:
              cpu: 1000m
              memory: 1Gi
```

---

## 10. デプロイチェックリスト

### 10.1 デプロイ前

```
□ コードレビュー完了
□ すべてのテストがパス
□ ステージング環境での動作確認
□ マイグレーションの確認
□ 破壊的変更の有無確認
□ 必要に応じてメンテナンス通知
□ バックアップの確認
```

### 10.2 デプロイ中

```
□ マイグレーション実行（必要な場合）
□ デプロイ実行
□ Pod の起動確認
□ ヘルスチェック通過確認
□ ログの異常確認
```

### 10.3 デプロイ後

```
□ 主要機能の動作確認
□ エラーレートの監視
□ レスポンスタイムの確認
□ リソース使用率の確認
□ デプロイ完了の共有
```

---

## 11. トラブルシューティング

### 11.1 Pod が起動しない

```bash
# Pod の状態確認
kubectl get pods -n gc-storage
kubectl describe pod <pod-name> -n gc-storage

# ログの確認
kubectl logs <pod-name> -n gc-storage
kubectl logs <pod-name> -n gc-storage --previous  # 前のコンテナのログ

# イベントの確認
kubectl get events -n gc-storage --sort-by='.lastTimestamp'
```

### 11.2 マイグレーションが失敗

```bash
# dirty 状態の確認
kubectl exec -n gc-storage deployment/api -- \
  /app/migrate -path /app/migrations -database "$DATABASE_URL" version

# 強制的にバージョンを設定
kubectl exec -n gc-storage deployment/api -- \
  /app/migrate -path /app/migrations -database "$DATABASE_URL" force <version>
```

### 11.3 ロールアウトがスタック

```bash
# ロールアウト状況
kubectl rollout status deployment/api -n gc-storage

# ロールアウトを中止
kubectl rollout pause deployment/api -n gc-storage

# ロールアウトを再開
kubectl rollout resume deployment/api -n gc-storage

# 強制的にロールバック
kubectl rollout undo deployment/api -n gc-storage
```

---

## 関連ドキュメント

- [SETUP.md](./SETUP.md) - 開発環境セットアップ
- [INFRASTRUCTURE.md](./INFRASTRUCTURE.md) - インフラストラクチャ設計
- [OPERATIONS.md](./OPERATIONS.md) - 運用ガイド
- [INCIDENT_RESPONSE.md](./INCIDENT_RESPONSE.md) - 障害対応

---

## 更新履歴

| 日付 | バージョン | 内容 |
|------|-----------|------|
| 2026-01-17 | 1.0.0 | 初版作成 |
