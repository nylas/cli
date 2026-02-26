package mcp

import (
	"context"
	"errors"
	"testing"

	"github.com/nylas/cli/internal/domain"
)

// ============================================================================
// TestExecuteUpdateMessage
// ============================================================================

func TestExecuteUpdateMessage(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	unread := true
	starred := false

	tests := []struct {
		name      string
		args      map[string]any
		mockFn    func(ctx context.Context, grantID, messageID string, req *domain.UpdateMessageRequest) (*domain.Message, error)
		wantError bool
		checkFn   func(t *testing.T, result map[string]any)
	}{
		{
			name: "happy path returns updated fields",
			args: map[string]any{
				"message_id": "msg1",
				"unread":     true,
				"starred":    false,
				"folders":    []any{"INBOX"},
			},
			mockFn: func(_ context.Context, _, _ string, _ *domain.UpdateMessageRequest) (*domain.Message, error) {
				return &domain.Message{
					ID:      "msg1",
					Unread:  true,
					Starred: false,
					Folders: []string{"INBOX"},
				}, nil
			},
			checkFn: func(t *testing.T, result map[string]any) {
				t.Helper()
				if result["id"] != "msg1" {
					t.Errorf("id = %v, want msg1", result["id"])
				}
				if result["unread"] != true {
					t.Errorf("unread = %v, want true", result["unread"])
				}
			},
		},
		{
			name:      "missing message_id returns error",
			args:      map[string]any{"unread": true},
			wantError: true,
		},
		{
			name: "API error propagates",
			args: map[string]any{"message_id": "msg1"},
			mockFn: func(_ context.Context, _, _ string, _ *domain.UpdateMessageRequest) (*domain.Message, error) {
				return nil, errors.New("api error")
			},
			wantError: true,
		},
		{
			name: "optional fields are passed to request",
			args: map[string]any{
				"message_id": "msg2",
				"unread":     true,
				"starred":    false,
			},
			mockFn: func(_ context.Context, _, _ string, req *domain.UpdateMessageRequest) (*domain.Message, error) {
				if req.Unread == nil || *req.Unread != unread {
					return nil, errors.New("unread not passed correctly")
				}
				if req.Starred == nil || *req.Starred != starred {
					return nil, errors.New("starred not passed correctly")
				}
				return &domain.Message{ID: "msg2"}, nil
			},
			checkFn: func(t *testing.T, result map[string]any) {
				t.Helper()
				if result["id"] != "msg2" {
					t.Errorf("id = %v, want msg2", result["id"])
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			s := newMockServer(&mockNylasClient{updateMessageFunc: tt.mockFn})
			resp := s.executeUpdateMessage(ctx, tt.args)
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
// TestExecuteDeleteMessage
// ============================================================================

func TestExecuteDeleteMessage(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	tests := []struct {
		name      string
		args      map[string]any
		mockFn    func(ctx context.Context, grantID, messageID string) error
		wantError bool
	}{
		{
			name:   "happy path returns deleted status",
			args:   map[string]any{"message_id": "msg-del-1"},
			mockFn: func(_ context.Context, _, _ string) error { return nil },
		},
		{
			name:      "missing message_id returns error",
			args:      map[string]any{},
			wantError: true,
		},
		{
			name: "API error propagates",
			args: map[string]any{"message_id": "msg-del-2"},
			mockFn: func(_ context.Context, _, _ string) error {
				return errors.New("not found")
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			s := newMockServer(&mockNylasClient{deleteMessageFunc: tt.mockFn})
			resp := s.executeDeleteMessage(ctx, tt.args)
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
			if result["message_id"] != "msg-del-1" {
				t.Errorf("message_id = %v, want msg-del-1", result["message_id"])
			}
		})
	}
}

// ============================================================================
// TestExecuteSmartCompose
// ============================================================================

func TestExecuteSmartCompose(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	tests := []struct {
		name      string
		args      map[string]any
		mockFn    func(ctx context.Context, grantID string, req *domain.SmartComposeRequest) (*domain.SmartComposeSuggestion, error)
		wantError bool
		checkFn   func(t *testing.T, result map[string]any)
	}{
		{
			name: "happy path returns suggestion",
			args: map[string]any{"prompt": "Write a follow-up email"},
			mockFn: func(_ context.Context, _ string, _ *domain.SmartComposeRequest) (*domain.SmartComposeSuggestion, error) {
				return &domain.SmartComposeSuggestion{Suggestion: "Hi, following up on our conversation..."}, nil
			},
			checkFn: func(t *testing.T, result map[string]any) {
				t.Helper()
				if result["suggestion"] != "Hi, following up on our conversation..." {
					t.Errorf("suggestion = %v, want specific text", result["suggestion"])
				}
			},
		},
		{
			name:      "missing prompt returns error",
			args:      map[string]any{},
			wantError: true,
		},
		{
			name: "API error propagates",
			args: map[string]any{"prompt": "Write something"},
			mockFn: func(_ context.Context, _ string, _ *domain.SmartComposeRequest) (*domain.SmartComposeSuggestion, error) {
				return nil, errors.New("service unavailable")
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			s := newMockServer(&mockNylasClient{smartComposeFunc: tt.mockFn})
			resp := s.executeSmartCompose(ctx, tt.args)
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
// TestExecuteSmartComposeReply
// ============================================================================

func TestExecuteSmartComposeReply(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	tests := []struct {
		name      string
		args      map[string]any
		mockFn    func(ctx context.Context, grantID, messageID string, req *domain.SmartComposeRequest) (*domain.SmartComposeSuggestion, error)
		wantError bool
		checkFn   func(t *testing.T, result map[string]any)
	}{
		{
			name: "happy path returns suggestion",
			args: map[string]any{
				"message_id": "orig-msg",
				"prompt":     "Reply saying I will be there",
			},
			mockFn: func(_ context.Context, _, _ string, _ *domain.SmartComposeRequest) (*domain.SmartComposeSuggestion, error) {
				return &domain.SmartComposeSuggestion{Suggestion: "Sounds great, I will be there!"}, nil
			},
			checkFn: func(t *testing.T, result map[string]any) {
				t.Helper()
				if result["suggestion"] != "Sounds great, I will be there!" {
					t.Errorf("suggestion = %v, want specific text", result["suggestion"])
				}
			},
		},
		{
			name:      "missing message_id returns error",
			args:      map[string]any{"prompt": "Reply yes"},
			wantError: true,
		},
		{
			name:      "missing prompt returns error",
			args:      map[string]any{"message_id": "orig-msg"},
			wantError: true,
		},
		{
			name: "API error propagates",
			args: map[string]any{"message_id": "orig-msg", "prompt": "Reply yes"},
			mockFn: func(_ context.Context, _, _ string, _ *domain.SmartComposeRequest) (*domain.SmartComposeSuggestion, error) {
				return nil, errors.New("compose failed")
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			s := newMockServer(&mockNylasClient{smartComposeReplyFunc: tt.mockFn})
			resp := s.executeSmartComposeReply(ctx, tt.args)
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
