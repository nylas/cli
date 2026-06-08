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
// CALENDAR COMMAND TESTS
// =============================================================================

func TestCLI_CalendarList(t *testing.T) {
	skipIfMissingCreds(t)

	stdout, stderr, err := runCLI("calendar", "list", testGrantID)
	skipIfProviderNotSupported(t, stderr)

	if err != nil {
		t.Fatalf("calendar list failed: %v\nstderr: %s", err, stderr)
	}

	// Should show calendar list or "No calendars found"
	if !strings.Contains(stdout, "Found") && !strings.Contains(stdout, "No calendars found") {
		t.Errorf("Expected calendar list output, got: %s", stdout)
	}

	t.Logf("calendar list output:\n%s", stdout)
}

func TestCLI_CalendarHelp(t *testing.T) {
	if testBinary == "" {
		t.Skip("CLI binary not found")
	}

	stdout, stderr, err := runCLI("calendar", "--help")

	if err != nil {
		t.Fatalf("calendar --help failed: %v\nstderr: %s", err, stderr)
	}

	// Should show calendar subcommands
	if !strings.Contains(stdout, "list") || !strings.Contains(stdout, "events") {
		t.Errorf("Expected calendar subcommands in help, got: %s", stdout)
	}

	t.Logf("calendar --help output:\n%s", stdout)
}

func TestCLI_CalendarEventsList(t *testing.T) {
	skipIfMissingCreds(t)

	stdout, stderr, err := runCLI("calendar", "events", "list", testGrantID, "--limit", "5")
	skipIfProviderNotSupported(t, stderr)

	if err != nil {
		// May fail if no calendars
		if strings.Contains(stderr, "no calendars") {
			t.Skip("No calendars available")
		}
		t.Fatalf("calendar events list failed: %v\nstderr: %s", err, stderr)
	}

	// Should show events list or "No events found"
	if !strings.Contains(stdout, "Found") && !strings.Contains(stdout, "No events found") {
		t.Errorf("Expected events list output, got: %s", stdout)
	}

	t.Logf("calendar events list output:\n%s", stdout)
}

func TestCLI_CalendarEventsListWithDays(t *testing.T) {
	skipIfMissingCreds(t)

	stdout, stderr, err := runCLI("calendar", "events", "list", testGrantID, "--days", "30", "--limit", "10")
	skipIfProviderNotSupported(t, stderr)

	if err != nil {
		if strings.Contains(stderr, "no calendars") {
			t.Skip("No calendars available")
		}
		t.Fatalf("calendar events list --days failed: %v\nstderr: %s", err, stderr)
	}

	t.Logf("calendar events list --days output:\n%s", stdout)
}

func TestCLI_CalendarEventsHelp(t *testing.T) {
	if testBinary == "" {
		t.Skip("CLI binary not found")
	}

	stdout, stderr, err := runCLI("calendar", "events", "--help")

	if err != nil {
		t.Fatalf("calendar events --help failed: %v\nstderr: %s", err, stderr)
	}

	// Should show events subcommands
	if !strings.Contains(stdout, "list") || !strings.Contains(stdout, "create") {
		t.Errorf("Expected events subcommands in help, got: %s", stdout)
	}
	if !strings.Contains(stdout, "show") || !strings.Contains(stdout, "delete") {
		t.Errorf("Expected show and delete subcommands in help, got: %s", stdout)
	}

	t.Logf("calendar events --help output:\n%s", stdout)
}

func TestCLI_CalendarEventsCreateHelp(t *testing.T) {
	if testBinary == "" {
		t.Skip("CLI binary not found")
	}

	stdout, stderr, err := runCLI("calendar", "events", "create", "--help")

	if err != nil {
		t.Fatalf("calendar events create --help failed: %v\nstderr: %s", err, stderr)
	}

	// Should show required flags
	if !strings.Contains(stdout, "--title") || !strings.Contains(stdout, "--start") {
		t.Errorf("Expected --title and --start flags in help, got: %s", stdout)
	}

	// Should show optional flags
	if !strings.Contains(stdout, "--end") || !strings.Contains(stdout, "--location") {
		t.Errorf("Expected --end and --location flags in help, got: %s", stdout)
	}

	// Should show examples
	if !strings.Contains(stdout, "Examples:") {
		t.Errorf("Expected 'Examples:' in help, got: %s", stdout)
	}

	t.Logf("calendar events create --help output:\n%s", stdout)
}

func TestCLI_CalendarEventsLifecycle(t *testing.T) {
	skipIfMissingCreds(t)

	if os.Getenv("NYLAS_TEST_DELETE") != "true" {
		t.Skip("NYLAS_TEST_DELETE not set to 'true'")
	}

	// Get tomorrow's date for the event
	tomorrow := time.Now().AddDate(0, 0, 1)
	startTime := tomorrow.Format("2006-01-02") + " 14:00"
	endTime := tomorrow.Format("2006-01-02") + " 15:00"
	eventTitle := fmt.Sprintf("CLI Test Event %d", time.Now().Unix())

	var eventID string

	// Create event
	t.Run("create", func(t *testing.T) {
		stdout, stderr, err := runCLI("calendar", "events", "create",
			"--title", eventTitle,
			"--start", startTime,
			"--end", endTime,
			"--location", "Test Location",
			testGrantID)

		if err != nil {
			if strings.Contains(stderr, "no writable calendar") || strings.Contains(stderr, "no calendars") {
				t.Skip("No writable calendar available")
			}
			t.Fatalf("calendar events create failed: %v\nstderr: %s", err, stderr)
		}

		if !strings.Contains(stdout, "Event created") {
			t.Errorf("Expected 'Event created' in output, got: %s", stdout)
		}

		// Extract event ID from output
		if idx := strings.Index(stdout, "ID:"); idx != -1 {
			eventID = strings.TrimSpace(stdout[idx+3:])
			if newline := strings.Index(eventID, "\n"); newline != -1 {
				eventID = eventID[:newline]
			}
		}

		t.Logf("calendar events create output: %s", stdout)
		t.Logf("Event ID: %s", eventID)
	})

	if eventID == "" {
		t.Fatal("Failed to get event ID from create output")
	}

	// Wait for event to sync
	time.Sleep(2 * time.Second)

	// Show event
	t.Run("show", func(t *testing.T) {
		stdout, stderr, err := runCLI("calendar", "events", "show", eventID, testGrantID)
		if err != nil {
			t.Fatalf("calendar events show failed: %v\nstderr: %s", err, stderr)
		}

		if !strings.Contains(stdout, eventTitle) {
			t.Errorf("Expected event title in output, got: %s", stdout)
		}

		t.Logf("calendar events show output:\n%s", stdout)
	})

	// Delete event
	t.Run("delete", func(t *testing.T) {
		stdout, stderr, err := runCLIWithInput("y\n", "calendar", "events", "delete", eventID, testGrantID)
		if err != nil {
			t.Fatalf("calendar events delete failed: %v\nstderr: %s", err, stderr)
		}

		if !strings.Contains(stdout, "deleted") {
			t.Errorf("Expected 'deleted' in output, got: %s", stdout)
		}

		t.Logf("calendar events delete output: %s", stdout)
	})
}

func TestCLI_CalendarEventsCreate_AllDay(t *testing.T) {
	skipIfMissingCreds(t)

	if os.Getenv("NYLAS_TEST_DELETE") != "true" {
		t.Skip("NYLAS_TEST_DELETE not set to 'true'")
	}

	// Get tomorrow's date
	tomorrow := time.Now().AddDate(0, 0, 1)
	dateStr := tomorrow.Format("2006-01-02")
	eventTitle := fmt.Sprintf("CLI Test All Day %d", time.Now().Unix())

	// Create all-day event
	stdout, stderr, err := runCLI("calendar", "events", "create",
		"--title", eventTitle,
		"--start", dateStr,
		"--all-day",
		testGrantID)

	if err != nil {
		if strings.Contains(stderr, "no writable calendar") || strings.Contains(stderr, "no calendars") {
			t.Skip("No writable calendar available")
		}
		t.Fatalf("calendar events create --all-day failed: %v\nstderr: %s", err, stderr)
	}

	if !strings.Contains(stdout, "Event created") {
		t.Errorf("Expected 'Event created' in output, got: %s", stdout)
	}

	t.Logf("calendar events create --all-day output: %s", stdout)

	// Extract event ID and delete it
	if idx := strings.Index(stdout, "ID:"); idx != -1 {
		eventID := strings.TrimSpace(stdout[idx+3:])
		if newline := strings.Index(eventID, "\n"); newline != -1 {
			eventID = eventID[:newline]
		}
		// Clean up
		time.Sleep(time.Second)
		runCLIWithInput("y\n", "calendar", "events", "delete", eventID, testGrantID)
	}
}
