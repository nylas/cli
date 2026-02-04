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

func TestExtractInlinePGP(t *testing.T) {
	tests := []struct {
		name         string
		rawMIME      string
		wantContains string
		wantNil      bool
	}{
		{
			name: "inline PGP in plain text body",
			rawMIME: `From: sender@example.com
To: recipient@example.com
Content-Type: text/plain

-----BEGIN PGP MESSAGE-----

hQEMAxxxxxxxxx
=xxxx
-----END PGP MESSAGE-----`,
			wantContains: "-----BEGIN PGP MESSAGE-----",
			wantNil:      false,
		},
		{
			name: "inline PGP in multipart/mixed (Outlook style)",
			rawMIME: `From: sender@example.com
Content-Type: multipart/mixed; boundary="boundary123"

--boundary123
Content-Type: text/plain; charset="UTF-8"

-----BEGIN PGP MESSAGE-----

hQIMAwfdV3YDsnmWARAAs8jMMsaoLnlg
=xxxx
-----END PGP MESSAGE-----
--boundary123--`,
			wantContains: "-----BEGIN PGP MESSAGE-----",
			wantNil:      false,
		},
		{
			name: "inline PGP with surrounding text",
			rawMIME: `Content-Type: text/plain

Some text before

-----BEGIN PGP MESSAGE-----
encrypted_data_here
-----END PGP MESSAGE-----

Some text after`,
			wantContains: "-----BEGIN PGP MESSAGE-----",
			wantNil:      false,
		},
		{
			name: "no PGP content",
			rawMIME: `From: sender@example.com
Content-Type: text/plain

Just a regular email with no encryption.`,
			wantNil: true,
		},
		{
			name: "incomplete PGP block - missing end marker",
			rawMIME: `Content-Type: text/plain

-----BEGIN PGP MESSAGE-----
encrypted_data_here
No end marker`,
			wantNil: true,
		},
		{
			name: "PGP in second part of multipart",
			rawMIME: `Content-Type: multipart/mixed; boundary="mixed"

--mixed
Content-Type: text/plain

Regular text part
--mixed
Content-Type: text/plain

-----BEGIN PGP MESSAGE-----
encrypted_in_second_part
-----END PGP MESSAGE-----
--mixed--`,
			wantContains: "-----BEGIN PGP MESSAGE-----",
			wantNil:      false,
		},
		{
			name:         "PGP with CRLF line endings",
			rawMIME:      "Content-Type: text/plain\r\n\r\n-----BEGIN PGP MESSAGE-----\r\nencrypted\r\n-----END PGP MESSAGE-----",
			wantContains: "-----BEGIN PGP MESSAGE-----",
			wantNil:      false,
		},
		{
			name:    "empty input",
			rawMIME: "",
			wantNil: true,
		},
		{
			name: "multipart without PGP",
			rawMIME: `Content-Type: multipart/mixed; boundary="b"

--b
Content-Type: text/plain

No encryption here
--b--`,
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractInlinePGP(tt.rawMIME)
			if tt.wantNil {
				if got != nil {
					t.Errorf("extractInlinePGP() = %q, want nil", string(got))
				}
				return
			}
			if got == nil {
				t.Error("extractInlinePGP() = nil, want non-nil")
				return
			}
			if !strings.Contains(string(got), tt.wantContains) {
				t.Errorf("extractInlinePGP() = %q, want to contain %q", string(got), tt.wantContains)
			}
			// Verify we extract the complete PGP block
			if !strings.HasPrefix(string(got), "-----BEGIN PGP MESSAGE-----") {
				t.Errorf("extractInlinePGP() should start with PGP header, got: %q", string(got))
			}
			if !strings.HasSuffix(string(got), "-----END PGP MESSAGE-----") {
				t.Errorf("extractInlinePGP() should end with PGP footer, got: %q", string(got))
			}
		})
	}
}

func TestExtractInlinePGP_ExtractsCompletePGPBlock(t *testing.T) {
	// Verify the extracted content is exactly the PGP block
	rawMIME := `Content-Type: text/plain

Preamble text here.

-----BEGIN PGP MESSAGE-----

hQEMAxxxxxxxxx
line2
line3
=xxxx
-----END PGP MESSAGE-----

Postamble text here.`

	got := extractInlinePGP(rawMIME)
	if got == nil {
		t.Fatal("extractInlinePGP() returned nil")
	}

	gotStr := string(got)

	// Should not contain preamble or postamble
	if strings.Contains(gotStr, "Preamble") {
		t.Error("extractInlinePGP() should not include preamble text")
	}
	if strings.Contains(gotStr, "Postamble") {
		t.Error("extractInlinePGP() should not include postamble text")
	}

	// Should contain the full PGP message
	if !strings.Contains(gotStr, "hQEMAxxxxxxxxx") {
		t.Error("extractInlinePGP() should contain the encrypted data")
	}
	if !strings.Contains(gotStr, "line2") {
		t.Error("extractInlinePGP() should contain all lines of encrypted data")
	}
}

func TestExtractInlinePGP_Base64EncodedAttachment(t *testing.T) {
	// Test Outlook-style email where PGP content is base64-encoded in an attachment
	// This is the actual format Microsoft/Outlook uses for PGP/MIME emails
	// The base64 decodes to: "-----BEGIN PGP MESSAGE-----\n\nhQEMAtest\n=xxxx\n-----END PGP MESSAGE-----\n"
	base64PGP := "LS0tLS1CRUdJTiBQR1AgTUVTU0FHRS0tLS0tCgpoUUVNQXRlc3QKPXh4eHgKLS0tLS1FTkQgUEdQIE1FU1NBR0UtLS0tLQo="

	rawMIME := `From: sender@outlook.com
To: recipient@example.com
Content-Type: multipart/mixed;
	boundary="_003_OutlookBoundary_"
MIME-Version: 1.0

--_003_OutlookBoundary_
Content-Type: text/plain; charset="us-ascii"
Content-Transfer-Encoding: quoted-printable


--_003_OutlookBoundary_
Content-Type: application/pgp-encrypted; name="PGPMIME version identification"
Content-Description: PGP/MIME version identification
Content-Disposition: attachment; filename="PGPMIME version identification"
Content-Transfer-Encoding: base64

VmVyc2lvbjogMQ0K

--_003_OutlookBoundary_
Content-Type: application/octet-stream; name="encrypted.asc"
Content-Description: OpenPGP encrypted message.asc
Content-Disposition: attachment; filename="encrypted.asc"
Content-Transfer-Encoding: base64

` + base64PGP + `

--_003_OutlookBoundary_--`

	got := extractInlinePGP(rawMIME)
	if got == nil {
		t.Fatal("extractInlinePGP() returned nil for base64-encoded Outlook attachment")
	}

	gotStr := string(got)

	// Should extract the decoded PGP message
	if !strings.HasPrefix(gotStr, "-----BEGIN PGP MESSAGE-----") {
		t.Errorf("extractInlinePGP() should start with PGP header, got: %q", gotStr)
	}
	if !strings.HasSuffix(gotStr, "-----END PGP MESSAGE-----") {
		t.Errorf("extractInlinePGP() should end with PGP footer, got: %q", gotStr)
	}
	if !strings.Contains(gotStr, "hQEMAtest") {
		t.Errorf("extractInlinePGP() should contain the encrypted data, got: %q", gotStr)
	}
}
