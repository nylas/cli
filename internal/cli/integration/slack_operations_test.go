//go:build integration

package integration

import (
	"os"
	"strings"
	"testing"
	"time"
)

// =============================================================================
// SLACK SEARCH TESTS
// =============================================================================

func TestSlack_Search(t *testing.T) {
	skipIfMissingSlackCreds(t)

	tests := []struct {
		name     string
		args     []string
		wantErr  bool
		contains []string
	}{
		{
			name:     "search messages",
			args:     []string{"slack", "search", "--query", "test"},
			contains: []string{}, // Just verify it runs (may return no results)
		},
		{
			name:     "search with limit",
			args:     []string{"slack", "search", "--query", "hello", "--limit", "5"},
			contains: []string{},
		},
		{
			name:    "search missing query",
			args:    []string{"slack", "search"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stdout, stderr, err := runSlackCLI(t, tt.args...)

			if tt.wantErr {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				if strings.Contains(stderr, "not authenticated") {
					t.Skip("Not authenticated with Slack")
				}
				// Search may fail with missing_scope if user token doesn't have search:read
				if strings.Contains(stderr, "missing_scope") {
					t.Skip("Token missing search:read scope")
				}
				t.Fatalf("Command failed: %v\nstderr: %s", err, stderr)
			}

			for _, expected := range tt.contains {
				if !strings.Contains(stdout, expected) {
					t.Errorf("Expected output to contain %q\nGot: %s", expected, stdout)
				}
			}

			t.Logf("Output:\n%s", stdout)
		})
	}
}

// =============================================================================
// SLACK SEND TESTS (Read-only by default)
// =============================================================================

func TestSlack_Send_DryRun(t *testing.T) {
	skipIfMissingSlackCreds(t)

	// This test verifies the send command validates inputs without actually sending
	// We expect an error because we're not confirming (no --yes flag and no stdin)

	tests := []struct {
		name    string
		args    []string
		wantErr bool
	}{
		{
			name:    "send requires channel",
			args:    []string{"slack", "send", "--text", "test"},
			wantErr: true,
		},
		{
			name:    "send requires text",
			args:    []string{"slack", "send", "--channel", slackUserChannel},
			wantErr: true,
		},
		{
			name:    "reply requires thread",
			args:    []string{"slack", "reply", "--channel", slackUserChannel, "--text", "test"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, stderr, err := runSlackCLI(t, tt.args...)

			if tt.wantErr && err == nil {
				t.Errorf("Expected error but got none. stderr: %s", stderr)
			}
			if !tt.wantErr && err != nil {
				t.Errorf("Unexpected error: %v\nstderr: %s", err, stderr)
			}
		})
	}
}

// TestSlack_SendMessage actually sends a message. Only runs if SLACK_TEST_SEND=true.
func TestSlack_SendMessage(t *testing.T) {
	skipIfMissingSlackCreds(t)

	if os.Getenv("SLACK_TEST_SEND") != "true" {
		t.Skip("SLACK_TEST_SEND not set to 'true' - skipping actual send test")
	}

	testMessage := "Integration test message from nylas CLI at " + time.Now().Format(time.RFC3339)

	stdout, stderr, err := runSlackCLI(t,
		"slack", "send",
		"--channel", slackUserChannel,
		"--text", testMessage,
		"--yes", // Skip confirmation
	)

	if err != nil {
		if strings.Contains(stderr, "not authenticated") {
			t.Skip("Not authenticated with Slack")
		}
		if strings.Contains(stderr, "channel not found") {
			t.Skipf("Channel %s not found", slackUserChannel)
		}
		t.Fatalf("Send failed: %v\nstderr: %s", err, stderr)
	}

	// Should confirm message was sent
	if !strings.Contains(stdout, "Message sent") && !strings.Contains(stdout, "ID:") {
		t.Errorf("Expected send confirmation, got: %s", stdout)
	}

	t.Logf("Send output:\n%s", stdout)
}
