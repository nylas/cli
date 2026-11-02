package domain

import "encoding/json"

// SendAlertMetrics are the metrics a send alert rule can watch.
var SendAlertMetrics = []string{"bounce_rate", "complaint_rate", "delivery_rate"}

// SendAlertRule is a send alert rule. Direction is "below" for delivery_rate and "above" otherwise.
type SendAlertRule struct {
	ID            string  `yaml:"id" json:"id"`
	ApplicationID string  `yaml:"application_id" json:"application_id"`
	Metric        string  `yaml:"metric" json:"metric"`
	Direction     string  `yaml:"direction" json:"direction"`
	Threshold     float64 `yaml:"threshold" json:"threshold"`
	WindowHours   int64   `yaml:"window_hours" json:"window_hours"`
	MinVolume     int64   `yaml:"min_volume" json:"min_volume"`
	CreatedAt     int64   `yaml:"created_at" json:"created_at"`
	UpdatedAt     int64   `yaml:"updated_at" json:"updated_at"`
}

// QuietField returns the rule ID for quiet output.
func (r SendAlertRule) QuietField() string { return r.ID }

// SendAlertRuleRequest is the body of POST and PUT /v3/alerts/send. Nil fields are left out,
// so PUT updates only the fields that are set. Threshold keeps the caller's exact decimals.
type SendAlertRuleRequest struct {
	Metric      *string      `json:"metric,omitempty"`
	Threshold   *json.Number `json:"threshold,omitempty"`
	WindowHours *int64       `json:"window_hours,omitempty"`
	MinVolume   *int64       `json:"min_volume,omitempty"`
}
