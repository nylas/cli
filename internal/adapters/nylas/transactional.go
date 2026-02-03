package nylas

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/nylas/cli/internal/domain"
)

// SendTransactionalMessage sends an email via the domain-based transactional endpoint.
// Used for Inbox provider grants: POST /v3/domains/{domain}/messages/send
func (c *HTTPClient) SendTransactionalMessage(ctx context.Context, domainName string, req *domain.SendMessageRequest) (*domain.Message, error) {
	queryURL := fmt.Sprintf("%s/v3/domains/%s/messages/send", c.baseURL, domainName)

	payload := map[string]any{
		"subject": req.Subject,
		"body":    req.Body,
		"to":      convertContactsToAPI(req.To),
	}

	if len(req.From) > 0 {
		payload["from"] = convertContactsToAPI(req.From)
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
	if req.TrackingOpts != nil {
		payload["tracking_options"] = req.TrackingOpts
	}
	if req.SendAt > 0 {
		payload["send_at"] = req.SendAt
	}
	if len(req.Metadata) > 0 {
		payload["metadata"] = req.Metadata
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}
	httpReq, err := http.NewRequestWithContext(ctx, "POST", queryURL, bytes.NewReader(body))
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

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusAccepted {
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
