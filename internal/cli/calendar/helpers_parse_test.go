package calendar

import (
	"testing"
	"time"
)

func TestParseNaturalTime(t *testing.T) {
	// Note: Tests use current time, so relative time tests may vary
	// Reference: Wednesday, Jan 15, 2025, 10:00 AM EST would be used for deterministic testing

	tests := []struct {
		name      string
		input     string
		tz        string
		wantError bool
		checkTime func(*testing.T, time.Time)
	}{
		{
			name:      "empty input returns error",
			input:     "",
			tz:        "America/New_York",
			wantError: true,
		},
		{
			name:      "invalid timezone returns error",
			input:     "tomorrow at 3pm",
			tz:        "Invalid/Zone",
			wantError: true,
		},
		{
			name:  "relative time - in 2 hours",
			input: "in 2 hours",
			tz:    "America/New_York",
			checkTime: func(t *testing.T, result time.Time) {
				// Check that time is roughly 2 hours from now
				now := time.Now()
				expected := now.Add(2 * time.Hour)
				diff := result.Sub(expected)
				if diff < -time.Minute || diff > time.Minute {
					t.Errorf("Expected time ~2 hours from now, got %v (diff: %v)", result, diff)
				}
			},
		},
		{
			name:  "relative time - in 30 minutes",
			input: "in 30 minutes",
			tz:    "America/New_York",
			checkTime: func(t *testing.T, result time.Time) {
				// Check that time is roughly 30 minutes from now
				now := time.Now()
				expected := now.Add(30 * time.Minute)
				diff := result.Sub(expected)
				if diff < -time.Minute || diff > time.Minute {
					t.Errorf("Expected time ~30 minutes from now, got %v (diff: %v)", result, diff)
				}
			},
		},
		{
			name:  "relative day - tomorrow at 3pm",
			input: "tomorrow at 3pm",
			tz:    "America/New_York",
			checkTime: func(t *testing.T, result time.Time) {
				// Check it's tomorrow and at 3pm
				// Use the same timezone as parseNaturalTime to avoid CI/CD timezone issues
				loc, _ := time.LoadLocation("America/New_York")
				now := time.Now().In(loc)
				if result.Day() != now.AddDate(0, 0, 1).Day() || result.Hour() != 15 {
					t.Errorf("Expected tomorrow at 15:00, got %v at %02d:00", result.Day(), result.Hour())
				}
			},
		},
		{
			name:  "relative day - today at 2:30pm",
			input: "today at 2:30pm",
			tz:    "America/New_York",
			checkTime: func(t *testing.T, result time.Time) {
				// Check it's today at 2:30pm
				// Use the same timezone as parseNaturalTime to avoid CI/CD timezone issues
				loc, _ := time.LoadLocation("America/New_York")
				now := time.Now().In(loc)
				if result.Day() != now.Day() || result.Hour() != 14 || result.Minute() != 30 {
					t.Errorf("Expected today at 14:30, got %v at %02d:%02d", result.Day(), result.Hour(), result.Minute())
				}
			},
		},
		{
			name:  "specific weekday - next tuesday 2pm",
			input: "next tuesday 2pm",
			tz:    "America/New_York",
			checkTime: func(t *testing.T, result time.Time) {
				// Check it's a Tuesday and at 2pm
				if result.Weekday() != time.Tuesday {
					t.Errorf("Expected Tuesday, got %v", result.Weekday())
				}
				if result.Hour() != 14 {
					t.Errorf("Expected 14:00, got %02d:00", result.Hour())
				}
				// Check it's in the future
				if !result.After(time.Now()) {
					t.Error("Expected future date")
				}
			},
		},
		{
			name:  "absolute time - dec 25 10:00 am",
			input: "Dec 25 10:00 AM",
			tz:    "America/New_York",
			checkTime: func(t *testing.T, result time.Time) {
				if result.Month() != time.December || result.Day() != 25 || result.Hour() != 10 {
					t.Errorf("Expected Dec 25 10:00, got %v %v %02d:00", result.Month(), result.Day(), result.Hour())
				}
			},
		},
		{
			name:  "ISO time - 2025-03-15 14:00",
			input: "2025-03-15 14:00",
			tz:    "America/New_York",
			checkTime: func(t *testing.T, result time.Time) {
				if result.Year() != 2025 || result.Month() != time.March || result.Day() != 15 || result.Hour() != 14 {
					t.Errorf("Expected 2025-03-15 14:00, got %v", result)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Mock time.Now() by using the parseNaturalTime implementation
			// For these tests, we'll test the individual parser functions
			result, err := parseNaturalTime(tt.input, tt.tz)

			if tt.wantError {
				if err == nil {
					t.Error("Expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if result == nil {
				t.Error("Expected result, got nil")
				return
			}

			if tt.checkTime != nil {
				tt.checkTime(t, result.Time)
			}

			if result.Original != tt.input {
				t.Errorf("Original = %q, want %q", result.Original, tt.input)
			}
		})
	}
}

func TestExtractTimezoneFromInput(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		wantTZ         string
		wantCleanInput string
	}{
		{
			name:           "3pm PST extracts Pacific time",
			input:          "3pm PST",
			wantTZ:         "America/Los_Angeles",
			wantCleanInput: "3pm",
		},
		{
			name:           "3pm pst lowercase",
			input:          "3pm pst",
			wantTZ:         "America/Los_Angeles",
			wantCleanInput: "3pm",
		},
		{
			name:           "2:30pm EST extracts Eastern time",
			input:          "2:30pm EST",
			wantTZ:         "America/New_York",
			wantCleanInput: "2:30pm",
		},
		{
			name:           "14:00 UTC extracts UTC",
			input:          "14:00 UTC",
			wantTZ:         "UTC",
			wantCleanInput: "14:00",
		},
		{
			name:           "3pm without timezone returns nil",
			input:          "3pm",
			wantTZ:         "",
			wantCleanInput: "3pm",
		},
		{
			name:           "10am JST extracts Japan time",
			input:          "10am JST",
			wantTZ:         "Asia/Tokyo",
			wantCleanInput: "10am",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			loc, cleanInput := extractTimezoneFromInput(tt.input)

			if tt.wantTZ == "" {
				if loc != nil {
					t.Errorf("Expected nil location, got %v", loc)
				}
			} else {
				if loc == nil {
					t.Errorf("Expected location %s, got nil", tt.wantTZ)
				} else if loc.String() != tt.wantTZ {
					t.Errorf("Location = %s, want %s", loc.String(), tt.wantTZ)
				}
			}

			if cleanInput != tt.wantCleanInput {
				t.Errorf("CleanInput = %q, want %q", cleanInput, tt.wantCleanInput)
			}
		})
	}
}

func TestParseTimeOfDay(t *testing.T) {
	loc, _ := time.LoadLocation("America/New_York")

	tests := []struct {
		name      string
		input     string
		wantHour  int
		wantMin   int
		wantError bool
	}{
		{
			name:     "3pm",
			input:    "3pm",
			wantHour: 15,
			wantMin:  0,
		},
		{
			name:     "3PM (uppercase)",
			input:    "3PM",
			wantHour: 15,
			wantMin:  0,
		},
		{
			name:     "2:30pm",
			input:    "2:30pm",
			wantHour: 14,
			wantMin:  30,
		},
		{
			name:     "2:30 PM (with space)",
			input:    "2:30 PM",
			wantHour: 14,
			wantMin:  30,
		},
		{
			name:     "14:00 (24-hour)",
			input:    "14:00",
			wantHour: 14,
			wantMin:  0,
		},
		{
			name:     "3pm PST (with timezone)",
			input:    "3pm PST",
			wantHour: 15,
			wantMin:  0,
		},
		{
			name:     "2:30pm EST (with timezone)",
			input:    "2:30pm EST",
			wantHour: 14,
			wantMin:  30,
		},
		{
			name:      "invalid format",
			input:     "invalid",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseTimeOfDay(tt.input, loc)

			if tt.wantError {
				if err == nil {
					t.Error("Expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if result.Hour() != tt.wantHour {
				t.Errorf("Hour = %d, want %d", result.Hour(), tt.wantHour)
			}

			if result.Minute() != tt.wantMin {
				t.Errorf("Minute = %d, want %d", result.Minute(), tt.wantMin)
			}
		})
	}
}

func TestNormalizeTimeString(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "uppercase to lowercase",
			input: "TOMORROW AT 3PM",
			want:  "tomorrow at 3pm",
		},
		{
			name:  "extra whitespace removed",
			input: "  tomorrow   at   3pm  ",
			want:  "tomorrow at 3pm",
		},
		{
			name:  "mixed case normalized",
			input: "Next Tuesday 2PM",
			want:  "next tuesday 2pm",
		},
		{
			name:  "already normalized",
			input: "tomorrow at 3pm",
			want:  "tomorrow at 3pm",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizeTimeString(tt.input)
			if result != tt.want {
				t.Errorf("normalizeTimeString() = %q, want %q", result, tt.want)
			}
		})
	}
}
