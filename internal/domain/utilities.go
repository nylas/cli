package domain

import "time"

// ============================================================================
// Time Zone Models
// ============================================================================

// MeetingFinderRequest contains parameters for finding meeting times across zones.
type MeetingFinderRequest struct {
	TimeZones         []string      `json:"time_zones"`
	Duration          time.Duration `json:"duration"`
	WorkingHoursStart string        `json:"working_hours_start"` // "09:00"
	WorkingHoursEnd   string        `json:"working_hours_end"`   // "17:00"
	DateRange         DateRange     `json:"date_range"`
	ExcludeWeekends   bool          `json:"exclude_weekends"`
}

// DateRange represents a range of dates.
type DateRange struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

// MeetingTimeSlots represents available meeting times across zones.
type MeetingTimeSlots struct {
	Slots      []MeetingSlot `json:"slots"`
	TimeZones  []string      `json:"time_zones"`
	TotalSlots int           `json:"total_slots"`
}

// MeetingSlot represents a potential meeting time across zones.
type MeetingSlot struct {
	StartTime time.Time            `json:"start_time"`
	EndTime   time.Time            `json:"end_time"`
	Times     map[string]time.Time `json:"times"` // zone -> local time
	Score     float64              `json:"score"` // Quality score (0-1)
}

// DSTTransition represents a DST change.
type DSTTransition struct {
	Date      time.Time `json:"date"`
	Offset    int       `json:"offset"`    // Seconds from UTC
	Name      string    `json:"name"`      // "PDT", "PST", etc.
	IsDST     bool      `json:"is_dst"`    // Whether this is DST or standard time
	Direction string    `json:"direction"` // "forward" (spring) or "backward" (fall)
}

// TimeZoneInfo provides detailed information about a time zone.
type TimeZoneInfo struct {
	Name         string     `json:"name"`               // IANA name (e.g., "America/Los_Angeles")
	Abbreviation string     `json:"abbreviation"`       // Current abbreviation (e.g., "PST", "PDT")
	Offset       int        `json:"offset"`             // Current offset from UTC in seconds
	IsDST        bool       `json:"is_dst"`             // Whether currently observing DST
	NextDST      *time.Time `json:"next_dst,omitempty"` // Next DST transition
}

// DSTWarning provides warning information about DST transitions.
// Used to alert users when scheduling events near or during DST changes.
type DSTWarning struct {
	IsNearTransition bool      `json:"is_near_transition"` // True if within warning window
	TransitionDate   time.Time `json:"transition_date"`    // When the DST change occurs
	Direction        string    `json:"direction"`          // "forward" (spring) or "backward" (fall)
	DaysUntil        int       `json:"days_until"`         // Days until transition (negative if past)
	TransitionName   string    `json:"transition_name"`    // Timezone name at transition (e.g., "PDT")
	InTransitionGap  bool      `json:"in_transition_gap"`  // True if time falls in spring forward gap
	InDuplicateHour  bool      `json:"in_duplicate_hour"`  // True if time occurs twice (fall back)
	Warning          string    `json:"warning"`            // User-facing warning message
	Severity         string    `json:"severity"`           // "error", "warning", or "info"
}

// ============================================================================
// Webhook Models
// ============================================================================

// WebhookServerConfig contains configuration for local webhook server.
type WebhookServerConfig struct {
	Port              int               `json:"port"`
	Host              string            `json:"host"`
	PersistentURL     string            `json:"persistent_url,omitempty"`
	SaveToFile        bool              `json:"save_to_file"`
	FilePath          string            `json:"file_path,omitempty"`
	ValidateSignature bool              `json:"validate_signature"`
	Secret            string            `json:"secret,omitempty"`
	Headers           map[string]string `json:"headers,omitempty"`
}

// WebhookPayload represents a captured webhook.
type WebhookPayload struct {
	ID        string            `json:"id"`
	Timestamp time.Time         `json:"timestamp"`
	Method    string            `json:"method"`
	URL       string            `json:"url"`
	Headers   map[string]string `json:"headers"`
	Body      []byte            `json:"body"`
	Signature string            `json:"signature,omitempty"`
	Verified  bool              `json:"verified"`
}

// ============================================================================
// Email Utility Models
// ============================================================================

// TemplateRequest contains parameters for building email templates.
type TemplateRequest struct {
	Name      string            `json:"name"`
	Subject   string            `json:"subject"`
	HTMLBody  string            `json:"html_body"`
	TextBody  string            `json:"text_body,omitempty"`
	Variables []string          `json:"variables"`
	InlineCSS bool              `json:"inline_css"`
	Sanitize  bool              `json:"sanitize"`
	Metadata  map[string]string `json:"metadata,omitempty"`
}

// EmailTemplate represents an email template.
type EmailTemplate struct {
	ID        string            `json:"id"`
	Name      string            `json:"name"`
	Subject   string            `json:"subject"`
	HTMLBody  string            `json:"html_body"`
	TextBody  string            `json:"text_body,omitempty"`
	Variables []string          `json:"variables"`
	CreatedAt time.Time         `json:"created_at"`
	UpdatedAt time.Time         `json:"updated_at"`
	Metadata  map[string]string `json:"metadata,omitempty"`
}

// DeliverabilityReport contains email deliverability analysis.
type DeliverabilityReport struct {
	Score           int                   `json:"score"` // 0-100
	Issues          []DeliverabilityIssue `json:"issues"`
	SPFStatus       string                `json:"spf_status"`
	DKIMStatus      string                `json:"dkim_status"`
	DMARCStatus     string                `json:"dmarc_status"`
	SpamScore       float64               `json:"spam_score"`
	MobileOptimized bool                  `json:"mobile_optimized"`
	Recommendations []string              `json:"recommendations"`
}

// DeliverabilityIssue represents a specific deliverability problem.
type DeliverabilityIssue struct {
	Severity string `json:"severity"` // "critical", "warning", "info"
	Category string `json:"category"` // "authentication", "content", "formatting", etc.
	Message  string `json:"message"`
	Fix      string `json:"fix,omitempty"`
}

// ParsedEmail represents a parsed .eml file.
type ParsedEmail struct {
	Headers     map[string]string `json:"headers"`
	From        string            `json:"from"`
	To          []string          `json:"to"`
	Cc          []string          `json:"cc,omitempty"`
	Bcc         []string          `json:"bcc,omitempty"`
	Subject     string            `json:"subject"`
	Date        time.Time         `json:"date"`
	HTMLBody    string            `json:"html_body,omitempty"`
	TextBody    string            `json:"text_body,omitempty"`
	Attachments []Attachment      `json:"attachments,omitempty"`
}

// EmailMessage represents an email message for generation.
type EmailMessage struct {
	From        string            `json:"from"`
	To          []string          `json:"to"`
	Cc          []string          `json:"cc,omitempty"`
	Bcc         []string          `json:"bcc,omitempty"`
	Subject     string            `json:"subject"`
	HTMLBody    string            `json:"html_body,omitempty"`
	TextBody    string            `json:"text_body,omitempty"`
	Headers     map[string]string `json:"headers,omitempty"`
	Attachments []Attachment      `json:"attachments,omitempty"`
}

// EmailValidation contains email address validation results.
type EmailValidation struct {
	Email       string `json:"email"`
	Valid       bool   `json:"valid"`
	FormatValid bool   `json:"format_valid"`
	MXExists    bool   `json:"mx_exists"`
	Disposable  bool   `json:"disposable"`
	Suggestion  string `json:"suggestion,omitempty"` // Did you mean...?
}

// SpamAnalysis contains spam score analysis.
type SpamAnalysis struct {
	Score       float64       `json:"score"` // 0-10 (lower is better)
	IsSpam      bool          `json:"is_spam"`
	Triggers    []SpamTrigger `json:"triggers"`
	Passed      []string      `json:"passed"`
	Suggestions []string      `json:"suggestions"`
}

// SpamTrigger represents a spam filter trigger.
type SpamTrigger struct {
	Rule        string  `json:"rule"`
	Description string  `json:"description"`
	Score       float64 `json:"score"`
	Severity    string  `json:"severity"` // "high", "medium", "low"
}

// ============================================================================
// Contact Utility Models
// ============================================================================

// DeduplicationRequest contains parameters for contact deduplication.
type DeduplicationRequest struct {
	Contacts       []Contact `json:"contacts"`
	FuzzyThreshold float64   `json:"fuzzy_threshold"` // 0.0-1.0 (similarity threshold)
	MatchFields    []string  `json:"match_fields"`    // Fields to compare (email, phone, name)
	AutoMerge      bool      `json:"auto_merge"`      // Automatically merge duplicates
	MergeStrategy  string    `json:"merge_strategy"`  // "newest", "oldest", "most_complete"
}

// DeduplicationResult contains deduplication results.
type DeduplicationResult struct {
	OriginalCount     int              `json:"original_count"`
	DeduplicatedCount int              `json:"deduplicated_count"`
	DuplicateGroups   []DuplicateGroup `json:"duplicate_groups"`
	MergedContacts    []Contact        `json:"merged_contacts,omitempty"`
}

// DuplicateGroup represents a group of duplicate contacts.
type DuplicateGroup struct {
	Contacts      []Contact `json:"contacts"`
	MatchScore    float64   `json:"match_score"` // 0.0-1.0
	MatchedFields []string  `json:"matched_fields"`
	Suggested     *Contact  `json:"suggested,omitempty"` // Suggested merge result
}
