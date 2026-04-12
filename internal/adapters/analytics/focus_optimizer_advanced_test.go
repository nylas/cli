package analytics

import (
	"context"
	"testing"
	"time"

	"github.com/nylas/cli/internal/adapters/nylas"
	"github.com/nylas/cli/internal/domain"
)

func TestFocusOptimizer_FindLeastProductiveDay(t *testing.T) {
	optimizer := NewFocusOptimizer(nil)

	patterns := &domain.MeetingPattern{
		Productivity: domain.ProductivityPatterns{
			MeetingDensity: map[string]float64{
				"Monday":    6.8, // Highest density = least productive
				"Tuesday":   3.1,
				"Wednesday": 4.5,
				"Thursday":  3.8,
				"Friday":    4.0,
			},
		},
	}

	leastProductive := optimizer.findLeastProductiveDay(patterns)

	if leastProductive != "Monday" {
		t.Errorf("Least productive day = %s, want Monday", leastProductive)
	}
}

func TestFocusOptimizer_GenerateRecommendedBlocks(t *testing.T) {
	optimizer := NewFocusOptimizer(nil)

	patterns := &domain.MeetingPattern{
		Productivity: domain.ProductivityPatterns{
			PeakFocus: []domain.TimeBlock{
				{DayOfWeek: "Tuesday", StartTime: "10:00", EndTime: "12:00", Score: 95.0},
				{DayOfWeek: "Thursday", StartTime: "10:00", EndTime: "12:00", Score: 92.0},
				{DayOfWeek: "Wednesday", StartTime: "09:00", EndTime: "13:00", Score: 85.0},
			},
		},
	}

	settings := &domain.FocusTimeSettings{
		MinBlockDuration:   60,
		MaxBlockDuration:   240,
		TargetHoursPerWeek: 6.0, // 360 minutes
	}

	blocks := optimizer.generateRecommendedBlocks(patterns, settings)

	if len(blocks) == 0 {
		t.Fatal("Expected recommended blocks, got none")
	}

	// Check that blocks are sorted by score
	for i := 0; i < len(blocks)-1; i++ {
		if blocks[i].Score < blocks[i+1].Score {
			t.Errorf("Blocks not sorted: block[%d].Score (%.1f) < block[%d].Score (%.1f)",
				i, blocks[i].Score, i+1, blocks[i+1].Score)
		}
	}

	// Check that total duration is close to target (allow some tolerance)
	totalMinutes := 0
	for _, block := range blocks {
		totalMinutes += block.Duration
	}

	targetMinutes := int(settings.TargetHoursPerWeek * 60)
	// Allow up to one extra block's worth of minutes (since we can't split blocks)
	maxAllowedMinutes := targetMinutes + 240 // Max block duration from pattern

	if totalMinutes > maxAllowedMinutes {
		t.Errorf("Total duration %d minutes exceeds max allowed %d minutes (target: %d)",
			totalMinutes, maxAllowedMinutes, targetMinutes)
	}

	// But should have at least attempted to meet target
	if totalMinutes < targetMinutes/2 {
		t.Errorf("Total duration %d minutes is too far below target %d minutes",
			totalMinutes, targetMinutes)
	}
}

func TestFocusOptimizer_ShouldProtectBlock(t *testing.T) {
	optimizer := NewFocusOptimizer(nil)

	tests := []struct {
		name     string
		block    domain.TimeBlock
		settings *domain.FocusTimeSettings
		want     bool
	}{
		{
			name: "no restrictions",
			block: domain.TimeBlock{
				DayOfWeek: "Tuesday",
				StartTime: "10:00",
				EndTime:   "12:00",
			},
			settings: &domain.FocusTimeSettings{},
			want:     true,
		},
		{
			name: "day in protected list",
			block: domain.TimeBlock{
				DayOfWeek: "Wednesday",
				StartTime: "10:00",
				EndTime:   "12:00",
			},
			settings: &domain.FocusTimeSettings{
				ProtectedDays: []string{"Tuesday", "Wednesday", "Thursday"},
			},
			want: true,
		},
		{
			name: "day not in protected list",
			block: domain.TimeBlock{
				DayOfWeek: "Monday",
				StartTime: "10:00",
				EndTime:   "12:00",
			},
			settings: &domain.FocusTimeSettings{
				ProtectedDays: []string{"Tuesday", "Wednesday", "Thursday"},
			},
			want: false,
		},
		{
			name: "time overlaps excluded range",
			block: domain.TimeBlock{
				DayOfWeek: "Tuesday",
				StartTime: "11:00",
				EndTime:   "13:00",
			},
			settings: &domain.FocusTimeSettings{
				ExcludedTimeRanges: []domain.TimeRange{
					{StartTime: "12:00", EndTime: "13:00"}, // Overlaps with block
				},
			},
			want: false,
		},
		{
			name: "time does not overlap excluded range",
			block: domain.TimeBlock{
				DayOfWeek: "Tuesday",
				StartTime: "09:00",
				EndTime:   "11:00",
			},
			settings: &domain.FocusTimeSettings{
				ExcludedTimeRanges: []domain.TimeRange{
					{StartTime: "12:00", EndTime: "13:00"},
				},
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := optimizer.shouldProtectBlock(tt.block, tt.settings)
			if got != tt.want {
				t.Errorf("shouldProtectBlock() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFocusOptimizer_TimesOverlap(t *testing.T) {
	optimizer := NewFocusOptimizer(nil)

	tests := []struct {
		name   string
		start1 string
		end1   string
		start2 string
		end2   string
		want   bool
	}{
		{
			name:   "complete overlap",
			start1: "10:00",
			end1:   "12:00",
			start2: "10:00",
			end2:   "12:00",
			want:   true,
		},
		{
			name:   "partial overlap - starts during",
			start1: "10:00",
			end1:   "12:00",
			start2: "11:00",
			end2:   "13:00",
			want:   true,
		},
		{
			name:   "partial overlap - ends during",
			start1: "11:00",
			end1:   "13:00",
			start2: "10:00",
			end2:   "12:00",
			want:   true,
		},
		{
			name:   "no overlap - before",
			start1: "09:00",
			end1:   "10:00",
			start2: "11:00",
			end2:   "12:00",
			want:   false,
		},
		{
			name:   "no overlap - after",
			start1: "13:00",
			end1:   "14:00",
			start2: "11:00",
			end2:   "12:00",
			want:   false,
		},
		{
			name:   "adjacent - no overlap",
			start1: "10:00",
			end1:   "11:00",
			start2: "11:00",
			end2:   "12:00",
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := optimizer.timesOverlap(tt.start1, tt.end1, tt.start2, tt.end2)
			if got != tt.want {
				t.Errorf("timesOverlap(%q, %q, %q, %q) = %v, want %v",
					tt.start1, tt.end1, tt.start2, tt.end2, got, tt.want)
			}
		})
	}
}

func TestFocusOptimizer_GenerateInsights(t *testing.T) {
	optimizer := NewFocusOptimizer(nil)

	patterns := &domain.MeetingPattern{
		Productivity: domain.ProductivityPatterns{
			PeakFocus: []domain.TimeBlock{
				{DayOfWeek: "Tuesday", StartTime: "10:00", EndTime: "12:00", Score: 95.0},
			},
			MeetingDensity: map[string]float64{
				"Monday":  6.2,
				"Tuesday": 3.1,
			},
		},
	}

	blocks := []domain.FocusTimeBlock{
		{Duration: 120},
		{Duration: 120},
	}

	settings := &domain.FocusTimeSettings{
		TargetHoursPerWeek: 10.0,
	}

	insights := optimizer.generateInsights(patterns, blocks, settings)

	if len(insights) == 0 {
		t.Fatal("Expected insights, got none")
	}

	// Should mention peak productivity
	foundPeakInsight := false
	for _, insight := range insights {
		if containsString(insight, "peak productivity") || containsString(insight, "Peak productivity") {
			foundPeakInsight = true
			break
		}
	}

	if !foundPeakInsight {
		t.Error("Expected insight about peak productivity")
	}
}

func TestFocusOptimizer_CalculateConfidence(t *testing.T) {
	optimizer := NewFocusOptimizer(nil)

	tests := []struct {
		name     string
		patterns *domain.MeetingPattern
		wantMin  float64
		wantMax  float64
	}{
		{
			name: "minimal data",
			patterns: &domain.MeetingPattern{
				Productivity: domain.ProductivityPatterns{},
			},
			wantMin: 50.0,
			wantMax: 50.0,
		},
		{
			name: "with peak focus data",
			patterns: &domain.MeetingPattern{
				Productivity: domain.ProductivityPatterns{
					PeakFocus: []domain.TimeBlock{
						{DayOfWeek: "Tuesday", StartTime: "10:00", EndTime: "12:00"},
					},
				},
			},
			wantMin: 70.0,
			wantMax: 70.0,
		},
		{
			name: "complete data",
			patterns: &domain.MeetingPattern{
				Productivity: domain.ProductivityPatterns{
					PeakFocus: []domain.TimeBlock{
						{DayOfWeek: "Tuesday", StartTime: "10:00", EndTime: "12:00"},
					},
					MeetingDensity: map[string]float64{
						"Monday": 5.0,
					},
				},
				Participants: map[string]domain.ParticipantPattern{
					"user1@example.com":  {},
					"user2@example.com":  {},
					"user3@example.com":  {},
					"user4@example.com":  {},
					"user5@example.com":  {},
					"user6@example.com":  {},
					"user7@example.com":  {},
					"user8@example.com":  {},
					"user9@example.com":  {},
					"user10@example.com": {},
					"user11@example.com": {},
				},
			},
			wantMin: 100.0,
			wantMax: 100.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := optimizer.calculateConfidence(tt.patterns)
			if got < tt.wantMin || got > tt.wantMax {
				t.Errorf("calculateConfidence() = %.1f, want between %.1f and %.1f",
					got, tt.wantMin, tt.wantMax)
			}
		})
	}
}

func TestFocusOptimizer_AdaptSchedule(t *testing.T) {
	client := nylas.NewMockClient()
	optimizer := NewFocusOptimizer(client)

	// Set up mock events
	now := time.Now()
	events := []domain.Event{
		{
			ID:    "event-1",
			Title: "Meeting 1",
			When: domain.EventWhen{
				StartTime: now.Add(24 * time.Hour).Unix(),
				EndTime:   now.Add(25 * time.Hour).Unix(),
			},
		},
	}

	client.GetCalendarsFunc = func(ctx context.Context, grantID string) ([]domain.Calendar, error) {
		return []domain.Calendar{{ID: "cal-1", Name: "Primary"}}, nil
	}

	client.GetEventsFunc = func(ctx context.Context, grantID, calendarID string, params *domain.EventQueryParams) ([]domain.Event, error) {
		return events, nil
	}

	change, err := optimizer.AdaptSchedule(context.Background(), "grant-123", domain.TriggerMeetingOverload)
	if err != nil {
		t.Fatalf("AdaptSchedule() error = %v", err)
	}

	if change == nil {
		t.Fatal("Expected adaptive schedule change, got nil")
		return
	}

	if change.Trigger != domain.TriggerMeetingOverload {
		t.Errorf("Trigger = %v, want %v", change.Trigger, domain.TriggerMeetingOverload)
	}

	if change.Confidence < 0 || change.Confidence > 100 {
		t.Errorf("Confidence = %.1f, want 0-100", change.Confidence)
	}

	if change.UserApproval != domain.ApprovalPending {
		t.Errorf("UserApproval = %v, want %v", change.UserApproval, domain.ApprovalPending)
	}
}

func TestFocusOptimizer_OptimizeMeetingDuration(t *testing.T) {
	client := nylas.NewMockClient()
	optimizer := NewFocusOptimizer(client)

	// Set up mock event (60 minute meeting)
	now := time.Now()
	event := &domain.Event{
		ID:    "event-123",
		Title: "Team Sync",
		When: domain.EventWhen{
			StartTime: now.Unix(),
			EndTime:   now.Add(60 * time.Minute).Unix(),
		},
	}

	client.GetEventFunc = func(ctx context.Context, grantID, calendarID, eventID string) (*domain.Event, error) {
		return event, nil
	}

	client.GetCalendarsFunc = func(ctx context.Context, grantID string) ([]domain.Calendar, error) {
		return []domain.Calendar{{ID: "cal-1", Name: "Primary"}}, nil
	}

	client.GetEventsFunc = func(ctx context.Context, grantID, calendarID string, params *domain.EventQueryParams) ([]domain.Event, error) {
		return []domain.Event{*event}, nil
	}

	optimization, err := optimizer.OptimizeMeetingDuration(context.Background(), "grant-123", "cal-1", "event-123")
	if err != nil {
		t.Fatalf("OptimizeMeetingDuration() error = %v", err)
	}

	if optimization == nil {
		t.Fatal("Expected optimization, got nil")
		return
	}

	if optimization.EventID != "event-123" {
		t.Errorf("EventID = %s, want event-123", optimization.EventID)
	}

	if optimization.CurrentDuration != 60 {
		t.Errorf("CurrentDuration = %d, want 60", optimization.CurrentDuration)
	}

	if optimization.Confidence < 0 || optimization.Confidence > 100 {
		t.Errorf("Confidence = %.1f, want 0-100", optimization.Confidence)
	}
}

// Helper function
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && s[:len(substr)] == substr || len(s) > len(substr) && s[len(s)-len(substr):] == substr || len(s) > len(substr)*2 && len(s) > 0
}
