package cli

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/nylas/cli/internal/adapters/config"
	"github.com/nylas/cli/internal/adapters/keyring"
	"github.com/nylas/cli/internal/adapters/nylas"
	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/domain"
	"github.com/nylas/cli/internal/ports"
)

func runDoctorChecks(useSpinner bool, onResult func(CheckResult)) []CheckResult {
	results := make([]CheckResult, 0, len(doctorChecks))

	for _, check := range doctorChecks {
		result := runDoctorCheck(check, useSpinner)
		results = append(results, result)

		if onResult != nil {
			onResult(result)
		}
	}

	return results
}

func runDoctorCheck(check doctorCheck, useSpinner bool) CheckResult {
	if !useSpinner {
		return check.run()
	}

	result, _ := common.RunWithSpinnerResult(check.spinnerMessage, func() (CheckResult, error) {
		return check.run(), nil
	})

	return result
}

func checkConfig() CheckResult {
	configStore := config.NewDefaultFileStore()
	cfg, err := configStore.Load()

	if err != nil {
		if os.IsNotExist(err) {
			return CheckResult{
				Name:    "Configuration",
				Status:  CheckStatusWarning,
				Message: "No config file found (using defaults)",
				Detail:  "Run 'nylas auth config' to create a configuration",
			}
		}
		return CheckResult{
			Name:    "Configuration",
			Status:  CheckStatusError,
			Message: "Failed to load config",
			Detail:  err.Error(),
		}
	}

	return CheckResult{
		Name:    "Configuration",
		Status:  CheckStatusOK,
		Message: fmt.Sprintf("Region: %s", cfg.Region),
	}
}

func checkSecretStore() CheckResult {
	// Check if keyring is disabled via environment
	keyringDisabled := os.Getenv("NYLAS_DISABLE_KEYRING") == "true"

	keyringAvailable := false
	if !keyringDisabled {
		kr := keyring.NewSystemKeyring()
		keyringAvailable = kr.IsAvailable()
	}

	secretStore, err := keyring.NewSecretStore(config.DefaultConfigDir())
	if err != nil {
		return CheckResult{
			Name:    "Secret Store",
			Status:  CheckStatusError,
			Message: "Failed to initialize",
			Detail:  err.Error(),
		}
	}

	if !secretStore.IsAvailable() {
		return CheckResult{
			Name:    "Secret Store",
			Status:  CheckStatusError,
			Message: "Not available",
			Detail:  "System keyring is not accessible. Check your desktop environment settings.",
		}
	}

	// Warn if using encrypted file when keyring should be available
	storeName := secretStore.Name()

	if storeName == "encrypted file" && keyringDisabled {
		return CheckResult{
			Name:    "Secret Store",
			Status:  CheckStatusWarning,
			Message: storeName,
			Detail:  "NYLAS_DISABLE_KEYRING is set. Set NYLAS_FILE_STORE_PASSPHRASE for the fallback store, or unset NYLAS_DISABLE_KEYRING to use the system keyring.",
		}
	}

	if storeName == "encrypted file" && !keyringAvailable {
		return CheckResult{
			Name:    "Secret Store",
			Status:  CheckStatusWarning,
			Message: storeName,
			Detail:  "System keyring unavailable. The encrypted file fallback requires NYLAS_FILE_STORE_PASSPHRASE.",
		}
	}

	if storeName == "encrypted file" && keyringAvailable {
		return CheckResult{
			Name:    "Secret Store",
			Status:  CheckStatusWarning,
			Message: storeName,
			Detail:  "Credentials in encrypted file. Run 'nylas auth migrate' to use system keyring.",
		}
	}

	return CheckResult{
		Name:    "Secret Store",
		Status:  CheckStatusOK,
		Message: storeName,
	}
}

func checkAPICredentials() CheckResult {
	secretStore, err := keyring.NewSecretStore(config.DefaultConfigDir())
	if err != nil {
		return CheckResult{
			Name:   "API Credentials",
			Status: CheckStatusSkipped,
			Detail: "Secret store not available",
		}
	}

	apiKey, err := secretStore.Get(ports.KeyAPIKey)
	if err != nil {
		return CheckResult{
			Name:    "API Credentials",
			Status:  CheckStatusError,
			Message: "API key not configured",
			Detail:  "Run 'nylas auth config' to set up your API key",
		}
	}

	if apiKey == "" {
		return CheckResult{
			Name:    "API Credentials",
			Status:  CheckStatusError,
			Message: "API key is empty",
			Detail:  "Run 'nylas auth config' to set a valid API key",
		}
	}

	// Check if API key format looks valid
	if len(apiKey) < 20 {
		return CheckResult{
			Name:    "API Credentials",
			Status:  CheckStatusWarning,
			Message: "API key format may be invalid",
			Detail:  "API key seems too short. Verify with 'nylas auth config'",
		}
	}

	return CheckResult{
		Name:    "API Credentials",
		Status:  CheckStatusOK,
		Message: "Configured",
	}
}

func checkNetworkConnectivity() CheckResult {
	ctx, cancel := common.CreateContextWithTimeout(domain.TimeoutHealthCheck)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", "https://api.us.nylas.com/v3/", nil)
	if err != nil {
		return CheckResult{
			Name:    "Network",
			Status:  CheckStatusError,
			Message: "Failed to create request",
			Detail:  err.Error(),
		}
	}

	client := &http.Client{Timeout: domain.TimeoutHealthCheck}
	start := time.Now()
	resp, err := client.Do(req)
	latency := time.Since(start)

	if err != nil {
		return CheckResult{
			Name:    "Network",
			Status:  CheckStatusError,
			Message: "Cannot reach Nylas API",
			// Surface the underlying transport error so the user can
			// distinguish DNS failure / TLS handshake / proxy / cert
			// expiry from a generic "internet down" prompt.
			Detail: fmt.Sprintf("Check your internet connection and firewall settings: %v", err),
		}
	}
	defer func() { _ = resp.Body.Close() }()

	// API should return 401 without auth, which means it's reachable
	if resp.StatusCode == 401 || resp.StatusCode == 200 || resp.StatusCode == 404 {
		msg := fmt.Sprintf("Connected (latency: %dms)", latency.Milliseconds())
		if latency > 2*time.Second {
			return CheckResult{
				Name:    "Network",
				Status:  CheckStatusWarning,
				Message: msg,
				Detail:  "High latency detected. API calls may be slow.",
			}
		}
		return CheckResult{
			Name:    "Network",
			Status:  CheckStatusOK,
			Message: msg,
		}
	}

	return CheckResult{
		Name:    "Network",
		Status:  CheckStatusWarning,
		Message: fmt.Sprintf("Unexpected status: %d", resp.StatusCode),
	}
}

func checkGrants() CheckResult {
	secretStore, err := keyring.NewSecretStore(config.DefaultConfigDir())
	if err != nil {
		return CheckResult{
			Name:   "Grants",
			Status: CheckStatusSkipped,
			Detail: "Secret store not available",
		}
	}

	grantStore, err := common.NewDefaultGrantStore()
	if err != nil {
		return CheckResult{
			Name:    "Grants",
			Status:  CheckStatusError,
			Message: "Failed to open grant store",
			Detail:  err.Error(),
		}
	}
	grants, err := grantStore.ListGrants()
	if err != nil {
		return CheckResult{
			Name:    "Grants",
			Status:  CheckStatusError,
			Message: "Failed to list grants",
			Detail:  err.Error(),
		}
	}

	if len(grants) == 0 {
		return CheckResult{
			Name:    "Grants",
			Status:  CheckStatusWarning,
			Message: "No grants configured",
			Detail:  "Run 'nylas auth login' to authenticate with your email provider",
		}
	}

	// Check default grant
	defaultGrant, err := grantStore.GetDefaultGrant()
	if err != nil {
		return CheckResult{
			Name:    "Grants",
			Status:  CheckStatusWarning,
			Message: fmt.Sprintf("%d grant(s), no default set", len(grants)),
			Detail:  "Run 'nylas auth switch <grant-id>' to set a default",
		}
	}

	// Validate default grant is still valid on Nylas. Surface config/secret
	// failures explicitly — a misleading 401 from Nylas is exactly what
	// `doctor` exists to prevent.
	configStore := config.NewDefaultFileStore()
	cfg, err := configStore.Load()
	if err != nil {
		return CheckResult{
			Name:    "Grants",
			Status:  CheckStatusError,
			Message: fmt.Sprintf("%d grant(s), failed to load config", len(grants)),
			Detail:  fmt.Sprintf("Run 'nylas auth config' to repair: %v", err),
		}
	}

	apiKey, apiErr := secretStore.Get(ports.KeyAPIKey)
	clientID, cidErr := secretStore.Get(ports.KeyClientID)
	if apiErr != nil || cidErr != nil {
		return CheckResult{
			Name:    "Grants",
			Status:  CheckStatusError,
			Message: fmt.Sprintf("%d grant(s), credentials missing", len(grants)),
			Detail:  "API key or client ID is missing from the keyring. Run 'nylas auth config'.",
		}
	}

	// client_secret is optional in this codebase (see auth/config.go,
	// auth/show.go, auth/helpers.go) — only some flows need it. Treat
	// "not found" as empty; surface only real keyring failures.
	clientSecret, csErr := secretStore.Get(ports.KeyClientSecret)
	if csErr != nil && !errors.Is(csErr, domain.ErrSecretNotFound) {
		return CheckResult{
			Name:    "Grants",
			Status:  CheckStatusError,
			Message: fmt.Sprintf("%d grant(s), keyring read failed", len(grants)),
			Detail:  fmt.Sprintf("Reading client_secret from the keyring failed: %v", csErr),
		}
	}

	client := nylas.NewHTTPClient()
	client.SetRegion(cfg.Region)
	client.SetCredentials(clientID, clientSecret, apiKey)

	ctx, cancel := common.CreateContextWithTimeout(domain.TimeoutHealthCheck)
	defer cancel()

	grant, err := client.GetGrant(ctx, defaultGrant)
	if err != nil {
		return CheckResult{
			Name:    "Grants",
			Status:  CheckStatusWarning,
			Message: fmt.Sprintf("%d grant(s), default may be invalid", len(grants)),
			Detail:  "Run 'nylas auth list' to check grant status",
		}
	}

	if !grant.IsValid() {
		return CheckResult{
			Name:    "Grants",
			Status:  CheckStatusWarning,
			Message: fmt.Sprintf("%d grant(s), default status: %s", len(grants), grant.GrantStatus),
			Detail:  "Your default grant may need re-authentication",
		}
	}

	return CheckResult{
		Name:    "Grants",
		Status:  CheckStatusOK,
		Message: fmt.Sprintf("%d grant(s), default: %s", len(grants), grant.Email),
	}
}
