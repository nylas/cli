package timezone

import (
	"context"
	"testing"
	"time"
)

func TestService_GetTimeZoneInfo(t *testing.T) {
	service := NewService()
	ctx := context.Background()

	tests := []struct {
		name    string
		zone    string
		at      time.Time
		wantErr bool
	}{
		{
			name:    "New York January",
			zone:    "America/New_York",
			at:      time.Date(2025, 1, 15, 12, 0, 0, 0, time.UTC),
			wantErr: false,
		},
		{
			name:    "London summer",
			zone:    "Europe/London",
			at:      time.Date(2025, 7, 15, 12, 0, 0, 0, time.UTC),
			wantErr: false,
		},
		{
			name:    "UTC",
			zone:    "UTC",
			at:      time.Now(),
			wantErr: false,
		},
		{
			name:    "invalid zone",
			zone:    "Invalid/Zone",
			at:      time.Now(),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := service.GetTimeZoneInfo(ctx, tt.zone, tt.at)

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

			if result.Name != tt.zone {
				t.Errorf("Name = %q, want %q", result.Name, tt.zone)
			}

			if result.Abbreviation == "" {
				t.Error("abbreviation is empty")
			}

			// UTC offset should be reasonable (-12 to +14 hours)
			if result.Offset < -12*3600 || result.Offset > 14*3600 {
				t.Errorf("offset = %d is out of reasonable range", result.Offset)
			}
		})
	}
}

func TestService_CheckDSTWarning(t *testing.T) {
	service := NewService()
	ctx := context.Background()

	// Test time just before spring forward (March 2025 for US)
	loc, _ := time.LoadLocation("America/New_York")
	beforeSpring := time.Date(2025, 3, 8, 14, 0, 0, 0, loc)  // Day before DST
	springForward := time.Date(2025, 3, 9, 2, 30, 0, 0, loc) // During gap

	tests := []struct {
		name         string
		t            time.Time
		zone         string
		warningDays  int
		wantErr      bool
		wantWarning  bool
		wantSeverity string
	}{
		{
			name:        "near spring forward",
			t:           beforeSpring,
			zone:        "America/New_York",
			warningDays: 7,
			wantErr:     false,
			wantWarning: true,
		},
		{
			name:         "in spring forward gap",
			t:            springForward,
			zone:         "America/New_York",
			warningDays:  0,
			wantErr:      false,
			wantWarning:  true,
			wantSeverity: "error",
		},
		{
			name:        "no DST zone",
			t:           time.Now(),
			zone:        "America/Phoenix",
			warningDays: 7,
			wantErr:     false,
			wantWarning: false,
		},
		{
			name:        "UTC no DST",
			t:           time.Now(),
			zone:        "UTC",
			warningDays: 7,
			wantErr:     false,
			wantWarning: false,
		},
		{
			name:        "invalid zone",
			t:           time.Now(),
			zone:        "Invalid/Zone",
			warningDays: 7,
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := service.CheckDSTWarning(ctx, tt.t, tt.zone, tt.warningDays)

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

			if tt.wantWarning {
				if result == nil {
					t.Error("expected warning, got nil")
					return
				}
				if !result.IsNearTransition {
					t.Error("expected IsNearTransition = true")
				}
				if tt.wantSeverity != "" && result.Severity != tt.wantSeverity {
					t.Errorf("Severity = %q, want %q", result.Severity, tt.wantSeverity)
				}
			} else {
				if result != nil && result.IsNearTransition {
					t.Error("expected no warning, got warning")
				}
			}
		})
	}
}

func TestService_SuggestAlternativeTimes(t *testing.T) {
	service := NewService()
	ctx := context.Background()

	// Time during spring forward gap
	loc, _ := time.LoadLocation("America/New_York")
	springGap := time.Date(2025, 3, 9, 2, 30, 0, 0, loc)
	normalTime := time.Date(2025, 1, 15, 10, 0, 0, 0, loc)

	tests := []struct {
		name         string
		t            time.Time
		zone         string
		duration     time.Duration
		wantErr      bool
		wantAltCount int
	}{
		{
			name:         "spring forward gap",
			t:            springGap,
			zone:         "America/New_York",
			duration:     1 * time.Hour,
			wantErr:      false,
			wantAltCount: 2, // Before and after gap
		},
		{
			name:         "normal time no alternatives",
			t:            normalTime,
			zone:         "America/New_York",
			duration:     1 * time.Hour,
			wantErr:      false,
			wantAltCount: 0,
		},
		{
			name:     "invalid zone",
			t:        time.Now(),
			zone:     "Invalid/Zone",
			duration: 1 * time.Hour,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := service.SuggestAlternativeTimes(ctx, tt.t, tt.zone, tt.duration)

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

			if len(result) != tt.wantAltCount {
				t.Errorf("alternatives count = %d, want %d", len(result), tt.wantAltCount)
			}
		})
	}
}

func TestParseTime(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			name:    "valid 24h format",
			input:   "09:00",
			wantErr: false,
		},
		{
			name:    "valid midnight",
			input:   "00:00",
			wantErr: false,
		},
		{
			name:    "valid 23:59",
			input:   "23:59",
			wantErr: false,
		},
		{
			name:    "invalid format",
			input:   "9:00 AM",
			wantErr: true,
		},
		{
			name:    "invalid hour",
			input:   "25:00",
			wantErr: true,
		},
		{
			name:    "invalid minute",
			input:   "09:60",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parseTime(tt.input)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestGetOffset(t *testing.T) {
	tests := []struct {
		name string
		zone string
		t    time.Time
	}{
		{
			name: "UTC offset zero",
			zone: "UTC",
			t:    time.Date(2025, 1, 15, 12, 0, 0, 0, time.UTC),
		},
		{
			name: "EST offset negative",
			zone: "America/New_York",
			t:    time.Date(2025, 1, 15, 12, 0, 0, 0, time.UTC),
		},
		{
			name: "JST offset positive",
			zone: "Asia/Tokyo",
			t:    time.Date(2025, 1, 15, 12, 0, 0, 0, time.UTC),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			loc, _ := time.LoadLocation(tt.zone)
			timeInZone := tt.t.In(loc)
			offset := getOffset(timeInZone)

			// Offset should be reasonable (-12 to +14 hours in seconds)
			if offset < -12*3600 || offset > 14*3600 {
				t.Errorf("offset = %d is out of reasonable range", offset)
			}
		})
	}
}

func TestIsDST(t *testing.T) {
	tests := []struct {
		name    string
		zone    string
		t       time.Time
		wantDST bool
	}{
		{
			name:    "New York January not DST",
			zone:    "America/New_York",
			t:       time.Date(2025, 1, 15, 12, 0, 0, 0, time.UTC),
			wantDST: false,
		},
		{
			name:    "New York July is DST",
			zone:    "America/New_York",
			t:       time.Date(2025, 7, 15, 12, 0, 0, 0, time.UTC),
			wantDST: true,
		},
		{
			name:    "UTC never DST",
			zone:    "UTC",
			t:       time.Date(2025, 7, 15, 12, 0, 0, 0, time.UTC),
			wantDST: false,
		},
		{
			name:    "Phoenix never DST",
			zone:    "America/Phoenix",
			t:       time.Date(2025, 7, 15, 12, 0, 0, 0, time.UTC),
			wantDST: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			loc, _ := time.LoadLocation(tt.zone)
			timeInZone := tt.t.In(loc)
			result := isDST(timeInZone)

			if result != tt.wantDST {
				t.Errorf("isDST() = %v, want %v (abbreviation: %s)", result, tt.wantDST, timeInZone.Format("MST"))
			}
		})
	}
}
