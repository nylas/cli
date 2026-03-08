package gpg

import (
	"context"
	"strings"
	"testing"
)

func TestDecryptData_Validation(t *testing.T) {
	ctx := context.Background()
	svc := NewService()

	// Check if GPG is available
	if err := svc.CheckGPGAvailable(ctx); err != nil {
		t.Skip("GPG not available, skipping test")
	}

	tests := []struct {
		name       string
		ciphertext []byte
		wantErrMsg string
	}{
		{
			name:       "empty ciphertext",
			ciphertext: []byte{},
			wantErrMsg: "ciphertext is empty",
		},
		{
			name:       "invalid PGP data",
			ciphertext: []byte("not valid PGP data"),
			wantErrMsg: "decryption failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := svc.DecryptData(ctx, tt.ciphertext)
			if err == nil {
				t.Fatal("DecryptData() expected error, got nil")
			}
			if !strings.Contains(err.Error(), tt.wantErrMsg) {
				t.Errorf("DecryptData() error = %v, want error containing %q", err, tt.wantErrMsg)
			}
		})
	}
}

func TestParseDecryptOutput(t *testing.T) {
	tests := []struct {
		name         string
		stderrOutput string
		wantSigned   bool
		wantSigOK    bool
		wantKeyID    string
		wantUID      string
	}{
		{
			name: "encrypted only, no signature",
			stderrOutput: `gpg: encrypted with 4096-bit RSA key, ID ABCD1234
gpg: decryption okay`,
			wantSigned: false,
			wantSigOK:  false,
		},
		{
			name: "signed and encrypted with good signature",
			stderrOutput: `gpg: encrypted with 4096-bit RSA key, ID ABCD1234
gpg: Signature made Mon 01 Jan 2024 12:00:00 PM EST
gpg:                using RSA key DBADDF54A44EB10E9714F386601FEE9B1D60185F
gpg: Good signature from "John Doe <john@example.com>" [ultimate]`,
			wantSigned: true,
			wantSigOK:  true,
			wantKeyID:  "DBADDF54A44EB10E9714F386601FEE9B1D60185F",
			wantUID:    "John Doe <john@example.com>",
		},
		{
			name: "signed and encrypted with bad signature",
			stderrOutput: `gpg: encrypted with 4096-bit RSA key, ID ABCD1234
gpg: Signature made Mon 01 Jan 2024 12:00:00 PM EST
gpg:                using RSA key DBADDF54A44EB10E9714F386601FEE9B1D60185F
gpg: BAD signature from "John Doe <john@example.com>" [ultimate]`,
			wantSigned: true,
			wantSigOK:  false,
			wantKeyID:  "DBADDF54A44EB10E9714F386601FEE9B1D60185F",
		},
		{
			name:         "GNUPG status format",
			stderrOutput: `[GNUPG:] GOODSIG 601FEE9B1D60185F John Doe <john@example.com>`,
			wantSigned:   true,
			wantSigOK:    true,
			wantKeyID:    "601FEE9B1D60185F",
			wantUID:      "John Doe <john@example.com>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseDecryptOutput(tt.stderrOutput)

			if result.WasSigned != tt.wantSigned {
				t.Errorf("WasSigned = %v, want %v", result.WasSigned, tt.wantSigned)
			}
			if result.SignatureOK != tt.wantSigOK {
				t.Errorf("SignatureOK = %v, want %v", result.SignatureOK, tt.wantSigOK)
			}
			if tt.wantKeyID != "" && result.SignerKeyID != tt.wantKeyID {
				t.Errorf("SignerKeyID = %v, want %v", result.SignerKeyID, tt.wantKeyID)
			}
			if tt.wantUID != "" && result.SignerUID != tt.wantUID {
				t.Errorf("SignerUID = %v, want %v", result.SignerUID, tt.wantUID)
			}
		})
	}
}

func TestDecryptData_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	ctx := context.Background()
	svc := NewService()

	// Check if GPG is available
	if err := svc.CheckGPGAvailable(ctx); err != nil {
		t.Skip("GPG not available, skipping integration test")
	}

	// Get a key to encrypt to (will use for self-decryption test)
	keys, err := svc.ListPublicKeys(ctx)
	if err != nil || len(keys) == 0 {
		t.Skip("No public keys available, skipping test")
	}

	// Find a key we have the private key for (secret key)
	signingKeys, err := svc.ListSigningKeys(ctx)
	if err != nil || len(signingKeys) == 0 {
		t.Skip("No secret keys available, skipping test")
	}

	// Use first signing key (we definitely have the private key for this)
	recipientKeyID := signingKeys[0].KeyID

	// Encrypt test data
	testData := []byte("This is a secret message for decryption test!")
	encResult, err := svc.EncryptData(ctx, []string{recipientKeyID}, testData)
	if err != nil {
		t.Fatalf("EncryptData() error = %v", err)
	}

	// Decrypt the data
	decResult, err := svc.DecryptData(ctx, encResult.Ciphertext)
	if err != nil {
		if isNonInteractiveGPGError(err.Error()) {
			t.Skipf("GPG requires interactive passphrase entry in this environment: %v", err)
		}
		t.Fatalf("DecryptData() error = %v", err)
	}

	// Verify decrypted content matches original
	if string(decResult.Plaintext) != string(testData) {
		t.Errorf("Decrypted content mismatch:\ngot:  %q\nwant: %q", string(decResult.Plaintext), string(testData))
	}

	// Should not be signed (encrypt-only)
	if decResult.WasSigned {
		t.Error("Expected WasSigned=false for encrypt-only message")
	}
}

func TestDecryptSignedAndEncryptedData_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	ctx := context.Background()
	svc := NewService()

	// Check if GPG is available
	if err := svc.CheckGPGAvailable(ctx); err != nil {
		t.Skip("GPG not available, skipping integration test")
	}

	// Get default signing key
	signerKey, err := svc.GetDefaultSigningKey(ctx)
	if err != nil {
		t.Skip("No default GPG key configured, skipping test")
	}

	// Use same key for recipient (self-test)
	recipientKeyID := signerKey.KeyID

	// Sign and encrypt test data
	testData := []byte("This is a signed and encrypted message!")
	encResult, err := svc.SignAndEncryptData(ctx, signerKey.KeyID, []string{recipientKeyID}, testData, "")
	if err != nil {
		t.Fatalf("SignAndEncryptData() error = %v", err)
	}

	// Decrypt the data
	decResult, err := svc.DecryptData(ctx, encResult.Ciphertext)
	if err != nil {
		if isNonInteractiveGPGError(err.Error()) {
			t.Skipf("GPG requires interactive passphrase entry in this environment: %v", err)
		}
		t.Fatalf("DecryptData() error = %v", err)
	}

	// Verify decrypted content matches original
	if string(decResult.Plaintext) != string(testData) {
		t.Errorf("Decrypted content mismatch:\ngot:  %q\nwant: %q", string(decResult.Plaintext), string(testData))
	}

	// Should be signed
	if !decResult.WasSigned {
		t.Error("Expected WasSigned=true for signed+encrypted message")
	}

	// Signature should be valid
	if !decResult.SignatureOK {
		t.Error("Expected SignatureOK=true for valid signature")
	}
}

func isNonInteractiveGPGError(errMsg string) bool {
	nonInteractiveErrors := []string{
		"cannot open '/dev/tty'",
		"no pinentry",
		"inappropriate ioctl for device",
		"need_passphrase",
		"inquire_maxlen",
		"operation cancelled",
		"problem with the agent",
	}

	lowerErr := strings.ToLower(errMsg)
	for _, marker := range nonInteractiveErrors {
		if strings.Contains(lowerErr, marker) {
			return true
		}
	}

	return false
}
