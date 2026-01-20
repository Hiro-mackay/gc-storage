package query

import (
	"context"

	"github.com/google/uuid"

	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/entity"
	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/repository"
	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/valueobject"
	"github.com/Hiro-mackay/gc-storage/backend/pkg/apperror"
)

// GetAncestorsInput は祖先フォルダ取得の入力を定義します
type GetAncestorsInput struct {
	FolderID uuid.UUID
	UserID   uuid.UUID
}

// GetAncestorsOutput は祖先フォルダ取得の出力を定義します
type GetAncestorsOutput struct {
	Ancestors []*entity.Folder // ルートから順に並ぶ
}

// GetAncestorsQuery は祖先フォルダ取得クエリです（パンくずリスト用）
type GetAncestorsQuery struct {
	folderRepo        repository.FolderRepository
	folderClosureRepo repository.FolderClosureRepository
}

// NewGetAncestorsQuery は新しいGetAncestorsQueryを作成します
func NewGetAncestorsQuery(
	folderRepo repository.FolderRepository,
	folderClosureRepo repository.FolderClosureRepository,
) *GetAncestorsQuery {
	return &GetAncestorsQuery{
		folderRepo:        folderRepo,
		folderClosureRepo: folderClosureRepo,
	}
}

// Execute は祖先フォルダ一覧を取得します
func (q *GetAncestorsQuery) Execute(ctx context.Context, input GetAncestorsInput) (*GetAncestorsOutput, error) {
	// 1. フォルダ取得して権限チェック
	folder, err := q.folderRepo.FindByID(ctx, input.FolderID)
	if err != nil {
		return nil, err
	}

	// 所有者チェック（ユーザー所有の場合のみ）
	if folder.OwnerType == valueobject.OwnerTypeUser && folder.OwnerID != input.UserID {
		return nil, apperror.NewForbiddenError("not authorized to access this folder")
	}

	// 2. 祖先フォルダIDを取得（閉包テーブルから）
	ancestorIDs, err := q.folderClosureRepo.FindAncestorIDs(ctx, input.FolderID)
	if err != nil {
		return nil, err
	}

	// 祖先がない場合（ルートフォルダの場合）
	if len(ancestorIDs) == 0 {
		return &GetAncestorsOutput{Ancestors: []*entity.Folder{}}, nil
	}

	// 3. 祖先フォルダを取得
	ancestors := make([]*entity.Folder, len(ancestorIDs))
	for i, id := range ancestorIDs {
		ancestor, err := q.folderRepo.FindByID(ctx, id)
		if err != nil {
			return nil, err
		}
		ancestors[i] = ancestor
	}

	// 4. 深さ順にソート（浅い順 = ルートから）
	// FindAncestorIDsは既にpath_length順で取得されるが、
	// 深さの浅い順（ルートから）になるよう確認
	// 閉包テーブルの祖先IDはpath_length昇順なので、深さの深い順になっている
	// 逆順にしてルートからの順にする
	for i, j := 0, len(ancestors)-1; i < j; i, j = i+1, j-1 {
		ancestors[i], ancestors[j] = ancestors[j], ancestors[i]
	}

	return &GetAncestorsOutput{Ancestors: ancestors}, nil
}
