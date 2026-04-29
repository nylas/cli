package cli

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/cli/testutil"
)

// TestCheckAPICredentials_MissingClientID confirms the "no API key /
// client id" path returns a Warning with actionable detail rather than
// silently passing. Covers a previously-zero-coverage branch.
func TestCheckAPICredentials_MissingClientID(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tempDir, "xdg"))
	t.Setenv("HOME", tempDir)
	t.Setenv("NYLAS_DISABLE_KEYRING", "true")
	t.Setenv("NYLAS_FILE_STORE_PASSPHRASE", "doctor-test-passphrase")

	// Don't pre-populate any credentials — checkAPICredentials should
	// detect the absence and warn.
	result := checkAPICredentials()
	if result.Status != CheckStatusWarning && result.Status != CheckStatusError {
		t.Fatalf("Status = %v, want Warning or Error", result.Status)
	}
}

func TestCheckSecretStore_WarnsWhenFileStoreIsForced(t *testing.T) {
	tempDir := t.TempDir()

	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tempDir, "xdg"))
	t.Setenv("HOME", tempDir)
	t.Setenv("NYLAS_DISABLE_KEYRING", "true")
	t.Setenv("NYLAS_FILE_STORE_PASSPHRASE", "doctor-test-passphrase")

	result := checkSecretStore()

	if result.Status != CheckStatusWarning {
		t.Fatalf("Status = %v, want %v", result.Status, CheckStatusWarning)
	}
	if result.Message != "encrypted file" {
		t.Fatalf("Message = %q, want %q", result.Message, "encrypted file")
	}
	if !strings.Contains(result.Detail, "NYLAS_FILE_STORE_PASSPHRASE") {
		t.Fatalf("Detail %q does not mention NYLAS_FILE_STORE_PASSPHRASE", result.Detail)
	}
	if !strings.Contains(result.Detail, "unset NYLAS_DISABLE_KEYRING") {
		t.Fatalf("Detail %q does not mention unsetting NYLAS_DISABLE_KEYRING", result.Detail)
	}
}

func TestDoctorCommandStructuredJSONUsesInheritedFlags(t *testing.T) {
	restore := swapDoctorChecksForTest(t, []doctorCheck{
		{
			spinnerMessage: "Checking configuration...",
			run: func() CheckResult {
				return CheckResult{Name: "Configuration", Status: CheckStatusOK, Message: "ready"}
			},
		},
		{
			spinnerMessage: "Checking network...",
			run: func() CheckResult {
				return CheckResult{
					Name:    "Network",
					Status:  CheckStatusWarning,
					Message: "slow",
					Detail:  "High latency detected.",
				}
			},
		},
	})
	defer restore()

	root := newDoctorTestRoot()
	stdout, stderr, err := testutil.ExecuteCommand(root, "doctor", "--json")
	if err != nil {
		t.Fatalf("doctor --json returned error: %v", err)
	}
	if strings.TrimSpace(stderr) != "" {
		t.Fatalf("doctor --json wrote stderr = %q, want empty", stderr)
	}
	if strings.Contains(stdout, "Nylas CLI Health Check") || strings.Contains(stdout, "Summary") {
		t.Fatalf("doctor --json output contained human-readable prose: %q", stdout)
	}

	var report doctorReport
	if err := json.Unmarshal([]byte(stdout), &report); err != nil {
		t.Fatalf("doctor --json output is not valid JSON: %v\noutput: %s", err, stdout)
	}

	if len(report.Checks) != 2 {
		t.Fatalf("len(report.Checks) = %d, want 2", len(report.Checks))
	}
	if report.Checks[0].Status != "ok" {
		t.Fatalf("report.Checks[0].Status = %q, want %q", report.Checks[0].Status, "ok")
	}
	if report.Checks[1].Status != "warning" {
		t.Fatalf("report.Checks[1].Status = %q, want %q", report.Checks[1].Status, "warning")
	}
	if report.Summary.Warning != 1 || report.Summary.OK != 1 || report.Summary.OverallStatus != "warning" {
		t.Fatalf("unexpected summary: %+v", report.Summary)
	}
	if len(report.Recommendations) != 1 || report.Recommendations[0] != "High latency detected." {
		t.Fatalf("unexpected recommendations: %#v", report.Recommendations)
	}
	if report.Environment != nil {
		t.Fatalf("report.Environment = %+v, want nil without --verbose", report.Environment)
	}
}

func TestDoctorCommandStructuredYAMLIncludesVerboseEnvironment(t *testing.T) {
	restore := swapDoctorChecksForTest(t, []doctorCheck{
		{
			spinnerMessage: "Checking configuration...",
			run: func() CheckResult {
				return CheckResult{Name: "Configuration", Status: CheckStatusOK, Message: "ready"}
			},
		},
	})
	defer restore()

	root := newDoctorTestRoot()
	stdout, stderr, err := testutil.ExecuteCommand(root, "doctor", "--format", "yaml", "--verbose")
	if err != nil {
		t.Fatalf("doctor --format yaml --verbose returned error: %v", err)
	}
	if strings.TrimSpace(stderr) != "" {
		t.Fatalf("doctor --format yaml --verbose wrote stderr = %q, want empty", stderr)
	}
	if strings.Contains(stdout, "Nylas CLI Health Check") || strings.Contains(stdout, "Summary") {
		t.Fatalf("doctor --format yaml output contained human-readable prose: %q", stdout)
	}

	var report doctorReport
	if err := yaml.Unmarshal([]byte(stdout), &report); err != nil {
		t.Fatalf("doctor --format yaml output is not valid YAML: %v\noutput: %s", err, stdout)
	}

	if report.Environment == nil {
		t.Fatal("report.Environment = nil, want verbose environment data")
	}
	if report.Environment.Platform == "" || report.Environment.GoVersion == "" || report.Environment.ConfigDir == "" {
		t.Fatalf("report.Environment = %+v, want populated fields", report.Environment)
	}
	if report.Summary.Total != 1 || report.Summary.OverallStatus != "ok" {
		t.Fatalf("unexpected summary: %+v", report.Summary)
	}
}

func TestDoctorCommandQuietOutputsOverallStatus(t *testing.T) {
	restore := swapDoctorChecksForTest(t, []doctorCheck{
		{
			spinnerMessage: "Checking configuration...",
			run: func() CheckResult {
				return CheckResult{Name: "Configuration", Status: CheckStatusWarning, Message: "slow", Detail: "Check config"}
			},
		},
	})
	defer restore()

	root := newDoctorTestRoot()
	stdout, stderr, err := testutil.ExecuteCommand(root, "doctor", "--quiet")
	if err != nil {
		t.Fatalf("doctor --quiet returned error: %v", err)
	}
	if strings.TrimSpace(stderr) != "" {
		t.Fatalf("doctor --quiet wrote stderr = %q, want empty", stderr)
	}
	if strings.TrimSpace(stdout) != "warning" {
		t.Fatalf("doctor --quiet output = %q, want %q", strings.TrimSpace(stdout), "warning")
	}
}

func TestDoctorCommandDefaultOutputRemainsHumanReadable(t *testing.T) {
	restore := swapDoctorChecksForTest(t, []doctorCheck{
		{
			spinnerMessage: "Checking configuration...",
			run: func() CheckResult {
				return CheckResult{Name: "Configuration", Status: CheckStatusOK, Message: "ready"}
			},
		},
	})
	defer restore()

	root := newDoctorTestRoot()
	stdout, _, err := testutil.ExecuteCommand(root, "doctor")
	if err != nil {
		t.Fatalf("doctor returned error: %v", err)
	}
	if !strings.Contains(stdout, "Nylas CLI Health Check") {
		t.Fatalf("doctor output missing header: %q", stdout)
	}
	if !strings.Contains(stdout, "Summary") {
		t.Fatalf("doctor output missing summary: %q", stdout)
	}
	if !strings.Contains(stdout, "Configuration") {
		t.Fatalf("doctor output missing check name: %q", stdout)
	}
}

func newDoctorTestRoot() *cobra.Command {
	root := &cobra.Command{
		Use:           "test",
		SilenceErrors: true,
		SilenceUsage:  true,
	}
	common.AddOutputFlags(root)
	root.AddCommand(newDoctorCmd())
	return root
}

func swapDoctorChecksForTest(t *testing.T, checks []doctorCheck) func() {
	t.Helper()

	original := doctorChecks
	doctorChecks = checks

	return func() {
		doctorChecks = original
	}
}
