package calendar

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewCalendarCmd(t *testing.T) {
	cmd := NewCalendarCmd()

	t.Run("command_name", func(t *testing.T) {
		assert.Equal(t, "calendar", cmd.Use)
	})

	t.Run("has_aliases", func(t *testing.T) {
		assert.Contains(t, cmd.Aliases, "cal")
	})

	t.Run("has_short_description", func(t *testing.T) {
		assert.NotEmpty(t, cmd.Short)
		assert.Contains(t, cmd.Short, "calendar")
	})

	t.Run("has_subcommands", func(t *testing.T) {
		subcommands := cmd.Commands()
		assert.NotEmpty(t, subcommands)
	})

	t.Run("has_required_subcommands", func(t *testing.T) {
		expectedCmds := []string{"list", "show", "create", "update", "delete", "events", "availability"}

		cmdMap := make(map[string]bool)
		for _, sub := range cmd.Commands() {
			cmdMap[sub.Name()] = true
		}

		for _, expected := range expectedCmds {
			assert.True(t, cmdMap[expected], "Missing expected subcommand: %s", expected)
		}
	})
}

func TestShowCmd(t *testing.T) {
	cmd := newShowCmd()

	t.Run("command_name", func(t *testing.T) {
		assert.Equal(t, "show <calendar-id> [grant-id]", cmd.Use)
	})

	t.Run("has_short_description", func(t *testing.T) {
		assert.NotEmpty(t, cmd.Short)
		assert.Contains(t, cmd.Short, "Show")
	})

	t.Run("requires_calendar_id_arg", func(t *testing.T) {
		// Args validator should require 1-2 args
		assert.NotNil(t, cmd.Args)
	})
}

func TestCreateCmd(t *testing.T) {
	cmd := newCreateCmd()

	t.Run("command_name", func(t *testing.T) {
		assert.Equal(t, "create <name> [grant-id]", cmd.Use)
	})

	t.Run("has_short_description", func(t *testing.T) {
		assert.NotEmpty(t, cmd.Short)
		assert.Contains(t, cmd.Short, "Create")
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

	t.Run("has_location_flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("location")
		assert.NotNil(t, flag)
	})

	t.Run("has_timezone_flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("timezone")
		assert.NotNil(t, flag)
	})
}

func TestUpdateCmd(t *testing.T) {
	cmd := newUpdateCmd()

	t.Run("command_name", func(t *testing.T) {
		assert.Equal(t, "update <calendar-id> [grant-id]", cmd.Use)
	})

	t.Run("has_short_description", func(t *testing.T) {
		assert.NotEmpty(t, cmd.Short)
		assert.Contains(t, cmd.Short, "Update")
	})

	t.Run("has_name_flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("name")
		assert.NotNil(t, flag)
	})

	t.Run("has_description_flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("description")
		assert.NotNil(t, flag)
	})

	t.Run("has_location_flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("location")
		assert.NotNil(t, flag)
	})

	t.Run("has_timezone_flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("timezone")
		assert.NotNil(t, flag)
	})

	t.Run("has_color_flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("color")
		assert.NotNil(t, flag)
	})
}

func TestDeleteCmd(t *testing.T) {
	cmd := newDeleteCmd()

	t.Run("command_name", func(t *testing.T) {
		assert.Equal(t, "delete <calendar-id> [grant-id]", cmd.Use)
	})

	t.Run("has_short_description", func(t *testing.T) {
		assert.NotEmpty(t, cmd.Short)
		assert.Contains(t, cmd.Short, "Delete")
	})

	t.Run("has_force_flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("force")
		assert.NotNil(t, flag)
	})

	t.Run("has_force_shorthand", func(t *testing.T) {
		flag := cmd.Flags().ShorthandLookup("f")
		assert.NotNil(t, flag)
		assert.Equal(t, "force", flag.Name)
	})
}

func TestListCmd(t *testing.T) {
	cmd := newListCmd()

	t.Run("command_name", func(t *testing.T) {
		assert.Equal(t, "list [grant-id]", cmd.Use)
	})

	t.Run("has_aliases", func(t *testing.T) {
		assert.Contains(t, cmd.Aliases, "ls")
	})

	t.Run("has_short_description", func(t *testing.T) {
		assert.NotEmpty(t, cmd.Short)
		assert.Contains(t, cmd.Short, "List")
	})
}

func TestEventsCmd(t *testing.T) {
	cmd := newEventsCmd()

	t.Run("command_name", func(t *testing.T) {
		assert.Equal(t, "events", cmd.Use)
	})

	t.Run("has_aliases", func(t *testing.T) {
		assert.Contains(t, cmd.Aliases, "ev")
		assert.Contains(t, cmd.Aliases, "event")
	})

	t.Run("has_subcommands", func(t *testing.T) {
		subcommands := cmd.Commands()
		assert.NotEmpty(t, subcommands)
	})

	t.Run("has_required_subcommands", func(t *testing.T) {
		expectedCmds := []string{"list", "show", "create", "update", "delete", "rsvp"}

		cmdMap := make(map[string]bool)
		for _, sub := range cmd.Commands() {
			cmdMap[sub.Name()] = true
		}

		for _, expected := range expectedCmds {
			assert.True(t, cmdMap[expected], "Missing expected subcommand: %s", expected)
		}
	})
}
