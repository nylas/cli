//go:build integration

package integration

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/nylas/cli/internal/adapters/config"
	"github.com/nylas/cli/internal/adapters/keyring"
	"github.com/nylas/cli/internal/ports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const dashboardTestPassphrase = "integration-test-file-store-passphrase"

func TestCLI_DashboardLoginEmailPasswordPersistsSession(t *testing.T) {
	if testBinary == "" {
		t.Skip("CLI binary not found")
	}

	configHome, tempHome, secretStore := newDashboardTestSecretStore(t)
	require.NoError(t, secretStore.Set(ports.KeyDashboardAppID, "stale-app"))
	require.NoError(t, secretStore.Set(ports.KeyDashboardAppRegion, "eu"))

	accountServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/auth/cli/login":
			var body map[string]any
			require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
			assert.Equal(t, "user@example.com", body["email"])
			assert.Equal(t, "secret", body["password"])
			assert.Equal(t, "org-2", body["orgPublicId"])
			assert.NotEmpty(t, r.Header.Get("DPoP"))
			writeDashboardResponse(t, w, map[string]any{
				"userToken": "user-token",
				"orgToken":  "org-token-initial",
				"user": map[string]any{
					"publicId": "user-1",
				},
				"organizations": []map[string]any{
					{"publicId": "org-1", "name": "Org One"},
					{"publicId": "org-2", "name": "Org Two"},
				},
			})
		case "/sessions/switch-org":
			var body map[string]any
			require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
			assert.Equal(t, "org-2", body["orgPublicId"])
			assert.Equal(t, "Bearer user-token", r.Header.Get("Authorization"))
			writeDashboardResponse(t, w, map[string]any{
				"orgToken": "org-token-switched",
				"org": map[string]any{
					"publicId": "org-2",
					"name":     "Org Two",
				},
			})
		case "/sessions/current":
			assert.Equal(t, "Bearer user-token", r.Header.Get("Authorization"))
			assert.Equal(t, "org-token-switched", r.Header.Get("X-Nylas-Org"))
			writeDashboardResponse(t, w, map[string]any{
				"user": map[string]any{
					"publicId": "user-1",
				},
				"currentOrg": "org-2",
				"relations": []map[string]any{
					{"orgPublicId": "org-1", "orgName": "Org One"},
					{"orgPublicId": "org-2", "orgName": "Org Two"},
				},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer accountServer.Close()

	stdout, stderr, err := runCLIWithOverrides(30*time.Second, dashboardEnvOverrides(configHome, tempHome, map[string]string{
		"NYLAS_DASHBOARD_ACCOUNT_URL": accountServer.URL,
		"NYLAS_API_KEY":               "",
		"NYLAS_GRANT_ID":              "",
	}), "dashboard", "login", "--email", "--user", "user@example.com", "--password", "secret", "--org", "org-2")
	if err != nil {
		t.Fatalf("dashboard login failed: %v\nstderr: %s", err, stderr)
	}

	assert.Contains(t, stdout, "Authenticated as user-1")
	assert.Contains(t, stdout, "Organization: Org Two (org-2)")

	userToken, err := secretStore.Get(ports.KeyDashboardUserToken)
	require.NoError(t, err)
	assert.Equal(t, "user-token", userToken)
	orgToken, err := secretStore.Get(ports.KeyDashboardOrgToken)
	require.NoError(t, err)
	assert.Equal(t, "org-token-switched", orgToken)
	orgID, err := secretStore.Get(ports.KeyDashboardOrgPublicID)
	require.NoError(t, err)
	assert.Equal(t, "org-2", orgID)
	appID, err := secretStore.Get(ports.KeyDashboardAppID)
	require.Error(t, err)
	assert.Empty(t, appID)
}

func TestCLI_DashboardLoginRequiresOrgForMultiOrgNonInteractive(t *testing.T) {
	if testBinary == "" {
		t.Skip("CLI binary not found")
	}

	configHome, tempHome, secretStore := newDashboardTestSecretStore(t)

	accountServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/auth/cli/login":
			writeDashboardResponse(t, w, map[string]any{
				"userToken": "user-token",
				"orgToken":  "org-token",
				"user": map[string]any{
					"publicId": "user-1",
				},
				"organizations": []map[string]any{
					{"publicId": "org-1", "name": "Org One"},
					{"publicId": "org-2", "name": "Org Two"},
				},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer accountServer.Close()

	stdout, stderr, err := runCLIWithOverrides(30*time.Second, dashboardEnvOverrides(configHome, tempHome, map[string]string{
		"NYLAS_DASHBOARD_ACCOUNT_URL": accountServer.URL,
		"NYLAS_API_KEY":               "",
		"NYLAS_GRANT_ID":              "",
	}), "dashboard", "login", "--email", "--user", "user@example.com", "--password", "secret")
	if err == nil {
		t.Fatalf("expected dashboard login without --org to fail for multi-org account\nstdout: %s\nstderr: %s", stdout, stderr)
	}

	assert.Contains(t, strings.ToLower(stderr), "multiple organizations available")
	assert.Contains(t, stderr, "--org")

	userToken, tokenErr := secretStore.Get(ports.KeyDashboardUserToken)
	require.Error(t, tokenErr)
	assert.Empty(t, userToken)
}

func TestCLI_DashboardLoginRequiresExplicitAuthMethodInNonInteractiveMode(t *testing.T) {
	if testBinary == "" {
		t.Skip("CLI binary not found")
	}

	configHome, tempHome, _ := newDashboardTestSecretStore(t)

	stdout, stderr, err := runCLIWithOverrides(30*time.Second, dashboardEnvOverrides(configHome, tempHome, map[string]string{
		"NYLAS_API_KEY":  "",
		"NYLAS_GRANT_ID": "",
	}), "dashboard", "login")
	if err == nil {
		t.Fatalf("expected dashboard login without auth method to fail in non-interactive mode\nstdout: %s\nstderr: %s", stdout, stderr)
	}

	assert.Contains(t, strings.ToLower(stderr), "auth method is required")
	assert.Contains(t, stderr, "--google")
}

func TestCLI_DashboardLoginRollsBackSessionWhenOrgSwitchFails(t *testing.T) {
	if testBinary == "" {
		t.Skip("CLI binary not found")
	}

	configHome, tempHome, secretStore := newDashboardTestSecretStore(t)
	require.NoError(t, secretStore.Set(ports.KeyDashboardAppID, "stale-app"))
	require.NoError(t, secretStore.Set(ports.KeyDashboardAppRegion, "eu"))

	accountServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/auth/cli/login":
			writeDashboardResponse(t, w, map[string]any{
				"userToken": "user-token",
				"orgToken":  "org-token-initial",
				"user": map[string]any{
					"publicId": "user-1",
				},
				"organizations": []map[string]any{
					{"publicId": "org-1", "name": "Org One"},
					{"publicId": "org-2", "name": "Org Two"},
				},
			})
		case "/sessions/switch-org":
			writeDashboardErrorResponse(t, w, http.StatusBadGateway, "UPSTREAM_UNAVAILABLE", "dashboard unavailable")
		case "/auth/cli/logout":
			writeDashboardResponse(t, w, map[string]any{})
		default:
			http.NotFound(w, r)
		}
	}))
	defer accountServer.Close()

	stdout, stderr, err := runCLIWithOverrides(30*time.Second, dashboardEnvOverrides(configHome, tempHome, map[string]string{
		"NYLAS_DASHBOARD_ACCOUNT_URL": accountServer.URL,
		"NYLAS_API_KEY":               "",
		"NYLAS_GRANT_ID":              "",
	}), "dashboard", "login", "--email", "--user", "user@example.com", "--password", "secret", "--org", "org-2")
	if err == nil {
		t.Fatalf("expected dashboard login to fail when org switch fails\nstdout: %s\nstderr: %s", stdout, stderr)
	}

	assert.Contains(t, strings.ToLower(stderr), "failed to switch organization")
	for _, key := range []string{
		ports.KeyDashboardUserToken,
		ports.KeyDashboardOrgToken,
		ports.KeyDashboardUserPublicID,
		ports.KeyDashboardOrgPublicID,
		ports.KeyDashboardAppID,
		ports.KeyDashboardAppRegion,
	} {
		value, getErr := secretStore.Get(key)
		require.Error(t, getErr, "expected %s to be removed", key)
		assert.Empty(t, value)
	}
}

func TestCLI_DashboardRefreshUpdatesStoredTokens(t *testing.T) {
	if testBinary == "" {
		t.Skip("CLI binary not found")
	}

	configHome, tempHome, secretStore := newDashboardTestSecretStore(t)
	require.NoError(t, secretStore.Set(ports.KeyDashboardUserToken, "user-token-old"))
	require.NoError(t, secretStore.Set(ports.KeyDashboardOrgToken, "org-token-old"))

	accountServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/auth/cli/refresh", r.URL.Path)
		assert.Equal(t, "Bearer user-token-old", r.Header.Get("Authorization"))
		assert.Equal(t, "org-token-old", r.Header.Get("X-Nylas-Org"))
		writeDashboardResponse(t, w, map[string]any{
			"userToken": "user-token-new",
			"orgToken":  "org-token-new",
		})
	}))
	defer accountServer.Close()

	stdout, stderr, err := runCLIWithOverrides(30*time.Second, dashboardEnvOverrides(configHome, tempHome, map[string]string{
		"NYLAS_DASHBOARD_ACCOUNT_URL": accountServer.URL,
		"NYLAS_API_KEY":               "",
		"NYLAS_GRANT_ID":              "",
	}), "dashboard", "refresh")
	if err != nil {
		t.Fatalf("dashboard refresh failed: %v\nstderr: %s", err, stderr)
	}

	assert.Contains(t, stdout, "Session refreshed")
	userToken, err := secretStore.Get(ports.KeyDashboardUserToken)
	require.NoError(t, err)
	assert.Equal(t, "user-token-new", userToken)
	orgToken, err := secretStore.Get(ports.KeyDashboardOrgToken)
	require.NoError(t, err)
	assert.Equal(t, "org-token-new", orgToken)
}

func TestCLI_DashboardRegisterRequiresExplicitAuthMethodInNonInteractiveMode(t *testing.T) {
	if testBinary == "" {
		t.Skip("CLI binary not found")
	}

	configHome, tempHome, _ := newDashboardTestSecretStore(t)

	stdout, stderr, err := runCLIWithOverrides(30*time.Second, dashboardEnvOverrides(configHome, tempHome, map[string]string{
		"NYLAS_API_KEY":  "",
		"NYLAS_GRANT_ID": "",
	}), "dashboard", "register")
	if err == nil {
		t.Fatalf("expected dashboard register without auth method to fail in non-interactive mode\nstdout: %s\nstderr: %s", stdout, stderr)
	}

	assert.Contains(t, strings.ToLower(stderr), "auth method is required")
	assert.Contains(t, stderr, "--google")
}

func TestCLI_DashboardRegisterRequiresExplicitPrivacyAcceptanceInNonInteractiveMode(t *testing.T) {
	if testBinary == "" {
		t.Skip("CLI binary not found")
	}

	configHome, tempHome, _ := newDashboardTestSecretStore(t)

	stdout, stderr, err := runCLIWithOverrides(30*time.Second, dashboardEnvOverrides(configHome, tempHome, map[string]string{
		"NYLAS_API_KEY":  "",
		"NYLAS_GRANT_ID": "",
	}), "dashboard", "register", "--google")
	if err == nil {
		t.Fatalf("expected dashboard register without privacy acceptance to fail in non-interactive mode\nstdout: %s\nstderr: %s", stdout, stderr)
	}

	assert.Contains(t, strings.ToLower(stderr), "privacy policy must be accepted")
	assert.Contains(t, stderr, "--accept-privacy-policy")
}

func TestCLI_DashboardLogoutClearsSession(t *testing.T) {
	if testBinary == "" {
		t.Skip("CLI binary not found")
	}

	configHome, tempHome, secretStore := newDashboardTestSecretStore(t)
	require.NoError(t, secretStore.Set(ports.KeyDashboardUserToken, "user-token"))
	require.NoError(t, secretStore.Set(ports.KeyDashboardOrgToken, "org-token"))
	require.NoError(t, secretStore.Set(ports.KeyDashboardAppID, "app-1"))
	require.NoError(t, secretStore.Set(ports.KeyDashboardAppRegion, "us"))

	accountServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/auth/cli/logout", r.URL.Path)
		assert.Equal(t, "Bearer user-token", r.Header.Get("Authorization"))
		writeDashboardResponse(t, w, map[string]any{})
	}))
	defer accountServer.Close()

	stdout, stderr, err := runCLIWithOverrides(30*time.Second, dashboardEnvOverrides(configHome, tempHome, map[string]string{
		"NYLAS_DASHBOARD_ACCOUNT_URL": accountServer.URL,
		"NYLAS_API_KEY":               "",
		"NYLAS_GRANT_ID":              "",
	}), "dashboard", "logout")
	if err != nil {
		t.Fatalf("dashboard logout failed: %v\nstderr: %s", err, stderr)
	}

	assert.Contains(t, stdout, "Logged out")
	userToken, err := secretStore.Get(ports.KeyDashboardUserToken)
	require.Error(t, err)
	assert.Empty(t, userToken)
	appID, err := secretStore.Get(ports.KeyDashboardAppID)
	require.Error(t, err)
	assert.Empty(t, appID)
}

func TestCLI_DashboardOrgsListUsesCurrentSession(t *testing.T) {
	if testBinary == "" {
		t.Skip("CLI binary not found")
	}

	configHome, tempHome, secretStore := newDashboardTestSecretStore(t)
	require.NoError(t, secretStore.Set(ports.KeyDashboardUserToken, "user-token"))
	require.NoError(t, secretStore.Set(ports.KeyDashboardOrgToken, "org-token"))

	accountServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/sessions/current", r.URL.Path)
		assert.Equal(t, "Bearer user-token", r.Header.Get("Authorization"))
		writeDashboardResponse(t, w, map[string]any{
			"user": map[string]any{
				"publicId": "user-1",
			},
			"currentOrg": "org-1",
			"relations": []map[string]any{
				{"orgPublicId": "org-1", "orgName": "Acme", "role": "admin"},
				{"orgPublicId": "org-2", "orgName": "Beta", "role": "member"},
			},
		})
	}))
	defer accountServer.Close()

	stdout, stderr, err := runCLIWithOverrides(30*time.Second, dashboardEnvOverrides(configHome, tempHome, map[string]string{
		"NYLAS_DASHBOARD_ACCOUNT_URL": accountServer.URL,
		"NYLAS_API_KEY":               "",
		"NYLAS_GRANT_ID":              "",
	}), "dashboard", "orgs", "list")
	if err != nil {
		t.Fatalf("dashboard orgs list failed: %v\nstderr: %s", err, stderr)
	}

	assert.Contains(t, stdout, "Acme")
	assert.Contains(t, stdout, "Beta")
}

func TestCLI_DashboardOrgsListRefreshesExpiredSession(t *testing.T) {
	if testBinary == "" {
		t.Skip("CLI binary not found")
	}

	configHome, tempHome, secretStore := newDashboardTestSecretStore(t)
	require.NoError(t, secretStore.Set(ports.KeyDashboardUserToken, "user-token-old"))
	require.NoError(t, secretStore.Set(ports.KeyDashboardOrgToken, "org-token-old"))

	currentCalls := 0
	accountServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/sessions/current":
			currentCalls++
			if currentCalls == 1 {
				assert.Equal(t, "Bearer user-token-old", r.Header.Get("Authorization"))
				assert.Equal(t, "org-token-old", r.Header.Get("X-Nylas-Org"))
				writeDashboardErrorResponse(t, w, http.StatusUnauthorized, "INVALID_SESSION", "")
				return
			}
			assert.Equal(t, "Bearer user-token-new", r.Header.Get("Authorization"))
			assert.Equal(t, "org-token-new", r.Header.Get("X-Nylas-Org"))
			writeDashboardResponse(t, w, map[string]any{
				"user": map[string]any{
					"publicId": "user-1",
				},
				"currentOrg": "org-1",
				"relations": []map[string]any{
					{"orgPublicId": "org-1", "orgName": "Acme"},
				},
			})
		case "/auth/cli/refresh":
			assert.Equal(t, "Bearer user-token-old", r.Header.Get("Authorization"))
			assert.Equal(t, "org-token-old", r.Header.Get("X-Nylas-Org"))
			writeDashboardResponse(t, w, map[string]any{
				"userToken": "user-token-new",
				"orgToken":  "org-token-new",
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer accountServer.Close()

	stdout, stderr, err := runCLIWithOverrides(30*time.Second, dashboardEnvOverrides(configHome, tempHome, map[string]string{
		"NYLAS_DASHBOARD_ACCOUNT_URL": accountServer.URL,
		"NYLAS_API_KEY":               "",
		"NYLAS_GRANT_ID":              "",
	}), "dashboard", "orgs", "list")
	if err != nil {
		t.Fatalf("dashboard orgs list with refresh failed: %v\nstderr: %s", err, stderr)
	}

	assert.Contains(t, stdout, "Acme")
	userToken, err := secretStore.Get(ports.KeyDashboardUserToken)
	require.NoError(t, err)
	assert.Equal(t, "user-token-new", userToken)
	orgToken, err := secretStore.Get(ports.KeyDashboardOrgToken)
	require.NoError(t, err)
	assert.Equal(t, "org-token-new", orgToken)
}

func TestCLI_DashboardSwitchOrgUpdatesStoredSession(t *testing.T) {
	if testBinary == "" {
		t.Skip("CLI binary not found")
	}

	configHome, tempHome, secretStore := newDashboardTestSecretStore(t)
	require.NoError(t, secretStore.Set(ports.KeyDashboardUserToken, "user-token"))
	require.NoError(t, secretStore.Set(ports.KeyDashboardOrgToken, "org-token-old"))
	require.NoError(t, secretStore.Set(ports.KeyDashboardOrgPublicID, "org-1"))

	accountServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/sessions/current":
			assert.Equal(t, "Bearer user-token", r.Header.Get("Authorization"))
			assert.Equal(t, "org-token-old", r.Header.Get("X-Nylas-Org"))
			writeDashboardResponse(t, w, map[string]any{
				"user": map[string]any{
					"publicId": "user-1",
				},
				"currentOrg": "org-1",
				"relations": []map[string]any{
					{"orgPublicId": "org-1", "orgName": "Acme"},
					{"orgPublicId": "org-2", "orgName": "Beta"},
				},
			})
		case "/sessions/switch-org":
			var body map[string]any
			require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
			assert.Equal(t, "org-2", body["orgPublicId"])
			assert.Equal(t, "Bearer user-token", r.Header.Get("Authorization"))
			assert.Equal(t, "org-token-old", r.Header.Get("X-Nylas-Org"))
			writeDashboardResponse(t, w, map[string]any{
				"orgToken": "org-token-new",
				"org": map[string]any{
					"publicId": "org-2",
					"name":     "Beta",
				},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer accountServer.Close()

	stdout, stderr, err := runCLIWithOverrides(30*time.Second, dashboardEnvOverrides(configHome, tempHome, map[string]string{
		"NYLAS_DASHBOARD_ACCOUNT_URL": accountServer.URL,
		"NYLAS_API_KEY":               "",
		"NYLAS_GRANT_ID":              "",
	}), "dashboard", "orgs", "switch", "--org", "org-2")
	if err != nil {
		t.Fatalf("dashboard orgs switch failed: %v\nstderr: %s", err, stderr)
	}

	assert.Contains(t, stdout, "Switched to organization: Beta (org-2)")

	orgID, err := secretStore.Get(ports.KeyDashboardOrgPublicID)
	require.NoError(t, err)
	assert.Equal(t, "org-2", orgID)
	orgToken, err := secretStore.Get(ports.KeyDashboardOrgToken)
	require.NoError(t, err)
	assert.Equal(t, "org-token-new", orgToken)
}

func TestCLI_DashboardSwitchOrgRequiresOrgInNonInteractiveMode(t *testing.T) {
	if testBinary == "" {
		t.Skip("CLI binary not found")
	}

	configHome, tempHome, _ := newDashboardTestSecretStore(t)

	stdout, stderr, err := runCLIWithOverrides(30*time.Second, dashboardEnvOverrides(configHome, tempHome, map[string]string{
		"NYLAS_API_KEY":  "",
		"NYLAS_GRANT_ID": "",
	}), "dashboard", "orgs", "switch")
	if err == nil {
		t.Fatalf("expected dashboard orgs switch without --org to fail in non-interactive mode\nstdout: %s\nstderr: %s", stdout, stderr)
	}

	assert.Contains(t, strings.ToLower(stderr), "interactive terminal")
	assert.Contains(t, stderr, "--org")
}

func TestCLI_DashboardStatusShowsCurrentSession(t *testing.T) {
	if testBinary == "" {
		t.Skip("CLI binary not found")
	}

	configHome, tempHome, secretStore := newDashboardTestSecretStore(t)
	require.NoError(t, secretStore.Set(ports.KeyDashboardUserToken, "user-token"))
	require.NoError(t, secretStore.Set(ports.KeyDashboardOrgToken, "org-token"))
	require.NoError(t, secretStore.Set(ports.KeyDashboardOrgPublicID, "org-1"))
	require.NoError(t, secretStore.Set(ports.KeyDashboardUserPublicID, "user-1"))
	require.NoError(t, secretStore.Set(ports.KeyDashboardAppID, "app-1"))
	require.NoError(t, secretStore.Set(ports.KeyDashboardAppRegion, "us"))

	accountServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/sessions/current", r.URL.Path)
		assert.Equal(t, "Bearer user-token", r.Header.Get("Authorization"))
		assert.Equal(t, "org-token", r.Header.Get("X-Nylas-Org"))
		writeDashboardResponse(t, w, map[string]any{
			"user": map[string]any{
				"publicId": "user-1",
			},
			"currentOrg": "org-1",
			"relations": []map[string]any{
				{"orgPublicId": "org-1", "orgName": "Acme"},
				{"orgPublicId": "org-2", "orgName": "Beta"},
			},
		})
	}))
	defer accountServer.Close()

	stdout, stderr, err := runCLIWithOverrides(30*time.Second, dashboardEnvOverrides(configHome, tempHome, map[string]string{
		"NYLAS_DASHBOARD_ACCOUNT_URL": accountServer.URL,
		"NYLAS_API_KEY":               "",
		"NYLAS_GRANT_ID":              "",
	}), "dashboard", "status")
	if err != nil {
		t.Fatalf("dashboard status failed: %v\nstderr: %s", err, stderr)
	}

	assert.Contains(t, stdout, "Logged in")
	assert.Contains(t, stdout, "User:         user-1")
	assert.Contains(t, stdout, "Organization: Acme (org-1)")
	assert.Contains(t, stdout, "Total orgs:   2")
	assert.Contains(t, stdout, "Org token:    present")
	assert.Contains(t, stdout, "Active app:   app-1 (us)")
	assert.Contains(t, stdout, "DPoP key:")
}

func TestCLI_DashboardStatusFailsWhenSessionCannotBeValidated(t *testing.T) {
	if testBinary == "" {
		t.Skip("CLI binary not found")
	}

	configHome, tempHome, secretStore := newDashboardTestSecretStore(t)
	require.NoError(t, secretStore.Set(ports.KeyDashboardUserToken, "user-token"))
	require.NoError(t, secretStore.Set(ports.KeyDashboardOrgToken, "org-token"))
	require.NoError(t, secretStore.Set(ports.KeyDashboardUserPublicID, "user-1"))

	accountServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/sessions/current":
			writeDashboardErrorResponse(t, w, http.StatusUnauthorized, "INVALID_SESSION", "")
		case "/auth/cli/refresh":
			writeDashboardErrorResponse(t, w, http.StatusUnauthorized, "INVALID_SESSION", "")
		default:
			http.NotFound(w, r)
		}
	}))
	defer accountServer.Close()

	stdout, stderr, err := runCLIWithOverrides(30*time.Second, dashboardEnvOverrides(configHome, tempHome, map[string]string{
		"NYLAS_DASHBOARD_ACCOUNT_URL": accountServer.URL,
		"NYLAS_API_KEY":               "",
		"NYLAS_GRANT_ID":              "",
	}), "dashboard", "status")
	if err == nil {
		t.Fatalf("expected dashboard status to fail when session validation fails\nstdout: %s\nstderr: %s", stdout, stderr)
	}

	assert.NotContains(t, stdout, "Logged in")
	assert.Contains(t, strings.ToLower(stderr), "invalid_session")
}

func TestCLI_DashboardAppsAndAPIKeysList(t *testing.T) {
	if testBinary == "" {
		t.Skip("CLI binary not found")
	}

	configHome, tempHome, secretStore := newDashboardTestSecretStore(t)
	require.NoError(t, secretStore.Set(ports.KeyDashboardUserToken, "user-token"))
	require.NoError(t, secretStore.Set(ports.KeyDashboardOrgToken, "org-token"))
	require.NoError(t, secretStore.Set(ports.KeyDashboardOrgPublicID, "org-1"))
	require.NoError(t, secretStore.Set(ports.KeyDashboardAppID, "app-1"))
	require.NoError(t, secretStore.Set(ports.KeyDashboardAppRegion, "us"))

	gatewayServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, err := io.ReadAll(r.Body)
		require.NoError(t, err)

		var body map[string]any
		require.NoError(t, json.Unmarshal(raw, &body))

		query := body["query"].(string)
		variables := body["variables"].(map[string]any)

		assert.Equal(t, "Bearer user-token", r.Header.Get("Authorization"))
		assert.Equal(t, "org-token", r.Header.Get("X-Nylas-Org"))
		assert.NotEmpty(t, r.Header.Get("DPoP"))

		switch {
		case strings.Contains(query, "applications("):
			filter := variables["filter"].(map[string]any)
			assert.Equal(t, "org-1", filter["orgPublicId"])
			require.NoError(t, json.NewEncoder(w).Encode(map[string]any{
				"data": map[string]any{
					"applications": map[string]any{
						"applications": []map[string]any{
							{
								"applicationId":  "app-1",
								"organizationId": "org-1",
								"region":         "us",
								"environment":    "sandbox",
								"branding": map[string]any{
									"name": "Primary",
								},
							},
						},
					},
				},
			}))
		case strings.Contains(query, "apiKeys("):
			assert.Equal(t, "app-1", variables["appId"])
			require.NoError(t, json.NewEncoder(w).Encode(map[string]any{
				"data": map[string]any{
					"apiKeys": []map[string]any{
						{
							"id":          "key-1",
							"name":        "CI",
							"status":      "active",
							"permissions": []string{"send"},
							"expiresAt":   0.0,
							"createdAt":   1710000000.0,
						},
					},
				},
			}))
		default:
			t.Fatalf("unexpected GraphQL query: %s", query)
		}
	}))
	defer gatewayServer.Close()

	env := dashboardEnvOverrides(configHome, tempHome, map[string]string{
		"NYLAS_DASHBOARD_GATEWAY_URL": gatewayServer.URL,
		"NYLAS_API_KEY":               "",
		"NYLAS_GRANT_ID":              "",
	})

	stdout, stderr, err := runCLIWithOverrides(30*time.Second, env, "dashboard", "apps", "list", "--region", "us")
	if err != nil {
		t.Fatalf("dashboard apps list failed: %v\nstderr: %s", err, stderr)
	}
	assert.Contains(t, stdout, "app-1")
	assert.Contains(t, stdout, "Primary")

	stdout, stderr, err = runCLIWithOverrides(30*time.Second, env, "dashboard", "apps", "apikeys", "list")
	if err != nil {
		t.Fatalf("dashboard apps apikeys list failed: %v\nstderr: %s", err, stderr)
	}
	assert.Contains(t, stdout, "key-1")
	assert.Contains(t, stdout, "CI")
}

func TestCLI_DashboardAppsListSurfacesGraphQLInvalidSession(t *testing.T) {
	if testBinary == "" {
		t.Skip("CLI binary not found")
	}

	configHome, tempHome, secretStore := newDashboardTestSecretStore(t)
	require.NoError(t, secretStore.Set(ports.KeyDashboardUserToken, "user-token"))
	require.NoError(t, secretStore.Set(ports.KeyDashboardOrgToken, "org-token"))
	require.NoError(t, secretStore.Set(ports.KeyDashboardOrgPublicID, "org-1"))

	gatewayServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "Bearer user-token", r.Header.Get("Authorization"))
		assert.Equal(t, "org-token", r.Header.Get("X-Nylas-Org"))
		assert.NotEmpty(t, r.Header.Get("DPoP"))

		w.WriteHeader(http.StatusUnauthorized)
		require.NoError(t, json.NewEncoder(w).Encode(map[string]any{
			"errors": []map[string]any{
				{
					"message": "INVALID_SESSION",
					"extensions": map[string]any{
						"code": "UNAUTHENTICATED",
					},
				},
			},
		}))
	}))
	defer gatewayServer.Close()

	stdout, stderr, err := runCLIWithOverrides(30*time.Second, dashboardEnvOverrides(configHome, tempHome, map[string]string{
		"NYLAS_DASHBOARD_GATEWAY_URL": gatewayServer.URL,
		"NYLAS_API_KEY":               "",
		"NYLAS_GRANT_ID":              "",
	}), "dashboard", "apps", "list", "--region", "us")
	if err == nil {
		t.Fatalf("expected dashboard apps list to fail on GraphQL INVALID_SESSION\nstdout: %s\nstderr: %s", stdout, stderr)
	}

	assert.Contains(t, strings.ToLower(stderr), "invalid_session")
	assert.Contains(t, strings.ToLower(stderr), "invalid or expired session")
}

func TestCLI_DashboardAppsListWarnsOnPartialRegionFailure(t *testing.T) {
	if testBinary == "" {
		t.Skip("CLI binary not found")
	}

	configHome, tempHome, secretStore := newDashboardTestSecretStore(t)
	require.NoError(t, secretStore.Set(ports.KeyDashboardUserToken, "user-token"))
	require.NoError(t, secretStore.Set(ports.KeyDashboardOrgToken, "org-token"))
	require.NoError(t, secretStore.Set(ports.KeyDashboardOrgPublicID, "org-1"))

	usServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.NoError(t, json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"applications": map[string]any{
					"applications": []map[string]any{
						{
							"applicationId":  "app-1",
							"organizationId": "org-1",
							"region":         "us",
							"environment":    "sandbox",
							"branding": map[string]any{
								"name": "Primary",
							},
						},
					},
				},
			},
		}))
	}))
	defer usServer.Close()

	euServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "EU unavailable", http.StatusBadGateway)
	}))
	defer euServer.Close()

	stdout, stderr, err := runCLIWithOverrides(30*time.Second, dashboardEnvOverrides(configHome, tempHome, map[string]string{
		"NYLAS_DASHBOARD_GATEWAY_US_URL": usServer.URL,
		"NYLAS_DASHBOARD_GATEWAY_EU_URL": euServer.URL,
		"NYLAS_API_KEY":                  "",
		"NYLAS_GRANT_ID":                 "",
	}), "dashboard", "apps", "list")
	if err != nil {
		t.Fatalf("dashboard apps list with partial failure failed: %v\nstderr: %s", err, stderr)
	}

	assert.Contains(t, stdout, "app-1")
	assert.Contains(t, stdout, "partial results")
}

func TestCLI_DashboardAPIKeysRequirePairedOverrides(t *testing.T) {
	if testBinary == "" {
		t.Skip("CLI binary not found")
	}

	configHome, tempHome, secretStore := newDashboardTestSecretStore(t)
	require.NoError(t, secretStore.Set(ports.KeyDashboardAppID, "app-1"))
	require.NoError(t, secretStore.Set(ports.KeyDashboardAppRegion, "us"))

	stdout, stderr, err := runCLIWithOverrides(30*time.Second, dashboardEnvOverrides(configHome, tempHome, map[string]string{
		"NYLAS_API_KEY":  "",
		"NYLAS_GRANT_ID": "",
	}), "dashboard", "apps", "apikeys", "list", "--app", "app-2")
	if err == nil {
		t.Fatalf("expected --app without --region to fail\nstdout: %s\nstderr: %s", stdout, stderr)
	}
	assert.Contains(t, strings.ToLower(stderr), "both --app and --region")

	stdout, stderr, err = runCLIWithOverrides(30*time.Second, dashboardEnvOverrides(configHome, tempHome, map[string]string{
		"NYLAS_API_KEY":  "",
		"NYLAS_GRANT_ID": "",
	}), "dashboard", "apps", "apikeys", "list", "--region", "eu")
	if err == nil {
		t.Fatalf("expected --region without --app to fail\nstdout: %s\nstderr: %s", stdout, stderr)
	}
	assert.Contains(t, strings.ToLower(stderr), "both --app and --region")
}

func TestCLI_DashboardAppsUseRequiresExplicitAppInNonInteractiveMode(t *testing.T) {
	if testBinary == "" {
		t.Skip("CLI binary not found")
	}

	configHome, tempHome, _ := newDashboardTestSecretStore(t)

	stdout, stderr, err := runCLIWithOverrides(30*time.Second, dashboardEnvOverrides(configHome, tempHome, map[string]string{
		"NYLAS_API_KEY":  "",
		"NYLAS_GRANT_ID": "",
	}), "dashboard", "apps", "use")
	if err == nil {
		t.Fatalf("expected dashboard apps use without app id to fail in non-interactive mode\nstdout: %s\nstderr: %s", stdout, stderr)
	}

	assert.Contains(t, strings.ToLower(stderr), "interactive terminal")
	assert.Contains(t, stderr, "--region")
}

func TestCLI_DashboardAppCreateRequiresSecretDeliveryInNonInteractiveMode(t *testing.T) {
	if testBinary == "" {
		t.Skip("CLI binary not found")
	}

	configHome, tempHome, secretStore := newDashboardTestSecretStore(t)
	require.NoError(t, secretStore.Set(ports.KeyDashboardOrgPublicID, "org-1"))

	stdout, stderr, err := runCLIWithOverrides(30*time.Second, dashboardEnvOverrides(configHome, tempHome, map[string]string{
		"NYLAS_API_KEY":  "",
		"NYLAS_GRANT_ID": "",
	}), "dashboard", "apps", "create", "--name", "My App", "--region", "us")
	if err == nil {
		t.Fatalf("expected dashboard apps create without secret delivery to fail in non-interactive mode\nstdout: %s\nstderr: %s", stdout, stderr)
	}

	assert.Contains(t, strings.ToLower(stderr), "client secret delivery requires an explicit choice")
	assert.Contains(t, stderr, "--secret-delivery")
}

func TestCLI_DashboardAPIKeyCreateRequiresDeliveryInNonInteractiveMode(t *testing.T) {
	if testBinary == "" {
		t.Skip("CLI binary not found")
	}

	configHome, tempHome, secretStore := newDashboardTestSecretStore(t)
	require.NoError(t, secretStore.Set(ports.KeyDashboardAppID, "app-1"))
	require.NoError(t, secretStore.Set(ports.KeyDashboardAppRegion, "us"))

	stdout, stderr, err := runCLIWithOverrides(30*time.Second, dashboardEnvOverrides(configHome, tempHome, map[string]string{
		"NYLAS_API_KEY":  "",
		"NYLAS_GRANT_ID": "",
	}), "dashboard", "apps", "apikeys", "create")
	if err == nil {
		t.Fatalf("expected dashboard apps apikeys create without delivery to fail in non-interactive mode\nstdout: %s\nstderr: %s", stdout, stderr)
	}

	assert.Contains(t, strings.ToLower(stderr), "api key delivery requires an explicit choice")
	assert.Contains(t, stderr, "--delivery")
}

func newDashboardTestSecretStore(t *testing.T) (configHome string, tempHome string, store ports.SecretStore) {
	t.Helper()

	origPassphrase := os.Getenv("NYLAS_FILE_STORE_PASSPHRASE")
	origDisableKeyring := os.Getenv("NYLAS_DISABLE_KEYRING")
	origXDGConfigHome := os.Getenv("XDG_CONFIG_HOME")
	origHome := os.Getenv("HOME")
	t.Cleanup(func() {
		setEnvOrUnset("NYLAS_FILE_STORE_PASSPHRASE", origPassphrase)
		setEnvOrUnset("NYLAS_DISABLE_KEYRING", origDisableKeyring)
		setEnvOrUnset("XDG_CONFIG_HOME", origXDGConfigHome)
		setEnvOrUnset("HOME", origHome)
	})

	tempHome = t.TempDir()
	configHome = filepath.Join(tempHome, "xdg")
	require.NoError(t, os.Setenv("XDG_CONFIG_HOME", configHome))
	require.NoError(t, os.Setenv("HOME", tempHome))
	require.NoError(t, os.Setenv("NYLAS_DISABLE_KEYRING", "true"))
	require.NoError(t, os.Setenv("NYLAS_FILE_STORE_PASSPHRASE", dashboardTestPassphrase))

	secretStore, err := keyring.NewEncryptedFileStore(config.DefaultConfigDir())
	require.NoError(t, err)
	return configHome, tempHome, secretStore
}

func dashboardEnvOverrides(configHome, tempHome string, extra map[string]string) map[string]string {
	overrides := map[string]string{
		"XDG_CONFIG_HOME":             configHome,
		"HOME":                        tempHome,
		"NYLAS_DISABLE_KEYRING":       "true",
		"NYLAS_FILE_STORE_PASSPHRASE": dashboardTestPassphrase,
	}
	for k, v := range extra {
		overrides[k] = v
	}
	return overrides
}

func writeDashboardResponse(t *testing.T, w http.ResponseWriter, data any) {
	t.Helper()

	w.Header().Set("Content-Type", "application/json")
	require.NoError(t, json.NewEncoder(w).Encode(map[string]any{
		"request_id": "req-1",
		"success":    true,
		"data":       data,
	}))
}

func writeDashboardErrorResponse(t *testing.T, w http.ResponseWriter, status int, code, message string) {
	t.Helper()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	errBody := map[string]any{
		"request_id": "req-1",
		"success":    false,
		"error": map[string]any{
			"code": code,
		},
	}
	if message != "" {
		errBody["error"].(map[string]any)["message"] = message
	}

	require.NoError(t, json.NewEncoder(w).Encode(errBody))
}
