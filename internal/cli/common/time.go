package common

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// FormatTimeAgo formats a time as a relative string (e.g., "2 hours ago").
func FormatTimeAgo(t time.Time) string {
	now := time.Now()
	diff := now.Sub(t)

	if diff < time.Minute {
		return "just now"
	} else if diff < time.Hour {
		mins := int(diff.Minutes())
		if mins == 1 {
			return "1 minute ago"
		}
		return fmt.Sprintf("%d minutes ago", mins)
	} else if diff < 24*time.Hour {
		hours := int(diff.Hours())
		if hours == 1 {
			return "1 hour ago"
		}
		return fmt.Sprintf("%d hours ago", hours)
	} else if diff < 7*24*time.Hour {
		days := int(diff.Hours() / 24)
		if days == 1 {
			return "1 day ago"
		}
		return fmt.Sprintf("%d days ago", days)
	} else if diff < 30*24*time.Hour {
		weeks := int(diff.Hours() / (24 * 7))
		if weeks == 1 {
			return "1 week ago"
		}
		return fmt.Sprintf("%d weeks ago", weeks)
	} else if diff < 365*24*time.Hour {
		months := int(diff.Hours() / (24 * 30))
		if months == 1 {
			return "1 month ago"
		}
		return fmt.Sprintf("%d months ago", months)
	}

	years := int(diff.Hours() / (24 * 365))
	if years == 1 {
		return "1 year ago"
	}
	return fmt.Sprintf("%d years ago", years)
}

// ParseTimeOfDay parses time strings like "9am", "14:30", "2:30pm".
// Uses UTC for the date component; only hour/minute are meaningful.
func ParseTimeOfDay(s string) (time.Time, error) {
	return ParseTimeOfDayInLocation(s, time.UTC)
}

// ParseTimeOfDayInLocation parses time strings with a specific location.
// Supports formats: "15:04" (24h), "3:04pm", "3:04 pm", "3pm", "3 pm".
func ParseTimeOfDayInLocation(s string, loc *time.Location) (time.Time, error) {
	if loc == nil {
		loc = time.UTC
	}

	s = strings.ToLower(strings.TrimSpace(s))

	// Try 24-hour format first
	if t, err := time.ParseInLocation("15:04", s, loc); err == nil {
		return t, nil
	}

	// Try 12-hour formats
	formats := []string{
		"3:04pm",
		"3:04 pm",
		"3pm",
		"3 pm",
	}

	for _, format := range formats {
		if t, err := time.ParseInLocation(format, s, loc); err == nil {
			return t, nil
		}
	}

	return time.Time{}, NewInputError(fmt.Sprintf("invalid time format: %s", s))
}

// ParseDuration parses duration strings with extended support for days and weeks.
// Supports: standard Go durations (1h30m, 30s) plus "d" (days) and "w" (weeks).
// Examples: "30m", "2h", "24h", "7d", "2w".
func ParseDuration(s string) (time.Duration, error) {
	s = strings.ToLower(strings.TrimSpace(s))
	if s == "" {
		return 0, fmt.Errorf("empty duration string")
	}

	// Check for day suffix
	if strings.HasSuffix(s, "d") {
		numStr := s[:len(s)-1]
		num, err := strconv.Atoi(numStr)
		if err != nil {
			return 0, fmt.Errorf("invalid duration number: %s", numStr)
		}
		return time.Duration(num) * 24 * time.Hour, nil
	}

	// Check for week suffix
	if strings.HasSuffix(s, "w") {
		numStr := s[:len(s)-1]
		num, err := strconv.Atoi(numStr)
		if err != nil {
			return 0, fmt.Errorf("invalid duration number: %s", numStr)
		}
		return time.Duration(num) * 7 * 24 * time.Hour, nil
	}

	// Try standard Go duration parsing (handles h, m, s, etc.)
	duration, err := time.ParseDuration(s)
	if err != nil {
		return 0, NewInputError(fmt.Sprintf("invalid duration format: %s (use 1h, 30m, 7d, 2w, etc.)", s))
	}
	return duration, nil
}
