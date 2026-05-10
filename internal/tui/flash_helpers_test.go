package tui

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/nylas/cli/internal/domain"
)

func TestFlashLoadErrorDoesNotDeadlockInsideQueuedUpdate(t *testing.T) {
	app := createTestApp(t)
	screen := tcell.NewSimulationScreen("")
	screen.SetSize(80, 24)
	app.SetScreen(screen)

	runErr := make(chan error, 1)
	go func() {
		runErr <- app.Run()
	}()
	t.Cleanup(func() {
		app.Stop()
		select {
		case err := <-runErr:
			if err != nil {
				t.Errorf("app.Run() returned error: %v", err)
			}
		case <-time.After(time.Second):
			t.Log("app did not stop before cleanup timeout")
		}
	})

	ready := make(chan struct{})
	go func() {
		app.QueueUpdateDraw(func() {
			close(ready)
		})
	}()
	select {
	case <-ready:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for TUI event loop")
	}

	done := make(chan struct{})
	go func() {
		app.QueueUpdateDraw(func() {
			app.FlashLoadError("Failed to load drafts", errors.New("boom"))
		})
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("FlashLoadError deadlocked when called during an existing queued update")
	}
}

func TestFlashLoadErrorBeforeRunIsVisibleAfterStart(t *testing.T) {
	app := createTestApp(t)
	app.FlashLoadError("Failed to load threads", &domain.APIError{
		StatusCode: 403,
		Type:       "insufficient_scopes",
		Message:    "Grant lacks gmail.readonly",
	})

	screen := tcell.NewSimulationScreen("")
	screen.SetSize(80, 24)
	app.SetScreen(screen)

	runErr := make(chan error, 1)
	go func() {
		runErr <- app.Run()
	}()
	t.Cleanup(func() {
		app.Stop()
		select {
		case err := <-runErr:
			if err != nil {
				t.Errorf("app.Run() returned error: %v", err)
			}
		case <-time.After(time.Second):
			t.Log("app did not stop before cleanup timeout")
		}
	})

	deadline := time.After(time.Second)
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-deadline:
			t.Fatal("expected initial load error to be flashed")
			return
		case <-ticker.C:
			text := readStatusText(t, app)
			if strings.Contains(text, "Failed to load threads") &&
				strings.Contains(text, "Grant lacks gmail.readonly") &&
				strings.Contains(text, "insufficient_scopes") {
				return
			}
		}
	}
}

func readStatusText(t *testing.T, app *App) string {
	t.Helper()

	var statusText string
	done := make(chan struct{})
	go func() {
		app.QueueUpdateDraw(func() {
			statusText = app.status.GetText(true)
			close(done)
		})
	}()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for status read")
	}
	return statusText
}
