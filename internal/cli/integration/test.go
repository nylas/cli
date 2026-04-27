//go:build integration
// +build integration

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
//   - NYLAS_TEST_USE_LOCAL_AUTH: Set to "true" to opt into loading API key/default grant from local CLI auth
//   - NYLAS_TEST_RATE_LIMIT_RPS: API rate limit (requests/sec, default: 2.0)
//   - NYLAS_TEST_RATE_LIMIT_BURST: API rate limit burst capacity (default: 5)
//   - NYLAS_FILE_STORE_PASSPHRASE: Passphrase for the encrypted file secret-store fallback
//
// Parallel Testing:
//
//	Tests can use t.Parallel() to run concurrently. The package includes a global
//	rate limiter that ensures API calls don't exceed Nylas rate limits.
//
//	Usage:
//	  func TestExample(t *testing.T) {
//	      skipIfMissingCreds(t)
//	      t.Parallel()  // Enable parallel execution
//
//	      // For API calls, use rate-limited functions:
//	      stdout, stderr, err := runCLIWithRateLimit(t, "calendar", "events", "list")
//	      // OR manually acquire rate limit:
//	      acquireRateLimit(t)
//	      stdout, stderr, err := runCLI("calendar", "events", "list")
//	  }
//
//	For offline commands (timezone, ai config, help), rate limiting is not needed:
//	  stdout, _, _ := runCLI("timezone", "list")  // No rate limit needed
//
// Test files are organized by feature:
//   - test.go: Common setup and helpers (this file)
//   - test_runners.go: CLI execution helpers
//   - test_setup.go: Environment, credentials, and API helpers
//   - test_extractors.go: Output parsing helpers
//   - auth_test.go: Auth command tests
//   - email_test.go: Email command tests
//   - folders_test.go: Folder command tests
//   - threads_test.go: Thread command tests
//   - drafts_test.go: Draft command tests
//   - calendar_test.go: Calendar command tests
//   - contacts_test.go: Contact command tests
//   - webhooks_test.go: Webhook command tests
//   - timezone_test.go: Timezone utility tests (offline, no API)
//   - ai_config_test.go: AI config tests (offline, no API)
//   - ai_features_test.go: AI features tests
//   - misc_test.go: Help, error handling, workflow tests
package integration

import (
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/domain"
	"golang.org/x/time/rate"
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

	if useLocalAuthForIntegration() {
		if testAPIKey == "" {
			if apiKey, err := common.GetAPIKey(); err == nil {
				testAPIKey = apiKey
			}
		}
		if testGrantID == "" {
			if grantID, err := common.GetGrantID(nil); err == nil {
				testGrantID = grantID
			}
		}
	}

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
			if abs, err := filepath.Abs(c); err == nil {
				testBinary = abs
			} else {
				testBinary = c
			}
			break
		}
	}
}

// Ensure imports are used (these are used in other test files)
var (
	_ = domain.ProviderGoogle
)
