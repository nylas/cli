//go:build integration
// +build integration

package mcp

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/nylas/cli/internal/domain"
)

// toolCall is a convenience wrapper that sends a tools/call request and returns
// the parsed JSON text from the first content block.
func toolCall(t *testing.T, c *mcpClient, id float64, name string, args map[string]any) string {
	t.Helper()
	resp := c.send(map[string]any{
		"jsonrpc": "2.0",
		"id":      id,
		"method":  "tools/call",
		"params": map[string]any{
			"name":      name,
			"arguments": args,
		},
	})
	return assertToolContent(t, resp)
}

// unmarshalToolText parses JSON text from a tool response into dst.
func unmarshalToolText(t *testing.T, text string, dst any) {
	t.Helper()
	if err := json.Unmarshal([]byte(text), dst); err != nil {
		t.Fatalf("unmarshalToolText: %v (text=%s)", err, text)
	}
}

// ============================================================================
// TestIntegration_ToolCall_CurrentTime
// ============================================================================

func TestIntegration_ToolCall_CurrentTime(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		args    map[string]any
		wantErr bool
	}{
		{
			name:    "no timezone uses local",
			args:    map[string]any{},
			wantErr: false,
		},
		{
			name:    "valid timezone",
			args:    map[string]any{"timezone": "UTC"},
			wantErr: false,
		},
		{
			name:    "invalid timezone",
			args:    map[string]any{"timezone": "Not/A_Real_Zone"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			s := newMockServer(&mockNylasClient{})
			c := newMCPTestClient(t, s)

			resp := c.send(map[string]any{
				"jsonrpc": "2.0",
				"id":      float64(1),
				"method":  "tools/call",
				"params": map[string]any{
					"name":      "current_time",
					"arguments": tt.args,
				},
			})

			result, ok := resp["result"].(map[string]any)
			if !ok {
				t.Fatalf("result field missing; resp=%v", resp)
			}

			isErr, _ := result["isError"].(bool)
			if tt.wantErr {
				if !isErr {
					t.Errorf("expected isError=true, got result=%v", result)
				}
				return
			}
			if isErr {
				content, _ := result["content"].([]any)
				if len(content) > 0 {
					if block, ok := content[0].(map[string]any); ok {
						t.Fatalf("unexpected tool error: %v", block["text"])
					}
				}
				t.Fatal("unexpected isError=true")
			}

			text := assertToolContent(t, resp)
			var out map[string]any
			unmarshalToolText(t, text, &out)
			if _, ok := out["datetime"]; !ok {
				t.Error("datetime field missing in current_time response")
			}
			if _, ok := out["unix_timestamp"]; !ok {
				t.Error("unix_timestamp field missing in current_time response")
			}
		})
	}
}

// ============================================================================
// TestIntegration_ToolCall_EpochToDatetime
// ============================================================================

func TestIntegration_ToolCall_EpochToDatetime(t *testing.T) {
	t.Parallel()

	s := newMockServer(&mockNylasClient{})
	c := newMCPTestClient(t, s)

	// epoch 0 in UTC → 1970-01-01T00:00:00Z
	text := toolCall(t, c, 1, "epoch_to_datetime", map[string]any{
		"epoch":    float64(0),
		"timezone": "UTC",
	})

	var out map[string]any
	unmarshalToolText(t, text, &out)

	if out["unix_timestamp"] != float64(0) {
		t.Errorf("unix_timestamp = %v, want 0", out["unix_timestamp"])
	}
	if _, ok := out["datetime"]; !ok {
		t.Error("datetime field missing")
	}
	if out["timezone"] != "UTC" {
		t.Errorf("timezone = %v, want UTC", out["timezone"])
	}
}

// ============================================================================
// TestIntegration_ToolCall_DatetimeToEpoch
// ============================================================================

func TestIntegration_ToolCall_DatetimeToEpoch(t *testing.T) {
	t.Parallel()

	s := newMockServer(&mockNylasClient{})
	c := newMCPTestClient(t, s)

	// 1970-01-01T00:00:00Z → epoch 0
	text := toolCall(t, c, 1, "datetime_to_epoch", map[string]any{
		"datetime": "1970-01-01T00:00:00Z",
		"timezone": "UTC",
	})

	var out map[string]any
	unmarshalToolText(t, text, &out)

	if out["unix_timestamp"] != float64(0) {
		t.Errorf("unix_timestamp = %v, want 0", out["unix_timestamp"])
	}
}

// ============================================================================
// TestIntegration_ToolCall_ListMessages
// ============================================================================

func TestIntegration_ToolCall_ListMessages(t *testing.T) {
	t.Parallel()

	now := time.Now().Truncate(time.Second)
	s := newMockServer(&mockNylasClient{
		getMessagesWithParamsFunc: func(_ context.Context, _ string, _ *domain.MessageQueryParams) ([]domain.Message, error) {
			return []domain.Message{
				{ID: "msg-a", Subject: "Hello", From: []domain.EmailParticipant{{Email: "a@b.com"}}, Date: now},
				{ID: "msg-b", Subject: "World", From: []domain.EmailParticipant{{Email: "c@d.com"}}, Date: now},
			}, nil
		},
	})
	c := newMCPTestClient(t, s)

	text := toolCall(t, c, 1, "list_messages", map[string]any{})

	var items []map[string]any
	unmarshalToolText(t, text, &items)

	if len(items) != 2 {
		t.Fatalf("message count = %d, want 2", len(items))
	}
	if items[0]["id"] != "msg-a" {
		t.Errorf("items[0].id = %v, want msg-a", items[0]["id"])
	}
	if items[1]["id"] != "msg-b" {
		t.Errorf("items[1].id = %v, want msg-b", items[1]["id"])
	}
}

// ============================================================================
// TestIntegration_ToolCall_GetMessage
// ============================================================================

func TestIntegration_ToolCall_GetMessage(t *testing.T) {
	t.Parallel()

	now := time.Now().Truncate(time.Second)
	s := newMockServer(&mockNylasClient{
		getMessageFunc: func(_ context.Context, _, msgID string) (*domain.Message, error) {
			return &domain.Message{
				ID:      msgID,
				Subject: "Integration Test",
				Body:    "Hello body",
				Date:    now,
			}, nil
		},
	})
	c := newMCPTestClient(t, s)

	text := toolCall(t, c, 1, "get_message", map[string]any{"message_id": "id-123"})

	var out map[string]any
	unmarshalToolText(t, text, &out)

	if out["id"] != "id-123" {
		t.Errorf("id = %v, want id-123", out["id"])
	}
	if out["subject"] != "Integration Test" {
		t.Errorf("subject = %v, want 'Integration Test'", out["subject"])
	}
	if out["body"] != "Hello body" {
		t.Errorf("body = %v, want 'Hello body'", out["body"])
	}
}

// ============================================================================
// TestIntegration_ToolCall_SendMessage
// ============================================================================

func TestIntegration_ToolCall_SendMessage(t *testing.T) {
	t.Parallel()

	var capturedReq *domain.SendMessageRequest
	s := newMockServer(&mockNylasClient{
		sendMessageFunc: func(_ context.Context, _ string, req *domain.SendMessageRequest) (*domain.Message, error) {
			capturedReq = req
			return &domain.Message{ID: "sent-xyz", ThreadID: "thread-1"}, nil
		},
	})
	c := newMCPTestClient(t, s)

	text := toolCall(t, c, 1, "send_message", map[string]any{
		"to":      []any{map[string]any{"email": "recipient@test.com", "name": "Recipient"}},
		"subject": "Test Subject",
		"body":    "Test body",
	})

	var out map[string]any
	unmarshalToolText(t, text, &out)

	if out["id"] != "sent-xyz" {
		t.Errorf("id = %v, want sent-xyz", out["id"])
	}
	if out["status"] != "sent" {
		t.Errorf("status = %v, want sent", out["status"])
	}

	if capturedReq == nil {
		t.Fatal("send request was never captured")
	}
	if len(capturedReq.To) == 0 || capturedReq.To[0].Email != "recipient@test.com" {
		t.Errorf("To[0].Email = %v, want recipient@test.com", capturedReq.To)
	}
	if capturedReq.Subject != "Test Subject" {
		t.Errorf("Subject = %v, want 'Test Subject'", capturedReq.Subject)
	}
}

// ============================================================================
// TestIntegration_ToolCall_ListCalendars
// ============================================================================

func TestIntegration_ToolCall_ListCalendars(t *testing.T) {
	t.Parallel()

	s := newMockServer(&mockNylasClient{
		getCalendarsFunc: func(_ context.Context, _ string) ([]domain.Calendar, error) {
			return []domain.Calendar{
				{ID: "cal-1", Name: "Work"},
				{ID: "cal-2", Name: "Personal"},
			}, nil
		},
	})
	c := newMCPTestClient(t, s)

	text := toolCall(t, c, 1, "list_calendars", map[string]any{})

	var items []map[string]any
	unmarshalToolText(t, text, &items)

	if len(items) != 2 {
		t.Fatalf("calendar count = %d, want 2", len(items))
	}
	if items[0]["id"] != "cal-1" {
		t.Errorf("items[0].id = %v, want cal-1", items[0]["id"])
	}
}

// ============================================================================
// TestIntegration_ToolCall_ListEvents
// ============================================================================

func TestIntegration_ToolCall_ListEvents(t *testing.T) {
	t.Parallel()

	var capturedCalendarID string
	s := newMockServer(&mockNylasClient{
		getEventsFunc: func(_ context.Context, _, calendarID string, _ *domain.EventQueryParams) ([]domain.Event, error) {
			capturedCalendarID = calendarID
			return []domain.Event{{ID: "ev-1", Title: "Standup"}}, nil
		},
	})
	c := newMCPTestClient(t, s)

	text := toolCall(t, c, 1, "list_events", map[string]any{})

	var items []map[string]any
	unmarshalToolText(t, text, &items)

	if len(items) != 1 {
		t.Fatalf("event count = %d, want 1", len(items))
	}
	if items[0]["id"] != "ev-1" {
		t.Errorf("items[0].id = %v, want ev-1", items[0]["id"])
	}
	// Default calendar_id is "primary"
	if capturedCalendarID != "primary" {
		t.Errorf("calendar_id passed = %q, want primary", capturedCalendarID)
	}
}

// ============================================================================
// TestIntegration_ToolCall_ListContacts
// ============================================================================

func TestIntegration_ToolCall_ListContacts(t *testing.T) {
	t.Parallel()

	s := newMockServer(&mockNylasClient{
		getContactsFunc: func(_ context.Context, _ string, _ *domain.ContactQueryParams) ([]domain.Contact, error) {
			return []domain.Contact{
				{
					ID:        "c-1",
					GivenName: "Alice",
					Surname:   "Smith",
					Emails:    []domain.ContactEmail{{Email: "alice@example.com", Type: "work"}},
				},
			}, nil
		},
	})
	c := newMCPTestClient(t, s)

	text := toolCall(t, c, 1, "list_contacts", map[string]any{})

	var items []map[string]any
	unmarshalToolText(t, text, &items)

	if len(items) != 1 {
		t.Fatalf("contact count = %d, want 1", len(items))
	}
	if items[0]["id"] != "c-1" {
		t.Errorf("items[0].id = %v, want c-1", items[0]["id"])
	}
	if items[0]["display_name"] != "Alice Smith" {
		t.Errorf("display_name = %v, want 'Alice Smith'", items[0]["display_name"])
	}
}

// ============================================================================
// TestIntegration_ToolCall_ListFolders
// ============================================================================

func TestIntegration_ToolCall_ListFolders(t *testing.T) {
	t.Parallel()

	s := newMockServer(&mockNylasClient{
		getFoldersFunc: func(_ context.Context, _ string) ([]domain.Folder, error) {
			return []domain.Folder{
				{ID: "f-1", Name: "Inbox"},
				{ID: "f-2", Name: "Sent"},
				{ID: "f-3", Name: "Drafts"},
			}, nil
		},
	})
	c := newMCPTestClient(t, s)

	text := toolCall(t, c, 1, "list_folders", map[string]any{})

	var items []map[string]any
	unmarshalToolText(t, text, &items)

	if len(items) != 3 {
		t.Fatalf("folder count = %d, want 3", len(items))
	}
	if items[0]["name"] != "Inbox" {
		t.Errorf("items[0].name = %v, want Inbox", items[0]["name"])
	}
}

// ============================================================================
// TestIntegration_GrantIDResolution
// ============================================================================

func TestIntegration_GrantIDResolution(t *testing.T) {
	t.Parallel()

	var capturedGrantID string
	s := &Server{
		client: &mockNylasClient{
			getFoldersFunc: func(_ context.Context, grantID string) ([]domain.Folder, error) {
				capturedGrantID = grantID
				return []domain.Folder{{ID: "f-1", Name: "Inbox"}}, nil
			},
		},
		defaultGrant: "default-grant",
	}
	c := newMCPTestClient(t, s)

	// Provide explicit grant_id in arguments — should override default.
	_ = toolCall(t, c, 1, "list_folders", map[string]any{"grant_id": "explicit-grant"})

	if capturedGrantID != "explicit-grant" {
		t.Errorf("grant_id used = %q, want explicit-grant", capturedGrantID)
	}

	// Second call without grant_id — should fall back to default.
	c2 := newMCPTestClient(t, s)
	_ = toolCall(t, c2, 1, "list_folders", map[string]any{})

	if capturedGrantID != "default-grant" {
		t.Errorf("grant_id used = %q, want default-grant", capturedGrantID)
	}
}
