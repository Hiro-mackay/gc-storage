package di

import (
	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/authz"
	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/repository"
	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/service"
	"github.com/Hiro-mackay/gc-storage/backend/internal/infrastructure/database"
	infraRepo "github.com/Hiro-mackay/gc-storage/backend/internal/infrastructure/repository"
	sharingcmd "github.com/Hiro-mackay/gc-storage/backend/internal/usecase/sharing/command"
	sharingqry "github.com/Hiro-mackay/gc-storage/backend/internal/usecase/sharing/query"
)

// SharingUseCases はSharing関連のUseCaseを保持します
type SharingUseCases struct {
	// Commands
	CreateShareLink *sharingcmd.CreateShareLinkCommand
	RevokeShareLink *sharingcmd.RevokeShareLinkCommand
	UpdateShareLink *sharingcmd.UpdateShareLinkCommand

	// Queries
	AccessShareLink     *sharingqry.AccessShareLinkQuery
	ListShareLinks      *sharingqry.ListShareLinksQuery
	GetShareLinkHistory *sharingqry.GetShareLinkHistoryQuery
	GetDownloadViaShare *sharingqry.GetDownloadViaShareQuery
}

// SharingRepositories はSharing関連のリポジトリを保持します
type SharingRepositories struct {
	ShareLinkRepo       repository.ShareLinkRepository
	ShareLinkAccessRepo repository.ShareLinkAccessRepository
}

// NewSharingRepositories は新しいSharingRepositoriesを作成します
func NewSharingRepositories(txManager *database.TxManager) *SharingRepositories {
	return &SharingRepositories{
		ShareLinkRepo:       infraRepo.NewShareLinkRepository(txManager),
		ShareLinkAccessRepo: infraRepo.NewShareLinkAccessRepository(txManager),
	}
}

// NewSharingUseCases は新しいSharingUseCasesを作成します
func NewSharingUseCases(
	repos *SharingRepositories,
	resolver authz.PermissionResolver,
	storageRepos *StorageRepositories,
	storageService service.StorageService,
) *SharingUseCases {
	return &SharingUseCases{
		// Commands
		CreateShareLink: sharingcmd.NewCreateShareLinkCommand(repos.ShareLinkRepo, resolver),
		RevokeShareLink: sharingcmd.NewRevokeShareLinkCommand(repos.ShareLinkRepo, resolver),
		UpdateShareLink: sharingcmd.NewUpdateShareLinkCommand(repos.ShareLinkRepo, resolver),

		// Queries
		AccessShareLink: sharingqry.NewAccessShareLinkQuery(
			repos.ShareLinkRepo,
			repos.ShareLinkAccessRepo,
			storageRepos.FileRepo,
			storageRepos.FileVersionRepo,
			storageRepos.FolderRepo,
			storageService,
		),
		ListShareLinks: sharingqry.NewListShareLinksQuery(repos.ShareLinkRepo, resolver),
		GetShareLinkHistory: sharingqry.NewGetShareLinkHistoryQuery(
			repos.ShareLinkRepo,
			repos.ShareLinkAccessRepo,
			resolver,
		),
		GetDownloadViaShare: sharingqry.NewGetDownloadViaShareQuery(
			repos.ShareLinkRepo,
			repos.ShareLinkAccessRepo,
			storageRepos.FileRepo,
			storageRepos.FileVersionRepo,
			storageRepos.FolderClosureRepo,
			storageService,
		),
	}
}
