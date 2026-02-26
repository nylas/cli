//go:build integration
// +build integration

package mcp

import (
	"context"
	"encoding/json"
	"io"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/nylas/cli/internal/domain"
)

// connectSDKClient creates an MCP SDK client connected to our server in-process
// via io.Pipe. The SDK performs the initialize handshake automatically.
func connectSDKClient(t *testing.T, server *Server) *mcp.ClientSession {
	t.Helper()

	// Pipes: SDK writes → server reads, server writes → SDK reads.
	serverStdinR, serverStdinW := io.Pipe()
	serverStdoutR, serverStdoutW := io.Pipe()

	ctx, cancel := context.WithCancel(context.Background())

	// Start our server.
	serverDone := make(chan error, 1)
	go func() {
		serverDone <- server.RunWithIO(ctx, serverStdinR, serverStdoutW)
		_ = serverStdoutW.Close()
	}()

	// Connect SDK client using IOTransport.
	client := mcp.NewClient(&mcp.Implementation{
		Name:    "test-client",
		Version: "v0.0.1",
	}, nil)

	transport := &mcp.IOTransport{
		Reader: serverStdoutR,
		Writer: serverStdinW,
	}

	session, err := client.Connect(ctx, transport, nil)
	if err != nil {
		cancel()
		t.Fatalf("SDK client.Connect: %v", err)
	}

	t.Cleanup(func() {
		_ = session.Close()
		cancel()
		select {
		case <-serverDone:
		case <-time.After(3 * time.Second):
			t.Log("warning: server goroutine did not stop within 3s")
		}
	})

	return session
}

// ============================================================================
// SDK Integration Tests
// ============================================================================

func TestSDK_Initialize(t *testing.T) {
	t.Parallel()

	s := newMockServer(&mockNylasClient{})
	session := connectSDKClient(t, s)

	result := session.InitializeResult()
	if result == nil {
		t.Fatal("InitializeResult is nil")
	}

	if result.ServerInfo.Name != serverName {
		t.Errorf("serverInfo.name = %q, want %q", result.ServerInfo.Name, serverName)
	}
	if result.ServerInfo.Version != serverVersion {
		t.Errorf("serverInfo.version = %q, want %q", result.ServerInfo.Version, serverVersion)
	}
	if result.Instructions == "" {
		t.Error("instructions is empty")
	}
}

func TestSDK_Ping(t *testing.T) {
	t.Parallel()

	s := newMockServer(&mockNylasClient{})
	session := connectSDKClient(t, s)

	ctx := context.Background()
	if err := session.Ping(ctx, nil); err != nil {
		t.Fatalf("Ping: %v", err)
	}
}

func TestSDK_ListTools(t *testing.T) {
	t.Parallel()

	s := newMockServer(&mockNylasClient{})
	session := connectSDKClient(t, s)

	ctx := context.Background()
	result, err := session.ListTools(ctx, nil)
	if err != nil {
		t.Fatalf("ListTools: %v", err)
	}

	const wantCount = 47
	if len(result.Tools) != wantCount {
		t.Errorf("tool count = %d, want %d", len(result.Tools), wantCount)
	}

	// Verify all tools have name + description.
	for _, tool := range result.Tools {
		if tool.Name == "" {
			t.Error("tool has empty name")
		}
		if tool.Description == "" {
			t.Errorf("tool %q has empty description", tool.Name)
		}
	}

	// Verify specific tools exist.
	toolNames := make(map[string]bool)
	for _, tool := range result.Tools {
		toolNames[tool.Name] = true
	}

	expected := []string{
		"list_messages", "send_message", "get_message",
		"list_events", "create_event", "send_rsvp",
		"list_contacts", "list_calendars",
		"current_time", "epoch_to_datetime", "datetime_to_epoch",
	}
	for _, name := range expected {
		if !toolNames[name] {
			t.Errorf("expected tool %q not found", name)
		}
	}
}

func TestSDK_ToolCall_CurrentTime(t *testing.T) {
	t.Parallel()

	s := newMockServer(&mockNylasClient{})
	session := connectSDKClient(t, s)
	ctx := context.Background()

	result, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "current_time",
		Arguments: map[string]any{"timezone": "UTC"},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if result.IsError {
		t.Fatal("tool returned isError=true")
	}
	if len(result.Content) == 0 {
		t.Fatal("empty content")
	}

	text := result.Content[0].(*mcp.TextContent).Text
	var data map[string]any
	if err := json.Unmarshal([]byte(text), &data); err != nil {
		t.Fatalf("unmarshal tool result: %v", err)
	}
	if tz, _ := data["timezone"].(string); tz != "UTC" {
		t.Errorf("timezone = %q, want UTC", tz)
	}
	if dt, _ := data["datetime"].(string); dt == "" {
		t.Error("datetime is empty")
	}
}

func TestSDK_ToolCall_EpochToDatetime(t *testing.T) {
	t.Parallel()

	s := newMockServer(&mockNylasClient{})
	session := connectSDKClient(t, s)
	ctx := context.Background()

	result, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "epoch_to_datetime",
		Arguments: map[string]any{"epoch": float64(0), "timezone": "UTC"},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if result.IsError {
		t.Fatal("tool returned isError=true")
	}

	text := result.Content[0].(*mcp.TextContent).Text
	var data map[string]any
	if err := json.Unmarshal([]byte(text), &data); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if dt, _ := data["datetime"].(string); dt != "1970-01-01T00:00:00Z" {
		t.Errorf("datetime = %q, want 1970-01-01T00:00:00Z", dt)
	}
}

func TestSDK_ToolCall_DatetimeToEpoch(t *testing.T) {
	t.Parallel()

	s := newMockServer(&mockNylasClient{})
	session := connectSDKClient(t, s)
	ctx := context.Background()

	result, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "datetime_to_epoch",
		Arguments: map[string]any{"datetime": "1970-01-01T00:00:00Z"},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if result.IsError {
		t.Fatal("tool returned isError=true")
	}

	text := result.Content[0].(*mcp.TextContent).Text
	var data map[string]any
	if err := json.Unmarshal([]byte(text), &data); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if epoch, _ := data["epoch"].(float64); epoch != 0 {
		t.Errorf("epoch = %v, want 0", epoch)
	}
}

func TestSDK_ToolCall_ListMessages(t *testing.T) {
	t.Parallel()

	mock := &mockNylasClient{
		getMessagesWithParamsFunc: func(_ context.Context, grantID string, _ *domain.MessageQueryParams) ([]domain.Message, error) {
			return []domain.Message{
				{ID: "msg-1", Subject: "Hello", Snippet: "Hi there"},
				{ID: "msg-2", Subject: "Meeting", Snippet: "Let's meet"},
			}, nil
		},
	}
	s := newMockServer(mock)
	session := connectSDKClient(t, s)
	ctx := context.Background()

	result, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "list_messages",
		Arguments: map[string]any{"limit": float64(5)},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if result.IsError {
		t.Fatal("tool returned isError=true")
	}

	text := result.Content[0].(*mcp.TextContent).Text
	var messages []map[string]any
	if err := json.Unmarshal([]byte(text), &messages); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(messages) != 2 {
		t.Fatalf("message count = %d, want 2", len(messages))
	}
	if messages[0]["id"] != "msg-1" {
		t.Errorf("messages[0].id = %v, want msg-1", messages[0]["id"])
	}
	if messages[1]["subject"] != "Meeting" {
		t.Errorf("messages[1].subject = %v, want Meeting", messages[1]["subject"])
	}
}

func TestSDK_ToolCall_ListCalendars(t *testing.T) {
	t.Parallel()

	mock := &mockNylasClient{
		getCalendarsFunc: func(_ context.Context, grantID string) ([]domain.Calendar, error) {
			return []domain.Calendar{
				{ID: "cal-1", Name: "Work", IsPrimary: true},
				{ID: "cal-2", Name: "Personal"},
			}, nil
		},
	}
	s := newMockServer(mock)
	session := connectSDKClient(t, s)
	ctx := context.Background()

	result, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "list_calendars",
		Arguments: map[string]any{},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if result.IsError {
		t.Fatal("tool returned isError=true")
	}

	text := result.Content[0].(*mcp.TextContent).Text
	var calendars []map[string]any
	if err := json.Unmarshal([]byte(text), &calendars); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(calendars) != 2 {
		t.Fatalf("calendar count = %d, want 2", len(calendars))
	}
	if calendars[0]["name"] != "Work" {
		t.Errorf("calendars[0].name = %v, want Work", calendars[0]["name"])
	}
}

func TestSDK_ToolCall_ListEvents(t *testing.T) {
	t.Parallel()

	mock := &mockNylasClient{
		getEventsFunc: func(_ context.Context, grantID, calendarID string, _ *domain.EventQueryParams) ([]domain.Event, error) {
			if calendarID != "primary" {
				t.Errorf("calendar_id = %q, want primary", calendarID)
			}
			return []domain.Event{
				{ID: "evt-1", Title: "Standup"},
			}, nil
		},
	}
	s := newMockServer(mock)
	session := connectSDKClient(t, s)
	ctx := context.Background()

	result, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "list_events",
		Arguments: map[string]any{},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if result.IsError {
		t.Fatal("tool returned isError=true")
	}

	text := result.Content[0].(*mcp.TextContent).Text
	var events []map[string]any
	if err := json.Unmarshal([]byte(text), &events); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("event count = %d, want 1", len(events))
	}
	if events[0]["title"] != "Standup" {
		t.Errorf("events[0].title = %v, want Standup", events[0]["title"])
	}
}

func TestSDK_ToolCall_ListFolders(t *testing.T) {
	t.Parallel()

	mock := &mockNylasClient{
		getFoldersFunc: func(_ context.Context, grantID string) ([]domain.Folder, error) {
			return []domain.Folder{
				{ID: "folder-inbox", Name: "Inbox"},
				{ID: "folder-sent", Name: "Sent"},
			}, nil
		},
	}
	s := newMockServer(mock)
	session := connectSDKClient(t, s)
	ctx := context.Background()

	result, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "list_folders",
		Arguments: map[string]any{},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if result.IsError {
		t.Fatal("tool returned isError=true")
	}

	text := result.Content[0].(*mcp.TextContent).Text
	var folders []map[string]any
	if err := json.Unmarshal([]byte(text), &folders); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(folders) != 2 {
		t.Fatalf("folder count = %d, want 2", len(folders))
	}
	if folders[0]["name"] != "Inbox" {
		t.Errorf("folders[0].name = %v, want Inbox", folders[0]["name"])
	}
}

func TestSDK_ToolCall_UnknownTool(t *testing.T) {
	t.Parallel()

	s := newMockServer(&mockNylasClient{})
	session := connectSDKClient(t, s)
	ctx := context.Background()

	result, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "totally_fake_tool",
		Arguments: map[string]any{},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if !result.IsError {
		t.Error("expected isError=true for unknown tool")
	}
}

func TestSDK_ToolCall_GrantIDInjection(t *testing.T) {
	t.Parallel()

	var capturedGrantID string
	mock := &mockNylasClient{
		getFoldersFunc: func(_ context.Context, grantID string) ([]domain.Folder, error) {
			capturedGrantID = grantID
			return nil, nil
		},
	}
	s := newMockServer(mock)
	session := connectSDKClient(t, s)
	ctx := context.Background()

	// Call without grant_id — should use default "test-grant".
	_, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "list_folders",
		Arguments: map[string]any{},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if capturedGrantID != "test-grant" {
		t.Errorf("default grant = %q, want test-grant", capturedGrantID)
	}

	// Call with explicit grant_id — should override default.
	capturedGrantID = ""
	_, err = session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "list_folders",
		Arguments: map[string]any{"grant_id": "explicit-grant"},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if capturedGrantID != "explicit-grant" {
		t.Errorf("explicit grant = %q, want explicit-grant", capturedGrantID)
	}
}

func TestSDK_ToolCall_SendMessage(t *testing.T) {
	t.Parallel()

	var capturedReq *domain.SendMessageRequest
	mock := &mockNylasClient{
		sendMessageFunc: func(_ context.Context, grantID string, req *domain.SendMessageRequest) (*domain.Message, error) {
			capturedReq = req
			return &domain.Message{ID: "sent-1", Subject: req.Subject}, nil
		},
	}
	s := newMockServer(mock)
	session := connectSDKClient(t, s)
	ctx := context.Background()

	result, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "send_message",
		Arguments: map[string]any{
			"to":      []any{map[string]any{"email": "bob@example.com", "name": "Bob"}},
			"subject": "SDK Test",
			"body":    "Hello from the Go SDK!",
		},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if result.IsError {
		t.Fatal("tool returned isError=true")
	}

	if capturedReq == nil {
		t.Fatal("send was not called")
	}
	if capturedReq.Subject != "SDK Test" {
		t.Errorf("subject = %q, want SDK Test", capturedReq.Subject)
	}
	if len(capturedReq.To) != 1 || capturedReq.To[0].Email != "bob@example.com" {
		t.Errorf("to = %v, want [{bob@example.com Bob}]", capturedReq.To)
	}

	text := result.Content[0].(*mcp.TextContent).Text
	var data map[string]any
	if err := json.Unmarshal([]byte(text), &data); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if data["status"] != "sent" {
		t.Errorf("status = %v, want sent", data["status"])
	}
}

func TestSDK_MultipleToolCalls(t *testing.T) {
	t.Parallel()

	mock := &mockNylasClient{
		getCalendarsFunc: func(_ context.Context, _ string) ([]domain.Calendar, error) {
			return []domain.Calendar{{ID: "cal-1", Name: "Work"}}, nil
		},
		getFoldersFunc: func(_ context.Context, _ string) ([]domain.Folder, error) {
			return []domain.Folder{{ID: "f-1", Name: "Inbox"}}, nil
		},
	}
	s := newMockServer(mock)
	session := connectSDKClient(t, s)
	ctx := context.Background()

	// Call current_time, then list_calendars, then list_folders in sequence.
	tools := []string{"current_time", "list_calendars", "list_folders"}
	for _, name := range tools {
		result, err := session.CallTool(ctx, &mcp.CallToolParams{
			Name:      name,
			Arguments: map[string]any{},
		})
		if err != nil {
			t.Fatalf("CallTool(%s): %v", name, err)
		}
		if result.IsError {
			t.Errorf("CallTool(%s): isError=true", name)
		}
		if len(result.Content) == 0 {
			t.Errorf("CallTool(%s): empty content", name)
		}
	}
}
