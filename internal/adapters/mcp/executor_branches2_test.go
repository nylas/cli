package mcp

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/nylas/cli/internal/domain"
)

// ============================================================================
// TestParseParticipants_NonArrayValue
// ============================================================================

func TestParseParticipants_NonArrayValue(t *testing.T) {
	t.Parallel()

	result := parseParticipants(map[string]any{"to": "not-an-array"}, "to")
	if result != nil {
		t.Errorf("expected nil for non-array value, got %v", result)
	}
}

// ============================================================================
// TestParseParticipants_EmptyEmail
// ============================================================================

func TestParseParticipants_EmptyEmail(t *testing.T) {
	t.Parallel()

	args := map[string]any{
		"to": []any{
			map[string]any{"name": "NoEmail", "email": ""},
			map[string]any{"email": "valid@test.com"},
		},
	}
	result := parseParticipants(args, "to")
	if len(result) != 1 {
		t.Fatalf("len = %d, want 1 (empty email should be skipped)", len(result))
	}
	if result[0].Email != "valid@test.com" {
		t.Errorf("email = %q, want valid@test.com", result[0].Email)
	}
}

// ============================================================================
// TestExecuteGetMessage_APIError
// ============================================================================

func TestExecuteGetMessage_APIError(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	s := newMockServer(&mockNylasClient{
		getMessageFunc: func(_ context.Context, _, _ string) (*domain.Message, error) {
			return nil, errors.New("not found")
		},
	})
	resp := s.executeGetMessage(ctx, map[string]any{"message_id": "m1"})
	if !resp.IsError {
		t.Errorf("expected error, got: %s", resp.Content[0].Text)
	}
}

// ============================================================================
// TestExecuteSendMessage_APIError
// ============================================================================

func TestExecuteSendMessage_APIError(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	s := newMockServer(&mockNylasClient{
		sendMessageFunc: func(_ context.Context, _ string, _ *domain.SendMessageRequest) (*domain.Message, error) {
			return nil, errors.New("rate limited")
		},
	})
	resp := s.executeSendMessage(ctx, map[string]any{
		"to":      []any{map[string]any{"email": "a@b.com"}},
		"subject": "Test",
		"body":    "Hello",
	})
	if !resp.IsError {
		t.Errorf("expected error, got: %s", resp.Content[0].Text)
	}
}

// ============================================================================
// TestExecuteGetMessage_EmptyFrom
// ============================================================================

func TestExecuteGetMessage_EmptyFrom(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	s := newMockServer(&mockNylasClient{
		getMessageFunc: func(_ context.Context, _, _ string) (*domain.Message, error) {
			return &domain.Message{ID: "m1", Subject: "Hi", Date: time.Now()}, nil
		},
	})
	resp := s.executeGetMessage(ctx, map[string]any{"message_id": "m1"})
	if resp.IsError {
		t.Fatalf("unexpected error: %s", resp.Content[0].Text)
	}
	var result map[string]any
	unmarshalText(t, resp, &result)
	if result["from"] != "" {
		t.Errorf("from = %v, want empty string for no From", result["from"])
	}
}

// ============================================================================
// TestParseEventParticipants_NonArrayValue
// ============================================================================

func TestParseEventParticipants_NonArrayValue(t *testing.T) {
	t.Parallel()

	result := parseEventParticipants(map[string]any{"participants": "not-an-array"})
	if result != nil {
		t.Errorf("expected nil for non-array value, got %v", result)
	}
}

// ============================================================================
// TestParseReminders_InvalidInputs
// ============================================================================

func TestParseReminders_InvalidInputs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		args    map[string]any
		wantNil bool
		wantLen int
	}{
		{
			name:    "non-array value returns nil",
			args:    map[string]any{"reminders": "not-an-array"},
			wantNil: true,
		},
		{
			name: "non-map items are skipped, valid items parsed",
			args: map[string]any{
				"reminders": []any{
					"not-a-map",
					map[string]any{"minutes": float64(5), "method": "email"},
				},
			},
			wantLen: 1,
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
// TestParseAvailabilityParticipants_InvalidInputs
// ============================================================================

func TestParseAvailabilityParticipants_InvalidInputs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		args      map[string]any
		wantNil   bool
		wantCount int
	}{
		{
			name:    "non-array value returns nil",
			args:    map[string]any{"participants": "not-an-array"},
			wantNil: true,
		},
		{
			name: "non-map items are skipped, valid items parsed",
			args: map[string]any{
				"participants": []any{
					"not-a-map",
					map[string]any{"email": "a@b.com"},
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
		})
	}
}

// ============================================================================
// TestParseContactEmails_NonMapItems
// ============================================================================

func TestParseContactEmails_NonMapItems(t *testing.T) {
	t.Parallel()

	args := map[string]any{
		"emails": []any{
			"not-a-map",
			map[string]any{"email": "a@b.com", "type": "work"},
		},
	}
	result := parseContactEmails(args)
	if len(result) != 1 {
		t.Fatalf("len = %d, want 1", len(result))
	}
	if result[0].Email != "a@b.com" {
		t.Errorf("email = %q, want a@b.com", result[0].Email)
	}
}

// ============================================================================
// TestParseContactPhones_InvalidInputs
// ============================================================================

func TestParseContactPhones_InvalidInputs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		args    map[string]any
		wantLen int
		wantNil bool
	}{
		{
			name:    "non-array value returns nil",
			args:    map[string]any{"phone_numbers": "not-an-array"},
			wantNil: true,
		},
		{
			name: "non-map items are skipped, valid items parsed",
			args: map[string]any{
				"phone_numbers": []any{
					"not-a-map",
					map[string]any{"number": "+1", "type": "mobile"},
				},
			},
			wantLen: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := parseContactPhones(tt.args)
			if tt.wantNil {
				if result != nil {
					t.Errorf("expected nil, got %v", result)
				}
				return
			}
			if len(result) != tt.wantLen {
				t.Fatalf("len = %d, want %d", len(result), tt.wantLen)
			}
			if result[0].Number != "+1" {
				t.Errorf("number = %q, want +1", result[0].Number)
			}
		})
	}
}

// ============================================================================
// TestToInt64_AllTypes
// ============================================================================

func TestToInt64_AllTypes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   any
		wantVal int64
		wantOK  bool
	}{
		{name: "float64", input: float64(42), wantVal: 42, wantOK: true},
		{name: "int", input: int(42), wantVal: 42, wantOK: true},
		{name: "int64", input: int64(42), wantVal: 42, wantOK: true},
		{name: "string not a number", input: "not-a-number", wantVal: 0, wantOK: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, ok := toInt64(tt.input)
			if ok != tt.wantOK {
				t.Errorf("ok = %v, want %v", ok, tt.wantOK)
			}
			if got != tt.wantVal {
				t.Errorf("val = %d, want %d", got, tt.wantVal)
			}
		})
	}
}

// ============================================================================
// TestExecuteEpochToDatetime_InvalidInputs
// ============================================================================

func TestExecuteEpochToDatetime_InvalidInputs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		args map[string]any
	}{
		{
			name: "invalid timezone returns error",
			args: map[string]any{"epoch": float64(1700000000), "timezone": "Invalid/Zone"},
		},
		{
			name: "non-numeric epoch returns error",
			args: map[string]any{"epoch": "not-a-number"},
		},
	}

	s := newMockServer(&mockNylasClient{})
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			resp := s.executeEpochToDatetime(tt.args)
			if !resp.IsError {
				t.Errorf("expected error, got: %s", resp.Content[0].Text)
			}
		})
	}
}

// ============================================================================
// TestExecuteDatetimeToEpoch_InvalidTimezone
// ============================================================================

func TestExecuteDatetimeToEpoch_InvalidTimezone(t *testing.T) {
	t.Parallel()

	s := newMockServer(&mockNylasClient{})
	resp := s.executeDatetimeToEpoch(map[string]any{
		"datetime": "2023-11-14T22:13:20Z",
		"timezone": "Invalid/Zone",
	})
	if !resp.IsError {
		t.Errorf("expected error for invalid timezone, got: %s", resp.Content[0].Text)
	}
}

// ============================================================================
// TestHandleToolCall_NilArguments
// ============================================================================

func TestHandleToolCall_NilArguments(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	s := newMockServer(&mockNylasClient{})

	req := &Request{
		JSONRPC: "2.0",
		Method:  "tools/call",
		Params: struct {
			Name      string         `json:"name"`
			Arguments map[string]any `json:"arguments"`
			Cursor    string         `json:"cursor,omitempty"`
		}{
			Name:      "current_time",
			Arguments: nil,
		},
	}

	raw := s.handleToolCall(ctx, req)
	if raw == nil {
		t.Fatal("handleToolCall returned nil for nil Arguments")
	}

	rpc := parseRPCResponse(t, raw)
	if rpc.Error != nil {
		t.Fatalf("unexpected JSON-RPC error: %s", rpc.Error.Message)
	}

	tr := parseToolResult(t, rpc)
	if tr.IsError {
		text := ""
		if len(tr.Content) > 0 {
			text = tr.Content[0].Text
		}
		t.Errorf("unexpected tool error with nil arguments: %s", text)
	}
}

