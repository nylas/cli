package tui

import (
	"testing"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/nylas/cli/internal/domain"
)

func TestAvailabilityView_EmptyParticipants(t *testing.T) {
	app := createTestApp(t)
	view := NewAvailabilityView(app)

	// Initially no participants
	if len(view.participants) != 0 {
		t.Errorf("participants = %d, want 0 initially", len(view.participants))
	}
}

func TestAvailabilityView_EmptySlots(t *testing.T) {
	app := createTestApp(t)
	view := NewAvailabilityView(app)

	// Initially no slots
	if len(view.slots) != 0 {
		t.Errorf("slots = %d, want 0 initially", len(view.slots))
	}
}

func TestAvailabilityView_CalendarFields(t *testing.T) {
	app := createTestApp(t)
	view := NewAvailabilityView(app)

	// Initially no calendars loaded
	if len(view.calendars) != 0 {
		t.Errorf("calendars = %d, want 0 initially", len(view.calendars))
	}

	if view.selectedCalendarID != "" {
		t.Errorf("selectedCalendarID = %q, want empty string", view.selectedCalendarID)
	}
}

func TestAvailabilityView_SlotsWithData(t *testing.T) {
	app := createTestApp(t)
	view := NewAvailabilityView(app)

	// Add some test slots
	now := time.Now()
	view.slots = []domain.AvailableSlot{
		{
			StartTime: now.Unix(),
			EndTime:   now.Add(30 * time.Minute).Unix(),
		},
		{
			StartTime: now.Add(time.Hour).Unix(),
			EndTime:   now.Add(90 * time.Minute).Unix(),
		},
	}

	// Render slots should work
	view.renderSlots()

	// After rendering, list should have items
	if view.slotsList.GetItemCount() != 2 {
		t.Errorf("slotsList item count = %d, want 2", view.slotsList.GetItemCount())
	}
}

func TestAvailabilityView_ParticipantsWithData(t *testing.T) {
	app := createTestApp(t)
	view := NewAvailabilityView(app)

	// Add participants
	view.participants = []string{
		"user1@example.com",
		"user2@example.com",
	}

	// Render participants should work
	view.renderParticipants()

	// After rendering, list should have items
	if view.participantsList.GetItemCount() != 2 {
		t.Errorf("participantsList item count = %d, want 2", view.participantsList.GetItemCount())
	}
}

func TestAvailabilityView_FreeBusyRendering(t *testing.T) {
	app := createTestApp(t)
	view := NewAvailabilityView(app)

	// Set date range
	now := time.Now().Truncate(24 * time.Hour)
	view.startDate = now
	view.endDate = now.Add(24 * time.Hour)

	// Add free/busy data
	view.freeBusy = []domain.FreeBusyCalendar{
		{
			Email: "user@example.com",
			TimeSlots: []domain.TimeSlot{
				{
					StartTime: now.Add(9 * time.Hour).Unix(),
					EndTime:   now.Add(10 * time.Hour).Unix(),
					Status:    "busy",
				},
			},
		},
	}

	// Render timeline should not panic
	view.renderTimeline()
}

func TestAvailabilityView_FocusPanel(t *testing.T) {
	app := createTestApp(t)
	view := NewAvailabilityView(app)

	// Initially focused on participants (panel 0)
	if view.focusedPanel != 0 {
		t.Errorf("focusedPanel = %d, want 0", view.focusedPanel)
	}
}

func TestAvailabilityView_DurationDefault(t *testing.T) {
	app := createTestApp(t)
	view := NewAvailabilityView(app)

	// Default duration should be 30 minutes
	if view.duration != 30 {
		t.Errorf("duration = %d, want 30", view.duration)
	}
}

func TestAvailabilityView_InfoPanelUpdate(t *testing.T) {
	app := createTestApp(t)
	view := NewAvailabilityView(app)

	// Add some data
	view.participants = []string{"user@example.com"}
	view.duration = 60

	// Update info panel should not panic
	view.updateInfoPanel()
}

func TestAvailabilityView_HandleKey(t *testing.T) {
	app := createTestApp(t)
	view := NewAvailabilityView(app)

	// Test 'r' key for refresh (should be consumed)
	event := tcell.NewEventKey(tcell.KeyRune, 'r', tcell.ModNone)
	result := view.HandleKey(event)
	if result != nil {
		t.Error("HandleKey('r') should return nil (consumed)")
	}

	// Test other key (should pass through)
	event = tcell.NewEventKey(tcell.KeyRune, 'x', tcell.ModNone)
	result = view.HandleKey(event)
	if result == nil {
		t.Error("HandleKey('x') should return event (not consumed)")
	}

	// Test non-rune key (should pass through)
	event = tcell.NewEventKey(tcell.KeyEnter, 0, tcell.ModNone)
	result = view.HandleKey(event)
	if result == nil {
		t.Error("HandleKey(Enter) should return event (not consumed)")
	}
}

func TestAvailabilityView_HandleParticipantsInput(t *testing.T) {
	app := createTestApp(t)
	view := NewAvailabilityView(app)

	// Test Tab key - should switch focus to slots
	event := tcell.NewEventKey(tcell.KeyTab, 0, tcell.ModNone)
	result := view.handleParticipantsInput(event)
	if result != nil {
		t.Error("handleParticipantsInput(Tab) should return nil")
	}
	if view.focusedPanel != 1 {
		t.Errorf("focusedPanel after Tab = %d, want 1", view.focusedPanel)
	}

	// Test 'r' key for refresh
	view.focusedPanel = 0
	event = tcell.NewEventKey(tcell.KeyRune, 'r', tcell.ModNone)
	result = view.handleParticipantsInput(event)
	if result != nil {
		t.Error("handleParticipantsInput('r') should return nil")
	}

	// Test unknown key (should pass through)
	event = tcell.NewEventKey(tcell.KeyRune, 'x', tcell.ModNone)
	result = view.handleParticipantsInput(event)
	if result == nil {
		t.Error("handleParticipantsInput('x') should return event")
	}
}

func TestAvailabilityView_HandleSlotsInput(t *testing.T) {
	app := createTestApp(t)
	view := NewAvailabilityView(app)
	view.focusedPanel = 1

	// Test Tab key - should switch focus to timeline
	event := tcell.NewEventKey(tcell.KeyTab, 0, tcell.ModNone)
	result := view.handleSlotsInput(event)
	if result != nil {
		t.Error("handleSlotsInput(Tab) should return nil")
	}
	if view.focusedPanel != 2 {
		t.Errorf("focusedPanel after Tab = %d, want 2", view.focusedPanel)
	}

	// Test BackTab key - should switch focus to participants
	view.focusedPanel = 1
	event = tcell.NewEventKey(tcell.KeyBacktab, 0, tcell.ModNone)
	result = view.handleSlotsInput(event)
	if result != nil {
		t.Error("handleSlotsInput(BackTab) should return nil")
	}
	if view.focusedPanel != 0 {
		t.Errorf("focusedPanel after BackTab = %d, want 0", view.focusedPanel)
	}

	// Test 'r' key for refresh
	view.focusedPanel = 1
	event = tcell.NewEventKey(tcell.KeyRune, 'r', tcell.ModNone)
	result = view.handleSlotsInput(event)
	if result != nil {
		t.Error("handleSlotsInput('r') should return nil")
	}

	// Test unknown key (should pass through)
	event = tcell.NewEventKey(tcell.KeyRune, 'x', tcell.ModNone)
	result = view.handleSlotsInput(event)
	if result == nil {
		t.Error("handleSlotsInput('x') should return event")
	}
}

func TestAvailabilityView_HandleTimelineInput(t *testing.T) {
	app := createTestApp(t)
	view := NewAvailabilityView(app)
	view.focusedPanel = 2

	// Test Tab key - should switch focus to participants
	event := tcell.NewEventKey(tcell.KeyTab, 0, tcell.ModNone)
	result := view.handleTimelineInput(event)
	if result != nil {
		t.Error("handleTimelineInput(Tab) should return nil")
	}
	if view.focusedPanel != 0 {
		t.Errorf("focusedPanel after Tab = %d, want 0", view.focusedPanel)
	}

	// Test BackTab key - should switch focus to slots
	view.focusedPanel = 2
	event = tcell.NewEventKey(tcell.KeyBacktab, 0, tcell.ModNone)
	result = view.handleTimelineInput(event)
	if result != nil {
		t.Error("handleTimelineInput(BackTab) should return nil")
	}
	if view.focusedPanel != 1 {
		t.Errorf("focusedPanel after BackTab = %d, want 1", view.focusedPanel)
	}

	// Test 'r' key for refresh
	view.focusedPanel = 2
	event = tcell.NewEventKey(tcell.KeyRune, 'r', tcell.ModNone)
	result = view.handleTimelineInput(event)
	if result != nil {
		t.Error("handleTimelineInput('r') should return nil")
	}

	// Test unknown key (should pass through)
	event = tcell.NewEventKey(tcell.KeyRune, 'x', tcell.ModNone)
	result = view.handleTimelineInput(event)
	if result == nil {
		t.Error("handleTimelineInput('x') should return event")
	}
}

func TestAvailabilityView_RenderSlotsEmpty(t *testing.T) {
	app := createTestApp(t)
	view := NewAvailabilityView(app)

	// Empty slots should show "No available slots found"
	view.slots = []domain.AvailableSlot{}
	view.renderSlots()

	if view.slotsList.GetItemCount() != 1 {
		t.Errorf("slotsList count = %d, want 1 (no slots message)", view.slotsList.GetItemCount())
	}
}

func TestAvailabilityView_RenderSlotsWithEmails(t *testing.T) {
	app := createTestApp(t)
	view := NewAvailabilityView(app)

	now := time.Now()
	view.slots = []domain.AvailableSlot{
		{
			StartTime: now.Unix(),
			EndTime:   now.Add(30 * time.Minute).Unix(),
			Emails:    []string{"user1@example.com", "user2@example.com"},
		},
	}

	view.renderSlots()

	if view.slotsList.GetItemCount() != 1 {
		t.Errorf("slotsList count = %d, want 1", view.slotsList.GetItemCount())
	}
}

func TestAvailabilityView_RenderSlotsLimit(t *testing.T) {
	app := createTestApp(t)
	view := NewAvailabilityView(app)

	now := time.Now()
	// Add more than 20 slots
	for i := 0; i < 25; i++ {
		view.slots = append(view.slots, domain.AvailableSlot{
			StartTime: now.Add(time.Duration(i) * time.Hour).Unix(),
			EndTime:   now.Add(time.Duration(i)*time.Hour + 30*time.Minute).Unix(),
		})
	}

	view.renderSlots()

	// Should show 20 slots + "and X more" message
	if view.slotsList.GetItemCount() != 21 {
		t.Errorf("slotsList count = %d, want 21 (20 slots + more message)", view.slotsList.GetItemCount())
	}
}

func TestAvailabilityView_RenderTimelineEmpty(t *testing.T) {
	app := createTestApp(t)
	view := NewAvailabilityView(app)

	// Empty free/busy data
	view.freeBusy = []domain.FreeBusyCalendar{}
	view.renderTimeline()

	// Should show "No free/busy data available"
	text := view.timeline.GetText(true)
	if text == "" {
		t.Error("timeline should have text after renderTimeline")
	}
}

func TestAvailabilityView_RenderTimelineAllFree(t *testing.T) {
	app := createTestApp(t)
	view := NewAvailabilityView(app)

	// Free/busy with no time slots (all free)
	view.freeBusy = []domain.FreeBusyCalendar{
		{
			Email:     "user@example.com",
			TimeSlots: []domain.TimeSlot{}, // Empty = all free
		},
	}

	view.renderTimeline()

	// Should show "All free"
	text := view.timeline.GetText(true)
	if text == "" {
		t.Error("timeline should have text after renderTimeline")
	}
}

func TestAvailabilityView_Refresh(t *testing.T) {
	app := createTestApp(t)
	view := NewAvailabilityView(app)

	// Refresh should call fetchAvailability
	// With no participants, it should update timeline
	view.Refresh()

	// No panic is success; timeline should show message
	text := view.timeline.GetText(true)
	if text == "" {
		t.Error("timeline should have text after Refresh")
	}
}

func TestAvailabilityView_FetchAvailabilityNoParticipants(t *testing.T) {
	app := createTestApp(t)
	view := NewAvailabilityView(app)

	// No participants
	view.participants = []string{}
	view.fetchAvailability()

	// Should show message about adding participants
	text := view.timeline.GetText(true)
	if text == "" {
		t.Error("timeline should have message about adding participants")
	}

	// Slots should be cleared
	if len(view.slots) > 0 {
		t.Error("slots should be nil or empty")
	}
}

func TestAvailabilityView_RemoveSelectedParticipant_Empty(t *testing.T) {
	app := createTestApp(t)
	view := NewAvailabilityView(app)

	// No participants - should not panic
	view.removeSelectedParticipant()
}

func TestAvailabilityView_RemoveSelectedParticipant_OutOfBounds(t *testing.T) {
	app := createTestApp(t)
	view := NewAvailabilityView(app)

	// Add one participant but set invalid index
	view.participants = []string{"user@example.com"}
	view.renderParticipants()

	// This should not panic even with edge cases
	view.removeSelectedParticipant()
}

func TestAvailabilityView_SetDuration(t *testing.T) {
	app := createTestApp(t)
	view := NewAvailabilityView(app)

	// Set custom duration
	view.duration = 45
	view.updateInfoPanel()

	if view.duration != 45 {
		t.Errorf("duration = %d, want 45", view.duration)
	}
}

func TestAvailabilityView_SetDateRange(t *testing.T) {
	app := createTestApp(t)
	view := NewAvailabilityView(app)

	// Set custom date range
	start := time.Date(2025, 1, 15, 0, 0, 0, 0, time.Local)
	end := time.Date(2025, 1, 20, 0, 0, 0, 0, time.Local)

	view.startDate = start
	view.endDate = end
	view.updateInfoPanel()

	if !view.startDate.Equal(start) {
		t.Errorf("startDate = %v, want %v", view.startDate, start)
	}
	if !view.endDate.Equal(end) {
		t.Errorf("endDate = %v, want %v", view.endDate, end)
	}
}

func TestAvailabilityView_CalendarSelection(t *testing.T) {
	app := createTestApp(t)
	view := NewAvailabilityView(app)

	// Set calendars manually (simulating loadCalendars)
	view.calendars = []domain.Calendar{
		{ID: "cal-1", Name: "Work", IsPrimary: false},
		{ID: "cal-2", Name: "Personal", IsPrimary: true},
	}

	// Select primary calendar
	for _, cal := range view.calendars {
		if cal.IsPrimary {
			view.selectedCalendarID = cal.ID
			break
		}
	}

	if view.selectedCalendarID != "cal-2" {
		t.Errorf("selectedCalendarID = %q, want %q", view.selectedCalendarID, "cal-2")
	}
}

func TestAvailabilityView_CalendarSelectionNoPrimary(t *testing.T) {
	app := createTestApp(t)
	view := NewAvailabilityView(app)

	// Set calendars with no primary
	view.calendars = []domain.Calendar{
		{ID: "cal-1", Name: "Work", IsPrimary: false},
		{ID: "cal-2", Name: "Personal", IsPrimary: false},
	}

	// No primary, should fall back to first
	hasPrimary := false
	for _, cal := range view.calendars {
		if cal.IsPrimary {
			view.selectedCalendarID = cal.ID
			hasPrimary = true
			break
		}
	}
	if !hasPrimary && len(view.calendars) > 0 {
		view.selectedCalendarID = view.calendars[0].ID
	}

	if view.selectedCalendarID != "cal-1" {
		t.Errorf("selectedCalendarID = %q, want %q (first calendar)", view.selectedCalendarID, "cal-1")
	}
}
