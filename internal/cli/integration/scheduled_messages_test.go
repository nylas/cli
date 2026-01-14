//go:build integration

package integration

import (
	"os"
	"strings"
	"testing"
	"time"
)

// =============================================================================
// EMAIL SCHEDULED MESSAGES COMMAND TESTS
// =============================================================================

func TestCLI_EmailScheduledHelp(t *testing.T) {
	if testBinary == "" {
		t.Skip("CLI binary not found")
	}

	stdout, stderr, err := runCLI("email", "scheduled", "--help")

	if err != nil {
		t.Fatalf("email scheduled --help failed: %v\nstderr: %s", err, stderr)
	}

	// Should show scheduled subcommands
	if !strings.Contains(stdout, "list") {
		t.Errorf("Expected 'list' subcommand in help, got: %s", stdout)
	}
	if !strings.Contains(stdout, "cancel") {
		t.Errorf("Expected 'cancel' subcommand in help, got: %s", stdout)
	}

	t.Logf("email scheduled --help output:\n%s", stdout)
}

func TestCLI_EmailScheduledList(t *testing.T) {
	skipIfMissingCreds(t)

	stdout, stderr, err := runCLI("email", "scheduled", "list", testGrantID)
	skipIfProviderNotSupported(t, stderr)

	if err != nil {
		// Scheduled list may fail if provider doesn't support it
		if strings.Contains(stderr, "not supported") || strings.Contains(stderr, "not available") {
			t.Skip("Scheduled messages not supported by provider")
		}
		t.Fatalf("email scheduled list failed: %v\nstderr: %s", err, stderr)
	}

	// Should show scheduled messages or "No scheduled messages"
	if !strings.Contains(stdout, "scheduled") && !strings.Contains(stdout, "No scheduled") &&
		!strings.Contains(stdout, "Found") {
		t.Errorf("Expected scheduled messages list output, got: %s", stdout)
	}

	t.Logf("email scheduled list output:\n%s", stdout)
}

func TestCLI_EmailScheduledListHelp(t *testing.T) {
	if testBinary == "" {
		t.Skip("CLI binary not found")
	}

	stdout, stderr, err := runCLI("email", "scheduled", "list", "--help")

	if err != nil {
		t.Fatalf("email scheduled list --help failed: %v\nstderr: %s", err, stderr)
	}

	// Should show usage information
	if !strings.Contains(stdout, "list") && !strings.Contains(stdout, "List") {
		t.Errorf("Expected 'list' in help, got: %s", stdout)
	}

	// Note: The scheduled list command may not have a --limit flag
	t.Logf("email scheduled list --help output:\n%s", stdout)
}

func TestCLI_EmailScheduledListJSON(t *testing.T) {
	skipIfMissingCreds(t)

	stdout, stderr, err := runCLI("email", "scheduled", "list", testGrantID, "--json")
	skipIfProviderNotSupported(t, stderr)

	if err != nil {
		if strings.Contains(stderr, "not supported") || strings.Contains(stderr, "not available") {
			t.Skip("Scheduled messages not supported by provider")
		}
		t.Fatalf("email scheduled list --json failed: %v\nstderr: %s", err, stderr)
	}

	// Note: The --json flag is a global flag but may not be fully implemented
	// for the scheduled list command. Just verify the command doesn't fail.
	if strings.Contains(stdout, "No scheduled") || strings.Contains(stdout, "scheduled") {
		t.Logf("email scheduled list --json output (formatted):\n%s", stdout)
	} else {
		// Check if it's JSON
		trimmed := strings.TrimSpace(stdout)
		if len(trimmed) > 0 && (trimmed[0] == '[' || trimmed[0] == '{') {
			t.Logf("email scheduled list --json output (JSON):\n%s", stdout)
		} else {
			t.Logf("email scheduled list --json output:\n%s", stdout)
		}
	}
}

func TestCLI_EmailScheduledListMultipleTimes(t *testing.T) {
	skipIfMissingCreds(t)

	// Test that the list command can be called multiple times without issues
	for i := 0; i < 2; i++ {
		stdout, stderr, err := runCLI("email", "scheduled", "list", testGrantID)
		skipIfProviderNotSupported(t, stderr)

		if err != nil {
			if strings.Contains(stderr, "not supported") || strings.Contains(stderr, "not available") {
				t.Skip("Scheduled messages not supported by provider")
			}
			t.Fatalf("email scheduled list (attempt %d) failed: %v\nstderr: %s", i+1, err, stderr)
		}

		t.Logf("email scheduled list (attempt %d) output:\n%s", i+1, stdout)
	}
}

// =============================================================================
// EMAIL SCHEDULED DELETE/CANCEL COMMAND TESTS
// =============================================================================

func TestCLI_EmailScheduledCancelHelpViaDelete(t *testing.T) {
	if testBinary == "" {
		t.Skip("CLI binary not found")
	}

	// Note: The command is "cancel" not "delete", so this will show the main help
	stdout, stderr, err := runCLI("email", "scheduled", "delete", "--help")

	// This should either fail or show the scheduled help (not a delete subcommand)
	if err != nil {
		// Expected - delete subcommand doesn't exist
		t.Logf("email scheduled delete --help correctly fails (use 'cancel' instead): %s", stderr)
	} else {
		// Shows main scheduled help
		t.Logf("email scheduled delete --help output (shows main help):\n%s", stdout)
	}
}

func TestCLI_EmailScheduledCancelHelp(t *testing.T) {
	if testBinary == "" {
		t.Skip("CLI binary not found")
	}

	stdout, stderr, err := runCLI("email", "scheduled", "cancel", "--help")

	if err != nil {
		t.Fatalf("email scheduled cancel --help failed: %v\nstderr: %s", err, stderr)
	}

	// Should show usage information (cancel may be an alias for delete)
	if !strings.Contains(stdout, "schedule-id") && !strings.Contains(stdout, "cancel") {
		t.Errorf("Expected 'schedule-id' or 'cancel' in help, got: %s", stdout)
	}

	t.Logf("email scheduled cancel --help output:\n%s", stdout)
}

func TestCLI_EmailScheduledCancelInvalidID(t *testing.T) {
	skipIfMissingCreds(t)

	_, stderr, err := runCLI("email", "scheduled", "cancel", "invalid-schedule-id", testGrantID, "--force")

	if err == nil {
		t.Error("Expected error for invalid schedule ID, but command succeeded")
	}

	t.Logf("email scheduled cancel invalid ID error: %s", stderr)
}

// =============================================================================
// EMAIL SCHEDULED LIFECYCLE TESTS
// =============================================================================

func TestCLI_EmailScheduledLifecycle(t *testing.T) {
	if os.Getenv("NYLAS_TEST_SEND_EMAIL") != "true" {
		t.Skip("Skipping send test - set NYLAS_TEST_SEND_EMAIL=true to enable")
	}
	skipIfMissingCreds(t)

	if os.Getenv("NYLAS_TEST_DELETE") != "true" {
		t.Skip("NYLAS_TEST_DELETE not set to 'true'")
	}

	email := testEmail
	if email == "" {
		email = "test@example.com"
	}

	var scheduleID string

	// Create a scheduled message (schedule for 1 hour from now)
	t.Run("create", func(t *testing.T) {
		stdout, stderr, err := runCLI("email", "send",
			"--to", email,
			"--subject", "CLI Scheduled Test",
			"--body", "This is a test scheduled email.",
			"--schedule", "1h",
			testGrantID)

		skipIfProviderNotSupported(t, stderr)

		if err != nil {
			// Scheduled send may not be supported by provider
			if strings.Contains(stderr, "not supported") || strings.Contains(stderr, "not available") {
				t.Skip("Scheduled send not supported by provider")
			}
			t.Fatalf("email send scheduled failed: %v\nstderr: %s", err, stderr)
		}

		if !strings.Contains(stdout, "scheduled") && !strings.Contains(stdout, "Scheduled") {
			t.Errorf("Expected 'scheduled' confirmation in output, got: %s", stdout)
		}

		// Extract schedule ID from output if available
		// Note: The output format may vary, so we'll try to find an ID pattern
		lines := strings.Split(stdout, "\n")
		for _, line := range lines {
			if strings.Contains(line, "ID:") {
				parts := strings.Split(line, "ID:")
				if len(parts) > 1 {
					scheduleID = strings.TrimSpace(parts[1])
					break
				}
			}
		}

		t.Logf("email send scheduled output: %s", stdout)
		if scheduleID != "" {
			t.Logf("Schedule ID: %s", scheduleID)
		}
	})

	if scheduleID == "" {
		t.Log("Could not extract schedule ID, skipping delete test")
		return
	}

	// Wait for scheduled message to appear in list
	time.Sleep(2 * time.Second)

	// List scheduled messages to verify it appears
	t.Run("list", func(t *testing.T) {
		stdout, stderr, err := runCLI("email", "scheduled", "list", testGrantID)
		if err != nil {
			t.Fatalf("email scheduled list failed: %v\nstderr: %s", err, stderr)
		}

		t.Logf("email scheduled list output:\n%s", stdout)
	})

	// Cancel the scheduled message
	t.Run("cancel", func(t *testing.T) {
		stdout, stderr, err := runCLI("email", "scheduled", "cancel", scheduleID, testGrantID, "--force")
		if err != nil {
			t.Fatalf("email scheduled cancel failed: %v\nstderr: %s", err, stderr)
		}

		if !strings.Contains(stdout, "deleted") && !strings.Contains(stdout, "cancelled") &&
			!strings.Contains(stdout, "canceled") && !strings.Contains(stdout, "Canceled") {
			t.Errorf("Expected cancel confirmation in output, got: %s", stdout)
		}

		t.Logf("email scheduled cancel output: %s", stdout)
	})
}

// =============================================================================
// EMAIL SCHEDULED EDGE CASES
// =============================================================================

func TestCLI_EmailScheduledListInvalidGrantID(t *testing.T) {
	skipIfMissingCreds(t)

	_, stderr, err := runCLI("email", "scheduled", "list", "invalid-grant-id")

	if err == nil {
		t.Error("Expected error for invalid grant ID, but command succeeded")
	}

	t.Logf("email scheduled list invalid grant error: %s", stderr)
}
