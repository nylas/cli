//go:build integration

package integration

import (
	"strings"
	"testing"
)

func TestCLI_SchedulerPagesList(t *testing.T) {
	skipIfMissingCreds(t)

	stdout, stderr, err := runCLI("scheduler", "pages", "list")
	skipIfProviderNotSupported(t, stderr)

	// Skip if pages endpoint isn't available in this Nylas API version
	if err != nil && strings.Contains(stderr, "Unrecognized request URL") {
		t.Skip("Scheduler pages endpoint not available in this Nylas API version")
	}

	if err != nil {
		t.Fatalf("scheduler pages list failed: %v\nstderr: %s", err, stderr)
	}

	// Should show pages list or "No scheduler pages found"
	if !strings.Contains(stdout, "Found") && !strings.Contains(stdout, "No scheduler pages found") {
		t.Errorf("Expected pages list output, got: %s", stdout)
	}

	t.Logf("scheduler pages list output:\n%s", stdout)
}

func TestCLI_SchedulerPagesListJSON(t *testing.T) {
	skipIfMissingCreds(t)

	stdout, stderr, err := runCLI("scheduler", "pages", "list", "--json")
	skipIfProviderNotSupported(t, stderr)

	// Skip if pages endpoint isn't available in this Nylas API version
	if err != nil && strings.Contains(stderr, "Unrecognized request URL") {
		t.Skip("Scheduler pages endpoint not available in this Nylas API version")
	}

	if err != nil {
		t.Fatalf("scheduler pages list --json failed: %v\nstderr: %s", err, stderr)
	}

	// Should output JSON (array)
	trimmed := strings.TrimSpace(stdout)
	if len(trimmed) > 0 && !strings.HasPrefix(trimmed, "[") {
		t.Errorf("Expected JSON array output, got: %s", stdout)
	}

	t.Logf("scheduler pages list --json output:\n%s", stdout)
}

// =============================================================================
// SCHEDULER PAGES CRUD TESTS (Phase 2.7)
// =============================================================================

func TestCLI_SchedulerPagesCreateHelp(t *testing.T) {
	if testBinary == "" {
		t.Skip("CLI binary not found")
	}

	stdout, stderr, err := runCLI("scheduler", "pages", "create", "--help")

	if err != nil {
		t.Fatalf("scheduler pages create --help failed: %v\nstderr: %s", err, stderr)
	}

	// Should show required flags
	if !strings.Contains(stdout, "--name") {
		t.Errorf("Expected '--name' flag in help, got: %s", stdout)
	}
	if !strings.Contains(stdout, "--config-id") {
		t.Errorf("Expected '--config-id' flag in help, got: %s", stdout)
	}
	if !strings.Contains(stdout, "--slug") {
		t.Errorf("Expected '--slug' flag in help, got: %s", stdout)
	}

	t.Logf("scheduler pages create --help output:\n%s", stdout)
}

func TestCLI_SchedulerPagesShowHelp(t *testing.T) {
	if testBinary == "" {
		t.Skip("CLI binary not found")
	}

	stdout, stderr, err := runCLI("scheduler", "pages", "show", "--help")

	if err != nil {
		t.Fatalf("scheduler pages show --help failed: %v\nstderr: %s", err, stderr)
	}

	// Should show usage with page-id
	if !strings.Contains(stdout, "page-id") && !strings.Contains(stdout, "<page-id>") {
		t.Errorf("Expected page-id in help, got: %s", stdout)
	}
	if !strings.Contains(stdout, "--json") {
		t.Errorf("Expected '--json' flag in help, got: %s", stdout)
	}

	t.Logf("scheduler pages show --help output:\n%s", stdout)
}

func TestCLI_SchedulerPagesUpdateHelp(t *testing.T) {
	if testBinary == "" {
		t.Skip("CLI binary not found")
	}

	stdout, stderr, err := runCLI("scheduler", "pages", "update", "--help")

	if err != nil {
		t.Fatalf("scheduler pages update --help failed: %v\nstderr: %s", err, stderr)
	}

	// Should show usage with page-id
	if !strings.Contains(stdout, "page-id") && !strings.Contains(stdout, "<page-id>") {
		t.Errorf("Expected page-id in help, got: %s", stdout)
	}
	// Should show update flags
	if !strings.Contains(stdout, "--name") {
		t.Errorf("Expected '--name' flag in help, got: %s", stdout)
	}
	if !strings.Contains(stdout, "--slug") {
		t.Errorf("Expected '--slug' flag in help, got: %s", stdout)
	}

	t.Logf("scheduler pages update --help output:\n%s", stdout)
}

func TestCLI_SchedulerPagesDeleteHelp(t *testing.T) {
	if testBinary == "" {
		t.Skip("CLI binary not found")
	}

	stdout, stderr, err := runCLI("scheduler", "pages", "delete", "--help")

	if err != nil {
		t.Fatalf("scheduler pages delete --help failed: %v\nstderr: %s", err, stderr)
	}

	// Should show usage with page-id
	if !strings.Contains(stdout, "page-id") && !strings.Contains(stdout, "<page-id>") {
		t.Errorf("Expected page-id in help, got: %s", stdout)
	}
	// Should show --yes flag for skipping confirmation
	if !strings.Contains(stdout, "--yes") && !strings.Contains(stdout, "-y") {
		t.Errorf("Expected '--yes' flag in help, got: %s", stdout)
	}

	t.Logf("scheduler pages delete --help output:\n%s", stdout)
}

// Lifecycle test: Full CRUD workflow (create, show, update, delete)
// NOTE: This test is skipped due to complex API requirements
// See skip message below for manual testing instructions
func TestCLI_SchedulerPagesLifecycle(t *testing.T) {
	skipIfMissingCreds(t)

	t.Skip("Scheduler pages CRUD operations require existing scheduler configurations.\n" +
		"This requires:\n" +
		"  1. Valid scheduler configuration ID\n" +
		"  2. Unique page slug that doesn't conflict with existing pages\n" +
		"  3. Proper account permissions for page management\n\n" +
		"These requirements cannot be reliably satisfied via simple CLI automation.\n\n" +
		"Manual testing:\n" +
		"  (1) Get a configuration ID: nylas scheduler configurations list\n" +
		"  (2) Create page: nylas scheduler pages create --name 'Test Page' --config-id <config-id> --slug test-page-123\n" +
		"  (3) Show page: nylas scheduler pages show <page-id>\n" +
		"  (4) Show JSON: nylas scheduler pages show <page-id> --json\n" +
		"  (5) Update page: nylas scheduler pages update <page-id> --name 'Updated Page'\n" +
		"  (6) Delete page: nylas scheduler pages delete <page-id> --yes\n")
}

// =============================================================================
// SCHEDULER SESSIONS TESTS (Phase 2.8)
// =============================================================================

func TestCLI_SchedulerSessionsCreateHelp(t *testing.T) {
	if testBinary == "" {
		t.Skip("CLI binary not found")
	}

	stdout, stderr, err := runCLI("scheduler", "sessions", "create", "--help")

	if err != nil {
		t.Fatalf("scheduler sessions create --help failed: %v\nstderr: %s", err, stderr)
	}

	// Should show required flags
	if !strings.Contains(stdout, "--config-id") {
		t.Errorf("Expected '--config-id' flag in help, got: %s", stdout)
	}
	// Should show optional ttl flag
	if !strings.Contains(stdout, "--ttl") {
		t.Errorf("Expected '--ttl' flag in help, got: %s", stdout)
	}

	t.Logf("scheduler sessions create --help output:\n%s", stdout)
}

func TestCLI_SchedulerSessionsShowHelp(t *testing.T) {
	if testBinary == "" {
		t.Skip("CLI binary not found")
	}

	stdout, stderr, err := runCLI("scheduler", "sessions", "show", "--help")

	if err != nil {
		t.Fatalf("scheduler sessions show --help failed: %v\nstderr: %s", err, stderr)
	}

	// Should show usage with session-id
	if !strings.Contains(stdout, "session-id") && !strings.Contains(stdout, "<session-id>") {
		t.Errorf("Expected session-id in help, got: %s", stdout)
	}
	// Should show --json flag
	if !strings.Contains(stdout, "--json") {
		t.Errorf("Expected '--json' flag in help, got: %s", stdout)
	}

	t.Logf("scheduler sessions show --help output:\n%s", stdout)
}

// Lifecycle test: Create and show session
// NOTE: This test is skipped due to complex API requirements
// See skip message below for manual testing instructions
func TestCLI_SchedulerSessionsLifecycle(t *testing.T) {
	skipIfMissingCreds(t)

	t.Skip("Scheduler sessions operations require existing scheduler configurations.\n" +
		"This requires:\n" +
		"  1. Valid scheduler configuration ID\n" +
		"  2. Proper account permissions for session management\n" +
		"  3. Sessions expire based on TTL (default 30 minutes)\n\n" +
		"These requirements cannot be reliably satisfied via simple CLI automation.\n\n" +
		"Manual testing:\n" +
		"  (1) Get a configuration ID: nylas scheduler configurations list\n" +
		"  (2) Create session: nylas scheduler sessions create --config-id <config-id>\n" +
		"  (3) Create session with custom TTL: nylas scheduler sessions create --config-id <config-id> --ttl 60\n" +
		"  (4) Show session: nylas scheduler sessions show <session-id>\n" +
		"  (5) Show JSON: nylas scheduler sessions show <session-id> --json\n" +
		"  (6) Verify session ID and configuration ID in output\n\n" +
		"Note: Sessions are temporary and automatically expire. There is no delete command.\n")
}

// =============================================================================
// SCHEDULER CONFIGURATIONS CRUD TESTS (Phase 2.5)
// =============================================================================

func TestCLI_SchedulerConfigurationsCreateHelp(t *testing.T) {
	if testBinary == "" {
		t.Skip("CLI binary not found")
	}

	stdout, stderr, err := runCLI("scheduler", "configurations", "create", "--help")

	if err != nil {
		t.Fatalf("scheduler configurations create --help failed: %v\nstderr: %s", err, stderr)
	}

	// Should show required flags
	if !strings.Contains(stdout, "--name") {
		t.Errorf("Expected '--name' flag in help, got: %s", stdout)
	}
	if !strings.Contains(stdout, "--title") {
		t.Errorf("Expected '--title' flag in help, got: %s", stdout)
	}
	if !strings.Contains(stdout, "--duration") {
		t.Errorf("Expected '--duration' flag in help, got: %s", stdout)
	}

	t.Logf("scheduler configurations create --help output:\n%s", stdout)
}

func TestCLI_SchedulerConfigurationsShowHelp(t *testing.T) {
	if testBinary == "" {
		t.Skip("CLI binary not found")
	}

	stdout, stderr, err := runCLI("scheduler", "configurations", "show", "--help")

	if err != nil {
		t.Fatalf("scheduler configurations show --help failed: %v\nstderr: %s", err, stderr)
	}

	// Should show usage with config-id
	if !strings.Contains(stdout, "config-id") && !strings.Contains(stdout, "<id>") {
		t.Errorf("Expected config-id in help, got: %s", stdout)
	}

	t.Logf("scheduler configurations show --help output:\n%s", stdout)
}

func TestCLI_SchedulerConfigurationsUpdateHelp(t *testing.T) {
	if testBinary == "" {
		t.Skip("CLI binary not found")
	}

	stdout, stderr, err := runCLI("scheduler", "configurations", "update", "--help")

	if err != nil {
		t.Fatalf("scheduler configurations update --help failed: %v\nstderr: %s", err, stderr)
	}

	// Should show update flags
	if !strings.Contains(stdout, "--name") {
		t.Errorf("Expected '--name' flag in help, got: %s", stdout)
	}
	if !strings.Contains(stdout, "--duration") {
		t.Errorf("Expected '--duration' flag in help, got: %s", stdout)
	}

	t.Logf("scheduler configurations update --help output:\n%s", stdout)
}

func TestCLI_SchedulerConfigurationsDeleteHelp(t *testing.T) {
	if testBinary == "" {
		t.Skip("CLI binary not found")
	}

	stdout, stderr, err := runCLI("scheduler", "configurations", "delete", "--help")

	if err != nil {
		t.Fatalf("scheduler configurations delete --help failed: %v\nstderr: %s", err, stderr)
	}

	// Should show --yes flag
	if !strings.Contains(stdout, "--yes") && !strings.Contains(stdout, "-y") {
		t.Errorf("Expected '--yes' flag in help, got: %s", stdout)
	}

	t.Logf("scheduler configurations delete --help output:\n%s", stdout)
}

// Lifecycle test: Full CRUD workflow (create, show, update, delete)
// NOTE: This test is skipped due to complex API requirements
// See skip message below for manual testing instructions
func TestCLI_SchedulerConfigurationsLifecycle(t *testing.T) {
	skipIfMissingCreds(t)

	t.Skip("Scheduler configurations create requires complex participant availability and booking data.\n" +
		"This endpoint requires:\n" +
		"  1. Participant with availability subobject (calendar_ids + open_hours)\n" +
		"  2. Participant with booking subobject (calendar_id for bookings)\n" +
		"  3. Participant email must match the grant/API key's associated email\n" +
		"  4. Proper calendar access and permissions\n\n" +
		"These requirements cannot be reliably satisfied via simple CLI flags or programmatic creation.\n\n" +
		"Manual testing:\n" +
		"  (1) Create configuration via Nylas Dashboard or API with proper availability/booking data\n" +
		"  (2) Use 'scheduler configurations list' to get config ID\n" +
		"  (3) Test show command: nylas scheduler configurations show <config-id>\n" +
		"  (4) Test update command: nylas scheduler configurations update <config-id> --duration 45\n" +
		"  (5) Test delete command: nylas scheduler configurations delete <config-id> --yes\n")
}
