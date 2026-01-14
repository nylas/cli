package domain

import (
	"testing"
	"time"
)

// =============================================================================
// EventWhen Tests
// =============================================================================

func TestEventWhen_StartDateTime(t *testing.T) {
	tests := []struct {
		name string
		when EventWhen
		want time.Time
	}{
		{
			name: "returns time from unix timestamp",
			when: EventWhen{
				StartTime: 1704067200, // 2024-01-01 00:00:00 UTC
			},
			want: time.Unix(1704067200, 0),
		},
		{
			name: "returns time with timezone conversion",
			when: EventWhen{
				StartTime:     1704067200, // 2024-01-01 00:00:00 UTC
				StartTimezone: "America/New_York",
			},
			want: func() time.Time {
				loc, _ := time.LoadLocation("America/New_York")
				return time.Unix(1704067200, 0).In(loc)
			}(),
		},
		{
			name: "returns date from Date field",
			when: EventWhen{
				Date: "2024-01-15",
			},
			want: func() time.Time {
				t, _ := time.Parse("2006-01-02", "2024-01-15")
				return t
			}(),
		},
		{
			name: "returns date from StartDate field",
			when: EventWhen{
				StartDate: "2024-02-20",
			},
			want: func() time.Time {
				t, _ := time.Parse("2006-01-02", "2024-02-20")
				return t
			}(),
		},
		{
			name: "returns zero time when no fields set",
			when: EventWhen{},
			want: time.Time{},
		},
		{
			name: "StartTime takes precedence over Date",
			when: EventWhen{
				StartTime: 1704067200,
				Date:      "2024-01-15",
			},
			want: time.Unix(1704067200, 0),
		},
		{
			name: "handles invalid timezone gracefully",
			when: EventWhen{
				StartTime:     1704067200,
				StartTimezone: "Invalid/Timezone",
			},
			want: time.Unix(1704067200, 0), // Falls back to UTC
		},
		{
			name: "Date takes precedence over StartDate",
			when: EventWhen{
				Date:      "2024-01-15",
				StartDate: "2024-02-20",
			},
			want: func() time.Time {
				t, _ := time.Parse("2006-01-02", "2024-01-15")
				return t
			}(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.when.StartDateTime()
			if !got.Equal(tt.want) {
				t.Errorf("StartDateTime() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEventWhen_EndDateTime(t *testing.T) {
	tests := []struct {
		name string
		when EventWhen
		want time.Time
	}{
		{
			name: "returns time from unix timestamp",
			when: EventWhen{
				EndTime: 1704153600, // 2024-01-02 00:00:00 UTC
			},
			want: time.Unix(1704153600, 0),
		},
		{
			name: "returns time with timezone conversion",
			when: EventWhen{
				EndTime:     1704153600,
				EndTimezone: "Europe/London",
			},
			want: func() time.Time {
				loc, _ := time.LoadLocation("Europe/London")
				return time.Unix(1704153600, 0).In(loc)
			}(),
		},
		{
			name: "returns date from EndDate field",
			when: EventWhen{
				EndDate: "2024-01-20",
			},
			want: func() time.Time {
				t, _ := time.Parse("2006-01-02", "2024-01-20")
				return t
			}(),
		},
		{
			name: "falls back to Date when EndDate empty",
			when: EventWhen{
				Date: "2024-01-15",
			},
			want: func() time.Time {
				t, _ := time.Parse("2006-01-02", "2024-01-15")
				return t
			}(),
		},
		{
			name: "returns zero time when no fields set",
			when: EventWhen{},
			want: time.Time{},
		},
		{
			name: "EndTime takes precedence over EndDate",
			when: EventWhen{
				EndTime: 1704153600,
				EndDate: "2024-01-20",
			},
			want: time.Unix(1704153600, 0),
		},
		{
			name: "handles invalid timezone gracefully",
			when: EventWhen{
				EndTime:     1704153600,
				EndTimezone: "NotA/Timezone",
			},
			want: time.Unix(1704153600, 0),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.when.EndDateTime()
			if !got.Equal(tt.want) {
				t.Errorf("EndDateTime() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEventWhen_IsAllDay(t *testing.T) {
	tests := []struct {
		name string
		when EventWhen
		want bool
	}{
		{
			name: "returns true for date object type",
			when: EventWhen{
				Object: "date",
				Date:   "2024-01-15",
			},
			want: true,
		},
		{
			name: "returns true for datespan object type",
			when: EventWhen{
				Object:    "datespan",
				StartDate: "2024-01-15",
				EndDate:   "2024-01-17",
			},
			want: true,
		},
		{
			name: "returns true when Date field is set",
			when: EventWhen{
				Date: "2024-01-15",
			},
			want: true,
		},
		{
			name: "returns true when StartDate field is set",
			when: EventWhen{
				StartDate: "2024-01-15",
			},
			want: true,
		},
		{
			name: "returns false for timespan object type",
			when: EventWhen{
				Object:    "timespan",
				StartTime: 1704067200,
				EndTime:   1704070800,
			},
			want: false,
		},
		{
			name: "returns false when only timestamps set",
			when: EventWhen{
				StartTime: 1704067200,
				EndTime:   1704070800,
			},
			want: false,
		},
		{
			name: "returns false for empty EventWhen",
			when: EventWhen{},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.when.IsAllDay()
			if got != tt.want {
				t.Errorf("IsAllDay() = %v, want %v", got, tt.want)
			}
		})
	}
}

// =============================================================================
// Event Tests
// =============================================================================

func TestEvent_IsTimezoneLocked(t *testing.T) {
	tests := []struct {
		name  string
		event Event
		want  bool
	}{
		{
			name:  "returns false when Metadata is nil",
			event: Event{},
			want:  false,
		},
		{
			name: "returns false when timezone_locked not set",
			event: Event{
				Metadata: map[string]string{
					"other_key": "value",
				},
			},
			want: false,
		},
		{
			name: "returns false when timezone_locked is false",
			event: Event{
				Metadata: map[string]string{
					"timezone_locked": "false",
				},
			},
			want: false,
		},
		{
			name: "returns true when timezone_locked is true",
			event: Event{
				Metadata: map[string]string{
					"timezone_locked": "true",
				},
			},
			want: true,
		},
		{
			name: "returns false for empty metadata value",
			event: Event{
				Metadata: map[string]string{
					"timezone_locked": "",
				},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.event.IsTimezoneLocked()
			if got != tt.want {
				t.Errorf("IsTimezoneLocked() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEvent_GetLockedTimezone(t *testing.T) {
	tests := []struct {
		name  string
		event Event
		want  string
	}{
		{
			name:  "returns empty when not locked",
			event: Event{},
			want:  "",
		},
		{
			name: "returns empty when locked but no timezone set",
			event: Event{
				Metadata: map[string]string{
					"timezone_locked": "true",
				},
				When: EventWhen{
					Object:    "timespan",
					StartTime: 1704067200,
				},
			},
			want: "",
		},
		{
			name: "returns empty for all-day events",
			event: Event{
				Metadata: map[string]string{
					"timezone_locked": "true",
				},
				When: EventWhen{
					Object: "date",
					Date:   "2024-01-15",
				},
			},
			want: "",
		},
		{
			name: "returns timezone when locked and timezone set",
			event: Event{
				Metadata: map[string]string{
					"timezone_locked": "true",
				},
				When: EventWhen{
					Object:        "timespan",
					StartTime:     1704067200,
					StartTimezone: "America/Los_Angeles",
				},
			},
			want: "America/Los_Angeles",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.event.GetLockedTimezone()
			if got != tt.want {
				t.Errorf("GetLockedTimezone() = %q, want %q", got, tt.want)
			}
		})
	}
}
