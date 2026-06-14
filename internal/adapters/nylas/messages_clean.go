package nylas

import (
	"context"
	"fmt"
	"net/url"

	"github.com/nylas/cli/internal/domain"
)

// CleanMessages parses one or more messages into clean, display-ready text,
// stripping quoted reply/forward chains, signatures, and conclusion phrases
// (PUT /v3/grants/{id}/messages/clean).
func (c *HTTPClient) CleanMessages(ctx context.Context, grantID string, req *domain.CleanMessagesRequest) ([]domain.CleanedMessage, error) {
	if err := validateRequired("grant ID", grantID); err != nil {
		return nil, err
	}
	if req == nil || len(req.MessageIDs) == 0 {
		return nil, fmt.Errorf("%w: at least one message ID is required", domain.ErrInvalidInput)
	}
	if len(req.MessageIDs) > domain.CleanMessagesMaxIDs {
		return nil, fmt.Errorf("%w: clean accepts at most %d message IDs (got %d)", domain.ErrInvalidInput, domain.CleanMessagesMaxIDs, len(req.MessageIDs))
	}

	queryURL := fmt.Sprintf("%s/v3/grants/%s/messages/clean", c.baseURL, url.PathEscape(grantID))

	resp, err := c.doJSONRequest(ctx, "PUT", queryURL, req)
	if err != nil {
		return nil, err
	}

	var result struct {
		Data []domain.CleanedMessage `json:"data"`
	}
	if err := c.decodeJSONResponse(resp, &result); err != nil {
		return nil, err
	}

	return result.Data, nil
}
