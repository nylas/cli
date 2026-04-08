package tui

import (
	"testing"

	"github.com/gdamore/tcell/v2"
	"github.com/nylas/cli/internal/domain"
)

func TestCommandRegistry(t *testing.T) {
	registry := NewCommandRegistry()

	if registry == nil {
		t.Fatal("NewCommandRegistry() returned nil")
	}

	// Get all commands
	commands := registry.GetAll()
	if len(commands) == 0 {
		t.Error("GetAll() returned empty list")
	}

	// Get specific command
	cmd := registry.Get("q")
	if cmd == nil {
		t.Error("Get('q') should find quit command")
	} else if cmd.Name != "quit" {
		t.Errorf("Get('q').Name = %q, want %q", cmd.Name, "quit")
	}

	// Get non-existent command
	cmd = registry.Get("nonexistent")
	if cmd != nil {
		t.Error("Get('nonexistent') should return nil")
	}

	// Test Search
	results := registry.Search("quit")
	if len(results) == 0 {
		t.Error("Search('quit') returned empty list")
	}
}

func TestStyles_DefaultStyles(t *testing.T) {
	styles := DefaultStyles()

	if styles == nil {
		t.Fatal("DefaultStyles() returned nil")
		return
	}

	// Verify some key colors are set
	if styles.BgColor == 0 {
		t.Error("BgColor not set")
	}
	if styles.FgColor == 0 {
		t.Error("FgColor not set")
	}
}

func TestCalendarView_Focus(t *testing.T) {
	app := createTestApp(t)
	view := NewCalendarView(app)

	// Focus should not panic
	view.Focus(nil)

	// HasFocus may return true or false depending on initialization
	// Just verify it doesn't panic
	_ = view.HasFocus()
}

func TestCalendarView_SetOnEventSelect(t *testing.T) {
	app := createTestApp(t)
	view := NewCalendarView(app)

	view.SetOnEventSelect(func(event *domain.Event) {
		// callback set
	})

	if view.onEventSelect == nil {
		t.Error("SetOnEventSelect did not set callback")
	}
}

func TestCalendarView_InputHandler(t *testing.T) {
	app := createTestApp(t)
	view := NewCalendarView(app)

	// Test various keys
	keys := []struct {
		key  tcell.Key
		rune rune
		desc string
	}{
		{tcell.KeyLeft, 0, "left arrow"},
		{tcell.KeyRight, 0, "right arrow"},
		{tcell.KeyUp, 0, "up arrow"},
		{tcell.KeyDown, 0, "down arrow"},
		{tcell.KeyRune, 'h', "h key"},
		{tcell.KeyRune, 'l', "l key"},
		{tcell.KeyRune, 'j', "j key"},
		{tcell.KeyRune, 'k', "k key"},
		{tcell.KeyRune, 'H', "H key (prev month)"},
		{tcell.KeyRune, 'L', "L key (next month)"},
		{tcell.KeyRune, 't', "t key (today)"},
		{tcell.KeyRune, 'v', "v key (toggle view)"},
		{tcell.KeyRune, 'm', "m key (month view)"},
		{tcell.KeyRune, 'w', "w key (week view)"},
		{tcell.KeyRune, 'a', "a key (agenda view)"},
		{tcell.KeyRune, ']', "] key (next calendar)"},
		{tcell.KeyRune, '[', "[ key (prev calendar)"},
		{tcell.KeyEnter, 0, "enter key"},
	}

	handler := view.InputHandler()
	if handler == nil {
		t.Fatal("InputHandler() returned nil")
	}

	for _, k := range keys {
		t.Run(k.desc, func(t *testing.T) {
			event := tcell.NewEventKey(k.key, k.rune, tcell.ModNone)
			// Should not panic
			handler(event, nil)
		})
	}
}

func TestDashboardView(t *testing.T) {
	app := createTestApp(t)
	view := NewDashboardView(app)

	if view == nil {
		t.Fatal("NewDashboardView() returned nil")
	}

	if view.Name() != "dashboard" {
		t.Errorf("Name() = %q, want %q", view.Name(), "dashboard")
	}

	if view.Title() != "Dashboard" {
		t.Errorf("Title() = %q, want %q", view.Title(), "Dashboard")
	}

	// Load should not panic
	view.Load()

	// Filter should not panic
	view.Filter("test")

	// Refresh should not panic
	view.Refresh()
}

func TestContactsView(t *testing.T) {
	app := createTestApp(t)
	view := NewContactsView(app)

	if view == nil {
		t.Fatal("NewContactsView() returned nil")
		return
	}

	if view.Name() != "contacts" {
		t.Errorf("Name() = %q, want %q", view.Name(), "contacts")
	}

	// Test keys
	keys := []struct {
		key  tcell.Key
		rune rune
	}{
		{tcell.KeyRune, 'n'}, // New contact
		{tcell.KeyRune, 'e'}, // Edit
		{tcell.KeyRune, 'd'}, // Delete
		{tcell.KeyEnter, 0},  // View
		{tcell.KeyRune, 'r'}, // Refresh
	}

	for _, k := range keys {
		event := tcell.NewEventKey(k.key, k.rune, tcell.ModNone)
		// Should not panic
		_ = view.HandleKey(event)
	}
}

func TestWebhooksView(t *testing.T) {
	app := createTestApp(t)
	view := NewWebhooksView(app)

	if view == nil {
		t.Fatal("NewWebhooksView() returned nil")
		return
	}

	if view.Name() != "webhooks" {
		t.Errorf("Name() = %q, want %q", view.Name(), "webhooks")
	}

	// Test keys
	keys := []struct {
		key  tcell.Key
		rune rune
	}{
		{tcell.KeyRune, 'n'}, // New webhook
		{tcell.KeyRune, 'e'}, // Edit
		{tcell.KeyRune, 'd'}, // Delete
		{tcell.KeyEnter, 0},  // View
		{tcell.KeyRune, 'r'}, // Refresh
	}

	for _, k := range keys {
		event := tcell.NewEventKey(k.key, k.rune, tcell.ModNone)
		_ = view.HandleKey(event)
	}
}

func TestAvailabilityView_FullInterface(t *testing.T) {
	app := createTestApp(t)
	view := NewAvailabilityView(app)

	// Test ResourceView interface
	if view.Name() != "availability" {
		t.Errorf("Name() = %q, want %q", view.Name(), "availability")
	}

	if view.Title() != "Availability" {
		t.Errorf("Title() = %q, want %q", view.Title(), "Availability")
	}

	// Verify Primitive returns a valid tview.Primitive (not nil by design)
	_ = view.Primitive()

	hints := view.Hints()
	if len(hints) == 0 {
		t.Error("Hints() returned empty")
	}

	// Filter should not panic
	view.Filter("test")
}

func TestDraftsView(t *testing.T) {
	app := createTestApp(t)
	view := NewDraftsView(app)

	if view == nil {
		t.Fatal("NewDraftsView() returned nil")
		return
	}

	if view.Name() != "drafts" {
		t.Errorf("Name() = %q, want %q", view.Name(), "drafts")
	}

	// Load should not panic
	view.Load()

	// Test keys
	keys := []struct {
		key  tcell.Key
		rune rune
	}{
		{tcell.KeyRune, 'n'}, // New draft
		{tcell.KeyRune, 'e'}, // Edit
		{tcell.KeyRune, 'd'}, // Delete
		{tcell.KeyEnter, 0},  // View/Edit
		{tcell.KeyRune, 'r'}, // Refresh
	}

	for _, k := range keys {
		event := tcell.NewEventKey(k.key, k.rune, tcell.ModNone)
		_ = view.HandleKey(event)
	}
}
