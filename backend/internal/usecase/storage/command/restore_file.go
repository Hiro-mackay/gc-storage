package command

import (
	"context"
	"errors"

	"github.com/google/uuid"

	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/entity"
	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/repository"
	"github.com/Hiro-mackay/gc-storage/backend/pkg/apperror"
)

// RestoreFileInput はファイル復元の入力を定義します
// Note: ファイルは必ずフォルダに所属するため、復元先は必ず存在するフォルダになる
type RestoreFileInput struct {
	ArchivedFileID  uuid.UUID
	RestoreFolderID *uuid.UUID // nilの場合は元のフォルダに復元を試みる
	UserID          uuid.UUID
}

// RestoreFileOutput はファイル復元の出力を定義します
type RestoreFileOutput struct {
	FileID   uuid.UUID
	FolderID uuid.UUID // 必須 - 復元先フォルダID
	Name     string
}

// RestoreFileCommand はファイルをゴミ箱から復元するコマンドです
type RestoreFileCommand struct {
	fileRepo                repository.FileRepository
	fileVersionRepo         repository.FileVersionRepository
	folderRepo              repository.FolderRepository
	archivedFileRepo        repository.ArchivedFileRepository
	archivedFileVersionRepo repository.ArchivedFileVersionRepository
	userRepo                repository.UserRepository
	txManager               repository.TransactionManager
}

// NewRestoreFileCommand は新しいRestoreFileCommandを作成します
func NewRestoreFileCommand(
	fileRepo repository.FileRepository,
	fileVersionRepo repository.FileVersionRepository,
	folderRepo repository.FolderRepository,
	archivedFileRepo repository.ArchivedFileRepository,
	archivedFileVersionRepo repository.ArchivedFileVersionRepository,
	userRepo repository.UserRepository,
	txManager repository.TransactionManager,
) *RestoreFileCommand {
	return &RestoreFileCommand{
		fileRepo:                fileRepo,
		fileVersionRepo:         fileVersionRepo,
		folderRepo:              folderRepo,
		archivedFileRepo:        archivedFileRepo,
		archivedFileVersionRepo: archivedFileVersionRepo,
		userRepo:                userRepo,
		txManager:               txManager,
	}
}

// Execute はファイルをゴミ箱から復元します
func (c *RestoreFileCommand) Execute(ctx context.Context, input RestoreFileInput) (*RestoreFileOutput, error) {
	// 1. アーカイブファイル取得
	archivedFile, err := c.archivedFileRepo.FindByID(ctx, input.ArchivedFileID)
	if err != nil {
		return nil, err
	}

	// 2. 所有者チェック
	if !archivedFile.IsOwnedBy(input.UserID) {
		return nil, apperror.NewForbiddenError("not authorized to restore this file")
	}

	// 3. 期限切れチェック
	if archivedFile.IsExpired() {
		return nil, apperror.NewValidationError("archived file has expired", nil)
	}

	// 4. 復元先フォルダを決定
	restoreFolderID, err := c.determineRestoreFolder(ctx, archivedFile, input.RestoreFolderID)
	if err != nil {
		return nil, err
	}

	// 5. 同名ファイルの存在チェック
	exists, err := c.fileRepo.ExistsByNameAndFolder(ctx, archivedFile.Name, restoreFolderID)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, apperror.NewConflictError("file with same name already exists in restore folder")
	}

	// 6. アーカイブバージョンを取得
	archivedVersions, err := c.archivedFileVersionRepo.FindByArchivedFileID(ctx, archivedFile.ID)
	if err != nil {
		return nil, err
	}

	// 7. ファイルデータを復元形式に変換
	file := archivedFile.ToFile(restoreFolderID) // restoreFolderID is now uuid.UUID

	// バージョン数を計算して設定
	if len(archivedVersions) > 0 {
		maxVersion := 0
		for _, v := range archivedVersions {
			if v.VersionNumber > maxVersion {
				maxVersion = v.VersionNumber
			}
		}
		file.CurrentVersion = maxVersion
	}

	// 8. トランザクションで復元処理
	err = c.txManager.WithTransaction(ctx, func(ctx context.Context) error {
		// ファイル作成
		if err := c.fileRepo.Create(ctx, file); err != nil {
			return err
		}

		// バージョン復元
		if len(archivedVersions) > 0 {
			versions := make([]*entity.FileVersion, len(archivedVersions))
			for i, av := range archivedVersions {
				versions[i] = av.ToFileVersion(file.ID)
			}
			if err := c.fileVersionRepo.BulkCreate(ctx, versions); err != nil {
				return err
			}
		}

		// アーカイブバージョン削除
		if err := c.archivedFileVersionRepo.DeleteByArchivedFileID(ctx, archivedFile.ID); err != nil {
			return err
		}

		// アーカイブファイル削除
		return c.archivedFileRepo.Delete(ctx, archivedFile.ID)
	})

	if err != nil {
		return nil, err
	}

	return &RestoreFileOutput{
		FileID:   file.ID,
		FolderID: restoreFolderID,
		Name:     archivedFile.Name.String(),
	}, nil
}

// determineRestoreFolder は復元先フォルダを決定します
// Note: ファイルは必ずフォルダに所属するため、戻り値はuuid.UUID
func (c *RestoreFileCommand) determineRestoreFolder(
	ctx context.Context,
	archivedFile *entity.ArchivedFile,
	requestedFolderID *uuid.UUID,
) (uuid.UUID, error) {
	// 明示的に指定された場合
	if requestedFolderID != nil {
		// フォルダが存在するか確認
		folder, err := c.folderRepo.FindByID(ctx, *requestedFolderID)
		if err != nil {
			return uuid.Nil, apperror.NewNotFoundError("restore folder not found")
		}

		// 所有者チェック
		if !folder.IsOwnedBy(archivedFile.OwnerID) {
			return uuid.Nil, apperror.NewForbiddenError("not authorized to restore to this folder")
		}

		return *requestedFolderID, nil
	}

	// 元のフォルダが存在する場合はそこへ復元
	exists, err := c.folderRepo.ExistsByID(ctx, archivedFile.OriginalFolderID)
	if err != nil {
		return uuid.Nil, err
	}
	if exists {
		return archivedFile.OriginalFolderID, nil
	}

	// 元のフォルダが存在しない場合はPersonal Folderにフォールバック (R-AF003)
	user, err := c.userRepo.FindByID(ctx, archivedFile.OwnerID)
	if err != nil {
		return uuid.Nil, err
	}
	if user.PersonalFolderID == nil {
		return uuid.Nil, apperror.NewInternalError(errors.New("user has no personal folder"))
	}
	return *user.PersonalFolderID, nil
}
