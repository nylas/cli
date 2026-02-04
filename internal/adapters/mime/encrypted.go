package mime

import (
	"bytes"
	"fmt"
	"time"

	"github.com/nylas/cli/internal/domain"
)

// EncryptedMessageRequest contains all data needed to build an encrypted email.
type EncryptedMessageRequest struct {
	// Standard email fields
	From    []domain.EmailParticipant
	To      []domain.EmailParticipant
	Cc      []domain.EmailParticipant
	Bcc     []domain.EmailParticipant
	ReplyTo []domain.EmailParticipant
	Subject string

	// Encrypted content from GPG (ASCII armored PGP message)
	Ciphertext []byte

	// Optional
	Headers   map[string]string
	MessageID string
	Date      time.Time
}

// Implement messageRequest interface for EncryptedMessageRequest.
func (r *EncryptedMessageRequest) getFrom() []domain.EmailParticipant    { return r.From }
func (r *EncryptedMessageRequest) getTo() []domain.EmailParticipant      { return r.To }
func (r *EncryptedMessageRequest) getCc() []domain.EmailParticipant      { return r.Cc }
func (r *EncryptedMessageRequest) getReplyTo() []domain.EmailParticipant { return r.ReplyTo }
func (r *EncryptedMessageRequest) getSubject() string                    { return r.Subject }
func (r *EncryptedMessageRequest) getHeaders() map[string]string         { return r.Headers }
func (r *EncryptedMessageRequest) getMessageID() string                  { return r.MessageID }
func (r *EncryptedMessageRequest) getDate() time.Time                    { return r.Date }

// BuildEncryptedMessage constructs a PGP/MIME encrypted message per RFC 3156 Section 4.
// Structure:
//
//	MIME-Version: 1.0
//	Content-Type: multipart/encrypted;
//	    protocol="application/pgp-encrypted";
//	    boundary="..."
//
//	--boundary
//	Content-Type: application/pgp-encrypted
//
//	Version: 1
//
//	--boundary
//	Content-Type: application/octet-stream
//
//	-----BEGIN PGP MESSAGE-----
//	[Encrypted content]
//	-----END PGP MESSAGE-----
//	--boundary--
func (b *builder) BuildEncryptedMessage(req *EncryptedMessageRequest) ([]byte, error) {
	if err := validateEncryptedRequest(req); err != nil {
		return nil, err
	}

	var buf bytes.Buffer

	// Write top-level headers
	if err := b.writeEncryptedHeaders(&buf, req); err != nil {
		return nil, err
	}

	// Create multipart/encrypted boundary
	encryptedBoundary := generateBoundary("encrypted")

	// Write Content-Type for multipart/encrypted (RFC 3156 Section 4)
	buf.WriteString("Content-Type: multipart/encrypted;\r\n")
	buf.WriteString("\tprotocol=\"application/pgp-encrypted\";\r\n")
	buf.WriteString(fmt.Sprintf("\tboundary=\"%s\"\r\n", encryptedBoundary))
	buf.WriteString("\r\n")

	// Part 1: Version identification (required by RFC 3156)
	buf.WriteString("--" + encryptedBoundary + "\r\n")
	buf.WriteString("Content-Type: application/pgp-encrypted\r\n")
	buf.WriteString("Content-Description: PGP/MIME version identification\r\n")
	buf.WriteString("\r\n")
	buf.WriteString("Version: 1\r\n")

	// Part 2: Encrypted data
	buf.WriteString("\r\n--" + encryptedBoundary + "\r\n")
	buf.WriteString("Content-Type: application/octet-stream; name=\"encrypted.asc\"\r\n")
	buf.WriteString("Content-Description: OpenPGP encrypted message\r\n")
	buf.WriteString("Content-Disposition: inline; filename=\"encrypted.asc\"\r\n")
	buf.WriteString("\r\n")
	buf.Write(req.Ciphertext)
	buf.WriteString("\r\n--" + encryptedBoundary + "--\r\n")

	return buf.Bytes(), nil
}

// writeEncryptedHeaders writes RFC 822 headers for encrypted messages.
func (b *builder) writeEncryptedHeaders(buf *bytes.Buffer, req *EncryptedMessageRequest) error {
	writeCommonHeaders(buf, req)
	return nil
}

// validateEncryptedRequest validates the encrypted message request.
func validateEncryptedRequest(req *EncryptedMessageRequest) error {
	if err := validateBaseRequest(req); err != nil {
		return err
	}
	if len(req.Ciphertext) == 0 {
		return fmt.Errorf("ciphertext is required")
	}
	return nil
}

// PrepareContentToEncrypt prepares the MIME content that will be encrypted.
// This builds a complete MIME body (with attachments if any) that gets encrypted as a whole.
func (b *builder) PrepareContentToEncrypt(body, contentType string, attachments []domain.Attachment) ([]byte, error) {
	// Reuse the PrepareContentToSign logic since the content structure is the same
	// The only difference is what we do with the result (encrypt vs sign)
	return b.PrepareContentToSign(body, contentType, attachments)
}
