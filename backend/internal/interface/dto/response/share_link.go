package response

import (
	"time"

	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/entity"
	sharingqry "github.com/Hiro-mackay/gc-storage/backend/internal/usecase/sharing/query"
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

// FolderContentResponse はフォルダ内コンテンツレスポンスです
type FolderContentResponse struct {
	ID       string  `json:"id"`
	Name     string  `json:"name"`
	Type     string  `json:"type"`
	MimeType *string `json:"mimeType,omitempty"`
	Size     *int64  `json:"size,omitempty"`
}

// ShareLinkAccessResponse は共有リンクアクセスレスポンスです
type ShareLinkAccessResponse struct {
	ResourceType string                  `json:"resourceType"`
	ResourceID   string                  `json:"resourceId"`
	ResourceName string                  `json:"resourceName"`
	Permission   string                  `json:"permission"`
	PresignedURL *string                 `json:"presignedUrl,omitempty"`
	Contents     []FolderContentResponse `json:"contents,omitempty"`
}

// ShareLinkAccessHistoryResponse は共有リンクアクセス履歴レスポンスです
type ShareLinkAccessHistoryResponse struct {
	ID         string  `json:"id"`
	AccessedAt string  `json:"accessedAt"`
	IPAddress  string  `json:"ipAddress"`
	UserAgent  string  `json:"userAgent"`
	UserID     *string `json:"userId,omitempty"`
	Action     string  `json:"action"`
}

// ShareLinkAccessListResponse は共有リンクアクセス履歴リストレスポンスです
type ShareLinkAccessListResponse struct {
	Items []ShareLinkAccessHistoryResponse `json:"items"`
	Total int                              `json:"total"`
}

// ShareDownloadResponse は共有リンク経由ダウンロードレスポンスです
type ShareDownloadResponse struct {
	PresignedURL string `json:"presignedUrl"`
	FileName     string `json:"fileName"`
	FileSize     int64  `json:"fileSize"`
	MimeType     string `json:"mimeType"`
	ExpiresAt    string `json:"expiresAt"`
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
func ToShareLinkAccessResponse(output *sharingqry.AccessShareLinkOutput) ShareLinkAccessResponse {
	var contents []FolderContentResponse
	if len(output.Contents) > 0 {
		contents = make([]FolderContentResponse, len(output.Contents))
		for i, c := range output.Contents {
			contents[i] = FolderContentResponse{
				ID:       c.ID.String(),
				Name:     c.Name,
				Type:     c.Type,
				MimeType: c.MimeType,
				Size:     c.Size,
			}
		}
	}

	return ShareLinkAccessResponse{
		ResourceType: output.ResourceType,
		ResourceID:   output.ResourceID.String(),
		ResourceName: output.ResourceName,
		Permission:   output.ShareLink.Permission.String(),
		PresignedURL: output.PresignedURL,
		Contents:     contents,
	}
}

// ToShareLinkAccessHistoryResponse はアクセスログエンティティからレスポンスに変換します
func ToShareLinkAccessHistoryResponse(access *entity.ShareLinkAccess) ShareLinkAccessHistoryResponse {
	var userID *string
	if access.UserID != nil {
		s := access.UserID.String()
		userID = &s
	}
	return ShareLinkAccessHistoryResponse{
		ID:         access.ID.String(),
		AccessedAt: access.AccessedAt.Format(time.RFC3339),
		IPAddress:  access.IPAddress,
		UserAgent:  access.UserAgent,
		UserID:     userID,
		Action:     access.Action.String(),
	}
}

// ToShareLinkAccessListResponse はアクセス履歴リストレスポンスに変換します
func ToShareLinkAccessListResponse(accesses []*entity.ShareLinkAccess, total int) ShareLinkAccessListResponse {
	items := make([]ShareLinkAccessHistoryResponse, len(accesses))
	for i, a := range accesses {
		items[i] = ToShareLinkAccessHistoryResponse(a)
	}
	return ShareLinkAccessListResponse{
		Items: items,
		Total: total,
	}
}

// ToShareDownloadResponse はダウンロード出力からレスポンスに変換します
func ToShareDownloadResponse(output *sharingqry.GetDownloadViaShareOutput) ShareDownloadResponse {
	return ShareDownloadResponse{
		PresignedURL: output.PresignedURL,
		FileName:     output.FileName,
		FileSize:     output.FileSize,
		MimeType:     output.MimeType,
		ExpiresAt:    output.ExpiresAt.Format(time.RFC3339),
	}
}
