package integration

import (
	"context"
	"net/http"

	"github.com/Hiro-mackay/gc-storage/backend/tests/testutil"
)

// createActiveFile creates a file via the upload API then directly sets it as active
// in the DB and inserts a file_version row, simulating a completed upload.
func (s *StorageTestSuite) createActiveFile(sessionID, folderID, fileName string) string {
	uploadResp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:    http.MethodPost,
		Path:      "/api/v1/files/upload",
		SessionID: sessionID,
		Body: map[string]interface{}{
			"folderId": folderID,
			"fileName": fileName,
			"mimeType": "text/plain",
			"size":     1024,
		},
	})
	uploadResp.AssertStatus(http.StatusCreated)
	fileIDStr := uploadResp.GetJSONData()["fileId"].(string)

	_, err := s.server.Pool.Exec(
		context.Background(),
		"UPDATE files SET status = 'active' WHERE id = $1",
		fileIDStr,
	)
	s.Require().NoError(err)

	_, err = s.server.Pool.Exec(
		context.Background(),
		`INSERT INTO file_versions (id, file_id, version_number, minio_version_id, size, checksum, uploaded_by, created_at)
		 SELECT gen_random_uuid(), id, 1, 'mock-minio-v1', 1024, 'sha256:abc123', owner_id, NOW()
		 FROM files WHERE id = $1`,
		fileIDStr,
	)
	s.Require().NoError(err)
	return fileIDStr
}

// createFolder creates a folder and returns its ID.
func (s *StorageTestSuite) createFolder(sessionID, name string) string {
	resp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:    http.MethodPost,
		Path:      "/api/v1/folders",
		SessionID: sessionID,
		Body:      map[string]interface{}{"name": name},
	})
	resp.AssertStatus(http.StatusCreated)
	return resp.GetJSONData()["id"].(string)
}

// =============================================================================
// Download URL Tests
// =============================================================================

// TestDownloadURL_ActiveFile_ReturnsPresignedURL - AC-01
func (s *StorageTestSuite) TestDownloadURL_ActiveFile_ReturnsPresignedURL() {
	sessionID := s.registerAndGetToken("dl-active@example.com", "Password123", "DL Active User")
	folderID := s.createFolder(sessionID, "DLFolder")
	fileID := s.createActiveFile(sessionID, folderID, "readme.txt")

	resp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:    http.MethodGet,
		Path:      "/api/v1/files/" + fileID + "/download",
		SessionID: sessionID,
	})

	resp.AssertStatus(http.StatusOK).
		AssertJSONPathExists("data.downloadUrl").
		AssertJSONPath("data.fileName", "readme.txt").
		AssertJSONPath("data.mimeType", "text/plain").
		AssertJSONPath("data.versionNumber", float64(1))
}

// TestDownloadURL_SpecificVersion_ReturnsPresignedURL - AC-01 with version param
func (s *StorageTestSuite) TestDownloadURL_SpecificVersion_ReturnsPresignedURL() {
	sessionID := s.registerAndGetToken("dl-version@example.com", "Password123", "DL Version User")
	folderID := s.createFolder(sessionID, "VersionFolder")
	fileID := s.createActiveFile(sessionID, folderID, "versioned.txt")

	resp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:    http.MethodGet,
		Path:      "/api/v1/files/" + fileID + "/download?version=1",
		SessionID: sessionID,
	})

	resp.AssertStatus(http.StatusOK).
		AssertJSONPath("data.versionNumber", float64(1))
}

// TestDownloadURL_UploadingFile_ReturnsBadRequest - AC-30
// A file still in uploading status cannot be downloaded (CanDownload() = false).
func (s *StorageTestSuite) TestDownloadURL_UploadingFile_ReturnsBadRequest() {
	sessionID := s.registerAndGetToken("dl-uploading@example.com", "Password123", "DL Uploading User")
	folderID := s.createFolder(sessionID, "UploadingFolder")

	// Initiate upload without completing — file stays in uploading status
	uploadResp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:    http.MethodPost,
		Path:      "/api/v1/files/upload",
		SessionID: sessionID,
		Body: map[string]interface{}{
			"folderId": folderID,
			"fileName": "notdone.txt",
			"mimeType": "text/plain",
			"size":     1024,
		},
	})
	uploadResp.AssertStatus(http.StatusCreated)
	fileID := uploadResp.GetJSONData()["fileId"].(string)

	resp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:    http.MethodGet,
		Path:      "/api/v1/files/" + fileID + "/download",
		SessionID: sessionID,
	})

	resp.AssertStatus(http.StatusBadRequest).
		AssertJSONError("VALIDATION_ERROR", "")
}

// TestDownloadURL_NotFound_Returns404
func (s *StorageTestSuite) TestDownloadURL_NotFound_Returns404() {
	sessionID := s.registerAndGetToken("dl-notfound@example.com", "Password123", "DL NotFound User")

	resp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:    http.MethodGet,
		Path:      "/api/v1/files/00000000-0000-0000-0000-000000000001/download",
		SessionID: sessionID,
	})

	resp.AssertStatus(http.StatusNotFound)
}

// TestDownloadURL_OtherUserFile_ReturnsForbidden - AC-20
func (s *StorageTestSuite) TestDownloadURL_OtherUserFile_ReturnsForbidden() {
	sessionID1 := s.registerAndGetToken("dl-owner@example.com", "Password123", "DL Owner")
	sessionID2 := s.registerAndGetToken("dl-other@example.com", "Password123", "DL Other")
	folderID := s.createFolder(sessionID1, "OwnerFolder")
	fileID := s.createActiveFile(sessionID1, folderID, "secret.txt")

	resp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:    http.MethodGet,
		Path:      "/api/v1/files/" + fileID + "/download",
		SessionID: sessionID2,
	})

	resp.AssertStatus(http.StatusForbidden)
}

// =============================================================================
// Version List Tests
// =============================================================================

// TestListFileVersions_ReturnsVersionList - AC-05
func (s *StorageTestSuite) TestListFileVersions_ReturnsVersionList() {
	sessionID := s.registerAndGetToken("versions@example.com", "Password123", "Versions User")
	folderID := s.createFolder(sessionID, "VersionsFolder")
	fileID := s.createActiveFile(sessionID, folderID, "versioned.txt")

	// Insert a second version directly
	_, err := s.server.Pool.Exec(
		context.Background(),
		`INSERT INTO file_versions (id, file_id, version_number, minio_version_id, size, checksum, uploaded_by, created_at)
		 SELECT gen_random_uuid(), id, 2, 'mock-minio-v2', 2048, 'sha256:def456', owner_id, NOW()
		 FROM files WHERE id = $1`,
		fileID,
	)
	s.Require().NoError(err)

	resp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:    http.MethodGet,
		Path:      "/api/v1/files/" + fileID + "/versions",
		SessionID: sessionID,
	})

	resp.AssertStatus(http.StatusOK).
		AssertJSONPath("data.fileId", fileID).
		AssertJSONPath("data.fileName", "versioned.txt").
		AssertJSONPathExists("data.versions")

	versions, ok := resp.GetJSONData()["versions"].([]interface{})
	s.True(ok, "versions should be an array")
	s.Len(versions, 2, "should have 2 versions")
}

// TestListFileVersions_OtherUserFile_ReturnsForbidden - AC-20
func (s *StorageTestSuite) TestListFileVersions_OtherUserFile_ReturnsForbidden() {
	sessionID1 := s.registerAndGetToken("ver-owner@example.com", "Password123", "Ver Owner")
	sessionID2 := s.registerAndGetToken("ver-other@example.com", "Password123", "Ver Other")
	folderID := s.createFolder(sessionID1, "VerOwnerFolder")
	fileID := s.createActiveFile(sessionID1, folderID, "private-ver.txt")

	resp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:    http.MethodGet,
		Path:      "/api/v1/files/" + fileID + "/versions",
		SessionID: sessionID2,
	})

	resp.AssertStatus(http.StatusForbidden)
}
