package nylas

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/nylas/cli/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newDeliverabilityTestClient(t *testing.T, handler http.HandlerFunc) *HTTPClient {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	client := NewHTTPClient()
	client.baseURL = server.URL
	client.SetCredentials("", "", "test-api-key")
	return client
}

func TestListSendEvents_QueryString(t *testing.T) {
	var gotQuery map[string][]string
	client := newDeliverabilityTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/v3/activity/send", r.URL.Path)
		gotQuery = r.URL.Query()
		_, _ = io.WriteString(w, `{"request_id":"req-1","data":[{"id":"e1","event_type":"bounced","bounce_class":"hard","timestamp":1730000000}],"next_cursor":"tok-2"}`)
	})

	resp, err := client.ListSendEvents(context.Background(), &domain.SendEventListParams{
		EventTypes:      []string{"bounced", "complaint"},
		RecipientDomain: "gmail.com",
		Start:           1729000000,
		Limit:           10,
		PageToken:       "tok-1",
	})
	require.NoError(t, err)

	// The API rejects a comma-joined event_type, so each value must be its own key.
	assert.Equal(t, []string{"bounced", "complaint"}, gotQuery["event_type"])
	assert.Equal(t, []string{"gmail.com"}, gotQuery["recipient_domain"])
	assert.Equal(t, []string{"1729000000"}, gotQuery["start"])
	assert.Equal(t, []string{"10"}, gotQuery["limit"])
	assert.Equal(t, []string{"tok-1"}, gotQuery["page_token"])
	// The API returns 400 for any empty filter, so unset filters must not be sent at all.
	for _, key := range []string{"end", "order", "from", "grant_id", "recipient", "bounce_class", "sending_domain", "message_id", "thread_id"} {
		_, present := gotQuery[key]
		assert.False(t, present, "unset filter %q was sent", key)
	}

	assert.Equal(t, "tok-2", resp.NextCursor)
	assert.Equal(t, "req-1", resp.RequestID)
	require.Len(t, resp.Data, 1)
	assert.Equal(t, "hard", resp.Data[0].BounceClass)
}

func TestListTrackingEvents_SendsOnlyTrackingFilters(t *testing.T) {
	client := newDeliverabilityTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/activity/tracking", r.URL.Path)
		assert.Equal(t, "event_type=link_clicked&label=welcome", r.URL.RawQuery)
		_, _ = io.WriteString(w, `{"data":[]}`)
	})

	resp, err := client.ListTrackingEvents(context.Background(), &domain.TrackingEventListParams{
		EventTypes: []string{"link_clicked"},
		Label:      "welcome",
	})
	require.NoError(t, err)
	assert.Empty(t, resp.NextCursor)
}

func TestGetSendMessageActivity_EscapesMessageID(t *testing.T) {
	client := newDeliverabilityTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		// Message IDs can contain characters that are not path-safe.
		assert.Equal(t, "/v3/activity/send/messages/%3Cabc%2Fdef@mail.example.com%3E", r.RequestURI)
		_, _ = io.WriteString(w, `{"data":{"message":{"message_id":"<abc/def@mail.example.com>"},"recipients":[]}}`)
	})

	activity, err := client.GetSendMessageActivity(context.Background(), "<abc/def@mail.example.com>")
	require.NoError(t, err)
	assert.Equal(t, "<abc/def@mail.example.com>", activity.Message.MessageID)
}

func TestDeliverabilityRoutes(t *testing.T) {
	const eventID = "5f2c7a4e9b1d3c8f0a6e2d4b7c9f1a3e5d7b9c2f4a6e8d0b1c3f5a7e9d2b4c6f"
	tests := []struct {
		name string
		path string
		call func(*HTTPClient) error
	}{
		{"send event", "/v3/activity/send/" + eventID, func(c *HTTPClient) error {
			_, err := c.GetSendEvent(context.Background(), eventID)
			return err
		}},
		{"tracking event", "/v3/activity/tracking/" + eventID, func(c *HTTPClient) error {
			_, err := c.GetTrackingEvent(context.Background(), eventID)
			return err
		}},
		{"tracking message", "/v3/activity/tracking/messages/m1", func(c *HTTPClient) error {
			_, err := c.GetTrackingMessageActivity(context.Background(), "m1")
			return err
		}},
		{"send summary", "/v3/metrics/send/summary", func(c *HTTPClient) error {
			_, err := c.GetSendMetricsSummary(context.Background(), nil)
			return err
		}},
		{"send timeseries", "/v3/metrics/send/timeseries", func(c *HTTPClient) error {
			_, err := c.GetSendMetricsTimeseries(context.Background(), nil)
			return err
		}},
		{"send breakdown", "/v3/metrics/send", func(c *HTTPClient) error {
			_, err := c.GetSendMetricsBreakdown(context.Background(), nil)
			return err
		}},
		{"tracking summary", "/v3/metrics/tracking/summary", func(c *HTTPClient) error {
			_, err := c.GetTrackingMetricsSummary(context.Background(), nil)
			return err
		}},
		{"tracking timeseries", "/v3/metrics/tracking/timeseries", func(c *HTTPClient) error {
			_, err := c.GetTrackingMetricsTimeseries(context.Background(), nil)
			return err
		}},
		{"tracking breakdown", "/v3/metrics/tracking", func(c *HTTPClient) error {
			_, err := c.GetTrackingMetricsBreakdown(context.Background(), nil)
			return err
		}},
		{"list alert rules", "/v3/alerts/send", func(c *HTTPClient) error {
			_, err := c.ListSendAlertRules(context.Background())
			return err
		}},
		{"get alert rule", "/v3/alerts/send/r1", func(c *HTTPClient) error {
			_, err := c.GetSendAlertRule(context.Background(), "r1")
			return err
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body := `{"data":{}}`
			if tt.path == "/v3/alerts/send" {
				body = `{"data":[]}`
			}
			client := newDeliverabilityTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, http.MethodGet, r.Method)
				assert.Equal(t, tt.path, r.URL.Path)
				assert.Empty(t, r.URL.RawQuery)
				_, _ = io.WriteString(w, body)
			})
			require.NoError(t, tt.call(client))
		})
	}
}

func TestGetSendMetricsBreakdown_QueryAndNullRates(t *testing.T) {
	client := newDeliverabilityTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/metrics/send", r.URL.Path)
		assert.Equal(t, "end=2026-10-31&group_by=recipient_domain&limit=5&sort=bounce_rate%3Adesc&start=2026-10-01", r.URL.RawQuery)
		_, _ = io.WriteString(w, `{"data":{"group_by":"recipient_domain","window":null,"rows":[{"key":"x.com","sent":0,"delivery_rate":null,"is_others":false}]}}`)
	})

	breakdown, err := client.GetSendMetricsBreakdown(context.Background(), &domain.MetricsParams{
		Start: "2026-10-01", End: "2026-10-31", GroupBy: "recipient_domain", Sort: "bounce_rate:desc", Limit: 5,
		Granularity: "week", // timeseries-only, must not leak into the breakdown query
	})
	require.NoError(t, err)
	// A null window and null rates mean "no data", which must stay distinguishable from 0.
	assert.Nil(t, breakdown.Window)
	require.Len(t, breakdown.Rows, 1)
	assert.Nil(t, breakdown.Rows[0].DeliveryRate)
}

func TestGetSendMetricsSummary_DecodesHealth(t *testing.T) {
	client := newDeliverabilityTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "sending_domain=mail.example.com", r.URL.RawQuery)
		_, _ = io.WriteString(w, `{"data":{"health":{"state":"critical","window":{"start":"2026-10-27","end":"2026-11-02"},"min_assess_volume":100,
			"findings":[{"rule":"concentration","severity":"high","title":"Bounces concentrate on x.com","evidence":{"metric":"bounce_rate","value":12.5,"threshold":10,"totals":{"sent":400}}}]},
			"totals":{"sent":400,"bounced_hard":50},"metrics":{"bounce_rate":{"value":12.5,"delta_pp":4,"prior_value":8.5,"status":"crit"},
			"delivery_rate":{"value":null,"delta_pp":null,"prior_value":null,"status":null},"complaint_rate":{"value":0,"delta_pp":null,"prior_value":null,"status":"ok"}},
			"prior_period":null}}`)
	})

	s, err := client.GetSendMetricsSummary(context.Background(), &domain.MetricsParams{SendingDomain: "mail.example.com"})
	require.NoError(t, err)
	assert.Equal(t, "critical", s.Health.State)
	require.Len(t, s.Health.Findings, 1)
	assert.Equal(t, int64(400), s.Health.Findings[0].Evidence.Totals.Sent)
	require.NotNil(t, s.Metrics.BounceRate.Status)
	assert.Equal(t, "crit", *s.Metrics.BounceRate.Status)
	assert.Nil(t, s.Metrics.DeliveryRate.Value)
	require.NotNil(t, s.Metrics.ComplaintRate.Value)
	assert.Zero(t, *s.Metrics.ComplaintRate.Value)
	assert.Nil(t, s.PriorPeriod)
}

func TestCreateSendAlertRule_BodyAndStatus(t *testing.T) {
	var body map[string]json.RawMessage
	client := newDeliverabilityTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/v3/alerts/send", r.URL.Path)
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		w.WriteHeader(http.StatusCreated)
		_, _ = io.WriteString(w, `{"request_id":"r","data":{"id":"rule-1","metric":"bounce_rate","direction":"above","threshold":0.25,"window_hours":1}}`)
	})

	metric := "bounce_rate"
	threshold := json.Number("0.25")
	hours := int64(1)
	rule, err := client.CreateSendAlertRule(context.Background(), &domain.SendAlertRuleRequest{
		Metric: &metric, Threshold: &threshold, WindowHours: &hours,
	})
	require.NoError(t, err)
	assert.Equal(t, "rule-1", rule.ID)

	// The API rejects string thresholds and unknown or null fields, so the body must be exactly these.
	assert.JSONEq(t, `"bounce_rate"`, string(body["metric"]))
	assert.Equal(t, "0.25", string(body["threshold"]))
	assert.Equal(t, "1", string(body["window_hours"]))
	_, hasMinVolume := body["min_volume"]
	assert.False(t, hasMinVolume)
}

func TestUpdateSendAlertRule_PartialBody(t *testing.T) {
	client := newDeliverabilityTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPut, r.Method)
		assert.Equal(t, "/v3/alerts/send/rule-1", r.URL.Path)
		raw, _ := io.ReadAll(r.Body)
		// PUT is partial: only the changed field may be sent, or other fields would be reset.
		assert.JSONEq(t, `{"threshold":3}`, string(raw))
		_, _ = io.WriteString(w, `{"data":{"id":"rule-1","threshold":3}}`)
	})

	threshold := json.Number("3")
	rule, err := client.UpdateSendAlertRule(context.Background(), "rule-1", &domain.SendAlertRuleRequest{Threshold: &threshold})
	require.NoError(t, err)
	assert.InDelta(t, 3.0, rule.Threshold, 0)
}

func TestDeleteSendAlertRule(t *testing.T) {
	client := newDeliverabilityTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodDelete, r.Method)
		assert.Equal(t, "/v3/alerts/send/rule-1", r.URL.Path)
		_, _ = io.WriteString(w, `{"request_id":"r"}`)
	})
	require.NoError(t, client.DeleteSendAlertRule(context.Background(), "rule-1"))
}

func TestDeliverability_APIErrorKeepsTypeAndMessage(t *testing.T) {
	client := newDeliverabilityTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = io.WriteString(w, `{"request_id":"r","error":{"type":"api.forbidden","message":"this endpoint requires an application API key"}}`)
	})

	_, err := client.ListSendEvents(context.Background(), nil)
	require.Error(t, err)
	// Users hitting this with a grant token need the server's explanation, not a generic failure.
	var apiErr *domain.APIError
	require.True(t, errors.As(err, &apiErr))
	assert.Equal(t, http.StatusForbidden, apiErr.StatusCode)
	assert.Equal(t, "api.forbidden", apiErr.Type)
	assert.Contains(t, apiErr.Message, "application API key")
}
