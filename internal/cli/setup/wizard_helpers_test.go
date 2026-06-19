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

func TestDomainRegistrationCommands(t *testing.T) {
	tests := []struct {
		name   string
		status SetupStatus
		want   []string
	}{
		{
			name:   "uses US active app region",
			status: SetupStatus{ActiveAppRegion: "us"},
			want: []string{
				"nylas dashboard domains create <subdomain>.nylas.email --region us",
			},
		},
		{
			name:   "uses EU active app region",
			status: SetupStatus{ActiveAppRegion: "eu"},
			want: []string{
				"nylas dashboard domains create <subdomain>.nylas.email --region eu",
			},
		},
		{
			name:   "falls back to copy-pasteable commands when active app region is unknown",
			status: SetupStatus{},
			want: []string{
				"nylas dashboard domains create <subdomain>.nylas.email --region us",
				"nylas dashboard domains create <subdomain>.nylas.email --region eu",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := domainRegistrationCommands(tt.status)
			if len(got) != len(tt.want) {
				t.Fatalf("expected %d commands, got %d: %v", len(tt.want), len(got), got)
			}
			for i := range tt.want {
				if got[i] != tt.want[i] {
					t.Fatalf("expected command %d to be %q, got %q", i, tt.want[i], got[i])
				}
			}
		})
	}
}
