package calendar

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestResourcesCommand(t *testing.T) {
	cmd := newResourcesCmd()

	t.Run("command_name", func(t *testing.T) {
		assert.Equal(t, "resources [grant-id]", cmd.Use)
	})

	t.Run("has_rooms_alias", func(t *testing.T) {
		assert.Contains(t, cmd.Aliases, "rooms")
	})

	t.Run("explains_email_as_calendar_id", func(t *testing.T) {
		// Callers reuse a resource's email as a calendar ID; the help must
		// surface that so the command is actually useful.
		assert.Contains(t, cmd.Long, "calendar ID")
	})
}

func TestCalendarCmd_RegistersResources(t *testing.T) {
	cmd := NewCalendarCmd()
	names := make(map[string]bool)
	for _, sub := range cmd.Commands() {
		names[sub.Name()] = true
	}
	assert.True(t, names["resources"], "calendar command must register the resources subcommand")
}
