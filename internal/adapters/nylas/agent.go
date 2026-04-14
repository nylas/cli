package nylas

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/nylas/cli/internal/domain"
)

// ListAgentAccounts lists all managed agent accounts (grants with provider=nylas).
func (c *HTTPClient) ListAgentAccounts(ctx context.Context) ([]domain.AgentAccount, error) {
	grants, err := c.listManagedGrants(ctx, domain.ProviderNylas)
	if err != nil {
		return nil, err
	}

	accounts := make([]domain.AgentAccount, 0, len(grants))
	for _, grant := range grants {
		accounts = append(accounts, convertManagedGrantToAgentAccount(grant))
	}

	return accounts, nil
}

// GetAgentAccount retrieves a specific agent account by grant ID.
func (c *HTTPClient) GetAgentAccount(ctx context.Context, grantID string) (*domain.AgentAccount, error) {
	grant, err := c.getManagedGrant(ctx, grantID)
	if err != nil {
		return nil, err
	}

	if grant.Provider != domain.ProviderNylas {
		return nil, fmt.Errorf("%w: grant is not a nylas agent account (provider=%s)", domain.ErrInvalidGrant, grant.Provider)
	}

	account := convertManagedGrantToAgentAccount(*grant)
	return &account, nil
}

// CreateAgentAccount creates a new managed agent account grant.
func (c *HTTPClient) CreateAgentAccount(ctx context.Context, email, appPassword string) (*domain.AgentAccount, error) {
	queryURL := fmt.Sprintf("%s/v3/connect/custom", c.baseURL)

	settings := map[string]any{
		"email": email,
	}
	if appPassword != "" {
		settings["app_password"] = appPassword
	}

	payload := map[string]any{
		"provider": string(domain.ProviderNylas),
		"settings": settings,
	}

	resp, err := c.doJSONRequest(ctx, "POST", queryURL, payload)
	if err != nil {
		return nil, err
	}

	grant, err := decodeCreatedManagedGrant(resp)
	if err != nil {
		return nil, err
	}

	account := convertManagedGrantToAgentAccount(*grant)
	return &account, nil
}

// DeleteAgentAccount deletes an agent account by revoking its grant.
func (c *HTTPClient) DeleteAgentAccount(ctx context.Context, grantID string) error {
	return c.deleteManagedGrant(ctx, grantID, domain.ProviderNylas)
}

func decodeCreatedManagedGrant(resp *http.Response) (*managedGrantResponse, error) {
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var wrapped struct {
		Data managedGrantResponse `json:"data"`
	}
	if err := json.Unmarshal(body, &wrapped); err == nil && wrapped.Data.ID != "" {
		return &wrapped.Data, nil
	}

	var grant managedGrantResponse
	if err := json.Unmarshal(body, &grant); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	if grant.ID == "" {
		return nil, fmt.Errorf("failed to decode response: missing grant id")
	}

	return &grant, nil
}
