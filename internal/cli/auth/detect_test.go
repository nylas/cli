package auth

import (
	"bytes"
	"strings"
	"testing"

	"github.com/nylas/cli/internal/domain"
)

func TestDetectCmd(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		wantOutput []string
		wantErr    bool
	}{
		{
			name:       "detect gmail",
			args:       []string{"user@gmail.com"},
			wantOutput: []string{"gmail.com", "google"},
			wantErr:    false,
		},
		{
			name:       "detect outlook",
			args:       []string{"user@outlook.com"},
			wantOutput: []string{"outlook.com", "microsoft"},
			wantErr:    false,
		},
		{
			name:       "detect icloud",
			args:       []string{"user@icloud.com"},
			wantOutput: []string{"icloud.com", "icloud"},
			wantErr:    false,
		},
		{
			name:       "detect yahoo",
			args:       []string{"user@yahoo.com"},
			wantOutput: []string{"yahoo.com", "yahoo"},
			wantErr:    false,
		},
		{
			name:       "detect custom domain",
			args:       []string{"user@company.com"},
			wantOutput: []string{"company.com", "imap"},
			wantErr:    false,
		},
		{
			name:       "detect json output",
			args:       []string{"user@gmail.com", "--json"},
			wantOutput: []string{"gmail.com", "google"},
			wantErr:    false,
		},
		{
			name:       "invalid email",
			args:       []string{"notanemail"},
			wantOutput: []string{},
			wantErr:    true,
		},
		{
			name:       "no args",
			args:       []string{},
			wantOutput: []string{},
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := newDetectCmd()
			buf := new(bytes.Buffer)
			cmd.SetOut(buf)
			cmd.SetErr(buf)
			cmd.SetArgs(tt.args)

			err := cmd.Execute()
			if (err != nil) != tt.wantErr {
				t.Errorf("newDetectCmd() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				output := buf.String()
				for _, want := range tt.wantOutput {
					if !strings.Contains(output, want) {
						t.Errorf("newDetectCmd() output = %v, want to contain %v", output, want)
					}
				}
			}
		})
	}
}

func TestDetectProvider(t *testing.T) {
	tests := []struct {
		domain   string
		expected domain.Provider
	}{
		{"gmail.com", domain.ProviderGoogle},
		{"googlemail.com", domain.ProviderGoogle},
		{"outlook.com", domain.ProviderMicrosoft},
		{"hotmail.com", domain.ProviderMicrosoft},
		{"live.com", domain.ProviderMicrosoft},
		{"icloud.com", "icloud"},
		{"me.com", "icloud"},
		{"yahoo.com", "yahoo"},
		{"yahoo.co.uk", "yahoo"},
		{"ymail.com", "yahoo"},
		{"company.com", domain.ProviderIMAP},
		{"example.org", domain.ProviderIMAP},
	}

	for _, tt := range tests {
		t.Run(tt.domain, func(t *testing.T) {
			result := detectProvider(tt.domain)
			if result != tt.expected {
				t.Errorf("detectProvider(%s) = %v, want %v", tt.domain, result, tt.expected)
			}
		})
	}
}

func TestIsGoogleDomain(t *testing.T) {
	tests := []struct {
		domain   string
		expected bool
	}{
		{"gmail.com", true},
		{"googlemail.com", true},
		{"google.com", true},
		{"outlook.com", false},
		{"company.com", false},
	}

	for _, tt := range tests {
		t.Run(tt.domain, func(t *testing.T) {
			result := isGoogleDomain(tt.domain)
			if result != tt.expected {
				t.Errorf("isGoogleDomain(%s) = %v, want %v", tt.domain, result, tt.expected)
			}
		})
	}
}

func TestIsMicrosoftDomain(t *testing.T) {
	tests := []struct {
		domain   string
		expected bool
	}{
		{"outlook.com", true},
		{"hotmail.com", true},
		{"live.com", true},
		{"msn.com", true},
		{"microsoft.com", true},
		{"gmail.com", false},
		{"company.com", false},
	}

	for _, tt := range tests {
		t.Run(tt.domain, func(t *testing.T) {
			result := isMicrosoftDomain(tt.domain)
			if result != tt.expected {
				t.Errorf("isMicrosoftDomain(%s) = %v, want %v", tt.domain, result, tt.expected)
			}
		})
	}
}

func TestIsICloudDomain(t *testing.T) {
	tests := []struct {
		domain   string
		expected bool
	}{
		{"icloud.com", true},
		{"me.com", true},
		{"mac.com", true},
		{"gmail.com", false},
		{"company.com", false},
	}

	for _, tt := range tests {
		t.Run(tt.domain, func(t *testing.T) {
			result := isICloudDomain(tt.domain)
			if result != tt.expected {
				t.Errorf("isICloudDomain(%s) = %v, want %v", tt.domain, result, tt.expected)
			}
		})
	}
}

func TestIsYahooDomain(t *testing.T) {
	tests := []struct {
		domain   string
		expected bool
	}{
		{"yahoo.com", true},
		{"yahoo.co.uk", true},
		{"yahoo.fr", true},
		{"ymail.com", true},
		{"rocketmail.com", true},
		{"gmail.com", false},
		{"company.com", false},
	}

	for _, tt := range tests {
		t.Run(tt.domain, func(t *testing.T) {
			result := isYahooDomain(tt.domain)
			if result != tt.expected {
				t.Errorf("isYahooDomain(%s) = %v, want %v", tt.domain, result, tt.expected)
			}
		})
	}
}
