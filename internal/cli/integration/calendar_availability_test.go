//go:build integration

package integration

import (
	"strings"
	"testing"
)

// =============================================================================
// CALENDAR AVAILABILITY COMMAND TESTS
// =============================================================================

func TestCLI_CalendarAvailabilityHelp(t *testing.T) {
	if testBinary == "" {
		t.Skip("CLI binary not found")
	}

	stdout, stderr, err := runCLI("calendar", "availability", "--help")

	if err != nil {
		t.Fatalf("calendar availability --help failed: %v\nstderr: %s", err, stderr)
	}

	// Should show availability subcommands
	if !strings.Contains(stdout, "check") || !strings.Contains(stdout, "find") {
		t.Errorf("Expected 'check' and 'find' subcommands in help, got: %s", stdout)
	}

	t.Logf("calendar availability --help output:\n%s", stdout)
}

func TestCLI_CalendarAvailabilityCheck(t *testing.T) {
	skipIfMissingCreds(t)

	stdout, stderr, err := runCLI("calendar", "availability", "check", testGrantID)
	skipIfProviderNotSupported(t, stderr)

	if err != nil {
		// May fail if no calendar access
		if strings.Contains(stderr, "no calendars") || strings.Contains(stderr, "not found") {
			t.Skip("No calendars available for availability check")
		}
		t.Fatalf("calendar availability check failed: %v\nstderr: %s", err, stderr)
	}

	// Should show free/busy status
	if !strings.Contains(stdout, "Free/Busy") && !strings.Contains(stdout, "free") && !strings.Contains(stdout, "Busy") {
		t.Errorf("Expected free/busy output, got: %s", stdout)
	}

	t.Logf("calendar availability check output:\n%s", stdout)
}

func TestCLI_CalendarAvailabilityCheckWithDuration(t *testing.T) {
	skipIfMissingCreds(t)

	stdout, stderr, err := runCLI("calendar", "availability", "check", testGrantID,
		"--duration", "2d")
	skipIfProviderNotSupported(t, stderr)

	if err != nil {
		if strings.Contains(stderr, "no calendars") || strings.Contains(stderr, "not found") {
			t.Skip("No calendars available")
		}
		t.Fatalf("calendar availability check --duration failed: %v\nstderr: %s", err, stderr)
	}

	t.Logf("calendar availability check --duration output:\n%s", stdout)
}

func TestCLI_CalendarAvailabilityCheckJSON(t *testing.T) {
	skipIfMissingCreds(t)

	stdout, stderr, err := runCLI("calendar", "availability", "check", testGrantID,
		"--format", "json")
	skipIfProviderNotSupported(t, stderr)

	if err != nil {
		if strings.Contains(stderr, "no calendars") || strings.Contains(stderr, "not found") {
			t.Skip("No calendars available")
		}
		t.Fatalf("calendar availability check --format json failed: %v\nstderr: %s", err, stderr)
	}

	// Should be valid JSON
	trimmed := strings.TrimSpace(stdout)
	if len(trimmed) > 0 && trimmed[0] != '{' {
		t.Errorf("Expected JSON output, got: %s", stdout)
	}

	t.Logf("calendar availability check JSON output:\n%s", stdout)
}

func TestCLI_CalendarAvailabilityFindHelp(t *testing.T) {
	if testBinary == "" {
		t.Skip("CLI binary not found")
	}

	stdout, stderr, err := runCLI("calendar", "availability", "find", "--help")

	if err != nil {
		t.Fatalf("calendar availability find --help failed: %v\nstderr: %s", err, stderr)
	}

	// Should show required flags
	if !strings.Contains(stdout, "--participants") {
		t.Errorf("Expected '--participants' flag in help, got: %s", stdout)
	}
	if !strings.Contains(stdout, "--duration") {
		t.Errorf("Expected '--duration' flag in help, got: %s", stdout)
	}

	t.Logf("calendar availability find --help output:\n%s", stdout)
}

func TestCLI_CalendarAvailabilityFind(t *testing.T) {
	skipIfMissingCreds(t)

	// Use test email if available
	email := testEmail
	if email == "" {
		email = "test@example.com"
	}

	stdout, stderr, err := runCLI("calendar", "availability", "find",
		"--participants", email,
		"--duration", "30")
	skipIfProviderNotSupported(t, stderr)

	if err != nil {
		// May fail if calendar feature not available or participant not found
		if strings.Contains(stderr, "not available") || strings.Contains(stderr, "not found") ||
			strings.Contains(stderr, "Failed to find a valid Grant") {
			t.Skip("Availability find not available or participant not found")
		}
		t.Fatalf("calendar availability find failed: %v\nstderr: %s", err, stderr)
	}

	// Should show available slots or "No available" message
	if !strings.Contains(stdout, "Available") && !strings.Contains(stdout, "available") && !strings.Contains(stdout, "No available") {
		t.Errorf("Expected availability output, got: %s", stdout)
	}

	t.Logf("calendar availability find output:\n%s", stdout)
}
