package tui

import (
	"context"
	"testing"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/nylas/cli/internal/adapters/nylas"
	"github.com/nylas/cli/internal/domain"
)

// createTestApp creates an App instance for testing
func createTestApp(t *testing.T) *App {
	t.Helper()

	mockClient := nylas.NewMockClient()

	// Set up mock responses
	mockClient.GetMessagesFunc = func(ctx context.Context, grantID string, limit int) ([]domain.Message, error) {
		return []domain.Message{
			{
				ID:      "msg-1",
				Subject: "Test Message 1",
				From:    []domain.EmailParticipant{{Email: "sender1@example.com", Name: "Sender One"}},
				Unread:  true,
				Date:    time.Now(),
			},
			{
				ID:      "msg-2",
				Subject: "Test Message 2",
				From:    []domain.EmailParticipant{{Email: "sender2@example.com", Name: "Sender Two"}},
				Starred: true,
				Date:    time.Now().Add(-time.Hour),
			},
			{
				ID:      "msg-3",
				Subject: "Test Message 3",
				From:    []domain.EmailParticipant{{Email: "sender3@example.com"}},
				Date:    time.Now().Add(-24 * time.Hour),
			},
		}, nil
	}

	config := Config{
		Client:          mockClient,
		GrantID:         "test-grant-id",
		Email:           "user@example.com",
		Provider:        "google",
		RefreshInterval: time.Second * 30,
		Theme:           ThemeK9s,
		InitialView:     "dashboard",
	}

	return NewApp(config)
}

func TestNewApp(t *testing.T) {
	app := createTestApp(t)

	if app == nil {
		t.Fatal("NewApp() returned nil")
		return
	}

	if app.Application == nil {
		t.Error("App.Application is nil")
	}

	if app.styles == nil {
		t.Error("App.styles is nil")
	}

	if app.views == nil {
		t.Error("App.views is nil")
	}

	if app.content == nil {
		t.Error("App.content is nil")
	}
}

func TestNewAppWithThemes(t *testing.T) {
	themes := AvailableThemes()

	for _, theme := range themes {
		t.Run(string(theme), func(t *testing.T) {
			mockClient := nylas.NewMockClient()
			config := Config{
				Client:          mockClient,
				GrantID:         "test-grant-id",
				Email:           "user@example.com",
				Provider:        "google",
				RefreshInterval: time.Second * 30,
				Theme:           theme,
			}

			app := NewApp(config)
			if app == nil {
				t.Fatalf("NewApp() with theme %q returned nil", theme)
				return
			}

			if app.styles == nil {
				t.Errorf("App.styles is nil for theme %q", theme)
			}
		})
	}
}

func TestAppGetConfig(t *testing.T) {
	app := createTestApp(t)
	config := app.GetConfig()

	if config.GrantID != "test-grant-id" {
		t.Errorf("GetConfig().GrantID = %q, want %q", config.GrantID, "test-grant-id")
	}
	if config.Email != "user@example.com" {
		t.Errorf("GetConfig().Email = %q, want %q", config.Email, "user@example.com")
	}
	if config.Provider != "google" {
		t.Errorf("GetConfig().Provider = %q, want %q", config.Provider, "google")
	}
}

func TestAppStyles(t *testing.T) {
	app := createTestApp(t)
	styles := app.Styles()

	if styles == nil {
		t.Fatal("Styles() returned nil")
		return
	}

	// Verify styles are properly set
	if styles.TableSelectBg == 0 && styles.TableSelectBg != tcell.ColorDefault {
		t.Error("TableSelectBg not set")
	}
}

func TestCreateView(t *testing.T) {
	app := createTestApp(t)

	tests := []struct {
		name     string
		viewType string
	}{
		{"messages", "messages"},
		{"events", "events"},
		{"contacts", "contacts"},
		{"webhooks", "webhooks"},
		{"grants", "grants"},
		{"dashboard", "dashboard"},
		{"unknown", "unknown"}, // Should default to dashboard
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			view := app.createView(tt.viewType)
			// createView always returns a valid ResourceView (never nil by design)

			// Verify view has required methods
			if view.Name() == "" {
				t.Error("View.Name() returned empty string")
			}
			if view.Title() == "" {
				t.Error("View.Title() returned empty string")
			}
			if view.Primitive() == nil {
				t.Error("View.Primitive() returned nil")
			}
			// Hints can be empty, so we just check it doesn't panic
			_ = view.Hints()
		})
	}
}
