package entity

import (
	"testing"

	"github.com/google/uuid"
)

func TestNewFolderPath_SetsFields(t *testing.T) {
	ancestorID := uuid.New()
	descendantID := uuid.New()

	fp := NewFolderPath(ancestorID, descendantID, 3)

	if fp.AncestorID != ancestorID {
		t.Errorf("expected AncestorID %v, got %v", ancestorID, fp.AncestorID)
	}
	if fp.DescendantID != descendantID {
		t.Errorf("expected DescendantID %v, got %v", descendantID, fp.DescendantID)
	}
	if fp.PathLength != 3 {
		t.Errorf("expected PathLength 3, got %v", fp.PathLength)
	}
}

func TestNewFolderPath_CreatedAtIsSet(t *testing.T) {
	fp := NewFolderPath(uuid.New(), uuid.New(), 1)

	if fp.CreatedAt.IsZero() {
		t.Error("CreatedAt should be set")
	}
}

func TestNewSelfReference_SetsAncestorAndDescendantToSameID(t *testing.T) {
	folderID := uuid.New()

	fp := NewSelfReference(folderID)

	if fp.AncestorID != folderID {
		t.Errorf("expected AncestorID %v, got %v", folderID, fp.AncestorID)
	}
	if fp.DescendantID != folderID {
		t.Errorf("expected DescendantID %v, got %v", folderID, fp.DescendantID)
	}
}

func TestNewSelfReference_SetsPathLengthZero(t *testing.T) {
	fp := NewSelfReference(uuid.New())

	if fp.PathLength != 0 {
		t.Errorf("expected PathLength 0, got %v", fp.PathLength)
	}
}

func TestFolderPath_IsSelfReference_SameIDAndZeroLength_ReturnsTrue(t *testing.T) {
	folderID := uuid.New()
	fp := NewSelfReference(folderID)

	if !fp.IsSelfReference() {
		t.Error("IsSelfReference should return true for self-reference entry")
	}
}

func TestFolderPath_IsSelfReference_DifferentIDs_ReturnsFalse(t *testing.T) {
	fp := NewFolderPath(uuid.New(), uuid.New(), 1)

	if fp.IsSelfReference() {
		t.Error("IsSelfReference should return false when ancestor and descendant differ")
	}
}

func TestFolderPath_IsSelfReference_SameIDNonZeroLength_ReturnsFalse(t *testing.T) {
	folderID := uuid.New()
	fp := NewFolderPath(folderID, folderID, 1)

	if fp.IsSelfReference() {
		t.Error("IsSelfReference should return false when PathLength is non-zero")
	}
}

func TestBuildAncestorPaths_EmptyParentPaths_ReturnsSelfReferenceOnly(t *testing.T) {
	folderID := uuid.New()

	paths := BuildAncestorPaths(folderID, []*FolderPath{})

	if len(paths) != 1 {
		t.Fatalf("expected 1 path, got %d", len(paths))
	}
	if !paths[0].IsSelfReference() {
		t.Error("single path should be self-reference")
	}
	if paths[0].DescendantID != folderID {
		t.Errorf("expected DescendantID %v, got %v", folderID, paths[0].DescendantID)
	}
}

func TestBuildAncestorPaths_WithParentPaths_IncludesSelfReference(t *testing.T) {
	folderID := uuid.New()
	parentID := uuid.New()
	parentPaths := []*FolderPath{NewSelfReference(parentID)}

	paths := BuildAncestorPaths(folderID, parentPaths)

	hasSelfRef := false
	for _, p := range paths {
		if p.AncestorID == folderID && p.DescendantID == folderID {
			hasSelfRef = true
		}
	}
	if !hasSelfRef {
		t.Error("BuildAncestorPaths should include self-reference for folderID")
	}
}

func TestBuildAncestorPaths_WithParentPaths_IncrementsPathLength(t *testing.T) {
	folderID := uuid.New()
	grandParentID := uuid.New()
	parentID := uuid.New()

	parentPaths := []*FolderPath{
		NewSelfReference(parentID),
		NewFolderPath(grandParentID, parentID, 1),
	}

	paths := BuildAncestorPaths(folderID, parentPaths)

	// Expected: self-ref (0), parent->folder (1), grandparent->folder (2)
	if len(paths) != 3 {
		t.Fatalf("expected 3 paths, got %d", len(paths))
	}
}

func TestBuildAncestorPaths_WithParentPaths_SetsCorrectDescendantID(t *testing.T) {
	folderID := uuid.New()
	parentID := uuid.New()
	parentPaths := []*FolderPath{NewSelfReference(parentID)}

	paths := BuildAncestorPaths(folderID, parentPaths)

	for _, p := range paths {
		if p.DescendantID != folderID {
			t.Errorf("all paths should have DescendantID %v, got %v", folderID, p.DescendantID)
		}
	}
}

func TestBuildAncestorPaths_WithParentPaths_IncrementedPathLengthByOne(t *testing.T) {
	folderID := uuid.New()
	parentID := uuid.New()
	parentPaths := []*FolderPath{
		{AncestorID: parentID, DescendantID: parentID, PathLength: 0},
	}

	paths := BuildAncestorPaths(folderID, parentPaths)

	// Find the path from parentID to folderID
	var parentToFolder *FolderPath
	for _, p := range paths {
		if p.AncestorID == parentID {
			parentToFolder = p
		}
	}
	if parentToFolder == nil {
		t.Fatal("expected path from parentID to folderID")
	}
	if parentToFolder.PathLength != 1 {
		t.Errorf("expected PathLength 1, got %v", parentToFolder.PathLength)
	}
}
