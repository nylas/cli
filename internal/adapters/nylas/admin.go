package nylas

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

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

	// Read body once (special handling: API may return array or single object).
	// Bound the read so a misbehaving upstream cannot OOM us, and so that an
	// auth/error JSON containing tokens or PII cannot be echoed unbounded into
	// our error string below.
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBodySize))
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

	// Both decodings failed. Report status only — never echo the raw body
	// since it may carry tokens, customer data, or other sensitive fields.
	return nil, fmt.Errorf("failed to decode applications response (status %d, %d bytes)", resp.StatusCode, len(body))
}

// maxResponseBodySize bounds bodies read for ad-hoc decoding (1 MiB). Larger
// bodies are an upstream bug; truncating prevents secret leakage in errors.
const maxResponseBodySize = 1 << 20

// GetApplication retrieves a specific application.
func (c *HTTPClient) GetApplication(ctx context.Context, appID string) (*domain.Application, error) {
	if err := validateRequired("application ID", appID); err != nil {
		return nil, err
	}

	queryURL := fmt.Sprintf("%s/v3/applications/%s", c.baseURL, url.PathEscape(appID))

	var result struct {
		Data domain.Application `json:"data"`
	}
	if err := c.doGetWithNotFound(ctx, queryURL, &result, domain.ErrApplicationNotFound); err != nil {
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

	queryURL := fmt.Sprintf("%s/v3/applications/%s", c.baseURL, url.PathEscape(appID))

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
	queryURL := fmt.Sprintf("%s/v3/applications/%s", c.baseURL, url.PathEscape(appID))
	return c.doDelete(ctx, queryURL)
}

// Callback URI Operations

// ListCallbackURIs retrieves all callback URIs for the application.
func (c *HTTPClient) ListCallbackURIs(ctx context.Context) ([]domain.CallbackURI, error) {
	queryURL := fmt.Sprintf("%s/v3/applications/callback-uris", c.baseURL)

	var result struct {
		Data []domain.CallbackURI `json:"data"`
	}
	if err := c.doGet(ctx, queryURL, &result); err != nil {
		return nil, err
	}
	return result.Data, nil
}

// GetCallbackURI retrieves a specific callback URI.
func (c *HTTPClient) GetCallbackURI(ctx context.Context, uriID string) (*domain.CallbackURI, error) {
	if err := validateRequired("callback URI ID", uriID); err != nil {
		return nil, err
	}

	queryURL := fmt.Sprintf("%s/v3/applications/callback-uris/%s", c.baseURL, url.PathEscape(uriID))

	var result struct {
		Data domain.CallbackURI `json:"data"`
	}
	if err := c.doGetWithNotFound(ctx, queryURL, &result, domain.ErrCallbackURINotFound); err != nil {
		return nil, err
	}
	return &result.Data, nil
}

// CreateCallbackURI creates a new callback URI for the application.
func (c *HTTPClient) CreateCallbackURI(ctx context.Context, req *domain.CreateCallbackURIRequest) (*domain.CallbackURI, error) {
	if req == nil {
		return nil, fmt.Errorf("create callback URI request is required")
	}
	if err := validateRequired("callback URI URL", req.URL); err != nil {
		return nil, err
	}

	queryURL := fmt.Sprintf("%s/v3/applications/callback-uris", c.baseURL)

	resp, err := c.doJSONRequest(ctx, "POST", queryURL, req)
	if err != nil {
		return nil, err
	}

	var result struct {
		Data domain.CallbackURI `json:"data"`
	}
	if err := c.decodeJSONResponse(resp, &result); err != nil {
		return nil, err
	}
	return &result.Data, nil
}

// UpdateCallbackURI updates an existing callback URI.
func (c *HTTPClient) UpdateCallbackURI(ctx context.Context, uriID string, req *domain.UpdateCallbackURIRequest) (*domain.CallbackURI, error) {
	if err := validateRequired("callback URI ID", uriID); err != nil {
		return nil, err
	}

	queryURL := fmt.Sprintf("%s/v3/applications/callback-uris/%s", c.baseURL, url.PathEscape(uriID))

	resp, err := c.doJSONRequest(ctx, "PATCH", queryURL, req)
	if err != nil {
		return nil, err
	}

	var result struct {
		Data domain.CallbackURI `json:"data"`
	}
	if err := c.decodeJSONResponse(resp, &result); err != nil {
		return nil, err
	}
	return &result.Data, nil
}

// DeleteCallbackURI deletes a callback URI.
func (c *HTTPClient) DeleteCallbackURI(ctx context.Context, uriID string) error {
	if err := validateRequired("callback URI ID", uriID); err != nil {
		return err
	}
	queryURL := fmt.Sprintf("%s/v3/applications/callback-uris/%s", c.baseURL, url.PathEscape(uriID))
	return c.doDelete(ctx, queryURL)
}

// Admin Connectors

// ListConnectors retrieves all connectors.
func (c *HTTPClient) ListConnectors(ctx context.Context) ([]domain.Connector, error) {
	queryURL := fmt.Sprintf("%s/v3/connectors", c.baseURL)

	var result struct {
		Data []domain.Connector `json:"data"`
	}
	if err := c.doGet(ctx, queryURL, &result); err != nil {
		return nil, err
	}
	return result.Data, nil
}

// GetConnector retrieves a specific connector.
func (c *HTTPClient) GetConnector(ctx context.Context, connectorID string) (*domain.Connector, error) {
	if err := validateRequired("connector ID", connectorID); err != nil {
		return nil, err
	}

	queryURL := fmt.Sprintf("%s/v3/connectors/%s", c.baseURL, url.PathEscape(connectorID))

	var result struct {
		Data domain.Connector `json:"data"`
	}
	if err := c.doGetWithNotFound(ctx, queryURL, &result, domain.ErrConnectorNotFound); err != nil {
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

	queryURL := fmt.Sprintf("%s/v3/connectors/%s", c.baseURL, url.PathEscape(connectorID))

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
	queryURL := fmt.Sprintf("%s/v3/connectors/%s", c.baseURL, url.PathEscape(connectorID))
	return c.doDelete(ctx, queryURL)
}

// Admin Credentials

// ListCredentials retrieves all credentials for a connector.
func (c *HTTPClient) ListCredentials(ctx context.Context, connectorID string) ([]domain.ConnectorCredential, error) {
	if err := validateRequired("connector ID", connectorID); err != nil {
		return nil, err
	}

	queryURL := fmt.Sprintf("%s/v3/connectors/%s/credentials", c.baseURL, url.PathEscape(connectorID))

	var result struct {
		Data []domain.ConnectorCredential `json:"data"`
	}
	if err := c.doGet(ctx, queryURL, &result); err != nil {
		return nil, err
	}
	return result.Data, nil
}

// GetCredential retrieves a specific credential.
func (c *HTTPClient) GetCredential(ctx context.Context, credentialID string) (*domain.ConnectorCredential, error) {
	if err := validateRequired("credential ID", credentialID); err != nil {
		return nil, err
	}

	queryURL := fmt.Sprintf("%s/v3/credentials/%s", c.baseURL, url.PathEscape(credentialID))

	var result struct {
		Data domain.ConnectorCredential `json:"data"`
	}
	if err := c.doGetWithNotFound(ctx, queryURL, &result, domain.ErrCredentialNotFound); err != nil {
		return nil, err
	}
	return &result.Data, nil
}

// CreateCredential creates a new credential.
func (c *HTTPClient) CreateCredential(ctx context.Context, connectorID string, req *domain.CreateCredentialRequest) (*domain.ConnectorCredential, error) {
	if err := validateRequired("connector ID", connectorID); err != nil {
		return nil, err
	}

	queryURL := fmt.Sprintf("%s/v3/connectors/%s/credentials", c.baseURL, url.PathEscape(connectorID))

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

	queryURL := fmt.Sprintf("%s/v3/credentials/%s", c.baseURL, url.PathEscape(credentialID))

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
	queryURL := fmt.Sprintf("%s/v3/credentials/%s", c.baseURL, url.PathEscape(credentialID))
	return c.doDelete(ctx, queryURL)
}

// Admin Grant Operations

// maxGrantPages caps the number of pages ListAllGrants will fetch even
// when the server keeps returning fresh next_cursor values. It's a
// safety ceiling above any realistic grant count — the cycle detector
// catches the typical misbehaviour, but a server that hands out
// distinct-but-empty pages forever still needs a hard stop.
const maxGrantPages = 1000

// ListAllGrants retrieves all grants matching the optional filters,
// transparently following next_cursor pagination so callers always see the
// complete result set.
//
// When params.Limit is positive, at most that many grants are returned and
// pagination stops once the cap is reached. When params is nil or Limit is
// zero, every page is fetched.
func (c *HTTPClient) ListAllGrants(ctx context.Context, params *domain.GrantsQueryParams) ([]domain.Grant, error) {
	baseURL := fmt.Sprintf("%s/v3/grants", c.baseURL)

	limit := 0
	connectorID := ""
	status := ""
	offset := 0
	if params != nil {
		limit = params.Limit
		connectorID = params.ConnectorID
		status = params.Status
		offset = params.Offset
	}

	pageToken := ""
	grants := make([]domain.Grant, 0)
	// seenCursors guards against cycles longer than length 1 (the simple
	// `result.NextCursor == pageToken` case is checked separately).
	seenCursors := make(map[string]struct{})
	for {
		qb := NewQueryBuilder().
			AddInt("limit", limit).
			Add("connector_id", connectorID).
			Add("status", status).
			AddInt("offset", offset).
			Add("page_token", pageToken)
		queryURL := qb.BuildURL(baseURL)

		var result struct {
			Data       []domain.Grant `json:"data"`
			NextCursor string         `json:"next_cursor,omitempty"`
		}
		if err := c.doGet(ctx, queryURL, &result); err != nil {
			return nil, err
		}

		grants = append(grants, result.Data...)
		if limit > 0 && len(grants) >= limit {
			return grants[:limit], nil
		}

		if result.NextCursor == "" {
			return grants, nil
		}
		if result.NextCursor == pageToken {
			return nil, fmt.Errorf("failed to paginate grants: repeated cursor %q", result.NextCursor)
		}
		// Detect cycles longer than length 1 (e.g. A → B → A) and bound the
		// total number of pages we'll walk so a misbehaving server can't
		// trap us in an unbounded loop. Cap at 1000 pages — well above
		// realistic grant counts but a hard ceiling on the worst case.
		if _, seen := seenCursors[result.NextCursor]; seen {
			return nil, fmt.Errorf("failed to paginate grants: cursor cycle detected near %q", result.NextCursor)
		}
		seenCursors[result.NextCursor] = struct{}{}
		if len(seenCursors) > maxGrantPages {
			return nil, fmt.Errorf("failed to paginate grants: exceeded max page count (%d)", maxGrantPages)
		}
		pageToken = result.NextCursor
		// offset only meaningful on the first request; the API uses cursors
		// to advance from there.
		offset = 0
	}
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
