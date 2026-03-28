package common

import "github.com/fatih/color"

// StatusColor returns the appropriate color for a status string.
// This centralizes the status-to-color mapping used across CLI commands
// (scheduler bookings, webhook list, notetaker list, etc.).
//
// Common mappings:
//   - Green: active, confirmed, complete, attending
//   - Yellow: pending, scheduled, inactive
//   - Red: failed, failing, error
//   - Cyan: connecting, waiting, processing, waiting_for_entry, media_processing
//   - Dim: cancelled, deleted, archived
func StatusColor(status string) *color.Color {
	switch status {
	case "active", "confirmed", "complete", "attending":
		return Green
	case "pending", "scheduled", "inactive":
		return Yellow
	case "failed", "failing", "error":
		return Red
	case "connecting", "waiting", "processing", "rescheduled",
		"waiting_for_entry", "media_processing":
		return Cyan
	case "cancelled", "deleted", "archived":
		return Dim
	default:
		return Reset
	}
}

// StatusIcon returns a colored status indicator dot for a status string.
// Returns a colored "●" for known statuses, or "○" for unknown.
func StatusIcon(status string) string {
	c := StatusColor(status)
	if c == Reset {
		return "○"
	}
	return c.Sprint("●")
}

// ColorSprint applies the status color to the given status string.
// Convenience method combining StatusColor lookup with Sprint.
func ColorSprint(status string) string {
	return StatusColor(status).Sprint(status)
}
