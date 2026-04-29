package admin

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewCallbackURIsCmd(t *testing.T) {
	cmd := newCallbackURIsCmd()

	t.Run("command_name", func(t *testing.T) {
		assert.Equal(t, "callback-uris", cmd.Use)
	})

	t.Run("has_aliases", func(t *testing.T) {
		assert.Contains(t, cmd.Aliases, "callbacks")
		assert.Contains(t, cmd.Aliases, "cb")
	})

	t.Run("has_short_description", func(t *testing.T) {
		assert.NotEmpty(t, cmd.Short)
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

func TestCallbackURIListCmd(t *testing.T) {
	cmd := newCallbackURIListCmd()

	t.Run("command_name", func(t *testing.T) {
		assert.Equal(t, "list", cmd.Use)
	})

	t.Run("has_ls_alias", func(t *testing.T) {
		assert.Contains(t, cmd.Aliases, "ls")
	})

}

func TestCallbackURIShowCmd(t *testing.T) {
	cmd := newCallbackURIShowCmd()

	t.Run("command_name", func(t *testing.T) {
		assert.Equal(t, "show <uri-id>", cmd.Use)
	})

	t.Run("requires_exactly_one_arg", func(t *testing.T) {
		assert.NotNil(t, cmd.Args)
	})

}

func TestCallbackURICreateCmd(t *testing.T) {
	cmd := newCallbackURICreateCmd()

	t.Run("command_name", func(t *testing.T) {
		assert.Equal(t, "create", cmd.Use)
	})

	t.Run("has_url_flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("url")
		assert.NotNil(t, flag)
	})

	t.Run("has_platform_flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("platform")
		assert.NotNil(t, flag)
		assert.Equal(t, "web", flag.DefValue)
	})

	t.Run("has_short_description", func(t *testing.T) {
		assert.NotEmpty(t, cmd.Short)
	})
}

func TestCallbackURIUpdateCmd(t *testing.T) {
	cmd := newCallbackURIUpdateCmd()

	t.Run("command_name", func(t *testing.T) {
		assert.Equal(t, "update <uri-id>", cmd.Use)
	})

	t.Run("requires_exactly_one_arg", func(t *testing.T) {
		assert.NotNil(t, cmd.Args)
	})

	t.Run("has_url_flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("url")
		assert.NotNil(t, flag)
	})

	t.Run("has_platform_flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("platform")
		assert.NotNil(t, flag)
	})
}

func TestCallbackURIDeleteCmd(t *testing.T) {
	cmd := newCallbackURIDeleteCmd()

	t.Run("command_name", func(t *testing.T) {
		assert.Equal(t, "delete <uri-id>", cmd.Use)
	})

	t.Run("requires_exactly_one_arg", func(t *testing.T) {
		assert.NotNil(t, cmd.Args)
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
