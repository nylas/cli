package tui

import (
	"testing"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/nylas/cli/internal/domain"
)

func TestNewDraftsView(t *testing.T) {
	app := createTestApp(t)

	view := NewDraftsView(app)

	if view == nil {
		t.Fatal("NewDraftsView returned nil")
		return
	}

	if view.BaseTableView == nil {
		t.Error("BaseTableView should not be nil")
	}

	if view.drafts != nil {
		t.Error("drafts should be nil initially")
	}

	if view.showingDraft {
		t.Error("showingDraft should be false initially")
	}

	if view.currentDraft != nil {
		t.Error("currentDraft should be nil initially")
	}
}

func TestDraftsViewHints(t *testing.T) {
	app := createTestApp(t)

	view := NewDraftsView(app)

	hints := view.hints
	if len(hints) == 0 {
		t.Fatal("hints should not be empty")
	}

	// Check expected hints
	expectedHints := map[string]string{
		"enter": "edit",
		"n":     "new",
		"s":     "send",
		"d":     "delete",
		"r":     "refresh",
	}

	for _, hint := range hints {
		if expected, ok := expectedHints[hint.Key]; ok {
			if hint.Desc != expected {
				t.Errorf("hint %q has desc %q, want %q", hint.Key, hint.Desc, expected)
			}
		}
	}
}

func TestDraftsViewRender(t *testing.T) {
	app := createTestApp(t)

	view := NewDraftsView(app)

	// Set up test drafts
	now := time.Now()
	view.drafts = []domain.Draft{
		{
			ID:        "draft-1",
			Subject:   "Test Subject 1",
			To:        []domain.EmailParticipant{{Email: "test@example.com", Name: "Test User"}},
			UpdatedAt: now,
		},
		{
			ID:        "draft-2",
			Subject:   "Test Subject 2",
			To:        []domain.EmailParticipant{{Email: "another@example.com"}},
			UpdatedAt: now.Add(-1 * time.Hour),
		},
		{
			ID:        "draft-3",
			Subject:   "",  // No subject
			To:        nil, // No recipient
			UpdatedAt: now.Add(-2 * time.Hour),
		},
	}

	view.render()

	// Check that table has data
	rowCount := view.table.GetRowCount()
	if rowCount != 3 {
		t.Errorf("table has %d rows, want 3", rowCount)
	}
}

func TestDraftsViewFilter(t *testing.T) {
	app := createTestApp(t)

	view := NewDraftsView(app)

	now := time.Now()
	view.drafts = []domain.Draft{
		{
			ID:        "draft-1",
			Subject:   "Important Meeting",
			To:        []domain.EmailParticipant{{Email: "boss@example.com", Name: "Boss"}},
			UpdatedAt: now,
		},
		{
			ID:        "draft-2",
			Subject:   "Casual Chat",
			To:        []domain.EmailParticipant{{Email: "friend@example.com", Name: "Friend"}},
			UpdatedAt: now,
		},
	}

	// Filter by subject
	view.filter = "important"
	view.render()

	rowCount := view.table.GetRowCount()
	if rowCount != 1 {
		t.Errorf("after subject filter, table has %d rows, want 1", rowCount)
	}

	// Filter by email
	view.filter = "friend@"
	view.render()

	rowCount = view.table.GetRowCount()
	if rowCount != 1 {
		t.Errorf("after email filter, table has %d rows, want 1", rowCount)
	}

	// No matches
	view.filter = "nonexistent"
	view.render()

	rowCount = view.table.GetRowCount()
	if rowCount != 0 {
		t.Errorf("after no-match filter, table has %d rows, want 0", rowCount)
	}
}

func TestDraftsViewHandleKeyEscape(t *testing.T) {
	app := createTestApp(t)

	view := NewDraftsView(app)
	view.showingDraft = true

	event := tcell.NewEventKey(tcell.KeyEscape, 0, tcell.ModNone)
	result := view.HandleKey(event)

	if result != nil {
		t.Error("HandleKey should return nil for Escape when showing draft")
	}

	if view.showingDraft {
		t.Error("showingDraft should be false after Escape")
	}

	// When not showing draft, Escape should propagate
	event = tcell.NewEventKey(tcell.KeyEscape, 0, tcell.ModNone)
	result = view.HandleKey(event)

	if result == nil {
		t.Error("HandleKey should return event when not showing draft")
	}
}

func TestDraftsViewHandleKeyNew(t *testing.T) {
	app := createTestApp(t)

	view := NewDraftsView(app)

	event := tcell.NewEventKey(tcell.KeyRune, 'n', tcell.ModNone)
	result := view.HandleKey(event)

	if result != nil {
		t.Error("HandleKey should return nil for 'n' (new draft)")
	}
}

func TestGetDraftRecipients(t *testing.T) {
	tests := []struct {
		name     string
		draft    *domain.Draft
		expected string
	}{
		{
			name:     "no recipients",
			draft:    &domain.Draft{To: nil},
			expected: "(no recipient)",
		},
		{
			name: "one recipient with name",
			draft: &domain.Draft{
				To: []domain.EmailParticipant{
					{Email: "test@example.com", Name: "Test User"},
				},
			},
			expected: "Test User",
		},
		{
			name: "one recipient without name",
			draft: &domain.Draft{
				To: []domain.EmailParticipant{
					{Email: "test@example.com"},
				},
			},
			expected: "test@example.com",
		},
		{
			name: "multiple recipients",
			draft: &domain.Draft{
				To: []domain.EmailParticipant{
					{Email: "test1@example.com", Name: "User One"},
					{Email: "test2@example.com", Name: "User Two"},
				},
			},
			expected: "User One, User Two",
		},
		{
			name: "mixed recipients",
			draft: &domain.Draft{
				To: []domain.EmailParticipant{
					{Email: "test1@example.com", Name: "User One"},
					{Email: "test2@example.com"}, // No name
				},
			},
			expected: "User One, test2@example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getDraftRecipients(tt.draft)
			if result != tt.expected {
				t.Errorf("getDraftRecipients() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestDraftsViewTableSetup(t *testing.T) {
	app := createTestApp(t)

	view := NewDraftsView(app)

	// Verify table exists
	if view.table == nil {
		t.Error("table should not be nil")
	}
}

func TestDraftsViewEmptyState(t *testing.T) {
	app := createTestApp(t)

	view := NewDraftsView(app)

	// Empty drafts
	view.drafts = []domain.Draft{}
	view.render()

	rowCount := view.table.GetRowCount()
	if rowCount != 0 {
		t.Errorf("empty drafts should have 0 rows, got %d", rowCount)
	}
}

func TestDraftsViewNoSubjectDisplay(t *testing.T) {
	app := createTestApp(t)

	view := NewDraftsView(app)

	view.drafts = []domain.Draft{
		{
			ID:        "draft-1",
			Subject:   "", // Empty subject
			To:        []domain.EmailParticipant{{Email: "test@example.com"}},
			UpdatedAt: time.Now(),
		},
	}

	view.render()

	// The table should have 1 row
	rowCount := view.table.GetRowCount()
	if rowCount != 1 {
		t.Errorf("table should have 1 row, got %d", rowCount)
	}
}

func TestDraftsViewNoRecipientDisplay(t *testing.T) {
	app := createTestApp(t)

	view := NewDraftsView(app)

	view.drafts = []domain.Draft{
		{
			ID:        "draft-1",
			Subject:   "Test Subject",
			To:        nil, // No recipient
			UpdatedAt: time.Now(),
		},
	}

	view.render()

	// The table should have 1 row
	rowCount := view.table.GetRowCount()
	if rowCount != 1 {
		t.Errorf("table should have 1 row, got %d", rowCount)
	}
}
