package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/nylas/cli/internal/domain"
	"github.com/nylas/cli/internal/ports"
)

// PatternLearner learns from calendar history to predict scheduling patterns.
type PatternLearner struct {
	nylasClient ports.NylasClient
	llmRouter   ports.LLMRouter
}

// NewPatternLearner creates a new pattern learner.
func NewPatternLearner(nylasClient ports.NylasClient, llmRouter ports.LLMRouter) *PatternLearner {
	return &PatternLearner{
		nylasClient: nylasClient,
		llmRouter:   llmRouter,
	}
}

// SchedulingPatterns represents discovered patterns from calendar history.
type SchedulingPatterns struct {
	UserID               string                `json:"user_id"`
	AnalysisPeriod       AnalysisPeriod        `json:"analysis_period"`
	AcceptancePatterns   []AcceptancePattern   `json:"acceptance_patterns"`
	DurationPatterns     []DurationPattern     `json:"duration_patterns"`
	TimezonePatterns     []TimezonePattern     `json:"timezone_patterns"`
	ProductivityInsights []ProductivityInsight `json:"productivity_insights"`
	Recommendations      []string              `json:"recommendations"`
	TotalEventsAnalyzed  int                   `json:"total_events_analyzed"`
	GeneratedAt          time.Time             `json:"generated_at"`
}

// AnalysisPeriod defines the time period analyzed.
type AnalysisPeriod struct {
	StartDate time.Time `json:"start_date"`
	EndDate   time.Time `json:"end_date"`
	Days      int       `json:"days"`
}

// AcceptancePattern represents meeting acceptance rates by time/day.
type AcceptancePattern struct {
	TimeSlot    string  `json:"time_slot"`   // e.g., "Monday 9-11 AM"
	AcceptRate  float64 `json:"accept_rate"` // 0-1
	EventCount  int     `json:"event_count"` // Number of events in this slot
	Description string  `json:"description"` // Human-readable explanation
	Confidence  float64 `json:"confidence"`  // 0-1, based on sample size
}

// DurationPattern represents typical meeting duration patterns.
type DurationPattern struct {
	MeetingType       string `json:"meeting_type"`       // e.g., "1-on-1", "Team standup"
	ScheduledDuration int    `json:"scheduled_duration"` // In minutes
	ActualDuration    int    `json:"actual_duration"`    // In minutes
	Variance          int    `json:"variance"`           // Difference
	EventCount        int    `json:"event_count"`        // Sample size
	Description       string `json:"description"`        // Pattern description
}

// TimezonePattern represents timezone preferences.
type TimezonePattern struct {
	Timezone      string  `json:"timezone"`       // e.g., "America/New_York"
	EventCount    int     `json:"event_count"`    // Number of events
	Percentage    float64 `json:"percentage"`     // % of total events
	PreferredTime string  `json:"preferred_time"` // e.g., "2-4 PM PST"
	Description   string  `json:"description"`    // Pattern description
}

// ProductivityInsight represents productivity patterns.
type ProductivityInsight struct {
	InsightType string   `json:"insight_type"` // e.g., "peak_focus", "low_energy"
	TimeSlot    string   `json:"time_slot"`    // e.g., "Tuesday 10 AM - 12 PM"
	Score       int      `json:"score"`        // 0-100
	Description string   `json:"description"`  // Explanation
	BasedOn     []string `json:"based_on"`     // What data this is based on
}

// LearnPatternsRequest represents a request to learn patterns.
type LearnPatternsRequest struct {
	GrantID          string  `json:"grant_id"`
	LookbackDays     int     `json:"lookback_days"`     // How far back to analyze
	MinConfidence    float64 `json:"min_confidence"`    // Minimum confidence threshold
	IncludeRecurring bool    `json:"include_recurring"` // Include recurring events
}

// LearnPatterns analyzes calendar history and learns scheduling patterns.
func (p *PatternLearner) LearnPatterns(ctx context.Context, req *LearnPatternsRequest) (*SchedulingPatterns, error) {
	// 1. Fetch historical events
	events, err := p.fetchHistoricalEvents(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("fetch historical events: %w", err)
	}

	if len(events) == 0 {
		return nil, fmt.Errorf("no events found in the specified period")
	}

	// 2. Calculate analysis period
	analysisPeriod := p.calculateAnalysisPeriod(events, req.LookbackDays)

	// 3. Analyze acceptance patterns
	acceptancePatterns := p.analyzeAcceptancePatterns(events)

	// 4. Analyze duration patterns
	durationPatterns := p.analyzeDurationPatterns(events)

	// 5. Analyze timezone patterns
	timezonePatterns := p.analyzeTimezonePatterns(events)

	// 6. Analyze productivity patterns
	productivityInsights := p.analyzeProductivityPatterns(events)

	// 7. Use LLM to generate recommendations
	recommendations, err := p.generateRecommendations(ctx, events, acceptancePatterns, durationPatterns, timezonePatterns, productivityInsights)
	if err != nil {
		// Non-fatal: continue without LLM recommendations
		recommendations = []string{"Unable to generate AI recommendations"}
	}

	patterns := &SchedulingPatterns{
		UserID:               req.GrantID,
		AnalysisPeriod:       analysisPeriod,
		AcceptancePatterns:   acceptancePatterns,
		DurationPatterns:     durationPatterns,
		TimezonePatterns:     timezonePatterns,
		ProductivityInsights: productivityInsights,
		Recommendations:      recommendations,
		TotalEventsAnalyzed:  len(events),
		GeneratedAt:          time.Now(),
	}

	return patterns, nil
}

// fetchHistoricalEvents fetches calendar events for pattern analysis.
func (p *PatternLearner) fetchHistoricalEvents(ctx context.Context, req *LearnPatternsRequest) ([]domain.Event, error) {
	now := time.Now()
	startDate := now.AddDate(0, 0, -req.LookbackDays)

	// First get list of calendars to fetch events from all
	calendars, err := p.nylasClient.GetCalendars(ctx, req.GrantID)
	if err != nil {
		return nil, fmt.Errorf("fetch calendars: %w", err)
	}

	allEvents := []domain.Event{}
	skipped := []string{}

	// Fetch events from each calendar
	for _, calendar := range calendars {
		events, err := p.nylasClient.GetEvents(ctx, req.GrantID, calendar.ID, &domain.EventQueryParams{
			Start: startDate.Unix(),
			End:   now.Unix(),
			Limit: 200, // Maximum allowed by Nylas API v3
		})

		if err != nil {
			// Some calendars are read-only or temporarily unavailable. Record
			// the skip with the underlying error so the caller (and test
			// harness) can see analysis was partial — silently dropping the
			// calendar gives the user "patterns" computed from incomplete
			// data and no way to know.
			skipped = append(skipped, fmt.Sprintf("%s: %v", calendar.ID, err))
			continue
		}

		allEvents = append(allEvents, events...)
	}

	if len(skipped) > 0 {
		// Log to stderr; downstream callers that already check for
		// PartialAnalysis on the returned struct (set below) get the same
		// signal without depending on a logger interface.
		fmt.Fprintf(os.Stderr, "warn: pattern analysis skipped %d calendar(s): %s\n",
			len(skipped), strings.Join(skipped, "; "))
	}

	// Filter out recurring events if not requested
	if !req.IncludeRecurring {
		filtered := []domain.Event{}
		for _, event := range allEvents {
			// Check if event is recurring (has recurrence or is part of series)
			if len(event.Recurrence) == 0 && event.MasterEventID == "" {
				filtered = append(filtered, event)
			}
		}
		return filtered, nil
	}

	return allEvents, nil
}

// calculateAnalysisPeriod calculates the actual period analyzed.
func (p *PatternLearner) calculateAnalysisPeriod(events []domain.Event, _ int) AnalysisPeriod {
	if len(events) == 0 {
		return AnalysisPeriod{}
	}

	earliest := events[0].When.StartTime
	latest := events[0].When.EndTime

	for _, event := range events {
		if event.When.StartTime < earliest {
			earliest = event.When.StartTime
		}
		if event.When.EndTime > latest {
			latest = event.When.EndTime
		}
	}

	// Convert Unix timestamps to time.Time
	earliestTime := time.Unix(earliest, 0)
	latestTime := time.Unix(latest, 0)

	days := int(latestTime.Sub(earliestTime).Hours() / 24)

	return AnalysisPeriod{
		StartDate: earliestTime,
		EndDate:   latestTime,
		Days:      days,
	}
}

// generateRecommendations uses LLM to generate actionable recommendations.
func (p *PatternLearner) generateRecommendations(ctx context.Context, events []domain.Event, acceptance []AcceptancePattern, duration []DurationPattern, timezone []TimezonePattern, productivity []ProductivityInsight) ([]string, error) {
	// Build context for LLM
	patternContext := p.buildPatternContext(events, acceptance, duration, timezone, productivity)

	// Create chat request
	chatReq := &domain.ChatRequest{
		Messages: []domain.ChatMessage{
			{
				Role:    "system",
				Content: "You are an expert productivity coach analyzing calendar patterns. Provide 3-5 actionable recommendations to improve scheduling and productivity.",
			},
			{
				Role:    "user",
				Content: fmt.Sprintf("Based on the following calendar analysis, provide specific recommendations:\n\n%s", patternContext),
			},
		},
		Temperature: 0.7,
		MaxTokens:   500,
	}

	// Call LLM
	response, err := p.llmRouter.Chat(ctx, chatReq)
	if err != nil {
		return nil, err
	}

	// Parse recommendations (simple line-based parsing)
	recommendations := []string{}
	lines := splitLines(response.Content)
	for _, line := range lines {
		trimmed := trimSpace(line)
		if trimmed != "" && len(trimmed) > 10 {
			// Remove numbering if present
			if len(trimmed) > 3 && trimmed[0] >= '1' && trimmed[0] <= '9' && trimmed[1] == '.' {
				trimmed = trimSpace(trimmed[3:])
			}
			recommendations = append(recommendations, trimmed)
		}
	}

	if len(recommendations) == 0 {
		recommendations = []string{"No specific recommendations available"}
	}

	return recommendations, nil
}

// buildPatternContext builds context string for LLM.
// Uses strings.Builder for efficient string concatenation.
func (p *PatternLearner) buildPatternContext(events []domain.Event, acceptance []AcceptancePattern, duration []DurationPattern, timezone []TimezonePattern, productivity []ProductivityInsight) string {
	var sb strings.Builder
	// Pre-allocate estimated capacity (header + patterns)
	sb.Grow(512)

	fmt.Fprintf(&sb, "Calendar Analysis (%d events analyzed):\n\n", len(events))

	// Acceptance patterns
	if len(acceptance) > 0 {
		sb.WriteString("Meeting Acceptance Patterns:\n")
		for i, pattern := range acceptance {
			if i >= 5 {
				break // Top 5
			}
			fmt.Fprintf(&sb, "- %s: %.0f%% acceptance (%d events) - %s\n",
				pattern.TimeSlot, pattern.AcceptRate*100, pattern.EventCount, pattern.Description)
		}
		sb.WriteByte('\n')
	}

	// Duration patterns
	if len(duration) > 0 {
		sb.WriteString("Meeting Duration Patterns:\n")
		for _, pattern := range duration {
			fmt.Fprintf(&sb, "- %s: avg %d minutes (%d events)\n",
				pattern.MeetingType, pattern.ScheduledDuration, pattern.EventCount)
		}
		sb.WriteByte('\n')
	}

	// Timezone patterns
	if len(timezone) > 0 {
		sb.WriteString("Timezone Patterns:\n")
		for i, pattern := range timezone {
			if i >= 3 {
				break // Top 3
			}
			fmt.Fprintf(&sb, "- %s: %.0f%% of meetings (%d events)\n",
				pattern.Timezone, pattern.Percentage*100, pattern.EventCount)
		}
		sb.WriteByte('\n')
	}

	// Productivity insights
	if len(productivity) > 0 {
		sb.WriteString("Productivity Insights:\n")
		for _, insight := range productivity {
			fmt.Fprintf(&sb, "- %s\n", insight.Description)
		}
		sb.WriteByte('\n')
	}

	return sb.String()
}

// SavePatterns saves learned patterns (stub for future storage implementation).
// Returns an error rather than nil so callers can't mistake the no-op for a
// successful persist — pairs with LoadPatterns which already errors.
func (p *PatternLearner) SavePatterns(ctx context.Context, patterns *SchedulingPatterns) error {
	return fmt.Errorf("pattern storage not yet implemented")
}

// LoadPatterns loads previously learned patterns (stub for future storage implementation).
func (p *PatternLearner) LoadPatterns(ctx context.Context, userID string) (*SchedulingPatterns, error) {
	// Future: Load from local storage/database
	return nil, fmt.Errorf("pattern storage not yet implemented")
}

// ExportPatterns exports patterns to JSON.
func (p *PatternLearner) ExportPatterns(patterns *SchedulingPatterns) ([]byte, error) {
	return json.MarshalIndent(patterns, "", "  ")
}
