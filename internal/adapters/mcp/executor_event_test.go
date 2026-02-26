package mcp

import (
	"context"
	"errors"
	"testing"

	"github.com/nylas/cli/internal/domain"
)

// ============================================================================
// TestExecuteUpdateEvent
// ============================================================================

func TestExecuteUpdateEvent(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	tests := []struct {
		name      string
		args      map[string]any
		mockFn    func(ctx context.Context, grantID, calendarID, eventID string, req *domain.UpdateEventRequest) (*domain.Event, error)
		wantError bool
	}{
		{
			name: "happy path returns updated status",
			args: map[string]any{"event_id": "ev1", "title": "Updated"},
			mockFn: func(_ context.Context, _, _, _ string, req *domain.UpdateEventRequest) (*domain.Event, error) {
				return &domain.Event{ID: "ev1", Title: *req.Title}, nil
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
			mockFn: func(_ context.Context, _, _, _ string, _ *domain.UpdateEventRequest) (*domain.Event, error) {
				return nil, errors.New("api down")
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			s := newMockServer(&mockNylasClient{updateEventFunc: tt.mockFn})
			resp := s.executeUpdateEvent(ctx, tt.args)
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
// TestExecuteDeleteEvent
// ============================================================================

func TestExecuteDeleteEvent(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	tests := []struct {
		name      string
		args      map[string]any
		mockFn    func(ctx context.Context, grantID, calendarID, eventID string) error
		wantError bool
	}{
		{
			name:   "happy path returns deleted status with event_id",
			args:   map[string]any{"event_id": "ev1"},
			mockFn: func(_ context.Context, _, _, _ string) error { return nil },
		},
		{
			name:      "missing event_id returns error",
			args:      map[string]any{},
			wantError: true,
		},
		{
			name: "api error propagates",
			args: map[string]any{"event_id": "ev1"},
			mockFn: func(_ context.Context, _, _, _ string) error {
				return errors.New("api down")
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			s := newMockServer(&mockNylasClient{deleteEventFunc: tt.mockFn})
			resp := s.executeDeleteEvent(ctx, tt.args)
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
			if result["status"] != "deleted" {
				t.Errorf("status = %v, want deleted", result["status"])
			}
			if result["event_id"] != "ev1" {
				t.Errorf("event_id = %v, want ev1", result["event_id"])
			}
		})
	}
}

// ============================================================================
// TestExecuteSendRSVP
// ============================================================================

func TestExecuteSendRSVP(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	tests := []struct {
		name      string
		args      map[string]any
		mockFn    func(ctx context.Context, grantID, calendarID, eventID string, req *domain.SendRSVPRequest) error
		wantError bool
		wantRSVP  string
	}{
		{
			name:     "yes rsvp succeeds",
			args:     map[string]any{"event_id": "ev1", "status": "yes"},
			mockFn:   func(_ context.Context, _, _, _ string, _ *domain.SendRSVPRequest) error { return nil },
			wantRSVP: "yes",
		},
		{
			name:      "missing event_id returns error",
			args:      map[string]any{"status": "yes"},
			wantError: true,
		},
		{
			name:      "missing status returns error",
			args:      map[string]any{"event_id": "ev1"},
			wantError: true,
		},
		{
			name: "api error propagates",
			args: map[string]any{"event_id": "ev1", "status": "no"},
			mockFn: func(_ context.Context, _, _, _ string, _ *domain.SendRSVPRequest) error {
				return errors.New("api down")
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			s := newMockServer(&mockNylasClient{sendRSVPFunc: tt.mockFn})
			resp := s.executeSendRSVP(ctx, tt.args)
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
			if result["status"] != "rsvp_sent" {
				t.Errorf("status = %v, want rsvp_sent", result["status"])
			}
			if result["rsvp"] != tt.wantRSVP {
				t.Errorf("rsvp = %v, want %v", result["rsvp"], tt.wantRSVP)
			}
		})
	}
}

// ============================================================================
// TestExecuteGetFreeBusy
// ============================================================================

func TestExecuteGetFreeBusy(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	happyMock := func(_ context.Context, _ string, _ *domain.FreeBusyRequest) (*domain.FreeBusyResponse, error) {
		return &domain.FreeBusyResponse{
			Data: []domain.FreeBusyCalendar{{Email: "a@b.com"}},
		}, nil
	}

	tests := []struct {
		name      string
		args      map[string]any
		mockFn    func(ctx context.Context, grantID string, req *domain.FreeBusyRequest) (*domain.FreeBusyResponse, error)
		wantError bool
	}{
		{
			name:   "happy path returns data",
			args:   map[string]any{"emails": []any{"a@b.com"}, "start_time": float64(1700000000), "end_time": float64(1700003600)},
			mockFn: happyMock,
		},
		{
			name:      "missing emails returns error",
			args:      map[string]any{"start_time": float64(1700000000), "end_time": float64(1700003600)},
			wantError: true,
		},
		{
			name:      "missing start_time returns error",
			args:      map[string]any{"emails": []any{"a@b.com"}, "end_time": float64(1700003600)},
			wantError: true,
		},
		{
			name:      "missing end_time returns error",
			args:      map[string]any{"emails": []any{"a@b.com"}, "start_time": float64(1700000000)},
			wantError: true,
		},
		{
			name: "api error propagates",
			args: map[string]any{"emails": []any{"a@b.com"}, "start_time": float64(1700000000), "end_time": float64(1700003600)},
			mockFn: func(_ context.Context, _ string, _ *domain.FreeBusyRequest) (*domain.FreeBusyResponse, error) {
				return nil, errors.New("api down")
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			s := newMockServer(&mockNylasClient{getFreeBusyFunc: tt.mockFn})
			resp := s.executeGetFreeBusy(ctx, tt.args)
			if tt.wantError {
				if !resp.IsError {
					t.Errorf("expected error, got: %s", resp.Content[0].Text)
				}
				return
			}
			if resp.IsError {
				t.Fatalf("unexpected error: %s", resp.Content[0].Text)
			}
			var result []map[string]any
			unmarshalText(t, resp, &result)
			if len(result) != 1 {
				t.Errorf("result len = %d, want 1", len(result))
			}
		})
	}
}

// ============================================================================
// TestExecuteGetAvailability
// ============================================================================

func TestExecuteGetAvailability(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	happyMock := func(_ context.Context, _ *domain.AvailabilityRequest) (*domain.AvailabilityResponse, error) {
		return &domain.AvailabilityResponse{
			Data: domain.AvailabilityData{
				TimeSlots: []domain.AvailableSlot{{StartTime: 1700000000, EndTime: 1700003600}},
			},
		}, nil
	}

	tests := []struct {
		name      string
		args      map[string]any
		mockFn    func(ctx context.Context, req *domain.AvailabilityRequest) (*domain.AvailabilityResponse, error)
		wantError bool
	}{
		{
			name:   "happy path returns data",
			args:   map[string]any{"start_time": float64(1700000000), "end_time": float64(1700086400), "duration_minutes": float64(30)},
			mockFn: happyMock,
		},
		{
			name:      "missing start_time returns error",
			args:      map[string]any{"end_time": float64(1700086400), "duration_minutes": float64(30)},
			wantError: true,
		},
		{
			name:      "missing end_time returns error",
			args:      map[string]any{"start_time": float64(1700000000), "duration_minutes": float64(30)},
			wantError: true,
		},
		{
			name:      "missing duration_minutes returns error",
			args:      map[string]any{"start_time": float64(1700000000), "end_time": float64(1700086400)},
			wantError: true,
		},
		{
			name: "api error propagates",
			args: map[string]any{"start_time": float64(1700000000), "end_time": float64(1700086400), "duration_minutes": float64(30)},
			mockFn: func(_ context.Context, _ *domain.AvailabilityRequest) (*domain.AvailabilityResponse, error) {
				return nil, errors.New("api down")
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			s := newMockServer(&mockNylasClient{getAvailabilityFunc: tt.mockFn})
			resp := s.executeGetAvailability(ctx, tt.args)
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
			if _, ok := result["time_slots"]; !ok {
				t.Error("time_slots field missing from availability data")
			}
		})
	}
}
