package ports

import (
	"context"

	"github.com/nylas/cli/internal/domain"
)

// DeliverabilityClient reads send and tracking activity and metrics, and manages send alert rules.
// Every route is application-level: it needs an API key and rejects grant tokens.
type DeliverabilityClient interface {
	ListSendEvents(ctx context.Context, params *domain.SendEventListParams) (*domain.SendEventListResponse, error)
	GetSendEvent(ctx context.Context, eventID string) (*domain.SendEvent, error)
	GetSendMessageActivity(ctx context.Context, messageID string) (*domain.SendMessageActivity, error)
	ListTrackingEvents(ctx context.Context, params *domain.TrackingEventListParams) (*domain.TrackingEventListResponse, error)
	GetTrackingEvent(ctx context.Context, eventID string) (*domain.TrackingEvent, error)
	GetTrackingMessageActivity(ctx context.Context, messageID string) (*domain.TrackingMessageActivity, error)

	GetSendMetricsSummary(ctx context.Context, params *domain.MetricsParams) (*domain.SendMetricsSummary, error)
	GetSendMetricsTimeseries(ctx context.Context, params *domain.MetricsParams) (*domain.SendMetricsTimeseries, error)
	GetSendMetricsBreakdown(ctx context.Context, params *domain.MetricsParams) (*domain.SendMetricsBreakdown, error)
	GetTrackingMetricsSummary(ctx context.Context, params *domain.MetricsParams) (*domain.TrackingMetricsSummary, error)
	GetTrackingMetricsTimeseries(ctx context.Context, params *domain.MetricsParams) (*domain.TrackingMetricsTimeseries, error)
	GetTrackingMetricsBreakdown(ctx context.Context, params *domain.MetricsParams) (*domain.TrackingMetricsBreakdown, error)

	ListSendAlertRules(ctx context.Context) ([]domain.SendAlertRule, error)
	GetSendAlertRule(ctx context.Context, ruleID string) (*domain.SendAlertRule, error)
	CreateSendAlertRule(ctx context.Context, req *domain.SendAlertRuleRequest) (*domain.SendAlertRule, error)
	UpdateSendAlertRule(ctx context.Context, ruleID string, req *domain.SendAlertRuleRequest) (*domain.SendAlertRule, error)
	DeleteSendAlertRule(ctx context.Context, ruleID string) error
}
