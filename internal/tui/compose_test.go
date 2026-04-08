package tui

import (
	"testing"
	"time"

	"github.com/nylas/cli/internal/domain"
)

func TestNewComposeView(t *testing.T) {
	app := createTestApp(t)

	view := NewComposeView(app, ComposeModeNew, nil)

	if view == nil {
		t.Fatal("NewComposeView returned nil")
		return
	}

	if view.mode != ComposeModeNew {
		t.Errorf("mode = %v, want ComposeModeNew", view.mode)
	}

	if view.app == nil {
		t.Error("app should not be nil")
	}

	if view.form == nil {
		t.Error("form should not be nil")
	}

	if view.draft != nil {
		t.Error("draft should be nil for new compose")
	}
}

func TestNewComposeViewForDraft(t *testing.T) {
	app := createTestApp(t)

	draft := &domain.Draft{
		ID:      "draft-123",
		Subject: "Test Subject",
		Body:    "Test body content",
		To: []domain.EmailParticipant{
			{Email: "test@example.com", Name: "Test User"},
		},
		Cc: []domain.EmailParticipant{
			{Email: "cc@example.com", Name: "CC User"},
		},
		UpdatedAt: time.Now(),
	}

	view := NewComposeViewForDraft(app, draft)

	if view == nil {
		t.Fatal("NewComposeViewForDraft returned nil")
		return
	}

	if view.mode != ComposeModeDraft {
		t.Errorf("mode = %v, want ComposeModeDraft", view.mode)
	}

	if view.draft == nil {
		t.Error("draft should not be nil")
	}

	if view.draft.ID != "draft-123" {
		t.Errorf("draft.ID = %q, want %q", view.draft.ID, "draft-123")
	}
}

func TestComposeViewReplyMode(t *testing.T) {
	app := createTestApp(t)

	msg := &domain.Message{
		ID:      "msg-123",
		Subject: "Original Subject",
		From: []domain.EmailParticipant{
			{Email: "sender@example.com", Name: "Sender"},
		},
		To: []domain.EmailParticipant{
			{Email: "me@example.com", Name: "Me"},
		},
	}

	view := NewComposeView(app, ComposeModeReply, msg)

	if view == nil {
		t.Fatal("NewComposeView returned nil for reply mode")
		return
	}

	if view.mode != ComposeModeReply {
		t.Errorf("mode = %v, want ComposeModeReply", view.mode)
	}

	if view.replyToMsg == nil {
		t.Error("replyToMsg should not be nil")
	}
}

func TestComposeViewReplyAllMode(t *testing.T) {
	app := createTestApp(t)

	msg := &domain.Message{
		ID:      "msg-123",
		Subject: "Original Subject",
		From: []domain.EmailParticipant{
			{Email: "sender@example.com", Name: "Sender"},
		},
		To: []domain.EmailParticipant{
			{Email: "me@example.com", Name: "Me"},
		},
		Cc: []domain.EmailParticipant{
			{Email: "other@example.com", Name: "Other"},
		},
	}

	view := NewComposeView(app, ComposeModeReplyAll, msg)

	if view == nil {
		t.Fatal("NewComposeView returned nil for reply all mode")
		return
	}

	if view.mode != ComposeModeReplyAll {
		t.Errorf("mode = %v, want ComposeModeReplyAll", view.mode)
	}
}

func TestComposeViewForwardMode(t *testing.T) {
	app := createTestApp(t)

	msg := &domain.Message{
		ID:      "msg-123",
		Subject: "Original Subject",
		Body:    "Original body content",
	}

	view := NewComposeView(app, ComposeModeForward, msg)

	if view == nil {
		t.Fatal("NewComposeView returned nil for forward mode")
		return
	}

	if view.mode != ComposeModeForward {
		t.Errorf("mode = %v, want ComposeModeForward", view.mode)
	}
}

func TestComposeViewSetOnSent(t *testing.T) {
	app := createTestApp(t)

	view := NewComposeView(app, ComposeModeNew, nil)

	called := false
	view.SetOnSent(func() {
		called = true
	})

	if view.onSent == nil {
		t.Error("onSent should be set")
	}

	// Call the handler
	view.onSent()

	if !called {
		t.Error("onSent callback was not called")
	}
}

func TestComposeViewSetOnCancel(t *testing.T) {
	app := createTestApp(t)

	view := NewComposeView(app, ComposeModeNew, nil)

	called := false
	view.SetOnCancel(func() {
		called = true
	})

	if view.onCancel == nil {
		t.Error("onCancel should be set")
	}

	// Call the handler
	view.onCancel()

	if !called {
		t.Error("onCancel callback was not called")
	}
}

func TestComposeViewSetOnSave(t *testing.T) {
	app := createTestApp(t)

	draft := &domain.Draft{
		ID:      "draft-123",
		Subject: "Test",
	}

	view := NewComposeViewForDraft(app, draft)

	called := false
	view.SetOnSave(func() {
		called = true
	})

	if view.onSave == nil {
		t.Error("onSave should be set")
	}

	// Call the handler
	view.onSave()

	if !called {
		t.Error("onSave callback was not called")
	}
}

func TestComposeModeDraftValue(t *testing.T) {
	// Verify ComposeModeDraft has expected value in the enum
	if ComposeModeDraft != 4 {
		t.Errorf("ComposeModeDraft = %v, want 4", ComposeModeDraft)
	}
}

func TestComposeModeValues(t *testing.T) {
	tests := []struct {
		mode     ComposeMode
		expected int
	}{
		{ComposeModeNew, 0},
		{ComposeModeReply, 1},
		{ComposeModeReplyAll, 2},
		{ComposeModeForward, 3},
		{ComposeModeDraft, 4},
	}

	for _, tt := range tests {
		if int(tt.mode) != tt.expected {
			t.Errorf("mode %v = %d, want %d", tt.mode, int(tt.mode), tt.expected)
		}
	}
}
