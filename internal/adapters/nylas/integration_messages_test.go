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

func TestIntegration_GetMessages(t *testing.T) {
	client, grantID := getTestClient(t)
	ctx, cancel := createTestContext()
	defer cancel()

	messages, err := client.GetMessages(ctx, grantID, 10)
	require.NoError(t, err, "GetMessages should not return error")

	t.Logf("Found %d messages", len(messages))
	for _, m := range messages {
		from := ""
		if len(m.From) > 0 {
			from = m.From[0].Email
		}
		t.Logf("  [%s] %s - %s (%s)",
			safeSubstring(m.ID, 8), from, safeSubstring(m.Subject, 40), m.Date.Format("Jan 2, 15:04"))

		// Validate message fields
		assert.NotEmpty(t, m.ID, "Message should have ID")
		assert.NotZero(t, m.Date, "Message should have date")
	}
}

func TestIntegration_GetMessages_LimitRespected(t *testing.T) {
	client, grantID := getTestClient(t)
	ctx, cancel := createTestContext()
	defer cancel()

	limits := []int{1, 5, 10, 25}

	for _, limit := range limits {
		t.Run(fmt.Sprintf("Limit_%d", limit), func(t *testing.T) {
			messages, err := client.GetMessages(ctx, grantID, limit)
			require.NoError(t, err)
			assert.LessOrEqual(t, len(messages), limit, "Should not exceed requested limit")
			t.Logf("Requested %d, got %d messages", limit, len(messages))
		})
	}
}

func TestIntegration_GetMessagesWithParams_Unread(t *testing.T) {
	client, grantID := getTestClient(t)
	ctx, cancel := createTestContext()
	defer cancel()

	unread := true
	params := &domain.MessageQueryParams{
		Limit:  10,
		Unread: &unread,
	}

	messages, err := client.GetMessagesWithParams(ctx, grantID, params)
	require.NoError(t, err)

	t.Logf("Found %d unread messages", len(messages))
	for _, m := range messages {
		assert.True(t, m.Unread, "All messages should be unread when filtering by unread=true")
		t.Logf("  Subject: %s, Unread: %v", safeSubstring(m.Subject, 50), m.Unread)
	}
}

func TestIntegration_GetMessagesWithParams_Read(t *testing.T) {
	client, grantID := getTestClient(t)
	ctx, cancel := createTestContext()
	defer cancel()

	unread := false
	params := &domain.MessageQueryParams{
		Limit:  10,
		Unread: &unread,
	}

	messages, err := client.GetMessagesWithParams(ctx, grantID, params)
	require.NoError(t, err)

	t.Logf("Found %d read messages", len(messages))
	for _, m := range messages {
		assert.False(t, m.Unread, "All messages should be read when filtering by unread=false")
	}
}

func TestIntegration_GetMessagesWithParams_Starred(t *testing.T) {
	client, grantID := getTestClient(t)
	ctx, cancel := createTestContext()
	defer cancel()

	starred := true
	params := &domain.MessageQueryParams{
		Limit:   10,
		Starred: &starred,
	}

	messages, err := client.GetMessagesWithParams(ctx, grantID, params)
	require.NoError(t, err)

	t.Logf("Found %d starred messages", len(messages))
	for _, m := range messages {
		assert.True(t, m.Starred, "All messages should be starred when filtering by starred=true")
	}
}

func TestIntegration_GetMessagesWithParams_InFolder(t *testing.T) {
	client, grantID := getTestClient(t)
	ctx, cancel := createTestContext()
	defer cancel()

	// First get folders
	folders, err := client.GetFolders(ctx, grantID)
	skipIfProviderNotSupported(t, err)
	require.NoError(t, err)
	require.NotEmpty(t, folders)

	// Find INBOX or first folder
	var targetFolder *domain.Folder
	for i := range folders {
		if strings.EqualFold(folders[i].Name, "INBOX") || strings.EqualFold(folders[i].Name, "Inbox") {
			targetFolder = &folders[i]
			break
		}
	}
	if targetFolder == nil {
		targetFolder = &folders[0]
	}

	params := &domain.MessageQueryParams{
		Limit: 5,
		In:    []string{targetFolder.ID},
	}

	messages, err := client.GetMessagesWithParams(ctx, grantID, params)
	require.NoError(t, err)

	t.Logf("Found %d messages in folder '%s' (%s)", len(messages), targetFolder.Name, targetFolder.ID)
	for _, m := range messages {
		t.Logf("  Subject: %s", safeSubstring(m.Subject, 50))
	}
}

func TestIntegration_GetSingleMessage(t *testing.T) {
	client, grantID := getTestClient(t)
	ctx, cancel := createTestContext()
	defer cancel()

	// First get a list of messages
	messages, err := client.GetMessages(ctx, grantID, 1)
	require.NoError(t, err)
	skipIfNoMessages(t, messages)

	messageID := messages[0].ID

	// Now get the single message
	msg, err := client.GetMessage(ctx, grantID, messageID)
	require.NoError(t, err)

	assert.Equal(t, messageID, msg.ID)
	assert.NotEmpty(t, msg.Subject, "Message should have subject")
	assert.NotZero(t, msg.Date, "Message should have date")

	t.Logf("Message ID: %s", msg.ID)
	t.Logf("Subject: %s", msg.Subject)
	t.Logf("From: %v", msg.From)
	t.Logf("To: %v", msg.To)
	t.Logf("Date: %s", msg.Date.Format(time.RFC3339))
	t.Logf("Body length: %d chars", len(msg.Body))
	t.Logf("Snippet: %s", safeSubstring(msg.Snippet, 100))
	t.Logf("Unread: %v, Starred: %v", msg.Unread, msg.Starred)
	t.Logf("Thread ID: %s", msg.ThreadID)
	t.Logf("Attachments: %d", len(msg.Attachments))
}

func TestIntegration_GetSingleMessage_FullContent(t *testing.T) {
	client, grantID := getTestClient(t)
	ctx, cancel := createTestContext()
	defer cancel()

	messages, err := client.GetMessages(ctx, grantID, 5)
	require.NoError(t, err)
	skipIfNoMessages(t, messages)

	// Get full details for each message
	for _, m := range messages {
		fullMsg, err := client.GetMessage(ctx, grantID, m.ID)
		require.NoError(t, err)

		// Full message should have more details
		assert.Equal(t, m.ID, fullMsg.ID)
		t.Logf("[%s] Subject: %s, Body: %d chars, Attachments: %d",
			safeSubstring(m.ID, 8), safeSubstring(fullMsg.Subject, 30), len(fullMsg.Body), len(fullMsg.Attachments))
	}
}

func TestIntegration_GetMessage_NotFound(t *testing.T) {
	client, grantID := getTestClient(t)
	ctx, cancel := createTestContext()
	defer cancel()

	_, err := client.GetMessage(ctx, grantID, "nonexistent-message-id-12345")
	assert.Error(t, err, "Should return error for non-existent message")
	t.Logf("Expected error: %v", err)
}

// =============================================================================
// Message Tests - Mark Operations
// =============================================================================

func TestIntegration_MarkMessageReadUnread(t *testing.T) {
	client, grantID := getTestClient(t)
	ctx, cancel := createLongTestContext()
	defer cancel()

	// Get a message to test with
	messages, err := client.GetMessages(ctx, grantID, 1)
	require.NoError(t, err)
	skipIfNoMessages(t, messages)

	messageID := messages[0].ID
	originalUnread := messages[0].Unread
	t.Logf("Original message unread status: %v", originalUnread)

	// Mark as opposite of current state
	newUnread := !originalUnread
	req := &domain.UpdateMessageRequest{
		Unread: &newUnread,
	}

	updated, err := client.UpdateMessage(ctx, grantID, messageID, req)
	require.NoError(t, err)
	assert.Equal(t, newUnread, updated.Unread, "Unread status should be updated")
	t.Logf("Updated unread status to: %v", updated.Unread)

	// Restore original state
	req.Unread = &originalUnread
	restored, err := client.UpdateMessage(ctx, grantID, messageID, req)
	require.NoError(t, err)
	assert.Equal(t, originalUnread, restored.Unread, "Unread status should be restored")
	t.Logf("Restored unread status to: %v", restored.Unread)
}

func TestIntegration_MarkMessageStarred(t *testing.T) {
	client, grantID := getTestClient(t)
	ctx, cancel := createLongTestContext()
	defer cancel()

	messages, err := client.GetMessages(ctx, grantID, 1)
	require.NoError(t, err)
	skipIfNoMessages(t, messages)

	messageID := messages[0].ID
	originalStarred := messages[0].Starred
	t.Logf("Original starred status: %v", originalStarred)

	// Toggle starred status
	newStarred := !originalStarred
	req := &domain.UpdateMessageRequest{
		Starred: &newStarred,
	}

	updated, err := client.UpdateMessage(ctx, grantID, messageID, req)
	require.NoError(t, err)
	assert.Equal(t, newStarred, updated.Starred)
	t.Logf("Updated starred status to: %v", updated.Starred)

	// Restore
	req.Starred = &originalStarred
	restored, err := client.UpdateMessage(ctx, grantID, messageID, req)
	require.NoError(t, err)
	assert.Equal(t, originalStarred, restored.Starred)
	t.Logf("Restored starred status to: %v", restored.Starred)
}

func TestIntegration_UpdateMessage_MultipleFlagsAtOnce(t *testing.T) {
	client, grantID := getTestClient(t)
	ctx, cancel := createLongTestContext()
	defer cancel()

	messages, err := client.GetMessages(ctx, grantID, 1)
	require.NoError(t, err)
	skipIfNoMessages(t, messages)

	messageID := messages[0].ID
	originalUnread := messages[0].Unread
	originalStarred := messages[0].Starred

	// Update both flags at once
	newUnread := !originalUnread
	newStarred := !originalStarred
	req := &domain.UpdateMessageRequest{
		Unread:  &newUnread,
		Starred: &newStarred,
	}

	updated, err := client.UpdateMessage(ctx, grantID, messageID, req)
	require.NoError(t, err)
	assert.Equal(t, newUnread, updated.Unread)
	assert.Equal(t, newStarred, updated.Starred)
	t.Logf("Updated both flags: unread=%v, starred=%v", updated.Unread, updated.Starred)

	// Restore
	req = &domain.UpdateMessageRequest{
		Unread:  &originalUnread,
		Starred: &originalStarred,
	}
	restored, err := client.UpdateMessage(ctx, grantID, messageID, req)
	require.NoError(t, err)
	assert.Equal(t, originalUnread, restored.Unread)
	assert.Equal(t, originalStarred, restored.Starred)
	t.Logf("Restored both flags: unread=%v, starred=%v", restored.Unread, restored.Starred)
}

// =============================================================================
// Search Tests
// =============================================================================

func TestIntegration_SearchMessages_BySubject(t *testing.T) {
	client, grantID := getTestClient(t)
	ctx, cancel := createTestContext()
	defer cancel()

	// First get some messages to find a search term
	messages, err := client.GetMessages(ctx, grantID, 5)
	require.NoError(t, err)
	skipIfNoMessages(t, messages)

	// Use a word from the first message's subject as search term
	searchTerm := ""
	for _, m := range messages {
		words := strings.Fields(m.Subject)
		for _, w := range words {
			if len(w) > 3 {
				searchTerm = w
				break
			}
		}
		if searchTerm != "" {
			break
		}
	}

	if searchTerm == "" {
		t.Skip("Could not find suitable search term")
	}

	// Search by subject field instead of full-text search
	params := &domain.MessageQueryParams{
		Limit:   20,
		Subject: searchTerm,
	}

	results, err := client.GetMessagesWithParams(ctx, grantID, params)
	require.NoError(t, err)

	t.Logf("Search for subject containing '%s' returned %d results", searchTerm, len(results))
	for _, m := range results {
		t.Logf("  Subject: %s", safeSubstring(m.Subject, 60))
	}
}

func TestIntegration_SearchMessages_ByFrom(t *testing.T) {
	client, grantID := getTestClient(t)
	ctx, cancel := createTestContext()
	defer cancel()

	// Get messages to find a sender
	messages, err := client.GetMessages(ctx, grantID, 10)
	require.NoError(t, err)
	skipIfNoMessages(t, messages)

	var fromEmail string
	for _, m := range messages {
		if len(m.From) > 0 && m.From[0].Email != "" {
			fromEmail = m.From[0].Email
			break
		}
	}

	if fromEmail == "" {
		t.Skip("Could not find sender email")
	}

	params := &domain.MessageQueryParams{
		Limit: 10,
		From:  fromEmail,
	}

	results, err := client.GetMessagesWithParams(ctx, grantID, params)
	require.NoError(t, err)

	t.Logf("Search from '%s' returned %d results", fromEmail, len(results))
	for _, m := range results {
		foundFrom := false
		for _, f := range m.From {
			if strings.EqualFold(f.Email, fromEmail) {
				foundFrom = true
				break
			}
		}
		assert.True(t, foundFrom, "Message should be from %s", fromEmail)
	}
}

func TestIntegration_SearchMessages_DateRange(t *testing.T) {
	client, grantID := getTestClient(t)
	ctx, cancel := createTestContext()
	defer cancel()

	// Search for messages from the last 7 days
	now := time.Now()
	weekAgo := now.AddDate(0, 0, -7)

	params := &domain.MessageQueryParams{
		Limit:         20,
		ReceivedAfter: weekAgo.Unix(),
	}

	results, err := client.GetMessagesWithParams(ctx, grantID, params)
	require.NoError(t, err)

	t.Logf("Found %d messages from last 7 days", len(results))
	for _, m := range results {
		assert.True(t, m.Date.After(weekAgo) || m.Date.Equal(weekAgo),
			"Message date %v should be after %v", m.Date, weekAgo)
		t.Logf("  [%s] %s", m.Date.Format("Jan 2"), safeSubstring(m.Subject, 50))
	}
}

func TestIntegration_SearchMessages_Combined(t *testing.T) {
	client, grantID := getTestClient(t)
	ctx, cancel := createTestContext()
	defer cancel()

	// Combined search: unread messages from last 30 days
	unread := true
	monthAgo := time.Now().AddDate(0, 0, -30)

	params := &domain.MessageQueryParams{
		Limit:         10,
		Unread:        &unread,
		ReceivedAfter: monthAgo.Unix(),
	}

	results, err := client.GetMessagesWithParams(ctx, grantID, params)
	require.NoError(t, err)

	t.Logf("Found %d unread messages from last 30 days", len(results))
	for _, m := range results {
		assert.True(t, m.Unread, "Message should be unread")
	}
}

// =============================================================================
// Folder Tests
// =============================================================================
