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
	SMTPResponse     string `json:"smtp_response,omitempty"`
	ProcessingTimeMs int64  `json:"processing_time_ms,omitempty"`
	ReportingMTA     string `json:"reporting_mta,omitempty"`
	RemoteMTAIP      string `json:"remote_mta_ip,omitempty"`
	BounceType       string `json:"bounce_type,omitempty"`
	BounceSubtype    string `json:"bounce_subtype,omitempty"`
	Status           string `json:"status,omitempty"`
	Action           string `json:"action,omitempty"`
	DiagnosticCode   string `json:"diagnostic_code,omitempty"`
	FeedbackType     string `json:"feedback_type,omitempty"`
	ComplaintSubtype string `json:"complaint_subtype,omitempty"`
}

// SendEvent is one row of the send activity feed.
type SendEvent struct {
	ID              string           `json:"id"`
	EventType       string           `json:"event_type"`
	BounceClass     string           `json:"bounce_class,omitempty"`
	Timestamp       int64            `json:"timestamp"`
	ApplicationID   string           `json:"application_id"`
	GrantID         string           `json:"grant_id"`
	MessageID       string           `json:"message_id"`
	ThreadID        string           `json:"thread_id"`
	From            string           `json:"from"`
	SendingDomain   string           `json:"sending_domain"`
	Recipient       string           `json:"recipient"`
	RecipientDomain string           `json:"recipient_domain"`
	Subject         *string          `json:"subject,omitempty"`
	Reason          string           `json:"reason"`
	Detail          *SendEventDetail `json:"detail,omitempty"`
}

// QuietField returns the event ID for quiet output.
func (e SendEvent) QuietField() string { return e.ID }

// SendEventListResponse is one page of the send activity feed.
type SendEventListResponse struct {
	Data       []SendEvent `json:"data"`
	NextCursor string      `json:"next_cursor,omitempty"`
	RequestID  string      `json:"request_id,omitempty"`
}

// SendMessageEnvelope describes the message in a per-message send view.
type SendMessageEnvelope struct {
	MessageID     string  `json:"message_id"`
	ThreadID      string  `json:"thread_id"`
	GrantID       string  `json:"grant_id"`
	ApplicationID string  `json:"application_id"`
	From          string  `json:"from"`
	SendingDomain string  `json:"sending_domain"`
	Subject       *string `json:"subject,omitempty"`
	SentAt        int64   `json:"sent_at"`
	LastEventAt   int64   `json:"last_event_at"`
}

// SendTimelineEvent is one event in a recipient's timeline.
type SendTimelineEvent struct {
	ID          string           `json:"id"`
	EventType   string           `json:"event_type"`
	Timestamp   int64            `json:"timestamp"`
	BounceClass string           `json:"bounce_class,omitempty"`
	Reason      string           `json:"reason"`
	Detail      *SendEventDetail `json:"detail,omitempty"`
}

// SendMessageRecipient is one recipient's status and timeline.
type SendMessageRecipient struct {
	Recipient       string              `json:"recipient"`
	RecipientDomain string              `json:"recipient_domain"`
	CurrentStatus   string              `json:"current_status"`
	BounceClass     string              `json:"bounce_class,omitempty"`
	Events          []SendTimelineEvent `json:"events"`
}

// SendMessageActivity is the response of GET /v3/activity/send/messages/{message_id}.
type SendMessageActivity struct {
	Message    SendMessageEnvelope    `json:"message"`
	Recipients []SendMessageRecipient `json:"recipients"`
}

// TrackingEvent is one row of the tracking activity feed.
type TrackingEvent struct {
	ID            string  `json:"id"`
	EventType     string  `json:"event_type"`
	Timestamp     int64   `json:"timestamp"`
	ApplicationID string  `json:"application_id"`
	GrantID       string  `json:"grant_id"`
	MessageID     string  `json:"message_id"`
	ThreadID      string  `json:"thread_id"`
	From          string  `json:"from"`
	SendingDomain string  `json:"sending_domain"`
	Label         string  `json:"label"`
	IsFirst       bool    `json:"is_first"`
	URL           *string `json:"url,omitempty"`
	LinkIndex     *int64  `json:"link_index,omitempty"`
	IP            string  `json:"ip"`
	UserAgent     string  `json:"user_agent"`
}

// QuietField returns the event ID for quiet output.
func (e TrackingEvent) QuietField() string { return e.ID }

// TrackingEventListResponse is one page of the tracking activity feed.
type TrackingEventListResponse struct {
	Data       []TrackingEvent `json:"data"`
	NextCursor string          `json:"next_cursor,omitempty"`
	RequestID  string          `json:"request_id,omitempty"`
}

// TrackingFlags reports which tracking was enabled for a message.
type TrackingFlags struct {
	Opens bool `json:"opens"`
	Links bool `json:"links"`
}

// TrackingTotals counts interactions; FirstAt and LastAt are nil when Total is 0.
type TrackingTotals struct {
	Total   int64  `json:"total"`
	FirstAt *int64 `json:"first_at"`
	LastAt  *int64 `json:"last_at"`
}

// TrackingLinkClicks counts clicks on one tracked link.
type TrackingLinkClicks struct {
	LinkIndex int64  `json:"link_index"`
	URL       string `json:"url"`
	Clicks    int64  `json:"clicks"`
}

// TrackingClickTotals counts clicks overall and per link.
type TrackingClickTotals struct {
	TrackingTotals
	Links []TrackingLinkClicks `json:"links"`
}

// TrackingInteraction is one open or click in a per-message tracking view.
type TrackingInteraction struct {
	ID        string  `json:"id"`
	EventType string  `json:"event_type"`
	Timestamp int64   `json:"timestamp"`
	IsFirst   bool    `json:"is_first"`
	URL       *string `json:"url,omitempty"`
	LinkIndex *int64  `json:"link_index,omitempty"`
	IP        string  `json:"ip"`
	UserAgent string  `json:"user_agent"`
}

// TrackingMessageActivity is the response of GET /v3/activity/tracking/messages/{message_id}.
type TrackingMessageActivity struct {
	MessageID     string                `json:"message_id"`
	ThreadID      string                `json:"thread_id"`
	GrantID       string                `json:"grant_id"`
	From          string                `json:"from"`
	SendingDomain string                `json:"sending_domain"`
	Label         string                `json:"label"`
	Tracking      TrackingFlags         `json:"tracking"`
	Opens         TrackingTotals        `json:"opens"`
	Clicks        TrackingClickTotals   `json:"clicks"`
	Events        []TrackingInteraction `json:"events"`
}
