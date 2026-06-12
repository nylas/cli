package tui

import (
	"context"
	"fmt"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/domain"
	"github.com/rivo/tview"
)

// EventsView displays a Google Calendar-style calendar view.
type EventsView struct {
	app          *App
	layout       *tview.Flex
	calendar     *CalendarView
	eventsList   *tview.TextView
	events       []domain.Event
	calendars    []domain.Calendar
	name         string
	title        string
	focusedPanel int // 0 = calendar, 1 = events list
}

// NewEventsView creates a new calendar-style events view.
func NewEventsView(app *App) *EventsView {
	v := &EventsView{
		app:   app,
		name:  "events",
		title: "Calendar",
	}

	// Create calendar view
	v.calendar = NewCalendarView(app)
	v.calendar.SetOnDateSelect(v.onDateSelect)
	v.calendar.SetOnCalendarChange(v.onCalendarChange)

	// Create events list panel
	v.eventsList = tview.NewTextView()
	v.eventsList.SetDynamicColors(true)
	v.eventsList.SetBackgroundColor(app.styles.BgColor)
	v.eventsList.SetBorder(true)
	v.eventsList.SetBorderColor(app.styles.BorderColor)
	v.eventsList.SetTitle(" Events ")
	v.eventsList.SetTitleColor(app.styles.TitleFg)
	v.eventsList.SetBorderPadding(0, 0, 1, 1)

	// Create split layout: Calendar (left) | Events List (right)
	v.layout = tview.NewFlex().SetDirection(tview.FlexColumn).
		AddItem(v.calendar, 0, 2, true).
		AddItem(v.eventsList, 0, 1, false)

	return v
}

func (v *EventsView) Name() string               { return v.name }
func (v *EventsView) Title() string              { return v.title }
func (v *EventsView) Primitive() tview.Primitive { return v.layout }
func (v *EventsView) Filter(string)              {}

func (v *EventsView) Hints() []Hint {
	return []Hint{
		{Key: "enter", Desc: "view day"},
		{Key: "n", Desc: "new event"},
		{Key: "c/C", Desc: "switch/list cal"},
		{Key: "m", Desc: "month"},
		{Key: "w", Desc: "week"},
		{Key: "a", Desc: "agenda"},
		{Key: "t", Desc: "today"},
		{Key: "H/L", Desc: "±month"},
		{Key: "r", Desc: "refresh"},
	}
}

// Load fetches calendars in a background goroutine and applies the results on
// the event loop via QueueUpdateDraw. Must be called from the event loop;
// it is non-blocking.
func (v *EventsView) Load() {
	grantID := v.app.config.GrantID

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		// Get calendars first
		calendars, err := v.app.config.Client.GetCalendars(ctx, grantID)
		if err != nil {
			v.app.FlashLoadError("Failed to load calendars", err)
			return
		}

		v.app.QueueUpdateDraw(func() {
			if !v.app.grantStillCurrent(grantID) {
				return // grant switched while fetch was in flight; drop stale data
			}
			v.calendars = calendars
			v.calendar.SetCalendars(calendars)

			if len(calendars) == 0 {
				v.app.Flash(FlashWarn, "No calendars found")
				return
			}

			// Load events for the current calendar
			v.loadEventsForCalendar(v.calendar.GetCurrentCalendarID())
		})
	}()
}

// loadEventsForCalendar fetches events in a background goroutine and applies
// the results on the event loop via QueueUpdateDraw. Must be called from the
// event loop; it is non-blocking.
func (v *EventsView) loadEventsForCalendar(calendarID string) {
	grantID := v.app.config.GrantID

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		// Get events from selected calendar (fetch 2 months range)
		now := time.Now()
		startTime := now.AddDate(0, -1, 0).Unix()
		endTime := now.AddDate(0, 2, 0).Unix()

		events, err := v.app.config.Client.GetEvents(ctx, grantID, calendarID, &domain.EventQueryParams{
			Start:           startTime,
			End:             endTime,
			ExpandRecurring: true,
			Limit:           200,
		})
		if err != nil {
			v.app.FlashLoadError("Failed to load events", err)
			return
		}

		v.app.QueueUpdateDraw(func() {
			if !v.app.grantStillCurrent(grantID) {
				return // grant switched while fetch was in flight; drop stale data
			}
			if v.calendar.GetCurrentCalendarID() != calendarID {
				return // calendar switched while fetch was in flight; drop stale data
			}
			v.events = events
			v.calendar.SetEvents(events)
			v.updateEventsList(v.calendar.GetSelectedDate())

			// Show calendar name in flash
			if cal := v.calendar.GetCurrentCalendar(); cal != nil {
				v.app.Flash(FlashInfo, "Calendar: %s (%d events)", cal.Name, len(events))
			}
		})
	}()
}

func (v *EventsView) Refresh() { v.Load() }

func (v *EventsView) onCalendarChange(calendarID string) {
	// Reload events for the new calendar (non-blocking)
	v.loadEventsForCalendar(calendarID)
}

func (v *EventsView) onDateSelect(date time.Time) {
	v.updateEventsList(date)
}

func (v *EventsView) updateEventsList(date time.Time) {
	v.eventsList.Clear()

	events := v.calendar.GetEventsForDate(date)
	// Use cached Hex() method
	s := v.app.styles
	title := s.Hex(s.TitleFg)
	info := s.Hex(s.InfoColor)
	muted := s.Hex(s.BorderColor)
	eventColor := s.Hex(s.FgColor)
	success := s.Hex(s.SuccessColor)

	// Header with date
	dateStr := date.Format(common.DisplayDateLong)
	_, _ = fmt.Fprintf(v.eventsList, "[%s::b]%s[-::-]\n\n", title, dateStr)

	if len(events) == 0 {
		_, _ = fmt.Fprintf(v.eventsList, "[%s]No events scheduled[-]\n", muted)
		return
	}

	for i, evt := range events {
		// Time
		timeStr := "All day"
		if !evt.When.IsAllDay() {
			start := evt.When.StartDateTime()
			end := evt.When.EndDateTime()
			timeStr = fmt.Sprintf("%s - %s", start.Format("3:04 PM"), end.Format("3:04 PM"))
		}

		// Event entry
		_, _ = fmt.Fprintf(v.eventsList, "[%s]%s[-]\n", info, timeStr)

		// Title with recurring indicator
		title := evt.Title
		if isRecurringEvent(&evt) {
			title = "🔁 " + title
		}
		_, _ = fmt.Fprintf(v.eventsList, "[%s::b]%s[-::-]\n", eventColor, title)

		// Location
		if evt.Location != "" {
			_, _ = fmt.Fprintf(v.eventsList, "[%s]📍 %s[-]\n", muted, evt.Location)
		}

		// Status
		statusIcon := "✓"
		switch evt.Status {
		case "tentative":
			statusIcon = "?"
		case "cancelled":
			statusIcon = "✗"
		}
		_, _ = fmt.Fprintf(v.eventsList, "[%s]%s %s[-]\n", success, statusIcon, evt.Status)

		// Separator between events
		if i < len(events)-1 {
			_, _ = fmt.Fprintf(v.eventsList, "\n[%s]───────────────────[-]\n\n", muted)
		}
	}
}

func (v *EventsView) HandleKey(event *tcell.EventKey) *tcell.EventKey {
	switch event.Key() {
	case tcell.KeyEscape:
		// Let the app handle Escape for navigation
		return event

	case tcell.KeyTab:
		// Switch focus between calendar and events list
		v.focusedPanel = (v.focusedPanel + 1) % 2
		if v.focusedPanel == 0 {
			v.app.SetFocus(v.calendar)
		} else {
			v.app.SetFocus(v.eventsList)
		}
		return nil

	case tcell.KeyEnter:
		// Show detailed view of selected day's events
		v.showDayDetail()
		return nil

	case tcell.KeyRune:
		switch event.Rune() {
		case 'n': // New event
			v.createNewEvent()
			return nil
		case 'C': // Show calendar list
			v.showCalendarList()
			return nil
		}
	}

	// Pass to calendar if it has focus
	if v.focusedPanel == 0 {
		handler := v.calendar.InputHandler()
		if handler != nil {
			handler(event, func(p tview.Primitive) {})
			v.updateEventsList(v.calendar.GetSelectedDate())
			return nil
		}
	}

	return event
}

func (v *EventsView) showCalendarList() {
	calendars := v.calendar.GetCalendars()
	if len(calendars) == 0 {
		v.app.Flash(FlashWarn, "No calendars available")
		return
	}

	// Create a list view for calendar selection - uses helper
	list := NewStyledList(v.app.styles, ListViewConfig{
		Title:             "Select Calendar",
		ShowSecondaryText: true,
		HighlightFullLine: true,
		UseTableSelectBg:  true,
	})
	list.SetBorderColor(v.app.styles.FocusColor) // Override for focus styling
	list.SetSecondaryTextColor(v.app.styles.BorderColor)

	currentCal := v.calendar.GetCurrentCalendar()

	for i, cal := range calendars {
		name := cal.Name
		secondary := cal.ID
		if len(secondary) > 40 {
			secondary = secondary[:37] + "..."
		}

		// Add description if available
		if cal.Description != "" {
			desc := cal.Description
			if len(desc) > 30 {
				desc = desc[:27] + "..."
			}
			secondary = desc + " | " + secondary
		}

		// Add color indicator
		if cal.HexColor != "" {
			name = "■ " + name // Color square (will be colored in custom draw)
		}

		// Mark primary and current
		if cal.IsPrimary {
			name = "★ " + name
		}
		if currentCal != nil && cal.ID == currentCal.ID {
			name = "● " + name
		}

		// Add read-only indicator
		if cal.ReadOnly {
			name = name + " [RO]"
		}

		idx := i // Capture for closure
		list.AddItem(name, secondary, rune('1'+i), func() {
			v.calendar.SetCalendarByIndex(idx)
			v.app.PopDetail()
			v.app.SetFocus(v.calendar)
		})
	}

	// Handle escape to close
	list.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEscape {
			v.app.PopDetail()
			v.app.SetFocus(v.calendar)
			return nil
		}
		return event
	})

	// Push the list as a detail view
	v.app.PushDetail("calendar-list", list)
	v.app.SetFocus(list)
}

func (v *EventsView) createNewEvent() {
	calendarID := v.calendar.GetCurrentCalendarID()
	if calendarID == "" {
		v.app.Flash(FlashWarn, "No calendar selected")
		return
	}

	v.app.ShowEventForm(calendarID, nil, func(event *domain.Event) {
		// Refresh events after creation (non-blocking)
		v.loadEventsForCalendar(calendarID)
	})
}
