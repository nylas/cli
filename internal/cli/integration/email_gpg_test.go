//go:build integration
// +build integration

package integration

import (
	"strings"
	"testing"
	"time"

	"github.com/nylas/cli/internal/cli/common"
)

// =============================================================================
// GPG EMAIL SIGNING AND VERIFICATION TESTS
// =============================================================================

func TestCLI_EmailSend_GPGSigned(t *testing.T) {
	skipIfMissingCreds(t)

	email := getTestEmail()
	if email == "" {
		t.Skip("NYLAS_TEST_EMAIL not set, skipping GPG send test")
	}

	acquireRateLimit(t)

	// Send a GPG-signed email to self
	stdout, stderr, err := runCLI("email", "send",
		"--to", email,
		"--subject", "[CLI Test] GPG Signed Email",
		"--body", "This is a GPG-signed test email from CLI integration tests.",
		"--sign",
		"--yes",
		testGrantID)

	if err != nil {
		// Skip if GPG is not available
		if strings.Contains(stderr, "GPG not found") || strings.Contains(stderr, "no GPG key") {
			t.Skip("GPG not available or no keys configured, skipping test")
		}
		t.Fatalf("email send --sign failed: %v\nstderr: %s", err, stderr)
	}

	// Should show signed email confirmation
	if !strings.Contains(stdout, "Signed email sent") && !strings.Contains(stdout, "Message ID") {
		t.Errorf("Expected signed email confirmation, got: %s", stdout)
	}

	t.Logf("GPG signed email sent:\n%s", stdout)
}

func TestCLI_EmailSend_GPGSignedAndVerify(t *testing.T) {
	skipIfMissingCreds(t)

	email := getTestEmail()
	if email == "" {
		t.Skip("NYLAS_TEST_EMAIL not set, skipping GPG verify test")
	}

	acquireRateLimit(t)

	// Step 1: Send a GPG-signed email to self
	sendStdout, sendStderr, err := runCLI("email", "send",
		"--to", email,
		"--subject", "[CLI Test] GPG Verify Test "+time.Now().Format("15:04:05"),
		"--body", "This email will be verified after sending.",
		"--sign",
		"--yes",
		testGrantID)

	if err != nil {
		if strings.Contains(sendStderr, "GPG not found") || strings.Contains(sendStderr, "no GPG key") {
			t.Skip("GPG not available or no keys configured, skipping test")
		}
		t.Fatalf("email send --sign failed: %v\nstderr: %s", err, sendStderr)
	}

	// Extract message ID from output
	messageID := extractMessageID(sendStdout)
	if messageID == "" {
		t.Fatalf("Could not extract message ID from output: %s", sendStdout)
	}

	t.Logf("Sent signed email with ID: %s", messageID)

	// Wait for email to be delivered
	time.Sleep(3 * time.Second)
	acquireRateLimit(t)

	// Step 2: Verify the signature
	verifyStdout, verifyStderr, err := runCLI("email", "read", messageID, "--verify", testGrantID)

	if err != nil {
		t.Fatalf("email read --verify failed: %v\nstderr: %s", err, verifyStderr)
	}

	// Should show "Good signature"
	if !strings.Contains(verifyStdout, "Good signature") {
		t.Errorf("Expected 'Good signature' in output, got: %s", verifyStdout)
	}

	// Should show signer info
	if !strings.Contains(verifyStdout, "Signer:") {
		t.Errorf("Expected 'Signer:' in verification output, got: %s", verifyStdout)
	}

	if !strings.Contains(verifyStdout, "Key ID:") {
		t.Errorf("Expected 'Key ID:' in verification output, got: %s", verifyStdout)
	}

	t.Logf("GPG verification output:\n%s", verifyStdout)

	// Cleanup: delete the test email
	acquireRateLimit(t)
	_, _, _ = runCLI("email", "delete", messageID, "--yes", testGrantID)
}

func TestCLI_EmailRead_VerifyBehavior(t *testing.T) {
	skipIfMissingCreds(t)

	email := getTestEmail()
	if email == "" {
		t.Skip("NYLAS_TEST_EMAIL not set, skipping verify behavior test")
	}

	acquireRateLimit(t)

	// Send an email without explicit --sign flag
	sendStdout, sendStderr, err := runCLI("email", "send",
		"--to", email,
		"--subject", "[CLI Test] Verify Behavior "+time.Now().Format("15:04:05"),
		"--body", "Testing verification behavior.",
		"--yes",
		testGrantID)

	if err != nil {
		t.Fatalf("email send failed: %v\nstderr: %s", err, sendStderr)
	}

	messageID := extractMessageID(sendStdout)
	if messageID == "" {
		t.Fatalf("Could not extract message ID from output: %s", sendStdout)
	}

	// Wait for email to be delivered
	time.Sleep(3 * time.Second)
	acquireRateLimit(t)

	// Try to verify the email
	verifyStdout, verifyStderr, err := runCLI("email", "read", messageID, "--verify", testGrantID)

	// Log the verification result regardless of outcome
	combined := verifyStdout + verifyStderr
	if err != nil {
		// Verification failed - expected for truly unsigned emails
		if strings.Contains(combined, "not PGP/MIME signed") || strings.Contains(combined, "not signed") {
			t.Log("Unsigned email correctly identified as not signed")
		} else {
			t.Logf("Verification error: %s", verifyStderr)
		}
	} else {
		// Verification succeeded
		if strings.Contains(combined, "Good signature") {
			t.Log("Email has valid signature (may be auto-signed by mail server)")
		} else if strings.Contains(combined, "BAD signature") {
			t.Log("Email has invalid/tampered signature")
		} else {
			t.Logf("Verify output: %s", verifyStdout)
		}
	}

	// Cleanup
	acquireRateLimit(t)
	_, _, _ = runCLI("email", "delete", messageID, "--yes", testGrantID)
}

// =============================================================================
// RAW MIME RETRIEVAL TESTS
// =============================================================================

func TestCLI_EmailRead_RawMIME(t *testing.T) {
	skipIfMissingCreds(t)

	email := getTestEmail()
	if email == "" {
		t.Skip("NYLAS_TEST_EMAIL not set, skipping raw MIME test")
	}

	acquireRateLimit(t)

	// Send a test email first
	sendStdout, sendStderr, err := runCLI("email", "send",
		"--to", email,
		"--subject", "[CLI Test] MIME Test "+time.Now().Format("15:04:05"),
		"--body", "Testing raw MIME retrieval.",
		"--yes",
		testGrantID)

	if err != nil {
		t.Fatalf("email send failed: %v\nstderr: %s", err, sendStderr)
	}

	messageID := extractMessageID(sendStdout)
	if messageID == "" {
		t.Fatalf("Could not extract message ID from output: %s", sendStdout)
	}

	// Wait for email to be delivered
	time.Sleep(3 * time.Second)
	acquireRateLimit(t)

	// Read with --mime flag
	mimeStdout, mimeStderr, err := runCLI("email", "read", messageID, "--mime", testGrantID)

	if err != nil {
		t.Fatalf("email read --mime failed: %v\nstderr: %s", err, mimeStderr)
	}

	// Should contain MIME headers
	if !strings.Contains(mimeStdout, "Content-Type:") {
		t.Errorf("Expected 'Content-Type:' in MIME output, got: %s", mimeStdout)
	}

	if !strings.Contains(mimeStdout, "MIME-Version:") && !strings.Contains(mimeStdout, "RFC822") {
		t.Errorf("Expected MIME headers in output, got: %s", mimeStdout)
	}

	t.Logf("Raw MIME output (first 500 chars):\n%s", common.Truncate(mimeStdout, 500))

	// Cleanup
	acquireRateLimit(t)
	_, _, _ = runCLI("email", "delete", messageID, "--yes", testGrantID)
}

func TestCLI_EmailRead_SignedMIME(t *testing.T) {
	skipIfMissingCreds(t)

	email := getTestEmail()
	if email == "" {
		t.Skip("NYLAS_TEST_EMAIL not set, skipping signed MIME test")
	}

	acquireRateLimit(t)

	// Send a GPG-signed email
	sendStdout, sendStderr, err := runCLI("email", "send",
		"--to", email,
		"--subject", "[CLI Test] Signed MIME Test "+time.Now().Format("15:04:05"),
		"--body", "Testing signed email MIME structure.",
		"--sign",
		"--yes",
		testGrantID)

	if err != nil {
		if strings.Contains(sendStderr, "GPG not found") || strings.Contains(sendStderr, "no GPG key") {
			t.Skip("GPG not available or no keys configured, skipping test")
		}
		t.Fatalf("email send --sign failed: %v\nstderr: %s", err, sendStderr)
	}

	messageID := extractMessageID(sendStdout)
	if messageID == "" {
		t.Fatalf("Could not extract message ID from output: %s", sendStdout)
	}

	// Wait for email to be delivered
	time.Sleep(3 * time.Second)
	acquireRateLimit(t)

	// Read with --mime flag
	mimeStdout, mimeStderr, err := runCLI("email", "read", messageID, "--mime", testGrantID)

	if err != nil {
		t.Fatalf("email read --mime failed: %v\nstderr: %s", err, mimeStderr)
	}

	// Should be multipart/signed
	if !strings.Contains(mimeStdout, "multipart/signed") {
		t.Errorf("Expected 'multipart/signed' in MIME output, got: %s", common.Truncate(mimeStdout, 500))
	}

	// Should contain PGP signature
	if !strings.Contains(mimeStdout, "application/pgp-signature") {
		t.Errorf("Expected 'application/pgp-signature' in MIME output, got: %s", common.Truncate(mimeStdout, 500))
	}

	// Should contain BEGIN PGP SIGNATURE
	if !strings.Contains(mimeStdout, "BEGIN PGP SIGNATURE") {
		t.Errorf("Expected 'BEGIN PGP SIGNATURE' in MIME output, got: %s", common.Truncate(mimeStdout, 500))
	}

	t.Logf("Signed MIME structure verified")

	// Cleanup
	acquireRateLimit(t)
	_, _, _ = runCLI("email", "delete", messageID, "--yes", testGrantID)
}

// =============================================================================
// GPG KEY LISTING TEST
// =============================================================================

func TestCLI_EmailSend_ListGPGKeys(t *testing.T) {
	skipIfMissingCreds(t)

	stdout, stderr, err := runCLI("email", "send", "--list-gpg-keys")

	if err != nil {
		if strings.Contains(stderr, "GPG not found") {
			t.Skip("GPG not installed, skipping test")
		}
		// No keys is also acceptable
		if strings.Contains(stderr, "No GPG signing keys") || strings.Contains(stdout, "No GPG signing keys") {
			t.Log("No GPG keys found (this is OK for test environments)")
			return
		}
		t.Fatalf("email send --gpg-list-keys failed: %v\nstderr: %s", err, stderr)
	}

	// Should show key info or "no keys" message
	if !strings.Contains(stdout, "Key ID:") && !strings.Contains(stdout, "No GPG") && !strings.Contains(stdout, "signing keys") {
		t.Errorf("Expected GPG key listing output, got: %s", stdout)
	}

	t.Logf("GPG key listing:\n%s", stdout)
}

// =============================================================================
// HELPER FUNCTIONS
// =============================================================================

// extractMessageID extracts the message ID from send command output
func extractMessageID(output string) string {
	// Look for "Message ID: <id>" pattern
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.Contains(line, "Message ID:") {
			parts := strings.Split(line, "Message ID:")
			if len(parts) > 1 {
				return strings.TrimSpace(parts[1])
			}
		}
		// Also try "ID: <id>" pattern
		if strings.Contains(line, "ID:") && !strings.Contains(line, "Key ID:") {
			parts := strings.Split(line, "ID:")
			if len(parts) > 1 {
				id := strings.TrimSpace(parts[1])
				// Filter out short IDs that might be key IDs
				if len(id) > 20 {
					return id
				}
			}
		}
	}
	return ""
}

