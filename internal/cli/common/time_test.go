package common

import (
	"testing"
	"time"
)

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
