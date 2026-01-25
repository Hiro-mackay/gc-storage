package query

import (
	"context"

	"github.com/google/uuid"

	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/entity"
	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/repository"
	"github.com/Hiro-mackay/gc-storage/backend/pkg/apperror"
)

// ListMembersInput はメンバー一覧取得の入力を定義します
type ListMembersInput struct {
	GroupID uuid.UUID
	UserID  uuid.UUID // 取得を要求しているユーザー
}

// ListMembersOutput はメンバー一覧取得の出力を定義します
type ListMembersOutput struct {
	Members []*entity.MembershipWithUser
}

// ListMembersQuery はメンバー一覧取得クエリです
type ListMembersQuery struct {
	groupRepo      repository.GroupRepository
	membershipRepo repository.MembershipRepository
}

// NewListMembersQuery は新しいListMembersQueryを作成します
func NewListMembersQuery(
	groupRepo repository.GroupRepository,
	membershipRepo repository.MembershipRepository,
) *ListMembersQuery {
	return &ListMembersQuery{
		groupRepo:      groupRepo,
		membershipRepo: membershipRepo,
	}
}

// Execute はメンバー一覧取得を実行します
func (q *ListMembersQuery) Execute(ctx context.Context, input ListMembersInput) (*ListMembersOutput, error) {
	// 1. グループの存在確認
	_, err := q.groupRepo.FindByID(ctx, input.GroupID)
	if err != nil {
		return nil, err
	}

	// 2. 要求者がメンバーかどうか確認
	isMember, err := q.membershipRepo.Exists(ctx, input.GroupID, input.UserID)
	if err != nil {
		return nil, err
	}
	if !isMember {
		return nil, apperror.NewForbiddenError("you are not a member of this group")
	}

	// 3. メンバー一覧を取得
	members, err := q.membershipRepo.FindByGroupIDWithUsers(ctx, input.GroupID)
	if err != nil {
		return nil, err
	}

	return &ListMembersOutput{Members: members}, nil
}
