package nylas

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/nylas/cli/internal/domain"
)

func buildTransactionalSendPayload(req *domain.SendMessageRequest) map[string]any {
	payload := buildSendMessagePayload(req, false)

	if len(req.From) > 0 {
		payload["from"] = map[string]string{
			"name":  req.From[0].Name,
			"email": req.From[0].Email,
		}
	}

	return payload
}

// SendTransactionalMessage sends an email via the domain-based transactional endpoint.
// Used for managed Nylas grants: POST /v3/domains/{domain}/messages/send
func (c *HTTPClient) SendTransactionalMessage(ctx context.Context, domainName string, req *domain.SendMessageRequest) (*domain.Message, error) {
	queryURL := fmt.Sprintf("%s/v3/domains/%s/messages/send", c.baseURL, url.PathEscape(domainName))

	resp, err := c.doJSONRequest(ctx, "POST", queryURL, buildTransactionalSendPayload(req), http.StatusOK, http.StatusCreated, http.StatusAccepted)
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
