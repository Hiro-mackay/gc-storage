package command

import (
	"context"

	"github.com/google/uuid"

	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/repository"
	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/service"
	"github.com/Hiro-mackay/gc-storage/backend/pkg/apperror"
)

// 一度に削除する最大件数
const EmptyTrashChunkSize = 100

// EmptyTrashInput はゴミ箱を空にする入力を定義します
type EmptyTrashInput struct {
	OwnerID uuid.UUID
	UserID  uuid.UUID
}

// EmptyTrashOutput はゴミ箱を空にする出力を定義します
type EmptyTrashOutput struct {
	DeletedCount int
}

// EmptyTrashCommand はゴミ箱を空にするコマンドです
type EmptyTrashCommand struct {
	archivedFileRepo        repository.ArchivedFileRepository
	archivedFileVersionRepo repository.ArchivedFileVersionRepository
	storageService          service.StorageService
	txManager               repository.TransactionManager
}

// NewEmptyTrashCommand は新しいEmptyTrashCommandを作成します
func NewEmptyTrashCommand(
	archivedFileRepo repository.ArchivedFileRepository,
	archivedFileVersionRepo repository.ArchivedFileVersionRepository,
	storageService service.StorageService,
	txManager repository.TransactionManager,
) *EmptyTrashCommand {
	return &EmptyTrashCommand{
		archivedFileRepo:        archivedFileRepo,
		archivedFileVersionRepo: archivedFileVersionRepo,
		storageService:          storageService,
		txManager:               txManager,
	}
}

// Execute はゴミ箱を空にします
func (c *EmptyTrashCommand) Execute(ctx context.Context, input EmptyTrashInput) (*EmptyTrashOutput, error) {
	// 1. 所有者チェック
	if input.OwnerID != input.UserID {
		return nil, apperror.NewForbiddenError("not authorized to empty this trash")
	}

	// 2. 全アーカイブファイル取得
	archivedFiles, err := c.archivedFileRepo.FindByOwner(ctx, input.OwnerID)
	if err != nil {
		return nil, err
	}

	if len(archivedFiles) == 0 {
		return &EmptyTrashOutput{DeletedCount: 0}, nil
	}

	// 3. チャンクに分けて削除
	totalDeleted := 0
	storageKeysToDelete := make([]string, 0, len(archivedFiles))

	for i := 0; i < len(archivedFiles); i += EmptyTrashChunkSize {
		end := i + EmptyTrashChunkSize
		if end > len(archivedFiles) {
			end = len(archivedFiles)
		}
		chunk := archivedFiles[i:end]

		// チャンクごとにトランザクション実行
		err = c.txManager.WithTransaction(ctx, func(ctx context.Context) error {
			for _, af := range chunk {
				// バージョン削除
				if err := c.archivedFileVersionRepo.DeleteByArchivedFileID(ctx, af.ID); err != nil {
					return err
				}
				// ファイル削除
				if err := c.archivedFileRepo.Delete(ctx, af.ID); err != nil {
					return err
				}
			}
			return nil
		})

		if err != nil {
			// 部分的に成功した場合は削除済み件数を返す
			return &EmptyTrashOutput{DeletedCount: totalDeleted}, err
		}

		totalDeleted += len(chunk)

		// 削除成功したストレージキーを記録
		for _, af := range chunk {
			storageKeysToDelete = append(storageKeysToDelete, af.StorageKey.String())
		}
	}

	// 4. MinIOからオブジェクト削除（トランザクション外）
	for _, key := range storageKeysToDelete {
		_ = c.storageService.DeleteObject(ctx, key)
	}

	return &EmptyTrashOutput{DeletedCount: totalDeleted}, nil
}
