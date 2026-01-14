package auth

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/nylas/cli/internal/adapters/config"
	"github.com/nylas/cli/internal/adapters/keyring"
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
			cmd := newProvidersCmd()
			buf := new(bytes.Buffer)
			cmd.SetOut(buf)
			cmd.SetErr(buf)
			cmd.SetArgs(tt.args)

			err := cmd.Execute()
			if (err != nil) != tt.wantErr {
				t.Errorf("newProvidersCmd() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			output := buf.String()
			for _, want := range tt.wantOutput {
				if !strings.Contains(output, want) {
					t.Errorf("newProvidersCmd() output = %v, want to contain %v", output, want)
				}
			}
		})
	}
}
