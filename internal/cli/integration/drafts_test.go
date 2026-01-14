//go:build integration

package integration

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"time"
)

// =============================================================================
// DRAFTS COMMAND TESTS
// =============================================================================

func TestCLI_DraftsList(t *testing.T) {
	skipIfMissingCreds(t)

	stdout, stderr, err := runCLI("email", "drafts", "list", testGrantID)
	skipIfProviderNotSupported(t, stderr)

	if err != nil {
		t.Fatalf("drafts list failed: %v\nstderr: %s", err, stderr)
	}

	// Should show drafts or "No drafts found"
	if !strings.Contains(stdout, "Found") && !strings.Contains(stdout, "No drafts found") {
		t.Errorf("Expected drafts list output, got: %s", stdout)
	}

	t.Logf("drafts list output:\n%s", stdout)
}

func TestCLI_DraftsLifecycle(t *testing.T) {
	skipIfMissingCreds(t)

	if os.Getenv("NYLAS_TEST_DELETE") != "true" {
		t.Skip("NYLAS_TEST_DELETE not set to 'true'")
	}

	email := testEmail
	if email == "" {
		email = "test@example.com"
	}

	subject := fmt.Sprintf("CLI Test Draft %d", time.Now().Unix())
	body := "This is a test draft created by integration tests"

	var draftID string

	// Create draft
	t.Run("create", func(t *testing.T) {
		stdout, stderr, err := runCLI("email", "drafts", "create",
			"--to", email,
			"--subject", subject,
			"--body", body,
			testGrantID)

		if err != nil {
			t.Fatalf("drafts create failed: %v\nstderr: %s", err, stderr)
		}

		if !strings.Contains(stdout, "Draft created") {
			t.Errorf("Expected 'Draft created' in output, got: %s", stdout)
		}

		// Extract draft ID from output
		if idx := strings.Index(stdout, "ID:"); idx != -1 {
			draftID = strings.TrimSpace(stdout[idx+3:])
			// Clean up any trailing whitespace or newlines
			if newline := strings.Index(draftID, "\n"); newline != -1 {
				draftID = draftID[:newline]
			}
		}

		t.Logf("drafts create output: %s", stdout)
		t.Logf("Draft ID: %s", draftID)
	})

	if draftID == "" {
		t.Fatal("Failed to get draft ID from create output")
	}

	// Wait for draft to sync
	time.Sleep(2 * time.Second)

	// Show draft
	t.Run("show", func(t *testing.T) {
		stdout, stderr, err := runCLI("email", "drafts", "show", draftID, testGrantID)
		if err != nil {
			t.Fatalf("drafts show failed: %v\nstderr: %s", err, stderr)
		}

		if !strings.Contains(stdout, "Draft:") {
			t.Errorf("Expected 'Draft:' in output, got: %s", stdout)
		}

		t.Logf("drafts show output:\n%s", stdout)
	})

	// List drafts (should include our draft)
	t.Run("list", func(t *testing.T) {
		stdout, stderr, err := runCLI("email", "drafts", "list", testGrantID)
		if err != nil {
			t.Fatalf("drafts list failed: %v\nstderr: %s", err, stderr)
		}

		if !strings.Contains(stdout, "Found") {
			t.Errorf("Expected to find drafts, got: %s", stdout)
		}

		t.Logf("drafts list output:\n%s", stdout)
	})

	// Delete draft
	t.Run("delete", func(t *testing.T) {
		stdout, stderr, err := runCLIWithInput("y\n", "email", "drafts", "delete", draftID, testGrantID)
		if err != nil {
			t.Fatalf("drafts delete failed: %v\nstderr: %s", err, stderr)
		}

		if !strings.Contains(stdout, "deleted") {
			t.Errorf("Expected 'deleted' in output, got: %s", stdout)
		}

		t.Logf("drafts delete output: %s", stdout)
	})
}
