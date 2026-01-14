package nylas

import (
	"context"

	"github.com/nylas/cli/internal/domain"
)

// GetDrafts returns demo drafts.
func (d *DemoClient) GetDrafts(ctx context.Context, grantID string, limit int) ([]domain.Draft, error) {
	return []domain.Draft{
		{
			ID:      "draft-001",
			Subject: "Re: Project proposal",
			To:      []domain.EmailParticipant{{Email: "client@company.com"}},
			Body:    "Thank you for the proposal...",
		},
	}, nil
}

// GetDraft returns a demo draft.
func (d *DemoClient) GetDraft(ctx context.Context, grantID, draftID string) (*domain.Draft, error) {
	return &domain.Draft{
		ID:      draftID,
		Subject: "Re: Project proposal",
		Body:    "Thank you for the proposal...",
	}, nil
}

// CreateDraft simulates creating a draft.
func (d *DemoClient) CreateDraft(ctx context.Context, grantID string, req *domain.CreateDraftRequest) (*domain.Draft, error) {
	return &domain.Draft{ID: "new-draft", Subject: req.Subject, Body: req.Body, To: req.To}, nil
}

// UpdateDraft simulates updating a draft.
func (d *DemoClient) UpdateDraft(ctx context.Context, grantID, draftID string, req *domain.CreateDraftRequest) (*domain.Draft, error) {
	return &domain.Draft{ID: draftID, Subject: req.Subject, Body: req.Body, To: req.To}, nil
}

// DeleteDraft simulates deleting a draft.
func (d *DemoClient) DeleteDraft(ctx context.Context, grantID, draftID string) error {
	return nil
}

// SendDraft simulates sending a draft.
func (d *DemoClient) SendDraft(ctx context.Context, grantID, draftID string) (*domain.Message, error) {
	return &domain.Message{ID: "sent-from-draft", Subject: "Sent Draft"}, nil
}
