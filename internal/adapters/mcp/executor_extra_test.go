package mcp

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/nylas/cli/internal/domain"
)

// ============================================================================
// TestExecuteListThreads
// ============================================================================

func TestExecuteListThreads(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	now := time.Now().Truncate(time.Second)

	tests := []struct {
		name      string
		args      map[string]any
		mockFn    func(ctx context.Context, grantID string, params *domain.ThreadQueryParams) ([]domain.Thread, error)
		wantError bool
		wantCount int
	}{
		{
			name: "happy path returns threads",
			args: map[string]any{},
			mockFn: func(_ context.Context, _ string, _ *domain.ThreadQueryParams) ([]domain.Thread, error) {
				return []domain.Thread{
					{ID: "t1", Subject: "Hello", LatestMessageRecvDate: now},
					{ID: "t2", Subject: "World", LatestMessageRecvDate: now},
				}, nil
			},
			wantCount: 2,
		},
		{
			name: "API error propagates",
			args: map[string]any{},
			mockFn: func(_ context.Context, _ string, _ *domain.ThreadQueryParams) ([]domain.Thread, error) {
				return nil, errors.New("api down")
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			s := newMockServer(&mockNylasClient{getThreadsFunc: tt.mockFn})
			resp := s.executeListThreads(ctx, tt.args)
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
// TestExecuteGetThread
// ============================================================================

func TestExecuteGetThread(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	now := time.Now().Truncate(time.Second)

	tests := []struct {
		name      string
		args      map[string]any
		mockFn    func(ctx context.Context, grantID, threadID string) (*domain.Thread, error)
		wantError bool
		checkFn   func(t *testing.T, result map[string]any)
	}{
		{
			name: "happy path",
			args: map[string]any{"thread_id": "t1"},
			mockFn: func(_ context.Context, _, _ string) (*domain.Thread, error) {
				return &domain.Thread{ID: "t1", Subject: "Test", EarliestMessageDate: now, LatestMessageRecvDate: now}, nil
			},
			checkFn: func(t *testing.T, result map[string]any) {
				t.Helper()
				if result["id"] != "t1" {
					t.Errorf("id = %v, want t1", result["id"])
				}
			},
		},
		{
			name:      "missing thread_id",
			args:      map[string]any{},
			wantError: true,
		},
		{
			name: "API error propagates",
			args: map[string]any{"thread_id": "t1"},
			mockFn: func(_ context.Context, _, _ string) (*domain.Thread, error) {
				return nil, errors.New("not found")
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			s := newMockServer(&mockNylasClient{getThreadFunc: tt.mockFn})
			resp := s.executeGetThread(ctx, tt.args)
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
// TestExecuteUpdateThread
// ============================================================================

func TestExecuteUpdateThread(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	tests := []struct {
		name      string
		args      map[string]any
		mockFn    func(ctx context.Context, grantID, threadID string, req *domain.UpdateMessageRequest) (*domain.Thread, error)
		wantError bool
	}{
		{
			name: "happy path returns updated status",
			args: map[string]any{"thread_id": "t1"},
			mockFn: func(_ context.Context, _, _ string, _ *domain.UpdateMessageRequest) (*domain.Thread, error) {
				return &domain.Thread{ID: "t1"}, nil
			},
		},
		{
			name:      "missing thread_id",
			args:      map[string]any{},
			wantError: true,
		},
		{
			name: "API error propagates",
			args: map[string]any{"thread_id": "t1"},
			mockFn: func(_ context.Context, _, _ string, _ *domain.UpdateMessageRequest) (*domain.Thread, error) {
				return nil, errors.New("update failed")
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			s := newMockServer(&mockNylasClient{updateThreadFunc: tt.mockFn})
			resp := s.executeUpdateThread(ctx, tt.args)
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
// TestExecuteDeleteThread
// ============================================================================

func TestExecuteDeleteThread(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	tests := []struct {
		name      string
		args      map[string]any
		mockFn    func(ctx context.Context, grantID, threadID string) error
		wantError bool
	}{
		{
			name:   "happy path returns deleted status",
			args:   map[string]any{"thread_id": "t1"},
			mockFn: func(_ context.Context, _, _ string) error { return nil },
		},
		{
			name:      "missing thread_id",
			args:      map[string]any{},
			wantError: true,
		},
		{
			name: "API error propagates",
			args: map[string]any{"thread_id": "t1"},
			mockFn: func(_ context.Context, _, _ string) error {
				return errors.New("delete failed")
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			s := newMockServer(&mockNylasClient{deleteThreadFunc: tt.mockFn})
			resp := s.executeDeleteThread(ctx, tt.args)
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
			if result["thread_id"] != "t1" {
				t.Errorf("thread_id = %v, want t1", result["thread_id"])
			}
		})
	}
}
