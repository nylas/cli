package scheduling

import (
	"time"
)

// Scoring weights for the 100-point scoring algorithm
const (
	// Working Hours: 40 points - Is this within working hours for all participants?
	ScoreWorkingHoursMax = 40.0

	// Time Quality: 25 points - How good is this time for participants?
	// - 9-11 AM: Excellent (25 points)
	// - 11 AM-2 PM: Good (20 points)
	// - 2-5 PM: Fair (15 points)
	// - 8-9 AM or 5-6 PM: Poor (10 points)
	// - Outside working hours: 0 points
	ScoreTimeQualityMax = 25.0

	// Cultural Considerations: 15 points - Respects cultural norms
	// - No Friday PM meetings (end > 3pm) for some cultures
	// - No lunch hour meetings (12-1pm)
	// - No Monday early AM (before 10am)
	ScoreCulturalMax = 15.0

	// Weekday Preference: 10 points
	// - Tuesday/Wednesday: 10 points (best)
	// - Monday/Thursday: 8 points
	// - Friday: 5 points
	ScoreWeekdayMax = 10.0

	// Holiday Check: 10 points
	// - No holidays: 10 points
	// - Holiday for some: 5 points
	// - Holiday for all: 0 points
	ScoreHolidayMax = 10.0
)

// TimeSlot represents a potential meeting time with scoring.
type TimeSlot struct {
	StartTime time.Time
	EndTime   time.Time
	Score     float64
	Breakdown ScoreBreakdown
}

// ScoreBreakdown shows detailed scoring for transparency.
type ScoreBreakdown struct {
	WorkingHours float64 // 0-40 points
	TimeQuality  float64 // 0-25 points
	Cultural     float64 // 0-15 points
	Weekday      float64 // 0-10 points
	Holiday      float64 // 0-10 points
	Total        float64 // Sum of all (0-100)
}

// ParticipantTime represents a participant's local time for a meeting slot.
type ParticipantTime struct {
	TimeZone    string
	LocalTime   time.Time
	IsWorking   bool
	Quality     string // "Excellent", "Good", "Fair", "Poor", "Bad"
	QualityIcon string // "✨", "", "⚠️", "🔴"
}

// WorkingHours represents standard working hours for a timezone.
type WorkingHours struct {
	Start time.Time // e.g., 9:00 AM
	End   time.Time // e.g., 5:00 PM
}

// ScoreTimeSlot calculates a 100-point score for a given time slot.
func ScoreTimeSlot(startTime, endTime time.Time, participants []ParticipantTime) ScoreBreakdown {
	var breakdown ScoreBreakdown

	// 1. Working Hours Score (40 points)
	breakdown.WorkingHours = scoreWorkingHours(participants)

	// 2. Time Quality Score (25 points)
	breakdown.TimeQuality = scoreTimeQuality(participants)

	// 3. Cultural Considerations (15 points)
	breakdown.Cultural = scoreCultural(startTime, endTime, participants)

	// 4. Weekday Preference (10 points)
	breakdown.Weekday = scoreWeekday(startTime)

	// 5. Holiday Check (10 points)
	breakdown.Holiday = scoreHoliday(startTime, participants)

	// Calculate total
	breakdown.Total = breakdown.WorkingHours +
		breakdown.TimeQuality +
		breakdown.Cultural +
		breakdown.Weekday +
		breakdown.Holiday

	return breakdown
}

// scoreWorkingHours gives full points if ALL participants are within working hours.
func scoreWorkingHours(participants []ParticipantTime) float64 {
	allInWorkingHours := true
	for _, p := range participants {
		if !p.IsWorking {
			allInWorkingHours = false
			break
		}
	}

	if allInWorkingHours {
		return ScoreWorkingHoursMax
	}

	// Partial credit: count percentage in working hours
	inWorkingHours := 0
	for _, p := range participants {
		if p.IsWorking {
			inWorkingHours++
		}
	}

	ratio := float64(inWorkingHours) / float64(len(participants))
	return ScoreWorkingHoursMax * ratio
}

// scoreTimeQuality evaluates how good the time is for participants.
// Average quality across all participants.
func scoreTimeQuality(participants []ParticipantTime) float64 {
	totalQuality := 0.0

	for _, p := range participants {
		hour := p.LocalTime.Hour()
		quality := 0.0

		// 9-11 AM: Excellent (25 points)
		if hour >= 9 && hour < 11 {
			quality = 25.0
			// 11 AM - 2 PM: Good (20 points)
		} else if hour >= 11 && hour < 14 {
			quality = 20.0
			// 2-5 PM: Fair (15 points)
		} else if hour >= 14 && hour < 17 {
			quality = 15.0
			// 8-9 AM or 5-6 PM: Poor (10 points)
		} else if (hour >= 8 && hour < 9) || (hour >= 17 && hour < 18) {
			quality = 10.0
			// Outside working hours: 0 points
		} else {
			quality = 0.0
		}

		totalQuality += quality
	}

	// Return average quality
	return totalQuality / float64(len(participants))
}

// scoreCultural evaluates cultural considerations.
func scoreCultural(startTime, endTime time.Time, participants []ParticipantTime) float64 {
	score := ScoreCulturalMax
	penaltyApplied := false

	// Check each participant for cultural concerns
	for _, p := range participants {
		hour := p.LocalTime.Hour()
		weekday := p.LocalTime.Weekday()

		// Penalty: Friday PM meetings (after 3pm) - some cultures prefer short Fridays
		if weekday == time.Friday && hour >= 15 {
			score -= 5.0
			penaltyApplied = true
		}

		// Penalty: Lunch hour meetings (12-1pm)
		if hour == 12 {
			score -= 3.0
			penaltyApplied = true
		}

		// Penalty: Monday early AM (before 10am)
		if weekday == time.Monday && hour < 10 {
			score -= 3.0
			penaltyApplied = true
		}

		// Cap penalties to avoid going negative
		if score < 0 {
			score = 0
		}
	}

	// If no penalties, give full score
	if !penaltyApplied {
		return ScoreCulturalMax
	}

	return score
}

// scoreWeekday gives preference to mid-week meetings.
func scoreWeekday(startTime time.Time) float64 {
	weekday := startTime.Weekday()

	switch weekday {
	case time.Tuesday, time.Wednesday:
		return 10.0 // Best days
	case time.Monday, time.Thursday:
		return 8.0
	case time.Friday:
		return 5.0
	case time.Saturday, time.Sunday:
		return 0.0 // Avoid weekends
	default:
		return 5.0
	}
}

// scoreHoliday checks if the date falls on a holiday for participants.
// For now, this is a placeholder. In a real implementation, this would
// check a holiday calendar API or database.
func scoreHoliday(startTime time.Time, participants []ParticipantTime) float64 {
	// TODO: Integrate with holiday calendar API
	// For now, assume no holidays
	return ScoreHolidayMax
}

// GetQualityLabel returns a human-readable quality label for a time.
func GetQualityLabel(localTime time.Time, isWorking bool) (string, string) {
	if !isWorking {
		return "Bad", "🔴"
	}

	hour := localTime.Hour()

	// 9-11 AM: Excellent
	if hour >= 9 && hour < 11 {
		return "Excellent", "✨"
	}

	// 11 AM - 2 PM: Good
	if hour >= 11 && hour < 14 {
		return "Good", ""
	}

	// 2-5 PM: Fair
	if hour >= 14 && hour < 17 {
		return "Fair", ""
	}

	// 8-9 AM or 5-6 PM: Poor
	if (hour >= 8 && hour < 9) || (hour >= 17 && hour < 18) {
		return "Poor", "⚠️"
	}

	// Outside working hours
	return "Bad", "🔴"
}

// GetScoreColor returns a color indicator based on the total score.
func GetScoreColor(score float64) string {
	if score >= 85 {
		return "🟢" // Green - Excellent
	} else if score >= 70 {
		return "🟡" // Yellow - Good
	}
	return "🔴" // Red - Poor
}
