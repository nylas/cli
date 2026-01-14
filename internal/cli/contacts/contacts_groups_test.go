package contacts

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGroupsCmd(t *testing.T) {
	cmd := newGroupsCmd()

	t.Run("command_name", func(t *testing.T) {
		assert.Equal(t, "groups", cmd.Use)
	})

	t.Run("has_aliases", func(t *testing.T) {
		assert.Contains(t, cmd.Aliases, "group")
	})

	t.Run("has_short_description", func(t *testing.T) {
		assert.NotEmpty(t, cmd.Short)
		assert.Contains(t, cmd.Short, "group")
	})

	t.Run("has_subcommands", func(t *testing.T) {
		subcommands := cmd.Commands()
		assert.NotEmpty(t, subcommands)
	})

	t.Run("has_required_subcommands", func(t *testing.T) {
		expectedCmds := []string{"list", "show", "create", "update", "delete"}

		cmdMap := make(map[string]bool)
		for _, sub := range cmd.Commands() {
			cmdMap[sub.Name()] = true
		}

		for _, expected := range expectedCmds {
			assert.True(t, cmdMap[expected], "Missing expected subcommand: %s", expected)
		}
	})
}

func TestGroupsListCmd(t *testing.T) {
	cmd := newGroupsListCmd()

	t.Run("command_name", func(t *testing.T) {
		assert.Equal(t, "list [grant-id]", cmd.Use)
	})

	t.Run("has_aliases", func(t *testing.T) {
		assert.Contains(t, cmd.Aliases, "ls")
	})

	t.Run("has_short_description", func(t *testing.T) {
		assert.NotEmpty(t, cmd.Short)
	})
}

func TestGroupsShowCmd(t *testing.T) {
	cmd := newGroupsShowCmd()

	t.Run("command_name", func(t *testing.T) {
		assert.Equal(t, "show <group-id> [grant-id]", cmd.Use)
	})

	t.Run("has_short_description", func(t *testing.T) {
		assert.NotEmpty(t, cmd.Short)
	})
}

func TestGroupsCreateCmd(t *testing.T) {
	cmd := newGroupsCreateCmd()

	t.Run("command_name", func(t *testing.T) {
		assert.Equal(t, "create <name> [grant-id]", cmd.Use)
	})

	t.Run("has_short_description", func(t *testing.T) {
		assert.NotEmpty(t, cmd.Short)
		assert.Contains(t, cmd.Short, "Create")
	})

	t.Run("has_long_description_with_examples", func(t *testing.T) {
		assert.Contains(t, cmd.Long, "Examples")
	})
}

func TestGroupsUpdateCmd(t *testing.T) {
	cmd := newGroupsUpdateCmd()

	t.Run("command_name", func(t *testing.T) {
		assert.Equal(t, "update <group-id> [grant-id]", cmd.Use)
	})

	t.Run("has_short_description", func(t *testing.T) {
		assert.NotEmpty(t, cmd.Short)
		assert.Contains(t, cmd.Short, "Update")
	})

	t.Run("has_name_flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("name")
		assert.NotNil(t, flag)
	})
}

func TestGroupsDeleteCmd(t *testing.T) {
	cmd := newGroupsDeleteCmd()

	t.Run("command_name", func(t *testing.T) {
		assert.Equal(t, "delete <group-id> [grant-id]", cmd.Use)
	})

	t.Run("has_aliases", func(t *testing.T) {
		assert.Contains(t, cmd.Aliases, "rm")
		assert.Contains(t, cmd.Aliases, "remove")
	})

	t.Run("has_force_flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("force")
		assert.NotNil(t, flag)
	})
}

func TestContactsCommandHelp(t *testing.T) {
	cmd := NewContactsCmd()
	stdout, _, err := executeCommand(cmd, "--help")

	assert.NoError(t, err)

	expectedStrings := []string{
		"contacts",
		"list",
		"show",
		"create",
		"update",
		"delete",
		"groups",
	}

	for _, expected := range expectedStrings {
		assert.Contains(t, stdout, expected, "Help output should contain %q", expected)
	}
}

func TestContactsListHelp(t *testing.T) {
	cmd := NewContactsCmd()
	stdout, _, err := executeCommand(cmd, "list", "--help")

	assert.NoError(t, err)
	assert.Contains(t, stdout, "list")
	assert.Contains(t, stdout, "--limit")
	assert.Contains(t, stdout, "--email")
	assert.Contains(t, stdout, "--source")
	assert.Contains(t, stdout, "--id")
}

func TestContactsCreateHelp(t *testing.T) {
	cmd := NewContactsCmd()
	stdout, _, err := executeCommand(cmd, "create", "--help")

	assert.NoError(t, err)
	assert.Contains(t, stdout, "create")
	assert.Contains(t, stdout, "--first-name")
	assert.Contains(t, stdout, "--last-name")
	assert.Contains(t, stdout, "--email")
	assert.Contains(t, stdout, "--phone")
	assert.Contains(t, stdout, "--company")
}

func TestContactsDeleteHelp(t *testing.T) {
	cmd := NewContactsCmd()
	stdout, _, err := executeCommand(cmd, "delete", "--help")

	assert.NoError(t, err)
	assert.Contains(t, stdout, "delete")
	assert.Contains(t, stdout, "--force")
}

func TestContactsUpdateHelp(t *testing.T) {
	cmd := NewContactsCmd()
	stdout, _, err := executeCommand(cmd, "update", "--help")

	assert.NoError(t, err)
	assert.Contains(t, stdout, "update")
	assert.Contains(t, stdout, "--given-name")
	assert.Contains(t, stdout, "--surname")
	assert.Contains(t, stdout, "--company")
	assert.Contains(t, stdout, "--email")
}

func TestContactsGroupsHelp(t *testing.T) {
	cmd := NewContactsCmd()
	stdout, _, err := executeCommand(cmd, "groups", "--help")

	assert.NoError(t, err)
	assert.Contains(t, stdout, "groups")
	assert.Contains(t, stdout, "list")
	assert.Contains(t, stdout, "show")
	assert.Contains(t, stdout, "create")
	assert.Contains(t, stdout, "update")
	assert.Contains(t, stdout, "delete")
}

func TestContactsGroupsCreateHelp(t *testing.T) {
	cmd := NewContactsCmd()
	stdout, _, err := executeCommand(cmd, "groups", "create", "--help")

	assert.NoError(t, err)
	assert.Contains(t, stdout, "create")
	assert.Contains(t, stdout, "<name>")
}

func TestContactsGroupsUpdateHelp(t *testing.T) {
	cmd := NewContactsCmd()
	stdout, _, err := executeCommand(cmd, "groups", "update", "--help")

	assert.NoError(t, err)
	assert.Contains(t, stdout, "update")
	assert.Contains(t, stdout, "--name")
}

func TestContactsGroupsDeleteHelp(t *testing.T) {
	cmd := NewContactsCmd()
	stdout, _, err := executeCommand(cmd, "groups", "delete", "--help")

	assert.NoError(t, err)
	assert.Contains(t, stdout, "delete")
	assert.Contains(t, stdout, "--force")
}
