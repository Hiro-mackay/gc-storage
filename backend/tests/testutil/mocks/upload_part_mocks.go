package mocks

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"

	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/entity"
)

// MockUploadPartRepository is a mock of repository.UploadPartRepository
type MockUploadPartRepository struct {
	mock.Mock
}

func NewMockUploadPartRepository(t *testing.T) *MockUploadPartRepository {
	m := &MockUploadPartRepository{}
	m.Mock.Test(t)
	t.Cleanup(func() { m.AssertExpectations(t) })
	return m
}

func (m *MockUploadPartRepository) Create(ctx context.Context, part *entity.UploadPart) error {
	args := m.Called(ctx, part)
	return args.Error(0)
}

func (m *MockUploadPartRepository) FindBySessionID(ctx context.Context, sessionID uuid.UUID) ([]*entity.UploadPart, error) {
	args := m.Called(ctx, sessionID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*entity.UploadPart), args.Error(1)
}

func (m *MockUploadPartRepository) DeleteBySessionID(ctx context.Context, sessionID uuid.UUID) error {
	args := m.Called(ctx, sessionID)
	return args.Error(0)
}
