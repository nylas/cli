package gpg

import (
	"context"
	"strings"
	"testing"
)

func TestEncryptData_Validation(t *testing.T) {
	ctx := context.Background()
	svc := NewService()

	// Check if GPG is available
	if err := svc.CheckGPGAvailable(ctx); err != nil {
		t.Skip("GPG not available, skipping test")
	}

	tests := []struct {
		name          string
		recipientKeys []string
		data          []byte
		wantErrMsg    string
	}{
		{
			name:          "no recipients",
			recipientKeys: []string{},
			data:          []byte("test"),
			wantErrMsg:    "at least one recipient key ID is required",
		},
		{
			name:          "invalid key ID format",
			recipientKeys: []string{"INVALID; rm -rf /"},
			data:          []byte("test"),
			wantErrMsg:    "invalid GPG key ID format",
		},
		{
			name:          "command injection attempt",
			recipientKeys: []string{"KEY`whoami`"},
			data:          []byte("test"),
			wantErrMsg:    "invalid GPG key ID format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := svc.EncryptData(ctx, tt.recipientKeys, tt.data)
			if err == nil {
				t.Fatal("EncryptData() expected error, got nil")
			}
			if !strings.Contains(err.Error(), tt.wantErrMsg) {
				t.Errorf("EncryptData() error = %v, want error containing %q", err, tt.wantErrMsg)
			}
		})
	}
}

func TestListPublicKeys_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	ctx := context.Background()
	svc := NewService()

	// Check if GPG is available
	if err := svc.CheckGPGAvailable(ctx); err != nil {
		t.Skip("GPG not available, skipping integration test")
	}

	keys, err := svc.ListPublicKeys(ctx)
	if err != nil {
		t.Fatalf("ListPublicKeys() error = %v", err)
	}

	// It's OK if no keys exist in the test environment
	t.Logf("Found %d public keys", len(keys))

	// Validate key structure if any keys exist
	for _, key := range keys {
		if key.KeyID == "" {
			t.Error("Key missing KeyID")
		}
	}
}

func TestFindPublicKeyByEmail_NotFound(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	ctx := context.Background()
	svc := NewService()

	// Check if GPG is available
	if err := svc.CheckGPGAvailable(ctx); err != nil {
		t.Skip("GPG not available, skipping integration test")
	}

	// Try to find a key for a non-existent email (random UUID domain)
	_, err := svc.FindPublicKeyByEmail(ctx, "nonexistent@e8f9a2b1-c3d4-5e6f-7g8h-9i0j1k2l3m4n.test")
	if err == nil {
		t.Error("FindPublicKeyByEmail() expected error for non-existent email, got nil")
	}

	// Should mention that key was not found
	if !strings.Contains(err.Error(), "no public key found") {
		t.Errorf("Error should mention 'no public key found', got: %v", err)
	}
}

func TestEncryptData_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	ctx := context.Background()
	svc := NewService()

	// Check if GPG is available
	if err := svc.CheckGPGAvailable(ctx); err != nil {
		t.Skip("GPG not available, skipping integration test")
	}

	// List public keys to find one to encrypt to
	keys, err := svc.ListPublicKeys(ctx)
	if err != nil || len(keys) == 0 {
		t.Skip("No public keys available, skipping test")
	}

	// Use the first available key
	testKeyID := keys[0].KeyID

	// Encrypt test data
	testData := []byte("Hello, this is a secret message!")
	result, err := svc.EncryptData(ctx, []string{testKeyID}, testData)
	if err != nil {
		t.Fatalf("EncryptData() error = %v", err)
	}

	if result == nil {
		t.Fatal("EncryptData() returned nil result")
	}

	// Validate ciphertext
	if len(result.Ciphertext) == 0 {
		t.Error("EncryptData() returned empty ciphertext")
	}

	// Check for PGP message markers
	ciphertextStr := string(result.Ciphertext)
	if !strings.Contains(ciphertextStr, "-----BEGIN PGP MESSAGE-----") {
		t.Error("Ciphertext missing PGP BEGIN marker")
	}
	if !strings.Contains(ciphertextStr, "-----END PGP MESSAGE-----") {
		t.Error("Ciphertext missing PGP END marker")
	}

	// Check recipient keys are recorded
	if len(result.RecipientKeys) == 0 {
		t.Error("EncryptResult missing RecipientKeys")
	}
	if result.RecipientKeys[0] != testKeyID {
		t.Errorf("RecipientKeys[0] = %v, want %v", result.RecipientKeys[0], testKeyID)
	}
}

func TestSignAndEncryptData_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	ctx := context.Background()
	svc := NewService()

	// Check if GPG is available
	if err := svc.CheckGPGAvailable(ctx); err != nil {
		t.Skip("GPG not available, skipping integration test")
	}

	// Get signing key
	signerKey, err := svc.GetDefaultSigningKey(ctx)
	if err != nil {
		t.Skip("No default GPG key configured, skipping test")
	}

	// List public keys to find one to encrypt to
	keys, err := svc.ListPublicKeys(ctx)
	if err != nil || len(keys) == 0 {
		t.Skip("No public keys available, skipping test")
	}

	// Use the first available key (could be same as signer for self-test)
	recipientKeyID := keys[0].KeyID

	// Sign and encrypt test data
	testData := []byte("Hello, this is a signed and encrypted message!")
	result, err := svc.SignAndEncryptData(ctx, signerKey.KeyID, []string{recipientKeyID}, testData, "")
	if err != nil {
		t.Fatalf("SignAndEncryptData() error = %v", err)
	}

	if result == nil {
		t.Fatal("SignAndEncryptData() returned nil result")
	}

	// Validate ciphertext
	if len(result.Ciphertext) == 0 {
		t.Error("SignAndEncryptData() returned empty ciphertext")
	}

	// Check for PGP message markers
	ciphertextStr := string(result.Ciphertext)
	if !strings.Contains(ciphertextStr, "-----BEGIN PGP MESSAGE-----") {
		t.Error("Ciphertext missing PGP BEGIN marker")
	}
	if !strings.Contains(ciphertextStr, "-----END PGP MESSAGE-----") {
		t.Error("Ciphertext missing PGP END marker")
	}
}

func TestSignAndEncryptData_Validation(t *testing.T) {
	ctx := context.Background()
	svc := NewService()

	// Check if GPG is available
	if err := svc.CheckGPGAvailable(ctx); err != nil {
		t.Skip("GPG not available, skipping test")
	}

	tests := []struct {
		name          string
		signerKeyID   string
		recipientKeys []string
		data          []byte
		senderEmail   string
		wantErrMsg    string
	}{
		{
			name:          "no signer key",
			signerKeyID:   "",
			recipientKeys: []string{"ABCD1234"},
			data:          []byte("test"),
			senderEmail:   "",
			wantErrMsg:    "signer key ID is required",
		},
		{
			name:          "no recipients",
			signerKeyID:   "ABCD1234",
			recipientKeys: []string{},
			data:          []byte("test"),
			senderEmail:   "",
			wantErrMsg:    "at least one recipient key ID is required",
		},
		{
			name:          "invalid signer key format",
			signerKeyID:   "INVALID; rm -rf /",
			recipientKeys: []string{"ABCD1234"},
			data:          []byte("test"),
			senderEmail:   "",
			wantErrMsg:    "invalid signer GPG key ID format",
		},
		{
			name:          "invalid recipient key format",
			signerKeyID:   "ABCD1234",
			recipientKeys: []string{"INVALID`whoami`"},
			data:          []byte("test"),
			senderEmail:   "",
			wantErrMsg:    "invalid recipient GPG key ID format",
		},
		{
			name:          "invalid sender email",
			signerKeyID:   "ABCD1234",
			recipientKeys: []string{"ABCD5678"},
			data:          []byte("test"),
			senderEmail:   "not-an-email",
			wantErrMsg:    "invalid sender email format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := svc.SignAndEncryptData(ctx, tt.signerKeyID, tt.recipientKeys, tt.data, tt.senderEmail)
			if err == nil {
				t.Fatal("SignAndEncryptData() expected error, got nil")
			}
			if !strings.Contains(err.Error(), tt.wantErrMsg) {
				t.Errorf("SignAndEncryptData() error = %v, want error containing %q", err, tt.wantErrMsg)
			}
		})
	}
}
