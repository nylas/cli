package mcp

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/nylas/cli/internal/domain"
)

// ============================================================================
// TestExecuteGetFolder
// ============================================================================

func TestExecuteGetFolder(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	tests := []struct {
		name      string
		args      map[string]any
		mockFn    func(ctx context.Context, grantID, folderID string) (*domain.Folder, error)
		wantError bool
		checkFn   func(t *testing.T, result map[string]any)
	}{
		{
			name: "happy path",
			args: map[string]any{"folder_id": "f1"},
			mockFn: func(_ context.Context, _, _ string) (*domain.Folder, error) {
				return &domain.Folder{ID: "f1", Name: "Inbox", TotalCount: 5, UnreadCount: 2}, nil
			},
			checkFn: func(t *testing.T, result map[string]any) {
				t.Helper()
				if result["id"] != "f1" {
					t.Errorf("id = %v, want f1", result["id"])
				}
				if result["name"] != "Inbox" {
					t.Errorf("name = %v, want Inbox", result["name"])
				}
			},
		},
		{
			name:      "missing folder_id",
			args:      map[string]any{},
			wantError: true,
		},
		{
			name: "API error propagates",
			args: map[string]any{"folder_id": "f1"},
			mockFn: func(_ context.Context, _, _ string) (*domain.Folder, error) {
				return nil, errors.New("not found")
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			s := newMockServer(&mockNylasClient{getFolderFunc: tt.mockFn})
			resp := s.executeGetFolder(ctx, tt.args)
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
// TestExecuteCreateFolder
// ============================================================================

func TestExecuteCreateFolder(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	tests := []struct {
		name      string
		args      map[string]any
		mockFn    func(ctx context.Context, grantID string, req *domain.CreateFolderRequest) (*domain.Folder, error)
		wantError bool
	}{
		{
			name: "happy path returns created status",
			args: map[string]any{"name": "Projects"},
			mockFn: func(_ context.Context, _ string, _ *domain.CreateFolderRequest) (*domain.Folder, error) {
				return &domain.Folder{ID: "f2", Name: "Projects"}, nil
			},
		},
		{
			name:      "missing name",
			args:      map[string]any{},
			wantError: true,
		},
		{
			name: "API error propagates",
			args: map[string]any{"name": "Projects"},
			mockFn: func(_ context.Context, _ string, _ *domain.CreateFolderRequest) (*domain.Folder, error) {
				return nil, errors.New("create failed")
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			s := newMockServer(&mockNylasClient{createFolderFunc: tt.mockFn})
			resp := s.executeCreateFolder(ctx, tt.args)
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
			if result["status"] != "created" {
				t.Errorf("status = %v, want created", result["status"])
			}
		})
	}
}

// ============================================================================
// TestExecuteUpdateFolder
// ============================================================================

func TestExecuteUpdateFolder(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	tests := []struct {
		name      string
		args      map[string]any
		mockFn    func(ctx context.Context, grantID, folderID string, req *domain.UpdateFolderRequest) (*domain.Folder, error)
		wantError bool
	}{
		{
			name: "happy path returns updated status",
			args: map[string]any{"folder_id": "f1", "name": "Renamed"},
			mockFn: func(_ context.Context, _, _ string, _ *domain.UpdateFolderRequest) (*domain.Folder, error) {
				return &domain.Folder{ID: "f1", Name: "Renamed"}, nil
			},
		},
		{
			name:      "missing folder_id",
			args:      map[string]any{},
			wantError: true,
		},
		{
			name: "API error propagates",
			args: map[string]any{"folder_id": "f1"},
			mockFn: func(_ context.Context, _, _ string, _ *domain.UpdateFolderRequest) (*domain.Folder, error) {
				return nil, errors.New("update failed")
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			s := newMockServer(&mockNylasClient{updateFolderFunc: tt.mockFn})
			resp := s.executeUpdateFolder(ctx, tt.args)
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
// TestExecuteDeleteFolder
// ============================================================================

func TestExecuteDeleteFolder(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	tests := []struct {
		name      string
		args      map[string]any
		mockFn    func(ctx context.Context, grantID, folderID string) error
		wantError bool
	}{
		{
			name:   "happy path returns deleted status",
			args:   map[string]any{"folder_id": "f1"},
			mockFn: func(_ context.Context, _, _ string) error { return nil },
		},
		{
			name:      "missing folder_id",
			args:      map[string]any{},
			wantError: true,
		},
		{
			name: "API error propagates",
			args: map[string]any{"folder_id": "f1"},
			mockFn: func(_ context.Context, _, _ string) error {
				return errors.New("delete failed")
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			s := newMockServer(&mockNylasClient{deleteFolderFunc: tt.mockFn})
			resp := s.executeDeleteFolder(ctx, tt.args)
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
			if !strings.Contains(text, "f1") {
				t.Errorf("response text = %q, want to contain 'f1'", text)
			}
		})
	}
}

// ============================================================================
// TestExecuteListAttachments
// ============================================================================

func TestExecuteListAttachments(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	tests := []struct {
		name      string
		args      map[string]any
		mockFn    func(ctx context.Context, grantID, messageID string) ([]domain.Attachment, error)
		wantError bool
		wantCount int
	}{
		{
			name: "happy path returns attachments",
			args: map[string]any{"message_id": "msg1"},
			mockFn: func(_ context.Context, _, _ string) ([]domain.Attachment, error) {
				return []domain.Attachment{
					{ID: "att1", Filename: "doc.pdf", ContentType: "application/pdf", Size: 1024},
					{ID: "att2", Filename: "img.png", ContentType: "image/png", Size: 2048, IsInline: true},
				}, nil
			},
			wantCount: 2,
		},
		{
			name:      "missing message_id",
			args:      map[string]any{},
			wantError: true,
		},
		{
			name: "API error propagates",
			args: map[string]any{"message_id": "msg1"},
			mockFn: func(_ context.Context, _, _ string) ([]domain.Attachment, error) {
				return nil, errors.New("api error")
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			s := newMockServer(&mockNylasClient{listAttachmentsFunc: tt.mockFn})
			resp := s.executeListAttachments(ctx, tt.args)
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
				t.Errorf("item count = %d, want %d", len(items), tt.wantCount)
			}
		})
	}
}

// ============================================================================
// TestExecuteGetAttachment
// ============================================================================

func TestExecuteGetAttachment(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	tests := []struct {
		name      string
		args      map[string]any
		mockFn    func(ctx context.Context, grantID, messageID, attachmentID string) (*domain.Attachment, error)
		wantError bool
		checkFn   func(t *testing.T, result map[string]any)
	}{
		{
			name: "happy path",
			args: map[string]any{"message_id": "msg1", "attachment_id": "att1"},
			mockFn: func(_ context.Context, _, _, _ string) (*domain.Attachment, error) {
				return &domain.Attachment{ID: "att1", Filename: "doc.pdf", ContentType: "application/pdf", Size: 1024}, nil
			},
			checkFn: func(t *testing.T, result map[string]any) {
				t.Helper()
				if result["id"] != "att1" {
					t.Errorf("id = %v, want att1", result["id"])
				}
				if result["filename"] != "doc.pdf" {
					t.Errorf("filename = %v, want doc.pdf", result["filename"])
				}
			},
		},
		{
			name:      "missing message_id",
			args:      map[string]any{"attachment_id": "att1"},
			wantError: true,
		},
		{
			name:      "missing attachment_id",
			args:      map[string]any{"message_id": "msg1"},
			wantError: true,
		},
		{
			name: "API error propagates",
			args: map[string]any{"message_id": "msg1", "attachment_id": "att1"},
			mockFn: func(_ context.Context, _, _, _ string) (*domain.Attachment, error) {
				return nil, errors.New("not found")
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			s := newMockServer(&mockNylasClient{getAttachmentFunc: tt.mockFn})
			resp := s.executeGetAttachment(ctx, tt.args)
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
// TestExecuteListScheduledMessages
// ============================================================================

func TestExecuteListScheduledMessages(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	tests := []struct {
		name      string
		mockFn    func(ctx context.Context, grantID string) ([]domain.ScheduledMessage, error)
		wantError bool
		wantCount int
	}{
		{
			name: "happy path returns scheduled messages",
			mockFn: func(_ context.Context, _ string) ([]domain.ScheduledMessage, error) {
				return []domain.ScheduledMessage{
					{ScheduleID: "sched1", Status: "pending", CloseTime: 1700000000},
					{ScheduleID: "sched2", Status: "scheduled", CloseTime: 1700001000},
				}, nil
			},
			wantCount: 2,
		},
		{
			name: "API error propagates",
			mockFn: func(_ context.Context, _ string) ([]domain.ScheduledMessage, error) {
				return nil, errors.New("api error")
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			s := newMockServer(&mockNylasClient{listScheduledMessagesFunc: tt.mockFn})
			resp := s.executeListScheduledMessages(ctx, map[string]any{})
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
				t.Errorf("item count = %d, want %d", len(items), tt.wantCount)
			}
		})
	}
}

// ============================================================================
// TestExecuteCancelScheduledMessage
// ============================================================================

func TestExecuteCancelScheduledMessage(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	tests := []struct {
		name      string
		args      map[string]any
		mockFn    func(ctx context.Context, grantID, scheduleID string) error
		wantError bool
	}{
		{
			name:   "happy path returns cancelled status",
			args:   map[string]any{"schedule_id": "sched1"},
			mockFn: func(_ context.Context, _, _ string) error { return nil },
		},
		{
			name:      "missing schedule_id",
			args:      map[string]any{},
			wantError: true,
		},
		{
			name: "API error propagates",
			args: map[string]any{"schedule_id": "sched1"},
			mockFn: func(_ context.Context, _, _ string) error {
				return errors.New("cancel failed")
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			s := newMockServer(&mockNylasClient{cancelScheduledMessageFunc: tt.mockFn})
			resp := s.executeCancelScheduledMessage(ctx, tt.args)
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
			if !strings.Contains(text, "Cancelled") {
				t.Errorf("response text = %q, want to contain 'Cancelled'", text)
			}
			if !strings.Contains(text, "sched1") {
				t.Errorf("response text = %q, want to contain 'sched1'", text)
			}
		})
	}
}
