package setup

import (
	"testing"

	"github.com/nylas/cli/internal/domain"
)

func TestSanitizeAPIKey(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "clean key",
			input: "nyl_abc123def456",
			want:  "nyl_abc123def456",
		},
		{
			name:  "key with newline",
			input: "nyl_abc123\n",
			want:  "nyl_abc123",
		},
		{
			name:  "key with carriage return",
			input: "nyl_abc123\r\n",
			want:  "nyl_abc123",
		},
		{
			name:  "key with tab",
			input: "\tnyl_abc123\t",
			want:  "nyl_abc123",
		},
		{
			name:  "key with control characters",
			input: "\x00nyl_abc123\x01\x02",
			want:  "nyl_abc123",
		},
		{
			name:  "key with leading/trailing spaces",
			input: "  nyl_abc123  ",
			want:  "nyl_abc123",
		},
		{
			name:  "empty key",
			input: "",
			want:  "",
		},
		{
			name:  "only whitespace",
			input: "  \n\r\t  ",
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sanitizeAPIKey(tt.input)
			if got != tt.want {
				t.Errorf("sanitizeAPIKey(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestAppDisplayName(t *testing.T) {
	tests := []struct {
		name string
		app  appDisplayInput
		want string
	}{
		{
			name: "basic app",
			app: appDisplayInput{
				appID:       "abc123",
				environment: "production",
				region:      "us",
			},
			want: "abc123 (production, us)",
		},
		{
			name: "app with name",
			app: appDisplayInput{
				appID:       "abc123",
				environment: "production",
				region:      "us",
				brandName:   "My App",
			},
			want: "My App — abc123 (production, us)",
		},
		{
			name: "long app ID truncated",
			app: appDisplayInput{
				appID:       "abcdefghij1234567890xyz",
				environment: "production",
				region:      "eu",
			},
			want: "abcdefghij1234567... (production, eu)",
		},
		{
			name: "empty environment defaults to production",
			app: appDisplayInput{
				appID:       "abc123",
				environment: "",
				region:      "us",
			},
			want: "abc123 (production, us)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := tt.app.toGatewayApp()
			got := appDisplayName(app)
			if got != tt.want {
				t.Errorf("appDisplayName() = %q, want %q", got, tt.want)
			}
		})
	}
}

// appDisplayInput is a test helper for constructing GatewayApplication.
type appDisplayInput struct {
	appID       string
	environment string
	region      string
	brandName   string
}

func (a appDisplayInput) toGatewayApp() domain.GatewayApplication {
	app := domain.GatewayApplication{
		ApplicationID: a.appID,
		Environment:   a.environment,
		Region:        a.region,
	}
	if a.brandName != "" {
		app.Branding = &domain.GatewayApplicationBrand{Name: a.brandName}
	}
	return app
}

func TestResolveProvider(t *testing.T) {
	tests := []struct {
		name string
		opts wizardOpts
		want string
	}{
		{
			name: "google flag",
			opts: wizardOpts{google: true},
			want: "google",
		},
		{
			name: "microsoft flag",
			opts: wizardOpts{microsoft: true},
			want: "microsoft",
		},
		{
			name: "github flag",
			opts: wizardOpts{github: true},
			want: "github",
		},
		{
			name: "no flags",
			opts: wizardOpts{},
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolveProvider(tt.opts)
			if got != tt.want {
				t.Errorf("resolveProvider() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSetupStatus(t *testing.T) {
	// Test that a zero-value SetupStatus has all fields false.
	status := SetupStatus{}
	if status.HasDashboardAuth {
		t.Error("zero SetupStatus.HasDashboardAuth should be false")
	}
	if status.HasAPIKey {
		t.Error("zero SetupStatus.HasAPIKey should be false")
	}
	if status.HasActiveApp {
		t.Error("zero SetupStatus.HasActiveApp should be false")
	}
	if status.HasGrants {
		t.Error("zero SetupStatus.HasGrants should be false")
	}
	if status.ActiveAppID != "" {
		t.Error("zero SetupStatus.ActiveAppID should be empty")
	}
	if status.ActiveAppRegion != "" {
		t.Error("zero SetupStatus.ActiveAppRegion should be empty")
	}
}
