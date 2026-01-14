//go:build !integration

package common

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseTimeOfDay(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantHour int
		wantMin  int
		wantErr  bool
	}{
		// 24-hour format
		{"24h with colon", "14:30", 14, 30, false},
		{"24h midnight", "00:00", 0, 0, false},
		{"24h noon", "12:00", 12, 0, false},
		{"24h end of day", "23:59", 23, 59, false},
		{"24h morning", "09:15", 9, 15, false},

		// 12-hour format with pm/am attached
		{"12h pm lowercase", "2:30pm", 14, 30, false},
		{"12h am lowercase", "9:00am", 9, 0, false},
		{"12h noon pm", "12:00pm", 12, 0, false},
		{"12h midnight am", "12:00am", 0, 0, false},

		// 12-hour format with space before pm/am
		{"12h pm with space", "2:30 pm", 14, 30, false},
		{"12h am with space", "9:00 am", 9, 0, false},

		// Hour only formats
		{"hour only pm", "3pm", 15, 0, false},
		{"hour only am", "9am", 9, 0, false},
		{"hour only pm space", "3 pm", 15, 0, false},
		{"hour only am space", "9 am", 9, 0, false},

		// Case insensitivity
		{"uppercase PM", "2:30PM", 14, 30, false},
		{"uppercase AM", "9:00AM", 9, 0, false},
		{"mixed case Pm", "2:30Pm", 14, 30, false},

		// Whitespace handling
		{"leading space", " 9am", 9, 0, false},
		{"trailing space", "9am ", 9, 0, false},
		{"both spaces", " 9am ", 9, 0, false},

		// Invalid formats
		{"invalid format", "not a time", 0, 0, true},
		{"empty string", "", 0, 0, true},
		{"just number", "14", 0, 0, true},
		{"invalid separator", "14.30", 0, 0, true},
		{"out of range hour", "25:00", 0, 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseTimeOfDay(tt.input)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "invalid time format")
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.wantHour, result.Hour())
				assert.Equal(t, tt.wantMin, result.Minute())
			}
		})
	}
}

func TestParseTimeOfDayInLocation(t *testing.T) {
	loc, err := time.LoadLocation("America/New_York")
	require.NoError(t, err)

	tests := []struct {
		name     string
		input    string
		loc      *time.Location
		wantHour int
		wantMin  int
	}{
		{"with location", "14:30", loc, 14, 30},
		{"nil location uses UTC", "14:30", nil, 14, 30},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseTimeOfDayInLocation(tt.input, tt.loc)
			require.NoError(t, err)

			assert.Equal(t, tt.wantHour, result.Hour())
			assert.Equal(t, tt.wantMin, result.Minute())
		})
	}
}

func TestParseDuration(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected time.Duration
		wantErr  bool
	}{
		// Standard Go duration formats
		{"seconds", "30s", 30 * time.Second, false},
		{"minutes", "5m", 5 * time.Minute, false},
		{"hours", "2h", 2 * time.Hour, false},
		{"complex duration", "1h30m", 90 * time.Minute, false},
		{"microseconds", "500ms", 500 * time.Millisecond, false},

		// Day format (extended)
		{"1 day", "1d", 24 * time.Hour, false},
		{"7 days", "7d", 7 * 24 * time.Hour, false},
		{"30 days", "30d", 30 * 24 * time.Hour, false},

		// Week format (extended)
		{"1 week", "1w", 7 * 24 * time.Hour, false},
		{"2 weeks", "2w", 14 * 24 * time.Hour, false},
		{"4 weeks", "4w", 28 * 24 * time.Hour, false},

		// Case insensitivity
		{"uppercase D", "7D", 7 * 24 * time.Hour, false},
		{"uppercase W", "2W", 14 * 24 * time.Hour, false},
		{"uppercase H", "2H", 2 * time.Hour, false},

		// Whitespace handling
		{"leading space", " 30m", 30 * time.Minute, false},
		{"trailing space", "30m ", 30 * time.Minute, false},

		// Invalid formats
		{"empty string", "", 0, true},
		{"no unit", "30", 0, true},
		{"invalid day number", "xd", 0, true},
		{"invalid week number", "yw", 0, true},
		{"invalid format", "not-a-duration", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseDuration(tt.input)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestParseDuration_EdgeCases(t *testing.T) {
	t.Run("zero duration", func(t *testing.T) {
		result, err := ParseDuration("0s")
		require.NoError(t, err)
		assert.Equal(t, time.Duration(0), result)
	})

	t.Run("zero days", func(t *testing.T) {
		result, err := ParseDuration("0d")
		require.NoError(t, err)
		assert.Equal(t, time.Duration(0), result)
	})

	t.Run("zero weeks", func(t *testing.T) {
		result, err := ParseDuration("0w")
		require.NoError(t, err)
		assert.Equal(t, time.Duration(0), result)
	})

	t.Run("large number of days", func(t *testing.T) {
		result, err := ParseDuration("365d")
		require.NoError(t, err)
		assert.Equal(t, 365*24*time.Hour, result)
	})
}

func TestFormatTimeAgo_EdgeCases(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name     string
		offset   time.Duration
		expected string
	}{
		{"exactly 1 second ago", 1 * time.Second, "just now"},
		{"exactly 59 seconds ago", 59 * time.Second, "just now"},
		{"exactly 60 seconds ago", 60 * time.Second, "1 minute ago"},
		{"exactly 59 minutes ago", 59 * time.Minute, "59 minutes ago"},
		{"exactly 60 minutes ago", 60 * time.Minute, "1 hour ago"},
		{"exactly 23 hours ago", 23 * time.Hour, "23 hours ago"},
		{"exactly 6 days ago", 6 * 24 * time.Hour, "6 days ago"},
		{"exactly 4 weeks ago", 4 * 7 * 24 * time.Hour, "4 weeks ago"},
		{"exactly 11 months ago", 330 * 24 * time.Hour, "11 months ago"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatTimeAgo(now.Add(-tt.offset))
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFormatTimeAgo_FutureTime(t *testing.T) {
	// Future times should return "just now" since diff would be negative
	future := time.Now().Add(1 * time.Hour)
	result := FormatTimeAgo(future)
	// The function doesn't handle future times specially,
	// but should not panic
	assert.NotEmpty(t, result)
}
