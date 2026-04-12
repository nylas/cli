package nylas

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/textproto"

	"github.com/nylas/cli/internal/domain"
)

// UpdateDraft updates an existing draft.
func (c *HTTPClient) UpdateDraft(ctx context.Context, grantID, draftID string, req *domain.CreateDraftRequest) (*domain.Draft, error) {
	// If there are attachments, use multipart; otherwise use JSON
	if len(req.Attachments) > 0 {
		return c.updateDraftWithMultipart(ctx, grantID, draftID, req)
	}
	return c.updateDraftWithJSON(ctx, grantID, draftID, req)
}

// updateDraftWithJSON updates a draft using JSON encoding (no attachments).
func (c *HTTPClient) updateDraftWithJSON(ctx context.Context, grantID, draftID string, req *domain.CreateDraftRequest) (*domain.Draft, error) {
	queryURL := fmt.Sprintf("%s/v3/grants/%s/drafts/%s", c.baseURL, grantID, draftID)

	resp, err := c.doJSONRequest(ctx, "PUT", queryURL, buildDraftPayload(req, false), http.StatusOK)
	if err != nil {
		return nil, err
	}

	var result struct {
		Data draftResponse `json:"data"`
	}
	if err := c.decodeJSONResponse(resp, &result); err != nil {
		return nil, err
	}

	draft := convertDraft(result.Data)
	return &draft, nil
}

// updateDraftWithMultipart updates a draft with attachments using multipart/form-data.
func (c *HTTPClient) updateDraftWithMultipart(ctx context.Context, grantID, draftID string, req *domain.CreateDraftRequest) (*domain.Draft, error) {
	queryURL := fmt.Sprintf("%s/v3/grants/%s/drafts/%s", c.baseURL, grantID, draftID)

	// Build the message JSON
	message := buildDraftPayload(req, false)

	// Create multipart form
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	// Add message as JSON field
	messageJSON, err := json.Marshal(message)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal message: %w", err)
	}
	if err := writer.WriteField("message", string(messageJSON)); err != nil {
		return nil, fmt.Errorf("failed to write message field: %w", err)
	}

	// Add each attachment as a file
	for i, att := range req.Attachments {
		if len(att.Content) == 0 {
			continue // Skip attachments without content
		}

		// Create form file with proper headers
		h := make(textproto.MIMEHeader)
		h.Set("Content-Disposition", fmt.Sprintf(`form-data; name="file%d"; filename="%s"`, i, att.Filename))
		if att.ContentType != "" {
			h.Set("Content-Type", att.ContentType)
		} else {
			h.Set("Content-Type", "application/octet-stream")
		}

		part, err := writer.CreatePart(h)
		if err != nil {
			return nil, fmt.Errorf("failed to create attachment part: %w", err)
		}
		if _, err := part.Write(att.Content); err != nil {
			return nil, fmt.Errorf("failed to write attachment content: %w", err)
		}
	}

	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("failed to close multipart writer: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "PUT", queryURL, &buf)
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", writer.FormDataContentType())
	c.setAuthHeader(httpReq)

	resp, err := c.doRequest(ctx, httpReq)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", domain.ErrNetworkError, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, c.parseError(resp)
	}

	var result struct {
		Data draftResponse `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	draft := convertDraft(result.Data)
	return &draft, nil
}
