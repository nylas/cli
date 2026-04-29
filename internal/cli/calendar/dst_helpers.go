package calendar

import (
	"fmt"
	"strings"
	"time"

	"github.com/nylas/cli/internal/adapters/utilities/timezone"
	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/domain"
)

// ============================================================================
// DST Warning Helpers
// ============================================================================

// checkDSTWarning checks if an event time has DST warnings and returns formatted message.
// Returns empty string if no warning.
func checkDSTWarning(eventTime time.Time, tz string) string {
	if tz == "" {
		return ""
	}

	// Use timezone service to check for DST warnings
	tzService := timezone.NewService()
	ctx, cancel := common.CreateContext()
	defer cancel()

	// Check for warnings within 7 days
	warning, err := tzService.CheckDSTWarning(ctx, eventTime, tz, 7)
	if err != nil || warning == nil {
		return ""
	}

	if !warning.IsNearTransition {
		return ""
	}

	// Format warning message with appropriate icon
	return formatDSTWarning(warning)
}

// formatDSTWarning formats a DST warning for display in the terminal.
func formatDSTWarning(warning *domain.DSTWarning) string {
	if warning == nil {
		return ""
	}

	var icon string
	switch warning.Severity {
	case "error":
		icon = "⛔"
	case "warning":
		icon = "⚠️"
	case "info":
		icon = "ℹ️"
	default:
		icon = "⚠️"
	}

	return fmt.Sprintf("%s %s", icon, warning.Warning)
}

// checkDSTConflict checks if an event time falls in a DST conflict.
// Returns the warning if there's a conflict, nil otherwise.
func checkDSTConflict(eventTime time.Time, tz string, _ time.Duration) (*domain.DSTWarning, error) {
	if tz == "" {
		return nil, nil
	}

	tzService := timezone.NewService()
	ctx, cancel := common.CreateContext()
	defer cancel()

	// Check for DST warning at event time (no warning window, only exact conflicts)
	warning, err := tzService.CheckDSTWarning(ctx, eventTime, tz, 0)
	if err != nil {
		return nil, err
	}

	// Only return warning if it's an actual conflict (gap or duplicate)
	if warning != nil && (warning.InTransitionGap || warning.InDuplicateHour) {
		return warning, nil
	}

	return nil, nil
}

// confirmDSTConflict displays a DST conflict warning and asks for user confirmation.
// Returns true if user wants to proceed, false if cancelled.
func confirmDSTConflict(warning *domain.DSTWarning) bool {
	if warning == nil {
		return true
	}

	yellow := common.Yellow
	red := common.Red

	fmt.Println()
	if warning.InTransitionGap {
		_, _ = red.Println("⚠️  DST Conflict Detected!")
	} else {
		_, _ = yellow.Println("⚠️  DST Conflict Detected!")
	}
	fmt.Println()

	fmt.Println(warning.Warning)
	fmt.Println()

	// Show suggested alternatives if available
	if warning.InTransitionGap {
		fmt.Println("Suggested alternatives:")
		fmt.Println("  1. Schedule 1 hour earlier (before DST)")
		fmt.Println("  2. Schedule at the requested time after DST")
		fmt.Println("  3. Use a different date")
		fmt.Println()
	} else if warning.InDuplicateHour {
		fmt.Println("Note: This time occurs twice due to falling back.")
		fmt.Println("The event will be created at the first occurrence.")
		fmt.Println()
	}

	// Ask for confirmation
	fmt.Print("Create anyway? [y/N]: ")
	var confirm string
	_, _ = fmt.Scanln(&confirm)

	return strings.ToLower(confirm) == "y" || strings.ToLower(confirm) == "yes"
}
