package deliverability

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/nylas/cli/internal/cli/testutil"
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

	stdout, _, err := testutil.ExecuteSubCommand(NewDeliverabilityCmd(), args...)
	return stdout, requests, err
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
	assert.Contains(t, out, "Next page token: tok-2")
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
