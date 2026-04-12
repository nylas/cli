package tui

import (
	"testing"
	"time"
)

func TestNewAvailabilityView(t *testing.T) {
	app := createTestApp(t)

	view := NewAvailabilityView(app)

	if view.name != "availability" {
		t.Errorf("name = %q, want %q", view.name, "availability")
	}

	if view.title != "Availability" {
		t.Errorf("title = %q, want %q", view.title, "Availability")
	}

	if view.duration != 30 {
		t.Errorf("duration = %d, want %d", view.duration, 30)
	}

	if view.layout == nil {
		t.Error("layout should not be nil")
	}

	if view.participantsList == nil {
		t.Error("participantsList should not be nil")
	}

	if view.slotsList == nil {
		t.Error("slotsList should not be nil")
	}

	if view.timeline == nil {
		t.Error("timeline should not be nil")
	}

	if view.infoPanel == nil {
		t.Error("infoPanel should not be nil")
	}
}

func TestAvailabilityView_Name(t *testing.T) {
	app := createTestApp(t)
	view := NewAvailabilityView(app)

	if view.Name() != "availability" {
		t.Errorf("Name() = %q, want %q", view.Name(), "availability")
	}
}

func TestAvailabilityView_Title(t *testing.T) {
	app := createTestApp(t)
	view := NewAvailabilityView(app)

	if view.Title() != "Availability" {
		t.Errorf("Title() = %q, want %q", view.Title(), "Availability")
	}
}

func TestAvailabilityView_Primitive(t *testing.T) {
	app := createTestApp(t)
	view := NewAvailabilityView(app)

	// Verify Primitive returns a valid tview.Primitive (not nil by design)
	p := view.Primitive()
	if p.HasFocus() {
		// Just verifying we can call methods on the primitive
		_ = p
	}
}

func TestAvailabilityView_Hints(t *testing.T) {
	app := createTestApp(t)
	view := NewAvailabilityView(app)

	hints := view.Hints()

	if len(hints) == 0 {
		t.Error("Hints() should return hints")
	}

	// Check for expected hints
	hintMap := make(map[string]bool)
	for _, h := range hints {
		hintMap[h.Key] = true
	}

	expectedKeys := []string{"a", "d", "enter", "r", "Tab", "D", "S"}
	for _, key := range expectedKeys {
		if !hintMap[key] {
			t.Errorf("Hints() missing key %q", key)
		}
	}
}

func TestAvailabilityView_DefaultDates(t *testing.T) {
	app := createTestApp(t)
	view := NewAvailabilityView(app)

	now := time.Now()

	// Start date should be today
	if view.startDate.Year() != now.Year() || view.startDate.YearDay() != now.YearDay() {
		t.Error("startDate should be today")
	}

	// End date should be 7 days from now
	expectedEnd := now.AddDate(0, 0, 7)
	if view.endDate.Year() != expectedEnd.Year() || view.endDate.YearDay() != expectedEnd.YearDay() {
		t.Error("endDate should be 7 days from now")
	}
}

func TestAvailabilityView_Filter(t *testing.T) {
	app := createTestApp(t)
	view := NewAvailabilityView(app)

	// Filter currently doesn't do anything for availability view
	// but the method should exist and not panic
	view.Filter("test@example.com")
}
