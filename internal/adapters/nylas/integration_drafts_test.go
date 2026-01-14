//go:build integration
// +build integration

package nylas_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/nylas/cli/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIntegration_GetDrafts(t *testing.T) {
	client, grantID := getTestClient(t)
	ctx, cancel := createTestContext()
	defer cancel()

	drafts, err := client.GetDrafts(ctx, grantID, 10)
	skipIfProviderNotSupported(t, err)
	require.NoError(t, err)

	t.Logf("Found %d drafts", len(drafts))
	for _, d := range drafts {
		t.Logf("  [%s] %s", safeSubstring(d.ID, 8), safeSubstring(d.Subject, 50))
	}
}

func TestIntegration_DraftLifecycle_Basic(t *testing.T) {
	client, grantID := getTestClient(t)
	ctx, cancel := createLongTestContext()
	defer cancel()

	timestamp := time.Now().Format("2006-01-02 15:04:05")

	// Create a draft
	createReq := &domain.CreateDraftRequest{
		Subject: fmt.Sprintf("Integration Test Draft - %s", timestamp),
		Body:    "<html><body><p>This is a test draft created by integration tests.</p></body></html>",
		To:      []domain.EmailParticipant{{Email: "test@example.com", Name: "Test User"}},
	}

	draft, err := client.CreateDraft(ctx, grantID, createReq)
	skipIfProviderNotSupported(t, err)
	require.NoError(t, err)
	require.NotEmpty(t, draft.ID)
	t.Logf("Created draft: %s", draft.ID)
	t.Logf("Subject: %s", draft.Subject)

	// Get the draft
	retrieved, err := client.GetDraft(ctx, grantID, draft.ID)
	require.NoError(t, err)
	assert.Equal(t, draft.ID, retrieved.ID)
	assert.Equal(t, createReq.Subject, retrieved.Subject)
	t.Logf("Retrieved draft: %s", retrieved.Subject)

	// Update the draft
	updateReq := &domain.CreateDraftRequest{
		Subject: createReq.Subject + " (UPDATED)",
		Body:    "<html><body><p>This draft has been updated.</p></body></html>",
		To:      createReq.To,
	}
	updated, err := client.UpdateDraft(ctx, grantID, draft.ID, updateReq)
	require.NoError(t, err)
	assert.Contains(t, updated.Subject, "(UPDATED)")
	t.Logf("Updated draft subject: %s", updated.Subject)

	// Delete the draft
	err = client.DeleteDraft(ctx, grantID, draft.ID)
	require.NoError(t, err)
	t.Logf("Deleted draft: %s", draft.ID)

	// Verify deletion (with tolerance for eventual consistency)
	_, err = client.GetDraft(ctx, grantID, draft.ID)
	if err != nil {
		t.Logf("Draft deletion verified: %v", err)
	} else {
		t.Logf("Draft still exists due to eventual consistency (this is OK)")
	}
}

func TestIntegration_DraftLifecycle_WithCC(t *testing.T) {
	client, grantID := getTestClient(t)
	ctx, cancel := createLongTestContext()
	defer cancel()

	createReq := &domain.CreateDraftRequest{
		Subject: fmt.Sprintf("Draft with CC - %s", time.Now().Format("15:04:05")),
		Body:    "Test draft with CC recipients",
		To:      []domain.EmailParticipant{{Email: "recipient@example.com", Name: "Main Recipient"}},
		Cc:      []domain.EmailParticipant{{Email: "cc1@example.com"}, {Email: "cc2@example.com", Name: "CC Two"}},
	}

	draft, err := client.CreateDraft(ctx, grantID, createReq)
	skipIfProviderNotSupported(t, err)
	require.NoError(t, err)
	require.NotEmpty(t, draft.ID)
	t.Logf("Created draft with CC: %s", draft.ID)

	// Verify CC was saved
	retrieved, err := client.GetDraft(ctx, grantID, draft.ID)
	require.NoError(t, err)
	assert.Len(t, retrieved.Cc, 2, "Draft should have 2 CC recipients")

	// Cleanup
	err = client.DeleteDraft(ctx, grantID, draft.ID)
	require.NoError(t, err)
}

func TestIntegration_DraftLifecycle_WithBCC(t *testing.T) {
	client, grantID := getTestClient(t)
	ctx, cancel := createLongTestContext()
	defer cancel()

	createReq := &domain.CreateDraftRequest{
		Subject: fmt.Sprintf("Draft with BCC - %s", time.Now().Format("15:04:05")),
		Body:    "Test draft with BCC recipients",
		To:      []domain.EmailParticipant{{Email: "recipient@example.com"}},
		Bcc:     []domain.EmailParticipant{{Email: "bcc@example.com", Name: "Secret Recipient"}},
	}

	draft, err := client.CreateDraft(ctx, grantID, createReq)
	skipIfProviderNotSupported(t, err)
	require.NoError(t, err)
	require.NotEmpty(t, draft.ID)
	t.Logf("Created draft with BCC: %s", draft.ID)

	// Cleanup
	err = client.DeleteDraft(ctx, grantID, draft.ID)
	require.NoError(t, err)
}

func TestIntegration_DraftLifecycle_ReplyTo(t *testing.T) {
	client, grantID := getTestClient(t)
	ctx, cancel := createLongTestContext()
	defer cancel()

	createReq := &domain.CreateDraftRequest{
		Subject: fmt.Sprintf("Draft with Reply-To - %s", time.Now().Format("15:04:05")),
		Body:    "Test draft with reply-to address",
		To:      []domain.EmailParticipant{{Email: "recipient@example.com"}},
		ReplyTo: []domain.EmailParticipant{{Email: "replyto@example.com", Name: "Reply Here"}},
	}

	draft, err := client.CreateDraft(ctx, grantID, createReq)
	skipIfProviderNotSupported(t, err)
	require.NoError(t, err)
	require.NotEmpty(t, draft.ID)
	t.Logf("Created draft with Reply-To: %s", draft.ID)

	// Cleanup
	err = client.DeleteDraft(ctx, grantID, draft.ID)
	require.NoError(t, err)
}

func TestIntegration_GetDraft_NotFound(t *testing.T) {
	client, grantID := getTestClient(t)
	ctx, cancel := createTestContext()
	defer cancel()

	_, err := client.GetDraft(ctx, grantID, "nonexistent-draft-id-12345")
	assert.Error(t, err, "Should return error for non-existent draft")
}

// =============================================================================
// Attachment Tests
// =============================================================================
