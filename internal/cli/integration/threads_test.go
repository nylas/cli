//go:build integration

package integration

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/nylas/cli/internal/domain"
)

// =============================================================================
// THREADS COMMAND TESTS
// =============================================================================

func TestCLI_ThreadsList(t *testing.T) {
	skipIfMissingCreds(t)

	stdout, stderr, err := runCLI("email", "threads", "list", testGrantID, "--limit", "5")
	skipIfProviderNotSupported(t, stderr)

	if err != nil {
		t.Fatalf("threads list failed: %v\nstderr: %s", err, stderr)
	}

	// Should show thread count or "No threads found"
	if !strings.Contains(stdout, "Found") && !strings.Contains(stdout, "No threads found") {
		t.Errorf("Expected threads list output, got: %s", stdout)
	}

	t.Logf("threads list output:\n%s", stdout)
}

func TestCLI_ThreadsList_WithFilters(t *testing.T) {
	skipIfMissingCreds(t)

	tests := []struct {
		name string
		args []string
	}{
		{"unread", []string{"email", "threads", "list", testGrantID, "--unread", "--limit", "3"}},
		{"starred", []string{"email", "threads", "list", testGrantID, "--starred", "--limit", "3"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stdout, stderr, err := runCLI(tt.args...)
			skipIfProviderNotSupported(t, stderr)
			if err != nil {
				t.Fatalf("threads list %s failed: %v\nstderr: %s", tt.name, err, stderr)
			}
			t.Logf("threads list %s output:\n%s", tt.name, stdout)
		})
	}
}

func TestCLI_ThreadsShow(t *testing.T) {
	skipIfMissingCreds(t)

	// Get a thread ID
	client := getTestClient()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	threads, err := client.GetThreads(ctx, testGrantID, &domain.ThreadQueryParams{Limit: 1})
	if err != nil {
		if strings.Contains(err.Error(), "Method not supported for provider") ||
			strings.Contains(err.Error(), "an internal error ocurred") {
			t.Skipf("Provider does not support threads: %v", err)
		}
		t.Fatalf("Failed to get threads: %v", err)
	}
	if len(threads) == 0 {
		t.Skip("No threads available for show test")
	}

	threadID := threads[0].ID

	stdout, stderr, err := runCLI("email", "threads", "show", threadID, testGrantID)

	if err != nil {
		t.Fatalf("threads show failed: %v\nstderr: %s", err, stderr)
	}

	// Should show thread details
	if !strings.Contains(stdout, "Thread:") {
		t.Errorf("Expected 'Thread:' in output, got: %s", stdout)
	}
	if !strings.Contains(stdout, "Participants:") {
		t.Errorf("Expected 'Participants:' in output, got: %s", stdout)
	}

	t.Logf("threads show output:\n%s", stdout)
}
