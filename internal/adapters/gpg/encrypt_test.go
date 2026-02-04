package gpg

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestParsePublicKeys(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    int
		wantErr bool
	}{
		{
			name: "single public key with UID",
			input: `pub:u:4096:1:601FEE9B1D60185F:1609459200:::u:::scESC:::+:::23::0:
fpr:::::::::1234567890ABCDEF1234567890ABCDEF12345678:
uid:u::::1609459200::1234567890ABCDEF1234567890ABCDEF12345678::John Doe <john@example.com>::::::::::0:
`,
			want:    1,
			wantErr: false,
		},
		{
			name: "multiple public keys",
			input: `pub:u:4096:1:601FEE9B1D60185F:1609459200:::u:::scESC:::+:::23::0:
fpr:::::::::AAAA567890ABCDEF1234567890ABCDEF12345678:
uid:u::::1609459200::AAAA567890ABCDEF1234567890ABCDEF12345678::Alice <alice@example.com>::::::::::0:
pub:u:2048:1:701FEE9B1D60185G:1609459200:::u:::scESC:::+:::23::0:
fpr:::::::::BBBB567890ABCDEF1234567890ABCDEF12345678:
uid:u::::1609459200::BBBB567890ABCDEF1234567890ABCDEF12345678::Bob <bob@example.com>::::::::::0:
`,
			want:    2,
			wantErr: false,
		},
		{
			name:    "no keys",
			input:   "",
			want:    0,
			wantErr: false, // Empty is OK for public keys (unlike secret keys)
		},
		{
			name:    "invalid format",
			input:   "invalid output",
			want:    0,
			wantErr: false, // Still returns empty slice, not error
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parsePublicKeys(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parsePublicKeys() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && len(got) != tt.want {
				t.Errorf("parsePublicKeys() got %d keys, want %d", len(got), tt.want)
			}
		})
	}
}

func TestKeyMatchesEmail(t *testing.T) {
	tests := []struct {
		name  string
		key   KeyInfo
		email string
		want  bool
	}{
		{
			name: "email in angle brackets",
			key: KeyInfo{
				UIDs: []string{"John Doe <john@example.com>"},
			},
			email: "john@example.com",
			want:  true,
		},
		{
			name: "case insensitive match",
			key: KeyInfo{
				UIDs: []string{"John Doe <John@EXAMPLE.COM>"},
			},
			email: "john@example.com",
			want:  true,
		},
		{
			name: "bare email match",
			key: KeyInfo{
				UIDs: []string{"user@example.com"},
			},
			email: "user@example.com",
			want:  true,
		},
		{
			name: "multiple UIDs with match",
			key: KeyInfo{
				UIDs: []string{
					"Work <work@company.com>",
					"Personal <john@example.com>",
				},
			},
			email: "john@example.com",
			want:  true,
		},
		{
			name: "no match",
			key: KeyInfo{
				UIDs: []string{"Other <other@example.com>"},
			},
			email: "john@example.com",
			want:  false,
		},
		{
			name: "partial match not accepted",
			key: KeyInfo{
				UIDs: []string{"John <john@example.com.malicious>"},
			},
			email: "john@example.com",
			want:  false,
		},
		{
			name: "empty UIDs",
			key: KeyInfo{
				UIDs: []string{},
			},
			email: "john@example.com",
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := keyMatchesEmail(&tt.key, tt.email)
			if got != tt.want {
				t.Errorf("keyMatchesEmail() = %v, want %v", got, tt.want)
			}
		})
	}
}

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

func TestKeyInfo_ExpiredKey(t *testing.T) {
	// Test that expired keys are properly detected
	pastTime := time.Now().Add(-24 * time.Hour)
	futureTime := time.Now().Add(24 * time.Hour)

	tests := []struct {
		name      string
		key       KeyInfo
		isExpired bool
	}{
		{
			name: "expired key",
			key: KeyInfo{
				KeyID:   "EXPIRED1234",
				Expires: &pastTime,
			},
			isExpired: true,
		},
		{
			name: "valid key",
			key: KeyInfo{
				KeyID:   "VALID1234",
				Expires: &futureTime,
			},
			isExpired: false,
		},
		{
			name: "no expiration",
			key: KeyInfo{
				KeyID:   "NOEXPIRE1234",
				Expires: nil,
			},
			isExpired: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isExpired := tt.key.Expires != nil && tt.key.Expires.Before(time.Now())
			if isExpired != tt.isExpired {
				t.Errorf("Key expired = %v, want %v", isExpired, tt.isExpired)
			}
		})
	}
}

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
