package query

import (
	"context"

	"github.com/google/uuid"

	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/entity"
	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/repository"
)

// ListMyGroupsInput はマイグループ一覧取得の入力を定義します
type ListMyGroupsInput struct {
	UserID uuid.UUID
}

// GroupWithMembership はグループとメンバーシップを結合した構造体
type GroupWithMembership struct {
	Group      *entity.Group
	Membership *entity.Membership
}

// ListMyGroupsOutput はマイグループ一覧取得の出力を定義します
type ListMyGroupsOutput struct {
	Groups []*GroupWithMembership
}

// ListMyGroupsQuery はマイグループ一覧取得クエリです
type ListMyGroupsQuery struct {
	groupRepo      repository.GroupRepository
	membershipRepo repository.MembershipRepository
}

// NewListMyGroupsQuery は新しいListMyGroupsQueryを作成します
func NewListMyGroupsQuery(
	groupRepo repository.GroupRepository,
	membershipRepo repository.MembershipRepository,
) *ListMyGroupsQuery {
	return &ListMyGroupsQuery{
		groupRepo:      groupRepo,
		membershipRepo: membershipRepo,
	}
}

// Execute はマイグループ一覧取得を実行します
func (q *ListMyGroupsQuery) Execute(ctx context.Context, input ListMyGroupsInput) (*ListMyGroupsOutput, error) {
	// 1. ユーザーが所属するグループを取得
	groups, err := q.groupRepo.FindByMemberID(ctx, input.UserID)
	if err != nil {
		return nil, err
	}

	// 2. 各グループのメンバーシップを取得
	result := make([]*GroupWithMembership, 0, len(groups))
	for _, group := range groups {
		membership, err := q.membershipRepo.FindByGroupAndUser(ctx, group.ID, input.UserID)
		if err != nil {
			continue // メンバーシップが見つからない場合はスキップ
		}

		result = append(result, &GroupWithMembership{
			Group:      group,
			Membership: membership,
		})
	}

	return &ListMyGroupsOutput{Groups: result}, nil
}
