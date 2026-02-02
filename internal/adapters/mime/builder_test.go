package mime

import (
	"strings"
	"testing"
	"time"

	"github.com/nylas/cli/internal/domain"
)

func TestBuildSignedMessage_Simple(t *testing.T) {
	builder := NewBuilder()

	req := &SignedMessageRequest{
		From: []domain.EmailParticipant{
			{Name: "Alice", Email: "alice@example.com"},
		},
		To: []domain.EmailParticipant{
			{Name: "Bob", Email: "bob@example.com"},
		},
		Subject:     "Test Subject",
		Body:        "This is a test email body.",
		ContentType: "text/plain",
		Signature:   []byte("-----BEGIN PGP SIGNATURE-----\ntest\n-----END PGP SIGNATURE-----"),
		HashAlgo:    "SHA256",
		Date:        time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
	}

	result, err := builder.BuildSignedMessage(req)
	if err != nil {
		t.Fatalf("BuildSignedMessage() error = %v", err)
	}

	resultStr := string(result)

	// Validate required headers
	requiredHeaders := []string{
		"MIME-Version: 1.0",
		"From: Alice <alice@example.com>",
		"To: Bob <bob@example.com>",
		"Subject: Test Subject",
		"Content-Type: multipart/signed",
		"protocol=\"application/pgp-signature\"",
		"micalg=pgp-sha256",
	}

	for _, header := range requiredHeaders {
		if !strings.Contains(resultStr, header) {
			t.Errorf("Missing required header: %s", header)
		}
	}

	// Validate signature is present
	if !strings.Contains(resultStr, "BEGIN PGP SIGNATURE") {
		t.Error("Signature not found in output")
	}

	// Validate body is present
	if !strings.Contains(resultStr, "This is a test email body") {
		t.Error("Body not found in output")
	}
}

func TestBuildSignedMessage_WithAttachment(t *testing.T) {
	builder := NewBuilder()

	req := &SignedMessageRequest{
		From: []domain.EmailParticipant{
			{Email: "sender@example.com"},
		},
		To: []domain.EmailParticipant{
			{Email: "recipient@example.com"},
		},
		Subject:     "Email with attachment",
		Body:        "See attached file.",
		ContentType: "text/plain",
		Signature:   []byte("-----BEGIN PGP SIGNATURE-----\ntest\n-----END PGP SIGNATURE-----"),
		HashAlgo:    "SHA256",
		Attachments: []domain.Attachment{
			{
				Filename:    "test.txt",
				ContentType: "text/plain",
				Content:     []byte("Hello, world!"),
			},
		},
	}

	result, err := builder.BuildSignedMessage(req)
	if err != nil {
		t.Fatalf("BuildSignedMessage() error = %v", err)
	}

	resultStr := string(result)

	// Validate multipart/mixed is used
	if !strings.Contains(resultStr, "multipart/mixed") {
		t.Error("Expected multipart/mixed for email with attachments")
	}

	// Validate attachment headers
	if !strings.Contains(resultStr, "test.txt") {
		t.Error("Attachment filename not found")
	}

	// Validate Content-Transfer-Encoding for attachment
	if !strings.Contains(resultStr, "Content-Transfer-Encoding: base64") {
		t.Error("Expected base64 encoding for attachment")
	}
}

func TestBuildSignedMessage_HTMLBody(t *testing.T) {
	builder := NewBuilder()

	req := &SignedMessageRequest{
		From: []domain.EmailParticipant{
			{Email: "sender@example.com"},
		},
		To: []domain.EmailParticipant{
			{Email: "recipient@example.com"},
		},
		Subject:     "HTML Email",
		Body:        "<html><body><h1>Hello</h1></body></html>",
		ContentType: "text/html",
		Signature:   []byte("-----BEGIN PGP SIGNATURE-----\ntest\n-----END PGP SIGNATURE-----"),
		HashAlgo:    "SHA512",
	}

	result, err := builder.BuildSignedMessage(req)
	if err != nil {
		t.Fatalf("BuildSignedMessage() error = %v", err)
	}

	resultStr := string(result)

	// Validate HTML content type
	if !strings.Contains(resultStr, "text/html") {
		t.Error("Expected text/html content type")
	}

	// Validate SHA512 micalg
	if !strings.Contains(resultStr, "micalg=pgp-sha512") {
		t.Error("Expected pgp-sha512 micalg")
	}
}

func TestBuildSignedMessage_WithCcBcc(t *testing.T) {
	builder := NewBuilder()

	req := &SignedMessageRequest{
		From: []domain.EmailParticipant{
			{Email: "sender@example.com"},
		},
		To: []domain.EmailParticipant{
			{Email: "to@example.com"},
		},
		Cc: []domain.EmailParticipant{
			{Email: "cc@example.com"},
		},
		Bcc: []domain.EmailParticipant{
			{Email: "bcc@example.com"},
		},
		ReplyTo: []domain.EmailParticipant{
			{Email: "replyto@example.com"},
		},
		Subject:   "Test Cc/Bcc",
		Body:      "Test body",
		Signature: []byte("-----BEGIN PGP SIGNATURE-----\ntest\n-----END PGP SIGNATURE-----"),
		HashAlgo:  "SHA256",
	}

	result, err := builder.BuildSignedMessage(req)
	if err != nil {
		t.Fatalf("BuildSignedMessage() error = %v", err)
	}

	resultStr := string(result)

	// Validate Cc header is present
	if !strings.Contains(resultStr, "Cc: cc@example.com") {
		t.Error("Cc header not found")
	}

	// Validate Bcc header is NOT present (security best practice)
	if strings.Contains(resultStr, "Bcc:") {
		t.Error("Bcc header should not be included in MIME message")
	}

	// Validate Reply-To header is present
	if !strings.Contains(resultStr, "Reply-To: replyto@example.com") {
		t.Error("Reply-To header not found")
	}
}

func TestBuildSignedMessage_Validation(t *testing.T) {
	builder := NewBuilder()

	tests := []struct {
		name    string
		req     *SignedMessageRequest
		wantErr string
	}{
		{
			name: "missing To",
			req: &SignedMessageRequest{
				Subject:   "Test",
				Body:      "Test",
				Signature: []byte("sig"),
			},
			wantErr: "recipient (To) is required",
		},
		{
			name: "missing Subject",
			req: &SignedMessageRequest{
				To: []domain.EmailParticipant{
					{Email: "test@example.com"},
				},
				Body:      "Test",
				Signature: []byte("sig"),
			},
			wantErr: "subject is required",
		},
		{
			name: "missing Body",
			req: &SignedMessageRequest{
				To: []domain.EmailParticipant{
					{Email: "test@example.com"},
				},
				Subject:   "Test",
				Signature: []byte("sig"),
			},
			wantErr: "body is required",
		},
		{
			name: "missing Signature",
			req: &SignedMessageRequest{
				To: []domain.EmailParticipant{
					{Email: "test@example.com"},
				},
				Subject: "Test",
				Body:    "Test",
			},
			wantErr: "signature is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := builder.BuildSignedMessage(tt.req)
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

func TestFormatAddresses(t *testing.T) {
	tests := []struct {
		name  string
		input []domain.EmailParticipant
		want  string
	}{
		{
			name: "single address with name",
			input: []domain.EmailParticipant{
				{Name: "John Doe", Email: "john@example.com"},
			},
			want: "John Doe <john@example.com>",
		},
		{
			name: "single address without name",
			input: []domain.EmailParticipant{
				{Email: "jane@example.com"},
			},
			want: "jane@example.com",
		},
		{
			name: "multiple addresses",
			input: []domain.EmailParticipant{
				{Name: "Alice", Email: "alice@example.com"},
				{Email: "bob@example.com"},
			},
			want: "Alice <alice@example.com>, bob@example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatAddresses(tt.input)
			if got != tt.want {
				t.Errorf("formatAddresses() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestIsASCII(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{
			name:  "pure ASCII",
			input: "Hello World",
			want:  true,
		},
		{
			name:  "with unicode",
			input: "Hello 世界",
			want:  false,
		},
		{
			name:  "empty string",
			input: "",
			want:  true,
		},
		{
			name:  "emoji",
			input: "Test 😊",
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isASCII(tt.input)
			if got != tt.want {
				t.Errorf("isASCII(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestGetMicAlg(t *testing.T) {
	tests := []struct {
		name     string
		hashAlgo string
		want     string
	}{
		{"SHA256", "SHA256", "pgp-sha256"},
		{"sha256 lowercase", "sha256", "pgp-sha256"},
		{"SHA512", "SHA512", "pgp-sha512"},
		{"SHA384", "SHA384", "pgp-sha384"},
		{"SHA1", "SHA1", "pgp-sha1"},
		{"unknown", "MD5", "pgp-sha256"}, // default
		{"empty", "", "pgp-sha256"},      // default
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getMicAlg(tt.hashAlgo)
			if got != tt.want {
				t.Errorf("getMicAlg(%q) = %q, want %q", tt.hashAlgo, got, tt.want)
			}
		})
	}
}

func TestGenerateBoundary(t *testing.T) {
	boundary1 := generateBoundary("test")
	boundary2 := generateBoundary("test")

	// Should contain prefix
	if !strings.HasPrefix(boundary1, "=_test_") {
		t.Errorf("Boundary should start with prefix: %s", boundary1)
	}

	// Should be unique
	if boundary1 == boundary2 {
		t.Error("Generated boundaries should be unique")
	}
}
