package dashboard

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/nylas/cli/internal/domain"
	"github.com/nylas/cli/internal/ports"
)

// GatewayClient implements ports.DashboardGatewayClient for the
// dashboard API gateway GraphQL endpoints.
type GatewayClient struct {
	httpClient *http.Client
	dpop       ports.DPoP
}

// NewGatewayClient creates a new dashboard gateway GraphQL client.
func NewGatewayClient(dpop ports.DPoP) *GatewayClient {
	return &GatewayClient{
		httpClient: newNonRedirectClient(),
		dpop:       dpop,
	}
}

// ListApplications retrieves applications from the dashboard API gateway.
func (c *GatewayClient) ListApplications(ctx context.Context, orgPublicID, region, userToken, orgToken string) ([]domain.GatewayApplication, error) {
	query := `query V3_GetApplications($filter: ApplicationFilter!) {
  applications(filter: $filter) {
    applications {
      applicationId
      organizationId
      region
      environment
      branding { name description }
    }
  }
}`

	variables := map[string]any{
		"filter": map[string]any{
			"orgPublicId": orgPublicID,
		},
	}

	url := gatewayURL(region)
	raw, err := c.doGraphQL(ctx, url, query, variables, userToken, orgToken)
	if err != nil {
		return nil, fmt.Errorf("failed to list applications: %w", err)
	}

	var resp struct {
		Data struct {
			Applications struct {
				Applications []domain.GatewayApplication `json:"applications"`
			} `json:"applications"`
		} `json:"data"`
		Errors []graphQLError `json:"errors"`
	}

	if err := json.Unmarshal(raw, &resp); err != nil {
		return nil, fmt.Errorf("failed to decode applications response: %w", err)
	}

	if len(resp.Errors) > 0 {
		return nil, fmt.Errorf("failed to list applications: %w", graphQLErrorAsError(resp.Errors[0]))
	}

	return resp.Data.Applications.Applications, nil
}

// CreateApplication creates a new application via the dashboard API gateway.
func (c *GatewayClient) CreateApplication(ctx context.Context, orgPublicID, region, name, userToken, orgToken string) (*domain.GatewayCreatedApplication, error) {
	query := `mutation V3_CreateApplication($orgPublicId: String!, $options: ApplicationOptions!) {
  createApplication(orgPublicId: $orgPublicId, options: $options) {
    applicationId
    clientSecret
    organizationId
    region
    environment
    branding { name }
  }
}`

	variables := map[string]any{
		"orgPublicId": orgPublicID,
		"options": map[string]any{
			"region": region,
			"branding": map[string]any{
				"name": name,
			},
		},
	}

	url := gatewayURL(region)
	raw, err := c.doGraphQL(ctx, url, query, variables, userToken, orgToken)
	if err != nil {
		return nil, fmt.Errorf("failed to create application: %w", err)
	}

	var resp struct {
		Data struct {
			CreateApplication domain.GatewayCreatedApplication `json:"createApplication"`
		} `json:"data"`
		Errors []graphQLError `json:"errors"`
	}

	if err := json.Unmarshal(raw, &resp); err != nil {
		return nil, fmt.Errorf("failed to decode create response: %w", err)
	}

	if len(resp.Errors) > 0 {
		return nil, fmt.Errorf("failed to create application: %w", graphQLErrorAsError(resp.Errors[0]))
	}

	return &resp.Data.CreateApplication, nil
}

// ListAPIKeys retrieves API keys for an application.
func (c *GatewayClient) ListAPIKeys(ctx context.Context, appID, region, userToken, orgToken string) ([]domain.GatewayAPIKey, error) {
	query := `query V3_ApiKeys($appId: String!) {
  apiKeys(appId: $appId) {
    id
    name
    status
    permissions
    expiresAt
    createdAt
  }
}`

	variables := map[string]any{
		"appId": appID,
	}

	url := gatewayURL(region)
	raw, err := c.doGraphQL(ctx, url, query, variables, userToken, orgToken)
	if err != nil {
		return nil, fmt.Errorf("failed to list API keys: %w", err)
	}

	var resp struct {
		Data struct {
			APIKeys []domain.GatewayAPIKey `json:"apiKeys"`
		} `json:"data"`
		Errors []graphQLError `json:"errors"`
	}

	if err := json.Unmarshal(raw, &resp); err != nil {
		return nil, fmt.Errorf("failed to decode API keys response: %w", err)
	}

	if len(resp.Errors) > 0 {
		return nil, fmt.Errorf("failed to list API keys: %w", graphQLErrorAsError(resp.Errors[0]))
	}

	return resp.Data.APIKeys, nil
}

// CreateAPIKey creates a new API key for an application.
func (c *GatewayClient) CreateAPIKey(ctx context.Context, appID, region, name string, expiresInDays int, userToken, orgToken string) (*domain.GatewayCreatedAPIKey, error) {
	query := `mutation V3_CreateApiKey($appId: String!, $options: ApiKeyOptions) {
  createApiKey(appId: $appId, options: $options) {
    id
    name
    apiKey
    status
    permissions
    expiresAt
    createdAt
  }
}`

	options := map[string]any{
		"name": name,
	}
	if expiresInDays > 0 {
		options["expiresIn"] = expiresInDays
	}

	variables := map[string]any{
		"appId":   appID,
		"options": options,
	}

	url := gatewayURL(region)
	raw, err := c.doGraphQL(ctx, url, query, variables, userToken, orgToken)
	if err != nil {
		return nil, fmt.Errorf("failed to create API key: %w", err)
	}

	var resp struct {
		Data struct {
			CreateAPIKey domain.GatewayCreatedAPIKey `json:"createApiKey"`
		} `json:"data"`
		Errors []graphQLError `json:"errors"`
	}

	if err := json.Unmarshal(raw, &resp); err != nil {
		return nil, fmt.Errorf("failed to decode create API key response: %w", err)
	}

	if len(resp.Errors) > 0 {
		return nil, fmt.Errorf("failed to create API key: %w", graphQLErrorAsError(resp.Errors[0]))
	}

	return &resp.Data.CreateAPIKey, nil
}

// doGraphQL sends a GraphQL request with auth headers and DPoP proof.
func (c *GatewayClient) doGraphQL(ctx context.Context, url, query string, variables map[string]any, userToken, orgToken string) ([]byte, error) {
	reqBody := map[string]any{
		"query":     query,
		"variables": variables,
	}

	bodyJSON, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to encode GraphQL request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(bodyJSON))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+userToken)
	if orgToken != "" {
		req.Header.Set("X-Nylas-Org", orgToken)
	}

	// Add DPoP proof with access token hash
	proof, err := c.dpop.GenerateProof(http.MethodPost, url, userToken)
	if err != nil {
		return nil, err
	}
	req.Header.Set("DPoP", proof)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBody))
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode >= 300 && resp.StatusCode < 400 {
		location := resp.Header.Get("Location")
		return nil, fmt.Errorf("server redirected to %s — the gateway URL may be incorrect", location)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		if gqlErr := parseGraphQLErrorResponse(resp.StatusCode, respBody); gqlErr != nil {
			return nil, gqlErr
		}
		return nil, parseErrorResponse(resp.StatusCode, respBody)
	}

	return respBody, nil
}

// gatewayURL returns the API gateway GraphQL URL for the given region.
// Per-region env vars take priority, then the shared override, then defaults.
//
//	NYLAS_DASHBOARD_GATEWAY_US_URL  → overrides US only
//	NYLAS_DASHBOARD_GATEWAY_EU_URL  → overrides EU only
//	NYLAS_DASHBOARD_GATEWAY_URL     → overrides both (single local gateway)
func gatewayURL(region string) string {
	if region == "eu" {
		if envURL := os.Getenv("NYLAS_DASHBOARD_GATEWAY_EU_URL"); envURL != "" {
			return envURL
		}
		if envURL := os.Getenv("NYLAS_DASHBOARD_GATEWAY_URL"); envURL != "" {
			return envURL
		}
		return domain.GatewayBaseURLEU
	}
	if envURL := os.Getenv("NYLAS_DASHBOARD_GATEWAY_US_URL"); envURL != "" {
		return envURL
	}
	if envURL := os.Getenv("NYLAS_DASHBOARD_GATEWAY_URL"); envURL != "" {
		return envURL
	}
	return domain.GatewayBaseURLUS
}

// graphQLError represents a GraphQL error from the gateway.
type graphQLError struct {
	Message    string             `json:"message"`
	Extensions *graphQLExtensions `json:"extensions,omitempty"`
}

type graphQLExtensions struct {
	Code    string `json:"code,omitempty"`
	Message string `json:"message,omitempty"`
}

// formatGraphQLError returns a human-readable error from a GraphQL error.
func formatGraphQLError(e graphQLError) string {
	// Prefer extensions.message (more specific), fall back to top-level message
	if e.Extensions != nil && e.Extensions.Message != "" && e.Extensions.Message != e.Message {
		return e.Extensions.Message
	}
	return e.Message
}

func graphQLErrorAsError(e graphQLError) error {
	return graphQLErrorAsErrorWithStatus(http.StatusOK, e)
}

func parseGraphQLErrorResponse(statusCode int, body []byte) error {
	var resp struct {
		Errors []graphQLError `json:"errors"`
	}
	if err := json.Unmarshal(body, &resp); err != nil || len(resp.Errors) == 0 {
		return nil
	}
	return graphQLErrorAsErrorWithStatus(statusCode, resp.Errors[0])
}

func graphQLErrorAsErrorWithStatus(statusCode int, e graphQLError) error {
	message := formatGraphQLError(e)
	if isGraphQLInvalidSession(statusCode, e) {
		return domain.NewDashboardAPIError(http.StatusUnauthorized, "INVALID_SESSION", invalidSessionMessage(e))
	}

	if e.Extensions == nil || e.Extensions.Code == "" {
		return fmt.Errorf("GraphQL error: %s", message)
	}

	return domain.NewDashboardAPIError(statusCode, e.Extensions.Code, message)
}

func isGraphQLInvalidSession(statusCode int, e graphQLError) bool {
	if e.Extensions == nil {
		return false
	}
	if e.Extensions.Code == "INVALID_SESSION" {
		return true
	}
	if statusCode != http.StatusUnauthorized || e.Extensions.Code != "UNAUTHENTICATED" {
		return false
	}

	topLevel := strings.TrimSpace(e.Message)
	extensionMsg := strings.TrimSpace(e.Extensions.Message)
	return strings.EqualFold(topLevel, "INVALID_SESSION") || strings.EqualFold(extensionMsg, "INVALID_SESSION")
}

func invalidSessionMessage(e graphQLError) string {
	if e.Extensions != nil {
		if msg := strings.TrimSpace(e.Extensions.Message); msg != "" && !strings.EqualFold(msg, "INVALID_SESSION") {
			return msg
		}
	}
	if msg := strings.TrimSpace(e.Message); msg != "" && !strings.EqualFold(msg, "INVALID_SESSION") {
		return msg
	}
	return "Invalid or expired session"
}
