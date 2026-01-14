package nylas

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/nylas/cli/internal/domain"
)

// Admin Applications

// ListApplications retrieves all applications.
func (c *HTTPClient) ListApplications(ctx context.Context) ([]domain.Application, error) {
	queryURL := fmt.Sprintf("%s/v3/applications", c.baseURL)

	req, err := http.NewRequestWithContext(ctx, "GET", queryURL, nil)
	if err != nil {
		return nil, err
	}
	c.setAuthHeader(req)

	resp, err := c.doRequest(ctx, req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, c.parseError(resp)
	}

	// Read body once (special handling: API may return array or single object)
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// Try to decode as an array first
	var multiResult struct {
		Data []domain.Application `json:"data"`
	}
	if err := json.Unmarshal(body, &multiResult); err == nil && len(multiResult.Data) > 0 {
		return multiResult.Data, nil
	}

	// Try to decode as a single application object (v3 API returns single app)
	var singleResult struct {
		Data domain.Application `json:"data"`
	}
	if err := json.Unmarshal(body, &singleResult); err == nil {
		// Check if we got valid application data (ID or ApplicationID set)
		if singleResult.Data.ID != "" || singleResult.Data.ApplicationID != "" {
			return []domain.Application{singleResult.Data}, nil
		}
	}

	// If both fail, return error with response body for debugging
	return nil, fmt.Errorf("failed to decode applications response: %s", string(body))
}

// GetApplication retrieves a specific application.
func (c *HTTPClient) GetApplication(ctx context.Context, appID string) (*domain.Application, error) {
	if err := validateRequired("application ID", appID); err != nil {
		return nil, err
	}

	queryURL := fmt.Sprintf("%s/v3/applications/%s", c.baseURL, appID)

	var result struct {
		Data domain.Application `json:"data"`
	}
	if err := c.doGetWithNotFound(ctx, queryURL, &result, fmt.Errorf("%w: application not found", domain.ErrAPIError)); err != nil {
		return nil, err
	}
	return &result.Data, nil
}

// CreateApplication creates a new application.
func (c *HTTPClient) CreateApplication(ctx context.Context, req *domain.CreateApplicationRequest) (*domain.Application, error) {
	queryURL := fmt.Sprintf("%s/v3/applications", c.baseURL)

	resp, err := c.doJSONRequest(ctx, "POST", queryURL, req)
	if err != nil {
		return nil, err
	}

	var result struct {
		Data domain.Application `json:"data"`
	}
	if err := c.decodeJSONResponse(resp, &result); err != nil {
		return nil, err
	}
	return &result.Data, nil
}

// UpdateApplication updates an existing application.
func (c *HTTPClient) UpdateApplication(ctx context.Context, appID string, req *domain.UpdateApplicationRequest) (*domain.Application, error) {
	if err := validateRequired("application ID", appID); err != nil {
		return nil, err
	}

	queryURL := fmt.Sprintf("%s/v3/applications/%s", c.baseURL, appID)

	resp, err := c.doJSONRequest(ctx, "PATCH", queryURL, req)
	if err != nil {
		return nil, err
	}

	var result struct {
		Data domain.Application `json:"data"`
	}
	if err := c.decodeJSONResponse(resp, &result); err != nil {
		return nil, err
	}
	return &result.Data, nil
}

// DeleteApplication deletes an application.
func (c *HTTPClient) DeleteApplication(ctx context.Context, appID string) error {
	if err := validateRequired("application ID", appID); err != nil {
		return err
	}
	queryURL := fmt.Sprintf("%s/v3/applications/%s", c.baseURL, appID)
	return c.doDelete(ctx, queryURL)
}

// Admin Connectors

// ListConnectors retrieves all connectors.
func (c *HTTPClient) ListConnectors(ctx context.Context) ([]domain.Connector, error) {
	queryURL := fmt.Sprintf("%s/v3/connectors", c.baseURL)

	resp, err := c.doJSONRequest(ctx, "GET", queryURL, nil)
	if err != nil {
		return nil, err
	}

	var result struct {
		Data []domain.Connector `json:"data"`
	}
	if err := c.decodeJSONResponse(resp, &result); err != nil {
		return nil, err
	}
	return result.Data, nil
}

// GetConnector retrieves a specific connector.
func (c *HTTPClient) GetConnector(ctx context.Context, connectorID string) (*domain.Connector, error) {
	if err := validateRequired("connector ID", connectorID); err != nil {
		return nil, err
	}

	queryURL := fmt.Sprintf("%s/v3/connectors/%s", c.baseURL, connectorID)

	var result struct {
		Data domain.Connector `json:"data"`
	}
	if err := c.doGetWithNotFound(ctx, queryURL, &result, fmt.Errorf("%w: connector not found", domain.ErrAPIError)); err != nil {
		return nil, err
	}
	return &result.Data, nil
}

// CreateConnector creates a new connector.
func (c *HTTPClient) CreateConnector(ctx context.Context, req *domain.CreateConnectorRequest) (*domain.Connector, error) {
	queryURL := fmt.Sprintf("%s/v3/connectors", c.baseURL)

	resp, err := c.doJSONRequest(ctx, "POST", queryURL, req)
	if err != nil {
		return nil, err
	}

	var result struct {
		Data domain.Connector `json:"data"`
	}
	if err := c.decodeJSONResponse(resp, &result); err != nil {
		return nil, err
	}
	return &result.Data, nil
}

// UpdateConnector updates an existing connector.
func (c *HTTPClient) UpdateConnector(ctx context.Context, connectorID string, req *domain.UpdateConnectorRequest) (*domain.Connector, error) {
	if err := validateRequired("connector ID", connectorID); err != nil {
		return nil, err
	}

	queryURL := fmt.Sprintf("%s/v3/connectors/%s", c.baseURL, connectorID)

	resp, err := c.doJSONRequest(ctx, "PATCH", queryURL, req)
	if err != nil {
		return nil, err
	}

	var result struct {
		Data domain.Connector `json:"data"`
	}
	if err := c.decodeJSONResponse(resp, &result); err != nil {
		return nil, err
	}
	return &result.Data, nil
}

// DeleteConnector deletes a connector.
func (c *HTTPClient) DeleteConnector(ctx context.Context, connectorID string) error {
	if err := validateRequired("connector ID", connectorID); err != nil {
		return err
	}
	queryURL := fmt.Sprintf("%s/v3/connectors/%s", c.baseURL, connectorID)
	return c.doDelete(ctx, queryURL)
}

// Admin Credentials

// ListCredentials retrieves all credentials for a connector.
func (c *HTTPClient) ListCredentials(ctx context.Context, connectorID string) ([]domain.ConnectorCredential, error) {
	if err := validateRequired("connector ID", connectorID); err != nil {
		return nil, err
	}

	queryURL := fmt.Sprintf("%s/v3/connectors/%s/credentials", c.baseURL, connectorID)

	resp, err := c.doJSONRequest(ctx, "GET", queryURL, nil)
	if err != nil {
		return nil, err
	}

	var result struct {
		Data []domain.ConnectorCredential `json:"data"`
	}
	if err := c.decodeJSONResponse(resp, &result); err != nil {
		return nil, err
	}
	return result.Data, nil
}

// GetCredential retrieves a specific credential.
func (c *HTTPClient) GetCredential(ctx context.Context, credentialID string) (*domain.ConnectorCredential, error) {
	if err := validateRequired("credential ID", credentialID); err != nil {
		return nil, err
	}

	queryURL := fmt.Sprintf("%s/v3/credentials/%s", c.baseURL, credentialID)

	var result struct {
		Data domain.ConnectorCredential `json:"data"`
	}
	if err := c.doGetWithNotFound(ctx, queryURL, &result, fmt.Errorf("%w: credential not found", domain.ErrAPIError)); err != nil {
		return nil, err
	}
	return &result.Data, nil
}

// CreateCredential creates a new credential.
func (c *HTTPClient) CreateCredential(ctx context.Context, connectorID string, req *domain.CreateCredentialRequest) (*domain.ConnectorCredential, error) {
	if err := validateRequired("connector ID", connectorID); err != nil {
		return nil, err
	}

	queryURL := fmt.Sprintf("%s/v3/connectors/%s/credentials", c.baseURL, connectorID)

	resp, err := c.doJSONRequest(ctx, "POST", queryURL, req)
	if err != nil {
		return nil, err
	}

	var result struct {
		Data domain.ConnectorCredential `json:"data"`
	}
	if err := c.decodeJSONResponse(resp, &result); err != nil {
		return nil, err
	}
	return &result.Data, nil
}

// UpdateCredential updates an existing credential.
func (c *HTTPClient) UpdateCredential(ctx context.Context, credentialID string, req *domain.UpdateCredentialRequest) (*domain.ConnectorCredential, error) {
	if err := validateRequired("credential ID", credentialID); err != nil {
		return nil, err
	}

	queryURL := fmt.Sprintf("%s/v3/credentials/%s", c.baseURL, credentialID)

	resp, err := c.doJSONRequest(ctx, "PATCH", queryURL, req)
	if err != nil {
		return nil, err
	}

	var result struct {
		Data domain.ConnectorCredential `json:"data"`
	}
	if err := c.decodeJSONResponse(resp, &result); err != nil {
		return nil, err
	}
	return &result.Data, nil
}

// DeleteCredential deletes a credential.
func (c *HTTPClient) DeleteCredential(ctx context.Context, credentialID string) error {
	if err := validateRequired("credential ID", credentialID); err != nil {
		return err
	}
	queryURL := fmt.Sprintf("%s/v3/credentials/%s", c.baseURL, credentialID)
	return c.doDelete(ctx, queryURL)
}

// Admin Grant Operations

// ListAllGrants retrieves all grants with optional filtering.
func (c *HTTPClient) ListAllGrants(ctx context.Context, params *domain.GrantsQueryParams) ([]domain.Grant, error) {
	baseURL := fmt.Sprintf("%s/v3/grants", c.baseURL)

	qb := NewQueryBuilder()
	if params != nil {
		qb.AddInt("limit", params.Limit).
			AddInt("offset", params.Offset).
			Add("connector_id", params.ConnectorID).
			Add("status", params.Status)
	}
	queryURL := qb.BuildURL(baseURL)

	resp, err := c.doJSONRequest(ctx, "GET", queryURL, nil)
	if err != nil {
		return nil, err
	}

	var result struct {
		Data []domain.Grant `json:"data"`
	}
	if err := c.decodeJSONResponse(resp, &result); err != nil {
		return nil, err
	}
	return result.Data, nil
}

// GetGrantStats retrieves grant statistics.
func (c *HTTPClient) GetGrantStats(ctx context.Context) (*domain.GrantStats, error) {
	// Get all grants
	grants, err := c.ListAllGrants(ctx, nil)
	if err != nil {
		return nil, err
	}

	// Calculate statistics
	stats := &domain.GrantStats{
		Total:      len(grants),
		ByProvider: make(map[string]int),
		ByStatus:   make(map[string]int),
	}

	for _, grant := range grants {
		// Count by provider
		stats.ByProvider[string(grant.Provider)]++

		// Count by status
		if grant.GrantStatus != "" {
			stats.ByStatus[grant.GrantStatus]++
			switch grant.GrantStatus {
			case "valid":
				stats.Valid++
			case "invalid":
				stats.Invalid++
			}
		}
	}

	return stats, nil
}
