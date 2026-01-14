//go:build integration

// Package integration provides integration tests for all CLI commands.
// Run with: go test -tags=integration -v ./internal/cli/integration/...
//
// Required environment variables:
//   - NYLAS_API_KEY: Your Nylas API key
//   - NYLAS_GRANT_ID: A valid grant ID
//   - NYLAS_CLIENT_ID: Your Nylas client ID (optional)
//
// Optional environment variables:
//   - NYLAS_TEST_EMAIL: Email address for send tests (default: uses grant email)
//   - NYLAS_TEST_SEND_EMAIL: Set to "true" to enable send tests
//   - NYLAS_TEST_DELETE: Set to "true" to enable delete tests
//   - NYLAS_TEST_AUTH_LOGOUT: Set to "true" to enable auth logout tests
//   - NYLAS_TEST_RATE_LIMIT_RPS: API rate limit (requests/sec, default: 2.0)
//   - NYLAS_TEST_RATE_LIMIT_BURST: API rate limit burst capacity (default: 5)
//
// Parallel Testing:
//   Tests can use t.Parallel() to run concurrently. The package includes a global
//   rate limiter that ensures API calls don't exceed Nylas rate limits.
//
//   Usage:
//     func TestExample(t *testing.T) {
//         skipIfMissingCreds(t)
//         t.Parallel()  // Enable parallel execution
//
//         // For API calls, use rate-limited functions:
//         stdout, stderr, err := runCLIWithRateLimit(t, "calendar", "events", "list")
//         // OR manually acquire rate limit:
//         acquireRateLimit(t)
//         stdout, stderr, err := runCLI("calendar", "events", "list")
//     }
//
//   For offline commands (version, help), rate limiting is not needed:
//     stdout, _, _ := runCLI("version")  // No rate limit needed
//
// Test files are organized by feature:
//   - test.go: Common setup and helpers (this file)
//   - auth_test.go: Auth command tests
//   - email_test.go: Email command tests
//   - folders_test.go: Folder command tests
//   - threads_test.go: Thread command tests
//   - drafts_test.go: Draft command tests
//   - calendar_test.go: Calendar command tests
//   - contacts_test.go: Contact command tests
//   - webhooks_test.go: Webhook command tests
//   - misc_test.go: Help, error handling, workflow tests
package integration

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/nylas/cli/internal/adapters/nylas"
	"github.com/nylas/cli/internal/domain"
	"golang.org/x/time/rate"
	"gopkg.in/yaml.v3"
)

// Test configuration loaded from environment
var (
	testAPIKey   string
	testGrantID  string
	testClientID string
	testEmail    string
	testBinary   string
)

// Rate limiter for API calls to prevent hitting Nylas rate limits
// Default: 2 requests per second with burst of 5
// This works across all parallel tests
var (
	apiRateLimiter *rate.Limiter
	rateLimitRPS   = 2.0 // Requests per second
	rateLimitBurst = 5   // Burst capacity
)

func init() {
	testAPIKey = os.Getenv("NYLAS_API_KEY")
	testGrantID = os.Getenv("NYLAS_GRANT_ID")
	testClientID = os.Getenv("NYLAS_CLIENT_ID")
	testEmail = os.Getenv("NYLAS_TEST_EMAIL")

	// Configure rate limiter from environment
	if rpsStr := os.Getenv("NYLAS_TEST_RATE_LIMIT_RPS"); rpsStr != "" {
		if rps, err := strconv.ParseFloat(rpsStr, 64); err == nil && rps > 0 {
			rateLimitRPS = rps
		}
	}
	if burstStr := os.Getenv("NYLAS_TEST_RATE_LIMIT_BURST"); burstStr != "" {
		if burst, err := strconv.Atoi(burstStr); err == nil && burst > 0 {
			rateLimitBurst = burst
		}
	}

	// Initialize rate limiter
	apiRateLimiter = rate.NewLimiter(rate.Limit(rateLimitRPS), rateLimitBurst)

	// Find the binary - try environment variable first, then common locations
	testBinary = os.Getenv("NYLAS_TEST_BINARY")
	if testBinary != "" {
		// If provided, try to make it absolute
		if !strings.HasPrefix(testBinary, "/") {
			if abs, err := exec.LookPath(testBinary); err == nil {
				testBinary = abs
			}
		}
		return
	}

	// Try to find binary relative to test directory
	candidates := []string{
		"../../bin/nylas",    // From internal/cli
		"../../../bin/nylas", // From internal/cli/subdir
		"./bin/nylas",        // From project root
		"bin/nylas",          // From project root
	}
	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			testBinary = c
			break
		}
	}
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

// runCLI executes a CLI command and returns stdout, stderr, and error.
// NOTE: This does NOT apply rate limiting. For tests that make API calls,
// either call acquireRateLimit(t) before this, or use runCLIWithRateLimit.
func runCLI(args ...string) (string, string, error) {
	return runCLIWithTimeout(2*time.Minute, args...)
}

// runCLIWithTimeout executes a CLI command with a specified timeout.
// Use this for commands that might take a long time (e.g., AI/LLM calls).
func runCLIWithTimeout(timeout time.Duration, args ...string) (string, string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, testBinary, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Build environment with all necessary variables
	env := []string{
		"NYLAS_API_KEY=" + testAPIKey,
		"NYLAS_GRANT_ID=" + testGrantID,
		"NYLAS_DISABLE_KEYRING=true", // Disable keyring during tests to avoid macOS prompts
	}

	// Pass through AI provider credentials if set
	if apiKey := os.Getenv("ANTHROPIC_API_KEY"); apiKey != "" {
		env = append(env, "ANTHROPIC_API_KEY="+apiKey)
	}
	if apiKey := os.Getenv("OPENAI_API_KEY"); apiKey != "" {
		env = append(env, "OPENAI_API_KEY="+apiKey)
	}
	if apiKey := os.Getenv("GROQ_API_KEY"); apiKey != "" {
		env = append(env, "GROQ_API_KEY="+apiKey)
	}
	if apiKey := os.Getenv("OPENROUTER_API_KEY"); apiKey != "" {
		env = append(env, "OPENROUTER_API_KEY="+apiKey)
	}
	if ollamaHost := os.Getenv("OLLAMA_HOST"); ollamaHost != "" {
		env = append(env, "OLLAMA_HOST="+ollamaHost)
	}

	// Set environment for the CLI
	cmd.Env = append(os.Environ(), env...)

	err := cmd.Run()
	return stdout.String(), stderr.String(), err
}

// runCLIWithRateLimit executes a CLI command with rate limiting.
// Use this for commands that make API calls when running tests with t.Parallel().
// For offline commands (version, help), use runCLI directly.
func runCLIWithRateLimit(t *testing.T, args ...string) (string, string, error) {
	t.Helper()
	acquireRateLimit(t)
	return runCLI(args...)
}

// runCLIWithInput executes a CLI command with stdin input.
// NOTE: This does NOT apply rate limiting. For tests that make API calls,
// either call acquireRateLimit(t) before this, or use runCLIWithInputAndRateLimit.
func runCLIWithInput(input string, args ...string) (string, string, error) {
	cmd := exec.Command(testBinary, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	cmd.Stdin = strings.NewReader(input)

	// Build environment with all necessary variables
	env := []string{
		"NYLAS_API_KEY=" + testAPIKey,
		"NYLAS_GRANT_ID=" + testGrantID,
		"NYLAS_DISABLE_KEYRING=true", // Disable keyring during tests to avoid macOS prompts
	}

	// Pass through AI provider credentials if set
	if apiKey := os.Getenv("ANTHROPIC_API_KEY"); apiKey != "" {
		env = append(env, "ANTHROPIC_API_KEY="+apiKey)
	}
	if apiKey := os.Getenv("OPENAI_API_KEY"); apiKey != "" {
		env = append(env, "OPENAI_API_KEY="+apiKey)
	}
	if apiKey := os.Getenv("GROQ_API_KEY"); apiKey != "" {
		env = append(env, "GROQ_API_KEY="+apiKey)
	}
	if apiKey := os.Getenv("OPENROUTER_API_KEY"); apiKey != "" {
		env = append(env, "OPENROUTER_API_KEY="+apiKey)
	}
	if ollamaHost := os.Getenv("OLLAMA_HOST"); ollamaHost != "" {
		env = append(env, "OLLAMA_HOST="+ollamaHost)
	}

	cmd.Env = append(os.Environ(), env...)

	err := cmd.Run()
	return stdout.String(), stderr.String(), err
}

// runCLIWithInputAndRateLimit executes a CLI command with stdin input and rate limiting.
// Use this for commands that make API calls when running tests with t.Parallel().
func runCLIWithInputAndRateLimit(t *testing.T, input string, args ...string) (string, string, error) {
	t.Helper()
	acquireRateLimit(t)
	return runCLIWithInput(input, args...)
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

// getTestEmail returns the test email from environment or default
func getTestEmail() string {
	if testEmail != "" {
		return testEmail
	}
	return getEnvOrDefault("NYLAS_TEST_EMAIL", "")
}

// extractEventID extracts event ID from CLI output
func extractEventID(output string) string {
	// Look for event ID patterns in output
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		// Look for "ID: <id>" or "Event ID: <id>"
		if strings.Contains(line, "Event ID:") || strings.Contains(line, "ID:") {
			parts := strings.Fields(line)
			for i, part := range parts {
				if (part == "ID:" || part == "Event" && i+1 < len(parts) && parts[i+1] == "ID:") && i+1 < len(parts) {
					// Next field should be the ID
					nextIdx := i + 1
					if part == "Event" {
						nextIdx = i + 2
					}
					if nextIdx < len(parts) {
						return parts[nextIdx]
					}
				}
			}
		}
		// Also try to match event_* or cal_event_* patterns
		if strings.Contains(line, "event_") || strings.Contains(line, "cal_event_") {
			parts := strings.Fields(line)
			for _, part := range parts {
				if strings.HasPrefix(part, "event_") || strings.HasPrefix(part, "cal_event_") {
					// Clean up any trailing punctuation
					id := strings.TrimRight(part, ".,;:\"'")
					return id
				}
			}
		}
	}
	return ""
}

// extractEventIDFromList extracts event ID from list output by finding title
func extractEventIDFromList(output, title string) string {
	lines := strings.Split(output, "\n")
	for i, line := range lines {
		if strings.Contains(line, title) {
			// Look for ID in the same line or nearby lines
			if strings.Contains(line, "ID:") {
				parts := strings.Split(line, "ID:")
				if len(parts) > 1 {
					idPart := strings.TrimSpace(parts[1])
					fields := strings.Fields(idPart)
					if len(fields) > 0 {
						return fields[0]
					}
				}
			}
			// Check previous lines for ID
			for j := i - 1; j >= 0 && j >= i-3; j-- {
				if strings.Contains(lines[j], "ID:") {
					parts := strings.Split(lines[j], "ID:")
					if len(parts) > 1 {
						idPart := strings.TrimSpace(parts[1])
						fields := strings.Fields(idPart)
						if len(fields) > 0 {
							return fields[0]
						}
					}
				}
			}
		}
	}
	return ""
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

// Ensure imports are used (these are used in other test files)
var (
	_ = context.Background
	_ = domain.ProviderGoogle
)
