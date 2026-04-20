package dashboard

import (
	"context"
	"os"
	"runtime"
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
		{
			name:    "non-interactive login requires explicit auth method",
			action:  "log in",
			wantErr: "auth method is required",
		},
		{
			name:    "non-interactive register requires explicit auth method",
			action:  "register",
			wantErr: "auth method is required",
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

func TestAcceptPrivacyPolicy(t *testing.T) {
	t.Parallel()

	t.Run("accepted flag skips prompt", func(t *testing.T) {
		t.Parallel()

		require.NoError(t, acceptPrivacyPolicy(true))
	})

	t.Run("non-interactive mode requires explicit acceptance", func(t *testing.T) {
		t.Parallel()

		err := acceptPrivacyPolicy(false)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "privacy policy must be accepted")
		assert.Contains(t, err.Error(), "--accept-privacy-policy")
	})
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
	}, "org-2")

	require.NoError(t, err)
	assert.Equal(t, "org-2", switchedOrg)

	storedOrgID, _ := store.Get(ports.KeyDashboardOrgPublicID)
	assert.Equal(t, "org-2", storedOrgID)
	storedOrgToken, _ := store.Get(ports.KeyDashboardOrgToken)
	assert.Equal(t, "new-org-token", storedOrgToken)
}

func TestPersistActiveOrgRejectsNonInteractiveMultiOrgSelection(t *testing.T) {
	t.Parallel()

	store := newMemSecretStore()
	require.NoError(t, store.Set(ports.KeyDashboardUserToken, "user-token"))
	require.NoError(t, store.Set(ports.KeyDashboardOrgToken, "org-token"))

	authSvc := dashboardapp.NewAuthService(&dashboardadapter.MockAccountClient{}, store)

	err := persistActiveOrg(authSvc, &domain.DashboardAuthResponse{
		Organizations: []domain.DashboardOrganization{
			{PublicID: "org-1", Name: "Org One"},
			{PublicID: "org-2", Name: "Org Two"},
		},
	}, "")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "multiple organizations available")
}

func TestRollbackPostAuthFailureClearsStoredSession(t *testing.T) {
	t.Parallel()

	store := newMemSecretStore()
	require.NoError(t, store.Set(ports.KeyDashboardUserToken, "user-token"))
	require.NoError(t, store.Set(ports.KeyDashboardOrgToken, "org-token"))
	require.NoError(t, store.Set(ports.KeyDashboardUserPublicID, "user-1"))
	require.NoError(t, store.Set(ports.KeyDashboardOrgPublicID, "org-1"))
	require.NoError(t, store.Set(ports.KeyDashboardAppID, "app-1"))
	require.NoError(t, store.Set(ports.KeyDashboardAppRegion, "us"))

	var logoutCalled bool
	authSvc := dashboardapp.NewAuthService(&dashboardadapter.MockAccountClient{
		LogoutFn: func(_ context.Context, userToken, orgToken string) error {
			logoutCalled = true
			assert.Equal(t, "user-token", userToken)
			assert.Equal(t, "org-token", orgToken)
			return nil
		},
	}, store)

	rollbackPostAuthFailure(authSvc)

	assert.True(t, logoutCalled)
	for _, key := range []string{
		ports.KeyDashboardUserToken,
		ports.KeyDashboardOrgToken,
		ports.KeyDashboardUserPublicID,
		ports.KeyDashboardOrgPublicID,
		ports.KeyDashboardAppID,
		ports.KeyDashboardAppRegion,
	} {
		_, ok := store.data[key]
		assert.False(t, ok, "expected %s to be removed", key)
	}
}

func TestGetActiveAppRequiresPairedFlags(t *testing.T) {
	t.Parallel()

	_, _, err := getActiveApp("app-1", "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "both --app and --region")

	_, _, err = getActiveApp("", "us")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "both --app and --region")

	appID, region, err := getActiveApp("app-1", "us")
	require.NoError(t, err)
	assert.Equal(t, "app-1", appID)
	assert.Equal(t, "us", region)
}

func TestWriteSecretTempFileCreatesUniqueFiles(t *testing.T) {
	t.Parallel()

	path1, err := writeSecretTempFile("secret-1", "nylas-api-key.txt")
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.Remove(path1) })

	path2, err := writeSecretTempFile("secret-2", "nylas-api-key.txt")
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.Remove(path2) })

	assert.NotEqual(t, path1, path2)

	data1, err := os.ReadFile(path1)
	require.NoError(t, err)
	assert.Equal(t, "secret-1\n", string(data1))

	data2, err := os.ReadFile(path2)
	require.NoError(t, err)
	assert.Equal(t, "secret-2\n", string(data2))

	if runtime.GOOS != "windows" {
		info, err := os.Stat(path1)
		require.NoError(t, err)
		assert.Equal(t, os.FileMode(0o600), info.Mode().Perm())
	}
}

func TestResolveSSOMFAOrg(t *testing.T) {
	t.Parallel()

	t.Run("uses explicit org when provided", func(t *testing.T) {
		t.Parallel()

		orgID, err := resolveSSOMFAOrg("org-2", []domain.DashboardOrganization{
			{PublicID: "org-1"},
			{PublicID: "org-2"},
		})
		require.NoError(t, err)
		assert.Equal(t, "org-2", orgID)
	})

	t.Run("uses the only organization", func(t *testing.T) {
		t.Parallel()

		orgID, err := resolveSSOMFAOrg("", []domain.DashboardOrganization{{PublicID: "org-1"}})
		require.NoError(t, err)
		assert.Equal(t, "org-1", orgID)
	})

	t.Run("rejects multi-org MFA without explicit org in non-interactive mode", func(t *testing.T) {
		t.Parallel()

		orgID, err := resolveSSOMFAOrg("", []domain.DashboardOrganization{
			{PublicID: "org-1"},
			{PublicID: "org-2"},
		})
		require.Error(t, err)
		assert.Empty(t, orgID)
		assert.Contains(t, err.Error(), "multiple organizations available for MFA")
	})
}

func TestHandleAPIKeyDeliveryRejectsUnsafeNonInteractivePrompt(t *testing.T) {
	t.Parallel()

	err := handleAPIKeyDelivery("secret", "app-1", "us", "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "API key delivery requires an explicit choice")
}

func TestHandleSecretDeliveryRejectsUnsafeNonInteractivePrompt(t *testing.T) {
	t.Parallel()

	err := handleSecretDelivery("secret", "Client Secret", "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "client secret delivery requires an explicit choice")
}

func TestValidateDeliveryChoices(t *testing.T) {
	t.Parallel()

	require.NoError(t, validateAPIKeyDelivery(""))
	require.NoError(t, validateAPIKeyDelivery("activate"))
	require.NoError(t, validateSecretDelivery("clipboard"))

	err := validateAPIKeyDelivery("print")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid API key delivery method")

	err = validateSecretDelivery("terminal")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid secret delivery method")
}
