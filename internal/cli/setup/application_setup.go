package setup

import (
	"context"
	"fmt"
	"strings"

	nylasadapter "github.com/nylas/cli/internal/adapters/nylas"
	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/domain"
)

type apiKeySetupClient interface {
	ListApplications(ctx context.Context) ([]domain.Application, error)
	ListCallbackURIs(ctx context.Context) ([]domain.CallbackURI, error)
	CreateCallbackURI(ctx context.Context, req *domain.CreateCallbackURIRequest) (*domain.CallbackURI, error)
}

// APIKeyApplication identifies the application that should be used with a
// provided API key during setup flows.
type APIKeyApplication struct {
	ClientID string
	OrgID    string
}

// CallbackURIProvisionResult reports whether the expected OAuth callback URI
// already existed or was created during setup.
type CallbackURIProvisionResult struct {
	RequiredURI   string
	AlreadyExists bool
	Created       bool
}

func newAPIKeySetupClient(region, clientID, apiKey string) apiKeySetupClient {
	client := nylasadapter.NewHTTPClient()
	client.SetRegion(region)
	client.SetCredentials(clientID, "", apiKey)
	return client
}

// AppClientID returns the stable client/application ID for a Nylas application.
func AppClientID(app domain.Application) string {
	if app.ApplicationID != "" {
		return app.ApplicationID
	}
	return app.ID
}

// AppDisplayName returns a human-readable display name for a Nylas application.
func AppDisplayName(app domain.Application) string {
	clientID := AppClientID(app)
	env := app.Environment
	if env == "" {
		env = "production"
	}

	region := app.Region
	if region == "" {
		region = "us"
	}

	displayID := common.Truncate(clientID, 20)

	return fmt.Sprintf("%s (%s, %s)", displayID, env, region)
}

// ResolveAPIKeyApplication selects the application that should back an API-key
// based setup flow. In non-interactive mode, multiple applications require an
// explicit client ID.
func ResolveAPIKeyApplication(apiKey, region, explicitClientID string, interactive bool) (*APIKeyApplication, error) {
	return resolveAPIKeyApplication(apiKey, region, explicitClientID, interactive, newAPIKeySetupClient)
}

func resolveAPIKeyApplication(
	apiKey, region, explicitClientID string,
	interactive bool,
	clientFactory func(region, clientID, apiKey string) apiKeySetupClient,
) (*APIKeyApplication, error) {
	explicitClientID = strings.TrimSpace(explicitClientID)

	client := clientFactory(region, "", apiKey)

	ctx, cancel := common.CreateContext()
	defer cancel()

	apps, err := client.ListApplications(ctx)
	if err != nil {
		return nil, fmt.Errorf("could not auto-detect application: %w", err)
	}
	if len(apps) == 0 {
		return nil, fmt.Errorf("no applications found for this API key")
	}

	if explicitClientID != "" {
		for _, app := range apps {
			if AppClientID(app) == explicitClientID {
				return &APIKeyApplication{
					ClientID: explicitClientID,
					OrgID:    app.OrganizationID,
				}, nil
			}
		}
		return nil, fmt.Errorf("client ID %q was not found for this API key", explicitClientID)
	}

	if len(apps) == 1 {
		app := apps[0]
		return &APIKeyApplication{
			ClientID: AppClientID(app),
			OrgID:    app.OrganizationID,
		}, nil
	}

	if !interactive {
		return nil, fmt.Errorf("multiple applications found for this API key; rerun with --client-id or use interactive setup")
	}

	options := make([]common.SelectOption[int], len(apps))
	for i, app := range apps {
		options[i] = common.SelectOption[int]{
			Label: AppDisplayName(app),
			Value: i,
		}
	}

	selectedIndex, err := common.Select("Select application", options)
	if err != nil {
		return nil, err
	}

	selected := apps[selectedIndex]
	return &APIKeyApplication{
		ClientID: AppClientID(selected),
		OrgID:    selected.OrganizationID,
	}, nil
}

// EnsureOAuthCallbackURI provisions the expected loopback callback URI for the
// selected application when it does not already exist.
func EnsureOAuthCallbackURI(apiKey, clientID, region string, callbackPort int) (*CallbackURIProvisionResult, error) {
	return ensureOAuthCallbackURI(apiKey, clientID, region, callbackPort, newAPIKeySetupClient)
}

func ensureOAuthCallbackURI(
	apiKey, clientID, region string,
	callbackPort int,
	clientFactory func(region, clientID, apiKey string) apiKeySetupClient,
) (*CallbackURIProvisionResult, error) {
	if callbackPort == 0 {
		callbackPort = 9007
	}

	result := &CallbackURIProvisionResult{
		RequiredURI: fmt.Sprintf("http://localhost:%d/callback", callbackPort),
	}
	if strings.TrimSpace(clientID) == "" {
		return result, fmt.Errorf("client ID is required to configure the OAuth callback URI")
	}

	client := clientFactory(region, clientID, apiKey)

	ctx, cancel := common.CreateContext()
	existingURIs, listErr := client.ListCallbackURIs(ctx)
	cancel()

	if listErr == nil {
		for _, cb := range existingURIs {
			if cb.URL == result.RequiredURI {
				result.AlreadyExists = true
				return result, nil
			}
		}
	}

	ctx, cancel = common.CreateContext()
	_, createErr := client.CreateCallbackURI(ctx, &domain.CreateCallbackURIRequest{
		URL:      result.RequiredURI,
		Platform: "web",
	})
	cancel()
	if createErr != nil {
		return result, createErr
	}

	result.Created = true
	return result, nil
}
