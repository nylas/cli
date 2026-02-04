package email

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"strings"

	"github.com/nylas/cli/internal/adapters/gpg"
	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/domain"
)

// decryptGPGEmail decrypts a PGP/MIME encrypted message.
func decryptGPGEmail(ctx context.Context, msg *domain.Message) (*gpg.DecryptResult, error) {
	if msg.RawMIME == "" {
		return nil, fmt.Errorf("no raw MIME data available for decryption")
	}

	// Check if this is an encrypted message
	contentType := extractFullContentType(msg.RawMIME)
	if !isEncryptedMessage(contentType) {
		return nil, fmt.Errorf("message is not PGP/MIME encrypted (Content-Type: %s)", contentType)
	}

	// Parse the multipart message to extract encrypted content
	ciphertext, err := parseEncryptedMIME(msg.RawMIME)
	if err != nil {
		return nil, fmt.Errorf("failed to parse PGP/MIME encrypted message: %w", err)
	}

	// Initialize GPG service
	gpgSvc := gpg.NewService()
	if err := gpgSvc.CheckGPGAvailable(ctx); err != nil {
		return nil, err
	}

	// Decrypt the message
	spinner := common.NewSpinner("Decrypting message...")
	spinner.Start()
	result, err := gpgSvc.DecryptData(ctx, ciphertext)
	spinner.Stop()

	if err != nil {
		return nil, err
	}

	return result, nil
}

// isEncryptedMessage checks if the content type indicates a PGP/MIME encrypted message.
func isEncryptedMessage(contentType string) bool {
	return strings.Contains(contentType, "multipart/encrypted") &&
		strings.Contains(contentType, "application/pgp-encrypted")
}

// parseEncryptedMIME parses a PGP/MIME encrypted message and extracts the ciphertext.
// RFC 3156 Section 4 defines the structure:
// Part 1: application/pgp-encrypted with "Version: 1"
// Part 2: application/octet-stream with the actual encrypted data
func parseEncryptedMIME(rawMIME string) ([]byte, error) {
	// Find the Content-Type header to get the boundary
	fullContentType := extractFullContentType(rawMIME)
	if fullContentType == "" {
		return nil, fmt.Errorf("Content-Type header not found")
	}

	// Parse the Content-Type to extract boundary
	_, params, err := mime.ParseMediaType(fullContentType)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Content-Type '%s': %w", fullContentType, err)
	}

	boundary := params["boundary"]
	if boundary == "" {
		return nil, fmt.Errorf("no boundary found in Content-Type")
	}

	// Find the body section (after headers) for multipart parsing
	headerEnd := findHeaderEnd(rawMIME)
	if headerEnd == -1 {
		return nil, fmt.Errorf("could not find end of headers")
	}

	// Create a reader for the multipart body
	bodySection := rawMIME[headerEnd:]
	mr := multipart.NewReader(strings.NewReader(bodySection), boundary)

	var ciphertext []byte
	partNum := 0

	for {
		part, err := mr.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to read MIME part: %w", err)
		}

		partNum++

		// Part 1: Version identification (application/pgp-encrypted)
		// Part 2: Encrypted data (application/octet-stream)
		if partNum == 2 {
			// This is the encrypted data part
			partContent, err := io.ReadAll(part)
			if err != nil {
				return nil, fmt.Errorf("failed to read encrypted part: %w", err)
			}
			ciphertext = partContent
		}
	}

	if ciphertext == nil {
		return nil, fmt.Errorf("could not find encrypted content part")
	}

	// Trim any surrounding whitespace
	ciphertext = bytes.TrimSpace(ciphertext)

	return ciphertext, nil
}

// printDecryptResult displays the decryption result.
// If showSignature is true and the message was signed, signature verification info is displayed.
func printDecryptResult(result *gpg.DecryptResult, showSignature bool) {
	fmt.Println()
	fmt.Println(strings.Repeat("─", 60))
	_, _ = common.Green.Println("✓ Message decrypted successfully")
	fmt.Println(strings.Repeat("─", 60))

	if result.DecryptKeyID != "" {
		fmt.Printf("  %s %s\n", common.Cyan.Sprint("Decrypted with:"), result.DecryptKeyID)
	}

	// Display signature info only if --verify was also passed
	if showSignature && result.WasSigned {
		fmt.Println()
		if result.SignatureOK {
			_, _ = common.Green.Println("  ✓ Signature verified")
		} else {
			_, _ = common.Red.Println("  ✗ BAD signature!")
		}

		if result.SignerUID != "" {
			fmt.Printf("    %s %s\n", common.Cyan.Sprint("Signer:"), result.SignerUID)
		}
		if result.SignerKeyID != "" {
			fmt.Printf("    %s %s\n", common.Cyan.Sprint("Key ID:"), result.SignerKeyID)
		}
	}

	fmt.Println()
}

// printDecryptedContent displays the decrypted message content.
func printDecryptedContent(plaintext []byte) {
	content := string(plaintext)

	// Check if the decrypted content is a MIME message itself
	if strings.Contains(content, "Content-Type:") {
		// Parse and extract the body from the MIME content
		body := extractBodyFromMIME(content)
		if body != "" {
			content = body
		}
	}

	fmt.Println(strings.Repeat("─", 60))
	_, _ = common.Cyan.Println("Decrypted Content:")
	fmt.Println(strings.Repeat("─", 60))
	fmt.Println()
	fmt.Println(content)
}

// extractBodyFromMIME extracts the body text from a MIME message.
// This handles the case where decrypted content is itself a MIME structure.
func extractBodyFromMIME(mimeContent string) string {
	// Find the Content-Type to determine if it's multipart
	contentType := extractFullContentType(mimeContent)

	// If it's a simple text message, extract after headers
	if strings.HasPrefix(contentType, "text/plain") || strings.HasPrefix(contentType, "text/html") {
		headerEnd := findHeaderEnd(mimeContent)
		if headerEnd != -1 && headerEnd < len(mimeContent) {
			return strings.TrimSpace(mimeContent[headerEnd:])
		}
	}

	// If multipart/mixed or multipart/alternative, extract the text part
	if strings.Contains(contentType, "multipart/") {
		_, params, err := mime.ParseMediaType(contentType)
		if err != nil {
			return mimeContent
		}

		boundary := params["boundary"]
		if boundary == "" {
			return mimeContent
		}

		headerEnd := findHeaderEnd(mimeContent)
		if headerEnd == -1 {
			return mimeContent
		}

		bodySection := mimeContent[headerEnd:]
		mr := multipart.NewReader(strings.NewReader(bodySection), boundary)

		for {
			part, err := mr.NextPart()
			if err == io.EOF {
				break
			}
			if err != nil {
				return mimeContent
			}

			partContentType := part.Header.Get("Content-Type")
			if strings.HasPrefix(partContentType, "text/plain") || strings.HasPrefix(partContentType, "text/html") {
				partContent, err := io.ReadAll(part)
				if err == nil {
					return strings.TrimSpace(string(partContent))
				}
			}
		}
	}

	return mimeContent
}
