package dashboard

import (
	"bytes"
	"context"
	"os"
	"runtime"
	"testing"

	"github.com/spf13/cobra"
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

type fakeDomainService struct {
	listFn              func(ctx context.Context, limit int, pageToken string) (domain.DashboardInboxDomainPage, error)
	getFn               func(ctx context.Context, domainIDOrAddress, region string) (*domain.DashboardInboxDomain, error)
	checkAvailabilityFn func(ctx context.Context, domainAddress string) (*domain.DashboardInboxDomainAvailability, error)
	createFn            func(ctx context.Context, input domain.DashboardCreateInboxDomainInput) (*domain.DashboardInboxDomain, error)
	updateFn            func(ctx context.Context, domainID, region string, input domain.DashboardUpdateInboxDomainInput) (*domain.DashboardInboxDomain, error)
	deleteFn            func(ctx context.Context, domainID, region string) (bool, error)
	getDomainInfoFn     func(ctx context.Context, domainID, region, verificationType string) (*domain.DashboardDomainVerificationResult, error)
	verifyFn            func(ctx context.Context, domainID, region string, input domain.DashboardVerifyInboxDomainInput) (*domain.DashboardDomainVerificationResult, error)
}

func (f *fakeDomainService) ListDomains(ctx context.Context, limit int, pageToken string) (domain.DashboardInboxDomainPage, error) {
	if f.listFn != nil {
		return f.listFn(ctx, limit, pageToken)
	}
	return domain.DashboardInboxDomainPage{}, nil
}

func (f *fakeDomainService) GetDomain(ctx context.Context, domainIDOrAddress, region string) (*domain.DashboardInboxDomain, error) {
	if f.getFn != nil {
		return f.getFn(ctx, domainIDOrAddress, region)
	}
	return &domain.DashboardInboxDomain{}, nil
}

func (f *fakeDomainService) CheckAvailability(ctx context.Context, domainAddress string) (*domain.DashboardInboxDomainAvailability, error) {
	if f.checkAvailabilityFn != nil {
		return f.checkAvailabilityFn(ctx, domainAddress)
	}
	return &domain.DashboardInboxDomainAvailability{}, nil
}

func (f *fakeDomainService) CreateDomain(ctx context.Context, input domain.DashboardCreateInboxDomainInput) (*domain.DashboardInboxDomain, error) {
	if f.createFn != nil {
		return f.createFn(ctx, input)
	}
	return &domain.DashboardInboxDomain{}, nil
}

func (f *fakeDomainService) UpdateDomain(ctx context.Context, domainID, region string, input domain.DashboardUpdateInboxDomainInput) (*domain.DashboardInboxDomain, error) {
	if f.updateFn != nil {
		return f.updateFn(ctx, domainID, region, input)
	}
	return &domain.DashboardInboxDomain{}, nil
}

func (f *fakeDomainService) DeleteDomain(ctx context.Context, domainID, region string) (bool, error) {
	if f.deleteFn != nil {
		return f.deleteFn(ctx, domainID, region)
	}
	return false, nil
}

func (f *fakeDomainService) GetDomainInfo(ctx context.Context, domainID, region, verificationType string) (*domain.DashboardDomainVerificationResult, error) {
	if f.getDomainInfoFn != nil {
		return f.getDomainInfoFn(ctx, domainID, region, verificationType)
	}
	return &domain.DashboardDomainVerificationResult{}, nil
}

func (f *fakeDomainService) VerifyDomain(ctx context.Context, domainID, region string, input domain.DashboardVerifyInboxDomainInput) (*domain.DashboardDomainVerificationResult, error) {
	if f.verifyFn != nil {
		return f.verifyFn(ctx, domainID, region, input)
	}
	return &domain.DashboardDomainVerificationResult{}, nil
}

func stubDomainService(t *testing.T, svc domainService) {
	t.Helper()
	orig := createDomainServiceFn
	createDomainServiceFn = func() (domainService, error) {
		return svc, nil
	}
	t.Cleanup(func() {
		createDomainServiceFn = orig
	})
}

func executeTestCommand(cmd *cobra.Command, args ...string) (string, error) {
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs(args)
	err := cmd.Execute()
	return out.String(), err
}

func addTestOutputFlags(cmd *cobra.Command) {
	cmd.Flags().String("format", "", "Output format")
	cmd.Flags().Bool("json", false, "Output in JSON format")
	cmd.Flags().Bool("quiet", false, "Quiet mode")
	cmd.Flags().Bool("no-color", false, "Disable colored output")
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

func TestDashboardDomainValidation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		domain  string
		wantErr string
	}{
		{name: "valid root domain", domain: "example.com"},
		{name: "valid subdomain", domain: "mail.example.com"},
		{name: "missing dot", domain: "localhost", wantErr: "at least one dot"},
		{name: "bad character", domain: "bad slug.com", wantErr: "letters, numbers, and hyphens"},
		{name: "leading hyphen", domain: "-bad.example.com", wantErr: "cannot start or end"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := validateDomainAddress(tt.domain)
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestResolveDomainAddressRejectsConflictingInputs(t *testing.T) {
	t.Parallel()

	got, err := resolveDomainAddress([]string{"example.com"}, "")
	require.NoError(t, err)
	assert.Equal(t, "example.com", got)

	got, err = resolveDomainAddress([]string{"Example.COM."}, "")
	require.NoError(t, err)
	assert.Equal(t, "example.com", got)

	got, err = resolveDomainAddress([]string{"Example.COM."}, "example.com")
	require.NoError(t, err)
	assert.Equal(t, "example.com", got)

	_, err = resolveDomainAddress([]string{"example.com"}, "other.com")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "domain specified twice")
}

func TestDashboardDomainRows(t *testing.T) {
	t.Parallel()

	row := toDomainRow(domain.DashboardInboxDomain{
		ID:                "dom_1",
		Name:              "Example",
		DomainAddress:     "example.com",
		Region:            "us",
		Branded:           true,
		VerifiedOwnership: true,
		VerifiedMX:        true,
		VerifiedSPF:       false,
		VerifiedDKIM:      true,
	})

	assert.Equal(t, "dom_1", row.ID)
	assert.Equal(t, "example.com", row.Domain)
	assert.Equal(t, "yes", row.Ownership)
	assert.Equal(t, "yes", row.MX)
	assert.Equal(t, "no", row.SPF)
	assert.Equal(t, "yes", row.DKIM)
}

func TestNormalizeVerificationTypes(t *testing.T) {
	t.Parallel()

	assert.Equal(t, domainInfoTypes, normalizeVerificationTypes(nil))
	assert.Equal(t, []string{"mx", "spf"}, normalizeVerificationTypes([]string{"MX", "spf", "mx", ""}))

	require.NoError(t, validateVerificationTypes([]string{"mx", "ownership"}))
	err := validateVerificationTypes([]string{"txt"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid verification type")
}

func TestWriteDomainDeleteResult(t *testing.T) {
	t.Parallel()

	t.Run("returns an error when the API reports no deletion", func(t *testing.T) {
		t.Parallel()

		cmd := newDomainsDeleteCmd()
		err := writeDomainDeleteResult(cmd, false)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "domain was not deleted")
	})

	t.Run("writes structured success when requested", func(t *testing.T) {
		t.Parallel()

		cmd := newDomainsDeleteCmd()
		var out bytes.Buffer
		cmd.SetOut(&out)
		cmd.Flags().String("format", "", "Output format")
		cmd.Flags().Bool("json", true, "Output in JSON format")
		cmd.Flags().Bool("quiet", false, "Quiet mode")
		cmd.Flags().Bool("no-color", false, "Disable colored output")
		require.NoError(t, cmd.Flags().Set("json", "true"))

		err := writeDomainDeleteResult(cmd, true)

		require.NoError(t, err)
		assert.JSONEq(t, `{"success":true}`, out.String())
	})
}

func TestDomainsListRunEClampsLimitAndPrintsNextCursor(t *testing.T) {
	svc := &fakeDomainService{
		listFn: func(_ context.Context, limit int, pageToken string) (domain.DashboardInboxDomainPage, error) {
			assert.Equal(t, 200, limit)
			assert.Equal(t, "cursor-1", pageToken)
			return domain.DashboardInboxDomainPage{
				Domains: []domain.DashboardInboxDomain{
					{
						ID:            "dom_1",
						Name:          "Example",
						DomainAddress: "example.com",
						Region:        "us",
					},
				},
				NextCursor: "cursor-2",
			}, nil
		},
	}
	stubDomainService(t, svc)

	out, err := executeTestCommand(newDomainsListCmd(), "--limit", "999", "--page-token", "cursor-1")

	require.NoError(t, err)
	assert.Contains(t, out, "dom_1")
	assert.Contains(t, out, "Next: nylas dashboard domains list --page-token cursor-2")
}

func TestDomainsListRunEStructuredOutputIncludesNextCursor(t *testing.T) {
	svc := &fakeDomainService{
		listFn: func(_ context.Context, limit int, pageToken string) (domain.DashboardInboxDomainPage, error) {
			assert.Equal(t, 100, limit)
			assert.Empty(t, pageToken)
			return domain.DashboardInboxDomainPage{
				Domains: []domain.DashboardInboxDomain{
					{
						ID:            "dom_1",
						Name:          "Example",
						DomainAddress: "example.com",
						Region:        "eu",
					},
				},
				NextCursor: "cursor-2",
			}, nil
		},
	}
	stubDomainService(t, svc)

	cmd := newDomainsListCmd()
	addTestOutputFlags(cmd)
	out, err := executeTestCommand(cmd, "--json")

	require.NoError(t, err)
	assert.JSONEq(t, `{"domains":[{"id":"dom_1","domain":"example.com","name":"Example","region":"eu","branded":false,"ownership":"no","mx":"no","spf":"no","dkim":"no","dmarc":"no","arc":"no","feedback":"no"}],"next_cursor":"cursor-2"}`, out)
}

func TestDomainsListRunEEmptyStructuredOutput(t *testing.T) {
	svc := &fakeDomainService{
		listFn: func(_ context.Context, limit int, pageToken string) (domain.DashboardInboxDomainPage, error) {
			assert.Equal(t, 100, limit)
			assert.Empty(t, pageToken)
			return domain.DashboardInboxDomainPage{
				NextCursor: "cursor-2",
			}, nil
		},
	}
	stubDomainService(t, svc)

	cmd := newDomainsListCmd()
	addTestOutputFlags(cmd)
	out, err := executeTestCommand(cmd, "--json")

	require.NoError(t, err)
	assert.JSONEq(t, `{"domains":[],"next_cursor":"cursor-2"}`, out)
}

func TestDomainsListRunEEmptyQuietOutput(t *testing.T) {
	svc := &fakeDomainService{
		listFn: func(_ context.Context, limit int, pageToken string) (domain.DashboardInboxDomainPage, error) {
			assert.Equal(t, 100, limit)
			assert.Empty(t, pageToken)
			return domain.DashboardInboxDomainPage{
				NextCursor: "cursor-2",
			}, nil
		},
	}
	stubDomainService(t, svc)

	cmd := newDomainsListCmd()
	addTestOutputFlags(cmd)
	out, err := executeTestCommand(cmd, "--quiet")

	require.NoError(t, err)
	assert.Empty(t, out)
}

func TestDomainsCreateRunERejectsIncompleteDashboardResponse(t *testing.T) {
	svc := &fakeDomainService{
		createFn: func(_ context.Context, input domain.DashboardCreateInboxDomainInput) (*domain.DashboardInboxDomain, error) {
			assert.Equal(t, "example.com", input.DomainAddress)
			assert.Equal(t, "example.com", input.Name)
			assert.Equal(t, "us", input.Region)
			return &domain.DashboardInboxDomain{}, nil
		},
	}
	stubDomainService(t, svc)

	_, err := executeTestCommand(newDomainsCreateCmd(), "example.com", "--region", "us")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "domain was not created")
}

func TestDomainsCreateRunEStillRequiresRegion(t *testing.T) {
	_, err := executeTestCommand(newDomainsCreateCmd(), "example.com")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid region")
}

func TestDomainsShowRunEInfersRegion(t *testing.T) {
	svc := &fakeDomainService{
		listFn: func(_ context.Context, limit int, pageToken string) (domain.DashboardInboxDomainPage, error) {
			assert.Equal(t, 200, limit)
			assert.Empty(t, pageToken)
			return domain.DashboardInboxDomainPage{
				Domains: []domain.DashboardInboxDomain{
					{ID: "dom_eu", DomainAddress: "asim.nylas.email", Region: "eu"},
				},
			}, nil
		},
		getFn: func(_ context.Context, domainIDOrAddress, region string) (*domain.DashboardInboxDomain, error) {
			assert.Equal(t, "dom_eu", domainIDOrAddress)
			assert.Equal(t, "eu", region)
			return &domain.DashboardInboxDomain{
				ID:            "dom_eu",
				Name:          "asim.nylas.email",
				DomainAddress: "asim.nylas.email",
				Region:        "eu",
			}, nil
		},
	}
	stubDomainService(t, svc)

	cmd := newDomainsShowCmd()
	addTestOutputFlags(cmd)
	out, err := executeTestCommand(cmd, "asim.nylas.email", "--json")

	require.NoError(t, err)
	assert.JSONEq(t, `{"id":"dom_eu","domain":"asim.nylas.email","name":"asim.nylas.email","region":"eu","branded":false,"ownership":"no","mx":"no","spf":"no","dkim":"no","dmarc":"no","arc":"no","feedback":"no"}`, out)
}

func TestDomainsUpdateRunERejectsIncompleteDashboardResponse(t *testing.T) {
	svc := &fakeDomainService{
		updateFn: func(_ context.Context, domainID, region string, input domain.DashboardUpdateInboxDomainInput) (*domain.DashboardInboxDomain, error) {
			assert.Equal(t, "dom_1", domainID)
			assert.Equal(t, "eu", region)
			assert.Equal(t, "Renamed", input.Name)
			return &domain.DashboardInboxDomain{}, nil
		},
	}
	stubDomainService(t, svc)

	_, err := executeTestCommand(newDomainsUpdateCmd(), "dom_1", "--region", "eu", "--name", "Renamed")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "domain was not updated")
}

func TestDomainsUpdateRunEInfersRegion(t *testing.T) {
	svc := &fakeDomainService{
		listFn: func(_ context.Context, limit int, pageToken string) (domain.DashboardInboxDomainPage, error) {
			assert.Equal(t, 200, limit)
			assert.Empty(t, pageToken)
			return domain.DashboardInboxDomainPage{
				Domains: []domain.DashboardInboxDomain{
					{ID: "dom_eu", DomainAddress: "asim.nylas.email", Region: "eu"},
				},
			}, nil
		},
		updateFn: func(_ context.Context, domainID, region string, input domain.DashboardUpdateInboxDomainInput) (*domain.DashboardInboxDomain, error) {
			assert.Equal(t, "dom_eu", domainID)
			assert.Equal(t, "eu", region)
			assert.Equal(t, "Renamed", input.Name)
			return &domain.DashboardInboxDomain{
				ID:            "dom_eu",
				Name:          "Renamed",
				DomainAddress: "asim.nylas.email",
				Region:        "eu",
			}, nil
		},
	}
	stubDomainService(t, svc)

	cmd := newDomainsUpdateCmd()
	addTestOutputFlags(cmd)
	out, err := executeTestCommand(cmd, "asim.nylas.email", "--name", "Renamed", "--json")

	require.NoError(t, err)
	assert.JSONEq(t, `{"id":"dom_eu","domain":"asim.nylas.email","name":"Renamed","region":"eu","branded":false,"ownership":"no","mx":"no","spf":"no","dkim":"no","dmarc":"no","arc":"no","feedback":"no"}`, out)
}

func TestDomainsDNSRunERequestsEachType(t *testing.T) {
	var gotTypes []string
	svc := &fakeDomainService{
		getDomainInfoFn: func(_ context.Context, domainID, region, verificationType string) (*domain.DashboardDomainVerificationResult, error) {
			assert.Equal(t, "dom_1", domainID)
			assert.Equal(t, "us", region)
			gotTypes = append(gotTypes, verificationType)
			return &domain.DashboardDomainVerificationResult{
				Status:  "pending",
				Message: "configure",
			}, nil
		},
	}
	stubDomainService(t, svc)

	cmd := newDomainsDNSCmd()
	addTestOutputFlags(cmd)
	out, err := executeTestCommand(cmd, "dom_1", "--region", "us", "--type", "mx", "--type", "spf", "--json")

	require.NoError(t, err)
	assert.Equal(t, []string{"mx", "spf"}, gotTypes)
	assert.JSONEq(t, `[{"type":"mx","host":"","record":"","value":"configure","status":"pending"},{"type":"spf","host":"","record":"","value":"configure","status":"pending"}]`, out)
}

func TestDomainsDNSRunEInfersRegion(t *testing.T) {
	svc := &fakeDomainService{
		listFn: func(_ context.Context, limit int, pageToken string) (domain.DashboardInboxDomainPage, error) {
			assert.Equal(t, 200, limit)
			assert.Empty(t, pageToken)
			return domain.DashboardInboxDomainPage{
				Domains: []domain.DashboardInboxDomain{
					{ID: "dom_eu", DomainAddress: "asim.nylas.email", Region: "eu"},
				},
			}, nil
		},
		getDomainInfoFn: func(_ context.Context, domainID, region, verificationType string) (*domain.DashboardDomainVerificationResult, error) {
			assert.Equal(t, "dom_eu", domainID)
			assert.Equal(t, "eu", region)
			assert.Equal(t, "mx", verificationType)
			return &domain.DashboardDomainVerificationResult{
				Status:  "done",
				Message: "verified",
			}, nil
		},
	}
	stubDomainService(t, svc)

	cmd := newDomainsDNSCmd()
	addTestOutputFlags(cmd)
	out, err := executeTestCommand(cmd, "asim.nylas.email", "--type", "mx", "--json")

	require.NoError(t, err)
	assert.JSONEq(t, `[{"type":"mx","host":"","record":"","value":"verified","status":"done"}]`, out)
}

func TestDomainsVerifyRunEAllRequestsEverySupportedType(t *testing.T) {
	var gotTypes []string
	svc := &fakeDomainService{
		verifyFn: func(_ context.Context, domainID, region string, input domain.DashboardVerifyInboxDomainInput) (*domain.DashboardDomainVerificationResult, error) {
			assert.Equal(t, "dom_1", domainID)
			assert.Equal(t, "eu", region)
			gotTypes = append(gotTypes, input.Type)
			return &domain.DashboardDomainVerificationResult{
				Status:  "done",
				Message: "verified",
			}, nil
		},
	}
	stubDomainService(t, svc)

	cmd := newDomainsVerifyCmd()
	addTestOutputFlags(cmd)
	out, err := executeTestCommand(cmd, "dom_1", "--region", "eu", "--all", "--json")

	require.NoError(t, err)
	assert.Equal(t, domainInfoTypes, gotTypes)
	assert.Contains(t, out, `"type": "ownership"`)
	assert.Contains(t, out, `"type": "arc"`)
}

func TestDomainsVerifyRunEInfersRegion(t *testing.T) {
	svc := &fakeDomainService{
		listFn: func(_ context.Context, limit int, pageToken string) (domain.DashboardInboxDomainPage, error) {
			assert.Equal(t, 200, limit)
			assert.Empty(t, pageToken)
			return domain.DashboardInboxDomainPage{
				Domains: []domain.DashboardInboxDomain{
					{ID: "dom_eu", DomainAddress: "asim.nylas.email", Region: "eu"},
				},
			}, nil
		},
		verifyFn: func(_ context.Context, domainID, region string, input domain.DashboardVerifyInboxDomainInput) (*domain.DashboardDomainVerificationResult, error) {
			assert.Equal(t, "dom_eu", domainID)
			assert.Equal(t, "eu", region)
			assert.Equal(t, "ownership", input.Type)
			return &domain.DashboardDomainVerificationResult{
				Status:  "done",
				Message: "verified",
			}, nil
		},
	}
	stubDomainService(t, svc)

	cmd := newDomainsVerifyCmd()
	addTestOutputFlags(cmd)
	out, err := executeTestCommand(cmd, "asim.nylas.email", "--type", "ownership", "--json")

	require.NoError(t, err)
	assert.JSONEq(t, `[{"type":"ownership","status":"done","message":"verified"}]`, out)
}

func TestDomainsDeleteRunEInfersRegion(t *testing.T) {
	svc := &fakeDomainService{
		listFn: func(_ context.Context, limit int, pageToken string) (domain.DashboardInboxDomainPage, error) {
			assert.Equal(t, 200, limit)
			assert.Empty(t, pageToken)
			return domain.DashboardInboxDomainPage{
				Domains: []domain.DashboardInboxDomain{
					{ID: "dom_eu", DomainAddress: "asim.nylas.email", Region: "eu"},
				},
			}, nil
		},
		deleteFn: func(_ context.Context, domainID, region string) (bool, error) {
			assert.Equal(t, "dom_eu", domainID)
			assert.Equal(t, "eu", region)
			return true, nil
		},
	}
	stubDomainService(t, svc)

	cmd := newDomainsDeleteCmd()
	addTestOutputFlags(cmd)
	out, err := executeTestCommand(cmd, "asim.nylas.email", "--yes", "--json")

	require.NoError(t, err)
	assert.JSONEq(t, `{"success":true}`, out)
}

func TestResolveExistingDomainRefRequiresRegionWhenDomainIsUnknown(t *testing.T) {
	svc := &fakeDomainService{
		listFn: func(_ context.Context, limit int, pageToken string) (domain.DashboardInboxDomainPage, error) {
			assert.Equal(t, 200, limit)
			assert.Empty(t, pageToken)
			return domain.DashboardInboxDomainPage{}, nil
		},
	}

	_, err := resolveExistingDomainRef(context.Background(), svc, "missing.nylas.email", "")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "domain not found")
	assert.Contains(t, err.Error(), "--region us or --region eu")
}

func TestResolveExistingDomainRefRejectsMultipleRegionMatches(t *testing.T) {
	svc := &fakeDomainService{
		listFn: func(_ context.Context, limit int, pageToken string) (domain.DashboardInboxDomainPage, error) {
			assert.Equal(t, 200, limit)
			assert.Empty(t, pageToken)
			return domain.DashboardInboxDomainPage{
				Domains: []domain.DashboardInboxDomain{
					{ID: "dom_us", DomainAddress: "asim.nylas.email", Region: "us"},
					{ID: "dom_eu", DomainAddress: "asim.nylas.email", Region: "eu"},
				},
			}, nil
		},
	}

	_, err := resolveExistingDomainRef(context.Background(), svc, "asim.nylas.email", "")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "domain matches multiple regions")
	assert.Contains(t, err.Error(), "--region us or --region eu")
}

func TestResolveExistingDomainRefMatchesByID(t *testing.T) {
	svc := &fakeDomainService{
		listFn: func(_ context.Context, limit int, pageToken string) (domain.DashboardInboxDomainPage, error) {
			assert.Equal(t, 200, limit)
			assert.Empty(t, pageToken)
			return domain.DashboardInboxDomainPage{
				Domains: []domain.DashboardInboxDomain{
					{ID: "dom_eu", DomainAddress: "asim.nylas.email", Region: "eu"},
				},
			}, nil
		},
	}

	ref, err := resolveExistingDomainRef(context.Background(), svc, "dom_eu", "")

	require.NoError(t, err)
	assert.Equal(t, "dom_eu", ref.IDOrAddress)
	assert.Equal(t, "eu", ref.Region)
	assert.Equal(t, "asim.nylas.email", ref.Display)
}

func TestResolveExistingDomainRefNormalizesExplicitRegionDomainAddress(t *testing.T) {
	svc := &fakeDomainService{}

	ref, err := resolveExistingDomainRef(context.Background(), svc, "ASIM.NYLAS.EMAIL.", "eu")

	require.NoError(t, err)
	assert.Equal(t, "asim.nylas.email", ref.IDOrAddress)
	assert.Equal(t, "eu", ref.Region)
	assert.Equal(t, "asim.nylas.email", ref.Display)
}

func TestListDomainsForResolutionAggregatesPagesAndStopsOnSeenCursor(t *testing.T) {
	calls := 0
	svc := &fakeDomainService{
		listFn: func(_ context.Context, limit int, pageToken string) (domain.DashboardInboxDomainPage, error) {
			assert.Equal(t, 200, limit)
			calls++
			switch pageToken {
			case "":
				return domain.DashboardInboxDomainPage{
					Domains: []domain.DashboardInboxDomain{
						{ID: "dom_1", DomainAddress: "one.nylas.email", Region: "us"},
					},
					NextCursor: "cursor-a",
				}, nil
			case "cursor-a":
				return domain.DashboardInboxDomainPage{
					Domains: []domain.DashboardInboxDomain{
						{ID: "dom_2", DomainAddress: "two.nylas.email", Region: "eu"},
					},
					NextCursor: "cursor-b",
				}, nil
			case "cursor-b":
				return domain.DashboardInboxDomainPage{
					Domains: []domain.DashboardInboxDomain{
						{ID: "dom_3", DomainAddress: "three.nylas.email", Region: "us"},
					},
					NextCursor: "cursor-a",
				}, nil
			default:
				t.Fatalf("unexpected page token %q", pageToken)
				return domain.DashboardInboxDomainPage{}, nil
			}
		},
	}

	domains, err := listDomainsForResolution(context.Background(), svc)

	require.NoError(t, err)
	assert.Equal(t, 3, calls)
	require.Len(t, domains, 3)
	assert.Equal(t, "dom_1", domains[0].ID)
	assert.Equal(t, "dom_2", domains[1].ID)
	assert.Equal(t, "dom_3", domains[2].ID)
}
