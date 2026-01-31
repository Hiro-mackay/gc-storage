// Package integration contains integration tests for the API
package integration

import (
	"context"
	"net/http"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"

	"github.com/Hiro-mackay/gc-storage/backend/tests/testutil"
)

// StorageTestSuite is the test suite for storage-related endpoints (folders and files)
type StorageTestSuite struct {
	suite.Suite
	server *testutil.TestServer
}

// SetupSuite runs once before all tests
func (s *StorageTestSuite) SetupSuite() {
	s.server = testutil.NewTestServer(s.T())
}

// TearDownSuite runs once after all tests
func (s *StorageTestSuite) TearDownSuite() {
	// Cleanup is handled by TestMain in main_test.go
}

// SetupTest runs before each test
func (s *StorageTestSuite) SetupTest() {
	s.server.Cleanup(s.T())
}

// TestMain is the entry point for the test suite
func TestStorageSuite(t *testing.T) {
	// Skip if not running integration tests
	if os.Getenv("INTEGRATION_TEST") != "true" {
		t.Skip("Skipping integration tests. Set INTEGRATION_TEST=true to run.")
	}
	suite.Run(t, new(StorageTestSuite))
}

// =============================================================================
// Folder Tests - Based on docs/03-domains/folder.md requirements
// =============================================================================

// -----------------------------------------------------------------------------
// R-FD001: Same parent folder, name is unique
// -----------------------------------------------------------------------------

func (s *StorageTestSuite) TestCreateFolder_Success_RootLevel() {
	sessionID := s.registerAndGetToken("folder-user@example.com", "Password123", "Folder User")

	resp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:      http.MethodPost,
		Path:        "/api/v1/folders",
		SessionID: sessionID,
		Body: map[string]interface{}{
			"name": "My Documents",
		},
	})

	resp.AssertStatus(http.StatusCreated).
		AssertJSONPathExists("data.id").
		AssertJSONPath("data.name", "My Documents").
		AssertJSONPath("data.depth", float64(0))
}

func (s *StorageTestSuite) TestCreateFolder_Success_Nested() {
	sessionID := s.registerAndGetToken("nested-folder@example.com", "Password123", "Nested Folder User")

	// Create parent folder
	parentResp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:      http.MethodPost,
		Path:        "/api/v1/folders",
		SessionID: sessionID,
		Body: map[string]interface{}{
			"name": "Parent",
		},
	})
	parentResp.AssertStatus(http.StatusCreated)
	parentData := parentResp.GetJSONData()
	parentID := parentData["id"].(string)

	// Create child folder
	childResp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:      http.MethodPost,
		Path:        "/api/v1/folders",
		SessionID: sessionID,
		Body: map[string]interface{}{
			"name":     "Child",
			"parentId": parentID,
		},
	})

	childResp.AssertStatus(http.StatusCreated).
		AssertJSONPath("data.name", "Child").
		AssertJSONPath("data.depth", float64(1)).
		AssertJSONPath("data.parentId", parentID)
}

func (s *StorageTestSuite) TestCreateFolder_R_FD001_DuplicateNameInSameParent() {
	// R-FD001: Same parent folder, name is unique
	sessionID := s.registerAndGetToken("dup-name@example.com", "Password123", "Dup Name User")

	// Create parent folder
	parentResp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:      http.MethodPost,
		Path:        "/api/v1/folders",
		SessionID: sessionID,
		Body: map[string]interface{}{
			"name": "Parent",
		},
	})
	parentResp.AssertStatus(http.StatusCreated)
	parentData := parentResp.GetJSONData()
	parentID := parentData["id"].(string)

	// Create first child
	testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:      http.MethodPost,
		Path:        "/api/v1/folders",
		SessionID: sessionID,
		Body: map[string]interface{}{
			"name":     "UniqueChild",
			"parentId": parentID,
		},
	}).AssertStatus(http.StatusCreated)

	// Try to create another child with same name - should fail
	dupResp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:      http.MethodPost,
		Path:        "/api/v1/folders",
		SessionID: sessionID,
		Body: map[string]interface{}{
			"name":     "UniqueChild",
			"parentId": parentID,
		},
	})

	dupResp.AssertStatus(http.StatusConflict).
		AssertJSONError("CONFLICT", "folder with same name already exists")
}

func (s *StorageTestSuite) TestCreateFolder_R_FD002_DuplicateNameAtRootLevel() {
	// R-FD002: Same owner's root level (parentId=null), name is unique
	sessionID := s.registerAndGetToken("dup-root@example.com", "Password123", "Dup Root User")

	// Create first root folder
	testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:      http.MethodPost,
		Path:        "/api/v1/folders",
		SessionID: sessionID,
		Body: map[string]interface{}{
			"name": "RootFolder",
		},
	}).AssertStatus(http.StatusCreated)

	// Try to create another root folder with same name - should fail
	dupResp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:      http.MethodPost,
		Path:        "/api/v1/folders",
		SessionID: sessionID,
		Body: map[string]interface{}{
			"name": "RootFolder",
		},
	})

	dupResp.AssertStatus(http.StatusConflict).
		AssertJSONError("CONFLICT", "folder with same name already exists")
}

func (s *StorageTestSuite) TestCreateFolder_SameNameDifferentParents_OK() {
	// Same name in different parents should be allowed
	sessionID := s.registerAndGetToken("diff-parent@example.com", "Password123", "Diff Parent User")

	// Create two parent folders
	parent1Resp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:      http.MethodPost,
		Path:        "/api/v1/folders",
		SessionID: sessionID,
		Body: map[string]interface{}{
			"name": "Parent1",
		},
	})
	parent1Resp.AssertStatus(http.StatusCreated)
	parent1ID := parent1Resp.GetJSONData()["id"].(string)

	parent2Resp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:      http.MethodPost,
		Path:        "/api/v1/folders",
		SessionID: sessionID,
		Body: map[string]interface{}{
			"name": "Parent2",
		},
	})
	parent2Resp.AssertStatus(http.StatusCreated)
	parent2ID := parent2Resp.GetJSONData()["id"].(string)

	// Create folder with same name in both parents - should succeed
	testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:      http.MethodPost,
		Path:        "/api/v1/folders",
		SessionID: sessionID,
		Body: map[string]interface{}{
			"name":     "SharedName",
			"parentId": parent1ID,
		},
	}).AssertStatus(http.StatusCreated)

	testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:      http.MethodPost,
		Path:        "/api/v1/folders",
		SessionID: sessionID,
		Body: map[string]interface{}{
			"name":     "SharedName",
			"parentId": parent2ID,
		},
	}).AssertStatus(http.StatusCreated)
}

// -----------------------------------------------------------------------------
// R-FD003: Cannot move folder to itself or descendants (circular reference prevention)
// -----------------------------------------------------------------------------

func (s *StorageTestSuite) TestMoveFolder_R_FD003_CannotMoveToSelf() {
	sessionID := s.registerAndGetToken("move-self@example.com", "Password123", "Move Self User")

	// Create folder
	folderResp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:      http.MethodPost,
		Path:        "/api/v1/folders",
		SessionID: sessionID,
		Body: map[string]interface{}{
			"name": "TestFolder",
		},
	})
	folderResp.AssertStatus(http.StatusCreated)
	folderID := folderResp.GetJSONData()["id"].(string)

	// Try to move folder to itself - should fail
	resp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:      http.MethodPatch,
		Path:        "/api/v1/folders/" + folderID + "/move",
		SessionID: sessionID,
		Body: map[string]interface{}{
			"newParentId": folderID,
		},
	})

	resp.AssertStatus(http.StatusBadRequest).
		AssertJSONError("VALIDATION_ERROR", "")
}

func (s *StorageTestSuite) TestMoveFolder_R_FD003_CannotMoveToDescendant() {
	sessionID := s.registerAndGetToken("move-desc@example.com", "Password123", "Move Desc User")

	// Create parent folder
	parentResp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:      http.MethodPost,
		Path:        "/api/v1/folders",
		SessionID: sessionID,
		Body: map[string]interface{}{
			"name": "Parent",
		},
	})
	parentResp.AssertStatus(http.StatusCreated)
	parentID := parentResp.GetJSONData()["id"].(string)

	// Create child folder
	childResp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:      http.MethodPost,
		Path:        "/api/v1/folders",
		SessionID: sessionID,
		Body: map[string]interface{}{
			"name":      "Child",
			"parentId": parentID,
		},
	})
	childResp.AssertStatus(http.StatusCreated)
	childID := childResp.GetJSONData()["id"].(string)

	// Create grandchild folder
	grandchildResp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:      http.MethodPost,
		Path:        "/api/v1/folders",
		SessionID: sessionID,
		Body: map[string]interface{}{
			"name":      "Grandchild",
			"parentId": childID,
		},
	})
	grandchildResp.AssertStatus(http.StatusCreated)
	grandchildID := grandchildResp.GetJSONData()["id"].(string)

	// Try to move parent to grandchild - should fail
	resp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:      http.MethodPatch,
		Path:        "/api/v1/folders/" + parentID + "/move",
		SessionID: sessionID,
		Body: map[string]interface{}{
			"newParentId": grandchildID,
		},
	})

	resp.AssertStatus(http.StatusBadRequest).
		AssertJSONError("VALIDATION_ERROR", "")
}

// -----------------------------------------------------------------------------
// R-FD004: Maximum folder depth is 20
// -----------------------------------------------------------------------------

func (s *StorageTestSuite) TestCreateFolder_R_FD004_MaxDepth20() {
	// R-FD004: Max depth is 20, meaning depth 0-20 are valid (21 levels total)
	// Per docs: "parent.depth + 1 <= 20"
	sessionID := s.registerAndGetToken("max-depth@example.com", "Password123", "Max Depth User")

	// Create folders from depth 0 to depth 20 (21 folders total)
	var currentParentID *string
	for i := 0; i <= 20; i++ {
		// Use letters A-U to avoid invalid characters (21 folders)
		name := "Level" + string(rune('A'+i))
		body := map[string]interface{}{
			"name": name,
		}
		if currentParentID != nil {
			body["parentId"] = *currentParentID
		}

		resp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
			Method:      http.MethodPost,
			Path:        "/api/v1/folders",
			SessionID: sessionID,
			Body:        body,
		})
		resp.AssertStatus(http.StatusCreated)
		id := resp.GetJSONData()["id"].(string)
		currentParentID = &id
	}

	// Try to create one more at depth 21 (exceeds max depth of 20) - should fail
	resp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:      http.MethodPost,
		Path:        "/api/v1/folders",
		SessionID: sessionID,
		Body: map[string]interface{}{
			"name":     "TooDeep",
			"parentId": *currentParentID,
		},
	})

	resp.AssertStatus(http.StatusBadRequest).
		AssertJSONError("VALIDATION_ERROR", "")
}

// -----------------------------------------------------------------------------
// R-FN001-R-FN005: FolderName validation
// -----------------------------------------------------------------------------

func (s *StorageTestSuite) TestCreateFolder_R_FN001_NameLength() {
	// R-FN001: 1-255 bytes (UTF-8)
	sessionID := s.registerAndGetToken("name-length@example.com", "Password123", "Name Length User")

	// Empty name - should fail (R-FN004)
	testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:      http.MethodPost,
		Path:        "/api/v1/folders",
		SessionID: sessionID,
		Body: map[string]interface{}{
			"name": "",
		},
	}).AssertStatus(http.StatusBadRequest)

	// Valid short name
	testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:      http.MethodPost,
		Path:        "/api/v1/folders",
		SessionID: sessionID,
		Body: map[string]interface{}{
			"name": "A",
		},
	}).AssertStatus(http.StatusCreated)
}

func (s *StorageTestSuite) TestCreateFolder_R_FN001_NameLength255Bytes() {
	// R-FN001: Exactly 255 bytes should be valid
	sessionID := s.registerAndGetToken("name-255@example.com", "Password123", "Name 255 User")

	// Create a name with exactly 255 bytes
	name255 := ""
	for i := 0; i < 255; i++ {
		name255 += "a"
	}

	resp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:      http.MethodPost,
		Path:        "/api/v1/folders",
		SessionID: sessionID,
		Body: map[string]interface{}{
			"name": name255,
		},
	})

	resp.AssertStatus(http.StatusCreated).
		AssertJSONPath("data.name", name255)
}

func (s *StorageTestSuite) TestCreateFolder_R_FN001_NameLength256Bytes() {
	// R-FN001: 256 bytes should fail
	sessionID := s.registerAndGetToken("name-256@example.com", "Password123", "Name 256 User")

	// Create a name with 256 bytes
	name256 := ""
	for i := 0; i < 256; i++ {
		name256 += "a"
	}

	resp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:      http.MethodPost,
		Path:        "/api/v1/folders",
		SessionID: sessionID,
		Body: map[string]interface{}{
			"name": name256,
		},
	})

	resp.AssertStatus(http.StatusBadRequest).
		AssertJSONError("VALIDATION_ERROR", "")
}

func (s *StorageTestSuite) TestCreateFolder_R_FN004_WhitespaceOnlyName() {
	// R-FN004: Whitespace-only names are not allowed
	sessionID := s.registerAndGetToken("whitespace@example.com", "Password123", "Whitespace User")

	// Spaces only
	testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:      http.MethodPost,
		Path:        "/api/v1/folders",
		SessionID: sessionID,
		Body: map[string]interface{}{
			"name": "   ",
		},
	}).AssertStatus(http.StatusBadRequest)

	// Tabs only
	testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:      http.MethodPost,
		Path:        "/api/v1/folders",
		SessionID: sessionID,
		Body: map[string]interface{}{
			"name": "\t\t\t",
		},
	}).AssertStatus(http.StatusBadRequest)

	// Mixed whitespace
	testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:      http.MethodPost,
		Path:        "/api/v1/folders",
		SessionID: sessionID,
		Body: map[string]interface{}{
			"name": " \t \n ",
		},
	}).AssertStatus(http.StatusBadRequest)
}

func (s *StorageTestSuite) TestCreateFolder_R_FN002_ForbiddenChars() {
	// R-FN002: Forbidden characters (/ \ : * ? " < > |)
	sessionID := s.registerAndGetToken("forbidden-chars@example.com", "Password123", "Forbidden Chars User")

	forbiddenChars := []string{"/", "\\", ":", "*", "?", "\"", "<", ">", "|"}

	for _, char := range forbiddenChars {
		resp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
			Method:      http.MethodPost,
			Path:        "/api/v1/folders",
			SessionID: sessionID,
			Body: map[string]interface{}{
				"name": "test" + char + "folder",
			},
		})
		resp.AssertStatus(http.StatusBadRequest)
	}
}

func (s *StorageTestSuite) TestCreateFolder_R_FN005_DotNotAllowed() {
	// R-FN005: "." and ".." are not allowed
	sessionID := s.registerAndGetToken("dot-names@example.com", "Password123", "Dot Names User")

	testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:      http.MethodPost,
		Path:        "/api/v1/folders",
		SessionID: sessionID,
		Body: map[string]interface{}{
			"name": ".",
		},
	}).AssertStatus(http.StatusBadRequest)

	testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:      http.MethodPost,
		Path:        "/api/v1/folders",
		SessionID: sessionID,
		Body: map[string]interface{}{
			"name": "..",
		},
	}).AssertStatus(http.StatusBadRequest)
}

// -----------------------------------------------------------------------------
// Folder Rename Tests
// -----------------------------------------------------------------------------

func (s *StorageTestSuite) TestRenameFolder_Success() {
	sessionID := s.registerAndGetToken("rename-folder@example.com", "Password123", "Rename Folder User")

	// Create folder
	createResp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:      http.MethodPost,
		Path:        "/api/v1/folders",
		SessionID: sessionID,
		Body: map[string]interface{}{
			"name": "OldName",
		},
	})
	createResp.AssertStatus(http.StatusCreated)
	folderID := createResp.GetJSONData()["id"].(string)

	// Rename folder
	renameResp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:      http.MethodPatch,
		Path:        "/api/v1/folders/" + folderID + "/rename",
		SessionID: sessionID,
		Body: map[string]interface{}{
			"name": "NewName",
		},
	})

	renameResp.AssertStatus(http.StatusOK).
		AssertJSONPath("data.name", "NewName")
}

func (s *StorageTestSuite) TestRenameFolder_DuplicateName() {
	sessionID := s.registerAndGetToken("rename-dup@example.com", "Password123", "Rename Dup User")

	// Create two folders
	testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:      http.MethodPost,
		Path:        "/api/v1/folders",
		SessionID: sessionID,
		Body: map[string]interface{}{
			"name": "Folder1",
		},
	}).AssertStatus(http.StatusCreated)

	folder2Resp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:      http.MethodPost,
		Path:        "/api/v1/folders",
		SessionID: sessionID,
		Body: map[string]interface{}{
			"name": "Folder2",
		},
	})
	folder2Resp.AssertStatus(http.StatusCreated)
	folder2ID := folder2Resp.GetJSONData()["id"].(string)

	// Try to rename Folder2 to Folder1 - should fail
	resp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:      http.MethodPatch,
		Path:        "/api/v1/folders/" + folder2ID + "/rename",
		SessionID: sessionID,
		Body: map[string]interface{}{
			"name": "Folder1",
		},
	})

	resp.AssertStatus(http.StatusConflict).
		AssertJSONError("CONFLICT", "folder with same name already exists")
}

// -----------------------------------------------------------------------------
// Folder Move Tests
// -----------------------------------------------------------------------------

func (s *StorageTestSuite) TestMoveFolder_Success() {
	sessionID := s.registerAndGetToken("move-folder@example.com", "Password123", "Move Folder User")

	// Create destination folder
	destResp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:      http.MethodPost,
		Path:        "/api/v1/folders",
		SessionID: sessionID,
		Body: map[string]interface{}{
			"name": "Destination",
		},
	})
	destResp.AssertStatus(http.StatusCreated)
	destID := destResp.GetJSONData()["id"].(string)

	// Create folder to move
	folderResp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:      http.MethodPost,
		Path:        "/api/v1/folders",
		SessionID: sessionID,
		Body: map[string]interface{}{
			"name": "ToMove",
		},
	})
	folderResp.AssertStatus(http.StatusCreated)
	folderID := folderResp.GetJSONData()["id"].(string)

	// Move folder
	moveResp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:      http.MethodPatch,
		Path:        "/api/v1/folders/" + folderID + "/move",
		SessionID: sessionID,
		Body: map[string]interface{}{
			"newParentId": destID,
		},
	})

	moveResp.AssertStatus(http.StatusOK).
		AssertJSONPath("data.parentId", destID).
		AssertJSONPath("data.depth", float64(1))
}

func (s *StorageTestSuite) TestMoveFolder_ToRoot() {
	sessionID := s.registerAndGetToken("move-to-root@example.com", "Password123", "Move To Root User")

	// Create parent folder
	parentResp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:      http.MethodPost,
		Path:        "/api/v1/folders",
		SessionID: sessionID,
		Body: map[string]interface{}{
			"name": "Parent",
		},
	})
	parentResp.AssertStatus(http.StatusCreated)
	parentID := parentResp.GetJSONData()["id"].(string)

	// Create child folder
	childResp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:      http.MethodPost,
		Path:        "/api/v1/folders",
		SessionID: sessionID,
		Body: map[string]interface{}{
			"name":      "Child",
			"parentId": parentID,
		},
	})
	childResp.AssertStatus(http.StatusCreated)
	childID := childResp.GetJSONData()["id"].(string)

	// Move child to root (null parent)
	moveResp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:      http.MethodPatch,
		Path:        "/api/v1/folders/" + childID + "/move",
		SessionID: sessionID,
		Body:        map[string]interface{}{},
	})

	moveResp.AssertStatus(http.StatusOK).
		AssertJSONPath("data.depth", float64(0))
}

// -----------------------------------------------------------------------------
// Folder Delete Tests - R-FD005, R-FD006
// -----------------------------------------------------------------------------

func (s *StorageTestSuite) TestDeleteFolder_R_FD006_CascadeDelete() {
	// R-FD006: On delete, subfolders are recursively deleted
	sessionID := s.registerAndGetToken("delete-cascade@example.com", "Password123", "Delete Cascade User")

	// Create parent folder
	parentResp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:      http.MethodPost,
		Path:        "/api/v1/folders",
		SessionID: sessionID,
		Body: map[string]interface{}{
			"name": "Parent",
		},
	})
	parentResp.AssertStatus(http.StatusCreated)
	parentID := parentResp.GetJSONData()["id"].(string)

	// Create child folder
	childResp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:      http.MethodPost,
		Path:        "/api/v1/folders",
		SessionID: sessionID,
		Body: map[string]interface{}{
			"name":      "Child",
			"parentId": parentID,
		},
	})
	childResp.AssertStatus(http.StatusCreated)
	childID := childResp.GetJSONData()["id"].(string)

	// Delete parent folder
	testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:      http.MethodDelete,
		Path:        "/api/v1/folders/" + parentID,
		SessionID: sessionID,
	}).AssertStatus(http.StatusNoContent)

	// Verify child folder is also deleted
	testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:      http.MethodGet,
		Path:        "/api/v1/folders/" + childID,
		SessionID: sessionID,
	}).AssertStatus(http.StatusNotFound)
}

// -----------------------------------------------------------------------------
// R-FD009: Personal Folder cannot be deleted
// -----------------------------------------------------------------------------

func (s *StorageTestSuite) TestDeleteFolder_R_FD009_PersonalFolderCannotBeDeleted() {
	// R-FD009: Personal Folder cannot be deleted
	sessionID := s.registerAndGetToken("pf-delete@example.com", "Password123", "Personal Folder Delete User")

	// Get the user's personal folder ID
	personalFolderID := s.getPersonalFolderID("pf-delete@example.com")

	// Try to delete personal folder - should fail with 403
	resp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:      http.MethodDelete,
		Path:        "/api/v1/folders/" + personalFolderID.String(),
		SessionID: sessionID,
	})

	resp.AssertStatus(http.StatusForbidden).
		AssertJSONError("FORBIDDEN", "personal folder cannot be deleted")
}

func (s *StorageTestSuite) TestDeleteFolder_R_FD009_PersonalFolderCannotBeMoved() {
	// R-FD009: Personal Folder cannot be moved
	sessionID := s.registerAndGetToken("pf-move@example.com", "Password123", "Personal Folder Move User")

	// Get the user's personal folder ID
	personalFolderID := s.getPersonalFolderID("pf-move@example.com")

	// Create a destination folder
	destResp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:      http.MethodPost,
		Path:        "/api/v1/folders",
		SessionID: sessionID,
		Body: map[string]interface{}{
			"name": "Destination",
		},
	})
	destResp.AssertStatus(http.StatusCreated)
	destID := destResp.GetJSONData()["id"].(string)

	// Try to move personal folder - should fail with 403
	resp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:      http.MethodPatch,
		Path:        "/api/v1/folders/" + personalFolderID.String() + "/move",
		SessionID: sessionID,
		Body: map[string]interface{}{
			"newParentId": destID,
		},
	})

	resp.AssertStatus(http.StatusForbidden).
		AssertJSONError("FORBIDDEN", "personal folder cannot be moved")
}

func (s *StorageTestSuite) TestDeleteFolder_R_FD009_PersonalFolderCannotBeRenamed() {
	// R-FD009: Personal Folder cannot be renamed
	sessionID := s.registerAndGetToken("pf-rename@example.com", "Password123", "Personal Folder Rename User")

	// Get the user's personal folder ID
	personalFolderID := s.getPersonalFolderID("pf-rename@example.com")

	// Try to rename personal folder - should fail with 403
	resp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:      http.MethodPatch,
		Path:        "/api/v1/folders/" + personalFolderID.String() + "/rename",
		SessionID: sessionID,
		Body: map[string]interface{}{
			"name": "New Personal Name",
		},
	})

	resp.AssertStatus(http.StatusForbidden).
		AssertJSONError("FORBIDDEN", "personal folder cannot be renamed")
}

// -----------------------------------------------------------------------------
// Folder Contents and Ancestors Tests
// -----------------------------------------------------------------------------

func (s *StorageTestSuite) TestListFolderContents_RootLevel() {
	sessionID := s.registerAndGetToken("list-root@example.com", "Password123", "List Root User")

	// Create folders at root
	testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:      http.MethodPost,
		Path:        "/api/v1/folders",
		SessionID: sessionID,
		Body: map[string]interface{}{
			"name": "Folder1",
		},
	}).AssertStatus(http.StatusCreated)

	testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:      http.MethodPost,
		Path:        "/api/v1/folders",
		SessionID: sessionID,
		Body: map[string]interface{}{
			"name": "Folder2",
		},
	}).AssertStatus(http.StatusCreated)

	// List root contents
	resp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:      http.MethodGet,
		Path:        "/api/v1/folders/root/contents",
		SessionID: sessionID,
	})

	resp.AssertStatus(http.StatusOK).
		AssertJSONPathExists("data.folders")
}

func (s *StorageTestSuite) TestGetAncestors_Breadcrumb() {
	sessionID := s.registerAndGetToken("ancestors@example.com", "Password123", "Ancestors User")

	// Create hierarchy: A -> B -> C
	aResp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:      http.MethodPost,
		Path:        "/api/v1/folders",
		SessionID: sessionID,
		Body: map[string]interface{}{
			"name": "FolderA",
		},
	})
	aResp.AssertStatus(http.StatusCreated)
	aID := aResp.GetJSONData()["id"].(string)

	bResp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:      http.MethodPost,
		Path:        "/api/v1/folders",
		SessionID: sessionID,
		Body: map[string]interface{}{
			"name":      "FolderB",
			"parentId": aID,
		},
	})
	bResp.AssertStatus(http.StatusCreated)
	bID := bResp.GetJSONData()["id"].(string)

	cResp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:      http.MethodPost,
		Path:        "/api/v1/folders",
		SessionID: sessionID,
		Body: map[string]interface{}{
			"name":      "FolderC",
			"parentId": bID,
		},
	})
	cResp.AssertStatus(http.StatusCreated)
	cID := cResp.GetJSONData()["id"].(string)

	// Get ancestors of C (should return [A, B])
	resp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:      http.MethodGet,
		Path:        "/api/v1/folders/" + cID + "/ancestors",
		SessionID: sessionID,
	})

	resp.AssertStatus(http.StatusOK)
	// Verify ancestors array exists (response uses BreadcrumbResponse with "items")
	data := resp.GetJSONData()
	ancestors, ok := data["items"].([]interface{})
	s.True(ok, "items should be an array")
	s.Len(ancestors, 2, "should have 2 ancestors")
}

// -----------------------------------------------------------------------------
// Authorization Tests
// -----------------------------------------------------------------------------

func (s *StorageTestSuite) TestFolder_UnauthorizedAccess() {
	// Create folder as user1
	sessionID1 := s.registerAndGetToken("user1@example.com", "Password123", "User1")
	folderResp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:      http.MethodPost,
		Path:        "/api/v1/folders",
		SessionID: sessionID1,
		Body: map[string]interface{}{
			"name": "User1Folder",
		},
	})
	folderResp.AssertStatus(http.StatusCreated)
	folderID := folderResp.GetJSONData()["id"].(string)

	// Try to access as user2
	sessionID2 := s.registerAndGetToken("user2@example.com", "Password123", "User2")
	resp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:      http.MethodGet,
		Path:        "/api/v1/folders/" + folderID,
		SessionID: sessionID2,
	})

	resp.AssertStatus(http.StatusForbidden)
}

// =============================================================================
// File Tests - Based on docs/03-domains/file.md requirements
// =============================================================================

// -----------------------------------------------------------------------------
// R-FL001: storage_key is generated from fileId
// R-FL002: Same folder, file name is unique
// -----------------------------------------------------------------------------

func (s *StorageTestSuite) TestInitiateUpload_Success() {
	sessionID := s.registerAndGetToken("upload-user@example.com", "Password123", "Upload User")

	// Create a folder first (folderId is required)
	folderResp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:      http.MethodPost,
		Path:        "/api/v1/folders",
		SessionID: sessionID,
		Body: map[string]interface{}{
			"name": "UploadFolder",
		},
	})
	folderResp.AssertStatus(http.StatusCreated)
	folderID := folderResp.GetJSONData()["id"].(string)

	resp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:      http.MethodPost,
		Path:        "/api/v1/files/upload",
		SessionID: sessionID,
		Body: map[string]interface{}{
			"folderId": folderID,
			"fileName": "test.txt",
			"mimeType": "text/plain",
			"size":     int64(1024),
		},
	})

	resp.AssertStatus(http.StatusCreated).
		AssertJSONPathExists("data.sessionId").
		AssertJSONPathExists("data.fileId").
		AssertJSONPathExists("data.uploadUrls").
		AssertJSONPathExists("data.expiresAt")
}

func (s *StorageTestSuite) TestInitiateUpload_R_FL002_DuplicateName() {
	// R-FL002: Same folder, file name is unique
	sessionID := s.registerAndGetToken("dup-file@example.com", "Password123", "Dup File User")

	// Create a folder
	folderResp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:      http.MethodPost,
		Path:        "/api/v1/folders",
		SessionID: sessionID,
		Body: map[string]interface{}{
			"name": "TestFolder",
		},
	})
	folderResp.AssertStatus(http.StatusCreated)
	folderID := folderResp.GetJSONData()["id"].(string)

	// Initiate first upload
	testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:      http.MethodPost,
		Path:        "/api/v1/files/upload",
		SessionID: sessionID,
		Body: map[string]interface{}{
			"folderId": folderID,
			"fileName": "duplicate.txt",
			"mimeType": "text/plain",
			"size":      int64(1024),
		},
	}).AssertStatus(http.StatusCreated)

	// Try to initiate second upload with same name in same folder - should fail
	resp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:      http.MethodPost,
		Path:        "/api/v1/files/upload",
		SessionID: sessionID,
		Body: map[string]interface{}{
			"folderId": folderID,
			"fileName": "duplicate.txt",
			"mimeType": "text/plain",
			"size":      int64(2048),
		},
	})

	resp.AssertStatus(http.StatusConflict).
		AssertJSONError("CONFLICT", "file with same name already exists")
}

func (s *StorageTestSuite) TestInitiateUpload_SameNameDifferentFolders_OK() {
	sessionID := s.registerAndGetToken("same-name-diff@example.com", "Password123", "Same Name Diff User")

	// Create two folders
	folder1Resp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:      http.MethodPost,
		Path:        "/api/v1/folders",
		SessionID: sessionID,
		Body: map[string]interface{}{
			"name": "Folder1",
		},
	})
	folder1Resp.AssertStatus(http.StatusCreated)
	folder1ID := folder1Resp.GetJSONData()["id"].(string)

	folder2Resp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:      http.MethodPost,
		Path:        "/api/v1/folders",
		SessionID: sessionID,
		Body: map[string]interface{}{
			"name": "Folder2",
		},
	})
	folder2Resp.AssertStatus(http.StatusCreated)
	folder2ID := folder2Resp.GetJSONData()["id"].(string)

	// Upload same name to different folders - should succeed
	testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:      http.MethodPost,
		Path:        "/api/v1/files/upload",
		SessionID: sessionID,
		Body: map[string]interface{}{
			"folderId": folder1ID,
			"fileName": "samename.txt",
			"mimeType": "text/plain",
			"size":      int64(1024),
		},
	}).AssertStatus(http.StatusCreated)

	testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:      http.MethodPost,
		Path:        "/api/v1/files/upload",
		SessionID: sessionID,
		Body: map[string]interface{}{
			"folderId": folder2ID,
			"fileName": "samename.txt",
			"mimeType": "text/plain",
			"size":      int64(1024),
		},
	}).AssertStatus(http.StatusCreated)
}

// -----------------------------------------------------------------------------
// R-US001: Session expires in 24 hours
// -----------------------------------------------------------------------------

func (s *StorageTestSuite) TestGetUploadStatus_SessionExpiry() {
	sessionID := s.registerAndGetToken("session-expiry@example.com", "Password123", "Session Expiry User")

	// Create a folder first (folderId is required)
	folderResp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:      http.MethodPost,
		Path:        "/api/v1/folders",
		SessionID: sessionID,
		Body: map[string]interface{}{
			"name": "ExpiryFolder",
		},
	})
	folderResp.AssertStatus(http.StatusCreated)
	folderID := folderResp.GetJSONData()["id"].(string)

	// Initiate upload
	resp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:      http.MethodPost,
		Path:        "/api/v1/files/upload",
		SessionID: sessionID,
		Body: map[string]interface{}{
			"folderId": folderID,
			"fileName": "expiry.txt",
			"mimeType": "text/plain",
			"size":     int64(1024),
		},
	})
	resp.AssertStatus(http.StatusCreated)
	uploadSessionID := resp.GetJSONData()["sessionId"].(string)

	// Get upload status
	statusResp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:    http.MethodGet,
		Path:      "/api/v1/files/upload/" + uploadSessionID,
		SessionID: sessionID,
	})

	statusResp.AssertStatus(http.StatusOK).
		AssertJSONPathExists("data.expiresAt")
}

// -----------------------------------------------------------------------------
// File Authorization Tests
// -----------------------------------------------------------------------------

func (s *StorageTestSuite) TestFile_UnauthorizedAccess() {
	// Create upload as user1
	sessionID1 := s.registerAndGetToken("fileuser1@example.com", "Password123", "FileUser1")

	// Create a folder first (folderId is required)
	folderResp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:      http.MethodPost,
		Path:        "/api/v1/folders",
		SessionID: sessionID1,
		Body: map[string]interface{}{
			"name": "PrivateFolder",
		},
	})
	folderResp.AssertStatus(http.StatusCreated)
	folderID := folderResp.GetJSONData()["id"].(string)

	uploadResp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:      http.MethodPost,
		Path:        "/api/v1/files/upload",
		SessionID: sessionID1,
		Body: map[string]interface{}{
			"folderId": folderID,
			"fileName": "private.txt",
			"mimeType": "text/plain",
			"size":     int64(1024),
		},
	})
	uploadResp.AssertStatus(http.StatusCreated)
	sessionID := uploadResp.GetJSONData()["sessionId"].(string)

	// Try to access upload status as user2
	sessionID2 := s.registerAndGetToken("fileuser2@example.com", "Password123", "FileUser2")
	resp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:      http.MethodGet,
		Path:        "/api/v1/files/upload/" + sessionID,
		SessionID: sessionID2,
	})

	resp.AssertStatus(http.StatusForbidden)
}

// =============================================================================
// Closure Table Tests - R-FC001-R-FC004
// =============================================================================

func (s *StorageTestSuite) TestClosureTable_R_FC001_SelfReference() {
	// R-FC001: Each folder has a self-reference entry (ancestor_id = descendant_id, path_length = 0)
	sessionID := s.registerAndGetToken("closure-self@example.com", "Password123", "Closure Self User")

	// Create folder
	resp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:      http.MethodPost,
		Path:        "/api/v1/folders",
		SessionID: sessionID,
		Body: map[string]interface{}{
			"name": "TestFolder",
		},
	})
	resp.AssertStatus(http.StatusCreated)
	folderID := resp.GetJSONData()["id"].(string)

	// Verify self-reference by checking ancestors of folder (should be empty)
	ancestorsResp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:      http.MethodGet,
		Path:        "/api/v1/folders/" + folderID + "/ancestors",
		SessionID: sessionID,
	})

	ancestorsResp.AssertStatus(http.StatusOK)
	data := ancestorsResp.GetJSONData()
	ancestors, ok := data["items"].([]interface{})
	s.True(ok, "items should be an array")
	s.Len(ancestors, 0, "root folder should have no ancestors")
}

func (s *StorageTestSuite) TestClosureTable_R_FC002_AncestorPaths() {
	// R-FC002: On folder creation, insert self-reference and all ancestor references
	sessionID := s.registerAndGetToken("closure-paths@example.com", "Password123", "Closure Paths User")

	// Create hierarchy: A -> B -> C -> D
	aResp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:      http.MethodPost,
		Path:        "/api/v1/folders",
		SessionID: sessionID,
		Body: map[string]interface{}{
			"name": "A",
		},
	})
	aResp.AssertStatus(http.StatusCreated)
	aID := aResp.GetJSONData()["id"].(string)

	bResp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:      http.MethodPost,
		Path:        "/api/v1/folders",
		SessionID: sessionID,
		Body: map[string]interface{}{
			"name":      "B",
			"parentId": aID,
		},
	})
	bResp.AssertStatus(http.StatusCreated)
	bID := bResp.GetJSONData()["id"].(string)

	cResp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:      http.MethodPost,
		Path:        "/api/v1/folders",
		SessionID: sessionID,
		Body: map[string]interface{}{
			"name":      "C",
			"parentId": bID,
		},
	})
	cResp.AssertStatus(http.StatusCreated)
	cID := cResp.GetJSONData()["id"].(string)

	dResp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:      http.MethodPost,
		Path:        "/api/v1/folders",
		SessionID: sessionID,
		Body: map[string]interface{}{
			"name":      "D",
			"parentId": cID,
		},
	})
	dResp.AssertStatus(http.StatusCreated)
	dID := dResp.GetJSONData()["id"].(string)

	// Get ancestors of D (should return [A, B, C] in root-to-leaf order)
	ancestorsResp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:      http.MethodGet,
		Path:        "/api/v1/folders/" + dID + "/ancestors",
		SessionID: sessionID,
	})

	ancestorsResp.AssertStatus(http.StatusOK)
	data := ancestorsResp.GetJSONData()
	ancestors, ok := data["items"].([]interface{})
	s.True(ok, "items should be an array")
	s.Len(ancestors, 3, "D should have 3 ancestors: A, B, C")

	// Verify order (root to leaf): A, B, C
	ancestorNames := make([]string, len(ancestors))
	for i, a := range ancestors {
		ancestor := a.(map[string]interface{})
		ancestorNames[i] = ancestor["name"].(string)
	}
	s.Equal([]string{"A", "B", "C"}, ancestorNames, "ancestors should be in root-to-leaf order")
}

// =============================================================================
// Helper Methods
// =============================================================================

// registerAndGetToken registers a user, activates them, and returns the access token
func (s *StorageTestSuite) registerAndGetToken(email, password, name string) string {
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

	// Activate user in database
	_, err := s.server.Pool.Exec(
		context.Background(),
		"UPDATE users SET status = 'active', email_verified_at = NOW() WHERE email = $1",
		email,
	)
	s.Require().NoError(err)

	// Login
	loginResp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method: http.MethodPost,
		Path:   "/api/v1/auth/login",
		Body: map[string]string{
			"email":    email,
			"password": password,
		},
	})
	loginResp.AssertStatus(http.StatusOK)

	cookie := loginResp.GetCookie("session_id")
	s.Require().NotNil(cookie, "session_id cookie should be set")
	return cookie.Value
}

// getPersonalFolderID retrieves the personal folder ID for a user by email
func (s *StorageTestSuite) getPersonalFolderID(email string) uuid.UUID {
	var personalFolderID uuid.UUID
	err := s.server.Pool.QueryRow(
		context.Background(),
		"SELECT personal_folder_id FROM users WHERE email = $1",
		email,
	).Scan(&personalFolderID)
	s.Require().NoError(err, "failed to get personal folder ID")
	return personalFolderID
}

// createFileInFolder is a helper to create a file in a folder and complete the upload
func (s *StorageTestSuite) createFileInFolder(sessionID string, folderID *uuid.UUID, fileName string) uuid.UUID {
	body := map[string]interface{}{
		"fileName": fileName,
		"mimeType": "text/plain",
		"size":     int64(1024),
	}
	if folderID != nil {
		body["folderId"] = folderID.String()
	}

	resp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:    http.MethodPost,
		Path:      "/api/v1/files/upload",
		SessionID: sessionID,
		Body:      body,
	})
	resp.AssertStatus(http.StatusCreated)

	data := resp.GetJSONData()
	fileID, _ := uuid.Parse(data["fileId"].(string))
	return fileID
}
