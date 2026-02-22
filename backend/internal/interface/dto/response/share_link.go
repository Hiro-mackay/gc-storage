package response

import (
	"time"

	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/entity"
)

// ShareLinkResponse は共有リンクレスポンスです
type ShareLinkResponse struct {
	ID             string     `json:"id"`
	Token          string     `json:"token"`
	URL            string     `json:"url"`
	ResourceType   string     `json:"resourceType"`
	ResourceID     string     `json:"resourceId"`
	Permission     string     `json:"permission"`
	HasPassword    bool       `json:"hasPassword"`
	ExpiresAt      *time.Time `json:"expiresAt,omitempty"`
	MaxAccessCount *int       `json:"maxAccessCount,omitempty"`
	AccessCount    int        `json:"accessCount"`
	Status         string     `json:"status"`
	CreatedAt      time.Time  `json:"createdAt"`
}

// ShareLinkInfoResponse は共有リンク情報レスポンス（アクセス前）です
type ShareLinkInfoResponse struct {
	ResourceType string `json:"resourceType"`
	Permission   string `json:"permission"`
	HasPassword  bool   `json:"hasPassword"`
}

// ShareLinkAccessResponse は共有リンクアクセスレスポンスです
type ShareLinkAccessResponse struct {
	ResourceType string `json:"resourceType"`
	ResourceID   string `json:"resourceId"`
	Permission   string `json:"permission"`
}

// ToShareLinkResponse はエンティティからレスポンスに変換します
func ToShareLinkResponse(link *entity.ShareLink, baseURL string) ShareLinkResponse {
	var expiresAt *time.Time
	if link.ExpiresAt != nil {
		expiresAt = link.ExpiresAt
	}

	var maxAccessCount *int
	if link.MaxAccessCount != nil {
		maxAccessCount = link.MaxAccessCount
	}

	return ShareLinkResponse{
		ID:             link.ID.String(),
		Token:          link.Token.String(),
		URL:            baseURL + "/share/" + link.Token.String(),
		ResourceType:   link.ResourceType.String(),
		ResourceID:     link.ResourceID.String(),
		Permission:     link.Permission.String(),
		HasPassword:    link.RequiresPassword(),
		ExpiresAt:      expiresAt,
		MaxAccessCount: maxAccessCount,
		AccessCount:    link.AccessCount,
		Status:         link.Status.String(),
		CreatedAt:      link.CreatedAt,
	}
}

// ToShareLinkListResponse は共有リンクリストをレスポンスリストに変換します
func ToShareLinkListResponse(links []*entity.ShareLink, baseURL string) []ShareLinkResponse {
	responses := make([]ShareLinkResponse, len(links))
	for i, link := range links {
		responses[i] = ToShareLinkResponse(link, baseURL)
	}
	return responses
}

// ToShareLinkInfoResponse はエンティティからアクセス前情報レスポンスに変換します
func ToShareLinkInfoResponse(link *entity.ShareLink) ShareLinkInfoResponse {
	return ShareLinkInfoResponse{
		ResourceType: link.ResourceType.String(),
		Permission:   link.Permission.String(),
		HasPassword:  link.RequiresPassword(),
	}
}

// ToShareLinkAccessResponse はアクセス結果レスポンスに変換します
func ToShareLinkAccessResponse(link *entity.ShareLink) ShareLinkAccessResponse {
	return ShareLinkAccessResponse{
		ResourceType: link.ResourceType.String(),
		ResourceID:   link.ResourceID.String(),
		Permission:   link.Permission.String(),
	}
}
