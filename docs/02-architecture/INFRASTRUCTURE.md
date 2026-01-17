# GC Storage インフラストラクチャ設計書

## 概要

本ドキュメントでは、GC StorageのKubernetesベースのインフラストラクチャ設計について説明します。

---

## 1. Kubernetes クラスター構成

### 1.1 全体構成図

```
┌─────────────────────────────────────────────────────────────────────────────────┐
│                              Kubernetes Cluster                                 │
│                                                                                 │
│  ┌─────────────────────────────────────────────────────────────────────────────┐│
│  │                              Namespace: gc-storage                          ││
│  │                                                                             ││
│  │  ┌──────────────────┐      ┌──────────────────┐      ┌──────────────────┐   ││
│  │  │   Ingress        │      │   API Deployment │      │ Frontend Deploy  │   ││
│  │  │   (NGINX)        │─────▶│   (3 replicas)   │      │ (2 replicas)     │   ││
│  │  └──────────────────┘      └──────────────────┘      └──────────────────┘   ││
│  │           │                         │                         │             ││
│  │           │                         ▼                         │             ││
│  │           │                ┌──────────────────┐               │             ││
│  │           │                │   API Service    │               │             ││
│  │           │                │   (ClusterIP)    │               │             ││
│  │           │                └──────────────────┘               │             ││
│  │           │                         │                         │             ││
│  │           │         ┌───────────────┼───────────────┐         │             ││
│  │           │         ▼               ▼               ▼         │             ││
│  │           │  ┌────────────┐  ┌────────────┐  ┌────────────┐   │             ││
│  │           │  │ PostgreSQL │  │   Redis    │  │   MinIO    │   │             ││
│  │           │  │ StatefulSet│  │ StatefulSet│  │ StatefulSet│   │             ││
│  │           │  └────────────┘  └────────────┘  └────────────┘   │             ││
│  │           │         │               │               │         │             ││
│  │           │         ▼               ▼               ▼         │             ││
│  │           │  ┌────────────┐  ┌────────────┐  ┌────────────┐   │             ││
│  │           │  │  PVC 100Gi │  │  PVC 10Gi  │  │  PVC 1Ti   │   │             ││
│  │           │  └────────────┘  └────────────┘  └────────────┘   │             ││
│  │           │                                                   │             ││
│  │           └───────────────────────────────────────────────────┘             ││
│  │                                                                             ││
│  │  ┌──────────────────────────────────────────────────────────────────────┐   ││
│  │  │                         ConfigMaps & Secrets                         │   ││
│  │  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐  │   ││
│  │  │  │ api-config  │  │ db-secret   │  │ redis-secret│  │ minio-secret│  │   ││
│  │  │  └─────────────┘  └─────────────┘  └─────────────┘  └─────────────┘  │   ││
│  │  └──────────────────────────────────────────────────────────────────────┘   ││
│  └─────────────────────────────────────────────────────────────────────────────┘│
│                                                                                 │
│  ┌─────────────────────────────────────────────────────────────────────────────┐│
│  │                           Namespace: monitoring                             ││
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐  ┌─────────────────┐     ││
│  │  │ Prometheus  │  │   Grafana   │  │   Loki      │  │ AlertManager    │     ││
│  │  └─────────────┘  └─────────────┘  └─────────────┘  └─────────────────┘     ││
│  └─────────────────────────────────────────────────────────────────────────────┘│
└─────────────────────────────────────────────────────────────────────────────────┘
```

### 1.2 Namespace構成

| Namespace | 用途 |
|-----------|------|
| gc-storage | アプリケーションコンポーネント |
| monitoring | 監視・ログ収集 |
| ingress-nginx | Ingressコントローラー |

---

## 2. Deployment 定義

### 2.1 API Server

```yaml
# k8s/api/deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: api
  namespace: gc-storage
  labels:
    app: gc-storage
    component: api
spec:
  replicas: 3
  selector:
    matchLabels:
      app: gc-storage
      component: api
  strategy:
    type: RollingUpdate
    rollingUpdate:
      maxSurge: 1
      maxUnavailable: 0
  template:
    metadata:
      labels:
        app: gc-storage
        component: api
      annotations:
        prometheus.io/scrape: "true"
        prometheus.io/port: "8080"
        prometheus.io/path: "/metrics"
    spec:
      serviceAccountName: api-service-account
      containers:
        - name: api
          image: gc-storage/api:latest
          imagePullPolicy: Always
          ports:
            - containerPort: 8080
              name: http
          env:
            - name: SERVER_PORT
              value: "8080"
            - name: DB_HOST
              value: postgres-primary
            - name: DB_PORT
              value: "5432"
            - name: DB_USER
              valueFrom:
                secretKeyRef:
                  name: db-secret
                  key: username
            - name: DB_PASSWORD
              valueFrom:
                secretKeyRef:
                  name: db-secret
                  key: password
            - name: DB_NAME
              value: gc_storage
            - name: REDIS_HOST
              value: redis-master
            - name: REDIS_PORT
              value: "6379"
            - name: REDIS_PASSWORD
              valueFrom:
                secretKeyRef:
                  name: redis-secret
                  key: password
            - name: MINIO_ENDPOINT
              value: minio:9000
            - name: MINIO_ACCESS_KEY
              valueFrom:
                secretKeyRef:
                  name: minio-secret
                  key: access-key
            - name: MINIO_SECRET_KEY
              valueFrom:
                secretKeyRef:
                  name: minio-secret
                  key: secret-key
            - name: JWT_ACCESS_SECRET
              valueFrom:
                secretKeyRef:
                  name: jwt-secret
                  key: access-secret
            - name: JWT_REFRESH_SECRET
              valueFrom:
                secretKeyRef:
                  name: jwt-secret
                  key: refresh-secret
          resources:
            requests:
              cpu: 100m
              memory: 256Mi
            limits:
              cpu: 500m
              memory: 512Mi
          livenessProbe:
            httpGet:
              path: /health/live
              port: http
            initialDelaySeconds: 10
            periodSeconds: 10
            timeoutSeconds: 5
            failureThreshold: 3
          readinessProbe:
            httpGet:
              path: /health/ready
              port: http
            initialDelaySeconds: 5
            periodSeconds: 5
            timeoutSeconds: 3
            failureThreshold: 3
          securityContext:
            runAsNonRoot: true
            runAsUser: 1000
            readOnlyRootFilesystem: true
            allowPrivilegeEscalation: false
      affinity:
        podAntiAffinity:
          preferredDuringSchedulingIgnoredDuringExecution:
            - weight: 100
              podAffinityTerm:
                labelSelector:
                  matchLabels:
                    app: gc-storage
                    component: api
                topologyKey: kubernetes.io/hostname
```

### 2.2 Frontend

```yaml
# k8s/frontend/deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: frontend
  namespace: gc-storage
  labels:
    app: gc-storage
    component: frontend
spec:
  replicas: 2
  selector:
    matchLabels:
      app: gc-storage
      component: frontend
  template:
    metadata:
      labels:
        app: gc-storage
        component: frontend
    spec:
      containers:
        - name: frontend
          image: gc-storage/frontend:latest
          imagePullPolicy: Always
          ports:
            - containerPort: 80
              name: http
          resources:
            requests:
              cpu: 50m
              memory: 64Mi
            limits:
              cpu: 200m
              memory: 128Mi
          livenessProbe:
            httpGet:
              path: /
              port: http
            initialDelaySeconds: 5
            periodSeconds: 10
          readinessProbe:
            httpGet:
              path: /
              port: http
            initialDelaySeconds: 3
            periodSeconds: 5
          securityContext:
            runAsNonRoot: true
            runAsUser: 101  # nginx user
            readOnlyRootFilesystem: true
            allowPrivilegeEscalation: false
          volumeMounts:
            - name: nginx-cache
              mountPath: /var/cache/nginx
            - name: nginx-run
              mountPath: /var/run
      volumes:
        - name: nginx-cache
          emptyDir: {}
        - name: nginx-run
          emptyDir: {}
```

---

## 3. Service 定義

### 3.1 API Service

```yaml
# k8s/api/service.yaml
apiVersion: v1
kind: Service
metadata:
  name: api
  namespace: gc-storage
  labels:
    app: gc-storage
    component: api
spec:
  type: ClusterIP
  ports:
    - port: 80
      targetPort: http
      protocol: TCP
      name: http
  selector:
    app: gc-storage
    component: api
```

### 3.2 Frontend Service

```yaml
# k8s/frontend/service.yaml
apiVersion: v1
kind: Service
metadata:
  name: frontend
  namespace: gc-storage
  labels:
    app: gc-storage
    component: frontend
spec:
  type: ClusterIP
  ports:
    - port: 80
      targetPort: http
      protocol: TCP
      name: http
  selector:
    app: gc-storage
    component: frontend
```

---

## 4. Ingress 定義

```yaml
# k8s/ingress.yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: gc-storage
  namespace: gc-storage
  annotations:
    kubernetes.io/ingress.class: nginx
    nginx.ingress.kubernetes.io/ssl-redirect: "true"
    nginx.ingress.kubernetes.io/proxy-body-size: "5g"
    nginx.ingress.kubernetes.io/proxy-read-timeout: "600"
    nginx.ingress.kubernetes.io/proxy-send-timeout: "600"
    cert-manager.io/cluster-issuer: letsencrypt-prod
spec:
  tls:
    - hosts:
        - gc-storage.example.com
        - api.gc-storage.example.com
      secretName: gc-storage-tls
  rules:
    - host: gc-storage.example.com
      http:
        paths:
          - path: /
            pathType: Prefix
            backend:
              service:
                name: frontend
                port:
                  number: 80
    - host: api.gc-storage.example.com
      http:
        paths:
          - path: /
            pathType: Prefix
            backend:
              service:
                name: api
                port:
                  number: 80
```

---

## 5. StatefulSet 定義

### 5.1 PostgreSQL

```yaml
# k8s/postgres/statefulset.yaml
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: postgres
  namespace: gc-storage
spec:
  serviceName: postgres
  replicas: 1
  selector:
    matchLabels:
      app: gc-storage
      component: postgres
  template:
    metadata:
      labels:
        app: gc-storage
        component: postgres
    spec:
      containers:
        - name: postgres
          image: postgres:16-alpine
          ports:
            - containerPort: 5432
              name: postgres
          env:
            - name: POSTGRES_USER
              valueFrom:
                secretKeyRef:
                  name: db-secret
                  key: username
            - name: POSTGRES_PASSWORD
              valueFrom:
                secretKeyRef:
                  name: db-secret
                  key: password
            - name: POSTGRES_DB
              value: gc_storage
            - name: PGDATA
              value: /var/lib/postgresql/data/pgdata
          resources:
            requests:
              cpu: 250m
              memory: 512Mi
            limits:
              cpu: 1000m
              memory: 2Gi
          volumeMounts:
            - name: postgres-data
              mountPath: /var/lib/postgresql/data
            - name: postgres-config
              mountPath: /etc/postgresql/postgresql.conf
              subPath: postgresql.conf
          livenessProbe:
            exec:
              command:
                - pg_isready
                - -U
                - $(POSTGRES_USER)
            initialDelaySeconds: 30
            periodSeconds: 10
          readinessProbe:
            exec:
              command:
                - pg_isready
                - -U
                - $(POSTGRES_USER)
            initialDelaySeconds: 5
            periodSeconds: 5
      volumes:
        - name: postgres-config
          configMap:
            name: postgres-config
  volumeClaimTemplates:
    - metadata:
        name: postgres-data
      spec:
        accessModes: ["ReadWriteOnce"]
        storageClassName: standard
        resources:
          requests:
            storage: 100Gi

---
# k8s/postgres/service.yaml
apiVersion: v1
kind: Service
metadata:
  name: postgres-primary
  namespace: gc-storage
spec:
  type: ClusterIP
  ports:
    - port: 5432
      targetPort: postgres
  selector:
    app: gc-storage
    component: postgres
```

### 5.2 Redis

```yaml
# k8s/redis/statefulset.yaml
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: redis
  namespace: gc-storage
spec:
  serviceName: redis
  replicas: 1
  selector:
    matchLabels:
      app: gc-storage
      component: redis
  template:
    metadata:
      labels:
        app: gc-storage
        component: redis
    spec:
      containers:
        - name: redis
          image: redis:7-alpine
          command:
            - redis-server
            - /etc/redis/redis.conf
          ports:
            - containerPort: 6379
              name: redis
          env:
            - name: REDIS_PASSWORD
              valueFrom:
                secretKeyRef:
                  name: redis-secret
                  key: password
          resources:
            requests:
              cpu: 100m
              memory: 256Mi
            limits:
              cpu: 500m
              memory: 1Gi
          volumeMounts:
            - name: redis-data
              mountPath: /data
            - name: redis-config
              mountPath: /etc/redis
          livenessProbe:
            exec:
              command:
                - redis-cli
                - ping
            initialDelaySeconds: 10
            periodSeconds: 10
          readinessProbe:
            exec:
              command:
                - redis-cli
                - ping
            initialDelaySeconds: 5
            periodSeconds: 5
      volumes:
        - name: redis-config
          configMap:
            name: redis-config
  volumeClaimTemplates:
    - metadata:
        name: redis-data
      spec:
        accessModes: ["ReadWriteOnce"]
        storageClassName: standard
        resources:
          requests:
            storage: 10Gi

---
# k8s/redis/service.yaml
apiVersion: v1
kind: Service
metadata:
  name: redis-master
  namespace: gc-storage
spec:
  type: ClusterIP
  ports:
    - port: 6379
      targetPort: redis
  selector:
    app: gc-storage
    component: redis
```

### 5.3 MinIO

```yaml
# k8s/minio/statefulset.yaml
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: minio
  namespace: gc-storage
spec:
  serviceName: minio
  replicas: 4
  selector:
    matchLabels:
      app: gc-storage
      component: minio
  template:
    metadata:
      labels:
        app: gc-storage
        component: minio
    spec:
      containers:
        - name: minio
          image: minio/minio:latest
          args:
            - server
            - http://minio-{0...3}.minio.gc-storage.svc.cluster.local/data
            - --console-address
            - ":9001"
          ports:
            - containerPort: 9000
              name: api
            - containerPort: 9001
              name: console
          env:
            - name: MINIO_ROOT_USER
              valueFrom:
                secretKeyRef:
                  name: minio-secret
                  key: access-key
            - name: MINIO_ROOT_PASSWORD
              valueFrom:
                secretKeyRef:
                  name: minio-secret
                  key: secret-key
          resources:
            requests:
              cpu: 250m
              memory: 512Mi
            limits:
              cpu: 1000m
              memory: 2Gi
          volumeMounts:
            - name: minio-data
              mountPath: /data
          livenessProbe:
            httpGet:
              path: /minio/health/live
              port: api
            initialDelaySeconds: 10
            periodSeconds: 10
          readinessProbe:
            httpGet:
              path: /minio/health/ready
              port: api
            initialDelaySeconds: 5
            periodSeconds: 5
  volumeClaimTemplates:
    - metadata:
        name: minio-data
      spec:
        accessModes: ["ReadWriteOnce"]
        storageClassName: standard
        resources:
          requests:
            storage: 250Gi

---
# k8s/minio/service.yaml
apiVersion: v1
kind: Service
metadata:
  name: minio
  namespace: gc-storage
spec:
  type: ClusterIP
  ports:
    - port: 9000
      targetPort: api
      name: api
    - port: 9001
      targetPort: console
      name: console
  selector:
    app: gc-storage
    component: minio

---
# Headless Service for StatefulSet
apiVersion: v1
kind: Service
metadata:
  name: minio-headless
  namespace: gc-storage
spec:
  clusterIP: None
  ports:
    - port: 9000
      name: api
  selector:
    app: gc-storage
    component: minio
```

---

## 6. ConfigMap & Secret

### 6.1 ConfigMaps

```yaml
# k8s/configmaps/postgres-config.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: postgres-config
  namespace: gc-storage
data:
  postgresql.conf: |
    listen_addresses = '*'
    max_connections = 200
    shared_buffers = 512MB
    effective_cache_size = 1536MB
    maintenance_work_mem = 128MB
    checkpoint_completion_target = 0.9
    wal_buffers = 16MB
    default_statistics_target = 100
    random_page_cost = 1.1
    effective_io_concurrency = 200
    min_wal_size = 1GB
    max_wal_size = 4GB
    log_timezone = 'Asia/Tokyo'
    timezone = 'Asia/Tokyo'

---
# k8s/configmaps/redis-config.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: redis-config
  namespace: gc-storage
data:
  redis.conf: |
    bind 0.0.0.0
    port 6379
    requirepass ${REDIS_PASSWORD}
    maxmemory 512mb
    maxmemory-policy allkeys-lru
    appendonly yes
    appendfsync everysec

---
# k8s/configmaps/api-config.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: api-config
  namespace: gc-storage
data:
  LOG_LEVEL: "info"
  LOG_FORMAT: "json"
  CORS_ALLOWED_ORIGINS: "https://gc-storage.example.com"
  UPLOAD_MAX_SIZE: "5368709120"  # 5GB
  JWT_ACCESS_EXPIRATION: "15m"
  JWT_REFRESH_EXPIRATION: "7d"
```

### 6.2 Secrets (Sealed Secrets)

```yaml
# k8s/secrets/db-secret.yaml (example - use Sealed Secrets or external secrets)
apiVersion: v1
kind: Secret
metadata:
  name: db-secret
  namespace: gc-storage
type: Opaque
stringData:
  username: gc_storage
  password: <generated-password>

---
# k8s/secrets/redis-secret.yaml
apiVersion: v1
kind: Secret
metadata:
  name: redis-secret
  namespace: gc-storage
type: Opaque
stringData:
  password: <generated-password>

---
# k8s/secrets/minio-secret.yaml
apiVersion: v1
kind: Secret
metadata:
  name: minio-secret
  namespace: gc-storage
type: Opaque
stringData:
  access-key: <generated-access-key>
  secret-key: <generated-secret-key>

---
# k8s/secrets/jwt-secret.yaml
apiVersion: v1
kind: Secret
metadata:
  name: jwt-secret
  namespace: gc-storage
type: Opaque
stringData:
  access-secret: <generated-jwt-access-secret>
  refresh-secret: <generated-jwt-refresh-secret>

---
# k8s/secrets/oauth-secret.yaml
apiVersion: v1
kind: Secret
metadata:
  name: oauth-secret
  namespace: gc-storage
type: Opaque
stringData:
  google-client-id: <google-oauth-client-id>
  google-client-secret: <google-oauth-client-secret>
  github-client-id: <github-oauth-client-id>
  github-client-secret: <github-oauth-client-secret>
```

---

## 7. HorizontalPodAutoscaler

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
  behavior:
    scaleDown:
      stabilizationWindowSeconds: 300
      policies:
        - type: Percent
          value: 10
          periodSeconds: 60
    scaleUp:
      stabilizationWindowSeconds: 0
      policies:
        - type: Percent
          value: 100
          periodSeconds: 15
        - type: Pods
          value: 4
          periodSeconds: 15
      selectPolicy: Max

---
# k8s/frontend/hpa.yaml
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: frontend-hpa
  namespace: gc-storage
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: frontend
  minReplicas: 2
  maxReplicas: 5
  metrics:
    - type: Resource
      resource:
        name: cpu
        target:
          type: Utilization
          averageUtilization: 70
```

---

## 8. PodDisruptionBudget

```yaml
# k8s/api/pdb.yaml
apiVersion: policy/v1
kind: PodDisruptionBudget
metadata:
  name: api-pdb
  namespace: gc-storage
spec:
  minAvailable: 2
  selector:
    matchLabels:
      app: gc-storage
      component: api

---
# k8s/postgres/pdb.yaml
apiVersion: policy/v1
kind: PodDisruptionBudget
metadata:
  name: postgres-pdb
  namespace: gc-storage
spec:
  minAvailable: 1
  selector:
    matchLabels:
      app: gc-storage
      component: postgres
```

---

## 9. NetworkPolicy

```yaml
# k8s/network-policies/api-network-policy.yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: api-network-policy
  namespace: gc-storage
spec:
  podSelector:
    matchLabels:
      app: gc-storage
      component: api
  policyTypes:
    - Ingress
    - Egress
  ingress:
    - from:
        - namespaceSelector:
            matchLabels:
              name: ingress-nginx
        - podSelector:
            matchLabels:
              app: gc-storage
              component: frontend
      ports:
        - protocol: TCP
          port: 8080
  egress:
    - to:
        - podSelector:
            matchLabels:
              app: gc-storage
              component: postgres
      ports:
        - protocol: TCP
          port: 5432
    - to:
        - podSelector:
            matchLabels:
              app: gc-storage
              component: redis
      ports:
        - protocol: TCP
          port: 6379
    - to:
        - podSelector:
            matchLabels:
              app: gc-storage
              component: minio
      ports:
        - protocol: TCP
          port: 9000
    # DNS
    - to:
        - namespaceSelector: {}
          podSelector:
            matchLabels:
              k8s-app: kube-dns
      ports:
        - protocol: UDP
          port: 53

---
# k8s/network-policies/postgres-network-policy.yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: postgres-network-policy
  namespace: gc-storage
spec:
  podSelector:
    matchLabels:
      app: gc-storage
      component: postgres
  policyTypes:
    - Ingress
  ingress:
    - from:
        - podSelector:
            matchLabels:
              app: gc-storage
              component: api
      ports:
        - protocol: TCP
          port: 5432
```

---

## 10. 監視設定

### 10.1 ServiceMonitor (Prometheus)

```yaml
# k8s/monitoring/service-monitor.yaml
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: api-monitor
  namespace: gc-storage
  labels:
    release: prometheus
spec:
  selector:
    matchLabels:
      app: gc-storage
      component: api
  endpoints:
    - port: http
      path: /metrics
      interval: 15s
  namespaceSelector:
    matchNames:
      - gc-storage
```

### 10.2 Grafana Dashboard ConfigMap

```yaml
# k8s/monitoring/grafana-dashboard.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: gc-storage-dashboard
  namespace: monitoring
  labels:
    grafana_dashboard: "1"
data:
  gc-storage.json: |
    {
      "dashboard": {
        "title": "GC Storage Dashboard",
        "panels": [
          {
            "title": "HTTP Request Rate",
            "type": "graph",
            "targets": [
              {
                "expr": "rate(http_requests_total{namespace=\"gc-storage\"}[5m])"
              }
            ]
          },
          {
            "title": "HTTP Request Latency",
            "type": "graph",
            "targets": [
              {
                "expr": "histogram_quantile(0.95, rate(http_request_duration_seconds_bucket{namespace=\"gc-storage\"}[5m]))"
              }
            ]
          }
        ]
      }
    }
```

---

## 11. ディレクトリ構成

```
k8s/
├── namespace.yaml
├── ingress.yaml
├── api/
│   ├── deployment.yaml
│   ├── service.yaml
│   ├── hpa.yaml
│   ├── pdb.yaml
│   └── service-account.yaml
├── frontend/
│   ├── deployment.yaml
│   ├── service.yaml
│   └── hpa.yaml
├── postgres/
│   ├── statefulset.yaml
│   ├── service.yaml
│   └── pdb.yaml
├── redis/
│   ├── statefulset.yaml
│   └── service.yaml
├── minio/
│   ├── statefulset.yaml
│   └── service.yaml
├── configmaps/
│   ├── api-config.yaml
│   ├── postgres-config.yaml
│   └── redis-config.yaml
├── secrets/
│   ├── db-secret.yaml
│   ├── redis-secret.yaml
│   ├── minio-secret.yaml
│   ├── jwt-secret.yaml
│   └── oauth-secret.yaml
├── network-policies/
│   ├── api-network-policy.yaml
│   ├── postgres-network-policy.yaml
│   └── redis-network-policy.yaml
├── monitoring/
│   ├── service-monitor.yaml
│   └── grafana-dashboard.yaml
└── kustomization.yaml
```

---

## 12. Kustomize 設定

```yaml
# k8s/kustomization.yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization

namespace: gc-storage

resources:
  - namespace.yaml
  - ingress.yaml
  - api/deployment.yaml
  - api/service.yaml
  - api/hpa.yaml
  - api/pdb.yaml
  - frontend/deployment.yaml
  - frontend/service.yaml
  - frontend/hpa.yaml
  - postgres/statefulset.yaml
  - postgres/service.yaml
  - redis/statefulset.yaml
  - redis/service.yaml
  - minio/statefulset.yaml
  - minio/service.yaml
  - configmaps/api-config.yaml
  - configmaps/postgres-config.yaml
  - configmaps/redis-config.yaml
  - network-policies/

configMapGenerator:
  - name: api-env
    envs:
      - env/api.env

images:
  - name: gc-storage/api
    newTag: v1.0.0
  - name: gc-storage/frontend
    newTag: v1.0.0
```

---

## 13. デプロイ手順

```bash
# 1. Namespace作成
kubectl apply -f k8s/namespace.yaml

# 2. Secretsの作成（事前にSealed Secretsで暗号化）
kubectl apply -f k8s/secrets/

# 3. ConfigMapsの作成
kubectl apply -f k8s/configmaps/

# 4. データベース関連のデプロイ
kubectl apply -f k8s/postgres/
kubectl apply -f k8s/redis/
kubectl apply -f k8s/minio/

# 5. データベースの準備完了を待つ
kubectl wait --for=condition=ready pod -l component=postgres -n gc-storage --timeout=300s
kubectl wait --for=condition=ready pod -l component=redis -n gc-storage --timeout=300s
kubectl wait --for=condition=ready pod -l component=minio -n gc-storage --timeout=300s

# 6. アプリケーションのデプロイ
kubectl apply -f k8s/api/
kubectl apply -f k8s/frontend/

# 7. Ingressの設定
kubectl apply -f k8s/ingress.yaml

# 8. NetworkPolicyの設定
kubectl apply -f k8s/network-policies/

# または Kustomizeを使用
kubectl apply -k k8s/
```

---

## 関連ドキュメント

- [アーキテクチャ設計](./ARCHITECTURE.md)
- [バックエンド設計](./BACKEND.md)
- [セキュリティ設計](./SECURITY.md)
