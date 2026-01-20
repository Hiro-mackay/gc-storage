package query

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/repository"
	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/valueobject"
	"github.com/Hiro-mackay/gc-storage/backend/pkg/apperror"
)

// ListFileVersionsInput はファイルバージョン一覧の入力を定義します
type ListFileVersionsInput struct {
	FileID uuid.UUID
	UserID uuid.UUID
}

// FileVersionInfo はファイルバージョン情報を定義します
type FileVersionInfo struct {
	ID            uuid.UUID
	VersionNumber int
	Size          int64
	Checksum      string
	UploadedBy    uuid.UUID
	CreatedAt     time.Time
	IsLatest      bool
}

// ListFileVersionsOutput はファイルバージョン一覧の出力を定義します
type ListFileVersionsOutput struct {
	FileID   uuid.UUID
	FileName string
	Versions []FileVersionInfo
}

// ListFileVersionsQuery はファイルバージョン一覧クエリです
type ListFileVersionsQuery struct {
	fileRepo        repository.FileRepository
	fileVersionRepo repository.FileVersionRepository
}

// NewListFileVersionsQuery は新しいListFileVersionsQueryを作成します
func NewListFileVersionsQuery(
	fileRepo repository.FileRepository,
	fileVersionRepo repository.FileVersionRepository,
) *ListFileVersionsQuery {
	return &ListFileVersionsQuery{
		fileRepo:        fileRepo,
		fileVersionRepo: fileVersionRepo,
	}
}

// Execute はファイルバージョン一覧を取得します
func (q *ListFileVersionsQuery) Execute(ctx context.Context, input ListFileVersionsInput) (*ListFileVersionsOutput, error) {
	// 1. ファイル取得
	file, err := q.fileRepo.FindByID(ctx, input.FileID)
	if err != nil {
		return nil, err
	}

	// 2. 権限チェック（ユーザー所有の場合のみ）
	if file.OwnerType == valueobject.OwnerTypeUser && file.OwnerID != input.UserID {
		return nil, apperror.NewForbiddenError("not authorized to view file versions")
	}

	// 3. バージョン一覧を取得
	versions, err := q.fileVersionRepo.FindByFileID(ctx, file.ID)
	if err != nil {
		return nil, err
	}

	// 4. 出力形式に変換
	versionInfos := make([]FileVersionInfo, len(versions))
	for i, v := range versions {
		versionInfos[i] = FileVersionInfo{
			ID:            v.ID,
			VersionNumber: v.VersionNumber,
			Size:          v.Size,
			Checksum:      v.Checksum,
			UploadedBy:    v.UploadedBy,
			CreatedAt:     v.CreatedAt,
			IsLatest:      v.VersionNumber == file.CurrentVersion,
		}
	}

	return &ListFileVersionsOutput{
		FileID:   file.ID,
		FileName: file.Name.String(),
		Versions: versionInfos,
	}, nil
}
