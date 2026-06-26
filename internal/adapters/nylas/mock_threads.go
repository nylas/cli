package nylas

import (
	"context"

	"github.com/nylas/cli/internal/domain"
)

func (m *MockClient) GetThreads(ctx context.Context, grantID string, params *domain.ThreadQueryParams) ([]domain.Thread, error) {
	m.GetThreadsCalled = true
	m.LastGrantID = grantID
	if m.GetThreadsFunc != nil {
		return m.GetThreadsFunc(ctx, grantID, params)
	}
	return []domain.Thread{}, nil
}

// GetThreadsWithCursor retrieves threads with pagination cursor support.
func (m *MockClient) GetThreadsWithCursor(ctx context.Context, grantID string, params *domain.ThreadQueryParams) (*domain.ThreadListResponse, error) {
	m.GetThreadsCalled = true
	m.LastGrantID = grantID
	if m.GetThreadsWithCursorFunc != nil {
		return m.GetThreadsWithCursorFunc(ctx, grantID, params)
	}
	if m.GetThreadsFunc != nil {
		threads, err := m.GetThreadsFunc(ctx, grantID, params)
		return &domain.ThreadListResponse{Data: threads}, err
	}
	return &domain.ThreadListResponse{Data: []domain.Thread{}}, nil
}

// GetThread retrieves a single thread.
func (m *MockClient) GetThread(ctx context.Context, grantID, threadID string) (*domain.Thread, error) {
	m.GetThreadCalled = true
	m.LastGrantID = grantID
	m.LastThreadID = threadID
	if m.GetThreadFunc != nil {
		return m.GetThreadFunc(ctx, grantID, threadID)
	}
	return &domain.Thread{
		ID:      threadID,
		GrantID: grantID,
		Subject: "Test Thread",
	}, nil
}

// UpdateThread updates thread properties.
func (m *MockClient) UpdateThread(ctx context.Context, grantID, threadID string, req *domain.UpdateMessageRequest) (*domain.Thread, error) {
	m.UpdateThreadCalled = true
	m.LastGrantID = grantID
	m.LastThreadID = threadID
	if m.UpdateThreadFunc != nil {
		return m.UpdateThreadFunc(ctx, grantID, threadID, req)
	}
	thread := &domain.Thread{
		ID:      threadID,
		GrantID: grantID,
		Subject: "Updated Thread",
	}
	if req.Unread != nil {
		thread.Unread = *req.Unread
	}
	if req.Starred != nil {
		thread.Starred = *req.Starred
	}
	return thread, nil
}

// DeleteThread deletes a thread.
func (m *MockClient) DeleteThread(ctx context.Context, grantID, threadID string) error {
	m.DeleteThreadCalled = true
	m.LastGrantID = grantID
	m.LastThreadID = threadID
	if m.DeleteThreadFunc != nil {
		return m.DeleteThreadFunc(ctx, grantID, threadID)
	}
	return nil
}

// GetDrafts retrieves drafts.
