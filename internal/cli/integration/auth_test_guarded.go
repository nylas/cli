//go:build integration

package integration

import (
	"os"
	"strings"
	"testing"
)

// =============================================================================
// AUTH TOKEN COMMAND TESTS (Phase 1.2) - GUARDED
// =============================================================================

func TestCLI_AuthTokenHelp(t *testing.T) {
	// GUARD: Only run if both environment variables are set
	if os.Getenv("NYLAS_TEST_DELETE") != "true" || os.Getenv("NYLAS_TEST_AUTH_LOGOUT") != "true" {
		t.Skip("Skipping auth test - requires NYLAS_TEST_DELETE=true and NYLAS_TEST_AUTH_LOGOUT=true")
	}
	if testBinary == "" {
		t.Skip("CLI binary not found")
	}

	stdout, stderr, err := runCLI("auth", "token", "--help")

	if err != nil {
		t.Fatalf("auth token --help failed: %v\nstderr: %s", err, stderr)
	}

	// Should show token command usage
	if !strings.Contains(stdout, "token") || !strings.Contains(stdout, "Show") {
		t.Errorf("Expected token command description in help, got: %s", stdout)
	}

	t.Logf("auth token --help output:\n%s", stdout)
}

func TestCLI_AuthToken(t *testing.T) {
	// GUARD: Only run if both environment variables are set
	if os.Getenv("NYLAS_TEST_DELETE") != "true" || os.Getenv("NYLAS_TEST_AUTH_LOGOUT") != "true" {
		t.Skip("Skipping auth test - requires NYLAS_TEST_DELETE=true and NYLAS_TEST_AUTH_LOGOUT=true")
	}
	skipIfMissingCreds(t)

	stdout, stderr, err := runCLI("auth", "token")

	if err != nil {
		t.Fatalf("auth token failed: %v\nstderr: %s", err, stderr)
	}

	// Should show API key/token
	if !strings.Contains(stdout, "nyk_") && !strings.Contains(stdout, "API") {
		t.Errorf("Expected API key in output, got: %s", stdout)
	}

	t.Logf("auth token output: [REDACTED - contains API key]")
}

// =============================================================================
// AUTH SWITCH COMMAND TESTS (Phase 1.2) - GUARDED
// =============================================================================

func TestCLI_AuthSwitchHelp(t *testing.T) {
	// GUARD: Only run if both environment variables are set
	if os.Getenv("NYLAS_TEST_DELETE") != "true" || os.Getenv("NYLAS_TEST_AUTH_LOGOUT") != "true" {
		t.Skip("Skipping auth test - requires NYLAS_TEST_DELETE=true and NYLAS_TEST_AUTH_LOGOUT=true")
	}
	if testBinary == "" {
		t.Skip("CLI binary not found")
	}

	stdout, stderr, err := runCLI("auth", "switch", "--help")

	if err != nil {
		t.Fatalf("auth switch --help failed: %v\nstderr: %s", err, stderr)
	}

	// Should show switch command usage
	if !strings.Contains(stdout, "switch") && !strings.Contains(stdout, "Switch") {
		t.Errorf("Expected switch command description in help, got: %s", stdout)
	}

	t.Logf("auth switch --help output:\n%s", stdout)
}

func TestCLI_AuthSwitch(t *testing.T) {
	// GUARD: Only run if both environment variables are set
	if os.Getenv("NYLAS_TEST_DELETE") != "true" || os.Getenv("NYLAS_TEST_AUTH_LOGOUT") != "true" {
		t.Skip("Skipping auth test - requires NYLAS_TEST_DELETE=true and NYLAS_TEST_AUTH_LOGOUT=true")
	}
	skipIfMissingCreds(t)

	// First, get the list of grants to find a grant to switch to
	listOut, _, err := runCLI("auth", "list")
	if err != nil {
		t.Fatalf("auth list failed: %v", err)
	}

	// Parse the list to find grant IDs (simple parsing)
	lines := strings.Split(listOut, "\n")
	var grants []string
	for _, line := range lines {
		// Skip header and empty lines
		if strings.Contains(line, "GRANT ID") || strings.TrimSpace(line) == "" {
			continue
		}
		// Extract grant ID (first column)
		fields := strings.Fields(line)
		if len(fields) > 0 && strings.Contains(fields[0], "-") {
			grants = append(grants, fields[0])
		}
	}

	if len(grants) < 1 {
		t.Skip("Need at least 1 grant for switch test")
	}

	// Get current default grant
	whoamiOut, _, err := runCLI("auth", "whoami")
	var currentGrant string
	if err == nil {
		// Extract current grant ID from whoami output
		for _, line := range strings.Split(whoamiOut, "\n") {
			if strings.Contains(line, "Grant ID:") {
				parts := strings.Split(line, ":")
				if len(parts) > 1 {
					currentGrant = strings.TrimSpace(parts[1])
					break
				}
			}
		}
	}

	// Switch to first available grant
	targetGrant := grants[0]

	stdout, stderr, err := runCLI("auth", "switch", targetGrant)

	if err != nil {
		t.Fatalf("auth switch failed: %v\nstderr: %s", err, stderr)
	}

	// Should show success message
	lowerOut := strings.ToLower(stdout)
	if !strings.Contains(lowerOut, "switched") && !strings.Contains(lowerOut, "default") {
		t.Errorf("Expected switch confirmation in output, got: %s", stdout)
	}

	t.Logf("auth switch output:\n%s", stdout)

	// Verify the switch by checking whoami
	whoamiAfter, _, err := runCLI("auth", "whoami")
	if err == nil {
		t.Logf("auth whoami after switch:\n%s", whoamiAfter)
	}

	// Cleanup: Switch back to original grant if we had one
	if currentGrant != "" && currentGrant != targetGrant {
		t.Cleanup(func() {
			_, _, _ = runCLI("auth", "switch", currentGrant)
		})
	}
}

func TestCLI_AuthSwitch_InvalidGrant(t *testing.T) {
	// GUARD: Only run if both environment variables are set
	if os.Getenv("NYLAS_TEST_DELETE") != "true" || os.Getenv("NYLAS_TEST_AUTH_LOGOUT") != "true" {
		t.Skip("Skipping auth test - requires NYLAS_TEST_DELETE=true and NYLAS_TEST_AUTH_LOGOUT=true")
	}
	skipIfMissingCreds(t)

	_, stderr, err := runCLI("auth", "switch", "invalid-grant-id-12345")

	if err == nil {
		t.Error("Expected error for invalid grant ID, but command succeeded")
	}

	t.Logf("auth switch invalid grant error: %s", stderr)
}

// =============================================================================
// AUTH LOGOUT COMMAND TESTS (Phase 1.2) - GUARDED
// =============================================================================

func TestCLI_AuthLogoutHelp(t *testing.T) {
	// GUARD: Only run if both environment variables are set
	if os.Getenv("NYLAS_TEST_DELETE") != "true" || os.Getenv("NYLAS_TEST_AUTH_LOGOUT") != "true" {
		t.Skip("Skipping auth test - requires NYLAS_TEST_DELETE=true and NYLAS_TEST_AUTH_LOGOUT=true")
	}
	if testBinary == "" {
		t.Skip("CLI binary not found")
	}

	stdout, stderr, err := runCLI("auth", "logout", "--help")

	if err != nil {
		t.Fatalf("auth logout --help failed: %v\nstderr: %s", err, stderr)
	}

	// Should show logout command usage
	if !strings.Contains(stdout, "logout") || !strings.Contains(stdout, "Revoke") {
		t.Errorf("Expected logout command description in help, got: %s", stdout)
	}

	t.Logf("auth logout --help output:\n%s", stdout)
}

func TestCLI_AuthLogout(t *testing.T) {
	// GUARD: Only run if both environment variables are set
	if os.Getenv("NYLAS_TEST_DELETE") != "true" || os.Getenv("NYLAS_TEST_AUTH_LOGOUT") != "true" {
		t.Skip("Skipping auth test - requires NYLAS_TEST_DELETE=true and NYLAS_TEST_AUTH_LOGOUT=true")
	}
	skipIfMissingCreds(t)

	// ADDITIONAL SAFETY: Verify we have multiple grants before logout
	listOut, _, err := runCLI("auth", "list")
	if err != nil {
		t.Fatalf("auth list failed: %v", err)
	}

	grantCount := strings.Count(listOut, "✓ valid")
	if grantCount < 2 {
		t.Skip("Need at least 2 grants to safely test logout (to avoid removing all grants)")
	}

	t.Log("⚠️  WARNING: Running auth logout test - this will remove the current grant from local config")

	// Run logout
	stdout, stderr, err := runCLI("auth", "logout")

	if err != nil {
		t.Fatalf("auth logout failed: %v\nstderr: %s", err, stderr)
	}

	// Should show confirmation
	lowerOut := strings.ToLower(stdout)
	if !strings.Contains(lowerOut, "revoked") && !strings.Contains(lowerOut, "logout") && !strings.Contains(lowerOut, "removed") {
		t.Errorf("Expected logout confirmation in output, got: %s", stdout)
	}

	t.Logf("auth logout output:\n%s", stdout)

	// Note: You'll need to manually re-add the grant after this test
	t.Log("⚠️  Note: You may need to run 'nylas auth add <grant-id>' to re-add the grant")
}
