package handler

import (
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/valueobject"
	"github.com/Hiro-mackay/gc-storage/backend/internal/interface/dto/request"
	"github.com/Hiro-mackay/gc-storage/backend/internal/interface/dto/response"
	"github.com/Hiro-mackay/gc-storage/backend/internal/interface/middleware"
	"github.com/Hiro-mackay/gc-storage/backend/internal/interface/presenter"
	storagecmd "github.com/Hiro-mackay/gc-storage/backend/internal/usecase/storage/command"
	storageqry "github.com/Hiro-mackay/gc-storage/backend/internal/usecase/storage/query"
	"github.com/Hiro-mackay/gc-storage/backend/pkg/apperror"
)

// FolderHandler はフォルダ関連のHTTPハンドラーです
type FolderHandler struct {
	// Commands
	createFolderCommand *storagecmd.CreateFolderCommand
	renameFolderCommand *storagecmd.RenameFolderCommand
	moveFolderCommand   *storagecmd.MoveFolderCommand
	deleteFolderCommand *storagecmd.DeleteFolderCommand

	// Queries
	getFolderQuery          *storageqry.GetFolderQuery
	listFolderContentsQuery *storageqry.ListFolderContentsQuery
	getAncestorsQuery       *storageqry.GetAncestorsQuery
}

// NewFolderHandler は新しいFolderHandlerを作成します
func NewFolderHandler(
	createFolderCommand *storagecmd.CreateFolderCommand,
	renameFolderCommand *storagecmd.RenameFolderCommand,
	moveFolderCommand *storagecmd.MoveFolderCommand,
	deleteFolderCommand *storagecmd.DeleteFolderCommand,
	getFolderQuery *storageqry.GetFolderQuery,
	listFolderContentsQuery *storageqry.ListFolderContentsQuery,
	getAncestorsQuery *storageqry.GetAncestorsQuery,
) *FolderHandler {
	return &FolderHandler{
		createFolderCommand:     createFolderCommand,
		renameFolderCommand:     renameFolderCommand,
		moveFolderCommand:       moveFolderCommand,
		deleteFolderCommand:     deleteFolderCommand,
		getFolderQuery:          getFolderQuery,
		listFolderContentsQuery: listFolderContentsQuery,
		getAncestorsQuery:       getAncestorsQuery,
	}
}

// CreateFolder はフォルダを作成します
// POST /api/v1/folders
func (h *FolderHandler) CreateFolder(c echo.Context) error {
	claims := middleware.GetAccessClaims(c)
	if claims == nil {
		return apperror.NewUnauthorizedError("invalid token")
	}

	var req request.CreateFolderRequest
	if err := c.Bind(&req); err != nil {
		return apperror.NewValidationError("invalid request body", nil)
	}
	if err := c.Validate(&req); err != nil {
		return err
	}

	var parentID *uuid.UUID
	if req.ParentID != nil {
		id, err := uuid.Parse(*req.ParentID)
		if err != nil {
			return apperror.NewValidationError("invalid parent ID", nil)
		}
		parentID = &id
	}

	output, err := h.createFolderCommand.Execute(c.Request().Context(), storagecmd.CreateFolderInput{
		Name:      req.Name,
		ParentID:  parentID,
		OwnerID:   claims.UserID,
		OwnerType: valueobject.OwnerTypeUser,
	})
	if err != nil {
		return err
	}

	return presenter.Created(c, response.ToFolderResponse(output.Folder))
}

// GetFolder はフォルダを取得します
// GET /api/v1/folders/:id
func (h *FolderHandler) GetFolder(c echo.Context) error {
	claims := middleware.GetAccessClaims(c)
	if claims == nil {
		return apperror.NewUnauthorizedError("invalid token")
	}

	folderID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return apperror.NewValidationError("invalid folder ID", nil)
	}

	output, err := h.getFolderQuery.Execute(c.Request().Context(), storageqry.GetFolderInput{
		FolderID: folderID,
		UserID:   claims.UserID,
	})
	if err != nil {
		return err
	}

	return presenter.OK(c, response.ToFolderResponse(output.Folder))
}

// ListFolderContents はフォルダ内容一覧を取得します
// GET /api/v1/folders/:id/contents または GET /api/v1/folders/root/contents
func (h *FolderHandler) ListFolderContents(c echo.Context) error {
	claims := middleware.GetAccessClaims(c)
	if claims == nil {
		return apperror.NewUnauthorizedError("invalid token")
	}

	var folderID *uuid.UUID
	idParam := c.Param("id")
	if idParam != "" && idParam != "root" {
		id, err := uuid.Parse(idParam)
		if err != nil {
			return apperror.NewValidationError("invalid folder ID", nil)
		}
		folderID = &id
	}

	output, err := h.listFolderContentsQuery.Execute(c.Request().Context(), storageqry.ListFolderContentsInput{
		FolderID:  folderID,
		OwnerID:   claims.UserID,
		OwnerType: valueobject.OwnerTypeUser,
		UserID:    claims.UserID,
	})
	if err != nil {
		return err
	}

	resp := response.FolderContentsResponse{
		Folders: response.ToFolderListResponse(output.Folders),
		Files:   response.ToFileListResponse(output.Files),
	}
	if output.Folder != nil {
		folder := response.ToFolderResponse(output.Folder)
		resp.Folder = &folder
	}

	return presenter.OK(c, resp)
}

// GetAncestors はフォルダの祖先一覧を取得します（パンくずリスト用）
// GET /api/v1/folders/:id/ancestors
func (h *FolderHandler) GetAncestors(c echo.Context) error {
	claims := middleware.GetAccessClaims(c)
	if claims == nil {
		return apperror.NewUnauthorizedError("invalid token")
	}

	folderID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return apperror.NewValidationError("invalid folder ID", nil)
	}

	output, err := h.getAncestorsQuery.Execute(c.Request().Context(), storageqry.GetAncestorsInput{
		FolderID: folderID,
		UserID:   claims.UserID,
	})
	if err != nil {
		return err
	}

	return presenter.OK(c, response.ToBreadcrumbResponse(output.Ancestors))
}

// RenameFolder はフォルダ名を変更します
// PATCH /api/v1/folders/:id/rename
func (h *FolderHandler) RenameFolder(c echo.Context) error {
	claims := middleware.GetAccessClaims(c)
	if claims == nil {
		return apperror.NewUnauthorizedError("invalid token")
	}

	folderID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return apperror.NewValidationError("invalid folder ID", nil)
	}

	var req request.RenameFolderRequest
	if err := c.Bind(&req); err != nil {
		return apperror.NewValidationError("invalid request body", nil)
	}
	if err := c.Validate(&req); err != nil {
		return err
	}

	output, err := h.renameFolderCommand.Execute(c.Request().Context(), storagecmd.RenameFolderInput{
		FolderID: folderID,
		NewName:  req.Name,
		UserID:   claims.UserID,
	})
	if err != nil {
		return err
	}

	return presenter.OK(c, response.ToFolderResponse(output.Folder))
}

// MoveFolder はフォルダを移動します
// PATCH /api/v1/folders/:id/move
func (h *FolderHandler) MoveFolder(c echo.Context) error {
	claims := middleware.GetAccessClaims(c)
	if claims == nil {
		return apperror.NewUnauthorizedError("invalid token")
	}

	folderID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return apperror.NewValidationError("invalid folder ID", nil)
	}

	var req request.MoveFolderRequest
	if err := c.Bind(&req); err != nil {
		return apperror.NewValidationError("invalid request body", nil)
	}

	var newParentID *uuid.UUID
	if req.NewParentID != nil {
		id, err := uuid.Parse(*req.NewParentID)
		if err != nil {
			return apperror.NewValidationError("invalid new parent ID", nil)
		}
		newParentID = &id
	}

	output, err := h.moveFolderCommand.Execute(c.Request().Context(), storagecmd.MoveFolderInput{
		FolderID:    folderID,
		NewParentID: newParentID,
		UserID:      claims.UserID,
	})
	if err != nil {
		return err
	}

	return presenter.OK(c, response.ToFolderResponse(output.Folder))
}

// DeleteFolder はフォルダを削除します
// DELETE /api/v1/folders/:id
func (h *FolderHandler) DeleteFolder(c echo.Context) error {
	claims := middleware.GetAccessClaims(c)
	if claims == nil {
		return apperror.NewUnauthorizedError("invalid token")
	}

	folderID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return apperror.NewValidationError("invalid folder ID", nil)
	}

	_, err = h.deleteFolderCommand.Execute(c.Request().Context(), storagecmd.DeleteFolderInput{
		FolderID: folderID,
		UserID:   claims.UserID,
	})
	if err != nil {
		return err
	}

	return presenter.NoContent(c)
}
