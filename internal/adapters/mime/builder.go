package mime

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"mime"
	"mime/quotedprintable"
	"strings"
	"time"

	"github.com/nylas/cli/internal/domain"
)

// Builder constructs MIME messages.
type Builder interface {
	// BuildSignedMessage builds a PGP/MIME signed message (RFC 3156).
	BuildSignedMessage(req *SignedMessageRequest) ([]byte, error)

	// PrepareContentToSign prepares the MIME content part that will be signed.
	// Returns the exact bytes that should be signed with GPG.
	// This includes the part headers and encoded body with CRLF line endings.
	PrepareContentToSign(body, contentType string, attachments []domain.Attachment) ([]byte, error)

	// BuildEncryptedMessage builds a PGP/MIME encrypted message (RFC 3156 Section 4).
	BuildEncryptedMessage(req *EncryptedMessageRequest) ([]byte, error)

	// PrepareContentToEncrypt prepares the MIME content that will be encrypted.
	PrepareContentToEncrypt(body, contentType string, attachments []domain.Attachment) ([]byte, error)
}

// messageRequest is an interface for shared email header fields.
// Both SignedMessageRequest and EncryptedMessageRequest implement this.
type messageRequest interface {
	getFrom() []domain.EmailParticipant
	getTo() []domain.EmailParticipant
	getCc() []domain.EmailParticipant
	getReplyTo() []domain.EmailParticipant
	getSubject() string
	getHeaders() map[string]string
	getMessageID() string
	getDate() time.Time
}

// SignedMessageRequest contains all data needed to build a signed email.
type SignedMessageRequest struct {
	// Standard email fields
	From        []domain.EmailParticipant
	To          []domain.EmailParticipant
	Cc          []domain.EmailParticipant
	Bcc         []domain.EmailParticipant
	ReplyTo     []domain.EmailParticipant
	Subject     string
	Body        string
	ContentType string // "text/plain" or "text/html"

	// PGP signature
	Signature []byte // Detached signature from GPG (ASCII armored)
	HashAlgo  string // Hash algorithm (e.g., "SHA256")

	// PreparedContent is the exact content that was signed.
	// If set, this is used instead of rebuilding the content part.
	// This ensures the signed content matches what's in the message.
	PreparedContent []byte

	// Optional
	Attachments []domain.Attachment
	Headers     map[string]string
	MessageID   string
	Date        time.Time
}

// Implement messageRequest interface for SignedMessageRequest.
func (r *SignedMessageRequest) getFrom() []domain.EmailParticipant    { return r.From }
func (r *SignedMessageRequest) getTo() []domain.EmailParticipant      { return r.To }
func (r *SignedMessageRequest) getCc() []domain.EmailParticipant      { return r.Cc }
func (r *SignedMessageRequest) getReplyTo() []domain.EmailParticipant { return r.ReplyTo }
func (r *SignedMessageRequest) getSubject() string                    { return r.Subject }
func (r *SignedMessageRequest) getHeaders() map[string]string         { return r.Headers }
func (r *SignedMessageRequest) getMessageID() string                  { return r.MessageID }
func (r *SignedMessageRequest) getDate() time.Time                    { return r.Date }

// builder implements Builder.
type builder struct{}

// NewBuilder creates a new MIME builder.
func NewBuilder() Builder {
	return &builder{}
}

// BuildSignedMessage constructs a PGP/MIME signed message per RFC 3156.
func (b *builder) BuildSignedMessage(req *SignedMessageRequest) ([]byte, error) {
	if err := validateSignedRequest(req); err != nil {
		return nil, err
	}

	var buf bytes.Buffer

	// Write top-level headers
	if err := b.writeHeaders(&buf, req); err != nil {
		return nil, err
	}

	// Create multipart/signed boundary
	signedBoundary := generateBoundary("signed")

	// Determine micalg parameter
	micalg := getMicAlg(req.HashAlgo)

	// Write Content-Type for multipart/signed
	buf.WriteString("Content-Type: multipart/signed; protocol=\"application/pgp-signature\";\r\n")
	buf.WriteString(fmt.Sprintf("\tmicalg=%s; boundary=\"%s\"\r\n", micalg, signedBoundary))
	buf.WriteString("\r\n")

	// Write first part: the content to be signed
	buf.WriteString("--" + signedBoundary + "\r\n")

	// Use PreparedContent if provided, otherwise build content inline
	if len(req.PreparedContent) > 0 {
		buf.Write(req.PreparedContent)
	} else {
		if err := b.writeContentPart(&buf, req); err != nil {
			return nil, err
		}
	}

	// Write second part: the signature
	buf.WriteString("\r\n--" + signedBoundary + "\r\n")
	buf.WriteString("Content-Type: application/pgp-signature; name=\"signature.asc\"\r\n")
	buf.WriteString("Content-Description: OpenPGP digital signature\r\n")
	buf.WriteString("Content-Disposition: attachment; filename=\"signature.asc\"\r\n")
	buf.WriteString("\r\n")
	buf.Write(req.Signature)
	buf.WriteString("\r\n--" + signedBoundary + "--\r\n")

	return buf.Bytes(), nil
}

// writeCommonHeaders writes RFC 822 headers common to all message types.
func writeCommonHeaders(buf *bytes.Buffer, req messageRequest) {
	// MIME-Version (required)
	buf.WriteString("MIME-Version: 1.0\r\n")

	// From
	if len(req.getFrom()) > 0 {
		buf.WriteString("From: " + formatAddresses(req.getFrom()) + "\r\n")
	}

	// To
	buf.WriteString("To: " + formatAddresses(req.getTo()) + "\r\n")

	// Cc
	if len(req.getCc()) > 0 {
		buf.WriteString("Cc: " + formatAddresses(req.getCc()) + "\r\n")
	}

	// Bcc (Note: typically not included in headers for security)
	// Omitting Bcc as per RFC 5322 best practices

	// Reply-To
	if len(req.getReplyTo()) > 0 {
		buf.WriteString("Reply-To: " + formatAddresses(req.getReplyTo()) + "\r\n")
	}

	// Subject (encode if contains non-ASCII)
	subject := encodeHeader(req.getSubject())
	buf.WriteString("Subject: " + subject + "\r\n")

	// Date
	date := req.getDate()
	if date.IsZero() {
		date = time.Now()
	}
	buf.WriteString("Date: " + date.Format(time.RFC1123Z) + "\r\n")

	// Message-ID
	if req.getMessageID() != "" {
		buf.WriteString("Message-ID: <" + req.getMessageID() + ">\r\n")
	}

	// Custom headers
	for key, value := range req.getHeaders() {
		buf.WriteString(key + ": " + value + "\r\n")
	}
}

// writeHeaders writes RFC 822 headers for signed messages.
func (b *builder) writeHeaders(buf *bytes.Buffer, req *SignedMessageRequest) error {
	writeCommonHeaders(buf, req)
	return nil
}

// writeContentPart writes the signed content part (body + attachments if any).
func (b *builder) writeContentPart(buf *bytes.Buffer, req *SignedMessageRequest) error {
	if len(req.Attachments) == 0 {
		// Simple case: just body
		return b.writeBodyPart(buf, req)
	}

	// Complex case: multipart/mixed with body and attachments
	mixedBoundary := generateBoundary("mixed")
	buf.WriteString("Content-Type: multipart/mixed; boundary=\"" + mixedBoundary + "\"\r\n")
	buf.WriteString("\r\n")

	// Write body
	buf.WriteString("--" + mixedBoundary + "\r\n")
	if err := b.writeBodyPart(buf, req); err != nil {
		return err
	}

	// Write attachments
	for _, att := range req.Attachments {
		buf.WriteString("\r\n--" + mixedBoundary + "\r\n")
		if err := b.writeAttachmentPart(buf, &att); err != nil {
			return err
		}
	}

	buf.WriteString("\r\n--" + mixedBoundary + "--")
	return nil
}

// writeBodyPart writes the email body.
func (b *builder) writeBodyPart(buf *bytes.Buffer, req *SignedMessageRequest) error {
	contentType := req.ContentType
	if contentType == "" {
		contentType = "text/plain"
	}

	buf.WriteString("Content-Type: " + contentType + "; charset=utf-8\r\n")
	buf.WriteString("Content-Transfer-Encoding: quoted-printable\r\n")
	buf.WriteString("\r\n")

	// Normalize line endings to CRLF (required by RFC 2045)
	body := normalizeLineEndings(req.Body)

	// Encode body with quoted-printable
	var qpBuf bytes.Buffer
	qpWriter := quotedprintable.NewWriter(&qpBuf)
	if _, err := qpWriter.Write([]byte(body)); err != nil {
		return fmt.Errorf("failed to encode body: %w", err)
	}
	if err := qpWriter.Close(); err != nil {
		return fmt.Errorf("failed to close quoted-printable writer: %w", err)
	}

	buf.Write(qpBuf.Bytes())
	return nil
}

// writeAttachmentPart writes an attachment part.
func (b *builder) writeAttachmentPart(buf *bytes.Buffer, att *domain.Attachment) error {
	contentType := att.ContentType
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	// Build Content-Type header
	buf.WriteString("Content-Type: " + contentType + ";\r\n")
	buf.WriteString("\tname=\"" + encodeHeaderParam(att.Filename) + "\"\r\n")

	// Content-Transfer-Encoding
	buf.WriteString("Content-Transfer-Encoding: base64\r\n")

	// Content-Disposition
	disposition := "attachment"
	if att.IsInline {
		disposition = "inline"
	}
	buf.WriteString("Content-Disposition: " + disposition + ";\r\n")
	buf.WriteString("\tfilename=\"" + encodeHeaderParam(att.Filename) + "\"\r\n")

	// Content-ID (for inline images)
	if att.ContentID != "" {
		buf.WriteString("Content-ID: <" + att.ContentID + ">\r\n")
	}

	buf.WriteString("\r\n")

	// Encode content as base64
	encoded := base64.StdEncoding.EncodeToString(att.Content)
	// Split into 76-character lines per RFC 2045
	for i := 0; i < len(encoded); i += 76 {
		end := i + 76
		if end > len(encoded) {
			end = len(encoded)
		}
		buf.WriteString(encoded[i:end] + "\r\n")
	}

	return nil
}

// validateBaseRequest validates the common fields of a message request.
func validateBaseRequest(req messageRequest) error {
	if len(req.getTo()) == 0 {
		return fmt.Errorf("recipient (To) is required")
	}
	if req.getSubject() == "" {
		return fmt.Errorf("subject is required")
	}
	return nil
}

// validateSignedRequest validates the signed message request.
func validateSignedRequest(req *SignedMessageRequest) error {
	if err := validateBaseRequest(req); err != nil {
		return err
	}
	if req.Body == "" {
		return fmt.Errorf("body is required")
	}
	if len(req.Signature) == 0 {
		return fmt.Errorf("signature is required")
	}
	return nil
}

// formatAddresses formats email participants as RFC 822 addresses.
func formatAddresses(participants []domain.EmailParticipant) string {
	var addrs []string
	for _, p := range participants {
		if p.Name != "" {
			// Name <email@example.com>
			encodedName := encodeHeader(p.Name)
			addrs = append(addrs, fmt.Sprintf("%s <%s>", encodedName, p.Email))
		} else {
			// email@example.com
			addrs = append(addrs, p.Email)
		}
	}
	return strings.Join(addrs, ", ")
}

// encodeHeader encodes a header value with RFC 2047 if it contains non-ASCII.
func encodeHeader(value string) string {
	if isASCII(value) {
		return value
	}
	return mime.QEncoding.Encode("utf-8", value)
}

// encodeHeaderParam encodes a header parameter value.
func encodeHeaderParam(value string) string {
	if isASCII(value) {
		return value
	}
	// Use RFC 2231 encoding for non-ASCII parameters
	return mime.QEncoding.Encode("utf-8", value)
}

// isASCII checks if a string contains only ASCII characters.
func isASCII(s string) bool {
	for _, r := range s {
		if r > 127 {
			return false
		}
	}
	return true
}

// normalizeLineEndings converts all line endings to CRLF (required by RFC 2045).
func normalizeLineEndings(s string) string {
	// Replace CRLF with LF first (to handle mixed line endings)
	s = strings.ReplaceAll(s, "\r\n", "\n")
	// Replace CR with LF (for old Mac line endings)
	s = strings.ReplaceAll(s, "\r", "\n")
	// Now replace all LF with CRLF
	s = strings.ReplaceAll(s, "\n", "\r\n")
	return s
}

// getMicAlg returns the micalg parameter for multipart/signed.
func getMicAlg(hashAlgo string) string {
	// Map hash algorithm to micalg value per RFC 3156
	switch strings.ToUpper(hashAlgo) {
	case "SHA256":
		return "pgp-sha256"
	case "SHA512":
		return "pgp-sha512"
	case "SHA384":
		return "pgp-sha384"
	case "SHA1":
		return "pgp-sha1"
	default:
		return "pgp-sha256" // Default
	}
}

// PrepareContentToSign prepares the MIME content part for signing.
// This returns the exact bytes that should be signed with GPG.
func (b *builder) PrepareContentToSign(body, contentType string, attachments []domain.Attachment) ([]byte, error) {
	if contentType == "" {
		contentType = "text/plain"
	}

	var buf bytes.Buffer

	if len(attachments) == 0 {
		// Simple case: just body with headers
		buf.WriteString("Content-Type: " + contentType + "; charset=utf-8\r\n")
		buf.WriteString("Content-Transfer-Encoding: quoted-printable\r\n")
		buf.WriteString("\r\n")

		// Normalize line endings to CRLF
		body = normalizeLineEndings(body)

		// Encode body with quoted-printable
		var qpBuf bytes.Buffer
		qpWriter := quotedprintable.NewWriter(&qpBuf)
		if _, err := qpWriter.Write([]byte(body)); err != nil {
			return nil, fmt.Errorf("failed to encode body: %w", err)
		}
		if err := qpWriter.Close(); err != nil {
			return nil, fmt.Errorf("failed to close quoted-printable writer: %w", err)
		}

		buf.Write(qpBuf.Bytes())
	} else {
		// Complex case: multipart/mixed with body and attachments
		mixedBoundary := generateBoundary("mixed")
		buf.WriteString("Content-Type: multipart/mixed; boundary=\"" + mixedBoundary + "\"\r\n")
		buf.WriteString("\r\n")

		// Write body part
		buf.WriteString("--" + mixedBoundary + "\r\n")
		buf.WriteString("Content-Type: " + contentType + "; charset=utf-8\r\n")
		buf.WriteString("Content-Transfer-Encoding: quoted-printable\r\n")
		buf.WriteString("\r\n")

		body = normalizeLineEndings(body)
		var qpBuf bytes.Buffer
		qpWriter := quotedprintable.NewWriter(&qpBuf)
		if _, err := qpWriter.Write([]byte(body)); err != nil {
			return nil, fmt.Errorf("failed to encode body: %w", err)
		}
		if err := qpWriter.Close(); err != nil {
			return nil, fmt.Errorf("failed to close quoted-printable writer: %w", err)
		}
		buf.Write(qpBuf.Bytes())

		// Write attachment parts
		for _, att := range attachments {
			buf.WriteString("\r\n--" + mixedBoundary + "\r\n")
			if err := b.writeAttachmentPart(&buf, &att); err != nil {
				return nil, err
			}
		}

		buf.WriteString("\r\n--" + mixedBoundary + "--")
	}

	return buf.Bytes(), nil
}

// generateBoundary generates a cryptographically random MIME boundary string.
func generateBoundary(prefix string) string {
	// Use crypto/rand for unpredictable boundaries (SEC-003)
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		// Fallback to less random but still unique - should never happen
		return fmt.Sprintf("=_%s_%d", prefix, b)
	}
	return fmt.Sprintf("=_%s_%s", prefix, hex.EncodeToString(b))
}
