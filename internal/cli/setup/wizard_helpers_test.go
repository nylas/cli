package setup

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nylas/cli/internal/adapters/config"
	"github.com/nylas/cli/internal/domain"
)

func TestEnsureSetupCallbackURI_AllowsManualFallbackWhenProvisioningFails(t *testing.T) {
	originalProvisioner := setupCallbackProvisioner
	t.Cleanup(func() {
		setupCallbackProvisioner = originalProvisioner
	})

	setupCallbackProvisioner = func(apiKey, clientID, region string, callbackPort int) (*CallbackURIProvisionResult, error) {
		return &CallbackURIProvisionResult{
			RequiredURI: "http://localhost:9007/callback",
		}, errors.New("admin api unavailable")
	}

	if err := ensureSetupCallbackURI("nyl_test", "client-123", "us"); err != nil {
		t.Fatalf("expected setup callback URI failure to degrade gracefully, got %v", err)
	}
}

func TestEnsureSetupCallbackURI_RequiresClientID(t *testing.T) {
	if err := ensureSetupCallbackURI("nyl_test", "", "us"); err == nil {
		t.Fatal("expected empty client ID to fail")
	}
}

func TestUpdateConfigGrantsStoresDefaultOnly(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.yaml")
	configStore := config.NewFileStore(configPath)
	cfg := &domain.Config{Region: "us"}
	result := &SyncResult{
		DefaultGrantID: "grant-1",
		ValidGrants: []domain.Grant{{
			ID:       "grant-1",
			Email:    "user@example.com",
			Provider: domain.ProviderGoogle,
		}},
	}

	updateConfigGrants(configStore, cfg, result)

	if cfg.DefaultGrant != "grant-1" {
		t.Fatalf("DefaultGrant = %q, want grant-1", cfg.DefaultGrant)
	}
	if len(cfg.Grants) != 0 {
		t.Fatalf("config object should not retain grant metadata, got %+v", cfg.Grants)
	}
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("failed to read config: %v", err)
	}
	if strings.Contains(string(data), "grants:") {
		t.Fatalf("saved config should not contain grants list:\n%s", string(data))
	}
}
