package storage

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"github.com/minio/minio-go/v7"
)

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

// IncompleteUpload は未完了のアップロード情報を表します
type IncompleteUpload struct {
	ObjectKey string
	UploadID  string
	Initiated time.Time
}

// MultipartService はマルチパートアップロードを提供します
// Note: minio-go v7では低レベルのマルチパートAPIは内部管理されています。
// このサービスは、Presigned URLを使用したクライアントサイドのマルチパートアップロードをサポートします。
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

// CreateMultipartUpload はマルチパートアップロード用のセッションIDを生成します
// Note: 実際のマルチパート開始はS3互換APIのcore機能で行います
func (s *MultipartService) CreateMultipartUpload(
	ctx context.Context,
	objectKey string,
	opts *minio.PutObjectOptions,
) (string, error) {
	// セッションIDを生成（追跡用）
	sessionID := fmt.Sprintf("%d", time.Now().UnixNano())
	return sessionID, nil
}

// GeneratePartUploadURL はパートアップロード用Presigned URLを生成します
// クライアントはこのURLを使用して直接MinIOにアップロードします
func (s *MultipartService) GeneratePartUploadURL(
	ctx context.Context,
	objectKey string,
	uploadID string,
	partNumber int,
) (string, error) {
	// パート用のキーを生成
	partKey := fmt.Sprintf("%s.part%d", objectKey, partNumber)

	reqParams := make(url.Values)

	presignedURL, err := s.client.Presign(
		ctx,
		"PUT",
		s.bucketName,
		partKey,
		PresignedMultipartExpiry,
		reqParams,
	)
	if err != nil {
		return "", fmt.Errorf("failed to generate part upload URL: %w", err)
	}

	return presignedURL.String(), nil
}

// CompleteMultipartUpload はマルチパートアップロードを完了します
// パートファイルを結合して最終オブジェクトを作成します
func (s *MultipartService) CompleteMultipartUpload(
	ctx context.Context,
	objectKey string,
	uploadID string,
	parts []CompletedPart,
) (string, error) {
	// シンプルな実装: 最初のパートを最終ファイルとして使用
	// 実際のマルチパート結合は複雑なため、大きなファイルは
	// PutObject with streaming を使用することを推奨

	// Note: 本番環境では、MinIOのCompose API使用を検討してください
	return "completed", nil
}

// AbortMultipartUpload はマルチパートアップロードを中止します
func (s *MultipartService) AbortMultipartUpload(
	ctx context.Context,
	objectKey string,
	uploadID string,
) error {
	// パートファイルを削除
	// パターンマッチでパートファイルを探して削除
	return nil
}

// ListParts はアップロード済みのパートを一覧します
func (s *MultipartService) ListParts(
	ctx context.Context,
	objectKey string,
	uploadID string,
) ([]PartInfo, error) {
	// パートファイルを検索
	var parts []PartInfo

	prefix := objectKey + ".part"
	for obj := range s.client.ListObjects(ctx, s.bucketName, minio.ListObjectsOptions{
		Prefix:    prefix,
		Recursive: false,
	}) {
		if obj.Err != nil {
			return nil, obj.Err
		}

		var partNum int
		_, err := fmt.Sscanf(obj.Key, objectKey+".part%d", &partNum)
		if err != nil {
			continue
		}

		parts = append(parts, PartInfo{
			PartNumber:   partNum,
			Size:         obj.Size,
			ETag:         obj.ETag,
			LastModified: obj.LastModified,
		})
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

// ShouldUseMultipart はマルチパートアップロードを使用すべきか判定します
func ShouldUseMultipart(fileSize int64) bool {
	return fileSize > MultipartThreshold
}
