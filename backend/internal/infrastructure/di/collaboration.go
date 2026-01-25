package di

import (
	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/repository"
	"github.com/Hiro-mackay/gc-storage/backend/internal/infrastructure/database"
	infraRepo "github.com/Hiro-mackay/gc-storage/backend/internal/infrastructure/repository"
	collabcmd "github.com/Hiro-mackay/gc-storage/backend/internal/usecase/collaboration/command"
	collabqry "github.com/Hiro-mackay/gc-storage/backend/internal/usecase/collaboration/query"
)

// CollaborationUseCases はCollaboration関連のUseCaseを保持します
type CollaborationUseCases struct {
	// Group Commands
	CreateGroup       *collabcmd.CreateGroupCommand
	UpdateGroup       *collabcmd.UpdateGroupCommand
	DeleteGroup       *collabcmd.DeleteGroupCommand
	TransferOwnership *collabcmd.TransferOwnershipCommand

	// Member Commands
	InviteMember      *collabcmd.InviteMemberCommand
	AcceptInvitation  *collabcmd.AcceptInvitationCommand
	DeclineInvitation *collabcmd.DeclineInvitationCommand
	CancelInvitation  *collabcmd.CancelInvitationCommand
	RemoveMember      *collabcmd.RemoveMemberCommand
	LeaveGroup        *collabcmd.LeaveGroupCommand
	ChangeRole        *collabcmd.ChangeRoleCommand

	// Queries
	GetGroup               *collabqry.GetGroupQuery
	ListMyGroups           *collabqry.ListMyGroupsQuery
	ListMembers            *collabqry.ListMembersQuery
	ListInvitations        *collabqry.ListInvitationsQuery
	ListPendingInvitations *collabqry.ListPendingInvitationsQuery
}

// CollaborationRepositories はCollaboration関連のリポジトリを保持します
type CollaborationRepositories struct {
	GroupRepo      repository.GroupRepository
	MembershipRepo repository.MembershipRepository
	InvitationRepo repository.InvitationRepository
}

// NewCollaborationRepositories は新しいCollaborationRepositoriesを作成します
func NewCollaborationRepositories(txManager *database.TxManager) *CollaborationRepositories {
	return &CollaborationRepositories{
		GroupRepo:      infraRepo.NewGroupRepository(txManager),
		MembershipRepo: infraRepo.NewMembershipRepository(txManager),
		InvitationRepo: infraRepo.NewInvitationRepository(txManager),
	}
}

// NewCollaborationUseCases は新しいCollaborationUseCasesを作成します
func NewCollaborationUseCases(repos *CollaborationRepositories, userRepo repository.UserRepository, txManager repository.TransactionManager) *CollaborationUseCases {
	return &CollaborationUseCases{
		// Group Commands
		CreateGroup:       collabcmd.NewCreateGroupCommand(repos.GroupRepo, repos.MembershipRepo, txManager),
		UpdateGroup:       collabcmd.NewUpdateGroupCommand(repos.GroupRepo, repos.MembershipRepo),
		DeleteGroup:       collabcmd.NewDeleteGroupCommand(repos.GroupRepo, repos.MembershipRepo, repos.InvitationRepo, txManager),
		TransferOwnership: collabcmd.NewTransferOwnershipCommand(repos.GroupRepo, repos.MembershipRepo, txManager),

		// Member Commands
		InviteMember:      collabcmd.NewInviteMemberCommand(repos.GroupRepo, repos.MembershipRepo, repos.InvitationRepo, userRepo),
		AcceptInvitation:  collabcmd.NewAcceptInvitationCommand(repos.InvitationRepo, repos.GroupRepo, repos.MembershipRepo, userRepo, txManager),
		DeclineInvitation: collabcmd.NewDeclineInvitationCommand(repos.InvitationRepo, userRepo),
		CancelInvitation:  collabcmd.NewCancelInvitationCommand(repos.InvitationRepo, repos.MembershipRepo, repos.GroupRepo),
		RemoveMember:      collabcmd.NewRemoveMemberCommand(repos.GroupRepo, repos.MembershipRepo),
		LeaveGroup:        collabcmd.NewLeaveGroupCommand(repos.GroupRepo, repos.MembershipRepo),
		ChangeRole:        collabcmd.NewChangeRoleCommand(repos.GroupRepo, repos.MembershipRepo),

		// Queries
		GetGroup:               collabqry.NewGetGroupQuery(repos.GroupRepo, repos.MembershipRepo),
		ListMyGroups:           collabqry.NewListMyGroupsQuery(repos.GroupRepo, repos.MembershipRepo),
		ListMembers:            collabqry.NewListMembersQuery(repos.GroupRepo, repos.MembershipRepo),
		ListInvitations:        collabqry.NewListInvitationsQuery(repos.InvitationRepo, repos.MembershipRepo, repos.GroupRepo),
		ListPendingInvitations: collabqry.NewListPendingInvitationsQuery(repos.InvitationRepo, userRepo, repos.GroupRepo),
	}
}
