package mcp

import (
	"testing"

	"github.com/nylas/cli/internal/domain"
)

// ============================================================================
// TestFormatEventWhen
// ============================================================================

func TestFormatEventWhen(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		when      domain.EventWhen
		wantKeys  []string
		wantEmpty bool
	}{
		{
			name:     "start_time branch",
			when:     domain.EventWhen{StartTime: 1700000000, EndTime: 1700003600},
			wantKeys: []string{"start_time", "end_time"},
		},
		{
			name:     "date branch (all-day)",
			when:     domain.EventWhen{Date: "2024-01-15"},
			wantKeys: []string{"date"},
		},
		{
			name:     "start_date branch (multi-day)",
			when:     domain.EventWhen{StartDate: "2024-01-15", EndDate: "2024-01-16"},
			wantKeys: []string{"start_date", "end_date"},
		},
		{
			name:      "empty when returns empty map",
			when:      domain.EventWhen{},
			wantEmpty: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := formatEventWhen(tt.when)
			if tt.wantEmpty {
				if len(result) != 0 {
					t.Errorf("expected empty map, got %v", result)
				}
				return
			}
			for _, key := range tt.wantKeys {
				if _, ok := result[key]; !ok {
					t.Errorf("key %q missing from result", key)
				}
			}
		})
	}
}

// ============================================================================
// TestParseEventParticipants
// ============================================================================

func TestParseEventParticipants(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		args      map[string]any
		wantCount int
		wantNil   bool
	}{
		{
			name: "valid array with name and email",
			args: map[string]any{
				"participants": []any{
					map[string]any{"name": "Alice", "email": "alice@test.com"},
					map[string]any{"email": "bob@test.com"},
				},
			},
			wantCount: 2,
		},
		{
			name:    "missing key returns nil",
			args:    map[string]any{},
			wantNil: true,
		},
		{
			name: "items without email are skipped",
			args: map[string]any{
				"participants": []any{
					map[string]any{"name": "NoEmail"},
					map[string]any{"email": "valid@test.com"},
				},
			},
			wantCount: 1,
		},
		{
			name: "non-map items are skipped",
			args: map[string]any{
				"participants": []any{"not-a-map", map[string]any{"email": "x@y.com"}},
			},
			wantCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := parseEventParticipants(tt.args)
			if tt.wantNil {
				if result != nil {
					t.Errorf("expected nil, got %v", result)
				}
				return
			}
			if len(result) != tt.wantCount {
				t.Errorf("len = %d, want %d", len(result), tt.wantCount)
			}
		})
	}
}

// ============================================================================
// TestParseReminders
// ============================================================================

func TestParseReminders(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		args    map[string]any
		wantNil bool
		wantLen int
	}{
		{
			name: "valid array returns Reminders",
			args: map[string]any{
				"reminders": []any{
					map[string]any{"minutes": float64(10), "method": "email"},
					map[string]any{"minutes": float64(5), "method": "popup"},
				},
			},
			wantLen: 2,
		},
		{
			name:    "missing key returns nil",
			args:    map[string]any{},
			wantNil: true,
		},
		{
			name:    "empty array returns nil",
			args:    map[string]any{"reminders": []any{}},
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := parseReminders(tt.args)
			if tt.wantNil {
				if result != nil {
					t.Errorf("expected nil, got %v", result)
				}
				return
			}
			if result == nil {
				t.Fatal("expected non-nil Reminders")
			}
			if len(result.Overrides) != tt.wantLen {
				t.Errorf("overrides len = %d, want %d", len(result.Overrides), tt.wantLen)
			}
		})
	}
}

// ============================================================================
// TestParseAvailabilityParticipants
// ============================================================================

func TestParseAvailabilityParticipants(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		args      map[string]any
		wantNil   bool
		wantCount int
		checkFn   func(t *testing.T, result []domain.AvailabilityParticipant)
	}{
		{
			name: "valid participant with calendar_ids",
			args: map[string]any{
				"participants": []any{
					map[string]any{
						"email":        "a@b.com",
						"calendar_ids": []any{"cal1", "cal2"},
					},
				},
			},
			wantCount: 1,
			checkFn: func(t *testing.T, result []domain.AvailabilityParticipant) {
				t.Helper()
				if len(result[0].CalendarIDs) != 2 {
					t.Errorf("calendar_ids len = %d, want 2", len(result[0].CalendarIDs))
				}
			},
		},
		{
			name:    "missing key returns nil",
			args:    map[string]any{},
			wantNil: true,
		},
		{
			name: "items without email are skipped",
			args: map[string]any{
				"participants": []any{
					map[string]any{"name": "no email here"},
					map[string]any{"email": "valid@test.com"},
				},
			},
			wantCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := parseAvailabilityParticipants(tt.args)
			if tt.wantNil {
				if result != nil {
					t.Errorf("expected nil, got %v", result)
				}
				return
			}
			if len(result) != tt.wantCount {
				t.Errorf("len = %d, want %d", len(result), tt.wantCount)
			}
			if tt.checkFn != nil {
				tt.checkFn(t, result)
			}
		})
	}
}
