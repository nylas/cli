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

func TestScheduleGrantArgs(t *testing.T) {
	t.Parallel()

	// The NL query must never be used as the grant. With no --grant the
	// result is empty (so WithClient uses the default grant); with --grant the
	// flag value is the only grant arg.
	if got := scheduleGrantArgs(""); got != nil {
		t.Fatalf("scheduleGrantArgs(\"\") = %v, want nil", got)
	}
	got := scheduleGrantArgs("grant-abc")
	if len(got) != 1 || got[0] != "grant-abc" {
		t.Fatalf("scheduleGrantArgs(\"grant-abc\") = %v, want [grant-abc]", got)
	}
}

func TestSelectScheduleOption(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		autoConfirm bool
		optionCount int
		choice      string
		want        int
	}{
		// --yes must create the first option, not silently skip creation.
		{name: "autoConfirm picks first", autoConfirm: true, optionCount: 3, want: 0},
		{name: "autoConfirm with no options", autoConfirm: true, optionCount: 0, want: -1},
		{name: "no options never creates", optionCount: 0, choice: "y", want: -1},
		{name: "yes picks first", optionCount: 3, choice: "y", want: 0},
		{name: "YES uppercase picks first", optionCount: 3, choice: "YES", want: 0},
		{name: "empty declines", optionCount: 3, choice: "", want: -1},
		{name: "n declines", optionCount: 3, choice: "n", want: -1},
		{name: "2 picks second", optionCount: 3, choice: "2", want: 1},
		{name: "3 picks third", optionCount: 3, choice: "3", want: 2},
		{name: "2 out of range declines", optionCount: 1, choice: "2", want: -1},
		{name: "3 out of range declines", optionCount: 2, choice: "3", want: -1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := selectScheduleOption(tt.autoConfirm, tt.optionCount, tt.choice); got != tt.want {
				t.Fatalf("selectScheduleOption(%v, %d, %q) = %d, want %d",
					tt.autoConfirm, tt.optionCount, tt.choice, got, tt.want)
			}
		})
	}
}
