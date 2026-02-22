package di

import (
	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/authz"
	infraAuthz "github.com/Hiro-mackay/gc-storage/backend/internal/infrastructure/authz"
	"github.com/Hiro-mackay/gc-storage/backend/internal/infrastructure/database"
	authzcmd "github.com/Hiro-mackay/gc-storage/backend/internal/usecase/authz/command"
	authzqry "github.com/Hiro-mackay/gc-storage/backend/internal/usecase/authz/query"
)

// AuthzUseCases はAuthorization関連のUseCaseを保持します
type AuthzUseCases struct {
	// Commands
	GrantRole   *authzcmd.GrantRoleCommand
	RevokeGrant *authzcmd.RevokeGrantCommand

	// Queries
	ListGrants      *authzqry.ListGrantsQuery
	CheckPermission *authzqry.CheckPermissionQuery
}

// AuthzRepositories はAuthorization関連のリポジトリを保持します
type AuthzRepositories struct {
	PermissionGrantRepo authz.PermissionGrantRepository
	RelationshipRepo    authz.RelationshipRepository
}

// NewAuthzRepositories は新しいAuthzRepositoriesを作成します
func NewAuthzRepositories(txManager *database.TxManager) *AuthzRepositories {
	return &AuthzRepositories{
		PermissionGrantRepo: infraAuthz.NewPermissionGrantRepository(txManager),
		RelationshipRepo:    infraAuthz.NewRelationshipRepository(txManager),
	}
}

// NewPermissionResolver は新しいPermissionResolverを作成します
func NewPermissionResolver(
	authzRepos *AuthzRepositories,
	collabRepos *CollaborationRepositories,
) authz.PermissionResolver {
	return infraAuthz.NewPermissionResolver(
		authzRepos.PermissionGrantRepo,
		authzRepos.RelationshipRepo,
		collabRepos.MembershipRepo,
	)
}

// NewAuthzUseCases は新しいAuthzUseCasesを作成します
func NewAuthzUseCases(
	repos *AuthzRepositories,
	resolver authz.PermissionResolver,
) *AuthzUseCases {
	return &AuthzUseCases{
		// Commands
		GrantRole:   authzcmd.NewGrantRoleCommand(repos.PermissionGrantRepo, resolver),
		RevokeGrant: authzcmd.NewRevokeGrantCommand(repos.PermissionGrantRepo, resolver),

		// Queries
		ListGrants:      authzqry.NewListGrantsQuery(repos.PermissionGrantRepo, resolver),
		CheckPermission: authzqry.NewCheckPermissionQuery(resolver),
	}
}
