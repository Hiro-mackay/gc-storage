package entity

import (
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/valueobject"
)

func newActiveFile() *File {
	fileID := uuid.New()
	name, _ := valueobject.NewFileName("test.txt")
	mimeType, _ := valueobject.NewMimeType("text/plain")
	storageKey := valueobject.NewStorageKey(fileID)
	ownerID := uuid.New()
	return ReconstructFile(
		fileID,
		uuid.New(),
		ownerID,
		ownerID,
		name,
		mimeType,
		1024,
		storageKey,
		1,
		FileStatusActive,
		time.Now(),
		time.Now(),
	)
}

func newUploadingFile() *File {
	fileID := uuid.New()
	name, _ := valueobject.NewFileName("test.txt")
	mimeType, _ := valueobject.NewMimeType("text/plain")
	storageKey := valueobject.NewStorageKey(fileID)
	ownerID := uuid.New()
	return ReconstructFile(
		fileID,
		uuid.New(),
		ownerID,
		ownerID,
		name,
		mimeType,
		1024,
		storageKey,
		1,
		FileStatusUploading,
		time.Now(),
		time.Now(),
	)
}

func newUploadFailedFile() *File {
	fileID := uuid.New()
	name, _ := valueobject.NewFileName("test.txt")
	mimeType, _ := valueobject.NewMimeType("text/plain")
	storageKey := valueobject.NewStorageKey(fileID)
	ownerID := uuid.New()
	return ReconstructFile(
		fileID,
		uuid.New(),
		ownerID,
		ownerID,
		name,
		mimeType,
		1024,
		storageKey,
		1,
		FileStatusUploadFailed,
		time.Now(),
		time.Now(),
	)
}

func TestFile_CanDownload_ActiveStatus_ReturnsTrue(t *testing.T) {
	file := newActiveFile()

	if !file.CanDownload() {
		t.Error("active file should be downloadable")
	}
}

func TestFile_CanDownload_UploadingStatus_ReturnsFalse(t *testing.T) {
	file := newUploadingFile()

	if file.CanDownload() {
		t.Error("uploading file should not be downloadable")
	}
}

func TestFile_CanDownload_UploadFailedStatus_ReturnsFalse(t *testing.T) {
	file := newUploadFailedFile()

	if file.CanDownload() {
		t.Error("upload-failed file should not be downloadable")
	}
}

func TestFile_Rename_ActiveFile_UpdatesNameAndTimestamp(t *testing.T) {
	file := newActiveFile()
	before := file.UpdatedAt

	// ensure time advances
	time.Sleep(time.Millisecond)

	newName, _ := valueobject.NewFileName("renamed.txt")
	err := file.Rename(newName)

	if err != nil {
		t.Errorf("Rename on active file should not return error, got: %v", err)
	}
	if !file.Name.Equals(newName) {
		t.Errorf("expected file name %q, got %q", newName.Value(), file.Name.Value())
	}
	if !file.UpdatedAt.After(before) {
		t.Error("Rename should update UpdatedAt timestamp")
	}
}

func TestFile_Rename_UploadingFile_ReturnsErrFileNotActive(t *testing.T) {
	file := newUploadingFile()
	newName, _ := valueobject.NewFileName("renamed.txt")

	err := file.Rename(newName)

	if err != ErrFileNotActive {
		t.Errorf("expected ErrFileNotActive, got: %v", err)
	}
}

func TestFile_Rename_UploadFailedFile_ReturnsErrFileNotActive(t *testing.T) {
	file := newUploadFailedFile()
	newName, _ := valueobject.NewFileName("renamed.txt")

	err := file.Rename(newName)

	if err != ErrFileNotActive {
		t.Errorf("expected ErrFileNotActive, got: %v", err)
	}
}

func TestFile_MoveTo_ActiveFile_UpdatesFolderIDAndTimestamp(t *testing.T) {
	file := newActiveFile()
	before := file.UpdatedAt

	// ensure time advances
	time.Sleep(time.Millisecond)

	newFolderID := uuid.New()
	err := file.MoveTo(newFolderID)

	if err != nil {
		t.Errorf("MoveTo on active file should not return error, got: %v", err)
	}
	if file.FolderID != newFolderID {
		t.Errorf("expected FolderID %v, got %v", newFolderID, file.FolderID)
	}
	if !file.UpdatedAt.After(before) {
		t.Error("MoveTo should update UpdatedAt timestamp")
	}
}

func TestFile_MoveTo_UploadingFile_ReturnsErrFileNotActive(t *testing.T) {
	file := newUploadingFile()
	newFolderID := uuid.New()

	err := file.MoveTo(newFolderID)

	if err != ErrFileNotActive {
		t.Errorf("expected ErrFileNotActive, got: %v", err)
	}
}

func TestFile_Activate_FromUploading_SetsStatusActive(t *testing.T) {
	file := newUploadingFile()

	err := file.Activate()

	if err != nil {
		t.Errorf("Activate on uploading file should not return error, got: %v", err)
	}
	if file.Status != FileStatusActive {
		t.Errorf("expected status %q, got %q", FileStatusActive, file.Status)
	}
}

func TestFile_Activate_FromActive_ReturnsErrInvalidTransition(t *testing.T) {
	file := newActiveFile()

	err := file.Activate()

	if err != ErrFileInvalidTransition {
		t.Errorf("expected ErrFileInvalidTransition, got: %v", err)
	}
}

func TestFile_MarkUploadFailed_FromUploading_SetsStatusFailed(t *testing.T) {
	file := newUploadingFile()

	err := file.MarkUploadFailed()

	if err != nil {
		t.Errorf("MarkUploadFailed on uploading file should not return error, got: %v", err)
	}
	if file.Status != FileStatusUploadFailed {
		t.Errorf("expected status %q, got %q", FileStatusUploadFailed, file.Status)
	}
}

func TestFile_MarkUploadFailed_FromActive_ReturnsErrInvalidTransition(t *testing.T) {
	file := newActiveFile()

	err := file.MarkUploadFailed()

	if err != ErrFileInvalidTransition {
		t.Errorf("expected ErrFileInvalidTransition, got: %v", err)
	}
}

func TestFile_IncrementVersion_IncrementsCurrentVersion(t *testing.T) {
	file := newActiveFile()
	initialVersion := file.CurrentVersion

	file.IncrementVersion()

	if file.CurrentVersion != initialVersion+1 {
		t.Errorf("expected CurrentVersion %d, got %d", initialVersion+1, file.CurrentVersion)
	}
}

func TestFile_IsOwnedBy_MatchingOwner_ReturnsTrue(t *testing.T) {
	file := newActiveFile()

	if !file.IsOwnedBy(file.OwnerID) {
		t.Error("IsOwnedBy should return true for the file owner")
	}
}

func TestFile_IsOwnedBy_DifferentOwner_ReturnsFalse(t *testing.T) {
	file := newActiveFile()
	differentOwner := uuid.New()

	if file.IsOwnedBy(differentOwner) {
		t.Error("IsOwnedBy should return false for a different owner")
	}
}
