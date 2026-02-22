package handler

import (
	"github.com/Hiro-mackay/gc-storage/backend/internal/interface/dto/response"
	"github.com/Hiro-mackay/gc-storage/backend/internal/interface/presenter"
)

// swagger:model を使って presenter.Response の interface{} を具体型に置き換える

// ---- Auth ----

// SwaggerRegisterResponse は登録レスポンスのラッパー（実態はLoginResponseと同じ）
type SwaggerRegisterResponse struct {
	Data response.LoginResponse `json:"data"`
	Meta *presenter.Meta        `json:"meta"`
}

// SwaggerLoginResponse は LoginResponse のラッパー
type SwaggerLoginResponse struct {
	Data response.LoginResponse `json:"data"`
	Meta *presenter.Meta        `json:"meta"`
}

// SwaggerLogoutResponse は LogoutResponse のラッパー
type SwaggerLogoutResponse struct {
	Data response.LogoutResponse `json:"data"`
	Meta *presenter.Meta         `json:"meta"`
}

// SwaggerUserResponse は UserResponse のラッパー
type SwaggerUserResponse struct {
	Data response.UserResponse `json:"data"`
	Meta *presenter.Meta       `json:"meta"`
}

// SwaggerVerifyEmailResponse は VerifyEmailResponse のラッパー
type SwaggerVerifyEmailResponse struct {
	Data response.VerifyEmailResponse `json:"data"`
	Meta *presenter.Meta              `json:"meta"`
}

// SwaggerResendEmailVerificationResponse は ResendEmailVerificationResponse のラッパー
type SwaggerResendEmailVerificationResponse struct {
	Data response.ResendEmailVerificationResponse `json:"data"`
	Meta *presenter.Meta                          `json:"meta"`
}

// SwaggerForgotPasswordResponse は ForgotPasswordResponse のラッパー
type SwaggerForgotPasswordResponse struct {
	Data response.ForgotPasswordResponse `json:"data"`
	Meta *presenter.Meta                 `json:"meta"`
}

// SwaggerResetPasswordResponse は ResetPasswordResponse のラッパー
type SwaggerResetPasswordResponse struct {
	Data response.ResetPasswordResponse `json:"data"`
	Meta *presenter.Meta                `json:"meta"`
}

// SwaggerChangePasswordResponse は ChangePasswordResponse のラッパー
type SwaggerChangePasswordResponse struct {
	Data response.ChangePasswordResponse `json:"data"`
	Meta *presenter.Meta                 `json:"meta"`
}

// SwaggerSetPasswordResponse は SetPasswordResponse のラッパー
type SwaggerSetPasswordResponse struct {
	Data response.SetPasswordResponse `json:"data"`
	Meta *presenter.Meta              `json:"meta"`
}

// SwaggerOAuthLoginResponse は OAuthLoginResponse のラッパー
type SwaggerOAuthLoginResponse struct {
	Data response.OAuthLoginResponse `json:"data"`
	Meta *presenter.Meta             `json:"meta"`
}

// ---- Profile ----

// SwaggerProfileResponse は ProfileResponse のラッパー
type SwaggerProfileResponse struct {
	Data response.ProfileResponse `json:"data"`
	Meta *presenter.Meta          `json:"meta"`
}

// SwaggerUpdateProfileResponse は UpdateProfileResponse のラッパー
type SwaggerUpdateProfileResponse struct {
	Data response.UpdateProfileResponse `json:"data"`
	Meta *presenter.Meta                `json:"meta"`
}

// ---- Folder ----

// SwaggerFolderResponse は FolderResponse のラッパー
type SwaggerFolderResponse struct {
	Data response.FolderResponse `json:"data"`
	Meta *presenter.Meta         `json:"meta"`
}

// SwaggerFolderContentsResponse は FolderContentsResponse のラッパー
type SwaggerFolderContentsResponse struct {
	Data response.FolderContentsResponse `json:"data"`
	Meta *presenter.Meta                 `json:"meta"`
}

// SwaggerBreadcrumbResponse は BreadcrumbResponse のラッパー
type SwaggerBreadcrumbResponse struct {
	Data response.BreadcrumbResponse `json:"data"`
	Meta *presenter.Meta             `json:"meta"`
}

// ---- File ----

// SwaggerInitiateUploadResponse は InitiateUploadResponse のラッパー
type SwaggerInitiateUploadResponse struct {
	Data response.InitiateUploadResponse `json:"data"`
	Meta *presenter.Meta                 `json:"meta"`
}

// SwaggerCompleteUploadResponse は CompleteUploadResponse のラッパー
type SwaggerCompleteUploadResponse struct {
	Data response.CompleteUploadResponse `json:"data"`
	Meta *presenter.Meta                 `json:"meta"`
}

// SwaggerUploadStatusResponse は UploadStatusResponse のラッパー
type SwaggerUploadStatusResponse struct {
	Data response.UploadStatusResponse `json:"data"`
	Meta *presenter.Meta               `json:"meta"`
}

// SwaggerDownloadURLResponse は DownloadURLResponse のラッパー
type SwaggerDownloadURLResponse struct {
	Data response.DownloadURLResponse `json:"data"`
	Meta *presenter.Meta              `json:"meta"`
}

// SwaggerFileVersionsResponse は FileVersionsResponse のラッパー
type SwaggerFileVersionsResponse struct {
	Data response.FileVersionsResponse `json:"data"`
	Meta *presenter.Meta               `json:"meta"`
}

// SwaggerRenameFileResponse は RenameFileResponse のラッパー
type SwaggerRenameFileResponse struct {
	Data response.RenameFileResponse `json:"data"`
	Meta *presenter.Meta             `json:"meta"`
}

// SwaggerMoveFileResponse は MoveFileResponse のラッパー
type SwaggerMoveFileResponse struct {
	Data response.MoveFileResponse `json:"data"`
	Meta *presenter.Meta           `json:"meta"`
}

// SwaggerTrashFileResponse は TrashFileResponse のラッパー
type SwaggerTrashFileResponse struct {
	Data response.TrashFileResponse `json:"data"`
	Meta *presenter.Meta            `json:"meta"`
}

// SwaggerTrashListResponse は TrashListResponse のラッパー
type SwaggerTrashListResponse struct {
	Data response.TrashListResponse `json:"data"`
	Meta *presenter.Meta            `json:"meta"`
}

// SwaggerRestoreFileResponse は RestoreFileResponse のラッパー
type SwaggerRestoreFileResponse struct {
	Data response.RestoreFileResponse `json:"data"`
	Meta *presenter.Meta              `json:"meta"`
}

// SwaggerDeletedResponse は削除成功レスポンス (data=null)
type SwaggerDeletedResponse struct {
	Data *struct{}       `json:"data"`
	Meta *presenter.Meta `json:"meta"`
}

// SwaggerEmptyTrashResponse は EmptyTrashResponse のラッパー
type SwaggerEmptyTrashResponse struct {
	Data response.EmptyTrashResponse `json:"data"`
	Meta *presenter.Meta             `json:"meta"`
}

// SwaggerAbortUploadResponse は AbortUploadResponse のラッパー
type SwaggerAbortUploadResponse struct {
	Data response.AbortUploadResponse `json:"data"`
	Meta *presenter.Meta              `json:"meta"`
}

// ---- Group ----

// SwaggerGroupWithMembershipResponse は GroupWithMembershipResponse のラッパー
type SwaggerGroupWithMembershipResponse struct {
	Data response.GroupWithMembershipResponse `json:"data"`
	Meta *presenter.Meta                      `json:"meta"`
}

// SwaggerGroupListResponse は GroupWithMembershipResponse リストのラッパー
type SwaggerGroupListResponse struct {
	Data []response.GroupWithMembershipResponse `json:"data"`
	Meta *presenter.Meta                        `json:"meta"`
}

// SwaggerGroupResponse は GroupResponse のラッパー
type SwaggerGroupResponse struct {
	Data response.GroupResponse `json:"data"`
	Meta *presenter.Meta        `json:"meta"`
}

// SwaggerMemberListResponse は MemberResponse リストのラッパー
type SwaggerMemberListResponse struct {
	Data []response.MemberResponse `json:"data"`
	Meta *presenter.Meta           `json:"meta"`
}

// SwaggerMembershipResponse は MembershipResponse のラッパー
type SwaggerMembershipResponse struct {
	Data response.MembershipResponse `json:"data"`
	Meta *presenter.Meta             `json:"meta"`
}

// SwaggerInvitationResponse は InvitationResponse のラッパー
type SwaggerInvitationResponse struct {
	Data response.InvitationResponse `json:"data"`
	Meta *presenter.Meta             `json:"meta"`
}

// SwaggerInvitationListResponse は InvitationResponse リストのラッパー
type SwaggerInvitationListResponse struct {
	Data []response.InvitationResponse `json:"data"`
	Meta *presenter.Meta               `json:"meta"`
}

// SwaggerPendingInvitationListResponse は PendingInvitationResponse リストのラッパー
type SwaggerPendingInvitationListResponse struct {
	Data []response.PendingInvitationResponse `json:"data"`
	Meta *presenter.Meta                      `json:"meta"`
}

// ---- Permission ----

// SwaggerPermissionGrantResponse は PermissionGrantResponse のラッパー
type SwaggerPermissionGrantResponse struct {
	Data response.PermissionGrantResponse `json:"data"`
	Meta *presenter.Meta                  `json:"meta"`
}

// SwaggerPermissionGrantListResponse は PermissionGrantResponse リストのラッパー
type SwaggerPermissionGrantListResponse struct {
	Data []response.PermissionGrantResponse `json:"data"`
	Meta *presenter.Meta                    `json:"meta"`
}

// SwaggerCheckPermissionResponse は CheckPermissionResponse のラッパー
type SwaggerCheckPermissionResponse struct {
	Data response.CheckPermissionResponse `json:"data"`
	Meta *presenter.Meta                  `json:"meta"`
}

// ---- Share Link ----

// SwaggerShareLinkResponse は ShareLinkResponse のラッパー
type SwaggerShareLinkResponse struct {
	Data response.ShareLinkResponse `json:"data"`
	Meta *presenter.Meta            `json:"meta"`
}

// SwaggerShareLinkListResponse は ShareLinkResponse リストのラッパー
type SwaggerShareLinkListResponse struct {
	Data []response.ShareLinkResponse `json:"data"`
	Meta *presenter.Meta              `json:"meta"`
}

// SwaggerShareLinkInfoResponse は ShareLinkInfoResponse のラッパー
type SwaggerShareLinkInfoResponse struct {
	Data response.ShareLinkInfoResponse `json:"data"`
	Meta *presenter.Meta                `json:"meta"`
}

// SwaggerShareLinkAccessResponse は ShareLinkAccessResponse のラッパー
type SwaggerShareLinkAccessResponse struct {
	Data response.ShareLinkAccessResponse `json:"data"`
	Meta *presenter.Meta                  `json:"meta"`
}

// ---- Error ----

// SwaggerErrorResponse はエラーレスポンス
type SwaggerErrorResponse struct {
	Error SwaggerErrorDetail `json:"error"`
}

// SwaggerErrorDetail はエラー詳細
type SwaggerErrorDetail struct {
	Code    string              `json:"code"`
	Message string              `json:"message"`
	Details []SwaggerFieldError `json:"details,omitempty"`
}

// SwaggerFieldError はバリデーションエラーのフィールド詳細
type SwaggerFieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}
