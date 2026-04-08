package tui

import (
	"testing"
	"time"

	"github.com/nylas/cli/internal/domain"
)

func TestNewEventFormCreate(t *testing.T) {
	app := createTestApp(t)
	calendarID := "test-calendar"

	form := NewEventForm(app, calendarID, nil, nil, nil)

	if form == nil {
		t.Fatal("NewEventForm returned nil")
		return
	}

	if form.mode != EventFormCreate {
		t.Errorf("mode = %v, want EventFormCreate", form.mode)
	}

	if form.calendarID != calendarID {
		t.Errorf("calendarID = %q, want %q", form.calendarID, calendarID)
	}

	// Check default values
	if form.busy != true {
		t.Error("busy should default to true")
	}

	if form.allDay != false {
		t.Error("allDay should default to false")
	}
}

func TestNewEventFormEdit(t *testing.T) {
	app := createTestApp(t)
	calendarID := "test-calendar"

	event := &domain.Event{
		ID:          "event-123",
		Title:       "Test Event",
		Description: "Test Description",
		Location:    "Test Location",
		Busy:        false,
		When: domain.EventWhen{
			Object:    "timespan",
			StartTime: time.Date(2024, 1, 15, 10, 0, 0, 0, time.Local).Unix(),
			EndTime:   time.Date(2024, 1, 15, 11, 0, 0, 0, time.Local).Unix(),
		},
	}

	form := NewEventForm(app, calendarID, event, nil, nil)

	if form == nil {
		t.Fatal("NewEventForm returned nil")
		return
	}

	if form.mode != EventFormEdit {
		t.Errorf("mode = %v, want EventFormEdit", form.mode)
	}

	if form.title != event.Title {
		t.Errorf("title = %q, want %q", form.title, event.Title)
	}

	if form.description != event.Description {
		t.Errorf("description = %q, want %q", form.description, event.Description)
	}

	if form.location != event.Location {
		t.Errorf("location = %q, want %q", form.location, event.Location)
	}

	if form.busy != event.Busy {
		t.Errorf("busy = %v, want %v", form.busy, event.Busy)
	}
}

func TestNewEventFormAllDayEvent(t *testing.T) {
	app := createTestApp(t)
	calendarID := "test-calendar"

	event := &domain.Event{
		ID:    "event-123",
		Title: "All Day Event",
		When: domain.EventWhen{
			Object: "date",
			Date:   "2024-01-15",
		},
	}

	form := NewEventForm(app, calendarID, event, nil, nil)

	if !form.allDay {
		t.Error("allDay should be true for date events")
	}

	if form.startDate != "2024-01-15" {
		t.Errorf("startDate = %q, want %q", form.startDate, "2024-01-15")
	}
}

func TestNewEventFormDatespan(t *testing.T) {
	app := createTestApp(t)
	calendarID := "test-calendar"

	event := &domain.Event{
		ID:    "event-123",
		Title: "Multi-day Event",
		When: domain.EventWhen{
			Object:    "datespan",
			StartDate: "2024-01-15",
			EndDate:   "2024-01-17",
		},
	}

	form := NewEventForm(app, calendarID, event, nil, nil)

	if !form.allDay {
		t.Error("allDay should be true for datespan events")
	}

	if form.startDate != "2024-01-15" {
		t.Errorf("startDate = %q, want %q", form.startDate, "2024-01-15")
	}

	if form.endDate != "2024-01-17" {
		t.Errorf("endDate = %q, want %q", form.endDate, "2024-01-17")
	}
}

func TestEventFormValidation(t *testing.T) {
	app := createTestApp(t)
	calendarID := "test-calendar"

	tests := []struct {
		name      string
		setup     func(f *EventForm)
		wantError bool
		errorMsg  string
	}{
		{
			name: "valid event",
			setup: func(f *EventForm) {
				f.title = "Test Event"
				f.startDate = "2024-01-15"
				f.endDate = "2024-01-15"
				f.startTime = "10:00"
				f.endTime = "11:00"
			},
			wantError: false,
		},
		{
			name: "missing title",
			setup: func(f *EventForm) {
				f.title = ""
				f.startDate = "2024-01-15"
				f.endDate = "2024-01-15"
				f.startTime = "10:00"
				f.endTime = "11:00"
			},
			wantError: true,
			errorMsg:  "Title is required",
		},
		{
			name: "missing start date",
			setup: func(f *EventForm) {
				f.title = "Test Event"
				f.startDate = ""
				f.endDate = "2024-01-15"
				f.startTime = "10:00"
				f.endTime = "11:00"
			},
			wantError: true,
			errorMsg:  "Start date is required",
		},
		{
			name: "invalid date format",
			setup: func(f *EventForm) {
				f.title = "Test Event"
				f.startDate = "15-01-2024" // Wrong format
				f.endDate = "2024-01-15"
				f.startTime = "10:00"
				f.endTime = "11:00"
			},
			wantError: true,
			errorMsg:  "Start date must be YYYY-MM-DD format",
		},
		{
			name: "missing start time for non-all-day",
			setup: func(f *EventForm) {
				f.title = "Test Event"
				f.startDate = "2024-01-15"
				f.endDate = "2024-01-15"
				f.startTime = ""
				f.endTime = "11:00"
				f.allDay = false
			},
			wantError: true,
			errorMsg:  "Start time is required",
		},
		{
			name: "all-day event without times is valid",
			setup: func(f *EventForm) {
				f.title = "All Day Event"
				f.startDate = "2024-01-15"
				f.endDate = "2024-01-15"
				f.startTime = ""
				f.endTime = ""
				f.allDay = true
			},
			wantError: false,
		},
		{
			name: "invalid time format",
			setup: func(f *EventForm) {
				f.title = "Test Event"
				f.startDate = "2024-01-15"
				f.endDate = "2024-01-15"
				f.startTime = "10:00 AM" // Wrong format
				f.endTime = "11:00"
				f.allDay = false
			},
			wantError: true,
			errorMsg:  "Start time must be HH:MM format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			form := NewEventForm(app, calendarID, nil, nil, nil)
			tt.setup(form)

			errors := form.validate()

			if tt.wantError {
				if len(errors) == 0 {
					t.Error("expected validation error, got none")
				} else {
					found := false
					for _, err := range errors {
						if err == tt.errorMsg || containsSubstring(err, tt.errorMsg) {
							found = true
							break
						}
					}
					if !found {
						t.Errorf("expected error containing %q, got %v", tt.errorMsg, errors)
					}
				}
			} else {
				if len(errors) > 0 {
					t.Errorf("unexpected validation errors: %v", errors)
				}
			}
		})
	}
}

func TestEventFormMode(t *testing.T) {
	if EventFormCreate != 0 {
		t.Errorf("EventFormCreate = %d, want 0", EventFormCreate)
	}
	if EventFormEdit != 1 {
		t.Errorf("EventFormEdit = %d, want 1", EventFormEdit)
	}
}

func containsSubstring(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstring(s[1:], substr) || (len(s) >= len(substr) && s[:len(substr)] == substr))
}
