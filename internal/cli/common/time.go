package common

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// ErrScheduleInPast is returned by ParseHumanTime when the parsed time falls
// at or before "now" and the caller asked for future-only parsing.
var ErrScheduleInPast = errors.New("scheduled time is in the past")

// ParseHumanTimeOpts customizes ParseHumanTime.
type ParseHumanTimeOpts struct {
	// RejectPast causes ParseHumanTime to return ErrScheduleInPast when the
	// parsed result is at or before Now. Used by scheduling commands; left
	// false by availability/free-busy callers.
	RejectPast bool
	// RollPastBareTimeToTomorrow causes bare time-of-day inputs such as "3pm"
	// to use tomorrow when today's instance is not in the future. Explicit
	// inputs such as "today 3pm" and dated times are still checked normally.
	RollPastBareTimeToTomorrow bool
	// Now overrides time.Now(). Zero means "now".
	Now time.Time
	// Loc overrides the local time zone. Nil means time.Local.
	Loc *time.Location
}

// ParseHumanTime parses a human-friendly time string.
//
// Recognized inputs:
//   - "tomorrow"          → 9am next day
//   - "tomorrow 3pm"      → specific time next day
//   - "today 3pm"         → today at 3pm (rejected if past and RejectPast=true)
//   - "30m" / "2h" / "1d" → relative to Now
//   - 10+ digit Unix timestamp
//   - RFC3339 / ISO8601 / "2006-01-02 15:04" / "Jan 2 3:04pm" / similar
//   - bare time-of-day ("3pm", "15:00") — assumes today; rolls over to
//     tomorrow when the caller allows bare-time rollover.
func ParseHumanTime(input string, opts ParseHumanTimeOpts) (time.Time, error) {
	now := opts.Now
	if now.IsZero() {
		now = time.Now()
	}
	loc := opts.Loc
	if loc == nil {
		loc = now.Location()
	}

	input = strings.TrimSpace(input)
	lower := strings.ToLower(input)

	rejectIfPast := func(t time.Time) (time.Time, error) {
		if opts.RejectPast && !t.After(now) {
			return time.Time{}, ErrScheduleInPast
		}
		return t, nil
	}

	// Unix timestamp (10+ digits).
	if ts, err := strconv.ParseInt(input, 10, 64); err == nil && ts > 1_000_000_000 {
		return rejectIfPast(time.Unix(ts, 0))
	}

	// Duration short-form: "30m", "2h", "1d".
	if len(input) >= 2 {
		numStr := input[:len(input)-1]
		unit := input[len(input)-1:]
		if num, err := strconv.Atoi(numStr); err == nil {
			switch unit {
			case "m":
				return rejectIfPast(now.Add(time.Duration(num) * time.Minute))
			case "h":
				return rejectIfPast(now.Add(time.Duration(num) * time.Hour))
			case "d":
				return rejectIfPast(now.AddDate(0, 0, num))
			}
		}
	}

	// "tomorrow" keyword.
	if rest, ok := strings.CutPrefix(lower, "tomorrow"); ok {
		tomorrow := now.AddDate(0, 0, 1)
		rest = strings.TrimSpace(rest)
		if rest == "" {
			return time.Date(tomorrow.Year(), tomorrow.Month(), tomorrow.Day(), 9, 0, 0, 0, loc), nil
		}
		if t, err := ParseTimeOfDayInLocation(rest, loc); err == nil {
			return time.Date(tomorrow.Year(), tomorrow.Month(), tomorrow.Day(), t.Hour(), t.Minute(), 0, 0, loc), nil
		}
	}

	// "today" keyword.
	if rest, ok := strings.CutPrefix(lower, "today"); ok {
		rest = strings.TrimSpace(rest)
		if rest == "" {
			if opts.RejectPast {
				return time.Time{}, NewInputError("please specify a time, e.g., 'today 3pm'")
			}
			return now, nil
		}
		if t, err := ParseTimeOfDayInLocation(rest, loc); err == nil {
			result := time.Date(now.Year(), now.Month(), now.Day(), t.Hour(), t.Minute(), 0, 0, loc)
			return rejectIfPast(result)
		}
	}

	// Standard date/time formats — most specific first so RFC3339 wins over the
	// bare-time fallback below.
	formats := []string{
		time.RFC3339,           // "2006-01-02T15:04:05Z07:00"
		"2006-01-02T15:04:05Z", // ISO8601 UTC
		"2006-01-02T15:04:05",  // ISO8601 with seconds
		"2006-01-02T15:04",     // ISO8601 without seconds
		"2006-01-02 15:04:05",  // Space separator with seconds
		"2006-01-02 15:04",     // Space separator
		"2006-01-02 3:04pm",    // 12-hour
		"2006-01-02 3:04PM",
		"Jan 2 15:04",
		"Jan 2 3:04pm",
		"Jan 2, 2006 15:04",
		"Jan 2, 2006 3:04pm",
	}
	for _, format := range formats {
		if t, err := time.ParseInLocation(format, input, loc); err == nil {
			if t.Year() == 0 {
				t = time.Date(now.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), 0, 0, loc)
			}
			return rejectIfPast(t)
		}
	}

	// Bare time-of-day. Interpret as today; roll forward to tomorrow when the
	// caller allows the shortcut. Future-only callers otherwise receive
	// ErrScheduleInPast via rejectIfPast.
	if t, err := ParseTimeOfDayInLocation(lower, loc); err == nil {
		result := time.Date(now.Year(), now.Month(), now.Day(), t.Hour(), t.Minute(), 0, 0, loc)
		if opts.RollPastBareTimeToTomorrow && !result.After(now) {
			result = result.AddDate(0, 0, 1)
		} else if !opts.RejectPast && !result.After(now) {
			// Treat exact-now identically to past for the rollover branch
			// — matches the line above and avoids the edge case where
			// result == now sticks at "today" with RollPastBareTime off.
			result = result.AddDate(0, 0, 1)
		}
		return rejectIfPast(result)
	}

	return time.Time{}, NewInputError(fmt.Sprintf("could not parse time: %s", input))
}

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
