package query

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/entity"
	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/repository"
	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/service"
	"github.com/Hiro-mackay/gc-storage/backend/pkg/apperror"
)

// ダウンロードURL有効期限
const DownloadURLExpiry = 1 * time.Hour

// GetDownloadURLInput はダウンロードURL取得の入力を定義します
type GetDownloadURLInput struct {
	FileID        uuid.UUID
	VersionNumber *int // nilの場合は最新バージョン
	UserID        uuid.UUID
}

// GetDownloadURLOutput はダウンロードURL取得の出力を定義します
type GetDownloadURLOutput struct {
	FileID        uuid.UUID
	FileName      string
	MimeType      string
	Size          int64
	VersionNumber int
	DownloadURL   string
	ExpiresAt     time.Time
}

// GetDownloadURLQuery はダウンロードURL取得クエリです
type GetDownloadURLQuery struct {
	fileRepo        repository.FileRepository
	fileVersionRepo repository.FileVersionRepository
	storageService  service.StorageService
}

// NewGetDownloadURLQuery は新しいGetDownloadURLQueryを作成します
func NewGetDownloadURLQuery(
	fileRepo repository.FileRepository,
	fileVersionRepo repository.FileVersionRepository,
	storageService service.StorageService,
) *GetDownloadURLQuery {
	return &GetDownloadURLQuery{
		fileRepo:        fileRepo,
		fileVersionRepo: fileVersionRepo,
		storageService:  storageService,
	}
}

// Execute はダウンロードURLを取得します
func (q *GetDownloadURLQuery) Execute(ctx context.Context, input GetDownloadURLInput) (*GetDownloadURLOutput, error) {
	// 1. ファイル取得
	file, err := q.fileRepo.FindByID(ctx, input.FileID)
	if err != nil {
		return nil, err
	}

	// 2. 所有者チェック
	if !file.IsOwnedBy(input.UserID) {
		return nil, apperror.NewForbiddenError("not authorized to download this file")
	}

	// 3. ダウンロード可能かチェック
	if !file.CanDownload() {
		return nil, apperror.NewValidationError("file cannot be downloaded in current state", nil)
	}

	// 4. バージョン取得
	var version *entity.FileVersion
	if input.VersionNumber != nil {
		version, err = q.fileVersionRepo.FindByFileAndVersion(ctx, file.ID, *input.VersionNumber)
	} else {
		version, err = q.fileVersionRepo.FindLatestByFileID(ctx, file.ID)
	}
	if err != nil {
		return nil, err
	}

	// 5. Presigned URL生成
	presigned, err := q.storageService.GenerateGetURL(ctx, file.StorageKey.String(), DownloadURLExpiry)
	if err != nil {
		return nil, apperror.NewInternalError(err)
	}

	return &GetDownloadURLOutput{
		FileID:        file.ID,
		FileName:      file.Name.String(),
		MimeType:      file.MimeType.String(),
		Size:          version.Size,
		VersionNumber: version.VersionNumber,
		DownloadURL:   presigned.URL,
		ExpiresAt:     presigned.ExpiresAt,
	}, nil
}
