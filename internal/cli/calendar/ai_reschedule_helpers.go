package calendar

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/nylas/cli/internal/adapters/analytics"
	"github.com/nylas/cli/internal/domain"
	"github.com/nylas/cli/internal/ports"
)

// fetchEventByID fetches an event from any calendar by its ID
func fetchEventByID(ctx context.Context, client ports.NylasClient, grantID, eventID string) (*domain.Event, error) {
	// Get all calendars
	calendars, err := client.GetCalendars(ctx, grantID)
	if err != nil {
		return nil, err
	}

	// Search each calendar for the event
	for _, calendar := range calendars {
		event, err := client.GetEvent(ctx, grantID, calendar.ID, eventID)
		if err == nil {
			return event, nil
		}
	}

	return nil, fmt.Errorf("event %s not found in any calendar", eventID)
}

// findRescheduleSuggestions finds alternative times for rescheduling
func findRescheduleSuggestions(
	ctx context.Context,
	_ ports.NylasClient,
	resolver *analytics.ConflictResolver,
	grantID string,
	event *domain.Event,
	request *domain.RescheduleRequest,
	patterns *domain.MeetingPattern,
) ([]domain.RescheduleOption, error) {
	originalStart := time.Unix(event.When.StartTime, 0)
	originalEnd := time.Unix(event.When.EndTime, 0)
	duration := int(originalEnd.Sub(originalStart).Minutes())

	var suggestions []domain.RescheduleOption

	// Try preferred times first
	for _, preferredTime := range request.PreferredTimes {
		proposedEvent := &domain.Event{
			Title:        event.Title,
			Participants: event.Participants,
			When: domain.EventWhen{
				StartTime: preferredTime.Unix(),
				EndTime:   preferredTime.Add(time.Duration(duration) * time.Minute).Unix(),
			},
		}

		analysis, err := resolver.DetectConflicts(ctx, grantID, proposedEvent, patterns)
		if err != nil {
			continue
		}

		// Only hard conflicts prevent this time
		if len(analysis.HardConflicts) == 0 {
			option := domain.RescheduleOption{
				ProposedTime: preferredTime,
				EndTime:      preferredTime.Add(time.Duration(duration) * time.Minute),
				Score:        calculateRescheduleScore(analysis, patterns, preferredTime),
				Conflicts:    analysis.SoftConflicts,
				Pros:         []string{"Preferred time"},
				Cons:         buildConsFromConflicts(analysis.SoftConflicts),
			}
			suggestions = append(suggestions, option)
		}
	}

	// Generate additional suggestions based on patterns and constraints
	maxDelay := time.Duration(request.MaxDelayDays) * 24 * time.Hour
	endSearch := originalStart.Add(maxDelay)

	// Try same time on different days
	for days := 1; days <= request.MaxDelayDays; days++ {
		proposedTime := originalStart.AddDate(0, 0, days)

		if proposedTime.After(endSearch) {
			break
		}

		// Skip avoided days
		if shouldAvoidDay(proposedTime, request.AvoidDays) {
			continue
		}

		proposedEvent := &domain.Event{
			Title:        event.Title,
			Participants: event.Participants,
			When: domain.EventWhen{
				StartTime: proposedTime.Unix(),
				EndTime:   proposedTime.Add(time.Duration(duration) * time.Minute).Unix(),
			},
		}

		analysis, err := resolver.DetectConflicts(ctx, grantID, proposedEvent, patterns)
		if err != nil {
			continue
		}

		// Only suggest if no hard conflicts
		if len(analysis.HardConflicts) == 0 {
			option := domain.RescheduleOption{
				ProposedTime:     proposedTime,
				EndTime:          proposedTime.Add(time.Duration(duration) * time.Minute),
				Score:            calculateRescheduleScore(analysis, patterns, proposedTime),
				Conflicts:        analysis.SoftConflicts,
				Pros:             buildProsFromPatterns(proposedTime, patterns),
				Cons:             buildConsFromConflicts(analysis.SoftConflicts),
				ParticipantMatch: 100.0, // Assume 100% if no hard conflicts
			}
			suggestions = append(suggestions, option)
		}

		// Limit suggestions to top candidates
		if len(suggestions) >= 10 {
			break
		}
	}

	// Sort by score (highest first)
	for i := 0; i < len(suggestions)-1; i++ {
		for j := i + 1; j < len(suggestions); j++ {
			if suggestions[j].Score > suggestions[i].Score {
				suggestions[i], suggestions[j] = suggestions[j], suggestions[i]
			}
		}
	}

	// Return top 5
	if len(suggestions) > 5 {
		suggestions = suggestions[:5]
	}

	return suggestions, nil
}

// calculateRescheduleScore scores a potential reschedule time
func calculateRescheduleScore(analysis *domain.ConflictAnalysis, patterns *domain.MeetingPattern, proposedTime time.Time) int {
	score := 100

	// Penalize soft conflicts
	score -= len(analysis.SoftConflicts) * 10

	// Bonus for high-acceptance day
	if patterns != nil {
		dayOfWeek := proposedTime.Weekday().String()
		if rate, exists := patterns.Acceptance.ByDayOfWeek[dayOfWeek]; exists {
			score += int(rate * 15) // Up to +15 for high acceptance
		}

		// Bonus for high-acceptance time
		timeKey := fmt.Sprintf("%02d:00", proposedTime.Hour())
		if rate, exists := patterns.Acceptance.ByTimeOfDay[timeKey]; exists {
			score += int(rate * 15) // Up to +15 for high acceptance
		}
	}

	// Ensure score is in 0-100 range
	if score < 0 {
		score = 0
	}
	if score > 100 {
		score = 100
	}

	return score
}

// shouldAvoidDay checks if the day should be avoided
func shouldAvoidDay(t time.Time, avoidDays []string) bool {
	dayName := t.Weekday().String()
	for _, avoid := range avoidDays {
		if strings.EqualFold(avoid, dayName) {
			return true
		}
	}
	return false
}

// buildProsFromPatterns builds pros list from patterns
func buildProsFromPatterns(t time.Time, patterns *domain.MeetingPattern) []string {
	var pros []string

	if patterns == nil {
		return pros
	}

	dayOfWeek := t.Weekday().String()
	if rate, exists := patterns.Acceptance.ByDayOfWeek[dayOfWeek]; exists && rate > 0.8 {
		pros = append(pros, fmt.Sprintf("High acceptance rate on %ss (%.0f%%)", dayOfWeek, rate*100))
	}

	timeKey := fmt.Sprintf("%02d:00", t.Hour())
	if rate, exists := patterns.Acceptance.ByTimeOfDay[timeKey]; exists && rate > 0.8 {
		pros = append(pros, fmt.Sprintf("Preferred time slot (%.0f%% acceptance)", rate*100))
	}

	// Check if it's in a productive time block
	for _, block := range patterns.Productivity.PeakFocus {
		if block.DayOfWeek == dayOfWeek {
			blockStart := parseHourFromString(block.StartTime)
			blockEnd := parseHourFromString(block.EndTime)
			hour := t.Hour()
			if hour >= blockStart && hour < blockEnd {
				pros = append(pros, "During typical focus time (good for productive meetings)")
			}
		}
	}

	return pros
}

// buildConsFromConflicts builds cons list from soft conflicts
func buildConsFromConflicts(conflicts []domain.Conflict) []string {
	var cons []string

	for _, conflict := range conflicts {
		cons = append(cons, conflict.Impact)
	}

	return cons
}

// parseHourFromString parses hour from "HH:MM" format
func parseHourFromString(timeStr string) int {
	var hour int
	_, _ = fmt.Sscanf(timeStr, "%d:", &hour) // Parse hour, default 0 on error
	return hour
}

// applyReschedule applies the selected reschedule option
func applyReschedule(
	ctx context.Context,
	client ports.NylasClient,
	grantID string,
	event *domain.Event,
	option domain.RescheduleOption,
	notify bool,
	_ string,
) (*domain.RescheduleResult, error) {
	// Find which calendar the event belongs to
	calendars, err := client.GetCalendars(ctx, grantID)
	if err != nil {
		return nil, err
	}

	var calendarID string
	for _, calendar := range calendars {
		_, err := client.GetEvent(ctx, grantID, calendar.ID, event.ID)
		if err == nil {
			calendarID = calendar.ID
			break
		}
	}

	if calendarID == "" {
		return nil, fmt.Errorf("could not find calendar for event")
	}

	// Update the event with new time
	updateReq := &domain.UpdateEventRequest{
		When: &domain.EventWhen{
			StartTime: option.ProposedTime.Unix(),
			EndTime:   option.EndTime.Unix(),
		},
	}

	newEvent, err := client.UpdateEvent(ctx, grantID, calendarID, event.ID, updateReq)
	if err != nil {
		return nil, err
	}

	result := &domain.RescheduleResult{
		Success:        true,
		OriginalEvent:  event,
		NewEvent:       newEvent,
		SelectedOption: &option,
		Message:        fmt.Sprintf("Successfully rescheduled to %s", option.ProposedTime.Format("Mon, Jan 2 at 3:04 PM MST")),
	}

	if notify {
		result.NotificationsSent = len(event.Participants)
	}

	return result, nil
}

// displayRescheduleSuggestions displays reschedule suggestions
func displayRescheduleSuggestions(event *domain.Event, suggestions []domain.RescheduleOption, reason string) {
	fmt.Println("\n📊 Reschedule Analysis")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	if reason != "" {
		fmt.Printf("\nReason: %s\n", reason)
	}

	if len(suggestions) == 0 {
		fmt.Println("\n❌ No suitable alternative times found.")
		fmt.Println("   Try increasing --max-delay-days or removing constraints.")
		return
	}

	fmt.Printf("\n🔄 Found %d Alternative Time(s)\n", len(suggestions))
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	for i, option := range suggestions {
		// Score color coding
		scoreIcon := "🟢"
		if option.Score < 50 {
			scoreIcon = "🔴"
		} else if option.Score < 75 {
			scoreIcon = "🟡"
		}

		fmt.Printf("\n%d. %s %s (Score: %d/100)\n",
			i+1,
			scoreIcon,
			option.ProposedTime.Format("Mon, Jan 2, 2006 at 3:04 PM MST"),
			option.Score)

		if len(option.Pros) > 0 {
			fmt.Println("\n   Pros:")
			for _, pro := range option.Pros {
				fmt.Printf("   ✓ %s\n", pro)
			}
		}

		if len(option.Cons) > 0 {
			fmt.Println("\n   Cons:")
			for _, con := range option.Cons {
				fmt.Printf("   ⚠️  %s\n", con)
			}
		}

		if option.AIInsight != "" {
			fmt.Printf("\n   💡 %s\n", option.AIInsight)
		}

		if len(option.Conflicts) > 0 {
			fmt.Printf("\n   ⚠️  %d soft conflict(s)\n", len(option.Conflicts))
		}
	}
}

// displayRescheduleResult displays the reschedule result
func displayRescheduleResult(result *domain.RescheduleResult) {
	fmt.Println("\n✅ Reschedule Complete")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	fmt.Printf("\n%s\n", result.Message)

	if result.NewEvent != nil {
		newStart := time.Unix(result.NewEvent.When.StartTime, 0)
		fmt.Printf("\nNew time: %s\n", newStart.Format("Mon, Jan 2, 2006 at 3:04 PM MST"))
	}

	if result.NotificationsSent > 0 {
		fmt.Printf("📧 Notifications sent to %d participant(s)\n", result.NotificationsSent)
	}

	if len(result.CascadingChanges) > 0 {
		fmt.Println("\n⚠️  Cascading changes:")
		for _, change := range result.CascadingChanges {
			fmt.Printf("   • %s\n", change)
		}
	}
}
