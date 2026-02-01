package email

import (
	"context"
	"strings"
	"testing"

	"github.com/nylas/cli/internal/adapters/nylas"
	"github.com/nylas/cli/internal/domain"
)

func TestHandleListGPGKeys_NoGPG(t *testing.T) {
	// This test will skip if GPG is not installed
	ctx := context.Background()
	err := handleListGPGKeys(ctx)

	// Either succeeds or fails with "GPG not found" error
	if err != nil && !strings.Contains(err.Error(), "GPG not found") {
		t.Errorf("handleListGPGKeys() unexpected error = %v", err)
	}
}

func TestSendSignedEmail_MockClient(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping GPG integration test in short mode")
	}

	ctx := context.Background()

	// Create mock client
	mockClient := &nylas.MockClient{
		SendRawMessageFunc: func(ctx context.Context, grantID string, rawMIME []byte) (*domain.Message, error) {
			// Validate raw MIME contains expected markers
			mimeStr := string(rawMIME)
			if !strings.Contains(mimeStr, "MIME-Version") {
				t.Error("Raw MIME missing MIME-Version header")
			}
			if !strings.Contains(mimeStr, "multipart/signed") {
				t.Error("Raw MIME missing multipart/signed content type")
			}
			if !strings.Contains(mimeStr, "application/pgp-signature") {
				t.Error("Raw MIME missing PGP signature content type")
			}

			return &domain.Message{
				ID:      "test-msg-id",
				GrantID: grantID,
			}, nil
		},
	}

	req := &domain.SendMessageRequest{
		Subject: "Test Subject",
		Body:    "Test body",
		To: []domain.EmailParticipant{
			{Email: "test@example.com"},
		},
	}

	toContacts := []domain.EmailParticipant{
		{Email: "test@example.com"},
	}

	// This will fail if GPG is not configured, which is expected
	msg, err := sendSignedEmail(ctx, mockClient, "test-grant", req, "", toContacts, "Test Subject", "Test body")

	// If GPG is not available, we expect an error
	if err != nil {
		// Expected errors:
		// - "GPG not found" (no GPG installed)
		// - "no default GPG key" (GPG installed but no keys/config)
		if !strings.Contains(err.Error(), "GPG") && !strings.Contains(err.Error(), "gpg") {
			t.Errorf("sendSignedEmail() unexpected error = %v", err)
		}
		t.Skipf("GPG not configured, skipping test: %v", err)
		return
	}

	// If we got here, GPG is configured and signing worked
	if msg == nil {
		t.Fatal("sendSignedEmail() returned nil message")
	}
	if msg.ID != "test-msg-id" {
		t.Errorf("sendSignedEmail() message ID = %v, want test-msg-id", msg.ID)
	}
}

func TestSendSignedEmail_WithSpecificKey(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping GPG integration test in short mode")
	}

	ctx := context.Background()

	// Create mock client
	mockClient := &nylas.MockClient{
		SendRawMessageFunc: func(ctx context.Context, grantID string, rawMIME []byte) (*domain.Message, error) {
			return &domain.Message{
				ID:      "test-msg-id-2",
				GrantID: grantID,
			}, nil
		},
	}

	req := &domain.SendMessageRequest{
		Subject: "Test",
		Body:    "Body",
		To: []domain.EmailParticipant{
			{Email: "test@example.com"},
		},
	}

	toContacts := []domain.EmailParticipant{
		{Email: "test@example.com"},
	}

	// Try with an invalid key ID - should fail
	_, err := sendSignedEmail(ctx, mockClient, "test-grant", req, "INVALID_KEY_ID", toContacts, "Test", "Body")

	// Should fail with GPG error
	if err == nil {
		t.Error("sendSignedEmail() with invalid key should fail")
	}

	// Error should mention the key or GPG
	if !strings.Contains(err.Error(), "GPG") && !strings.Contains(err.Error(), "key") {
		t.Errorf("sendSignedEmail() error should mention GPG or key, got: %v", err)
	}
}

func TestSendSignedEmail_HTMLBody(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping GPG integration test in short mode")
	}

	ctx := context.Background()

	// Create mock client that validates HTML content type
	mockClient := &nylas.MockClient{
		SendRawMessageFunc: func(ctx context.Context, grantID string, rawMIME []byte) (*domain.Message, error) {
			mimeStr := string(rawMIME)
			if !strings.Contains(mimeStr, "text/html") {
				t.Error("Expected text/html content type for HTML body")
			}
			return &domain.Message{
				ID:      "test-html-msg",
				GrantID: grantID,
			}, nil
		},
	}

	req := &domain.SendMessageRequest{
		Subject: "HTML Test",
		Body:    "<html><body><h1>Hello</h1></body></html>",
		To: []domain.EmailParticipant{
			{Email: "test@example.com"},
		},
	}

	toContacts := []domain.EmailParticipant{
		{Email: "test@example.com"},
	}

	// This will skip if GPG is not configured
	msg, err := sendSignedEmail(ctx, mockClient, "test-grant", req, "", toContacts, "HTML Test", "<html><body><h1>Hello</h1></body></html>")

	if err != nil {
		if strings.Contains(err.Error(), "GPG") || strings.Contains(err.Error(), "gpg") {
			t.Skipf("GPG not configured, skipping test: %v", err)
			return
		}
		t.Errorf("sendSignedEmail() error = %v", err)
		return
	}

	if msg == nil {
		t.Fatal("sendSignedEmail() returned nil message")
	}
}
