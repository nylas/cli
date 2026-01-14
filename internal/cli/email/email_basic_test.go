package email

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

func TestNewEmailCmd(t *testing.T) {
	cmd := NewEmailCmd()

	t.Run("command_name", func(t *testing.T) {
		assert.Equal(t, "email", cmd.Use)
	})

	t.Run("has_short_description", func(t *testing.T) {
		assert.NotEmpty(t, cmd.Short)
	})

	t.Run("has_subcommands", func(t *testing.T) {
		subcommands := cmd.Commands()
		assert.NotEmpty(t, subcommands)
	})

	t.Run("has_required_subcommands", func(t *testing.T) {
		expectedCmds := []string{"list", "read", "send", "search", "mark", "delete", "folders", "threads", "drafts"}

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

	t.Run("has_limit_flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("limit")
		assert.NotNil(t, flag)
		assert.Equal(t, "10", flag.DefValue)
	})

	t.Run("has_unread_flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("unread")
		assert.NotNil(t, flag)
	})

	t.Run("has_starred_flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("starred")
		assert.NotNil(t, flag)
	})

	t.Run("has_from_flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("from")
		assert.NotNil(t, flag)
	})
}

func TestReadCommand(t *testing.T) {
	cmd := newReadCmd()

	t.Run("command_name", func(t *testing.T) {
		assert.Equal(t, "read <message-id> [grant-id]", cmd.Use)
	})

	t.Run("has_show_alias", func(t *testing.T) {
		assert.Contains(t, cmd.Aliases, "show")
	})

	t.Run("has_mark_read_flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("mark-read")
		assert.NotNil(t, flag)
	})

	t.Run("has_raw_flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("raw")
		assert.NotNil(t, flag)
	})
}

func TestSendCommand(t *testing.T) {
	cmd := newSendCmd()

	t.Run("command_name", func(t *testing.T) {
		assert.Equal(t, "send [grant-id]", cmd.Use)
	})

	t.Run("has_to_flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("to")
		assert.NotNil(t, flag)
	})

	t.Run("has_subject_flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("subject")
		assert.NotNil(t, flag)
	})

	t.Run("has_body_flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("body")
		assert.NotNil(t, flag)
	})

	t.Run("has_cc_flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("cc")
		assert.NotNil(t, flag)
	})

	t.Run("has_bcc_flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("bcc")
		assert.NotNil(t, flag)
	})
}

func TestSearchCommand(t *testing.T) {
	cmd := newSearchCmd()

	t.Run("command_name", func(t *testing.T) {
		assert.Equal(t, "search <query> [grant-id]", cmd.Use)
	})

	t.Run("has_limit_flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("limit")
		assert.NotNil(t, flag)
		assert.Equal(t, "20", flag.DefValue)
	})

	t.Run("has_from_flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("from")
		assert.NotNil(t, flag)
	})

	t.Run("has_after_flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("after")
		assert.NotNil(t, flag)
	})

	t.Run("has_before_flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("before")
		assert.NotNil(t, flag)
	})

	t.Run("has_unread_flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("unread")
		assert.NotNil(t, flag)
	})

	t.Run("has_starred_flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("starred")
		assert.NotNil(t, flag)
	})

	t.Run("has_in_flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("in")
		assert.NotNil(t, flag)
	})

	t.Run("has_long_description_with_examples", func(t *testing.T) {
		assert.Contains(t, cmd.Long, "Examples")
	})
}

func TestMarkCommand(t *testing.T) {
	cmd := newMarkCmd()

	t.Run("command_name", func(t *testing.T) {
		assert.Equal(t, "mark", cmd.Use)
	})

	t.Run("has_subcommands", func(t *testing.T) {
		subcommands := cmd.Commands()
		assert.Len(t, subcommands, 4) // read, unread, starred, unstarred
	})

	t.Run("has_read_subcommand", func(t *testing.T) {
		found := false
		for _, sub := range cmd.Commands() {
			if sub.Name() == "read" {
				found = true
				break
			}
		}
		assert.True(t, found)
	})
}

func TestFoldersCommand(t *testing.T) {
	cmd := newFoldersCmd()

	t.Run("command_name", func(t *testing.T) {
		assert.Equal(t, "folders", cmd.Use)
	})

	t.Run("has_subcommands", func(t *testing.T) {
		subcommands := cmd.Commands()
		assert.GreaterOrEqual(t, len(subcommands), 3) // list, create, delete
	})
}

func TestFoldersListCommand(t *testing.T) {
	cmd := newFoldersListCmd()

	t.Run("command_name", func(t *testing.T) {
		assert.Equal(t, "list [grant-id]", cmd.Use)
	})

	t.Run("has_id_flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("id")
		assert.NotNil(t, flag)
		assert.Equal(t, "false", flag.DefValue)
	})

	t.Run("has_short_description", func(t *testing.T) {
		assert.NotEmpty(t, cmd.Short)
		assert.Contains(t, cmd.Short, "folders")
	})
}

func TestThreadsCommand(t *testing.T) {
	cmd := newThreadsCmd()

	t.Run("command_name", func(t *testing.T) {
		assert.Equal(t, "threads", cmd.Use)
	})

	t.Run("has_required_subcommands", func(t *testing.T) {
		expectedCmds := []string{"list", "show", "mark", "delete", "search"}

		cmdMap := make(map[string]bool)
		for _, sub := range cmd.Commands() {
			cmdMap[sub.Name()] = true
		}

		for _, expected := range expectedCmds {
			assert.True(t, cmdMap[expected], "Missing expected subcommand: %s", expected)
		}
	})
}
