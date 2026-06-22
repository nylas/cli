package tui

import (
	"time"

	"github.com/nylas/cli/internal/cli/common"
)

// formatDate formats a time for display in the UI.
func formatDate(t time.Time) string {
	now := time.Now()
	if t.Year() == now.Year() && t.YearDay() == now.YearDay() {
		return t.Format("3:04 PM")
	}
	if t.Year() == now.Year() {
		return t.Format("Jan 2")
	}
	return t.Format("Jan 2, 06")
}

// formatFileSize formats a file size in bytes to a human-readable string.
func formatFileSize(size int64) string {
	return common.FormatSize(size)
}

// stripHTMLForTUI removes HTML tags from a string for terminal display.
func stripHTMLForTUI(s string) string {
	return common.StripHTML(s)
}
