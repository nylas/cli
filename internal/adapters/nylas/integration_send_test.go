//go:build integration
// +build integration

package nylas_test

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/nylas/cli/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIntegration_SendMessage(t *testing.T) {
	if os.Getenv("NYLAS_TEST_SEND_EMAIL") != "true" {
		t.Skip("Skipping send email test - set NYLAS_TEST_SEND_EMAIL=true to enable")
	}

	client, grantID := getTestClient(t)
	ctx, cancel := createLongTestContext()
	defer cancel()

	testEmail := os.Getenv("NYLAS_TEST_EMAIL")
	if testEmail == "" {
		t.Skip("NYLAS_TEST_EMAIL not set")
	}

	timestamp := time.Now().Format("2006-01-02 15:04:05")
	req := &domain.SendMessageRequest{
		Subject: fmt.Sprintf("Integration Test - %s", timestamp),
		Body:    "<html><body><h1>Test Email</h1><p>This is a test email sent by the integration tests at " + timestamp + "</p></body></html>",
		To:      []domain.EmailParticipant{{Email: testEmail, Name: "Test Recipient"}},
	}

	msg, err := client.SendMessage(ctx, grantID, req)
	require.NoError(t, err)
	require.NotEmpty(t, msg.ID)

	t.Logf("Sent email: %s", msg.ID)
	t.Logf("Subject: %s", msg.Subject)
	t.Logf("To: %v", msg.To)
}

func TestIntegration_SendMessage_WithCC(t *testing.T) {
	if os.Getenv("NYLAS_TEST_SEND_EMAIL") != "true" {
		t.Skip("Skipping send email test - set NYLAS_TEST_SEND_EMAIL=true to enable")
	}

	client, grantID := getTestClient(t)
	ctx, cancel := createLongTestContext()
	defer cancel()

	testEmail := os.Getenv("NYLAS_TEST_EMAIL")
	ccEmail := os.Getenv("NYLAS_TEST_CC_EMAIL")
	if testEmail == "" || ccEmail == "" {
		t.Skip("NYLAS_TEST_EMAIL and NYLAS_TEST_CC_EMAIL required")
	}

	req := &domain.SendMessageRequest{
		Subject: fmt.Sprintf("Integration Test with CC - %s", time.Now().Format("15:04:05")),
		Body:    "Test email with CC recipient",
		To:      []domain.EmailParticipant{{Email: testEmail}},
		Cc:      []domain.EmailParticipant{{Email: ccEmail}},
	}

	msg, err := client.SendMessage(ctx, grantID, req)
	require.NoError(t, err)
	t.Logf("Sent email with CC: %s", msg.ID)
}

func TestIntegration_SendDraft(t *testing.T) {
	if os.Getenv("NYLAS_TEST_SEND_EMAIL") != "true" {
		t.Skip("Skipping send email test - set NYLAS_TEST_SEND_EMAIL=true to enable")
	}

	client, grantID := getTestClient(t)
	ctx, cancel := createLongTestContext()
	defer cancel()

	testEmail := os.Getenv("NYLAS_TEST_EMAIL")
	if testEmail == "" {
		t.Skip("NYLAS_TEST_EMAIL not set")
	}

	// Create a draft first
	createReq := &domain.CreateDraftRequest{
		Subject: fmt.Sprintf("Draft to Send - %s", time.Now().Format("15:04:05")),
		Body:    "This draft will be sent as an email",
		To:      []domain.EmailParticipant{{Email: testEmail}},
	}

	draft, err := client.CreateDraft(ctx, grantID, createReq)
	require.NoError(t, err)
	t.Logf("Created draft: %s", draft.ID)

	// Send the draft
	msg, err := client.SendDraft(ctx, grantID, draft.ID)
	require.NoError(t, err)
	require.NotEmpty(t, msg.ID)
	t.Logf("Sent draft as message: %s", msg.ID)
}

// =============================================================================
// Delete Message Tests (Destructive - requires explicit opt-in)
// =============================================================================

func TestIntegration_DeleteMessage(t *testing.T) {
	if os.Getenv("NYLAS_TEST_DELETE_MESSAGE") != "true" {
		t.Skip("Skipping delete message test - set NYLAS_TEST_DELETE_MESSAGE=true to enable")
	}

	client, grantID := getTestClient(t)
	ctx, cancel := createLongTestContext()
	defer cancel()

	// Create a draft to delete (safer than deleting real messages)
	createReq := &domain.CreateDraftRequest{
		Subject: fmt.Sprintf("Draft for Deletion Test - %s", time.Now().Format("15:04:05")),
		Body:    "This draft will be deleted",
		To:      []domain.EmailParticipant{{Email: "test@example.com"}},
	}

	draft, err := client.CreateDraft(ctx, grantID, createReq)
	require.NoError(t, err)
	t.Logf("Created draft for deletion: %s", draft.ID)

	// Delete it (drafts can be deleted as messages too in some cases)
	err = client.DeleteDraft(ctx, grantID, draft.ID)
	require.NoError(t, err)
	t.Logf("Deleted draft: %s", draft.ID)

	// Verify
	_, err = client.GetDraft(ctx, grantID, draft.ID)
	assert.Error(t, err, "Draft should be deleted")
}

// =============================================================================
// Error Handling Tests
// =============================================================================

func TestIntegration_InvalidGrantID(t *testing.T) {
	client, _ := getTestClient(t)
	ctx, cancel := createTestContext()
	defer cancel()

	_, err := client.GetMessages(ctx, "invalid-grant-id-12345", 10)
	assert.Error(t, err, "Should return error for invalid grant ID")
	t.Logf("Expected error: %v", err)
}

func TestIntegration_EmptyMessageID(t *testing.T) {
	client, grantID := getTestClient(t)
	ctx, cancel := createTestContext()
	defer cancel()

	_, err := client.GetMessage(ctx, grantID, "")
	assert.Error(t, err, "Should return error for empty message ID")
}

// =============================================================================
// Concurrency Tests
// =============================================================================

func TestIntegration_ConcurrentRequests(t *testing.T) {
	client, grantID := getTestClient(t)
	ctx, cancel := createLongTestContext()
	defer cancel()

	// Run multiple concurrent requests (reduced to avoid rate limiting)
	const numRequests = 2
	results := make(chan error, numRequests)

	for i := 0; i < numRequests; i++ {
		go func(i int) {
			_, err := client.GetMessages(ctx, grantID, 3)
			results <- err
		}(i)
	}

	// Collect results - allow some rate limiting errors
	successCount := 0
	for i := 0; i < numRequests; i++ {
		err := <-results
		if err == nil {
			successCount++
		} else {
			t.Logf("Request %d hit rate limit (expected with some providers): %v", i, err)
		}
	}

	assert.Greater(t, successCount, 0, "At least one concurrent request should succeed")
	t.Logf("%d of %d concurrent requests completed successfully", successCount, numRequests)
}

func TestIntegration_ConcurrentDifferentOperations(t *testing.T) {
	client, grantID := getTestClient(t)
	ctx, cancel := createLongTestContext()
	defer cancel()

	type result struct {
		name string
		err  error
	}

	results := make(chan result, 4)

	// Run different operations concurrently
	go func() {
		_, err := client.GetMessages(ctx, grantID, 3)
		results <- result{"GetMessages", err}
	}()

	go func() {
		_, err := client.GetFolders(ctx, grantID)
		results <- result{"GetFolders", err}
	}()

	go func() {
		_, err := client.GetThreads(ctx, grantID, &domain.ThreadQueryParams{Limit: 3})
		results <- result{"GetThreads", err}
	}()

	go func() {
		_, err := client.GetDrafts(ctx, grantID, 3)
		results <- result{"GetDrafts", err}
	}()

	// Collect results - allow some operations to fail if provider doesn't support them
	successCount := 0
	for i := 0; i < 4; i++ {
		r := <-results
		if r.err != nil {
			errMsg := r.err.Error()
			if strings.Contains(errMsg, "Method not supported for provider") ||
				strings.Contains(errMsg, "an internal error ocurred") ||
				strings.Contains(errMsg, "an internal error occurred") {
				t.Logf("%s: Skipped (not supported by provider)", r.name)
			} else {
				t.Logf("%s: Error: %v", r.name, r.err)
			}
		} else {
			successCount++
			t.Logf("%s: OK", r.name)
		}
	}
	assert.Greater(t, successCount, 0, "At least one operation should succeed")
}

// =============================================================================
// Rate Limiting / Timeout Tests
// =============================================================================

func TestIntegration_RequestTimeout(t *testing.T) {
	client, grantID := getTestClient(t)

	// Create a very short timeout context
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()

	_, err := client.GetMessages(ctx, grantID, 100)
	assert.Error(t, err, "Should return error on timeout")
	t.Logf("Timeout error (expected): %v", err)
}

// =============================================================================
// Data Validation Tests
// =============================================================================

func TestIntegration_MessageFieldsValidation(t *testing.T) {
	client, grantID := getTestClient(t)
	ctx, cancel := createTestContext()
	defer cancel()

	messages, err := client.GetMessages(ctx, grantID, 10)
	skipIfRateLimited(t, err)
	require.NoError(t, err)
	skipIfNoMessages(t, messages)

	for _, m := range messages {
		t.Run("Message_"+safeSubstring(m.ID, 8), func(t *testing.T) {
			// Required fields
			assert.NotEmpty(t, m.ID, "ID should not be empty")
			assert.NotZero(t, m.Date, "Date should not be zero")

			// From should have at least one contact for received messages
			if len(m.From) > 0 {
				for _, f := range m.From {
					assert.NotEmpty(t, f.Email, "From email should not be empty")
				}
			}

			// Boolean fields should be set (even if false)
			// Just checking they're accessible
			_ = m.Unread
			_ = m.Starred
		})
	}
}

func TestIntegration_FolderFieldsValidation(t *testing.T) {
	client, grantID := getTestClient(t)
	ctx, cancel := createTestContext()
	defer cancel()

	folders, err := client.GetFolders(ctx, grantID)
	skipIfProviderNotSupported(t, err)
	require.NoError(t, err)
	require.NotEmpty(t, folders)

	for _, f := range folders {
		t.Run("Folder_"+safeSubstring(f.Name, 20), func(t *testing.T) {
			assert.NotEmpty(t, f.ID, "ID should not be empty")
			assert.NotEmpty(t, f.Name, "Name should not be empty")
			assert.GreaterOrEqual(t, f.TotalCount, 0, "TotalCount should be >= 0")
			assert.GreaterOrEqual(t, f.UnreadCount, 0, "UnreadCount should be >= 0")
			assert.LessOrEqual(t, f.UnreadCount, f.TotalCount, "UnreadCount should be <= TotalCount")
		})
	}
}

func TestIntegration_ThreadFieldsValidation(t *testing.T) {
	client, grantID := getTestClient(t)
	ctx, cancel := createTestContext()
	defer cancel()

	threads, err := client.GetThreads(ctx, grantID, &domain.ThreadQueryParams{Limit: 10})
	skipIfProviderNotSupported(t, err)
	require.NoError(t, err)

	for _, th := range threads {
		t.Run("Thread_"+safeSubstring(th.ID, 8), func(t *testing.T) {
			assert.NotEmpty(t, th.ID, "ID should not be empty")
			// Note: MessageIDs can be empty if messages were deleted or thread is draft-only
			if len(th.MessageIDs) == 0 {
				t.Logf("Thread %s has no message IDs (deleted messages or draft-only)", th.ID)
			}

			_ = th.Unread
			_ = th.Starred
		})
	}
}

// =============================================================================
// Pagination Tests
// =============================================================================

func TestIntegration_MessagePagination(t *testing.T) {
	client, grantID := getTestClient(t)
	ctx, cancel := createLongTestContext()
	defer cancel()

	// Get first page
	params := &domain.MessageQueryParams{
		Limit: 5,
	}

	page1, err := client.GetMessagesWithParams(ctx, grantID, params)
	require.NoError(t, err)

	if len(page1) < 5 {
		t.Skip("Not enough messages to test pagination")
	}

	t.Logf("Page 1: %d messages", len(page1))
	for _, m := range page1 {
		t.Logf("  [%s] %s", safeSubstring(m.ID, 8), safeSubstring(m.Subject, 30))
	}

	// Note: Nylas v3 uses page_token for pagination, which would need to be
	// returned from the API. For now, we just verify we can get a consistent page.
}

// Workflow tests are in integration_workflow_test.go
