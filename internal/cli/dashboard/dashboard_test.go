package dashboard

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	dashboardadapter "github.com/nylas/cli/internal/adapters/dashboard"
	dashboardapp "github.com/nylas/cli/internal/app/dashboard"
	"github.com/nylas/cli/internal/domain"
	"github.com/nylas/cli/internal/ports"
)

type memSecretStore struct {
	data map[string]string
}

func newMemSecretStore() *memSecretStore {
	return &memSecretStore{data: make(map[string]string)}
}

func (m *memSecretStore) Set(key, value string) error {
	m.data[key] = value
	return nil
}

func (m *memSecretStore) Get(key string) (string, error) {
	return m.data[key], nil
}

func (m *memSecretStore) Delete(key string) error {
	delete(m.data, key)
	return nil
}

func (m *memSecretStore) IsAvailable() bool { return true }

func (m *memSecretStore) Name() string { return "mem" }

func TestResolveAuthMethod(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		google    bool
		microsoft bool
		github    bool
		email     bool
		action    string
		want      string
		wantErr   string
	}{
		{
			name:   "google flag wins",
			google: true,
			action: "log in",
			want:   methodGoogle,
		},
		{
			name:      "microsoft flag wins",
			microsoft: true,
			action:    "log in",
			want:      methodMicrosoft,
		},
		{
			name:   "github flag wins",
			github: true,
			action: "log in",
			want:   methodGitHub,
		},
		{
			name:   "email login is allowed",
			email:  true,
			action: "log in",
			want:   methodEmailPassword,
		},
		{
			name:    "email registration is rejected",
			email:   true,
			action:  "register",
			wantErr: "temporarily disabled",
		},
		{
			name:    "multiple flags are rejected",
			google:  true,
			github:  true,
			action:  "log in",
			wantErr: "only one auth method flag allowed",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := resolveAuthMethod(tt.google, tt.microsoft, tt.github, tt.email, tt.action)

			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				assert.Empty(t, got)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestGetDashboardAccountBaseURL(t *testing.T) {
	t.Parallel()

	origURL := os.Getenv("NYLAS_DASHBOARD_ACCOUNT_URL")
	defer func() {
		if origURL == "" {
			_ = os.Unsetenv("NYLAS_DASHBOARD_ACCOUNT_URL")
			return
		}
		_ = os.Setenv("NYLAS_DASHBOARD_ACCOUNT_URL", origURL)
	}()

	require.NoError(t, os.Setenv("NYLAS_DASHBOARD_ACCOUNT_URL", "https://dashboard.example.com"))
	assert.Equal(t, "https://dashboard.example.com", getDashboardAccountBaseURL(nil))
}

func TestMapProvider(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		provider string
		want     string
		wantErr  string
	}{
		{name: "google", provider: "google", want: domain.SSOLoginTypeGoogle},
		{name: "microsoft", provider: "microsoft", want: domain.SSOLoginTypeMicrosoft},
		{name: "github", provider: "github", want: domain.SSOLoginTypeGitHub},
		{name: "case insensitive", provider: "GitHub", want: domain.SSOLoginTypeGitHub},
		{name: "unsupported", provider: "okta", wantErr: "unsupported SSO provider"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := mapProvider(tt.provider)

			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				assert.Empty(t, got)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestPresentAbsent(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "present", presentAbsent(true))
	assert.Equal(t, "absent", presentAbsent(false))
}

func TestFormatOrgLabel(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "Acme (org-123)", formatOrgLabel("org-123", "Acme"))
	assert.Equal(t, "org-123", formatOrgLabel("org-123", ""))
}

func TestFormatSessionOrg(t *testing.T) {
	t.Parallel()

	session := &domain.DashboardSessionResponse{
		CurrentOrg: "org-1",
		Relations: []domain.DashboardSessionRelation{
			{OrgPublicID: "org-1", OrgName: "Acme"},
			{OrgPublicID: "org-2", OrgName: "Beta"},
		},
	}

	assert.Equal(t, "Acme (org-1)", formatSessionOrg(session, "org-1"))
	assert.Equal(t, "org-missing", formatSessionOrg(session, "org-missing"))
}

func TestToAppRows(t *testing.T) {
	t.Parallel()

	apps := []domain.GatewayApplication{
		{
			ApplicationID: "app-1",
			Region:        "us",
			Environment:   "",
			Branding:      &domain.GatewayApplicationBrand{Name: "Primary"},
		},
		{
			ApplicationID: "app-2",
			Region:        "eu",
			Environment:   "sandbox",
		},
	}

	rows := toAppRows(apps)

	require.Len(t, rows, 2)
	assert.Equal(t, appRow{
		ApplicationID: "app-1",
		Region:        "us",
		Environment:   "production",
		Name:          "Primary",
	}, rows[0])
	assert.Equal(t, appRow{
		ApplicationID: "app-2",
		Region:        "eu",
		Environment:   "sandbox",
		Name:          "",
	}, rows[1])
}

func TestPersistActiveOrgSwitchesServerSession(t *testing.T) {
	t.Parallel()

	store := newMemSecretStore()
	require.NoError(t, store.Set(ports.KeyDashboardUserToken, "user-token"))
	require.NoError(t, store.Set(ports.KeyDashboardOrgToken, "org-token"))

	var switchedOrg string
	authSvc := dashboardapp.NewAuthService(&dashboardadapter.MockAccountClient{
		SwitchOrgFn: func(_ context.Context, orgPublicID, userToken, orgToken string) (*domain.DashboardSwitchOrgResponse, error) {
			switchedOrg = orgPublicID
			assert.Equal(t, "user-token", userToken)
			assert.Equal(t, "org-token", orgToken)
			return &domain.DashboardSwitchOrgResponse{
				OrgToken: "new-org-token",
				Org:      domain.DashboardSwitchOrgOrg{PublicID: orgPublicID},
			}, nil
		},
	}, store)

	err := persistActiveOrg(authSvc, &domain.DashboardAuthResponse{
		Organizations: []domain.DashboardOrganization{
			{PublicID: "org-1", Name: "Org One"},
			{PublicID: "org-2", Name: "Org Two"},
		},
	}, "")

	require.NoError(t, err)
	assert.Equal(t, "org-1", switchedOrg, "non-interactive selection should fall back to the first org")

	storedOrgID, _ := store.Get(ports.KeyDashboardOrgPublicID)
	assert.Equal(t, "org-1", storedOrgID)
	storedOrgToken, _ := store.Get(ports.KeyDashboardOrgToken)
	assert.Equal(t, "new-org-token", storedOrgToken)
}
