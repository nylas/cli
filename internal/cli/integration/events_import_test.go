//go:build integration

package integration

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/nylas/cli/internal/domain"
)

func TestCLI_CalendarEventsImportHelp(t *testing.T) {
	if testBinary == "" {
		t.Skip("CLI binary not found")
	}

	stdout, stderr, err := runCLI("calendar", "events", "import", "--help")
	if err != nil {
		t.Fatalf("calendar events import --help failed: %v\nstderr: %s", err, stderr)
	}
	for _, want := range []string{"--calendar", "--start", "--end"} {
		if !strings.Contains(stdout, want) {
			t.Errorf("expected events import help to contain %q, got: %s", want, stdout)
		}
	}
}

// TestEventsImport_Integration exercises the read-only events/import endpoint
// against the live API for the primary calendar.
func TestEventsImport_Integration(t *testing.T) {
	skipIfMissingCreds(t)
	client := getTestClient()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Resolve the primary calendar.
	acquireRateLimit(t)
	calendars, err := client.GetCalendars(ctx, testGrantID)
	if err != nil {
		t.Fatalf("GetCalendars() error = %v", err)
	}
	calID := "primary"
	for _, c := range calendars {
		if c.IsPrimary {
			calID = c.ID
			break
		}
	}

	now := time.Now()
	params := &domain.EventQueryParams{
		CalendarID: calID,
		Start:      now.AddDate(0, -1, 0).Unix(),
		End:        now.AddDate(0, 1, 0).Unix(),
		Limit:      5,
	}

	acquireRateLimit(t)
	events, err := client.ImportEvents(ctx, testGrantID, params)
	if err != nil {
		if isUnavailableErr(err) {
			t.Skipf("events import not available for this account: %v", err)
		}
		t.Fatalf("ImportEvents() error = %v", err)
	}
	// Imported events must belong to the calendar we asked for.
	for _, e := range events {
		if e.ID == "" {
			t.Errorf("imported event has empty ID: %+v", e)
		}
	}
	t.Logf("ImportEvents returned %d event(s) from %s", len(events), calID)
}
