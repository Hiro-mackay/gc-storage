package di

import (
	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/authz"
	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/repository"
	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/service"
	"github.com/Hiro-mackay/gc-storage/backend/internal/infrastructure/database"
	infraRepo "github.com/Hiro-mackay/gc-storage/backend/internal/infrastructure/repository"
	storagecmd "github.com/Hiro-mackay/gc-storage/backend/internal/usecase/storage/command"
	storageqry "github.com/Hiro-mackay/gc-storage/backend/internal/usecase/storage/query"
)

// StorageUseCases はStorage関連のUseCaseを保持します
type StorageUseCases struct {
	// Folder Commands
	CreateFolder *storagecmd.CreateFolderCommand
	RenameFolder *storagecmd.RenameFolderCommand
	MoveFolder   *storagecmd.MoveFolderCommand
	DeleteFolder *storagecmd.DeleteFolderCommand

	// Folder Queries
	GetFolder          *storageqry.GetFolderQuery
	ListFolderContents *storageqry.ListFolderContentsQuery
	GetAncestors       *storageqry.GetAncestorsQuery

	// File Commands
	InitiateUpload        *storagecmd.InitiateUploadCommand
	CompleteUpload        *storagecmd.CompleteUploadCommand
	AbortUpload           *storagecmd.AbortUploadCommand
	RenameFile            *storagecmd.RenameFileCommand
	MoveFile              *storagecmd.MoveFileCommand
	TrashFile             *storagecmd.TrashFileCommand
	RestoreFile           *storagecmd.RestoreFileCommand
	PermanentlyDeleteFile *storagecmd.PermanentlyDeleteFileCommand
	EmptyTrash            *storagecmd.EmptyTrashCommand

	// File Queries
	GetDownloadURL   *storageqry.GetDownloadURLQuery
	GetUploadStatus  *storageqry.GetUploadStatusQuery
	ListFileVersions *storageqry.ListFileVersionsQuery
	ListTrash        *storageqry.ListTrashQuery
}

// StorageRepositories はStorage関連のリポジトリを保持します
type StorageRepositories struct {
	FolderRepo              repository.FolderRepository
	FolderClosureRepo       repository.FolderClosureRepository
	FileRepo                repository.FileRepository
	FileVersionRepo         repository.FileVersionRepository
	ArchivedFileRepo        repository.ArchivedFileRepository
	ArchivedFileVersionRepo repository.ArchivedFileVersionRepository
	UploadSessionRepo       repository.UploadSessionRepository
	UploadPartRepo          repository.UploadPartRepository
}

// NewStorageRepositories は新しいStorageRepositoriesを作成します
func NewStorageRepositories(txManager *database.TxManager) *StorageRepositories {
	return &StorageRepositories{
		FolderRepo:              infraRepo.NewFolderRepository(txManager),
		FolderClosureRepo:       infraRepo.NewFolderClosureRepository(txManager),
		FileRepo:                infraRepo.NewFileRepository(txManager),
		FileVersionRepo:         infraRepo.NewFileVersionRepository(txManager),
		ArchivedFileRepo:        infraRepo.NewArchivedFileRepository(txManager),
		ArchivedFileVersionRepo: infraRepo.NewArchivedFileVersionRepository(txManager),
		UploadSessionRepo:       infraRepo.NewUploadSessionRepository(txManager),
		UploadPartRepo:          infraRepo.NewUploadPartRepository(txManager),
	}
}

// NewStorageUseCases は新しいStorageUseCasesを作成します
func NewStorageUseCases(repos *StorageRepositories, userRepo repository.UserRepository, relationshipRepo authz.RelationshipRepository, permissionResolver authz.PermissionResolver, txManager repository.TransactionManager, storageService service.StorageService) *StorageUseCases {
	return &StorageUseCases{
		// Folder Commands
		CreateFolder: storagecmd.NewCreateFolderCommand(repos.FolderRepo, repos.FolderClosureRepo, relationshipRepo, permissionResolver, txManager),
		RenameFolder: storagecmd.NewRenameFolderCommand(repos.FolderRepo, userRepo),
		MoveFolder:   storagecmd.NewMoveFolderCommand(repos.FolderRepo, repos.FolderClosureRepo, txManager, userRepo, permissionResolver),
		DeleteFolder: storagecmd.NewDeleteFolderCommand(
			repos.FolderRepo,
			repos.FolderClosureRepo,
			repos.FileRepo,
			repos.FileVersionRepo,
			repos.ArchivedFileRepo,
			repos.ArchivedFileVersionRepo,
			txManager,
			userRepo,
		),

		// Folder Queries
		GetFolder:          storageqry.NewGetFolderQuery(repos.FolderRepo, permissionResolver),
		ListFolderContents: storageqry.NewListFolderContentsQuery(repos.FolderRepo, repos.FileRepo),
		GetAncestors:       storageqry.NewGetAncestorsQuery(repos.FolderRepo, repos.FolderClosureRepo),

		// File Commands
		InitiateUpload:        storagecmd.NewInitiateUploadCommand(repos.FileRepo, repos.FolderRepo, repos.UploadSessionRepo, storageService, txManager),
		CompleteUpload:        storagecmd.NewCompleteUploadCommand(repos.FileRepo, repos.FileVersionRepo, repos.UploadSessionRepo, repos.UploadPartRepo, txManager),
		AbortUpload:           storagecmd.NewAbortUploadCommand(repos.UploadSessionRepo, repos.FileRepo, storageService, txManager),
		RenameFile:            storagecmd.NewRenameFileCommand(repos.FileRepo),
		MoveFile:              storagecmd.NewMoveFileCommand(repos.FileRepo, repos.FolderRepo, permissionResolver),
		TrashFile:             storagecmd.NewTrashFileCommand(repos.FileRepo, repos.FileVersionRepo, repos.FolderRepo, repos.FolderClosureRepo, repos.ArchivedFileRepo, repos.ArchivedFileVersionRepo, txManager),
		RestoreFile:           storagecmd.NewRestoreFileCommand(repos.FileRepo, repos.FileVersionRepo, repos.FolderRepo, repos.ArchivedFileRepo, repos.ArchivedFileVersionRepo, userRepo, txManager),
		PermanentlyDeleteFile: storagecmd.NewPermanentlyDeleteFileCommand(repos.ArchivedFileRepo, repos.ArchivedFileVersionRepo, storageService, txManager),
		EmptyTrash:            storagecmd.NewEmptyTrashCommand(repos.ArchivedFileRepo, repos.ArchivedFileVersionRepo, storageService, txManager),

		// File Queries
		GetDownloadURL:   storageqry.NewGetDownloadURLQuery(repos.FileRepo, repos.FileVersionRepo, storageService),
		GetUploadStatus:  storageqry.NewGetUploadStatusQuery(repos.UploadSessionRepo),
		ListFileVersions: storageqry.NewListFileVersionsQuery(repos.FileRepo, repos.FileVersionRepo),
		ListTrash:        storageqry.NewListTrashQuery(repos.ArchivedFileRepo),
	}
}
