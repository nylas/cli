package nylas

import (
	"context"
	"io"
	"strings"

	"github.com/nylas/cli/internal/domain"
)

func (m *MockClient) ListAttachments(ctx context.Context, grantID, messageID string) ([]domain.Attachment, error) {
	m.ListAttachmentsCalled = true
	m.LastGrantID = grantID
	m.LastMessageID = messageID
	if m.ListAttachmentsFunc != nil {
		return m.ListAttachmentsFunc(ctx, grantID, messageID)
	}
	return []domain.Attachment{
		{
			ID:          "attach-1",
			GrantID:     grantID,
			Filename:    "test.pdf",
			ContentType: "application/pdf",
			Size:        1024,
		},
		{
			ID:          "attach-2",
			GrantID:     grantID,
			Filename:    "image.png",
			ContentType: "image/png",
			Size:        2048,
		},
	}, nil
}

// GetAttachment retrieves attachment metadata.
func (m *MockClient) GetAttachment(ctx context.Context, grantID, messageID, attachmentID string) (*domain.Attachment, error) {
	m.GetAttachmentCalled = true
	m.LastGrantID = grantID
	m.LastAttachmentID = attachmentID
	if m.GetAttachmentFunc != nil {
		return m.GetAttachmentFunc(ctx, grantID, messageID, attachmentID)
	}
	return &domain.Attachment{
		ID:          attachmentID,
		GrantID:     grantID,
		Filename:    "test.pdf",
		ContentType: "application/pdf",
		Size:        1024,
	}, nil
}

// DownloadAttachment downloads attachment content.
func (m *MockClient) DownloadAttachment(ctx context.Context, grantID, messageID, attachmentID string) (io.ReadCloser, error) {
	m.DownloadAttachmentCalled = true
	m.LastGrantID = grantID
	m.LastAttachmentID = attachmentID
	if m.DownloadAttachmentFunc != nil {
		return m.DownloadAttachmentFunc(ctx, grantID, messageID, attachmentID)
	}
	return io.NopCloser(strings.NewReader("mock attachment content")), nil
}

// GetCalendars retrieves all calendars.
