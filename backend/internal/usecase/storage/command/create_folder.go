package command

import (
	"context"

	"github.com/google/uuid"

	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/entity"
	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/repository"
	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/valueobject"
	"github.com/Hiro-mackay/gc-storage/backend/pkg/apperror"
)

// CreateFolderInput はフォルダ作成の入力を定義します
type CreateFolderInput struct {
	Name      string
	ParentID  *uuid.UUID
	OwnerID   uuid.UUID
	OwnerType valueobject.OwnerType
}

// CreateFolderOutput はフォルダ作成の出力を定義します
type CreateFolderOutput struct {
	Folder *entity.Folder
}

// CreateFolderCommand はフォルダ作成コマンドです
type CreateFolderCommand struct {
	folderRepo        repository.FolderRepository
	folderClosureRepo repository.FolderClosureRepository
	txManager         repository.TransactionManager
}

// NewCreateFolderCommand は新しいCreateFolderCommandを作成します
func NewCreateFolderCommand(
	folderRepo repository.FolderRepository,
	folderClosureRepo repository.FolderClosureRepository,
	txManager repository.TransactionManager,
) *CreateFolderCommand {
	return &CreateFolderCommand{
		folderRepo:        folderRepo,
		folderClosureRepo: folderClosureRepo,
		txManager:         txManager,
	}
}

// Execute はフォルダ作成を実行します
func (c *CreateFolderCommand) Execute(ctx context.Context, input CreateFolderInput) (*CreateFolderOutput, error) {
	// 1. フォルダ名のバリデーション
	folderName, err := valueobject.NewFolderName(input.Name)
	if err != nil {
		return nil, apperror.NewValidationError(err.Error(), nil)
	}

	// 2. 親フォルダの取得と深さの計算
	depth := 0 // ルートレベルの場合
	if input.ParentID != nil {
		parent, err := c.folderRepo.FindByID(ctx, *input.ParentID)
		if err != nil {
			return nil, err
		}

		// 親フォルダの所有者チェック
		if parent.OwnerID != input.OwnerID || parent.OwnerType != input.OwnerType {
			return nil, apperror.NewForbiddenError("not authorized to create folder in this location")
		}

		depth = parent.Depth + 1
	}

	// 3. 同名フォルダの存在チェック
	var exists bool
	if input.ParentID != nil {
		exists, err = c.folderRepo.ExistsByNameAndParent(ctx, folderName, input.ParentID, input.OwnerID, input.OwnerType)
	} else {
		exists, err = c.folderRepo.ExistsByNameAndOwnerRoot(ctx, folderName, input.OwnerID, input.OwnerType)
	}
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, apperror.NewConflictError("folder with same name already exists")
	}

	// 4. フォルダエンティティの作成
	folder, err := entity.NewFolder(folderName, input.ParentID, input.OwnerID, input.OwnerType, depth)
	if err != nil {
		return nil, apperror.NewValidationError(err.Error(), nil)
	}

	// 5. トランザクションでフォルダと閉包テーブルエントリを作成
	err = c.txManager.WithTransaction(ctx, func(ctx context.Context) error {
		// フォルダ作成
		if err := c.folderRepo.Create(ctx, folder); err != nil {
			return err
		}

		// 自己参照パスの挿入
		if err := c.folderClosureRepo.InsertSelfReference(ctx, folder.ID); err != nil {
			return err
		}

		// 祖先パスの挿入（親がある場合）
		if input.ParentID != nil {
			ancestorPaths, err := c.folderClosureRepo.FindAncestorPaths(ctx, *input.ParentID)
			if err != nil {
				return err
			}

			// 新しいフォルダ用のパスエントリを作成
			newPaths := make([]*entity.FolderPath, len(ancestorPaths)+1)
			// 親フォルダからのパス（path_length = 1）
			newPaths[0] = entity.NewFolderPath(*input.ParentID, folder.ID, 1)

			// 祖先からのパス（path_length + 1）
			for i, path := range ancestorPaths {
				newPaths[i+1] = entity.NewFolderPath(path.AncestorID, folder.ID, path.PathLength+1)
			}

			if err := c.folderClosureRepo.InsertAncestorPaths(ctx, newPaths); err != nil {
				return err
			}
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return &CreateFolderOutput{Folder: folder}, nil
}
