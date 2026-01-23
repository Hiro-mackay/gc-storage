package query

import (
	"context"

	"github.com/google/uuid"

	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/entity"
	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/repository"
	"github.com/Hiro-mackay/gc-storage/backend/pkg/apperror"
)

// ListFolderContentsInput はフォルダ内容一覧の入力を定義します
type ListFolderContentsInput struct {
	FolderID *uuid.UUID // nil の場合はルートレベル
	OwnerID  uuid.UUID
	UserID   uuid.UUID
}

// ListFolderContentsOutput はフォルダ内容一覧の出力を定義します
type ListFolderContentsOutput struct {
	Folder  *entity.Folder   // ルートの場合はnil
	Folders []*entity.Folder
	Files   []*entity.File
}

// ListFolderContentsQuery はフォルダ内容一覧クエリです
type ListFolderContentsQuery struct {
	folderRepo repository.FolderRepository
	fileRepo   repository.FileRepository
}

// NewListFolderContentsQuery は新しいListFolderContentsQueryを作成します
func NewListFolderContentsQuery(
	folderRepo repository.FolderRepository,
	fileRepo repository.FileRepository,
) *ListFolderContentsQuery {
	return &ListFolderContentsQuery{
		folderRepo: folderRepo,
		fileRepo:   fileRepo,
	}
}

// Execute はフォルダ内容一覧を取得します
func (q *ListFolderContentsQuery) Execute(ctx context.Context, input ListFolderContentsInput) (*ListFolderContentsOutput, error) {
	var folder *entity.Folder

	// 1. 特定フォルダの場合、フォルダを取得して権限チェック
	if input.FolderID != nil {
		var err error
		folder, err = q.folderRepo.FindByID(ctx, *input.FolderID)
		if err != nil {
			return nil, err
		}

		// 所有者チェック
		if !folder.IsOwnedBy(input.UserID) {
			return nil, apperror.NewForbiddenError("not authorized to access this folder")
		}

		// 入力の所有者情報をフォルダから取得
		input.OwnerID = folder.OwnerID
	}

	// 2. サブフォルダ取得
	var folders []*entity.Folder
	var err error
	if input.FolderID != nil {
		folders, err = q.folderRepo.FindByParentID(ctx, input.FolderID, input.OwnerID)
	} else {
		folders, err = q.folderRepo.FindRootByOwner(ctx, input.OwnerID)
	}
	if err != nil {
		return nil, err
	}

	// 3. ファイル取得
	// Note: FolderIDがnilの場合はルートレベルなのでファイルは取得しない
	// （ファイルは必ずフォルダに所属するため）
	var files []*entity.File
	if input.FolderID != nil {
		files, err = q.fileRepo.FindByFolderID(ctx, *input.FolderID)
		if err != nil {
			return nil, err
		}

		// アクティブなファイルのみをフィルタ
		activeFiles := make([]*entity.File, 0, len(files))
		for _, f := range files {
			if f.IsActive() {
				activeFiles = append(activeFiles, f)
			}
		}
		files = activeFiles
	} else {
		files = []*entity.File{}
	}

	return &ListFolderContentsOutput{
		Folder:  folder,
		Folders: folders,
		Files:   files,
	}, nil
}
