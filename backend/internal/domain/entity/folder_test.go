package entity

import (
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/valueobject"
)

func newFolderName(name string) valueobject.FolderName {
	fn, _ := valueobject.NewFolderName(name)
	return fn
}

func newRootFolder(createdBy uuid.UUID) *Folder {
	f, _ := NewFolder(newFolderName("root"), nil, createdBy, 0)
	return f
}

func newChildFolder(parentID *uuid.UUID, createdBy uuid.UUID, depth int) *Folder {
	f, _ := NewFolder(newFolderName("child"), parentID, createdBy, depth)
	return f
}

func TestFolder_NewFolder_ValidRootParams_SetsAllFields(t *testing.T) {
	fn := newFolderName("Documents")
	createdBy := uuid.New()

	folder, err := NewFolder(fn, nil, createdBy, 0)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if folder.Name != fn {
		t.Errorf("got name %q, want %q", folder.Name.Value(), fn.Value())
	}
	if folder.OwnerID != createdBy {
		t.Errorf("expected OwnerID %v, got %v", createdBy, folder.OwnerID)
	}
	if folder.CreatedBy != createdBy {
		t.Errorf("expected CreatedBy %v, got %v", createdBy, folder.CreatedBy)
	}
	if folder.Status != FolderStatusActive {
		t.Errorf("expected status %q, got %q", FolderStatusActive, folder.Status)
	}
}

func TestFolder_NewFolder_GeneratesUniqueIDs(t *testing.T) {
	fn := newFolderName("Documents")
	f1, _ := NewFolder(fn, nil, uuid.New(), 0)
	f2, _ := NewFolder(fn, nil, uuid.New(), 0)

	if f1.ID == f2.ID {
		t.Error("NewFolder should generate unique IDs")
	}
}

func TestFolder_NewFolder_MaxDepth_Succeeds(t *testing.T) {
	parentID := uuid.New()
	_, err := NewFolder(newFolderName("deep"), &parentID, uuid.New(), MaxFolderDepth)
	if err != nil {
		t.Fatalf("unexpected error at max depth: %v", err)
	}
}

func TestFolder_NewFolder_ExceedsMaxDepth_ReturnsError(t *testing.T) {
	parentID := uuid.New()
	_, err := NewFolder(newFolderName("deep"), &parentID, uuid.New(), MaxFolderDepth+1)
	if err != ErrFolderMaxDepthExceeded {
		t.Errorf("expected ErrFolderMaxDepthExceeded, got: %v", err)
	}
}

func TestFolder_NewFolder_WithParentID_SetsParentID(t *testing.T) {
	parentID := uuid.New()
	folder, err := NewFolder(newFolderName("child"), &parentID, uuid.New(), 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if folder.ParentID == nil || *folder.ParentID != parentID {
		t.Errorf("expected ParentID %v, got %v", parentID, folder.ParentID)
	}
}

func TestFolder_ReconstructFolder_PreservesAllFields(t *testing.T) {
	id := uuid.New()
	fn := newFolderName("test")
	parentID := uuid.New()
	ownerID := uuid.New()
	createdBy := uuid.New()
	now := time.Now()

	folder := ReconstructFolder(id, fn, &parentID, ownerID, createdBy, 2, FolderStatusActive, now, now)

	if folder.ID != id {
		t.Errorf("expected ID %v, got %v", id, folder.ID)
	}
	if folder.OwnerID != ownerID {
		t.Errorf("expected OwnerID %v, got %v", ownerID, folder.OwnerID)
	}
	if folder.CreatedBy != createdBy {
		t.Errorf("expected CreatedBy %v, got %v", createdBy, folder.CreatedBy)
	}
	if folder.Depth != 2 {
		t.Errorf("expected Depth 2, got %v", folder.Depth)
	}
}

func TestFolder_IsRoot_NilParentID_ReturnsTrue(t *testing.T) {
	folder := newRootFolder(uuid.New())
	if !folder.IsRoot() {
		t.Error("IsRoot should return true when ParentID is nil")
	}
}

func TestFolder_IsRoot_WithParentID_ReturnsFalse(t *testing.T) {
	parentID := uuid.New()
	folder := newChildFolder(&parentID, uuid.New(), 1)
	if folder.IsRoot() {
		t.Error("IsRoot should return false when ParentID is set")
	}
}

func TestFolder_IsOwnedBy_MatchingOwner_ReturnsTrue(t *testing.T) {
	createdBy := uuid.New()
	folder := newRootFolder(createdBy)
	if !folder.IsOwnedBy(createdBy) {
		t.Error("IsOwnedBy should return true for the folder owner")
	}
}

func TestFolder_IsOwnedBy_DifferentOwner_ReturnsFalse(t *testing.T) {
	folder := newRootFolder(uuid.New())
	if folder.IsOwnedBy(uuid.New()) {
		t.Error("IsOwnedBy should return false for a different user")
	}
}

func TestFolder_IsCreatedBy_MatchingCreator_ReturnsTrue(t *testing.T) {
	createdBy := uuid.New()
	folder := newRootFolder(createdBy)
	if !folder.IsCreatedBy(createdBy) {
		t.Error("IsCreatedBy should return true for the folder creator")
	}
}

func TestFolder_IsCreatedBy_DifferentUser_ReturnsFalse(t *testing.T) {
	folder := newRootFolder(uuid.New())
	if folder.IsCreatedBy(uuid.New()) {
		t.Error("IsCreatedBy should return false for a different user")
	}
}

func TestFolder_IsActive_ActiveStatus_ReturnsTrue(t *testing.T) {
	if !newRootFolder(uuid.New()).IsActive() {
		t.Error("IsActive should return true for active folder")
	}
}

func TestFolder_IsActive_NonActiveStatus_ReturnsFalse(t *testing.T) {
	folder := ReconstructFolder(uuid.New(), newFolderName("test"), nil, uuid.New(), uuid.New(), 0, FolderStatus("deleted"), time.Now(), time.Now())
	if folder.IsActive() {
		t.Error("IsActive should return false for non-active folder")
	}
}

func TestFolder_EqualsName_SameName_ReturnsTrue(t *testing.T) {
	folder, _ := NewFolder(newFolderName("Documents"), nil, uuid.New(), 0)
	if !folder.EqualsName(newFolderName("Documents")) {
		t.Error("EqualsName should return true for same name")
	}
}

func TestFolder_EqualsName_DifferentName_ReturnsFalse(t *testing.T) {
	folder, _ := NewFolder(newFolderName("Documents"), nil, uuid.New(), 0)
	if folder.EqualsName(newFolderName("Photos")) {
		t.Error("EqualsName should return false for different name")
	}
}

func TestFolder_Rename_UpdatesNameAndTimestamp(t *testing.T) {
	folder := newRootFolder(uuid.New())
	before := folder.UpdatedAt
	time.Sleep(time.Millisecond)
	newName := newFolderName("Renamed")

	folder.Rename(newName)

	if !folder.Name.Equals(newName) {
		t.Errorf("expected name %q, got %q", newName.Value(), folder.Name.Value())
	}
	if !folder.UpdatedAt.After(before) {
		t.Error("Rename should update UpdatedAt timestamp")
	}
}

func TestFolder_MoveTo_UpdatesParentIDDepthAndTimestamp(t *testing.T) {
	folder := newRootFolder(uuid.New())
	newParentID := uuid.New()
	before := folder.UpdatedAt
	time.Sleep(time.Millisecond)

	folder.MoveTo(&newParentID, 1)

	if folder.ParentID == nil || *folder.ParentID != newParentID {
		t.Errorf("expected ParentID %v, got %v", newParentID, folder.ParentID)
	}
	if folder.Depth != 1 {
		t.Errorf("expected Depth 1, got %v", folder.Depth)
	}
	if !folder.UpdatedAt.After(before) {
		t.Error("MoveTo should update UpdatedAt timestamp")
	}
}

func TestFolder_MoveTo_ToRoot_SetsParentIDNil(t *testing.T) {
	parentID := uuid.New()
	folder := newChildFolder(&parentID, uuid.New(), 1)
	folder.MoveTo(nil, 0)
	if folder.ParentID != nil {
		t.Errorf("expected ParentID nil, got %v", folder.ParentID)
	}
}

func TestFolder_CanMoveTo_ValidMove_ReturnsNil(t *testing.T) {
	folder := newRootFolder(uuid.New())
	newParentID := uuid.New()
	if err := folder.CanMoveTo(&newParentID, 1, []uuid.UUID{}); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestFolder_CanMoveTo_SelfMove_ReturnsCircularError(t *testing.T) {
	folder := newRootFolder(uuid.New())
	selfID := folder.ID
	if err := folder.CanMoveTo(&selfID, 0, []uuid.UUID{}); err != ErrFolderCircularMove {
		t.Errorf("expected ErrFolderCircularMove, got: %v", err)
	}
}

func TestFolder_CanMoveTo_DescendantParent_ReturnsCircularError(t *testing.T) {
	folder := newRootFolder(uuid.New())
	descendantID := uuid.New()
	if err := folder.CanMoveTo(&descendantID, 1, []uuid.UUID{descendantID}); err != ErrFolderCircularMove {
		t.Errorf("expected ErrFolderCircularMove, got: %v", err)
	}
}

func TestFolder_CanMoveTo_ExceedsMaxDepth_ReturnsDepthError(t *testing.T) {
	folder := newRootFolder(uuid.New())
	newParentID := uuid.New()
	if err := folder.CanMoveTo(&newParentID, MaxFolderDepth+1, []uuid.UUID{}); err != ErrFolderMaxDepthExceeded {
		t.Errorf("expected ErrFolderMaxDepthExceeded, got: %v", err)
	}
}

func TestFolder_CanMoveTo_NilNewParent_ReturnsNil(t *testing.T) {
	parentID := uuid.New()
	folder := newChildFolder(&parentID, uuid.New(), 1)
	if err := folder.CanMoveTo(nil, 0, []uuid.UUID{}); err != nil {
		t.Errorf("unexpected error when moving to root: %v", err)
	}
}

func TestFolder_TransferOwnership_UpdatesOwnerPreservesCreatedBy(t *testing.T) {
	folder := newRootFolder(uuid.New())
	originalCreatedBy := folder.CreatedBy
	newOwner := uuid.New()
	before := folder.UpdatedAt
	time.Sleep(time.Millisecond)

	folder.TransferOwnership(newOwner)

	if folder.OwnerID != newOwner {
		t.Errorf("expected OwnerID %v, got %v", newOwner, folder.OwnerID)
	}
	if folder.CreatedBy != originalCreatedBy {
		t.Error("TransferOwnership should not change CreatedBy")
	}
	if !folder.UpdatedAt.After(before) {
		t.Error("TransferOwnership should update UpdatedAt timestamp")
	}
}

func TestFolder_UpdateDepth_UpdatesDepthAndTimestamp(t *testing.T) {
	folder := newRootFolder(uuid.New())
	before := folder.UpdatedAt
	time.Sleep(time.Millisecond)

	folder.UpdateDepth(5)

	if folder.Depth != 5 {
		t.Errorf("expected Depth 5, got %v", folder.Depth)
	}
	if !folder.UpdatedAt.After(before) {
		t.Error("UpdateDepth should update UpdatedAt timestamp")
	}
}
