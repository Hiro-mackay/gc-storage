package query

import (
	"context"

	"github.com/google/uuid"

	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/entity"
	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/repository"
	"github.com/Hiro-mackay/gc-storage/backend/pkg/apperror"
)

// ListInvitationsInput は招待一覧取得の入力を定義します
type ListInvitationsInput struct {
	GroupID   uuid.UUID
	RequestBy uuid.UUID
}

// ListInvitationsOutput は招待一覧取得の出力を定義します
type ListInvitationsOutput struct {
	Invitations []*entity.Invitation
}

// ListInvitationsQuery は招待一覧取得クエリです
type ListInvitationsQuery struct {
	invitationRepo repository.InvitationRepository
	membershipRepo repository.MembershipRepository
	groupRepo      repository.GroupRepository
}

// NewListInvitationsQuery は新しいListInvitationsQueryを作成します
func NewListInvitationsQuery(
	invitationRepo repository.InvitationRepository,
	membershipRepo repository.MembershipRepository,
	groupRepo repository.GroupRepository,
) *ListInvitationsQuery {
	return &ListInvitationsQuery{
		invitationRepo: invitationRepo,
		membershipRepo: membershipRepo,
		groupRepo:      groupRepo,
	}
}

// Execute は招待一覧取得を実行します
func (q *ListInvitationsQuery) Execute(ctx context.Context, input ListInvitationsInput) (*ListInvitationsOutput, error) {
	// 1. グループの存在確認
	_, err := q.groupRepo.FindByID(ctx, input.GroupID)
	if err != nil {
		return nil, apperror.NewNotFoundError("group")
	}

	// 2. 操作者のメンバーシップ確認（contributor以上）
	membership, err := q.membershipRepo.FindByGroupAndUser(ctx, input.GroupID, input.RequestBy)
	if err != nil {
		return nil, apperror.NewForbiddenError("not a member of this group")
	}
	if !membership.CanInvite() {
		return nil, apperror.NewForbiddenError("insufficient permission to view invitations")
	}

	// 3. 保留中の招待一覧を取得
	invitations, err := q.invitationRepo.FindPendingByGroupID(ctx, input.GroupID)
	if err != nil {
		return nil, err
	}

	return &ListInvitationsOutput{Invitations: invitations}, nil
}
