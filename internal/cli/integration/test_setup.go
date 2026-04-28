//go:build integration
// +build integration

package integration

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/nylas/cli/internal/adapters/nylas"
	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/domain"
	"gopkg.in/yaml.v3"
)

func useLocalAuthForIntegration() bool {
	value := strings.TrimSpace(os.Getenv("NYLAS_TEST_USE_LOCAL_AUTH"))
	return value == "1" || strings.EqualFold(value, "true")
}

// acquireRateLimit waits for permission to make an API call.
// This ensures we don't exceed Nylas rate limits when running parallel tests.
// Safe to use with t.Parallel() - the rate limiter is shared across all tests.
func acquireRateLimit(t *testing.T) {
	t.Helper()
	ctx := context.Background()
	if err := apiRateLimiter.Wait(ctx); err != nil {
		t.Fatalf("Rate limiter error: %v", err)
	}
}

// skipIfMissingCreds skips the test if required credentials are missing.
// Call this at the start of tests that require API credentials.
func skipIfMissingCreds(t *testing.T) {
	t.Helper()

	if testBinary == "" {
		t.Skip("CLI binary not found - run 'go build -o bin/nylas ./cmd/nylas' first")
	}
	if testAPIKey == "" {
		t.Skip("NYLAS_API_KEY not set")
	}
	if testGrantID == "" {
		t.Skip("NYLAS_GRANT_ID not set")
	}
}

// skipIfKeyringDisabled skips the test if NYLAS_DISABLE_KEYRING=true.
// Call this for tests that require local grant store access (auth list, whoami, etc.)
// These tests can't work when keyring is disabled because they need to read/write
// local grants which are stored in the encrypted file store.
func skipIfKeyringDisabled(t *testing.T) {
	t.Helper()
	if os.Getenv("NYLAS_DISABLE_KEYRING") == "true" {
		t.Skip("Test requires keyring access - skipping when NYLAS_DISABLE_KEYRING=true")
	}
}

func cliTestEnv(overrides map[string]string) []string {
	env := os.Environ()
	for key := range overrides {
		env = removeEnvKey(env, key)
	}

	defaults := map[string]string{
		"NYLAS_API_KEY":               testAPIKey,
		"NYLAS_GRANT_ID":              testGrantID,
		"NYLAS_DISABLE_KEYRING":       "true",
		"NYLAS_FILE_STORE_PASSPHRASE": "integration-test-file-store-passphrase",
	}

	for key, value := range defaults {
		if _, overridden := overrides[key]; overridden {
			continue
		}
		env = append(env, key+"="+value)
	}

	for _, key := range []string{
		"ANTHROPIC_API_KEY",
		"OPENAI_API_KEY",
		"GROQ_API_KEY",
		"OPENROUTER_API_KEY",
		"OLLAMA_HOST",
	} {
		if _, overridden := overrides[key]; overridden {
			continue
		}
		if value := os.Getenv(key); value != "" {
			env = append(env, key+"="+value)
		}
	}

	for key, value := range overrides {
		env = append(env, key+"="+value)
	}

	return env
}

func removeEnvKey(env []string, key string) []string {
	prefix := key + "="
	filtered := env[:0]
	for _, entry := range env {
		if !strings.HasPrefix(entry, prefix) {
			filtered = append(filtered, entry)
		}
	}
	return filtered
}

// getTestClient creates a test API client
func getTestClient() *nylas.HTTPClient {
	client := nylas.NewHTTPClient()
	client.SetCredentials(testClientID, "", testAPIKey)
	return client
}

// skipIfProviderNotSupported checks if the stderr indicates the provider doesn't support
// the operation and skips the test if so.
func skipIfProviderNotSupported(t *testing.T, stderr string) {
	t.Helper()
	// Various error messages that indicate provider limitation
	if strings.Contains(stderr, "Method not supported for provider") ||
		strings.Contains(stderr, "an internal error ocurred") || // Nylas API typo
		strings.Contains(stderr, "an internal error occurred") {
		t.Skipf("Provider does not support this operation: %s", strings.TrimSpace(stderr))
	}
}

// getEnvOrDefault returns the environment variable value or a default value
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// hasAnyAIProvider checks if any AI provider is configured
func hasAnyAIProvider() bool {
	return checkOllamaAvailable() ||
		os.Getenv("ANTHROPIC_API_KEY") != "" ||
		os.Getenv("OPENAI_API_KEY") != "" ||
		os.Getenv("GROQ_API_KEY") != ""
}

// checkOllamaAvailable checks if Ollama is running
func checkOllamaAvailable() bool {
	// Check if Ollama is running by making a request to its API
	client := &http.Client{Timeout: 2 * time.Second}

	// Try common Ollama locations
	hosts := []string{
		"http://localhost:11434",
		"http://192.168.1.100:11434",
		"http://linux.local:11434",
	}

	for _, host := range hosts {
		resp, err := client.Get(host + "/api/tags")
		if err == nil {
			defer func() { _ = resp.Body.Close() }()
			if resp.StatusCode == http.StatusOK {
				return true
			}
		}
	}

	return false
}

// getTestEmail returns the test email from environment or default
func getTestEmail() string {
	if testEmail != "" {
		return testEmail
	}
	return getEnvOrDefault("NYLAS_TEST_EMAIL", "")
}

// getGrantEmail resolves the mailbox email for the active integration grant.
func getGrantEmail(t *testing.T) string {
	t.Helper()
	skipIfMissingCreds(t)
	acquireRateLimit(t)

	client := getTestClient()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	grant, err := client.GetGrant(ctx, testGrantID)
	if err != nil {
		t.Fatalf("failed to resolve grant email: %v", err)
		return ""
	}
	if grant == nil || strings.TrimSpace(grant.Email) == "" {
		t.Fatal("active grant does not expose an email address")
		return ""
	}

	return strings.TrimSpace(grant.Email)
}

// getSendTargetEmail returns the configured test email or falls back to the grant mailbox.
func getSendTargetEmail(t *testing.T) string {
	t.Helper()

	if email := strings.TrimSpace(getTestEmail()); email != "" {
		return email
	}

	return getGrantEmail(t)
}

func newSeededGrantStoreEnv(t *testing.T, grant domain.GrantInfo) map[string]string {
	t.Helper()

	configHome := t.TempDir()
	cacheHome := filepath.Join(configHome, "cache")
	passphrase := "integration-test-file-store-passphrase"

	t.Setenv("NYLAS_DISABLE_KEYRING", "true")
	t.Setenv("NYLAS_FILE_STORE_PASSPHRASE", passphrase)
	t.Setenv("XDG_CONFIG_HOME", configHome)
	t.Setenv("XDG_CACHE_HOME", cacheHome)
	t.Setenv("HOME", configHome)

	grantStore, err := common.NewDefaultGrantStore()
	if err != nil {
		t.Fatalf("failed to create seeded grant store: %v", err)
	}
	if err := grantStore.SaveGrant(grant); err != nil {
		t.Fatalf("failed to seed grant store: %v", err)
	}

	return map[string]string{
		"NYLAS_GRANT_ID":              "",
		"NYLAS_DISABLE_KEYRING":       "true",
		"NYLAS_FILE_STORE_PASSPHRASE": passphrase,
		"XDG_CONFIG_HOME":             configHome,
		"XDG_CACHE_HOME":              cacheHome,
		"HOME":                        configHome,
	}
}

// getAvailableProvider returns the first available AI provider
func getAvailableProvider() string {
	if checkOllamaAvailable() {
		return "ollama"
	}
	if getEnvOrDefault("ANTHROPIC_API_KEY", "") != "" {
		return "claude"
	}
	if getEnvOrDefault("OPENAI_API_KEY", "") != "" {
		return "openai"
	}
	if getEnvOrDefault("GROQ_API_KEY", "") != "" {
		return "groq"
	}
	return "" // Return empty string when no provider is available
}

// getAIConfigFromUserConfig reads the AI configuration from ~/.config/nylas/config.yaml
func getAIConfigFromUserConfig() map[string]any {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}

	configPath := filepath.Join(home, ".config", "nylas", "config.yaml")
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil
	}

	var config map[string]any
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil
	}

	aiConfig, ok := config["ai"].(map[string]any)
	if !ok {
		return nil
	}

	return aiConfig
}

// hasDefaultAIProvider checks if a default AI provider is configured in config.yaml
func hasDefaultAIProvider() bool {
	aiConfig := getAIConfigFromUserConfig()
	if aiConfig == nil {
		return false
	}

	provider, ok := aiConfig["default_provider"].(string)
	return ok && provider != ""
}

// skipIfNoDefaultAIProvider skips tests if no default AI provider is configured
func skipIfNoDefaultAIProvider(t *testing.T) {
	t.Helper()
	if !hasDefaultAIProvider() {
		t.Skip("No default AI provider configured in config.yaml. Set ai.default_provider to run AI tests.")
	}
}

// getWorkingHoursFromUserConfig reads working hours configuration from ~/.config/nylas/config.yaml
func getWorkingHoursFromUserConfig() map[string]any {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}

	configPath := filepath.Join(home, ".config", "nylas", "config.yaml")
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil
	}

	var config map[string]any
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil
	}

	whConfig, ok := config["working_hours"].(map[string]any)
	if !ok {
		return nil
	}

	return whConfig
}
