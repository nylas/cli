//go:build integration

package integration

import (
	"strings"
	"testing"
)

// =============================================================================
// EMAIL LIST COMMAND TESTS
// =============================================================================
func TestCLI_ThreadsHelp(t *testing.T) {
	if testBinary == "" {
		t.Skip("CLI binary not found")
	}

	stdout, stderr, err := runCLI("email", "threads", "--help")

	if err != nil {
		t.Fatalf("email threads --help failed: %v\nstderr: %s", err, stderr)
	}

	// Should show threads subcommands including search
	if !strings.Contains(stdout, "search") {
		t.Errorf("Expected 'search' subcommand in threads help, got: %s", stdout)
	}
	if !strings.Contains(stdout, "list") {
		t.Errorf("Expected 'list' subcommand in threads help, got: %s", stdout)
	}

	t.Logf("email threads help output:\n%s", stdout)
}

func TestCLI_ThreadsSearchHelp(t *testing.T) {
	if testBinary == "" {
		t.Skip("CLI binary not found")
	}

	stdout, stderr, err := runCLI("email", "threads", "search", "--help")

	if err != nil {
		t.Fatalf("email threads search --help failed: %v\nstderr: %s", err, stderr)
	}

	// Should show search flags
	if !strings.Contains(stdout, "--from") {
		t.Errorf("Expected '--from' flag in threads search help, got: %s", stdout)
	}
	if !strings.Contains(stdout, "--to") {
		t.Errorf("Expected '--to' flag in threads search help, got: %s", stdout)
	}
	if !strings.Contains(stdout, "--subject") {
		t.Errorf("Expected '--subject' flag in threads search help, got: %s", stdout)
	}
	if !strings.Contains(stdout, "--has-attachment") {
		t.Errorf("Expected '--has-attachment' flag in threads search help, got: %s", stdout)
	}
	if !strings.Contains(stdout, "--unread") {
		t.Errorf("Expected '--unread' flag in threads search help, got: %s", stdout)
	}
	if !strings.Contains(stdout, "--starred") {
		t.Errorf("Expected '--starred' flag in threads search help, got: %s", stdout)
	}

	t.Logf("email threads search help output:\n%s", stdout)
}

func TestCLI_ThreadsSearch(t *testing.T) {
	skipIfMissingCreds(t)

	// Thread search uses filters (no full-text query), so we search by subject
	stdout, stderr, err := runCLI("email", "threads", "search", testGrantID, "--subject", "test", "--limit", "3")

	if err != nil {
		t.Fatalf("email threads search failed: %v\nstderr: %s", err, stderr)
	}

	// Should show results or "No threads found"
	if !strings.Contains(stdout, "Found") && !strings.Contains(stdout, "No threads found") {
		t.Errorf("Expected search results output, got: %s", stdout)
	}

	t.Logf("email threads search output:\n%s", stdout)
}

func TestCLI_ThreadsSearch_WithFilters(t *testing.T) {
	skipIfMissingCreds(t)

	tests := []struct {
		name string
		args []string
	}{
		{"with-from", []string{"email", "threads", "search", testGrantID, "--from", "test@example.com", "--limit", "3"}},
		{"with-subject", []string{"email", "threads", "search", testGrantID, "--subject", "test", "--limit", "3"}},
		{"unread", []string{"email", "threads", "search", testGrantID, "--unread", "--limit", "3"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stdout, stderr, err := runCLI(tt.args...)
			if err != nil {
				t.Fatalf("threads search %s failed: %v\nstderr: %s", tt.name, err, stderr)
			}
			t.Logf("threads search %s output:\n%s", tt.name, stdout)
		})
	}
}

// =============================================================================
// EMAIL DELETE COMMAND TESTS (Phase 1.1)
// =============================================================================
