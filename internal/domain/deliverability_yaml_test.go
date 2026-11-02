package domain

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// TestDeliverabilityYAMLMatchesJSON pins that --format yaml has the same keys and nesting as --json
// (and the API): a missing yaml tag lowercases the Go name, and a missing inline nests embedded structs.
func TestDeliverabilityYAMLMatchesJSON(t *testing.T) {
	f := func(v float64) *float64 { return &v }
	s := func(v string) *string { return &v }
	i := func(v int64) *int64 { return &v }
	window := MetricsWindow{Start: "2026-10-27", End: "2026-11-02"}
	counts := SendCounts{Sent: 400, Delivered: 340, BouncedHard: 50, BouncedSoft: 5, BouncedUndetermined: 1, Rejected: 2, Complaints: 3}
	engagement := EngagementCounts{TrackedOpens: 9, TrackedLinks: 6, Opened: 4, Clicked: 2, OpensTotal: 8, ClicksTotal: 3}
	tile := GradedRateTile{RateTile: RateTile{Value: f(12.5), DeltaPP: f(4), PriorValue: f(8.5)}, Status: s("crit")}
	event := SendEvent{ID: "e1", EventType: "bounced", BounceClass: "hard", Timestamp: 1, ApplicationID: "a", GrantID: "g",
		MessageID: "m", ThreadID: "th", From: "f@x.com", SendingDomain: "x.com", Recipient: "r@y.com", RecipientDomain: "y.com",
		Subject: s("Hi"), Reason: "General (5.1.1)", Detail: &SendEventDetail{SMTPResponse: "550", ProcessingTimeMs: 9,
			ReportingMTA: "mta", RemoteMTAIP: "1.2.3.4", BounceType: "Permanent", BounceSubtype: "General", Status: "5.1.1",
			Action: "failed", DiagnosticCode: "smtp; 550", FeedbackType: "abuse", ComplaintSubtype: "x"}}
	tracking := TrackingEvent{ID: "t1", EventType: "link_clicked", Timestamp: 2, ApplicationID: "a", GrantID: "g", MessageID: "m",
		ThreadID: "th", From: "f@x.com", SendingDomain: "x.com", Label: "l", IsFirst: true, URL: s("https://x.com"),
		LinkIndex: i(0), IP: "1.2.3.4", UserAgent: "ua"}

	values := map[string]any{
		"send list":  SendEventListResponse{Data: []SendEvent{event}, NextCursor: "tok", RequestID: "req"},
		"send event": event,
		"send message": SendMessageActivity{
			Message: SendMessageEnvelope{MessageID: "m", ThreadID: "th", GrantID: "g", ApplicationID: "a", From: "f@x.com",
				SendingDomain: "x.com", Subject: s("Hi"), SentAt: 1, LastEventAt: 2},
			Recipients: []SendMessageRecipient{{Recipient: "r@y.com", RecipientDomain: "y.com", CurrentStatus: "bounced",
				BounceClass: "hard", Events: []SendTimelineEvent{{ID: "e1", EventType: "bounced", Timestamp: 1,
					BounceClass: "hard", Reason: "r", Detail: event.Detail}}}},
		},
		"tracking list": TrackingEventListResponse{Data: []TrackingEvent{tracking}, NextCursor: "tok", RequestID: "req"},
		"tracking message": TrackingMessageActivity{MessageID: "m", ThreadID: "th", GrantID: "g", From: "f@x.com",
			SendingDomain: "x.com", Label: "l", Tracking: TrackingFlags{Opens: true, Links: true},
			Opens:  TrackingTotals{Total: 2, FirstAt: i(1), LastAt: i(2)},
			Clicks: TrackingClickTotals{TrackingTotals: TrackingTotals{Total: 1, FirstAt: i(1), LastAt: i(1)}, Links: []TrackingLinkClicks{{LinkIndex: 0, URL: "https://x.com", Clicks: 1}}},
			Events: []TrackingInteraction{{ID: "t1", EventType: "opened", Timestamp: 1, IsFirst: true, URL: s("u"), LinkIndex: i(0), IP: "ip", UserAgent: "ua"}}},
		"send summary": SendMetricsSummary{
			Health: SendHealth{State: "critical", Window: window, MinAssessVolume: 100, Findings: []SendHealthFinding{{
				Rule: "concentration", Severity: "high", Title: "t", Recommendation: "r", LogsFilter: map[string]any{"sending_domain": "x.com"},
				Evidence: SendHealthEvidence{Metric: "bounce_rate", Value: 12.5, Threshold: 10, Dimension: "recipient_domain",
					DimensionValue: "y.com", ShareOfExcess: f(0.5), Day: s("2026-10-30"), BaselineValue: f(2), Totals: counts, Window: window}}}},
			Totals:      counts,
			Metrics:     SendRateTiles{DeliveryRate: tile, BounceRate: tile, ComplaintRate: GradedRateTile{}},
			PriorPeriod: &window,
		},
		"send timeseries": SendMetricsTimeseries{Granularity: "day", Buckets: []SendMetricsBucket{{Start: "a", End: "b",
			SendCounts: counts, BounceRate: f(1), ComplaintRate: nil}}},
		"send breakdown": SendMetricsBreakdown{GroupBy: "recipient_domain", Window: nil, Rows: []SendMetricsRow{{Key: "k",
			Sent: 1, Delivered: 1, BouncedHard: 0, Complaints: 0, DeliveryRate: f(100), BounceRate: f(0), ComplaintRate: nil, IsOthers: true}}},
		"tracking summary": TrackingMetricsSummary{Totals: engagement, Metrics: EngagementRateTiles{
			OpenRate: RateTile{Value: f(44.4)}, ClickRate: RateTile{}}, PriorPeriod: nil},
		"tracking timeseries": TrackingMetricsTimeseries{Granularity: "week", Buckets: []TrackingMetricsBucket{{Start: "a", End: "b",
			EngagementCounts: engagement, OpenRate: f(44.4), ClickRate: nil}}},
		"tracking breakdown": TrackingMetricsBreakdown{GroupBy: "agent_account", Window: &window, Rows: []TrackingMetricsRow{{Key: "k",
			EngagementCounts: engagement, OpenRate: f(44.4), ClickRate: f(33.3), IsOthers: false}}},
		"alert rule": SendAlertRule{ID: "r", ApplicationID: "a", Metric: "bounce_rate", Direction: "above", Threshold: 5,
			WindowHours: 1, MinVolume: 50, CreatedAt: 1, UpdatedAt: 2},
	}

	for name, v := range values {
		t.Run(name, func(t *testing.T) {
			jsonBytes, err := json.Marshal(v)
			require.NoError(t, err)
			var fromJSON any
			require.NoError(t, json.Unmarshal(jsonBytes, &fromJSON))

			yamlBytes, err := yaml.Marshal(v)
			require.NoError(t, err)
			var fromYAML any
			require.NoError(t, yaml.Unmarshal(yamlBytes, &fromYAML))
			viaYAML, err := json.Marshal(fromYAML)
			require.NoError(t, err)
			var fromYAMLAsJSON any
			require.NoError(t, json.Unmarshal(viaYAML, &fromYAMLAsJSON))

			require.Equal(t, fromJSON, fromYAMLAsJSON, "YAML:\n%s", yamlBytes)
		})
	}
}
