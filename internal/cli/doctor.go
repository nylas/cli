package cli

import (
	"fmt"
	"net/http"
	"os"
	"runtime"
	"time"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/nylas/cli/internal/adapters/config"
	"github.com/nylas/cli/internal/adapters/keyring"
	"github.com/nylas/cli/internal/adapters/nylas"
	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/domain"
	"github.com/nylas/cli/internal/ports"
)

// CheckResult represents the result of a health check.
type CheckResult struct {
	Name    string
	Status  CheckStatus
	Message string
	Detail  string
}

// CheckStatus represents the status of a check.
type CheckStatus int

const (
	CheckStatusOK CheckStatus = iota
	CheckStatusWarning
	CheckStatusError
	CheckStatusSkipped
)

func newDoctorCmd() *cobra.Command {
	var verbose bool

	cmd := &cobra.Command{
		Use:   "doctor",
		Short: "Check CLI health and configuration",
		Long: `Run diagnostic checks on your Nylas CLI configuration.

This command verifies:
  - API credentials are configured and valid
  - Grant(s) are accessible and not expired
  - Secret store is working properly
  - Network connectivity to Nylas API
  - Configuration file is valid

Examples:
  nylas doctor
  nylas doctor --verbose`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDoctor(verbose)
		},
	}

	cmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Show detailed information")

	return cmd
}

func runDoctor(verbose bool) error {
	_, _ = common.Bold.Println("Nylas CLI Health Check")
	fmt.Println()

	results := []CheckResult{}

	// Show environment info
	if verbose {
		_, _ = common.Dim.Printf("  Platform: %s/%s\n", runtime.GOOS, runtime.GOARCH)
		_, _ = common.Dim.Printf("  Go Version: %s\n", runtime.Version())
		_, _ = common.Dim.Printf("  Config Dir: %s\n", config.DefaultConfigDir())
		fmt.Println()
	}

	// 1. Check configuration file
	configResult, _ := common.RunWithSpinnerResult("Checking configuration...", func() (CheckResult, error) {
		return checkConfig(), nil
	})
	results = append(results, configResult)
	printCheckResult(configResult, verbose)

	// 2. Check secret store
	secretResult, _ := common.RunWithSpinnerResult("Checking secret store...", func() (CheckResult, error) {
		return checkSecretStore(), nil
	})
	results = append(results, secretResult)
	printCheckResult(secretResult, verbose)

	// 3. Check API credentials
	apiKeyResult, _ := common.RunWithSpinnerResult("Checking API credentials...", func() (CheckResult, error) {
		return checkAPICredentials(), nil
	})
	results = append(results, apiKeyResult)
	printCheckResult(apiKeyResult, verbose)

	// 4. Check network connectivity
	networkResult, _ := common.RunWithSpinnerResult("Checking network connectivity...", func() (CheckResult, error) {
		return checkNetworkConnectivity(), nil
	})
	results = append(results, networkResult)
	printCheckResult(networkResult, verbose)

	// 5. Check grants
	grantsResult, _ := common.RunWithSpinnerResult("Checking grants...", func() (CheckResult, error) {
		return checkGrants(), nil
	})
	results = append(results, grantsResult)
	printCheckResult(grantsResult, verbose)

	// Summary
	fmt.Println()
	_, _ = common.Bold.Println("Summary")
	fmt.Println()

	okCount := 0
	warnCount := 0
	errCount := 0

	for _, r := range results {
		switch r.Status {
		case CheckStatusOK:
			okCount++
		case CheckStatusWarning:
			warnCount++
		case CheckStatusError:
			errCount++
		}
	}

	if errCount > 0 {
		_, _ = common.Red.Printf("  %d error(s), %d warning(s), %d passed\n", errCount, warnCount, okCount)
		fmt.Println()
		_, _ = common.Cyan.Println("  Recommendations:")
		for _, r := range results {
			if r.Status == CheckStatusError && r.Detail != "" {
				fmt.Printf("    - %s\n", r.Detail)
			}
		}
	} else if warnCount > 0 {
		_, _ = common.Yellow.Printf("  %d warning(s), %d passed\n", warnCount, okCount)
		fmt.Println()
		_, _ = common.Cyan.Println("  Recommendations:")
		for _, r := range results {
			if r.Status == CheckStatusWarning && r.Detail != "" {
				fmt.Printf("    - %s\n", r.Detail)
			}
		}
	} else {
		_, _ = common.Green.Printf("  All %d checks passed!\n", okCount)
	}

	fmt.Println()

	if errCount > 0 {
		return fmt.Errorf("%d health check(s) failed", errCount)
	}

	return nil
}

func printCheckResult(r CheckResult, verbose bool) {
	var icon string
	var colorFn *color.Color

	switch r.Status {
	case CheckStatusOK:
		icon = "✓"
		colorFn = common.Green
	case CheckStatusWarning:
		icon = "⚠"
		colorFn = common.Yellow
	case CheckStatusError:
		icon = "✗"
		colorFn = common.Red
	case CheckStatusSkipped:
		icon = "○"
		colorFn = common.Dim
	}

	_, _ = colorFn.Printf("  %s %s", icon, r.Name)
	if r.Message != "" {
		_, _ = common.Dim.Printf(" - %s", r.Message)
	}
	fmt.Println()

	if verbose && r.Detail != "" {
		_, _ = common.Dim.Printf("    %s\n", r.Detail)
	}
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

	// First check if system keyring is available
	kr := keyring.NewSystemKeyring()
	keyringAvailable := kr.IsAvailable()

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
			Detail:  "NYLAS_DISABLE_KEYRING is set. Unset to use system keyring.",
		}
	}

	if storeName == "encrypted file" && !keyringAvailable {
		return CheckResult{
			Name:    "Secret Store",
			Status:  CheckStatusWarning,
			Message: storeName,
			Detail:  "System keyring unavailable. Using encrypted file fallback.",
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
			Detail:  "Check your internet connection and firewall settings",
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

	grantStore := keyring.NewGrantStore(secretStore)
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

	// Validate default grant is still valid on Nylas
	configStore := config.NewDefaultFileStore()
	cfg, _ := configStore.Load()

	apiKey, _ := secretStore.Get(ports.KeyAPIKey)
	clientID, _ := secretStore.Get(ports.KeyClientID)
	clientSecret, _ := secretStore.Get(ports.KeyClientSecret)

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

func init() {
	rootCmd.AddCommand(newDoctorCmd())
}
