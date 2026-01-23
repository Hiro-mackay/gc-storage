package query

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/repository"
	"github.com/Hiro-mackay/gc-storage/backend/pkg/apperror"
)

// ページネーションのデフォルト値
const (
	DefaultTrashLimit = 50
	MaxTrashLimit     = 100
)

// ListTrashInput はゴミ箱一覧の入力を定義します
type ListTrashInput struct {
	OwnerID uuid.UUID
	UserID  uuid.UUID
	Limit   int        // 取得件数（デフォルト: 50, 最大: 100）
	Cursor  *uuid.UUID // ページネーションカーソル（前回の最後のID）
}

// TrashItem はゴミ箱内のアイテム情報を定義します
type TrashItem struct {
	ID               uuid.UUID
	OriginalFileID   uuid.UUID
	OriginalFolderID uuid.UUID // 必須 - 復元先フォルダID
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
	Items      []TrashItem
	NextCursor *uuid.UUID // 次ページのカーソル（最後のアイテムのID）
	HasMore    bool       // 次ページが存在するか
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
	// 1. 所有者チェック
	if input.OwnerID != input.UserID {
		return nil, apperror.NewForbiddenError("not authorized to view this trash")
	}

	// 2. Limit の正規化
	limit := input.Limit
	if limit <= 0 {
		limit = DefaultTrashLimit
	}
	if limit > MaxTrashLimit {
		limit = MaxTrashLimit
	}

	// 3. アーカイブファイル一覧を取得（1件多く取得して次ページの存在を確認）
	archivedFiles, err := q.archivedFileRepo.FindByOwnerWithPagination(ctx, input.OwnerID, limit+1, input.Cursor)
	if err != nil {
		return nil, err
	}

	// 4. 次ページの存在確認
	hasMore := len(archivedFiles) > limit
	if hasMore {
		archivedFiles = archivedFiles[:limit]
	}

	// 5. 出力形式に変換
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

	// 6. 次ページのカーソルを設定
	var nextCursor *uuid.UUID
	if hasMore && len(items) > 0 {
		lastID := items[len(items)-1].ID
		nextCursor = &lastID
	}

	return &ListTrashOutput{
		Items:      items,
		NextCursor: nextCursor,
		HasMore:    hasMore,
	}, nil
}
