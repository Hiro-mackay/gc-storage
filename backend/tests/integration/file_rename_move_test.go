package integration

import (
	"net/http"

	"github.com/Hiro-mackay/gc-storage/backend/tests/testutil"
)

// =============================================================================
// Rename Tests
// =============================================================================

// TestRenameFile_Success_ReturnsNewName - AC-03
func (s *StorageTestSuite) TestRenameFile_Success_ReturnsNewName() {
	sessionID := s.registerAndGetToken("rename-ok@example.com", "Password123", "Rename OK User")
	folderID := s.createFolder(sessionID, "RenameFolder")
	fileID := s.createActiveFile(sessionID, folderID, "original.txt")

	resp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:    http.MethodPatch,
		Path:      "/api/v1/files/" + fileID + "/rename",
		SessionID: sessionID,
		Body:      map[string]interface{}{"name": "renamed.txt"},
	})

	resp.AssertStatus(http.StatusOK).
		AssertJSONPath("data.fileId", fileID).
		AssertJSONPath("data.name", "renamed.txt")
}

// TestRenameFile_EmptyName_ReturnsBadRequest - AC-10
func (s *StorageTestSuite) TestRenameFile_EmptyName_ReturnsBadRequest() {
	sessionID := s.registerAndGetToken("rename-empty@example.com", "Password123", "Rename Empty User")
	folderID := s.createFolder(sessionID, "EmptyNameFolder")
	fileID := s.createActiveFile(sessionID, folderID, "file.txt")

	resp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:    http.MethodPatch,
		Path:      "/api/v1/files/" + fileID + "/rename",
		SessionID: sessionID,
		Body:      map[string]interface{}{"name": ""},
	})

	resp.AssertStatus(http.StatusBadRequest).
		AssertJSONError("VALIDATION_ERROR", "")
}

// TestRenameFile_ForbiddenChar_ReturnsBadRequest - AC-11
// A file name containing "/" is rejected.
func (s *StorageTestSuite) TestRenameFile_ForbiddenChar_ReturnsBadRequest() {
	sessionID := s.registerAndGetToken("rename-slash@example.com", "Password123", "Rename Slash User")
	folderID := s.createFolder(sessionID, "SlashFolder")
	fileID := s.createActiveFile(sessionID, folderID, "slashfile.txt")

	resp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:    http.MethodPatch,
		Path:      "/api/v1/files/" + fileID + "/rename",
		SessionID: sessionID,
		Body:      map[string]interface{}{"name": "bad/name.txt"},
	})

	resp.AssertStatus(http.StatusBadRequest).
		AssertJSONError("VALIDATION_ERROR", "")
}

// TestRenameFile_DuplicateName_ReturnsConflict - AC-12
func (s *StorageTestSuite) TestRenameFile_DuplicateName_ReturnsConflict() {
	sessionID := s.registerAndGetToken("rename-dup-file@example.com", "Password123", "Rename Dup File User")
	folderID := s.createFolder(sessionID, "DupFolder")

	s.createActiveFile(sessionID, folderID, "existing.txt")
	fileID := s.createActiveFile(sessionID, folderID, "other.txt")

	resp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:    http.MethodPatch,
		Path:      "/api/v1/files/" + fileID + "/rename",
		SessionID: sessionID,
		Body:      map[string]interface{}{"name": "existing.txt"},
	})

	resp.AssertStatus(http.StatusConflict).
		AssertJSONError("CONFLICT", "")
}

// TestRenameFile_OtherUserFile_ReturnsForbidden - AC-21
func (s *StorageTestSuite) TestRenameFile_OtherUserFile_ReturnsForbidden() {
	sessionID1 := s.registerAndGetToken("rename-owner@example.com", "Password123", "Rename Owner")
	sessionID2 := s.registerAndGetToken("rename-other@example.com", "Password123", "Rename Other")
	folderID := s.createFolder(sessionID1, "OwnerRenameFolder")
	fileID := s.createActiveFile(sessionID1, folderID, "owner-file.txt")

	resp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:    http.MethodPatch,
		Path:      "/api/v1/files/" + fileID + "/rename",
		SessionID: sessionID2,
		Body:      map[string]interface{}{"name": "hacked.txt"},
	})

	resp.AssertStatus(http.StatusForbidden)
}

// =============================================================================
// Move Tests
// =============================================================================

// TestMoveFile_Success_ReturnsNewFolderID - AC-04
func (s *StorageTestSuite) TestMoveFile_Success_ReturnsNewFolderID() {
	sessionID := s.registerAndGetToken("move-ok@example.com", "Password123", "Move OK User")
	srcFolderID := s.createFolder(sessionID, "SourceFolder")
	dstFolderID := s.createFolder(sessionID, "DestFolder")
	fileID := s.createActiveFile(sessionID, srcFolderID, "moveme.txt")

	resp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:    http.MethodPatch,
		Path:      "/api/v1/files/" + fileID + "/move",
		SessionID: sessionID,
		Body:      map[string]interface{}{"newFolderId": dstFolderID},
	})

	resp.AssertStatus(http.StatusOK).
		AssertJSONPath("data.fileId", fileID).
		AssertJSONPath("data.folderId", dstFolderID)
}

// TestMoveFile_DuplicateNameInDestination_ReturnsConflict - AC-12
func (s *StorageTestSuite) TestMoveFile_DuplicateNameInDestination_ReturnsConflict() {
	sessionID := s.registerAndGetToken("move-dup@example.com", "Password123", "Move Dup User")
	srcFolderID := s.createFolder(sessionID, "MoveSrc")
	dstFolderID := s.createFolder(sessionID, "MoveDst")

	s.createActiveFile(sessionID, dstFolderID, "collision.txt")
	fileID := s.createActiveFile(sessionID, srcFolderID, "collision.txt")

	resp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:    http.MethodPatch,
		Path:      "/api/v1/files/" + fileID + "/move",
		SessionID: sessionID,
		Body:      map[string]interface{}{"newFolderId": dstFolderID},
	})

	resp.AssertStatus(http.StatusConflict).
		AssertJSONError("CONFLICT", "")
}

// TestMoveFile_NonexistentFolder_ReturnsNotFound - AC-32
func (s *StorageTestSuite) TestMoveFile_NonexistentFolder_ReturnsNotFound() {
	sessionID := s.registerAndGetToken("move-nofolder@example.com", "Password123", "Move NoFolder User")
	folderID := s.createFolder(sessionID, "ExistingFolder")
	fileID := s.createActiveFile(sessionID, folderID, "tomove.txt")

	resp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:    http.MethodPatch,
		Path:      "/api/v1/files/" + fileID + "/move",
		SessionID: sessionID,
		Body:      map[string]interface{}{"newFolderId": "00000000-0000-0000-0000-000000000002"},
	})

	resp.AssertStatus(http.StatusNotFound)
}

// TestMoveFile_OtherUserFile_ReturnsForbidden - AC-21
func (s *StorageTestSuite) TestMoveFile_OtherUserFile_ReturnsForbidden() {
	sessionID1 := s.registerAndGetToken("move-fileowner@example.com", "Password123", "Move FileOwner")
	sessionID2 := s.registerAndGetToken("move-fileother@example.com", "Password123", "Move FileOther")
	ownerFolderID := s.createFolder(sessionID1, "FileOwnerFolder")
	otherFolderID := s.createFolder(sessionID2, "OtherUserFolder")
	fileID := s.createActiveFile(sessionID1, ownerFolderID, "protected.txt")

	resp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:    http.MethodPatch,
		Path:      "/api/v1/files/" + fileID + "/move",
		SessionID: sessionID2,
		Body:      map[string]interface{}{"newFolderId": otherFolderID},
	})

	resp.AssertStatus(http.StatusForbidden)
}
