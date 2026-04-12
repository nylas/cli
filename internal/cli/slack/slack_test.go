//go:build !integration

package slack

import (
	"testing"

	"github.com/nylas/cli/internal/cli/testutil"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

// executeCommand executes a command and captures its output.
func executeCommand(root *cobra.Command, args ...string) (string, string, error) {
	return testutil.ExecuteCommand(root, args...)
}

func TestNewSlackCmd(t *testing.T) {
	cmd := NewSlackCmd()

	t.Run("command_name", func(t *testing.T) {
		assert.Equal(t, "slack", cmd.Use)
	})

	t.Run("has_alias", func(t *testing.T) {
		assert.Contains(t, cmd.Aliases, "sl")
	})

	t.Run("has_short_description", func(t *testing.T) {
		assert.NotEmpty(t, cmd.Short)
		assert.Contains(t, cmd.Short, "Slack")
	})

	t.Run("has_long_description", func(t *testing.T) {
		assert.NotEmpty(t, cmd.Long)
	})

	t.Run("has_subcommands", func(t *testing.T) {
		subcommands := cmd.Commands()
		assert.NotEmpty(t, subcommands, "Slack command should have subcommands")
	})

	t.Run("has_required_subcommands", func(t *testing.T) {
		expectedCmds := []string{"auth", "channels", "messages", "files", "send", "reply", "users", "search"}

		cmdMap := make(map[string]bool)
		for _, sub := range cmd.Commands() {
			cmdMap[sub.Name()] = true
		}

		for _, expected := range expectedCmds {
			assert.True(t, cmdMap[expected], "Missing expected subcommand: %s", expected)
		}
	})
}

func TestSlackCommandHelp(t *testing.T) {
	cmd := NewSlackCmd()
	stdout, _, err := executeCommand(cmd, "--help")

	assert.NoError(t, err)

	// Check that help contains expected content
	expectedStrings := []string{
		"slack",
		"Slack",
		"auth",
		"channels",
		"messages",
		"send",
		"search",
	}

	for _, expected := range expectedStrings {
		assert.Contains(t, stdout, expected, "Help output should contain %q", expected)
	}
}

func TestAuthCommand(t *testing.T) {
	cmd := newAuthCmd()

	t.Run("command_name", func(t *testing.T) {
		assert.Equal(t, "auth", cmd.Use)
	})

	t.Run("has_subcommands", func(t *testing.T) {
		subcommands := cmd.Commands()
		assert.NotEmpty(t, subcommands)

		cmdNames := make([]string, 0, len(subcommands))
		for _, sub := range subcommands {
			cmdNames = append(cmdNames, sub.Name())
		}

		// Should have set, status, and remove subcommands
		assert.Contains(t, cmdNames, "set")
		assert.Contains(t, cmdNames, "status")
		assert.Contains(t, cmdNames, "remove")
	})
}

func TestChannelsCommand(t *testing.T) {
	cmd := newChannelsCmd()

	t.Run("command_name", func(t *testing.T) {
		assert.Equal(t, "channels", cmd.Use)
	})

	t.Run("has_subcommands", func(t *testing.T) {
		subcommands := cmd.Commands()
		assert.NotEmpty(t, subcommands)

		cmdNames := make([]string, 0, len(subcommands))
		for _, sub := range subcommands {
			cmdNames = append(cmdNames, sub.Name())
		}

		// Should have list and info subcommands
		assert.Contains(t, cmdNames, "list")
		assert.Contains(t, cmdNames, "info")
	})
}

func TestMessagesCommand(t *testing.T) {
	cmd := newMessagesCmd()

	t.Run("command_name", func(t *testing.T) {
		assert.Equal(t, "messages", cmd.Use)
	})

	t.Run("has_list_subcommand", func(t *testing.T) {
		subcommands := cmd.Commands()
		assert.NotEmpty(t, subcommands)

		found := false
		for _, sub := range subcommands {
			if sub.Name() == "list" {
				found = true
				break
			}
		}
		assert.True(t, found, "Should have 'list' subcommand")
	})
}

func TestFilesCommand(t *testing.T) {
	cmd := newFilesCmd()

	t.Run("command_name", func(t *testing.T) {
		assert.Equal(t, "files", cmd.Use)
	})

	t.Run("has_subcommands", func(t *testing.T) {
		subcommands := cmd.Commands()
		assert.NotEmpty(t, subcommands)

		cmdNames := make([]string, 0, len(subcommands))
		for _, sub := range subcommands {
			cmdNames = append(cmdNames, sub.Name())
		}

		// Should have list and download subcommands
		assert.Contains(t, cmdNames, "list")
		assert.Contains(t, cmdNames, "download")
	})
}

func TestUsersCommand(t *testing.T) {
	cmd := newUsersCmd()

	t.Run("command_name", func(t *testing.T) {
		assert.Equal(t, "users", cmd.Use)
	})

	t.Run("has_list_subcommand", func(t *testing.T) {
		subcommands := cmd.Commands()
		assert.NotEmpty(t, subcommands)

		found := false
		for _, sub := range subcommands {
			if sub.Name() == "list" {
				found = true
				break
			}
		}
		assert.True(t, found, "Should have 'list' subcommand")
	})
}

func TestSendCommand(t *testing.T) {
	cmd := newSendCmd()

	t.Run("command_name", func(t *testing.T) {
		assert.Equal(t, "send", cmd.Use)
	})

	t.Run("has_channel_flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("channel")
		assert.NotNil(t, flag, "Expected --channel flag")
	})

	t.Run("has_text_flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("text")
		assert.NotNil(t, flag, "Expected --text flag")
	})
}

func TestReplyCommand(t *testing.T) {
	cmd := newReplyCmd()

	t.Run("command_name", func(t *testing.T) {
		assert.Equal(t, "reply", cmd.Use)
	})

	t.Run("has_channel_flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("channel")
		assert.NotNil(t, flag, "Expected --channel flag")
	})

	t.Run("has_thread_flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("thread")
		assert.NotNil(t, flag, "Expected --thread flag")
	})

	t.Run("has_text_flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("text")
		assert.NotNil(t, flag, "Expected --text flag")
	})
}

func TestSearchCommand(t *testing.T) {
	cmd := newSearchCmd()

	t.Run("command_name", func(t *testing.T) {
		assert.Equal(t, "search", cmd.Use)
	})

	t.Run("has_query_flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("query")
		assert.NotNil(t, flag, "Expected --query flag")
	})
}
