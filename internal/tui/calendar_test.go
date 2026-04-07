package tui

import (
	"testing"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/nylas/cli/internal/domain"
)

func TestNewCalendarView(t *testing.T) {
	app := createTestApp(t)

	view := NewCalendarView(app)

	if view == nil {
		t.Fatal("NewCalendarView returned nil")
		return
	}

	if view.viewMode != CalendarMonthView {
		t.Errorf("viewMode = %d, want %d (CalendarMonthView)", view.viewMode, CalendarMonthView)
	}
}

func TestCalendarView_GetCalendars(t *testing.T) {
	app := createTestApp(t)
	view := NewCalendarView(app)

	// Initially empty
	calendars := view.GetCalendars()
	if len(calendars) != 0 {
		t.Errorf("GetCalendars() = %d items, want 0", len(calendars))
	}
}

func TestCalendarView_SetViewMode(t *testing.T) {
	app := createTestApp(t)
	view := NewCalendarView(app)

	view.SetViewMode(CalendarWeekView)
	if view.viewMode != CalendarWeekView {
		t.Errorf("viewMode = %d, want %d (CalendarWeekView)", view.viewMode, CalendarWeekView)
	}

	view.SetViewMode(CalendarMonthView)
	if view.viewMode != CalendarMonthView {
		t.Errorf("viewMode = %d, want %d (CalendarMonthView)", view.viewMode, CalendarMonthView)
	}
}

func TestCalendarView_GoToToday(t *testing.T) {
	app := createTestApp(t)
	view := NewCalendarView(app)

	// Move to a different date first
	view.selectedDate = view.selectedDate.AddDate(0, 1, 0)
	originalDate := view.selectedDate

	view.GoToToday()

	if view.selectedDate.Equal(originalDate) {
		t.Error("GoToToday() did not change date")
	}
}

func TestCalendarView_GetSelectedDate(t *testing.T) {
	app := createTestApp(t)
	view := NewCalendarView(app)

	date := view.GetSelectedDate()
	if date.IsZero() {
		t.Error("GetSelectedDate() returned zero time")
	}
}

func TestCalendarView_GetEventsForDate(t *testing.T) {
	app := createTestApp(t)
	view := NewCalendarView(app)

	// No events initially
	events := view.GetEventsForDate(view.selectedDate)
	if len(events) != 0 {
		t.Errorf("GetEventsForDate() = %d events, want 0", len(events))
	}
}

func TestCalendarView_SetOnCalendarChange(t *testing.T) {
	app := createTestApp(t)
	view := NewCalendarView(app)

	view.SetOnCalendarChange(func(calendarID string) {
		// callback set
	})

	if view.onCalendarChange == nil {
		t.Error("SetOnCalendarChange did not set callback")
	}
}

func TestCalendarView_SetOnDateSelect(t *testing.T) {
	app := createTestApp(t)
	view := NewCalendarView(app)

	view.SetOnDateSelect(func(selectedDate time.Time) {
		// callback set
	})

	if view.onDateSelect == nil {
		t.Error("SetOnDateSelect did not set callback")
	}
}

func TestParseHexColor(t *testing.T) {
	tests := []struct {
		name        string
		hex         string
		wantR       int32
		wantG       int32
		wantB       int32
		wantDefault bool
	}{
		{
			name:  "red with hash",
			hex:   "#FF0000",
			wantR: 255, wantG: 0, wantB: 0,
		},
		{
			name:  "green without hash",
			hex:   "00FF00",
			wantR: 0, wantG: 255, wantB: 0,
		},
		{
			name:  "blue lowercase",
			hex:   "#0000ff",
			wantR: 0, wantG: 0, wantB: 255,
		},
		{
			name:  "mixed case",
			hex:   "#AbCdEf",
			wantR: 171, wantG: 205, wantB: 239,
		},
		{
			name:  "gray",
			hex:   "#808080",
			wantR: 128, wantG: 128, wantB: 128,
		},
		{
			name:        "too short",
			hex:         "#FFF",
			wantDefault: true,
		},
		{
			name:        "too long",
			hex:         "#FFFFFFF",
			wantDefault: true,
		},
		{
			name:        "empty",
			hex:         "",
			wantDefault: true,
		},
		{
			name:  "invalid chars become black",
			hex:   "#GGGGGG",
			wantR: 0, wantG: 0, wantB: 0, // Invalid chars parse to 0
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseHexColor(tt.hex)

			if tt.wantDefault {
				if result != tcell.ColorDefault {
					t.Errorf("parseHexColor(%q) = %v, want ColorDefault", tt.hex, result)
				}
				return
			}

			expected := tcell.NewRGBColor(tt.wantR, tt.wantG, tt.wantB)
			if result != expected {
				t.Errorf("parseHexColor(%q) = %v, want %v", tt.hex, result, expected)
			}
		})
	}
}

func TestParseHexDigits(t *testing.T) {
	tests := []struct {
		input    string
		expected int
	}{
		{"00", 0},
		{"FF", 255},
		{"ff", 255},
		{"80", 128},
		{"0A", 10},
		{"a0", 160},
		{"AB", 171},
		{"cd", 205},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := parseHexDigits(tt.input)
			if result != tt.expected {
				t.Errorf("parseHexDigits(%q) = %d, want %d", tt.input, result, tt.expected)
			}
		})
	}
}

func TestHexCharToInt(t *testing.T) {
	tests := []struct {
		input    byte
		expected int
	}{
		{'0', 0},
		{'1', 1},
		{'9', 9},
		{'a', 10},
		{'A', 10},
		{'f', 15},
		{'F', 15},
		{'g', 0}, // invalid
		{'G', 0}, // invalid
		{'z', 0}, // invalid
	}

	for _, tt := range tests {
		t.Run(string(tt.input), func(t *testing.T) {
			result := hexCharToInt(tt.input)
			if result != tt.expected {
				t.Errorf("hexCharToInt(%q) = %d, want %d", tt.input, result, tt.expected)
			}
		})
	}
}

func TestCalendarView_GetCalendarColor(t *testing.T) {
	app := createTestApp(t)
	view := NewCalendarView(app)

	// No calendar - should return default
	color := view.getCalendarColor()
	if color != tcell.ColorDefault {
		t.Errorf("getCalendarColor() with no calendar = %v, want ColorDefault", color)
	}

	// Set calendar with color
	view.calendars = []domain.Calendar{
		{ID: "cal-1", Name: "Test", HexColor: "#FF0000"},
	}
	view.calendarIndex = 0

	color = view.getCalendarColor()
	expected := tcell.NewRGBColor(255, 0, 0)
	if color != expected {
		t.Errorf("getCalendarColor() = %v, want %v", color, expected)
	}
}

func TestCalendarView_GetCurrentCalendar(t *testing.T) {
	app := createTestApp(t)
	view := NewCalendarView(app)

	// No calendars
	cal := view.GetCurrentCalendar()
	if cal != nil {
		t.Error("GetCurrentCalendar() with no calendars should return nil")
	}

	// Add calendars
	view.calendars = []domain.Calendar{
		{ID: "cal-1", Name: "Calendar 1"},
		{ID: "cal-2", Name: "Calendar 2"},
	}
	view.calendarIndex = 1

	cal = view.GetCurrentCalendar()
	if cal == nil {
		t.Fatal("GetCurrentCalendar() returned nil")
		return
	}
	if cal.ID != "cal-2" {
		t.Errorf("GetCurrentCalendar().ID = %q, want %q", cal.ID, "cal-2")
	}
}

func TestCalendarView_GetCurrentCalendarID(t *testing.T) {
	app := createTestApp(t)
	view := NewCalendarView(app)

	// No calendars
	id := view.GetCurrentCalendarID()
	if id != "" {
		t.Errorf("GetCurrentCalendarID() with no calendars = %q, want empty", id)
	}

	// Use SetCalendars which properly sets calendarID
	view.SetCalendars([]domain.Calendar{
		{ID: "cal-1", Name: "Calendar 1"},
	})

	id = view.GetCurrentCalendarID()
	if id != "cal-1" {
		t.Errorf("GetCurrentCalendarID() = %q, want %q", id, "cal-1")
	}
}

func TestCalendarView_NextCalendar(t *testing.T) {
	app := createTestApp(t)
	view := NewCalendarView(app)

	// No calendars - should not panic
	view.NextCalendar()

	// Add calendars
	view.calendars = []domain.Calendar{
		{ID: "cal-1", Name: "Calendar 1"},
		{ID: "cal-2", Name: "Calendar 2"},
		{ID: "cal-3", Name: "Calendar 3"},
	}
	view.calendarIndex = 0

	view.NextCalendar()
	if view.calendarIndex != 1 {
		t.Errorf("calendarIndex = %d, want 1", view.calendarIndex)
	}

	view.NextCalendar()
	if view.calendarIndex != 2 {
		t.Errorf("calendarIndex = %d, want 2", view.calendarIndex)
	}

	// Wrap around
	view.NextCalendar()
	if view.calendarIndex != 0 {
		t.Errorf("calendarIndex = %d, want 0 (wrap)", view.calendarIndex)
	}
}

func TestCalendarView_PrevCalendar(t *testing.T) {
	app := createTestApp(t)
	view := NewCalendarView(app)

	// No calendars - should not panic
	view.PrevCalendar()

	// Add calendars
	view.calendars = []domain.Calendar{
		{ID: "cal-1", Name: "Calendar 1"},
		{ID: "cal-2", Name: "Calendar 2"},
		{ID: "cal-3", Name: "Calendar 3"},
	}
	view.calendarIndex = 2

	view.PrevCalendar()
	if view.calendarIndex != 1 {
		t.Errorf("calendarIndex = %d, want 1", view.calendarIndex)
	}

	view.PrevCalendar()
	if view.calendarIndex != 0 {
		t.Errorf("calendarIndex = %d, want 0", view.calendarIndex)
	}

	// Wrap around
	view.PrevCalendar()
	if view.calendarIndex != 2 {
		t.Errorf("calendarIndex = %d, want 2 (wrap)", view.calendarIndex)
	}
}

func TestCalendarView_SetCalendarByIndex(t *testing.T) {
	app := createTestApp(t)
	view := NewCalendarView(app)

	view.calendars = []domain.Calendar{
		{ID: "cal-1", Name: "Calendar 1"},
		{ID: "cal-2", Name: "Calendar 2"},
	}

	view.SetCalendarByIndex(1)
	if view.calendarIndex != 1 {
		t.Errorf("calendarIndex = %d, want 1", view.calendarIndex)
	}

	// Out of bounds - should not change
	view.SetCalendarByIndex(10)
	if view.calendarIndex != 1 {
		t.Errorf("calendarIndex = %d, want 1 (unchanged)", view.calendarIndex)
	}

	view.SetCalendarByIndex(-1)
	if view.calendarIndex != 1 {
		t.Errorf("calendarIndex = %d, want 1 (unchanged)", view.calendarIndex)
	}
}

func TestCalendarView_ToggleViewMode(t *testing.T) {
	app := createTestApp(t)
	view := NewCalendarView(app)

	if view.viewMode != CalendarMonthView {
		t.Fatalf("initial viewMode = %d, want %d", view.viewMode, CalendarMonthView)
	}

	view.ToggleViewMode()
	if view.viewMode != CalendarWeekView {
		t.Errorf("viewMode after toggle = %d, want %d", view.viewMode, CalendarWeekView)
	}

	view.ToggleViewMode()
	if view.viewMode != CalendarAgendaView {
		t.Errorf("viewMode after second toggle = %d, want %d", view.viewMode, CalendarAgendaView)
	}

	view.ToggleViewMode()
	if view.viewMode != CalendarMonthView {
		t.Errorf("viewMode after third toggle = %d, want %d", view.viewMode, CalendarMonthView)
	}
}

func TestCalendarView_NextMonth(t *testing.T) {
	app := createTestApp(t)
	view := NewCalendarView(app)

	originalMonth := view.currentMonth.Month()
	view.NextMonth()

	expectedMonth := originalMonth + 1
	if expectedMonth > 12 {
		expectedMonth = 1
	}

	if view.currentMonth.Month() != expectedMonth {
		t.Errorf("month after NextMonth = %d, want %d", view.currentMonth.Month(), expectedMonth)
	}
}

func TestCalendarView_PrevMonth(t *testing.T) {
	app := createTestApp(t)
	view := NewCalendarView(app)

	originalMonth := view.currentMonth.Month()
	view.PrevMonth()

	expectedMonth := originalMonth - 1
	if expectedMonth < 1 {
		expectedMonth = 12
	}

	if view.currentMonth.Month() != expectedMonth {
		t.Errorf("month after PrevMonth = %d, want %d", view.currentMonth.Month(), expectedMonth)
	}
}

func TestCalendarView_NextWeek(t *testing.T) {
	app := createTestApp(t)
	view := NewCalendarView(app)

	originalDay := view.selectedDate.YearDay()
	view.NextWeek()

	expectedDay := originalDay + 7
	if view.selectedDate.YearDay() != expectedDay && view.selectedDate.YearDay() != expectedDay-365 {
		t.Errorf("day after NextWeek = %d, want %d", view.selectedDate.YearDay(), expectedDay)
	}
}

func TestCalendarView_PrevWeek(t *testing.T) {
	app := createTestApp(t)
	view := NewCalendarView(app)

	originalDay := view.selectedDate.YearDay()
	view.PrevWeek()

	expectedDay := originalDay - 7
	if expectedDay < 1 {
		expectedDay += 365
	}

	if view.selectedDate.YearDay() != expectedDay && view.selectedDate.YearDay() != expectedDay+365 {
		t.Errorf("day after PrevWeek = %d, want around %d", view.selectedDate.YearDay(), expectedDay)
	}
}

func TestCalendarView_SetCalendars(t *testing.T) {
	app := createTestApp(t)
	view := NewCalendarView(app)

	calendars := []domain.Calendar{
		{ID: "cal-1", Name: "Work", IsPrimary: false},
		{ID: "cal-2", Name: "Personal", IsPrimary: true},
		{ID: "cal-3", Name: "Holidays", IsPrimary: false},
	}

	view.SetCalendars(calendars)

	if len(view.calendars) != 3 {
		t.Errorf("calendars length = %d, want 3", len(view.calendars))
	}

	// Should select primary calendar
	if view.calendarIndex != 1 {
		t.Errorf("calendarIndex = %d, want 1 (primary)", view.calendarIndex)
	}
}

func TestCalendarView_SetEvents(t *testing.T) {
	app := createTestApp(t)
	view := NewCalendarView(app)

	events := []domain.Event{
		{ID: "evt-1", Title: "Meeting 1"},
		{ID: "evt-2", Title: "Meeting 2"},
	}

	view.SetEvents(events)

	if len(view.events) != 2 {
		t.Errorf("events length = %d, want 2", len(view.events))
	}
}
