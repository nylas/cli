package mcp

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/nylas/cli/internal/domain"
)

// jsonRPCResponse is a minimal structure for parsing JSON-RPC responses in routing tests.
type jsonRPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Result  json.RawMessage `json:"result"`
	Error   *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

// toolResult mirrors ToolResponse for JSON parsing in routing tests.
type toolResult struct {
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	IsError bool `json:"isError,omitempty"`
}

// newRoutingServer creates a Server with a mock that returns safe non-nil zero values.
// This avoids nil pointer dereferences in executors that succeed past validation.
func newRoutingServer() *Server {
	return newMockServer(&mockNylasClient{
		getMessagesWithParamsFunc: func(_ context.Context, _ string, _ *domain.MessageQueryParams) ([]domain.Message, error) {
			return nil, nil
		},
		getMessageFunc: func(_ context.Context, _, _ string) (*domain.Message, error) {
			return &domain.Message{}, nil
		},
		sendMessageFunc: func(_ context.Context, _ string, _ *domain.SendMessageRequest) (*domain.Message, error) {
			return &domain.Message{}, nil
		},
		updateMessageFunc: func(_ context.Context, _, _ string, _ *domain.UpdateMessageRequest) (*domain.Message, error) {
			return &domain.Message{}, nil
		},
		smartComposeFunc: func(_ context.Context, _ string, _ *domain.SmartComposeRequest) (*domain.SmartComposeSuggestion, error) {
			return &domain.SmartComposeSuggestion{}, nil
		},
		smartComposeReplyFunc: func(_ context.Context, _, _ string, _ *domain.SmartComposeRequest) (*domain.SmartComposeSuggestion, error) {
			return &domain.SmartComposeSuggestion{}, nil
		},
		getDraftsFunc: func(_ context.Context, _ string, _ int) ([]domain.Draft, error) {
			return nil, nil
		},
		getDraftFunc: func(_ context.Context, _, _ string) (*domain.Draft, error) {
			return &domain.Draft{}, nil
		},
		createDraftFunc: func(_ context.Context, _ string, _ *domain.CreateDraftRequest) (*domain.Draft, error) {
			return &domain.Draft{}, nil
		},
		updateDraftFunc: func(_ context.Context, _, _ string, _ *domain.CreateDraftRequest) (*domain.Draft, error) {
			return &domain.Draft{}, nil
		},
		sendDraftFunc: func(_ context.Context, _, _ string) (*domain.Message, error) {
			return &domain.Message{}, nil
		},
		getThreadsFunc: func(_ context.Context, _ string, _ *domain.ThreadQueryParams) ([]domain.Thread, error) {
			return nil, nil
		},
		getThreadFunc: func(_ context.Context, _, _ string) (*domain.Thread, error) {
			return &domain.Thread{}, nil
		},
		updateThreadFunc: func(_ context.Context, _, _ string, _ *domain.UpdateMessageRequest) (*domain.Thread, error) {
			return &domain.Thread{}, nil
		},
		getFoldersFunc: func(_ context.Context, _ string) ([]domain.Folder, error) {
			return nil, nil
		},
		getFolderFunc: func(_ context.Context, _, _ string) (*domain.Folder, error) {
			return &domain.Folder{}, nil
		},
		createFolderFunc: func(_ context.Context, _ string, _ *domain.CreateFolderRequest) (*domain.Folder, error) {
			return &domain.Folder{}, nil
		},
		updateFolderFunc: func(_ context.Context, _, _ string, _ *domain.UpdateFolderRequest) (*domain.Folder, error) {
			return &domain.Folder{}, nil
		},
		getCalendarsFunc: func(_ context.Context, _ string) ([]domain.Calendar, error) {
			return nil, nil
		},
		getCalendarFunc: func(_ context.Context, _, _ string) (*domain.Calendar, error) {
			return &domain.Calendar{}, nil
		},
		createCalendarFunc: func(_ context.Context, _ string, _ *domain.CreateCalendarRequest) (*domain.Calendar, error) {
			return &domain.Calendar{}, nil
		},
		updateCalendarFunc: func(_ context.Context, _, _ string, _ *domain.UpdateCalendarRequest) (*domain.Calendar, error) {
			return &domain.Calendar{}, nil
		},
		getEventsFunc: func(_ context.Context, _, _ string, _ *domain.EventQueryParams) ([]domain.Event, error) {
			return nil, nil
		},
		getEventFunc: func(_ context.Context, _, _, _ string) (*domain.Event, error) {
			return &domain.Event{}, nil
		},
		createEventFunc: func(_ context.Context, _, _ string, _ *domain.CreateEventRequest) (*domain.Event, error) {
			return &domain.Event{}, nil
		},
		updateEventFunc: func(_ context.Context, _, _, _ string, _ *domain.UpdateEventRequest) (*domain.Event, error) {
			return &domain.Event{}, nil
		},
		getContactsFunc: func(_ context.Context, _ string, _ *domain.ContactQueryParams) ([]domain.Contact, error) {
			return nil, nil
		},
		getContactFunc: func(_ context.Context, _, _ string) (*domain.Contact, error) {
			return &domain.Contact{}, nil
		},
		createContactFunc: func(_ context.Context, _ string, _ *domain.CreateContactRequest) (*domain.Contact, error) {
			return &domain.Contact{}, nil
		},
		updateContactFunc: func(_ context.Context, _, _ string, _ *domain.UpdateContactRequest) (*domain.Contact, error) {
			return &domain.Contact{}, nil
		},
		getAttachmentFunc: func(_ context.Context, _, _, _ string) (*domain.Attachment, error) {
			return &domain.Attachment{}, nil
		},
		getFreeBusyFunc: func(_ context.Context, _ string, _ *domain.FreeBusyRequest) (*domain.FreeBusyResponse, error) {
			return &domain.FreeBusyResponse{}, nil
		},
		getAvailabilityFunc: func(_ context.Context, _ *domain.AvailabilityRequest) (*domain.AvailabilityResponse, error) {
			return &domain.AvailabilityResponse{}, nil
		},
	})
}

// parseRPCResponse parses raw bytes as a JSON-RPC response.
func parseRPCResponse(t *testing.T, data []byte) jsonRPCResponse {
	t.Helper()
	var resp jsonRPCResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		t.Fatalf("failed to parse JSON-RPC response: %v (raw: %s)", err, data)
	}
	return resp
}

// parseToolResult parses the result field of a JSON-RPC response as a ToolResponse.
func parseToolResult(t *testing.T, resp jsonRPCResponse) toolResult {
	t.Helper()
	var tr toolResult
	if err := json.Unmarshal(resp.Result, &tr); err != nil {
		t.Fatalf("failed to parse tool result: %v (raw result: %s)", err, resp.Result)
	}
	return tr
}

// makeReq builds a minimal tools/call Request with the given tool name and args.
func makeReq(name string, args map[string]any) *Request {
	params := ToolCallParams{
		Name:      name,
		Arguments: args,
	}
	raw, _ := json.Marshal(params)
	return &Request{
		JSONRPC: "2.0",
		Method:  "tools/call",
		Params:  raw,
	}
}

// ============================================================================
// TestHandleToolCall_AllToolsRoute
// ============================================================================

func TestHandleToolCall_AllToolsRoute(t *testing.T) {
	t.Parallel()

	s := newRoutingServer()
	ctx := context.Background()

	tests := []struct {
		name string
		args map[string]any
	}{
		// Email tools
		{name: "list_messages", args: map[string]any{}},
		{name: "get_message", args: map[string]any{"message_id": "m1"}},
		{name: "send_message", args: map[string]any{
			"to":      []any{map[string]any{"email": "a@b.com"}},
			"subject": "s",
			"body":    "b",
		}},
		{name: "update_message", args: map[string]any{"message_id": "m1"}},
		{name: "delete_message", args: map[string]any{"message_id": "m1"}},
		{name: "smart_compose", args: map[string]any{"prompt": "test"}},
		{name: "smart_compose_reply", args: map[string]any{"message_id": "m1", "prompt": "test"}},

		// Draft tools
		{name: "list_drafts", args: map[string]any{}},
		{name: "get_draft", args: map[string]any{"draft_id": "d1"}},
		{name: "create_draft", args: map[string]any{}},
		{name: "update_draft", args: map[string]any{"draft_id": "d1"}},
		{name: "send_draft", args: map[string]any{"draft_id": "d1"}},
		{name: "delete_draft", args: map[string]any{"draft_id": "d1"}},

		// Thread tools
		{name: "list_threads", args: map[string]any{}},
		{name: "get_thread", args: map[string]any{"thread_id": "t1"}},
		{name: "update_thread", args: map[string]any{"thread_id": "t1"}},
		{name: "delete_thread", args: map[string]any{"thread_id": "t1"}},

		// Folder tools
		{name: "list_folders", args: map[string]any{}},
		{name: "get_folder", args: map[string]any{"folder_id": "f1"}},
		{name: "create_folder", args: map[string]any{"name": "test"}},
		{name: "update_folder", args: map[string]any{"folder_id": "f1"}},
		{name: "delete_folder", args: map[string]any{"folder_id": "f1"}},

		// Attachment tools
		{name: "list_attachments", args: map[string]any{"message_id": "m1"}},
		{name: "get_attachment", args: map[string]any{"message_id": "m1", "attachment_id": "a1"}},

		// Scheduled message tools
		{name: "list_scheduled_messages", args: map[string]any{}},
		{name: "cancel_scheduled_message", args: map[string]any{"schedule_id": "s1"}},

		// Calendar tools
		{name: "list_calendars", args: map[string]any{}},
		{name: "get_calendar", args: map[string]any{"calendar_id": "c1"}},
		{name: "create_calendar", args: map[string]any{"name": "test"}},
		{name: "update_calendar", args: map[string]any{"calendar_id": "c1"}},
		{name: "delete_calendar", args: map[string]any{"calendar_id": "c1"}},

		// Event tools
		{name: "list_events", args: map[string]any{}},
		{name: "get_event", args: map[string]any{"event_id": "e1"}},
		{name: "create_event", args: map[string]any{"title": "test", "start_time": float64(1700000000), "end_time": float64(1700003600)}},
		{name: "update_event", args: map[string]any{"event_id": "e1"}},
		{name: "delete_event", args: map[string]any{"event_id": "e1"}},
		{name: "send_rsvp", args: map[string]any{"event_id": "e1", "status": "yes"}},

		// Availability tools
		{name: "get_free_busy", args: map[string]any{
			"emails":     []any{"a@b.com"},
			"start_time": float64(1000),
			"end_time":   float64(2000),
		}},
		{name: "get_availability", args: map[string]any{
			"start_time":       float64(1000),
			"end_time":         float64(2000),
			"duration_minutes": float64(30),
			"participants":     []any{map[string]any{"email": "test@example.com"}},
		}},

		// Contact tools
		{name: "list_contacts", args: map[string]any{}},
		{name: "get_contact", args: map[string]any{"contact_id": "c1"}},
		{name: "create_contact", args: map[string]any{}},
		{name: "update_contact", args: map[string]any{"contact_id": "c1"}},
		{name: "delete_contact", args: map[string]any{"contact_id": "c1"}},

		// Utility tools
		{name: "current_time", args: map[string]any{}},
		{name: "epoch_to_datetime", args: map[string]any{"epoch": float64(1700000000)}},
		{name: "datetime_to_epoch", args: map[string]any{"datetime": "2023-11-14T22:13:20Z"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			req := makeReq(tt.name, tt.args)
			raw := s.handleToolCall(ctx, req)

			// Must parse as valid JSON-RPC.
			rpc := parseRPCResponse(t, raw)
			if rpc.Error != nil {
				t.Fatalf("tool %q: unexpected JSON-RPC error: code=%d msg=%s",
					tt.name, rpc.Error.Code, rpc.Error.Message)
			}
			if rpc.Result == nil {
				t.Fatalf("tool %q: result field is missing", tt.name)
			}

			// Result must contain a tool response (not a JSON-RPC level error).
			tr := parseToolResult(t, rpc)
			if tr.IsError {
				text := ""
				if len(tr.Content) > 0 {
					text = tr.Content[0].Text
				}
				t.Errorf("tool %q: executor returned isError=true: %s", tt.name, text)
			}
		})
	}
}

// ============================================================================
// TestHandleToolCall_UnknownTool
// ============================================================================

func TestHandleToolCall_UnknownTool(t *testing.T) {
	t.Parallel()

	s := newMockServer(&mockNylasClient{})
	ctx := context.Background()

	req := makeReq("no_such_tool", map[string]any{})
	raw := s.handleToolCall(ctx, req)

	// Unknown tool now returns a JSON-RPC error with code -32602.
	rpc := parseRPCResponse(t, raw)
	if rpc.Error == nil {
		t.Fatal("expected JSON-RPC error for unknown tool, got result")
	}
	if rpc.Error.Code != codeInvalidParams {
		t.Errorf("error.code = %d, want %d", rpc.Error.Code, codeInvalidParams)
	}
	if !strings.Contains(rpc.Error.Message, "unknown tool") {
		t.Errorf("expected error message to contain 'unknown tool', got: %s", rpc.Error.Message)
	}
}
