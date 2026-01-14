package admin

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// Grants Tests
func TestNewGrantsCmd(t *testing.T) {
	cmd := newGrantsCmd()

	t.Run("command_name", func(t *testing.T) {
		assert.Equal(t, "grants", cmd.Use)
	})

	t.Run("has_aliases", func(t *testing.T) {
		assert.Contains(t, cmd.Aliases, "grant")
	})

	t.Run("has_subcommands", func(t *testing.T) {
		expectedCmds := []string{"list", "stats"}

		cmdMap := make(map[string]bool)
		for _, sub := range cmd.Commands() {
			cmdMap[sub.Name()] = true
		}

		for _, expected := range expectedCmds {
			assert.True(t, cmdMap[expected], "Missing expected subcommand: %s", expected)
		}
	})
}

func TestGrantListCmd(t *testing.T) {
	cmd := newGrantListCmd()

	t.Run("command_name", func(t *testing.T) {
		assert.Equal(t, "list", cmd.Use)
	})

	t.Run("has_ls_alias", func(t *testing.T) {
		assert.Contains(t, cmd.Aliases, "ls")
	})

	t.Run("has_filter_flags", func(t *testing.T) {
		assert.NotNil(t, cmd.Flags().Lookup("connector-id"))
		assert.NotNil(t, cmd.Flags().Lookup("status"))
		assert.NotNil(t, cmd.Flags().Lookup("limit"))
		assert.NotNil(t, cmd.Flags().Lookup("offset"))
	})

	t.Run("has_json_flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("json")
		assert.NotNil(t, flag)
	})

	t.Run("has_correct_limit_default", func(t *testing.T) {
		flag := cmd.Flags().Lookup("limit")
		assert.Equal(t, "50", flag.DefValue)
	})
}

func TestGrantStatsCmd(t *testing.T) {
	cmd := newGrantStatsCmd()

	t.Run("command_name", func(t *testing.T) {
		assert.Equal(t, "stats", cmd.Use)
	})

	t.Run("has_short_description", func(t *testing.T) {
		assert.NotEmpty(t, cmd.Short)
	})
}
