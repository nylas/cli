package domain

import "time"

// MeetingPattern represents learned patterns from calendar history.
type MeetingPattern struct {
	UserEmail      string                        `json:"user_email"`
	AnalyzedPeriod DateRange                     `json:"analyzed_period"`
	LastUpdated    time.Time                     `json:"last_updated"`
	Acceptance     AcceptancePatterns            `json:"acceptance"`
	Duration       DurationPatterns              `json:"duration"`
	Timezone       TimezonePatterns              `json:"timezone"`
	Productivity   ProductivityPatterns          `json:"productivity"`
	Participants   map[string]ParticipantPattern `json:"participants"`
}

// AcceptancePatterns tracks meeting acceptance rates.
type AcceptancePatterns struct {
	ByDayOfWeek  map[string]float64 `json:"by_day_of_week"`  // Monday -> 0.92
	ByTimeOfDay  map[string]float64 `json:"by_time_of_day"`  // "09:00" -> 0.85
	ByDayAndTime map[string]float64 `json:"by_day_and_time"` // "Monday-09:00" -> 0.95
	Overall      float64            `json:"overall"`         // Overall acceptance rate
}

// DurationPatterns tracks actual vs scheduled meeting durations.
type DurationPatterns struct {
	ByParticipant map[string]DurationStats `json:"by_participant"`
	ByType        map[string]DurationStats `json:"by_type"` // "1-on-1", "team", "client"
	Overall       DurationStats            `json:"overall"`
}

// DurationStats contains duration statistics.
type DurationStats struct {
	AverageScheduled int     `json:"average_scheduled"` // Minutes
	AverageActual    int     `json:"average_actual"`    // Minutes
	Variance         float64 `json:"variance"`          // Standard deviation
	OverrunRate      float64 `json:"overrun_rate"`      // % of meetings that run over
}

// TimezonePatterns tracks timezone preferences.
type TimezonePatterns struct {
	PreferredTimes map[string][]string `json:"preferred_times"` // Timezone -> preferred hours
	Distribution   map[string]int      `json:"distribution"`    // Timezone -> count
	CrossTZTimes   []string            `json:"cross_tz_times"`  // Preferred times for cross-TZ meetings
}

// ProductivityPatterns tracks productive time blocks.
type ProductivityPatterns struct {
	PeakFocus      []TimeBlock        `json:"peak_focus"`      // Best focus time blocks
	LowEnergy      []TimeBlock        `json:"low_energy"`      // Low productivity times
	MeetingDensity map[string]float64 `json:"meeting_density"` // DayOfWeek -> avg meetings per day
	FocusBlocks    []TimeBlock        `json:"focus_blocks"`    // Recommended focus time blocks
}

// TimeBlock represents a recurring time block.
type TimeBlock struct {
	DayOfWeek string  `json:"day_of_week"` // "Monday", "Tuesday", etc.
	StartTime string  `json:"start_time"`  // "09:00"
	EndTime   string  `json:"end_time"`    // "11:00"
	Score     float64 `json:"score"`       // Productivity score 0-100
}

// ParticipantPattern tracks patterns for specific participants.
type ParticipantPattern struct {
	Email           string   `json:"email"`
	MeetingCount    int      `json:"meeting_count"`
	AcceptanceRate  float64  `json:"acceptance_rate"`
	PreferredDays   []string `json:"preferred_days"`
	PreferredTimes  []string `json:"preferred_times"`
	AverageDuration int      `json:"average_duration"` // Minutes
	Timezone        string   `json:"timezone"`
}

// MeetingAnalysis represents the analysis of historical meetings.
type MeetingAnalysis struct {
	Period          DateRange        `json:"period"`
	TotalMeetings   int              `json:"total_meetings"`
	Patterns        *MeetingPattern  `json:"patterns"`
	Recommendations []Recommendation `json:"recommendations"`
	Insights        []string         `json:"insights"`
}

// Recommendation represents an AI-generated recommendation.
type Recommendation struct {
	Type        string  `json:"type"`     // "focus_time", "decline_pattern", "duration_adjustment"
	Priority    string  `json:"priority"` // "high", "medium", "low"
	Title       string  `json:"title"`
	Description string  `json:"description"`
	Confidence  float64 `json:"confidence"` // 0-100
	Action      string  `json:"action"`     // Suggested action
	Impact      string  `json:"impact"`     // Expected impact
}

// MeetingScore represents the predicted value/success of a meeting.
type MeetingScore struct {
	Score            int           `json:"score"`        // 0-100
	Confidence       float64       `json:"confidence"`   // 0-100
	SuccessRate      float64       `json:"success_rate"` // Historical success rate
	Factors          []ScoreFactor `json:"factors"`      // Contributing factors
	Recommendation   string        `json:"recommendation"`
	AlternativeTimes []time.Time   `json:"alternative_times,omitempty"`
}

// ScoreFactor represents a factor contributing to the meeting score.
type ScoreFactor struct {
	Name        string `json:"name"`
	Impact      int    `json:"impact"` // -100 to +100
	Description string `json:"description"`
}

// FocusTimeBlock represents a suggested focus time block.
type FocusTimeBlock struct {
	DayOfWeek string  `json:"day_of_week"`
	StartTime string  `json:"start_time"`
	EndTime   string  `json:"end_time"`
	Duration  int     `json:"duration"` // Minutes
	Score     float64 `json:"score"`    // Productivity score
	Reason    string  `json:"reason"`
	Conflicts int     `json:"conflicts"` // Number of meetings that would conflict
}

// ============================================================================
// Conflict Detection & Resolution
// ============================================================================

// ConflictType represents the type of scheduling conflict.
type ConflictType string

const (
	ConflictTypeHard            ConflictType = "hard"              // Overlapping times
	ConflictTypeSoftBackToBack  ConflictType = "soft_back_to_back" // No buffer time
	ConflictTypeSoftFocusTime   ConflictType = "soft_focus_time"   // Interrupts focus time
	ConflictTypeSoftTravelTime  ConflictType = "soft_travel_time"  // Insufficient travel time
	ConflictTypeSoftOverload    ConflictType = "soft_overload"     // Too many meetings
	ConflictTypeSoftLowPriority ConflictType = "soft_low_priority" // Low-value meeting
)

// ConflictSeverity represents how severe a conflict is.
type ConflictSeverity string

const (
	SeverityCritical ConflictSeverity = "critical" // Must resolve
	SeverityHigh     ConflictSeverity = "high"     // Should resolve
	SeverityMedium   ConflictSeverity = "medium"   // Consider resolving
	SeverityLow      ConflictSeverity = "low"      // Optional to resolve
)

// Conflict represents a detected scheduling conflict.
type Conflict struct {
	ID               string           `json:"id"`
	Type             ConflictType     `json:"type"`
	Severity         ConflictSeverity `json:"severity"`
	ProposedEvent    *Event           `json:"proposed_event"`
	ConflictingEvent *Event           `json:"conflicting_event,omitempty"`
	Description      string           `json:"description"`
	Impact           string           `json:"impact"`
	Suggestion       string           `json:"suggestion"`
	CanAutoResolve   bool             `json:"can_auto_resolve"`
}

// ConflictAnalysis represents the result of conflict detection.
type ConflictAnalysis struct {
	ProposedEvent    *Event             `json:"proposed_event"`
	HardConflicts    []Conflict         `json:"hard_conflicts"`
	SoftConflicts    []Conflict         `json:"soft_conflicts"`
	TotalConflicts   int                `json:"total_conflicts"`
	CanProceed       bool               `json:"can_proceed"`
	Recommendations  []string           `json:"recommendations"`
	AlternativeTimes []RescheduleOption `json:"alternative_times,omitempty"`
	AIRecommendation string             `json:"ai_recommendation"`
}

// RescheduleOption represents an alternative time for rescheduling.
type RescheduleOption struct {
	ProposedTime     time.Time  `json:"proposed_time"`
	EndTime          time.Time  `json:"end_time"`
	Score            int        `json:"score"` // 0-100
	Confidence       float64    `json:"confidence"`
	Pros             []string   `json:"pros"`
	Cons             []string   `json:"cons"`
	Conflicts        []Conflict `json:"conflicts"`         // Any remaining conflicts
	ParticipantMatch float64    `json:"participant_match"` // % of participants available
	AIInsight        string     `json:"ai_insight"`
}

// RescheduleRequest represents a request to reschedule a meeting.
type RescheduleRequest struct {
	EventID            string      `json:"event_id"`
	Reason             string      `json:"reason"`
	PreferredTimes     []time.Time `json:"preferred_times,omitempty"`
	MustInclude        []string    `json:"must_include,omitempty"` // Participant emails
	AvoidDays          []string    `json:"avoid_days,omitempty"`
	MinNoticeDays      int         `json:"min_notice_days"`
	MaxDelayDays       int         `json:"max_delay_days"`
	NotifyParticipants bool        `json:"notify_participants"`
}

// RescheduleResult represents the result of a rescheduling operation.
type RescheduleResult struct {
	Success           bool              `json:"success"`
	OriginalEvent     *Event            `json:"original_event"`
	NewEvent          *Event            `json:"new_event,omitempty"`
	SelectedOption    *RescheduleOption `json:"selected_option,omitempty"`
	NotificationsSent int               `json:"notifications_sent"`
	Message           string            `json:"message"`
	CascadingChanges  []string          `json:"cascading_changes,omitempty"`
}

// MeetingPriority represents the priority level of a meeting.
type MeetingPriority string

const (
	PriorityCritical MeetingPriority = "critical" // Cannot be moved
	PriorityHigh     MeetingPriority = "high"     // Hard to move
	PriorityMedium   MeetingPriority = "medium"   // Can be moved
	PriorityLow      MeetingPriority = "low"      // Easy to move
	PriorityFlexible MeetingPriority = "flexible" // Very flexible
)

// MeetingMetadata contains learned metadata about a meeting.
type MeetingMetadata struct {
	EventID           string          `json:"event_id"`
	Priority          MeetingPriority `json:"priority"`
	IsRecurring       bool            `json:"is_recurring"`
	ParticipantCount  int             `json:"participant_count"`
	HistoricalMoves   int             `json:"historical_moves"` // Times this meeting was rescheduled
	LastMoved         time.Time       `json:"last_moved,omitempty"`
	DeclineRate       float64         `json:"decline_rate"`
	AvgRescheduleLead int             `json:"avg_reschedule_lead"` // Days notice for rescheduling
}

// ============================================================================
// Focus Time Protection (Task 3.4)
// ============================================================================

// FocusTimeSettings represents user preferences for focus time protection.
// Field order optimized for memory alignment (8-byte fields first, bools grouped at end).
type FocusTimeSettings struct {
	// 8-byte aligned fields
	TargetHoursPerWeek   float64                    `json:"target_hours_per_week"` // Target focus hours per week
	MinBlockDuration     int                        `json:"min_block_duration"`    // Minimum focus block minutes
	MaxBlockDuration     int                        `json:"max_block_duration"`    // Maximum focus block minutes
	ProtectedDays        []string                   `json:"protected_days"`        // Days to protect (e.g., "Wednesday")
	ExcludedTimeRanges   []TimeRange                `json:"excluded_time_ranges"`  // Times to exclude from protection
	NotificationSettings FocusTimeNotificationPrefs `json:"notification_settings"`
	// Bool fields grouped to minimize padding
	Enabled             bool `json:"enabled"`
	AutoBlock           bool `json:"auto_block"`            // Auto-create focus blocks
	AutoDecline         bool `json:"auto_decline"`          // Auto-decline meeting requests
	AllowUrgentOverride bool `json:"allow_urgent_override"` // Allow override for urgent meetings
	RequireApproval     bool `json:"require_approval"`      // Require approval for overrides
}

// TimeRange represents a time range within a day.
type TimeRange struct {
	StartTime string `json:"start_time"` // "09:00"
	EndTime   string `json:"end_time"`   // "17:00"
}

// FocusTimeNotificationPrefs represents notification preferences.
type FocusTimeNotificationPrefs struct {
	NotifyOnDecline    bool `json:"notify_on_decline"`    // Notify when declining meetings
	NotifyOnOverride   bool `json:"notify_on_override"`   // Notify when override requested
	NotifyOnAdaptation bool `json:"notify_on_adaptation"` // Notify on adaptive changes
	DailySummary       bool `json:"daily_summary"`        // Send daily focus time summary
	WeeklySummary      bool `json:"weekly_summary"`       // Send weekly focus time summary
}

// FocusTimeAnalysis represents the analysis of productivity patterns for focus time.
type FocusTimeAnalysis struct {
	UserEmail          string           `json:"user_email"`
	AnalyzedPeriod     DateRange        `json:"analyzed_period"`
	GeneratedAt        time.Time        `json:"generated_at"`
	PeakProductivity   []TimeBlock      `json:"peak_productivity"`    // Peak focus time blocks
	DeepWorkSessions   DurationStats    `json:"deep_work_sessions"`   // Deep work session stats
	MostProductiveDay  string           `json:"most_productive_day"`  // Best day for deep work
	LeastProductiveDay string           `json:"least_productive_day"` // Worst day for deep work
	RecommendedBlocks  []FocusTimeBlock `json:"recommended_blocks"`   // AI-recommended focus blocks
	CurrentProtection  float64          `json:"current_protection"`   // Current hours/week protected
	TargetProtection   float64          `json:"target_protection"`    // Target hours/week to protect
	Insights           []string         `json:"insights"`             // AI insights about focus patterns
	Confidence         float64          `json:"confidence"`           // Confidence in recommendations (0-100)
}

// ProtectedBlock represents an active focus time block on the calendar.
// Field order optimized for memory alignment (8-byte fields first, bools grouped at end).
type ProtectedBlock struct {
	// 8-byte aligned fields
	ID                string              `json:"id"`
	CalendarEventID   string              `json:"calendar_event_id,omitempty"` // Linked calendar event
	StartTime         time.Time           `json:"start_time"`
	EndTime           time.Time           `json:"end_time"`
	CreatedAt         time.Time           `json:"created_at"`
	UpdatedAt         time.Time           `json:"updated_at"`
	RecurrencePattern string              `json:"recurrence_pattern,omitempty"` // e.g., "weekly"
	Priority          MeetingPriority     `json:"priority"`
	Reason            string              `json:"reason"` // Why this block is protected
	OverrideReason    string              `json:"override_reason,omitempty"`
	ProtectionRules   FocusProtectionRule `json:"protection_rules"`
	Duration          int                 `json:"duration"` // Minutes
	// Bool fields grouped to minimize padding
	IsRecurring      bool `json:"is_recurring"`
	AllowOverride    bool `json:"allow_override"`    // Can be overridden
	OverrideApproved bool `json:"override_approved"` // Override was approved
}

// FocusProtectionRule defines how a focus block is protected.
// Field order optimized for memory alignment.
type FocusProtectionRule struct {
	// 8-byte aligned fields
	DeclineMessage   string   `json:"decline_message"`   // Custom decline message
	AlternativeTimes []string `json:"alternative_times"` // Suggested alternative time slots
	// Bool fields grouped to minimize padding
	AutoDecline          bool `json:"auto_decline"`           // Auto-decline meeting requests
	SuggestAlternatives  bool `json:"suggest_alternatives"`   // Suggest alternative times
	AllowCriticalMeeting bool `json:"allow_critical_meeting"` // Allow critical priority meetings
	RequireApproval      bool `json:"require_approval"`       // Require manual approval
}

// OverrideRequest represents a request to override a protected focus block.
type OverrideRequest struct {
	ID                    string          `json:"id"`
	ProtectedBlockID      string          `json:"protected_block_id"`
	MeetingRequest        *Event          `json:"meeting_request"`
	RequestedBy           string          `json:"requested_by"`
	Reason                string          `json:"reason"`
	Priority              MeetingPriority `json:"priority"`
	IsUrgent              bool            `json:"is_urgent"`
	ApprovalStatus        ApprovalStatus  `json:"approval_status"`
	ApprovedBy            string          `json:"approved_by,omitempty"`
	ApprovedAt            time.Time       `json:"approved_at,omitempty"`
	AlternativesSuggested []time.Time     `json:"alternatives_suggested,omitempty"`
	CreatedAt             time.Time       `json:"created_at"`
}

// ApprovalStatus represents the status of an override approval.
type ApprovalStatus string

const (
	ApprovalPending  ApprovalStatus = "pending"
	ApprovalApproved ApprovalStatus = "approved"
	ApprovalDenied   ApprovalStatus = "denied"
	ApprovalExpired  ApprovalStatus = "expired"
)

// AdaptiveScheduleChange represents a change made by adaptive scheduling.
type AdaptiveScheduleChange struct {
	ID             string                 `json:"id"`
	Timestamp      time.Time              `json:"timestamp"`
	Trigger        AdaptiveTrigger        `json:"trigger"`
	ChangeType     AdaptiveChangeType     `json:"change_type"`
	AffectedEvents []string               `json:"affected_events"` // Event IDs
	Changes        []ScheduleModification `json:"changes"`
	Reason         string                 `json:"reason"`
	Impact         AdaptiveImpact         `json:"impact"`
	UserApproval   ApprovalStatus         `json:"user_approval"`
	AutoApplied    bool                   `json:"auto_applied"`
	Confidence     float64                `json:"confidence"` // 0-100
}

// AdaptiveTrigger represents what triggered the adaptive scheduling.
type AdaptiveTrigger string

const (
	TriggerDeadlineChange   AdaptiveTrigger = "deadline_change"    // Project deadline changed
	TriggerMeetingOverload  AdaptiveTrigger = "meeting_overload"   // Too many meetings scheduled
	TriggerPriorityShift    AdaptiveTrigger = "priority_shift"     // Priority changed
	TriggerFocusTimeAtRisk  AdaptiveTrigger = "focus_time_at_risk" // Focus time being eroded
	TriggerConflictDetected AdaptiveTrigger = "conflict_detected"  // Schedule conflict
	TriggerPatternDetected  AdaptiveTrigger = "pattern_detected"   // Pattern learned
)

// AdaptiveChangeType represents the type of adaptive change.
type AdaptiveChangeType string

const (
	ChangeTypeIncreaseFocusTime AdaptiveChangeType = "increase_focus_time" // Add more focus blocks
	ChangeTypeRescheduleMeeting AdaptiveChangeType = "reschedule_meeting"  // Move meeting
	ChangeTypeShortenMeeting    AdaptiveChangeType = "shorten_meeting"     // Reduce duration
	ChangeTypeDeclineMeeting    AdaptiveChangeType = "decline_meeting"     // Decline meeting
	ChangeTypeMoveMeetingLater  AdaptiveChangeType = "move_meeting_later"  // Postpone meeting
	ChangeTypeProtectBlock      AdaptiveChangeType = "protect_block"       // Add focus protection
)

// ScheduleModification represents a specific schedule modification.
type ScheduleModification struct {
	EventID      string    `json:"event_id"`
	Action       string    `json:"action"` // "reschedule", "shorten", "decline", "protect"
	OldStartTime time.Time `json:"old_start_time,omitempty"`
	NewStartTime time.Time `json:"new_start_time,omitempty"`
	OldDuration  int       `json:"old_duration,omitempty"` // Minutes
	NewDuration  int       `json:"new_duration,omitempty"` // Minutes
	Description  string    `json:"description"`
}

// AdaptiveImpact represents the impact of adaptive scheduling changes.
type AdaptiveImpact struct {
	FocusTimeGained      float64  `json:"focus_time_gained"` // Hours gained
	MeetingsRescheduled  int      `json:"meetings_rescheduled"`
	MeetingsDeclined     int      `json:"meetings_declined"`
	DurationSaved        int      `json:"duration_saved"` // Minutes saved
	ConflictsResolved    int      `json:"conflicts_resolved"`
	ParticipantsAffected int      `json:"participants_affected"`
	PredictedBenefit     string   `json:"predicted_benefit"`
	Risks                []string `json:"risks,omitempty"`
}

// DurationOptimization represents meeting duration optimization recommendations.
type DurationOptimization struct {
	EventID             string        `json:"event_id"`
	CurrentDuration     int           `json:"current_duration"`     // Minutes
	RecommendedDuration int           `json:"recommended_duration"` // Minutes
	HistoricalData      DurationStats `json:"historical_data"`
	TimeSavings         int           `json:"time_savings"` // Minutes saved
	Confidence          float64       `json:"confidence"`   // 0-100
	Reason              string        `json:"reason"`
	Recommendation      string        `json:"recommendation"`
}
