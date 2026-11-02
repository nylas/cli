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
	unixPattern    = regexp.MustCompile(`^[0-9]{10,12}$`)
	// thresholdPattern matches the API's threshold literal: no sign or exponent, up to 9 integer digits and 2 decimals.
	thresholdPattern = regexp.MustCompile(`^(0|[1-9][0-9]{0,8})(\.[0-9]{1,2})?$`)
)

// activityTimeLayouts are the absolute --start/--end forms; each means the same instant on every run.
var activityTimeLayouts = []string{time.RFC3339, "2006-01-02 15:04"}

// parseActivityTime turns a --start/--end value into Unix seconds. It accepts Unix seconds, a UTC
// date (its last second for --end), "YYYY-MM-DD HH:MM" in UTC, RFC3339, or a look-back such as 24h
// or 7d. relative is true for look-backs, the only form that moves between runs.
func parseActivityTime(flag, value string, now time.Time, isEnd bool) (ts int64, relative bool, err error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, false, nil
	}
	invalid := common.NewUserError(
		fmt.Sprintf("invalid --%s value %q", flag, value),
		`Use Unix seconds, YYYY-MM-DD, "YYYY-MM-DD HH:MM" (UTC), RFC3339, or a look-back such as 24h or 7d`,
	)

	if day, dateErr := time.ParseInLocation(metricsDateLayout, value, time.UTC); dateErr == nil {
		ts = day.Unix()
		if isEnd {
			ts = day.Add(24*time.Hour - time.Second).Unix()
		}
	} else if unixPattern.MatchString(value) {
		ts, _ = strconv.ParseInt(value, 10, 64)
	} else if d, durErr := common.ParseDuration(value); durErr == nil {
		if d <= 0 {
			return 0, false, invalid
		}
		ts, relative = now.Add(-d).Unix(), true
	} else {
		ts = parseAbsoluteTime(value)
	}
	// The API reads 0 and below as "not set", so such a time would silently fall back to the default window.
	if ts <= 0 {
		return 0, false, invalid
	}
	return ts, relative, nil
}

// parseAbsoluteTime returns 0 when value matches no activity layout.
func parseAbsoluteTime(value string) int64 {
	for _, layout := range activityTimeLayouts {
		if t, err := time.ParseInLocation(layout, value, time.UTC); err == nil {
			return t.Unix()
		}
	}
	return 0
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

// nextCursorNote prints the next page token; pageFlags, when set, are window flags the next page must repeat.
func nextCursorNote(w io.Writer, nextCursor, pageFlags string) {
	if nextCursor == "" {
		return
	}
	if pageFlags == "" {
		_, _ = fmt.Fprintf(w, "\nNext page token: %s\n", nextCursor)
		return
	}
	_, _ = fmt.Fprintf(w, "\nNext page: --page-token %s %s\n", nextCursor, pageFlags)
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
