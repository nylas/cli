//go:build integration
// +build integration

package nylas_test

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/nylas/cli/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIntegration_GetFolders(t *testing.T) {
	client, grantID := getTestClient(t)
	ctx, cancel := createTestContext()
	defer cancel()

	folders, err := client.GetFolders(ctx, grantID)
	skipIfProviderNotSupported(t, err)
	require.NoError(t, err)
	require.NotEmpty(t, folders, "Should have at least one folder")

	t.Logf("Found %d folders", len(folders))

	standardFolders := []string{"inbox", "sent", "drafts", "trash", "spam", "archive"}
	foundStandard := make(map[string]bool)

	for _, f := range folders {
		assert.NotEmpty(t, f.ID, "Folder should have ID")
		assert.NotEmpty(t, f.Name, "Folder should have name")

		t.Logf("  [%s] %s (total: %d, unread: %d)",
			safeSubstring(f.ID, 8), f.Name, f.TotalCount, f.UnreadCount)

		// Track standard folders found
		nameLower := strings.ToLower(f.Name)
		for _, std := range standardFolders {
			if strings.Contains(nameLower, std) {
				foundStandard[std] = true
			}
		}
	}

	t.Logf("Standard folders found: %v", foundStandard)
}

func TestIntegration_GetSingleFolder(t *testing.T) {
	client, grantID := getTestClient(t)
	ctx, cancel := createTestContext()
	defer cancel()

	// First get folders
	folders, err := client.GetFolders(ctx, grantID)
	skipIfProviderNotSupported(t, err)
	require.NoError(t, err)
	require.NotEmpty(t, folders)

	folderID := folders[0].ID

	// Get single folder
	folder, err := client.GetFolder(ctx, grantID, folderID)
	require.NoError(t, err)

	assert.Equal(t, folderID, folder.ID)
	assert.NotEmpty(t, folder.Name)

	t.Logf("Folder: %s (%s)", folder.Name, folder.ID)
	t.Logf("Total messages: %d, Unread: %d", folder.TotalCount, folder.UnreadCount)
}

func TestIntegration_FolderLifecycle(t *testing.T) {
	client, grantID := getTestClient(t)
	ctx, cancel := createLongTestContext()
	defer cancel()

	// Create a unique folder name
	folderName := fmt.Sprintf("IntegrationTest_%d", time.Now().Unix())

	// Create folder
	createReq := &domain.CreateFolderRequest{
		Name: folderName,
	}

	folder, err := client.CreateFolder(ctx, grantID, createReq)
	skipIfProviderNotSupported(t, err)
	require.NoError(t, err)
	require.NotEmpty(t, folder.ID)
	assert.Equal(t, folderName, folder.Name)
	t.Logf("Created folder: %s (%s)", folder.Name, folder.ID)

	// Get the folder
	retrieved, err := client.GetFolder(ctx, grantID, folder.ID)
	require.NoError(t, err)
	assert.Equal(t, folder.ID, retrieved.ID)
	assert.Equal(t, folderName, retrieved.Name)

	// Update folder name
	newName := folderName + "_Updated"
	updateReq := &domain.UpdateFolderRequest{
		Name: newName,
	}
	updated, err := client.UpdateFolder(ctx, grantID, folder.ID, updateReq)
	require.NoError(t, err)
	assert.Equal(t, newName, updated.Name)
	t.Logf("Updated folder name to: %s", updated.Name)

	// Delete the folder
	err = client.DeleteFolder(ctx, grantID, folder.ID)
	require.NoError(t, err)
	t.Logf("Deleted folder: %s", folder.ID)

	// Verify deletion
	_, err = client.GetFolder(ctx, grantID, folder.ID)
	assert.Error(t, err, "Folder should be deleted")
}

func TestIntegration_GetFolder_NotFound(t *testing.T) {
	client, grantID := getTestClient(t)
	ctx, cancel := createTestContext()
	defer cancel()

	_, err := client.GetFolder(ctx, grantID, "nonexistent-folder-id-12345")
	assert.Error(t, err, "Should return error for non-existent folder")
	t.Logf("Expected error: %v", err)
}

// =============================================================================
// Thread Tests
// =============================================================================
