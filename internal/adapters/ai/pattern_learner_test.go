//go:build !integration

package ai

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/nylas/cli/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewPatternLearner(t *testing.T) {
	t.Run("creates learner with nil dependencies", func(t *testing.T) {
		learner := NewPatternLearner(nil, nil)

		assert.NotNil(t, learner)
		assert.Nil(t, learner.nylasClient)
		assert.Nil(t, learner.llmRouter)
	})
}

func TestPatternLearner_CalculateAnalysisPeriod(t *testing.T) {
	learner := &PatternLearner{}

	tests := []struct {
		name          string
		events        []domain.Event
		requestedDays int
		wantDays      int
		wantEmpty     bool
	}{
		{
			name:          "returns empty for no events",
			events:        []domain.Event{},
			requestedDays: 30,
			wantEmpty:     true,
		},
		{
			name: "calculates correct period",
			events: []domain.Event{
				{When: domain.EventWhen{StartTime: time.Now().Add(-7 * 24 * time.Hour).Unix(), EndTime: time.Now().Add(-7*24*time.Hour + time.Hour).Unix()}},
				{When: domain.EventWhen{StartTime: time.Now().Unix(), EndTime: time.Now().Add(time.Hour).Unix()}},
			},
			requestedDays: 30,
			wantDays:      7, // Approximately 7 days span
		},
		{
			name: "handles single event",
			events: []domain.Event{
				{When: domain.EventWhen{StartTime: time.Now().Unix(), EndTime: time.Now().Add(time.Hour).Unix()}},
			},
			requestedDays: 30,
			wantDays:      0, // Same day
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := learner.calculateAnalysisPeriod(tt.events, tt.requestedDays)

			if tt.wantEmpty {
				assert.True(t, result.StartDate.IsZero())
				return
			}

			assert.False(t, result.StartDate.IsZero())
			assert.False(t, result.EndDate.IsZero())
			// Allow some tolerance in days calculation
			assert.InDelta(t, tt.wantDays, result.Days, 1)
		})
	}
}

// Note: Tests requiring NylasClient/LLMRouter mocks are in integration tests

func TestPatternLearner_BuildPatternContext(t *testing.T) {
	learner := &PatternLearner{}

	events := []domain.Event{{ID: "test-event"}}
	acceptance := []AcceptancePattern{
		{TimeSlot: "Monday 9-11 AM", AcceptRate: 0.85, EventCount: 10, Description: "High acceptance"},
	}
	duration := []DurationPattern{
		{MeetingType: "1-on-1", ScheduledDuration: 30, EventCount: 5},
	}
	timezone := []TimezonePattern{
		{Timezone: "America/New_York", EventCount: 20, Percentage: 0.8},
	}
	productivity := []ProductivityInsight{
		{Description: "Peak productivity in morning"},
	}

	result := learner.buildPatternContext(events, acceptance, duration, timezone, productivity)

	assert.Contains(t, result, "Calendar Analysis (1 events analyzed)")
	assert.Contains(t, result, "Meeting Acceptance Patterns")
	assert.Contains(t, result, "Monday 9-11 AM")
	assert.Contains(t, result, "85%")
	assert.Contains(t, result, "Meeting Duration Patterns")
	assert.Contains(t, result, "1-on-1")
	assert.Contains(t, result, "Timezone Patterns")
	assert.Contains(t, result, "America/New_York")
	assert.Contains(t, result, "Productivity Insights")
	assert.Contains(t, result, "Peak productivity")
}

func TestPatternLearner_BuildPatternContext_LimitsOutput(t *testing.T) {
	learner := &PatternLearner{}

	// Create more than 5 acceptance patterns
	acceptance := make([]AcceptancePattern, 10)
	for i := 0; i < 10; i++ {
		acceptance[i] = AcceptancePattern{
			TimeSlot:   "Slot " + string(rune('A'+i)),
			AcceptRate: 0.5,
			EventCount: 5,
		}
	}

	// Create more than 3 timezone patterns
	timezone := make([]TimezonePattern, 5)
	for i := 0; i < 5; i++ {
		timezone[i] = TimezonePattern{
			Timezone:   "TZ" + string(rune('0'+i)),
			EventCount: 10,
			Percentage: 0.2,
		}
	}

	result := learner.buildPatternContext(
		[]domain.Event{{ID: "test"}},
		acceptance,
		[]DurationPattern{},
		timezone,
		[]ProductivityInsight{},
	)

	// Should only include top 5 acceptance patterns
	assert.Contains(t, result, "Slot A")
	assert.Contains(t, result, "Slot E")
	assert.NotContains(t, result, "Slot F") // 6th pattern should be excluded

	// Should only include top 3 timezone patterns
	assert.Contains(t, result, "TZ0")
	assert.Contains(t, result, "TZ2")
	assert.NotContains(t, result, "TZ3") // 4th pattern should be excluded
}

func TestPatternLearner_SaveLoadPatterns(t *testing.T) {
	ctx := context.Background()
	learner := &PatternLearner{}

	t.Run("SavePatterns returns not implemented error", func(t *testing.T) {
		// SavePatterns is a stub; returning a real error keeps a caller
		// from mistaking the no-op for a successful persist.
		err := learner.SavePatterns(ctx, &SchedulingPatterns{})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not yet implemented")
	})

	t.Run("LoadPatterns returns not implemented error", func(t *testing.T) {
		_, err := learner.LoadPatterns(ctx, "user-123")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not yet implemented")
	})
}

func TestPatternLearner_ExportPatterns(t *testing.T) {
	learner := &PatternLearner{}

	patterns := &SchedulingPatterns{
		UserID: "user-123",
		AnalysisPeriod: AnalysisPeriod{
			StartDate: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			EndDate:   time.Date(2024, 1, 31, 0, 0, 0, 0, time.UTC),
			Days:      30,
		},
		AcceptancePatterns: []AcceptancePattern{
			{TimeSlot: "Monday 9-11 AM", AcceptRate: 0.9},
		},
		DurationPatterns: []DurationPattern{
			{MeetingType: "Standup", ScheduledDuration: 15},
		},
		TimezonePatterns: []TimezonePattern{
			{Timezone: "UTC", Percentage: 1.0},
		},
		ProductivityInsights: []ProductivityInsight{
			{InsightType: "peak_focus", TimeSlot: "Morning"},
		},
		Recommendations:     []string{"Schedule focus blocks"},
		TotalEventsAnalyzed: 100,
		GeneratedAt:         time.Now(),
	}

	data, err := learner.ExportPatterns(patterns)

	require.NoError(t, err)
	assert.NotEmpty(t, data)

	// Verify it's valid JSON
	var parsed SchedulingPatterns
	err = json.Unmarshal(data, &parsed)
	require.NoError(t, err)
	assert.Equal(t, "user-123", parsed.UserID)
	assert.Equal(t, 100, parsed.TotalEventsAnalyzed)
	assert.Len(t, parsed.AcceptancePatterns, 1)
}

func TestSchedulingPatternsTypes(t *testing.T) {
	t.Run("AnalysisPeriod serializes correctly", func(t *testing.T) {
		period := AnalysisPeriod{
			StartDate: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			EndDate:   time.Date(2024, 1, 31, 0, 0, 0, 0, time.UTC),
			Days:      30,
		}

		data, err := json.Marshal(period)
		require.NoError(t, err)
		assert.Contains(t, string(data), `"days":30`)
	})

	t.Run("AcceptancePattern serializes correctly", func(t *testing.T) {
		pattern := AcceptancePattern{
			TimeSlot:    "Monday 9-11 AM",
			AcceptRate:  0.85,
			EventCount:  10,
			Description: "High acceptance",
			Confidence:  0.9,
		}

		data, err := json.Marshal(pattern)
		require.NoError(t, err)
		assert.Contains(t, string(data), `"time_slot":"Monday 9-11 AM"`)
		assert.Contains(t, string(data), `"accept_rate":0.85`)
		assert.Contains(t, string(data), `"confidence":0.9`)
	})

	t.Run("DurationPattern serializes correctly", func(t *testing.T) {
		pattern := DurationPattern{
			MeetingType:       "1-on-1",
			ScheduledDuration: 30,
			ActualDuration:    35,
			Variance:          5,
			EventCount:        20,
			Description:       "Typically runs over",
		}

		data, err := json.Marshal(pattern)
		require.NoError(t, err)
		assert.Contains(t, string(data), `"meeting_type":"1-on-1"`)
		assert.Contains(t, string(data), `"variance":5`)
	})

	t.Run("TimezonePattern serializes correctly", func(t *testing.T) {
		pattern := TimezonePattern{
			Timezone:      "America/Los_Angeles",
			EventCount:    50,
			Percentage:    0.6,
			PreferredTime: "2-4 PM PST",
			Description:   "Most meetings in PST",
		}

		data, err := json.Marshal(pattern)
		require.NoError(t, err)
		assert.Contains(t, string(data), `"timezone":"America/Los_Angeles"`)
		assert.Contains(t, string(data), `"preferred_time":"2-4 PM PST"`)
	})

	t.Run("ProductivityInsight serializes correctly", func(t *testing.T) {
		insight := ProductivityInsight{
			InsightType: "peak_focus",
			TimeSlot:    "Tuesday 10 AM - 12 PM",
			Score:       90,
			Description: "Best time for deep work",
			BasedOn:     []string{"Meeting density", "Focus blocks"},
		}

		data, err := json.Marshal(insight)
		require.NoError(t, err)
		assert.Contains(t, string(data), `"insight_type":"peak_focus"`)
		assert.Contains(t, string(data), `"score":90`)
		assert.Contains(t, string(data), `"based_on"`)
	})

	t.Run("LearnPatternsRequest contains all fields", func(t *testing.T) {
		req := LearnPatternsRequest{
			GrantID:          "grant-123",
			LookbackDays:     30,
			MinConfidence:    0.8,
			IncludeRecurring: true,
		}

		data, err := json.Marshal(req)
		require.NoError(t, err)
		assert.Contains(t, string(data), `"grant_id":"grant-123"`)
		assert.Contains(t, string(data), `"lookback_days":30`)
		assert.Contains(t, string(data), `"min_confidence":0.8`)
		assert.Contains(t, string(data), `"include_recurring":true`)
	})
}
