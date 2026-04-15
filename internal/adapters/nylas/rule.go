package nylas

import (
	"context"
	"fmt"
	"net/url"

	"github.com/nylas/cli/internal/domain"
)

type ruleListResponse struct {
	Data struct {
		Items []domain.Rule `json:"items"`
	} `json:"data"`
	NextCursor string `json:"next_cursor,omitempty"`
}

type ruleResponse struct {
	Data domain.Rule `json:"data"`
}

// ListRules lists all rules available to the authenticated application.
func (c *HTTPClient) ListRules(ctx context.Context) ([]domain.Rule, error) {
	baseURL := fmt.Sprintf("%s/v3/rules", c.baseURL)
	pageToken := ""
	rules := make([]domain.Rule, 0)

	for {
		queryBuilder := NewQueryBuilder()
		if pageToken != "" {
			queryBuilder.Add("page_token", pageToken)
		}
		queryURL := queryBuilder.BuildURL(baseURL)

		var result ruleListResponse
		if err := c.doGet(ctx, queryURL, &result); err != nil {
			return nil, err
		}

		rules = append(rules, result.Data.Items...)

		if result.NextCursor == "" {
			break
		}
		if result.NextCursor == pageToken {
			return nil, fmt.Errorf("failed to paginate rules: repeated cursor %q", result.NextCursor)
		}
		pageToken = result.NextCursor
	}

	return rules, nil
}

// GetRule retrieves a rule by ID.
func (c *HTTPClient) GetRule(ctx context.Context, ruleID string) (*domain.Rule, error) {
	queryURL := fmt.Sprintf("%s/v3/rules/%s", c.baseURL, url.PathEscape(ruleID))

	var result ruleResponse
	if err := c.doGetWithNotFound(ctx, queryURL, &result, domain.ErrRuleNotFound); err != nil {
		return nil, err
	}

	return &result.Data, nil
}

// CreateRule creates a new rule.
func (c *HTTPClient) CreateRule(ctx context.Context, payload map[string]any) (*domain.Rule, error) {
	queryURL := fmt.Sprintf("%s/v3/rules", c.baseURL)

	resp, err := c.doJSONRequest(ctx, "POST", queryURL, payload)
	if err != nil {
		return nil, err
	}

	var result ruleResponse
	if err := c.decodeJSONResponse(resp, &result); err != nil {
		return nil, err
	}

	return &result.Data, nil
}

// UpdateRule updates an existing rule.
func (c *HTTPClient) UpdateRule(ctx context.Context, ruleID string, payload map[string]any) (*domain.Rule, error) {
	queryURL := fmt.Sprintf("%s/v3/rules/%s", c.baseURL, url.PathEscape(ruleID))

	resp, err := c.doJSONRequest(ctx, "PUT", queryURL, payload)
	if err != nil {
		return nil, err
	}

	var result ruleResponse
	if err := c.decodeJSONResponse(resp, &result); err != nil {
		return nil, err
	}
	if result.Data.ID == "" {
		result.Data.ID = ruleID
	}

	return &result.Data, nil
}

// DeleteRule deletes a rule by ID.
func (c *HTTPClient) DeleteRule(ctx context.Context, ruleID string) error {
	queryURL := fmt.Sprintf("%s/v3/rules/%s", c.baseURL, url.PathEscape(ruleID))
	return c.doDelete(ctx, queryURL)
}
