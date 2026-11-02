package nylas

import (
	"context"

	"github.com/nylas/cli/internal/domain"
)

// GetSendMetricsSummary returns GET /v3/metrics/send/summary.
func (c *HTTPClient) GetSendMetricsSummary(ctx context.Context, params *domain.MetricsParams) (*domain.SendMetricsSummary, error) {
	return getMetrics[domain.SendMetricsSummary](ctx, c, metricsWindowQuery(params).BuildURL(c.baseURL+"/v3/metrics/send/summary"))
}

// GetSendMetricsTimeseries returns GET /v3/metrics/send/timeseries.
func (c *HTTPClient) GetSendMetricsTimeseries(ctx context.Context, params *domain.MetricsParams) (*domain.SendMetricsTimeseries, error) {
	return getMetrics[domain.SendMetricsTimeseries](ctx, c, metricsTimeseriesURL(c.baseURL+"/v3/metrics/send/timeseries", params))
}

// GetSendMetricsBreakdown returns GET /v3/metrics/send.
func (c *HTTPClient) GetSendMetricsBreakdown(ctx context.Context, params *domain.MetricsParams) (*domain.SendMetricsBreakdown, error) {
	return getMetrics[domain.SendMetricsBreakdown](ctx, c, metricsBreakdownURL(c.baseURL+"/v3/metrics/send", params))
}

// GetTrackingMetricsSummary returns GET /v3/metrics/tracking/summary.
func (c *HTTPClient) GetTrackingMetricsSummary(ctx context.Context, params *domain.MetricsParams) (*domain.TrackingMetricsSummary, error) {
	return getMetrics[domain.TrackingMetricsSummary](ctx, c, metricsWindowQuery(params).BuildURL(c.baseURL+"/v3/metrics/tracking/summary"))
}

// GetTrackingMetricsTimeseries returns GET /v3/metrics/tracking/timeseries.
func (c *HTTPClient) GetTrackingMetricsTimeseries(ctx context.Context, params *domain.MetricsParams) (*domain.TrackingMetricsTimeseries, error) {
	return getMetrics[domain.TrackingMetricsTimeseries](ctx, c, metricsTimeseriesURL(c.baseURL+"/v3/metrics/tracking/timeseries", params))
}

// GetTrackingMetricsBreakdown returns GET /v3/metrics/tracking.
func (c *HTTPClient) GetTrackingMetricsBreakdown(ctx context.Context, params *domain.MetricsParams) (*domain.TrackingMetricsBreakdown, error) {
	return getMetrics[domain.TrackingMetricsBreakdown](ctx, c, metricsBreakdownURL(c.baseURL+"/v3/metrics/tracking", params))
}

// getMetrics GETs a metrics route and unwraps its data envelope.
func getMetrics[T any](ctx context.Context, c *HTTPClient, queryURL string) (*T, error) {
	var result struct {
		Data T `json:"data"`
	}
	if err := c.doGet(ctx, queryURL, &result); err != nil {
		return nil, err
	}
	return &result.Data, nil
}

// metricsWindowQuery adds the window params every metrics route takes.
func metricsWindowQuery(params *domain.MetricsParams) *QueryBuilder {
	p := orEmptyMetricsParams(params)
	return NewQueryBuilder().Add("start", p.Start).Add("end", p.End).Add("sending_domain", p.SendingDomain)
}

func metricsTimeseriesURL(base string, params *domain.MetricsParams) string {
	p := orEmptyMetricsParams(params)
	return metricsWindowQuery(p).Add("granularity", p.Granularity).BuildURL(base)
}

func metricsBreakdownURL(base string, params *domain.MetricsParams) string {
	p := orEmptyMetricsParams(params)
	return metricsWindowQuery(p).Add("group_by", p.GroupBy).Add("sort", p.Sort).AddInt("limit", p.Limit).BuildURL(base)
}

func orEmptyMetricsParams(params *domain.MetricsParams) *domain.MetricsParams {
	if params == nil {
		return &domain.MetricsParams{}
	}
	return params
}
