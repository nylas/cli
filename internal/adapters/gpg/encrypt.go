package gpg

import (
	"bytes"
	"context"
	"fmt"
	"net/mail"
	"os/exec"
	"regexp"
	"strings"
	"time"
)

// ListPublicKeys lists all public keys in the keyring.
func (s *service) ListPublicKeys(ctx context.Context) ([]KeyInfo, error) {
	cmd := exec.CommandContext(ctx, "gpg", "--list-keys", "--with-colons", "--with-fingerprint")
	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && len(exitErr.Stderr) > 0 {
			return nil, fmt.Errorf("gpg list public keys failed: %s", string(exitErr.Stderr))
		}
		// Empty keyring is not an error - return empty slice
		return []KeyInfo{}, nil
	}

	return parsePublicKeys(string(output))
}

// FindPublicKeyByEmail finds a public key by email, auto-fetching from key servers if not found locally.
func (s *service) FindPublicKeyByEmail(ctx context.Context, email string) (*KeyInfo, error) {
	// Normalize email for comparison
	email = strings.ToLower(strings.TrimSpace(email))

	// Step 1: Search local keyring first
	keys, err := s.ListPublicKeys(ctx)
	if err != nil {
		return nil, err
	}

	for i := range keys {
		if keyMatchesEmail(&keys[i], email) {
			// Check if key is expired
			if keys[i].Expires != nil && keys[i].Expires.Before(time.Now()) {
				continue // Skip expired keys
			}
			return &keys[i], nil
		}
	}

	// Step 2: Not found locally - try to fetch from key servers
	if fetchErr := s.fetchKeyByEmail(ctx, email); fetchErr != nil {
		return nil, fmt.Errorf("no public key found for %s (checked local keyring and %d key servers): %w",
			email, len(KeyServers), fetchErr)
	}

	// Step 3: Retry local search after fetch
	keys, err = s.ListPublicKeys(ctx)
	if err != nil {
		return nil, err
	}

	for i := range keys {
		if keyMatchesEmail(&keys[i], email) {
			return &keys[i], nil
		}
	}

	return nil, fmt.Errorf("key fetched but not found for %s", email)
}

// fetchKeyByEmail tries to fetch a public key by email from key servers.
func (s *service) fetchKeyByEmail(ctx context.Context, email string) error {
	// Validate email format
	if _, err := mail.ParseAddress(email); err != nil {
		return fmt.Errorf("invalid email format: %q", email)
	}

	var lastErr error
	for _, server := range KeyServers {
		// Use --auto-key-locate with WKD (Web Key Directory) and keyserver fallback
		// #nosec G204 - email is validated by mail.ParseAddress above
		cmd := exec.CommandContext(ctx, "gpg", "--auto-key-locate", "wkd,keyserver", "--keyserver", server, "--locate-keys", email)
		var stderr bytes.Buffer
		cmd.Stderr = &stderr

		if err := cmd.Run(); err != nil {
			lastErr = fmt.Errorf("failed to fetch from %s: %w", server, err)
			continue
		}
		// Success
		return nil
	}

	return fmt.Errorf("failed to fetch key for %s from any server: %w", email, lastErr)
}

// keyMatchesEmail checks if a key contains the given email in its UIDs.
func keyMatchesEmail(key *KeyInfo, email string) bool {
	email = strings.ToLower(email)
	for _, uid := range key.UIDs {
		uidLower := strings.ToLower(uid)
		// Check for email in angle brackets: "Name <email@example.com>"
		if strings.Contains(uidLower, "<"+email+">") {
			return true
		}
		// Check for bare email
		if uidLower == email {
			return true
		}
	}
	return false
}

// EncryptData encrypts data for one or more recipients using their public keys.
func (s *service) EncryptData(ctx context.Context, recipientKeyIDs []string, data []byte) (*EncryptResult, error) {
	if len(recipientKeyIDs) == 0 {
		return nil, fmt.Errorf("at least one recipient key ID is required")
	}

	// Validate all key IDs
	for _, keyID := range recipientKeyIDs {
		if !isValidGPGKeyID(keyID) {
			return nil, fmt.Errorf("invalid GPG key ID format: %q", keyID)
		}
	}

	// Build GPG arguments
	args := []string{
		"--encrypt",
		"--armor",
		"--trust-model", "always", // Trust the key for this operation
	}

	// Add each recipient
	for _, keyID := range recipientKeyIDs {
		args = append(args, "--recipient", keyID)
	}

	args = append(args, "--output", "-")

	// #nosec G204 - recipientKeyIDs are validated above by isValidGPGKeyID
	cmd := exec.CommandContext(ctx, "gpg", args...)

	cmd.Stdin = bytes.NewReader(data)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		errMsg := stderr.String()
		if strings.Contains(errMsg, "No public key") {
			return nil, fmt.Errorf("public key not found for one or more recipients")
		}
		if strings.Contains(errMsg, "unusable public key") {
			return nil, fmt.Errorf("one or more recipient keys are unusable (expired, revoked, or invalid)")
		}
		return nil, fmt.Errorf("gpg encryption failed: %s", errMsg)
	}

	ciphertext := stdout.Bytes()
	if len(ciphertext) == 0 {
		return nil, fmt.Errorf("gpg produced empty ciphertext")
	}

	return &EncryptResult{
		Ciphertext:    ciphertext,
		RecipientKeys: recipientKeyIDs,
	}, nil
}

// SignAndEncryptData signs data with the sender's private key and encrypts for recipients.
// This provides maximum security: only recipients can decrypt, and they can verify the sender.
func (s *service) SignAndEncryptData(ctx context.Context, signerKeyID string, recipientKeyIDs []string, data []byte, senderEmail string) (*EncryptResult, error) {
	if signerKeyID == "" {
		return nil, fmt.Errorf("signer key ID is required for sign+encrypt")
	}
	if len(recipientKeyIDs) == 0 {
		return nil, fmt.Errorf("at least one recipient key ID is required")
	}

	// Validate signer key ID
	if !isValidGPGKeyID(signerKeyID) {
		return nil, fmt.Errorf("invalid signer GPG key ID format: %q", signerKeyID)
	}

	// Validate all recipient key IDs
	for _, keyID := range recipientKeyIDs {
		if !isValidGPGKeyID(keyID) {
			return nil, fmt.Errorf("invalid recipient GPG key ID format: %q", keyID)
		}
	}

	// Validate senderEmail if provided
	if senderEmail != "" {
		if _, err := mail.ParseAddress(senderEmail); err != nil {
			return nil, fmt.Errorf("invalid sender email format: %q", senderEmail)
		}
	}

	// Build GPG arguments for sign+encrypt
	args := []string{
		"--sign",
		"--encrypt",
		"--armor",
		"--trust-model", "always",
		"--local-user", signerKeyID,
	}

	// Add --sender for proper UID embedding
	if senderEmail != "" {
		args = append(args, "--sender", senderEmail)
	}

	// Add each recipient
	for _, keyID := range recipientKeyIDs {
		args = append(args, "--recipient", keyID)
	}

	args = append(args, "--output", "-")

	// #nosec G204 - all key IDs are validated above by isValidGPGKeyID
	cmd := exec.CommandContext(ctx, "gpg", args...)

	cmd.Stdin = bytes.NewReader(data)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		errMsg := stderr.String()
		if strings.Contains(errMsg, "No secret key") {
			return nil, fmt.Errorf("GPG signing key %s not found or not usable", signerKeyID)
		}
		if strings.Contains(errMsg, "No public key") {
			return nil, fmt.Errorf("public key not found for one or more recipients")
		}
		if strings.Contains(errMsg, "unusable public key") {
			return nil, fmt.Errorf("one or more recipient keys are unusable (expired, revoked, or invalid)")
		}
		if strings.Contains(errMsg, "Timeout") || strings.Contains(errMsg, "timeout") {
			return nil, fmt.Errorf("GPG passphrase prompt timed out. Please ensure gpg-agent is running")
		}
		return nil, fmt.Errorf("gpg sign+encrypt failed: %s", errMsg)
	}

	ciphertext := stdout.Bytes()
	if len(ciphertext) == 0 {
		return nil, fmt.Errorf("gpg produced empty ciphertext")
	}

	return &EncryptResult{
		Ciphertext:    ciphertext,
		RecipientKeys: recipientKeyIDs,
	}, nil
}

// DecryptData decrypts PGP encrypted data using the user's private key.
// It also handles signed+encrypted messages, returning signature verification info.
func (s *service) DecryptData(ctx context.Context, ciphertext []byte) (*DecryptResult, error) {
	if len(ciphertext) == 0 {
		return nil, fmt.Errorf("ciphertext is empty")
	}

	// Build GPG arguments
	args := []string{
		"--decrypt",
		"--status-fd", "2", // Output status to stderr for parsing
	}

	// #nosec G204 - no user input in command
	cmd := exec.CommandContext(ctx, "gpg", args...)

	cmd.Stdin = bytes.NewReader(ciphertext)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	stderrOutput := stderr.String()

	// Parse the result even if there was an error (bad signature still decrypts)
	result := parseDecryptOutput(stderrOutput)
	result.Plaintext = stdout.Bytes()

	if err != nil {
		// Check for common errors
		if strings.Contains(stderrOutput, "No secret key") {
			return nil, fmt.Errorf("no secret key available to decrypt this message. The message was encrypted for a different recipient")
		}
		if strings.Contains(stderrOutput, "decryption failed") {
			return nil, fmt.Errorf("decryption failed: %s", stderrOutput)
		}
		// If we got plaintext despite the error (e.g., bad signature), return the result
		if len(result.Plaintext) > 0 {
			return result, nil
		}
		return nil, fmt.Errorf("gpg decryption failed: %s", stderrOutput)
	}

	if len(result.Plaintext) == 0 {
		return nil, fmt.Errorf("gpg produced empty plaintext")
	}

	return result, nil
}

// parseDecryptOutput parses GPG status output during decryption.
func parseDecryptOutput(stderrOutput string) *DecryptResult {
	result := &DecryptResult{}

	// Check for signature status
	if strings.Contains(stderrOutput, "GOODSIG") || strings.Contains(stderrOutput, "Good signature") {
		result.WasSigned = true
		result.SignatureOK = true
	} else if strings.Contains(stderrOutput, "BADSIG") || strings.Contains(stderrOutput, "BAD signature") {
		result.WasSigned = true
		result.SignatureOK = false
	}

	// Extract signer key ID using regex
	// Pattern: "gpg: Signature made ... using RSA key <KEY_ID>"
	// or "[GNUPG:] GOODSIG <KEY_ID> <UID>"
	keyIDPattern := regexp.MustCompile(`using \w+ key ([A-F0-9]+)`)
	if matches := keyIDPattern.FindStringSubmatch(stderrOutput); len(matches) > 1 {
		result.SignerKeyID = matches[1]
	}

	// Also try GOODSIG/BADSIG format: "[GNUPG:] GOODSIG <keyid> <uid>"
	if result.SignerKeyID == "" {
		goodsigPattern := regexp.MustCompile(`\[GNUPG:\] (?:GOODSIG|BADSIG) ([A-F0-9]+) (.+)`)
		if matches := goodsigPattern.FindStringSubmatch(stderrOutput); len(matches) > 2 {
			result.SignerKeyID = matches[1]
			result.SignerUID = strings.TrimSpace(matches[2])
		}
	}

	// Extract signer UID from "Good signature from" pattern
	if result.SignerUID == "" {
		uidPattern := regexp.MustCompile(`Good signature from "([^"]+)"`)
		if matches := uidPattern.FindStringSubmatch(stderrOutput); len(matches) > 1 {
			result.SignerUID = matches[1]
		}
	}

	// Extract decryption key ID
	// Pattern: "gpg: encrypted with ... <KEY_ID>"
	decKeyPattern := regexp.MustCompile(`encrypted with.*?([A-F0-9]{8,})`)
	if matches := decKeyPattern.FindStringSubmatch(stderrOutput); len(matches) > 1 {
		result.DecryptKeyID = matches[1]
	}

	return result
}
