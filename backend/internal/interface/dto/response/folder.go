package response

import (
	"time"

	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/entity"
)

// FolderResponse はフォルダレスポンスです
type FolderResponse struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	ParentID  *string   `json:"parentId"`
	OwnerID   string    `json:"ownerId"`
	Depth     int       `json:"depth"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// FolderContentsResponse はフォルダ内容一覧レスポンスです
type FolderContentsResponse struct {
	Folder  *FolderResponse  `json:"folder,omitempty"`
	Folders []FolderResponse `json:"folders"`
	Files   []FileResponse   `json:"files"`
}

// BreadcrumbItem はパンくずリストアイテムです
type BreadcrumbItem struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// BreadcrumbResponse はパンくずリストレスポンスです
type BreadcrumbResponse struct {
	Items []BreadcrumbItem `json:"items"`
}

// ToFolderResponse はエンティティからレスポンスに変換します
func ToFolderResponse(folder *entity.Folder) FolderResponse {
	var parentID *string
	if folder.ParentID != nil {
		id := folder.ParentID.String()
		parentID = &id
	}

	return FolderResponse{
		ID:        folder.ID.String(),
		Name:      folder.Name.String(),
		ParentID:  parentID,
		OwnerID:   folder.OwnerID.String(),
		Depth:     folder.Depth,
		Status:    string(folder.Status),
		CreatedAt: folder.CreatedAt,
		UpdatedAt: folder.UpdatedAt,
	}
}

// ToFolderListResponse はエンティティリストからレスポンスリストに変換します
func ToFolderListResponse(folders []*entity.Folder) []FolderResponse {
	responses := make([]FolderResponse, len(folders))
	for i, f := range folders {
		responses[i] = ToFolderResponse(f)
	}
	return responses
}

// ToBreadcrumbResponse はエンティティリストからパンくずリストレスポンスに変換します
func ToBreadcrumbResponse(folders []*entity.Folder) BreadcrumbResponse {
	items := make([]BreadcrumbItem, len(folders))
	for i, f := range folders {
		items[i] = BreadcrumbItem{
			ID:   f.ID.String(),
			Name: f.Name.String(),
		}
	}
	return BreadcrumbResponse{Items: items}
}
