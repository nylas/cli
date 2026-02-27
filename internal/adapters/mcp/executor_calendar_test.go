package mcp

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/nylas/cli/internal/domain"
)

// ============================================================================
// TestExecuteGetCalendar
// ============================================================================

func TestExecuteGetCalendar(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	tests := []struct {
		name      string
		args      map[string]any
		mockFn    func(ctx context.Context, grantID, calendarID string) (*domain.Calendar, error)
		wantError bool
		checkFn   func(t *testing.T, result map[string]any)
	}{
		{
			name: "happy path returns calendar fields",
			args: map[string]any{"calendar_id": "cal1"},
			mockFn: func(_ context.Context, _, _ string) (*domain.Calendar, error) {
				return &domain.Calendar{ID: "cal1", Name: "Work", Timezone: "UTC", HexColor: "#ff0000", IsPrimary: true}, nil
			},
			checkFn: func(t *testing.T, result map[string]any) {
				t.Helper()
				if result["id"] != "cal1" {
					t.Errorf("id = %v, want cal1", result["id"])
				}
				if result["hex_color"] != "#ff0000" {
					t.Errorf("hex_color = %v, want #ff0000", result["hex_color"])
				}
			},
		},
		{
			name:      "missing calendar_id returns error",
			args:      map[string]any{},
			wantError: true,
		},
		{
			name: "api error propagates",
			args: map[string]any{"calendar_id": "cal1"},
			mockFn: func(_ context.Context, _, _ string) (*domain.Calendar, error) {
				return nil, errors.New("api down")
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			s := newMockServer(&mockNylasClient{getCalendarFunc: tt.mockFn})
			resp := s.executeGetCalendar(ctx, tt.args)
			if tt.wantError {
				if !resp.IsError {
					t.Errorf("expected error, got: %s", resp.Content[0].Text)
				}
				return
			}
			if resp.IsError {
				t.Fatalf("unexpected error: %s", resp.Content[0].Text)
			}
			var result map[string]any
			unmarshalText(t, resp, &result)
			if tt.checkFn != nil {
				tt.checkFn(t, result)
			}
		})
	}
}

// ============================================================================
// TestExecuteCreateCalendar
// ============================================================================

func TestExecuteCreateCalendar(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	tests := []struct {
		name      string
		args      map[string]any
		mockFn    func(ctx context.Context, grantID string, req *domain.CreateCalendarRequest) (*domain.Calendar, error)
		wantError bool
	}{
		{
			name: "happy path returns created status",
			args: map[string]any{"name": "My Calendar"},
			mockFn: func(_ context.Context, _ string, req *domain.CreateCalendarRequest) (*domain.Calendar, error) {
				return &domain.Calendar{ID: "new1", Name: req.Name}, nil
			},
		},
		{
			name:      "missing name returns error",
			args:      map[string]any{},
			wantError: true,
		},
		{
			name: "api error propagates",
			args: map[string]any{"name": "My Calendar"},
			mockFn: func(_ context.Context, _ string, _ *domain.CreateCalendarRequest) (*domain.Calendar, error) {
				return nil, errors.New("api down")
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			s := newMockServer(&mockNylasClient{createCalendarFunc: tt.mockFn})
			resp := s.executeCreateCalendar(ctx, tt.args)
			if tt.wantError {
				if !resp.IsError {
					t.Errorf("expected error, got: %s", resp.Content[0].Text)
				}
				return
			}
			if resp.IsError {
				t.Fatalf("unexpected error: %s", resp.Content[0].Text)
			}
			var result map[string]any
			unmarshalText(t, resp, &result)
			if result["status"] != "created" {
				t.Errorf("status = %v, want created", result["status"])
			}
		})
	}
}

// ============================================================================
// TestExecuteUpdateCalendar
// ============================================================================

func TestExecuteUpdateCalendar(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	tests := []struct {
		name      string
		args      map[string]any
		mockFn    func(ctx context.Context, grantID, calendarID string, req *domain.UpdateCalendarRequest) (*domain.Calendar, error)
		wantError bool
	}{
		{
			name: "happy path returns updated status",
			args: map[string]any{"calendar_id": "cal1", "name": "New Name"},
			mockFn: func(_ context.Context, _, _ string, req *domain.UpdateCalendarRequest) (*domain.Calendar, error) {
				return &domain.Calendar{ID: "cal1", Name: *req.Name}, nil
			},
		},
		{
			name:      "missing calendar_id returns error",
			args:      map[string]any{},
			wantError: true,
		},
		{
			name: "api error propagates",
			args: map[string]any{"calendar_id": "cal1"},
			mockFn: func(_ context.Context, _, _ string, _ *domain.UpdateCalendarRequest) (*domain.Calendar, error) {
				return nil, errors.New("api down")
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			s := newMockServer(&mockNylasClient{updateCalendarFunc: tt.mockFn})
			resp := s.executeUpdateCalendar(ctx, tt.args)
			if tt.wantError {
				if !resp.IsError {
					t.Errorf("expected error, got: %s", resp.Content[0].Text)
				}
				return
			}
			if resp.IsError {
				t.Fatalf("unexpected error: %s", resp.Content[0].Text)
			}
			var result map[string]any
			unmarshalText(t, resp, &result)
			if result["status"] != "updated" {
				t.Errorf("status = %v, want updated", result["status"])
			}
		})
	}
}

// ============================================================================
// TestExecuteDeleteCalendar
// ============================================================================

func TestExecuteDeleteCalendar(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	tests := []struct {
		name      string
		args      map[string]any
		mockFn    func(ctx context.Context, grantID, calendarID string) error
		wantError bool
	}{
		{
			name:   "happy path returns deleted status",
			args:   map[string]any{"calendar_id": "cal1"},
			mockFn: func(_ context.Context, _, _ string) error { return nil },
		},
		{
			name:      "missing calendar_id returns error",
			args:      map[string]any{},
			wantError: true,
		},
		{
			name: "api error propagates",
			args: map[string]any{"calendar_id": "cal1"},
			mockFn: func(_ context.Context, _, _ string) error {
				return errors.New("api down")
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			s := newMockServer(&mockNylasClient{deleteCalendarFunc: tt.mockFn})
			resp := s.executeDeleteCalendar(ctx, tt.args)
			if tt.wantError {
				if !resp.IsError {
					t.Errorf("expected error, got: %s", resp.Content[0].Text)
				}
				return
			}
			if resp.IsError {
				t.Fatalf("unexpected error: %s", resp.Content[0].Text)
			}
			text := resp.Content[0].Text
			if !strings.Contains(text, "Deleted") {
				t.Errorf("response text = %q, want to contain 'Deleted'", text)
			}
			if !strings.Contains(text, "cal1") {
				t.Errorf("response text = %q, want to contain 'cal1'", text)
			}
		})
	}
}

// ============================================================================
// TestExecuteGetEvent
// ============================================================================

func TestExecuteGetEvent(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	tests := []struct {
		name      string
		args      map[string]any
		mockFn    func(ctx context.Context, grantID, calendarID, eventID string) (*domain.Event, error)
		wantError bool
		checkFn   func(t *testing.T, result map[string]any)
	}{
		{
			name: "happy path with organizer and conferencing",
			args: map[string]any{"event_id": "ev1"},
			mockFn: func(_ context.Context, _, _, _ string) (*domain.Event, error) {
				return &domain.Event{
					ID:    "ev1",
					Title: "Standup",
					Organizer: &domain.Participant{
						Person: domain.Person{Name: "Alice", Email: "alice@test.com"},
					},
					Conferencing: &domain.Conferencing{
						Provider: "zoom",
						Details:  &domain.ConferencingDetails{URL: "https://zoom.us/j/123"},
					},
					Participants: []domain.Participant{
						{Person: domain.Person{Name: "Bob", Email: "bob@test.com"}, Status: "yes"},
					},
				}, nil
			},
			checkFn: func(t *testing.T, result map[string]any) {
				t.Helper()
				if result["id"] != "ev1" {
					t.Errorf("id = %v, want ev1", result["id"])
				}
				org, ok := result["organizer"].(map[string]any)
				if !ok {
					t.Fatal("organizer field missing or wrong type")
				}
				if org["email"] != "alice@test.com" {
					t.Errorf("organizer.email = %v, want alice@test.com", org["email"])
				}
				conf, ok := result["conferencing"].(map[string]any)
				if !ok {
					t.Fatal("conferencing field missing or wrong type")
				}
				if conf["provider"] != "zoom" {
					t.Errorf("conferencing.provider = %v, want zoom", conf["provider"])
				}
			},
		},
		{
			name: "event without organizer or conferencing",
			args: map[string]any{"event_id": "ev2", "calendar_id": "cal1"},
			mockFn: func(_ context.Context, _, calendarID, _ string) (*domain.Event, error) {
				if calendarID != "cal1" {
					return nil, errors.New("wrong calendarID")
				}
				return &domain.Event{ID: "ev2", Title: "Solo"}, nil
			},
			checkFn: func(t *testing.T, result map[string]any) {
				t.Helper()
				if _, ok := result["organizer"]; ok {
					t.Error("organizer field should be absent when nil")
				}
				if _, ok := result["conferencing"]; ok {
					t.Error("conferencing field should be absent when nil")
				}
			},
		},
		{
			name:      "missing event_id returns error",
			args:      map[string]any{},
			wantError: true,
		},
		{
			name: "api error propagates",
			args: map[string]any{"event_id": "ev1"},
			mockFn: func(_ context.Context, _, _, _ string) (*domain.Event, error) {
				return nil, errors.New("not found")
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			s := newMockServer(&mockNylasClient{getEventFunc: tt.mockFn})
			resp := s.executeGetEvent(ctx, tt.args)
			if tt.wantError {
				if !resp.IsError {
					t.Errorf("expected error, got: %s", resp.Content[0].Text)
				}
				return
			}
			if resp.IsError {
				t.Fatalf("unexpected error: %s", resp.Content[0].Text)
			}
			var result map[string]any
			unmarshalText(t, resp, &result)
			if tt.checkFn != nil {
				tt.checkFn(t, result)
			}
		})
	}
}

// ============================================================================
// TestExecuteCreateEvent
// ============================================================================

func TestExecuteCreateEvent(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	happyMock := func(_ context.Context, _, _ string, req *domain.CreateEventRequest) (*domain.Event, error) {
		return &domain.Event{ID: "ev-new", Title: req.Title}, nil
	}

	tests := []struct {
		name      string
		args      map[string]any
		mockFn    func(ctx context.Context, grantID, calendarID string, req *domain.CreateEventRequest) (*domain.Event, error)
		wantError bool
	}{
		{
			name:   "happy path with start_time/end_time",
			args:   map[string]any{"title": "Meeting", "start_time": float64(1700000000), "end_time": float64(1700003600)},
			mockFn: happyMock,
		},
		{
			name:   "happy path with start_date/end_date",
			args:   map[string]any{"title": "All Day", "start_date": "2024-01-15", "end_date": "2024-01-15"},
			mockFn: happyMock,
		},
		{
			name:      "missing title returns error",
			args:      map[string]any{},
			wantError: true,
		},
		{
			name:      "missing time returns error",
			args:      map[string]any{"title": "Meeting"},
			wantError: true,
		},
		{
			name: "api error propagates",
			args: map[string]any{"title": "Meeting", "start_time": float64(1700000000), "end_time": float64(1700003600)},
			mockFn: func(_ context.Context, _, _ string, _ *domain.CreateEventRequest) (*domain.Event, error) {
				return nil, errors.New("api down")
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			s := newMockServer(&mockNylasClient{createEventFunc: tt.mockFn})
			resp := s.executeCreateEvent(ctx, tt.args)
			if tt.wantError {
				if !resp.IsError {
					t.Errorf("expected error, got: %s", resp.Content[0].Text)
				}
				return
			}
			if resp.IsError {
				t.Fatalf("unexpected error: %s", resp.Content[0].Text)
			}
			var result map[string]any
			unmarshalText(t, resp, &result)
			if result["status"] != "created" {
				t.Errorf("status = %v, want created", result["status"])
			}
		})
	}
}
