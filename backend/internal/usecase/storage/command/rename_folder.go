package command

import (
	"context"

	"github.com/google/uuid"

	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/entity"
	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/repository"
	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/valueobject"
	"github.com/Hiro-mackay/gc-storage/backend/pkg/apperror"
)

// RenameFolderInput はフォルダ名変更の入力を定義します
type RenameFolderInput struct {
	FolderID uuid.UUID
	NewName  string
	UserID   uuid.UUID
}

// RenameFolderOutput はフォルダ名変更の出力を定義します
type RenameFolderOutput struct {
	Folder *entity.Folder
}

// RenameFolderCommand はフォルダ名変更コマンドです
type RenameFolderCommand struct {
	folderRepo repository.FolderRepository
}

// NewRenameFolderCommand は新しいRenameFolderCommandを作成します
func NewRenameFolderCommand(folderRepo repository.FolderRepository) *RenameFolderCommand {
	return &RenameFolderCommand{
		folderRepo: folderRepo,
	}
}

// Execute はフォルダ名変更を実行します
func (c *RenameFolderCommand) Execute(ctx context.Context, input RenameFolderInput) (*RenameFolderOutput, error) {
	// 1. 新しいフォルダ名のバリデーション
	newName, err := valueobject.NewFolderName(input.NewName)
	if err != nil {
		return nil, apperror.NewValidationError(err.Error(), nil)
	}

	// 2. フォルダ取得
	folder, err := c.folderRepo.FindByID(ctx, input.FolderID)
	if err != nil {
		return nil, err
	}

	// 3. 所有者チェック
	if !folder.IsOwnedBy(input.UserID) {
		return nil, apperror.NewForbiddenError("not authorized to rename this folder")
	}

	// 4. 同名フォルダの存在チェック（同じ名前への変更は許可）
	if !folder.EqualsName(newName) {
		var exists bool
		if folder.ParentID != nil {
			exists, err = c.folderRepo.ExistsByNameAndParent(ctx, newName, folder.ParentID, folder.OwnerID)
		} else {
			exists, err = c.folderRepo.ExistsByNameAndOwnerRoot(ctx, newName, folder.OwnerID)
		}
		if err != nil {
			return nil, err
		}
		if exists {
			return nil, apperror.NewConflictError("folder with same name already exists")
		}
	}

	// 5. フォルダ名変更（エンティティメソッド）
	folder.Rename(newName)

	// 6. 永続化
	if err := c.folderRepo.Update(ctx, folder); err != nil {
		return nil, err
	}

	return &RenameFolderOutput{Folder: folder}, nil
}
