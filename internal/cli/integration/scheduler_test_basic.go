//go:build integration

package integration

import (
	"strings"
	"testing"
)

// =============================================================================
// SCHEDULER COMMAND TESTS
// =============================================================================

func TestCLI_SchedulerHelp(t *testing.T) {
	if testBinary == "" {
		t.Skip("CLI binary not found")
	}

	stdout, stderr, err := runCLI("scheduler", "--help")

	if err != nil {
		t.Fatalf("scheduler --help failed: %v\nstderr: %s", err, stderr)
	}

	// Should show scheduler subcommands
	if !strings.Contains(stdout, "configurations") || !strings.Contains(stdout, "bookings") {
		t.Errorf("Expected scheduler subcommands in help, got: %s", stdout)
	}

	t.Logf("scheduler --help output:\n%s", stdout)
}

// Configurations Tests

func TestCLI_SchedulerConfigurationsHelp(t *testing.T) {
	if testBinary == "" {
		t.Skip("CLI binary not found")
	}

	stdout, stderr, err := runCLI("scheduler", "configurations", "--help")

	if err != nil {
		t.Fatalf("scheduler configurations --help failed: %v\nstderr: %s", err, stderr)
	}

	// Should show configuration subcommands
	if !strings.Contains(stdout, "list") || !strings.Contains(stdout, "create") {
		t.Errorf("Expected configuration subcommands in help, got: %s", stdout)
	}

	t.Logf("scheduler configurations --help output:\n%s", stdout)
}

func TestCLI_SchedulerConfigurationsList(t *testing.T) {
	skipIfMissingCreds(t)

	stdout, stderr, err := runCLI("scheduler", "configurations", "list")
	skipIfProviderNotSupported(t, stderr)

	if err != nil {
		t.Fatalf("scheduler configurations list failed: %v\nstderr: %s", err, stderr)
	}

	// Should show configurations list or "No scheduler configurations found"
	if !strings.Contains(stdout, "Found") && !strings.Contains(stdout, "No scheduler configurations found") {
		t.Errorf("Expected configurations list output, got: %s", stdout)
	}

	t.Logf("scheduler configurations list output:\n%s", stdout)
}

func TestCLI_SchedulerConfigurationsListJSON(t *testing.T) {
	skipIfMissingCreds(t)

	stdout, stderr, err := runCLI("scheduler", "configurations", "list", "--json")
	skipIfProviderNotSupported(t, stderr)

	if err != nil {
		t.Fatalf("scheduler configurations list --json failed: %v\nstderr: %s", err, stderr)
	}

	// Should output JSON (array)
	trimmed := strings.TrimSpace(stdout)
	if len(trimmed) > 0 && !strings.HasPrefix(trimmed, "[") {
		t.Errorf("Expected JSON array output, got: %s", stdout)
	}

	t.Logf("scheduler configurations list --json output:\n%s", stdout)
}

// Sessions Tests

func TestCLI_SchedulerSessionsHelp(t *testing.T) {
	if testBinary == "" {
		t.Skip("CLI binary not found")
	}

	stdout, stderr, err := runCLI("scheduler", "sessions", "--help")

	if err != nil {
		t.Fatalf("scheduler sessions --help failed: %v\nstderr: %s", err, stderr)
	}

	// Should show session subcommands
	if !strings.Contains(stdout, "create") || !strings.Contains(stdout, "show") {
		t.Errorf("Expected session subcommands in help, got: %s", stdout)
	}

	t.Logf("scheduler sessions --help output:\n%s", stdout)
}

// Bookings Tests

func TestCLI_SchedulerBookingsHelp(t *testing.T) {
	if testBinary == "" {
		t.Skip("CLI binary not found")
	}

	stdout, stderr, err := runCLI("scheduler", "bookings", "--help")

	if err != nil {
		t.Fatalf("scheduler bookings --help failed: %v\nstderr: %s", err, stderr)
	}

	// Should show booking subcommands
	if !strings.Contains(stdout, "list") || !strings.Contains(stdout, "confirm") {
		t.Errorf("Expected booking subcommands in help, got: %s", stdout)
	}

	t.Logf("scheduler bookings --help output:\n%s", stdout)
}

func TestCLI_SchedulerBookingsList(t *testing.T) {
	skipIfMissingCreds(t)

	stdout, stderr, err := runCLI("scheduler", "bookings", "list")
	skipIfProviderNotSupported(t, stderr)

	// Skip if bookings endpoint isn't available in this Nylas API version
	if err != nil && strings.Contains(stderr, "Unrecognized request URL") {
		t.Skip("Scheduler bookings endpoint not available in this Nylas API version")
	}

	if err != nil {
		t.Fatalf("scheduler bookings list failed: %v\nstderr: %s", err, stderr)
	}

	// Should show bookings list or "No bookings found"
	if !strings.Contains(stdout, "Found") && !strings.Contains(stdout, "No bookings found") {
		t.Errorf("Expected bookings list output, got: %s", stdout)
	}

	t.Logf("scheduler bookings list output:\n%s", stdout)
}

func TestCLI_SchedulerBookingsListJSON(t *testing.T) {
	skipIfMissingCreds(t)

	stdout, stderr, err := runCLI("scheduler", "bookings", "list", "--json")
	skipIfProviderNotSupported(t, stderr)

	// Skip if bookings endpoint isn't available in this Nylas API version
	if err != nil && strings.Contains(stderr, "Unrecognized request URL") {
		t.Skip("Scheduler bookings endpoint not available in this Nylas API version")
	}

	if err != nil {
		t.Fatalf("scheduler bookings list --json failed: %v\nstderr: %s", err, stderr)
	}

	// Should output JSON (array)
	trimmed := strings.TrimSpace(stdout)
	if len(trimmed) > 0 && !strings.HasPrefix(trimmed, "[") {
		t.Errorf("Expected JSON array output, got: %s", stdout)
	}

	t.Logf("scheduler bookings list --json output:\n%s", stdout)
}

// =============================================================================
// SCHEDULER BOOKINGS CRUD TESTS (Phase 2.6)
// =============================================================================

func TestCLI_SchedulerBookingsShowHelp(t *testing.T) {
	if testBinary == "" {
		t.Skip("CLI binary not found")
	}

	stdout, stderr, err := runCLI("scheduler", "bookings", "show", "--help")

	if err != nil {
		t.Fatalf("scheduler bookings show --help failed: %v\nstderr: %s", err, stderr)
	}

	// Should show usage with booking-id
	if !strings.Contains(stdout, "booking-id") && !strings.Contains(stdout, "<booking-id>") {
		t.Errorf("Expected booking-id in help, got: %s", stdout)
	}
	if !strings.Contains(stdout, "--json") {
		t.Errorf("Expected '--json' flag in help, got: %s", stdout)
	}

	t.Logf("scheduler bookings show --help output:\n%s", stdout)
}

func TestCLI_SchedulerBookingsConfirmHelp(t *testing.T) {
	if testBinary == "" {
		t.Skip("CLI binary not found")
	}

	stdout, stderr, err := runCLI("scheduler", "bookings", "confirm", "--help")

	if err != nil {
		t.Fatalf("scheduler bookings confirm --help failed: %v\nstderr: %s", err, stderr)
	}

	// Should show usage with booking-id
	if !strings.Contains(stdout, "booking-id") && !strings.Contains(stdout, "<booking-id>") {
		t.Errorf("Expected booking-id in help, got: %s", stdout)
	}

	t.Logf("scheduler bookings confirm --help output:\n%s", stdout)
}

func TestCLI_SchedulerBookingsRescheduleHelp(t *testing.T) {
	if testBinary == "" {
		t.Skip("CLI binary not found")
	}

	stdout, stderr, err := runCLI("scheduler", "bookings", "reschedule", "--help")

	if err != nil {
		t.Fatalf("scheduler bookings reschedule --help failed: %v\nstderr: %s", err, stderr)
	}

	// Should show usage with booking-id
	if !strings.Contains(stdout, "booking-id") && !strings.Contains(stdout, "<booking-id>") {
		t.Errorf("Expected booking-id in help, got: %s", stdout)
	}

	t.Logf("scheduler bookings reschedule --help output:\n%s", stdout)
}

func TestCLI_SchedulerBookingsCancelHelp(t *testing.T) {
	if testBinary == "" {
		t.Skip("CLI binary not found")
	}

	stdout, stderr, err := runCLI("scheduler", "bookings", "cancel", "--help")

	if err != nil {
		t.Fatalf("scheduler bookings cancel --help failed: %v\nstderr: %s", err, stderr)
	}

	// Should show usage with booking-id
	if !strings.Contains(stdout, "booking-id") && !strings.Contains(stdout, "<booking-id>") {
		t.Errorf("Expected booking-id in help, got: %s", stdout)
	}
	// Should show --yes flag for skipping confirmation
	if !strings.Contains(stdout, "--yes") && !strings.Contains(stdout, "-y") {
		t.Errorf("Expected '--yes' flag in help, got: %s", stdout)
	}
	// Should show --reason flag
	if !strings.Contains(stdout, "--reason") {
		t.Errorf("Expected '--reason' flag in help, got: %s", stdout)
	}

	t.Logf("scheduler bookings cancel --help output:\n%s", stdout)
}

// Lifecycle test: Full CRUD workflow (show, confirm, reschedule, cancel)
// NOTE: This test is skipped due to complex API requirements
// See skip message below for manual testing instructions
func TestCLI_SchedulerBookingsLifecycle(t *testing.T) {
	skipIfMissingCreds(t)

	t.Skip("Scheduler bookings CRUD operations require existing bookings created via sessions.\n" +
		"This requires:\n" +
		"  1. Valid scheduler configuration with availability rules\n" +
		"  2. Scheduler session created via API or web interface\n" +
		"  3. Booking created through the session booking flow\n" +
		"  4. Proper participant and calendar permissions\n\n" +
		"These requirements cannot be reliably satisfied via simple CLI automation.\n\n" +
		"Manual testing:\n" +
		"  (1) Create a scheduler configuration via Dashboard\n" +
		"  (2) Create a booking via the public booking page or sessions API\n" +
		"  (3) Get booking ID from 'scheduler bookings list'\n" +
		"  (4) Test show command: nylas scheduler bookings show <booking-id>\n" +
		"  (5) Test show JSON: nylas scheduler bookings show <booking-id> --json\n" +
		"  (6) Test confirm (if pending): nylas scheduler bookings confirm <booking-id>\n" +
		"  (7) Test reschedule: nylas scheduler bookings reschedule <booking-id>\n" +
		"  (8) Test cancel: nylas scheduler bookings cancel <booking-id> --yes --reason 'Testing'\n")
}

// Pages Tests

func TestCLI_SchedulerPagesHelp(t *testing.T) {
	if testBinary == "" {
		t.Skip("CLI binary not found")
	}

	stdout, stderr, err := runCLI("scheduler", "pages", "--help")

	if err != nil {
		t.Fatalf("scheduler pages --help failed: %v\nstderr: %s", err, stderr)
	}

	// Should show page subcommands
	if !strings.Contains(stdout, "list") || !strings.Contains(stdout, "create") {
		t.Errorf("Expected page subcommands in help, got: %s", stdout)
	}

	t.Logf("scheduler pages --help output:\n%s", stdout)
}
