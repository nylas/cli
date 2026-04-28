//go:build integration

package integration

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/nylas/cli/internal/adapters/config"
	"github.com/nylas/cli/internal/adapters/keyring"
	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/domain"
	"github.com/nylas/cli/internal/ports"
)

func TestIntegration_GetGrantID_PrefersStoredDefaultOverConfig(t *testing.T) {
	origGrantID := os.Getenv("NYLAS_GRANT_ID")
	origDisableKeyring := os.Getenv("NYLAS_DISABLE_KEYRING")
	origFileStorePassphrase := os.Getenv("NYLAS_FILE_STORE_PASSPHRASE")
	origXDGConfigHome := os.Getenv("XDG_CONFIG_HOME")
	origXDGCacheHome := os.Getenv("XDG_CACHE_HOME")
	origHome := os.Getenv("HOME")

	defer func() {
		setEnvOrUnset("NYLAS_GRANT_ID", origGrantID)
		setEnvOrUnset("NYLAS_DISABLE_KEYRING", origDisableKeyring)
		setEnvOrUnset("NYLAS_FILE_STORE_PASSPHRASE", origFileStorePassphrase)
		setEnvOrUnset("XDG_CONFIG_HOME", origXDGConfigHome)
		setEnvOrUnset("XDG_CACHE_HOME", origXDGCacheHome)
		setEnvOrUnset("HOME", origHome)
	}()

	tempDir := t.TempDir()
	configHome := filepath.Join(tempDir, "xdg")
	cacheHome := filepath.Join(tempDir, "cache")
	_ = os.Setenv("XDG_CONFIG_HOME", configHome)
	_ = os.Setenv("XDG_CACHE_HOME", cacheHome)
	_ = os.Setenv("HOME", tempDir)
	_ = os.Setenv("NYLAS_DISABLE_KEYRING", "true")
	_ = os.Setenv("NYLAS_FILE_STORE_PASSPHRASE", "integration-test-file-store-passphrase")
	_ = os.Setenv("NYLAS_GRANT_ID", "")

	configPath := filepath.Join(configHome, "nylas", "config.yaml")
	configStore := config.NewFileStore(configPath)
	if err := configStore.Save(&domain.Config{
		Region:       "us",
		DefaultGrant: "stale-config-grant",
		Grants:       []domain.GrantInfo{{ID: "stale-config-grant", Email: "stale@example.com"}},
	}); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	grantStore, err := common.NewDefaultGrantStore()
	if err != nil {
		t.Fatalf("failed to create grant store: %v", err)
	}
	if err := grantStore.SaveGrant(domain.GrantInfo{ID: "stored-default", Email: "active@example.com"}); err != nil {
		t.Fatalf("failed to save grant: %v", err)
	}
	if err := grantStore.SetDefaultGrant("stored-default"); err != nil {
		t.Fatalf("failed to set default grant: %v", err)
	}

	grantID, err := common.GetGrantID(nil)
	if err != nil {
		t.Fatalf("GetGrantID failed: %v", err)
	}
	if grantID != "stored-default" {
		t.Fatalf("GetGrantID returned %q, want %q", grantID, "stored-default")
	}
}

func TestIntegration_GetGrantID_FallsBackToConfigWhenLegacyStoreLocked(t *testing.T) {
	origGrantID := os.Getenv("NYLAS_GRANT_ID")
	origDisableKeyring := os.Getenv("NYLAS_DISABLE_KEYRING")
	origFileStorePassphrase := os.Getenv("NYLAS_FILE_STORE_PASSPHRASE")
	origXDGConfigHome := os.Getenv("XDG_CONFIG_HOME")
	origXDGCacheHome := os.Getenv("XDG_CACHE_HOME")
	origHome := os.Getenv("HOME")

	defer func() {
		setEnvOrUnset("NYLAS_GRANT_ID", origGrantID)
		setEnvOrUnset("NYLAS_DISABLE_KEYRING", origDisableKeyring)
		setEnvOrUnset("NYLAS_FILE_STORE_PASSPHRASE", origFileStorePassphrase)
		setEnvOrUnset("XDG_CONFIG_HOME", origXDGConfigHome)
		setEnvOrUnset("XDG_CACHE_HOME", origXDGCacheHome)
		setEnvOrUnset("HOME", origHome)
	}()

	tempDir := t.TempDir()
	configHome := filepath.Join(tempDir, "xdg")
	cacheHome := filepath.Join(tempDir, "cache")
	configDir := filepath.Join(configHome, "nylas")
	_ = os.Setenv("XDG_CONFIG_HOME", configHome)
	_ = os.Setenv("XDG_CACHE_HOME", cacheHome)
	_ = os.Setenv("HOME", tempDir)
	_ = os.Setenv("NYLAS_DISABLE_KEYRING", "true")
	_ = os.Setenv("NYLAS_FILE_STORE_PASSPHRASE", "integration-test-file-store-passphrase")
	_ = os.Setenv("NYLAS_GRANT_ID", "")

	configStore := config.NewFileStore(filepath.Join(configDir, "config.yaml"))
	if err := configStore.Save(&domain.Config{
		Region:       "us",
		DefaultGrant: "stale-config-grant",
		Grants:       []domain.GrantInfo{{ID: "stale-config-grant", Email: "stale@example.com"}},
	}); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	secretStore, err := keyring.NewEncryptedFileStore(configDir)
	if err != nil {
		t.Fatalf("failed to create secret store: %v", err)
	}
	if err := secretStore.Set("grants", `[{"id":"stored-default","email":"active@example.com","provider":"google"}]`); err != nil {
		t.Fatalf("failed to save legacy grants: %v", err)
	}
	if err := secretStore.Set("default_grant", "stored-default"); err != nil {
		t.Fatalf("failed to save legacy default grant: %v", err)
	}

	_ = os.Setenv("NYLAS_FILE_STORE_PASSPHRASE", "")

	grantID, err := common.GetGrantID(nil)
	if err != nil {
		t.Fatalf("GetGrantID failed: %v", err)
	}
	if grantID != "stale-config-grant" {
		t.Fatalf("grantID = %q, want %q", grantID, "stale-config-grant")
	}
}

func TestCLI_AuthRemove_UpdatesDefaultGrantAndConfig(t *testing.T) {
	if testBinary == "" {
		t.Skip("CLI binary not found")
	}

	origFileStorePassphrase := os.Getenv("NYLAS_FILE_STORE_PASSPHRASE")
	origDisableKeyring := os.Getenv("NYLAS_DISABLE_KEYRING")
	origXDGConfigHome := os.Getenv("XDG_CONFIG_HOME")
	origXDGCacheHome := os.Getenv("XDG_CACHE_HOME")
	origHome := os.Getenv("HOME")
	defer func() {
		setEnvOrUnset("NYLAS_FILE_STORE_PASSPHRASE", origFileStorePassphrase)
		setEnvOrUnset("NYLAS_DISABLE_KEYRING", origDisableKeyring)
		setEnvOrUnset("XDG_CONFIG_HOME", origXDGConfigHome)
		setEnvOrUnset("XDG_CACHE_HOME", origXDGCacheHome)
		setEnvOrUnset("HOME", origHome)
	}()
	_ = os.Setenv("NYLAS_FILE_STORE_PASSPHRASE", "integration-test-file-store-passphrase")
	_ = os.Setenv("NYLAS_DISABLE_KEYRING", "true")

	tempDir := t.TempDir()
	configHome := filepath.Join(tempDir, "xdg")
	cacheHome := filepath.Join(tempDir, "cache")
	_ = os.Setenv("XDG_CONFIG_HOME", configHome)
	_ = os.Setenv("XDG_CACHE_HOME", cacheHome)
	_ = os.Setenv("HOME", tempDir)
	configPath := filepath.Join(configHome, "nylas", "config.yaml")
	configStore := config.NewFileStore(configPath)
	if err := configStore.Save(&domain.Config{
		Region:       "us",
		DefaultGrant: "grant-1",
		Grants: []domain.GrantInfo{
			{ID: "grant-1", Email: "user1@example.com"},
			{ID: "grant-2", Email: "user2@example.com"},
		},
	}); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	grantStore, err := common.NewDefaultGrantStore()
	if err != nil {
		t.Fatalf("failed to create grant store: %v", err)
	}
	if err := grantStore.SaveGrant(domain.GrantInfo{ID: "grant-1", Email: "user1@example.com"}); err != nil {
		t.Fatalf("failed to save first grant: %v", err)
	}
	if err := grantStore.SaveGrant(domain.GrantInfo{ID: "grant-2", Email: "user2@example.com"}); err != nil {
		t.Fatalf("failed to save second grant: %v", err)
	}
	if err := grantStore.SetDefaultGrant("grant-1"); err != nil {
		t.Fatalf("failed to set default grant: %v", err)
	}

	stdout, stderr, err := runCLIWithOverrides(30*time.Second, map[string]string{
		"XDG_CONFIG_HOME":       configHome,
		"XDG_CACHE_HOME":        cacheHome,
		"HOME":                  tempDir,
		"NYLAS_DISABLE_KEYRING": "true",
		"NYLAS_API_KEY":         "",
		"NYLAS_GRANT_ID":        "",
	}, "auth", "remove", "grant-1")
	if err != nil {
		t.Fatalf("auth remove failed: %v\nstderr: %s", err, stderr)
	}
	if stdout == "" {
		t.Fatal("expected auth remove output")
	}

	defaultGrant, err := grantStore.GetDefaultGrant()
	if err != nil {
		t.Fatalf("failed to read default grant: %v", err)
	}
	if defaultGrant != "grant-2" {
		t.Fatalf("default grant = %q, want %q", defaultGrant, "grant-2")
	}

	grants, err := grantStore.ListGrants()
	if err != nil {
		t.Fatalf("failed to list grants: %v", err)
	}
	if len(grants) != 1 || grants[0].ID != "grant-2" {
		t.Fatalf("unexpected grants after remove: %+v", grants)
	}

	cfg, err := configStore.Load()
	if err != nil {
		t.Fatalf("failed to reload config: %v", err)
	}
	if cfg.DefaultGrant != "grant-2" {
		t.Fatalf("config default grant = %q, want %q", cfg.DefaultGrant, "grant-2")
	}
	if len(cfg.Grants) != 0 {
		t.Fatalf("unexpected config grants after remove: %+v", cfg.Grants)
	}
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("failed to read config file: %v", err)
	}
	if strings.Contains(string(data), "grants:") {
		t.Fatalf("config file should not contain grants list:\n%s", string(data))
	}
}

func TestCLI_AuthList_DoesNotRequireFileStorePassphrase(t *testing.T) {
	if testBinary == "" {
		t.Skip("CLI binary not found")
	}

	tempDir := t.TempDir()
	configHome := filepath.Join(tempDir, "xdg")
	cacheHome := filepath.Join(tempDir, "cache")

	stdout, stderr, err := runCLIWithOverrides(30*time.Second, map[string]string{
		"XDG_CONFIG_HOME":             configHome,
		"XDG_CACHE_HOME":              cacheHome,
		"HOME":                        tempDir,
		"NYLAS_DISABLE_KEYRING":       "true",
		"NYLAS_API_KEY":               "invalid-api-key",
		"NYLAS_GRANT_ID":              "",
		"NYLAS_FILE_STORE_PASSPHRASE": "",
	}, "auth", "list")
	if err == nil {
		t.Fatalf("expected auth list to fail with invalid API key\nstdout: %s\nstderr: %s", stdout, stderr)
	}
	if strings.Contains(stderr, "NYLAS_FILE_STORE_PASSPHRASE") {
		t.Fatalf("auth list should not require file-store passphrase, stderr: %q", stderr)
	}
	if !strings.Contains(stderr, "access credential") && !strings.Contains(stderr, "API key") {
		t.Fatalf("stderr %q does not mention API credential failure", stderr)
	}
}

func TestCLI_AuthProviders_RequiresFileStorePassphrase(t *testing.T) {
	if testBinary == "" {
		t.Skip("CLI binary not found")
	}

	origFileStorePassphrase := os.Getenv("NYLAS_FILE_STORE_PASSPHRASE")
	defer setEnvOrUnset("NYLAS_FILE_STORE_PASSPHRASE", origFileStorePassphrase)
	_ = os.Setenv("NYLAS_FILE_STORE_PASSPHRASE", "integration-test-file-store-passphrase")

	tempDir := t.TempDir()
	configHome := filepath.Join(tempDir, "xdg")
	configDir := filepath.Join(configHome, "nylas")

	secretStore, err := keyring.NewEncryptedFileStore(configDir)
	if err != nil {
		t.Fatalf("failed to create secret store: %v", err)
	}
	if err := secretStore.Set(ports.KeyAPIKey, "stored-api-key"); err != nil {
		t.Fatalf("failed to save api key: %v", err)
	}

	stdout, stderr, err := runCLIWithOverrides(30*time.Second, map[string]string{
		"XDG_CONFIG_HOME":             configHome,
		"HOME":                        tempDir,
		"NYLAS_API_KEY":               "",
		"NYLAS_GRANT_ID":              "",
		"NYLAS_FILE_STORE_PASSPHRASE": "",
	}, "auth", "providers")
	if err == nil {
		t.Fatalf("expected auth providers to fail without passphrase\nstdout: %s\nstderr: %s", stdout, stderr)
	}
	if !strings.Contains(stderr, "NYLAS_FILE_STORE_PASSPHRASE") {
		t.Fatalf("stderr %q does not mention NYLAS_FILE_STORE_PASSPHRASE", stderr)
	}
}

func TestCLI_ConnectorSurfaces_HideInboxProvider(t *testing.T) {
	if testBinary == "" {
		t.Skip("CLI binary not found")
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v3/connectors" {
			http.NotFound(w, r)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[
			{"id":"conn-inbox-1","name":"Inbox","provider":"inbox"},
			{"id":"conn-google-1","name":"","provider":"google","settings":{"client_id":"google-client-id"}},
			{"id":"conn-imap-1","name":"Custom IMAP","provider":"imap","scopes":["mail.read_only","mail.send"]}
		]}`))
	}))
	defer server.Close()

	tempDir := t.TempDir()
	configHome := filepath.Join(tempDir, "xdg")
	configPath := filepath.Join(configHome, "nylas", "config.yaml")
	configStore := config.NewFileStore(configPath)
	if err := configStore.Save(&domain.Config{
		Region: "us",
		API:    &domain.APIConfig{BaseURL: server.URL},
	}); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	overrides := map[string]string{
		"XDG_CONFIG_HOME":             configHome,
		"HOME":                        tempDir,
		"NYLAS_API_KEY":               "test-api-key",
		"NYLAS_CLIENT_ID":             "",
		"NYLAS_CLIENT_SECRET":         "",
		"NYLAS_GRANT_ID":              "",
		"NYLAS_DISABLE_KEYRING":       "true",
		"NYLAS_FILE_STORE_PASSPHRASE": "integration-test-file-store-passphrase",
	}

	stdout, stderr, err := runCLIWithOverrides(30*time.Second, overrides, "auth", "providers")
	if err != nil {
		t.Fatalf("auth providers failed: %v\nstdout: %s\nstderr: %s", err, stdout, stderr)
	}
	for _, unwanted := range []string{"Provider:   inbox", "\n  Inbox\n"} {
		if strings.Contains(stdout, unwanted) {
			t.Fatalf("auth providers unexpectedly exposed inbox connector: %s", stdout)
		}
	}
	for _, wanted := range []string{"Provider:   google", "Provider:   imap"} {
		if !strings.Contains(stdout, wanted) {
			t.Fatalf("auth providers output %q does not contain %q", stdout, wanted)
		}
	}

	stdout, stderr, err = runCLIWithOverrides(30*time.Second, overrides, "auth", "providers", "--json")
	if err != nil {
		t.Fatalf("auth providers --json failed: %v\nstdout: %s\nstderr: %s", err, stdout, stderr)
	}
	assertNoInboxConnector(t, stdout)

	stdout, stderr, err = runCLIWithOverrides(30*time.Second, overrides, "admin", "connectors", "list")
	if err != nil {
		t.Fatalf("admin connectors list failed: %v\nstdout: %s\nstderr: %s", err, stdout, stderr)
	}
	if strings.Contains(stdout, "inbox") {
		t.Fatalf("admin connectors list unexpectedly exposed inbox connector: %s", stdout)
	}
	for _, wanted := range []string{"google", "imap"} {
		if !strings.Contains(stdout, wanted) {
			t.Fatalf("admin connectors list output %q does not contain %q", stdout, wanted)
		}
	}

	stdout, stderr, err = runCLIWithOverrides(30*time.Second, overrides, "admin", "connectors", "list", "--json")
	if err != nil {
		t.Fatalf("admin connectors list --json failed: %v\nstdout: %s\nstderr: %s", err, stdout, stderr)
	}
	assertNoInboxConnector(t, stdout)
}

func TestCLI_AuthProviders_HidesEmptyConnectorFields(t *testing.T) {
	if testBinary == "" {
		t.Skip("CLI binary not found")
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v3/connectors" {
			http.NotFound(w, r)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[
			{"name":"","provider":"google","settings":{"client_id":"google-client-id"}},
			{"id":"conn-imap-1","name":"Custom IMAP","provider":"imap","scopes":["mail.read_only","mail.send"]}
		]}`))
	}))
	defer server.Close()

	tempDir := t.TempDir()
	configHome := filepath.Join(tempDir, "xdg")
	configPath := filepath.Join(configHome, "nylas", "config.yaml")
	configStore := config.NewFileStore(configPath)
	if err := configStore.Save(&domain.Config{
		Region: "us",
		API:    &domain.APIConfig{BaseURL: server.URL},
	}); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	stdout, stderr, err := runCLIWithOverrides(30*time.Second, map[string]string{
		"XDG_CONFIG_HOME":             configHome,
		"HOME":                        tempDir,
		"NYLAS_API_KEY":               "test-api-key",
		"NYLAS_CLIENT_ID":             "",
		"NYLAS_CLIENT_SECRET":         "",
		"NYLAS_GRANT_ID":              "",
		"NYLAS_DISABLE_KEYRING":       "true",
		"NYLAS_FILE_STORE_PASSPHRASE": "integration-test-file-store-passphrase",
	}, "auth", "providers")
	if err != nil {
		t.Fatalf("auth providers failed: %v\nstdout: %s\nstderr: %s", err, stdout, stderr)
	}

	for _, want := range []string{
		"Available Authentication Providers:",
		"  Google",
		"    Provider:   google",
		"  Custom IMAP",
		"    ID:         conn-imap-1",
		"    Scopes:     2 configured",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout %q does not contain %q", stdout, want)
		}
	}

	for _, unwanted := range []string{
		"Name:       \n",
		"ID:         \n",
	} {
		if strings.Contains(stdout, unwanted) {
			t.Fatalf("stdout %q unexpectedly contains blank field %q", stdout, unwanted)
		}
	}
}

func TestCLI_AdminConnectorsCreate_RejectsInboxProvider(t *testing.T) {
	if testBinary == "" {
		t.Skip("CLI binary not found")
	}

	tempDir := t.TempDir()
	configHome := filepath.Join(tempDir, "xdg")
	configPath := filepath.Join(configHome, "nylas", "config.yaml")
	configStore := config.NewFileStore(configPath)
	if err := configStore.Save(&domain.Config{Region: "us"}); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	stdout, stderr, err := runCLIWithOverrides(30*time.Second, map[string]string{
		"XDG_CONFIG_HOME":             configHome,
		"HOME":                        tempDir,
		"NYLAS_API_KEY":               "test-api-key",
		"NYLAS_DISABLE_KEYRING":       "true",
		"NYLAS_FILE_STORE_PASSPHRASE": "integration-test-file-store-passphrase",
	}, "admin", "connectors", "create", "--name", "Removed Inbox", "--provider", "inbox")
	if err == nil {
		t.Fatalf("expected admin connectors create --provider inbox to fail\nstdout: %s\nstderr: %s", stdout, stderr)
	}
	if !strings.Contains(stderr, "invalid provider: inbox") {
		t.Fatalf("stderr %q does not mention rejected inbox provider", stderr)
	}
}

func TestCLI_AdminConnectorsShow_HidesInboxProvider(t *testing.T) {
	if testBinary == "" {
		t.Skip("CLI binary not found")
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v3/connectors/conn-inbox-1" {
			http.NotFound(w, r)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"id":"conn-inbox-1","name":"Inbox","provider":"inbox"}}`))
	}))
	defer server.Close()

	tempDir := t.TempDir()
	configHome := filepath.Join(tempDir, "xdg")
	configPath := filepath.Join(configHome, "nylas", "config.yaml")
	configStore := config.NewFileStore(configPath)
	if err := configStore.Save(&domain.Config{
		Region: "us",
		API:    &domain.APIConfig{BaseURL: server.URL},
	}); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	overrides := map[string]string{
		"XDG_CONFIG_HOME":             configHome,
		"HOME":                        tempDir,
		"NYLAS_API_KEY":               "test-api-key",
		"NYLAS_DISABLE_KEYRING":       "true",
		"NYLAS_FILE_STORE_PASSPHRASE": "integration-test-file-store-passphrase",
	}

	for _, args := range [][]string{
		{"admin", "connectors", "show", "conn-inbox-1"},
		{"admin", "connectors", "show", "conn-inbox-1", "--json"},
	} {
		stdout, stderr, err := runCLIWithOverrides(30*time.Second, overrides, args...)
		if err == nil {
			t.Fatalf("expected %v to fail\nstdout: %s\nstderr: %s", args, stdout, stderr)
		}
		if !strings.Contains(stderr, "connector not found") {
			t.Fatalf("stderr %q does not report connector not found for %v", stderr, args)
		}
		if strings.Contains(stdout, "provider") || strings.Contains(stdout, "inbox") {
			t.Fatalf("stdout %q unexpectedly exposed inbox connector for %v", stdout, args)
		}
	}
}

func assertNoInboxConnector(t *testing.T, stdout string) {
	t.Helper()

	var connectors []map[string]any
	if err := json.Unmarshal([]byte(stdout), &connectors); err != nil {
		t.Fatalf("failed to parse connectors JSON: %v\noutput: %s", err, stdout)
	}

	for _, connector := range connectors {
		if provider, _ := connector["provider"].(string); strings.EqualFold(provider, "inbox") {
			t.Fatalf("JSON output still exposed removed inbox provider: %s", stdout)
		}
	}
}

func TestCLI_MCPServe_RequiresFileStorePassphrase(t *testing.T) {
	if testBinary == "" {
		t.Skip("CLI binary not found")
	}

	origFileStorePassphrase := os.Getenv("NYLAS_FILE_STORE_PASSPHRASE")
	defer setEnvOrUnset("NYLAS_FILE_STORE_PASSPHRASE", origFileStorePassphrase)
	_ = os.Setenv("NYLAS_FILE_STORE_PASSPHRASE", "integration-test-file-store-passphrase")

	tempDir := t.TempDir()
	configHome := filepath.Join(tempDir, "xdg")
	configDir := filepath.Join(configHome, "nylas")

	secretStore, err := keyring.NewEncryptedFileStore(configDir)
	if err != nil {
		t.Fatalf("failed to create secret store: %v", err)
	}
	if err := secretStore.Set(ports.KeyAPIKey, "stored-api-key"); err != nil {
		t.Fatalf("failed to save api key: %v", err)
	}

	stdout, stderr, err := runCLIWithOverrides(30*time.Second, map[string]string{
		"XDG_CONFIG_HOME":             configHome,
		"HOME":                        tempDir,
		"NYLAS_API_KEY":               "",
		"NYLAS_GRANT_ID":              "",
		"NYLAS_FILE_STORE_PASSPHRASE": "",
	}, "mcp", "serve")
	if err == nil {
		t.Fatalf("expected mcp serve to fail without passphrase\nstdout: %s\nstderr: %s", stdout, stderr)
	}
	if !strings.Contains(stderr, "NYLAS_FILE_STORE_PASSPHRASE") {
		t.Fatalf("stderr %q does not mention NYLAS_FILE_STORE_PASSPHRASE", stderr)
	}
}

func TestCLI_WebhookServer_RejectsUnsignedRequests(t *testing.T) {
	if testBinary == "" {
		t.Skip("CLI binary not found")
	}

	port := freeTCPPort(t)
	secret := "test-webhook-secret"
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, testBinary, "webhook", "server", "--quiet", "--port", strconv.Itoa(port), "--secret", secret)
	cmd.Env = cliTestEnv(map[string]string{
		"NYLAS_API_KEY":  "",
		"NYLAS_GRANT_ID": "",
	})
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Start(); err != nil {
		t.Fatalf("failed to start webhook server: %v", err)
	}
	t.Cleanup(func() {
		if cmd.Process != nil {
			_ = cmd.Process.Signal(os.Interrupt)
		}
		_ = cmd.Wait()
	})

	baseURL := "http://127.0.0.1:" + strconv.Itoa(port)
	waitForServer(t, baseURL+"/health")

	payload := []byte(`{"type":"message.created","id":"event-123"}`)

	resp, err := http.Post(baseURL+"/webhook", "application/json", bytes.NewReader(payload))
	if err != nil {
		t.Fatalf("missing-signature request failed: %v", err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("missing signature status = %d, want %d", resp.StatusCode, http.StatusUnauthorized)
	}

	req, err := http.NewRequest(http.MethodPost, baseURL+"/webhook", bytes.NewReader(payload))
	if err != nil {
		t.Fatalf("failed to create invalid-signature request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Nylas-Signature", "invalid-signature")
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("invalid-signature request failed: %v", err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("invalid signature status = %d, want %d", resp.StatusCode, http.StatusForbidden)
	}

	req, err = http.NewRequest(http.MethodPost, baseURL+"/webhook", bytes.NewReader(payload))
	if err != nil {
		t.Fatalf("failed to create valid-signature request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Nylas-Signature", signTestWebhookPayload(secret, payload))
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("valid-signature request failed: %v", err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("valid signature status = %d, want %d\nstdout: %s\nstderr: %s", resp.StatusCode, http.StatusOK, stdout.String(), stderr.String())
	}
}

func freeTCPPort(t *testing.T) int {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to allocate test port: %v", err)
	}
	defer func() { _ = listener.Close() }()

	return listener.Addr().(*net.TCPAddr).Port
}

func waitForServer(t *testing.T, url string) {
	t.Helper()

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		resp, err := http.Get(url)
		if err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return
			}
		}
		time.Sleep(100 * time.Millisecond)
	}

	t.Fatalf("server %s did not become healthy in time", url)
}

func signTestWebhookPayload(secret string, payload []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write(payload)
	return hex.EncodeToString(mac.Sum(nil))
}

func setEnvOrUnset(key, value string) {
	if value != "" {
		_ = os.Setenv(key, value)
	} else {
		_ = os.Unsetenv(key)
	}
}
