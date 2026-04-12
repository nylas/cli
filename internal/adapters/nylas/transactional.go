package nylas

import (
	"context"
	"fmt"
	"net/http"

	"github.com/nylas/cli/internal/domain"
)

// SendTransactionalMessage sends an email via the domain-based transactional endpoint.
// Used for Inbox provider grants: POST /v3/domains/{domain}/messages/send
func (c *HTTPClient) SendTransactionalMessage(ctx context.Context, domainName string, req *domain.SendMessageRequest) (*domain.Message, error) {
	queryURL := fmt.Sprintf("%s/v3/domains/%s/messages/send", c.baseURL, domainName)

	resp, err := c.doJSONRequest(ctx, "POST", queryURL, buildSendMessagePayload(req, false), http.StatusOK, http.StatusCreated, http.StatusAccepted)
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
