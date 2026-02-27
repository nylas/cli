package mcp

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/nylas/cli/internal/domain"
)

// ============================================================================
// TestExecuteEpochToDatetime
// ============================================================================

func TestExecuteEpochToDatetime(t *testing.T) {
	t.Parallel()

	s := newMockServer(&mockNylasClient{})

	tests := []struct {
		name      string
		args      map[string]any
		wantError bool
		wantText  string
	}{
		{
			name:     "happy path with timezone",
			args:     map[string]any{"epoch": float64(1700000000), "timezone": "UTC"},
			wantText: "1700000000",
		},
		{
			name:      "missing epoch returns error",
			args:      map[string]any{"timezone": "UTC"},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			resp := s.executeEpochToDatetime(tt.args)
			if tt.wantError {
				if !resp.IsError {
					t.Errorf("expected error, got: %s", resp.Content[0].Text)
				}
				return
			}
			if resp.IsError {
				t.Fatalf("unexpected error: %s", resp.Content[0].Text)
			}
			var out map[string]any
			if err := json.Unmarshal([]byte(resp.Content[0].Text), &out); err != nil {
				t.Fatalf("unmarshal response: %v", err)
			}
			if tt.wantText != "" && !strings.Contains(resp.Content[0].Text, tt.wantText) {
				t.Errorf("response text = %q, want to contain %q", resp.Content[0].Text, tt.wantText)
			}
			if _, ok := out["datetime"]; !ok {
				t.Errorf("datetime field missing: %v", out)
			}
		})
	}
}

// ============================================================================
// TestExecuteDatetimeToEpoch
// ============================================================================

func TestExecuteDatetimeToEpoch(t *testing.T) {
	t.Parallel()

	s := newMockServer(&mockNylasClient{})

	tests := []struct {
		name      string
		args      map[string]any
		wantError bool
	}{
		{name: "RFC3339 format", args: map[string]any{"datetime": "2023-11-14T22:13:20Z", "timezone": "UTC"}},
		{name: "datetime format", args: map[string]any{"datetime": "2023-11-14 22:13:20", "timezone": "UTC"}},
		{name: "date only format", args: map[string]any{"datetime": "2023-11-14", "timezone": "UTC"}},
		{name: "invalid format returns error", args: map[string]any{"datetime": "not-a-date"}, wantError: true},
		{name: "missing datetime returns error", args: map[string]any{}, wantError: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			resp := s.executeDatetimeToEpoch(tt.args)
			if tt.wantError {
				if !resp.IsError {
					t.Errorf("expected error, got: %s", resp.Content[0].Text)
				}
				return
			}
			if resp.IsError {
				t.Fatalf("unexpected error: %s", resp.Content[0].Text)
			}
			var out map[string]any
			if err := json.Unmarshal([]byte(resp.Content[0].Text), &out); err != nil {
				t.Fatalf("unmarshal response: %v", err)
			}
			if _, ok := out["unix_timestamp"]; !ok {
				t.Errorf("unix_timestamp missing: %v", out)
			}
		})
	}
}

// ============================================================================
// TestParseParticipants
// ============================================================================

func TestParseParticipants(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		args      map[string]any
		key       string
		wantCount int
	}{
		{
			name: "valid array with name and email",
			args: map[string]any{
				"to": []any{
					map[string]any{"name": "Alice", "email": "alice@example.com"},
					map[string]any{"email": "bob@example.com"},
				},
			},
			key:       "to",
			wantCount: 2,
		},
		{
			name:      "empty array",
			args:      map[string]any{"to": []any{}},
			key:       "to",
			wantCount: 0,
		},
		{
			name:      "missing key returns nil",
			args:      map[string]any{},
			key:       "to",
			wantCount: 0,
		},
		{
			name:      "invalid type in array is skipped",
			args:      map[string]any{"to": []any{"not-a-map", map[string]any{"email": "x@y.com"}}},
			key:       "to",
			wantCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := parseParticipants(tt.args, tt.key)
			if len(result) != tt.wantCount {
				t.Errorf("len = %d, want %d", len(result), tt.wantCount)
			}
		})
	}
}

// ============================================================================
// TestFormatParticipants
// ============================================================================

func TestFormatParticipants(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		participants []domain.EmailParticipant
		want         []string
	}{
		{
			name:         "with name uses Name <email> format",
			participants: []domain.EmailParticipant{{Name: "Alice", Email: "alice@example.com"}},
			want:         []string{"Alice <alice@example.com>"},
		},
		{
			name:         "without name uses email only",
			participants: []domain.EmailParticipant{{Email: "bob@example.com"}},
			want:         []string{"bob@example.com"},
		},
		{
			name:         "mixed",
			participants: []domain.EmailParticipant{{Name: "A", Email: "a@b.com"}, {Email: "c@d.com"}},
			want:         []string{"A <a@b.com>", "c@d.com"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := formatParticipants(tt.participants)
			if len(got) != len(tt.want) {
				t.Fatalf("len = %d, want %d", len(got), len(tt.want))
			}
			for i := range tt.want {
				if got[i] != tt.want[i] {
					t.Errorf("[%d] = %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}
