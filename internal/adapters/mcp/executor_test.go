package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/nylas/cli/internal/domain"
)

// newMockServer creates a Server backed by the given mock for unit tests.
func newMockServer(mock *mockNylasClient) *Server {
	return &Server{client: mock, defaultGrant: "test-grant"}
}

// unmarshalText parses the first content block text as JSON into dst.
func unmarshalText(t *testing.T, resp *ToolResponse, dst any) {
	t.Helper()
	if len(resp.Content) == 0 {
		t.Fatal("response has no content")
	}
	if err := json.Unmarshal([]byte(resp.Content[0].Text), dst); err != nil {
		t.Fatalf("unmarshal content text: %v", err)
	}
}

// ============================================================================
// TestExecuteListMessages
// ============================================================================

func TestExecuteListMessages(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	now := time.Now().Truncate(time.Second)

	tests := []struct {
		name      string
		args      map[string]any
		mockFn    func(ctx context.Context, grantID string, params *domain.MessageQueryParams) ([]domain.Message, error)
		wantError bool
		wantCount int
	}{
		{
			name: "happy path returns 2 messages",
			args: map[string]any{},
			mockFn: func(_ context.Context, _ string, _ *domain.MessageQueryParams) ([]domain.Message, error) {
				return []domain.Message{
					{ID: "msg1", Subject: "Hello", From: []domain.EmailParticipant{{Email: "a@b.com"}}, Date: now},
					{ID: "msg2", Subject: "World", From: []domain.EmailParticipant{{Email: "c@d.com"}}, Date: now},
				}, nil
			},
			wantCount: 2,
		},
		{
			name: "error propagates as toolError",
			args: map[string]any{},
			mockFn: func(_ context.Context, _ string, _ *domain.MessageQueryParams) ([]domain.Message, error) {
				return nil, errors.New("api down")
			},
			wantError: true,
		},
		{
			name: "filters are passed to params",
			args: map[string]any{"subject": "Re:", "from": "boss@work.com", "unread": true},
			mockFn: func(_ context.Context, _ string, params *domain.MessageQueryParams) ([]domain.Message, error) {
				if params.Subject != "Re:" {
					return nil, errors.New("subject not passed")
				}
				if params.From != "boss@work.com" {
					return nil, errors.New("from not passed")
				}
				if params.Unread == nil || !*params.Unread {
					return nil, errors.New("unread not passed")
				}
				return []domain.Message{{ID: "m1", Date: now}}, nil
			},
			wantCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			s := newMockServer(&mockNylasClient{getMessagesWithParamsFunc: tt.mockFn})
			resp := s.executeListMessages(ctx, tt.args)
			if tt.wantError {
				if !resp.IsError {
					t.Errorf("expected error response, got: %s", resp.Content[0].Text)
				}
				return
			}
			if resp.IsError {
				t.Fatalf("unexpected error: %s", resp.Content[0].Text)
			}
			var items []map[string]any
			unmarshalText(t, resp, &items)
			if len(items) != tt.wantCount {
				t.Errorf("item count = %d, want %d", len(items), tt.wantCount)
			}
		})
	}
}

// ============================================================================
// TestExecuteGetMessage
// ============================================================================

func TestExecuteGetMessage(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	now := time.Now().Truncate(time.Second)

	longBody := strings.Repeat("x", 15000)

	tests := []struct {
		name      string
		args      map[string]any
		mockFn    func(ctx context.Context, grantID, messageID string) (*domain.Message, error)
		wantError bool
		checkFn   func(t *testing.T, result map[string]any)
	}{
		{
			name: "happy path",
			args: map[string]any{"message_id": "msg123"},
			mockFn: func(_ context.Context, _, _ string) (*domain.Message, error) {
				return &domain.Message{ID: "msg123", Subject: "Test", Date: now}, nil
			},
			checkFn: func(t *testing.T, result map[string]any) {
				t.Helper()
				if result["id"] != "msg123" {
					t.Errorf("id = %v, want msg123", result["id"])
				}
			},
		},
		{
			name:      "missing message_id",
			args:      map[string]any{},
			wantError: true,
		},
		{
			name: "long body is truncated to 10000 chars",
			args: map[string]any{"message_id": "long"},
			mockFn: func(_ context.Context, _, _ string) (*domain.Message, error) {
				return &domain.Message{ID: "long", Body: longBody, Date: now}, nil
			},
			checkFn: func(t *testing.T, result map[string]any) {
				t.Helper()
				body, _ := result["body"].(string)
				if len(body) != 10000 {
					t.Errorf("body len = %d, want 10000", len(body))
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			s := newMockServer(&mockNylasClient{getMessageFunc: tt.mockFn})
			resp := s.executeGetMessage(ctx, tt.args)
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
// TestExecuteSendMessage
// ============================================================================

func TestExecuteSendMessage(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	happyMock := func(_ context.Context, _ string, _ *domain.SendMessageRequest) (*domain.Message, error) {
		return &domain.Message{ID: "sent123", ThreadID: "thread1"}, nil
	}

	tests := []struct {
		name      string
		args      map[string]any
		mockFn    func(ctx context.Context, grantID string, req *domain.SendMessageRequest) (*domain.Message, error)
		wantError bool
		checkFn   func(t *testing.T, result map[string]any)
	}{
		{
			name: "happy path",
			args: map[string]any{
				"to":      []any{map[string]any{"email": "recipient@test.com"}},
				"subject": "Hello",
				"body":    "World",
			},
			mockFn: happyMock,
			checkFn: func(t *testing.T, result map[string]any) {
				t.Helper()
				if result["id"] != "sent123" {
					t.Errorf("id = %v, want sent123", result["id"])
				}
				if result["status"] != "sent" {
					t.Errorf("status = %v, want sent", result["status"])
				}
			},
		},
		{
			name:      "missing to",
			args:      map[string]any{"subject": "Hi", "body": "test"},
			wantError: true,
		},
		{
			name:      "missing subject",
			args:      map[string]any{"to": []any{map[string]any{"email": "x@y.com"}}, "body": "b"},
			wantError: true,
		},
		{
			name:      "missing body",
			args:      map[string]any{"to": []any{map[string]any{"email": "x@y.com"}}, "subject": "s"},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			s := newMockServer(&mockNylasClient{sendMessageFunc: tt.mockFn})
			resp := s.executeSendMessage(ctx, tt.args)
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
// TestExecuteListFolders
// ============================================================================

func TestExecuteListFolders(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	s := newMockServer(&mockNylasClient{
		getFoldersFunc: func(_ context.Context, _ string) ([]domain.Folder, error) {
			return []domain.Folder{
				{ID: "f1", Name: "Inbox"},
				{ID: "f2", Name: "Sent"},
			}, nil
		},
	})
	resp := s.executeListFolders(ctx, map[string]any{})
	if resp.IsError {
		t.Fatalf("unexpected error: %s", resp.Content[0].Text)
	}
	var items []map[string]any
	unmarshalText(t, resp, &items)
	if len(items) != 2 {
		t.Errorf("folder count = %d, want 2", len(items))
	}
	if items[0]["id"] != "f1" {
		t.Errorf("items[0].id = %v, want f1", items[0]["id"])
	}
}

// ============================================================================
// TestExecuteListCalendars
// ============================================================================

func TestExecuteListCalendars(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	s := newMockServer(&mockNylasClient{
		getCalendarsFunc: func(_ context.Context, _ string) ([]domain.Calendar, error) {
			return []domain.Calendar{
				{ID: "cal1", Name: "Work"},
				{ID: "cal2", Name: "Personal"},
			}, nil
		},
	})
	resp := s.executeListCalendars(ctx, map[string]any{})
	if resp.IsError {
		t.Fatalf("unexpected error: %s", resp.Content[0].Text)
	}
	var items []map[string]any
	unmarshalText(t, resp, &items)
	if len(items) != 2 {
		t.Errorf("calendar count = %d, want 2", len(items))
	}
}

// ============================================================================
// TestExecuteListEvents
// ============================================================================

func TestExecuteListEvents(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	var capturedCalendarID string
	s := newMockServer(&mockNylasClient{
		getEventsFunc: func(_ context.Context, _, calendarID string, _ *domain.EventQueryParams) ([]domain.Event, error) {
			capturedCalendarID = calendarID
			return []domain.Event{{ID: "ev1", Title: "Standup"}}, nil
		},
	})
	resp := s.executeListEvents(ctx, map[string]any{})
	if resp.IsError {
		t.Fatalf("unexpected error: %s", resp.Content[0].Text)
	}
	var items []map[string]any
	unmarshalText(t, resp, &items)
	if len(items) != 1 {
		t.Errorf("event count = %d, want 1", len(items))
	}
	if capturedCalendarID != "primary" {
		t.Errorf("default calendarID = %q, want primary", capturedCalendarID)
	}
}

// ============================================================================
// TestExecuteListContacts
// ============================================================================

func TestExecuteListContacts(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	s := newMockServer(&mockNylasClient{
		getContactsFunc: func(_ context.Context, _ string, _ *domain.ContactQueryParams) ([]domain.Contact, error) {
			return []domain.Contact{
				{
					ID:        "c1",
					GivenName: "Alice",
					Surname:   "Smith",
					Emails:    []domain.ContactEmail{{Email: "alice@example.com", Type: "work"}},
				},
			}, nil
		},
	})
	resp := s.executeListContacts(ctx, map[string]any{})
	if resp.IsError {
		t.Fatalf("unexpected error: %s", resp.Content[0].Text)
	}
	var items []map[string]any
	unmarshalText(t, resp, &items)
	if len(items) != 1 {
		t.Fatalf("contact count = %d, want 1", len(items))
	}
	if items[0]["display_name"] != "Alice Smith" {
		t.Errorf("display_name = %v, want 'Alice Smith'", items[0]["display_name"])
	}
}

// ============================================================================
// TestExecuteCurrentTime
// ============================================================================

func TestExecuteCurrentTime(t *testing.T) {
	t.Parallel()

	s := newMockServer(&mockNylasClient{})

	tests := []struct {
		name      string
		args      map[string]any
		wantError bool
		checkTZ   string
	}{
		{
			name: "without timezone uses local",
			args: map[string]any{},
		},
		{
			name:    "valid timezone",
			args:    map[string]any{"timezone": "America/New_York"},
			checkTZ: "America/New_York",
		},
		{
			name:      "invalid timezone returns error",
			args:      map[string]any{"timezone": "Not/Valid_Zone"},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			resp := s.executeCurrentTime(tt.args)
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
			if !strings.Contains(text, "unix:") {
				t.Errorf("response text = %q, want to contain 'unix:'", text)
			}
			if tt.checkTZ != "" {
				if !strings.Contains(text, tt.checkTZ) {
					t.Errorf("response text = %q, want to contain timezone %q", text, tt.checkTZ)
				}
			}
		})
	}
}
