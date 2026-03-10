package calendar

import (
	"testing"
	"time"
)

func TestFormatTimezoneBadge(t *testing.T) {
	tests := []struct {
		name            string
		tz              string
		useAbbreviation bool
		wantContains    string
		wantEmpty       bool
	}{
		{
			name:            "empty timezone",
			tz:              "",
			useAbbreviation: false,
			wantEmpty:       true,
		},
		{
			name:            "full timezone name",
			tz:              "America/New_York",
			useAbbreviation: false,
			wantContains:    "[America/New_York]",
		},
		{
			name:            "timezone abbreviation",
			tz:              "America/Los_Angeles",
			useAbbreviation: true,
			wantContains:    "[P", // PST or PDT
		},
		{
			name:            "UTC timezone",
			tz:              "UTC",
			useAbbreviation: true,
			wantContains:    "[UTC]",
		},
		{
			name:            "Europe timezone abbreviation",
			tz:              "Europe/London",
			useAbbreviation: true,
			wantContains:    "[", // GMT or BST
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatTimezoneBadge(tt.tz, tt.useAbbreviation)

			if tt.wantEmpty {
				if got != "" {
					t.Errorf("formatTimezoneBadge() = %q, want empty string", got)
				}
				return
			}

			if len(got) == 0 {
				t.Errorf("formatTimezoneBadge() = empty, want %q", tt.wantContains)
				return
			}

			// Badge should start with '[' and end with ']'
			if got[0] != '[' || got[len(got)-1] != ']' {
				t.Errorf("formatTimezoneBadge() = %q, want format [...]", got)
			}

			// Check if contains expected substring (for partial matches)
			if tt.wantContains != "" {
				found := false
				for i := 0; i <= len(got)-len(tt.wantContains); i++ {
					if got[i:i+len(tt.wantContains)] == tt.wantContains {
						found = true
						break
					}
				}
				if !found && got != tt.wantContains {
					t.Errorf("formatTimezoneBadge() = %q, want to contain %q", got, tt.wantContains)
				}
			}
		})
	}
}

// expectedColorForTZ computes the expected color code for a timezone using the
// same offset-based logic as getTimezoneColor. This avoids hardcoding values
// that change with DST transitions.
func expectedColorForTZ(tz string) int {
	if tz == "" {
		return 7
	}
	loc, err := time.LoadLocation(tz)
	if err != nil {
		return 7
	}
	_, offset := time.Now().In(loc).Zone()
	offsetHours := offset / 3600
	switch {
	case offsetHours <= -8:
		return 34
	case offsetHours <= -5:
		return 36
	case offsetHours <= 0:
		return 32
	case offsetHours <= 3:
		return 33
	case offsetHours <= 12:
		return 35
	default:
		return 31
	}
}

func TestGetTimezoneColor(t *testing.T) {
	tests := []struct {
		name string
		tz   string
	}{
		{name: "empty timezone returns default", tz: ""},
		{name: "Pacific timezone (PST/PDT)", tz: "America/Los_Angeles"},
		{name: "Eastern timezone (EST/EDT)", tz: "America/New_York"},
		{name: "UTC timezone", tz: "UTC"},
		{name: "Europe timezone", tz: "Europe/London"},
		{name: "Asia timezone", tz: "Asia/Tokyo"},
		{name: "India timezone", tz: "Asia/Kolkata"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getTimezoneColor(tt.tz)
			want := expectedColorForTZ(tt.tz)

			if got != want {
				t.Errorf("getTimezoneColor(%q) = %d, want %d", tt.tz, got, want)
			}

			// Verify it's a valid ANSI color code
			validCodes := []int{7, 31, 32, 33, 34, 35, 36}
			isValid := false
			for _, code := range validCodes {
				if got == code {
					isValid = true
					break
				}
			}
			if !isValid {
				t.Errorf("getTimezoneColor(%q) = %d, not a valid ANSI color code", tt.tz, got)
			}
		})
	}
}
