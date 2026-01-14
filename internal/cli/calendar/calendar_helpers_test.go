package calendar

import (
	"testing"
	"time"

	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/domain"
	"github.com/stretchr/testify/assert"
)

func TestParseTimeInput(t *testing.T) {
	t.Run("parses_tomorrow", func(t *testing.T) {
		result, err := parseTimeInput("tomorrow")
		assert.NoError(t, err)
		expected := time.Now().AddDate(0, 0, 1)
		assert.Equal(t, expected.Day(), result.Day())
		assert.Equal(t, expected.Month(), result.Month())
	})

	t.Run("parses_tomorrow_with_time", func(t *testing.T) {
		result, err := parseTimeInput("tomorrow 9am")
		assert.NoError(t, err)
		expected := time.Now().AddDate(0, 0, 1)
		assert.Equal(t, expected.Day(), result.Day())
		assert.Equal(t, 9, result.Hour())
	})

	t.Run("parses_today", func(t *testing.T) {
		result, err := parseTimeInput("today")
		assert.NoError(t, err)
		now := time.Now()
		assert.Equal(t, now.Day(), result.Day())
		assert.Equal(t, now.Month(), result.Month())
	})

	t.Run("parses_iso_datetime", func(t *testing.T) {
		result, err := parseTimeInput("2024-01-15 14:30")
		assert.NoError(t, err)
		assert.Equal(t, 2024, result.Year())
		assert.Equal(t, time.January, result.Month())
		assert.Equal(t, 15, result.Day())
		assert.Equal(t, 14, result.Hour())
		assert.Equal(t, 30, result.Minute())
	})

	t.Run("parses_time_only", func(t *testing.T) {
		result, err := parseTimeInput("15:00")
		assert.NoError(t, err)
		assert.Equal(t, 15, result.Hour())
		assert.Equal(t, 0, result.Minute())
	})

	t.Run("returns_error_for_invalid_input", func(t *testing.T) {
		_, err := parseTimeInput("invalid time string xyz")
		assert.Error(t, err)
	})
}

func TestParseDuration(t *testing.T) {
	t.Run("parses_hours", func(t *testing.T) {
		result, err := common.ParseDuration("8h")
		assert.NoError(t, err)
		assert.Equal(t, 8*time.Hour, result)
	})

	t.Run("parses_days", func(t *testing.T) {
		result, err := common.ParseDuration("7d")
		assert.NoError(t, err)
		assert.Equal(t, 7*24*time.Hour, result)
	})

	t.Run("parses_minutes", func(t *testing.T) {
		result, err := common.ParseDuration("30m")
		assert.NoError(t, err)
		assert.Equal(t, 30*time.Minute, result)
	})

	t.Run("returns_error_for_invalid", func(t *testing.T) {
		_, err := common.ParseDuration("invalid")
		assert.Error(t, err)
	})
}

func TestParseEventTime(t *testing.T) {
	t.Run("parses_all_day_event", func(t *testing.T) {
		when, err := parseEventTime("2024-01-15", "", true)
		assert.NoError(t, err)
		assert.Equal(t, "date", when.Object)
		assert.Equal(t, "2024-01-15", when.Date)
	})

	t.Run("parses_timed_event", func(t *testing.T) {
		when, err := parseEventTime("2024-01-15 14:00", "2024-01-15 15:00", false)
		assert.NoError(t, err)
		assert.Equal(t, "timespan", when.Object)
		assert.NotZero(t, when.StartTime)
		assert.NotZero(t, when.EndTime)
	})

	t.Run("defaults_end_to_one_hour", func(t *testing.T) {
		when, err := parseEventTime("2024-01-15 14:00", "", false)
		assert.NoError(t, err)
		assert.Equal(t, "timespan", when.Object)
		// End should be 1 hour after start
		assert.Equal(t, when.StartTime+3600, when.EndTime)
	})

	t.Run("parses_date_range", func(t *testing.T) {
		when, err := parseEventTime("2024-01-15", "2024-01-17", true)
		assert.NoError(t, err)
		assert.Equal(t, "datespan", when.Object)
		assert.Equal(t, "2024-01-15", when.StartDate)
		assert.Equal(t, "2024-01-17", when.EndDate)
	})

	t.Run("returns_error_for_invalid_start", func(t *testing.T) {
		_, err := parseEventTime("invalid", "", false)
		assert.Error(t, err)
	})
}

func TestFormatEventTime(t *testing.T) {
	t.Run("formats_all_day_event", func(t *testing.T) {
		when := domain.EventWhen{
			Object: "date",
			Date:   "2024-01-15",
		}
		result := formatEventTime(when)
		assert.Contains(t, result, "Jan 15, 2024")
		assert.Contains(t, result, "all day")
	})

	t.Run("formats_timed_event_same_day", func(t *testing.T) {
		start := time.Date(2024, 1, 15, 14, 0, 0, 0, time.Local)
		end := time.Date(2024, 1, 15, 15, 0, 0, 0, time.Local)
		when := domain.EventWhen{
			Object:    "timespan",
			StartTime: start.Unix(),
			EndTime:   end.Unix(),
		}
		result := formatEventTime(when)
		assert.Contains(t, result, "Jan 15, 2024")
		assert.Contains(t, result, "2:00 PM")
		assert.Contains(t, result, "3:00 PM")
	})
}

func TestFormatParticipantStatus(t *testing.T) {
	t.Run("formats_yes", func(t *testing.T) {
		result := formatParticipantStatus("yes")
		assert.Contains(t, result, "accepted")
	})

	t.Run("formats_no", func(t *testing.T) {
		result := formatParticipantStatus("no")
		assert.Contains(t, result, "declined")
	})

	t.Run("formats_maybe", func(t *testing.T) {
		result := formatParticipantStatus("maybe")
		assert.Contains(t, result, "tentative")
	})

	t.Run("formats_noreply", func(t *testing.T) {
		result := formatParticipantStatus("noreply")
		assert.Contains(t, result, "pending")
	})

	t.Run("empty_for_unknown", func(t *testing.T) {
		result := formatParticipantStatus("unknown")
		assert.Empty(t, result)
	})
}

func TestCalendarCommandHelp(t *testing.T) {
	cmd := NewCalendarCmd()
	stdout, _, err := executeCommand(cmd, "--help")

	assert.NoError(t, err)

	expectedStrings := []string{
		"calendar",
		"list",
		"events",
		"availability",
	}

	for _, expected := range expectedStrings {
		assert.Contains(t, stdout, expected, "Help output should contain %q", expected)
	}
}

func TestCalendarEventsHelp(t *testing.T) {
	cmd := NewCalendarCmd()
	stdout, _, err := executeCommand(cmd, "events", "--help")

	assert.NoError(t, err)
	assert.Contains(t, stdout, "events")
	assert.Contains(t, stdout, "list")
	assert.Contains(t, stdout, "show")
	assert.Contains(t, stdout, "create")
	assert.Contains(t, stdout, "update")
	assert.Contains(t, stdout, "delete")
	assert.Contains(t, stdout, "rsvp")
}

func TestCalendarCRUDHelp(t *testing.T) {
	cmd := NewCalendarCmd()
	stdout, _, err := executeCommand(cmd, "--help")

	assert.NoError(t, err)
	assert.Contains(t, stdout, "show")
	assert.Contains(t, stdout, "create")
	assert.Contains(t, stdout, "update")
	assert.Contains(t, stdout, "delete")
}

func TestCalendarAvailabilityHelp(t *testing.T) {
	cmd := NewCalendarCmd()
	stdout, _, err := executeCommand(cmd, "availability", "--help")

	assert.NoError(t, err)
	assert.Contains(t, stdout, "availability")
	assert.Contains(t, stdout, "check")
	assert.Contains(t, stdout, "find")
}
