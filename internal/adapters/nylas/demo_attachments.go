package nylas

import (
	"context"
	"io"
	"strings"

	"github.com/nylas/cli/internal/domain"
)

func (d *DemoClient) ListAttachments(ctx context.Context, grantID, messageID string) ([]domain.Attachment, error) {
	return []domain.Attachment{
		{
			ID:          "attach-001",
			Filename:    "quarterly-report.pdf",
			ContentType: "application/pdf",
			Size:        245760,
		},
		{
			ID:          "attach-002",
			Filename:    "presentation.pptx",
			ContentType: "application/vnd.openxmlformats-officedocument.presentationml.presentation",
			Size:        1048576,
		},
	}, nil
}

// GetAttachment returns demo attachment metadata.
func (d *DemoClient) GetAttachment(ctx context.Context, grantID, messageID, attachmentID string) (*domain.Attachment, error) {
	return &domain.Attachment{
		ID:          attachmentID,
		Filename:    "quarterly-report.pdf",
		ContentType: "application/pdf",
		Size:        245760,
	}, nil
}

// DownloadAttachment returns mock attachment content.
func (d *DemoClient) DownloadAttachment(ctx context.Context, grantID, messageID, attachmentID string) (io.ReadCloser, error) {
	return io.NopCloser(strings.NewReader("demo attachment content")), nil
}
