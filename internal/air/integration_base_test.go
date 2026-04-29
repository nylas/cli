//go:build integration
// +build integration

package air

import (
	"strconv"
	"strings"
	"testing"

	"github.com/nylas/cli/internal/adapters/config"
	"github.com/nylas/cli/internal/adapters/keyring"
	"github.com/nylas/cli/internal/adapters/nylas"
	authapp "github.com/nylas/cli/internal/app/auth"
	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/domain"
	"github.com/nylas/cli/internal/ports"
)

// testServer creates a real Air server with actual credentials for integration testing.
func testServer(t *testing.T) *Server {
	t.Helper()

	configStore := config.NewDefaultFileStore()
	secretStore, err := keyring.NewSecretStore(config.DefaultConfigDir())
	if err != nil {
		t.Skipf("Skipping: cannot access secret store: %v", err)
	}

	grantStore, err := common.NewDefaultGrantStore()
	if err != nil {
		t.Skipf("Skipping: cannot access grant store: %v", err)
	}
	configSvc := authapp.NewConfigService(configStore, secretStore)

	// Check configuration
	status, err := configSvc.GetStatus()
	if err != nil || !status.IsConfigured {
		t.Skip("Skipping: Nylas CLI not configured. Run 'nylas auth login' first.")
	}

	// Check for default grant
	defaultGrantID, err := grantStore.GetDefaultGrant()
	if err != nil || defaultGrantID == "" {
		t.Skip("Skipping: No default grant configured. Run 'nylas auth login' first.")
	}

	// Check that default grant is Google provider
	grants, err := grantStore.ListGrants()
	if err != nil {
		t.Skipf("Skipping: cannot list grants: %v", err)
	}

	var defaultGrant *domain.GrantInfo
	for i := range grants {
		if grants[i].ID == defaultGrantID {
			defaultGrant = &grants[i]
			break
		}
	}

	if defaultGrant == nil {
		t.Skip("Skipping: default grant not found in grant list")
	}

	if defaultGrant.Provider != domain.ProviderGoogle {
		t.Skipf("Skipping: default grant is %s, not Google. These tests require a Google account as default.", defaultGrant.Provider)
	}

	t.Logf("Running integration tests with Google account: %s", defaultGrant.Email)

	// Create Nylas client
	cfg, err := configStore.Load()
	if err != nil {
		t.Skipf("Skipping: cannot load config: %v", err)
	}

	apiKey, _ := secretStore.Get(ports.KeyAPIKey)
	clientID, _ := secretStore.Get(ports.KeyClientID)
	clientSecret, _ := secretStore.Get(ports.KeyClientSecret)

	if apiKey == "" {
		t.Skip("Skipping: no API key configured")
	}

	client := nylas.NewHTTPClient()
	client.SetRegion(cfg.Region)
	client.SetCredentials(clientID, clientSecret, apiKey)

	// Load templates
	tmpl, err := loadTemplates()
	if err != nil {
		t.Fatalf("failed to load templates: %v", err)
	}

	return &Server{
		addr:        ":7365",
		demoMode:    false,
		configSvc:   configSvc,
		configStore: configStore,
		secretStore: secretStore,
		grantStore:  grantStore,
		nylasClient: client,
		templates:   tmpl,
	}
}

// Helper functions used across integration tests

func formatInt64(n int64) string {
	return strconv.FormatInt(n, 10)
}

func containsAny(s string, substrings ...string) bool {
	for _, sub := range substrings {
		if strings.Contains(s, sub) {
			return true
		}
	}
	return false
}
