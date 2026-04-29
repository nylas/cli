package auth

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/nylas/cli/internal/adapters/config"
	"github.com/nylas/cli/internal/adapters/keyring"
	"github.com/nylas/cli/internal/cli/testutil"
	"github.com/nylas/cli/internal/domain"
	"github.com/nylas/cli/internal/ports"
)

// hasAPIKey checks if API key is configured (from env or keyring)
func hasAPIKey() bool {
	// Check environment variable first
	if os.Getenv("NYLAS_API_KEY") != "" {
		return true
	}

	// Check keyring
	secretStore, err := keyring.NewSecretStore(config.DefaultConfigDir())
	if err != nil {
		return false
	}

	_, err = secretStore.Get(ports.KeyAPIKey)
	return err == nil
}

func TestProvidersCmd(t *testing.T) {
	// Skip in short mode - this test requires valid API credentials
	if testing.Short() {
		t.Skip("Skipping API test in short mode")
	}

	// Skip if API key is not configured
	if !hasAPIKey() {
		t.Skip("API key not configured - run 'nylas auth config' to set it")
	}

	tests := []struct {
		name       string
		args       []string
		wantOutput []string
		wantErr    bool
	}{
		{
			name:       "list providers",
			args:       []string{},
			wantOutput: []string{"Available Authentication Providers"},
			wantErr:    false,
		},
		{
			name:       "list providers json",
			args:       []string{"--json"},
			wantOutput: []string{},
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stdout, stderr, err := testutil.ExecuteSubCommand(newProvidersCmd(), tt.args...)
			if (err != nil) != tt.wantErr {
				t.Errorf("newProvidersCmd() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			output := stdout + stderr
			for _, want := range tt.wantOutput {
				if !strings.Contains(output, want) {
					t.Errorf("newProvidersCmd() output = %v, want to contain %v", output, want)
				}
			}
		})
	}
}

func TestRenderProviders(t *testing.T) {
	tests := []struct {
		name        string
		connectors  []domain.Connector
		wantContain []string
		wantAbsent  []string
	}{
		{
			name: "omits empty connector fields",
			connectors: []domain.Connector{
				{
					Provider: "google",
					Settings: &domain.ConnectorSettings{ClientID: "client-id"},
				},
			},
			wantContain: []string{
				"Available Authentication Providers:",
				"  Google",
				"    Provider:   google",
			},
			wantAbsent: []string{
				"Name:       ",
				"ID:         ",
			},
		},
		{
			name: "prints populated connector metadata",
			connectors: []domain.Connector{
				{
					ID:       "conn-imap-1",
					Name:     "Custom IMAP",
					Provider: "imap",
					Scopes:   []string{"mail.read_only", "mail.send"},
				},
			},
			wantContain: []string{
				"  Custom IMAP",
				"    Provider:   imap",
				"    ID:         conn-imap-1",
				"    Scopes:     2 configured",
			},
		},
		{
			name:       "shows empty state",
			connectors: nil,
			wantContain: []string{
				"No providers configured.",
				"nylas admin connectors create",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			renderProviders(&buf, tt.connectors)

			output := buf.String()
			for _, want := range tt.wantContain {
				if !strings.Contains(output, want) {
					t.Fatalf("renderProviders() output = %q, want to contain %q", output, want)
				}
			}
			for _, unwanted := range tt.wantAbsent {
				if strings.Contains(output, unwanted) {
					t.Fatalf("renderProviders() output = %q, should not contain %q", output, unwanted)
				}
			}
		})
	}
}

func TestProviderDisplayName(t *testing.T) {
	tests := []struct {
		provider string
		want     string
	}{
		{provider: "google", want: "Google"},
		{provider: "microsoft", want: "Microsoft"},
		{provider: "imap", want: "IMAP"},
		{provider: "icloud", want: "iCloud"},
		{provider: "ews", want: "EWS"},
		{provider: "virtual-calendar", want: "Virtual Calendar"},
		{provider: "custom-provider", want: "Custom Provider"},
	}

	for _, tt := range tests {
		t.Run(tt.provider, func(t *testing.T) {
			if got := providerDisplayName(tt.provider); got != tt.want {
				t.Fatalf("providerDisplayName(%q) = %q, want %q", tt.provider, got, tt.want)
			}
		})
	}
}
