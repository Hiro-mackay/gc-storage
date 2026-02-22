package mocks

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"

	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/entity"
)

// MockPasswordResetTokenRepository is a mock implementation of repository.PasswordResetTokenRepository
type MockPasswordResetTokenRepository struct {
	mock.Mock
}

func NewMockPasswordResetTokenRepository(t *testing.T) *MockPasswordResetTokenRepository {
	m := &MockPasswordResetTokenRepository{}
	m.Mock.Test(t)
	t.Cleanup(func() { m.AssertExpectations(t) })
	return m
}

func (m *MockPasswordResetTokenRepository) Create(ctx context.Context, token *entity.PasswordResetToken) error {
	args := m.Called(ctx, token)
	return args.Error(0)
}

func (m *MockPasswordResetTokenRepository) FindByToken(ctx context.Context, token string) (*entity.PasswordResetToken, error) {
	args := m.Called(ctx, token)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.PasswordResetToken), args.Error(1)
}

func (m *MockPasswordResetTokenRepository) MarkAsUsed(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockPasswordResetTokenRepository) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockPasswordResetTokenRepository) DeleteByUserID(ctx context.Context, userID uuid.UUID) error {
	args := m.Called(ctx, userID)
	return args.Error(0)
}

func (m *MockPasswordResetTokenRepository) DeleteExpired(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}
