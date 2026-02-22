package command

import (
	"context"

	"github.com/google/uuid"

	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/authz"
	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/entity"
	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/repository"
	"github.com/Hiro-mackay/gc-storage/backend/pkg/apperror"
)

// MoveFolderInput はフォルダ移動の入力を定義します
type MoveFolderInput struct {
	FolderID    uuid.UUID
	NewParentID *uuid.UUID // nil の場合はルートへ移動
	UserID      uuid.UUID
}

// MoveFolderOutput はフォルダ移動の出力を定義します
type MoveFolderOutput struct {
	Folder *entity.Folder
}

// MoveFolderCommand はフォルダ移動コマンドです
type MoveFolderCommand struct {
	folderRepo         repository.FolderRepository
	folderClosureRepo  repository.FolderClosureRepository
	txManager          repository.TransactionManager
	userRepo           repository.UserRepository
	permissionResolver authz.PermissionResolver
}

// NewMoveFolderCommand は新しいMoveFolderCommandを作成します
func NewMoveFolderCommand(
	folderRepo repository.FolderRepository,
	folderClosureRepo repository.FolderClosureRepository,
	txManager repository.TransactionManager,
	userRepo repository.UserRepository,
	permissionResolver authz.PermissionResolver,
) *MoveFolderCommand {
	return &MoveFolderCommand{
		folderRepo:         folderRepo,
		folderClosureRepo:  folderClosureRepo,
		txManager:          txManager,
		userRepo:           userRepo,
		permissionResolver: permissionResolver,
	}
}

// Execute はフォルダ移動を実行します
func (c *MoveFolderCommand) Execute(ctx context.Context, input MoveFolderInput) (*MoveFolderOutput, error) {
	// 1. フォルダ取得
	folder, err := c.folderRepo.FindByID(ctx, input.FolderID)
	if err != nil {
		return nil, err
	}

	// 2. 移動元の権限チェック (AC-50: folder:move_out)
	if folder.ParentID != nil {
		hasMoveOut, err := c.permissionResolver.HasPermission(ctx, input.UserID, authz.ResourceTypeFolder, *folder.ParentID, authz.PermFolderMoveOut)
		if err != nil {
			return nil, err
		}
		if !hasMoveOut {
			return nil, apperror.NewForbiddenError("not authorized to move this folder")
		}
	} else {
		// ルートレベルのフォルダは所有者のみ移動可能
		if !folder.IsOwnedBy(input.UserID) {
			return nil, apperror.NewForbiddenError("not authorized to move this folder")
		}
	}

	// 3. パーソナルフォルダチェック (R-FD009)
	user, err := c.userRepo.FindByID(ctx, input.UserID)
	if err != nil {
		return nil, err
	}
	if user.PersonalFolderID != nil && *user.PersonalFolderID == input.FolderID {
		return nil, apperror.NewForbiddenError("personal folder cannot be moved")
	}

	// 4. 同じ場所への移動は何もしない
	if (folder.ParentID == nil && input.NewParentID == nil) ||
		(folder.ParentID != nil && input.NewParentID != nil && *folder.ParentID == *input.NewParentID) {
		return &MoveFolderOutput{Folder: folder}, nil
	}

	// 5. 移動先の検証 (AC-51: folder:move_in)
	var newParent *entity.Folder
	newDepth := 0
	if input.NewParentID != nil {
		newParent, err = c.folderRepo.FindByID(ctx, *input.NewParentID)
		if err != nil {
			return nil, err
		}

		// 移動先フォルダに対するfolder:move_in権限チェック
		hasMoveIn, err := c.permissionResolver.HasPermission(ctx, input.UserID, authz.ResourceTypeFolder, *input.NewParentID, authz.PermFolderMoveIn)
		if err != nil {
			return nil, err
		}
		if !hasMoveIn {
			return nil, apperror.NewForbiddenError("not authorized to move to this folder")
		}

		newDepth = newParent.Depth + 1
	} else {
		// ルートへの移動は所有者のみ
		if !folder.IsOwnedBy(input.UserID) {
			return nil, apperror.NewForbiddenError("not authorized to move to root")
		}
	}

	// 6. 子孫フォルダIDを取得（循環参照チェック用）
	descendantIDs, err := c.folderClosureRepo.FindDescendantIDs(ctx, folder.ID)
	if err != nil {
		return nil, err
	}

	// 7. 移動可能性のバリデーション（エンティティメソッド）
	// 子孫の最大深さを計算
	maxDescendantDepth := 0
	if len(descendantIDs) > 0 {
		descendantsWithDepth, err := c.folderClosureRepo.FindDescendantsWithDepth(ctx, folder.ID)
		if err != nil {
			return nil, err
		}
		for _, d := range descendantsWithDepth {
			if d > maxDescendantDepth {
				maxDescendantDepth = d
			}
		}
	}

	// 深さ制限チェック（新しい深さ + 子孫の相対深さ）
	if err := folder.CanMoveTo(input.NewParentID, newDepth+maxDescendantDepth, descendantIDs); err != nil {
		return nil, apperror.NewValidationError(err.Error(), nil)
	}

	// 8. 同名フォルダの存在チェック
	var exists bool
	if input.NewParentID != nil {
		exists, err = c.folderRepo.ExistsByNameAndParent(ctx, folder.Name, input.NewParentID, folder.OwnerID)
	} else {
		exists, err = c.folderRepo.ExistsByNameAndOwnerRoot(ctx, folder.Name, folder.OwnerID)
	}
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, apperror.NewConflictError("folder with same name already exists in destination")
	}

	// 9. トランザクションでフォルダと閉包テーブルを更新
	err = c.txManager.WithTransaction(ctx, func(ctx context.Context) error {
		// 新しい親のパスを取得
		var newParentPaths []*entity.FolderPath
		if input.NewParentID != nil {
			newParentPaths, err = c.folderClosureRepo.FindAncestorPaths(ctx, *input.NewParentID)
			if err != nil {
				return err
			}
		}

		// 閉包テーブルを更新（MoveSubtreeが全ての処理を行う）
		if err := c.folderClosureRepo.MoveSubtree(ctx, folder.ID, newParentPaths); err != nil {
			return err
		}

		// フォルダを更新（エンティティメソッド）
		folder.MoveTo(input.NewParentID, newDepth)
		if err := c.folderRepo.Update(ctx, folder); err != nil {
			return err
		}

		// 子孫フォルダの深さを更新
		if len(descendantIDs) > 0 {
			descendantsWithDepth, err := c.folderClosureRepo.FindDescendantsWithDepth(ctx, folder.ID)
			if err != nil {
				return err
			}

			folderDepths := make(map[uuid.UUID]int)
			for descendantID, relativeDepth := range descendantsWithDepth {
				folderDepths[descendantID] = newDepth + relativeDepth
			}

			if err := c.folderRepo.BulkUpdateDepth(ctx, folderDepths); err != nil {
				return err
			}
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return &MoveFolderOutput{Folder: folder}, nil
}
