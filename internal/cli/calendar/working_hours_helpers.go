package calendar

import (
	"fmt"
	"strings"
	"time"

	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/domain"
)

// ============================================================================
// Working Hours Validation
// ============================================================================

// checkWorkingHoursViolation checks if an event time falls outside working hours.
// Returns warning message if outside working hours, empty string otherwise.
func checkWorkingHoursViolation(eventTime time.Time, config *domain.Config) string {
	if config == nil || config.WorkingHours == nil {
		// No working hours configured, skip validation
		return ""
	}

	// Get schedule for the event's day of week
	weekday := strings.ToLower(eventTime.Weekday().String())
	schedule := config.WorkingHours.GetScheduleForDay(weekday)

	// If working hours not enabled for this day, no violation
	if schedule == nil || !schedule.Enabled {
		return ""
	}

	// Parse working hours
	startHour, startMin, err := parseTimeString(schedule.Start)
	if err != nil {
		return "" // Invalid config, skip validation
	}

	endHour, endMin, err := parseTimeString(schedule.End)
	if err != nil {
		return "" // Invalid config, skip validation
	}

	// Check if event time is outside working hours
	eventHour := eventTime.Hour()
	eventMin := eventTime.Minute()

	// Convert to minutes for easier comparison
	eventMinutes := eventHour*60 + eventMin
	startMinutes := startHour*60 + startMin
	endMinutes := endHour*60 + endMin

	if eventMinutes < startMinutes || eventMinutes >= endMinutes {
		// Outside working hours
		var offset string
		if eventMinutes < startMinutes {
			hoursBefore := (startMinutes - eventMinutes) / 60
			minsBefore := (startMinutes - eventMinutes) % 60
			if minsBefore > 0 {
				offset = fmt.Sprintf("%dh %dm before start", hoursBefore, minsBefore)
			} else {
				offset = fmt.Sprintf("%d hour(s) before start", hoursBefore)
			}
		} else {
			hoursAfter := (eventMinutes - endMinutes) / 60
			minsAfter := (eventMinutes - endMinutes) % 60
			if minsAfter > 0 {
				offset = fmt.Sprintf("%dh %dm after end", hoursAfter, minsAfter)
			} else {
				offset = fmt.Sprintf("%d hour(s) after end", hoursAfter)
			}
		}

		return fmt.Sprintf("Event scheduled outside working hours (%s - %s) - %s",
			schedule.Start, schedule.End, offset)
	}

	return ""
}

// confirmWorkingHoursViolation displays a working hours warning and asks for confirmation.
// Returns true if user wants to proceed, false if cancelled.
func confirmWorkingHoursViolation(violation string, eventTime time.Time, schedule *domain.DaySchedule) bool {
	if violation == "" {
		return true
	}

	fmt.Println()
	_, _ = common.BoldYellow.Println("⚠️  Working Hours Warning")
	fmt.Println()

	fmt.Printf("This event is scheduled outside your working hours:\n")
	fmt.Printf("  • Your hours: %s - %s\n", schedule.Start, schedule.End)
	fmt.Printf("  • Event time: %s\n", eventTime.Format("3:04 PM MST"))
	fmt.Printf("  • %s\n", violation)
	fmt.Println()

	// Ask for confirmation
	fmt.Print("Create anyway? [y/N]: ")
	var confirm string
	_, _ = fmt.Scanln(&confirm)

	return strings.ToLower(confirm) == "y" || strings.ToLower(confirm) == "yes"
}

// parseTimeString parses a time string in "HH:MM" format.
func parseTimeString(s string) (hour, min int, err error) {
	_, err = fmt.Sscanf(s, "%d:%d", &hour, &min)
	if err != nil {
		return 0, 0, err
	}
	if hour < 0 || hour > 23 || min < 0 || min > 59 {
		return 0, 0, fmt.Errorf("invalid time")
	}
	return hour, min, nil
}

// checkBreakViolation checks if an event time falls during a break period.
// Returns error message if during break (hard block), empty string otherwise.
func checkBreakViolation(eventTime time.Time, config *domain.Config) string {
	if config == nil || config.WorkingHours == nil {
		return "" // No working hours or breaks configured
	}

	// Get schedule for the event's day of week
	weekday := strings.ToLower(eventTime.Weekday().String())
	schedule := config.WorkingHours.GetScheduleForDay(weekday)

	// If no schedule or breaks, no violation
	if schedule == nil || len(schedule.Breaks) == 0 {
		return ""
	}

	// Check each break period
	eventHour := eventTime.Hour()
	eventMin := eventTime.Minute()
	eventMinutes := eventHour*60 + eventMin

	for _, breakBlock := range schedule.Breaks {
		// Parse break start/end times
		startHour, startMin, err := parseTimeString(breakBlock.Start)
		if err != nil {
			continue // Skip invalid break config
		}

		endHour, endMin, err := parseTimeString(breakBlock.End)
		if err != nil {
			continue // Skip invalid break config
		}

		// Convert to minutes for comparison
		breakStart := startHour*60 + startMin
		breakEnd := endHour*60 + endMin

		// Check if event falls within this break period
		if eventMinutes >= breakStart && eventMinutes < breakEnd {
			return fmt.Sprintf("Event cannot be scheduled during %s (%s - %s)",
				breakBlock.Name, breakBlock.Start, breakBlock.End)
		}
	}

	return ""
}
