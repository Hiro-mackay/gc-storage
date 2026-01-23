package command

import (
	"context"

	"github.com/google/uuid"

	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/repository"
	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/valueobject"
	"github.com/Hiro-mackay/gc-storage/backend/pkg/apperror"
)

// RenameFileInput はファイル名変更の入力を定義します
type RenameFileInput struct {
	FileID  uuid.UUID
	NewName string
	UserID  uuid.UUID
}

// RenameFileOutput はファイル名変更の出力を定義します
type RenameFileOutput struct {
	FileID uuid.UUID
	Name   string
}

// RenameFileCommand はファイル名を変更するコマンドです
type RenameFileCommand struct {
	fileRepo repository.FileRepository
}

// NewRenameFileCommand は新しいRenameFileCommandを作成します
func NewRenameFileCommand(fileRepo repository.FileRepository) *RenameFileCommand {
	return &RenameFileCommand{
		fileRepo: fileRepo,
	}
}

// Execute はファイル名を変更します
func (c *RenameFileCommand) Execute(ctx context.Context, input RenameFileInput) (*RenameFileOutput, error) {
	// 1. ファイル名のバリデーション
	newName, err := valueobject.NewFileName(input.NewName)
	if err != nil {
		return nil, apperror.NewValidationError(err.Error(), nil)
	}

	// 2. ファイル取得
	file, err := c.fileRepo.FindByID(ctx, input.FileID)
	if err != nil {
		return nil, err
	}

	// 3. 所有者チェック
	if !file.IsOwnedBy(input.UserID) {
		return nil, apperror.NewForbiddenError("not authorized to rename this file")
	}

	// 4. 同名ファイルの存在チェック
	exists, err := c.fileRepo.ExistsByNameAndFolder(ctx, newName, file.FolderID)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, apperror.NewConflictError("file with same name already exists")
	}

	// 5. ファイル名を変更（エンティティメソッド使用）
	if err := file.Rename(newName); err != nil {
		return nil, apperror.NewValidationError(err.Error(), nil)
	}

	// 6. 保存
	if err := c.fileRepo.Update(ctx, file); err != nil {
		return nil, err
	}

	return &RenameFileOutput{
		FileID: file.ID,
		Name:   file.Name.String(),
	}, nil
}
