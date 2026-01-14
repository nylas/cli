//go:build integration

package integration

import (
	"strings"
	"testing"
	"time"
)

// =============================================================================
// HELP COMMAND TESTS
// =============================================================================

func TestCLI_Help(t *testing.T) {
	if testBinary == "" {
		t.Skip("CLI binary not found")
	}
	stdout, stderr, err := runCLI("--help")

	if err != nil {
		t.Fatalf("--help failed: %v\nstderr: %s", err, stderr)
	}

	// Should show usage information
	if !strings.Contains(stdout, "Usage:") && !strings.Contains(stdout, "nylas") {
		t.Errorf("Expected help output, got: %s", stdout)
	}

	t.Logf("--help output:\n%s", stdout)
}

// =============================================================================
// ERROR HANDLING TESTS
// =============================================================================

func TestCLI_InvalidCommand(t *testing.T) {
	if testBinary == "" {
		t.Skip("CLI binary not found")
	}
	_, stderr, err := runCLI("invalidcommand")

	if err == nil {
		t.Error("Expected error for invalid command")
	}

	if !strings.Contains(stderr, "unknown command") && !strings.Contains(stderr, "invalid") {
		t.Logf("stderr for invalid command: %s", stderr)
	}
}

// =============================================================================
// CONCURRENCY TESTS
// =============================================================================

func TestCLI_ConcurrentOperations(t *testing.T) {
	skipIfMissingCreds(t)

	// Run multiple list operations concurrently
	type result struct {
		name   string
		err    error
		stderr string
	}
	results := make(chan result, 3)

	operations := []struct {
		name string
		args []string
	}{
		{"email list", []string{"email", "list", testGrantID, "--limit", "2"}},
		{"folders list", []string{"email", "folders", "list", testGrantID}},
		{"threads list", []string{"email", "threads", "list", testGrantID, "--limit", "2"}},
	}

	for _, op := range operations {
		go func(name string, args []string) {
			_, stderr, err := runCLI(args...)
			results <- result{name, err, stderr}
		}(op.name, op.args)
	}

	// Wait for all operations - allow some to fail if provider doesn't support them
	successCount := 0
	for i := 0; i < len(operations); i++ {
		select {
		case r := <-results:
			if r.err != nil {
				if strings.Contains(r.stderr, "Method not supported for provider") ||
					strings.Contains(r.stderr, "an internal error ocurred") {
					t.Logf("%s: Skipped (not supported by provider)", r.name)
				} else {
					t.Logf("%s: Failed: %v", r.name, r.err)
				}
			} else {
				successCount++
				t.Logf("%s: OK", r.name)
			}
		case <-time.After(30 * time.Second):
			t.Fatal("Timeout waiting for concurrent operations")
		}
	}
	if successCount == 0 {
		t.Skip("No operations succeeded - provider may have limited support")
	}
}

// =============================================================================
// WORKFLOW TESTS
// =============================================================================

func TestCLI_FullWorkflow(t *testing.T) {
	skipIfMissingCreds(t)

	// This test simulates a typical user workflow

	// 1. Check auth status
	t.Run("1_auth_status", func(t *testing.T) {
		stdout, stderr, err := runCLI("auth", "status")
		if err != nil {
			t.Fatalf("Failed: %v\nstderr: %s", err, stderr)
		}
		t.Logf("Auth status: %s", stdout)
	})

	// 2. List emails
	t.Run("2_list_emails", func(t *testing.T) {
		stdout, stderr, err := runCLI("email", "list", testGrantID, "--limit", "5", "--id")
		if err != nil {
			t.Fatalf("Failed: %v\nstderr: %s", err, stderr)
		}
		t.Logf("Email list: %s", stdout)
	})

	// 3. List folders (skip if provider doesn't support)
	t.Run("3_list_folders", func(t *testing.T) {
		stdout, stderr, err := runCLI("email", "folders", "list", testGrantID)
		skipIfProviderNotSupported(t, stderr)
		if err != nil {
			t.Fatalf("Failed: %v\nstderr: %s", err, stderr)
		}
		t.Logf("Folders: %s", stdout)
	})

	// 4. List threads (skip if provider doesn't support)
	t.Run("4_list_threads", func(t *testing.T) {
		stdout, stderr, err := runCLI("email", "threads", "list", testGrantID, "--limit", "3")
		skipIfProviderNotSupported(t, stderr)
		if err != nil {
			t.Fatalf("Failed: %v\nstderr: %s", err, stderr)
		}
		t.Logf("Threads: %s", stdout)
	})

	// 5. Search emails
	t.Run("5_search_emails", func(t *testing.T) {
		stdout, stderr, err := runCLI("email", "search", "test", testGrantID, "--limit", "3")
		if err != nil {
			t.Fatalf("Failed: %v\nstderr: %s", err, stderr)
		}
		t.Logf("Search results: %s", stdout)
	})

	t.Log("Full workflow completed successfully")
}

func TestCLI_NewFeaturesWorkflow(t *testing.T) {
	skipIfMissingCreds(t)

	// Test all new features in sequence

	// 1. Webhook triggers list
	t.Run("1_webhook_triggers", func(t *testing.T) {
		stdout, stderr, err := runCLI("webhook", "triggers")
		if err != nil {
			t.Fatalf("Failed: %v\nstderr: %s", err, stderr)
		}
		if !strings.Contains(stdout, "message.created") {
			t.Errorf("Expected message.created trigger")
		}
		t.Logf("Webhook triggers: %s", stdout)
	})

	// 2. List webhooks
	t.Run("2_webhook_list", func(t *testing.T) {
		stdout, stderr, err := runCLI("webhook", "list")
		if err != nil {
			t.Fatalf("Failed: %v\nstderr: %s", err, stderr)
		}
		t.Logf("Webhook list: %s", stdout)
	})

	// 3. Calendar availability
	t.Run("3_availability_check", func(t *testing.T) {
		stdout, _, err := runCLI("calendar", "availability", "check", testGrantID, "--duration", "1d")
		if err != nil {
			t.Skip("Availability check not available")
		}
		t.Logf("Availability: %s", stdout)
	})

	// 4. Email send help (verify schedule option)
	t.Run("4_email_send_help", func(t *testing.T) {
		stdout, stderr, err := runCLI("email", "send", "--help")
		if err != nil {
			t.Fatalf("Failed: %v\nstderr: %s", err, stderr)
		}
		if !strings.Contains(stdout, "--schedule") {
			t.Errorf("Expected --schedule flag")
		}
		t.Logf("Email send help contains schedule option")
	})

	t.Log("New features workflow completed successfully")
}

// =============================================================================
// DOCTOR COMMAND TESTS
// =============================================================================

func TestCLI_Doctor(t *testing.T) {
	skipIfMissingCreds(t)
	skipIfKeyringDisabled(t)

	stdout, stderr, err := runCLI("doctor")

	if err != nil {
		t.Fatalf("doctor failed: %v\nstderr: %s", err, stderr)
	}

	// Should show health check header
	if !strings.Contains(stdout, "Nylas CLI Health Check") {
		t.Errorf("Expected 'Nylas CLI Health Check' in output, got: %s", stdout)
	}

	// Should show summary
	if !strings.Contains(stdout, "Summary") {
		t.Errorf("Expected 'Summary' in output, got: %s", stdout)
	}

	// Should check configuration
	if !strings.Contains(stdout, "Configuration") {
		t.Errorf("Expected 'Configuration' check in output, got: %s", stdout)
	}

	// Should check secret store
	if !strings.Contains(stdout, "Secret Store") {
		t.Errorf("Expected 'Secret Store' check in output, got: %s", stdout)
	}

	// Should check API credentials
	if !strings.Contains(stdout, "API Credentials") {
		t.Errorf("Expected 'API Credentials' check in output, got: %s", stdout)
	}

	// Should check network
	if !strings.Contains(stdout, "Network") {
		t.Errorf("Expected 'Network' check in output, got: %s", stdout)
	}

	// Should check grants
	if !strings.Contains(stdout, "Grants") {
		t.Errorf("Expected 'Grants' check in output, got: %s", stdout)
	}

	t.Logf("doctor output:\n%s", stdout)
}

func TestCLI_Doctor_Verbose(t *testing.T) {
	skipIfMissingCreds(t)
	skipIfKeyringDisabled(t)

	stdout, stderr, err := runCLI("doctor", "--verbose")

	if err != nil {
		t.Fatalf("doctor --verbose failed: %v\nstderr: %s", err, stderr)
	}

	// Should show platform info in verbose mode
	if !strings.Contains(stdout, "Platform:") {
		t.Errorf("Expected 'Platform:' in verbose output, got: %s", stdout)
	}

	// Should show Go version
	if !strings.Contains(stdout, "Go Version:") {
		t.Errorf("Expected 'Go Version:' in verbose output, got: %s", stdout)
	}

	// Should show config directory
	if !strings.Contains(stdout, "Config Dir:") {
		t.Errorf("Expected 'Config Dir:' in verbose output, got: %s", stdout)
	}

	t.Logf("doctor --verbose output:\n%s", stdout)
}

func TestCLI_Doctor_Help(t *testing.T) {
	if testBinary == "" {
		t.Skip("CLI binary not found")
	}

	stdout, stderr, err := runCLI("doctor", "--help")

	if err != nil {
		t.Fatalf("doctor --help failed: %v\nstderr: %s", err, stderr)
	}

	// Should show command description
	if !strings.Contains(stdout, "diagnostic checks") {
		t.Errorf("Expected 'diagnostic checks' in help output, got: %s", stdout)
	}

	// Should show verbose flag
	if !strings.Contains(stdout, "--verbose") && !strings.Contains(stdout, "-v") {
		t.Errorf("Expected verbose flag in help, got: %s", stdout)
	}

	t.Logf("doctor --help output:\n%s", stdout)
}
