package mcp

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/nylas/cli/internal/domain"
)

// ============================================================================
// TestExecuteListDrafts
// ============================================================================

func TestExecuteListDrafts(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	now := time.Now().Truncate(time.Second)

	tests := []struct {
		name      string
		args      map[string]any
		mockFn    func(ctx context.Context, grantID string, limit int) ([]domain.Draft, error)
		wantError bool
		wantCount int
	}{
		{
			name: "happy path returns drafts",
			args: map[string]any{},
			mockFn: func(_ context.Context, _ string, _ int) ([]domain.Draft, error) {
				return []domain.Draft{
					{ID: "d1", Subject: "Draft One", To: []domain.EmailParticipant{{Email: "a@b.com"}}, CreatedAt: now},
					{ID: "d2", Subject: "Draft Two", CreatedAt: now},
				}, nil
			},
			wantCount: 2,
		},
		{
			name: "empty list is valid",
			args: map[string]any{},
			mockFn: func(_ context.Context, _ string, _ int) ([]domain.Draft, error) {
				return []domain.Draft{}, nil
			},
			wantCount: 0,
		},
		{
			name: "API error propagates",
			args: map[string]any{},
			mockFn: func(_ context.Context, _ string, _ int) ([]domain.Draft, error) {
				return nil, errors.New("drafts unavailable")
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			s := newMockServer(&mockNylasClient{getDraftsFunc: tt.mockFn})
			resp := s.executeListDrafts(ctx, tt.args)
			if tt.wantError {
				if !resp.IsError {
					t.Errorf("expected error, got: %s", resp.Content[0].Text)
				}
				return
			}
			if resp.IsError {
				t.Fatalf("unexpected error: %s", resp.Content[0].Text)
			}
			var items []map[string]any
			unmarshalText(t, resp, &items)
			if len(items) != tt.wantCount {
				t.Errorf("draft count = %d, want %d", len(items), tt.wantCount)
			}
		})
	}
}

// ============================================================================
// TestExecuteGetDraft
// ============================================================================

func TestExecuteGetDraft(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	tests := []struct {
		name      string
		args      map[string]any
		mockFn    func(ctx context.Context, grantID, draftID string) (*domain.Draft, error)
		wantError bool
		checkFn   func(t *testing.T, result map[string]any)
	}{
		{
			name: "happy path returns draft fields",
			args: map[string]any{"draft_id": "draft-1"},
			mockFn: func(_ context.Context, _, _ string) (*domain.Draft, error) {
				return &domain.Draft{
					ID:           "draft-1",
					Subject:      "Meeting notes",
					From:         []domain.EmailParticipant{{Email: "me@work.com", Name: "Me"}},
					To:           []domain.EmailParticipant{{Email: "you@work.com"}},
					Body:         "Here are the notes...",
					ReplyToMsgID: "orig-123",
				}, nil
			},
			checkFn: func(t *testing.T, result map[string]any) {
				t.Helper()
				if result["id"] != "draft-1" {
					t.Errorf("id = %v, want draft-1", result["id"])
				}
				if result["subject"] != "Meeting notes" {
					t.Errorf("subject = %v, want 'Meeting notes'", result["subject"])
				}
				if result["reply_to_message_id"] != "orig-123" {
					t.Errorf("reply_to_message_id = %v, want orig-123", result["reply_to_message_id"])
				}
			},
		},
		{
			name:      "missing draft_id returns error",
			args:      map[string]any{},
			wantError: true,
		},
		{
			name: "API error propagates",
			args: map[string]any{"draft_id": "draft-1"},
			mockFn: func(_ context.Context, _, _ string) (*domain.Draft, error) {
				return nil, errors.New("not found")
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			s := newMockServer(&mockNylasClient{getDraftFunc: tt.mockFn})
			resp := s.executeGetDraft(ctx, tt.args)
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
// TestExecuteCreateDraft
// ============================================================================

func TestExecuteCreateDraft(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	tests := []struct {
		name      string
		args      map[string]any
		mockFn    func(ctx context.Context, grantID string, req *domain.CreateDraftRequest) (*domain.Draft, error)
		wantError bool
		checkFn   func(t *testing.T, result map[string]any)
	}{
		{
			name: "happy path returns created status",
			args: map[string]any{
				"subject": "New Draft",
				"body":    "Draft body",
				"to":      []any{map[string]any{"email": "recipient@test.com"}},
			},
			mockFn: func(_ context.Context, _ string, _ *domain.CreateDraftRequest) (*domain.Draft, error) {
				return &domain.Draft{ID: "new-draft-1", Subject: "New Draft"}, nil
			},
			checkFn: func(t *testing.T, result map[string]any) {
				t.Helper()
				if result["id"] != "new-draft-1" {
					t.Errorf("id = %v, want new-draft-1", result["id"])
				}
				if result["status"] != "created" {
					t.Errorf("status = %v, want created", result["status"])
				}
			},
		},
		{
			name: "API error propagates",
			args: map[string]any{"subject": "Draft"},
			mockFn: func(_ context.Context, _ string, _ *domain.CreateDraftRequest) (*domain.Draft, error) {
				return nil, errors.New("create failed")
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			s := newMockServer(&mockNylasClient{createDraftFunc: tt.mockFn})
			resp := s.executeCreateDraft(ctx, tt.args)
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
// TestExecuteUpdateDraft
// ============================================================================

func TestExecuteUpdateDraft(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	tests := []struct {
		name      string
		args      map[string]any
		mockFn    func(ctx context.Context, grantID, draftID string, req *domain.CreateDraftRequest) (*domain.Draft, error)
		wantError bool
		checkFn   func(t *testing.T, result map[string]any)
	}{
		{
			name: "happy path returns updated status",
			args: map[string]any{"draft_id": "draft-42", "subject": "Updated subject"},
			mockFn: func(_ context.Context, _, _ string, _ *domain.CreateDraftRequest) (*domain.Draft, error) {
				return &domain.Draft{ID: "draft-42", Subject: "Updated subject"}, nil
			},
			checkFn: func(t *testing.T, result map[string]any) {
				t.Helper()
				if result["id"] != "draft-42" {
					t.Errorf("id = %v, want draft-42", result["id"])
				}
				if result["status"] != "updated" {
					t.Errorf("status = %v, want updated", result["status"])
				}
			},
		},
		{
			name:      "missing draft_id returns error",
			args:      map[string]any{"subject": "New subject"},
			wantError: true,
		},
		{
			name: "API error propagates",
			args: map[string]any{"draft_id": "draft-42"},
			mockFn: func(_ context.Context, _, _ string, _ *domain.CreateDraftRequest) (*domain.Draft, error) {
				return nil, errors.New("update failed")
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			s := newMockServer(&mockNylasClient{updateDraftFunc: tt.mockFn})
			resp := s.executeUpdateDraft(ctx, tt.args)
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
// TestExecuteSendDraft
// ============================================================================

func TestExecuteSendDraft(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	tests := []struct {
		name      string
		args      map[string]any
		mockFn    func(ctx context.Context, grantID, draftID string) (*domain.Message, error)
		wantError bool
		checkFn   func(t *testing.T, result map[string]any)
	}{
		{
			name: "happy path returns sent status",
			args: map[string]any{"draft_id": "draft-send-1"},
			mockFn: func(_ context.Context, _, _ string) (*domain.Message, error) {
				return &domain.Message{ID: "sent-msg-1", ThreadID: "thread-1"}, nil
			},
			checkFn: func(t *testing.T, result map[string]any) {
				t.Helper()
				if result["id"] != "sent-msg-1" {
					t.Errorf("id = %v, want sent-msg-1", result["id"])
				}
				if result["thread_id"] != "thread-1" {
					t.Errorf("thread_id = %v, want thread-1", result["thread_id"])
				}
				if result["status"] != "sent" {
					t.Errorf("status = %v, want sent", result["status"])
				}
			},
		},
		{
			name:      "missing draft_id returns error",
			args:      map[string]any{},
			wantError: true,
		},
		{
			name: "API error propagates",
			args: map[string]any{"draft_id": "draft-send-1"},
			mockFn: func(_ context.Context, _, _ string) (*domain.Message, error) {
				return nil, errors.New("send failed")
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			s := newMockServer(&mockNylasClient{sendDraftFunc: tt.mockFn})
			resp := s.executeSendDraft(ctx, tt.args)
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
// TestExecuteDeleteDraft
// ============================================================================

func TestExecuteDeleteDraft(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	tests := []struct {
		name      string
		args      map[string]any
		mockFn    func(ctx context.Context, grantID, draftID string) error
		wantError bool
	}{
		{
			name:   "happy path returns deleted status",
			args:   map[string]any{"draft_id": "draft-del-1"},
			mockFn: func(_ context.Context, _, _ string) error { return nil },
		},
		{
			name:      "missing draft_id returns error",
			args:      map[string]any{},
			wantError: true,
		},
		{
			name: "API error propagates",
			args: map[string]any{"draft_id": "draft-del-2"},
			mockFn: func(_ context.Context, _, _ string) error {
				return errors.New("delete failed")
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			s := newMockServer(&mockNylasClient{deleteDraftFunc: tt.mockFn})
			resp := s.executeDeleteDraft(ctx, tt.args)
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
			if !strings.Contains(text, "draft-del-1") {
				t.Errorf("response text = %q, want to contain 'draft-del-1'", text)
			}
		})
	}
}
