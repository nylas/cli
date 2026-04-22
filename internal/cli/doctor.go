package cli

import (
	"fmt"
	"io"
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

type doctorCheck struct {
	spinnerMessage string
	run            func() CheckResult
}

type doctorEnvironment struct {
	Platform  string `json:"platform,omitempty" yaml:"platform,omitempty"`
	GoVersion string `json:"go_version,omitempty" yaml:"go_version,omitempty"`
	ConfigDir string `json:"config_dir,omitempty" yaml:"config_dir,omitempty"`
}

type doctorCheckOutput struct {
	Name    string `json:"name" yaml:"name"`
	Status  string `json:"status" yaml:"status"`
	Message string `json:"message,omitempty" yaml:"message,omitempty"`
	Detail  string `json:"detail,omitempty" yaml:"detail,omitempty"`
}

type doctorSummary struct {
	OK            int    `json:"ok" yaml:"ok"`
	Warning       int    `json:"warning" yaml:"warning"`
	Error         int    `json:"error" yaml:"error"`
	Skipped       int    `json:"skipped" yaml:"skipped"`
	Total         int    `json:"total" yaml:"total"`
	OverallStatus string `json:"overall_status" yaml:"overall_status"`
}

type doctorReport struct {
	Environment     *doctorEnvironment  `json:"environment,omitempty" yaml:"environment,omitempty"`
	Checks          []doctorCheckOutput `json:"checks" yaml:"checks"`
	Summary         doctorSummary       `json:"summary" yaml:"summary"`
	Recommendations []string            `json:"recommendations,omitempty" yaml:"recommendations,omitempty"`
}

func (r doctorReport) QuietField() string {
	return r.Summary.OverallStatus
}

// CheckStatus represents the status of a check.
type CheckStatus int

const (
	CheckStatusOK CheckStatus = iota
	CheckStatusWarning
	CheckStatusError
	CheckStatusSkipped
)

var doctorChecks = []doctorCheck{
	{
		spinnerMessage: "Checking configuration...",
		run:            checkConfig,
	},
	{
		spinnerMessage: "Checking secret store...",
		run:            checkSecretStore,
	},
	{
		spinnerMessage: "Checking API credentials...",
		run:            checkAPICredentials,
	},
	{
		spinnerMessage: "Checking network connectivity...",
		run:            checkNetworkConnectivity,
	},
	{
		spinnerMessage: "Checking grants...",
		run:            checkGrants,
	},
}

func (s CheckStatus) String() string {
	switch s {
	case CheckStatusOK:
		return "ok"
	case CheckStatusWarning:
		return "warning"
	case CheckStatusError:
		return "error"
	case CheckStatusSkipped:
		return "skipped"
	default:
		return "unknown"
	}
}

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
			return runDoctor(cmd, verbose)
		},
	}

	cmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Show detailed information")

	return cmd
}

func runDoctor(cmd *cobra.Command, verbose bool) error {
	if isDoctorStructuredOutput(cmd) {
		results := runDoctorChecks(false, nil)
		if err := common.GetOutputWriter(cmd).Write(buildDoctorReport(results, verbose)); err != nil {
			return err
		}
		return doctorResultsError(results)
	}

	w := cmd.OutOrStdout()
	_, _ = common.Bold.Fprintln(w, "Nylas CLI Health Check")
	_, _ = fmt.Fprintln(w)

	// Show environment info
	if verbose {
		env := currentDoctorEnvironment()
		_, _ = common.Dim.Fprintf(w, "  Platform: %s\n", env.Platform)
		_, _ = common.Dim.Fprintf(w, "  Go Version: %s\n", env.GoVersion)
		_, _ = common.Dim.Fprintf(w, "  Config Dir: %s\n", env.ConfigDir)
		_, _ = fmt.Fprintln(w)
	}

	results := runDoctorChecks(true, func(result CheckResult) {
		printCheckResult(w, result, verbose)
	})

	// Summary
	_, _ = fmt.Fprintln(w)
	_, _ = common.Bold.Fprintln(w, "Summary")
	_, _ = fmt.Fprintln(w)

	summary := summarizeDoctorResults(results)
	recommendations := doctorRecommendations(results)

	if summary.Error > 0 {
		_, _ = common.Red.Fprintf(w, "  %d error(s), %d warning(s), %d passed\n", summary.Error, summary.Warning, summary.OK)
		_, _ = fmt.Fprintln(w)
		_, _ = common.Cyan.Fprintln(w, "  Recommendations:")
		for _, recommendation := range recommendations {
			_, _ = fmt.Fprintf(w, "    - %s\n", recommendation)
		}
	} else if summary.Warning > 0 {
		_, _ = common.Yellow.Fprintf(w, "  %d warning(s), %d passed\n", summary.Warning, summary.OK)
		_, _ = fmt.Fprintln(w)
		_, _ = common.Cyan.Fprintln(w, "  Recommendations:")
		for _, recommendation := range recommendations {
			_, _ = fmt.Fprintf(w, "    - %s\n", recommendation)
		}
	} else {
		_, _ = common.Green.Fprintf(w, "  All %d checks passed!\n", summary.OK)
	}

	_, _ = fmt.Fprintln(w)

	return doctorResultsError(results)
}

func isDoctorStructuredOutput(cmd *cobra.Command) bool {
	return common.IsStructuredOutput(cmd)
}

func currentDoctorEnvironment() doctorEnvironment {
	return doctorEnvironment{
		Platform:  fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH),
		GoVersion: runtime.Version(),
		ConfigDir: config.DefaultConfigDir(),
	}
}

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

func buildDoctorReport(results []CheckResult, verbose bool) doctorReport {
	checks := make([]doctorCheckOutput, 0, len(results))
	for _, result := range results {
		checks = append(checks, doctorCheckOutput{
			Name:    result.Name,
			Status:  result.Status.String(),
			Message: result.Message,
			Detail:  result.Detail,
		})
	}

	report := doctorReport{
		Checks:          checks,
		Summary:         summarizeDoctorResults(results),
		Recommendations: doctorRecommendations(results),
	}

	if verbose {
		env := currentDoctorEnvironment()
		report.Environment = &env
	}

	return report
}

func summarizeDoctorResults(results []CheckResult) doctorSummary {
	summary := doctorSummary{
		Total: len(results),
	}

	for _, result := range results {
		switch result.Status {
		case CheckStatusOK:
			summary.OK++
		case CheckStatusWarning:
			summary.Warning++
		case CheckStatusError:
			summary.Error++
		case CheckStatusSkipped:
			summary.Skipped++
		}
	}

	switch {
	case summary.Error > 0:
		summary.OverallStatus = CheckStatusError.String()
	case summary.Warning > 0:
		summary.OverallStatus = CheckStatusWarning.String()
	case summary.OK > 0:
		summary.OverallStatus = CheckStatusOK.String()
	default:
		summary.OverallStatus = CheckStatusSkipped.String()
	}

	return summary
}

func doctorRecommendations(results []CheckResult) []string {
	targetStatus := CheckStatusWarning
	if summarizeDoctorResults(results).Error > 0 {
		targetStatus = CheckStatusError
	}

	recommendations := make([]string, 0)
	for _, result := range results {
		if result.Status == targetStatus && result.Detail != "" {
			recommendations = append(recommendations, result.Detail)
		}
	}

	return recommendations
}

func doctorResultsError(results []CheckResult) error {
	summary := summarizeDoctorResults(results)
	if summary.Error > 0 {
		return fmt.Errorf("%d health check(s) failed", summary.Error)
	}
	return nil
}

func printCheckResult(w io.Writer, r CheckResult, verbose bool) {
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

	_, _ = colorFn.Fprintf(w, "  %s %s", icon, r.Name)
	if r.Message != "" {
		_, _ = common.Dim.Fprintf(w, " - %s", r.Message)
	}
	_, _ = fmt.Fprintln(w)

	if verbose && r.Detail != "" {
		_, _ = common.Dim.Fprintf(w, "    %s\n", r.Detail)
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
