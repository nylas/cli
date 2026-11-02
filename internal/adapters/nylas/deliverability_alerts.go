package nylas

import (
	"context"
	"net/http"
	"net/url"

	"github.com/nylas/cli/internal/domain"
)

// ListSendAlertRules returns every send alert rule for the application (at most 20, unpaginated).
func (c *HTTPClient) ListSendAlertRules(ctx context.Context) ([]domain.SendAlertRule, error) {
	var result struct {
		Data []domain.SendAlertRule `json:"data"`
	}
	if err := c.doGet(ctx, c.baseURL+"/v3/alerts/send", &result); err != nil {
		return nil, err
	}
	return result.Data, nil
}

// GetSendAlertRule returns one send alert rule.
func (c *HTTPClient) GetSendAlertRule(ctx context.Context, ruleID string) (*domain.SendAlertRule, error) {
	if err := validateRequired("rule ID", ruleID); err != nil {
		return nil, err
	}
	var result struct {
		Data domain.SendAlertRule `json:"data"`
	}
	if err := c.doGet(ctx, sendAlertRuleURL(c.baseURL, ruleID), &result); err != nil {
		return nil, err
	}
	return &result.Data, nil
}

// CreateSendAlertRule creates a send alert rule. It does not retry, so a timeout cannot create a duplicate.
func (c *HTTPClient) CreateSendAlertRule(ctx context.Context, req *domain.SendAlertRuleRequest) (*domain.SendAlertRule, error) {
	resp, err := c.doJSONRequestNoRetry(ctx, http.MethodPost, c.baseURL+"/v3/alerts/send", req, http.StatusCreated)
	if err != nil {
		return nil, err
	}
	var result struct {
		Data domain.SendAlertRule `json:"data"`
	}
	if err := c.decodeJSONResponse(resp, &result); err != nil {
		return nil, err
	}
	return &result.Data, nil
}

// UpdateSendAlertRule changes only the fields set in req (PUT with partial semantics).
func (c *HTTPClient) UpdateSendAlertRule(ctx context.Context, ruleID string, req *domain.SendAlertRuleRequest) (*domain.SendAlertRule, error) {
	if err := validateRequired("rule ID", ruleID); err != nil {
		return nil, err
	}
	resp, err := c.doJSONRequest(ctx, http.MethodPut, sendAlertRuleURL(c.baseURL, ruleID), req, http.StatusOK)
	if err != nil {
		return nil, err
	}
	var result struct {
		Data domain.SendAlertRule `json:"data"`
	}
	if err := c.decodeJSONResponse(resp, &result); err != nil {
		return nil, err
	}
	return &result.Data, nil
}

// DeleteSendAlertRule deletes a send alert rule.
func (c *HTTPClient) DeleteSendAlertRule(ctx context.Context, ruleID string) error {
	if err := validateRequired("rule ID", ruleID); err != nil {
		return err
	}
	return c.doDelete(ctx, sendAlertRuleURL(c.baseURL, ruleID))
}

func sendAlertRuleURL(baseURL, ruleID string) string {
	return baseURL + "/v3/alerts/send/" + url.PathEscape(ruleID)
}
