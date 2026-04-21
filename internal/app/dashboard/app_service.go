package dashboard

import (
	"context"
	"fmt"
	"sync"

	"github.com/nylas/cli/internal/domain"
	"github.com/nylas/cli/internal/ports"
)

// AppService handles application management via the dashboard API gateway.
type AppService struct {
	gateway ports.DashboardGatewayClient
	secrets ports.SecretStore
}

// NewAppService creates a new application management service.
func NewAppService(gateway ports.DashboardGatewayClient, secrets ports.SecretStore) *AppService {
	return &AppService{
		gateway: gateway,
		secrets: secrets,
	}
}

// ListApplications retrieves applications from both US and EU regions in parallel.
// If regionFilter is non-empty, only that region is queried.
func (s *AppService) ListApplications(ctx context.Context, orgPublicID, regionFilter string) ([]domain.GatewayApplication, error) {
	userToken, orgToken, err := s.loadTokens()
	if err != nil {
		return nil, err
	}

	if regionFilter != "" {
		return s.gateway.ListApplications(ctx, orgPublicID, regionFilter, userToken, orgToken)
	}

	// Query both regions in parallel
	type result struct {
		region string
		apps   []domain.GatewayApplication
		err    error
	}

	var wg sync.WaitGroup
	results := make([]result, 2)
	regions := []string{"us", "eu"}

	for i, region := range regions {
		wg.Add(1)
		go func(idx int, r string) {
			defer wg.Done()
			apps, err := s.gateway.ListApplications(ctx, orgPublicID, r, userToken, orgToken)
			results[idx] = result{region: r, apps: apps, err: err}
		}(i, region)
	}
	wg.Wait()

	var allApps []domain.GatewayApplication
	failures := make(map[string]error)
	for _, r := range results {
		if r.err != nil {
			failures[r.region] = r.err
			continue
		}
		allApps = append(allApps, r.apps...)
	}

	// If both failed, return the first error
	if len(failures) == len(regions) {
		for _, region := range regions {
			if err := failures[region]; err != nil {
				return nil, fmt.Errorf("failed to list applications: %w", err)
			}
		}
	}

	allApps = deduplicateApps(allApps)
	if len(failures) > 0 {
		return allApps, &domain.DashboardPartialResultError{
			Operation: "application list",
			Failures:  failures,
		}
	}

	return allApps, nil
}

// CreateApplication creates a new application in the specified region.
func (s *AppService) CreateApplication(ctx context.Context, orgPublicID, region, name string) (*domain.GatewayCreatedApplication, error) {
	userToken, orgToken, err := s.loadTokens()
	if err != nil {
		return nil, err
	}

	return s.gateway.CreateApplication(ctx, orgPublicID, region, name, userToken, orgToken)
}

// ListAPIKeys retrieves API keys for an application.
func (s *AppService) ListAPIKeys(ctx context.Context, appID, region string) ([]domain.GatewayAPIKey, error) {
	userToken, orgToken, err := s.loadTokens()
	if err != nil {
		return nil, err
	}

	return s.gateway.ListAPIKeys(ctx, appID, region, userToken, orgToken)
}

// CreateAPIKey creates a new API key for an application.
func (s *AppService) CreateAPIKey(ctx context.Context, appID, region, name string, expiresInDays int) (*domain.GatewayCreatedAPIKey, error) {
	userToken, orgToken, err := s.loadTokens()
	if err != nil {
		return nil, err
	}

	return s.gateway.CreateAPIKey(ctx, appID, region, name, expiresInDays, userToken, orgToken)
}

// deduplicateApps removes duplicate applications (same applicationId).
func deduplicateApps(apps []domain.GatewayApplication) []domain.GatewayApplication {
	seen := make(map[string]bool, len(apps))
	out := make([]domain.GatewayApplication, 0, len(apps))
	for _, app := range apps {
		key := app.ApplicationID
		if key == "" {
			// Use a composite key for apps without an ID
			key = app.Region + ":" + app.Environment + ":"
			if app.Branding != nil {
				key += app.Branding.Name
			}
		}
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, app)
	}
	return out
}

// loadTokens retrieves the stored dashboard tokens.
func (s *AppService) loadTokens() (userToken, orgToken string, err error) {
	return loadDashboardTokens(s.secrets)
}
