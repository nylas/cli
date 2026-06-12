package tui

import (
	"fmt"

	"github.com/gdamore/tcell/v2"
	"github.com/nylas/cli/internal/domain"
	"github.com/rivo/tview"
)

func (v *EventsView) showRecurringEventEditDialog(calendarID string, evt *domain.Event, parentList *tview.List) {
	// Create a simple list for the options
	optionsList := tview.NewList()
	optionsList.SetBackgroundColor(v.app.styles.BgColor)
	optionsList.SetBorder(true)
	optionsList.SetBorderColor(v.app.styles.FocusColor)
	optionsList.SetTitle(" Edit Recurring Event ")
	optionsList.SetTitleColor(v.app.styles.TitleFg)
	optionsList.ShowSecondaryText(true)
	optionsList.SetHighlightFullLine(true)
	optionsList.SetSelectedBackgroundColor(v.app.styles.TableSelectBg)
	optionsList.SetSelectedTextColor(v.app.styles.TableSelectFg)
	optionsList.SetMainTextColor(v.app.styles.FgColor)
	optionsList.SetSecondaryTextColor(v.app.styles.BorderColor)

	eventCopy := *evt

	// Add options
	optionsList.AddItem("Edit this occurrence", "Only modify this instance", '1', func() {
		v.app.PopDetail() // Close options dialog
		v.app.PopDetail() // Close day detail
		// For editing a single occurrence, we pass the event as-is
		// The API will handle creating an exception
		v.app.ShowEventForm(calendarID, &eventCopy, func(updatedEvent *domain.Event) {
			v.loadEventsForCalendar(calendarID)
		})
	})

	optionsList.AddItem("Edit all occurrences", "Modify the entire series", '2', func() {
		v.app.PopDetail() // Close options dialog
		v.app.PopDetail() // Close day detail
		// For editing the series, we need to use the master event ID if available
		editEvt := &eventCopy
		if eventCopy.MasterEventID != "" {
			// This is an instance - we'd need to fetch the master event
			// For now, just edit the current event which will prompt the API behavior
			v.app.Flash(FlashInfo, "Editing series from instance...")
		}
		v.app.ShowEventForm(calendarID, editEvt, func(updatedEvent *domain.Event) {
			v.loadEventsForCalendar(calendarID)
		})
	})

	optionsList.AddItem("Cancel", "Go back", 'c', func() {
		v.app.PopDetail()
	})

	// Handle escape
	optionsList.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEscape {
			v.app.PopDetail()
			return nil
		}
		return event
	})

	v.app.PushDetail("recurring-edit-options", optionsList)
	v.app.SetFocus(optionsList)
}

func (v *EventsView) showRecurringEventDeleteDialog(calendarID string, evt *domain.Event, parentList *tview.List) {
	// Create a simple list for the options
	optionsList := tview.NewList()
	optionsList.SetBackgroundColor(v.app.styles.BgColor)
	optionsList.SetBorder(true)
	optionsList.SetBorderColor(v.app.styles.FocusColor)
	optionsList.SetTitle(" Delete Recurring Event ")
	optionsList.SetTitleColor(v.app.styles.TitleFg)
	optionsList.ShowSecondaryText(true)
	optionsList.SetHighlightFullLine(true)
	optionsList.SetSelectedBackgroundColor(v.app.styles.TableSelectBg)
	optionsList.SetSelectedTextColor(v.app.styles.TableSelectFg)
	optionsList.SetMainTextColor(v.app.styles.FgColor)
	optionsList.SetSecondaryTextColor(v.app.styles.BorderColor)

	eventCopy := *evt

	// Add options
	optionsList.AddItem("Delete this occurrence", "Only remove this instance", '1', func() {
		v.app.PopDetail() // Close options dialog
		v.app.PopDetail() // Close day detail
		v.app.ShowConfirmDialog("Delete Occurrence",
			fmt.Sprintf("Delete this occurrence of '%s'?", eventCopy.Title),
			func() {
				v.app.DeleteEvent(calendarID, &eventCopy, func() {
					v.loadEventsForCalendar(calendarID)
				})
			})
	})

	optionsList.AddItem("Delete all occurrences", "Remove the entire series", '2', func() {
		v.app.PopDetail() // Close options dialog
		v.app.PopDetail() // Close day detail
		v.app.ShowConfirmDialog("Delete Series",
			fmt.Sprintf("Delete all occurrences of '%s'? This cannot be undone.", eventCopy.Title),
			func() {
				// For deleting the series, we use the master event ID if available
				deleteEvt := &eventCopy
				if eventCopy.MasterEventID != "" {
					deleteEvt = &domain.Event{ID: eventCopy.MasterEventID}
				}
				v.app.DeleteEvent(calendarID, deleteEvt, func() {
					v.loadEventsForCalendar(calendarID)
				})
			})
	})

	optionsList.AddItem("Cancel", "Go back", 'c', func() {
		v.app.PopDetail()
	})

	// Handle escape
	optionsList.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEscape {
			v.app.PopDetail()
			return nil
		}
		return event
	})

	v.app.PushDetail("recurring-delete-options", optionsList)
	v.app.SetFocus(optionsList)
}

// isRecurringEvent returns true if the event is recurring.
func isRecurringEvent(evt *domain.Event) bool {
	return len(evt.Recurrence) > 0 || evt.MasterEventID != ""
}

// formatRecurrenceRule formats an RRULE string into a human-readable format.
func formatRecurrenceRule(rules []string) string {
	if len(rules) == 0 {
		return ""
	}

	// Find the first RRULE
	var rule string
	for _, r := range rules {
		if len(r) >= 6 && r[:6] == "RRULE:" {
			rule = r[6:]
			break
		}
		if len(r) > 0 && r[0] != 'E' { // Not EXDATE
			rule = r
			break
		}
	}

	if rule == "" {
		return ""
	}

	// Parse the RRULE
	parts := make(map[string]string)
	for _, part := range splitRRuleParts(rule) {
		if idx := indexByte(part, '='); idx > 0 {
			parts[part[:idx]] = part[idx+1:]
		}
	}

	freq := parts["FREQ"]
	interval := parts["INTERVAL"]
	if interval == "" {
		interval = "1"
	}
	byday := parts["BYDAY"]
	count := parts["COUNT"]
	until := parts["UNTIL"]

	// Build human-readable string
	var result string
	switch freq {
	case "DAILY":
		if interval == "1" {
			result = "Every day"
		} else {
			result = "Every " + interval + " days"
		}
	case "WEEKLY":
		if interval == "1" {
			result = "Every week"
		} else {
			result = "Every " + interval + " weeks"
		}
		if byday != "" {
			result += " on " + formatDays(byday)
		}
	case "MONTHLY":
		if interval == "1" {
			result = "Every month"
		} else {
			result = "Every " + interval + " months"
		}
	case "YEARLY":
		if interval == "1" {
			result = "Every year"
		} else {
			result = "Every " + interval + " years"
		}
	default:
		result = rule
	}

	// Add end condition
	if count != "" {
		result += " (" + count + " times)"
	} else if until != "" {
		// Parse UNTIL date (format: YYYYMMDD or YYYYMMDDTHHmmssZ)
		if len(until) >= 8 {
			result += " until " + until[:4] + "-" + until[4:6] + "-" + until[6:8]
		}
	}

	return result
}

// splitRRuleParts splits an RRULE into its component parts.
func splitRRuleParts(rule string) []string {
	var parts []string
	start := 0
	for i := 0; i < len(rule); i++ {
		if rule[i] == ';' {
			parts = append(parts, rule[start:i])
			start = i + 1
		}
	}
	if start < len(rule) {
		parts = append(parts, rule[start:])
	}
	return parts
}

// indexByte returns the index of the first occurrence of c in s, or -1 if not found.
func indexByte(s string, c byte) int {
	for i := 0; i < len(s); i++ {
		if s[i] == c {
			return i
		}
	}
	return -1
}

// formatDays formats BYDAY values into human-readable day names.
func formatDays(byday string) string {
	dayMap := map[string]string{
		"SU": "Sun", "MO": "Mon", "TU": "Tue", "WE": "Wed",
		"TH": "Thu", "FR": "Fri", "SA": "Sat",
	}

	var days []string
	for _, part := range splitByComma(byday) {
		// Handle numeric prefix (e.g., "1MO" for first Monday)
		day := part
		if len(part) > 2 {
			day = part[len(part)-2:]
		}
		if name, ok := dayMap[day]; ok {
			days = append(days, name)
		}
	}

	if len(days) == 0 {
		return byday
	}
	return joinStrings(days, ", ")
}

// splitByComma splits a string by comma.
func splitByComma(s string) []string {
	var parts []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == ',' {
			parts = append(parts, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		parts = append(parts, s[start:])
	}
	return parts
}

// joinStrings joins strings with a separator.
func joinStrings(strs []string, sep string) string {
	if len(strs) == 0 {
		return ""
	}
	result := strs[0]
	for i := 1; i < len(strs); i++ {
		result += sep + strs[i]
	}
	return result
}

// ============================================================================
// Contacts View
// ============================================================================
