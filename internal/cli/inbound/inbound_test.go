package inbound

import (
	"bytes"
	"testing"

	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/domain"
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
		assert.Equal(t, "create <email>", cmd.Use)
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
		assert.Contains(t, cmd.Use, "<email>")
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

// =============================================================================
// HELPER FUNCTION TESTS
// =============================================================================

func TestGetInboxID(t *testing.T) {
	t.Run("returns_first_arg_when_provided", func(t *testing.T) {
		id, err := getInboxID([]string{"test-inbox-id"})
		assert.NoError(t, err)
		assert.Equal(t, "test-inbox-id", id)
	})

	t.Run("returns_error_when_no_args_and_no_env", func(t *testing.T) {
		// Ensure env var is not set
		t.Setenv("NYLAS_INBOUND_GRANT_ID", "")
		_, err := getInboxID([]string{})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "inbox ID required")
	})

	t.Run("returns_env_var_when_no_args", func(t *testing.T) {
		t.Setenv("NYLAS_INBOUND_GRANT_ID", "env-inbox-id")
		id, err := getInboxID([]string{})
		assert.NoError(t, err)
		assert.Equal(t, "env-inbox-id", id)
	})

	t.Run("prefers_arg_over_env_var", func(t *testing.T) {
		t.Setenv("NYLAS_INBOUND_GRANT_ID", "env-inbox-id")
		id, err := getInboxID([]string{"arg-inbox-id"})
		assert.NoError(t, err)
		assert.Equal(t, "arg-inbox-id", id)
	})
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		input    string
		maxLen   int
		expected string
	}{
		{"hello", 10, "hello"},
		{"hello world", 8, "hello..."},
		{"short", 5, "short"},
		{"exactly10!", 10, "exactly10!"},
		{"longer than max", 10, "longer ..."},
	}

	for _, tt := range tests {
		got := common.Truncate(tt.input, tt.maxLen)
		assert.Equal(t, tt.expected, got)
	}
}

func TestFormatParticipant(t *testing.T) {
	tests := []struct {
		contact  domain.EmailParticipant
		expected string
	}{
		{domain.EmailParticipant{Name: "John", Email: "john@example.com"}, "John"},
		{domain.EmailParticipant{Name: "", Email: "jane@example.com"}, "jane@example.com"},
		{domain.EmailParticipant{Name: "Alice", Email: ""}, "Alice"},
	}

	for _, tt := range tests {
		got := common.FormatParticipant(tt.contact)
		assert.Equal(t, tt.expected, got)
	}
}

func TestFormatParticipants(t *testing.T) {
	contacts := []domain.EmailParticipant{
		{Name: "John", Email: "john@example.com"},
		{Name: "", Email: "jane@example.com"},
	}
	got := common.FormatParticipants(contacts)
	assert.Equal(t, "John, jane@example.com", got)
}

func TestFormatStatus(t *testing.T) {
	tests := []struct {
		status   string
		contains string
	}{
		{"valid", "active"},
		{"invalid", "invalid"},
		{"pending", "pending"},
	}

	for _, tt := range tests {
		got := formatStatus(tt.status)
		assert.Contains(t, got, tt.contains)
	}
}

// =============================================================================
// HELP OUTPUT TESTS
// =============================================================================

func TestInboundCommandHelp(t *testing.T) {
	cmd := NewInboundCmd()
	stdout, _, err := executeCommand(cmd, "--help")

	assert.NoError(t, err)

	expectedStrings := []string{
		"inbound",
		"list",
		"show",
		"create",
		"delete",
		"messages",
		"monitor",
	}

	for _, expected := range expectedStrings {
		assert.Contains(t, stdout, expected, "Help output should contain %q", expected)
	}
}

func TestInboundListHelp(t *testing.T) {
	cmd := NewInboundCmd()
	stdout, _, err := executeCommand(cmd, "list", "--help")

	assert.NoError(t, err)
	assert.Contains(t, stdout, "list")
	assert.Contains(t, stdout, "--json")
}

func TestInboundCreateHelp(t *testing.T) {
	cmd := NewInboundCmd()
	stdout, _, err := executeCommand(cmd, "create", "--help")

	assert.NoError(t, err)
	assert.Contains(t, stdout, "create")
	assert.Contains(t, stdout, "--json")
	assert.Contains(t, stdout, "email")
}

func TestInboundDeleteHelp(t *testing.T) {
	cmd := NewInboundCmd()
	stdout, _, err := executeCommand(cmd, "delete", "--help")

	assert.NoError(t, err)
	assert.Contains(t, stdout, "delete")
	assert.Contains(t, stdout, "--yes")
	assert.Contains(t, stdout, "--force")
}

func TestInboundMessagesHelp(t *testing.T) {
	cmd := NewInboundCmd()
	stdout, _, err := executeCommand(cmd, "messages", "--help")

	assert.NoError(t, err)
	assert.Contains(t, stdout, "messages")
	assert.Contains(t, stdout, "--limit")
	assert.Contains(t, stdout, "--unread")
	assert.Contains(t, stdout, "--json")
}

func TestInboundMonitorHelp(t *testing.T) {
	cmd := NewInboundCmd()
	stdout, _, err := executeCommand(cmd, "monitor", "--help")

	assert.NoError(t, err)
	assert.Contains(t, stdout, "monitor")
	assert.Contains(t, stdout, "--port")
	assert.Contains(t, stdout, "--tunnel")
	assert.Contains(t, stdout, "--secret")
	assert.Contains(t, stdout, "--json")
	assert.Contains(t, stdout, "--quiet")
}
