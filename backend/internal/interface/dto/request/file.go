package request

// InitiateUploadRequest はアップロード開始リクエストです
type InitiateUploadRequest struct {
	FolderID *string `json:"folderId"`
	FileName string  `json:"fileName" validate:"required,min=1,max=255"`
	MimeType string  `json:"mimeType" validate:"required"`
	Size     int64   `json:"size" validate:"required,min=1"`
}

// CompleteUploadRequest はアップロード完了リクエストです（Webhook用）
type CompleteUploadRequest struct {
	StorageKey     string `json:"storageKey" validate:"required"`
	MinioVersionID string `json:"minioVersionId" validate:"required"`
	Size           int64  `json:"size" validate:"required"`
	ETag           string `json:"etag" validate:"required"`
}

// RenameFileRequest はファイル名変更リクエストです
type RenameFileRequest struct {
	Name string `json:"name" validate:"required,min=1,max=255"`
}

// MoveFileRequest はファイル移動リクエストです
type MoveFileRequest struct {
	NewFolderID *string `json:"newFolderId"`
}

// RestoreFileRequest はファイル復元リクエストです
type RestoreFileRequest struct {
	RestoreFolderID *string `json:"restoreFolderId"`
}
