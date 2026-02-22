package service

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/entity"
	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/valueobject"
)

func newTestFolder(depth int) *entity.Folder {
	fn, err := valueobject.NewFolderName("test")
	if err != nil {
		panic(err)
	}
	f, err := entity.NewFolder(fn, nil, uuid.New(), depth)
	if err != nil {
		panic(err)
	}
	return f
}

func TestFolderHierarchyService_CalculateNewDepth_NilParent_ReturnsZero(t *testing.T) {
	svc := NewFolderHierarchyService()

	depth := svc.CalculateNewDepth(nil)

	if depth != 0 {
		t.Errorf("expected depth 0, got %d", depth)
	}
}

func TestFolderHierarchyService_CalculateNewDepth_ParentAtDepthZero_ReturnsOne(t *testing.T) {
	svc := NewFolderHierarchyService()
	parent := newTestFolder(0)

	depth := svc.CalculateNewDepth(parent)

	if depth != 1 {
		t.Errorf("expected depth 1, got %d", depth)
	}
}

func TestFolderHierarchyService_CalculateNewDepth_ParentAtDepthFive_ReturnsSix(t *testing.T) {
	svc := NewFolderHierarchyService()
	parent := newTestFolder(5)

	depth := svc.CalculateNewDepth(parent)

	if depth != 6 {
		t.Errorf("expected depth 6, got %d", depth)
	}
}

func TestFolderHierarchyService_ValidateMove_ValidMove_ReturnsNil(t *testing.T) {
	svc := NewFolderHierarchyService()
	folder := newTestFolder(0)
	newParentID := uuid.New()

	err := svc.ValidateMove(context.Background(), folder, &newParentID, []uuid.UUID{})

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestFolderHierarchyService_ValidateMove_SelfMove_ReturnsErrFolderCircularMove(t *testing.T) {
	svc := NewFolderHierarchyService()
	folder := newTestFolder(0)
	selfID := folder.ID

	err := svc.ValidateMove(context.Background(), folder, &selfID, []uuid.UUID{})

	if err != entity.ErrFolderCircularMove {
		t.Errorf("expected ErrFolderCircularMove, got: %v", err)
	}
}

func TestFolderHierarchyService_ValidateMove_DescendantParent_ReturnsErrFolderCircularMove(t *testing.T) {
	svc := NewFolderHierarchyService()
	folder := newTestFolder(0)
	descendantID := uuid.New()
	newParentID := descendantID

	err := svc.ValidateMove(context.Background(), folder, &newParentID, []uuid.UUID{descendantID})

	if err != entity.ErrFolderCircularMove {
		t.Errorf("expected ErrFolderCircularMove, got: %v", err)
	}
}

func TestFolderHierarchyService_ValidateMove_NilNewParent_ReturnsNil(t *testing.T) {
	svc := NewFolderHierarchyService()
	folder := newTestFolder(1)

	err := svc.ValidateMove(context.Background(), folder, nil, []uuid.UUID{})

	if err != nil {
		t.Errorf("unexpected error when moving to root: %v", err)
	}
}

func TestFolderHierarchyService_ValidateDepthAfterMove_WithinLimit_ReturnsNil(t *testing.T) {
	svc := NewFolderHierarchyService()
	folder := newTestFolder(0)

	err := svc.ValidateDepthAfterMove(folder, 5, 10)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestFolderHierarchyService_ValidateDepthAfterMove_ExceedsLimit_ReturnsErrFolderMaxDepthExceeded(t *testing.T) {
	svc := NewFolderHierarchyService()
	folder := newTestFolder(0)
	newDepth := 15
	maxDescendantDepth := 6 // 15+6=21 > MaxFolderDepth(20)

	err := svc.ValidateDepthAfterMove(folder, newDepth, maxDescendantDepth)

	if err != entity.ErrFolderMaxDepthExceeded {
		t.Errorf("expected ErrFolderMaxDepthExceeded, got: %v", err)
	}
}

func TestFolderHierarchyService_ValidateDepthAfterMove_ExactlyAtLimit_ReturnsNil(t *testing.T) {
	svc := NewFolderHierarchyService()
	folder := newTestFolder(0)
	// newDepth + maxDescendantDepth == MaxFolderDepth exactly
	newDepth := 10
	maxDescendantDepth := 10 // 10+10=20 == MaxFolderDepth(20)

	err := svc.ValidateDepthAfterMove(folder, newDepth, maxDescendantDepth)

	if err != nil {
		t.Errorf("unexpected error at exact limit: %v", err)
	}
}

func TestFolderHierarchyService_BuildAncestorPaths_EmptyParentPaths_ReturnsSelfRefOnly(t *testing.T) {
	svc := NewFolderHierarchyService()
	folderID := uuid.New()

	paths := svc.BuildAncestorPaths(folderID, []*entity.FolderPath{})

	if len(paths) != 1 {
		t.Fatalf("expected 1 path, got %d", len(paths))
	}
	if paths[0].AncestorID != folderID || paths[0].DescendantID != folderID {
		t.Error("single path should be self-reference")
	}
}

func TestFolderHierarchyService_BuildAncestorPaths_WithParentPaths_ReturnsCorrectCount(t *testing.T) {
	svc := NewFolderHierarchyService()
	folderID := uuid.New()
	parentID := uuid.New()
	grandParentID := uuid.New()

	parentPaths := []*entity.FolderPath{
		entity.NewSelfReference(parentID),
		entity.NewFolderPath(grandParentID, parentID, 1),
	}

	paths := svc.BuildAncestorPaths(folderID, parentPaths)

	// self-ref + parentID->folderID + grandParentID->folderID
	if len(paths) != 3 {
		t.Fatalf("expected 3 paths, got %d", len(paths))
	}
}

func TestFolderHierarchyService_BuildAncestorPaths_AllDescendantIDsMatch(t *testing.T) {
	svc := NewFolderHierarchyService()
	folderID := uuid.New()
	parentID := uuid.New()
	parentPaths := []*entity.FolderPath{entity.NewSelfReference(parentID)}

	paths := svc.BuildAncestorPaths(folderID, parentPaths)

	for _, p := range paths {
		if p.DescendantID != folderID {
			t.Errorf("expected all DescendantIDs to be %v, got %v", folderID, p.DescendantID)
		}
	}
}
