package entity

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	ErrInvalidAccessAction = errors.New("invalid access action")
)

// AccessAction はアクセスアクションを表す型
type AccessAction string

const (
	AccessActionView     AccessAction = "view"
	AccessActionDownload AccessAction = "download"
	AccessActionUpload   AccessAction = "upload"
)

// IsValid はアクションが有効かを判定します
func (a AccessAction) IsValid() bool {
	switch a {
	case AccessActionView, AccessActionDownload, AccessActionUpload:
		return true
	default:
		return false
	}
}

// String は文字列を返します
func (a AccessAction) String() string {
	return string(a)
}

// ShareLinkAccess は共有リンクアクセスログエンティティ
type ShareLinkAccess struct {
	ID          uuid.UUID
	ShareLinkID uuid.UUID
	AccessedAt  time.Time
	IPAddress   string
	UserAgent   string
	UserID      *uuid.UUID
	Action      AccessAction
}

// NewShareLinkAccess は新しいアクセスログを作成します
func NewShareLinkAccess(
	shareLinkID uuid.UUID,
	ipAddress string,
	userAgent string,
	userID *uuid.UUID,
	action AccessAction,
) (*ShareLinkAccess, error) {
	if !action.IsValid() {
		return nil, ErrInvalidAccessAction
	}

	return &ShareLinkAccess{
		ID:          uuid.New(),
		ShareLinkID: shareLinkID,
		AccessedAt:  time.Now(),
		IPAddress:   ipAddress,
		UserAgent:   userAgent,
		UserID:      userID,
		Action:      action,
	}, nil
}

// ReconstructShareLinkAccess はDBからアクセスログを復元します
func ReconstructShareLinkAccess(
	id uuid.UUID,
	shareLinkID uuid.UUID,
	accessedAt time.Time,
	ipAddress string,
	userAgent string,
	userID *uuid.UUID,
	action AccessAction,
) *ShareLinkAccess {
	return &ShareLinkAccess{
		ID:          id,
		ShareLinkID: shareLinkID,
		AccessedAt:  accessedAt,
		IPAddress:   ipAddress,
		UserAgent:   userAgent,
		UserID:      userID,
		Action:      action,
	}
}

// IsAnonymous は匿名アクセスかを判定します
func (a *ShareLinkAccess) IsAnonymous() bool {
	return a.UserID == nil
}

// IsView は閲覧アクセスかを判定します
func (a *ShareLinkAccess) IsView() bool {
	return a.Action == AccessActionView
}

// IsDownload はダウンロードアクセスかを判定します
func (a *ShareLinkAccess) IsDownload() bool {
	return a.Action == AccessActionDownload
}

// IsUpload はアップロードアクセスかを判定します
func (a *ShareLinkAccess) IsUpload() bool {
	return a.Action == AccessActionUpload
}
