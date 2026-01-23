package command

import (
	"context"

	"github.com/google/uuid"

	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/repository"
	"github.com/Hiro-mackay/gc-storage/backend/pkg/apperror"
)

// MoveFileInput はファイル移動の入力を定義します
// Note: ファイルは必ずフォルダに所属するため、NewFolderIDは必須
type MoveFileInput struct {
	FileID      uuid.UUID
	NewFolderID uuid.UUID // 必須 - 移動先フォルダID
	UserID      uuid.UUID
}

// MoveFileOutput はファイル移動の出力を定義します
type MoveFileOutput struct {
	FileID   uuid.UUID
	FolderID uuid.UUID // 必須 - 移動先フォルダID
}

// MoveFileCommand はファイルを別フォルダに移動するコマンドです
type MoveFileCommand struct {
	fileRepo   repository.FileRepository
	folderRepo repository.FolderRepository
}

// NewMoveFileCommand は新しいMoveFileCommandを作成します
func NewMoveFileCommand(
	fileRepo repository.FileRepository,
	folderRepo repository.FolderRepository,
) *MoveFileCommand {
	return &MoveFileCommand{
		fileRepo:   fileRepo,
		folderRepo: folderRepo,
	}
}

// Execute はファイルを別フォルダに移動します
func (c *MoveFileCommand) Execute(ctx context.Context, input MoveFileInput) (*MoveFileOutput, error) {
	// 1. ファイル取得
	file, err := c.fileRepo.FindByID(ctx, input.FileID)
	if err != nil {
		return nil, err
	}

	// 2. 所有者チェック
	if !file.IsOwnedBy(input.UserID) {
		return nil, apperror.NewForbiddenError("not authorized to move this file")
	}

	// 3. 同じフォルダへの移動はスキップ
	if file.FolderID == input.NewFolderID {
		return &MoveFileOutput{
			FileID:   file.ID,
			FolderID: file.FolderID,
		}, nil
	}

	// 4. 移動先フォルダの存在と権限チェック
	folder, err := c.folderRepo.FindByID(ctx, input.NewFolderID)
	if err != nil {
		return nil, err
	}

	// 所有者チェック
	if !folder.IsOwnedBy(file.OwnerID) {
		return nil, apperror.NewForbiddenError("not authorized to move to this folder")
	}

	// 5. 移動先での同名ファイル存在チェック
	exists, err := c.fileRepo.ExistsByNameAndFolder(ctx, file.Name, input.NewFolderID)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, apperror.NewConflictError("file with same name already exists in destination folder")
	}

	// 6. ファイルを移動（エンティティメソッド使用）
	if err := file.MoveTo(input.NewFolderID); err != nil {
		return nil, err
	}

	// 7. 保存
	if err := c.fileRepo.Update(ctx, file); err != nil {
		return nil, err
	}

	return &MoveFileOutput{
		FileID:   file.ID,
		FolderID: file.FolderID,
	}, nil
}
