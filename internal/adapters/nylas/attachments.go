package nylas

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/nylas/cli/internal/domain"
)

// attachmentResponse represents an API attachment response.
type attachmentResponse struct {
	ID          string `json:"id"`
	GrantID     string `json:"grant_id"`
	Filename    string `json:"filename"`
	ContentType string `json:"content_type"`
	Size        int64  `json:"size"`
	ContentID   string `json:"content_id"`
	IsInline    bool   `json:"is_inline"`
}

// GetAttachment retrieves attachment metadata.
func (c *HTTPClient) GetAttachment(ctx context.Context, grantID, messageID, attachmentID string) (*domain.Attachment, error) {
	queryURL := fmt.Sprintf("%s/v3/grants/%s/messages/%s/attachments/%s", c.baseURL, grantID, messageID, attachmentID)

	req, err := http.NewRequestWithContext(ctx, "GET", queryURL, nil)
	if err != nil {
		return nil, err
	}
	c.setAuthHeader(req)

	resp, err := c.doRequest(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", domain.ErrNetworkError, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("%w: attachment not found", domain.ErrAPIError)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, c.parseError(resp)
	}

	var result struct {
		Data attachmentResponse `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &domain.Attachment{
		ID:          result.Data.ID,
		GrantID:     result.Data.GrantID,
		Filename:    result.Data.Filename,
		ContentType: result.Data.ContentType,
		Size:        result.Data.Size,
		ContentID:   result.Data.ContentID,
		IsInline:    result.Data.IsInline,
	}, nil
}

// DownloadAttachment downloads attachment content.
func (c *HTTPClient) DownloadAttachment(ctx context.Context, grantID, messageID, attachmentID string) (io.ReadCloser, error) {
	queryURL := fmt.Sprintf("%s/v3/grants/%s/messages/%s/attachments/%s/download", c.baseURL, grantID, messageID, attachmentID)

	req, err := http.NewRequestWithContext(ctx, "GET", queryURL, nil)
	if err != nil {
		return nil, err
	}
	c.setAuthHeader(req)

	resp, err := c.doRequest(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", domain.ErrNetworkError, err)
	}

	if resp.StatusCode == http.StatusNotFound {
		_ = resp.Body.Close()
		return nil, fmt.Errorf("%w: attachment not found", domain.ErrAPIError)
	}
	if resp.StatusCode != http.StatusOK {
		defer func() { _ = resp.Body.Close() }()
		return nil, c.parseError(resp)
	}

	// Return the body directly - caller is responsible for closing
	return resp.Body, nil
}

// ListAttachments retrieves all attachments for a message.
// This is a convenience method that fetches the message and extracts attachments.
func (c *HTTPClient) ListAttachments(ctx context.Context, grantID, messageID string) ([]domain.Attachment, error) {
	msg, err := c.GetMessage(ctx, grantID, messageID)
	if err != nil {
		return nil, err
	}
	return msg.Attachments, nil
}
