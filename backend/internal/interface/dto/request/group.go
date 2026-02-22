package request

// CreateGroupRequest はグループ作成リクエストです
type CreateGroupRequest struct {
	Name        string `json:"name" validate:"required,min=1,max=100"`
	Description string `json:"description" validate:"max=500"`
}

// UpdateGroupRequest はグループ更新リクエストです
type UpdateGroupRequest struct {
	Name        *string `json:"name" validate:"omitempty,min=1,max=100"`
	Description *string `json:"description" validate:"omitempty,max=500"`
}

// InviteMemberRequest はメンバー招待リクエストです
type InviteMemberRequest struct {
	Email string `json:"email" validate:"required,email,max=255"`
	Role  string `json:"role" validate:"omitempty,oneof=viewer contributor"`
}

// ChangeRoleRequest はロール変更リクエストです
type ChangeRoleRequest struct {
	Role string `json:"role" validate:"required,oneof=viewer contributor"`
}

// TransferOwnershipRequest は所有権譲渡リクエストです
type TransferOwnershipRequest struct {
	NewOwnerID string `json:"newOwnerId" validate:"required,uuid"`
}
