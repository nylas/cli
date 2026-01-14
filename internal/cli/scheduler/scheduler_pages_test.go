package scheduler

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// =============================================================================
// Pages Tests
// =============================================================================

func TestNewPagesCmd(t *testing.T) {
	cmd := newPagesCmd()

	t.Run("command_name", func(t *testing.T) {
		assert.Equal(t, "pages", cmd.Use)
	})

	t.Run("has_alias", func(t *testing.T) {
		assert.Contains(t, cmd.Aliases, "page")
	})

	t.Run("has_subcommands", func(t *testing.T) {
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

func TestPageListCmd(t *testing.T) {
	cmd := newPageListCmd()

	t.Run("command_name", func(t *testing.T) {
		assert.Equal(t, "list", cmd.Use)
	})

	t.Run("has_ls_alias", func(t *testing.T) {
		assert.Contains(t, cmd.Aliases, "ls")
	})

	t.Run("has_json_flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("json")
		assert.NotNil(t, flag)
	})
}

func TestPageShowCmd(t *testing.T) {
	cmd := newPageShowCmd()

	t.Run("command_name", func(t *testing.T) {
		assert.Equal(t, "show <page-id>", cmd.Use)
	})

	t.Run("has_json_flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("json")
		assert.NotNil(t, flag)
	})
}

func TestPageCreateCmd(t *testing.T) {
	cmd := newPageCreateCmd()

	t.Run("command_name", func(t *testing.T) {
		assert.Equal(t, "create", cmd.Use)
	})

	t.Run("has_required_flags", func(t *testing.T) {
		assert.NotNil(t, cmd.Flags().Lookup("name"))
		assert.NotNil(t, cmd.Flags().Lookup("config-id"))
	})

	t.Run("has_slug_flag", func(t *testing.T) {
		assert.NotNil(t, cmd.Flags().Lookup("slug"))
	})
}

func TestPageUpdateCmd(t *testing.T) {
	cmd := newPageUpdateCmd()

	t.Run("command_name", func(t *testing.T) {
		assert.Equal(t, "update <page-id>", cmd.Use)
	})

	t.Run("has_update_flags", func(t *testing.T) {
		assert.NotNil(t, cmd.Flags().Lookup("name"))
		assert.NotNil(t, cmd.Flags().Lookup("slug"))
	})
}

func TestPageDeleteCmd(t *testing.T) {
	cmd := newPageDeleteCmd()

	t.Run("command_name", func(t *testing.T) {
		assert.Equal(t, "delete <page-id>", cmd.Use)
	})

	t.Run("has_yes_flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("yes")
		assert.NotNil(t, flag)
		assert.Equal(t, "false", flag.DefValue)
	})

	t.Run("has_yes_shorthand", func(t *testing.T) {
		flag := cmd.Flags().ShorthandLookup("y")
		assert.NotNil(t, flag)
	})
}
