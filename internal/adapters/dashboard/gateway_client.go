package dashboard

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

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
		httpClient: &http.Client{},
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
		return nil, fmt.Errorf("GraphQL error: %s", resp.Errors[0].Message)
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
		return nil, fmt.Errorf("GraphQL error: %s", resp.Errors[0].Message)
	}

	return &resp.Data.CreateApplication, nil
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

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
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
	Message string `json:"message"`
}
