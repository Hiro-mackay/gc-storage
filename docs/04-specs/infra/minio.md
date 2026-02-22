# MinIO インフラストラクチャ仕様書

## 概要

本ドキュメントでは、GC StorageにおけるMinIO（S3互換オブジェクトストレージ）の接続管理、Presigned URL生成、マルチパートアップロードの実装仕様を定義します。

**関連アーキテクチャ:**
- [SYSTEM.md](../../02-architecture/SYSTEM.md) - アップロード/ダウンロードフロー
- [file.md](../../03-domains/file.md) - Fileドメイン定義

---

## 1. MinIO接続管理

### 1.1 クライアント構成

```go
// backend/internal/infrastructure/storage/minio/client.go

package minio

import (
    "context"
    "fmt"
    "net/url"
    "time"

    "github.com/minio/minio-go/v7"
    "github.com/minio/minio-go/v7/pkg/credentials"
)

// Config はMinIO接続設定を定義します
type Config struct {
    Endpoint        string // MinIOエンドポイント (例: localhost:9000)
    AccessKeyID     string // アクセスキーID
    SecretAccessKey string // シークレットアクセスキー
    BucketName      string // バケット名
    UseSSL          bool   // SSL使用有無
    Region          string // リージョン (default: us-east-1)
}

// DefaultConfig はデフォルト設定を返します
func DefaultConfig() Config {
    return Config{
        UseSSL: false,
        Region: "us-east-1",
    }
}

// MinIOClient はMinIO操作を提供します
type MinIOClient struct {
    client *minio.Client
    config Config
}

// NewMinIOClient は新しいMinIOClientを作成します
func NewMinIOClient(cfg Config) (*MinIOClient, error) {
    client, err := minio.New(cfg.Endpoint, &minio.Options{
        Creds:  credentials.NewStaticV4(cfg.AccessKeyID, cfg.SecretAccessKey, ""),
        Secure: cfg.UseSSL,
        Region: cfg.Region,
    })
    if err != nil {
        return nil, fmt.Errorf("failed to create minio client: %w", err)
    }

    return &MinIOClient{
        client: client,
        config: cfg,
    }, nil
}

// Client は内部のminio.Clientを返します
func (m *MinIOClient) Client() *minio.Client {
    return m.client
}

// BucketName はバケット名を返します
func (m *MinIOClient) BucketName() string {
    return m.config.BucketName
}

// HealthCheck はMinIOの接続状態を確認します
func (m *MinIOClient) HealthCheck(ctx context.Context) error {
    _, err := m.client.BucketExists(ctx, m.config.BucketName)
    return err
}
```

### 1.2 環境変数設定

```bash
# .env.local
MINIO_ENDPOINT=localhost:9000
MINIO_ACCESS_KEY=minioadmin
MINIO_SECRET_KEY=minioadmin
MINIO_BUCKET=gc-storage
MINIO_USE_SSL=false

# 本番環境 (.env.sample)
MINIO_ENDPOINT=
MINIO_ACCESS_KEY=
MINIO_SECRET_KEY=
MINIO_BUCKET=
MINIO_USE_SSL=true
```

### 1.3 ディレクトリ構成

```
backend/internal/infrastructure/storage/minio/
├── client.go             # MinIO接続管理
├── storage.go            # ストレージサービス実装
├── presigned.go          # Presigned URL生成
├── multipart.go          # マルチパートアップロード
├── lifecycle.go          # ライフサイクル管理
└── keys.go               # オブジェクトキー生成
```

---

## 2. オブジェクトキー設計

### 2.1 設計方針

**MinIOバージョニングを使用**し、同一キーに対してMinIOがバージョンIDを自動管理する。

| 項目 | 設計 |
|------|------|
| キー形式 | `{file_id}` のみ（UUIDv4） |
| バージョン管理 | MinIOネイティブバージョニング |
| 所有者情報 | キーに含めない（DBで管理） |
| セキュリティ | UUIDによる推測困難性 |

**この設計の理由:**
- **所有者変更耐性**: 所有者が変わってもオブジェクト移動不要
- **セキュリティ**: キーから所有者情報が漏洩しない
- **シンプルさ**: MinIOバージョニングでバージョン管理を委譲

### 2.2 キー形式

```go
// backend/internal/domain/valueobject/storage_key.go

package valueobject

import (
    "errors"
    "fmt"

    "github.com/google/uuid"
)

const (
    StorageKeyMaxBytes = 1024
)

var (
    ErrInvalidStorageKey = errors.New("invalid storage key")
)

// StorageKey はMinIO内のオブジェクトキーを表す値オブジェクト
// 形式: {file_id} (UUIDv4)
type StorageKey struct {
    value string
}

// NewStorageKey はファイルIDからStorageKeyを生成します
func NewStorageKey(fileID uuid.UUID) StorageKey {
    return StorageKey{
        value: fileID.String(),
    }
}

// NewStorageKeyFromString は文字列からStorageKeyを生成します
func NewStorageKeyFromString(key string) (StorageKey, error) {
    // UUIDとしてパース可能か検証
    if _, err := uuid.Parse(key); err != nil {
        return StorageKey{}, fmt.Errorf("%w: %v", ErrInvalidStorageKey, err)
    }

    if len(key) > StorageKeyMaxBytes {
        return StorageKey{}, fmt.Errorf("%w: key too long", ErrInvalidStorageKey)
    }

    return StorageKey{value: key}, nil
}

// Value はキー文字列を返します
func (k StorageKey) Value() string {
    return k.value
}

// FileID はファイルIDを取得します
func (k StorageKey) FileID() (uuid.UUID, error) {
    return uuid.Parse(k.value)
}

// String はキー文字列を返します（Stringerインターフェース）
func (k StorageKey) String() string {
    return k.value
}

// ThumbnailKey はサムネイル用のキーを返します
func (k StorageKey) ThumbnailKey(size string) string {
    return fmt.Sprintf("%s/thumbnails/%s", k.value, size)
}
```

### 2.3 キー命名ガイドライン

| パターン | 形式 | 例 |
|---------|------|-----|
| ファイル本体 | `{file_id}` | `550e8400-e29b-41d4-a716-446655440000` |
| サムネイル | `{file_id}/thumbnails/{size}` | `550e8400-e29b.../thumbnails/256x256` |

**注意:** バージョンはMinIOが自動管理（`minio_version_id`でアクセス）

---

## 3. Presigned URL生成

### 3.1 Presigned URL 設定値

| 用途 | 有効期限 | HTTPメソッド |
|------|---------|------------|
| アップロード（通常） | 15分 | PUT |
| アップロード（マルチパート） | 1時間 | PUT |
| ダウンロード | 1時間 | GET |
| プレビュー | 15分 | GET |

### 3.2 Presigned URL 実装

```go
// backend/internal/infrastructure/storage/minio/presigned.go

package minio

import (
    "context"
    "fmt"
    "net/url"
    "time"

    "github.com/minio/minio-go/v7"
)

// PresignedURLOptions はPresigned URL生成のオプションを定義します
type PresignedURLOptions struct {
    ContentType        string            // Content-Type (PUT時のみ)
    ContentDisposition string            // Content-Disposition (GET時のダウンロード名)
    Metadata           map[string]string // カスタムメタデータ
}

const (
    // Presigned URL有効期限
    PresignedUploadExpiry         = 15 * time.Minute
    PresignedMultipartExpiry      = 1 * time.Hour
    PresignedDownloadExpiry       = 1 * time.Hour
    PresignedPreviewExpiry        = 15 * time.Minute

    // 最大ファイルサイズ
    MaxFileSize           int64 = 5 * 1024 * 1024 * 1024 // 5GB
    MultipartThreshold    int64 = 100 * 1024 * 1024      // 100MB
    MultipartPartSize     int64 = 64 * 1024 * 1024       // 64MB
    MaxMultipartParts     int   = 10000
    MaxConcurrentUploads  int   = 5
)

// PresignedURLService はPresigned URL生成を提供します
type PresignedURLService struct {
    client     *minio.Client
    bucketName string
}

// NewPresignedURLService は新しいPresignedURLServiceを作成します
func NewPresignedURLService(client *MinIOClient) *PresignedURLService {
    return &PresignedURLService{
        client:     client.Client(),
        bucketName: client.BucketName(),
    }
}

// GeneratePutURL はアップロード用Presigned URLを生成します
func (s *PresignedURLService) GeneratePutURL(
    ctx context.Context,
    objectKey string,
    expiry time.Duration,
    opts *PresignedURLOptions,
) (string, error) {
    reqParams := make(url.Values)

    if opts != nil && opts.ContentType != "" {
        reqParams.Set("Content-Type", opts.ContentType)
    }

    presignedURL, err := s.client.PresignedPutObject(
        ctx,
        s.bucketName,
        objectKey,
        expiry,
    )
    if err != nil {
        return "", fmt.Errorf("failed to generate presigned put URL: %w", err)
    }

    return presignedURL.String(), nil
}

// GenerateGetURL はダウンロード用Presigned URLを生成します
func (s *PresignedURLService) GenerateGetURL(
    ctx context.Context,
    objectKey string,
    expiry time.Duration,
    opts *PresignedURLOptions,
) (string, error) {
    reqParams := make(url.Values)

    if opts != nil && opts.ContentDisposition != "" {
        reqParams.Set("response-content-disposition", opts.ContentDisposition)
    }

    presignedURL, err := s.client.PresignedGetObject(
        ctx,
        s.bucketName,
        objectKey,
        expiry,
        reqParams,
    )
    if err != nil {
        return "", fmt.Errorf("failed to generate presigned get URL: %w", err)
    }

    return presignedURL.String(), nil
}

// GenerateDownloadURL はダウンロード用URLを生成します（ファイル名付き）
func (s *PresignedURLService) GenerateDownloadURL(
    ctx context.Context,
    objectKey string,
    filename string,
) (string, error) {
    opts := &PresignedURLOptions{
        ContentDisposition: fmt.Sprintf(`attachment; filename="%s"`, filename),
    }
    return s.GenerateGetURL(ctx, objectKey, PresignedDownloadExpiry, opts)
}

// GeneratePreviewURL はプレビュー用URLを生成します（インライン表示）
func (s *PresignedURLService) GeneratePreviewURL(
    ctx context.Context,
    objectKey string,
    filename string,
) (string, error) {
    opts := &PresignedURLOptions{
        ContentDisposition: fmt.Sprintf(`inline; filename="%s"`, filename),
    }
    return s.GenerateGetURL(ctx, objectKey, PresignedPreviewExpiry, opts)
}
```

### 3.3 ストレージサービスインターフェース

```go
// backend/internal/domain/service/storage.go

package service

import (
    "context"
    "io"
    "time"
)

// ObjectInfo はオブジェクト情報を表します
type ObjectInfo struct {
    Key          string
    Size         int64
    ContentType  string
    ETag         string
    MD5          string
    SHA256       string
    LastModified time.Time
    Metadata     map[string]string
}

// StorageService はオブジェクトストレージ操作のインターフェースを定義します
type StorageService interface {
    // Presigned URL
    GeneratePutURL(ctx context.Context, objectKey string, expiry time.Duration) (string, error)
    GenerateGetURL(ctx context.Context, objectKey string, expiry time.Duration) (string, error)
    GenerateDownloadURL(ctx context.Context, objectKey string, filename string) (string, error)
    GeneratePreviewURL(ctx context.Context, objectKey string, filename string) (string, error)

    // オブジェクト操作
    ObjectExists(ctx context.Context, objectKey string) (bool, error)
    GetObjectInfo(ctx context.Context, objectKey string) (*ObjectInfo, error)
    DeleteObject(ctx context.Context, objectKey string) error
    CopyObject(ctx context.Context, srcKey, dstKey string) error

    // マルチパートアップロード
    CreateMultipartUpload(ctx context.Context, objectKey string) (uploadID string, err error)
    GeneratePartUploadURL(ctx context.Context, objectKey, uploadID string, partNumber int) (string, error)
    CompleteMultipartUpload(ctx context.Context, objectKey, uploadID string, parts []CompletedPart) error
    AbortMultipartUpload(ctx context.Context, objectKey, uploadID string) error
    ListParts(ctx context.Context, objectKey, uploadID string) ([]PartInfo, error)
}

// CompletedPart は完了したパート情報を表します
type CompletedPart struct {
    PartNumber int
    ETag       string
}

// PartInfo はパート情報を表します
type PartInfo struct {
    PartNumber   int
    Size         int64
    ETag         string
    LastModified time.Time
}
```

---

## 4. マルチパートアップロード

### 4.1 マルチパートアップロードの流れ（Webhook駆動）

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                   Multipart Upload Flow（Webhook駆動）                       │
└─────────────────────────────────────────────────────────────────────────────┘

【Phase 1: 初期化】
┌────────┐          ┌──────────┐          ┌──────────┐          ┌───────┐
│ Client │          │   API    │          │   DB     │          │ MinIO │
└────┬───┘          └────┬─────┘          └────┬─────┘          └───┬───┘
     │  1. POST /upload/multipart/init         │                    │
     │─────────────────▶│                      │                    │
     │                  │  2. CreateMultipartUpload                 │
     │                  │─────────────────────────────────────────▶│
     │                  │                      │                    │
     │                  │  3. uploadId                              │
     │                  │◀─────────────────────────────────────────│
     │                  │                      │                    │
     │                  │  4. Save UploadSession (status=pending)  │
     │                  │─────────────────────▶│                    │
     │                  │                      │                    │
     │  5. Return uploadId, partCount, presignedUrls               │
     │◀─────────────────│                      │                    │

【Phase 2: パートアップロード】（並列5つまで）
     │  6. PUT to MinIO directly (part 1)                           │
     │──────────────────────────────────────────────────────────────▶
     │                  │                      │                    │
     │  7. 200 OK + ETag                                            │
     │◀──────────────────────────────────────────────────────────────
     │                  │                      │                    │
     │  (Repeat for each part)                 │                    │

【Phase 3: 完了（クライアント → MinIO S3 API）】
     │  8. POST CompleteMultipartUpload (S3 API) to MinIO           │
     │      {uploadId, parts: [{partNumber, eTag},...]}            │
     │──────────────────────────────────────────────────────────────▶
     │                  │                      │                    │
     │  9. 200 OK (Final ETag, VersionID)                           │
     │◀──────────────────────────────────────────────────────────────

【Phase 4: Webhook通知（MinIO → API）】
     │                  │                      │                    │
     │                  │  10. s3:ObjectCreated:CompleteMultipartUpload
     │                  │◀─────────────────────────────────────────│
     │                  │                      │                    │
     │                  │  11. Update File status (pending→active) │
     │                  │      Create FileVersion                  │
     │                  │─────────────────────▶│                    │
     │                  │                      │                    │
     │                  │  12. Return 200 OK   │                    │
     │                  │─────────────────────────────────────────▶│
```

**変更点（旧設計との違い）:**

| 項目 | 旧設計 | 新設計（Webhook駆動） |
|------|--------|----------------------|
| 完了処理 | クライアント → API → MinIO | クライアント → MinIO（直接S3 API） |
| ファイル有効化 | APIがMinIO完了後に実行 | Webhook受信時に実行 |
| 整合性 | クライアント依存 | MinIO完了を保証 |
| パートURL取得 | パートごとにAPIコール | 初期化時に全パート分を一括返却 |

### 4.2 マルチパートアップロード実装

```go
// backend/internal/infrastructure/storage/minio/multipart.go

package minio

import (
    "context"
    "fmt"
    "net/url"
    "sort"
    "time"

    "github.com/minio/minio-go/v7"

    "github.com/Hiro-mackay/gc-storage/backend/internal/domain/service"
)

// MultipartService はマルチパートアップロードを提供します
type MultipartService struct {
    client     *minio.Client
    bucketName string
}

// NewMultipartService は新しいMultipartServiceを作成します
func NewMultipartService(client *MinIOClient) *MultipartService {
    return &MultipartService{
        client:     client.Client(),
        bucketName: client.BucketName(),
    }
}

// CreateMultipartUpload はマルチパートアップロードを開始します
func (s *MultipartService) CreateMultipartUpload(
    ctx context.Context,
    objectKey string,
    opts *minio.PutObjectOptions,
) (string, error) {
    if opts == nil {
        opts = &minio.PutObjectOptions{}
    }

    uploadID, err := s.client.NewMultipartUpload(ctx, s.bucketName, objectKey, *opts)
    if err != nil {
        return "", fmt.Errorf("failed to create multipart upload: %w", err)
    }

    return uploadID, nil
}

// GeneratePartUploadURL はパートアップロード用Presigned URLを生成します
func (s *MultipartService) GeneratePartUploadURL(
    ctx context.Context,
    objectKey string,
    uploadID string,
    partNumber int,
) (string, error) {
    // MinIO SDKにはパート用のPresigned URL生成がないため、
    // 手動でURLを構築する必要があります
    reqParams := make(url.Values)
    reqParams.Set("uploadId", uploadID)
    reqParams.Set("partNumber", fmt.Sprintf("%d", partNumber))

    presignedURL, err := s.client.Presign(
        ctx,
        "PUT",
        s.bucketName,
        objectKey,
        PresignedMultipartExpiry,
        reqParams,
    )
    if err != nil {
        return "", fmt.Errorf("failed to generate part upload URL: %w", err)
    }

    return presignedURL.String(), nil
}

// CompleteMultipartUpload はマルチパートアップロードを完了します
func (s *MultipartService) CompleteMultipartUpload(
    ctx context.Context,
    objectKey string,
    uploadID string,
    parts []service.CompletedPart,
) (string, error) {
    // パート番号でソート
    sort.Slice(parts, func(i, j int) bool {
        return parts[i].PartNumber < parts[j].PartNumber
    })

    // minio.CompletePart に変換
    completeParts := make([]minio.CompletePart, len(parts))
    for i, p := range parts {
        completeParts[i] = minio.CompletePart{
            PartNumber: p.PartNumber,
            ETag:       p.ETag,
        }
    }

    uploadInfo, err := s.client.CompleteMultipartUpload(
        ctx,
        s.bucketName,
        objectKey,
        uploadID,
        completeParts,
        minio.PutObjectOptions{},
    )
    if err != nil {
        return "", fmt.Errorf("failed to complete multipart upload: %w", err)
    }

    return uploadInfo.ETag, nil
}

// AbortMultipartUpload はマルチパートアップロードを中止します
func (s *MultipartService) AbortMultipartUpload(
    ctx context.Context,
    objectKey string,
    uploadID string,
) error {
    err := s.client.AbortMultipartUpload(ctx, s.bucketName, objectKey, uploadID)
    if err != nil {
        return fmt.Errorf("failed to abort multipart upload: %w", err)
    }
    return nil
}

// ListParts はアップロード済みのパートを一覧します
func (s *MultipartService) ListParts(
    ctx context.Context,
    objectKey string,
    uploadID string,
) ([]service.PartInfo, error) {
    result, err := s.client.ListObjectParts(
        ctx,
        s.bucketName,
        objectKey,
        uploadID,
        0,
        MaxMultipartParts,
    )
    if err != nil {
        return nil, fmt.Errorf("failed to list parts: %w", err)
    }

    parts := make([]service.PartInfo, len(result.ObjectParts))
    for i, p := range result.ObjectParts {
        parts[i] = service.PartInfo{
            PartNumber:   p.PartNumber,
            Size:         p.Size,
            ETag:         p.ETag,
            LastModified: p.LastModified,
        }
    }

    return parts, nil
}

// ListIncompleteUploads は未完了のマルチパートアップロードを一覧します
func (s *MultipartService) ListIncompleteUploads(
    ctx context.Context,
    prefix string,
) ([]IncompleteUpload, error) {
    var uploads []IncompleteUpload

    for upload := range s.client.ListIncompleteUploads(ctx, s.bucketName, prefix, true) {
        if upload.Err != nil {
            return nil, fmt.Errorf("error listing incomplete uploads: %w", upload.Err)
        }
        uploads = append(uploads, IncompleteUpload{
            ObjectKey: upload.Key,
            UploadID:  upload.UploadID,
            Initiated: upload.Initiated,
        })
    }

    return uploads, nil
}

// IncompleteUpload は未完了のアップロード情報を表します
type IncompleteUpload struct {
    ObjectKey string
    UploadID  string
    Initiated time.Time
}

// CalculatePartCount はファイルサイズからパート数を計算します
func CalculatePartCount(fileSize int64) int {
    if fileSize <= MultipartPartSize {
        return 1
    }

    partCount := int(fileSize / MultipartPartSize)
    if fileSize%MultipartPartSize > 0 {
        partCount++
    }

    if partCount > MaxMultipartParts {
        partCount = MaxMultipartParts
    }

    return partCount
}

// CalculatePartSize は各パートのサイズを計算します
func CalculatePartSize(fileSize int64, partNumber, totalParts int) int64 {
    if partNumber < totalParts {
        return MultipartPartSize
    }
    // 最後のパートは残りサイズ
    return fileSize - MultipartPartSize*int64(totalParts-1)
}
```

---

## 5. オブジェクト操作

### 5.1 ストレージサービス実装

```go
// backend/internal/infrastructure/storage/minio/storage.go

package minio

import (
    "context"
    "fmt"
    "io"
    "time"

    "github.com/minio/minio-go/v7"

    "github.com/Hiro-mackay/gc-storage/backend/internal/domain/service"
)

// StorageServiceImpl はStorageServiceの実装です
type StorageServiceImpl struct {
    client     *minio.Client
    bucketName string
    presigned  *PresignedURLService
    multipart  *MultipartService
}

// NewStorageService は新しいStorageServiceを作成します
func NewStorageService(client *MinIOClient) *StorageServiceImpl {
    return &StorageServiceImpl{
        client:     client.Client(),
        bucketName: client.BucketName(),
        presigned:  NewPresignedURLService(client),
        multipart:  NewMultipartService(client),
    }
}

// GeneratePutURL はアップロード用Presigned URLを生成します
func (s *StorageServiceImpl) GeneratePutURL(ctx context.Context, objectKey string, expiry time.Duration) (string, error) {
    return s.presigned.GeneratePutURL(ctx, objectKey, expiry, nil)
}

// GenerateGetURL はダウンロード用Presigned URLを生成します
func (s *StorageServiceImpl) GenerateGetURL(ctx context.Context, objectKey string, expiry time.Duration) (string, error) {
    return s.presigned.GenerateGetURL(ctx, objectKey, expiry, nil)
}

// GenerateDownloadURL はダウンロード用URLを生成します
func (s *StorageServiceImpl) GenerateDownloadURL(ctx context.Context, objectKey string, filename string) (string, error) {
    return s.presigned.GenerateDownloadURL(ctx, objectKey, filename)
}

// GeneratePreviewURL はプレビュー用URLを生成します
func (s *StorageServiceImpl) GeneratePreviewURL(ctx context.Context, objectKey string, filename string) (string, error) {
    return s.presigned.GeneratePreviewURL(ctx, objectKey, filename)
}

// ObjectExists はオブジェクトが存在するか確認します
func (s *StorageServiceImpl) ObjectExists(ctx context.Context, objectKey string) (bool, error) {
    _, err := s.client.StatObject(ctx, s.bucketName, objectKey, minio.StatObjectOptions{})
    if err != nil {
        errResponse := minio.ToErrorResponse(err)
        if errResponse.Code == "NoSuchKey" {
            return false, nil
        }
        return false, fmt.Errorf("failed to check object existence: %w", err)
    }
    return true, nil
}

// GetObjectInfo はオブジェクト情報を取得します
func (s *StorageServiceImpl) GetObjectInfo(ctx context.Context, objectKey string) (*service.ObjectInfo, error) {
    info, err := s.client.StatObject(ctx, s.bucketName, objectKey, minio.StatObjectOptions{})
    if err != nil {
        return nil, fmt.Errorf("failed to get object info: %w", err)
    }

    return &service.ObjectInfo{
        Key:          info.Key,
        Size:         info.Size,
        ContentType:  info.ContentType,
        ETag:         info.ETag,
        MD5:          info.ETag, // ETagはMD5の場合が多いが、マルチパートの場合は異なる
        LastModified: info.LastModified,
        Metadata:     info.UserMetadata,
    }, nil
}

// DeleteObject はオブジェクトを削除します
func (s *StorageServiceImpl) DeleteObject(ctx context.Context, objectKey string) error {
    err := s.client.RemoveObject(ctx, s.bucketName, objectKey, minio.RemoveObjectOptions{})
    if err != nil {
        return fmt.Errorf("failed to delete object: %w", err)
    }
    return nil
}

// DeleteObjects は複数オブジェクトを一括削除します
func (s *StorageServiceImpl) DeleteObjects(ctx context.Context, objectKeys []string) error {
    objectsCh := make(chan minio.ObjectInfo, len(objectKeys))

    go func() {
        defer close(objectsCh)
        for _, key := range objectKeys {
            objectsCh <- minio.ObjectInfo{Key: key}
        }
    }()

    errorCh := s.client.RemoveObjects(ctx, s.bucketName, objectsCh, minio.RemoveObjectsOptions{})

    var errors []error
    for e := range errorCh {
        if e.Err != nil {
            errors = append(errors, fmt.Errorf("failed to delete %s: %w", e.ObjectName, e.Err))
        }
    }

    if len(errors) > 0 {
        return fmt.Errorf("failed to delete some objects: %v", errors)
    }

    return nil
}

// CopyObject はオブジェクトをコピーします
func (s *StorageServiceImpl) CopyObject(ctx context.Context, srcKey, dstKey string) error {
    src := minio.CopySrcOptions{
        Bucket: s.bucketName,
        Object: srcKey,
    }

    dst := minio.CopyDestOptions{
        Bucket: s.bucketName,
        Object: dstKey,
    }

    _, err := s.client.CopyObject(ctx, dst, src)
    if err != nil {
        return fmt.Errorf("failed to copy object: %w", err)
    }

    return nil
}

// GetObject はオブジェクトを直接取得します（内部使用のみ）
func (s *StorageServiceImpl) GetObject(ctx context.Context, objectKey string) (io.ReadCloser, error) {
    object, err := s.client.GetObject(ctx, s.bucketName, objectKey, minio.GetObjectOptions{})
    if err != nil {
        return nil, fmt.Errorf("failed to get object: %w", err)
    }
    return object, nil
}

// PutObject はオブジェクトを直接アップロードします（内部使用のみ、小さなファイル向け）
func (s *StorageServiceImpl) PutObject(ctx context.Context, objectKey string, reader io.Reader, size int64, contentType string) error {
    _, err := s.client.PutObject(ctx, s.bucketName, objectKey, reader, size, minio.PutObjectOptions{
        ContentType: contentType,
    })
    if err != nil {
        return fmt.Errorf("failed to put object: %w", err)
    }
    return nil
}

// マルチパート操作の委譲
func (s *StorageServiceImpl) CreateMultipartUpload(ctx context.Context, objectKey string) (string, error) {
    return s.multipart.CreateMultipartUpload(ctx, objectKey, nil)
}

func (s *StorageServiceImpl) GeneratePartUploadURL(ctx context.Context, objectKey, uploadID string, partNumber int) (string, error) {
    return s.multipart.GeneratePartUploadURL(ctx, objectKey, uploadID, partNumber)
}

func (s *StorageServiceImpl) CompleteMultipartUpload(ctx context.Context, objectKey, uploadID string, parts []service.CompletedPart) error {
    _, err := s.multipart.CompleteMultipartUpload(ctx, objectKey, uploadID, parts)
    return err
}

func (s *StorageServiceImpl) AbortMultipartUpload(ctx context.Context, objectKey, uploadID string) error {
    return s.multipart.AbortMultipartUpload(ctx, objectKey, uploadID)
}

func (s *StorageServiceImpl) ListParts(ctx context.Context, objectKey, uploadID string) ([]service.PartInfo, error) {
    return s.multipart.ListParts(ctx, objectKey, uploadID)
}

// Verify interface compliance
var _ service.StorageService = (*StorageServiceImpl)(nil)
```

---

## 6. バケットセットアップ

### 6.1 バケット初期化

```go
// backend/internal/infrastructure/storage/minio/setup.go

package minio

import (
    "context"
    "fmt"

    "github.com/minio/minio-go/v7"
)

// BucketSetup はバケットの初期化を行います
type BucketSetup struct {
    client     *minio.Client
    bucketName string
    region     string
}

// NewBucketSetup は新しいBucketSetupを作成します
func NewBucketSetup(client *MinIOClient, region string) *BucketSetup {
    return &BucketSetup{
        client:     client.Client(),
        bucketName: client.BucketName(),
        region:     region,
    }
}

// Setup はバケットを作成し初期設定を行います
func (s *BucketSetup) Setup(ctx context.Context) error {
    // バケット存在確認
    exists, err := s.client.BucketExists(ctx, s.bucketName)
    if err != nil {
        return fmt.Errorf("failed to check bucket existence: %w", err)
    }

    // バケットが存在しない場合は作成
    if !exists {
        err = s.client.MakeBucket(ctx, s.bucketName, minio.MakeBucketOptions{
            Region: s.region,
        })
        if err != nil {
            return fmt.Errorf("failed to create bucket: %w", err)
        }
    }

    // バージョニングを有効化（必須）
    if err := s.SetVersioning(ctx, true); err != nil {
        return fmt.Errorf("failed to enable versioning: %w", err)
    }

    return nil
}

// SetVersioning はバージョニングを有効にします
func (s *BucketSetup) SetVersioning(ctx context.Context, enabled bool) error {
    config := minio.BucketVersioningConfiguration{}
    if enabled {
        config.Status = "Enabled"
    } else {
        config.Status = "Suspended"
    }

    err := s.client.SetBucketVersioning(ctx, s.bucketName, config)
    if err != nil {
        return fmt.Errorf("failed to set versioning: %w", err)
    }

    return nil
}
```

### 6.2 ライフサイクル管理

```go
// backend/internal/infrastructure/storage/minio/lifecycle.go

package minio

import (
    "context"
    "fmt"

    "github.com/minio/minio-go/v7/pkg/lifecycle"
)

// LifecycleManager はライフサイクルポリシーを管理します
type LifecycleManager struct {
    client     *MinIOClient
    bucketName string
}

// NewLifecycleManager は新しいLifecycleManagerを作成します
func NewLifecycleManager(client *MinIOClient) *LifecycleManager {
    return &LifecycleManager{
        client:     client,
        bucketName: client.BucketName(),
    }
}

// SetIncompleteUploadExpiration は未完了のマルチパートアップロードの有効期限を設定します
func (m *LifecycleManager) SetIncompleteUploadExpiration(ctx context.Context, days int) error {
    config := lifecycle.NewConfiguration()

    config.Rules = []lifecycle.Rule{
        {
            ID:     "abort-incomplete-multipart-uploads",
            Status: "Enabled",
            AbortIncompleteMultipartUpload: lifecycle.AbortIncompleteMultipartUpload{
                DaysAfterInitiation: lifecycle.ExpirationDays(days),
            },
        },
    }

    err := m.client.Client().SetBucketLifecycle(ctx, m.bucketName, config)
    if err != nil {
        return fmt.Errorf("failed to set lifecycle policy: %w", err)
    }

    return nil
}

// SetObjectExpiration はオブジェクトの有効期限を設定します（プレフィックス指定可）
func (m *LifecycleManager) SetObjectExpiration(ctx context.Context, prefix string, days int) error {
    config := lifecycle.NewConfiguration()

    rule := lifecycle.Rule{
        ID:     fmt.Sprintf("expire-%s-%d-days", prefix, days),
        Status: "Enabled",
        Expiration: lifecycle.Expiration{
            Days: lifecycle.ExpirationDays(days),
        },
    }

    if prefix != "" {
        rule.RuleFilter = lifecycle.Filter{
            Prefix: prefix,
        }
    }

    config.Rules = []lifecycle.Rule{rule}

    err := m.client.Client().SetBucketLifecycle(ctx, m.bucketName, config)
    if err != nil {
        return fmt.Errorf("failed to set lifecycle policy: %w", err)
    }

    return nil
}
```

---

## 7. Bucket Notification（Webhook）

アップロード完了検知のため、MinIOのBucket Notificationを使用してWebhook通知を受信する。

### 7.1 設計方針

| 項目 | 設計 |
|------|------|
| 通知方式 | Webhook（HTTP POST） |
| イベント | `s3:ObjectCreated:*` |
| 同期モード | `MINIO_API_SYNC_EVENTS=on`（推奨） |
| 冪等性 | storage_key + minio_version_id で重複チェック |

**Webhook駆動の利点:**
- クライアントからの完了通知が不要（整合性保証）
- MinIOが完了を確定してから通知（データ整合性）
- サーバー側で一意に完了判定

### 7.2 MinIO設定

**環境変数（docker-compose.yml）:**

```yaml
services:
  minio:
    image: minio/minio:latest
    environment:
      MINIO_ROOT_USER: minioadmin
      MINIO_ROOT_PASSWORD: minioadmin
      # Webhook設定
      MINIO_NOTIFY_WEBHOOK_ENABLE_PRIMARY: "on"
      MINIO_NOTIFY_WEBHOOK_ENDPOINT_PRIMARY: "http://api:8080/internal/webhooks/minio"
      MINIO_NOTIFY_WEBHOOK_AUTH_TOKEN_PRIMARY: "${MINIO_WEBHOOK_SECRET}"
      # 同期モード（Webhookを同期的に処理）
      MINIO_API_SYNC_EVENTS: "on"
    command: server /data --console-address ":9001"
```

**mcコマンドによるイベント設定:**

```bash
# バケットにWebhook通知を設定
mc event add myminio/gc-storage arn:minio:sqs::primary:webhook \
  --event put \
  --suffix ""

# 設定確認
mc event list myminio/gc-storage
```

### 7.3 Webhookハンドラー実装

```go
// backend/internal/interface/handler/webhook_handler.go

package handler

import (
    "crypto/subtle"
    "encoding/json"
    "net/http"

    "github.com/labstack/echo/v4"

    "github.com/Hiro-mackay/gc-storage/backend/internal/usecase/file/command"
)

// MinIOEvent はMinIOからのS3イベント通知
type MinIOEvent struct {
    Records []MinIOEventRecord `json:"Records"`
}

type MinIOEventRecord struct {
    EventVersion string        `json:"eventVersion"`
    EventSource  string        `json:"eventSource"`
    EventName    string        `json:"eventName"`
    EventTime    string        `json:"eventTime"`
    S3           MinIOEventS3  `json:"s3"`
}

type MinIOEventS3 struct {
    Bucket MinIOEventBucket `json:"bucket"`
    Object MinIOEventObject `json:"object"`
}

type MinIOEventBucket struct {
    Name string `json:"name"`
}

type MinIOEventObject struct {
    Key       string `json:"key"`
    Size      int64  `json:"size"`
    ETag      string `json:"eTag"`
    VersionID string `json:"versionId"`
}

// WebhookHandler はMinIO Webhookを処理
type WebhookHandler struct {
    completeUploadCmd *command.CompleteUploadCommand
    webhookSecret     string
}

// NewWebhookHandler は新しいWebhookHandlerを作成
func NewWebhookHandler(
    completeUploadCmd *command.CompleteUploadCommand,
    webhookSecret string,
) *WebhookHandler {
    return &WebhookHandler{
        completeUploadCmd: completeUploadCmd,
        webhookSecret:     webhookSecret,
    }
}

// HandleMinIOWebhook はMinIOからのWebhook通知を処理
func (h *WebhookHandler) HandleMinIOWebhook(c echo.Context) error {
    // 1. 認証トークン検証
    authToken := c.Request().Header.Get("Authorization")
    if !h.validateToken(authToken) {
        return c.NoContent(http.StatusUnauthorized)
    }

    // 2. イベントパース
    var event MinIOEvent
    if err := json.NewDecoder(c.Request().Body).Decode(&event); err != nil {
        return c.JSON(http.StatusBadRequest, map[string]string{
            "error": "invalid event format",
        })
    }

    // 3. 各レコードを処理
    for _, record := range event.Records {
        // ObjectCreated イベントのみ処理
        if record.EventName != "s3:ObjectCreated:Put" &&
           record.EventName != "s3:ObjectCreated:CompleteMultipartUpload" {
            continue
        }

        // サムネイルは除外
        if strings.Contains(record.S3.Object.Key, "/thumbnails/") {
            continue
        }

        // アップロード完了処理
        input := command.CompleteUploadInput{
            StorageKey:     record.S3.Object.Key,
            MinioVersionID: record.S3.Object.VersionID,
            Size:           record.S3.Object.Size,
            ETag:           record.S3.Object.ETag,
        }

        if err := h.completeUploadCmd.Execute(c.Request().Context(), input); err != nil {
            // エラーログ出力（リトライのため200を返す場合もある）
            c.Logger().Error("failed to complete upload", "error", err, "key", record.S3.Object.Key)
            // 冪等性のため、既に処理済みの場合はエラーとしない
            continue
        }
    }

    return c.NoContent(http.StatusOK)
}

// validateToken は認証トークンを検証
func (h *WebhookHandler) validateToken(authToken string) bool {
    if h.webhookSecret == "" {
        return true // 開発環境では認証スキップ可能
    }
    expected := "Bearer " + h.webhookSecret
    return subtle.ConstantTimeCompare([]byte(authToken), []byte(expected)) == 1
}
```

### 7.4 ルーティング設定

```go
// backend/internal/interface/router/router.go

func (r *Router) setupInternalRoutes() {
    internal := r.echo.Group("/internal")

    // MinIO Webhook（認証はハンドラー内で実施）
    internal.POST("/webhooks/minio", r.handlers.Webhook.HandleMinIOWebhook)
}
```

### 7.5 環境変数

```bash
# .env.local
MINIO_WEBHOOK_SECRET=local-dev-secret

# .env.sample（本番用）
MINIO_WEBHOOK_SECRET=  # 強力なランダム文字列を設定
```

### 7.6 冪等性保証

同じイベントが複数回送信される可能性があるため、冪等性を保証する。

```go
// UploadSessionRepository に追加
type UploadSessionRepository interface {
    // ...

    // FindByStorageKeyAndVersionID は重複チェック用
    FindByStorageKeyAndVersionID(ctx context.Context, storageKey string, versionID string) (*entity.UploadSession, error)

    // IsAlreadyCompleted は既に完了済みかチェック
    IsAlreadyCompleted(ctx context.Context, storageKey string, versionID string) (bool, error)
}
```

```go
// CompleteUploadCommand での冪等性チェック
func (c *CompleteUploadCommand) Execute(ctx context.Context, input CompleteUploadInput) error {
    // 冪等性チェック：既に同じバージョンで完了済みか
    completed, err := c.sessionRepo.IsAlreadyCompleted(ctx, input.StorageKey, input.MinioVersionID)
    if err != nil {
        return err
    }
    if completed {
        return nil // 既に処理済み、正常終了
    }

    // ... 以降の処理
}
```

---

## 8. クリーンアップジョブ

### 8.1 孤立オブジェクトクリーンアップ

```go
// backend/internal/infrastructure/storage/minio/cleanup.go

package minio

import (
    "context"
    "fmt"
    "log/slog"
    "time"

    "github.com/minio/minio-go/v7"

    "github.com/Hiro-mackay/gc-storage/backend/internal/domain/repository"
)

// CleanupService は不要オブジェクトのクリーンアップを提供します
type CleanupService struct {
    client     *minio.Client
    bucketName string
    fileRepo   repository.FileRepository
    logger     *slog.Logger
}

// NewCleanupService は新しいCleanupServiceを作成します
func NewCleanupService(
    client *MinIOClient,
    fileRepo repository.FileRepository,
    logger *slog.Logger,
) *CleanupService {
    return &CleanupService{
        client:     client.Client(),
        bucketName: client.BucketName(),
        fileRepo:   fileRepo,
        logger:     logger,
    }
}

// CleanupOrphanedObjects は孤立オブジェクト（DBに対応するレコードがない）を削除します
func (s *CleanupService) CleanupOrphanedObjects(ctx context.Context) (int, error) {
    deleted := 0

    opts := minio.ListObjectsOptions{
        Recursive: true,
    }

    for object := range s.client.ListObjects(ctx, s.bucketName, opts) {
        if object.Err != nil {
            s.logger.Error("error listing objects", "error", object.Err)
            continue
        }

        // storage_keyからファイルを検索
        storageKey, err := ParseStorageKey(object.Key)
        if err != nil {
            // 不正なキー形式のオブジェクトは削除候補
            s.logger.Warn("found object with invalid key format", "key", object.Key)
            continue
        }

        // DBにファイルが存在するか確認
        exists, err := s.fileRepo.ExistsByStorageKey(ctx, object.Key)
        if err != nil {
            s.logger.Error("error checking file existence", "key", object.Key, "error", err)
            continue
        }

        if !exists {
            // 孤立オブジェクトを削除
            err = s.client.RemoveObject(ctx, s.bucketName, object.Key, minio.RemoveObjectOptions{})
            if err != nil {
                s.logger.Error("error deleting orphaned object", "key", object.Key, "error", err)
                continue
            }

            s.logger.Info("deleted orphaned object", "key", object.Key)
            deleted++
        }
    }

    return deleted, nil
}

// CleanupIncompleteUploads は未完了のマルチパートアップロードをクリーンアップします
func (s *CleanupService) CleanupIncompleteUploads(ctx context.Context, olderThan time.Duration) (int, error) {
    deleted := 0

    threshold := time.Now().Add(-olderThan)

    multipartSvc := NewMultipartService(&MinIOClient{client: s.client, config: Config{BucketName: s.bucketName}})

    uploads, err := multipartSvc.ListIncompleteUploads(ctx, "")
    if err != nil {
        return 0, err
    }

    for _, upload := range uploads {
        if upload.Initiated.Before(threshold) {
            err := multipartSvc.AbortMultipartUpload(ctx, upload.ObjectKey, upload.UploadID)
            if err != nil {
                s.logger.Error("error aborting incomplete upload",
                    "key", upload.ObjectKey,
                    "uploadId", upload.UploadID,
                    "error", err,
                )
                continue
            }

            s.logger.Info("aborted incomplete upload",
                "key", upload.ObjectKey,
                "uploadId", upload.UploadID,
                "initiated", upload.Initiated,
            )
            deleted++
        }
    }

    return deleted, nil
}

// CleanupDeletedFiles は削除済みファイルのオブジェクトをクリーンアップします
func (s *CleanupService) CleanupDeletedFiles(ctx context.Context) (int, error) {
    deleted := 0

    // 削除済みファイルを取得
    files, err := s.fileRepo.FindDeletedWithStorageKeys(ctx)
    if err != nil {
        return 0, fmt.Errorf("failed to find deleted files: %w", err)
    }

    for _, file := range files {
        // MinIOオブジェクトを削除
        err := s.client.RemoveObject(ctx, s.bucketName, file.StorageKey, minio.RemoveObjectOptions{})
        if err != nil {
            s.logger.Error("error deleting object for deleted file",
                "fileId", file.ID,
                "storageKey", file.StorageKey,
                "error", err,
            )
            continue
        }

        s.logger.Info("deleted object for deleted file",
            "fileId", file.ID,
            "storageKey", file.StorageKey,
        )
        deleted++
    }

    return deleted, nil
}
```

---

## 9. エラーハンドリング

### 9.1 MinIOエラーの分類

```go
// pkg/apperror/minio.go

package apperror

import (
    "errors"
    "fmt"

    "github.com/minio/minio-go/v7"
)

// MinIO関連エラー
var (
    ErrMinIOConnection     = errors.New("minio connection error")
    ErrMinIOObjectNotFound = errors.New("object not found")
    ErrMinIOAccessDenied   = errors.New("access denied")
    ErrMinIOBucketNotFound = errors.New("bucket not found")
    ErrMinIOUploadFailed   = errors.New("upload failed")
    ErrMinIOInvalidKey     = errors.New("invalid storage key")
)

// WrapMinIOError はMinIOエラーをアプリケーションエラーにラップします
func WrapMinIOError(err error) error {
    if err == nil {
        return nil
    }

    errResponse := minio.ToErrorResponse(err)

    switch errResponse.Code {
    case "NoSuchKey":
        return fmt.Errorf("%w: %v", ErrMinIOObjectNotFound, err)
    case "NoSuchBucket":
        return fmt.Errorf("%w: %v", ErrMinIOBucketNotFound, err)
    case "AccessDenied":
        return fmt.Errorf("%w: %v", ErrMinIOAccessDenied, err)
    default:
        return fmt.Errorf("minio error: %w", err)
    }
}

// IsNotFound はオブジェクトが見つからないエラーかどうかを判定します
func IsNotFound(err error) bool {
    return errors.Is(err, ErrMinIOObjectNotFound)
}
```

---

## 10. 初期化とDI

### 10.1 MinIO依存関係の初期化

```go
// backend/internal/infrastructure/di/minio.go

package di

import (
    "github.com/Hiro-mackay/gc-storage/backend/internal/infrastructure/storage/minio"
)

// MinIOComponents はMinIO関連の依存関係を保持します
type MinIOComponents struct {
    Client          *minio.MinIOClient
    StorageService  *minio.StorageServiceImpl
    PresignedService *minio.PresignedURLService
    MultipartService *minio.MultipartService
    CleanupService   *minio.CleanupService
}

// NewMinIOComponents はMinIO関連の依存関係を初期化します
func NewMinIOComponents(cfg minio.Config, fileRepo repository.FileRepository, logger *slog.Logger) (*MinIOComponents, error) {
    client, err := minio.NewMinIOClient(cfg)
    if err != nil {
        return nil, err
    }

    // バケットセットアップ
    setup := minio.NewBucketSetup(client, cfg.Region)
    if err := setup.Setup(context.Background()); err != nil {
        return nil, fmt.Errorf("failed to setup bucket: %w", err)
    }

    // ライフサイクル設定（未完了アップロードは7日で自動削除）
    lifecycle := minio.NewLifecycleManager(client)
    if err := lifecycle.SetIncompleteUploadExpiration(context.Background(), 7); err != nil {
        logger.Warn("failed to set lifecycle policy", "error", err)
    }

    return &MinIOComponents{
        Client:           client,
        StorageService:   minio.NewStorageService(client),
        PresignedService: minio.NewPresignedURLService(client),
        MultipartService: minio.NewMultipartService(client),
        CleanupService:   minio.NewCleanupService(client, fileRepo, logger),
    }, nil
}
```

---

## 11. テストヘルパー

### 11.1 MinIO Testcontainer

```go
// backend/internal/infrastructure/storage/minio/testhelper/minio.go

package testhelper

import (
    "context"
    "fmt"
    "testing"

    "github.com/testcontainers/testcontainers-go"
    "github.com/testcontainers/testcontainers-go/wait"

    minioinfra "github.com/Hiro-mackay/gc-storage/backend/internal/infrastructure/storage/minio"
)

// MinIOContainer はテスト用MinIOコンテナを管理します
type MinIOContainer struct {
    Container  testcontainers.Container
    Client     *minioinfra.MinIOClient
    Config     minioinfra.Config
    Endpoint   string
}

// NewMinIOContainer はテスト用MinIOコンテナを起動します
func NewMinIOContainer(t *testing.T, bucketName string) *MinIOContainer {
    t.Helper()

    ctx := context.Background()

    accessKey := "minioadmin"
    secretKey := "minioadmin"

    req := testcontainers.ContainerRequest{
        Image:        "minio/minio:latest",
        ExposedPorts: []string{"9000/tcp"},
        Env: map[string]string{
            "MINIO_ROOT_USER":     accessKey,
            "MINIO_ROOT_PASSWORD": secretKey,
        },
        Cmd: []string{"server", "/data"},
        WaitingFor: wait.ForHTTP("/minio/health/live").
            WithPort("9000/tcp"),
    }

    container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
        ContainerRequest: req,
        Started:          true,
    })
    if err != nil {
        t.Fatalf("failed to start minio container: %v", err)
    }

    host, err := container.Host(ctx)
    if err != nil {
        t.Fatalf("failed to get container host: %v", err)
    }

    port, err := container.MappedPort(ctx, "9000")
    if err != nil {
        t.Fatalf("failed to get container port: %v", err)
    }

    endpoint := fmt.Sprintf("%s:%s", host, port.Port())

    cfg := minioinfra.Config{
        Endpoint:        endpoint,
        AccessKeyID:     accessKey,
        SecretAccessKey: secretKey,
        BucketName:      bucketName,
        UseSSL:          false,
        Region:          "us-east-1",
    }

    client, err := minioinfra.NewMinIOClient(cfg)
    if err != nil {
        t.Fatalf("failed to create minio client: %v", err)
    }

    // バケット作成
    setup := minioinfra.NewBucketSetup(client, cfg.Region)
    if err := setup.Setup(ctx); err != nil {
        t.Fatalf("failed to setup bucket: %v", err)
    }

    t.Cleanup(func() {
        container.Terminate(ctx)
    })

    return &MinIOContainer{
        Container: container,
        Client:    client,
        Config:    cfg,
        Endpoint:  endpoint,
    }
}

// ClearBucket はバケット内の全オブジェクトを削除します
func (c *MinIOContainer) ClearBucket(ctx context.Context) error {
    client := c.Client.Client()

    objectsCh := make(chan minio.ObjectInfo)
    go func() {
        defer close(objectsCh)
        for object := range client.ListObjects(ctx, c.Config.BucketName, minio.ListObjectsOptions{Recursive: true}) {
            if object.Err != nil {
                continue
            }
            objectsCh <- object
        }
    }()

    for err := range client.RemoveObjects(ctx, c.Config.BucketName, objectsCh, minio.RemoveObjectsOptions{}) {
        if err.Err != nil {
            return err.Err
        }
    }

    return nil
}
```

### 11.2 ストレージサービステスト例

```go
// backend/internal/infrastructure/storage/minio/storage_test.go

package minio_test

import (
    "bytes"
    "context"
    "testing"

    "github.com/google/uuid"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"

    minioinfra "github.com/Hiro-mackay/gc-storage/backend/internal/infrastructure/storage/minio"
    "github.com/Hiro-mackay/gc-storage/backend/internal/infrastructure/storage/minio/testhelper"
)

func TestStorageService_PutAndGet(t *testing.T) {
    container := testhelper.NewMinIOContainer(t, "test-bucket")
    service := minioinfra.NewStorageService(container.Client)
    ctx := context.Background()

    // テストデータ
    objectKey := minioinfra.NewStorageKey(
        minioinfra.OwnerTypeUser,
        uuid.New(),
        uuid.New(),
        1,
    ).String()
    content := []byte("Hello, MinIO!")

    // アップロード
    err := service.PutObject(ctx, objectKey, bytes.NewReader(content), int64(len(content)), "text/plain")
    require.NoError(t, err)

    // 存在確認
    exists, err := service.ObjectExists(ctx, objectKey)
    require.NoError(t, err)
    assert.True(t, exists)

    // 情報取得
    info, err := service.GetObjectInfo(ctx, objectKey)
    require.NoError(t, err)
    assert.Equal(t, int64(len(content)), info.Size)
    assert.Equal(t, "text/plain", info.ContentType)

    // 削除
    err = service.DeleteObject(ctx, objectKey)
    require.NoError(t, err)

    // 削除確認
    exists, err = service.ObjectExists(ctx, objectKey)
    require.NoError(t, err)
    assert.False(t, exists)
}

func TestStorageService_PresignedURL(t *testing.T) {
    container := testhelper.NewMinIOContainer(t, "test-bucket")
    service := minioinfra.NewStorageService(container.Client)
    ctx := context.Background()

    objectKey := minioinfra.NewStorageKey(
        minioinfra.OwnerTypeUser,
        uuid.New(),
        uuid.New(),
        1,
    ).String()

    // PUT URL生成
    putURL, err := service.GeneratePutURL(ctx, objectKey, minioinfra.PresignedUploadExpiry)
    require.NoError(t, err)
    assert.Contains(t, putURL, objectKey)
    assert.Contains(t, putURL, "X-Amz-Signature")

    // GET URL生成
    getURL, err := service.GenerateGetURL(ctx, objectKey, minioinfra.PresignedDownloadExpiry)
    require.NoError(t, err)
    assert.Contains(t, getURL, objectKey)
    assert.Contains(t, getURL, "X-Amz-Signature")
}

func TestStorageService_MultipartUpload(t *testing.T) {
    container := testhelper.NewMinIOContainer(t, "test-bucket")
    service := minioinfra.NewStorageService(container.Client)
    ctx := context.Background()

    objectKey := minioinfra.NewStorageKey(
        minioinfra.OwnerTypeUser,
        uuid.New(),
        uuid.New(),
        1,
    ).String()

    // マルチパートアップロード開始
    uploadID, err := service.CreateMultipartUpload(ctx, objectKey)
    require.NoError(t, err)
    assert.NotEmpty(t, uploadID)

    // パートURL生成
    partURL, err := service.GeneratePartUploadURL(ctx, objectKey, uploadID, 1)
    require.NoError(t, err)
    assert.Contains(t, partURL, "uploadId")
    assert.Contains(t, partURL, "partNumber")

    // アボート
    err = service.AbortMultipartUpload(ctx, objectKey, uploadID)
    require.NoError(t, err)
}
```

---

## 12. 受け入れ基準

### 12.1 機能要件

| 項目 | 基準 |
|------|------|
| Presigned PUT URL | 有効期限内でアップロードできる |
| Presigned GET URL | 有効期限内でダウンロードできる |
| マルチパート初期化 | uploadIDが取得できる |
| マルチパート完了 | 全パートが結合される |
| マルチパート中止 | アップロード済みパートが削除される |
| オブジェクト存在確認 | 存在有無を正しく判定できる |
| オブジェクト情報取得 | サイズ、Content-Type、ETagが取得できる |
| オブジェクト削除 | 削除後に存在確認でfalseになる |
| オブジェクトコピー | 同一バケット内でコピーできる |

### 12.2 非機能要件

| 項目 | 基準 |
|------|------|
| Presigned URL有効期限 | PUT: 15分, GET: 1時間 |
| 最大ファイルサイズ | 5GB |
| マルチパート閾値 | 100MB以上でマルチパート |
| パートサイズ | 64MB |
| 最大並列アップロード | 5 |
| 未完了アップロード自動削除 | 7日 |

### 12.3 チェックリスト

- [ ] MinIO接続が確立できる
- [ ] バケットが自動作成される
- [ ] Presigned PUT URLでアップロードできる
- [ ] Presigned GET URLでダウンロードできる
- [ ] マルチパートアップロードが完了できる
- [ ] マルチパートアップロードを中止できる
- [ ] オブジェクト情報（サイズ、ETag）が取得できる
- [ ] オブジェクトの削除・コピーができる
- [ ] ライフサイクルポリシーが設定される
- [ ] 孤立オブジェクトのクリーンアップが動作する
- [ ] テストコンテナでのテストが通過する

---

## 13. 関連ドキュメント

- [database.md](./database.md) - PostgreSQL基盤仕様
- [file-upload.md](../features/file-upload.md) - ファイルアップロード仕様
- [folder-management.md](../features/folder-management.md) - フォルダ管理仕様
- [SYSTEM.md](../../02-architecture/SYSTEM.md) - アップロード/ダウンロードフロー
- [file.md](../../03-domains/file.md) - Fileドメイン定義
- [folder.md](../../03-domains/folder.md) - Folderドメイン定義
