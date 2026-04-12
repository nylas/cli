package notetaker

import (
	"testing"

	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/cli/testutil"
	"github.com/nylas/cli/internal/domain"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

// executeCommand executes a command and captures its output.
func executeCommand(root *cobra.Command, args ...string) (string, string, error) {
	return testutil.ExecuteCommand(root, args...)
}

func TestNewNotetakerCmd(t *testing.T) {
	cmd := NewNotetakerCmd()

	t.Run("command_name", func(t *testing.T) {
		assert.Equal(t, "notetaker", cmd.Use)
	})

	t.Run("has_aliases", func(t *testing.T) {
		assert.Contains(t, cmd.Aliases, "nt")
		assert.Contains(t, cmd.Aliases, "bot")
	})

	t.Run("has_short_description", func(t *testing.T) {
		assert.NotEmpty(t, cmd.Short)
		assert.Contains(t, cmd.Short, "Notetaker")
	})

	t.Run("has_long_description", func(t *testing.T) {
		assert.NotEmpty(t, cmd.Long)
		assert.Contains(t, cmd.Long, "recording")
		assert.Contains(t, cmd.Long, "transcription")
	})

	t.Run("has_subcommands", func(t *testing.T) {
		subcommands := cmd.Commands()
		assert.NotEmpty(t, subcommands)
	})

	t.Run("has_required_subcommands", func(t *testing.T) {
		expectedCmds := []string{"list", "show", "create", "delete", "media"}

		cmdMap := make(map[string]bool)
		for _, sub := range cmd.Commands() {
			cmdMap[sub.Name()] = true
		}

		for _, expected := range expectedCmds {
			assert.True(t, cmdMap[expected], "Missing expected subcommand: %s", expected)
		}
	})
}

func TestListCommand(t *testing.T) {
	cmd := newListCmd()

	t.Run("command_name", func(t *testing.T) {
		assert.Equal(t, "list [grant-id]", cmd.Use)
	})

	t.Run("has_ls_alias", func(t *testing.T) {
		assert.Contains(t, cmd.Aliases, "ls")
	})

	t.Run("has_limit_flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("limit")
		assert.NotNil(t, flag)
		assert.Equal(t, "20", flag.DefValue)
	})

	t.Run("has_state_flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("state")
		assert.NotNil(t, flag)
	})

	t.Run("has_json_flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("json")
		assert.NotNil(t, flag)
	})
}

func TestShowCommand(t *testing.T) {
	cmd := newShowCmd()

	t.Run("command_name", func(t *testing.T) {
		assert.Equal(t, "show <notetaker-id> [grant-id]", cmd.Use)
	})

	t.Run("has_short_description", func(t *testing.T) {
		assert.NotEmpty(t, cmd.Short)
		assert.Contains(t, cmd.Short, "notetaker")
	})

	t.Run("has_json_flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("json")
		assert.NotNil(t, flag)
	})
}

func TestCreateCommand(t *testing.T) {
	cmd := newCreateCmd()

	t.Run("command_name", func(t *testing.T) {
		assert.Equal(t, "create [grant-id]", cmd.Use)
	})

	t.Run("has_short_description", func(t *testing.T) {
		assert.NotEmpty(t, cmd.Short)
		assert.Contains(t, cmd.Short, "notetaker")
	})

	t.Run("has_meeting_link_flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("meeting-link")
		assert.NotNil(t, flag)
	})

	t.Run("has_join_time_flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("join-time")
		assert.NotNil(t, flag)
	})

	t.Run("has_bot_name_flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("bot-name")
		assert.NotNil(t, flag)
	})

	t.Run("has_json_flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("json")
		assert.NotNil(t, flag)
	})
}

func TestDeleteCommand(t *testing.T) {
	cmd := newDeleteCmd()

	t.Run("command_name", func(t *testing.T) {
		assert.Equal(t, "delete <notetaker-id> [grant-id]", cmd.Use)
	})

	t.Run("has_rm_alias", func(t *testing.T) {
		assert.Contains(t, cmd.Aliases, "rm")
	})

	t.Run("has_cancel_alias", func(t *testing.T) {
		assert.Contains(t, cmd.Aliases, "cancel")
	})

	t.Run("has_force_flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("force")
		assert.NotNil(t, flag)
	})
}

func TestMediaCommand(t *testing.T) {
	cmd := newMediaCmd()

	t.Run("command_name", func(t *testing.T) {
		assert.Equal(t, "media <notetaker-id> [grant-id]", cmd.Use)
	})

	t.Run("has_json_flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("json")
		assert.NotNil(t, flag)
	})

	t.Run("has_long_description_with_info", func(t *testing.T) {
		assert.Contains(t, cmd.Long, "Recording")
		assert.Contains(t, cmd.Long, "Transcript")
	})
}

func TestFormatState(t *testing.T) {
	tests := []struct {
		state    string
		expected string
	}{
		{domain.NotetakerStateScheduled, "scheduled"},
		{domain.NotetakerStateConnecting, "connecting"},
		{domain.NotetakerStateWaitingForEntry, "waiting"},
		{domain.NotetakerStateAttending, "attending"},
		{domain.NotetakerStateMediaProcessing, "processing"},
		{domain.NotetakerStateComplete, "complete"},
		{domain.NotetakerStateCancelled, "cancelled"},
		{domain.NotetakerStateFailed, "failed"},
		{"unknown", "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.state, func(t *testing.T) {
			result := formatState(tt.state)
			// Check that the result contains the expected text
			// (color codes may be added)
			assert.Contains(t, result, tt.expected)
		})
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		input    string
		length   int
		expected string
	}{
		{"hello", 10, "hello"},
		{"hello world this is a long string", 15, "hello world ..."},
		{"short", 10, "short"},
		{"exactly10c", 10, "exactly10c"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := common.Truncate(tt.input, tt.length)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestNotetakerCommandHelp(t *testing.T) {
	cmd := NewNotetakerCmd()
	stdout, _, err := executeCommand(cmd, "--help")

	assert.NoError(t, err)

	expectedStrings := []string{
		"notetaker",
		"list",
		"show",
		"create",
		"delete",
		"media",
	}

	for _, expected := range expectedStrings {
		assert.Contains(t, stdout, expected, "Help output should contain %q", expected)
	}
}

func TestNotetakerListHelp(t *testing.T) {
	cmd := NewNotetakerCmd()
	stdout, _, err := executeCommand(cmd, "list", "--help")

	assert.NoError(t, err)
	assert.Contains(t, stdout, "list")
	assert.Contains(t, stdout, "--limit")
	assert.Contains(t, stdout, "--state")
	assert.Contains(t, stdout, "--json")
}

func TestNotetakerCreateHelp(t *testing.T) {
	cmd := NewNotetakerCmd()
	stdout, _, err := executeCommand(cmd, "create", "--help")

	assert.NoError(t, err)
	assert.Contains(t, stdout, "create")
	assert.Contains(t, stdout, "--meeting-link")
	assert.Contains(t, stdout, "--join-time")
	assert.Contains(t, stdout, "--bot-name")
}

func TestNotetakerMediaHelp(t *testing.T) {
	cmd := NewNotetakerCmd()
	stdout, _, err := executeCommand(cmd, "media", "--help")

	assert.NoError(t, err)
	assert.Contains(t, stdout, "media")
	assert.Contains(t, stdout, "--json")
	assert.Contains(t, stdout, "recording")
	assert.Contains(t, stdout, "transcript")
}
