package mime

import (
	"strings"
	"testing"
	"time"

	"github.com/nylas/cli/internal/domain"
)

func TestBuildEncryptedMessage_Simple(t *testing.T) {
	builder := NewBuilder()

	req := &EncryptedMessageRequest{
		From: []domain.EmailParticipant{
			{Name: "Alice", Email: "alice@example.com"},
		},
		To: []domain.EmailParticipant{
			{Name: "Bob", Email: "bob@example.com"},
		},
		Subject:    "Encrypted Test",
		Ciphertext: []byte("-----BEGIN PGP MESSAGE-----\nencrypted content\n-----END PGP MESSAGE-----"),
		Date:       time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
	}

	result, err := builder.BuildEncryptedMessage(req)
	if err != nil {
		t.Fatalf("BuildEncryptedMessage() error = %v", err)
	}

	resultStr := string(result)

	// Validate RFC 3156 multipart/encrypted structure
	requiredParts := []string{
		"MIME-Version: 1.0",
		"From: Alice <alice@example.com>",
		"To: Bob <bob@example.com>",
		"Subject: Encrypted Test",
		"Content-Type: multipart/encrypted",
		"protocol=\"application/pgp-encrypted\"",
		"application/pgp-encrypted",   // Version part content-type
		"Version: 1",                  // Version identifier
		"application/octet-stream",    // Encrypted data content-type
		"-----BEGIN PGP MESSAGE-----", // Encrypted content marker
		"-----END PGP MESSAGE-----",   // Encrypted content marker
	}

	for _, part := range requiredParts {
		if !strings.Contains(resultStr, part) {
			t.Errorf("Missing required part: %s", part)
		}
	}
}

func TestBuildEncryptedMessage_WithCcAndReplyTo(t *testing.T) {
	builder := NewBuilder()

	req := &EncryptedMessageRequest{
		From: []domain.EmailParticipant{
			{Email: "sender@example.com"},
		},
		To: []domain.EmailParticipant{
			{Email: "to@example.com"},
		},
		Cc: []domain.EmailParticipant{
			{Email: "cc@example.com"},
		},
		ReplyTo: []domain.EmailParticipant{
			{Email: "replyto@example.com"},
		},
		Subject:    "Test Cc/ReplyTo",
		Ciphertext: []byte("-----BEGIN PGP MESSAGE-----\ntest\n-----END PGP MESSAGE-----"),
	}

	result, err := builder.BuildEncryptedMessage(req)
	if err != nil {
		t.Fatalf("BuildEncryptedMessage() error = %v", err)
	}

	resultStr := string(result)

	// Validate Cc header is present
	if !strings.Contains(resultStr, "Cc: cc@example.com") {
		t.Error("Cc header not found")
	}

	// Validate Reply-To header is present
	if !strings.Contains(resultStr, "Reply-To: replyto@example.com") {
		t.Error("Reply-To header not found")
	}
}

func TestBuildEncryptedMessage_Validation(t *testing.T) {
	builder := NewBuilder()

	tests := []struct {
		name    string
		req     *EncryptedMessageRequest
		wantErr string
	}{
		{
			name: "missing To",
			req: &EncryptedMessageRequest{
				Subject:    "Test",
				Ciphertext: []byte("encrypted"),
			},
			wantErr: "recipient (To) is required",
		},
		{
			name: "missing Subject",
			req: &EncryptedMessageRequest{
				To: []domain.EmailParticipant{
					{Email: "test@example.com"},
				},
				Ciphertext: []byte("encrypted"),
			},
			wantErr: "subject is required",
		},
		{
			name: "missing Ciphertext",
			req: &EncryptedMessageRequest{
				To: []domain.EmailParticipant{
					{Email: "test@example.com"},
				},
				Subject: "Test",
			},
			wantErr: "ciphertext is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := builder.BuildEncryptedMessage(tt.req)
			if err == nil {
				t.Error("Expected error but got nil")
				return
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("Expected error containing %q, got %q", tt.wantErr, err.Error())
			}
		})
	}
}

func TestBuildEncryptedMessage_CustomHeaders(t *testing.T) {
	builder := NewBuilder()

	req := &EncryptedMessageRequest{
		From: []domain.EmailParticipant{
			{Email: "sender@example.com"},
		},
		To: []domain.EmailParticipant{
			{Email: "recipient@example.com"},
		},
		Subject:    "Custom Headers Test",
		Ciphertext: []byte("-----BEGIN PGP MESSAGE-----\ntest\n-----END PGP MESSAGE-----"),
		Headers: map[string]string{
			"X-Custom-Header": "custom-value",
			"X-Priority":      "high",
		},
		MessageID: "custom-message-id@example.com",
	}

	result, err := builder.BuildEncryptedMessage(req)
	if err != nil {
		t.Fatalf("BuildEncryptedMessage() error = %v", err)
	}

	resultStr := string(result)

	// Validate custom headers
	if !strings.Contains(resultStr, "X-Custom-Header: custom-value") {
		t.Error("Custom header not found")
	}
	if !strings.Contains(resultStr, "X-Priority: high") {
		t.Error("X-Priority header not found")
	}
	if !strings.Contains(resultStr, "Message-ID: <custom-message-id@example.com>") {
		t.Error("Message-ID header not found")
	}
}

func TestBuildEncryptedMessage_MIMEStructure(t *testing.T) {
	builder := NewBuilder()

	ciphertext := `-----BEGIN PGP MESSAGE-----

hQEMA8nJ...encrypted content...
=abcd
-----END PGP MESSAGE-----`

	req := &EncryptedMessageRequest{
		From: []domain.EmailParticipant{
			{Email: "sender@example.com"},
		},
		To: []domain.EmailParticipant{
			{Email: "recipient@example.com"},
		},
		Subject:    "MIME Structure Test",
		Ciphertext: []byte(ciphertext),
	}

	result, err := builder.BuildEncryptedMessage(req)
	if err != nil {
		t.Fatalf("BuildEncryptedMessage() error = %v", err)
	}

	resultStr := string(result)

	// Find the boundary
	if !strings.Contains(resultStr, "boundary=") {
		t.Fatal("Missing boundary in Content-Type")
	}

	// Validate two-part structure (RFC 3156 Section 4)
	// Part 1: application/pgp-encrypted with "Version: 1"
	// Part 2: application/octet-stream with the actual encrypted data

	// Count boundary occurrences (should be 3: start of part 1, start of part 2, end)
	boundaryCount := strings.Count(resultStr, "--=_encrypted_")
	if boundaryCount < 3 {
		t.Errorf("Expected at least 3 boundary markers (2 parts + end), got %d", boundaryCount)
	}

	// Verify part 1 comes before part 2
	versionPos := strings.Index(resultStr, "Version: 1")
	ciphertextPos := strings.Index(resultStr, "-----BEGIN PGP MESSAGE-----")

	if versionPos == -1 {
		t.Error("Version: 1 not found in output")
	}
	if ciphertextPos == -1 {
		t.Error("PGP MESSAGE not found in output")
	}
	if versionPos > ciphertextPos {
		t.Error("Version part should come before encrypted data part")
	}
}

func TestPrepareContentToEncrypt(t *testing.T) {
	builder := NewBuilder()

	// Test simple body
	content, err := builder.PrepareContentToEncrypt("Hello, World!", "text/plain", nil)
	if err != nil {
		t.Fatalf("PrepareContentToEncrypt() error = %v", err)
	}

	contentStr := string(content)

	// Should have proper content headers
	if !strings.Contains(contentStr, "Content-Type: text/plain") {
		t.Error("Missing Content-Type header")
	}
	if !strings.Contains(contentStr, "Content-Transfer-Encoding: quoted-printable") {
		t.Error("Missing Content-Transfer-Encoding header")
	}

	// Body should be encoded
	if !strings.Contains(contentStr, "Hello") {
		t.Error("Body content not found")
	}
}

func TestPrepareContentToEncrypt_WithAttachments(t *testing.T) {
	builder := NewBuilder()

	attachments := []domain.Attachment{
		{
			Filename:    "secret.txt",
			ContentType: "text/plain",
			Content:     []byte("Secret content"),
		},
	}

	content, err := builder.PrepareContentToEncrypt("Email body", "text/plain", attachments)
	if err != nil {
		t.Fatalf("PrepareContentToEncrypt() error = %v", err)
	}

	contentStr := string(content)

	// Should be multipart/mixed
	if !strings.Contains(contentStr, "multipart/mixed") {
		t.Error("Expected multipart/mixed for email with attachments")
	}

	// Should include attachment
	if !strings.Contains(contentStr, "secret.txt") {
		t.Error("Attachment filename not found")
	}
}

func TestEncryptedMessageRequest_NonASCIISubject(t *testing.T) {
	builder := NewBuilder()

	req := &EncryptedMessageRequest{
		From: []domain.EmailParticipant{
			{Email: "sender@example.com"},
		},
		To: []domain.EmailParticipant{
			{Email: "recipient@example.com"},
		},
		Subject:    "Test с кириллицей 日本語",
		Ciphertext: []byte("-----BEGIN PGP MESSAGE-----\ntest\n-----END PGP MESSAGE-----"),
	}

	result, err := builder.BuildEncryptedMessage(req)
	if err != nil {
		t.Fatalf("BuildEncryptedMessage() error = %v", err)
	}

	resultStr := string(result)

	// Non-ASCII subject should be RFC 2047 encoded
	// Should contain "=?" which indicates encoded word
	if !strings.Contains(resultStr, "Subject:") {
		t.Error("Subject header not found")
	}

	// Verify the subject is encoded (RFC 2047)
	lines := strings.Split(resultStr, "\r\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "Subject:") {
			if !strings.Contains(line, "=?utf-8?") {
				t.Errorf("Non-ASCII subject should be RFC 2047 encoded: %s", line)
			}
			break
		}
	}
}
