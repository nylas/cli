//go:build integration

package integration

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCLI_SignatureFlags_Help(t *testing.T) {
	if testBinary == "" {
		t.Skip("CLI binary not found - run 'go build -o bin/nylas ./cmd/nylas' first")
	}

	tests := []struct {
		name string
		args []string
	}{
		{
			name: "email send help includes signature flag",
			args: []string{"email", "send", "--help"},
		},
		{
			name: "draft create help includes signature flag",
			args: []string{"email", "drafts", "create", "--help"},
		},
		{
			name: "draft send help includes signature flag",
			args: []string{"email", "drafts", "send", "--help"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stdout, stderr, err := runCLI(tt.args...)
			if err != nil {
				t.Fatalf("help command failed: %v\nstderr: %s", err, stderr)
			}
			if !strings.Contains(stdout, "--signature-id") {
				t.Fatalf("expected --signature-id in help output, got:\n%s", stdout)
			}
		})
	}
}

func TestCLI_SignaturesList(t *testing.T) {
	skipIfMissingCreds(t)

	stdout, stderr, err := runCLI("email", "signatures", "list", testGrantID)
	skipIfProviderNotSupported(t, stderr)

	if err != nil {
		t.Fatalf("signatures list failed: %v\nstderr: %s", err, stderr)
	}

	if !strings.Contains(stdout, "No signatures found") && !strings.Contains(stdout, "UPDATED") && !strings.Contains(stdout, "ID") {
		t.Errorf("Expected signatures list output, got: %s", stdout)
	}
}

func TestCLI_SignaturesLifecycle(t *testing.T) {
	skipIfMissingCreds(t)

	if os.Getenv("NYLAS_TEST_DELETE") != "true" {
		t.Skip("NYLAS_TEST_DELETE not set to 'true'")
	}

	listStdout, listStderr, err := runCLI("email", "signatures", "list", testGrantID)
	skipIfProviderNotSupported(t, listStderr)
	if err != nil {
		t.Fatalf("signatures list pre-check failed: %v\nstderr: %s", err, listStderr)
	}
	if strings.Count(listStdout, "\n") >= 11 && strings.Contains(listStdout, "UPDATED") {
		t.Skip("Grant already has 10 signatures; skipping lifecycle test to avoid hard limit failures")
	}

	bodyPath := filepath.Join(t.TempDir(), "signature.html")
	body := fmt.Sprintf("<p>CLI Signature %d</p>", time.Now().UnixNano())
	if err := os.WriteFile(bodyPath, []byte(body), 0o600); err != nil {
		t.Fatalf("failed to write signature body file: %v", err)
	}

	name := fmt.Sprintf("CLI Signature %d", time.Now().Unix())
	var signatureID string

	t.Run("create", func(t *testing.T) {
		stdout, stderr, err := runCLI("email", "signatures", "create",
			"--name", name,
			"--body-file", bodyPath,
			testGrantID)
		skipIfProviderNotSupported(t, stderr)

		if err != nil {
			t.Fatalf("signatures create failed: %v\nstderr: %s", err, stderr)
		}
		if !strings.Contains(stdout, "Signature created successfully") {
			t.Fatalf("expected create success output, got: %s", stdout)
		}

		signatureID = extractFieldValue(stdout, "ID:")
		if signatureID == "" {
			t.Fatalf("failed to extract signature ID from output: %s", stdout)
		}
	})

	t.Run("show", func(t *testing.T) {
		stdout, stderr, err := runCLI("email", "signatures", "show", signatureID, testGrantID)
		skipIfProviderNotSupported(t, stderr)
		if err != nil {
			t.Fatalf("signatures show failed: %v\nstderr: %s", err, stderr)
		}
		if !strings.Contains(stdout, name) {
			t.Fatalf("expected signature name in output, got: %s", stdout)
		}
	})

	updatedName := name + " Updated"
	updatedBody := "<p>CLI Signature Updated</p>"

	t.Run("update", func(t *testing.T) {
		stdout, stderr, err := runCLI("email", "signatures", "update", signatureID,
			"--name", updatedName,
			"--body", updatedBody,
			testGrantID)
		skipIfProviderNotSupported(t, stderr)

		if err != nil {
			t.Fatalf("signatures update failed: %v\nstderr: %s", err, stderr)
		}
		if !strings.Contains(stdout, "Signature updated successfully") {
			t.Fatalf("expected update success output, got: %s", stdout)
		}
		if !strings.Contains(stdout, updatedName) {
			t.Fatalf("expected updated name in output, got: %s", stdout)
		}
	})

	t.Run("delete", func(t *testing.T) {
		stdout, stderr, err := runCLI("email", "signatures", "delete", signatureID, "--yes", testGrantID)
		skipIfProviderNotSupported(t, stderr)
		if err != nil {
			t.Fatalf("signatures delete failed: %v\nstderr: %s", err, stderr)
		}
		if !strings.Contains(stdout, "Signature deleted successfully") {
			t.Fatalf("expected delete success output, got: %s", stdout)
		}
	})
}

func TestCLI_DraftsSendRejectsDuplicateStoredSignature(t *testing.T) {
	skipIfMissingCreds(t)

	if os.Getenv("NYLAS_TEST_DELETE") != "true" {
		t.Skip("NYLAS_TEST_DELETE not set to 'true'")
	}

	listStdout, listStderr, err := runCLI("email", "signatures", "list", testGrantID)
	skipIfProviderNotSupported(t, listStderr)
	if err != nil {
		t.Fatalf("signatures list pre-check failed: %v\nstderr: %s", err, listStderr)
	}
	if strings.Count(listStdout, "\n") >= 11 && strings.Contains(listStdout, "UPDATED") {
		t.Skip("Grant already has 10 signatures; skipping duplicate signature test to avoid hard limit failures")
	}

	recipient := testEmail
	if recipient == "" {
		recipient = "test@example.com"
	}

	name := fmt.Sprintf("CLI Duplicate Signature %d", time.Now().Unix())
	body := fmt.Sprintf("<p>CLI duplicate signature marker %d</p>", time.Now().UnixNano())

	createSigStdout, createSigStderr, err := runCLI(
		"email", "signatures", "create",
		"--name", name,
		"--body", body,
		testGrantID,
	)
	skipIfProviderNotSupported(t, createSigStderr)
	if err != nil {
		t.Fatalf("signatures create failed: %v\nstderr: %s", err, createSigStderr)
	}

	signatureID := extractFieldValue(createSigStdout, "ID:")
	if signatureID == "" {
		t.Fatalf("failed to extract signature ID from output: %s", createSigStdout)
	}
	defer func() {
		stdout, stderr, cleanupErr := runCLI("email", "signatures", "delete", signatureID, "--yes", testGrantID)
		skipIfProviderNotSupported(t, stderr)
		if cleanupErr != nil {
			t.Fatalf("signatures delete failed: %v\nstderr: %s\nstdout: %s", cleanupErr, stderr, stdout)
		}
	}()

	createDraftStdout, createDraftStderr, err := runCLI(
		"email", "drafts", "create",
		"--to", recipient,
		"--subject", fmt.Sprintf("Duplicate Signature Draft %d", time.Now().Unix()),
		"--body", "This draft should keep a single stored signature",
		"--signature-id", signatureID,
		testGrantID,
	)
	if err != nil {
		t.Fatalf("draft create failed: %v\nstderr: %s", err, createDraftStderr)
	}

	draftID := extractInlineID(createDraftStdout)
	if draftID == "" {
		t.Fatalf("failed to extract draft ID from output: %s", createDraftStdout)
	}
	defer func() {
		stdout, stderr, cleanupErr := runCLI("email", "drafts", "delete", draftID, "--force", testGrantID)
		if cleanupErr != nil {
			t.Fatalf("draft delete failed: %v\nstderr: %s\nstdout: %s", cleanupErr, stderr, stdout)
		}
	}()

	_, stderr, err := runCLI(
		"email", "drafts", "send", draftID,
		"--signature-id", signatureID,
		"--force",
		testGrantID,
	)
	require.Error(t, err)
	assert.Contains(t, stderr, "already contains a stored signature")
}

func extractFieldValue(output, prefix string) string {
	for _, line := range strings.Split(output, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, prefix) {
			return strings.TrimSpace(strings.TrimPrefix(trimmed, prefix))
		}
	}
	return ""
}

func extractInlineID(output string) string {
	if idx := strings.Index(output, "ID:"); idx != -1 {
		value := strings.TrimSpace(output[idx+3:])
		if newline := strings.Index(value, "\n"); newline != -1 {
			value = value[:newline]
		}
		return strings.TrimSpace(value)
	}
	return ""
}
