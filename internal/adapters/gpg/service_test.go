package gpg

import (
	"context"
	"strings"
	"testing"
)

func TestParseSecretKeys(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    int
		wantErr bool
	}{
		{
			name: "single key with UID",
			input: `sec:u:4096:1:601FEE9B1D60185F:1609459200:::u:::scESC:::+:::23::0:
fpr:::::::::1234567890ABCDEF1234567890ABCDEF12345678:
uid:u::::1609459200::1234567890ABCDEF1234567890ABCDEF12345678::John Doe <john@example.com>::::::::::0:
`,
			want:    1,
			wantErr: false,
		},
		{
			name: "multiple keys",
			input: `sec:u:4096:1:601FEE9B1D60185F:1609459200:::u:::scESC:::+:::23::0:
fpr:::::::::AAAA567890ABCDEF1234567890ABCDEF12345678:
uid:u::::1609459200::AAAA567890ABCDEF1234567890ABCDEF12345678::Alice <alice@example.com>::::::::::0:
sec:u:2048:1:701FEE9B1D60185G:1609459200:::u:::scESC:::+:::23::0:
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
			wantErr: true,
		},
		{
			name:    "invalid format",
			input:   "invalid output",
			want:    0,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseSecretKeys(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseSecretKeys() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && len(got) != tt.want {
				t.Errorf("parseSecretKeys() got %d keys, want %d", len(got), tt.want)
			}
		})
	}
}

func TestExtractHashAlgorithm(t *testing.T) {
	tests := []struct {
		name   string
		stderr string
		want   string
	}{
		{
			name:   "SHA256 in output",
			stderr: "gpg: using RSA key ABC123 digest algorithm SHA256",
			want:   "SHA256",
		},
		{
			name:   "SHA512 in output",
			stderr: "gpg: using RSA key XYZ789 digest algorithm SHA512",
			want:   "SHA512",
		},
		{
			name:   "no hash algorithm",
			stderr: "gpg: signing failed",
			want:   "SHA256", // default
		},
		{
			name:   "empty stderr",
			stderr: "",
			want:   "SHA256", // default
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractHashAlgorithm(tt.stderr)
			if got != tt.want {
				t.Errorf("extractHashAlgorithm() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsValidGPGKeyID(t *testing.T) {
	tests := []struct {
		name  string
		keyID string
		want  bool
	}{
		// Valid hex key IDs
		{"8 char hex", "ABCD1234", true},
		{"16 char hex", "601FEE9B1D60185F", true},
		{"40 char fingerprint", "DBADDF54A44EB10E9714F386601FEE9B1D60185F", true},
		{"lowercase hex", "abcd1234", true},
		{"mixed case hex", "AbCd1234", true},

		// Valid email addresses
		{"simple email", "user@example.com", true},
		{"email with name", "John Doe <john@example.com>", true},
		{"email with subdomain", "user@mail.example.com", true},

		// Invalid inputs
		{"empty string", "", false},
		{"whitespace only", "   ", false},
		{"too short hex", "ABC123", false},               // Less than 8 chars
		{"too long hex", strings.Repeat("A", 41), false}, // More than 40 chars
		{"non-hex chars", "GHIJ1234", false},
		{"command injection attempt", "KEY; rm -rf /", false},
		{"shell metachar", "KEY`whoami`", false},
		{"newline injection", "KEY\nmalicious", false},
		{"special chars", "KEY--malicious", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isValidGPGKeyID(tt.keyID)
			if got != tt.want {
				t.Errorf("isValidGPGKeyID(%q) = %v, want %v", tt.keyID, got, tt.want)
			}
		})
	}
}

func TestCheckGPGAvailable(t *testing.T) {
	ctx := context.Background()
	svc := NewService()

	err := svc.CheckGPGAvailable(ctx)
	if err != nil {
		// GPG might not be installed in test environment
		if !strings.Contains(err.Error(), "GPG not found") {
			t.Errorf("CheckGPGAvailable() unexpected error = %v", err)
		}
		t.Skip("GPG not installed, skipping test")
	}
}

// Note: The following tests require GPG to be installed and configured.
// They will be skipped if GPG is not available.

func TestListSigningKeys_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	ctx := context.Background()
	svc := NewService()

	// Check if GPG is available
	if err := svc.CheckGPGAvailable(ctx); err != nil {
		t.Skip("GPG not available, skipping integration test")
	}

	keys, err := svc.ListSigningKeys(ctx)
	if err != nil {
		// It's OK if no keys exist
		if !strings.Contains(err.Error(), "no GPG secret keys found") {
			t.Errorf("ListSigningKeys() error = %v", err)
		}
		return
	}

	if len(keys) == 0 {
		t.Skip("No GPG keys found, skipping test")
	}

	// Validate key structure
	for _, key := range keys {
		if key.KeyID == "" {
			t.Error("Key missing KeyID")
		}
		if key.Fingerprint == "" {
			t.Error("Key missing Fingerprint")
		}
		if len(key.UIDs) == 0 {
			t.Error("Key missing UIDs")
		}
	}
}

func TestGetDefaultSigningKey_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	ctx := context.Background()
	svc := NewService()

	// Check if GPG is available
	if err := svc.CheckGPGAvailable(ctx); err != nil {
		t.Skip("GPG not available, skipping integration test")
	}

	key, err := svc.GetDefaultSigningKey(ctx)
	if err != nil {
		// It's OK if no default key is configured
		if strings.Contains(err.Error(), "no default GPG key") || strings.Contains(err.Error(), "not found") {
			t.Skip("No default GPG key configured, skipping test")
		}
		t.Errorf("GetDefaultSigningKey() error = %v", err)
		return
	}

	if key == nil {
		t.Error("GetDefaultSigningKey() returned nil key")
		return
	}

	if key.KeyID == "" {
		t.Error("Default key missing KeyID")
	}
	if key.Fingerprint == "" {
		t.Error("Default key missing Fingerprint")
	}
}

func TestSignData_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	ctx := context.Background()
	svc := NewService()

	// Check if GPG is available
	if err := svc.CheckGPGAvailable(ctx); err != nil {
		t.Skip("GPG not available, skipping integration test")
	}

	// Get default key
	key, err := svc.GetDefaultSigningKey(ctx)
	if err != nil {
		t.Skip("No default GPG key configured, skipping test")
	}

	// Sign test data (empty sender email - optional parameter)
	testData := []byte("Hello, GPG world!")
	result, err := svc.SignData(ctx, key.KeyID, testData, "")
	if err != nil {
		t.Fatalf("SignData() error = %v", err)
	}

	if result == nil {
		t.Fatal("SignData() returned nil result")
	}

	// Validate signature
	if len(result.Signature) == 0 {
		t.Error("SignData() returned empty signature")
	}

	// Check for PGP signature markers
	sigStr := string(result.Signature)
	if !strings.Contains(sigStr, "-----BEGIN PGP SIGNATURE-----") {
		t.Error("Signature missing PGP BEGIN marker")
	}
	if !strings.Contains(sigStr, "-----END PGP SIGNATURE-----") {
		t.Error("Signature missing PGP END marker")
	}

	if result.KeyID == "" {
		t.Error("SignResult missing KeyID")
	}
	if result.SignedAt.IsZero() {
		t.Error("SignResult missing SignedAt timestamp")
	}
}

func TestParseVerifyOutput(t *testing.T) {
	tests := []struct {
		name         string
		statusOutput string
		stderrOutput string
		wantValid    bool
		wantKeyID    string
		wantUID      string
		wantTrust    string
	}{
		{
			name: "good signature with ultimate trust",
			statusOutput: `[GNUPG:] GOODSIG 601FEE9B1D60185F John Doe <john@example.com>
[GNUPG:] VALIDSIG DBADDF54A44EB10E9714F386601FEE9B1D60185F 2026-02-01 1738425743 0 4 0 1 10 00 DBADDF54A44EB10E9714F386601FEE9B1D60185F
[GNUPG:] TRUST_ULTIMATE 0 pgp`,
			stderrOutput: `gpg: Signature made Sun 01 Feb 2026 12:02:23 PM EST
gpg:                using RSA key DBADDF54A44EB10E9714F386601FEE9B1D60185F
gpg: Good signature from "John Doe <john@example.com>" [ultimate]`,
			wantValid: true,
			wantKeyID: "601FEE9B1D60185F",
			wantUID:   "John Doe <john@example.com>",
			wantTrust: "ultimate",
		},
		{
			name:         "bad signature",
			statusOutput: `[GNUPG:] BADSIG 601FEE9B1D60185F John Doe <john@example.com>`,
			stderrOutput: `gpg: Signature made Sun 01 Feb 2026 12:02:23 PM EST
gpg: BAD signature from "John Doe <john@example.com>" [ultimate]`,
			wantValid: false,
			wantKeyID: "601FEE9B1D60185F",
			wantUID:   "John Doe <john@example.com>",
			wantTrust: "",
		},
		{
			name:         "good signature from stderr only",
			statusOutput: "",
			stderrOutput: `gpg: Signature made Sun 01 Feb 2026 12:02:23 PM EST
gpg:                using RSA key DBADDF54A44EB10E9714F386601FEE9B1D60185F
gpg: Good signature from "Alice <alice@example.com>" [full]`,
			wantValid: true,
			wantKeyID: "DBADDF54A44EB10E9714F386601FEE9B1D60185F",
			wantUID:   "Alice <alice@example.com>",
			wantTrust: "",
		},
		{
			name: "marginal trust",
			statusOutput: `[GNUPG:] GOODSIG ABC123 Test User <test@test.com>
[GNUPG:] TRUST_MARGINAL 0 pgp`,
			stderrOutput: "",
			wantValid:    true,
			wantKeyID:    "ABC123",
			wantUID:      "Test User <test@test.com>",
			wantTrust:    "marginal",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseVerifyOutput(tt.statusOutput, tt.stderrOutput)

			if result.Valid != tt.wantValid {
				t.Errorf("Valid = %v, want %v", result.Valid, tt.wantValid)
			}
			if result.SignerKeyID != tt.wantKeyID {
				t.Errorf("SignerKeyID = %v, want %v", result.SignerKeyID, tt.wantKeyID)
			}
			if result.SignerUID != tt.wantUID {
				t.Errorf("SignerUID = %v, want %v", result.SignerUID, tt.wantUID)
			}
			if result.TrustLevel != tt.wantTrust {
				t.Errorf("TrustLevel = %v, want %v", result.TrustLevel, tt.wantTrust)
			}
		})
	}
}

func TestExtractKeyIDFromError(t *testing.T) {
	tests := []struct {
		name   string
		stderr string
		want   string
	}{
		{
			name:   "RSA key in output",
			stderr: "gpg: using RSA key DBADDF54A44EB10E9714F386601FEE9B1D60185F\ngpg: Can't check signature: No public key",
			want:   "DBADDF54A44EB10E9714F386601FEE9B1D60185F",
		},
		{
			name:   "EdDSA key in output",
			stderr: "gpg: using EdDSA key ABC123DEF456\ngpg: Can't check signature: No public key",
			want:   "ABC123DEF456",
		},
		{
			name:   "key ID pattern",
			stderr: "gpg: key 601FEE9B1D60185F: public key not found",
			want:   "601FEE9B1D60185F",
		},
		{
			name:   "no key ID",
			stderr: "gpg: some other error",
			want:   "",
		},
		{
			name:   "empty stderr",
			stderr: "",
			want:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractKeyIDFromError(tt.stderr)
			if got != tt.want {
				t.Errorf("extractKeyIDFromError() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestVerifyDetachedSignature_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	ctx := context.Background()
	svc := NewService()

	// Check if GPG is available
	if err := svc.CheckGPGAvailable(ctx); err != nil {
		t.Skip("GPG not available, skipping integration test")
	}

	// Get default key
	key, err := svc.GetDefaultSigningKey(ctx)
	if err != nil {
		t.Skip("No default GPG key configured, skipping test")
	}

	// Sign test data
	testData := []byte("Test message for verification")
	signResult, err := svc.SignData(ctx, key.KeyID, testData, "")
	if err != nil {
		t.Fatalf("SignData() error = %v", err)
	}

	// Verify the signature
	verifyResult, err := svc.VerifyDetachedSignature(ctx, testData, signResult.Signature)
	if err != nil {
		t.Fatalf("VerifyDetachedSignature() error = %v", err)
	}

	if !verifyResult.Valid {
		t.Error("VerifyDetachedSignature() returned invalid for valid signature")
	}

	if verifyResult.SignerKeyID == "" {
		t.Error("VerifyResult missing SignerKeyID")
	}

	if verifyResult.Fingerprint == "" {
		t.Error("VerifyResult missing Fingerprint")
	}
}

func TestVerifyDetachedSignature_BadSignature_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	ctx := context.Background()
	svc := NewService()

	// Check if GPG is available
	if err := svc.CheckGPGAvailable(ctx); err != nil {
		t.Skip("GPG not available, skipping integration test")
	}

	// Get default key
	key, err := svc.GetDefaultSigningKey(ctx)
	if err != nil {
		t.Skip("No default GPG key configured, skipping test")
	}

	// Sign test data
	testData := []byte("Original message")
	signResult, err := svc.SignData(ctx, key.KeyID, testData, "")
	if err != nil {
		t.Fatalf("SignData() error = %v", err)
	}

	// Try to verify with modified data
	modifiedData := []byte("Modified message")
	verifyResult, err := svc.VerifyDetachedSignature(ctx, modifiedData, signResult.Signature)
	if err != nil {
		t.Fatalf("VerifyDetachedSignature() error = %v", err)
	}

	if verifyResult.Valid {
		t.Error("VerifyDetachedSignature() returned valid for tampered data")
	}
}

func TestFindKeyByEmail_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	ctx := context.Background()
	svc := NewService()

	// Check if GPG is available
	if err := svc.CheckGPGAvailable(ctx); err != nil {
		t.Skip("GPG not available, skipping integration test")
	}

	// List keys to find an email to test with
	keys, err := svc.ListSigningKeys(ctx)
	if err != nil || len(keys) == 0 {
		t.Skip("No GPG keys available, skipping test")
	}

	// Find a key with UIDs
	var testEmail string
	for _, key := range keys {
		for _, uid := range key.UIDs {
			// Extract email from UID (format: "Name <email@domain.com>")
			if start := strings.Index(uid, "<"); start != -1 {
				if end := strings.Index(uid, ">"); end > start {
					testEmail = uid[start+1 : end]
					break
				}
			}
		}
		if testEmail != "" {
			break
		}
	}

	if testEmail == "" {
		t.Skip("No email found in GPG keys, skipping test")
	}

	// Find key by email
	foundKey, err := svc.FindKeyByEmail(ctx, testEmail)
	if err != nil {
		t.Fatalf("FindKeyByEmail() error = %v", err)
	}

	if foundKey == nil {
		t.Fatal("FindKeyByEmail() returned nil")
	}

	if foundKey.KeyID == "" {
		t.Error("Found key missing KeyID")
	}
}
