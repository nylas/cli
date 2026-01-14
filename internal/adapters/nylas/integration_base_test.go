//go:build integration
// +build integration

package nylas_test

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/nylas/cli/internal/adapters/nylas"
	"github.com/nylas/cli/internal/domain"
)

type testConfig struct {
	apiKey   string
	grantID  string
	clientID string
}

// getTestConfig loads test configuration from environment variables.
// This ensures credentials are never stored in code.
func getTestConfig(t *testing.T) testConfig {
	t.Helper()

	apiKey := os.Getenv("NYLAS_API_KEY")
	grantID := os.Getenv("NYLAS_GRANT_ID")
	clientID := os.Getenv("NYLAS_CLIENT_ID")

	if apiKey == "" {
		t.Skip("NYLAS_API_KEY not set, skipping integration test")
	}
	if grantID == "" {
		t.Skip("NYLAS_GRANT_ID not set, skipping integration test")
	}

	return testConfig{
		apiKey:   apiKey,
		grantID:  grantID,
		clientID: clientID,
	}
}

// getTestClient creates a configured Nylas client for integration tests.
// It also adds a rate limit delay to avoid hitting API rate limits.
func getTestClient(t *testing.T) (*nylas.HTTPClient, string) {
	t.Helper()

	// Add delay to avoid rate limiting between tests
	waitForRateLimit()

	cfg := getTestConfig(t)
	client := nylas.NewHTTPClient()
	client.SetCredentials(cfg.clientID, "", cfg.apiKey)

	return client, cfg.grantID
}

// createTestContext creates a context with standard test timeout.
func createTestContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 30*time.Second)
}

// createLongTestContext creates a context with extended timeout for slower operations.
func createLongTestContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 60*time.Second)
}

// rateLimitDelay adds a delay between API calls to avoid rate limiting.
// Nylas API has rate limits, so we add a pause between tests.
// Increased to 2 seconds to provide better protection against rate limiting.
const rateLimitDelay = 2 * time.Second

// waitForRateLimit pauses execution to avoid hitting API rate limits.
func waitForRateLimit() {
	time.Sleep(rateLimitDelay)
}

// safeSubstring safely extracts a substring, avoiding panics on short strings.
func safeSubstring(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen]
}

// skipIfProviderNotSupported checks if the error indicates the provider doesn't support
// the operation and skips the test if so.
func skipIfProviderNotSupported(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		return
	}
	errMsg := err.Error()
	// Various error messages that indicate provider limitation
	if strings.Contains(errMsg, "Method not supported for provider") ||
		strings.Contains(errMsg, "an internal error ocurred") || // Nylas API typo
		strings.Contains(errMsg, "an internal error occurred") {
		t.Skipf("Provider does not support this operation: %v", err)
	}
}

// skipIfNoMessages skips the test if no messages are available to test with.
func skipIfNoMessages(t *testing.T, messages []domain.Message) {
	t.Helper()
	if len(messages) == 0 {
		t.Skip("No messages available in inbox, skipping test")
	}
}

// skipIfTrialAccountLimitation skips the test if error indicates trial account limitation.
func skipIfTrialAccountLimitation(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		return
	}
	errMsg := err.Error()
	if strings.Contains(errMsg, "trial account") ||
		strings.Contains(errMsg, "Please upgrade your account") {
		t.Skipf("Feature not available on trial account: %v", err)
	}
}

// skipIfRateLimited skips the test if error indicates rate limiting.
func skipIfRateLimited(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		return
	}
	errMsg := err.Error()
	if strings.Contains(errMsg, "Too many requests") ||
		strings.Contains(errMsg, "rate limit") {
		t.Skipf("Rate limited by API: %v", err)
	}
}

// =============================================================================
// Grant Tests
// =============================================================================
