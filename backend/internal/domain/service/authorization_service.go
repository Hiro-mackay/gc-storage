package service

import (
	"github.com/google/uuid"

	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/entity"
	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/valueobject"
	"github.com/Hiro-mackay/gc-storage/backend/pkg/apperror"
)

// AuthorizationService は認可に関するドメインサービス
type AuthorizationService interface {
	// AuthorizeFolderAccess はフォルダへのアクセス権限を確認します
	AuthorizeFolderAccess(folder *entity.Folder, userID uuid.UUID, action string) error

	// AuthorizeFileAccess はファイルへのアクセス権限を確認します
	AuthorizeFileAccess(file *entity.File, userID uuid.UUID, action string) error
}

// authorizationServiceImpl はAuthorizationServiceの実装
type authorizationServiceImpl struct{}

// NewAuthorizationService は新しいAuthorizationServiceを作成します
func NewAuthorizationService() AuthorizationService {
	return &authorizationServiceImpl{}
}

// AuthorizeFolderAccess はフォルダへのアクセス権限を確認します
// TODO: 将来的にはPBAC/ReBACの権限チェックを実装
func (s *authorizationServiceImpl) AuthorizeFolderAccess(folder *entity.Folder, userID uuid.UUID, action string) error {
	// 現時点では所有者のみアクセス可能
	if !folder.IsOwnedBy(userID, valueobject.OwnerTypeUser) {
		return apperror.NewForbiddenError("not authorized to " + action + " this folder")
	}
	return nil
}

// AuthorizeFileAccess はファイルへのアクセス権限を確認します
// TODO: 将来的にはPBAC/ReBACの権限チェックを実装
func (s *authorizationServiceImpl) AuthorizeFileAccess(file *entity.File, userID uuid.UUID, action string) error {
	// 現時点では所有者のみアクセス可能
	if !file.IsOwnedBy(userID, valueobject.OwnerTypeUser) {
		return apperror.NewForbiddenError("not authorized to " + action + " this file")
	}
	return nil
}
