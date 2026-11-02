package nylas

import (
	"context"

	"github.com/nylas/cli/internal/domain"
)

func (m *MockClient) ListSendEvents(ctx context.Context, params *domain.SendEventListParams) (*domain.SendEventListResponse, error) {
	if m.ListSendEventsFunc != nil {
		return m.ListSendEventsFunc(ctx, params)
	}
	return &domain.SendEventListResponse{Data: []domain.SendEvent{}}, nil
}

func (m *MockClient) GetSendEvent(ctx context.Context, eventID string) (*domain.SendEvent, error) {
	if m.GetSendEventFunc != nil {
		return m.GetSendEventFunc(ctx, eventID)
	}
	return &domain.SendEvent{ID: eventID, EventType: "delivered"}, nil
}

func (m *MockClient) GetSendMessageActivity(ctx context.Context, messageID string) (*domain.SendMessageActivity, error) {
	if m.GetSendMessageActivityFunc != nil {
		return m.GetSendMessageActivityFunc(ctx, messageID)
	}
	return &domain.SendMessageActivity{Message: domain.SendMessageEnvelope{MessageID: messageID}}, nil
}

func (m *MockClient) ListTrackingEvents(ctx context.Context, params *domain.TrackingEventListParams) (*domain.TrackingEventListResponse, error) {
	if m.ListTrackingEventsFunc != nil {
		return m.ListTrackingEventsFunc(ctx, params)
	}
	return &domain.TrackingEventListResponse{Data: []domain.TrackingEvent{}}, nil
}

func (m *MockClient) GetTrackingEvent(ctx context.Context, eventID string) (*domain.TrackingEvent, error) {
	if m.GetTrackingEventFunc != nil {
		return m.GetTrackingEventFunc(ctx, eventID)
	}
	return &domain.TrackingEvent{ID: eventID, EventType: "opened"}, nil
}

func (m *MockClient) GetTrackingMessageActivity(ctx context.Context, messageID string) (*domain.TrackingMessageActivity, error) {
	if m.GetTrackingMessageActivityFunc != nil {
		return m.GetTrackingMessageActivityFunc(ctx, messageID)
	}
	return &domain.TrackingMessageActivity{MessageID: messageID}, nil
}

func (m *MockClient) GetSendMetricsSummary(ctx context.Context, params *domain.MetricsParams) (*domain.SendMetricsSummary, error) {
	if m.GetSendMetricsSummaryFunc != nil {
		return m.GetSendMetricsSummaryFunc(ctx, params)
	}
	return &domain.SendMetricsSummary{Health: domain.SendHealth{State: "insufficient_data"}}, nil
}

func (m *MockClient) GetSendMetricsTimeseries(ctx context.Context, params *domain.MetricsParams) (*domain.SendMetricsTimeseries, error) {
	if m.GetSendMetricsTimeseriesFunc != nil {
		return m.GetSendMetricsTimeseriesFunc(ctx, params)
	}
	return &domain.SendMetricsTimeseries{Granularity: "day"}, nil
}

func (m *MockClient) GetSendMetricsBreakdown(ctx context.Context, params *domain.MetricsParams) (*domain.SendMetricsBreakdown, error) {
	if m.GetSendMetricsBreakdownFunc != nil {
		return m.GetSendMetricsBreakdownFunc(ctx, params)
	}
	return &domain.SendMetricsBreakdown{GroupBy: "sending_domain"}, nil
}

func (m *MockClient) GetTrackingMetricsSummary(ctx context.Context, params *domain.MetricsParams) (*domain.TrackingMetricsSummary, error) {
	if m.GetTrackingMetricsSummaryFunc != nil {
		return m.GetTrackingMetricsSummaryFunc(ctx, params)
	}
	return &domain.TrackingMetricsSummary{}, nil
}

func (m *MockClient) GetTrackingMetricsTimeseries(ctx context.Context, params *domain.MetricsParams) (*domain.TrackingMetricsTimeseries, error) {
	if m.GetTrackingMetricsTimeseriesFunc != nil {
		return m.GetTrackingMetricsTimeseriesFunc(ctx, params)
	}
	return &domain.TrackingMetricsTimeseries{Granularity: "day"}, nil
}

func (m *MockClient) GetTrackingMetricsBreakdown(ctx context.Context, params *domain.MetricsParams) (*domain.TrackingMetricsBreakdown, error) {
	if m.GetTrackingMetricsBreakdownFunc != nil {
		return m.GetTrackingMetricsBreakdownFunc(ctx, params)
	}
	return &domain.TrackingMetricsBreakdown{GroupBy: "sending_domain"}, nil
}

func (m *MockClient) ListSendAlertRules(ctx context.Context) ([]domain.SendAlertRule, error) {
	if m.ListSendAlertRulesFunc != nil {
		return m.ListSendAlertRulesFunc(ctx)
	}
	return []domain.SendAlertRule{}, nil
}

func (m *MockClient) GetSendAlertRule(ctx context.Context, ruleID string) (*domain.SendAlertRule, error) {
	if m.GetSendAlertRuleFunc != nil {
		return m.GetSendAlertRuleFunc(ctx, ruleID)
	}
	return &domain.SendAlertRule{ID: ruleID, Metric: "bounce_rate", Direction: "above"}, nil
}

func (m *MockClient) CreateSendAlertRule(ctx context.Context, req *domain.SendAlertRuleRequest) (*domain.SendAlertRule, error) {
	if m.CreateSendAlertRuleFunc != nil {
		return m.CreateSendAlertRuleFunc(ctx, req)
	}
	return &domain.SendAlertRule{ID: "rule-mock"}, nil
}

func (m *MockClient) UpdateSendAlertRule(ctx context.Context, ruleID string, req *domain.SendAlertRuleRequest) (*domain.SendAlertRule, error) {
	if m.UpdateSendAlertRuleFunc != nil {
		return m.UpdateSendAlertRuleFunc(ctx, ruleID, req)
	}
	return &domain.SendAlertRule{ID: ruleID}, nil
}

func (m *MockClient) DeleteSendAlertRule(ctx context.Context, ruleID string) error {
	if m.DeleteSendAlertRuleFunc != nil {
		return m.DeleteSendAlertRuleFunc(ctx, ruleID)
	}
	return nil
}
