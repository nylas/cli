package cli

import (
	"fmt"
	"runtime"

	"github.com/spf13/cobra"

	"github.com/nylas/cli/internal/adapters/config"
	"github.com/nylas/cli/internal/cli/common"
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

func init() {
	rootCmd.AddCommand(newDoctorCmd())
}
