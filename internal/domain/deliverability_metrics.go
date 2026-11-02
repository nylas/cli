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
	Start string `yaml:"start" json:"start"`
	End   string `yaml:"end" json:"end"`
}

// SendCounts are send outcome counts.
type SendCounts struct {
	Sent                int64 `yaml:"sent" json:"sent"`
	Delivered           int64 `yaml:"delivered" json:"delivered"`
	BouncedHard         int64 `yaml:"bounced_hard" json:"bounced_hard"`
	BouncedSoft         int64 `yaml:"bounced_soft" json:"bounced_soft"`
	BouncedUndetermined int64 `yaml:"bounced_undetermined" json:"bounced_undetermined"`
	Rejected            int64 `yaml:"rejected" json:"rejected"`
	Complaints          int64 `yaml:"complaints" json:"complaints"`
}

// RateTile is a percentage with its change against the prior period. Nil means no data.
type RateTile struct {
	Value      *float64 `yaml:"value" json:"value"`
	DeltaPP    *float64 `yaml:"delta_pp" json:"delta_pp"`
	PriorValue *float64 `yaml:"prior_value" json:"prior_value"`
}

// GradedRateTile is a RateTile with a health status: ok, warn, crit or nil.
type GradedRateTile struct {
	RateTile `yaml:",inline"`
	Status   *string `yaml:"status" json:"status"`
}

// SendHealthEvidence is the data behind a health finding.
type SendHealthEvidence struct {
	Metric         string        `yaml:"metric" json:"metric"`
	Value          float64       `yaml:"value" json:"value"`
	Threshold      float64       `yaml:"threshold" json:"threshold"`
	Dimension      string        `yaml:"dimension" json:"dimension"`
	DimensionValue string        `yaml:"dimension_value" json:"dimension_value"`
	ShareOfExcess  *float64      `yaml:"share_of_excess,omitempty" json:"share_of_excess,omitempty"`
	Day            *string       `yaml:"day,omitempty" json:"day,omitempty"`
	BaselineValue  *float64      `yaml:"baseline_value,omitempty" json:"baseline_value,omitempty"`
	Totals         SendCounts    `yaml:"totals" json:"totals"`
	Window         MetricsWindow `yaml:"window" json:"window"`
}

// SendHealthFinding explains an at_risk or critical health state.
type SendHealthFinding struct {
	Rule           string             `yaml:"rule" json:"rule"`
	Severity       string             `yaml:"severity" json:"severity"`
	Title          string             `yaml:"title" json:"title"`
	Evidence       SendHealthEvidence `yaml:"evidence" json:"evidence"`
	Recommendation string             `yaml:"recommendation" json:"recommendation"`
	LogsFilter     map[string]any     `yaml:"logs_filter" json:"logs_filter"`
}

// SendHealth grades the last 7 days, whatever window was requested.
type SendHealth struct {
	State           string              `yaml:"state" json:"state"`
	Window          MetricsWindow       `yaml:"window" json:"window"`
	MinAssessVolume int64               `yaml:"min_assess_volume" json:"min_assess_volume"`
	Findings        []SendHealthFinding `yaml:"findings" json:"findings"`
}

// SendRateTiles are the three graded send rates.
type SendRateTiles struct {
	DeliveryRate  GradedRateTile `yaml:"delivery_rate" json:"delivery_rate"`
	BounceRate    GradedRateTile `yaml:"bounce_rate" json:"bounce_rate"`
	ComplaintRate GradedRateTile `yaml:"complaint_rate" json:"complaint_rate"`
}

// SendMetricsSummary is the response of GET /v3/metrics/send/summary.
type SendMetricsSummary struct {
	Health      SendHealth     `yaml:"health" json:"health"`
	Totals      SendCounts     `yaml:"totals" json:"totals"`
	Metrics     SendRateTiles  `yaml:"metrics" json:"metrics"`
	PriorPeriod *MetricsWindow `yaml:"prior_period" json:"prior_period"`
}

// SendMetricsBucket is one timeseries bucket of send counts.
type SendMetricsBucket struct {
	Start         string `yaml:"start" json:"start"`
	End           string `yaml:"end" json:"end"`
	SendCounts    `yaml:",inline"`
	BounceRate    *float64 `yaml:"bounce_rate" json:"bounce_rate"`
	ComplaintRate *float64 `yaml:"complaint_rate" json:"complaint_rate"`
}

// SendMetricsTimeseries is the response of GET /v3/metrics/send/timeseries.
type SendMetricsTimeseries struct {
	Granularity string              `yaml:"granularity" json:"granularity"`
	Buckets     []SendMetricsBucket `yaml:"buckets" json:"buckets"`
}

// SendMetricsRow is one breakdown row; Key is "(others)" when IsOthers is true.
type SendMetricsRow struct {
	Key           string   `yaml:"key" json:"key"`
	Sent          int64    `yaml:"sent" json:"sent"`
	Delivered     int64    `yaml:"delivered" json:"delivered"`
	BouncedHard   int64    `yaml:"bounced_hard" json:"bounced_hard"`
	Complaints    int64    `yaml:"complaints" json:"complaints"`
	DeliveryRate  *float64 `yaml:"delivery_rate" json:"delivery_rate"`
	BounceRate    *float64 `yaml:"bounce_rate" json:"bounce_rate"`
	ComplaintRate *float64 `yaml:"complaint_rate" json:"complaint_rate"`
	IsOthers      bool     `yaml:"is_others" json:"is_others"`
}

// SendMetricsBreakdown is the response of GET /v3/metrics/send. Window is nil when
// a recipient_domain breakdown falls entirely outside the last 90 days.
type SendMetricsBreakdown struct {
	GroupBy string           `yaml:"group_by" json:"group_by"`
	Window  *MetricsWindow   `yaml:"window" json:"window"`
	Rows    []SendMetricsRow `yaml:"rows" json:"rows"`
}

// EngagementCounts are tracking counts; Opened and Clicked count first interactions.
type EngagementCounts struct {
	TrackedOpens int64 `yaml:"tracked_opens" json:"tracked_opens"`
	TrackedLinks int64 `yaml:"tracked_links" json:"tracked_links"`
	Opened       int64 `yaml:"opened" json:"opened"`
	Clicked      int64 `yaml:"clicked" json:"clicked"`
	OpensTotal   int64 `yaml:"opens_total" json:"opens_total"`
	ClicksTotal  int64 `yaml:"clicks_total" json:"clicks_total"`
}

// EngagementRateTiles are the open and click rate tiles.
type EngagementRateTiles struct {
	OpenRate  RateTile `yaml:"open_rate" json:"open_rate"`
	ClickRate RateTile `yaml:"click_rate" json:"click_rate"`
}

// TrackingMetricsSummary is the response of GET /v3/metrics/tracking/summary.
type TrackingMetricsSummary struct {
	Totals      EngagementCounts    `yaml:"totals" json:"totals"`
	Metrics     EngagementRateTiles `yaml:"metrics" json:"metrics"`
	PriorPeriod *MetricsWindow      `yaml:"prior_period" json:"prior_period"`
}

// TrackingMetricsBucket is one timeseries bucket of engagement counts.
type TrackingMetricsBucket struct {
	Start            string `yaml:"start" json:"start"`
	End              string `yaml:"end" json:"end"`
	EngagementCounts `yaml:",inline"`
	OpenRate         *float64 `yaml:"open_rate" json:"open_rate"`
	ClickRate        *float64 `yaml:"click_rate" json:"click_rate"`
}

// TrackingMetricsTimeseries is the response of GET /v3/metrics/tracking/timeseries.
type TrackingMetricsTimeseries struct {
	Granularity string                  `yaml:"granularity" json:"granularity"`
	Buckets     []TrackingMetricsBucket `yaml:"buckets" json:"buckets"`
}

// TrackingMetricsRow is one breakdown row; Key is "(others)" when IsOthers is true.
type TrackingMetricsRow struct {
	Key              string `yaml:"key" json:"key"`
	EngagementCounts `yaml:",inline"`
	OpenRate         *float64 `yaml:"open_rate" json:"open_rate"`
	ClickRate        *float64 `yaml:"click_rate" json:"click_rate"`
	IsOthers         bool     `yaml:"is_others" json:"is_others"`
}

// TrackingMetricsBreakdown is the response of GET /v3/metrics/tracking.
type TrackingMetricsBreakdown struct {
	GroupBy string               `yaml:"group_by" json:"group_by"`
	Window  *MetricsWindow       `yaml:"window" json:"window"`
	Rows    []TrackingMetricsRow `yaml:"rows" json:"rows"`
}
