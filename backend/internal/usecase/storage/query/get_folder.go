package query

import (
	"context"

	"github.com/google/uuid"

	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/entity"
	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/repository"
	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/valueobject"
	"github.com/Hiro-mackay/gc-storage/backend/pkg/apperror"
)

// GetFolderInput はフォルダ取得の入力を定義します
type GetFolderInput struct {
	FolderID uuid.UUID
	UserID   uuid.UUID
}

// GetFolderOutput はフォルダ取得の出力を定義します
type GetFolderOutput struct {
	Folder *entity.Folder
}

// GetFolderQuery はフォルダ取得クエリです
type GetFolderQuery struct {
	folderRepo repository.FolderRepository
}

// NewGetFolderQuery は新しいGetFolderQueryを作成します
func NewGetFolderQuery(folderRepo repository.FolderRepository) *GetFolderQuery {
	return &GetFolderQuery{
		folderRepo: folderRepo,
	}
}

// Execute はフォルダ取得を実行します
func (q *GetFolderQuery) Execute(ctx context.Context, input GetFolderInput) (*GetFolderOutput, error) {
	// 1. フォルダ取得
	folder, err := q.folderRepo.FindByID(ctx, input.FolderID)
	if err != nil {
		return nil, err
	}

	// 2. 所有者チェック（ユーザー所有の場合のみ）
	if folder.OwnerType == valueobject.OwnerTypeUser && folder.OwnerID != input.UserID {
		return nil, apperror.NewForbiddenError("not authorized to access this folder")
	}

	return &GetFolderOutput{Folder: folder}, nil
}
