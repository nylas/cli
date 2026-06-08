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
// CALENDAR CRUD COMMAND TESTS
// =============================================================================

func TestCLI_CalendarShowHelp(t *testing.T) {
	if testBinary == "" {
		t.Skip("CLI binary not found")
	}

	stdout, stderr, err := runCLI("calendar", "show", "--help")

	if err != nil {
		t.Fatalf("calendar show --help failed: %v\nstderr: %s", err, stderr)
	}

	// Should show usage
	if !strings.Contains(stdout, "<calendar-id>") {
		t.Errorf("Expected '<calendar-id>' in help, got: %s", stdout)
	}

	t.Logf("calendar show --help output:\n%s", stdout)
}

func TestCLI_CalendarCreateHelp(t *testing.T) {
	if testBinary == "" {
		t.Skip("CLI binary not found")
	}

	stdout, stderr, err := runCLI("calendar", "create", "--help")

	if err != nil {
		t.Fatalf("calendar create --help failed: %v\nstderr: %s", err, stderr)
	}

	// Should show usage and flags
	if !strings.Contains(stdout, "<name>") {
		t.Errorf("Expected '<name>' in help, got: %s", stdout)
	}
	if !strings.Contains(stdout, "--description") || !strings.Contains(stdout, "--timezone") {
		t.Errorf("Expected --description and --timezone flags in help, got: %s", stdout)
	}

	t.Logf("calendar create --help output:\n%s", stdout)
}

func TestCLI_CalendarUpdateHelp(t *testing.T) {
	if testBinary == "" {
		t.Skip("CLI binary not found")
	}

	stdout, stderr, err := runCLI("calendar", "update", "--help")

	if err != nil {
		t.Fatalf("calendar update --help failed: %v\nstderr: %s", err, stderr)
	}

	// Should show usage and flags
	if !strings.Contains(stdout, "<calendar-id>") {
		t.Errorf("Expected '<calendar-id>' in help, got: %s", stdout)
	}
	if !strings.Contains(stdout, "--name") || !strings.Contains(stdout, "--color") {
		t.Errorf("Expected --name and --color flags in help, got: %s", stdout)
	}

	t.Logf("calendar update --help output:\n%s", stdout)
}

func TestCLI_CalendarDeleteHelp(t *testing.T) {
	if testBinary == "" {
		t.Skip("CLI binary not found")
	}

	stdout, stderr, err := runCLI("calendar", "delete", "--help")

	if err != nil {
		t.Fatalf("calendar delete --help failed: %v\nstderr: %s", err, stderr)
	}

	// Should show usage and flags
	if !strings.Contains(stdout, "<calendar-id>") {
		t.Errorf("Expected '<calendar-id>' in help, got: %s", stdout)
	}
	if !strings.Contains(stdout, "--force") {
		t.Errorf("Expected --force flag in help, got: %s", stdout)
	}

	t.Logf("calendar delete --help output:\n%s", stdout)
}

func TestCLI_CalendarCRUDLifecycle(t *testing.T) {
	skipIfMissingCreds(t)

	if os.Getenv("NYLAS_TEST_DELETE") != "true" {
		t.Skip("NYLAS_TEST_DELETE not set to 'true'")
	}

	calendarName := fmt.Sprintf("CLI Test Calendar %d", time.Now().Unix())
	var calendarID string

	// Create calendar
	t.Run("create", func(t *testing.T) {
		stdout, stderr, err := runCLI("calendar", "create", calendarName,
			"--description", "Test calendar created by CLI",
			"--timezone", "America/New_York",
			testGrantID)

		if err != nil {
			// Some providers may not support calendar creation
			if strings.Contains(stderr, "not supported") || strings.Contains(stderr, "read-only") {
				t.Skip("Calendar creation not supported by provider")
			}
			t.Fatalf("calendar create failed: %v\nstderr: %s", err, stderr)
		}

		if !strings.Contains(stdout, "Created calendar") {
			t.Errorf("Expected 'Created calendar' in output, got: %s", stdout)
		}

		// Extract calendar ID from output
		if idx := strings.Index(stdout, "ID:"); idx != -1 {
			calendarID = strings.TrimSpace(stdout[idx+3:])
			if paren := strings.Index(calendarID, ")"); paren != -1 {
				calendarID = calendarID[:paren]
			}
			if newline := strings.Index(calendarID, "\n"); newline != -1 {
				calendarID = calendarID[:newline]
			}
		}

		t.Logf("calendar create output: %s", stdout)
		t.Logf("Calendar ID: %s", calendarID)
	})

	if calendarID == "" {
		t.Fatal("Failed to get calendar ID from create output")
	}

	// Wait for calendar to sync
	time.Sleep(2 * time.Second)

	// Show calendar
	t.Run("show", func(t *testing.T) {
		stdout, stderr, err := runCLI("calendar", "show", calendarID, testGrantID)
		if err != nil {
			t.Fatalf("calendar show failed: %v\nstderr: %s", err, stderr)
		}

		if !strings.Contains(stdout, calendarName) {
			t.Errorf("Expected calendar name in output, got: %s", stdout)
		}

		t.Logf("calendar show output:\n%s", stdout)
	})

	// Update calendar
	t.Run("update", func(t *testing.T) {
		newName := calendarName + " Updated"
		stdout, stderr, err := runCLI("calendar", "update", calendarID,
			"--name", newName,
			"--description", "Updated description",
			testGrantID)

		if err != nil {
			// Some providers may not support calendar updates
			if strings.Contains(stderr, "not supported") {
				t.Skip("Calendar update not supported by provider")
			}
			t.Fatalf("calendar update failed: %v\nstderr: %s", err, stderr)
		}

		if !strings.Contains(stdout, "Updated calendar") {
			t.Errorf("Expected 'Updated calendar' in output, got: %s", stdout)
		}

		t.Logf("calendar update output: %s", stdout)
	})

	// Delete calendar
	t.Run("delete", func(t *testing.T) {
		stdout, stderr, err := runCLI("calendar", "delete", calendarID, "--force", testGrantID)
		if err != nil {
			// Some providers may not support calendar deletion
			if strings.Contains(stderr, "not supported") {
				t.Skip("Calendar deletion not supported by provider")
			}
			t.Fatalf("calendar delete failed: %v\nstderr: %s", err, stderr)
		}

		if !strings.Contains(stdout, "deleted") {
			t.Errorf("Expected 'deleted' in output, got: %s", stdout)
		}

		t.Logf("calendar delete output: %s", stdout)
	})
}
