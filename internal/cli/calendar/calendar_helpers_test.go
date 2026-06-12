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
		when, err := parseEventTime("2024-01-15", "", true, "")
		assert.NoError(t, err)
		assert.Equal(t, "date", when.Object)
		assert.Equal(t, "2024-01-15", when.Date)
	})

	t.Run("parses_timed_event", func(t *testing.T) {
		when, err := parseEventTime("2024-01-15 14:00", "2024-01-15 15:00", false, "")
		assert.NoError(t, err)
		assert.Equal(t, "timespan", when.Object)
		assert.NotZero(t, when.StartTime)
		assert.NotZero(t, when.EndTime)
	})

	t.Run("defaults_end_to_one_hour", func(t *testing.T) {
		when, err := parseEventTime("2024-01-15 14:00", "", false, "")
		assert.NoError(t, err)
		assert.Equal(t, "timespan", when.Object)
		// End should be 1 hour after start
		assert.Equal(t, when.StartTime+3600, when.EndTime)
	})

	t.Run("parses_date_range", func(t *testing.T) {
		when, err := parseEventTime("2024-01-15", "2024-01-17", true, "")
		assert.NoError(t, err)
		assert.Equal(t, "datespan", when.Object)
		assert.Equal(t, "2024-01-15", when.StartDate)
		assert.Equal(t, "2024-01-17", when.EndDate)
	})

	t.Run("returns_error_for_invalid_start", func(t *testing.T) {
		_, err := parseEventTime("invalid", "", false, "")
		assert.Error(t, err)
	})

	t.Run("records_explicit_timezone_on_timed_event", func(t *testing.T) {
		when, err := parseEventTime("2024-01-15 14:00", "2024-01-15 15:00", false, "America/New_York")
		assert.NoError(t, err)
		assert.Equal(t, "timespan", when.Object)
		assert.Equal(t, "America/New_York", when.StartTimezone)
		assert.Equal(t, "America/New_York", when.EndTimezone)

		// Timestamps must agree with the recorded zone: 14:00 wall clock in NY
		loc, lerr := time.LoadLocation("America/New_York")
		assert.NoError(t, lerr)
		assert.Equal(t, time.Date(2024, 1, 15, 14, 0, 0, 0, loc).Unix(), when.StartTime)
		assert.Equal(t, time.Date(2024, 1, 15, 15, 0, 0, 0, loc).Unix(), when.EndTime)
	})

	t.Run("defaults_timezone_to_system_zone", func(t *testing.T) {
		when, err := parseEventTime("2024-01-15 14:00", "", false, "")
		assert.NoError(t, err)
		assert.Equal(t, getLocalTimeZone(), when.StartTimezone)
		assert.Equal(t, getLocalTimeZone(), when.EndTimezone)
	})

	t.Run("returns_error_for_invalid_timezone", func(t *testing.T) {
		_, err := parseEventTime("2024-01-15 14:00", "", false, "Not/AZone")
		assert.Error(t, err)
	})

	t.Run("all_day_does_not_set_timezone", func(t *testing.T) {
		when, err := parseEventTime("2024-01-15", "", true, "America/New_York")
		assert.NoError(t, err)
		assert.Empty(t, when.StartTimezone)
		assert.Empty(t, when.EndTimezone)
	})

	t.Run("rfc3339_offset_matching_timezone_accepted", func(t *testing.T) {
		// June 15: America/New_York is EDT (-04:00), so the offset agrees
		// with the recorded zone and the wall time is preserved.
		when, err := parseEventTime("2026-06-15T14:00:00-04:00", "", false, "America/New_York")
		assert.NoError(t, err)
		assert.Equal(t, "timespan", when.Object)
		assert.Equal(t, "America/New_York", when.StartTimezone)

		loc, lerr := time.LoadLocation("America/New_York")
		assert.NoError(t, lerr)
		assert.Equal(t, time.Date(2026, 6, 15, 14, 0, 0, 0, loc).Unix(), when.StartTime)
	})

	t.Run("rfc3339_offset_conflicting_timezone_errors", func(t *testing.T) {
		// +09:00 disagrees with America/New_York (-04:00 on June 15). The
		// epoch would follow the offset while start_timezone records New
		// York, so the event would display at a different wall time than
		// the user typed — reject instead of storing the mismatch.
		_, err := parseEventTime("2026-06-15T14:00:00+09:00", "", false, "America/New_York")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "offset")
	})

	t.Run("rfc3339_end_offset_conflicting_timezone_errors", func(t *testing.T) {
		_, err := parseEventTime("2026-06-15 14:00", "2026-06-15T16:00:00+09:00", false, "America/New_York")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "offset")
	})

	t.Run("rfc3339_dst_gap_offset_errors", func(t *testing.T) {
		// 02:30 EST on 2026-03-08 doesn't exist: clocks jump 02:00 EST →
		// 03:00 EDT, so at that UTC instant New York is already -04:00.
		_, err := parseEventTime("2026-03-08T02:30:00-05:00", "", false, "America/New_York")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "offset")
	})

	t.Run("rfc3339_dst_fold_offsets_accepted", func(t *testing.T) {
		// Both fall-back representations name real instants: 01:30-04:00 is
		// before the 06:00 UTC fallback (still EDT), 01:30-05:00 after (EST).
		_, err := parseEventTime("2026-11-01T01:30:00-04:00", "", false, "America/New_York")
		assert.NoError(t, err)
		_, err = parseEventTime("2026-11-01T01:30:00-05:00", "", false, "America/New_York")
		assert.NoError(t, err)
	})

	t.Run("rfc3339_z_with_utc_equivalent_zone_accepted", func(t *testing.T) {
		// Africa/Abidjan is permanently UTC+00:00, so a Z input agrees.
		when, err := parseEventTime("2026-06-15T14:00:00Z", "", false, "Africa/Abidjan")
		assert.NoError(t, err)
		assert.Equal(t, "Africa/Abidjan", when.StartTimezone)
	})

	t.Run("rfc3339_z_with_utc_timezone_accepted", func(t *testing.T) {
		when, err := parseEventTime("2026-06-15T14:00:00Z", "", false, "UTC")
		assert.NoError(t, err)
		assert.Equal(t, "UTC", when.StartTimezone)
		assert.Equal(t, time.Date(2026, 6, 15, 14, 0, 0, 0, time.UTC).Unix(), when.StartTime)
	})

	t.Run("all_day_with_time_component_errors", func(t *testing.T) {
		// --all-day must never silently create a timed event
		when, err := parseEventTime("2024-01-15 10:00", "", true, "")
		assert.Error(t, err)
		assert.Nil(t, when)
		assert.Contains(t, err.Error(), "all-day")
	})

	t.Run("locked_timezone_round_trip", func(t *testing.T) {
		// --lock-timezone relies on When.StartTimezone being populated:
		// GetLockedTimezone() must return the zone the event was created in.
		when, err := parseEventTime("2024-01-15 14:00", "", false, "Asia/Tokyo")
		assert.NoError(t, err)

		event := domain.Event{
			Metadata: map[string]string{"timezone_locked": "true"},
			When:     *when,
		}
		assert.Equal(t, "Asia/Tokyo", event.GetLockedTimezone())
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
