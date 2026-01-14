//go:build integration
// +build integration

package nylas_test

import (
	"testing"

	"github.com/nylas/cli/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIntegration_GetThreads(t *testing.T) {
	client, grantID := getTestClient(t)
	ctx, cancel := createTestContext()
	defer cancel()

	params := &domain.ThreadQueryParams{
		Limit: 10,
	}

	threads, err := client.GetThreads(ctx, grantID, params)
	skipIfProviderNotSupported(t, err)
	require.NoError(t, err)

	t.Logf("Found %d threads", len(threads))
	for _, th := range threads {
		assert.NotEmpty(t, th.ID, "Thread should have ID")

		t.Logf("  [%s] %s (%d messages, unread: %v, starred: %v)",
			safeSubstring(th.ID, 8), safeSubstring(th.Subject, 40),
			len(th.MessageIDs), th.Unread, th.Starred)
	}
}

func TestIntegration_GetThreads_WithParams(t *testing.T) {
	client, grantID := getTestClient(t)
	ctx, cancel := createTestContext()
	defer cancel()

	// Get unread threads
	unread := true
	params := &domain.ThreadQueryParams{
		Limit:  5,
		Unread: &unread,
	}

	threads, err := client.GetThreads(ctx, grantID, params)
	skipIfProviderNotSupported(t, err)
	require.NoError(t, err)

	t.Logf("Found %d unread threads", len(threads))
	for _, th := range threads {
		assert.True(t, th.Unread, "Thread should be unread")
	}
}

func TestIntegration_GetSingleThread(t *testing.T) {
	client, grantID := getTestClient(t)
	ctx, cancel := createTestContext()
	defer cancel()

	// Get threads first
	params := &domain.ThreadQueryParams{
		Limit: 1,
	}
	threads, err := client.GetThreads(ctx, grantID, params)
	skipIfProviderNotSupported(t, err)
	require.NoError(t, err)
	require.NotEmpty(t, threads)

	threadID := threads[0].ID

	// Get single thread
	thread, err := client.GetThread(ctx, grantID, threadID)
	require.NoError(t, err)

	assert.Equal(t, threadID, thread.ID)
	assert.NotEmpty(t, thread.MessageIDs, "Thread should have message IDs")

	t.Logf("Thread: %s", thread.ID)
	t.Logf("Subject: %s", thread.Subject)
	t.Logf("Messages: %d", len(thread.MessageIDs))
	t.Logf("Participants: %v", thread.Participants)
	t.Logf("Latest Message ID: %s", thread.LatestDraftOrMessage.ID)
}

func TestIntegration_UpdateThread(t *testing.T) {
	client, grantID := getTestClient(t)
	ctx, cancel := createLongTestContext()
	defer cancel()

	// Get a thread
	params := &domain.ThreadQueryParams{
		Limit: 1,
	}
	threads, err := client.GetThreads(ctx, grantID, params)
	skipIfProviderNotSupported(t, err)
	require.NoError(t, err)
	require.NotEmpty(t, threads)

	threadID := threads[0].ID
	originalUnread := threads[0].Unread
	originalStarred := threads[0].Starred

	// Update thread
	newUnread := !originalUnread
	newStarred := !originalStarred
	req := &domain.UpdateMessageRequest{
		Unread:  &newUnread,
		Starred: &newStarred,
	}

	updated, err := client.UpdateThread(ctx, grantID, threadID, req)
	require.NoError(t, err)
	assert.Equal(t, newUnread, updated.Unread)
	assert.Equal(t, newStarred, updated.Starred)
	t.Logf("Updated thread: unread=%v, starred=%v", updated.Unread, updated.Starred)

	// Restore
	req = &domain.UpdateMessageRequest{
		Unread:  &originalUnread,
		Starred: &originalStarred,
	}
	restored, err := client.UpdateThread(ctx, grantID, threadID, req)
	require.NoError(t, err)
	assert.Equal(t, originalUnread, restored.Unread)
	assert.Equal(t, originalStarred, restored.Starred)
	t.Logf("Restored thread: unread=%v, starred=%v", restored.Unread, restored.Starred)
}

func TestIntegration_GetThread_NotFound(t *testing.T) {
	client, grantID := getTestClient(t)
	ctx, cancel := createTestContext()
	defer cancel()

	_, err := client.GetThread(ctx, grantID, "nonexistent-thread-id-12345")
	assert.Error(t, err, "Should return error for non-existent thread")
}

// =============================================================================
// Draft Tests
// =============================================================================
