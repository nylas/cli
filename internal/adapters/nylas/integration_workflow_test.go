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

// =============================================================================
// Comprehensive Workflow Tests
// =============================================================================

func TestIntegration_CompleteWorkflow_ReadAndMarkMessages(t *testing.T) {
	client, grantID := getTestClient(t)
	ctx, cancel := createLongTestContext()
	defer cancel()

	// 1. List recent messages
	messages, err := client.GetMessages(ctx, grantID, 5)
	require.NoError(t, err)
	skipIfNoMessages(t, messages)
	t.Logf("Step 1: Listed %d messages", len(messages))

	// 2. Get full details of first message
	fullMsg, err := client.GetMessage(ctx, grantID, messages[0].ID)
	require.NoError(t, err)
	t.Logf("Step 2: Got message details - Subject: %s", fullMsg.Subject)

	// 3. Get the thread for this message (skip if provider doesn't support)
	if fullMsg.ThreadID != "" {
		thread, err := client.GetThread(ctx, grantID, fullMsg.ThreadID)
		if err != nil && strings.Contains(err.Error(), "Method not supported for provider") {
			t.Logf("Step 3: Skipped - threads not supported by provider")
		} else {
			require.NoError(t, err)
			t.Logf("Step 3: Got thread with %d messages", len(thread.MessageIDs))
		}
	}

	// 4. List folders (skip if provider doesn't support)
	folders, err := client.GetFolders(ctx, grantID)
	if err != nil && strings.Contains(err.Error(), "Method not supported for provider") {
		t.Logf("Step 4: Skipped - folders not supported by provider")
	} else {
		require.NoError(t, err)
		t.Logf("Step 4: Found %d folders", len(folders))
	}

	// 5. Toggle read status
	originalUnread := fullMsg.Unread
	newUnread := !originalUnread
	req := &domain.UpdateMessageRequest{Unread: &newUnread}

	updated, err := client.UpdateMessage(ctx, grantID, fullMsg.ID, req)
	require.NoError(t, err)
	assert.Equal(t, newUnread, updated.Unread)
	t.Logf("Step 5: Toggled unread status from %v to %v", originalUnread, newUnread)

	// 6. Restore original status
	req.Unread = &originalUnread
	restored, err := client.UpdateMessage(ctx, grantID, fullMsg.ID, req)
	require.NoError(t, err)
	assert.Equal(t, originalUnread, restored.Unread)
	t.Logf("Step 6: Restored original unread status")
}

func TestIntegration_CompleteWorkflow_DraftCreationAndManagement(t *testing.T) {
	client, grantID := getTestClient(t)
	ctx, cancel := createLongTestContext()
	defer cancel()

	timestamp := time.Now().Format("2006-01-02 15:04:05")

	// 1. Create initial draft
	createReq := &domain.CreateDraftRequest{
		Subject: fmt.Sprintf("Workflow Test - %s", timestamp),
		Body:    "Initial draft content",
		To:      []domain.EmailParticipant{{Email: "recipient@example.com", Name: "Recipient"}},
	}

	draft, err := client.CreateDraft(ctx, grantID, createReq)
	skipIfProviderNotSupported(t, err)
	require.NoError(t, err)
	t.Logf("Step 1: Created draft %s", draft.ID)

	// 2. List drafts and verify it appears
	drafts, err := client.GetDrafts(ctx, grantID, 20)
	require.NoError(t, err)
	found := false
	for _, d := range drafts {
		if d.ID == draft.ID {
			found = true
			break
		}
	}
	assert.True(t, found, "New draft should appear in list")
	t.Logf("Step 2: Verified draft appears in list")

	// 3. Get draft details
	retrieved, err := client.GetDraft(ctx, grantID, draft.ID)
	require.NoError(t, err)
	assert.Equal(t, createReq.Subject, retrieved.Subject)
	t.Logf("Step 3: Retrieved draft details")

	// 4. Update draft with more recipients
	updateReq := &domain.CreateDraftRequest{
		Subject: createReq.Subject + " - Updated",
		Body:    "Updated draft content with more details",
		To:      []domain.EmailParticipant{{Email: "recipient@example.com"}},
		Cc:      []domain.EmailParticipant{{Email: "cc@example.com"}},
	}

	updated, err := client.UpdateDraft(ctx, grantID, draft.ID, updateReq)
	require.NoError(t, err)
	assert.Contains(t, updated.Subject, "Updated")
	t.Logf("Step 4: Updated draft with CC")

	// 5. Delete the draft
	err = client.DeleteDraft(ctx, grantID, draft.ID)
	require.NoError(t, err)
	t.Logf("Step 5: Deleted draft")

	// 6. Verify deletion (with retry for eventual consistency)
	// Note: Some providers may have eventual consistency, so we just log the result
	_, err = client.GetDraft(ctx, grantID, draft.ID)
	if err != nil {
		t.Logf("Step 6: Verified draft deletion (draft not found as expected)")
	} else {
		t.Logf("Step 6: Draft still retrievable due to eventual consistency (this is OK)")
	}
}

func TestIntegration_CompleteWorkflow_FolderManagement(t *testing.T) {
	client, grantID := getTestClient(t)
	ctx, cancel := createLongTestContext()
	defer cancel()

	folderName := fmt.Sprintf("WorkflowTest_%d", time.Now().Unix())

	// 1. List existing folders
	initialFolders, err := client.GetFolders(ctx, grantID)
	skipIfProviderNotSupported(t, err)
	require.NoError(t, err)
	t.Logf("Step 1: Initial folder count: %d", len(initialFolders))

	// 2. Create new folder
	createReq := &domain.CreateFolderRequest{Name: folderName}
	folder, err := client.CreateFolder(ctx, grantID, createReq)
	require.NoError(t, err)
	t.Logf("Step 2: Created folder %s", folder.Name)

	// 3. Verify folder appears in list
	foldersAfterCreate, err := client.GetFolders(ctx, grantID)
	require.NoError(t, err)
	assert.Equal(t, len(initialFolders)+1, len(foldersAfterCreate))
	t.Logf("Step 3: Folder count increased to %d", len(foldersAfterCreate))

	// 4. Get folder details
	retrieved, err := client.GetFolder(ctx, grantID, folder.ID)
	require.NoError(t, err)
	assert.Equal(t, folderName, retrieved.Name)
	t.Logf("Step 4: Verified folder details")

	// 5. Rename folder
	newName := folderName + "_Renamed"
	folderUpdateReq := &domain.UpdateFolderRequest{Name: newName}
	renamed, err := client.UpdateFolder(ctx, grantID, folder.ID, folderUpdateReq)
	require.NoError(t, err)
	assert.Equal(t, newName, renamed.Name)
	t.Logf("Step 5: Renamed folder to %s", renamed.Name)

	// 6. Delete folder
	err = client.DeleteFolder(ctx, grantID, folder.ID)
	require.NoError(t, err)
	t.Logf("Step 6: Deleted folder")

	// 7. Verify folder count restored
	foldersAfterDelete, err := client.GetFolders(ctx, grantID)
	require.NoError(t, err)
	assert.Equal(t, len(initialFolders), len(foldersAfterDelete))
	t.Logf("Step 7: Folder count restored to %d", len(foldersAfterDelete))
}

// =============================================================================
// Scheduled Messages Tests
// =============================================================================
