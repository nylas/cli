package deliverability

import (
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/nylas/cli/internal/cli/common"
	"github.com/spf13/cobra"
)

const metricsDateLayout = "2006-01-02"

var (
	eventIDPattern = regexp.MustCompile(`^[0-9a-f]{64}$`)
	// thresholdPattern is a plain non-negative JSON number; the API rejects exponents and strings.
	thresholdPattern = regexp.MustCompile(`^(0|[1-9][0-9]*)(\.[0-9]+)?$`)
)

// parseActivityTime turns a --start/--end value into Unix seconds. Times without a zone are UTC,
// a bare date is a whole UTC day (its end for --end), and a duration such as 24h or 7d means that long ago.
func parseActivityTime(flag, value string, now time.Time, isEnd bool) (int64, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, nil
	}
	invalid := common.NewUserError(
		fmt.Sprintf("invalid --%s value %q", flag, value),
		"Use a Unix timestamp, YYYY-MM-DD (UTC), RFC3339, or a look-back such as 24h or 7d",
	)

	if day, err := time.ParseInLocation(metricsDateLayout, value, time.UTC); err == nil {
		if isEnd {
			return day.Add(24*time.Hour - time.Second).Unix(), nil
		}
		return day.Unix(), nil
	}
	if d, err := common.ParseDuration(value); err == nil {
		if d <= 0 {
			return 0, invalid
		}
		return now.Add(-d).Unix(), nil
	}
	t, err := common.ParseHumanTime(value, common.ParseHumanTimeOpts{Now: now.UTC(), Loc: time.UTC})
	if err != nil {
		return 0, invalid
	}
	return t.Unix(), nil
}

// validateMetricsDate checks a --start/--end metrics value is YYYY-MM-DD.
func validateMetricsDate(flag, value string) error {
	if value == "" {
		return nil
	}
	if _, err := time.Parse(metricsDateLayout, value); err != nil {
		return common.NewUserError(fmt.Sprintf("invalid --%s value %q", flag, value), "Use a UTC date in YYYY-MM-DD form")
	}
	return nil
}

// validateEventID rejects IDs the API cannot route (they get a non-JSON 404).
func validateEventID(id string) error {
	if !eventIDPattern.MatchString(id) {
		return common.NewUserError(fmt.Sprintf("invalid event ID %q", id), "Event IDs are 64 lowercase hex characters, as shown by the list command")
	}
	return nil
}

func formatUnix(ts int64) string {
	if ts == 0 {
		return ""
	}
	return time.Unix(ts, 0).Local().Format("2006-01-02 15:04:05")
}

// formatRate prints a percentage; nil means no data (a zero denominator), never 0%.
func formatRate(v *float64) string {
	if v == nil {
		return "—"
	}
	return strconv.FormatFloat(*v, 'f', 2, 64) + "%"
}

func formatDelta(v *float64) string {
	if v == nil {
		return "—"
	}
	return fmt.Sprintf("%+.2f pp", *v)
}

func formatCount(v int64) string { return strconv.FormatInt(v, 10) }

func formatThreshold(v float64) string { return strconv.FormatFloat(v, 'f', -1, 64) + "%" }

func newTable(w io.Writer, headers ...string) *common.Table {
	return common.NewTable(headers...).SetWriter(w)
}

func nextCursorNote(cmd *cobra.Command, nextCursor string) {
	if nextCursor == "" {
		return
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "\nNext page token: %s\n", nextCursor)
}

// writeStructuredList writes a list response as JSON/YAML, or only the row IDs in quiet mode.
func writeStructuredList(cmd *cobra.Command, resp, rows any) (bool, error) {
	quiet, _ := cmd.Flags().GetBool("quiet")
	format, _ := cmd.Flags().GetString("format")
	if quiet || format == "quiet" {
		return true, common.GetOutputWriter(cmd).WriteList(rows, nil)
	}
	return writeStructured(cmd, resp)
}

// writeStructured writes v as JSON/YAML/quiet and reports whether it did.
func writeStructured(cmd *cobra.Command, v any) (bool, error) {
	if !common.IsStructuredOutput(cmd) {
		return false, nil
	}
	return true, common.GetOutputWriter(cmd).Write(v)
}
