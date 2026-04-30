package setup

import (
	"errors"
	"testing"
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
