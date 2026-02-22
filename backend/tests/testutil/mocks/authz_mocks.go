package mocks

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"

	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/authz"
)

// MockRelationshipRepository is a mock of authz.RelationshipRepository
type MockRelationshipRepository struct {
	mock.Mock
}

func NewMockRelationshipRepository(t *testing.T) *MockRelationshipRepository {
	m := &MockRelationshipRepository{}
	m.Mock.Test(t)
	t.Cleanup(func() { m.AssertExpectations(t) })
	return m
}

func (m *MockRelationshipRepository) Create(ctx context.Context, rel *authz.Relationship) error {
	args := m.Called(ctx, rel)
	return args.Error(0)
}

func (m *MockRelationshipRepository) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockRelationshipRepository) DeleteByTuple(ctx context.Context, tuple authz.Tuple) error {
	args := m.Called(ctx, tuple)
	return args.Error(0)
}

func (m *MockRelationshipRepository) Exists(ctx context.Context, tuple authz.Tuple) (bool, error) {
	args := m.Called(ctx, tuple)
	return args.Bool(0), args.Error(1)
}

func (m *MockRelationshipRepository) FindSubjects(ctx context.Context, objectType authz.ObjectType, objectID uuid.UUID, relation authz.RelationType) ([]authz.Resource, error) {
	args := m.Called(ctx, objectType, objectID, relation)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]authz.Resource), args.Error(1)
}

func (m *MockRelationshipRepository) FindObjects(ctx context.Context, subjectType authz.SubjectType, subjectID uuid.UUID, relation authz.RelationType, objectType authz.ObjectType) ([]uuid.UUID, error) {
	args := m.Called(ctx, subjectType, subjectID, relation, objectType)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]uuid.UUID), args.Error(1)
}

func (m *MockRelationshipRepository) FindByObject(ctx context.Context, objectType authz.ObjectType, objectID uuid.UUID) ([]*authz.Relationship, error) {
	args := m.Called(ctx, objectType, objectID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*authz.Relationship), args.Error(1)
}

func (m *MockRelationshipRepository) FindBySubject(ctx context.Context, subjectType authz.SubjectType, subjectID uuid.UUID) ([]*authz.Relationship, error) {
	args := m.Called(ctx, subjectType, subjectID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*authz.Relationship), args.Error(1)
}

func (m *MockRelationshipRepository) FindParent(ctx context.Context, objectType authz.ObjectType, objectID uuid.UUID) (*authz.Resource, error) {
	args := m.Called(ctx, objectType, objectID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*authz.Resource), args.Error(1)
}

func (m *MockRelationshipRepository) DeleteByObject(ctx context.Context, objectType authz.ObjectType, objectID uuid.UUID) error {
	args := m.Called(ctx, objectType, objectID)
	return args.Error(0)
}

func (m *MockRelationshipRepository) DeleteBySubject(ctx context.Context, subjectType authz.SubjectType, subjectID uuid.UUID) error {
	args := m.Called(ctx, subjectType, subjectID)
	return args.Error(0)
}
