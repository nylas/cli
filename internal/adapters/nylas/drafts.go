package nylas

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"time"

	"github.com/nylas/cli/internal/domain"
	"github.com/nylas/cli/internal/util"
)

// draftResponse represents an API draft response.
type draftResponse struct {
	ID      string `json:"id"`
	GrantID string `json:"grant_id"`
	Subject string `json:"subject"`
	Body    string `json:"body"`
	From    []struct {
		Name  string `json:"name"`
		Email string `json:"email"`
	} `json:"from"`
	To []struct {
		Name  string `json:"name"`
		Email string `json:"email"`
	} `json:"to"`
	Cc []struct {
		Name  string `json:"name"`
		Email string `json:"email"`
	} `json:"cc"`
	Bcc []struct {
		Name  string `json:"name"`
		Email string `json:"email"`
	} `json:"bcc"`
	ReplyTo []struct {
		Name  string `json:"name"`
		Email string `json:"email"`
	} `json:"reply_to"`
	ReplyToMsgID string `json:"reply_to_message_id"`
	ThreadID     string `json:"thread_id"`
	Attachments  []struct {
		ID          string `json:"id"`
		Filename    string `json:"filename"`
		ContentType string `json:"content_type"`
		Size        int64  `json:"size"`
	} `json:"attachments"`
	CreatedAt int64 `json:"created_at"`
	UpdatedAt int64 `json:"updated_at"`
}

// GetDrafts retrieves drafts for a grant.
func (c *HTTPClient) GetDrafts(ctx context.Context, grantID string, limit int) ([]domain.Draft, error) {
	if limit <= 0 {
		limit = 10
	}

	baseURL := fmt.Sprintf("%s/v3/grants/%s/drafts", c.baseURL, grantID)
	queryURL := NewQueryBuilder().AddInt("limit", limit).BuildURL(baseURL)

	var result struct {
		Data []draftResponse `json:"data"`
	}
	if err := c.doGet(ctx, queryURL, &result); err != nil {
		return nil, err
	}

	return convertDrafts(result.Data), nil
}

// GetDraft retrieves a single draft by ID.
func (c *HTTPClient) GetDraft(ctx context.Context, grantID, draftID string) (*domain.Draft, error) {
	queryURL := fmt.Sprintf("%s/v3/grants/%s/drafts/%s", c.baseURL, grantID, draftID)

	var result struct {
		Data draftResponse `json:"data"`
	}
	if err := c.doGetWithNotFound(ctx, queryURL, &result, domain.ErrDraftNotFound); err != nil {
		return nil, err
	}

	draft := convertDraft(result.Data)
	return &draft, nil
}

// CreateDraft creates a new draft.
func (c *HTTPClient) CreateDraft(ctx context.Context, grantID string, req *domain.CreateDraftRequest) (*domain.Draft, error) {
	// If there are attachments, use multipart; otherwise use JSON
	if len(req.Attachments) > 0 {
		return c.createDraftWithMultipart(ctx, grantID, req)
	}
	return c.createDraftWithJSON(ctx, grantID, req)
}

// buildDraftPayload builds the common payload for draft creation requests.
// This consolidates the repeated payload building logic across draft creation methods.
func buildDraftPayload(req *domain.CreateDraftRequest, includeSignature bool) map[string]any {
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
	if includeSignature && req.SignatureID != "" {
		payload["signature_id"] = req.SignatureID
	}
	if len(req.Metadata) > 0 {
		payload["metadata"] = req.Metadata
	}
	return payload
}

// createDraftWithJSON creates a draft using JSON encoding (no attachments or small attachments).
func (c *HTTPClient) createDraftWithJSON(ctx context.Context, grantID string, req *domain.CreateDraftRequest) (*domain.Draft, error) {
	queryURL := fmt.Sprintf("%s/v3/grants/%s/drafts", c.baseURL, grantID)

	resp, err := c.doJSONRequest(ctx, "POST", queryURL, buildDraftPayload(req, true))
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

// createDraftWithMultipart creates a draft with attachments using multipart/form-data.
func (c *HTTPClient) createDraftWithMultipart(ctx context.Context, grantID string, req *domain.CreateDraftRequest) (*domain.Draft, error) {
	queryURL := fmt.Sprintf("%s/v3/grants/%s/drafts", c.baseURL, grantID)

	// Create multipart form
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	// Add message as JSON field
	messageJSON, err := json.Marshal(buildDraftPayload(req, true))
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

	httpReq, err := http.NewRequestWithContext(ctx, "POST", queryURL, &buf)
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

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
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

// CreateDraftWithAttachmentFromReader creates a draft with an attachment from an io.Reader.
// This is useful for large attachments or streaming file uploads.
func (c *HTTPClient) CreateDraftWithAttachmentFromReader(ctx context.Context, grantID string, req *domain.CreateDraftRequest, filename string, contentType string, reader io.Reader) (*domain.Draft, error) {
	queryURL := fmt.Sprintf("%s/v3/grants/%s/drafts", c.baseURL, grantID)
	payload := buildDraftPayload(req, true)

	// Use pipe to stream multipart data
	pr, pw := io.Pipe()
	writer := multipart.NewWriter(pw)

	// Write multipart in a goroutine
	errCh := make(chan error, 1)
	go func() {
		defer func() { _ = pw.Close() }()
		defer func() { _ = writer.Close() }()

		// Add message as JSON field
		messageJSON, err := json.Marshal(payload)
		if err != nil {
			errCh <- fmt.Errorf("failed to marshal message: %w", err)
			return
		}
		if err := writer.WriteField("message", string(messageJSON)); err != nil {
			errCh <- fmt.Errorf("failed to write message field: %w", err)
			return
		}

		// Create form file with proper headers
		h := make(textproto.MIMEHeader)
		h.Set("Content-Disposition", fmt.Sprintf(`form-data; name="file"; filename="%s"`, filename))
		if contentType != "" {
			h.Set("Content-Type", contentType)
		} else {
			h.Set("Content-Type", "application/octet-stream")
		}

		part, err := writer.CreatePart(h)
		if err != nil {
			errCh <- fmt.Errorf("failed to create attachment part: %w", err)
			return
		}
		if _, err := io.Copy(part, reader); err != nil {
			errCh <- fmt.Errorf("failed to copy attachment content: %w", err)
			return
		}

		errCh <- nil
	}()

	httpReq, err := http.NewRequestWithContext(ctx, "POST", queryURL, pr)
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

	// Wait for writer goroutine to finish
	if writerErr := <-errCh; writerErr != nil {
		return nil, writerErr
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
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

// DeleteDraft deletes a draft.
func (c *HTTPClient) DeleteDraft(ctx context.Context, grantID, draftID string) error {
	queryURL := fmt.Sprintf("%s/v3/grants/%s/drafts/%s", c.baseURL, grantID, draftID)
	return c.doDelete(ctx, queryURL)
}

// SendDraft sends a draft.
func (c *HTTPClient) SendDraft(ctx context.Context, grantID, draftID string, req *domain.SendDraftRequest) (*domain.Message, error) {
	queryURL := fmt.Sprintf("%s/v3/grants/%s/drafts/%s", c.baseURL, grantID, draftID)

	var bodyReader io.Reader
	if req != nil && req.SignatureID != "" {
		body, err := json.Marshal(map[string]string{"signature_id": req.SignatureID})
		if err != nil {
			return nil, fmt.Errorf("failed to marshal send draft request: %w", err)
		}
		bodyReader = bytes.NewReader(body)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", queryURL, bodyReader)
	if err != nil {
		return nil, err
	}
	c.setAuthHeader(httpReq)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.doRequest(ctx, httpReq)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", domain.ErrNetworkError, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, c.parseError(resp)
	}

	var result struct {
		Data messageResponse `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	msg := convertMessage(result.Data)
	return &msg, nil
}

// convertDrafts converts API draft responses to domain models.
func convertDrafts(drafts []draftResponse) []domain.Draft {
	return util.Map(drafts, convertDraft)
}

// convertDraft converts an API draft response to domain model.
func convertDraft(d draftResponse) domain.Draft {
	// Helper to convert participant structs
	convertParticipant := func(p struct {
		Name  string `json:"name"`
		Email string `json:"email"`
	}) domain.EmailParticipant {
		return domain.EmailParticipant{Name: p.Name, Email: p.Email}
	}
	// Helper to convert attachment structs
	convertAttachment := func(a struct {
		ID          string `json:"id"`
		Filename    string `json:"filename"`
		ContentType string `json:"content_type"`
		Size        int64  `json:"size"`
	}) domain.Attachment {
		return domain.Attachment{
			ID:          a.ID,
			Filename:    a.Filename,
			ContentType: a.ContentType,
			Size:        a.Size,
		}
	}

	return domain.Draft{
		ID:           d.ID,
		GrantID:      d.GrantID,
		Subject:      d.Subject,
		Body:         d.Body,
		From:         util.Map(d.From, convertParticipant),
		To:           util.Map(d.To, convertParticipant),
		Cc:           util.Map(d.Cc, convertParticipant),
		Bcc:          util.Map(d.Bcc, convertParticipant),
		ReplyTo:      util.Map(d.ReplyTo, convertParticipant),
		ReplyToMsgID: d.ReplyToMsgID,
		ThreadID:     d.ThreadID,
		Attachments:  util.Map(d.Attachments, convertAttachment),
		CreatedAt:    time.Unix(d.CreatedAt, 0),
		UpdatedAt:    time.Unix(d.UpdatedAt, 0),
	}
}
