//go:build integration

package integration

import (
	"strings"
	"testing"
)

// =============================================================================
// AUTH COMMAND TESTS
// =============================================================================

func TestCLI_AuthStatus(t *testing.T) {
	skipIfMissingCreds(t)

	stdout, stderr, err := runCLI("auth", "status")

	if err != nil {
		t.Fatalf("auth status failed: %v\nstderr: %s", err, stderr)
	}

	// Should contain key status information
	if !strings.Contains(stdout, "Authentication Status") {
		t.Errorf("Expected 'Authentication Status' in output, got: %s", stdout)
	}
	if !strings.Contains(stdout, "Secret Store:") {
		t.Errorf("Expected 'Secret Store:' in output, got: %s", stdout)
	}

	t.Logf("auth status output:\n%s", stdout)
}

func TestCLI_AuthList(t *testing.T) {
	skipIfMissingCreds(t)
	skipIfKeyringDisabled(t)

	stdout, stderr, err := runCLI("auth", "list")

	// This may show "No authenticated accounts" if grant isn't registered locally
	// but should not error
	if err != nil && !strings.Contains(stderr, "No authenticated accounts") {
		t.Fatalf("auth list failed: %v\nstderr: %s", err, stderr)
	}

	t.Logf("auth list output:\n%s", stdout)
}

func TestCLI_AuthWhoami(t *testing.T) {
	skipIfMissingCreds(t)
	skipIfKeyringDisabled(t)

	stdout, stderr, err := runCLI("auth", "whoami")

	// May fail if no default grant is set
	if err != nil {
		if strings.Contains(stderr, "no default grant") {
			t.Skip("No default grant set")
		}
		t.Fatalf("auth whoami failed: %v\nstderr: %s", err, stderr)
	}

	// Should show email and provider
	if !strings.Contains(stdout, "@") {
		t.Errorf("Expected email in output, got: %s", stdout)
	}

	t.Logf("auth whoami output:\n%s", stdout)
}

func TestCLI_AuthAdd(t *testing.T) {
	skipIfMissingCreds(t)
	skipIfKeyringDisabled(t)

	// Test adding with auto-detection (no --email or --provider flags)
	stdout, stderr, err := runCLI("auth", "add", testGrantID, "--default")

	if err != nil {
		t.Fatalf("auth add failed: %v\nstderr: %s", err, stderr)
	}

	// Should show success message
	if !strings.Contains(stdout, "Added grant") {
		t.Errorf("Expected 'Added grant' in output, got: %s", stdout)
	}
	if !strings.Contains(stdout, testGrantID) {
		t.Errorf("Expected grant ID in output, got: %s", stdout)
	}
	// Should auto-detect email and provider
	if !strings.Contains(stdout, "Email:") {
		t.Errorf("Expected 'Email:' in output (auto-detected), got: %s", stdout)
	}
	if !strings.Contains(stdout, "Provider:") {
		t.Errorf("Expected 'Provider:' in output (auto-detected), got: %s", stdout)
	}

	t.Logf("auth add output:\n%s", stdout)

	// Verify the grant appears in list
	listOut, _, err := runCLI("auth", "list")
	if err != nil {
		t.Fatalf("auth list after add failed: %v", err)
	}
	if !strings.Contains(listOut, testGrantID) {
		t.Errorf("Expected grant ID in auth list output, got: %s", listOut)
	}

	t.Logf("auth list after add:\n%s", listOut)
}

func TestCLI_AuthAdd_AutoDetect(t *testing.T) {
	skipIfMissingCreds(t)
	skipIfKeyringDisabled(t)

	// Test that auto-detection gets correct info from Nylas API
	stdout, stderr, err := runCLI("auth", "add", testGrantID)

	if err != nil {
		t.Fatalf("auth add auto-detect failed: %v\nstderr: %s", err, stderr)
	}

	// Should show success with auto-detected values
	if !strings.Contains(stdout, "Added grant") {
		t.Errorf("Expected 'Added grant' in output, got: %s", stdout)
	}

	// The output should contain an email with @ symbol (auto-detected)
	if !strings.Contains(stdout, "@") {
		t.Errorf("Expected auto-detected email in output, got: %s", stdout)
	}

	t.Logf("auth add auto-detect output:\n%s", stdout)
}

func TestCLI_AuthAdd_OverrideAutoDetect(t *testing.T) {
	skipIfMissingCreds(t)
	skipIfKeyringDisabled(t)

	// Test that flags can override auto-detected values
	stdout, stderr, err := runCLI("auth", "add", testGrantID,
		"--email", "override@example.com",
		"--provider", "google")

	if err != nil {
		t.Fatalf("auth add with overrides failed: %v\nstderr: %s", err, stderr)
	}

	// Should show overridden email
	if !strings.Contains(stdout, "override@example.com") {
		t.Errorf("Expected overridden email in output, got: %s", stdout)
	}
	// Should show overridden provider
	if !strings.Contains(stdout, "Google") {
		t.Errorf("Expected overridden provider 'Google' in output, got: %s", stdout)
	}

	t.Logf("auth add with overrides output:\n%s", stdout)
}

func TestCLI_AuthAdd_InvalidGrant(t *testing.T) {
	skipIfMissingCreds(t)
	skipIfKeyringDisabled(t)

	// Test adding a non-existent grant - should fail when fetching from API
	_, stderr, err := runCLI("auth", "add", "invalid-grant-id-12345")

	if err == nil {
		t.Error("Expected error for invalid grant ID, but command succeeded")
	}

	// Should show fetch error
	if !strings.Contains(stderr, "not valid") && !strings.Contains(stderr, "not found") && !strings.Contains(stderr, "failed to fetch") {
		t.Logf("Error output for invalid grant: %s", stderr)
	}
}

func TestCLI_AuthAdd_ProviderOverride(t *testing.T) {
	skipIfMissingCreds(t)
	skipIfKeyringDisabled(t)

	// Test that provider flag can override auto-detected provider
	providers := []string{"google", "microsoft"}

	for _, provider := range providers {
		t.Run(provider, func(t *testing.T) {
			stdout, stderr, err := runCLI("auth", "add", testGrantID,
				"--provider", provider)

			if err != nil {
				t.Fatalf("auth add with provider %s failed: %v\nstderr: %s", provider, err, stderr)
			}

			if !strings.Contains(stdout, "Added grant") {
				t.Errorf("Expected 'Added grant' in output, got: %s", stdout)
			}

			t.Logf("auth add with provider %s output:\n%s", provider, stdout)
		})
	}
}

func TestCLI_AuthHelp(t *testing.T) {
	skipIfMissingCreds(t)

	stdout, stderr, err := runCLI("auth", "--help")

	if err != nil {
		t.Fatalf("auth --help failed: %v\nstderr: %s", err, stderr)
	}

	// Should show auth subcommands
	if !strings.Contains(stdout, "login") {
		t.Errorf("Expected 'login' in auth help, got: %s", stdout)
	}
	if !strings.Contains(stdout, "logout") {
		t.Errorf("Expected 'logout' in auth help, got: %s", stdout)
	}
	if !strings.Contains(stdout, "status") {
		t.Errorf("Expected 'status' in auth help, got: %s", stdout)
	}
	if !strings.Contains(stdout, "show") {
		t.Errorf("Expected 'show' in auth help, got: %s", stdout)
	}

	t.Logf("auth help output:\n%s", stdout)
}

// =============================================================================
// AUTH SHOW COMMAND TESTS (Phase 3)
// =============================================================================

func TestCLI_AuthShowHelp(t *testing.T) {
	if testBinary == "" {
		t.Skip("CLI binary not found")
	}

	stdout, stderr, err := runCLI("auth", "show", "--help")

	if err != nil {
		t.Fatalf("auth show --help failed: %v\nstderr: %s", err, stderr)
	}

	// Should show help for show command
	if !strings.Contains(stdout, "grant") {
		t.Errorf("Expected 'grant' in show help, got: %s", stdout)
	}
	if !strings.Contains(stdout, "detailed") || !strings.Contains(stdout, "information") {
		t.Errorf("Expected detailed information description in help, got: %s", stdout)
	}

	t.Logf("auth show help output:\n%s", stdout)
}

func TestCLI_AuthShow(t *testing.T) {
	skipIfMissingCreds(t)
	// Note: auth show with explicit grant ID works with env vars (no keyring needed)

	stdout, stderr, err := runCLI("auth", "show", testGrantID)

	if err != nil {
		t.Fatalf("auth show failed: %v\nstderr: %s", err, stderr)
	}

	// Should show grant details
	if !strings.Contains(stdout, "Grant ID:") {
		t.Errorf("Expected 'Grant ID:' in output, got: %s", stdout)
	}
	if !strings.Contains(stdout, "Email:") {
		t.Errorf("Expected 'Email:' in output, got: %s", stdout)
	}
	if !strings.Contains(stdout, "Provider:") {
		t.Errorf("Expected 'Provider:' in output, got: %s", stdout)
	}
	if !strings.Contains(stdout, "Status:") {
		t.Errorf("Expected 'Status:' in output, got: %s", stdout)
	}

	t.Logf("auth show output:\n%s", stdout)
}

func TestCLI_AuthShow_InvalidGrant(t *testing.T) {
	skipIfMissingCreds(t)
	// Note: auth show with explicit grant ID works with env vars (no keyring needed)

	_, stderr, err := runCLI("auth", "show", "invalid-grant-id-12345")

	if err == nil {
		t.Error("Expected error for invalid grant ID, but command succeeded")
	}

	// Should show error message
	t.Logf("auth show invalid grant error: %s", stderr)
}
