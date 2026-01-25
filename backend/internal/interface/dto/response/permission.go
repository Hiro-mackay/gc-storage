package response

import (
	"time"

	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/authz"
)

// PermissionGrantResponse は権限付与レスポンスです
type PermissionGrantResponse struct {
	ID           string    `json:"id"`
	ResourceType string    `json:"resourceType"`
	ResourceID   string    `json:"resourceId"`
	GranteeType  string    `json:"granteeType"`
	GranteeID    string    `json:"granteeId"`
	Role         string    `json:"role"`
	GrantedBy    string    `json:"grantedBy"`
	GrantedAt    time.Time `json:"grantedAt"`
}

// CheckPermissionResponse は権限確認レスポンスです
type CheckPermissionResponse struct {
	HasPermission bool   `json:"hasPermission"`
	EffectiveRole string `json:"effectiveRole"`
}

// ToPermissionGrantResponse はエンティティからレスポンスに変換します
func ToPermissionGrantResponse(grant *authz.PermissionGrant) PermissionGrantResponse {
	return PermissionGrantResponse{
		ID:           grant.ID.String(),
		ResourceType: grant.ResourceType.String(),
		ResourceID:   grant.ResourceID.String(),
		GranteeType:  grant.GranteeType.String(),
		GranteeID:    grant.GranteeID.String(),
		Role:         grant.Role.String(),
		GrantedBy:    grant.GrantedBy.String(),
		GrantedAt:    grant.GrantedAt,
	}
}

// ToPermissionGrantListResponse は権限付与リストをレスポンスリストに変換します
func ToPermissionGrantListResponse(grants []*authz.PermissionGrant) []PermissionGrantResponse {
	responses := make([]PermissionGrantResponse, len(grants))
	for i, g := range grants {
		responses[i] = ToPermissionGrantResponse(g)
	}
	return responses
}
