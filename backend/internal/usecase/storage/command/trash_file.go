package command

import (
	"context"

	"github.com/google/uuid"

	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/entity"
	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/repository"
	"github.com/Hiro-mackay/gc-storage/backend/pkg/apperror"
)

// TrashFileInput はファイルのゴミ箱移動入力を定義します
type TrashFileInput struct {
	FileID uuid.UUID
	UserID uuid.UUID
}

// TrashFileOutput はファイルのゴミ箱移動出力を定義します
type TrashFileOutput struct {
	ArchivedFileID uuid.UUID
}

// TrashFileCommand はファイルをゴミ箱に移動するコマンドです
type TrashFileCommand struct {
	fileRepo                repository.FileRepository
	fileVersionRepo         repository.FileVersionRepository
	folderRepo              repository.FolderRepository
	folderClosureRepo       repository.FolderClosureRepository
	archivedFileRepo        repository.ArchivedFileRepository
	archivedFileVersionRepo repository.ArchivedFileVersionRepository
	txManager               repository.TransactionManager
}

// NewTrashFileCommand は新しいTrashFileCommandを作成します
func NewTrashFileCommand(
	fileRepo repository.FileRepository,
	fileVersionRepo repository.FileVersionRepository,
	folderRepo repository.FolderRepository,
	folderClosureRepo repository.FolderClosureRepository,
	archivedFileRepo repository.ArchivedFileRepository,
	archivedFileVersionRepo repository.ArchivedFileVersionRepository,
	txManager repository.TransactionManager,
) *TrashFileCommand {
	return &TrashFileCommand{
		fileRepo:                fileRepo,
		fileVersionRepo:         fileVersionRepo,
		folderRepo:              folderRepo,
		folderClosureRepo:       folderClosureRepo,
		archivedFileRepo:        archivedFileRepo,
		archivedFileVersionRepo: archivedFileVersionRepo,
		txManager:               txManager,
	}
}

// Execute はファイルをゴミ箱に移動します
func (c *TrashFileCommand) Execute(ctx context.Context, input TrashFileInput) (*TrashFileOutput, error) {
	// 1. ファイル取得
	file, err := c.fileRepo.FindByID(ctx, input.FileID)
	if err != nil {
		return nil, err
	}

	// 2. 所有者チェック
	if !file.IsOwnedBy(input.UserID) {
		return nil, apperror.NewForbiddenError("not authorized to trash this file")
	}

	// 3. アクティブなファイルのみゴミ箱に移動可能
	if !file.IsActive() {
		return nil, apperror.NewValidationError("only active files can be trashed", nil)
	}

	// 4. 元のパスを取得（復元時の参考用）
	originalPath, err := c.buildFilePath(ctx, file.FolderID, file.Name.String())
	if err != nil {
		return nil, err
	}

	// 5. アーカイブデータを作成
	archivedFile := file.ToArchived(originalPath, input.UserID)

	// 6. バージョン情報を取得
	versions, err := c.fileVersionRepo.FindByFileID(ctx, file.ID)
	if err != nil {
		return nil, err
	}

	// 7. トランザクションで移動処理
	err = c.txManager.WithTransaction(ctx, func(ctx context.Context) error {
		// アーカイブファイル作成
		if err := c.archivedFileRepo.Create(ctx, archivedFile); err != nil {
			return err
		}

		// アーカイブバージョン作成
		if len(versions) > 0 {
			archivedVersions := make([]*entity.ArchivedFileVersion, len(versions))
			for i, v := range versions {
				archivedVersions[i] = v.ToArchived(archivedFile.ID)
			}
			if err := c.archivedFileVersionRepo.BulkCreate(ctx, archivedVersions); err != nil {
				return err
			}
		}

		// 元のバージョン削除
		if err := c.fileVersionRepo.DeleteByFileID(ctx, file.ID); err != nil {
			return err
		}

		// 元のファイル削除
		return c.fileRepo.Delete(ctx, file.ID)
	})

	if err != nil {
		return nil, err
	}

	return &TrashFileOutput{
		ArchivedFileID: archivedFile.ID,
	}, nil
}

// buildFilePath はファイルのフルパスを構築します
// Note: ファイルは必ずフォルダに所属するため、folderIDは必須
func (c *TrashFileCommand) buildFilePath(ctx context.Context, folderID uuid.UUID, fileName string) (string, error) {
	// 祖先フォルダIDを取得（ルートから順）
	ancestorIDs, err := c.folderClosureRepo.FindAncestorIDs(ctx, folderID)
	if err != nil {
		return "", err
	}

	// 自身を含める
	allIDs := append(ancestorIDs, folderID)

	// フォルダ名を取得してパスを構築
	path := ""
	for _, id := range allIDs {
		folder, err := c.folderRepo.FindByID(ctx, id)
		if err != nil {
			return "", err
		}
		path += "/" + folder.Name.String()
	}

	return path + "/" + fileName, nil
}
