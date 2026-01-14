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

	payload := map[string]any{
		"subject": req.Subject,
		"body":    req.Body,
	}

	if len(req.To) > 0 {
		payload["to"] = convertContactsToAPI(req.To)
	}
	if len(req.Cc) > 0 {
		payload["cc"] = convertContactsToAPI(req.Cc)
	}
	if len(req.Bcc) > 0 {
		payload["bcc"] = convertContactsToAPI(req.Bcc)
	}
	if len(req.ReplyTo) > 0 {
		payload["reply_to"] = convertContactsToAPI(req.ReplyTo)
	}
	if req.ReplyToMsgID != "" {
		payload["reply_to_message_id"] = req.ReplyToMsgID
	}
	if len(req.Metadata) > 0 {
		payload["metadata"] = req.Metadata
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}
	httpReq, err := http.NewRequestWithContext(ctx, "PUT", queryURL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
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

// updateDraftWithMultipart updates a draft with attachments using multipart/form-data.
func (c *HTTPClient) updateDraftWithMultipart(ctx context.Context, grantID, draftID string, req *domain.CreateDraftRequest) (*domain.Draft, error) {
	queryURL := fmt.Sprintf("%s/v3/grants/%s/drafts/%s", c.baseURL, grantID, draftID)

	// Build the message JSON
	message := map[string]any{
		"subject": req.Subject,
		"body":    req.Body,
	}
	if len(req.To) > 0 {
		message["to"] = convertContactsToAPI(req.To)
	}
	if len(req.Cc) > 0 {
		message["cc"] = convertContactsToAPI(req.Cc)
	}
	if len(req.Bcc) > 0 {
		message["bcc"] = convertContactsToAPI(req.Bcc)
	}
	if len(req.ReplyTo) > 0 {
		message["reply_to"] = convertContactsToAPI(req.ReplyTo)
	}
	if req.ReplyToMsgID != "" {
		message["reply_to_message_id"] = req.ReplyToMsgID
	}
	if len(req.Metadata) > 0 {
		message["metadata"] = req.Metadata
	}

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
