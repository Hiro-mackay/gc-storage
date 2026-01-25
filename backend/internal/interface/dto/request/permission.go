package request

// GrantRoleRequest は権限付与リクエストです
type GrantRoleRequest struct {
	GranteeType string `json:"granteeType" validate:"required,oneof=user group"`
	GranteeID   string `json:"granteeId" validate:"required,uuid"`
	Role        string `json:"role" validate:"required,oneof=viewer contributor content_manager"`
}
