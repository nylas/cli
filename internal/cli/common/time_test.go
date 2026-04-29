package common

import (
	"errors"
	"testing"
	"time"
)

func TestParseHumanTime(t *testing.T) {
	t.Parallel()
	loc := time.UTC
	now := time.Date(2026, 6, 15, 14, 0, 0, 0, loc) // Mon 2026-06-15 14:00 UTC

	cases := []struct {
		name       string
		input      string
		rejectPast bool
		rollBare   bool
		want       time.Time
		wantErr    bool
		isPastErr  bool
	}{
		{name: "tomorrow defaults to 9am", input: "tomorrow", want: time.Date(2026, 6, 16, 9, 0, 0, 0, loc)},
		{name: "tomorrow 3pm", input: "tomorrow 3pm", want: time.Date(2026, 6, 16, 15, 0, 0, 0, loc)},
		{name: "today 6pm", input: "today 6pm", want: time.Date(2026, 6, 15, 18, 0, 0, 0, loc)},
		{name: "today 1pm in past with RejectPast", input: "today 1pm", rejectPast: true, isPastErr: true, wantErr: true},
		{name: "today empty + RejectPast errors", input: "today", rejectPast: true, wantErr: true},
		{name: "30m relative", input: "30m", want: now.Add(30 * time.Minute)},
		{name: "2h relative", input: "2h", want: now.Add(2 * time.Hour)},
		{name: "1d relative", input: "1d", want: now.AddDate(0, 0, 1)},
		{name: "ISO date+time", input: "2026-12-25T10:00:00Z", want: time.Date(2026, 12, 25, 10, 0, 0, 0, time.UTC)},
		{name: "bare time-of-day rolls forward when past", input: "1pm", want: time.Date(2026, 6, 16, 13, 0, 0, 0, loc)},
		{name: "bare time-of-day past + RejectPast errors", input: "1pm", rejectPast: true, isPastErr: true, wantErr: true},
		{name: "bare time-of-day past + RejectPast + rollover", input: "1pm", rejectPast: true, rollBare: true, want: time.Date(2026, 6, 16, 13, 0, 0, 0, loc)},
		{name: "garbage errors", input: "not a time", wantErr: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := ParseHumanTime(tc.input, ParseHumanTimeOpts{
				Now:                        now,
				Loc:                        loc,
				RejectPast:                 tc.rejectPast,
				RollPastBareTimeToTomorrow: tc.rollBare,
			})
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				if tc.isPastErr && !errors.Is(err, ErrScheduleInPast) {
					t.Fatalf("expected ErrScheduleInPast, got %v", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !got.Equal(tc.want) {
				t.Fatalf("got %s, want %s", got, tc.want)
			}
		})
	}
}

func TestFormatTimeAgo(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name     string
		time     time.Time
		expected string
	}{
		{
			name:     "just now",
			time:     now.Add(-30 * time.Second),
			expected: "just now",
		},
		{
			name:     "1 minute ago",
			time:     now.Add(-1 * time.Minute),
			expected: "1 minute ago",
		},
		{
			name:     "5 minutes ago",
			time:     now.Add(-5 * time.Minute),
			expected: "5 minutes ago",
		},
		{
			name:     "1 hour ago",
			time:     now.Add(-1 * time.Hour),
			expected: "1 hour ago",
		},
		{
			name:     "3 hours ago",
			time:     now.Add(-3 * time.Hour),
			expected: "3 hours ago",
		},
		{
			name:     "1 day ago",
			time:     now.Add(-24 * time.Hour),
			expected: "1 day ago",
		},
		{
			name:     "5 days ago",
			time:     now.Add(-5 * 24 * time.Hour),
			expected: "5 days ago",
		},
		{
			name:     "1 week ago",
			time:     now.Add(-7 * 24 * time.Hour),
			expected: "1 week ago",
		},
		{
			name:     "2 weeks ago",
			time:     now.Add(-14 * 24 * time.Hour),
			expected: "2 weeks ago",
		},
		{
			name:     "1 month ago",
			time:     now.Add(-35 * 24 * time.Hour),
			expected: "1 month ago",
		},
		{
			name:     "6 months ago",
			time:     now.Add(-180 * 24 * time.Hour),
			expected: "6 months ago",
		},
		{
			name:     "1 year ago",
			time:     now.Add(-365 * 24 * time.Hour),
			expected: "1 year ago",
		},
		{
			name:     "2 years ago",
			time:     now.Add(-730 * 24 * time.Hour),
			expected: "2 years ago",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatTimeAgo(tt.time)
			if result != tt.expected {
				t.Errorf("FormatTimeAgo() = %q, want %q", result, tt.expected)
			}
		})
	}
}
