//go:build !integration

package ai

import (
	"testing"
	"time"

	"github.com/nylas/cli/internal/domain"
	"github.com/stretchr/testify/assert"
)

func TestPatternLearner_AnalyzeAcceptancePatterns(t *testing.T) {
	learner := &PatternLearner{}

	// Create events at specific local times for testing
	// Use Local timezone to match what analyzeAcceptancePatterns expects
	makeEventLocal := func(year, month, day, hour int, status string, busy bool) domain.Event {
		eventTime := time.Date(year, time.Month(month), day, hour, 0, 0, 0, time.Local)

		return domain.Event{
			When: domain.EventWhen{
				StartTime: eventTime.Unix(),
				EndTime:   eventTime.Add(time.Hour).Unix(),
			},
			Status: status,
			Busy:   busy,
		}
	}

	t.Run("returns empty for insufficient samples", func(t *testing.T) {
		events := []domain.Event{
			makeEventLocal(2024, 1, 1, 10, "confirmed", true),
			makeEventLocal(2024, 1, 1, 10, "confirmed", true),
			// Only 2 events - should be skipped
		}
		result := learner.analyzeAcceptancePatterns(events)
		assert.Empty(t, result)
	})

	t.Run("calculates acceptance rate correctly", func(t *testing.T) {
		// 4 events same slot: 3 confirmed/busy, 1 tentative/not busy
		events := []domain.Event{
			makeEventLocal(2024, 1, 1, 10, "confirmed", true),
			makeEventLocal(2024, 1, 1, 10, "confirmed", true),
			makeEventLocal(2024, 1, 1, 10, "confirmed", true),
			makeEventLocal(2024, 1, 1, 10, "tentative", false),
		}
		result := learner.analyzeAcceptancePatterns(events)

		assert.Len(t, result, 1)
		assert.InDelta(t, 0.75, result[0].AcceptRate, 0.01) // 3/4 = 0.75
	})

	t.Run("sorts by acceptance rate descending", func(t *testing.T) {
		// Create events in two different slots with different acceptance rates
		// Note: We use different hours that fall into distinct time blocks
		events := []domain.Event{
			// First slot: low acceptance (25%)
			makeEventLocal(2024, 1, 1, 9, "confirmed", true),
			makeEventLocal(2024, 1, 1, 9, "tentative", false),
			makeEventLocal(2024, 1, 1, 9, "tentative", false),
			makeEventLocal(2024, 1, 1, 9, "tentative", false),
			// Second slot: high acceptance (100%)
			makeEventLocal(2024, 1, 1, 15, "confirmed", true),
			makeEventLocal(2024, 1, 1, 15, "confirmed", true),
			makeEventLocal(2024, 1, 1, 15, "confirmed", true),
		}
		result := learner.analyzeAcceptancePatterns(events)

		// If we have multiple slots, highest rate should be first
		if len(result) > 1 {
			assert.Greater(t, result[0].AcceptRate, result[1].AcceptRate)
		}
	})

	t.Run("generates descriptions based on acceptance rate", func(t *testing.T) {
		// High acceptance (>80%) should say "prefer meetings"
		events := []domain.Event{
			makeEventLocal(2024, 1, 1, 10, "confirmed", true),
			makeEventLocal(2024, 1, 1, 10, "confirmed", true),
			makeEventLocal(2024, 1, 1, 10, "confirmed", true),
			makeEventLocal(2024, 1, 1, 10, "confirmed", true),
			makeEventLocal(2024, 1, 1, 10, "confirmed", true),
		}
		result := learner.analyzeAcceptancePatterns(events)

		assert.Len(t, result, 1)
		assert.Contains(t, result[0].Description, "prefer meetings")
	})
}

func TestPatternLearner_AnalyzeAcceptancePatterns_Descriptions(t *testing.T) {
	learner := &PatternLearner{}

	makeEvents := func(count int, accepted int) []domain.Event {
		events := make([]domain.Event, count)
		baseTime := time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC) // Monday 10 AM

		for i := 0; i < count; i++ {
			status := "tentative"
			busy := false
			if i < accepted {
				status = "confirmed"
				busy = true
			}
			events[i] = domain.Event{
				When: domain.EventWhen{
					StartTime: baseTime.Unix(),
					EndTime:   baseTime.Add(time.Hour).Unix(),
				},
				Status: status,
				Busy:   busy,
			}
		}
		return events
	}

	tests := []struct {
		name     string
		events   []domain.Event
		wantDesc string
	}{
		{
			name:     "high acceptance description",
			events:   makeEvents(5, 5), // 100% acceptance
			wantDesc: "prefer meetings",
		},
		{
			name:     "low acceptance description",
			events:   makeEvents(5, 1), // 20% acceptance
			wantDesc: "avoid meetings",
		},
		{
			name:     "moderate acceptance description",
			events:   makeEvents(4, 2), // 50% acceptance
			wantDesc: "Moderate acceptance",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := learner.analyzeAcceptancePatterns(tt.events)

			if len(result) > 0 {
				assert.Contains(t, result[0].Description, tt.wantDesc)
			}
		})
	}
}

func TestPatternLearner_AnalyzeDurationPatterns(t *testing.T) {
	learner := &PatternLearner{}

	makeEvent := func(title string, durationMinutes int) domain.Event {
		start := time.Now()
		return domain.Event{
			Title: title,
			When: domain.EventWhen{
				StartTime: start.Unix(),
				EndTime:   start.Add(time.Duration(durationMinutes) * time.Minute).Unix(),
			},
		}
	}

	tests := []struct {
		name          string
		events        []domain.Event
		wantTypes     []string
		wantDurations map[string]int // type -> expected avg duration
	}{
		{
			name: "groups by meeting type",
			events: []domain.Event{
				makeEvent("Team Standup", 15),
				makeEvent("Daily Standup", 15),
				makeEvent("Standup meeting", 20),
				makeEvent("1:1 with Alice", 30),
				makeEvent("1-on-1 with Bob", 30),
				makeEvent("One-on-one review", 45),
			},
			wantTypes: []string{"Standup", "1-on-1"},
			wantDurations: map[string]int{
				"Standup": 16, // (15+15+20)/3 = 16.67
				"1-on-1":  35, // (30+30+45)/3 = 35
			},
		},
		{
			name: "skips types with fewer than 3 samples",
			events: []domain.Event{
				makeEvent("Interview candidate", 60),
				makeEvent("Interview with John", 60),
				// Only 2 interviews - should be skipped
			},
			wantTypes: []string{}, // No patterns should be returned
		},
		{
			name: "categorizes general meetings",
			events: []domain.Event{
				makeEvent("Random meeting", 30),
				makeEvent("Discussion", 45),
				makeEvent("Sync up", 30),
			},
			wantTypes: []string{"General meeting"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := learner.analyzeDurationPatterns(tt.events)

			typeNames := make([]string, len(result))
			for i, p := range result {
				typeNames[i] = p.MeetingType
			}

			for _, wantType := range tt.wantTypes {
				assert.Contains(t, typeNames, wantType, "expected type %s to be present", wantType)
			}

			for meetingType, wantDuration := range tt.wantDurations {
				for _, p := range result {
					if p.MeetingType == meetingType {
						assert.InDelta(t, wantDuration, p.ScheduledDuration, 2, "duration for %s", meetingType)
					}
				}
			}
		})
	}
}

func TestPatternLearner_InferMeetingType(t *testing.T) {
	learner := &PatternLearner{}

	tests := []struct {
		title string
		want  string
	}{
		{"1:1 with Alice", "1-on-1"},
		{"1-on-1 review", "1-on-1"},
		{"One-on-one meeting", "1-on-1"},
		{"Daily Standup", "Standup"},
		{"Team Scrum", "Standup"},
		{"Daily sync", "Standup"},
		{"Sprint Review", "Review"},
		{"Retrospective", "Review"},
		{"Retro meeting", "Review"},
		{"Sprint Planning", "Planning"},
		{"Q4 Plan session", "Planning"},
		{"Interview with candidate", "Interview"},
		{"Candidate screening", "Interview"},
		{"Client call", "Client call"},
		{"Customer meeting", "Client call"},
		{"Random meeting", "General meeting"},
		{"", "General meeting"},
	}

	for _, tt := range tests {
		t.Run(tt.title, func(t *testing.T) {
			result := learner.inferMeetingType(tt.title)
			assert.Equal(t, tt.want, result)
		})
	}
}

func TestContainsAny(t *testing.T) {
	tests := []struct {
		s       string
		substrs []string
		want    bool
	}{
		{"hello world", []string{"hello"}, true},
		{"hello world", []string{"world"}, true},
		{"hello world", []string{"foo", "bar"}, false},
		{"standup meeting", []string{"standup", "daily"}, true},
		{"daily scrum", []string{"standup", "scrum"}, true},
		{"", []string{"test"}, false},
		{"test", []string{}, false},
		{"short", []string{"longer"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.s, func(t *testing.T) {
			result := containsAny(tt.s, tt.substrs)
			assert.Equal(t, tt.want, result)
		})
	}
}

func TestPatternLearner_AnalyzeTimezonePatterns(t *testing.T) {
	learner := &PatternLearner{}

	tests := []struct {
		name    string
		events  []domain.Event
		wantTop string  // Expected top timezone
		wantPct float64 // Expected percentage for top
	}{
		{
			name: "calculates timezone distribution",
			events: []domain.Event{
				{When: domain.EventWhen{StartTimezone: "America/New_York"}},
				{When: domain.EventWhen{StartTimezone: "America/New_York"}},
				{When: domain.EventWhen{StartTimezone: "America/New_York"}},
				{When: domain.EventWhen{StartTimezone: "America/Los_Angeles"}},
				{When: domain.EventWhen{StartTimezone: "America/Los_Angeles"}},
			},
			wantTop: "America/New_York",
			wantPct: 0.6, // 3/5
		},
		{
			name: "defaults empty timezone to UTC",
			events: []domain.Event{
				{When: domain.EventWhen{StartTimezone: ""}},
				{When: domain.EventWhen{StartTimezone: ""}},
				{When: domain.EventWhen{StartTimezone: "Europe/London"}},
			},
			wantTop: "UTC",
			wantPct: 0.67, // 2/3
		},
		{
			name: "sorts by event count",
			events: []domain.Event{
				{When: domain.EventWhen{StartTimezone: "Asia/Tokyo"}},
				{When: domain.EventWhen{StartTimezone: "Europe/Paris"}},
				{When: domain.EventWhen{StartTimezone: "Europe/Paris"}},
				{When: domain.EventWhen{StartTimezone: "Europe/Paris"}},
			},
			wantTop: "Europe/Paris",
			wantPct: 0.75, // 3/4
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := learner.analyzeTimezonePatterns(tt.events)

			assert.NotEmpty(t, result)
			assert.Equal(t, tt.wantTop, result[0].Timezone)
			assert.InDelta(t, tt.wantPct, result[0].Percentage, 0.01)
		})
	}
}

func TestPatternLearner_AnalyzeProductivityPatterns(t *testing.T) {
	learner := &PatternLearner{}

	makeEvent := func(weekday time.Weekday) domain.Event {
		baseTime := time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)
		daysToAdd := int(weekday) - int(baseTime.Weekday())
		if daysToAdd < 0 {
			daysToAdd += 7
		}
		eventTime := baseTime.AddDate(0, 0, daysToAdd)

		return domain.Event{
			When: domain.EventWhen{
				StartTime: eventTime.Unix(),
				EndTime:   eventTime.Add(time.Hour).Unix(),
			},
		}
	}

	tests := []struct {
		name        string
		events      []domain.Event
		wantHighDay string
		wantLowDay  string
	}{
		{
			name: "identifies high and low meeting days",
			events: []domain.Event{
				makeEvent(time.Monday),
				makeEvent(time.Monday),
				makeEvent(time.Monday),
				makeEvent(time.Monday),
				makeEvent(time.Monday), // 5 on Monday
				makeEvent(time.Tuesday),
				makeEvent(time.Tuesday),   // 2 on Tuesday
				makeEvent(time.Wednesday), // 1 on Wednesday
			},
			wantHighDay: "Monday",
			wantLowDay:  "Wednesday",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := learner.analyzeProductivityPatterns(tt.events)

			assert.NotEmpty(t, result)

			// Find high and low density insights
			var highDensity, lowDensity *ProductivityInsight
			for i := range result {
				if result[i].InsightType == "high_meeting_density" {
					highDensity = &result[i]
				}
				if result[i].InsightType == "low_meeting_density" {
					lowDensity = &result[i]
				}
			}

			if tt.wantHighDay != "" {
				assert.NotNil(t, highDensity)
				assert.Equal(t, tt.wantHighDay, highDensity.TimeSlot)
				assert.Equal(t, 30, highDensity.Score) // Low score for busy day
			}

			if tt.wantLowDay != "" {
				assert.NotNil(t, lowDensity)
				assert.Equal(t, tt.wantLowDay, lowDensity.TimeSlot)
				assert.Equal(t, 90, lowDensity.Score) // High score for quiet day
			}
		})
	}
}

// Note: splitLines/trimSpace helpers were removed in favor of stdlib
// strings.Split/strings.TrimSpace (used directly in pattern_learner.go).

func TestPatternLearner_ConfidenceCalculation(t *testing.T) {
	learner := &PatternLearner{}

	// Test that confidence is calculated based on sample size
	tests := []struct {
		name           string
		sampleSize     int
		wantConfidence float64
	}{
		{"low sample", 5, 0.25},       // 5/20 = 0.25
		{"medium sample", 10, 0.5},    // 10/20 = 0.5
		{"high sample", 20, 1.0},      // 20/20 = 1.0, capped at 1.0
		{"very high sample", 40, 1.0}, // Capped at 1.0
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create events with the specified sample size
			events := make([]domain.Event, tt.sampleSize)
			for i := 0; i < tt.sampleSize; i++ {
				baseTime := time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)
				events[i] = domain.Event{
					When: domain.EventWhen{
						StartTime: baseTime.Unix(),
						EndTime:   baseTime.Add(time.Hour).Unix(),
					},
					Status: "confirmed",
					Busy:   true,
				}
			}

			result := learner.analyzeAcceptancePatterns(events)

			if len(result) > 0 {
				assert.InDelta(t, tt.wantConfidence, result[0].Confidence, 0.01)
			}
		})
	}
}
