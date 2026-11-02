package deliverability

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/fatih/color"
	"github.com/nylas/cli/internal/cli/testutil"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type capturedRequest struct {
	method string
	path   string
	query  map[string][]string
	body   string
}

// runAgainstFakeAPI runs the real command tree against a fake Nylas API that answers every request with respBody.
func runAgainstFakeAPI(t *testing.T, status int, respBody string, args ...string) (string, []capturedRequest, error) {
	t.Helper()
	stdout, _, requests, err := runAgainstFakeAPIWithStderr(t, status, respBody, args...)
	return stdout, requests, err
}

// runAgainstFakeAPIWithStderr is runAgainstFakeAPI that also returns stderr.
func runAgainstFakeAPIWithStderr(t *testing.T, status int, respBody string, args ...string) (string, string, []capturedRequest, error) {
	t.Helper()
	var requests []capturedRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		requests = append(requests, capturedRequest{method: r.Method, path: r.URL.Path, query: r.URL.Query(), body: string(body)})
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = io.WriteString(w, respBody)
	}))
	t.Cleanup(server.Close)

	// Isolate from the developer's real config/keyring; env API key wins.
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("NYLAS_API_KEY", "test-api-key")
	t.Setenv("NYLAS_API_BASE_URL", server.URL)

	// common.PrintSuccess writes through fatih/color, whose Output is bound to the real stdout at init.
	var colorOut bytes.Buffer
	oldColorOut := color.Output
	color.Output = &colorOut
	t.Cleanup(func() { color.Output = oldColorOut })

	stdout, stderr, err := testutil.ExecuteSubCommand(NewDeliverabilityCmd(), args...)
	return stdout + colorOut.String(), stderr, requests, err
}

const sendEventsPage = `{"request_id":"req-1","next_cursor":"tok-2","data":[
	{"id":"aaa","event_type":"bounced","bounce_class":"hard","timestamp":1730000000,"recipient":"bob@example.net","from":"agent@mail.example.com","reason":"General (5.1.1)"},
	{"id":"bbb","event_type":"delivered","timestamp":1730000100,"recipient":"alice@example.org","from":"agent@mail.example.com","reason":"250 OK"}]}`

func TestSendEventsList_FlagsBecomeQuery(t *testing.T) {
	before := time.Now().Add(-24 * time.Hour).Unix()
	out, reqs, err := runAgainstFakeAPI(t, http.StatusOK, sendEventsPage,
		"send-events", "list", "--event-type", "bounced", "--event-type", "complaint", "--start", "24h", "--recipient-domain", "example.net")
	require.NoError(t, err)
	require.Len(t, reqs, 1)
	q := reqs[0].query
	assert.Equal(t, "/v3/activity/send", reqs[0].path)
	// Repeated --event-type must reach the API as repeated keys; it rejects "bounced,complaint".
	assert.Equal(t, []string{"bounced", "complaint"}, q["event_type"])
	assert.Equal(t, []string{"example.net"}, q["recipient_domain"])
	// "24h" is a look-back: start is 24 hours ago, not 24 hours from now.
	start, convErr := strconv.ParseInt(q["start"][0], 10, 64)
	require.NoError(t, convErr)
	assert.InDelta(t, before, start, 5)

	assert.Contains(t, out, "bounced (hard)")
	assert.Contains(t, out, "Next page: --page-token tok-2")
}

func TestSendEventsList_JSONKeepsEnvelope(t *testing.T) {
	out, _, err := runAgainstFakeAPI(t, http.StatusOK, sendEventsPage, "send-events", "list", "--json")
	require.NoError(t, err)
	// Scripts page with next_cursor, so JSON output must keep the envelope, not just the rows.
	var resp struct {
		Data       []map[string]any `json:"data"`
		NextCursor string           `json:"next_cursor"`
	}
	require.NoError(t, json.Unmarshal([]byte(out), &resp))
	assert.Len(t, resp.Data, 2)
	assert.Equal(t, "tok-2", resp.NextCursor)
}

func TestSendEventsList_QuietPrintsIDs(t *testing.T) {
	out, _, err := runAgainstFakeAPI(t, http.StatusOK, sendEventsPage, "send-events", "list", "--quiet")
	require.NoError(t, err)
	assert.Equal(t, "aaa\nbbb\n", out)
}

func TestSendEventsList_RejectsBadStartWithoutCallingAPI(t *testing.T) {
	_, reqs, err := runAgainstFakeAPI(t, http.StatusOK, sendEventsPage, "send-events", "list", "--start", "last tuesday-ish")
	require.Error(t, err)
	assert.Empty(t, reqs)
}

func TestShow_RejectsMalformedEventID(t *testing.T) {
	// The API returns a non-JSON 404 for IDs that are not 64 hex characters, so catch them first.
	for _, group := range []string{"send-events", "tracking-events"} {
		_, reqs, err := runAgainstFakeAPI(t, http.StatusOK, `{}`, group, "show", "not-an-id")
		require.Error(t, err, group)
		assert.Contains(t, err.Error(), "invalid event ID")
		assert.Empty(t, reqs, group)
	}
}

func TestSendSummary_NullRatesAreNotZero(t *testing.T) {
	resp := `{"data":{"health":{"state":"critical","window":{"start":"2026-10-27","end":"2026-11-02"},"min_assess_volume":100,
		"findings":[{"rule":"concentration","severity":"high","title":"Hard bounces concentrate on example.net","recommendation":"Clean your list"}]},
		"totals":{"sent":400,"delivered":0,"bounced_hard":50},
		"metrics":{"delivery_rate":{"value":null,"delta_pp":null,"prior_value":null,"status":null},
		"bounce_rate":{"value":12.5,"delta_pp":4,"prior_value":8.5,"status":"crit"},
		"complaint_rate":{"value":null,"delta_pp":null,"prior_value":null,"status":null}},"prior_period":null}}`
	out, reqs, err := runAgainstFakeAPI(t, http.StatusOK, resp, "send-events", "summary", "--start", "2026-10-01", "--end", "2026-10-31")
	require.NoError(t, err)
	require.Len(t, reqs, 1)
	assert.Equal(t, "/v3/metrics/send/summary", reqs[0].path)
	assert.Equal(t, []string{"2026-10-01"}, reqs[0].query["start"])

	// A null rate means a zero denominator; printing 0% would claim a perfect complaint rate.
	assert.NotContains(t, out, "0.00%")
	assert.Contains(t, out, "—")
	assert.Contains(t, out, "12.50%")
	assert.Contains(t, out, "+4.00 pp")
	assert.Contains(t, out, "Hard bounces concentrate on example.net")
}

func TestMetrics_RejectsNonDateWindowWithoutCallingAPI(t *testing.T) {
	// Metrics take UTC dates; an epoch here is a common mix-up with the activity feed.
	_, reqs, err := runAgainstFakeAPI(t, http.StatusOK, `{}`, "tracking-events", "summary", "--start", "1730000000")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid --start")
	assert.Empty(t, reqs)
}

func TestSendBreakdown_NullWindowExplainsNoData(t *testing.T) {
	out, reqs, err := runAgainstFakeAPI(t, http.StatusOK, `{"data":{"group_by":"recipient_domain","window":null,"rows":[]}}`,
		"send-events", "breakdown", "--group-by", "recipient_domain", "--start", "2025-01-01", "--end", "2025-01-31")
	require.NoError(t, err)
	assert.Equal(t, []string{"recipient_domain"}, reqs[0].query["group_by"])
	assert.Equal(t, []string{"25"}, reqs[0].query["limit"])
	assert.Contains(t, out, "only cover the last 90 days")
}

func TestTrackingMessage_ShowsPerLinkClicks(t *testing.T) {
	resp := `{"data":{"message_id":"m1","from":"agent@mail.example.com","tracking":{"opens":true,"links":true},
		"opens":{"total":2,"first_at":1730000000,"last_at":1730000500},
		"clicks":{"total":3,"first_at":1730000100,"last_at":1730000400,"links":[{"link_index":0,"url":"https://example.com/a","clicks":3}]},
		"events":[]}}`
	out, reqs, err := runAgainstFakeAPI(t, http.StatusOK, resp, "tracking-events", "message", "m1")
	require.NoError(t, err)
	assert.Equal(t, "/v3/activity/tracking/messages/m1", reqs[0].path)
	assert.Contains(t, out, "https://example.com/a")
	assert.Contains(t, out, "opens on, links on")
}

const ruleResp = `{"data":{"id":"3d6f8a1c-2b4e-4c7d-9f0a-1b2c3d4e5f60","metric":"bounce_rate","direction":"above","threshold":0.25,"window_hours":1,"min_volume":0}}`

func TestSendAlertsCreate_SendsExactBody(t *testing.T) {
	out, reqs, err := runAgainstFakeAPI(t, http.StatusCreated, ruleResp,
		"send-alerts", "create", "--metric", "bounce_rate", "--threshold", "0.25", "--window-hours", "1")
	require.NoError(t, err)
	require.Len(t, reqs, 1)
	assert.Equal(t, http.MethodPost, reqs[0].method)
	// threshold must be a JSON number with the user's decimals; min_volume was not set so it is omitted.
	assert.JSONEq(t, `{"metric":"bounce_rate","threshold":0.25,"window_hours":1}`, reqs[0].body)
	assert.Contains(t, out, "bounce_rate above 0.25%")
}

func TestSendAlertsCreate_RequiresCoreFlags(t *testing.T) {
	_, reqs, err := runAgainstFakeAPI(t, http.StatusCreated, ruleResp, "send-alerts", "create", "--metric", "bounce_rate")
	require.Error(t, err)
	assert.Empty(t, reqs)
}

func TestSendAlertsCreate_RejectsNonNumericThreshold(t *testing.T) {
	_, reqs, err := runAgainstFakeAPI(t, http.StatusCreated, ruleResp,
		"send-alerts", "create", "--metric", "bounce_rate", "--threshold", "five", "--window-hours", "1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid --threshold")
	assert.Empty(t, reqs)
}

func TestSendAlertsUpdate_SendsOnlyChangedFields(t *testing.T) {
	_, reqs, err := runAgainstFakeAPI(t, http.StatusOK, ruleResp, "send-alerts", "update", "rule-1", "--min-volume", "0")
	require.NoError(t, err)
	require.Len(t, reqs, 1)
	assert.Equal(t, http.MethodPut, reqs[0].method)
	assert.Equal(t, "/v3/alerts/send/rule-1", reqs[0].path)
	// PUT is partial: sending unset flags as zero values would overwrite the rule. An explicit 0 must still be sent.
	assert.JSONEq(t, `{"min_volume":0}`, reqs[0].body)
}

func TestSendAlertsUpdate_NoFlagsDoesNotCallAPI(t *testing.T) {
	_, reqs, err := runAgainstFakeAPI(t, http.StatusOK, ruleResp, "send-alerts", "update", "rule-1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "nothing to update")
	assert.Empty(t, reqs)
}

func TestSendAlertsDelete_RequiresYes(t *testing.T) {
	_, reqs, err := runAgainstFakeAPI(t, http.StatusOK, `{"request_id":"r"}`, "send-alerts", "delete", "rule-1")
	require.Error(t, err)
	assert.Empty(t, reqs)

	_, reqs, err = runAgainstFakeAPI(t, http.StatusOK, `{"request_id":"r"}`, "send-alerts", "delete", "rule-1", "--yes")
	require.NoError(t, err)
	require.Len(t, reqs, 1)
	assert.Equal(t, http.MethodDelete, reqs[0].method)
}

func TestAPIErrorMessageReachesUser(t *testing.T) {
	_, _, err := runAgainstFakeAPI(t, http.StatusForbidden,
		`{"request_id":"r","error":{"type":"api.forbidden","message":"this endpoint requires an application API key"}}`,
		"send-alerts", "list")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "application API key")
}

func TestCommandTree(t *testing.T) {
	cmd := NewDeliverabilityCmd()
	want := map[string][]string{
		"send-events":     {"list", "show", "message", "summary", "timeseries", "breakdown"},
		"tracking-events": {"list", "show", "message", "summary", "timeseries", "breakdown"},
		"send-alerts":     {"list", "show", "create", "update", "delete"},
	}
	for group, subs := range want {
		g, _, err := cmd.Find([]string{group})
		require.NoError(t, err, group)
		var got []string
		for _, c := range g.Commands() {
			got = append(got, c.Name())
		}
		assert.ElementsMatch(t, subs, got, group)
	}
}

func TestParseActivityTime(t *testing.T) {
	now := time.Date(2026, 11, 2, 15, 30, 0, 0, time.UTC)
	tests := []struct {
		name    string
		value   string
		isEnd   bool
		want    time.Time
		wantErr bool
	}{
		{name: "empty means server default", value: ""},
		// Dates are UTC days and --end is inclusive, so a one-day window covers the whole day.
		{name: "date start is UTC midnight", value: "2026-10-01", want: time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)},
		{name: "date end is last second of the UTC day", value: "2026-10-01", isEnd: true, want: time.Date(2026, 10, 1, 23, 59, 59, 0, time.UTC)},
		// Without a zone the result must not depend on the laptop's time zone.
		{name: "zoneless date-time is UTC", value: "2026-10-01 07:00", want: time.Date(2026, 10, 1, 7, 0, 0, 0, time.UTC)},
		{name: "RFC3339 keeps its zone", value: "2026-10-01T07:00:00-04:00", want: time.Date(2026, 10, 1, 11, 0, 0, 0, time.UTC)},
		{name: "zoneless ISO with seconds is UTC", value: "2026-10-01T07:00:00", want: time.Date(2026, 10, 1, 7, 0, 0, 0, time.UTC)},
		{name: "zoneless ISO without seconds is UTC", value: "2026-10-01T07:00", want: time.Date(2026, 10, 1, 7, 0, 0, 0, time.UTC)},
		{name: "date-time with seconds is UTC", value: "2026-10-01 07:00:00", want: time.Date(2026, 10, 1, 7, 0, 0, 0, time.UTC)},
		{name: "printed form with UTC suffix", value: "2026-10-01 07:00:00 UTC", want: time.Date(2026, 10, 1, 7, 0, 0, 0, time.UTC)},
		// Keyword and time-of-day forms resolve differently from run to run (and 9am rolls to tomorrow),
		// which breaks --page-token, so only look-backs may move and they are flagged as relative.
		{name: "today rejected", value: "today", wantErr: true},
		{name: "tomorrow rejected", value: "tomorrow", wantErr: true},
		{name: "time of day rejected", value: "9am", wantErr: true},
		{name: "month-day time rejected", value: "Jan 2 3pm", wantErr: true},
		{name: "look-back counts back from now", value: "7d", want: now.Add(-7 * 24 * time.Hour)},
		{name: "unix seconds", value: "1730000000", want: time.Unix(1730000000, 0)},
		// A non-positive look-back would put start at or after now and silently return nothing.
		{name: "negative look-back rejected", value: "-7d", wantErr: true},
		{name: "zero look-back rejected", value: "0", wantErr: true},
		{name: "garbage rejected", value: "soon", wantErr: true},
		// The API treats 0 and below as "not set", so these would silently become the default window.
		{name: "pre-1970 date rejected", value: "1969-12-31", wantErr: true},
		{name: "epoch zero rejected", value: "1970-01-01", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, _, err := parseActivityTime("start", tt.value, now, tt.isEnd)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			if tt.want.IsZero() {
				assert.Zero(t, got)
				return
			}
			assert.Equal(t, tt.want.Unix(), got)
		})
	}
}

func TestSendAlertsCreate_RejectsThresholdsThatAreNotJSONNumbers(t *testing.T) {
	// Rejected locally so the user gets a clear error, not a marshal failure or a server 400:
	// .5, +5, 5. and NaN are not JSON numbers; the API refuses exponents, negatives, more than
	// 2 decimals and more than 9 integer digits.
	for _, threshold := range []string{".5", "+5", "5.", "NaN", "1e2", "-1", "0.125", "1234567890"} {
		_, reqs, err := runAgainstFakeAPI(t, http.StatusCreated, ruleResp,
			"send-alerts", "create", "--metric", "bounce_rate", "--threshold", threshold, "--window-hours", "1")
		require.Error(t, err, threshold)
		assert.Contains(t, err.Error(), "invalid --threshold", threshold)
		assert.Empty(t, reqs, threshold)
	}
}

func TestSendAlertsDelete_JSONOutputHasNoText(t *testing.T) {
	out, reqs, err := runAgainstFakeAPI(t, http.StatusOK, `{"request_id":"r"}`, "send-alerts", "delete", "rule-1", "--yes", "--json")
	require.NoError(t, err)
	require.Len(t, reqs, 1)
	// Scripts pipe --json output to a parser, so no human text may reach stdout.
	assert.NotContains(t, out, "Deleted")
}

func TestSendAlertsCreate_AcceptsThresholdsTheAPIAccepts(t *testing.T) {
	for _, threshold := range []string{"0", "5", "99.99"} {
		_, reqs, err := runAgainstFakeAPI(t, http.StatusCreated, ruleResp,
			"send-alerts", "create", "--metric", "bounce_rate", "--threshold", threshold, "--window-hours", "1")
		require.NoError(t, err, threshold)
		require.Len(t, reqs, 1, threshold)
	}
}

func TestSendEventsList_LookBackWithPageTokenFailsLocally(t *testing.T) {
	// The page token is bound to the exact start/end, and a look-back resolves differently on every run.
	_, reqs, err := runAgainstFakeAPI(t, http.StatusOK, sendEventsPage, "send-events", "list", "--start", "7d", "--page-token", "tok-1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--page-token")
	assert.Empty(t, reqs)
}

func TestSendEventsList_LookBackPrintsReusableNextPage(t *testing.T) {
	out, reqs, err := runAgainstFakeAPI(t, http.StatusOK, sendEventsPage, "send-events", "list", "--start", "7d")
	require.NoError(t, err)
	require.Len(t, reqs, 1)
	// The next-page line must repeat the epoch actually sent, so pasting it reproduces the same query.
	assert.Contains(t, out, "Next page: --page-token tok-2 --start "+reqs[0].query["start"][0])
}

func TestSendEventsList_EndOnlyAnchorsDefaultStart(t *testing.T) {
	// Without this, an --end older than 6 days is before the server's default start and returns 400.
	endDay := time.Now().UTC().AddDate(0, 0, -30).Truncate(24 * time.Hour)
	_, reqs, err := runAgainstFakeAPI(t, http.StatusOK, sendEventsPage, "send-events", "list", "--end", endDay.Format("2006-01-02"))
	require.NoError(t, err)
	require.Len(t, reqs, 1)
	assert.Equal(t, []string{strconv.FormatInt(endDay.AddDate(0, 0, -6).Unix(), 10)}, reqs[0].query["start"])
	assert.Equal(t, []string{strconv.FormatInt(endDay.Add(24*time.Hour-time.Second).Unix(), 10)}, reqs[0].query["end"])
}

func TestSendEventsList_EndOlderThanRetentionFailsLocally(t *testing.T) {
	// The server clamps start to 90 days ago, so an older --end would return an empty list with no error.
	old := time.Now().UTC().AddDate(0, 0, -100).Format("2006-01-02")
	_, reqs, err := runAgainstFakeAPI(t, http.StatusOK, sendEventsPage, "send-events", "list", "--end", old)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "90 days")
	assert.Empty(t, reqs)
}

func TestSendEventsList_JSONLookBackPrintsNextPageOnStderr(t *testing.T) {
	stdout, stderr, reqs, err := runAgainstFakeAPIWithStderr(t, http.StatusOK, sendEventsPage, "send-events", "list", "--start", "7d", "--json")
	require.NoError(t, err)
	require.Len(t, reqs, 1)
	// Stdout must stay parseable; scripts need the resolved start to page, since 7d is refused with --page-token.
	var resp map[string]any
	require.NoError(t, json.Unmarshal([]byte(stdout), &resp))
	assert.Contains(t, stderr, "Next page: --page-token tok-2 --start "+reqs[0].query["start"][0])
}

func TestFormatUnix_RoundTripsAsInput(t *testing.T) {
	// Users copy times from the tables into --start/--end, so every printed time must parse back to itself.
	ts := time.Date(2026, 10, 29, 16, 13, 55, 0, time.UTC).Unix()
	printed := formatUnix(ts)
	assert.Equal(t, "2026-10-29 16:13:55 UTC", printed)
	got, relative, err := parseActivityTime("start", printed, time.Now(), false)
	require.NoError(t, err)
	assert.False(t, relative)
	assert.Equal(t, ts, got)
}

func TestSendEventsList_QuietPrintsOnlyIDs(t *testing.T) {
	// Pipelines such as `list -q 2>&1 | xargs ... show` merge stderr into stdout, so quiet must print
	// only IDs, even when there is a next page or a look-back; scripts page with --json instead.
	for _, start := range []string{"2026-10-29", "7d"} {
		stdout, stderr, _, err := runAgainstFakeAPIWithStderr(t, http.StatusOK, sendEventsPage,
			"send-events", "list", "--start", start, "--quiet")
		require.NoError(t, err, start)
		assert.Equal(t, "aaa\nbbb\n", stdout, start)
		assert.Empty(t, stderr, start)
	}
}

func TestSendEventsList_EndJustInsideRetentionIsSent(t *testing.T) {
	// Guards the edge of the 90-day check: one hour inside the server's floor must still be allowed.
	end := time.Now().UTC().Add(-activityRetention + time.Hour).Format("2006-01-02 15:04")
	_, reqs, err := runAgainstFakeAPI(t, http.StatusOK, sendEventsPage, "send-events", "list", "--end", end)
	require.NoError(t, err)
	require.Len(t, reqs, 1)
}

// allCommands returns root and every descendant command.
func allCommands(cmd *cobra.Command) []*cobra.Command {
	cmds := []*cobra.Command{cmd}
	for _, c := range cmd.Commands() {
		cmds = append(cmds, allCommands(c)...)
	}
	return cmds
}

func TestHelpStatesScope(t *testing.T) {
	// Users must see that sends from connected grants (Google, Microsoft) are not in this data;
	// a new subcommand without the note should fail here.
	for _, c := range allCommands(NewDeliverabilityCmd()) {
		assert.Contains(t, c.Long, "Agent Account", c.CommandPath())
		assert.Contains(t, c.Long, "Transactional send", c.CommandPath())
	}
}

func TestIDCommandsHaveExamples(t *testing.T) {
	// A command that takes an ID must show what the ID looks like.
	for _, c := range allCommands(NewDeliverabilityCmd()) {
		if strings.Contains(c.Use, "<") {
			assert.NotEmpty(t, c.Example, c.CommandPath())
		}
	}
}

func TestAlertFlagHelp(t *testing.T) {
	root := NewDeliverabilityCmd()
	find := func(args ...string) *cobra.Command {
		c, _, err := root.Find(args)
		require.NoError(t, err)
		return c
	}
	create, update, del := find("send-alerts", "create"), find("send-alerts", "update"), find("send-alerts", "delete")
	// create rejects a request without these, so the help must say so up front.
	for _, name := range []string{"metric", "threshold", "window-hours"} {
		assert.Contains(t, create.Flags().Lookup(name).Usage, "(required)", name)
	}
	// update is a partial PUT, so no flag has a default that would overwrite the rule.
	for _, name := range []string{"metric", "threshold", "window-hours", "min-volume"} {
		assert.Contains(t, update.Flags().Lookup(name).Usage, "unchanged if not set", name)
		assert.NotContains(t, update.Flags().Lookup(name).Usage, "default", name)
	}
	assert.Contains(t, del.Flags().Lookup("yes").Usage, "required")
}
