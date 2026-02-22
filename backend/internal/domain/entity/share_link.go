package entity

import (
	"errors"
	"time"

	"github.com/google/uuid"

	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/authz"
	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/valueobject"
)

var (
	ErrShareLinkExpired          = errors.New("share link has expired")
	ErrShareLinkRevoked          = errors.New("share link has been revoked")
	ErrShareLinkMaxAccessReached = errors.New("share link has reached maximum access count")
	ErrShareLinkInvalidPassword  = errors.New("invalid share link password")
	ErrShareLinkNotActive        = errors.New("share link is not active")
)

// ShareLink は共有リンクエンティティ
type ShareLink struct {
	ID             uuid.UUID
	Token          valueobject.ShareToken
	ResourceType   authz.ResourceType
	ResourceID     uuid.UUID
	CreatedBy      uuid.UUID
	Permission     valueobject.SharePermission
	PasswordHash   string
	ExpiresAt      *time.Time
	MaxAccessCount *int
	AccessCount    int
	Status         valueobject.ShareLinkStatus
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// NewShareLink は新しい共有リンクを作成します
func NewShareLink(
	resourceType authz.ResourceType,
	resourceID uuid.UUID,
	createdBy uuid.UUID,
	permission valueobject.SharePermission,
	passwordHash string,
	expiresAt *time.Time,
	maxAccessCount *int,
) (*ShareLink, error) {
	token, err := valueobject.NewShareToken()
	if err != nil {
		return nil, err
	}

	now := time.Now()
	return &ShareLink{
		ID:             uuid.New(),
		Token:          token,
		ResourceType:   resourceType,
		ResourceID:     resourceID,
		CreatedBy:      createdBy,
		Permission:     permission,
		PasswordHash:   passwordHash,
		ExpiresAt:      expiresAt,
		MaxAccessCount: maxAccessCount,
		AccessCount:    0,
		Status:         valueobject.ShareLinkStatusActive,
		CreatedAt:      now,
		UpdatedAt:      now,
	}, nil
}

// ReconstructShareLink はDBから共有リンクを復元します
func ReconstructShareLink(
	id uuid.UUID,
	token valueobject.ShareToken,
	resourceType authz.ResourceType,
	resourceID uuid.UUID,
	createdBy uuid.UUID,
	permission valueobject.SharePermission,
	passwordHash string,
	expiresAt *time.Time,
	maxAccessCount *int,
	accessCount int,
	status valueobject.ShareLinkStatus,
	createdAt time.Time,
	updatedAt time.Time,
) *ShareLink {
	return &ShareLink{
		ID:             id,
		Token:          token,
		ResourceType:   resourceType,
		ResourceID:     resourceID,
		CreatedBy:      createdBy,
		Permission:     permission,
		PasswordHash:   passwordHash,
		ExpiresAt:      expiresAt,
		MaxAccessCount: maxAccessCount,
		AccessCount:    accessCount,
		Status:         status,
		CreatedAt:      createdAt,
		UpdatedAt:      updatedAt,
	}
}

// IsActive はアクティブ状態かを判定します
func (s *ShareLink) IsActive() bool {
	return s.Status.IsActive()
}

// IsExpired は期限切れかを判定します
func (s *ShareLink) IsExpired() bool {
	if s.ExpiresAt == nil {
		return false
	}
	return time.Now().After(*s.ExpiresAt)
}

// HasReachedMaxAccess は最大アクセス数に達しているかを判定します
func (s *ShareLink) HasReachedMaxAccess() bool {
	if s.MaxAccessCount == nil {
		return false
	}
	return s.AccessCount >= *s.MaxAccessCount
}

// RequiresPassword はパスワードが必要かを判定します
func (s *ShareLink) RequiresPassword() bool {
	return s.PasswordHash != ""
}

// ValidatePassword は提供されたパスワードを検証します
// compare関数はパスワードハッシュと平文パスワードを受け取り、一致しない場合にエラーを返します
// ドメイン層の純粋性を保つため、実際のハッシュ比較ロジックは外部から注入します
func (s *ShareLink) ValidatePassword(password string, compare func(hash, password string) error) error {
	if !s.RequiresPassword() {
		return nil
	}
	if err := compare(s.PasswordHash, password); err != nil {
		return ErrShareLinkInvalidPassword
	}
	return nil
}

// CanAccess はアクセス可能かを判定します
func (s *ShareLink) CanAccess() error {
	if !s.IsActive() {
		if s.Status.IsRevoked() {
			return ErrShareLinkRevoked
		}
		if s.Status.IsExpired() {
			return ErrShareLinkExpired
		}
		return ErrShareLinkNotActive
	}
	if s.IsExpired() {
		return ErrShareLinkExpired
	}
	if s.HasReachedMaxAccess() {
		return ErrShareLinkMaxAccessReached
	}
	return nil
}

// IncrementAccessCount はアクセスカウントを増やします
func (s *ShareLink) IncrementAccessCount() {
	s.AccessCount++
	s.UpdatedAt = time.Now()
}

// Revoke は共有リンクを無効化します
func (s *ShareLink) Revoke() {
	s.Status = valueobject.ShareLinkStatusRevoked
	s.UpdatedAt = time.Now()
}

// MarkExpired は期限切れにします
func (s *ShareLink) MarkExpired() {
	if s.IsActive() {
		s.Status = valueobject.ShareLinkStatusExpired
		s.UpdatedAt = time.Now()
	}
}

// UpdateExpiry は有効期限を更新します
func (s *ShareLink) UpdateExpiry(expiresAt *time.Time) {
	s.ExpiresAt = expiresAt
	s.UpdatedAt = time.Now()
}

// UpdateMaxAccessCount は最大アクセス数を更新します
func (s *ShareLink) UpdateMaxAccessCount(maxAccessCount *int) {
	s.MaxAccessCount = maxAccessCount
	s.UpdatedAt = time.Now()
}

// UpdatePassword はパスワードを更新します
func (s *ShareLink) UpdatePassword(passwordHash string) {
	s.PasswordHash = passwordHash
	s.UpdatedAt = time.Now()
}

// CanDownload はダウンロード可能かを判定します
func (s *ShareLink) CanDownload() bool {
	return s.Permission.CanDownload()
}

// CanUpload はアップロード可能かを判定します
func (s *ShareLink) CanUpload() bool {
	return s.Permission.CanUpload()
}

// IsCreatedBy は指定ユーザーが作成者かを判定します
func (s *ShareLink) IsCreatedBy(userID uuid.UUID) bool {
	return s.CreatedBy == userID
}

// IsForResource は指定リソースの共有リンクかを判定します
func (s *ShareLink) IsForResource(resourceType authz.ResourceType, resourceID uuid.UUID) bool {
	return s.ResourceType == resourceType && s.ResourceID == resourceID
}
