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
// CALENDAR EVENT UPDATE/RSVP COMMAND TESTS
// =============================================================================

func TestCLI_CalendarEventsUpdateHelp(t *testing.T) {
	if testBinary == "" {
		t.Skip("CLI binary not found")
	}

	stdout, stderr, err := runCLI("calendar", "events", "update", "--help")

	if err != nil {
		t.Fatalf("calendar events update --help failed: %v\nstderr: %s", err, stderr)
	}

	// Should show usage and flags
	if !strings.Contains(stdout, "<event-id>") {
		t.Errorf("Expected '<event-id>' in help, got: %s", stdout)
	}
	if !strings.Contains(stdout, "--title") || !strings.Contains(stdout, "--location") {
		t.Errorf("Expected --title and --location flags in help, got: %s", stdout)
	}
	if !strings.Contains(stdout, "--visibility") {
		t.Errorf("Expected --visibility flag in help, got: %s", stdout)
	}

	t.Logf("calendar events update --help output:\n%s", stdout)
}

func TestCLI_CalendarEventsRSVPHelp(t *testing.T) {
	if testBinary == "" {
		t.Skip("CLI binary not found")
	}

	stdout, stderr, err := runCLI("calendar", "events", "rsvp", "--help")

	if err != nil {
		t.Fatalf("calendar events rsvp --help failed: %v\nstderr: %s", err, stderr)
	}

	// Should show usage and status options
	if !strings.Contains(stdout, "<event-id>") {
		t.Errorf("Expected '<event-id>' in help, got: %s", stdout)
	}
	if !strings.Contains(stdout, "<status>") {
		t.Errorf("Expected '<status>' in help, got: %s", stdout)
	}
	if !strings.Contains(stdout, "yes") || !strings.Contains(stdout, "no") || !strings.Contains(stdout, "maybe") {
		t.Errorf("Expected RSVP status options (yes, no, maybe) in help, got: %s", stdout)
	}
	if !strings.Contains(stdout, "--comment") {
		t.Errorf("Expected --comment flag in help, got: %s", stdout)
	}

	t.Logf("calendar events rsvp --help output:\n%s", stdout)
}

func TestCLI_CalendarEventsUpdateLifecycle(t *testing.T) {
	skipIfMissingCreds(t)

	if os.Getenv("NYLAS_TEST_DELETE") != "true" {
		t.Skip("NYLAS_TEST_DELETE not set to 'true'")
	}

	// Get tomorrow's date for the event
	tomorrow := time.Now().AddDate(0, 0, 1)
	startTime := tomorrow.Format("2006-01-02") + " 14:00"
	endTime := tomorrow.Format("2006-01-02") + " 15:00"
	eventTitle := fmt.Sprintf("CLI Update Test %d", time.Now().Unix())

	var eventID string

	// Create event first
	t.Run("create", func(t *testing.T) {
		stdout, stderr, err := runCLI("calendar", "events", "create",
			"--title", eventTitle,
			"--start", startTime,
			"--end", endTime,
			"--location", "Original Location",
			testGrantID)

		if err != nil {
			if strings.Contains(stderr, "no writable calendar") || strings.Contains(stderr, "no calendars") {
				t.Skip("No writable calendar available")
			}
			t.Fatalf("calendar events create failed: %v\nstderr: %s", err, stderr)
		}

		// Extract event ID from output
		if idx := strings.Index(stdout, "ID:"); idx != -1 {
			eventID = strings.TrimSpace(stdout[idx+3:])
			if newline := strings.Index(eventID, "\n"); newline != -1 {
				eventID = eventID[:newline]
			}
		}

		t.Logf("Event created with ID: %s", eventID)
	})

	if eventID == "" {
		t.Fatal("Failed to get event ID from create output")
	}

	// Wait for event to sync
	time.Sleep(2 * time.Second)

	// Update event
	t.Run("update", func(t *testing.T) {
		newTitle := eventTitle + " Updated"
		stdout, stderr, err := runCLI("calendar", "events", "update", eventID,
			"--title", newTitle,
			"--location", "Updated Location",
			"--description", "Updated description",
			testGrantID)

		if err != nil {
			t.Fatalf("calendar events update failed: %v\nstderr: %s", err, stderr)
		}

		if !strings.Contains(stdout, "Event updated") {
			t.Errorf("Expected 'Event updated' in output, got: %s", stdout)
		}

		t.Logf("calendar events update output: %s", stdout)
	})

	// Verify update by showing the event
	t.Run("verify", func(t *testing.T) {
		stdout, stderr, err := runCLI("calendar", "events", "show", eventID, testGrantID)
		if err != nil {
			t.Fatalf("calendar events show failed: %v\nstderr: %s", err, stderr)
		}

		if !strings.Contains(stdout, "Updated") {
			t.Errorf("Expected updated title in output, got: %s", stdout)
		}

		t.Logf("calendar events show (updated) output:\n%s", stdout)
	})

	// Clean up
	t.Run("cleanup", func(t *testing.T) {
		runCLIWithInput("y\n", "calendar", "events", "delete", eventID, testGrantID)
	})
}
