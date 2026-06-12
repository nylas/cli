package ai

import (
	"cmp"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/nylas/cli/internal/domain"
)

// analyzeAcceptancePatterns analyzes meeting acceptance rates by time slots.
func (p *PatternLearner) analyzeAcceptancePatterns(events []domain.Event) []AcceptancePattern {
	// Group events by day and time slot
	slotCounts := make(map[string]int)
	slotTotal := make(map[string]int)

	for _, event := range events {
		// Convert Unix timestamp to time.Time
		startTime := time.Unix(event.When.StartTime, 0)
		day := startTime.Weekday().String()
		hour := startTime.Hour()

		// Categorize into time blocks
		var timeBlock string
		if hour >= 9 && hour < 11 {
			timeBlock = "9-11 AM"
		} else if hour >= 11 && hour < 13 {
			timeBlock = "11 AM-1 PM"
		} else if hour >= 13 && hour < 15 {
			timeBlock = "1-3 PM"
		} else if hour >= 15 && hour < 17 {
			timeBlock = "3-5 PM"
		} else {
			timeBlock = "Outside hours"
		}

		slot := fmt.Sprintf("%s %s", day, timeBlock)
		slotTotal[slot]++

		// Consider event "accepted" if status is confirmed or busy is true
		if event.Status == "confirmed" || event.Busy {
			slotCounts[slot]++
		}
	}

	// Calculate acceptance rates
	patterns := []AcceptancePattern{}
	for slot, total := range slotTotal {
		if total < 3 {
			// Skip slots with too few samples
			continue
		}

		accepted := slotCounts[slot]
		acceptRate := float64(accepted) / float64(total)

		// Confidence based on sample size (higher samples = higher confidence)
		confidence := float64(total) / 20.0
		if confidence > 1.0 {
			confidence = 1.0
		}

		description := ""
		if acceptRate > 0.8 {
			description = "You prefer meetings during this time"
		} else if acceptRate < 0.4 {
			description = "You tend to avoid meetings during this time"
		} else {
			description = "Moderate acceptance rate"
		}

		patterns = append(patterns, AcceptancePattern{
			TimeSlot:    slot,
			AcceptRate:  acceptRate,
			EventCount:  total,
			Description: description,
			Confidence:  confidence,
		})
	}

	// Sort by accept rate (highest first)
	slices.SortFunc(patterns, func(a, b AcceptancePattern) int {
		return cmp.Compare(b.AcceptRate, a.AcceptRate) // Descending order
	})

	return patterns
}

// analyzeDurationPatterns analyzes meeting duration patterns.
func (p *PatternLearner) analyzeDurationPatterns(events []domain.Event) []DurationPattern {
	// Group events by type (inferred from title patterns)
	typeMap := make(map[string][]domain.Event)

	for _, event := range events {
		meetingType := p.inferMeetingType(event.Title)
		typeMap[meetingType] = append(typeMap[meetingType], event)
	}

	patterns := []DurationPattern{}

	for meetingType, typeEvents := range typeMap {
		if len(typeEvents) < 3 {
			// Skip types with too few samples
			continue
		}

		// Calculate average scheduled duration
		var totalScheduled, totalActual int
		for _, event := range typeEvents {
			// Calculate duration from Unix timestamps (in seconds)
			durationSec := event.When.EndTime - event.When.StartTime
			scheduledDuration := int(durationSec / 60) // Convert to minutes
			totalScheduled += scheduledDuration

			// Actual duration is same as scheduled (we don't have end-time tracking)
			totalActual += scheduledDuration
		}

		avgScheduled := totalScheduled / len(typeEvents)
		avgActual := totalActual / len(typeEvents)

		patterns = append(patterns, DurationPattern{
			MeetingType:       meetingType,
			ScheduledDuration: avgScheduled,
			ActualDuration:    avgActual,
			Variance:          avgActual - avgScheduled,
			EventCount:        len(typeEvents),
			Description:       fmt.Sprintf("Average %d-minute %s meetings", avgScheduled, meetingType),
		})
	}

	return patterns
}

// inferMeetingType infers meeting type from title.
func (p *PatternLearner) inferMeetingType(title string) string {
	titleLower := strings.ToLower(title)

	if containsAny(titleLower, []string{"1:1", "1-on-1", "one-on-one"}) {
		return "1-on-1"
	}
	if containsAny(titleLower, []string{"standup", "daily", "scrum"}) {
		return "Standup"
	}
	if containsAny(titleLower, []string{"review", "retrospective", "retro"}) {
		return "Review"
	}
	if containsAny(titleLower, []string{"planning", "plan"}) {
		return "Planning"
	}
	if containsAny(titleLower, []string{"interview", "candidate"}) {
		return "Interview"
	}
	if containsAny(titleLower, []string{"client", "customer"}) {
		return "Client call"
	}

	return "General meeting"
}

// containsAny checks if string contains any of the substrings.
func containsAny(s string, substrs []string) bool {
	for _, substr := range substrs {
		if len(s) >= len(substr) {
			// Simple substring check
			for i := 0; i <= len(s)-len(substr); i++ {
				if s[i:i+len(substr)] == substr {
					return true
				}
			}
		}
	}
	return false
}

// analyzeTimezonePatterns analyzes timezone preferences.
func (p *PatternLearner) analyzeTimezonePatterns(events []domain.Event) []TimezonePattern {
	tzCounts := make(map[string]int)
	totalEvents := len(events)

	for _, event := range events {
		tz := event.When.StartTimezone
		if tz == "" {
			tz = "UTC"
		}
		tzCounts[tz]++
	}

	patterns := []TimezonePattern{}
	for tz, count := range tzCounts {
		percentage := float64(count) / float64(totalEvents)

		description := fmt.Sprintf("%d%% of meetings in this timezone", int(percentage*100))

		patterns = append(patterns, TimezonePattern{
			Timezone:      tz,
			EventCount:    count,
			Percentage:    percentage,
			PreferredTime: "Varies", // Would need more analysis
			Description:   description,
		})
	}

	// Sort by event count (most common first)
	slices.SortFunc(patterns, func(a, b TimezonePattern) int {
		return cmp.Compare(b.EventCount, a.EventCount) // Descending order
	})

	return patterns
}

// analyzeProductivityPatterns analyzes productivity patterns.
func (p *PatternLearner) analyzeProductivityPatterns(events []domain.Event) []ProductivityInsight {
	// Analyze meeting density by day
	dayDensity := make(map[string]int)
	for _, event := range events {
		startTime := time.Unix(event.When.StartTime, 0)
		day := startTime.Weekday().String()
		dayDensity[day]++
	}

	insights := []ProductivityInsight{}

	// Find peak and low days
	maxDay := ""
	maxCount := 0
	minDay := ""
	minCount := len(events) + 1

	for day, count := range dayDensity {
		if count > maxCount {
			maxCount = count
			maxDay = day
		}
		if count < minCount {
			minCount = count
			minDay = day
		}
	}

	if maxDay != "" {
		insights = append(insights, ProductivityInsight{
			InsightType: "high_meeting_density",
			TimeSlot:    maxDay,
			Score:       30, // Lower score for high meeting days
			Description: fmt.Sprintf("%s has the most meetings (%d) - may impact focus time", maxDay, maxCount),
			BasedOn:     []string{"Meeting count by day"},
		})
	}

	if minDay != "" {
		insights = append(insights, ProductivityInsight{
			InsightType: "low_meeting_density",
			TimeSlot:    minDay,
			Score:       90, // Higher score for low meeting days
			Description: fmt.Sprintf("%s has the fewest meetings (%d) - good for deep work", minDay, minCount),
			BasedOn:     []string{"Meeting count by day"},
		})
	}

	return insights
}
