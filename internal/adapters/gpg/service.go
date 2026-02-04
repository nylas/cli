package gpg

import (
	"bytes"
	"context"
	"fmt"
	"net/mail"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// gpgKeyIDPattern matches valid GPG key IDs (8-40 hex characters)
var gpgKeyIDPattern = regexp.MustCompile(`^[A-Fa-f0-9]{8,40}$`)

// isValidGPGKeyID validates that a key ID is in a safe format for use with GPG commands.
// Valid formats:
// - 8-40 hexadecimal characters (short key ID, long key ID, or fingerprint)
// - Valid email address (GPG allows email as key identifier)
func isValidGPGKeyID(keyID string) bool {
	keyID = strings.TrimSpace(keyID)
	if keyID == "" {
		return false
	}

	// Check if it's a valid hex key ID (8-40 chars)
	if gpgKeyIDPattern.MatchString(keyID) {
		return true
	}

	// Check if it's a valid email address
	_, err := mail.ParseAddress(keyID)
	if err == nil {
		return true
	}

	// Also accept "Name <email>" format
	if strings.Contains(keyID, "<") && strings.Contains(keyID, ">") {
		_, err := mail.ParseAddress(keyID)
		return err == nil
	}

	return false
}

// Service provides GPG signing, verification, and encryption operations.
type Service interface {
	// CheckGPGAvailable verifies GPG is installed and accessible.
	CheckGPGAvailable(ctx context.Context) error

	// ListSigningKeys lists all available secret keys for signing.
	ListSigningKeys(ctx context.Context) ([]KeyInfo, error)

	// GetDefaultSigningKey gets the default signing key from git config.
	GetDefaultSigningKey(ctx context.Context) (*KeyInfo, error)

	// FindKeyByEmail finds a signing key that contains the given email in its UIDs.
	// Returns the key ID (not the email) for use with --local-user.
	FindKeyByEmail(ctx context.Context, email string) (*KeyInfo, error)

	// SignData signs data with the specified key and returns a detached signature.
	// senderEmail is optional - when provided, it embeds that email in the Signer's User ID subpacket.
	SignData(ctx context.Context, keyID string, data []byte, senderEmail string) (*SignResult, error)

	// VerifyDetachedSignature verifies a detached signature against data.
	// Returns verification result including signer info and trust level.
	VerifyDetachedSignature(ctx context.Context, data []byte, signature []byte) (*VerifyResult, error)

	// ListPublicKeys lists all public keys in the keyring.
	ListPublicKeys(ctx context.Context) ([]KeyInfo, error)

	// FindPublicKeyByEmail finds a public key by email, auto-fetching from key servers if not found locally.
	FindPublicKeyByEmail(ctx context.Context, email string) (*KeyInfo, error)

	// EncryptData encrypts data for one or more recipients using their public keys.
	EncryptData(ctx context.Context, recipientKeyIDs []string, data []byte) (*EncryptResult, error)

	// SignAndEncryptData signs data with the sender's private key and encrypts for recipients.
	// This provides maximum security: only recipients can decrypt, and they can verify the sender.
	SignAndEncryptData(ctx context.Context, signerKeyID string, recipientKeyIDs []string, data []byte, senderEmail string) (*EncryptResult, error)

	// DecryptData decrypts PGP encrypted data using the user's private key.
	// Returns the decrypted plaintext along with optional signature verification info.
	DecryptData(ctx context.Context, ciphertext []byte) (*DecryptResult, error)
}

// service implements Service using the system GPG command.
type service struct{}

// NewService creates a new GPG service.
func NewService() Service {
	return &service{}
}

// CheckGPGAvailable verifies GPG is installed.
func (s *service) CheckGPGAvailable(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, "gpg", "--version")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("GPG not found. Install with: sudo apt install gnupg (Linux) or brew install gnupg (macOS)")
	}
	return nil
}

// ListSigningKeys lists all secret keys available for signing.
func (s *service) ListSigningKeys(ctx context.Context) ([]KeyInfo, error) {
	// Use --with-colons format for reliable parsing
	cmd := exec.CommandContext(ctx, "gpg", "--list-secret-keys", "--with-colons", "--with-fingerprint")
	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && len(exitErr.Stderr) > 0 {
			return nil, fmt.Errorf("gpg list keys failed: %s", string(exitErr.Stderr))
		}
		return nil, fmt.Errorf("gpg list keys failed: %w", err)
	}

	return parseSecretKeys(string(output))
}

// GetDefaultSigningKey retrieves the default signing key from git config.
func (s *service) GetDefaultSigningKey(ctx context.Context) (*KeyInfo, error) {
	// Try to get key from git config
	cmd := exec.CommandContext(ctx, "git", "config", "--get", "user.signingkey")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("no default GPG key configured. Set with: git config --global user.signingkey <KEY_ID>")
	}

	keyID := strings.TrimSpace(string(output))
	if keyID == "" {
		return nil, fmt.Errorf("git user.signingkey is empty")
	}

	// Get full key info
	keys, err := s.ListSigningKeys(ctx)
	if err != nil {
		return nil, err
	}

	// Find matching key
	for i := range keys {
		if strings.HasSuffix(keys[i].Fingerprint, keyID) || keys[i].KeyID == keyID {
			return &keys[i], nil
		}
	}

	return nil, fmt.Errorf("git signing key %s not found in GPG keyring", keyID)
}

// FindKeyByEmail finds a signing key that contains the given email in its UIDs.
// Returns the KeyInfo with the actual key ID for use with --local-user.
// This is important because GPG's --sender option only works correctly when
// --local-user is a key ID, not an email address.
func (s *service) FindKeyByEmail(ctx context.Context, email string) (*KeyInfo, error) {
	keys, err := s.ListSigningKeys(ctx)
	if err != nil {
		return nil, err
	}

	// Normalize email for comparison
	email = strings.ToLower(strings.TrimSpace(email))

	// Find key with matching email in UIDs
	for i := range keys {
		for _, uid := range keys[i].UIDs {
			// Extract email from UID (format: "Name <email@domain.com>")
			uidLower := strings.ToLower(uid)
			if strings.Contains(uidLower, "<"+email+">") || uidLower == email {
				return &keys[i], nil
			}
		}
	}

	return nil, fmt.Errorf("no GPG key found for email %s", email)
}

// SignData creates a detached signature for the given data.
// senderEmail is optional - when provided, it embeds that email in the Signer's User ID subpacket.
func (s *service) SignData(ctx context.Context, keyID string, data []byte, senderEmail string) (*SignResult, error) {
	// Validate keyID to prevent command injection (SEC-001)
	if !isValidGPGKeyID(keyID) {
		return nil, fmt.Errorf("invalid GPG key ID format: %q", keyID)
	}

	// Validate senderEmail if provided
	if senderEmail != "" {
		if _, err := mail.ParseAddress(senderEmail); err != nil {
			return nil, fmt.Errorf("invalid sender email format: %q", senderEmail)
		}
	}

	// Build GPG arguments
	args := []string{
		"--detach-sign",
		"--armor",
		"--local-user", keyID,
	}

	// Add --sender to explicitly set Signer's User ID subpacket
	// This ensures the correct email appears in the signature when
	// the key has multiple UIDs
	if senderEmail != "" {
		args = append(args, "--sender", senderEmail)
	}

	args = append(args, "--output", "-")

	// Use detached signature with ASCII armor
	// #nosec G204 - keyID and senderEmail are validated above (isValidGPGKeyID, mail.ParseAddress)
	cmd := exec.CommandContext(ctx, "gpg", args...)

	cmd.Stdin = bytes.NewReader(data)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		errMsg := stderr.String()
		if strings.Contains(errMsg, "No secret key") {
			return nil, fmt.Errorf("GPG key %s not found or not usable for signing", keyID)
		}
		if strings.Contains(errMsg, "Timeout") || strings.Contains(errMsg, "timeout") {
			return nil, fmt.Errorf("GPG passphrase prompt timed out. Please ensure gpg-agent is running")
		}
		return nil, fmt.Errorf("gpg signing failed: %s", errMsg)
	}

	signature := stdout.Bytes()
	if len(signature) == 0 {
		return nil, fmt.Errorf("gpg produced empty signature")
	}

	// Extract hash algorithm from signature (if present in stderr)
	hashAlgo := extractHashAlgorithm(stderr.String())

	return &SignResult{
		Signature: signature,
		KeyID:     keyID,
		SignedAt:  time.Now(),
		HashAlgo:  hashAlgo,
	}, nil
}

// KeyServers is the list of public key servers to try when fetching keys.
// Servers are tried in order until one succeeds.
var KeyServers = []string{
	"keys.openpgp.org",     // Modern, privacy-focused
	"keyserver.ubuntu.com", // Ubuntu's server, widely used
	"pgp.mit.edu",          // MIT's classic server
	"keys.gnupg.net",       // GnuPG project's pool
}

// VerifyDetachedSignature verifies a detached signature against data.
func (s *service) VerifyDetachedSignature(ctx context.Context, data []byte, signature []byte) (*VerifyResult, error) {
	// Create temporary files for data and signature
	dataFile, err := createTempFile("gpg-verify-data-", data)
	if err != nil {
		return nil, fmt.Errorf("failed to create temp data file: %w", err)
	}
	defer func() { _ = dataFile.Close(); _ = removeFile(dataFile.Name()) }()

	sigFile, err := createTempFile("gpg-verify-sig-", signature)
	if err != nil {
		return nil, fmt.Errorf("failed to create temp signature file: %w", err)
	}
	defer func() { _ = sigFile.Close(); _ = removeFile(sigFile.Name()) }()

	// First attempt: verify with local keys
	result, stderrOutput, err := s.runVerify(ctx, sigFile.Name(), dataFile.Name())

	// If key not found, try to fetch from key server
	if err != nil && (strings.Contains(stderrOutput, "No public key") || strings.Contains(stderrOutput, "Can't check signature: No public key")) {
		// Extract key ID from error message
		keyID := extractKeyIDFromError(stderrOutput)
		// Validate extracted keyID before using (SEC-002)
		if keyID != "" && gpgKeyIDPattern.MatchString(keyID) {
			// Try to fetch the key from key server
			if fetchErr := s.fetchKeyFromServer(ctx, keyID); fetchErr == nil {
				// Retry verification with the newly imported key
				result, stderrOutput, err = s.runVerify(ctx, sigFile.Name(), dataFile.Name())
			}
		}

		// If still no key, return helpful error
		if err != nil && (strings.Contains(stderrOutput, "No public key") || strings.Contains(stderrOutput, "Can't check signature: No public key")) {
			return nil, fmt.Errorf("public key not found (tried %d key servers). Import manually with: gpg --keyserver keys.openpgp.org --recv-keys <KEY_ID>", len(KeyServers))
		}
	}

	if err != nil {
		if strings.Contains(stderrOutput, "BAD signature") {
			result.Valid = false
			return result, nil
		}
		// For other errors, return the result if we got valid parsing
		if result != nil && result.SignerKeyID != "" {
			return result, nil
		}
		return nil, fmt.Errorf("gpg verification failed: %s", stderrOutput)
	}

	return result, nil
}

// runVerify executes gpg --verify and returns the parsed result.
func (s *service) runVerify(ctx context.Context, sigFile, dataFile string) (*VerifyResult, string, error) {
	cmd := exec.CommandContext(ctx, "gpg", "--verify", "--status-fd", "1", sigFile, dataFile)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	statusOutput := stdout.String()
	stderrOutput := stderr.String()

	result := parseVerifyOutput(statusOutput, stderrOutput)
	return result, stderrOutput, err
}

// fetchKeyFromServer attempts to fetch a public key from multiple key servers.
// It tries each server in order until one succeeds.
func (s *service) fetchKeyFromServer(ctx context.Context, keyID string) error {
	var lastErr error

	for _, server := range KeyServers {
		// #nosec G204 - keyID is validated by gpgKeyIDPattern.MatchString before this function is called
		cmd := exec.CommandContext(ctx, "gpg", "--keyserver", server, "--recv-keys", keyID)
		var stderr bytes.Buffer
		cmd.Stderr = &stderr

		if err := cmd.Run(); err != nil {
			lastErr = fmt.Errorf("failed to fetch from %s: %w", server, err)
			continue
		}
		// Success
		return nil
	}

	return fmt.Errorf("failed to fetch key %s from any server: %w", keyID, lastErr)
}

// extractKeyIDFromError extracts the key ID from GPG's "No public key" error message.
func extractKeyIDFromError(stderr string) string {
	// GPG outputs: "gpg: using RSA key ABCD1234..." or "gpg: Can't check signature: No public key"
	// Try to find key ID in the output
	re := regexp.MustCompile(`using \w+ key ([A-F0-9]+)`)
	if matches := re.FindStringSubmatch(stderr); len(matches) > 1 {
		return matches[1]
	}

	// Also try: "gpg: Good signature from..." followed by key ID
	re = regexp.MustCompile(`key ([A-F0-9]{8,})`)
	if matches := re.FindStringSubmatch(stderr); len(matches) > 1 {
		return matches[1]
	}

	return ""
}

// createTempFile creates a temporary file with the given content.
func createTempFile(prefix string, content []byte) (*os.File, error) {
	f, err := os.CreateTemp("", prefix)
	if err != nil {
		return nil, err
	}
	if _, err := f.Write(content); err != nil {
		_ = f.Close()
		_ = os.Remove(f.Name())
		return nil, err
	}
	return f, nil
}

// removeFile removes a file, ignoring errors.
func removeFile(path string) error {
	return os.Remove(path)
}

// parseVerifyOutput parses GPG --status-fd output and stderr for verification info.
func parseVerifyOutput(statusOutput, stderrOutput string) *VerifyResult {
	result := &VerifyResult{}

	// Parse status output for structured info
	lines := strings.Split(statusOutput, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)

		// [GNUPG:] GOODSIG <keyid> <uid>
		if strings.HasPrefix(line, "[GNUPG:] GOODSIG ") {
			result.Valid = true
			parts := strings.SplitN(line, " ", 4)
			if len(parts) >= 3 {
				result.SignerKeyID = parts[2]
			}
			if len(parts) >= 4 {
				result.SignerUID = parts[3]
			}
		}

		// [GNUPG:] BADSIG <keyid> <uid>
		if strings.HasPrefix(line, "[GNUPG:] BADSIG ") {
			result.Valid = false
			parts := strings.SplitN(line, " ", 4)
			if len(parts) >= 3 {
				result.SignerKeyID = parts[2]
			}
			if len(parts) >= 4 {
				result.SignerUID = parts[3]
			}
		}

		// [GNUPG:] VALIDSIG <fpr> <date> <timestamp> ...
		if strings.HasPrefix(line, "[GNUPG:] VALIDSIG ") {
			parts := strings.Fields(line)
			if len(parts) >= 3 {
				result.Fingerprint = parts[2]
			}
			if len(parts) >= 5 {
				if ts, err := strconv.ParseInt(parts[4], 10, 64); err == nil {
					result.SignedAt = time.Unix(ts, 0)
				}
			}
		}

		// [GNUPG:] TRUST_ULTIMATE, TRUST_FULL, TRUST_MARGINAL, TRUST_UNDEFINED, TRUST_NEVER
		if strings.HasPrefix(line, "[GNUPG:] TRUST_") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				trust := strings.TrimPrefix(parts[1], "TRUST_")
				result.TrustLevel = strings.ToLower(trust)
			}
		}
	}

	// Also check stderr for "Good signature" as fallback
	if !result.Valid && strings.Contains(stderrOutput, "Good signature") {
		result.Valid = true
	}

	// Extract key ID from stderr if not found in status
	if result.SignerKeyID == "" {
		// Look for "using RSA key <keyid>" pattern
		re := regexp.MustCompile(`using \w+ key ([A-F0-9]+)`)
		if matches := re.FindStringSubmatch(stderrOutput); len(matches) > 1 {
			result.SignerKeyID = matches[1]
		}
	}

	// Extract signer UID from stderr if not found
	if result.SignerUID == "" {
		// Look for "Good signature from "Name <email>"" pattern
		re := regexp.MustCompile(`Good signature from "([^"]+)"`)
		if matches := re.FindStringSubmatch(stderrOutput); len(matches) > 1 {
			result.SignerUID = matches[1]
		}
	}

	return result
}

// parseSecretKeys parses GPG --with-colons output format for secret keys.
// Format: https://github.com/CSNW/gnupg/blob/master/doc/DETAILS
func parseSecretKeys(output string) ([]KeyInfo, error) {
	keys, err := parseKeys(output, "sec")
	if err != nil {
		return nil, err
	}
	if len(keys) == 0 {
		return nil, fmt.Errorf("no GPG secret keys found. Generate one with: gpg --gen-key")
	}
	return keys, nil
}

// parsePublicKeys parses GPG --with-colons output format for public keys.
func parsePublicKeys(output string) ([]KeyInfo, error) {
	return parseKeys(output, "pub")
}

// parseKeys parses GPG --with-colons output format.
// recordPrefix is "sec" for secret keys or "pub" for public keys.
func parseKeys(output string, recordPrefix string) ([]KeyInfo, error) {
	var keys []KeyInfo
	var currentKey *KeyInfo

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		fields := strings.Split(line, ":")
		if len(fields) < 2 {
			continue
		}

		recordType := fields[0]

		switch recordType {
		case recordPrefix: // Primary key (sec or pub)
			if currentKey != nil {
				keys = append(keys, *currentKey)
			}
			currentKey = &KeyInfo{
				Trust: fields[1],
			}
			if len(fields) > 4 {
				currentKey.KeyID = fields[4]
			}
			if len(fields) > 5 && fields[5] != "" {
				if ts, err := strconv.ParseInt(fields[5], 10, 64); err == nil {
					created := time.Unix(ts, 0)
					currentKey.Created = created
				}
			}
			if len(fields) > 6 && fields[6] != "" {
				if ts, err := strconv.ParseInt(fields[6], 10, 64); err == nil {
					expires := time.Unix(ts, 0)
					currentKey.Expires = &expires
				}
			}

		case "fpr": // Fingerprint
			if currentKey != nil && len(fields) > 9 {
				currentKey.Fingerprint = fields[9]
			}

		case "uid": // User ID
			if currentKey != nil && len(fields) > 9 {
				currentKey.UIDs = append(currentKey.UIDs, fields[9])
			}
		}
	}

	// Append last key
	if currentKey != nil {
		keys = append(keys, *currentKey)
	}

	return keys, nil
}

// extractHashAlgorithm extracts the hash algorithm from GPG stderr output.
func extractHashAlgorithm(stderr string) string {
	// GPG outputs hash info like: "gpg: using RSA key ... digest algorithm SHA256"
	re := regexp.MustCompile(`digest algorithm (\w+)`)
	matches := re.FindStringSubmatch(stderr)
	if len(matches) > 1 {
		return strings.ToUpper(matches[1])
	}
	return "SHA256" // Default assumption
}
