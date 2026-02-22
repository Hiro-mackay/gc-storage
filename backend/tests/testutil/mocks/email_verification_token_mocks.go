package mocks

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"

	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/entity"
)

// MockEmailVerificationTokenRepository is a mock implementation of repository.EmailVerificationTokenRepository
type MockEmailVerificationTokenRepository struct {
	mock.Mock
}

func NewMockEmailVerificationTokenRepository(t *testing.T) *MockEmailVerificationTokenRepository {
	m := &MockEmailVerificationTokenRepository{}
	m.Mock.Test(t)
	t.Cleanup(func() { m.AssertExpectations(t) })
	return m
}

func (m *MockEmailVerificationTokenRepository) Create(ctx context.Context, token *entity.EmailVerificationToken) error {
	args := m.Called(ctx, token)
	return args.Error(0)
}

func (m *MockEmailVerificationTokenRepository) FindByToken(ctx context.Context, token string) (*entity.EmailVerificationToken, error) {
	args := m.Called(ctx, token)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.EmailVerificationToken), args.Error(1)
}

func (m *MockEmailVerificationTokenRepository) FindByUserID(ctx context.Context, userID uuid.UUID) (*entity.EmailVerificationToken, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.EmailVerificationToken), args.Error(1)
}

func (m *MockEmailVerificationTokenRepository) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockEmailVerificationTokenRepository) DeleteByUserID(ctx context.Context, userID uuid.UUID) error {
	args := m.Called(ctx, userID)
	return args.Error(0)
}

func (m *MockEmailVerificationTokenRepository) DeleteExpired(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}
