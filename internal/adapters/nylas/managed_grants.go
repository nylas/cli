package nylas

import (
	"context"
	"fmt"

	"github.com/nylas/cli/internal/domain"
)

type managedGrantResponse struct {
	ID           string               `json:"id"`
	Email        string               `json:"email"`
	Provider     domain.Provider      `json:"provider"`
	GrantStatus  string               `json:"grant_status"`
	CreatedAt    domain.UnixTime      `json:"created_at"`
	UpdatedAt    domain.UnixTime      `json:"updated_at"`
	CredentialID string               `json:"credential_id,omitempty"`
	Blocked      bool                 `json:"blocked,omitempty"`
	Settings     agentSettingsPayload `json:"settings,omitempty"`
}

type agentSettingsPayload struct {
	PolicyID string `json:"policy_id,omitempty"`
}

func (c *HTTPClient) listManagedGrants(ctx context.Context, provider domain.Provider) ([]managedGrantResponse, error) {
	baseURL := fmt.Sprintf("%s/v3/grants", c.baseURL)
	pageToken := ""
	grants := make([]managedGrantResponse, 0)

	for {
		queryURL := NewQueryBuilder().
			Add("provider", string(provider)).
			Add("page_token", pageToken).
			BuildURL(baseURL)

		var result struct {
			Data       []managedGrantResponse `json:"data"`
			NextCursor string                 `json:"next_cursor,omitempty"`
		}
		if err := c.doGet(ctx, queryURL, &result); err != nil {
			return nil, err
		}

		for _, grant := range result.Data {
			if grant.Provider == provider {
				grants = append(grants, grant)
			}
		}

		if result.NextCursor == "" {
			break
		}
		if result.NextCursor == pageToken {
			return nil, fmt.Errorf("failed to paginate managed grants: repeated cursor %q", result.NextCursor)
		}
		pageToken = result.NextCursor
	}

	return grants, nil
}

func (c *HTTPClient) getManagedGrant(ctx context.Context, grantID string) (*managedGrantResponse, error) {
	queryURL := fmt.Sprintf("%s/v3/grants/%s", c.baseURL, grantID)

	var result struct {
		Data managedGrantResponse `json:"data"`
	}
	if err := c.doGetWithNotFound(ctx, queryURL, &result, domain.ErrGrantNotFound); err != nil {
		return nil, err
	}

	return &result.Data, nil
}

func (c *HTTPClient) createManagedGrant(ctx context.Context, provider domain.Provider, email string) (*managedGrantResponse, error) {
	queryURL := fmt.Sprintf("%s/v3/grants", c.baseURL)

	payload := map[string]any{
		"provider": string(provider),
		"settings": map[string]string{
			"email": email,
		},
	}

	resp, err := c.doJSONRequest(ctx, "POST", queryURL, payload)
	if err != nil {
		return nil, err
	}

	var result struct {
		Data managedGrantResponse `json:"data"`
	}
	if err := c.decodeJSONResponse(resp, &result); err != nil {
		return nil, err
	}

	return &result.Data, nil
}

func (c *HTTPClient) deleteManagedGrant(ctx context.Context, grantID string, expectedProvider domain.Provider) error {
	grant, err := c.getManagedGrant(ctx, grantID)
	if err != nil {
		return err
	}
	if grant == nil {
		return domain.ErrGrantNotFound
	}
	if grant.Provider != expectedProvider {
		return fmt.Errorf("%w: grant is not a %s managed grant (provider=%s)", domain.ErrInvalidGrant, expectedProvider, grant.Provider)
	}

	return c.RevokeGrant(ctx, grantID)
}

func convertManagedGrantToInboundInbox(grant managedGrantResponse) domain.InboundInbox {
	return domain.InboundInbox{
		ID:          grant.ID,
		Email:       grant.Email,
		PolicyID:    grant.Settings.PolicyID,
		GrantStatus: grant.GrantStatus,
		CreatedAt:   grant.CreatedAt,
		UpdatedAt:   grant.UpdatedAt,
	}
}

func convertManagedGrantToAgentAccount(grant managedGrantResponse) domain.AgentAccount {
	return domain.AgentAccount{
		ID:           grant.ID,
		Provider:     grant.Provider,
		Email:        grant.Email,
		GrantStatus:  grant.GrantStatus,
		CreatedAt:    grant.CreatedAt,
		UpdatedAt:    grant.UpdatedAt,
		CredentialID: grant.CredentialID,
		Blocked:      grant.Blocked,
		Settings: domain.AgentAccountSettings{
			PolicyID: grant.Settings.PolicyID,
		},
	}
}
