package tui

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/nylas/cli/internal/adapters/nylas"
	"github.com/nylas/cli/internal/domain"
	"github.com/nylas/cli/internal/ports"
)

// startTestAppRunning starts the app's event loop on a simulation screen and
// registers cleanup. Tests use it to exercise code paths that marshal work
// through QueueUpdateDraw.
func startTestAppRunning(t *testing.T, app *App) {
	t.Helper()

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

	// Wait for the event loop to start processing updates.
	ready := make(chan struct{})
	go func() {
		app.QueueUpdateDraw(func() { close(ready) })
	}()
	select {
	case <-ready:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for TUI event loop")
	}
}

// runOnEventLoop runs fn on the event loop and waits for it to complete.
func runOnEventLoop(t *testing.T, app *App, fn func()) {
	t.Helper()

	done := make(chan struct{})
	go func() {
		app.QueueUpdateDraw(func() {
			fn()
			close(done)
		})
	}()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for event loop update")
	}
}

// TestFlashFromBackgroundGoroutines verifies App.Flash is safe to call bare
// from background goroutines: the status mutation must be marshaled onto the
// event loop instead of racing the status ticker and input handlers.
func TestFlashFromBackgroundGoroutines(t *testing.T) {
	app := createTestApp(t)
	startTestAppRunning(t, app)

	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			app.Flash(FlashInfo, "background flash %d", n)
		}(i)
	}
	wg.Wait()

	deadline := time.After(time.Second)
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-deadline:
			t.Fatal("expected background flash to be visible in status bar")
		case <-ticker.C:
			if strings.Contains(readStatusText(t, app), "background flash") {
				return
			}
		}
	}
}

// TestFlashFromEventLoopDoesNotDeadlock guards the QueueUpdateDraw
// re-entrancy hazard: tview's QueueUpdate blocks until the event loop runs
// the update, so Flash must not call it synchronously from event-loop
// callbacks (input handlers, queued updates).
func TestFlashFromEventLoopDoesNotDeadlock(t *testing.T) {
	app := createTestApp(t)
	startTestAppRunning(t, app)

	runOnEventLoop(t, app, func() {
		app.Flash(FlashWarn, "flash from event loop")
	})
}

// TestWebhookServerViewRecordEventConcurrent verifies webhook events arriving
// on server goroutines are applied on the event loop: concurrent recordEvent
// calls must not race the renderers and the buffer must stay capped at
// maxEvents.
func TestWebhookServerViewRecordEventConcurrent(t *testing.T) {
	app := createTestApp(t)
	startTestAppRunning(t, app)

	view := NewWebhookServerView(app)

	total := view.maxEvents + 10 // exceed maxEvents to exercise truncation
	var wg sync.WaitGroup
	for i := 0; i < total; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			view.recordEvent(&ports.WebhookEvent{
				Type:       fmt.Sprintf("message.created.%d", n),
				ReceivedAt: time.Now(),
			})
		}(i)
	}
	wg.Wait()

	var got int
	runOnEventLoop(t, app, func() { got = len(view.events) })
	if got != view.maxEvents {
		t.Fatalf("events length = %d, want %d (capped at maxEvents)", got, view.maxEvents)
	}
}

// TestMessagesViewLoadSnapshotsGrantID verifies the fetch/apply split
// snapshots app.config.GrantID on the event loop before spawning the fetch
// goroutine. A regression that reads v.app.config.GrantID inside the
// goroutine would race a concurrent grant switch (caught by -race) and could
// fetch with the wrong grant (caught by the assertion below).
func TestMessagesViewLoadSnapshotsGrantID(t *testing.T) {
	app := createTestApp(t)
	mock, ok := app.config.Client.(*nylas.MockClient)
	if !ok {
		t.Fatal("test app client is not a MockClient")
	}

	gotGrant := make(chan string, 1)
	mock.GetThreadsFunc = func(ctx context.Context, grantID string, params *domain.ThreadQueryParams) ([]domain.Thread, error) {
		gotGrant <- grantID
		return nil, nil
	}

	view := NewMessagesView(app)
	view.folderPanel.folders = []domain.Folder{{ID: "INBOX", Name: "Inbox"}}

	startTestAppRunning(t, app)

	// Call Load and switch the grant in the SAME event-loop callback: the
	// snapshot is taken synchronously inside Load, so the fetch must still
	// use grant-A even though the config changed before the goroutine ran.
	runOnEventLoop(t, app, func() {
		app.config.GrantID = "grant-A"
		view.Load()
		app.config.GrantID = "grant-B"
	})

	select {
	case grantID := <-gotGrant:
		if grantID != "grant-A" {
			t.Fatalf("fetch used grant %q, want snapshot %q taken when Load was called", grantID, "grant-A")
		}
	case <-time.After(time.Second):
		t.Fatal("GetThreads was never called")
	}
}

// TestMessagesViewLoadDropsStaleGrantData verifies the apply callback drops
// in-flight results after a grant switch: if the user switches grants while a
// fetch is in flight, the old grant's data must never be written into the
// view (cross-account data leak).
func TestMessagesViewLoadDropsStaleGrantData(t *testing.T) {
	app := createTestApp(t)
	mock, ok := app.config.Client.(*nylas.MockClient)
	if !ok {
		t.Fatal("test app client is not a MockClient")
	}

	release := make(chan struct{})
	returned := make(chan struct{})
	mock.GetThreadsFunc = func(ctx context.Context, grantID string, params *domain.ThreadQueryParams) ([]domain.Thread, error) {
		<-release // hold the fetch in flight until the grant switch happened
		defer close(returned)
		return []domain.Thread{{ID: "thread-A", Subject: "Grant A secret"}}, nil
	}

	view := NewMessagesView(app)
	view.folderPanel.folders = []domain.Folder{{ID: "INBOX", Name: "Inbox"}}

	startTestAppRunning(t, app)

	// Start a load for grant-A.
	runOnEventLoop(t, app, func() {
		app.config.GrantID = "grant-A"
		view.Load()
	})

	// Switch to grant-B while the grant-A fetch is still in flight, then let
	// the fetch complete.
	runOnEventLoop(t, app, func() { app.config.GrantID = "grant-B" })
	close(release)
	select {
	case <-returned:
	case <-time.After(time.Second):
		t.Fatal("GetThreads never returned")
	}

	// The apply callback is queued right after GetThreads returns. Poll a few
	// event-loop turns and assert grant-A's data is never rendered.
	deadline := time.After(300 * time.Millisecond)
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	for {
		var threadCount, rowCount int
		runOnEventLoop(t, app, func() {
			threadCount = len(view.threads)
			rowCount = view.table.GetRowCount()
		})
		if threadCount != 0 {
			t.Fatalf("view shows %d threads from grant-A after switching to grant-B; stale apply must be dropped", threadCount)
		}
		if rowCount > 1 { // header row only
			t.Fatalf("table rendered %d rows with grant-A data after switching to grant-B", rowCount)
		}

		select {
		case <-deadline:
			return
		case <-ticker.C:
		}
	}
}

// TestMessagesViewLoadDropsStaleFolderData verifies the apply callback drops
// in-flight results after a folder switch: selecting folders quickly must
// never show one folder's threads under another folder's title.
func TestMessagesViewLoadDropsStaleFolderData(t *testing.T) {
	app := createTestApp(t)
	mock, ok := app.config.Client.(*nylas.MockClient)
	if !ok {
		t.Fatal("test app client is not a MockClient")
	}

	release := make(chan struct{})
	returned := make(chan struct{})
	mock.GetThreadsFunc = func(ctx context.Context, grantID string, params *domain.ThreadQueryParams) ([]domain.Thread, error) {
		<-release // hold folder-A's fetch in flight until the folder switch happened
		defer close(returned)
		return []domain.Thread{{ID: "thread-A", Subject: "Folder A thread"}}, nil
	}

	view := NewMessagesView(app)
	view.folderPanel.folders = []domain.Folder{
		{ID: "folder-A", Name: "Folder A"},
		{ID: "folder-B", Name: "Folder B"},
	}

	startTestAppRunning(t, app)

	// Start a load for folder-A.
	runOnEventLoop(t, app, func() {
		view.currentFolderID = "folder-A"
		view.Load()
	})

	// Select folder-B while folder-A's fetch is still in flight, then let
	// folder-A's fetch complete.
	runOnEventLoop(t, app, func() { view.currentFolderID = "folder-B" })
	close(release)
	select {
	case <-returned:
	case <-time.After(time.Second):
		t.Fatal("GetThreads never returned")
	}

	// The stale apply callback is queued right after GetThreads returns.
	// Poll a few event-loop turns and assert folder-A's threads are never
	// rendered.
	deadline := time.After(300 * time.Millisecond)
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	for {
		var threadCount int
		runOnEventLoop(t, app, func() { threadCount = len(view.threads) })
		if threadCount != 0 {
			t.Fatalf("view shows %d threads from folder-A after switching to folder-B; stale apply must be dropped", threadCount)
		}

		select {
		case <-deadline:
			return
		case <-ticker.C:
		}
	}
}

// TestEventsViewLoadDropsStaleCalendarData verifies the apply callback drops
// in-flight results after a calendar switch: paging quickly through calendars
// must never let an earlier slow fetch overwrite the newly selected
// calendar's events.
func TestEventsViewLoadDropsStaleCalendarData(t *testing.T) {
	app := createTestApp(t)
	mock, ok := app.config.Client.(*nylas.MockClient)
	if !ok {
		t.Fatal("test app client is not a MockClient")
	}

	release := make(chan struct{})
	returned := make(chan struct{})
	mock.GetEventsFunc = func(ctx context.Context, grantID, calendarID string, params *domain.EventQueryParams) ([]domain.Event, error) {
		if calendarID != "cal-A" {
			return nil, nil // cal-B load completes immediately
		}
		<-release // hold cal-A's fetch in flight until the calendar switch happened
		defer close(returned)
		return []domain.Event{{ID: "event-A", CalendarID: "cal-A", Title: "Calendar A event"}}, nil
	}

	view := NewEventsView(app)

	startTestAppRunning(t, app)

	// Seed calendars and start a load for cal-A.
	runOnEventLoop(t, app, func() {
		view.calendar.SetCalendars([]domain.Calendar{{ID: "cal-A", Name: "A"}, {ID: "cal-B", Name: "B"}})
		view.loadEventsForCalendar("cal-A")
	})

	// Switch to cal-B while cal-A's fetch is still in flight (NextCalendar
	// kicks off cal-B's load, which returns immediately), then let cal-A's
	// fetch complete.
	runOnEventLoop(t, app, func() { view.calendar.NextCalendar() })
	close(release)
	select {
	case <-returned:
	case <-time.After(time.Second):
		t.Fatal("GetEvents never returned")
	}

	// The stale apply callback is queued right after GetEvents returns. Poll
	// a few event-loop turns and assert cal-A's events are never rendered.
	deadline := time.After(300 * time.Millisecond)
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	for {
		var eventCount, widgetCount int
		runOnEventLoop(t, app, func() {
			eventCount = len(view.events)
			widgetCount = len(view.calendar.events)
		})
		if eventCount != 0 {
			t.Fatalf("view shows %d events from cal-A after switching to cal-B; stale apply must be dropped", eventCount)
		}
		if widgetCount != 0 {
			t.Fatalf("calendar widget shows %d events from cal-A after switching to cal-B; stale apply must be dropped", widgetCount)
		}

		select {
		case <-deadline:
			return
		case <-ticker.C:
		}
	}
}

// TestAvailabilityViewLoadRendersBeforeGrantFetch verifies Load renders the
// current (empty) state synchronously when it spawns the current-user grant
// fetch: the info panel must reflect the view's settings immediately instead
// of staying stale/blank until the fetch returns.
func TestAvailabilityViewLoadRendersBeforeGrantFetch(t *testing.T) {
	app := createTestApp(t)
	mock, ok := app.config.Client.(*nylas.MockClient)
	if !ok {
		t.Fatal("test app client is not a MockClient")
	}

	release := make(chan struct{})
	defer close(release)
	mock.GetGrantFunc = func(ctx context.Context, grantID string) (*domain.Grant, error) {
		<-release // keep the grant fetch in flight for the whole test
		return nil, context.Canceled
	}

	view := NewAvailabilityView(app)

	startTestAppRunning(t, app)

	// Change a setting and Load with no participants: the synchronous render
	// must pick up the new duration even though the grant fetch never returns.
	runOnEventLoop(t, app, func() {
		view.duration = 45
		view.Load()
	})

	var infoText string
	runOnEventLoop(t, app, func() { infoText = view.infoPanel.GetText(true) })
	if !strings.Contains(infoText, "45 min") {
		t.Fatalf("info panel = %q, want it rendered synchronously with duration 45 min", infoText)
	}
}

// TestMessagesViewLoadAppliesThreadsViaEventLoop verifies the Load split: the
// network fetch runs off the event loop (Load returns immediately) and the
// fetched threads are applied through QueueUpdateDraw so the view state is
// only mutated on the event loop.
func TestMessagesViewLoadAppliesThreadsViaEventLoop(t *testing.T) {
	app := createTestApp(t)
	mock, ok := app.config.Client.(*nylas.MockClient)
	if !ok {
		t.Fatal("test app client is not a MockClient")
	}
	mock.GetThreadsFunc = func(ctx context.Context, grantID string, params *domain.ThreadQueryParams) ([]domain.Thread, error) {
		return []domain.Thread{
			{ID: "thread-1", Subject: "First"},
			{ID: "thread-2", Subject: "Second"},
		}, nil
	}

	view := NewMessagesView(app)
	// Pre-populate folders so Load skips the folder fetch; the mock client
	// records call metadata without locking, so the test keeps client calls
	// sequential.
	view.folderPanel.folders = []domain.Folder{{ID: "INBOX", Name: "Inbox"}}

	startTestAppRunning(t, app)

	runOnEventLoop(t, app, func() { view.Load() })

	deadline := time.After(time.Second)
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	for {
		var threadCount, rowCount int
		runOnEventLoop(t, app, func() {
			threadCount = len(view.threads)
			rowCount = view.table.GetRowCount()
		})
		if threadCount == 2 {
			if rowCount < 2 { // both fetched threads rendered
				t.Fatalf("table row count = %d, want >= 2 after render", rowCount)
			}
			return
		}

		select {
		case <-deadline:
			t.Fatalf("threads not applied on event loop: got %d, want 2", threadCount)
		case <-ticker.C:
		}
	}
}
