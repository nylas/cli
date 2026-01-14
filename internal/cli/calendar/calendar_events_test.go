package calendar

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEventsListCmd(t *testing.T) {
	cmd := newEventsListCmd()

	t.Run("command_name", func(t *testing.T) {
		assert.Equal(t, "list [grant-id]", cmd.Use)
	})

	t.Run("has_aliases", func(t *testing.T) {
		assert.Contains(t, cmd.Aliases, "ls")
	})

	t.Run("has_calendar_flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("calendar")
		assert.NotNil(t, flag)
	})

	t.Run("has_calendar_shorthand", func(t *testing.T) {
		flag := cmd.Flags().ShorthandLookup("c")
		assert.NotNil(t, flag)
		assert.Equal(t, "calendar", flag.Name)
	})

	t.Run("has_limit_flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("limit")
		assert.NotNil(t, flag)
		assert.Equal(t, "10", flag.DefValue)
	})

	t.Run("has_days_flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("days")
		assert.NotNil(t, flag)
		assert.Equal(t, "7", flag.DefValue)
	})

	t.Run("has_show_cancelled_flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("show-cancelled")
		assert.NotNil(t, flag)
	})
}

func TestEventsShowCmd(t *testing.T) {
	cmd := newEventsShowCmd()

	t.Run("command_name", func(t *testing.T) {
		assert.Equal(t, "show <event-id> [grant-id]", cmd.Use)
	})

	t.Run("has_aliases", func(t *testing.T) {
		assert.Contains(t, cmd.Aliases, "read")
		assert.Contains(t, cmd.Aliases, "get")
	})

	t.Run("has_calendar_flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("calendar")
		assert.NotNil(t, flag)
	})
}

func TestEventsCreateCmd(t *testing.T) {
	cmd := newEventsCreateCmd()

	t.Run("command_name", func(t *testing.T) {
		assert.Equal(t, "create [grant-id]", cmd.Use)
	})

	t.Run("has_title_flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("title")
		assert.NotNil(t, flag)
	})

	t.Run("has_title_shorthand", func(t *testing.T) {
		flag := cmd.Flags().ShorthandLookup("t")
		assert.NotNil(t, flag)
		assert.Equal(t, "title", flag.Name)
	})

	t.Run("has_description_flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("description")
		assert.NotNil(t, flag)
	})

	t.Run("has_location_flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("location")
		assert.NotNil(t, flag)
	})

	t.Run("has_start_flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("start")
		assert.NotNil(t, flag)
	})

	t.Run("has_end_flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("end")
		assert.NotNil(t, flag)
	})

	t.Run("has_all_day_flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("all-day")
		assert.NotNil(t, flag)
	})

	t.Run("has_participant_flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("participant")
		assert.NotNil(t, flag)
	})

	t.Run("has_busy_flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("busy")
		assert.NotNil(t, flag)
		assert.Equal(t, "true", flag.DefValue)
	})

	t.Run("has_calendar_flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("calendar")
		assert.NotNil(t, flag)
	})
}

func TestEventsDeleteCmd(t *testing.T) {
	cmd := newEventsDeleteCmd()

	t.Run("command_name", func(t *testing.T) {
		assert.Equal(t, "delete <event-id> [grant-id]", cmd.Use)
	})

	t.Run("has_aliases", func(t *testing.T) {
		assert.Contains(t, cmd.Aliases, "rm")
		assert.Contains(t, cmd.Aliases, "remove")
	})

	t.Run("has_force_flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("force")
		assert.NotNil(t, flag)
	})

	t.Run("has_calendar_flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("calendar")
		assert.NotNil(t, flag)
	})
}

func TestEventsUpdateCmd(t *testing.T) {
	cmd := newEventsUpdateCmd()

	t.Run("command_name", func(t *testing.T) {
		assert.Equal(t, "update <event-id> [grant-id]", cmd.Use)
	})

	t.Run("has_short_description", func(t *testing.T) {
		assert.NotEmpty(t, cmd.Short)
		assert.Contains(t, cmd.Short, "Update")
	})

	t.Run("has_title_flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("title")
		assert.NotNil(t, flag)
	})

	t.Run("has_title_shorthand", func(t *testing.T) {
		flag := cmd.Flags().ShorthandLookup("t")
		assert.NotNil(t, flag)
		assert.Equal(t, "title", flag.Name)
	})

	t.Run("has_description_flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("description")
		assert.NotNil(t, flag)
	})

	t.Run("has_location_flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("location")
		assert.NotNil(t, flag)
	})

	t.Run("has_start_flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("start")
		assert.NotNil(t, flag)
	})

	t.Run("has_end_flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("end")
		assert.NotNil(t, flag)
	})

	t.Run("has_all_day_flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("all-day")
		assert.NotNil(t, flag)
	})

	t.Run("has_participant_flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("participant")
		assert.NotNil(t, flag)
	})

	t.Run("has_busy_flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("busy")
		assert.NotNil(t, flag)
	})

	t.Run("has_visibility_flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("visibility")
		assert.NotNil(t, flag)
	})

	t.Run("has_calendar_flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("calendar")
		assert.NotNil(t, flag)
	})
}

func TestEventsRSVPCmd(t *testing.T) {
	cmd := newEventsRSVPCmd()

	t.Run("command_name", func(t *testing.T) {
		assert.Equal(t, "rsvp <event-id> <status> [grant-id]", cmd.Use)
	})

	t.Run("has_short_description", func(t *testing.T) {
		assert.NotEmpty(t, cmd.Short)
		assert.Contains(t, cmd.Short, "RSVP")
	})

	t.Run("has_long_description_with_status_options", func(t *testing.T) {
		assert.Contains(t, cmd.Long, "yes")
		assert.Contains(t, cmd.Long, "no")
		assert.Contains(t, cmd.Long, "maybe")
	})

	t.Run("has_calendar_flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("calendar")
		assert.NotNil(t, flag)
	})

	t.Run("has_comment_flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("comment")
		assert.NotNil(t, flag)
	})
}
