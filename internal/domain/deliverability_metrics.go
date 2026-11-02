package domain

// MetricsGranularities are the timeseries bucket sizes the API accepts.
var MetricsGranularities = []string{"auto", "day", "week", "month"}

// SendMetricsGroupBy are the send breakdown dimensions.
var SendMetricsGroupBy = []string{"sending_domain", "agent_account", "recipient_domain"}

// TrackingMetricsGroupBy are the tracking breakdown dimensions.
var TrackingMetricsGroupBy = []string{"sending_domain", "agent_account"}

// MetricsParams configures the /v3/metrics routes. Start and End are YYYY-MM-DD (UTC, inclusive).
type MetricsParams struct {
	Start         string
	End           string
	SendingDomain string
	Granularity   string // timeseries only
	GroupBy       string // breakdown only
	Sort          string // breakdown only, "<key>:<asc|desc>"
	Limit         int    // breakdown only
}

// MetricsWindow is an inclusive UTC date range.
type MetricsWindow struct {
	Start string `json:"start"`
	End   string `json:"end"`
}

// SendCounts are send outcome counts.
type SendCounts struct {
	Sent                int64 `json:"sent"`
	Delivered           int64 `json:"delivered"`
	BouncedHard         int64 `json:"bounced_hard"`
	BouncedSoft         int64 `json:"bounced_soft"`
	BouncedUndetermined int64 `json:"bounced_undetermined"`
	Rejected            int64 `json:"rejected"`
	Complaints          int64 `json:"complaints"`
}

// RateTile is a percentage with its change against the prior period. Nil means no data.
type RateTile struct {
	Value      *float64 `json:"value"`
	DeltaPP    *float64 `json:"delta_pp"`
	PriorValue *float64 `json:"prior_value"`
}

// GradedRateTile is a RateTile with a health status: ok, warn, crit or nil.
type GradedRateTile struct {
	RateTile
	Status *string `json:"status"`
}

// SendHealthEvidence is the data behind a health finding.
type SendHealthEvidence struct {
	Metric         string        `json:"metric"`
	Value          float64       `json:"value"`
	Threshold      float64       `json:"threshold"`
	Dimension      string        `json:"dimension"`
	DimensionValue string        `json:"dimension_value"`
	ShareOfExcess  *float64      `json:"share_of_excess,omitempty"`
	Day            *string       `json:"day,omitempty"`
	BaselineValue  *float64      `json:"baseline_value,omitempty"`
	Totals         SendCounts    `json:"totals"`
	Window         MetricsWindow `json:"window"`
}

// SendHealthFinding explains an at_risk or critical health state.
type SendHealthFinding struct {
	Rule           string             `json:"rule"`
	Severity       string             `json:"severity"`
	Title          string             `json:"title"`
	Evidence       SendHealthEvidence `json:"evidence"`
	Recommendation string             `json:"recommendation"`
	LogsFilter     map[string]any     `json:"logs_filter"`
}

// SendHealth grades the last 7 days, whatever window was requested.
type SendHealth struct {
	State           string              `json:"state"`
	Window          MetricsWindow       `json:"window"`
	MinAssessVolume int64               `json:"min_assess_volume"`
	Findings        []SendHealthFinding `json:"findings"`
}

// SendRateTiles are the three graded send rates.
type SendRateTiles struct {
	DeliveryRate  GradedRateTile `json:"delivery_rate"`
	BounceRate    GradedRateTile `json:"bounce_rate"`
	ComplaintRate GradedRateTile `json:"complaint_rate"`
}

// SendMetricsSummary is the response of GET /v3/metrics/send/summary.
type SendMetricsSummary struct {
	Health      SendHealth     `json:"health"`
	Totals      SendCounts     `json:"totals"`
	Metrics     SendRateTiles  `json:"metrics"`
	PriorPeriod *MetricsWindow `json:"prior_period"`
}

// SendMetricsBucket is one timeseries bucket of send counts.
type SendMetricsBucket struct {
	Start string `json:"start"`
	End   string `json:"end"`
	SendCounts
	BounceRate    *float64 `json:"bounce_rate"`
	ComplaintRate *float64 `json:"complaint_rate"`
}

// SendMetricsTimeseries is the response of GET /v3/metrics/send/timeseries.
type SendMetricsTimeseries struct {
	Granularity string              `json:"granularity"`
	Buckets     []SendMetricsBucket `json:"buckets"`
}

// SendMetricsRow is one breakdown row; Key is "(others)" when IsOthers is true.
type SendMetricsRow struct {
	Key           string   `json:"key"`
	Sent          int64    `json:"sent"`
	Delivered     int64    `json:"delivered"`
	BouncedHard   int64    `json:"bounced_hard"`
	Complaints    int64    `json:"complaints"`
	DeliveryRate  *float64 `json:"delivery_rate"`
	BounceRate    *float64 `json:"bounce_rate"`
	ComplaintRate *float64 `json:"complaint_rate"`
	IsOthers      bool     `json:"is_others"`
}

// SendMetricsBreakdown is the response of GET /v3/metrics/send. Window is nil when
// a recipient_domain breakdown falls entirely outside the last 90 days.
type SendMetricsBreakdown struct {
	GroupBy string           `json:"group_by"`
	Window  *MetricsWindow   `json:"window"`
	Rows    []SendMetricsRow `json:"rows"`
}

// EngagementCounts are tracking counts; Opened and Clicked count first interactions.
type EngagementCounts struct {
	TrackedOpens int64 `json:"tracked_opens"`
	TrackedLinks int64 `json:"tracked_links"`
	Opened       int64 `json:"opened"`
	Clicked      int64 `json:"clicked"`
	OpensTotal   int64 `json:"opens_total"`
	ClicksTotal  int64 `json:"clicks_total"`
}

// EngagementRateTiles are the open and click rate tiles.
type EngagementRateTiles struct {
	OpenRate  RateTile `json:"open_rate"`
	ClickRate RateTile `json:"click_rate"`
}

// TrackingMetricsSummary is the response of GET /v3/metrics/tracking/summary.
type TrackingMetricsSummary struct {
	Totals      EngagementCounts    `json:"totals"`
	Metrics     EngagementRateTiles `json:"metrics"`
	PriorPeriod *MetricsWindow      `json:"prior_period"`
}

// TrackingMetricsBucket is one timeseries bucket of engagement counts.
type TrackingMetricsBucket struct {
	Start string `json:"start"`
	End   string `json:"end"`
	EngagementCounts
	OpenRate  *float64 `json:"open_rate"`
	ClickRate *float64 `json:"click_rate"`
}

// TrackingMetricsTimeseries is the response of GET /v3/metrics/tracking/timeseries.
type TrackingMetricsTimeseries struct {
	Granularity string                  `json:"granularity"`
	Buckets     []TrackingMetricsBucket `json:"buckets"`
}

// TrackingMetricsRow is one breakdown row; Key is "(others)" when IsOthers is true.
type TrackingMetricsRow struct {
	Key string `json:"key"`
	EngagementCounts
	OpenRate  *float64 `json:"open_rate"`
	ClickRate *float64 `json:"click_rate"`
	IsOthers  bool     `json:"is_others"`
}

// TrackingMetricsBreakdown is the response of GET /v3/metrics/tracking.
type TrackingMetricsBreakdown struct {
	GroupBy string               `json:"group_by"`
	Window  *MetricsWindow       `json:"window"`
	Rows    []TrackingMetricsRow `json:"rows"`
}
