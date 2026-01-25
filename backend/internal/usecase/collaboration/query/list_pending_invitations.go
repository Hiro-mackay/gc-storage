package query

import (
	"context"

	"github.com/google/uuid"

	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/entity"
	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/repository"
	"github.com/Hiro-mackay/gc-storage/backend/pkg/apperror"
)

// ListPendingInvitationsInput はユーザー宛招待一覧取得の入力を定義します
type ListPendingInvitationsInput struct {
	UserID uuid.UUID
}

// InvitationWithGroup は招待とグループ情報を結合した構造体
type InvitationWithGroup struct {
	Invitation *entity.Invitation
	Group      *entity.Group
}

// ListPendingInvitationsOutput はユーザー宛招待一覧取得の出力を定義します
type ListPendingInvitationsOutput struct {
	Invitations []*InvitationWithGroup
}

// ListPendingInvitationsQuery はユーザー宛招待一覧取得クエリです
type ListPendingInvitationsQuery struct {
	invitationRepo repository.InvitationRepository
	userRepo       repository.UserRepository
	groupRepo      repository.GroupRepository
}

// NewListPendingInvitationsQuery は新しいListPendingInvitationsQueryを作成します
func NewListPendingInvitationsQuery(
	invitationRepo repository.InvitationRepository,
	userRepo repository.UserRepository,
	groupRepo repository.GroupRepository,
) *ListPendingInvitationsQuery {
	return &ListPendingInvitationsQuery{
		invitationRepo: invitationRepo,
		userRepo:       userRepo,
		groupRepo:      groupRepo,
	}
}

// Execute はユーザー宛招待一覧取得を実行します
func (q *ListPendingInvitationsQuery) Execute(ctx context.Context, input ListPendingInvitationsInput) (*ListPendingInvitationsOutput, error) {
	// 1. ユーザーの取得
	user, err := q.userRepo.FindByID(ctx, input.UserID)
	if err != nil {
		return nil, apperror.NewNotFoundError("user")
	}

	// 2. ユーザーのメールアドレス宛の保留中招待を取得
	invitations, err := q.invitationRepo.FindPendingByEmail(ctx, user.Email)
	if err != nil {
		return nil, err
	}

	// 3. グループ情報を付加
	result := make([]*InvitationWithGroup, 0, len(invitations))
	for _, inv := range invitations {
		group, err := q.groupRepo.FindByID(ctx, inv.GroupID)
		if err != nil {
			continue // グループが見つからない場合はスキップ
		}
		result = append(result, &InvitationWithGroup{
			Invitation: inv,
			Group:      group,
		})
	}

	return &ListPendingInvitationsOutput{Invitations: result}, nil
}
