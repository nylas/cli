package nylas

import (
	"context"
	"fmt"

	"github.com/nylas/cli/internal/domain"
)

func (m *MockClient) GetDrafts(ctx context.Context, grantID string, limit int) ([]domain.Draft, error) {
	m.GetDraftsCalled = true
	m.LastGrantID = grantID
	if m.GetDraftsFunc != nil {
		return m.GetDraftsFunc(ctx, grantID, limit)
	}
	return []domain.Draft{}, nil
}

// GetDraft retrieves a single draft.
func (m *MockClient) GetDraft(ctx context.Context, grantID, draftID string) (*domain.Draft, error) {
	m.GetDraftCalled = true
	m.LastGrantID = grantID
	m.LastDraftID = draftID
	if m.GetDraftFunc != nil {
		return m.GetDraftFunc(ctx, grantID, draftID)
	}
	return &domain.Draft{
		ID:      draftID,
		GrantID: grantID,
		Subject: "Test Draft",
	}, nil
}

// CreateDraft creates a new draft.
func (m *MockClient) CreateDraft(ctx context.Context, grantID string, req *domain.CreateDraftRequest) (*domain.Draft, error) {
	m.CreateDraftCalled = true
	m.LastGrantID = grantID
	if m.CreateDraftFunc != nil {
		return m.CreateDraftFunc(ctx, grantID, req)
	}

	// Convert request attachments to response attachments (with generated IDs)
	var attachments []domain.Attachment
	for i, a := range req.Attachments {
		attachments = append(attachments, domain.Attachment{
			ID:          fmt.Sprintf("attach-%d", i+1),
			Filename:    a.Filename,
			ContentType: a.ContentType,
			Size:        a.Size,
		})
	}

	return &domain.Draft{
		ID:          "new-draft-id",
		GrantID:     grantID,
		Subject:     req.Subject,
		Body:        req.Body,
		To:          req.To,
		Attachments: attachments,
	}, nil
}

// UpdateDraft updates an existing draft.
func (m *MockClient) UpdateDraft(ctx context.Context, grantID, draftID string, req *domain.CreateDraftRequest) (*domain.Draft, error) {
	m.UpdateDraftCalled = true
	m.LastGrantID = grantID
	m.LastDraftID = draftID
	if m.UpdateDraftFunc != nil {
		return m.UpdateDraftFunc(ctx, grantID, draftID, req)
	}

	// Convert request attachments to response attachments
	var attachments []domain.Attachment
	for i, a := range req.Attachments {
		attachments = append(attachments, domain.Attachment{
			ID:          fmt.Sprintf("attach-%d", i+1),
			Filename:    a.Filename,
			ContentType: a.ContentType,
			Size:        a.Size,
		})
	}

	return &domain.Draft{
		ID:          draftID,
		GrantID:     grantID,
		Subject:     req.Subject,
		Body:        req.Body,
		To:          req.To,
		Attachments: attachments,
	}, nil
}

// DeleteDraft deletes a draft.
func (m *MockClient) DeleteDraft(ctx context.Context, grantID, draftID string) error {
	m.DeleteDraftCalled = true
	m.LastGrantID = grantID
	m.LastDraftID = draftID
	if m.DeleteDraftFunc != nil {
		return m.DeleteDraftFunc(ctx, grantID, draftID)
	}
	return nil
}

// SendDraft sends a draft.
func (m *MockClient) SendDraft(ctx context.Context, grantID, draftID string) (*domain.Message, error) {
	m.SendDraftCalled = true
	m.LastGrantID = grantID
	m.LastDraftID = draftID
	if m.SendDraftFunc != nil {
		return m.SendDraftFunc(ctx, grantID, draftID)
	}
	return &domain.Message{
		ID:      "sent-from-draft-id",
		GrantID: grantID,
		Subject: "Sent Draft",
	}, nil
}

// GetFolders retrieves all folders.
