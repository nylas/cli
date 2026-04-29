package cli

import (
	"fmt"
	"io"

	"github.com/fatih/color"

	"github.com/nylas/cli/internal/cli/common"
)

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
