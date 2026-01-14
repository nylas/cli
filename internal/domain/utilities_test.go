package domain

import (
	"testing"
	"time"
)

// =============================================================================
// DateRange Tests
// =============================================================================

func TestDateRange_Creation(t *testing.T) {
	now := time.Now()
	dr := DateRange{
		Start: now.AddDate(0, 0, -7),
		End:   now,
	}

	if dr.Start.IsZero() {
		t.Error("DateRange.Start should not be zero")
	}
	if dr.End.IsZero() {
		t.Error("DateRange.End should not be zero")
	}
	if dr.Start.After(dr.End) {
		t.Error("DateRange.Start should be before End")
	}
}

// =============================================================================
// MeetingFinderRequest Tests
// =============================================================================

func TestMeetingFinderRequest_Creation(t *testing.T) {
	now := time.Now()
	req := MeetingFinderRequest{
		TimeZones:         []string{"America/New_York", "Europe/London", "Asia/Tokyo"},
		Duration:          time.Hour,
		WorkingHoursStart: "09:00",
		WorkingHoursEnd:   "17:00",
		DateRange: DateRange{
			Start: now,
			End:   now.AddDate(0, 0, 7),
		},
		ExcludeWeekends: true,
	}

	if len(req.TimeZones) != 3 {
		t.Errorf("MeetingFinderRequest.TimeZones length = %d, want 3", len(req.TimeZones))
	}
	if req.Duration != time.Hour {
		t.Errorf("MeetingFinderRequest.Duration = %v, want 1h", req.Duration)
	}
	if req.WorkingHoursStart != "09:00" {
		t.Errorf("MeetingFinderRequest.WorkingHoursStart = %q, want %q", req.WorkingHoursStart, "09:00")
	}
	if !req.ExcludeWeekends {
		t.Error("MeetingFinderRequest.ExcludeWeekends should be true")
	}
}

// =============================================================================
// MeetingTimeSlots Tests
// =============================================================================

func TestMeetingTimeSlots_Creation(t *testing.T) {
	now := time.Now()
	slots := MeetingTimeSlots{
		Slots: []MeetingSlot{
			{
				StartTime: now,
				EndTime:   now.Add(time.Hour),
				Times: map[string]time.Time{
					"America/New_York": now,
					"Europe/London":    now.Add(5 * time.Hour),
				},
				Score: 0.95,
			},
		},
		TimeZones:  []string{"America/New_York", "Europe/London"},
		TotalSlots: 1,
	}

	if len(slots.Slots) != 1 {
		t.Errorf("MeetingTimeSlots.Slots length = %d, want 1", len(slots.Slots))
	}
	if slots.TotalSlots != 1 {
		t.Errorf("MeetingTimeSlots.TotalSlots = %d, want 1", slots.TotalSlots)
	}
	if slots.Slots[0].Score != 0.95 {
		t.Errorf("MeetingSlot.Score = %f, want 0.95", slots.Slots[0].Score)
	}
}

// =============================================================================
// MeetingSlot Tests
// =============================================================================

func TestMeetingSlot_Creation(t *testing.T) {
	now := time.Now()
	slot := MeetingSlot{
		StartTime: now,
		EndTime:   now.Add(30 * time.Minute),
		Times: map[string]time.Time{
			"UTC":              now,
			"America/New_York": now.Add(-5 * time.Hour),
		},
		Score: 0.85,
	}

	if slot.Score != 0.85 {
		t.Errorf("MeetingSlot.Score = %f, want 0.85", slot.Score)
	}
	if len(slot.Times) != 2 {
		t.Errorf("MeetingSlot.Times length = %d, want 2", len(slot.Times))
	}
}

// =============================================================================
// DSTTransition Tests
// =============================================================================

func TestDSTTransition_Creation(t *testing.T) {
	tests := []struct {
		name       string
		transition DSTTransition
	}{
		{
			name: "spring forward",
			transition: DSTTransition{
				Date:      time.Date(2024, 3, 10, 2, 0, 0, 0, time.UTC),
				Offset:    -7 * 3600,
				Name:      "PDT",
				IsDST:     true,
				Direction: "forward",
			},
		},
		{
			name: "fall back",
			transition: DSTTransition{
				Date:      time.Date(2024, 11, 3, 2, 0, 0, 0, time.UTC),
				Offset:    -8 * 3600,
				Name:      "PST",
				IsDST:     false,
				Direction: "backward",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.transition.Name == "" {
				t.Error("DSTTransition.Name should not be empty")
			}
			if tt.transition.Direction != "forward" && tt.transition.Direction != "backward" {
				t.Errorf("DSTTransition.Direction = %q, want 'forward' or 'backward'", tt.transition.Direction)
			}
		})
	}
}

// =============================================================================
// TimeZoneInfo Tests
// =============================================================================

func TestTimeZoneInfo_Creation(t *testing.T) {
	nextDST := time.Date(2024, 3, 10, 2, 0, 0, 0, time.UTC)
	info := TimeZoneInfo{
		Name:         "America/Los_Angeles",
		Abbreviation: "PST",
		Offset:       -8 * 3600,
		IsDST:        false,
		NextDST:      &nextDST,
	}

	if info.Name != "America/Los_Angeles" {
		t.Errorf("TimeZoneInfo.Name = %q, want %q", info.Name, "America/Los_Angeles")
	}
	if info.Abbreviation != "PST" {
		t.Errorf("TimeZoneInfo.Abbreviation = %q, want %q", info.Abbreviation, "PST")
	}
	if info.Offset != -8*3600 {
		t.Errorf("TimeZoneInfo.Offset = %d, want %d", info.Offset, -8*3600)
	}
	if info.IsDST {
		t.Error("TimeZoneInfo.IsDST should be false")
	}
	if info.NextDST == nil {
		t.Error("TimeZoneInfo.NextDST should not be nil")
	}
}

// =============================================================================
// DSTWarning Tests
// =============================================================================

func TestDSTWarning_Creation(t *testing.T) {
	tests := []struct {
		name    string
		warning DSTWarning
	}{
		{
			name: "near transition warning",
			warning: DSTWarning{
				IsNearTransition: true,
				TransitionDate:   time.Date(2024, 3, 10, 2, 0, 0, 0, time.UTC),
				Direction:        "forward",
				DaysUntil:        3,
				TransitionName:   "PDT",
				InTransitionGap:  false,
				InDuplicateHour:  false,
				Warning:          "DST transition in 3 days",
				Severity:         "warning",
			},
		},
		{
			name: "in transition gap error",
			warning: DSTWarning{
				IsNearTransition: true,
				TransitionDate:   time.Date(2024, 3, 10, 2, 0, 0, 0, time.UTC),
				Direction:        "forward",
				DaysUntil:        0,
				InTransitionGap:  true,
				InDuplicateHour:  false,
				Warning:          "Time does not exist (spring forward gap)",
				Severity:         "error",
			},
		},
		{
			name: "in duplicate hour warning",
			warning: DSTWarning{
				IsNearTransition: true,
				TransitionDate:   time.Date(2024, 11, 3, 1, 0, 0, 0, time.UTC),
				Direction:        "backward",
				DaysUntil:        0,
				InTransitionGap:  false,
				InDuplicateHour:  true,
				Warning:          "Time occurs twice (fall back)",
				Severity:         "warning",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.warning.Warning == "" {
				t.Error("DSTWarning.Warning should not be empty")
			}
			if tt.warning.Severity != "error" && tt.warning.Severity != "warning" && tt.warning.Severity != "info" {
				t.Errorf("DSTWarning.Severity = %q, want 'error', 'warning', or 'info'", tt.warning.Severity)
			}
		})
	}
}
