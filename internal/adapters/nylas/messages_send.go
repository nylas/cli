package nylas

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/nylas/cli/internal/domain"
	"github.com/nylas/cli/internal/util"
)

// SendMessage sends an email.
func (c *HTTPClient) SendMessage(ctx context.Context, grantID string, req *domain.SendMessageRequest) (*domain.Message, error) {
	queryURL := fmt.Sprintf("%s/v3/grants/%s/messages/send", c.baseURL, grantID)

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

// ListScheduledMessages retrieves all scheduled messages for a grant.
func (c *HTTPClient) ListScheduledMessages(ctx context.Context, grantID string) ([]domain.ScheduledMessage, error) {
	queryURL := fmt.Sprintf("%s/v3/grants/%s/messages/schedules", c.baseURL, grantID)

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

	if resp.StatusCode != http.StatusOK {
		return nil, c.parseError(resp)
	}

	var result struct {
		Data []struct {
			ScheduleID string `json:"schedule_id"`
			Status     struct {
				Code        string `json:"code"`
				Description string `json:"description"`
			} `json:"status"`
			CloseTime int64 `json:"close_time"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return util.Map(result.Data, func(s struct {
		ScheduleID string `json:"schedule_id"`
		Status     struct {
			Code        string `json:"code"`
			Description string `json:"description"`
		} `json:"status"`
		CloseTime int64 `json:"close_time"`
	}) domain.ScheduledMessage {
		return domain.ScheduledMessage{
			ScheduleID: s.ScheduleID,
			Status:     s.Status.Code,
			CloseTime:  s.CloseTime,
		}
	}), nil
}

// GetScheduledMessage retrieves a specific scheduled message.
func (c *HTTPClient) GetScheduledMessage(ctx context.Context, grantID, scheduleID string) (*domain.ScheduledMessage, error) {
	queryURL := fmt.Sprintf("%s/v3/grants/%s/messages/schedules/%s", c.baseURL, grantID, scheduleID)

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
		return nil, fmt.Errorf("%w: scheduled message not found", domain.ErrAPIError)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, c.parseError(resp)
	}

	var result struct {
		Data struct {
			ScheduleID string `json:"schedule_id"`
			Status     struct {
				Code        string `json:"code"`
				Description string `json:"description"`
			} `json:"status"`
			CloseTime int64 `json:"close_time"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &domain.ScheduledMessage{
		ScheduleID: result.Data.ScheduleID,
		Status:     result.Data.Status.Code,
		CloseTime:  result.Data.CloseTime,
	}, nil
}

// CancelScheduledMessage cancels a scheduled message.
func (c *HTTPClient) CancelScheduledMessage(ctx context.Context, grantID, scheduleID string) error {
	queryURL := fmt.Sprintf("%s/v3/grants/%s/messages/schedules/%s", c.baseURL, grantID, scheduleID)

	req, err := http.NewRequestWithContext(ctx, "DELETE", queryURL, nil)
	if err != nil {
		return err
	}
	c.setAuthHeader(req)

	resp, err := c.doRequest(ctx, req)
	if err != nil {
		return fmt.Errorf("%w: %v", domain.ErrNetworkError, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusAccepted {
		return c.parseError(resp)
	}

	return nil
}

// SmartCompose generates an AI-powered email draft based on a prompt.
// Uses Nylas Smart Compose API (requires Plus package).
func (c *HTTPClient) SmartCompose(ctx context.Context, grantID string, req *domain.SmartComposeRequest) (*domain.SmartComposeSuggestion, error) {
	queryURL := fmt.Sprintf("%s/v3/grants/%s/messages/smart-compose", c.baseURL, grantID)

	payload := map[string]any{
		"prompt": req.Prompt,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", queryURL, bytes.NewReader(jsonData))
	if err != nil {
		return nil, err
	}
	c.setAuthHeader(httpReq)
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")

	resp, err := c.doRequest(ctx, httpReq)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", domain.ErrNetworkError, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, c.parseError(resp)
	}

	var result struct {
		Data domain.SmartComposeSuggestion `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &result.Data, nil
}

// SmartComposeReply generates an AI-powered reply to a specific message.
// Uses Nylas Smart Compose API (requires Plus package).
func (c *HTTPClient) SmartComposeReply(ctx context.Context, grantID, messageID string, req *domain.SmartComposeRequest) (*domain.SmartComposeSuggestion, error) {
	queryURL := fmt.Sprintf("%s/v3/grants/%s/messages/%s/smart-compose", c.baseURL, grantID, messageID)

	payload := map[string]any{
		"prompt": req.Prompt,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", queryURL, bytes.NewReader(jsonData))
	if err != nil {
		return nil, err
	}
	c.setAuthHeader(httpReq)
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")

	resp, err := c.doRequest(ctx, httpReq)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", domain.ErrNetworkError, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, c.parseError(resp)
	}

	var result struct {
		Data domain.SmartComposeSuggestion `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &result.Data, nil
}

// convertContactsToAPI converts domain contacts to API format.
func convertContactsToAPI(contacts []domain.EmailParticipant) []map[string]string {
	result := make([]map[string]string, len(contacts))
	for i, c := range contacts {
		result[i] = map[string]string{
			"name":  c.Name,
			"email": c.Email,
		}
	}
	return result
}
