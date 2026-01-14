//go:build integration
// +build integration

package nylas_test

import (
	"io"
	"strings"
	"testing"

	"github.com/nylas/cli/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIntegration_GetMessageWithAttachments(t *testing.T) {
	client, grantID := getTestClient(t)
	ctx, cancel := createTestContext()
	defer cancel()

	// Get messages and look for one with attachments
	messages, err := client.GetMessages(ctx, grantID, 50)
	require.NoError(t, err)

	var messageWithAttachment *domain.Message
	for i := range messages {
		if len(messages[i].Attachments) > 0 {
			messageWithAttachment = &messages[i]
			break
		}
	}

	if messageWithAttachment == nil {
		t.Skip("No messages with attachments found")
	}

	t.Logf("Found message with %d attachments: %s",
		len(messageWithAttachment.Attachments), messageWithAttachment.Subject)

	for _, a := range messageWithAttachment.Attachments {
		t.Logf("  Attachment: %s (%s, %d bytes)",
			a.Filename, a.ContentType, a.Size)

		assert.NotEmpty(t, a.ID, "Attachment should have ID")
		assert.NotEmpty(t, a.Filename, "Attachment should have filename")
		assert.NotEmpty(t, a.ContentType, "Attachment should have content type")
	}
}

func TestIntegration_GetAttachment(t *testing.T) {
	client, grantID := getTestClient(t)
	ctx, cancel := createTestContext()
	defer cancel()

	// Find a message with attachments (prefer non-inline attachments)
	messages, err := client.GetMessages(ctx, grantID, 50)
	require.NoError(t, err)

	var messageID, attachmentID string
	for _, m := range messages {
		for _, a := range m.Attachments {
			// Prefer non-inline attachments as they're more reliably accessible
			if !a.IsInline {
				messageID = m.ID
				attachmentID = a.ID
				break
			}
		}
		if attachmentID != "" {
			break
		}
	}

	// Fall back to any attachment if no non-inline found
	if attachmentID == "" {
		for _, m := range messages {
			if len(m.Attachments) > 0 {
				messageID = m.ID
				attachmentID = m.Attachments[0].ID
				break
			}
		}
	}

	if attachmentID == "" {
		t.Skip("No attachments found")
	}

	attachment, err := client.GetAttachment(ctx, grantID, messageID, attachmentID)
	if err != nil && strings.Contains(err.Error(), "attachment not found") {
		// Some inline attachments may not be accessible via the individual endpoint
		t.Skipf("Attachment not accessible via individual endpoint (may be inline): %v", err)
	}
	require.NoError(t, err)

	assert.Equal(t, attachmentID, attachment.ID)
	assert.NotEmpty(t, attachment.Filename)
	t.Logf("Retrieved attachment: %s (%s)", attachment.Filename, attachment.ContentType)
}

func TestIntegration_DownloadAttachment(t *testing.T) {
	client, grantID := getTestClient(t)
	ctx, cancel := createLongTestContext()
	defer cancel()

	// Find a message with attachments (prefer non-inline, smaller attachments)
	messages, err := client.GetMessages(ctx, grantID, 50)
	require.NoError(t, err)

	var messageID string
	var attachment *domain.Attachment

	// First pass: look for non-inline, appropriately sized attachments
	for _, m := range messages {
		for i := range m.Attachments {
			a := &m.Attachments[i]
			// Skip very large attachments and prefer non-inline
			if a.Size < 1000000 && a.Size > 0 && !a.IsInline {
				messageID = m.ID
				attachment = a
				break
			}
		}
		if attachment != nil {
			break
		}
	}

	// Second pass: fall back to any sized attachment
	if attachment == nil {
		for _, m := range messages {
			for i := range m.Attachments {
				a := &m.Attachments[i]
				if a.Size < 1000000 && a.Size > 0 {
					messageID = m.ID
					attachment = a
					break
				}
			}
			if attachment != nil {
				break
			}
		}
	}

	if attachment == nil {
		t.Skip("No suitable attachments found")
	}

	reader, err := client.DownloadAttachment(ctx, grantID, messageID, attachment.ID)
	if err != nil && strings.Contains(err.Error(), "attachment not found") {
		// Some inline attachments may not be accessible via the download endpoint
		t.Skipf("Attachment not accessible via download endpoint (may be inline): %v", err)
	}
	require.NoError(t, err)
	defer reader.Close()

	data, err := io.ReadAll(reader)
	require.NoError(t, err)
	require.NotEmpty(t, data, "Downloaded data should not be empty")

	t.Logf("Downloaded %s: %d bytes", attachment.Filename, len(data))
}

func TestIntegration_ListAttachments(t *testing.T) {
	client, grantID := getTestClient(t)
	ctx, cancel := createTestContext()
	defer cancel()

	// Find a message with attachments first
	messages, err := client.GetMessages(ctx, grantID, 50)
	require.NoError(t, err)

	var messageID string
	var expectedCount int
	for _, m := range messages {
		if len(m.Attachments) > 0 {
			messageID = m.ID
			expectedCount = len(m.Attachments)
			break
		}
	}

	if messageID == "" {
		t.Skip("No messages with attachments found")
	}

	// Test ListAttachments
	attachments, err := client.ListAttachments(ctx, grantID, messageID)
	require.NoError(t, err)
	assert.Len(t, attachments, expectedCount)

	t.Logf("ListAttachments returned %d attachments for message %s", len(attachments), messageID)
	for _, a := range attachments {
		t.Logf("  - %s (%s, %d bytes, inline: %v)",
			a.Filename, a.ContentType, a.Size, a.IsInline)

		assert.NotEmpty(t, a.ID, "Attachment should have ID")
		assert.NotEmpty(t, a.Filename, "Attachment should have filename")
		assert.NotEmpty(t, a.ContentType, "Attachment should have content type")
	}
}

func TestIntegration_ListAttachments_EmptyMessage(t *testing.T) {
	client, grantID := getTestClient(t)
	ctx, cancel := createTestContext()
	defer cancel()

	// Find a message without attachments
	messages, err := client.GetMessages(ctx, grantID, 50)
	require.NoError(t, err)
	skipIfNoMessages(t, messages)

	var messageID string
	for _, m := range messages {
		if len(m.Attachments) == 0 {
			messageID = m.ID
			break
		}
	}

	if messageID == "" {
		t.Skip("All messages have attachments, can't test empty attachment list")
	}

	attachments, err := client.ListAttachments(ctx, grantID, messageID)
	require.NoError(t, err)
	assert.Empty(t, attachments, "Message without attachments should return empty list")
	t.Logf("Verified ListAttachments returns empty list for message without attachments")
}

// =============================================================================
// Send Message Tests (Destructive - requires explicit opt-in)
// =============================================================================
