package query

import (
	"context"

	"github.com/google/uuid"

	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/entity"
	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/repository"
	"github.com/Hiro-mackay/gc-storage/backend/pkg/apperror"
)

// GetGroupInput はグループ取得の入力を定義します
type GetGroupInput struct {
	GroupID uuid.UUID
	UserID  uuid.UUID
}

// GetGroupOutput はグループ取得の出力を定義します
type GetGroupOutput struct {
	Group       *entity.Group
	Membership  *entity.Membership
	MemberCount int
}

// GetGroupQuery はグループ取得クエリです
type GetGroupQuery struct {
	groupRepo      repository.GroupRepository
	membershipRepo repository.MembershipRepository
}

// NewGetGroupQuery は新しいGetGroupQueryを作成します
func NewGetGroupQuery(
	groupRepo repository.GroupRepository,
	membershipRepo repository.MembershipRepository,
) *GetGroupQuery {
	return &GetGroupQuery{
		groupRepo:      groupRepo,
		membershipRepo: membershipRepo,
	}
}

// Execute はグループ取得を実行します
func (q *GetGroupQuery) Execute(ctx context.Context, input GetGroupInput) (*GetGroupOutput, error) {
	// 1. グループの取得
	group, err := q.groupRepo.FindByID(ctx, input.GroupID)
	if err != nil {
		return nil, err
	}

	// 2. ユーザーのメンバーシップ確認
	membership, err := q.membershipRepo.FindByGroupAndUser(ctx, input.GroupID, input.UserID)
	if err != nil {
		return nil, apperror.NewForbiddenError("you are not a member of this group")
	}

	// 3. メンバー数を取得
	memberCount, err := q.membershipRepo.CountByGroupID(ctx, input.GroupID)
	if err != nil {
		return nil, err
	}

	return &GetGroupOutput{
		Group:       group,
		Membership:  membership,
		MemberCount: memberCount,
	}, nil
}
