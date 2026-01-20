package query

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/repository"
	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/valueobject"
	"github.com/Hiro-mackay/gc-storage/backend/pkg/apperror"
)

// ListTrashInput はゴミ箱一覧の入力を定義します
type ListTrashInput struct {
	OwnerID   uuid.UUID
	OwnerType valueobject.OwnerType
	UserID    uuid.UUID
}

// TrashItem はゴミ箱内のアイテム情報を定義します
type TrashItem struct {
	ID               uuid.UUID
	OriginalFileID   uuid.UUID
	OriginalFolderID *uuid.UUID
	OriginalPath     string
	Name             string
	MimeType         string
	Size             int64
	ArchivedAt       time.Time
	ExpiresAt        time.Time
	DaysUntilExpiry  int
}

// ListTrashOutput はゴミ箱一覧の出力を定義します
type ListTrashOutput struct {
	Items []TrashItem
}

// ListTrashQuery はゴミ箱一覧クエリです
type ListTrashQuery struct {
	archivedFileRepo repository.ArchivedFileRepository
}

// NewListTrashQuery は新しいListTrashQueryを作成します
func NewListTrashQuery(archivedFileRepo repository.ArchivedFileRepository) *ListTrashQuery {
	return &ListTrashQuery{
		archivedFileRepo: archivedFileRepo,
	}
}

// Execute はゴミ箱一覧を取得します
func (q *ListTrashQuery) Execute(ctx context.Context, input ListTrashInput) (*ListTrashOutput, error) {
	// 1. 権限チェック（ユーザー所有の場合は自身のみ）
	if input.OwnerType == valueobject.OwnerTypeUser && input.OwnerID != input.UserID {
		return nil, apperror.NewForbiddenError("not authorized to view this trash")
	}

	// 2. アーカイブファイル一覧を取得
	archivedFiles, err := q.archivedFileRepo.FindByOwner(ctx, input.OwnerID, input.OwnerType)
	if err != nil {
		return nil, err
	}

	// 3. 出力形式に変換
	items := make([]TrashItem, len(archivedFiles))
	for i, af := range archivedFiles {
		items[i] = TrashItem{
			ID:               af.ID,
			OriginalFileID:   af.OriginalFileID,
			OriginalFolderID: af.OriginalFolderID,
			OriginalPath:     af.OriginalPath,
			Name:             af.Name.String(),
			MimeType:         af.MimeType.String(),
			Size:             af.Size,
			ArchivedAt:       af.ArchivedAt,
			ExpiresAt:        af.ExpiresAt,
			DaysUntilExpiry:  af.DaysUntilExpiration(),
		}
	}

	return &ListTrashOutput{Items: items}, nil
}
