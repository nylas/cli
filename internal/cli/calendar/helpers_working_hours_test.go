package calendar

import (
	"testing"
	"time"

	"github.com/nylas/cli/internal/domain"
)

func TestCheckBreakViolation(t *testing.T) {
	tests := []struct {
		name          string
		eventTime     time.Time
		config        *domain.Config
		wantViolation bool
		wantContains  string
	}{
		{
			name:          "no config - no violation",
			eventTime:     time.Date(2025, 1, 15, 12, 30, 0, 0, time.UTC), // 12:30 PM
			config:        nil,
			wantViolation: false,
		},
		{
			name:      "no breaks configured - no violation",
			eventTime: time.Date(2025, 1, 15, 12, 30, 0, 0, time.UTC), // 12:30 PM (Wednesday)
			config: &domain.Config{
				WorkingHours: &domain.WorkingHoursConfig{
					Default: &domain.DaySchedule{
						Enabled: true,
						Start:   "09:00",
						End:     "17:00",
						Breaks:  nil, // No breaks
					},
				},
			},
			wantViolation: false,
		},
		{
			name:      "event outside break time - no violation",
			eventTime: time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC), // 10:00 AM (before lunch)
			config: &domain.Config{
				WorkingHours: &domain.WorkingHoursConfig{
					Default: &domain.DaySchedule{
						Enabled: true,
						Start:   "09:00",
						End:     "17:00",
						Breaks: []domain.BreakBlock{
							{Name: "Lunch", Start: "12:00", End: "13:00", Type: "lunch"},
						},
					},
				},
			},
			wantViolation: false,
		},
		{
			name:      "event during lunch break - violation",
			eventTime: time.Date(2025, 1, 15, 12, 30, 0, 0, time.UTC), // 12:30 PM (during lunch)
			config: &domain.Config{
				WorkingHours: &domain.WorkingHoursConfig{
					Default: &domain.DaySchedule{
						Enabled: true,
						Start:   "09:00",
						End:     "17:00",
						Breaks: []domain.BreakBlock{
							{Name: "Lunch", Start: "12:00", End: "13:00", Type: "lunch"},
						},
					},
				},
			},
			wantViolation: true,
			wantContains:  "Lunch",
		},
		{
			name:      "event at break start time - violation",
			eventTime: time.Date(2025, 1, 15, 12, 0, 0, 0, time.UTC), // Exactly 12:00 PM
			config: &domain.Config{
				WorkingHours: &domain.WorkingHoursConfig{
					Default: &domain.DaySchedule{
						Enabled: true,
						Start:   "09:00",
						End:     "17:00",
						Breaks: []domain.BreakBlock{
							{Name: "Lunch", Start: "12:00", End: "13:00", Type: "lunch"},
						},
					},
				},
			},
			wantViolation: true,
			wantContains:  "Lunch",
		},
		{
			name:      "event at break end time - no violation",
			eventTime: time.Date(2025, 1, 15, 13, 0, 0, 0, time.UTC), // Exactly 1:00 PM (after break)
			config: &domain.Config{
				WorkingHours: &domain.WorkingHoursConfig{
					Default: &domain.DaySchedule{
						Enabled: true,
						Start:   "09:00",
						End:     "17:00",
						Breaks: []domain.BreakBlock{
							{Name: "Lunch", Start: "12:00", End: "13:00", Type: "lunch"},
						},
					},
				},
			},
			wantViolation: false,
		},
		{
			name:      "event during coffee break - violation",
			eventTime: time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC), // 10:30 AM
			config: &domain.Config{
				WorkingHours: &domain.WorkingHoursConfig{
					Default: &domain.DaySchedule{
						Enabled: true,
						Start:   "09:00",
						End:     "17:00",
						Breaks: []domain.BreakBlock{
							{Name: "Coffee Break", Start: "10:30", End: "10:45", Type: "coffee"},
						},
					},
				},
			},
			wantViolation: true,
			wantContains:  "Coffee Break",
		},
		{
			name:      "multiple breaks - violation in second break",
			eventTime: time.Date(2025, 1, 15, 15, 10, 0, 0, time.UTC), // 3:10 PM
			config: &domain.Config{
				WorkingHours: &domain.WorkingHoursConfig{
					Default: &domain.DaySchedule{
						Enabled: true,
						Start:   "09:00",
						End:     "17:00",
						Breaks: []domain.BreakBlock{
							{Name: "Lunch", Start: "12:00", End: "13:00", Type: "lunch"},
							{Name: "Afternoon Break", Start: "15:00", End: "15:15", Type: "coffee"},
						},
					},
				},
			},
			wantViolation: true,
			wantContains:  "Afternoon Break",
		},
		{
			name:      "multiple breaks - between breaks no violation",
			eventTime: time.Date(2025, 1, 15, 14, 0, 0, 0, time.UTC), // 2:00 PM (between lunch and afternoon break)
			config: &domain.Config{
				WorkingHours: &domain.WorkingHoursConfig{
					Default: &domain.DaySchedule{
						Enabled: true,
						Start:   "09:00",
						End:     "17:00",
						Breaks: []domain.BreakBlock{
							{Name: "Lunch", Start: "12:00", End: "13:00", Type: "lunch"},
							{Name: "Afternoon Break", Start: "15:00", End: "15:15", Type: "coffee"},
						},
					},
				},
			},
			wantViolation: false,
		},
		{
			name:      "day-specific break - Monday",
			eventTime: time.Date(2025, 1, 20, 11, 30, 0, 0, time.UTC), // Monday 11:30 AM
			config: &domain.Config{
				WorkingHours: &domain.WorkingHoursConfig{
					Monday: &domain.DaySchedule{
						Enabled: true,
						Start:   "09:00",
						End:     "17:00",
						Breaks: []domain.BreakBlock{
							{Name: "Early Lunch", Start: "11:00", End: "12:00", Type: "lunch"}, // Monday has early lunch
						},
					},
				},
			},
			wantViolation: true,
			wantContains:  "Early Lunch",
		},
		{
			name:      "invalid break config - ignored",
			eventTime: time.Date(2025, 1, 15, 12, 30, 0, 0, time.UTC),
			config: &domain.Config{
				WorkingHours: &domain.WorkingHoursConfig{
					Default: &domain.DaySchedule{
						Enabled: true,
						Start:   "09:00",
						End:     "17:00",
						Breaks: []domain.BreakBlock{
							{Name: "Invalid", Start: "invalid", End: "13:00"}, // Invalid start time
						},
					},
				},
			},
			wantViolation: false, // Invalid config is skipped
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := checkBreakViolation(tt.eventTime, tt.config)

			if tt.wantViolation {
				if result == "" {
					t.Error("Expected violation message, got empty string")
					return
				}
				if tt.wantContains != "" {
					// Check if result contains expected string
					found := false
					for i := 0; i <= len(result)-len(tt.wantContains); i++ {
						if result[i:i+len(tt.wantContains)] == tt.wantContains {
							found = true
							break
						}
					}
					if !found {
						t.Errorf("checkBreakViolation() = %q, want to contain %q", result, tt.wantContains)
					}
				}
			} else {
				if result != "" {
					t.Errorf("Expected no violation, got: %q", result)
				}
			}
		})
	}
}

func TestCheckWorkingHoursViolation(t *testing.T) {
	tests := []struct {
		name          string
		eventTime     time.Time
		config        *domain.Config
		wantViolation bool
		wantContains  string
	}{
		{
			name:          "no config - no violation",
			eventTime:     time.Date(2025, 1, 15, 20, 0, 0, 0, time.UTC), // 8 PM
			config:        nil,
			wantViolation: false,
		},
		{
			name:      "no working hours config - no violation",
			eventTime: time.Date(2025, 1, 15, 20, 0, 0, 0, time.UTC), // 8 PM
			config: &domain.Config{
				WorkingHours: nil,
			},
			wantViolation: false,
		},
		{
			name:      "working hours disabled - no violation",
			eventTime: time.Date(2025, 1, 15, 20, 0, 0, 0, time.UTC), // 8 PM (Wednesday)
			config: &domain.Config{
				WorkingHours: &domain.WorkingHoursConfig{
					Default: &domain.DaySchedule{
						Enabled: false,
						Start:   "09:00",
						End:     "17:00",
					},
				},
			},
			wantViolation: false,
		},
		{
			name:      "event before working hours - violation",
			eventTime: time.Date(2025, 1, 15, 8, 0, 0, 0, time.UTC), // 8 AM (before 9 AM)
			config: &domain.Config{
				WorkingHours: &domain.WorkingHoursConfig{
					Default: &domain.DaySchedule{
						Enabled: true,
						Start:   "09:00",
						End:     "17:00",
					},
				},
			},
			wantViolation: true,
			wantContains:  "before start",
		},
		{
			name:      "event after working hours - violation",
			eventTime: time.Date(2025, 1, 15, 18, 0, 0, 0, time.UTC), // 6 PM (after 5 PM)
			config: &domain.Config{
				WorkingHours: &domain.WorkingHoursConfig{
					Default: &domain.DaySchedule{
						Enabled: true,
						Start:   "09:00",
						End:     "17:00",
					},
				},
			},
			wantViolation: true,
			wantContains:  "after end",
		},
		{
			name:      "event during working hours - no violation",
			eventTime: time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC), // 10 AM
			config: &domain.Config{
				WorkingHours: &domain.WorkingHoursConfig{
					Default: &domain.DaySchedule{
						Enabled: true,
						Start:   "09:00",
						End:     "17:00",
					},
				},
			},
			wantViolation: false,
		},
		{
			name:      "event exactly at start time - no violation",
			eventTime: time.Date(2025, 1, 15, 9, 0, 0, 0, time.UTC), // 9 AM
			config: &domain.Config{
				WorkingHours: &domain.WorkingHoursConfig{
					Default: &domain.DaySchedule{
						Enabled: true,
						Start:   "09:00",
						End:     "17:00",
					},
				},
			},
			wantViolation: false,
		},
		{
			name:      "event exactly at end time - violation",
			eventTime: time.Date(2025, 1, 15, 17, 0, 0, 0, time.UTC), // 5 PM (at boundary)
			config: &domain.Config{
				WorkingHours: &domain.WorkingHoursConfig{
					Default: &domain.DaySchedule{
						Enabled: true,
						Start:   "09:00",
						End:     "17:00",
					},
				},
			},
			wantViolation: true,
			wantContains:  "after end",
		},
		{
			name:      "event 1 minute before start - violation",
			eventTime: time.Date(2025, 1, 15, 8, 59, 0, 0, time.UTC), // 8:59 AM
			config: &domain.Config{
				WorkingHours: &domain.WorkingHoursConfig{
					Default: &domain.DaySchedule{
						Enabled: true,
						Start:   "09:00",
						End:     "17:00",
					},
				},
			},
			wantViolation: true,
			wantContains:  "before start",
		},
		{
			name:      "event 1 minute before end - no violation",
			eventTime: time.Date(2025, 1, 15, 16, 59, 0, 0, time.UTC), // 4:59 PM
			config: &domain.Config{
				WorkingHours: &domain.WorkingHoursConfig{
					Default: &domain.DaySchedule{
						Enabled: true,
						Start:   "09:00",
						End:     "17:00",
					},
				},
			},
			wantViolation: false,
		},
		{
			name:      "day-specific schedule - Monday early start",
			eventTime: time.Date(2025, 1, 20, 8, 30, 0, 0, time.UTC), // Monday 8:30 AM
			config: &domain.Config{
				WorkingHours: &domain.WorkingHoursConfig{
					Monday: &domain.DaySchedule{
						Enabled: true,
						Start:   "08:00", // Monday starts at 8 AM
						End:     "16:00",
					},
					Default: &domain.DaySchedule{
						Enabled: true,
						Start:   "09:00",
						End:     "17:00",
					},
				},
			},
			wantViolation: false, // Within Monday's working hours
		},
		{
			name:      "day-specific schedule - Tuesday uses default",
			eventTime: time.Date(2025, 1, 21, 8, 30, 0, 0, time.UTC), // Tuesday 8:30 AM
			config: &domain.Config{
				WorkingHours: &domain.WorkingHoursConfig{
					Monday: &domain.DaySchedule{
						Enabled: true,
						Start:   "08:00",
						End:     "16:00",
					},
					Default: &domain.DaySchedule{
						Enabled: true,
						Start:   "09:00",
						End:     "17:00",
					},
				},
			},
			wantViolation: true, // Before default start time
			wantContains:  "before start",
		},
		{
			name:      "invalid start time in config - skip validation",
			eventTime: time.Date(2025, 1, 15, 8, 0, 0, 0, time.UTC),
			config: &domain.Config{
				WorkingHours: &domain.WorkingHoursConfig{
					Default: &domain.DaySchedule{
						Enabled: true,
						Start:   "invalid",
						End:     "17:00",
					},
				},
			},
			wantViolation: false, // Invalid config is skipped
		},
		{
			name:      "invalid end time in config - skip validation",
			eventTime: time.Date(2025, 1, 15, 18, 0, 0, 0, time.UTC),
			config: &domain.Config{
				WorkingHours: &domain.WorkingHoursConfig{
					Default: &domain.DaySchedule{
						Enabled: true,
						Start:   "09:00",
						End:     "invalid",
					},
				},
			},
			wantViolation: false, // Invalid config is skipped
		},
		{
			name:      "very early morning - multiple hours before start",
			eventTime: time.Date(2025, 1, 15, 6, 30, 0, 0, time.UTC), // 6:30 AM (2.5 hours before)
			config: &domain.Config{
				WorkingHours: &domain.WorkingHoursConfig{
					Default: &domain.DaySchedule{
						Enabled: true,
						Start:   "09:00",
						End:     "17:00",
					},
				},
			},
			wantViolation: true,
			wantContains:  "2h 30m before start",
		},
		{
			name:      "late evening - multiple hours after end",
			eventTime: time.Date(2025, 1, 15, 20, 15, 0, 0, time.UTC), // 8:15 PM (3.25 hours after)
			config: &domain.Config{
				WorkingHours: &domain.WorkingHoursConfig{
					Default: &domain.DaySchedule{
						Enabled: true,
						Start:   "09:00",
						End:     "17:00",
					},
				},
			},
			wantViolation: true,
			wantContains:  "3h 15m after end",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := checkWorkingHoursViolation(tt.eventTime, tt.config)

			if tt.wantViolation {
				if result == "" {
					t.Error("Expected violation message, got empty string")
					return
				}
				if tt.wantContains != "" {
					// Check if result contains expected string
					found := false
					for i := 0; i <= len(result)-len(tt.wantContains); i++ {
						if result[i:i+len(tt.wantContains)] == tt.wantContains {
							found = true
							break
						}
					}
					if !found {
						t.Errorf("checkWorkingHoursViolation() = %q, want to contain %q", result, tt.wantContains)
					}
				}
			} else {
				if result != "" {
					t.Errorf("Expected no violation, got: %q", result)
				}
			}
		})
	}
}
