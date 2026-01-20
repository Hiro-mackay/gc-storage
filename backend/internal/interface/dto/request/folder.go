package request

// CreateFolderRequest はフォルダ作成リクエストです
type CreateFolderRequest struct {
	Name     string  `json:"name" validate:"required,min=1,max=255"`
	ParentID *string `json:"parentId"`
}

// RenameFolderRequest はフォルダ名変更リクエストです
type RenameFolderRequest struct {
	Name string `json:"name" validate:"required,min=1,max=255"`
}

// MoveFolderRequest はフォルダ移動リクエストです
type MoveFolderRequest struct {
	NewParentID *string `json:"newParentId"`
}
