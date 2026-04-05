//go:build integration

package integration

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
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
	origHome := os.Getenv("HOME")

	defer func() {
		setEnvOrUnset("NYLAS_GRANT_ID", origGrantID)
		setEnvOrUnset("NYLAS_DISABLE_KEYRING", origDisableKeyring)
		setEnvOrUnset("NYLAS_FILE_STORE_PASSPHRASE", origFileStorePassphrase)
		setEnvOrUnset("XDG_CONFIG_HOME", origXDGConfigHome)
		setEnvOrUnset("HOME", origHome)
	}()

	tempDir := t.TempDir()
	configHome := filepath.Join(tempDir, "xdg")
	_ = os.Setenv("XDG_CONFIG_HOME", configHome)
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

	secretStore, err := keyring.NewEncryptedFileStore(filepath.Dir(configPath))
	if err != nil {
		t.Fatalf("failed to create secret store: %v", err)
	}
	grantStore := keyring.NewGrantStore(secretStore)
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

func TestIntegration_GetGrantID_DoesNotFallbackToConfigWhenStoreLocked(t *testing.T) {
	origGrantID := os.Getenv("NYLAS_GRANT_ID")
	origDisableKeyring := os.Getenv("NYLAS_DISABLE_KEYRING")
	origFileStorePassphrase := os.Getenv("NYLAS_FILE_STORE_PASSPHRASE")
	origXDGConfigHome := os.Getenv("XDG_CONFIG_HOME")
	origHome := os.Getenv("HOME")

	defer func() {
		setEnvOrUnset("NYLAS_GRANT_ID", origGrantID)
		setEnvOrUnset("NYLAS_DISABLE_KEYRING", origDisableKeyring)
		setEnvOrUnset("NYLAS_FILE_STORE_PASSPHRASE", origFileStorePassphrase)
		setEnvOrUnset("XDG_CONFIG_HOME", origXDGConfigHome)
		setEnvOrUnset("HOME", origHome)
	}()

	tempDir := t.TempDir()
	configHome := filepath.Join(tempDir, "xdg")
	configDir := filepath.Join(configHome, "nylas")
	_ = os.Setenv("XDG_CONFIG_HOME", configHome)
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
	grantStore := keyring.NewGrantStore(secretStore)
	if err := grantStore.SaveGrant(domain.GrantInfo{ID: "stored-default", Email: "active@example.com"}); err != nil {
		t.Fatalf("failed to save grant: %v", err)
	}
	if err := grantStore.SetDefaultGrant("stored-default"); err != nil {
		t.Fatalf("failed to set default grant: %v", err)
	}

	_ = os.Setenv("NYLAS_FILE_STORE_PASSPHRASE", "")

	grantID, err := common.GetGrantID(nil)
	if err == nil {
		t.Fatalf("expected GetGrantID to fail, got %q", grantID)
	}
	if grantID != "" {
		t.Fatalf("grantID = %q, want empty string", grantID)
	}
	if !strings.Contains(err.Error(), "NYLAS_FILE_STORE_PASSPHRASE") {
		t.Fatalf("error %q does not mention NYLAS_FILE_STORE_PASSPHRASE", err.Error())
	}
}

func TestCLI_AuthRemove_UpdatesDefaultGrantAndConfig(t *testing.T) {
	if testBinary == "" {
		t.Skip("CLI binary not found")
	}

	origFileStorePassphrase := os.Getenv("NYLAS_FILE_STORE_PASSPHRASE")
	defer setEnvOrUnset("NYLAS_FILE_STORE_PASSPHRASE", origFileStorePassphrase)
	_ = os.Setenv("NYLAS_FILE_STORE_PASSPHRASE", "integration-test-file-store-passphrase")

	tempDir := t.TempDir()
	configHome := filepath.Join(tempDir, "xdg")
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

	secretStore, err := keyring.NewEncryptedFileStore(filepath.Dir(configPath))
	if err != nil {
		t.Fatalf("failed to create secret store: %v", err)
	}
	grantStore := keyring.NewGrantStore(secretStore)
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
		"XDG_CONFIG_HOME": configHome,
		"HOME":            tempDir,
		"NYLAS_API_KEY":   "",
		"NYLAS_GRANT_ID":  "",
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
	if len(cfg.Grants) != 1 || cfg.Grants[0].ID != "grant-2" {
		t.Fatalf("unexpected config grants after remove: %+v", cfg.Grants)
	}
}

func TestCLI_AuthList_RequiresFileStorePassphrase(t *testing.T) {
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
	grantStore := keyring.NewGrantStore(secretStore)
	if err := grantStore.SaveGrant(domain.GrantInfo{
		ID:       "grant-locked",
		Email:    "locked@example.com",
		Provider: domain.ProviderGoogle,
	}); err != nil {
		t.Fatalf("failed to save grant: %v", err)
	}

	stdout, stderr, err := runCLIWithOverrides(30*time.Second, map[string]string{
		"XDG_CONFIG_HOME":             configHome,
		"HOME":                        tempDir,
		"NYLAS_API_KEY":               "",
		"NYLAS_GRANT_ID":              "",
		"NYLAS_FILE_STORE_PASSPHRASE": "",
	}, "auth", "list")
	if err == nil {
		t.Fatalf("expected auth list to fail without passphrase\nstdout: %s\nstderr: %s", stdout, stderr)
	}
	if !strings.Contains(stderr, "NYLAS_FILE_STORE_PASSPHRASE") {
		t.Fatalf("stderr %q does not mention NYLAS_FILE_STORE_PASSPHRASE", stderr)
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
