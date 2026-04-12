package calendar

import (
	"context"
	"testing"
	"time"

	"github.com/nylas/cli/internal/adapters/ai"
	"github.com/nylas/cli/internal/adapters/nylas"
	"github.com/nylas/cli/internal/domain"
)

func TestCreateMeetingFromOptionCreatesRealEvent(t *testing.T) {
	t.Parallel()

	client := nylas.NewMockClient()
	client.GetCalendarsFunc = func(ctx context.Context, grantID string) ([]domain.Calendar, error) {
		return []domain.Calendar{{ID: "primary", IsPrimary: true}}, nil
	}

	var created *domain.CreateEventRequest
	client.CreateEventFunc = func(ctx context.Context, grantID, calendarID string, req *domain.CreateEventRequest) (*domain.Event, error) {
		created = req
		if grantID != "grant-123" {
			t.Fatalf("grantID = %s, want grant-123", grantID)
		}
		if calendarID != "primary" {
			t.Fatalf("calendarID = %s, want primary", calendarID)
		}
		return &domain.Event{ID: "event-123", CalendarID: calendarID, Title: req.Title}, nil
	}

	start := time.Date(2026, 1, 12, 15, 0, 0, 0, time.UTC)
	option := ai.ScheduleOption{
		StartTime: start,
		EndTime:   start.Add(time.Hour),
		Timezone:  "UTC",
		Reasoning: "Best overlap for the team.",
		Participants: map[string]ai.ParticipantTime{
			"alice@example.com": {Email: "alice@example.com"},
			"bob@example.com":   {Email: "bob@example.com"},
		},
	}

	if err := createMeetingFromOption(nil, option, "grant-123", client); err != nil {
		t.Fatalf("createMeetingFromOption() error = %v", err)
	}

	if created == nil {
		t.Fatal("expected CreateEvent to be called")
	}
	if created.Title != "Meeting with alice, bob" {
		t.Fatalf("created title = %q, want %q", created.Title, "Meeting with alice, bob")
	}
	if created.When.StartTimezone != "UTC" {
		t.Fatalf("start timezone = %q, want UTC", created.When.StartTimezone)
	}
	if len(created.Participants) != 2 {
		t.Fatalf("participants = %d, want 2", len(created.Participants))
	}
}
