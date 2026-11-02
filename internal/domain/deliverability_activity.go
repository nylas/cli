package domain

// Send event types accepted by GET /v3/activity/send.
var SendEventTypes = []string{"accepted", "delivered", "bounced", "complaint", "rejected"}

// Tracking event types accepted by GET /v3/activity/tracking.
var TrackingEventTypes = []string{"opened", "link_clicked"}

// SendEventListParams filters GET /v3/activity/send. Start and End are Unix seconds.
type SendEventListParams struct {
	EventTypes      []string
	BounceClass     string
	SendingDomain   string
	From            string
	GrantID         string
	Recipient       string
	RecipientDomain string
	MessageID       string
	ThreadID        string
	Start           int64
	End             int64
	Order           string
	Limit           int
	PageToken       string
}

// TrackingEventListParams filters GET /v3/activity/tracking. Start and End are Unix seconds.
type TrackingEventListParams struct {
	EventTypes    []string
	SendingDomain string
	From          string
	GrantID       string
	MessageID     string
	ThreadID      string
	Label         string
	Start         int64
	End           int64
	Order         string
	Limit         int
	PageToken     string
}

// SendEventDetail carries provider (SES) detail for a send event.
type SendEventDetail struct {
	SMTPResponse     string `yaml:"smtp_response,omitempty" json:"smtp_response,omitempty"`
	ProcessingTimeMs int64  `yaml:"processing_time_ms,omitempty" json:"processing_time_ms,omitempty"`
	ReportingMTA     string `yaml:"reporting_mta,omitempty" json:"reporting_mta,omitempty"`
	RemoteMTAIP      string `yaml:"remote_mta_ip,omitempty" json:"remote_mta_ip,omitempty"`
	BounceType       string `yaml:"bounce_type,omitempty" json:"bounce_type,omitempty"`
	BounceSubtype    string `yaml:"bounce_subtype,omitempty" json:"bounce_subtype,omitempty"`
	Status           string `yaml:"status,omitempty" json:"status,omitempty"`
	Action           string `yaml:"action,omitempty" json:"action,omitempty"`
	DiagnosticCode   string `yaml:"diagnostic_code,omitempty" json:"diagnostic_code,omitempty"`
	FeedbackType     string `yaml:"feedback_type,omitempty" json:"feedback_type,omitempty"`
	ComplaintSubtype string `yaml:"complaint_subtype,omitempty" json:"complaint_subtype,omitempty"`
}

// SendEvent is one row of the send activity feed.
type SendEvent struct {
	ID              string           `yaml:"id" json:"id"`
	EventType       string           `yaml:"event_type" json:"event_type"`
	BounceClass     string           `yaml:"bounce_class,omitempty" json:"bounce_class,omitempty"`
	Timestamp       int64            `yaml:"timestamp" json:"timestamp"`
	ApplicationID   string           `yaml:"application_id" json:"application_id"`
	GrantID         string           `yaml:"grant_id" json:"grant_id"`
	MessageID       string           `yaml:"message_id" json:"message_id"`
	ThreadID        string           `yaml:"thread_id" json:"thread_id"`
	From            string           `yaml:"from" json:"from"`
	SendingDomain   string           `yaml:"sending_domain" json:"sending_domain"`
	Recipient       string           `yaml:"recipient" json:"recipient"`
	RecipientDomain string           `yaml:"recipient_domain" json:"recipient_domain"`
	Subject         *string          `yaml:"subject,omitempty" json:"subject,omitempty"`
	Reason          string           `yaml:"reason" json:"reason"`
	Detail          *SendEventDetail `yaml:"detail,omitempty" json:"detail,omitempty"`
}

// QuietField returns the event ID for quiet output.
func (e SendEvent) QuietField() string { return e.ID }

// SendEventListResponse is one page of the send activity feed.
type SendEventListResponse struct {
	Data       []SendEvent `yaml:"data" json:"data"`
	NextCursor string      `yaml:"next_cursor,omitempty" json:"next_cursor,omitempty"`
	RequestID  string      `yaml:"request_id,omitempty" json:"request_id,omitempty"`
}

// SendMessageEnvelope describes the message in a per-message send view.
type SendMessageEnvelope struct {
	MessageID     string  `yaml:"message_id" json:"message_id"`
	ThreadID      string  `yaml:"thread_id" json:"thread_id"`
	GrantID       string  `yaml:"grant_id" json:"grant_id"`
	ApplicationID string  `yaml:"application_id" json:"application_id"`
	From          string  `yaml:"from" json:"from"`
	SendingDomain string  `yaml:"sending_domain" json:"sending_domain"`
	Subject       *string `yaml:"subject,omitempty" json:"subject,omitempty"`
	SentAt        int64   `yaml:"sent_at" json:"sent_at"`
	LastEventAt   int64   `yaml:"last_event_at" json:"last_event_at"`
}

// SendTimelineEvent is one event in a recipient's timeline.
type SendTimelineEvent struct {
	ID          string           `yaml:"id" json:"id"`
	EventType   string           `yaml:"event_type" json:"event_type"`
	Timestamp   int64            `yaml:"timestamp" json:"timestamp"`
	BounceClass string           `yaml:"bounce_class,omitempty" json:"bounce_class,omitempty"`
	Reason      string           `yaml:"reason" json:"reason"`
	Detail      *SendEventDetail `yaml:"detail,omitempty" json:"detail,omitempty"`
}

// SendMessageRecipient is one recipient's status and timeline.
type SendMessageRecipient struct {
	Recipient       string              `yaml:"recipient" json:"recipient"`
	RecipientDomain string              `yaml:"recipient_domain" json:"recipient_domain"`
	CurrentStatus   string              `yaml:"current_status" json:"current_status"`
	BounceClass     string              `yaml:"bounce_class,omitempty" json:"bounce_class,omitempty"`
	Events          []SendTimelineEvent `yaml:"events" json:"events"`
}

// SendMessageActivity is the response of GET /v3/activity/send/messages/{message_id}.
type SendMessageActivity struct {
	Message    SendMessageEnvelope    `yaml:"message" json:"message"`
	Recipients []SendMessageRecipient `yaml:"recipients" json:"recipients"`
}

// TrackingEvent is one row of the tracking activity feed.
type TrackingEvent struct {
	ID            string  `yaml:"id" json:"id"`
	EventType     string  `yaml:"event_type" json:"event_type"`
	Timestamp     int64   `yaml:"timestamp" json:"timestamp"`
	ApplicationID string  `yaml:"application_id" json:"application_id"`
	GrantID       string  `yaml:"grant_id" json:"grant_id"`
	MessageID     string  `yaml:"message_id" json:"message_id"`
	ThreadID      string  `yaml:"thread_id" json:"thread_id"`
	From          string  `yaml:"from" json:"from"`
	SendingDomain string  `yaml:"sending_domain" json:"sending_domain"`
	Label         string  `yaml:"label" json:"label"`
	IsFirst       bool    `yaml:"is_first" json:"is_first"`
	URL           *string `yaml:"url,omitempty" json:"url,omitempty"`
	LinkIndex     *int64  `yaml:"link_index,omitempty" json:"link_index,omitempty"`
	IP            string  `yaml:"ip" json:"ip"`
	UserAgent     string  `yaml:"user_agent" json:"user_agent"`
}

// QuietField returns the event ID for quiet output.
func (e TrackingEvent) QuietField() string { return e.ID }

// TrackingEventListResponse is one page of the tracking activity feed.
type TrackingEventListResponse struct {
	Data       []TrackingEvent `yaml:"data" json:"data"`
	NextCursor string          `yaml:"next_cursor,omitempty" json:"next_cursor,omitempty"`
	RequestID  string          `yaml:"request_id,omitempty" json:"request_id,omitempty"`
}

// TrackingFlags reports which tracking was enabled for a message.
type TrackingFlags struct {
	Opens bool `yaml:"opens" json:"opens"`
	Links bool `yaml:"links" json:"links"`
}

// TrackingTotals counts interactions; FirstAt and LastAt are nil when Total is 0.
type TrackingTotals struct {
	Total   int64  `yaml:"total" json:"total"`
	FirstAt *int64 `yaml:"first_at" json:"first_at"`
	LastAt  *int64 `yaml:"last_at" json:"last_at"`
}

// TrackingLinkClicks counts clicks on one tracked link.
type TrackingLinkClicks struct {
	LinkIndex int64  `yaml:"link_index" json:"link_index"`
	URL       string `yaml:"url" json:"url"`
	Clicks    int64  `yaml:"clicks" json:"clicks"`
}

// TrackingClickTotals counts clicks overall and per link.
type TrackingClickTotals struct {
	TrackingTotals `yaml:",inline"`
	Links          []TrackingLinkClicks `yaml:"links" json:"links"`
}

// TrackingInteraction is one open or click in a per-message tracking view.
type TrackingInteraction struct {
	ID        string  `yaml:"id" json:"id"`
	EventType string  `yaml:"event_type" json:"event_type"`
	Timestamp int64   `yaml:"timestamp" json:"timestamp"`
	IsFirst   bool    `yaml:"is_first" json:"is_first"`
	URL       *string `yaml:"url,omitempty" json:"url,omitempty"`
	LinkIndex *int64  `yaml:"link_index,omitempty" json:"link_index,omitempty"`
	IP        string  `yaml:"ip" json:"ip"`
	UserAgent string  `yaml:"user_agent" json:"user_agent"`
}

// TrackingMessageActivity is the response of GET /v3/activity/tracking/messages/{message_id}.
type TrackingMessageActivity struct {
	MessageID     string                `yaml:"message_id" json:"message_id"`
	ThreadID      string                `yaml:"thread_id" json:"thread_id"`
	GrantID       string                `yaml:"grant_id" json:"grant_id"`
	From          string                `yaml:"from" json:"from"`
	SendingDomain string                `yaml:"sending_domain" json:"sending_domain"`
	Label         string                `yaml:"label" json:"label"`
	Tracking      TrackingFlags         `yaml:"tracking" json:"tracking"`
	Opens         TrackingTotals        `yaml:"opens" json:"opens"`
	Clicks        TrackingClickTotals   `yaml:"clicks" json:"clicks"`
	Events        []TrackingInteraction `yaml:"events" json:"events"`
}
