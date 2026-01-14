//go:build !integration

package calendar

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseISOTime(t *testing.T) {
	loc, err := time.LoadLocation("America/New_York")
	require.NoError(t, err)
	now := time.Now().In(loc)

	tests := []struct {
		name      string
		input     string
		wantYear  int
		wantMonth time.Month
		wantDay   int
		wantHour  int
		wantMin   int
		wantErr   bool
	}{
		{
			name:      "RFC3339 format",
			input:     "2025-03-15T14:30:00-05:00",
			wantYear:  2025,
			wantMonth: time.March,
			wantDay:   15,
			wantHour:  14,
			wantMin:   30,
			wantErr:   false,
		},
		{
			name:      "ISO format with T separator",
			input:     "2025-03-15T14:30:00",
			wantYear:  2025,
			wantMonth: time.March,
			wantDay:   15,
			wantHour:  14,
			wantMin:   30,
			wantErr:   false,
		},
		{
			name:      "ISO format with space separator",
			input:     "2025-03-15 14:30:00",
			wantYear:  2025,
			wantMonth: time.March,
			wantDay:   15,
			wantHour:  14,
			wantMin:   30,
			wantErr:   false,
		},
		{
			name:      "ISO format with T and no seconds",
			input:     "2025-03-15T14:30",
			wantYear:  2025,
			wantMonth: time.March,
			wantDay:   15,
			wantHour:  14,
			wantMin:   30,
			wantErr:   false,
		},
		{
			name:      "ISO format with space and no seconds",
			input:     "2025-03-15 14:30",
			wantYear:  2025,
			wantMonth: time.March,
			wantDay:   15,
			wantHour:  14,
			wantMin:   30,
			wantErr:   false,
		},
		{
			name:      "December date",
			input:     "2025-12-25T10:00:00",
			wantYear:  2025,
			wantMonth: time.December,
			wantDay:   25,
			wantHour:  10,
			wantMin:   0,
			wantErr:   false,
		},
		{
			name:      "January 1st",
			input:     "2026-01-01T00:00:00",
			wantYear:  2026,
			wantMonth: time.January,
			wantDay:   1,
			wantHour:  0,
			wantMin:   0,
			wantErr:   false,
		},
		{
			name:    "invalid format",
			input:   "not a date",
			wantErr: true,
		},
		{
			name:    "invalid ISO format",
			input:   "2025/03/15 14:30",
			wantErr: true,
		},
		{
			name:    "date only without time",
			input:   "2025-03-15",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseISOTime(tt.input, loc, now)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, result)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, result)

			assert.Equal(t, tt.wantYear, result.Time.Year())
			assert.Equal(t, tt.wantMonth, result.Time.Month())
			assert.Equal(t, tt.wantDay, result.Time.Day())
			assert.Equal(t, tt.wantHour, result.Time.Hour())
			assert.Equal(t, tt.wantMin, result.Time.Minute())
			assert.Equal(t, loc.String(), result.Timezone)
		})
	}
}

func TestParseRelativeTime_Extended(t *testing.T) {
	loc, err := time.LoadLocation("America/New_York")
	require.NoError(t, err)
	now := time.Now().In(loc)

	tests := []struct {
		name          string
		input         string
		expectedDelta time.Duration
		tolerance     time.Duration
		wantErr       bool
	}{
		{
			name:          "in 1 hour",
			input:         "in 1 hour",
			expectedDelta: 1 * time.Hour,
			tolerance:     time.Second,
			wantErr:       false,
		},
		{
			name:          "in 5 hours",
			input:         "in 5 hours",
			expectedDelta: 5 * time.Hour,
			tolerance:     time.Second,
			wantErr:       false,
		},
		{
			name:          "in 15 minutes",
			input:         "in 15 minutes",
			expectedDelta: 15 * time.Minute,
			tolerance:     time.Second,
			wantErr:       false,
		},
		{
			name:          "in 1 day",
			input:         "in 1 day",
			expectedDelta: 24 * time.Hour,
			tolerance:     time.Second,
			wantErr:       false,
		},
		{
			name:          "in 2 days",
			input:         "in 2 days",
			expectedDelta: 48 * time.Hour,
			tolerance:     time.Second,
			wantErr:       false,
		},
		{
			name:    "invalid relative time",
			input:   "in abc hours",
			wantErr: true,
		},
		{
			name:    "not relative format",
			input:   "tomorrow at 3pm",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseRelativeTime(tt.input, loc, now)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, result)

			expected := now.Add(tt.expectedDelta)
			diff := result.Time.Sub(expected)
			assert.True(t, diff >= -tt.tolerance && diff <= tt.tolerance,
				"Expected time around %v, got %v (diff: %v)", expected, result.Time, diff)
		})
	}
}

func TestParseAbsoluteTime_Extended(t *testing.T) {
	loc, err := time.LoadLocation("America/New_York")
	require.NoError(t, err)
	now := time.Now().In(loc)

	tests := []struct {
		name      string
		input     string
		wantMonth time.Month
		wantDay   int
		wantHour  int
		wantMin   int
		wantErr   bool
	}{
		// Note: Go time parsing with lowercase "jan" only matches literal "jan"
		// For other months, use titlecase formats like "Jan" or "January"
		{
			name:      "jan 15 3:00 pm lowercase",
			input:     "jan 15 3:00 pm",
			wantMonth: time.January,
			wantDay:   15,
			wantHour:  15,
			wantMin:   0,
			wantErr:   false,
		},
		{
			name:      "jan 5 9:30 am lowercase",
			input:     "jan 5 9:30 am",
			wantMonth: time.January,
			wantDay:   5,
			wantHour:  9,
			wantMin:   30,
			wantErr:   false,
		},
		{
			name:      "jan 1 8:00 am lowercase",
			input:     "jan 1 8:00 am",
			wantMonth: time.January,
			wantDay:   1,
			wantHour:  8,
			wantMin:   0,
			wantErr:   false,
		},
		{
			name:      "jan 25 2:15 pm lowercase",
			input:     "jan 25 2:15 pm",
			wantMonth: time.January,
			wantDay:   25,
			wantHour:  14,
			wantMin:   15,
			wantErr:   false,
		},
		{
			name:      "Jan 4 3:00 PM titlecase",
			input:     "Jan 4 3:00 PM",
			wantMonth: time.January,
			wantDay:   4,
			wantHour:  15,
			wantMin:   0,
			wantErr:   false,
		},
		{
			name:      "February 10 3:00 PM full month",
			input:     "February 10 3:00 PM",
			wantMonth: time.February,
			wantDay:   10,
			wantHour:  15,
			wantMin:   0,
			wantErr:   false,
		},
		{
			name:    "invalid month",
			input:   "foo 15 3:00 pm",
			wantErr: true,
		},
		{
			name:    "no time specified",
			input:   "jan 15",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseAbsoluteTime(tt.input, loc, now)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, result)

			assert.Equal(t, tt.wantMonth, result.Time.Month())
			assert.Equal(t, tt.wantDay, result.Time.Day())
			assert.Equal(t, tt.wantHour, result.Time.Hour())
			assert.Equal(t, tt.wantMin, result.Time.Minute())
		})
	}
}
