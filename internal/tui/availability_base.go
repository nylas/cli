package tui

import (
	"context"
	"fmt"
	"time"

	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/domain"
	"github.com/rivo/tview"
)

// AvailabilityView displays free/busy information and helps find meeting times.
type AvailabilityView struct {
	app          *App
	layout       *tview.Flex
	name         string
	title        string
	participants []string
	startDate    time.Time
	endDate      time.Time
	duration     int // in minutes
	slots        []domain.AvailableSlot
	freeBusy     []domain.FreeBusyCalendar

	// Calendar selection for creating events
	calendars          []domain.Calendar
	selectedCalendarID string

	// UI components
	participantsList *tview.List
	slotsList        *tview.List
	timeline         *tview.TextView
	infoPanel        *tview.TextView
	focusedPanel     int // 0=participants, 1=slots, 2=timeline
}

// NewAvailabilityView creates a new availability view.
func NewAvailabilityView(app *App) *AvailabilityView {
	v := &AvailabilityView{
		app:       app,
		name:      "availability",
		title:     "Availability",
		startDate: time.Now(),
		endDate:   time.Now().AddDate(0, 0, 7), // Default 1 week
		duration:  30,                          // Default 30 minutes
	}

	v.init()
	return v
}

func (v *AvailabilityView) init() {
	styles := v.app.styles

	// Participants list - uses helper to eliminate ~10 lines of styling
	v.participantsList = NewStyledList(styles, ListViewConfig{
		Title:             "Participants (a=add, d=delete)",
		ShowSecondaryText: false,
	})

	// Available slots list - uses helper
	v.slotsList = NewStyledList(styles, ListViewConfig{
		Title:             "Available Slots",
		ShowSecondaryText: true,
	})

	// Handle slot selection to create event
	v.slotsList.SetSelectedFunc(func(index int, _, _ string, _ rune) {
		if index < len(v.slots) {
			v.createEventFromSlot(v.slots[index])
		}
	})

	// Timeline visualization - uses helper
	v.timeline = NewStyledInfoPanel(styles, "Timeline (Free/Busy)")

	// Info panel - uses helper
	v.infoPanel = NewStyledInfoPanel(styles, "Settings")
	v.updateInfoPanel()

	// Layout:
	// Left column: Participants | Info
	// Right column: Timeline | Available Slots
	leftCol := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(v.participantsList, 0, 1, true).
		AddItem(v.infoPanel, 7, 0, false)

	rightCol := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(v.timeline, 0, 1, false).
		AddItem(v.slotsList, 0, 1, false)

	v.layout = tview.NewFlex().SetDirection(tview.FlexColumn).
		AddItem(leftCol, 35, 0, true).
		AddItem(rightCol, 0, 1, false)

	// Set up input handling
	v.participantsList.SetInputCapture(v.handleParticipantsInput)
	v.slotsList.SetInputCapture(v.handleSlotsInput)
	v.timeline.SetInputCapture(v.handleTimelineInput)
}

func (v *AvailabilityView) Name() string               { return v.name }
func (v *AvailabilityView) Title() string              { return v.title }
func (v *AvailabilityView) Primitive() tview.Primitive { return v.layout }
func (v *AvailabilityView) Filter(string)              {}

func (v *AvailabilityView) Hints() []Hint {
	return []Hint{
		{Key: "a", Desc: "add participant"},
		{Key: "d", Desc: "remove"},
		{Key: "enter", Desc: "create event"},
		{Key: "D", Desc: "set duration"},
		{Key: "S", Desc: "set date range"},
		{Key: "Tab", Desc: "switch panel"},
		{Key: "r", Desc: "refresh"},
	}
}

// Load resolves the current user, calendars, and availability. Network
// fetches run in background goroutines and results are applied on the event
// loop via QueueUpdateDraw. Must be called from the event loop; it is
// non-blocking.
func (v *AvailabilityView) Load() {
	// Load calendars for event creation
	v.loadCalendars()

	// Render the current (possibly empty) state synchronously so the view is
	// never blank while background fetches are in flight.
	v.renderParticipants()
	v.updateInfoPanel()

	// Add current user as first participant if empty. fetchAvailability is
	// skipped here: with no participants it makes no network call and would
	// only show the "add participants" hint while the user is being resolved.
	if len(v.participants) == 0 && v.app.config.GrantID != "" {
		grantID := v.app.config.GrantID
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			grant, err := v.app.config.Client.GetGrant(ctx, grantID)
			v.app.QueueUpdateDraw(func() {
				if !v.app.grantStillCurrent(grantID) {
					return // grant switched while fetch was in flight; drop stale data
				}
				if err == nil && grant.Email != "" {
					v.participants = append(v.participants, grant.Email)
				}
				v.renderParticipants()
				v.updateInfoPanel()
				v.fetchAvailability()
			})
		}()
		return
	}

	v.fetchAvailability()
}

// loadCalendars fetches calendars in a background goroutine and applies the
// results on the event loop via QueueUpdateDraw. Must be called from the
// event loop; it is non-blocking.
func (v *AvailabilityView) loadCalendars() {
	grantID := v.app.config.GrantID

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

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

			// Select primary calendar by default
			for _, cal := range calendars {
				if cal.IsPrimary {
					v.selectedCalendarID = cal.ID
					return
				}
			}

			// Fall back to first calendar
			if len(calendars) > 0 {
				v.selectedCalendarID = calendars[0].ID
			}
		})
	}()
}

func (v *AvailabilityView) Refresh() {
	v.fetchAvailability()
}

func (v *AvailabilityView) updateInfoPanel() {
	styles := v.app.styles
	infoHex := styles.Hex(styles.InfoColor)
	info := fmt.Sprintf("[%s]Duration:[-] %d min\n", infoHex, v.duration)
	info += fmt.Sprintf("[%s]Start:[-] %s\n", infoHex, v.startDate.Format(common.DisplayDateFormat))
	info += fmt.Sprintf("[%s]End:[-] %s\n", infoHex, v.endDate.Format(common.DisplayDateFormat))
	info += fmt.Sprintf("[%s]Participants:[-] %d", infoHex, len(v.participants))
	v.infoPanel.SetText(info)
}

func (v *AvailabilityView) renderParticipants() {
	v.participantsList.Clear()
	for i, email := range v.participants {
		idx := i
		v.participantsList.AddItem(email, "", rune('1'+i), func() {
			// Could show participant details or remove
			_ = idx
		})
	}
}
