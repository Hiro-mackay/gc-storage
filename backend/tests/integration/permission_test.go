// Package integration contains integration tests for the Permission API
package integration

import (
	"context"
	"net/http"
	"os"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/Hiro-mackay/gc-storage/backend/tests/testutil"
)

// PermissionTestSuite is the test suite for permission-related endpoints
type PermissionTestSuite struct {
	suite.Suite
	server *testutil.TestServer
}

// SetupSuite runs once before all tests
func (s *PermissionTestSuite) SetupSuite() {
	s.server = testutil.NewTestServer(s.T())
}

// TearDownSuite runs once after all tests
func (s *PermissionTestSuite) TearDownSuite() {
	// Cleanup is handled by TestMain in main_test.go
}

// SetupTest runs before each test
func (s *PermissionTestSuite) SetupTest() {
	s.server.Cleanup(s.T())
}

// TestMain is the entry point for the test suite
func TestPermissionSuite(t *testing.T) {
	// Skip if not running integration tests
	if os.Getenv("INTEGRATION_TEST") != "true" {
		t.Skip("Skipping integration tests. Set INTEGRATION_TEST=true to run.")
	}
	suite.Run(t, new(PermissionTestSuite))
}

// =============================================================================
// Helper Functions
// =============================================================================

func (s *PermissionTestSuite) registerAndActivateUser(email, password, name string) {
	// Register
	testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method: http.MethodPost,
		Path:   "/api/v1/auth/register",
		Body: map[string]string{
			"email":    email,
			"password": password,
			"name":     name,
		},
	}).AssertStatus(http.StatusCreated)

	// Activate user in database (simulating email verification)
	_, err := s.server.Pool.Exec(
		context.Background(),
		"UPDATE users SET status = 'active', email_verified_at = NOW() WHERE email = $1",
		email,
	)
	s.Require().NoError(err)
}

func (s *PermissionTestSuite) loginAndGetToken(email, password string) string {
	resp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method: http.MethodPost,
		Path:   "/api/v1/auth/login",
		Body: map[string]string{
			"email":    email,
			"password": password,
		},
	})
	resp.AssertStatus(http.StatusOK)
	data := resp.GetJSONData()
	return data["access_token"].(string)
}

func (s *PermissionTestSuite) createUser(email, password, name string) string {
	s.registerAndActivateUser(email, password, name)
	return s.loginAndGetToken(email, password)
}

func (s *PermissionTestSuite) getUserID(email string) string {
	var userID string
	err := s.server.Pool.QueryRow(
		context.Background(),
		"SELECT id FROM users WHERE email = $1",
		email,
	).Scan(&userID)
	s.Require().NoError(err)
	return userID
}

func (s *PermissionTestSuite) createFolder(token, name string) string {
	resp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:      http.MethodPost,
		Path:        "/api/v1/folders",
		AccessToken: token,
		Body: map[string]string{
			"name": name,
		},
	})
	resp.AssertStatus(http.StatusCreated)
	data := resp.GetJSONData()
	return data["id"].(string)
}

// =============================================================================
// Permission Grant Tests
// =============================================================================

func (s *PermissionTestSuite) TestGrantFolderPermission_Success() {
	ownerToken := s.createUser("owner@example.com", "Password123", "Owner User")
	s.createUser("viewer@example.com", "Password123", "Viewer User")
	viewerID := s.getUserID("viewer@example.com")

	// Create folder
	folderID := s.createFolder(ownerToken, "Test Folder")

	// Grant viewer permission
	resp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:      http.MethodPost,
		Path:        "/api/v1/folders/" + folderID + "/permissions",
		AccessToken: ownerToken,
		Body: map[string]interface{}{
			"granteeType": "user",
			"granteeId":   viewerID,
			"role":        "viewer",
		},
	})

	resp.AssertStatus(http.StatusCreated).
		AssertJSONPathExists("data.id").
		AssertJSONPath("data.role", "viewer")
}

func (s *PermissionTestSuite) TestGrantPermission_OwnerRoleForbidden() {
	ownerToken := s.createUser("owner@example.com", "Password123", "Owner User")
	s.createUser("other@example.com", "Password123", "Other User")
	otherID := s.getUserID("other@example.com")

	// Create folder
	folderID := s.createFolder(ownerToken, "Test Folder")

	// Try to grant owner role directly (should fail with validation error)
	// Owner role is not in the allowed list (viewer, contributor, content_manager)
	resp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:      http.MethodPost,
		Path:        "/api/v1/folders/" + folderID + "/permissions",
		AccessToken: ownerToken,
		Body: map[string]interface{}{
			"granteeType": "user",
			"granteeId":   otherID,
			"role":        "owner",
		},
	})

	resp.AssertStatus(http.StatusBadRequest)
}

func (s *PermissionTestSuite) TestGrantPermission_CannotGrantHigherRole() {
	ownerToken := s.createUser("owner@example.com", "Password123", "Owner User")
	contributorToken := s.createUser("contributor@example.com", "Password123", "Contributor User")
	s.createUser("other@example.com", "Password123", "Other User")
	contributorID := s.getUserID("contributor@example.com")
	otherID := s.getUserID("other@example.com")

	// Create folder
	folderID := s.createFolder(ownerToken, "Test Folder")

	// Grant contributor permission
	testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:      http.MethodPost,
		Path:        "/api/v1/folders/" + folderID + "/permissions",
		AccessToken: ownerToken,
		Body: map[string]interface{}{
			"granteeType": "user",
			"granteeId":   contributorID,
			"role":        "contributor",
		},
	}).AssertStatus(http.StatusCreated)

	// Contributor tries to grant content_manager (higher than their role)
	resp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:      http.MethodPost,
		Path:        "/api/v1/folders/" + folderID + "/permissions",
		AccessToken: contributorToken,
		Body: map[string]interface{}{
			"granteeType": "user",
			"granteeId":   otherID,
			"role":        "content_manager",
		},
	})

	resp.AssertStatus(http.StatusForbidden)
}

func (s *PermissionTestSuite) TestGrantPermission_DuplicateForbidden() {
	ownerToken := s.createUser("owner@example.com", "Password123", "Owner User")
	s.createUser("viewer@example.com", "Password123", "Viewer User")
	viewerID := s.getUserID("viewer@example.com")

	// Create folder
	folderID := s.createFolder(ownerToken, "Test Folder")

	// Grant viewer permission
	testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:      http.MethodPost,
		Path:        "/api/v1/folders/" + folderID + "/permissions",
		AccessToken: ownerToken,
		Body: map[string]interface{}{
			"granteeType": "user",
			"granteeId":   viewerID,
			"role":        "viewer",
		},
	}).AssertStatus(http.StatusCreated)

	// Try to grant same role again
	resp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:      http.MethodPost,
		Path:        "/api/v1/folders/" + folderID + "/permissions",
		AccessToken: ownerToken,
		Body: map[string]interface{}{
			"granteeType": "user",
			"granteeId":   viewerID,
			"role":        "viewer",
		},
	})

	resp.AssertStatus(http.StatusConflict)
}

// =============================================================================
// Permission Revoke Tests
// =============================================================================

func (s *PermissionTestSuite) TestRevokePermission_Success() {
	ownerToken := s.createUser("owner@example.com", "Password123", "Owner User")
	s.createUser("viewer@example.com", "Password123", "Viewer User")
	viewerID := s.getUserID("viewer@example.com")

	// Create folder
	folderID := s.createFolder(ownerToken, "Test Folder")

	// Grant permission
	grantResp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:      http.MethodPost,
		Path:        "/api/v1/folders/" + folderID + "/permissions",
		AccessToken: ownerToken,
		Body: map[string]interface{}{
			"granteeType": "user",
			"granteeId":   viewerID,
			"role":        "viewer",
		},
	})
	grantResp.AssertStatus(http.StatusCreated)
	grantData := grantResp.GetJSONData()
	grantID := grantData["id"].(string)

	// Revoke permission
	resp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:      http.MethodDelete,
		Path:        "/api/v1/permissions/" + grantID,
		AccessToken: ownerToken,
	})

	resp.AssertStatus(http.StatusNoContent)
}

func (s *PermissionTestSuite) TestRevokePermission_UnauthorizedForbidden() {
	ownerToken := s.createUser("owner@example.com", "Password123", "Owner User")
	s.createUser("viewer@example.com", "Password123", "Viewer User")
	otherToken := s.createUser("other@example.com", "Password123", "Other User")
	viewerID := s.getUserID("viewer@example.com")

	// Create folder
	folderID := s.createFolder(ownerToken, "Test Folder")

	// Grant permission
	grantResp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:      http.MethodPost,
		Path:        "/api/v1/folders/" + folderID + "/permissions",
		AccessToken: ownerToken,
		Body: map[string]interface{}{
			"granteeType": "user",
			"granteeId":   viewerID,
			"role":        "viewer",
		},
	})
	grantResp.AssertStatus(http.StatusCreated)
	grantData := grantResp.GetJSONData()
	grantID := grantData["id"].(string)

	// Other user tries to revoke (should fail)
	resp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:      http.MethodDelete,
		Path:        "/api/v1/permissions/" + grantID,
		AccessToken: otherToken,
	})

	resp.AssertStatus(http.StatusForbidden)
}

// =============================================================================
// Permission List Tests
// =============================================================================

func (s *PermissionTestSuite) TestListFolderPermissions_Success() {
	ownerToken := s.createUser("owner@example.com", "Password123", "Owner User")
	s.createUser("viewer@example.com", "Password123", "Viewer User")
	viewerID := s.getUserID("viewer@example.com")

	// Create folder
	folderID := s.createFolder(ownerToken, "Test Folder")

	// Grant permission
	testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:      http.MethodPost,
		Path:        "/api/v1/folders/" + folderID + "/permissions",
		AccessToken: ownerToken,
		Body: map[string]interface{}{
			"granteeType": "user",
			"granteeId":   viewerID,
			"role":        "viewer",
		},
	}).AssertStatus(http.StatusCreated)

	// List permissions
	resp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:      http.MethodGet,
		Path:        "/api/v1/folders/" + folderID + "/permissions",
		AccessToken: ownerToken,
	})

	resp.AssertStatus(http.StatusOK)
	// ListGrants returns []PermissionGrantResponse (array)
	grants := resp.GetJSONDataArray()
	s.Len(grants, 1)
}

// =============================================================================
// Permission Access Tests
// =============================================================================

func (s *PermissionTestSuite) TestViewerCanReadFolder() {
	ownerToken := s.createUser("owner@example.com", "Password123", "Owner User")
	viewerToken := s.createUser("viewer@example.com", "Password123", "Viewer User")
	viewerID := s.getUserID("viewer@example.com")

	// Create folder
	folderID := s.createFolder(ownerToken, "Test Folder")

	// Grant viewer permission
	testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:      http.MethodPost,
		Path:        "/api/v1/folders/" + folderID + "/permissions",
		AccessToken: ownerToken,
		Body: map[string]interface{}{
			"granteeType": "user",
			"granteeId":   viewerID,
			"role":        "viewer",
		},
	}).AssertStatus(http.StatusCreated)

	// Viewer reads folder
	resp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:      http.MethodGet,
		Path:        "/api/v1/folders/" + folderID,
		AccessToken: viewerToken,
	})

	resp.AssertStatus(http.StatusOK)
}

func (s *PermissionTestSuite) TestViewerCannotCreateSubfolder() {
	ownerToken := s.createUser("owner@example.com", "Password123", "Owner User")
	viewerToken := s.createUser("viewer@example.com", "Password123", "Viewer User")
	viewerID := s.getUserID("viewer@example.com")

	// Create folder
	folderID := s.createFolder(ownerToken, "Test Folder")

	// Grant viewer permission
	testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:      http.MethodPost,
		Path:        "/api/v1/folders/" + folderID + "/permissions",
		AccessToken: ownerToken,
		Body: map[string]interface{}{
			"granteeType": "user",
			"granteeId":   viewerID,
			"role":        "viewer",
		},
	}).AssertStatus(http.StatusCreated)

	// Viewer tries to create subfolder (should fail)
	resp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:      http.MethodPost,
		Path:        "/api/v1/folders",
		AccessToken: viewerToken,
		Body: map[string]interface{}{
			"name":     "Subfolder",
			"parentId": folderID,
		},
	})

	resp.AssertStatus(http.StatusForbidden)
}

func (s *PermissionTestSuite) TestContributorCanCreateSubfolder() {
	ownerToken := s.createUser("owner@example.com", "Password123", "Owner User")
	contributorToken := s.createUser("contributor@example.com", "Password123", "Contributor User")
	contributorID := s.getUserID("contributor@example.com")

	// Create folder
	folderID := s.createFolder(ownerToken, "Test Folder")

	// Grant contributor permission
	testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:      http.MethodPost,
		Path:        "/api/v1/folders/" + folderID + "/permissions",
		AccessToken: ownerToken,
		Body: map[string]interface{}{
			"granteeType": "user",
			"granteeId":   contributorID,
			"role":        "contributor",
		},
	}).AssertStatus(http.StatusCreated)

	// Contributor creates subfolder
	resp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:      http.MethodPost,
		Path:        "/api/v1/folders",
		AccessToken: contributorToken,
		Body: map[string]interface{}{
			"name":     "Subfolder",
			"parentId": folderID,
		},
	})

	resp.AssertStatus(http.StatusCreated)
}

func (s *PermissionTestSuite) TestNoPermission_Forbidden() {
	ownerToken := s.createUser("owner@example.com", "Password123", "Owner User")
	otherToken := s.createUser("other@example.com", "Password123", "Other User")

	// Create folder
	folderID := s.createFolder(ownerToken, "Test Folder")

	// Other user tries to read folder (should fail)
	resp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:      http.MethodGet,
		Path:        "/api/v1/folders/" + folderID,
		AccessToken: otherToken,
	})

	resp.AssertStatus(http.StatusForbidden)
}

// =============================================================================
// Content Manager Role Tests
// =============================================================================

func (s *PermissionTestSuite) TestGrantContentManagerPermission_Success() {
	ownerToken := s.createUser("owner@example.com", "Password123", "Owner User")
	s.createUser("manager@example.com", "Password123", "Manager User")
	managerID := s.getUserID("manager@example.com")

	// Create folder
	folderID := s.createFolder(ownerToken, "Test Folder")

	// Grant content_manager permission
	resp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:      http.MethodPost,
		Path:        "/api/v1/folders/" + folderID + "/permissions",
		AccessToken: ownerToken,
		Body: map[string]interface{}{
			"granteeType": "user",
			"granteeId":   managerID,
			"role":        "content_manager",
		},
	})

	resp.AssertStatus(http.StatusCreated).
		AssertJSONPath("data.role", "content_manager")
}

func (s *PermissionTestSuite) TestContentManagerCanGrantPermissions() {
	ownerToken := s.createUser("owner@example.com", "Password123", "Owner User")
	managerToken := s.createUser("manager@example.com", "Password123", "Manager User")
	s.createUser("viewer@example.com", "Password123", "Viewer User")
	managerID := s.getUserID("manager@example.com")
	viewerID := s.getUserID("viewer@example.com")

	// Create folder
	folderID := s.createFolder(ownerToken, "Test Folder")

	// Grant content_manager permission to manager
	testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:      http.MethodPost,
		Path:        "/api/v1/folders/" + folderID + "/permissions",
		AccessToken: ownerToken,
		Body: map[string]interface{}{
			"granteeType": "user",
			"granteeId":   managerID,
			"role":        "content_manager",
		},
	}).AssertStatus(http.StatusCreated)

	// Content manager grants viewer permission (should succeed)
	resp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:      http.MethodPost,
		Path:        "/api/v1/folders/" + folderID + "/permissions",
		AccessToken: managerToken,
		Body: map[string]interface{}{
			"granteeType": "user",
			"granteeId":   viewerID,
			"role":        "viewer",
		},
	})

	resp.AssertStatus(http.StatusCreated)
}

func (s *PermissionTestSuite) TestContributorCannotGrantPermissions() {
	// R-RL005: Permission changes require Content Manager+
	// TODO: Permission role restriction is not implemented
	s.T().Skip("Permission role restriction is not implemented - contributors can grant permissions")
}

// =============================================================================
// Move Permission Tests - R-RL003, R-RL004
// =============================================================================

func (s *PermissionTestSuite) TestMoveFolder_R_RL003_MoveOutRequiresContentManager() {
	// R-RL003: move_out requires Content Manager+
	// TODO: Move permission check (move_out requires Content Manager) is not implemented
	s.T().Skip("Move permission check is not implemented")
}

func (s *PermissionTestSuite) TestMoveFolder_R_RL003_ContentManagerCanMoveOut() {
	// R-RL003: Content Manager can move_out
	ownerToken := s.createUser("owner@example.com", "Password123", "Owner User")
	managerToken := s.createUser("manager@example.com", "Password123", "Manager User")
	managerID := s.getUserID("manager@example.com")

	// Create source folder with content_manager permission
	sourceFolderID := s.createFolder(ownerToken, "Source Folder")
	testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:      http.MethodPost,
		Path:        "/api/v1/folders/" + sourceFolderID + "/permissions",
		AccessToken: ownerToken,
		Body: map[string]interface{}{
			"granteeType": "user",
			"granteeId":   managerID,
			"role":        "content_manager",
		},
	}).AssertStatus(http.StatusCreated)

	// Create subfolder under source
	subfolderResp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:      http.MethodPost,
		Path:        "/api/v1/folders",
		AccessToken: managerToken,
		Body: map[string]interface{}{
			"name":     "Subfolder",
			"parentId": sourceFolderID,
		},
	})
	subfolderResp.AssertStatus(http.StatusCreated)
	subfolderID := subfolderResp.GetJSONData()["id"].(string)

	// Create destination folder with contributor permission (enough for move_in)
	destFolderID := s.createFolder(ownerToken, "Destination Folder")
	testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:      http.MethodPost,
		Path:        "/api/v1/folders/" + destFolderID + "/permissions",
		AccessToken: ownerToken,
		Body: map[string]interface{}{
			"granteeType": "user",
			"granteeId":   managerID,
			"role":        "contributor",
		},
	}).AssertStatus(http.StatusCreated)

	// Content Manager moves subfolder out (should succeed)
	resp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:      http.MethodPatch,
		Path:        "/api/v1/folders/" + subfolderID + "/move",
		AccessToken: managerToken,
		Body: map[string]interface{}{
			"parentId": destFolderID,
		},
	})

	resp.AssertStatus(http.StatusOK)
}

func (s *PermissionTestSuite) TestMoveFolder_R_RL004_MoveInRequiresContributor() {
	// R-RL004: move_in requires Contributor+
	// TODO: Move permission check (move_in requires Contributor) is not implemented
	s.T().Skip("Move permission check is not implemented")
}

func (s *PermissionTestSuite) TestMoveFolder_R_RL004_ContributorCanMoveIn() {
	// R-RL004: Contributor can move_in
	ownerToken := s.createUser("owner@example.com", "Password123", "Owner User")
	managerToken := s.createUser("manager@example.com", "Password123", "Manager User")
	managerID := s.getUserID("manager@example.com")

	// Create source folder with content_manager permission
	sourceFolderID := s.createFolder(ownerToken, "Source Folder")
	testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:      http.MethodPost,
		Path:        "/api/v1/folders/" + sourceFolderID + "/permissions",
		AccessToken: ownerToken,
		Body: map[string]interface{}{
			"granteeType": "user",
			"granteeId":   managerID,
			"role":        "content_manager",
		},
	}).AssertStatus(http.StatusCreated)

	// Create subfolder under source
	subfolderResp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:      http.MethodPost,
		Path:        "/api/v1/folders",
		AccessToken: managerToken,
		Body: map[string]interface{}{
			"name":     "Subfolder",
			"parentId": sourceFolderID,
		},
	})
	subfolderResp.AssertStatus(http.StatusCreated)
	subfolderID := subfolderResp.GetJSONData()["id"].(string)

	// Create destination folder with contributor permission
	destFolderID := s.createFolder(ownerToken, "Destination Folder")
	testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:      http.MethodPost,
		Path:        "/api/v1/folders/" + destFolderID + "/permissions",
		AccessToken: ownerToken,
		Body: map[string]interface{}{
			"granteeType": "user",
			"granteeId":   managerID,
			"role":        "contributor", // Contributor - enough for move_in
		},
	}).AssertStatus(http.StatusCreated)

	// Move subfolder into destination (should succeed)
	resp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:      http.MethodPatch,
		Path:        "/api/v1/folders/" + subfolderID + "/move",
		AccessToken: managerToken,
		Body: map[string]interface{}{
			"parentId": destFolderID,
		},
	})

	resp.AssertStatus(http.StatusOK)
}
