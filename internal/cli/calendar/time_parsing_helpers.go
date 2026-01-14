package calendar

import (
	"fmt"
	"strings"
	"time"

	"github.com/nylas/cli/internal/cli/common"
)

// ============================================================================
// Natural Language Time Parsing
// ============================================================================

// ParsedTime represents a parsed natural language time expression.
type ParsedTime struct {
	Time     time.Time
	Timezone string
	Original string
}

// parseNaturalTime parses natural language time expressions.
// Supports formats like:
// - "tomorrow at 3pm"
// - "next Tuesday 2pm PST"
// - "Dec 25 10:00 AM"
// - "in 2 hours"
// - "2024-12-25 14:00"
func parseNaturalTime(input string, defaultTZ string) (*ParsedTime, error) {
	if input == "" {
		return nil, common.NewUserError(
			"time expression is empty",
			"Provide a time like 'tomorrow at 3pm' or 'Dec 25 10:00 AM'",
		)
	}

	// Default timezone if not specified
	if defaultTZ == "" {
		defaultTZ = "Local"
	}

	// Load the timezone
	loc, err := time.LoadLocation(defaultTZ)
	if err != nil {
		return nil, common.NewUserError(
			fmt.Sprintf("invalid timezone: %s", defaultTZ),
			"Use IANA timezone IDs like 'America/Los_Angeles'",
		)
	}

	now := time.Now().In(loc)
	normalizedInput := normalizeTimeString(input)

	// Try parsing in order of specificity
	// Note: Some parsers need normalized input, others need original
	parsers := []struct {
		fn            func(string, *time.Location, time.Time) (*ParsedTime, error)
		useNormalized bool
	}{
		{parseRelativeTime, true},
		{parseRelativeDayTime, true},
		{parseSpecificDayTime, true},
		{parseAbsoluteTime, false}, // Keep original for proper month name parsing
		{parseISOTime, false},      // Keep original for ISO formats
	}

	for _, parser := range parsers {
		inputToUse := input
		if parser.useNormalized {
			inputToUse = normalizedInput
		}
		result, err := parser.fn(inputToUse, loc, now)
		if err == nil && result != nil {
			result.Original = input
			return result, nil
		}
	}

	return nil, common.NewUserError(
		fmt.Sprintf("could not parse time: %s", input),
		"Try formats like:\n"+
			"  - tomorrow at 3pm\n"+
			"  - next Tuesday 2pm PST\n"+
			"  - Dec 25 10:00 AM\n"+
			"  - in 2 hours\n"+
			"  - 2024-12-25 14:00",
	)
}

// normalizeTimeString normalizes the input string for parsing.
func normalizeTimeString(s string) string {
	// Convert to lowercase for case-insensitive matching
	s = strings.ToLower(s)
	// Remove extra whitespace
	s = strings.TrimSpace(s)
	// Collapse multiple spaces into one
	s = strings.Join(strings.Fields(s), " ")
	return s
}

// parseRelativeTime parses relative time expressions like "in 2 hours", "in 30 minutes".
func parseRelativeTime(input string, loc *time.Location, now time.Time) (*ParsedTime, error) {
	// Pattern: "in X hours/minutes/days"
	patterns := []struct {
		pattern string
		unit    time.Duration
	}{
		{"in %d hour", time.Hour},
		{"in %d hours", time.Hour},
		{"in %d minute", time.Minute},
		{"in %d minutes", time.Minute},
		{"in %d day", 24 * time.Hour},
		{"in %d days", 24 * time.Hour},
	}

	for _, p := range patterns {
		var amount int
		_, err := fmt.Sscanf(input, p.pattern, &amount)
		if err == nil {
			result := now.Add(time.Duration(amount) * p.unit)
			return &ParsedTime{
				Time:     result,
				Timezone: loc.String(),
			}, nil
		}
	}

	return nil, fmt.Errorf("not a relative time")
}

// parseRelativeDayTime parses relative day + time like "tomorrow at 3pm", "today at 2:30pm".
func parseRelativeDayTime(input string, loc *time.Location, now time.Time) (*ParsedTime, error) {
	relativeDays := map[string]int{
		"today":    0,
		"tomorrow": 1,
	}

	for day, offset := range relativeDays {
		if len(input) > len(day) && input[:len(day)] == day {
			// Extract the time part
			timePart := input[len(day):]
			timePart = strings.TrimSpace(timePart)

			// Remove "at" if present
			if len(timePart) > 3 && timePart[:3] == "at " {
				timePart = timePart[3:]
			}

			// Parse the time
			parsedTime, err := parseTimeOfDay(timePart, loc)
			if err != nil {
				return nil, err
			}

			// Set to the target day
			targetDay := now.AddDate(0, 0, offset)
			result := time.Date(
				targetDay.Year(),
				targetDay.Month(),
				targetDay.Day(),
				parsedTime.Hour(),
				parsedTime.Minute(),
				0, 0, loc,
			)

			return &ParsedTime{
				Time:     result,
				Timezone: loc.String(),
			}, nil
		}
	}

	return nil, fmt.Errorf("not a relative day time")
}

// parseSpecificDayTime parses specific weekday + time like "next Tuesday 2pm", "Monday at 10am".
func parseSpecificDayTime(input string, loc *time.Location, now time.Time) (*ParsedTime, error) {
	weekdays := map[string]time.Weekday{
		"monday":    time.Monday,
		"tuesday":   time.Tuesday,
		"wednesday": time.Wednesday,
		"thursday":  time.Thursday,
		"friday":    time.Friday,
		"saturday":  time.Saturday,
		"sunday":    time.Sunday,
	}

	// Check for "next" prefix
	isNext := false
	checkInput := input
	if len(input) > 5 && input[:5] == "next " {
		isNext = true
		checkInput = input[5:]
	}

	for dayName, weekday := range weekdays {
		if len(checkInput) > len(dayName) && checkInput[:len(dayName)] == dayName {
			// Extract time part
			timePart := checkInput[len(dayName):]
			timePart = strings.TrimSpace(timePart)

			// Remove "at" if present
			if len(timePart) > 3 && timePart[:3] == "at " {
				timePart = timePart[3:]
			}

			// Parse the time
			parsedTime, err := parseTimeOfDay(timePart, loc)
			if err != nil {
				return nil, err
			}

			// Find next occurrence of this weekday
			daysUntil := int(weekday - now.Weekday())
			if daysUntil <= 0 || isNext {
				daysUntil += 7
			}

			targetDay := now.AddDate(0, 0, daysUntil)
			result := time.Date(
				targetDay.Year(),
				targetDay.Month(),
				targetDay.Day(),
				parsedTime.Hour(),
				parsedTime.Minute(),
				0, 0, loc,
			)

			return &ParsedTime{
				Time:     result,
				Timezone: loc.String(),
			}, nil
		}
	}

	return nil, fmt.Errorf("not a specific day time")
}

// parseAbsoluteTime parses absolute dates like "Dec 25 10:00 AM", "December 25, 2024 2pm".
func parseAbsoluteTime(input string, loc *time.Location, now time.Time) (*ParsedTime, error) {
	// Common date/time formats - try both lowercase and titlecase
	formats := []string{
		// Lowercase formats (after normalization) - with leading zero for hours
		"jan 2 03:04 pm",
		"jan 2 03:04pm",
		"jan 2 3:04 pm",
		"jan 2 3:04pm",
		"jan 2, 2006 03:04 pm",
		"jan 2, 2006 3:04 pm",
		"january 2 03:04 pm",
		"january 2 3:04 pm",
		"january 2, 2006 03:04 pm",
		"january 2, 2006 3:04 pm",
		// Titlecase formats (original input)
		"Jan 2 03:04 PM",
		"Jan 2 3:04 PM",
		"Jan 2 03:04PM",
		"Jan 2 3:04PM",
		"Jan 2, 2006 03:04 PM",
		"Jan 2, 2006 3:04 PM",
		"January 2 03:04 PM",
		"January 2 3:04 PM",
		"January 2, 2006 03:04 PM",
		"January 2, 2006 3:04 PM",
		// Numeric formats
		"2006-01-02 15:04",
		"01/02/2006 03:04 PM",
		"01/02/2006 3:04 PM",
		"01/02/2006 15:04",
	}

	for _, format := range formats {
		t, err := time.ParseInLocation(format, input, loc)
		if err == nil {
			// If year is not in input, use current year
			if t.Year() == 0 {
				t = time.Date(now.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), 0, 0, loc)
			}
			return &ParsedTime{
				Time:     t,
				Timezone: loc.String(),
			}, nil
		}
	}

	return nil, fmt.Errorf("not an absolute time")
}

// parseISOTime parses ISO format times like "2024-12-25T14:00:00".
func parseISOTime(input string, loc *time.Location, now time.Time) (*ParsedTime, error) {
	formats := []string{
		time.RFC3339,
		"2006-01-02T15:04:05",
		"2006-01-02 15:04:05",
		"2006-01-02T15:04",
		"2006-01-02 15:04",
	}

	for _, format := range formats {
		t, err := time.ParseInLocation(format, input, loc)
		if err == nil {
			return &ParsedTime{
				Time:     t,
				Timezone: loc.String(),
			}, nil
		}
	}

	return nil, fmt.Errorf("not an ISO time")
}

// timezoneAbbreviations maps common timezone abbreviations to IANA names.
var timezoneAbbreviations = map[string]string{
	"PST":  "America/Los_Angeles",
	"PDT":  "America/Los_Angeles",
	"EST":  "America/New_York",
	"EDT":  "America/New_York",
	"CST":  "America/Chicago",
	"CDT":  "America/Chicago",
	"MST":  "America/Denver",
	"MDT":  "America/Denver",
	"GMT":  "Europe/London",
	"BST":  "Europe/London",
	"IST":  "Asia/Kolkata",
	"JST":  "Asia/Tokyo",
	"AEST": "Australia/Sydney",
	"AEDT": "Australia/Sydney",
	"UTC":  "UTC",
}

// extractTimezoneFromInput extracts a timezone abbreviation from input and returns
// the location and cleaned input string. If no timezone found, returns nil location.
func extractTimezoneFromInput(input string) (*time.Location, string) {
	upperInput := strings.ToUpper(input)

	// Check for timezone abbreviations at the end of input
	for abbrev, iana := range timezoneAbbreviations {
		// Check if input ends with the abbreviation (with space before)
		suffix := " " + abbrev
		if strings.HasSuffix(upperInput, suffix) {
			cleanInput := strings.TrimSuffix(input, input[len(input)-len(suffix):])
			cleanInput = strings.TrimSpace(cleanInput)
			if loc, err := time.LoadLocation(iana); err == nil {
				return loc, cleanInput
			}
		}
	}

	return nil, input
}

// parseTimeOfDay parses time of day like "3pm", "2:30pm", "14:00", "3pm PST".
func parseTimeOfDay(input string, loc *time.Location) (time.Time, error) {
	// Extract timezone from input if present (e.g., "3pm PST")
	extractedLoc, cleanInput := extractTimezoneFromInput(input)
	if extractedLoc != nil {
		loc = extractedLoc
	}

	// Normalize to lowercase, then try both lowercase and uppercase formats
	originalInput := cleanInput
	lowerInput := strings.ToLower(cleanInput)

	formats := []string{
		"3pm",
		"3:04pm",
		"3 pm",
		"3:04 pm",
		"15:04",
	}

	// Try lowercase formats
	for _, format := range formats {
		t, err := time.ParseInLocation(format, lowerInput, loc)
		if err == nil {
			return t, nil
		}
	}

	// Try original input with uppercase formats (for backward compatibility)
	upperFormats := []string{
		"3PM",
		"3:04PM",
		"3 PM",
		"3:04 PM",
	}

	for _, format := range upperFormats {
		t, err := time.ParseInLocation(format, originalInput, loc)
		if err == nil {
			return t, nil
		}
	}

	return time.Time{}, common.NewInputError(fmt.Sprintf("invalid time format: %s", input))
}
