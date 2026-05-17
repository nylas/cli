//go:build integration

package integration

import (
	"strings"
	"testing"
)

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

	if !strings.Contains(stdout, "--config-id") {
		t.Errorf("Expected '--config-id' flag in help, got: %s", stdout)
	}
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

	if !strings.Contains(stdout, "session-id") && !strings.Contains(stdout, "<session-id>") {
		t.Errorf("Expected session-id in help, got: %s", stdout)
	}
	if !strings.Contains(stdout, "--json") {
		t.Errorf("Expected '--json' flag in help, got: %s", stdout)
	}

	t.Logf("scheduler sessions show --help output:\n%s", stdout)
}

func TestCLI_SchedulerSessionsLifecycle(t *testing.T) {
	skipIfMissingCreds(t)

	t.Skip("Scheduler sessions operations require existing scheduler configurations.")
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

	if !strings.Contains(stdout, "--yes") && !strings.Contains(stdout, "-y") {
		t.Errorf("Expected '--yes' flag in help, got: %s", stdout)
	}

	t.Logf("scheduler configurations delete --help output:\n%s", stdout)
}

func TestCLI_SchedulerConfigurationsLifecycle(t *testing.T) {
	skipIfMissingCreds(t)

	t.Skip("Scheduler configurations create requires complex participant availability and booking data.")
}
