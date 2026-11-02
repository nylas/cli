package nylas

import (
	"cmp"
	"context"
	"slices"
	"time"

	"github.com/nylas/cli/internal/domain"
)

const (
	demoDeliverabilityApp    = "demo-app-id"
	demoDeliverabilityDomain = "mail.example.com"
	demoDeliverabilityFrom   = "agent@mail.example.com"
	demoDeliverabilityMsgID  = "<demo-message-1@mail.example.com>"
	demoSendEventID          = "5f2c7a4e9b1d3c8f0a6e2d4b7c9f1a3e5d7b9c2f4a6e8d0b1c3f5a7e9d2b4c6f"
	demoAcceptedEventID      = "e7f9b1d3a5c70829416b8d0f2a4c6e8b1d3f5a7c9e2b4d6f8a0c3e5b7d9f1a2c"
	demoBouncedEventID       = "c4e6a8b0d2f41638507a9cbe1d3f5a7c9e0b2d4f6a8c1e3b5d7f9a0c2e4b6d8f"
	demoTrackingEventID      = "a1b2c3d4e5f60718293a4b5c6d7e8f90a1b2c3d4e5f60718293a4b5c6d7e8f90"
	demoSendAlertRuleID      = "3d6f8a1c-2b4e-4c7d-9f0a-1b2c3d4e5f60"
)

func demoRate(v float64) *float64 { return &v }

// demoSendEvents is the one source of demo send events, so list, show and message agree on every ID.
func demoSendEvents(now int64) []domain.SendEvent {
	base := domain.SendEvent{ApplicationID: demoDeliverabilityApp, MessageID: demoDeliverabilityMsgID,
		From: demoDeliverabilityFrom, SendingDomain: demoDeliverabilityDomain}
	accepted, delivered, bounced := base, base, base

	accepted.ID, accepted.EventType, accepted.Timestamp = demoAcceptedEventID, "accepted", now-120
	accepted.Recipient, accepted.RecipientDomain = "alice@example.org", "example.org"

	delivered.ID, delivered.EventType, delivered.Timestamp = demoSendEventID, "delivered", now-60
	delivered.Recipient, delivered.RecipientDomain, delivered.Reason = "alice@example.org", "example.org", "250 2.0.0 OK"
	delivered.Detail = &domain.SendEventDetail{SMTPResponse: "250 2.0.0 OK", ProcessingTimeMs: 812}

	bounced.ID, bounced.EventType, bounced.BounceClass, bounced.Timestamp = demoBouncedEventID, "bounced", "hard", now-90
	bounced.Recipient, bounced.RecipientDomain, bounced.Reason = "bob@example.net", "example.net", "General (5.1.1)"

	return []domain.SendEvent{delivered, bounced, accepted}
}

func (d *DemoClient) ListSendEvents(ctx context.Context, params *domain.SendEventListParams) (*domain.SendEventListResponse, error) {
	return &domain.SendEventListResponse{Data: demoSendEvents(time.Now().Unix()), RequestID: "demo-request-id"}, nil
}

func (d *DemoClient) GetSendEvent(ctx context.Context, eventID string) (*domain.SendEvent, error) {
	events := demoSendEvents(time.Now().Unix())
	for i := range events {
		if events[i].ID == eventID {
			return &events[i], nil
		}
	}
	fallback := events[0]
	fallback.ID = eventID
	return &fallback, nil
}

func (d *DemoClient) GetSendMessageActivity(ctx context.Context, messageID string) (*domain.SendMessageActivity, error) {
	events := demoSendEvents(time.Now().Unix())
	recipients := map[string]*domain.SendMessageRecipient{}
	var order []string
	for _, e := range events {
		r, ok := recipients[e.Recipient]
		if !ok {
			r = &domain.SendMessageRecipient{Recipient: e.Recipient, RecipientDomain: e.RecipientDomain}
			recipients[e.Recipient] = r
			order = append(order, e.Recipient)
		}
		r.Events = append(r.Events, domain.SendTimelineEvent{ID: e.ID, EventType: e.EventType, Timestamp: e.Timestamp,
			BounceClass: e.BounceClass, Reason: e.Reason, Detail: e.Detail})
	}

	activity := &domain.SendMessageActivity{Message: domain.SendMessageEnvelope{MessageID: messageID,
		ApplicationID: demoDeliverabilityApp, From: demoDeliverabilityFrom, SendingDomain: demoDeliverabilityDomain}}
	slices.Sort(order)
	for _, name := range order {
		r := recipients[name]
		slices.SortFunc(r.Events, func(a, b domain.SendTimelineEvent) int { return cmp.Compare(a.Timestamp, b.Timestamp) })
		last := r.Events[len(r.Events)-1]
		r.CurrentStatus, r.BounceClass = last.EventType, last.BounceClass
		activity.Recipients = append(activity.Recipients, *r)
		if activity.Message.SentAt == 0 || r.Events[0].Timestamp < activity.Message.SentAt {
			activity.Message.SentAt = r.Events[0].Timestamp
		}
		activity.Message.LastEventAt = max(activity.Message.LastEventAt, last.Timestamp)
	}
	return activity, nil
}

func (d *DemoClient) ListTrackingEvents(ctx context.Context, params *domain.TrackingEventListParams) (*domain.TrackingEventListResponse, error) {
	event, _ := d.GetTrackingEvent(ctx, demoTrackingEventID)
	return &domain.TrackingEventListResponse{Data: []domain.TrackingEvent{*event}, RequestID: "demo-request-id"}, nil
}

func (d *DemoClient) GetTrackingEvent(ctx context.Context, eventID string) (*domain.TrackingEvent, error) {
	return &domain.TrackingEvent{ID: eventID, EventType: "opened", Timestamp: time.Now().Unix() - 30,
		ApplicationID: demoDeliverabilityApp, MessageID: demoDeliverabilityMsgID, From: demoDeliverabilityFrom,
		SendingDomain: demoDeliverabilityDomain, Label: "welcome", IsFirst: true, UserAgent: "Mozilla/5.0"}, nil
}

func (d *DemoClient) GetTrackingMessageActivity(ctx context.Context, messageID string) (*domain.TrackingMessageActivity, error) {
	first := time.Now().Unix() - 30
	return &domain.TrackingMessageActivity{
		MessageID: messageID, From: demoDeliverabilityFrom, SendingDomain: demoDeliverabilityDomain, Label: "welcome",
		Tracking: domain.TrackingFlags{Opens: true, Links: true},
		Opens:    domain.TrackingTotals{Total: 1, FirstAt: &first, LastAt: &first},
		Clicks:   domain.TrackingClickTotals{Links: []domain.TrackingLinkClicks{}},
		Events: []domain.TrackingInteraction{
			{ID: demoTrackingEventID, EventType: "opened", Timestamp: first, IsFirst: true, UserAgent: "Mozilla/5.0"},
		},
	}, nil
}

func (d *DemoClient) GetSendMetricsSummary(ctx context.Context, params *domain.MetricsParams) (*domain.SendMetricsSummary, error) {
	ok := "ok"
	return &domain.SendMetricsSummary{
		Health: domain.SendHealth{State: "healthy", MinAssessVolume: 100, Findings: []domain.SendHealthFinding{}},
		Totals: domain.SendCounts{Sent: 1200, Delivered: 1180, BouncedHard: 12, BouncedSoft: 5, Complaints: 1},
		Metrics: domain.SendRateTiles{
			DeliveryRate:  domain.GradedRateTile{RateTile: domain.RateTile{Value: demoRate(98.33), DeltaPP: demoRate(0.4)}, Status: &ok},
			BounceRate:    domain.GradedRateTile{RateTile: domain.RateTile{Value: demoRate(1.0), DeltaPP: demoRate(-0.2)}, Status: &ok},
			ComplaintRate: domain.GradedRateTile{RateTile: domain.RateTile{Value: demoRate(0.08)}, Status: &ok},
		},
	}, nil
}

func (d *DemoClient) GetSendMetricsTimeseries(ctx context.Context, params *domain.MetricsParams) (*domain.SendMetricsTimeseries, error) {
	day := time.Now().UTC().Format("2006-01-02")
	return &domain.SendMetricsTimeseries{Granularity: "day", Buckets: []domain.SendMetricsBucket{{
		Start: day, End: day, SendCounts: domain.SendCounts{Sent: 180, Delivered: 177, BouncedHard: 2},
		BounceRate: demoRate(1.11), ComplaintRate: demoRate(0),
	}}}, nil
}

func (d *DemoClient) GetSendMetricsBreakdown(ctx context.Context, params *domain.MetricsParams) (*domain.SendMetricsBreakdown, error) {
	return &domain.SendMetricsBreakdown{GroupBy: "sending_domain", Rows: []domain.SendMetricsRow{{
		Key: demoDeliverabilityDomain, Sent: 1200, Delivered: 1180, BouncedHard: 12, Complaints: 1,
		DeliveryRate: demoRate(98.33), BounceRate: demoRate(1.0), ComplaintRate: demoRate(0.08),
	}}}, nil
}

func (d *DemoClient) GetTrackingMetricsSummary(ctx context.Context, params *domain.MetricsParams) (*domain.TrackingMetricsSummary, error) {
	return &domain.TrackingMetricsSummary{
		Totals:  domain.EngagementCounts{TrackedOpens: 900, TrackedLinks: 600, Opened: 410, Clicked: 75, OpensTotal: 820, ClicksTotal: 110},
		Metrics: domain.EngagementRateTiles{OpenRate: domain.RateTile{Value: demoRate(45.56)}, ClickRate: domain.RateTile{Value: demoRate(12.5)}},
	}, nil
}

func (d *DemoClient) GetTrackingMetricsTimeseries(ctx context.Context, params *domain.MetricsParams) (*domain.TrackingMetricsTimeseries, error) {
	day := time.Now().UTC().Format("2006-01-02")
	return &domain.TrackingMetricsTimeseries{Granularity: "day", Buckets: []domain.TrackingMetricsBucket{{
		Start: day, End: day, EngagementCounts: domain.EngagementCounts{TrackedOpens: 120, Opened: 50},
		OpenRate: demoRate(41.67),
	}}}, nil
}

func (d *DemoClient) GetTrackingMetricsBreakdown(ctx context.Context, params *domain.MetricsParams) (*domain.TrackingMetricsBreakdown, error) {
	return &domain.TrackingMetricsBreakdown{GroupBy: "sending_domain", Rows: []domain.TrackingMetricsRow{{
		Key: demoDeliverabilityDomain, EngagementCounts: domain.EngagementCounts{TrackedOpens: 900, TrackedLinks: 600, Opened: 410, Clicked: 75},
		OpenRate: demoRate(45.56), ClickRate: demoRate(12.5),
	}}}, nil
}

func (d *DemoClient) ListSendAlertRules(ctx context.Context) ([]domain.SendAlertRule, error) {
	rule, _ := d.GetSendAlertRule(ctx, demoSendAlertRuleID)
	return []domain.SendAlertRule{*rule}, nil
}

func (d *DemoClient) GetSendAlertRule(ctx context.Context, ruleID string) (*domain.SendAlertRule, error) {
	now := time.Now().Unix()
	return &domain.SendAlertRule{ID: ruleID, ApplicationID: demoDeliverabilityApp, Metric: "bounce_rate",
		Direction: "above", Threshold: 5, WindowHours: 1, MinVolume: 50, CreatedAt: now - 86400, UpdatedAt: now - 86400}, nil
}

func (d *DemoClient) CreateSendAlertRule(ctx context.Context, req *domain.SendAlertRuleRequest) (*domain.SendAlertRule, error) {
	return d.GetSendAlertRule(ctx, demoSendAlertRuleID)
}

func (d *DemoClient) UpdateSendAlertRule(ctx context.Context, ruleID string, req *domain.SendAlertRuleRequest) (*domain.SendAlertRule, error) {
	return d.GetSendAlertRule(ctx, ruleID)
}

func (d *DemoClient) DeleteSendAlertRule(ctx context.Context, ruleID string) error {
	return nil
}
