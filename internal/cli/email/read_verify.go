package email

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"mime/quotedprintable"
	"strings"

	"github.com/nylas/cli/internal/adapters/gpg"
	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/domain"
)

// verifyGPGSignature verifies the GPG signature of a PGP/MIME message.
func verifyGPGSignature(ctx context.Context, msg *domain.Message) error {
	if msg.RawMIME == "" {
		return fmt.Errorf("no raw MIME data available for verification")
	}

	// Check if this is a signed message
	contentType := extractFullContentType(msg.RawMIME)
	if !strings.Contains(contentType, "multipart/signed") ||
		!strings.Contains(contentType, "application/pgp-signature") {
		return fmt.Errorf("message is not PGP/MIME signed (Content-Type: %s)", contentType)
	}

	// Parse the multipart message to extract body and signature
	body, signature, err := parsePGPMIME(msg.RawMIME)
	if err != nil {
		return fmt.Errorf("failed to parse PGP/MIME message: %w", err)
	}

	// Initialize GPG service
	gpgSvc := gpg.NewService()
	if err := gpgSvc.CheckGPGAvailable(ctx); err != nil {
		return err
	}

	// Verify the signature
	spinner := common.NewSpinner("Verifying GPG signature...")
	spinner.Start()
	result, err := gpgSvc.VerifyDetachedSignature(ctx, body, signature)
	spinner.Stop()

	if err != nil {
		return err
	}

	// Display verification result
	printVerifyResult(result)
	return nil
}

// parsePGPMIME parses a PGP/MIME signed message and extracts the signed body and signature.
func parsePGPMIME(rawMIME string) (body []byte, signature []byte, err error) {
	// Find the Content-Type header to get the boundary
	fullContentType := extractFullContentType(rawMIME)
	if fullContentType == "" {
		return nil, nil, fmt.Errorf("Content-Type header not found")
	}

	// Parse the Content-Type to extract boundary
	_, params, err := mime.ParseMediaType(fullContentType)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse Content-Type '%s': %w", fullContentType, err)
	}

	boundary := params["boundary"]
	if boundary == "" {
		return nil, nil, fmt.Errorf("no boundary found in Content-Type")
	}

	// Find the body section (after headers) for multipart parsing
	headerEnd := findHeaderEnd(rawMIME)
	if headerEnd == -1 {
		return nil, nil, fmt.Errorf("could not find end of headers")
	}

	// Create a reader for the multipart body
	bodySection := rawMIME[headerEnd:]
	mr := multipart.NewReader(strings.NewReader(bodySection), boundary)

	var signedContent []byte
	var signatureContent []byte
	partNum := 0

	for {
		part, err := mr.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, nil, fmt.Errorf("failed to read MIME part: %w", err)
		}

		partNum++

		switch partNum {
		case 1:
			// First part: the signed content
			// For PGP/MIME verification, we need the EXACT bytes that were signed.
			// This includes the part headers.
			signedContent, err = extractSignedContent(rawMIME, boundary)
			if err != nil {
				return nil, nil, fmt.Errorf("failed to extract signed content: %w", err)
			}
		case 2:
			// Second part: the signature
			partContent, err := io.ReadAll(part)
			if err != nil {
				return nil, nil, fmt.Errorf("failed to read signature part: %w", err)
			}

			// Check if signature needs quoted-printable decoding
			cte := part.Header.Get("Content-Transfer-Encoding")
			if strings.EqualFold(cte, "quoted-printable") {
				// Decode quoted-printable
				qpReader := quotedprintable.NewReader(bytes.NewReader(partContent))
				signatureContent, err = io.ReadAll(qpReader)
				if err != nil {
					// If QP decoding fails, try manual decoding
					signatureContent = decodeQuotedPrintable(partContent)
				}
			} else {
				signatureContent = partContent
			}
		}
	}

	if signedContent == nil {
		return nil, nil, fmt.Errorf("could not find signed content part")
	}
	if signatureContent == nil {
		return nil, nil, fmt.Errorf("could not find signature part")
	}

	// Trim any trailing whitespace/newlines from signature
	signatureContent = bytes.TrimSpace(signatureContent)

	return signedContent, signatureContent, nil
}

// extractFullContentType extracts the complete Content-Type header including continuations.
func extractFullContentType(rawMIME string) string {
	lines := strings.Split(rawMIME, "\n")
	var contentTypeParts []string
	inContentType := false

	for _, line := range lines {
		// Remove trailing CR if present
		line = strings.TrimSuffix(line, "\r")

		// Empty line marks end of headers
		if line == "" {
			break
		}

		lineLower := strings.ToLower(line)
		if strings.HasPrefix(lineLower, "content-type:") {
			inContentType = true
			value := strings.TrimSpace(line[len("Content-Type:"):])
			contentTypeParts = append(contentTypeParts, value)
		} else if inContentType {
			if strings.HasPrefix(line, " ") || strings.HasPrefix(line, "\t") {
				contentTypeParts = append(contentTypeParts, strings.TrimSpace(line))
			} else {
				break
			}
		}
	}

	return strings.Join(contentTypeParts, " ")
}

// findHeaderEnd finds the index where headers end (double newline).
func findHeaderEnd(rawMIME string) int {
	// Try CRLF first
	idx := strings.Index(rawMIME, "\r\n\r\n")
	if idx != -1 {
		return idx + 4
	}
	// Try LF
	idx = strings.Index(rawMIME, "\n\n")
	if idx != -1 {
		return idx + 2
	}
	return -1
}

// extractSignedContent extracts the exact bytes of the first MIME part for verification.
// PGP/MIME requires the EXACT bytes including headers, with CRLF line endings.
func extractSignedContent(rawMIME string, boundary string) ([]byte, error) {
	boundaryMarker := "--" + boundary

	// Find the first boundary
	firstBoundaryIdx := strings.Index(rawMIME, boundaryMarker)
	if firstBoundaryIdx == -1 {
		return nil, fmt.Errorf("could not find first boundary")
	}

	// Move past the boundary line and its line ending
	partStart := firstBoundaryIdx + len(boundaryMarker)
	// Skip CRLF or LF after boundary
	if partStart < len(rawMIME) {
		if rawMIME[partStart] == '\r' && partStart+1 < len(rawMIME) && rawMIME[partStart+1] == '\n' {
			partStart += 2
		} else if rawMIME[partStart] == '\n' {
			partStart++
		}
	}

	// Find the second boundary
	secondBoundaryIdx := strings.Index(rawMIME[partStart:], boundaryMarker)
	if secondBoundaryIdx == -1 {
		return nil, fmt.Errorf("could not find second boundary")
	}

	// Extract the signed content (everything between boundaries)
	signedContent := rawMIME[partStart : partStart+secondBoundaryIdx]

	// The signed content should NOT include the CRLF immediately before the boundary
	// PGP/MIME: the delimiter CRLF belongs to the boundary, not the content
	signedContent = strings.TrimSuffix(signedContent, "\r\n")
	signedContent = strings.TrimSuffix(signedContent, "\n")

	// Normalize line endings to CRLF for signature verification
	// First convert any CRLF to LF (to handle mixed line endings)
	signedContent = strings.ReplaceAll(signedContent, "\r\n", "\n")
	// Then convert all LF to CRLF
	signedContent = strings.ReplaceAll(signedContent, "\n", "\r\n")

	return []byte(signedContent), nil
}

// decodeQuotedPrintable decodes quoted-printable encoded content.
// This is a fallback decoder that handles common patterns when the stdlib fails.
func decodeQuotedPrintable(data []byte) []byte {
	result := string(data)

	// FIRST: Handle soft line breaks (= at end of line)
	// The order is critical - soft breaks must be removed before decoding hex
	result = strings.ReplaceAll(result, "=\r\n", "")
	result = strings.ReplaceAll(result, "=\n", "")

	// THEN: Decode =XX hex sequences
	// Process character by character to handle all hex codes
	var decoded strings.Builder
	i := 0
	for i < len(result) {
		if result[i] == '=' && i+2 < len(result) {
			// Check if this is a valid hex sequence
			hex := result[i+1 : i+3]
			if isHexPair(hex) {
				b := hexToByte(hex)
				decoded.WriteByte(b)
				i += 3
				continue
			}
		}
		decoded.WriteByte(result[i])
		i++
	}

	return []byte(decoded.String())
}

// isHexPair checks if a 2-character string is a valid hex pair.
func isHexPair(s string) bool {
	if len(s) != 2 {
		return false
	}
	for _, c := range s {
		isDigit := c >= '0' && c <= '9'
		isUpperHex := c >= 'A' && c <= 'F'
		isLowerHex := c >= 'a' && c <= 'f'
		if !isDigit && !isUpperHex && !isLowerHex {
			return false
		}
	}
	return true
}

// hexToByte converts a 2-character hex string to a byte.
func hexToByte(s string) byte {
	var result byte
	for _, c := range s {
		result <<= 4
		switch {
		case c >= '0' && c <= '9':
			result |= byte(c - '0')
		case c >= 'A' && c <= 'F':
			result |= byte(c - 'A' + 10)
		case c >= 'a' && c <= 'f':
			result |= byte(c - 'a' + 10)
		}
	}
	return result
}

// printVerifyResult displays the GPG verification result.
func printVerifyResult(result *gpg.VerifyResult) {
	fmt.Println()
	fmt.Println(strings.Repeat("─", 60))

	if result.Valid {
		_, _ = common.Green.Println("✓ Good signature")
	} else {
		_, _ = common.Red.Println("✗ BAD signature")
	}

	fmt.Println(strings.Repeat("─", 60))

	if result.SignerUID != "" {
		fmt.Printf("  %s %s\n", common.Cyan.Sprint("Signer:"), result.SignerUID)
	}
	if result.SignerKeyID != "" {
		fmt.Printf("  %s %s\n", common.Cyan.Sprint("Key ID:"), result.SignerKeyID)
	}
	if result.Fingerprint != "" {
		fmt.Printf("  %s %s\n", common.Cyan.Sprint("Fingerprint:"), result.Fingerprint)
	}
	if !result.SignedAt.IsZero() {
		fmt.Printf("  %s %s\n", common.Cyan.Sprint("Signed:"), result.SignedAt.Format("Mon, 02 Jan 2006 15:04:05 MST"))
	}
	if result.TrustLevel != "" {
		trustColor := common.Yellow
		switch result.TrustLevel {
		case "ultimate", "full":
			trustColor = common.Green
		case "never":
			trustColor = common.Red
		}
		fmt.Printf("  %s %s\n", common.Cyan.Sprint("Trust:"), trustColor.Sprint(result.TrustLevel))
	}

	fmt.Println()
}
