package nylas

import (
	"context"
	"fmt"
	"net/url"

	"github.com/nylas/cli/internal/domain"
)

type policyListResponse struct {
	Data       []domain.Policy `json:"data"`
	NextCursor string          `json:"next_cursor,omitempty"`
}

type policyResponse struct {
	Data domain.Policy `json:"data"`
}

// ListPolicies lists all policies available to the authenticated application.
func (c *HTTPClient) ListPolicies(ctx context.Context) ([]domain.Policy, error) {
	baseURL := fmt.Sprintf("%s/v3/policies", c.baseURL)
	pageToken := ""
	policies := make([]domain.Policy, 0)

	for {
		queryBuilder := NewQueryBuilder()
		if pageToken != "" {
			queryBuilder.Add("page_token", pageToken)
		}
		queryURL := queryBuilder.BuildURL(baseURL)

		var result policyListResponse
		if err := c.doGet(ctx, queryURL, &result); err != nil {
			return nil, err
		}

		policies = append(policies, result.Data...)

		if result.NextCursor == "" {
			break
		}
		if result.NextCursor == pageToken {
			return nil, fmt.Errorf("failed to paginate policies: repeated cursor %q", result.NextCursor)
		}
		pageToken = result.NextCursor
	}

	return policies, nil
}

// GetPolicy retrieves a policy by ID.
func (c *HTTPClient) GetPolicy(ctx context.Context, policyID string) (*domain.Policy, error) {
	queryURL := fmt.Sprintf("%s/v3/policies/%s", c.baseURL, url.PathEscape(policyID))

	var result policyResponse
	if err := c.doGetWithNotFound(ctx, queryURL, &result, domain.ErrPolicyNotFound); err != nil {
		return nil, err
	}

	return &result.Data, nil
}

// CreatePolicy creates a new policy.
func (c *HTTPClient) CreatePolicy(ctx context.Context, payload map[string]any) (*domain.Policy, error) {
	queryURL := fmt.Sprintf("%s/v3/policies", c.baseURL)

	resp, err := c.doJSONRequest(ctx, "POST", queryURL, payload)
	if err != nil {
		return nil, err
	}

	var result policyResponse
	if err := c.decodeJSONResponse(resp, &result); err != nil {
		return nil, err
	}

	return &result.Data, nil
}

// UpdatePolicy updates an existing policy.
func (c *HTTPClient) UpdatePolicy(ctx context.Context, policyID string, payload map[string]any) (*domain.Policy, error) {
	queryURL := fmt.Sprintf("%s/v3/policies/%s", c.baseURL, url.PathEscape(policyID))

	resp, err := c.doJSONRequest(ctx, "PUT", queryURL, payload)
	if err != nil {
		return nil, err
	}

	var result policyResponse
	if err := c.decodeJSONResponse(resp, &result); err != nil {
		return nil, err
	}
	if result.Data.ID == "" {
		result.Data.ID = policyID
	}

	return &result.Data, nil
}

// DeletePolicy deletes a policy by ID.
func (c *HTTPClient) DeletePolicy(ctx context.Context, policyID string) error {
	queryURL := fmt.Sprintf("%s/v3/policies/%s", c.baseURL, url.PathEscape(policyID))
	return c.doDelete(ctx, queryURL)
}
