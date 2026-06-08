//go:build integration

package integration

import (
	"strings"
	"testing"
)

// =============================================================================
// SLACK CHANNELS TESTS
// =============================================================================

func TestSlack_ChannelsList(t *testing.T) {
	skipIfMissingSlackCreds(t)

	tests := []struct {
		name     string
		args     []string
		contains []string
	}{
		{
			name:     "list all channels (subcommand)",
			args:     []string{"slack", "channels", "list"},
			contains: []string{}, // Just verify it runs
		},
		{
			name:     "list with limit",
			args:     []string{"slack", "channels", "list", "--limit", "5"},
			contains: []string{},
		},
		{
			name:     "list public channels only",
			args:     []string{"slack", "channels", "list", "--type", "public_channel"},
			contains: []string{},
		},
		{
			name:     "list with IDs",
			args:     []string{"slack", "channels", "list", "--id"},
			contains: []string{"[C"}, // Channel IDs start with C
		},
		{
			name:     "exclude archived",
			args:     []string{"slack", "channels", "list", "--exclude-archived"},
			contains: []string{},
		},
		{
			name:     "list all workspace channels",
			args:     []string{"slack", "channels", "list", "--all-workspace", "--limit", "5"},
			contains: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stdout, stderr, err := runSlackCLI(t, tt.args...)

			if err != nil {
				if strings.Contains(stderr, "not authenticated") {
					t.Skip("Not authenticated with Slack")
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

func TestSlack_ChannelInfo(t *testing.T) {
	skipIfMissingSlackCreds(t)

	// Test channel info command
	stdout, stderr, err := runSlackCLI(t, "slack", "channels", "info", slackUserChannel)

	if err != nil {
		if strings.Contains(stderr, "not authenticated") {
			t.Skip("Not authenticated with Slack")
		}
		if strings.Contains(stderr, "channel_not_found") {
			t.Skipf("Channel %s not found", slackUserChannel)
		}
		t.Fatalf("Command failed: %v\nstderr: %s", err, stderr)
	}

	// Should show channel details
	expectedFields := []string{"ID:", "Is Channel:", "Is Private:"}
	for _, field := range expectedFields {
		if !strings.Contains(stdout, field) {
			t.Errorf("Expected output to contain %q\nGot: %s", field, stdout)
		}
	}

	t.Logf("Output:\n%s", stdout)
}

// =============================================================================
// SLACK MESSAGES TESTS
// =============================================================================

func TestSlack_MessagesList(t *testing.T) {
	skipIfMissingSlackCreds(t)

	tests := []struct {
		name     string
		args     []string
		wantErr  bool
		contains []string
	}{
		{
			name:     "list messages from channel (subcommand)",
			args:     []string{"slack", "messages", "list", "--channel-id", slackUserChannel},
			contains: []string{}, // Just verify it runs
		},
		{
			name:     "list messages with limit",
			args:     []string{"slack", "messages", "list", "--channel-id", slackUserChannel, "--limit", "5"},
			contains: []string{},
		},
		{
			name:     "list messages with IDs",
			args:     []string{"slack", "messages", "list", "--channel-id", slackUserChannel, "--id"},
			contains: []string{}, // Should show message timestamps
		},
		{
			name:    "missing channel",
			args:    []string{"slack", "messages", "list"},
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
				if strings.Contains(stderr, "channel not found") {
					t.Skipf("Channel %s not found", slackUserChannel)
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
