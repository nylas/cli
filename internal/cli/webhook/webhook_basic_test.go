package webhook

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

func TestNewWebhookCmd(t *testing.T) {
	cmd := NewWebhookCmd()

	t.Run("command_name", func(t *testing.T) {
		assert.Equal(t, "webhook", cmd.Use)
	})

	t.Run("has_aliases", func(t *testing.T) {
		assert.Contains(t, cmd.Aliases, "webhooks")
		assert.Contains(t, cmd.Aliases, "wh")
	})

	t.Run("has_short_description", func(t *testing.T) {
		assert.NotEmpty(t, cmd.Short)
		assert.Contains(t, cmd.Short, "webhook")
	})

	t.Run("has_long_description", func(t *testing.T) {
		assert.NotEmpty(t, cmd.Long)
		assert.Contains(t, cmd.Long, "Nylas webhooks")
	})

	t.Run("has_subcommands", func(t *testing.T) {
		subcommands := cmd.Commands()
		assert.NotEmpty(t, subcommands)
	})

	t.Run("has_required_subcommands", func(t *testing.T) {
		expectedCmds := []string{"list", "show", "create", "update", "delete", "test", "triggers", "server"}

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
		assert.Equal(t, "list", cmd.Use)
	})

	t.Run("has_aliases", func(t *testing.T) {
		assert.Contains(t, cmd.Aliases, "ls")
	})

	t.Run("has_format_flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("format")
		assert.NotNil(t, flag)
		assert.Equal(t, "table", flag.DefValue)
	})

	t.Run("has_format_shorthand", func(t *testing.T) {
		flag := cmd.Flags().ShorthandLookup("f")
		assert.NotNil(t, flag)
		assert.Equal(t, "format", flag.Name)
	})

	t.Run("has_full_ids_flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("full-ids")
		assert.NotNil(t, flag)
		assert.Equal(t, "false", flag.DefValue)
	})

	t.Run("has_short_description", func(t *testing.T) {
		assert.NotEmpty(t, cmd.Short)
		assert.Contains(t, cmd.Short, "List")
	})

	t.Run("has_examples", func(t *testing.T) {
		assert.NotEmpty(t, cmd.Example)
		assert.Contains(t, cmd.Example, "webhook list")
		assert.Contains(t, cmd.Example, "--full-ids")
	})
}

func TestShowCommand(t *testing.T) {
	cmd := newShowCmd()

	t.Run("command_name", func(t *testing.T) {
		assert.Equal(t, "show <webhook-id>", cmd.Use)
	})

	t.Run("has_format_flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("format")
		assert.NotNil(t, flag)
		assert.Equal(t, "text", flag.DefValue)
	})

	t.Run("has_format_shorthand", func(t *testing.T) {
		flag := cmd.Flags().ShorthandLookup("f")
		assert.NotNil(t, flag)
		assert.Equal(t, "format", flag.Name)
	})

	t.Run("requires_one_arg", func(t *testing.T) {
		// The command expects exactly 1 argument (webhook-id)
		assert.NotNil(t, cmd.Args)
	})

	t.Run("has_examples", func(t *testing.T) {
		assert.NotEmpty(t, cmd.Example)
		assert.Contains(t, cmd.Example, "webhook show")
	})
}

func TestCreateCommand(t *testing.T) {
	cmd := newCreateCmd()

	t.Run("command_name", func(t *testing.T) {
		assert.Equal(t, "create", cmd.Use)
	})

	t.Run("has_url_flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("url")
		assert.NotNil(t, flag)
	})

	t.Run("has_url_shorthand", func(t *testing.T) {
		flag := cmd.Flags().ShorthandLookup("u")
		assert.NotNil(t, flag)
		assert.Equal(t, "url", flag.Name)
	})

	t.Run("has_description_flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("description")
		assert.NotNil(t, flag)
	})

	t.Run("has_description_shorthand", func(t *testing.T) {
		flag := cmd.Flags().ShorthandLookup("d")
		assert.NotNil(t, flag)
		assert.Equal(t, "description", flag.Name)
	})

	t.Run("has_triggers_flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("triggers")
		assert.NotNil(t, flag)
	})

	t.Run("has_triggers_shorthand", func(t *testing.T) {
		flag := cmd.Flags().ShorthandLookup("t")
		assert.NotNil(t, flag)
		assert.Equal(t, "triggers", flag.Name)
	})

	t.Run("has_notify_flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("notify")
		assert.NotNil(t, flag)
	})

	t.Run("has_format_flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("format")
		assert.NotNil(t, flag)
	})

	t.Run("has_examples", func(t *testing.T) {
		assert.NotEmpty(t, cmd.Example)
		assert.Contains(t, cmd.Example, "webhook create")
		assert.Contains(t, cmd.Example, "--url")
		assert.Contains(t, cmd.Example, "--triggers")
	})
}

func TestUpdateCommand(t *testing.T) {
	cmd := newUpdateCmd()

	t.Run("command_name", func(t *testing.T) {
		assert.Equal(t, "update <webhook-id>", cmd.Use)
	})

	t.Run("has_url_flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("url")
		assert.NotNil(t, flag)
	})

	t.Run("has_triggers_flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("triggers")
		assert.NotNil(t, flag)
	})

	t.Run("has_description_flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("description")
		assert.NotNil(t, flag)
	})

	t.Run("has_status_flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("status")
		assert.NotNil(t, flag)
	})

	t.Run("has_notify_flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("notify")
		assert.NotNil(t, flag)
	})

	t.Run("has_format_flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("format")
		assert.NotNil(t, flag)
	})

	t.Run("has_examples", func(t *testing.T) {
		assert.NotEmpty(t, cmd.Example)
		assert.Contains(t, cmd.Example, "webhook update")
	})
}

func TestDeleteCommand(t *testing.T) {
	cmd := newDeleteCmd()

	t.Run("command_name", func(t *testing.T) {
		assert.Equal(t, "delete <webhook-id>", cmd.Use)
	})

	t.Run("has_aliases", func(t *testing.T) {
		assert.Contains(t, cmd.Aliases, "rm")
		assert.Contains(t, cmd.Aliases, "remove")
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

	t.Run("has_examples", func(t *testing.T) {
		assert.NotEmpty(t, cmd.Example)
		assert.Contains(t, cmd.Example, "webhook delete")
	})
}

func TestTestCommand(t *testing.T) {
	cmd := newTestCmd()

	t.Run("command_name", func(t *testing.T) {
		assert.Equal(t, "test", cmd.Use)
	})

	t.Run("has_short_description", func(t *testing.T) {
		assert.NotEmpty(t, cmd.Short)
	})

	t.Run("has_subcommands", func(t *testing.T) {
		subcommands := cmd.Commands()
		assert.NotEmpty(t, subcommands)
	})

	t.Run("has_send_subcommand", func(t *testing.T) {
		found := false
		for _, sub := range cmd.Commands() {
			if sub.Name() == "send" {
				found = true
				break
			}
		}
		assert.True(t, found, "Missing 'send' subcommand")
	})

	t.Run("has_payload_subcommand", func(t *testing.T) {
		found := false
		for _, sub := range cmd.Commands() {
			if sub.Name() == "payload" {
				found = true
				break
			}
		}
		assert.True(t, found, "Missing 'payload' subcommand")
	})
}

func TestTestSendCommand(t *testing.T) {
	cmd := newTestSendCmd()

	t.Run("command_name", func(t *testing.T) {
		assert.Equal(t, "send <webhook-url>", cmd.Use)
	})

	t.Run("has_short_description", func(t *testing.T) {
		assert.NotEmpty(t, cmd.Short)
	})

	t.Run("has_examples", func(t *testing.T) {
		assert.NotEmpty(t, cmd.Example)
		assert.Contains(t, cmd.Example, "test send")
	})

	t.Run("requires_one_arg", func(t *testing.T) {
		assert.NotNil(t, cmd.Args)
	})
}

func TestTestPayloadCommand(t *testing.T) {
	cmd := newTestPayloadCmd()

	t.Run("command_name", func(t *testing.T) {
		assert.Equal(t, "payload [trigger-type]", cmd.Use)
	})

	t.Run("has_format_flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("format")
		assert.NotNil(t, flag)
		assert.Equal(t, "json", flag.DefValue)
	})

	t.Run("has_trigger_flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("trigger")
		assert.NotNil(t, flag)
	})

	t.Run("has_trigger_shorthand", func(t *testing.T) {
		flag := cmd.Flags().ShorthandLookup("t")
		assert.NotNil(t, flag)
		assert.Equal(t, "trigger", flag.Name)
	})

	t.Run("has_examples", func(t *testing.T) {
		assert.NotEmpty(t, cmd.Example)
		assert.Contains(t, cmd.Example, "test payload")
	})
}
