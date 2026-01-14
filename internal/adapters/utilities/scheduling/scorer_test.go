package scheduling

import (
	"testing"
	"time"
)

func TestScoreTimeSlot(t *testing.T) {
	// Reference time: Tuesday, Jan 7, 2025, 10:00 AM PST
	loc, _ := time.LoadLocation("America/Los_Angeles")
	baseTime := time.Date(2025, 1, 7, 10, 0, 0, 0, loc)

	tests := []struct {
		name           string
		startTime      time.Time
		endTime        time.Time
		participants   []ParticipantTime
		wantMinScore   float64
		wantMaxScore   float64
		wantWorkingMin float64
		wantQualityMin float64
	}{
		{
			name:      "perfect time - tuesday 10am all zones",
			startTime: baseTime,
			endTime:   baseTime.Add(1 * time.Hour),
			participants: []ParticipantTime{
				{TimeZone: "America/Los_Angeles", LocalTime: baseTime, IsWorking: true},                 // 10am - excellent (25)
				{TimeZone: "America/New_York", LocalTime: baseTime.Add(3 * time.Hour), IsWorking: true}, // 1pm - good (20)
				{TimeZone: "Europe/London", LocalTime: baseTime.Add(8 * time.Hour), IsWorking: true},    // 6pm - bad (0)
			},
			wantMinScore:   85, // Good score despite one bad time
			wantMaxScore:   95,
			wantWorkingMin: 40, // All in working hours
			wantQualityMin: 10, // Average: (25+20+0)/3 = 15
		},
		{
			name:      "one participant outside working hours",
			startTime: baseTime,
			endTime:   baseTime.Add(1 * time.Hour),
			participants: []ParticipantTime{
				{TimeZone: "America/Los_Angeles", LocalTime: baseTime, IsWorking: true},
				{TimeZone: "America/New_York", LocalTime: baseTime.Add(3 * time.Hour), IsWorking: true},
				{TimeZone: "Asia/Tokyo", LocalTime: baseTime.Add(17 * time.Hour), IsWorking: false}, // 3am next day
			},
			wantMinScore:   70, // Reduced due to one outside working hours
			wantMaxScore:   80,
			wantWorkingMin: 26, // 2/3 in working hours
			wantQualityMin: 0,  // Average pulled down by bad time
		},
		{
			name:      "friday afternoon penalty",
			startTime: time.Date(2025, 1, 10, 15, 0, 0, 0, loc), // Friday 3pm
			endTime:   time.Date(2025, 1, 10, 16, 0, 0, 0, loc),
			participants: []ParticipantTime{
				{TimeZone: "America/Los_Angeles", LocalTime: time.Date(2025, 1, 10, 15, 0, 0, 0, loc), IsWorking: true},
			},
			wantMinScore:   75, // Penalized for Friday PM
			wantMaxScore:   85,
			wantWorkingMin: 40, // In working hours
			wantQualityMin: 10, // Fair time quality (15)
		},
		{
			name:      "monday early morning penalty",
			startTime: time.Date(2025, 1, 6, 9, 0, 0, 0, loc), // Monday 9am
			endTime:   time.Date(2025, 1, 6, 10, 0, 0, 0, loc),
			participants: []ParticipantTime{
				{TimeZone: "America/Los_Angeles", LocalTime: time.Date(2025, 1, 6, 9, 0, 0, 0, loc), IsWorking: true},
			},
			wantMinScore:   90, // Slightly penalized for Monday early
			wantMaxScore:   100,
			wantWorkingMin: 40, // In working hours
			wantQualityMin: 20, // Excellent time quality (25)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			breakdown := ScoreTimeSlot(tt.startTime, tt.endTime, tt.participants)

			// Check total score range
			if breakdown.Total < tt.wantMinScore {
				t.Errorf("Total score = %.2f, want >= %.2f", breakdown.Total, tt.wantMinScore)
			}
			if breakdown.Total > tt.wantMaxScore {
				t.Errorf("Total score = %.2f, want <= %.2f", breakdown.Total, tt.wantMaxScore)
			}

			// Check working hours score
			if breakdown.WorkingHours < tt.wantWorkingMin {
				t.Errorf("WorkingHours score = %.2f, want >= %.2f", breakdown.WorkingHours, tt.wantWorkingMin)
			}

			// Check time quality score
			if breakdown.TimeQuality < tt.wantQualityMin {
				t.Errorf("TimeQuality score = %.2f, want >= %.2f", breakdown.TimeQuality, tt.wantQualityMin)
			}

			// Validate score is between 0 and 100
			if breakdown.Total < 0 || breakdown.Total > 100 {
				t.Errorf("Total score %.2f is out of valid range [0, 100]", breakdown.Total)
			}
		})
	}
}

func TestScoreWorkingHours(t *testing.T) {
	tests := []struct {
		name         string
		participants []ParticipantTime
		wantScore    float64
	}{
		{
			name: "all participants in working hours",
			participants: []ParticipantTime{
				{IsWorking: true},
				{IsWorking: true},
				{IsWorking: true},
			},
			wantScore: 40.0,
		},
		{
			name: "2 out of 3 in working hours",
			participants: []ParticipantTime{
				{IsWorking: true},
				{IsWorking: true},
				{IsWorking: false},
			},
			wantScore: 26.67, // 2/3 * 40
		},
		{
			name: "1 out of 3 in working hours",
			participants: []ParticipantTime{
				{IsWorking: true},
				{IsWorking: false},
				{IsWorking: false},
			},
			wantScore: 13.33, // 1/3 * 40
		},
		{
			name: "none in working hours",
			participants: []ParticipantTime{
				{IsWorking: false},
				{IsWorking: false},
			},
			wantScore: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := scoreWorkingHours(tt.participants)

			// Allow small floating point differences
			diff := score - tt.wantScore
			if diff < 0 {
				diff = -diff
			}
			if diff > 0.01 {
				t.Errorf("scoreWorkingHours() = %.2f, want %.2f", score, tt.wantScore)
			}
		})
	}
}

func TestScoreTimeQuality(t *testing.T) {
	loc, _ := time.LoadLocation("America/Los_Angeles")

	tests := []struct {
		name         string
		participants []ParticipantTime
		wantScore    float64
	}{
		{
			name: "excellent time 9-11am for all",
			participants: []ParticipantTime{
				{LocalTime: time.Date(2025, 1, 7, 10, 0, 0, 0, loc)}, // 10am
				{LocalTime: time.Date(2025, 1, 7, 9, 30, 0, 0, loc)}, // 9:30am
			},
			wantScore: 25.0, // Average of 25 + 25
		},
		{
			name: "mixed quality times",
			participants: []ParticipantTime{
				{LocalTime: time.Date(2025, 1, 7, 10, 0, 0, 0, loc)}, // 10am - excellent (25)
				{LocalTime: time.Date(2025, 1, 7, 13, 0, 0, 0, loc)}, // 1pm - good (20)
				{LocalTime: time.Date(2025, 1, 7, 15, 0, 0, 0, loc)}, // 3pm - fair (15)
			},
			wantScore: 20.0, // (25 + 20 + 15) / 3
		},
		{
			name: "poor time 8am",
			participants: []ParticipantTime{
				{LocalTime: time.Date(2025, 1, 7, 8, 0, 0, 0, loc)}, // 8am - poor (10)
			},
			wantScore: 10.0,
		},
		{
			name: "outside working hours",
			participants: []ParticipantTime{
				{LocalTime: time.Date(2025, 1, 7, 20, 0, 0, 0, loc)}, // 8pm - bad (0)
			},
			wantScore: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := scoreTimeQuality(tt.participants)

			// Allow small floating point differences
			diff := score - tt.wantScore
			if diff < 0 {
				diff = -diff
			}
			if diff > 0.01 {
				t.Errorf("scoreTimeQuality() = %.2f, want %.2f", score, tt.wantScore)
			}
		})
	}
}

func TestScoreCultural(t *testing.T) {
	loc, _ := time.LoadLocation("America/Los_Angeles")

	tests := []struct {
		name         string
		startTime    time.Time
		endTime      time.Time
		participants []ParticipantTime
		wantMin      float64
		wantMax      float64
	}{
		{
			name:      "no cultural issues",
			startTime: time.Date(2025, 1, 7, 10, 0, 0, 0, loc), // Tuesday 10am
			endTime:   time.Date(2025, 1, 7, 11, 0, 0, 0, loc),
			participants: []ParticipantTime{
				{LocalTime: time.Date(2025, 1, 7, 10, 0, 0, 0, loc)},
			},
			wantMin: 15.0,
			wantMax: 15.0,
		},
		{
			name:      "friday afternoon penalty",
			startTime: time.Date(2025, 1, 10, 15, 0, 0, 0, loc), // Friday 3pm
			endTime:   time.Date(2025, 1, 10, 16, 0, 0, 0, loc),
			participants: []ParticipantTime{
				{LocalTime: time.Date(2025, 1, 10, 15, 0, 0, 0, loc)},
			},
			wantMin: 10.0, // 15 - 5 (Friday PM penalty)
			wantMax: 10.0,
		},
		{
			name:      "lunch hour penalty",
			startTime: time.Date(2025, 1, 7, 12, 0, 0, 0, loc), // Tuesday 12pm
			endTime:   time.Date(2025, 1, 7, 13, 0, 0, 0, loc),
			participants: []ParticipantTime{
				{LocalTime: time.Date(2025, 1, 7, 12, 0, 0, 0, loc)},
			},
			wantMin: 12.0, // 15 - 3 (lunch penalty)
			wantMax: 12.0,
		},
		{
			name:      "monday early morning penalty",
			startTime: time.Date(2025, 1, 6, 9, 0, 0, 0, loc), // Monday 9am
			endTime:   time.Date(2025, 1, 6, 10, 0, 0, 0, loc),
			participants: []ParticipantTime{
				{LocalTime: time.Date(2025, 1, 6, 9, 0, 0, 0, loc)},
			},
			wantMin: 12.0, // 15 - 3 (Monday early penalty)
			wantMax: 12.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := scoreCultural(tt.startTime, tt.endTime, tt.participants)

			if score < tt.wantMin || score > tt.wantMax {
				t.Errorf("scoreCultural() = %.2f, want between %.2f and %.2f", score, tt.wantMin, tt.wantMax)
			}
		})
	}
}

func TestScoreWeekday(t *testing.T) {
	loc, _ := time.LoadLocation("America/Los_Angeles")

	tests := []struct {
		name      string
		startTime time.Time
		wantScore float64
	}{
		{
			name:      "tuesday - best",
			startTime: time.Date(2025, 1, 7, 10, 0, 0, 0, loc),
			wantScore: 10.0,
		},
		{
			name:      "wednesday - best",
			startTime: time.Date(2025, 1, 8, 10, 0, 0, 0, loc),
			wantScore: 10.0,
		},
		{
			name:      "monday - good",
			startTime: time.Date(2025, 1, 6, 10, 0, 0, 0, loc),
			wantScore: 8.0,
		},
		{
			name:      "thursday - good",
			startTime: time.Date(2025, 1, 9, 10, 0, 0, 0, loc),
			wantScore: 8.0,
		},
		{
			name:      "friday - ok",
			startTime: time.Date(2025, 1, 10, 10, 0, 0, 0, loc),
			wantScore: 5.0,
		},
		{
			name:      "saturday - avoid",
			startTime: time.Date(2025, 1, 11, 10, 0, 0, 0, loc),
			wantScore: 0.0,
		},
		{
			name:      "sunday - avoid",
			startTime: time.Date(2025, 1, 12, 10, 0, 0, 0, loc),
			wantScore: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := scoreWeekday(tt.startTime)

			if score != tt.wantScore {
				t.Errorf("scoreWeekday() = %.2f, want %.2f", score, tt.wantScore)
			}
		})
	}
}

func TestGetQualityLabel(t *testing.T) {
	loc, _ := time.LoadLocation("America/Los_Angeles")

	tests := []struct {
		name      string
		localTime time.Time
		isWorking bool
		wantLabel string
		wantIcon  string
	}{
		{
			name:      "excellent time 10am",
			localTime: time.Date(2025, 1, 7, 10, 0, 0, 0, loc),
			isWorking: true,
			wantLabel: "Excellent",
			wantIcon:  "✨",
		},
		{
			name:      "good time 1pm",
			localTime: time.Date(2025, 1, 7, 13, 0, 0, 0, loc),
			isWorking: true,
			wantLabel: "Good",
			wantIcon:  "",
		},
		{
			name:      "fair time 3pm",
			localTime: time.Date(2025, 1, 7, 15, 0, 0, 0, loc),
			isWorking: true,
			wantLabel: "Fair",
			wantIcon:  "",
		},
		{
			name:      "poor time 8am",
			localTime: time.Date(2025, 1, 7, 8, 0, 0, 0, loc),
			isWorking: true,
			wantLabel: "Poor",
			wantIcon:  "⚠️",
		},
		{
			name:      "poor time 5pm",
			localTime: time.Date(2025, 1, 7, 17, 0, 0, 0, loc),
			isWorking: true,
			wantLabel: "Poor",
			wantIcon:  "⚠️",
		},
		{
			name:      "bad time outside working hours",
			localTime: time.Date(2025, 1, 7, 20, 0, 0, 0, loc),
			isWorking: false,
			wantLabel: "Bad",
			wantIcon:  "🔴",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			label, icon := GetQualityLabel(tt.localTime, tt.isWorking)

			if label != tt.wantLabel {
				t.Errorf("GetQualityLabel() label = %q, want %q", label, tt.wantLabel)
			}
			if icon != tt.wantIcon {
				t.Errorf("GetQualityLabel() icon = %q, want %q", icon, tt.wantIcon)
			}
		})
	}
}

func TestGetScoreColor(t *testing.T) {
	tests := []struct {
		name      string
		score     float64
		wantColor string
	}{
		{
			name:      "excellent score 95",
			score:     95.0,
			wantColor: "🟢",
		},
		{
			name:      "excellent score 85",
			score:     85.0,
			wantColor: "🟢",
		},
		{
			name:      "good score 75",
			score:     75.0,
			wantColor: "🟡",
		},
		{
			name:      "good score 70",
			score:     70.0,
			wantColor: "🟡",
		},
		{
			name:      "poor score 65",
			score:     65.0,
			wantColor: "🔴",
		},
		{
			name:      "poor score 50",
			score:     50.0,
			wantColor: "🔴",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			color := GetScoreColor(tt.score)

			if color != tt.wantColor {
				t.Errorf("GetScoreColor() = %q, want %q", color, tt.wantColor)
			}
		})
	}
}
