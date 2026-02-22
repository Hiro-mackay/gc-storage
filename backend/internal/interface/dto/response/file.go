package response

import (
	"time"

	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/entity"
	storagecmd "github.com/Hiro-mackay/gc-storage/backend/internal/usecase/storage/command"
	storageqry "github.com/Hiro-mackay/gc-storage/backend/internal/usecase/storage/query"
)

// FileResponse はファイルレスポンスです
type FileResponse struct {
	ID             string    `json:"id"`
	Name           string    `json:"name"`
	MimeType       string    `json:"mimeType"`
	Size           int64     `json:"size"`
	FolderID       string    `json:"folderId"`
	OwnerID        string    `json:"ownerId"`
	CurrentVersion int       `json:"currentVersion"`
	Status         string    `json:"status"`
	CreatedAt      time.Time `json:"createdAt"`
	UpdatedAt      time.Time `json:"updatedAt"`
}

// InitiateUploadResponse はアップロード開始レスポンスです
type InitiateUploadResponse struct {
	SessionID   string              `json:"sessionId"`
	FileID      string              `json:"fileId"`
	IsMultipart bool                `json:"isMultipart"`
	UploadURLs  []UploadURLResponse `json:"uploadUrls"`
	ExpiresAt   time.Time           `json:"expiresAt"`
}

// UploadURLResponse はアップロードURL情報です
type UploadURLResponse struct {
	PartNumber int       `json:"partNumber"`
	URL        string    `json:"url"`
	ExpiresAt  time.Time `json:"expiresAt"`
}

// UploadStatusResponse はアップロード状態レスポンスです
type UploadStatusResponse struct {
	SessionID     string    `json:"sessionId"`
	FileID        string    `json:"fileId"`
	Status        string    `json:"status"`
	IsMultipart   bool      `json:"isMultipart"`
	TotalParts    int       `json:"totalParts"`
	UploadedParts int       `json:"uploadedParts"`
	ExpiresAt     time.Time `json:"expiresAt"`
	IsExpired     bool      `json:"isExpired"`
}

// DownloadURLResponse はダウンロードURLレスポンスです
type DownloadURLResponse struct {
	FileID        string    `json:"fileId"`
	FileName      string    `json:"fileName"`
	MimeType      string    `json:"mimeType"`
	Size          int64     `json:"size"`
	VersionNumber int       `json:"versionNumber"`
	DownloadURL   string    `json:"downloadUrl"`
	ExpiresAt     time.Time `json:"expiresAt"`
}

// FileVersionResponse はファイルバージョンレスポンスです
type FileVersionResponse struct {
	ID            string    `json:"id"`
	VersionNumber int       `json:"versionNumber"`
	Size          int64     `json:"size"`
	Checksum      string    `json:"checksum"`
	UploadedBy    string    `json:"uploadedBy"`
	CreatedAt     time.Time `json:"createdAt"`
	IsLatest      bool      `json:"isLatest"`
}

// FileVersionsResponse はファイルバージョン一覧レスポンスです
type FileVersionsResponse struct {
	FileID   string                `json:"fileId"`
	FileName string                `json:"fileName"`
	Versions []FileVersionResponse `json:"versions"`
}

// TrashItemResponse はゴミ箱アイテムレスポンスです
// Note: OriginalFolderIDは必須。ファイルは必ずフォルダに所属。
type TrashItemResponse struct {
	ID               string    `json:"id"`
	Type             string    `json:"type"`
	OriginalFileID   string    `json:"originalFileId"`
	OriginalFolderID string    `json:"originalFolderId"`
	OriginalPath     string    `json:"originalPath"`
	Name             string    `json:"name"`
	MimeType         string    `json:"mimeType"`
	Size             int64     `json:"size"`
	ArchivedAt       time.Time `json:"archivedAt"`
	ExpiresAt        time.Time `json:"expiresAt"`
	DaysUntilExpiry  int       `json:"daysUntilExpiry"`
}

// TrashListResponse はゴミ箱一覧レスポンスです
type TrashListResponse struct {
	Items      []TrashItemResponse `json:"items"`
	NextCursor *string             `json:"nextCursor"`
}

// CompleteUploadResponse はアップロード完了レスポンスです
type CompleteUploadResponse struct {
	FileID    string `json:"fileId"`
	SessionID string `json:"sessionId"`
	Completed bool   `json:"completed"`
}

// RenameFileResponse はファイル名変更レスポンスです
type RenameFileResponse struct {
	FileID string `json:"fileId"`
	Name   string `json:"name"`
}

// MoveFileResponse はファイル移動レスポンスです
type MoveFileResponse struct {
	FileID   string `json:"fileId"`
	FolderID string `json:"folderId"`
}

// TrashFileResponse はファイルゴミ箱移動レスポンスです
type TrashFileResponse struct {
	ArchivedFileID string    `json:"archivedFileId"`
	ExpiresAt      time.Time `json:"expiresAt"`
}

// RestoreFileResponse はファイル復元レスポンスです
type RestoreFileResponse struct {
	FileID   string `json:"fileId"`
	FolderID string `json:"folderId"`
	Name     string `json:"name"`
}

// EmptyTrashResponse はゴミ箱空にするレスポンスです
type EmptyTrashResponse struct {
	Message      string `json:"message"`
	DeletedCount int    `json:"deletedCount"`
}

// AbortUploadResponse はアップロード中断レスポンスです
type AbortUploadResponse struct {
	SessionID string `json:"sessionId"`
	Aborted   bool   `json:"aborted"`
}

// ToFileResponse はエンティティからレスポンスに変換します
func ToFileResponse(file *entity.File) FileResponse {
	return FileResponse{
		ID:             file.ID.String(),
		Name:           file.Name.String(),
		MimeType:       file.MimeType.String(),
		Size:           file.Size,
		FolderID:       file.FolderID.String(),
		OwnerID:        file.OwnerID.String(),
		CurrentVersion: file.CurrentVersion,
		Status:         string(file.Status),
		CreatedAt:      file.CreatedAt,
		UpdatedAt:      file.UpdatedAt,
	}
}

// ToFileListResponse はエンティティリストからレスポンスリストに変換します
func ToFileListResponse(files []*entity.File) []FileResponse {
	responses := make([]FileResponse, len(files))
	for i, f := range files {
		responses[i] = ToFileResponse(f)
	}
	return responses
}

// ToInitiateUploadResponse はUseCaseの出力からレスポンスに変換します
func ToInitiateUploadResponse(output *storagecmd.InitiateUploadOutput) InitiateUploadResponse {
	urls := make([]UploadURLResponse, len(output.UploadURLs))
	for i, u := range output.UploadURLs {
		urls[i] = UploadURLResponse{
			PartNumber: u.PartNumber,
			URL:        u.URL,
			ExpiresAt:  u.ExpiresAt,
		}
	}

	return InitiateUploadResponse{
		SessionID:   output.SessionID.String(),
		FileID:      output.FileID.String(),
		IsMultipart: output.IsMultipart,
		UploadURLs:  urls,
		ExpiresAt:   output.ExpiresAt,
	}
}

// ToUploadStatusResponse はUseCaseの出力からレスポンスに変換します
func ToUploadStatusResponse(output *storageqry.GetUploadStatusOutput) UploadStatusResponse {
	return UploadStatusResponse{
		SessionID:     output.SessionID.String(),
		FileID:        output.FileID.String(),
		Status:        string(output.Status),
		IsMultipart:   output.IsMultipart,
		TotalParts:    output.TotalParts,
		UploadedParts: output.UploadedParts,
		ExpiresAt:     output.ExpiresAt,
		IsExpired:     output.IsExpired,
	}
}

// ToDownloadURLResponse はUseCaseの出力からレスポンスに変換します
func ToDownloadURLResponse(output *storageqry.GetDownloadURLOutput) DownloadURLResponse {
	return DownloadURLResponse{
		FileID:        output.FileID.String(),
		FileName:      output.FileName,
		MimeType:      output.MimeType,
		Size:          output.Size,
		VersionNumber: output.VersionNumber,
		DownloadURL:   output.DownloadURL,
		ExpiresAt:     output.ExpiresAt,
	}
}

// ToFileVersionsResponse はUseCaseの出力からレスポンスに変換します
func ToFileVersionsResponse(output *storageqry.ListFileVersionsOutput) FileVersionsResponse {
	versions := make([]FileVersionResponse, len(output.Versions))
	for i, v := range output.Versions {
		versions[i] = FileVersionResponse{
			ID:            v.ID.String(),
			VersionNumber: v.VersionNumber,
			Size:          v.Size,
			Checksum:      v.Checksum,
			UploadedBy:    v.UploadedBy.String(),
			CreatedAt:     v.CreatedAt,
			IsLatest:      v.IsLatest,
		}
	}

	return FileVersionsResponse{
		FileID:   output.FileID.String(),
		FileName: output.FileName,
		Versions: versions,
	}
}

// ToTrashListResponse はUseCaseの出力からレスポンスに変換します
func ToTrashListResponse(output *storageqry.ListTrashOutput) TrashListResponse {
	items := make([]TrashItemResponse, len(output.Items))
	for i, item := range output.Items {
		items[i] = TrashItemResponse{
			ID:               item.ID.String(),
			Type:             "file",
			OriginalFileID:   item.OriginalFileID.String(),
			OriginalFolderID: item.OriginalFolderID.String(),
			OriginalPath:     item.OriginalPath,
			Name:             item.Name,
			MimeType:         item.MimeType,
			Size:             item.Size,
			ArchivedAt:       item.ArchivedAt,
			ExpiresAt:        item.ExpiresAt,
			DaysUntilExpiry:  item.DaysUntilExpiry,
		}
	}

	var nextCursor *string
	if output.NextCursor != nil {
		s := output.NextCursor.String()
		nextCursor = &s
	}

	return TrashListResponse{Items: items, NextCursor: nextCursor}
}
