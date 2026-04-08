package analytics

import (
	"context"
	"testing"
	"time"

	"github.com/nylas/cli/internal/adapters/nylas"
	"github.com/nylas/cli/internal/domain"
)

func TestFocusOptimizer_AnalyzeFocusTimePatterns(t *testing.T) {
	client := nylas.NewMockClient()
	optimizer := NewFocusOptimizer(client)

	// Set up mock events
	now := time.Now()
	events := []domain.Event{
		{
			ID:    "event-1",
			Title: "Team Meeting",
			When: domain.EventWhen{
				StartTime: now.Add(-24 * time.Hour).Unix(),
				EndTime:   now.Add(-23 * time.Hour).Unix(),
			},
		},
		{
			ID:    "event-2",
			Title: "1-on-1",
			When: domain.EventWhen{
				StartTime: now.Add(-12 * time.Hour).Unix(),
				EndTime:   now.Add(-11 * time.Hour).Unix(),
			},
		},
	}

	client.GetCalendarsFunc = func(ctx context.Context, grantID string) ([]domain.Calendar, error) {
		return []domain.Calendar{{ID: "cal-1", Name: "Primary"}}, nil
	}

	client.GetEventsFunc = func(ctx context.Context, grantID, calendarID string, params *domain.EventQueryParams) ([]domain.Event, error) {
		return events, nil
	}

	settings := &domain.FocusTimeSettings{
		Enabled:            true,
		TargetHoursPerWeek: 14.0,
		MinBlockDuration:   60,
		MaxBlockDuration:   240,
	}

	analysis, err := optimizer.AnalyzeFocusTimePatterns(context.Background(), "grant-123", settings)
	if err != nil {
		t.Fatalf("AnalyzeFocusTimePatterns() error = %v", err)
	}

	if analysis == nil {
		t.Fatal("Expected analysis, got nil")
		return
	}

	if analysis.UserEmail != "grant-123" {
		t.Errorf("UserEmail = %q, want %q", analysis.UserEmail, "grant-123")
	}

	if analysis.TargetProtection != 14.0 {
		t.Errorf("TargetProtection = %.1f, want %.1f", analysis.TargetProtection, 14.0)
	}

	if len(analysis.Insights) == 0 {
		t.Error("Expected insights, got none")
	}

	if analysis.Confidence < 0 || analysis.Confidence > 100 {
		t.Errorf("Confidence = %.1f, want 0-100", analysis.Confidence)
	}
}

func TestFocusOptimizer_CalculateDeepWorkStats(t *testing.T) {
	optimizer := NewFocusOptimizer(nil)

	patterns := &domain.MeetingPattern{
		Productivity: domain.ProductivityPatterns{
			FocusBlocks: []domain.TimeBlock{
				{StartTime: "09:00", EndTime: "11:00"}, // 120 minutes
				{StartTime: "14:00", EndTime: "16:30"}, // 150 minutes
			},
		},
	}

	stats := optimizer.calculateDeepWorkStats(patterns)

	if stats.AverageScheduled != 135 { // (120 + 150) / 2 = 135
		t.Errorf("AverageScheduled = %d, want 135", stats.AverageScheduled)
	}
}

func TestFocusOptimizer_CalculateBlockDuration(t *testing.T) {
	optimizer := NewFocusOptimizer(nil)

	tests := []struct {
		name      string
		startTime string
		endTime   string
		want      int
	}{
		{
			name:      "2 hour block",
			startTime: "09:00",
			endTime:   "11:00",
			want:      120,
		},
		{
			name:      "1.5 hour block",
			startTime: "14:00",
			endTime:   "15:30",
			want:      90,
		},
		{
			name:      "4 hour block",
			startTime: "09:00",
			endTime:   "13:00",
			want:      240,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := optimizer.calculateBlockDuration(tt.startTime, tt.endTime)
			if got != tt.want {
				t.Errorf("calculateBlockDuration(%q, %q) = %d, want %d",
					tt.startTime, tt.endTime, got, tt.want)
			}
		})
	}
}

func TestFocusOptimizer_FindPeakProductivityBlocks(t *testing.T) {
	optimizer := NewFocusOptimizer(nil)

	patterns := &domain.MeetingPattern{
		Productivity: domain.ProductivityPatterns{
			PeakFocus: []domain.TimeBlock{
				{DayOfWeek: "Monday", StartTime: "09:00", EndTime: "11:00", Score: 75.0},
				{DayOfWeek: "Tuesday", StartTime: "10:00", EndTime: "12:00", Score: 95.0},
				{DayOfWeek: "Wednesday", StartTime: "09:00", EndTime: "13:00", Score: 85.0},
				{DayOfWeek: "Thursday", StartTime: "10:00", EndTime: "12:00", Score: 92.0},
			},
		},
	}

	blocks := optimizer.findPeakProductivityBlocks(patterns)

	if len(blocks) != 3 {
		t.Fatalf("Expected 3 peak blocks, got %d", len(blocks))
	}

	// Should be sorted by score (highest first)
	if blocks[0].DayOfWeek != "Tuesday" || blocks[0].Score != 95.0 {
		t.Errorf("First block = %s (%.1f), want Tuesday (95.0)", blocks[0].DayOfWeek, blocks[0].Score)
	}

	if blocks[1].DayOfWeek != "Thursday" || blocks[1].Score != 92.0 {
		t.Errorf("Second block = %s (%.1f), want Thursday (92.0)", blocks[1].DayOfWeek, blocks[1].Score)
	}

	if blocks[2].DayOfWeek != "Wednesday" || blocks[2].Score != 85.0 {
		t.Errorf("Third block = %s (%.1f), want Wednesday (85.0)", blocks[2].DayOfWeek, blocks[2].Score)
	}
}

func TestFocusOptimizer_FindMostProductiveDay(t *testing.T) {
	optimizer := NewFocusOptimizer(nil)

	patterns := &domain.MeetingPattern{
		Productivity: domain.ProductivityPatterns{
			MeetingDensity: map[string]float64{
				"Monday":    5.2,
				"Tuesday":   3.1, // Lowest density = most productive
				"Wednesday": 4.5,
				"Thursday":  3.8,
				"Friday":    4.0,
			},
		},
	}

	mostProductive := optimizer.findMostProductiveDay(patterns)

	if mostProductive != "Tuesday" {
		t.Errorf("Most productive day = %s, want Tuesday", mostProductive)
	}
}
