package query

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/entity"
	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/repository"
	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/valueobject"
	"github.com/Hiro-mackay/gc-storage/backend/pkg/apperror"
)

// GetUploadStatusInput はアップロード状態取得の入力を定義します
type GetUploadStatusInput struct {
	SessionID uuid.UUID
	UserID    uuid.UUID
}

// GetUploadStatusOutput はアップロード状態取得の出力を定義します
type GetUploadStatusOutput struct {
	SessionID     uuid.UUID
	FileID        uuid.UUID
	Status        entity.UploadSessionStatus
	IsMultipart   bool
	TotalParts    int
	UploadedParts int
	ExpiresAt     time.Time
	IsExpired     bool
}

// GetUploadStatusQuery はアップロード状態取得クエリです
type GetUploadStatusQuery struct {
	uploadSessionRepo repository.UploadSessionRepository
}

// NewGetUploadStatusQuery は新しいGetUploadStatusQueryを作成します
func NewGetUploadStatusQuery(uploadSessionRepo repository.UploadSessionRepository) *GetUploadStatusQuery {
	return &GetUploadStatusQuery{
		uploadSessionRepo: uploadSessionRepo,
	}
}

// Execute はアップロード状態を取得します
func (q *GetUploadStatusQuery) Execute(ctx context.Context, input GetUploadStatusInput) (*GetUploadStatusOutput, error) {
	// 1. セッション取得
	session, err := q.uploadSessionRepo.FindByID(ctx, input.SessionID)
	if err != nil {
		return nil, err
	}

	// 2. 権限チェック（ユーザー所有の場合のみ）
	if session.OwnerType == valueobject.OwnerTypeUser && session.OwnerID != input.UserID {
		return nil, apperror.NewForbiddenError("not authorized to view this upload session")
	}

	return &GetUploadStatusOutput{
		SessionID:     session.ID,
		FileID:        session.FileID,
		Status:        session.Status,
		IsMultipart:   session.IsMultipart,
		TotalParts:    session.TotalParts,
		UploadedParts: session.UploadedParts,
		ExpiresAt:     session.ExpiresAt,
		IsExpired:     session.IsExpired(),
	}, nil
}
