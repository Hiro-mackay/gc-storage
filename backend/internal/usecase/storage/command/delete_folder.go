package command

import (
	"context"

	"github.com/google/uuid"

	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/entity"
	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/repository"
	"github.com/Hiro-mackay/gc-storage/backend/pkg/apperror"
)

// DeleteFolderInput はフォルダ削除の入力を定義します
type DeleteFolderInput struct {
	FolderID uuid.UUID
	UserID   uuid.UUID
}

// DeleteFolderOutput はフォルダ削除の出力を定義します
type DeleteFolderOutput struct {
	DeletedFolderCount int
	ArchivedFileCount  int
}

// DeleteFolderCommand はフォルダ削除コマンドです
type DeleteFolderCommand struct {
	folderRepo              repository.FolderRepository
	folderClosureRepo       repository.FolderClosureRepository
	fileRepo                repository.FileRepository
	fileVersionRepo         repository.FileVersionRepository
	archivedFileRepo        repository.ArchivedFileRepository
	archivedFileVersionRepo repository.ArchivedFileVersionRepository
	txManager               repository.TransactionManager
}

// NewDeleteFolderCommand は新しいDeleteFolderCommandを作成します
func NewDeleteFolderCommand(
	folderRepo repository.FolderRepository,
	folderClosureRepo repository.FolderClosureRepository,
	fileRepo repository.FileRepository,
	fileVersionRepo repository.FileVersionRepository,
	archivedFileRepo repository.ArchivedFileRepository,
	archivedFileVersionRepo repository.ArchivedFileVersionRepository,
	txManager repository.TransactionManager,
) *DeleteFolderCommand {
	return &DeleteFolderCommand{
		folderRepo:              folderRepo,
		folderClosureRepo:       folderClosureRepo,
		fileRepo:                fileRepo,
		fileVersionRepo:         fileVersionRepo,
		archivedFileRepo:        archivedFileRepo,
		archivedFileVersionRepo: archivedFileVersionRepo,
		txManager:               txManager,
	}
}

// Execute はフォルダ削除を実行します
func (c *DeleteFolderCommand) Execute(ctx context.Context, input DeleteFolderInput) (*DeleteFolderOutput, error) {
	// 1. フォルダ取得
	folder, err := c.folderRepo.FindByID(ctx, input.FolderID)
	if err != nil {
		return nil, err
	}

	// 2. 所有者チェック
	if !folder.IsOwnedBy(input.UserID) {
		return nil, apperror.NewForbiddenError("not authorized to delete this folder")
	}

	// 3. 削除対象のフォルダID一覧を取得（自身 + 子孫）
	descendantIDs, err := c.folderClosureRepo.FindDescendantIDs(ctx, folder.ID)
	if err != nil {
		return nil, err
	}
	folderIDs := append([]uuid.UUID{folder.ID}, descendantIDs...)

	// 4. 削除対象のファイル取得
	files, err := c.fileRepo.FindByFolderIDs(ctx, folderIDs)
	if err != nil {
		return nil, err
	}

	// 5. トランザクションで削除処理を実行
	archivedFileCount := 0
	err = c.txManager.WithTransaction(ctx, func(ctx context.Context) error {
		// 5a. ファイルをアーカイブ（ゴミ箱へ移動）
		for _, file := range files {
			if !file.IsActive() {
				continue
			}

			// ファイルのバージョン取得
			versions, err := c.fileVersionRepo.FindByFileID(ctx, file.ID)
			if err != nil {
				return err
			}

			// パスを構築（閉包テーブルを使用して正確なパスを取得）
			originalPath, err := c.buildFilePath(ctx, file.FolderID, file.Name.String())
			if err != nil {
				return err
			}

			// ArchivedFile作成
			archivedFile := entity.NewArchivedFile(
				file.ID,
				file.FolderID,
				originalPath,
				file.Name,
				file.MimeType,
				file.Size,
				file.OwnerID,
				file.CreatedBy,
				file.StorageKey,
				input.UserID,
			)

			if err := c.archivedFileRepo.Create(ctx, archivedFile); err != nil {
				return err
			}

			// ArchivedFileVersion作成
			if len(versions) > 0 {
				archivedVersions := make([]*entity.ArchivedFileVersion, len(versions))
				for i, v := range versions {
					archivedVersions[i] = entity.NewArchivedFileVersion(
						archivedFile.ID,
						v.ID,
						v.VersionNumber,
						v.MinioVersionID,
						v.Size,
						v.Checksum,
						v.UploadedBy,
						v.CreatedAt,
					)
				}

				if err := c.archivedFileVersionRepo.BulkCreate(ctx, archivedVersions); err != nil {
					return err
				}
			}

			// 元ファイルとバージョンを削除
			if err := c.fileVersionRepo.DeleteByFileID(ctx, file.ID); err != nil {
				return err
			}
			if err := c.fileRepo.Delete(ctx, file.ID); err != nil {
				return err
			}

			archivedFileCount++
		}

		// 5b. 閉包テーブルからサブツリーのパスを削除
		if err := c.folderClosureRepo.DeleteSubtreePaths(ctx, folder.ID); err != nil {
			return err
		}

		// 5c. フォルダを削除（子孫から順に削除）
		// 深い順に並び替え（逆順）
		for i := len(folderIDs) - 1; i >= 0; i-- {
			if err := c.folderRepo.Delete(ctx, folderIDs[i]); err != nil {
				return err
			}
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return &DeleteFolderOutput{
		DeletedFolderCount: len(folderIDs),
		ArchivedFileCount:  archivedFileCount,
	}, nil
}

// buildFilePath はファイルのフルパスを構築します
// Note: ファイルは必ずフォルダに所属するため、folderIDは必須
func (c *DeleteFolderCommand) buildFilePath(ctx context.Context, folderID uuid.UUID, fileName string) (string, error) {
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
