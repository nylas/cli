package nylas

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"github.com/nylas/cli/internal/domain"
	"github.com/nylas/cli/internal/util"
)

// messageResponse represents an API message response.
type messageResponse struct {
	ID       string `json:"id"`
	GrantID  string `json:"grant_id"`
	ThreadID string `json:"thread_id"`
	Subject  string `json:"subject"`
	From     []struct {
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
	Body        string   `json:"body"`
	Snippet     string   `json:"snippet"`
	Date        int64    `json:"date"`
	Unread      bool     `json:"unread"`
	Starred     bool     `json:"starred"`
	Folders     []string `json:"folders"`
	Attachments []struct {
		ID          string `json:"id"`
		Filename    string `json:"filename"`
		ContentType string `json:"content_type"`
		Size        int64  `json:"size"`
		ContentID   string `json:"content_id"`
		IsInline    bool   `json:"is_inline"`
	} `json:"attachments"`
	Metadata map[string]string `json:"metadata"`
	RawMIME  string            `json:"raw_mime,omitempty"` // Base64url-encoded
	Headers  []struct {
		Name  string `json:"name"`
		Value string `json:"value"`
	} `json:"headers,omitempty"`
	CreatedAt int64  `json:"created_at"`
	Object    string `json:"object"`
}

// GetMessages retrieves recent messages for a grant (simple version).
func (c *HTTPClient) GetMessages(ctx context.Context, grantID string, limit int) ([]domain.Message, error) {
	if err := validateRequired("grant ID", grantID); err != nil {
		return nil, err
	}
	params := &domain.MessageQueryParams{Limit: limit}
	return c.GetMessagesWithParams(ctx, grantID, params)
}

// GetMessagesWithParams retrieves messages with query parameters.
func (c *HTTPClient) GetMessagesWithParams(ctx context.Context, grantID string, params *domain.MessageQueryParams) ([]domain.Message, error) {
	if err := validateRequired("grant ID", grantID); err != nil {
		return nil, err
	}
	resp, err := c.GetMessagesWithCursor(ctx, grantID, params)
	if err != nil {
		return nil, err
	}
	return resp.Data, nil
}

// GetMessagesWithCursor retrieves messages with pagination cursor support.
func (c *HTTPClient) GetMessagesWithCursor(ctx context.Context, grantID string, params *domain.MessageQueryParams) (*domain.MessageListResponse, error) {
	if err := validateRequired("grant ID", grantID); err != nil {
		return nil, err
	}
	if params == nil {
		params = &domain.MessageQueryParams{Limit: 10}
	}
	if params.Limit <= 0 {
		params.Limit = 10
	}

	baseURL := fmt.Sprintf("%s/v3/grants/%s/messages", c.baseURL, grantID)
	queryURL := NewQueryBuilder().
		AddInt("limit", params.Limit).
		Add("page_token", params.PageToken).
		AddInt("offset", params.Offset).
		Add("subject", params.Subject).
		Add("from", params.From).
		Add("to", params.To).
		Add("thread_id", params.ThreadID).
		AddBoolPtr("unread", params.Unread).
		AddBoolPtr("starred", params.Starred).
		AddBoolPtr("has_attachment", params.HasAttachment).
		AddInt64("received_before", params.ReceivedBefore).
		AddInt64("received_after", params.ReceivedAfter).
		Add("q", params.SearchQuery).
		AddSlice("in", params.In).
		Add("fields", params.Fields).
		Add("metadata_pair", params.MetadataPair).
		BuildURL(baseURL)

	var result struct {
		Data       []messageResponse `json:"data"`
		NextCursor string            `json:"next_cursor,omitempty"`
		RequestID  string            `json:"request_id,omitempty"`
	}
	if err := c.doGet(ctx, queryURL, &result); err != nil {
		return nil, err
	}

	return &domain.MessageListResponse{
		Data: convertMessages(result.Data),
		Pagination: domain.Pagination{
			NextCursor: result.NextCursor,
			HasMore:    result.NextCursor != "",
		},
	}, nil
}

// GetMessage retrieves a single message by ID.
func (c *HTTPClient) GetMessage(ctx context.Context, grantID, messageID string) (*domain.Message, error) {
	return c.GetMessageWithFields(ctx, grantID, messageID, "")
}

// GetMessageWithFields retrieves a message with optional field selection.
// Pass "raw_mime" to fields to retrieve the full RFC822/MIME message.
func (c *HTTPClient) GetMessageWithFields(ctx context.Context, grantID, messageID string, fields string) (*domain.Message, error) {
	if err := validateRequired("grant ID", grantID); err != nil {
		return nil, err
	}
	if err := validateRequired("message ID", messageID); err != nil {
		return nil, err
	}

	baseURL := fmt.Sprintf("%s/v3/grants/%s/messages/%s", c.baseURL, grantID, messageID)
	queryURL := NewQueryBuilder().Add("fields", fields).BuildURL(baseURL)

	var result struct {
		Data messageResponse `json:"data"`
	}
	if err := c.doGetWithNotFound(ctx, queryURL, &result, domain.ErrMessageNotFound); err != nil {
		return nil, err
	}

	msg := convertMessage(result.Data)
	return &msg, nil
}

// UpdateMessage updates message properties.
func (c *HTTPClient) UpdateMessage(ctx context.Context, grantID, messageID string, req *domain.UpdateMessageRequest) (*domain.Message, error) {
	queryURL := fmt.Sprintf("%s/v3/grants/%s/messages/%s", c.baseURL, grantID, messageID)

	payload := make(map[string]any, 3) // Pre-allocate for up to 3 fields
	if req.Unread != nil {
		payload["unread"] = *req.Unread
	}
	if req.Starred != nil {
		payload["starred"] = *req.Starred
	}
	if len(req.Folders) > 0 {
		payload["folders"] = req.Folders
	}

	resp, err := c.doJSONRequest(ctx, "PUT", queryURL, payload)
	if err != nil {
		return nil, err
	}

	var result struct {
		Data messageResponse `json:"data"`
	}
	if err := c.decodeJSONResponse(resp, &result); err != nil {
		return nil, err
	}

	msg := convertMessage(result.Data)
	return &msg, nil
}

// DeleteMessage deletes a message (moves to trash).
func (c *HTTPClient) DeleteMessage(ctx context.Context, grantID, messageID string) error {
	queryURL := fmt.Sprintf("%s/v3/grants/%s/messages/%s", c.baseURL, grantID, messageID)
	return c.doDelete(ctx, queryURL)
}

// convertMessages converts API message responses to domain models.
func convertMessages(msgs []messageResponse) []domain.Message {
	return util.Map(msgs, convertMessage)
}

// convertMessage converts an API message response to domain model.
func convertMessage(m messageResponse) domain.Message {
	convertParticipant := func(p struct {
		Name  string `json:"name"`
		Email string `json:"email"`
	}) domain.EmailParticipant {
		return domain.EmailParticipant{Name: p.Name, Email: p.Email}
	}
	convertAttachment := func(a struct {
		ID          string `json:"id"`
		Filename    string `json:"filename"`
		ContentType string `json:"content_type"`
		Size        int64  `json:"size"`
		ContentID   string `json:"content_id"`
		IsInline    bool   `json:"is_inline"`
	}) domain.Attachment {
		return domain.Attachment{
			ID:          a.ID,
			Filename:    a.Filename,
			ContentType: a.ContentType,
			Size:        a.Size,
			ContentID:   a.ContentID,
			IsInline:    a.IsInline,
		}
	}

	from := util.Map(m.From, convertParticipant)
	to := util.Map(m.To, convertParticipant)
	cc := util.Map(m.Cc, convertParticipant)
	bcc := util.Map(m.Bcc, convertParticipant)
	replyTo := util.Map(m.ReplyTo, convertParticipant)
	attachments := util.Map(m.Attachments, convertAttachment)

	// Convert headers
	headers := make([]domain.Header, 0, len(m.Headers))
	for _, h := range m.Headers {
		headers = append(headers, domain.Header{Name: h.Name, Value: h.Value})
	}

	// Decode raw MIME if present
	// Note: Nylas API may return different base64 variants:
	// - Standard base64 with padding (production API)
	// - Base64url without padding (some contexts)
	// We try multiple decoders to handle all cases.
	rawMIME := ""
	if m.RawMIME != "" {
		var decoded []byte
		var err error

		// First, normalize URL-safe characters to standard base64
		normalized := strings.ReplaceAll(m.RawMIME, "-", "+")
		normalized = strings.ReplaceAll(normalized, "_", "/")

		// Try standard encoding with padding first
		decoded, err = base64.StdEncoding.DecodeString(normalized)
		if err != nil {
			// Try without padding (RawStdEncoding)
			decoded, err = base64.RawStdEncoding.DecodeString(normalized)
		}
		if err == nil {
			rawMIME = string(decoded)
		}
		// If decode fails, rawMIME stays empty (graceful degradation)
	}

	return domain.Message{
		ID:          m.ID,
		GrantID:     m.GrantID,
		ThreadID:    m.ThreadID,
		Subject:     m.Subject,
		From:        from,
		To:          to,
		Cc:          cc,
		Bcc:         bcc,
		ReplyTo:     replyTo,
		Body:        m.Body,
		Snippet:     m.Snippet,
		Date:        time.Unix(m.Date, 0),
		Unread:      m.Unread,
		Starred:     m.Starred,
		Folders:     m.Folders,
		Attachments: attachments,
		Headers:     headers,
		RawMIME:     rawMIME,
		Metadata:    m.Metadata,
		CreatedAt:   time.Unix(m.CreatedAt, 0),
		Object:      m.Object,
	}
}
