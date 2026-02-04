package email

import (
	"strings"
	"testing"
)

func TestIsEncryptedMessage(t *testing.T) {
	tests := []struct {
		name        string
		contentType string
		want        bool
	}{
		{
			name:        "valid PGP/MIME encrypted",
			contentType: `multipart/encrypted; protocol="application/pgp-encrypted"; boundary="xyz"`,
			want:        true,
		},
		{
			name:        "valid with different order",
			contentType: `multipart/encrypted; boundary="abc"; protocol="application/pgp-encrypted"`,
			want:        true,
		},
		{
			name:        "multipart/signed not encrypted",
			contentType: `multipart/signed; protocol="application/pgp-signature"; boundary="xyz"`,
			want:        false,
		},
		{
			name:        "plain text",
			contentType: "text/plain; charset=utf-8",
			want:        false,
		},
		{
			name:        "multipart/encrypted without pgp protocol",
			contentType: `multipart/encrypted; protocol="application/x-pkcs7-mime"; boundary="xyz"`,
			want:        false,
		},
		{
			name:        "empty content type",
			contentType: "",
			want:        false,
		},
		{
			name:        "only multipart/encrypted",
			contentType: "multipart/encrypted",
			want:        false,
		},
		{
			name:        "only application/pgp-encrypted",
			contentType: "application/pgp-encrypted",
			want:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isEncryptedMessage(tt.contentType)
			if got != tt.want {
				t.Errorf("isEncryptedMessage(%q) = %v, want %v", tt.contentType, got, tt.want)
			}
		})
	}
}

func TestParseEncryptedMIME(t *testing.T) {
	tests := []struct {
		name           string
		rawMIME        string
		wantContains   string
		wantErr        bool
		wantErrContain string
	}{
		{
			name: "valid PGP/MIME encrypted message",
			rawMIME: `Content-Type: multipart/encrypted; protocol="application/pgp-encrypted";
	boundary="encrypted_boundary"

--encrypted_boundary
Content-Type: application/pgp-encrypted

Version: 1

--encrypted_boundary
Content-Type: application/octet-stream

-----BEGIN PGP MESSAGE-----

hQEMA...encrypted...data
-----END PGP MESSAGE-----
--encrypted_boundary--`,
			wantContains: "-----BEGIN PGP MESSAGE-----",
			wantErr:      false,
		},
		{
			name:         "encrypted message with CRLF",
			rawMIME:      "Content-Type: multipart/encrypted; protocol=\"application/pgp-encrypted\";\r\n\tboundary=\"enc123\"\r\n\r\n--enc123\r\nContent-Type: application/pgp-encrypted\r\n\r\nVersion: 1\r\n\r\n--enc123\r\nContent-Type: application/octet-stream\r\n\r\n-----BEGIN PGP MESSAGE-----\r\nencrypted\r\n-----END PGP MESSAGE-----\r\n--enc123--",
			wantContains: "-----BEGIN PGP MESSAGE-----",
			wantErr:      false,
		},
		{
			name:           "missing Content-Type header",
			rawMIME:        "From: test@example.com\n\nBody",
			wantErr:        true,
			wantErrContain: "Content-Type header not found",
		},
		{
			name:           "missing boundary",
			rawMIME:        "Content-Type: multipart/encrypted\n\nBody",
			wantErr:        true,
			wantErrContain: "no boundary found",
		},
		{
			name:           "no header/body separator",
			rawMIME:        "Content-Type: multipart/encrypted; boundary=\"xyz\"",
			wantErr:        true,
			wantErrContain: "could not find end of headers",
		},
		{
			name: "only one part (missing encrypted data)",
			rawMIME: `Content-Type: multipart/encrypted; protocol="application/pgp-encrypted"; boundary="enc"

--enc
Content-Type: application/pgp-encrypted

Version: 1
--enc--`,
			wantErr:        true,
			wantErrContain: "could not find encrypted content part",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseEncryptedMIME(tt.rawMIME)
			if tt.wantErr {
				if err == nil {
					t.Errorf("parseEncryptedMIME() expected error containing %q, got nil", tt.wantErrContain)
					return
				}
				if !strings.Contains(err.Error(), tt.wantErrContain) {
					t.Errorf("parseEncryptedMIME() error = %q, want error containing %q", err.Error(), tt.wantErrContain)
				}
				return
			}
			if err != nil {
				t.Errorf("parseEncryptedMIME() unexpected error: %v", err)
				return
			}
			if !strings.Contains(string(got), tt.wantContains) {
				t.Errorf("parseEncryptedMIME() result doesn't contain %q, got: %q", tt.wantContains, string(got))
			}
		})
	}
}

func TestExtractBodyFromMIME(t *testing.T) {
	tests := []struct {
		name        string
		mimeContent string
		want        string
	}{
		{
			name: "simple text/plain",
			mimeContent: `Content-Type: text/plain; charset=utf-8

Hello World`,
			want: "Hello World",
		},
		{
			name: "text/html",
			mimeContent: `Content-Type: text/html; charset=utf-8

<p>Hello World</p>`,
			want: "<p>Hello World</p>",
		},
		{
			name: "multipart/alternative with text part",
			mimeContent: `Content-Type: multipart/alternative; boundary="alt"

--alt
Content-Type: text/plain

Plain text version
--alt
Content-Type: text/html

<p>HTML version</p>
--alt--`,
			want: "Plain text version",
		},
		{
			name: "multipart/mixed with text part",
			mimeContent: `Content-Type: multipart/mixed; boundary="mixed"

--mixed
Content-Type: text/plain

Message body here
--mixed
Content-Type: application/pdf
Content-Disposition: attachment; filename="doc.pdf"

PDF content
--mixed--`,
			want: "Message body here",
		},
		{
			name:        "no Content-Type returns original",
			mimeContent: "Just plain content without headers",
			want:        "Just plain content without headers",
		},
		{
			name: "unknown content type returns original",
			mimeContent: `Content-Type: application/octet-stream

Binary data here`,
			want: `Content-Type: application/octet-stream

Binary data here`,
		},
		{
			name: "text/plain with extra whitespace",
			mimeContent: `Content-Type: text/plain

   Trimmed content   `,
			want: "Trimmed content",
		},
		{
			name:        "multipart with CRLF line endings",
			mimeContent: "Content-Type: multipart/alternative; boundary=\"b\"\r\n\r\n--b\r\nContent-Type: text/plain\r\n\r\nCRLF content\r\n--b--",
			want:        "CRLF content",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractBodyFromMIME(tt.mimeContent)
			if got != tt.want {
				t.Errorf("extractBodyFromMIME() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestParseEncryptedMIME_ExtractsCiphertext(t *testing.T) {
	// Test that we correctly extract just the ciphertext, trimmed
	rawMIME := `Content-Type: multipart/encrypted; protocol="application/pgp-encrypted"; boundary="enc"

--enc
Content-Type: application/pgp-encrypted

Version: 1

--enc
Content-Type: application/octet-stream

-----BEGIN PGP MESSAGE-----

hQEMAxxxxxxxxx
=xxxx
-----END PGP MESSAGE-----

--enc--`

	ciphertext, err := parseEncryptedMIME(rawMIME)
	if err != nil {
		t.Fatalf("parseEncryptedMIME() error: %v", err)
	}

	// Should start with PGP header
	if !strings.HasPrefix(string(ciphertext), "-----BEGIN PGP MESSAGE-----") {
		t.Errorf("Ciphertext should start with PGP header, got: %q", string(ciphertext)[:50])
	}

	// Should end with PGP footer
	if !strings.HasSuffix(string(ciphertext), "-----END PGP MESSAGE-----") {
		t.Errorf("Ciphertext should end with PGP footer, got: %q", string(ciphertext)[len(ciphertext)-50:])
	}

	// Should not have leading/trailing whitespace
	if strings.HasPrefix(string(ciphertext), " ") || strings.HasPrefix(string(ciphertext), "\n") {
		t.Error("Ciphertext has leading whitespace")
	}
	if strings.HasSuffix(string(ciphertext), " ") || strings.HasSuffix(string(ciphertext), "\n") {
		t.Error("Ciphertext has trailing whitespace")
	}
}

func TestExtractBodyFromMIME_NestedMultipart(t *testing.T) {
	// Test handling of nested multipart (common in email)
	mimeContent := `Content-Type: multipart/mixed; boundary="outer"

--outer
Content-Type: text/plain

This is the main message body.
--outer
Content-Type: application/pdf

PDF attachment data
--outer--`

	got := extractBodyFromMIME(mimeContent)
	want := "This is the main message body."

	if got != want {
		t.Errorf("extractBodyFromMIME() = %q, want %q", got, want)
	}
}

func TestExtractBodyFromMIME_PreferPlainText(t *testing.T) {
	// When multipart/alternative has both text/plain and text/html,
	// we should get the first text part (text/plain)
	mimeContent := `Content-Type: multipart/alternative; boundary="alt"

--alt
Content-Type: text/plain

Plain text preferred
--alt
Content-Type: text/html

<p>HTML version</p>
--alt--`

	got := extractBodyFromMIME(mimeContent)
	want := "Plain text preferred"

	if got != want {
		t.Errorf("extractBodyFromMIME() = %q, want %q", got, want)
	}
}

func TestIsEncryptedMessage_CaseInsensitive(t *testing.T) {
	// Content-Type values should work regardless of case
	tests := []struct {
		name        string
		contentType string
		want        bool
	}{
		{
			name:        "lowercase",
			contentType: `multipart/encrypted; protocol="application/pgp-encrypted"`,
			want:        true,
		},
		{
			name:        "uppercase MULTIPART",
			contentType: `MULTIPART/ENCRYPTED; protocol="application/pgp-encrypted"`,
			want:        false, // strings.Contains is case-sensitive
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isEncryptedMessage(tt.contentType)
			if got != tt.want {
				t.Errorf("isEncryptedMessage(%q) = %v, want %v", tt.contentType, got, tt.want)
			}
		})
	}
}
