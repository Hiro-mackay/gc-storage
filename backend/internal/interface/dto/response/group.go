package response

import (
	"time"

	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/entity"
	"github.com/Hiro-mackay/gc-storage/backend/internal/usecase/collaboration/query"
)

// GroupResponse はグループレスポンスです
// Note: Groupは論理削除をサポートしないため、Statusフィールドは削除されました
type GroupResponse struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	OwnerID     string    `json:"ownerId"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

// GroupWithMembershipResponse はグループとメンバーシップ情報付きレスポンスです
type GroupWithMembershipResponse struct {
	Group       GroupResponse `json:"group"`
	MyRole      string        `json:"myRole"`
	MemberCount int           `json:"memberCount,omitempty"`
}

// MembershipResponse はメンバーシップレスポンスです
type MembershipResponse struct {
	ID       string    `json:"id"`
	GroupID  string    `json:"groupId"`
	UserID   string    `json:"userId"`
	Role     string    `json:"role"`
	JoinedAt time.Time `json:"joinedAt"`
}

// MemberResponse はメンバー情報レスポンスです
type MemberResponse struct {
	ID       string    `json:"id"`
	UserID   string    `json:"userId"`
	Email    string    `json:"email"`
	Name     string    `json:"name"`
	Role     string    `json:"role"`
	Status   string    `json:"status"`
	JoinedAt time.Time `json:"joinedAt"`
}

// InvitationResponse は招待レスポンスです
type InvitationResponse struct {
	ID        string    `json:"id"`
	GroupID   string    `json:"groupId"`
	Email     string    `json:"email"`
	Role      string    `json:"role"`
	Status    string    `json:"status"`
	ExpiresAt time.Time `json:"expiresAt"`
	CreatedAt time.Time `json:"createdAt"`
}

// ToGroupResponse はエンティティからレスポンスに変換します
func ToGroupResponse(group *entity.Group) GroupResponse {
	return GroupResponse{
		ID:          group.ID.String(),
		Name:        group.Name.String(),
		Description: group.Description,
		OwnerID:     group.OwnerID.String(),
		CreatedAt:   group.CreatedAt,
		UpdatedAt:   group.UpdatedAt,
	}
}

// ToGroupWithMembershipResponse はグループとメンバーシップからレスポンスに変換します
func ToGroupWithMembershipResponse(gm *query.GroupWithMembership) GroupWithMembershipResponse {
	return GroupWithMembershipResponse{
		Group:  ToGroupResponse(gm.Group),
		MyRole: gm.Membership.Role.String(),
	}
}

// ToGroupWithMembershipAndCountResponse はグループとメンバーシップとカウントからレスポンスに変換します
func ToGroupWithMembershipAndCountResponse(group *entity.Group, membership *entity.Membership, memberCount int) GroupWithMembershipResponse {
	return GroupWithMembershipResponse{
		Group:       ToGroupResponse(group),
		MyRole:      membership.Role.String(),
		MemberCount: memberCount,
	}
}

// ToMembershipResponse はエンティティからレスポンスに変換します
func ToMembershipResponse(membership *entity.Membership) MembershipResponse {
	return MembershipResponse{
		ID:       membership.ID.String(),
		GroupID:  membership.GroupID.String(),
		UserID:   membership.UserID.String(),
		Role:     membership.Role.String(),
		JoinedAt: membership.JoinedAt,
	}
}

// ToMemberResponse はメンバーシップとユーザーからレスポンスに変換します
func ToMemberResponse(mwu *entity.MembershipWithUser) MemberResponse {
	return MemberResponse{
		ID:       mwu.Membership.ID.String(),
		UserID:   mwu.User.ID.String(),
		Email:    mwu.User.Email.String(),
		Name:     mwu.User.Name,
		Role:     mwu.Membership.Role.String(),
		Status:   string(mwu.User.Status),
		JoinedAt: mwu.Membership.JoinedAt,
	}
}

// ToMemberListResponse はメンバーリストをレスポンスリストに変換します
func ToMemberListResponse(members []*entity.MembershipWithUser) []MemberResponse {
	responses := make([]MemberResponse, len(members))
	for i, m := range members {
		responses[i] = ToMemberResponse(m)
	}
	return responses
}

// ToInvitationResponse はエンティティからレスポンスに変換します
func ToInvitationResponse(invitation *entity.Invitation) InvitationResponse {
	return InvitationResponse{
		ID:        invitation.ID.String(),
		GroupID:   invitation.GroupID.String(),
		Email:     invitation.Email.String(),
		Role:      invitation.Role.String(),
		Status:    invitation.Status.String(),
		ExpiresAt: invitation.ExpiresAt,
		CreatedAt: invitation.CreatedAt,
	}
}

// ToInvitationListResponse は招待リストをレスポンスリストに変換します
func ToInvitationListResponse(invitations []*entity.Invitation) []InvitationResponse {
	responses := make([]InvitationResponse, len(invitations))
	for i, inv := range invitations {
		responses[i] = ToInvitationResponse(inv)
	}
	return responses
}

// ToGroupListResponse はグループリストをレスポンスリストに変換します
func ToGroupListResponse(groups []*query.GroupWithMembership) []GroupWithMembershipResponse {
	responses := make([]GroupWithMembershipResponse, len(groups))
	for i, gm := range groups {
		responses[i] = ToGroupWithMembershipResponse(gm)
	}
	return responses
}

// PendingInvitationResponse はユーザー宛保留中招待レスポンスです
type PendingInvitationResponse struct {
	Invitation InvitationResponse `json:"invitation"`
	Group      GroupResponse      `json:"group"`
}

// ToPendingInvitationResponse は招待とグループからレスポンスに変換します
func ToPendingInvitationResponse(iwg *query.InvitationWithGroup) PendingInvitationResponse {
	return PendingInvitationResponse{
		Invitation: ToInvitationResponse(iwg.Invitation),
		Group:      ToGroupResponse(iwg.Group),
	}
}

// ToPendingInvitationListResponse は保留中招待リストをレスポンスリストに変換します
func ToPendingInvitationListResponse(invitations []*query.InvitationWithGroup) []PendingInvitationResponse {
	responses := make([]PendingInvitationResponse, len(invitations))
	for i, inv := range invitations {
		responses[i] = ToPendingInvitationResponse(inv)
	}
	return responses
}
