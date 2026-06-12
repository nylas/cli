package tui

import (
	"fmt"

	"github.com/gdamore/tcell/v2"
	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/domain"
	"github.com/rivo/tview"
)

func (v *EventsView) showDayDetail() {
	date := v.calendar.GetSelectedDate()
	events := v.calendar.GetEventsForDate(date)

	if len(events) == 0 {
		v.app.Flash(FlashInfo, "No events on %s - press 'n' to create one", date.Format("Jan 2"))
		return
	}

	// Create a list for event selection (supports edit/delete)
	list := tview.NewList()
	list.SetBackgroundColor(v.app.styles.BgColor)
	list.SetBorder(true)
	list.SetBorderColor(v.app.styles.FocusColor)
	list.SetTitle(fmt.Sprintf(" %s (%d events) ", date.Format(common.DisplayDateFormat), len(events)))
	list.SetTitleColor(v.app.styles.TitleFg)
	list.ShowSecondaryText(true)
	list.SetHighlightFullLine(true)
	list.SetSelectedBackgroundColor(v.app.styles.TableSelectBg)
	list.SetSelectedTextColor(v.app.styles.TableSelectFg)
	list.SetMainTextColor(v.app.styles.FgColor)
	list.SetSecondaryTextColor(v.app.styles.BorderColor)

	calendarID := v.calendar.GetCurrentCalendarID()

	for i, evt := range events {
		// Build main text
		title := evt.Title
		if evt.When.IsAllDay() {
			title = "📅 " + title
		}
		// Add recurring indicator
		if isRecurringEvent(&evt) {
			title = "🔁 " + title
		}

		// Build secondary text with time
		timeStr := "All day"
		if !evt.When.IsAllDay() {
			start := evt.When.StartDateTime()
			end := evt.When.EndDateTime()
			timeStr = fmt.Sprintf("%s - %s", start.Format("3:04 PM"), end.Format("3:04 PM"))
		}
		secondary := timeStr
		if evt.Location != "" {
			secondary += " | 📍 " + evt.Location
		}

		// Capture event for closure
		eventCopy := events[i]

		list.AddItem(title, secondary, 0, func() {
			// Show event detail on Enter
			v.showEventDetail(&eventCopy)
		})
	}

	// Handle keyboard events
	list.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEscape:
			v.app.PopDetail()
			v.app.SetFocus(v.calendar)
			return nil
		case tcell.KeyRune:
			switch event.Rune() {
			case 'e': // Edit selected event
				idx := list.GetCurrentItem()
				if idx >= 0 && idx < len(events) {
					evt := events[idx]
					if isRecurringEvent(&evt) {
						// Show dialog for recurring event
						v.showRecurringEventEditDialog(calendarID, &evt, list)
					} else {
						v.app.PopDetail()
						v.app.ShowEventForm(calendarID, &evt, func(updatedEvent *domain.Event) {
							v.loadEventsForCalendar(calendarID)
						})
					}
				}
				return nil
			case 'd': // Delete selected event
				idx := list.GetCurrentItem()
				if idx >= 0 && idx < len(events) {
					evt := events[idx]
					if isRecurringEvent(&evt) {
						// Show dialog for recurring event
						v.showRecurringEventDeleteDialog(calendarID, &evt, list)
					} else {
						v.app.PopDetail()
						v.app.DeleteEvent(calendarID, &evt, func() {
							v.loadEventsForCalendar(calendarID)
						})
					}
				}
				return nil
			case 'n': // New event
				v.app.PopDetail()
				v.createNewEvent()
				return nil
			}
		}
		return event
	})

	// Push list onto page stack
	v.app.PushDetail("day-detail", list)
	v.app.SetFocus(list)
}

func (v *EventsView) showEventDetail(evt *domain.Event) {
	// Create detailed view of a single event
	detail := tview.NewTextView()
	detail.SetDynamicColors(true)
	detail.SetBackgroundColor(v.app.styles.BgColor)
	detail.SetBorder(true)
	detail.SetBorderColor(v.app.styles.FocusColor)
	detail.SetTitle(fmt.Sprintf(" %s ", evt.Title))
	detail.SetTitleColor(v.app.styles.TitleFg)
	detail.SetBorderPadding(1, 1, 2, 2)
	detail.SetScrollable(true)

	// Use cached Hex() method
	s := v.app.styles
	info := s.Hex(s.InfoColor)
	key := s.Hex(s.FgColor)
	value := s.Hex(s.InfoSectionFg)
	muted := s.Hex(s.BorderColor)

	// Time
	var timeStr string
	if !evt.When.IsAllDay() {
		start := evt.When.StartDateTime()
		end := evt.When.EndDateTime()
		dateStr := start.Format(common.DisplayDateLong)
		timeStr = fmt.Sprintf("%s\n%s - %s", dateStr, start.Format(common.DisplayTimeFormat), end.Format(common.DisplayTimeFormat))
	} else {
		timeStr = evt.When.StartDateTime().Format(common.DisplayDateLong) + " (All day)"
	}
	_, _ = fmt.Fprintf(detail, "[%s::b]When[-::-]\n[%s]%s[-]\n\n", info, value, timeStr)

	// Location
	if evt.Location != "" {
		_, _ = fmt.Fprintf(detail, "[%s::b]Location[-::-]\n[%s]%s[-]\n\n", info, value, evt.Location)
	}

	// Description
	if evt.Description != "" {
		_, _ = fmt.Fprintf(detail, "[%s::b]Description[-::-]\n[%s]%s[-]\n\n", info, value, evt.Description)
	}

	// Participants
	if len(evt.Participants) > 0 {
		_, _ = fmt.Fprintf(detail, "[%s::b]Participants[-::-]\n", info)
		for _, p := range evt.Participants {
			name := p.Name
			if name == "" {
				name = p.Email
			}
			status := p.Status
			if status == "" {
				status = "pending"
			}
			statusIcon := "⏳"
			switch status {
			case "yes":
				statusIcon = "✓"
			case "no":
				statusIcon = "✗"
			case "maybe":
				statusIcon = "?"
			}
			_, _ = fmt.Fprintf(detail, "[%s]  %s %s[-]\n", value, statusIcon, name)
		}
		_, _ = fmt.Fprintln(detail)
	}

	// Conferencing
	if evt.Conferencing != nil && evt.Conferencing.Details != nil && evt.Conferencing.Details.URL != "" {
		_, _ = fmt.Fprintf(detail, "[%s::b]Meeting Link[-::-]\n[%s]%s[-]\n\n", info, value, evt.Conferencing.Details.URL)
	}

	// Recurrence
	if isRecurringEvent(evt) {
		_, _ = fmt.Fprintf(detail, "[%s::b]Recurrence[-::-]\n", info)
		if len(evt.Recurrence) > 0 {
			recurrenceStr := formatRecurrenceRule(evt.Recurrence)
			if recurrenceStr != "" {
				_, _ = fmt.Fprintf(detail, "[%s]🔁 %s[-]\n\n", value, recurrenceStr)
			} else {
				_, _ = fmt.Fprintf(detail, "[%s]🔁 Recurring event[-]\n\n", value)
			}
		} else if evt.MasterEventID != "" {
			_, _ = fmt.Fprintf(detail, "[%s]🔁 Instance of recurring event[-]\n\n", value)
		}
	}

	// Status
	_, _ = fmt.Fprintf(detail, "[%s]Status:[-] [%s]%s[-]\n", key, value, evt.Status)
	if evt.Busy {
		_, _ = fmt.Fprintf(detail, "[%s]Availability:[-] [%s]Busy[-]\n", key, value)
	} else {
		_, _ = fmt.Fprintf(detail, "[%s]Availability:[-] [%s]Free[-]\n", key, value)
	}

	_, _ = fmt.Fprintf(detail, "\n\n[%s::d]Press Esc to go back[-::-]", muted)

	// Handle escape
	detail.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEscape {
			v.app.PopDetail()
			return nil
		}
		return event
	})

	v.app.PushDetail("event-detail", detail)
	v.app.SetFocus(detail)
}
