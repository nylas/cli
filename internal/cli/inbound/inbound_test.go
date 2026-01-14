package inbound

import (
	"bytes"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

// executeCommand executes a command and captures its output.
func executeCommand(root *cobra.Command, args ...string) (string, string, error) {
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)

	root.SetOut(stdout)
	root.SetErr(stderr)
	root.SetArgs(args)

	err := root.Execute()

	return stdout.String(), stderr.String(), err
}

// =============================================================================
// MAIN COMMAND TESTS
// =============================================================================

func TestNewInboundCmd(t *testing.T) {
	cmd := NewInboundCmd()

	t.Run("command_name", func(t *testing.T) {
		assert.Equal(t, "inbound", cmd.Use)
	})

	t.Run("has_inbox_alias", func(t *testing.T) {
		assert.Contains(t, cmd.Aliases, "inbox")
	})

	t.Run("has_short_description", func(t *testing.T) {
		assert.NotEmpty(t, cmd.Short)
		assert.Contains(t, cmd.Short, "inbound")
	})

	t.Run("has_long_description", func(t *testing.T) {
		assert.NotEmpty(t, cmd.Long)
		assert.Contains(t, cmd.Long, "Nylas Inbound")
	})

	t.Run("has_subcommands", func(t *testing.T) {
		subcommands := cmd.Commands()
		assert.NotEmpty(t, subcommands)
	})

	t.Run("has_required_subcommands", func(t *testing.T) {
		expectedCmds := []string{"list", "show", "create", "delete", "messages", "monitor"}

		cmdMap := make(map[string]bool)
		for _, sub := range cmd.Commands() {
			cmdMap[sub.Name()] = true
		}

		for _, expected := range expectedCmds {
			assert.True(t, cmdMap[expected], "Missing expected subcommand: %s", expected)
		}
	})
}

// =============================================================================
// LIST COMMAND TESTS
// =============================================================================

func TestListCommand(t *testing.T) {
	cmd := newListCmd()

	t.Run("command_name", func(t *testing.T) {
		assert.Equal(t, "list", cmd.Use)
	})

	t.Run("has_short_description", func(t *testing.T) {
		assert.NotEmpty(t, cmd.Short)
		assert.Contains(t, cmd.Short, "List")
	})

	t.Run("has_json_flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("json")
		assert.NotNil(t, flag)
		assert.Equal(t, "false", flag.DefValue)
	})

	t.Run("has_long_description_with_examples", func(t *testing.T) {
		assert.Contains(t, cmd.Long, "Examples")
	})
}

// =============================================================================
// SHOW COMMAND TESTS
// =============================================================================

func TestShowCommand(t *testing.T) {
	cmd := newShowCmd()

	t.Run("command_name", func(t *testing.T) {
		assert.Equal(t, "show <inbox-id>", cmd.Use)
	})

	t.Run("has_short_description", func(t *testing.T) {
		assert.NotEmpty(t, cmd.Short)
		assert.Contains(t, cmd.Short, "Show")
	})

	t.Run("has_json_flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("json")
		assert.NotNil(t, flag)
		assert.Equal(t, "false", flag.DefValue)
	})

	t.Run("has_long_description_with_examples", func(t *testing.T) {
		assert.Contains(t, cmd.Long, "Examples")
		assert.Contains(t, cmd.Long, "NYLAS_INBOUND_GRANT_ID")
	})
}

// =============================================================================
// CREATE COMMAND TESTS
// =============================================================================

func TestCreateCommand(t *testing.T) {
	cmd := newCreateCmd()

	t.Run("command_name", func(t *testing.T) {
		assert.Equal(t, "create <email-prefix>", cmd.Use)
	})

	t.Run("has_short_description", func(t *testing.T) {
		assert.NotEmpty(t, cmd.Short)
		assert.Contains(t, cmd.Short, "Create")
	})

	t.Run("has_json_flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("json")
		assert.NotNil(t, flag)
		assert.Equal(t, "false", flag.DefValue)
	})

	t.Run("requires_one_argument", func(t *testing.T) {
		// Test that the command requires exactly one argument by checking the help text
		assert.Contains(t, cmd.Use, "<email-prefix>")
	})

	t.Run("has_long_description_with_examples", func(t *testing.T) {
		assert.Contains(t, cmd.Long, "Examples")
		assert.Contains(t, cmd.Long, "support")
		assert.Contains(t, cmd.Long, "nylas.email")
	})
}

// =============================================================================
// DELETE COMMAND TESTS
// =============================================================================

func TestDeleteCommand(t *testing.T) {
	cmd := newDeleteCmd()

	t.Run("command_name", func(t *testing.T) {
		assert.Equal(t, "delete <inbox-id>", cmd.Use)
	})

	t.Run("has_short_description", func(t *testing.T) {
		assert.NotEmpty(t, cmd.Short)
		assert.Contains(t, cmd.Short, "Delete")
	})

	t.Run("has_yes_flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("yes")
		assert.NotNil(t, flag)
		assert.Equal(t, "false", flag.DefValue)
	})

	t.Run("has_yes_shorthand", func(t *testing.T) {
		flag := cmd.Flags().ShorthandLookup("y")
		assert.NotNil(t, flag)
		assert.Equal(t, "yes", flag.Name)
	})

	t.Run("has_force_flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("force")
		assert.NotNil(t, flag)
		assert.Equal(t, "false", flag.DefValue)
	})

	t.Run("has_force_shorthand", func(t *testing.T) {
		flag := cmd.Flags().ShorthandLookup("f")
		assert.NotNil(t, flag)
		assert.Equal(t, "force", flag.Name)
	})

	t.Run("has_long_description_with_examples", func(t *testing.T) {
		assert.Contains(t, cmd.Long, "Examples")
		assert.Contains(t, cmd.Long, "--yes")
	})
}

// =============================================================================
// MESSAGES COMMAND TESTS
// =============================================================================

func TestMessagesCommand(t *testing.T) {
	cmd := newMessagesCmd()

	t.Run("command_name", func(t *testing.T) {
		assert.Equal(t, "messages [inbox-id]", cmd.Use)
	})

	t.Run("has_short_description", func(t *testing.T) {
		assert.NotEmpty(t, cmd.Short)
		assert.Contains(t, cmd.Short, "messages")
	})

	t.Run("has_limit_flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("limit")
		assert.NotNil(t, flag)
		assert.Equal(t, "10", flag.DefValue)
	})

	t.Run("has_limit_shorthand", func(t *testing.T) {
		flag := cmd.Flags().ShorthandLookup("l")
		assert.NotNil(t, flag)
		assert.Equal(t, "limit", flag.Name)
	})

	t.Run("has_unread_flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("unread")
		assert.NotNil(t, flag)
		assert.Equal(t, "false", flag.DefValue)
	})

	t.Run("has_unread_shorthand", func(t *testing.T) {
		flag := cmd.Flags().ShorthandLookup("u")
		assert.NotNil(t, flag)
		assert.Equal(t, "unread", flag.Name)
	})

	t.Run("has_json_flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("json")
		assert.NotNil(t, flag)
		assert.Equal(t, "false", flag.DefValue)
	})

	t.Run("has_long_description_with_examples", func(t *testing.T) {
		assert.Contains(t, cmd.Long, "Examples")
		assert.Contains(t, cmd.Long, "NYLAS_INBOUND_GRANT_ID")
	})
}

// =============================================================================
// MONITOR COMMAND TESTS
// =============================================================================

func TestMonitorCommand(t *testing.T) {
	cmd := newMonitorCmd()

	t.Run("command_name", func(t *testing.T) {
		assert.Equal(t, "monitor [inbox-id]", cmd.Use)
	})

	t.Run("has_short_description", func(t *testing.T) {
		assert.NotEmpty(t, cmd.Short)
		assert.Contains(t, cmd.Short, "Monitor")
	})

	t.Run("has_port_flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("port")
		assert.NotNil(t, flag)
		assert.Equal(t, "3000", flag.DefValue)
	})

	t.Run("has_port_shorthand", func(t *testing.T) {
		flag := cmd.Flags().ShorthandLookup("p")
		assert.NotNil(t, flag)
		assert.Equal(t, "port", flag.Name)
	})

	t.Run("has_tunnel_flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("tunnel")
		assert.NotNil(t, flag)
	})

	t.Run("has_tunnel_shorthand", func(t *testing.T) {
		flag := cmd.Flags().ShorthandLookup("t")
		assert.NotNil(t, flag)
		assert.Equal(t, "tunnel", flag.Name)
	})

	t.Run("has_secret_flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("secret")
		assert.NotNil(t, flag)
	})

	t.Run("has_secret_shorthand", func(t *testing.T) {
		flag := cmd.Flags().ShorthandLookup("s")
		assert.NotNil(t, flag)
		assert.Equal(t, "secret", flag.Name)
	})

	t.Run("has_json_flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("json")
		assert.NotNil(t, flag)
		assert.Equal(t, "false", flag.DefValue)
	})

	t.Run("has_quiet_flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("quiet")
		assert.NotNil(t, flag)
		assert.Equal(t, "false", flag.DefValue)
	})

	t.Run("has_quiet_shorthand", func(t *testing.T) {
		flag := cmd.Flags().ShorthandLookup("q")
		assert.NotNil(t, flag)
		assert.Equal(t, "quiet", flag.Name)
	})

	t.Run("has_long_description_with_examples", func(t *testing.T) {
		assert.Contains(t, cmd.Long, "Examples")
		assert.Contains(t, cmd.Long, "cloudflared")
		assert.Contains(t, cmd.Long, "Ctrl+C")
	})
}

// Helper and help output tests are in inbound_helpers_test.go
