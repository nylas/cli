package timezone

import (
	"context"
	"testing"
	"time"

	"github.com/nylas/cli/internal/domain"
)

func TestNewService(t *testing.T) {
	service := NewService()
	if service == nil {
		t.Fatal("NewService() returned nil")
	}
}

func TestService_ConvertTime(t *testing.T) {
	service := NewService()
	ctx := context.Background()

	tests := []struct {
		name      string
		fromZone  string
		toZone    string
		inputTime time.Time
		wantErr   bool
	}{
		{
			name:      "UTC to EST",
			fromZone:  "UTC",
			toZone:    "America/New_York",
			inputTime: time.Date(2025, 1, 15, 12, 0, 0, 0, time.UTC),
			wantErr:   false,
		},
		{
			name:      "PST to JST",
			fromZone:  "America/Los_Angeles",
			toZone:    "Asia/Tokyo",
			inputTime: time.Date(2025, 1, 15, 9, 0, 0, 0, time.UTC),
			wantErr:   false,
		},
		{
			name:      "same timezone",
			fromZone:  "UTC",
			toZone:    "UTC",
			inputTime: time.Date(2025, 1, 15, 12, 0, 0, 0, time.UTC),
			wantErr:   false,
		},
		{
			name:      "invalid from zone",
			fromZone:  "Invalid/Zone",
			toZone:    "UTC",
			inputTime: time.Now(),
			wantErr:   true,
		},
		{
			name:      "invalid to zone",
			fromZone:  "UTC",
			toZone:    "Invalid/Zone",
			inputTime: time.Now(),
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := service.ConvertTime(ctx, tt.fromZone, tt.toZone, tt.inputTime)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			// Verify result is a valid time
			if result.IsZero() {
				t.Error("result time is zero")
			}

			// For same timezone, time should be identical
			if tt.fromZone == tt.toZone {
				if !result.Equal(tt.inputTime) {
					t.Errorf("same timezone conversion: got %v, want %v", result, tt.inputTime)
				}
			}
		})
	}
}

func TestService_FindMeetingTime(t *testing.T) {
	service := NewService()
	ctx := context.Background()
	startDate := time.Date(2026, 1, 12, 0, 0, 0, 0, time.UTC)
	endDate := startDate.AddDate(0, 0, 2)

	tests := []struct {
		name         string
		req          *domain.MeetingFinderRequest
		wantErr      bool
		minSlots     int
		expectNoSlot bool
	}{
		{
			name: "valid request single zone",
			req: &domain.MeetingFinderRequest{
				TimeZones:         []string{"America/New_York"},
				Duration:          time.Hour,
				WorkingHoursStart: "09:00",
				WorkingHoursEnd:   "17:00",
				DateRange: domain.DateRange{
					Start: startDate,
					End:   endDate,
				},
			},
			wantErr:  false,
			minSlots: 1,
		},
		{
			name: "valid request overlapping zones",
			req: &domain.MeetingFinderRequest{
				TimeZones:         []string{"America/New_York", "Europe/London"},
				Duration:          time.Hour,
				WorkingHoursStart: "09:00",
				WorkingHoursEnd:   "17:00",
				DateRange: domain.DateRange{
					Start: startDate,
					End:   endDate,
				},
			},
			wantErr:  false,
			minSlots: 1,
		},
		{
			name: "single zone with longer duration",
			req: &domain.MeetingFinderRequest{
				TimeZones:         []string{"America/New_York"},
				Duration:          2 * time.Hour,
				WorkingHoursStart: "09:00",
				WorkingHoursEnd:   "17:00",
				DateRange: domain.DateRange{
					Start: startDate,
					End:   startDate,
				},
			},
			wantErr:  false,
			minSlots: 1,
		},
		{
			name: "no overlap across distant zones",
			req: &domain.MeetingFinderRequest{
				TimeZones:         []string{"America/Los_Angeles", "Asia/Tokyo"},
				Duration:          2 * time.Hour,
				WorkingHoursStart: "09:00",
				WorkingHoursEnd:   "17:00",
				DateRange: domain.DateRange{
					Start: startDate,
					End:   startDate,
				},
			},
			wantErr:      false,
			expectNoSlot: true,
		},
		{
			name: "no timezones",
			req: &domain.MeetingFinderRequest{
				TimeZones:         []string{},
				Duration:          time.Hour,
				WorkingHoursStart: "09:00",
				WorkingHoursEnd:   "17:00",
			},
			wantErr: true,
		},
		{
			name: "invalid working hours start",
			req: &domain.MeetingFinderRequest{
				TimeZones:         []string{"UTC"},
				Duration:          time.Hour,
				WorkingHoursStart: "invalid",
				WorkingHoursEnd:   "17:00",
			},
			wantErr: true,
		},
		{
			name: "invalid working hours end",
			req: &domain.MeetingFinderRequest{
				TimeZones:         []string{"UTC"},
				Duration:          time.Hour,
				WorkingHoursStart: "09:00",
				WorkingHoursEnd:   "invalid",
			},
			wantErr: true,
		},
		{
			name: "invalid timezone",
			req: &domain.MeetingFinderRequest{
				TimeZones:         []string{"Invalid/Zone"},
				Duration:          time.Hour,
				WorkingHoursStart: "09:00",
				WorkingHoursEnd:   "17:00",
			},
			wantErr: true,
		},
		{
			name: "invalid duration",
			req: &domain.MeetingFinderRequest{
				TimeZones:         []string{"UTC"},
				Duration:          0,
				WorkingHoursStart: "09:00",
				WorkingHoursEnd:   "17:00",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := service.FindMeetingTime(ctx, tt.req)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if result == nil {
				t.Fatal("result is nil")
				return
			}

			if len(result.TimeZones) != len(tt.req.TimeZones) {
				t.Errorf("TimeZones count = %d, want %d", len(result.TimeZones), len(tt.req.TimeZones))
			}

			if tt.expectNoSlot {
				if len(result.Slots) != 0 {
					t.Fatalf("expected no slots, got %d", len(result.Slots))
				}
				return
			}

			if len(result.Slots) < tt.minSlots {
				t.Fatalf("expected at least %d slot(s), got %d", tt.minSlots, len(result.Slots))
			}

			for _, slot := range result.Slots {
				if !slot.EndTime.After(slot.StartTime) {
					t.Fatalf("slot end %v must be after start %v", slot.EndTime, slot.StartTime)
				}
				if len(slot.Times) != len(tt.req.TimeZones) {
					t.Fatalf("slot times count = %d, want %d", len(slot.Times), len(tt.req.TimeZones))
				}
			}
		})
	}
}

func TestService_FindMeetingTime_RespectsWorkingHourMinutes(t *testing.T) {
	service := NewService()
	ctx := context.Background()

	req := &domain.MeetingFinderRequest{
		TimeZones:         []string{"America/New_York"},
		Duration:          30 * time.Minute,
		WorkingHoursStart: "09:30",
		WorkingHoursEnd:   "10:30",
		DateRange: domain.DateRange{
			Start: time.Date(2026, 1, 12, 0, 0, 0, 0, time.UTC),
			End:   time.Date(2026, 1, 12, 0, 0, 0, 0, time.UTC),
		},
	}

	result, err := service.FindMeetingTime(ctx, req)
	if err != nil {
		t.Fatalf("FindMeetingTime() error = %v", err)
	}
	if len(result.Slots) == 0 {
		t.Fatal("expected at least one slot")
	}

	for _, slot := range result.Slots {
		local := slot.StartTime.In(mustLoadLocation(t, "America/New_York"))
		minutes := local.Hour()*60 + local.Minute()
		if minutes < 9*60+30 || minutes >= 10*60+30 {
			t.Fatalf("slot %v falls outside 09:30-10:30 working window", local)
		}
	}
}

func TestService_GetDSTTransitions(t *testing.T) {
	service := NewService()
	ctx := context.Background()

	tests := []struct {
		name    string
		zone    string
		year    int
		wantErr bool
		wantMin int // Minimum expected transitions
		wantMax int // Maximum expected transitions
	}{
		{
			name:    "New York 2025",
			zone:    "America/New_York",
			year:    2025,
			wantErr: false,
			wantMin: 2, // Spring forward, fall back
			wantMax: 2,
		},
		{
			name:    "Phoenix no DST",
			zone:    "America/Phoenix",
			year:    2025,
			wantErr: false,
			wantMin: 0, // Arizona doesn't observe DST
			wantMax: 0,
		},
		{
			name:    "London 2025",
			zone:    "Europe/London",
			year:    2025,
			wantErr: false,
			wantMin: 2, // BST transitions
			wantMax: 2,
		},
		{
			name:    "invalid zone",
			zone:    "Invalid/Zone",
			year:    2025,
			wantErr: true,
		},
		{
			name:    "UTC no DST",
			zone:    "UTC",
			year:    2025,
			wantErr: false,
			wantMin: 0,
			wantMax: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := service.GetDSTTransitions(ctx, tt.zone, tt.year)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if len(result) < tt.wantMin || len(result) > tt.wantMax {
				t.Errorf("transitions count = %d, want between %d and %d", len(result), tt.wantMin, tt.wantMax)
			}

			// Verify each transition has required fields
			for i, trans := range result {
				if trans.Date.IsZero() {
					t.Errorf("transition[%d] has zero date", i)
				}
				if trans.Name == "" {
					t.Errorf("transition[%d] has empty name", i)
				}
				if trans.Direction != "forward" && trans.Direction != "backward" {
					t.Errorf("transition[%d] has invalid direction %q", i, trans.Direction)
				}
			}
		})
	}
}

func TestService_ListTimeZones(t *testing.T) {
	service := NewService()
	ctx := context.Background()

	zones, err := service.ListTimeZones(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(zones) == 0 {
		t.Error("expected non-empty zone list")
	}

	// Check for some common zones
	expectedZones := []string{
		"America/New_York",
		"America/Los_Angeles",
		"Europe/London",
		"Asia/Tokyo",
		"UTC",
	}

	for _, expected := range expectedZones {
		found := false
		for _, zone := range zones {
			if zone == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected zone %q not found in list", expected)
		}
	}

	// Verify zones are sorted
	for i := 1; i < len(zones); i++ {
		if zones[i-1] > zones[i] {
			t.Errorf("zones not sorted: %q > %q", zones[i-1], zones[i])
			break
		}
	}
}

func mustLoadLocation(t *testing.T, zone string) *time.Location {
	t.Helper()

	loc, err := time.LoadLocation(zone)
	if err != nil {
		t.Fatalf("LoadLocation(%q) error = %v", zone, err)
	}

	return loc
}
