package air

import (
	"net/http"
	"sync"
	"time"

	"github.com/nylas/cli/internal/httputil"
)

// EmailAnalytics represents email analytics data
type EmailAnalytics struct {
	// Volume metrics
	TotalReceived int `json:"totalReceived"`
	TotalSent     int `json:"totalSent"`
	TotalArchived int `json:"totalArchived"`
	TotalDeleted  int `json:"totalDeleted"`

	// Response metrics
	AvgResponseTime float64 `json:"avgResponseTimeHours"`
	ResponseRate    float64 `json:"responseRate"` // percentage

	// Top senders/recipients
	TopSenders    []SenderStats `json:"topSenders"`
	TopRecipients []SenderStats `json:"topRecipients"`

	// Time-based metrics
	BusiestHour  int            `json:"busiestHour"` // 0-23
	BusiestDay   string         `json:"busiestDay"`  // Monday, Tuesday, etc.
	HourlyVolume map[int]int    `json:"hourlyVolume"`
	DailyVolume  map[string]int `json:"dailyVolume"`
	WeeklyTrend  []DayVolume    `json:"weeklyTrend"`

	// Productivity metrics
	InboxZeroCount int     `json:"inboxZeroCount"` // Times achieved inbox zero
	CurrentStreak  int     `json:"currentStreak"`  // Consecutive inbox zero days
	BestStreak     int     `json:"bestStreak"`
	FocusTimeHours float64 `json:"focusTimeHours"`

	// Period info
	PeriodStart time.Time `json:"periodStart"`
	PeriodEnd   time.Time `json:"periodEnd"`
}

// SenderStats represents stats for a sender/recipient
type SenderStats struct {
	Email    string  `json:"email"`
	Name     string  `json:"name,omitempty"`
	Count    int     `json:"count"`
	AvgReply float64 `json:"avgReplyTimeHours,omitempty"`
}

// DayVolume represents volume for a specific day
type DayVolume struct {
	Date     string `json:"date"`
	Received int    `json:"received"`
	Sent     int    `json:"sent"`
}

// FocusTimeSuggestion represents a suggested focus time block
type FocusTimeSuggestion struct {
	StartHour int    `json:"startHour"`
	EndHour   int    `json:"endHour"`
	Day       string `json:"day"`
	Reason    string `json:"reason"`
	Score     int    `json:"score"` // 1-100, higher is better
}

// analyticsStore holds analytics data
type analyticsStore struct {
	analytics *EmailAnalytics
	mu        sync.RWMutex
}

var aStore = &analyticsStore{
	analytics: &EmailAnalytics{
		TotalReceived:   1250,
		TotalSent:       320,
		TotalArchived:   890,
		TotalDeleted:    210,
		AvgResponseTime: 2.5,
		ResponseRate:    78.5,
		TopSenders: []SenderStats{
			{Email: "team@company.com", Name: "Team Updates", Count: 145},
			{Email: "github@notifications.github.com", Name: "GitHub", Count: 98},
			{Email: "calendar@google.com", Name: "Google Calendar", Count: 67},
		},
		TopRecipients: []SenderStats{
			{Email: "boss@company.com", Name: "Manager", Count: 42, AvgReply: 1.2},
			{Email: "team@company.com", Name: "Team", Count: 38, AvgReply: 3.5},
		},
		BusiestHour: 10,
		BusiestDay:  "Tuesday",
		HourlyVolume: map[int]int{
			8: 15, 9: 45, 10: 78, 11: 65, 12: 32,
			13: 28, 14: 55, 15: 48, 16: 42, 17: 25,
		},
		DailyVolume: map[string]int{
			"Monday": 180, "Tuesday": 220, "Wednesday": 195,
			"Thursday": 175, "Friday": 150, "Saturday": 25, "Sunday": 15,
		},
		WeeklyTrend: []DayVolume{
			{Date: "2024-12-22", Received: 45, Sent: 12},
			{Date: "2024-12-23", Received: 62, Sent: 18},
			{Date: "2024-12-24", Received: 38, Sent: 8},
			{Date: "2024-12-25", Received: 12, Sent: 2},
			{Date: "2024-12-26", Received: 55, Sent: 15},
			{Date: "2024-12-27", Received: 48, Sent: 14},
			{Date: "2024-12-28", Received: 35, Sent: 10},
		},
		InboxZeroCount: 15,
		CurrentStreak:  3,
		BestStreak:     7,
		FocusTimeHours: 12.5,
		PeriodStart:    time.Now().AddDate(0, 0, -30),
		PeriodEnd:      time.Now(),
	},
}

// handleGetAnalyticsDashboard returns the full analytics dashboard
func (s *Server) handleGetAnalyticsDashboard(w http.ResponseWriter, r *http.Request) {
	aStore.mu.RLock()
	defer aStore.mu.RUnlock()

	httputil.WriteJSON(w, http.StatusOK, aStore.analytics)
}

// handleGetAnalyticsTrends returns email trends
func (s *Server) handleGetAnalyticsTrends(w http.ResponseWriter, r *http.Request) {
	period := r.URL.Query().Get("period") // "week", "month", "quarter"
	if period == "" {
		period = "week"
	}

	aStore.mu.RLock()
	defer aStore.mu.RUnlock()

	response := map[string]any{
		"period":       period,
		"weeklyTrend":  aStore.analytics.WeeklyTrend,
		"hourlyVolume": aStore.analytics.HourlyVolume,
		"dailyVolume":  aStore.analytics.DailyVolume,
	}

	httputil.WriteJSON(w, http.StatusOK, response)
}

// handleGetFocusTimeSuggestions returns suggested focus time blocks
func (s *Server) handleGetFocusTimeSuggestions(w http.ResponseWriter, r *http.Request) {
	aStore.mu.RLock()
	hourlyVolume := aStore.analytics.HourlyVolume
	aStore.mu.RUnlock()

	// Find low-volume hours for focus time
	suggestions := make([]FocusTimeSuggestion, 0)

	// Morning focus block
	morningVolume := 0
	for h := 6; h < 9; h++ {
		morningVolume += hourlyVolume[h]
	}
	if morningVolume < 30 {
		suggestions = append(suggestions, FocusTimeSuggestion{
			StartHour: 6,
			EndHour:   9,
			Day:       "Weekdays",
			Reason:    "Low email volume in early morning",
			Score:     85,
		})
	}

	// Afternoon focus block
	afternoonVolume := 0
	for h := 13; h < 15; h++ {
		afternoonVolume += hourlyVolume[h]
	}
	if afternoonVolume < 60 {
		suggestions = append(suggestions, FocusTimeSuggestion{
			StartHour: 13,
			EndHour:   15,
			Day:       "Weekdays",
			Reason:    "Post-lunch lull in email activity",
			Score:     72,
		})
	}

	// Evening focus block
	suggestions = append(suggestions, FocusTimeSuggestion{
		StartHour: 18,
		EndHour:   20,
		Day:       "Weekdays",
		Reason:    "Most colleagues offline",
		Score:     90,
	})

	httputil.WriteJSON(w, http.StatusOK, suggestions)
}

// handleGetProductivityStats returns productivity metrics
func (s *Server) handleGetProductivityStats(w http.ResponseWriter, r *http.Request) {
	aStore.mu.RLock()
	defer aStore.mu.RUnlock()

	response := map[string]any{
		"responseRate":    aStore.analytics.ResponseRate,
		"avgResponseTime": aStore.analytics.AvgResponseTime,
		"inboxZeroCount":  aStore.analytics.InboxZeroCount,
		"currentStreak":   aStore.analytics.CurrentStreak,
		"bestStreak":      aStore.analytics.BestStreak,
		"focusTimeHours":  aStore.analytics.FocusTimeHours,
		"emailsProcessed": aStore.analytics.TotalArchived + aStore.analytics.TotalDeleted,
	}

	httputil.WriteJSON(w, http.StatusOK, response)
}

// RecordEmailReceived records a received email
func RecordEmailReceived() {
	aStore.mu.Lock()
	defer aStore.mu.Unlock()
	aStore.analytics.TotalReceived++
}

// RecordEmailSent records a sent email
func RecordEmailSent() {
	aStore.mu.Lock()
	defer aStore.mu.Unlock()
	aStore.analytics.TotalSent++
}

// RecordInboxZero records achieving inbox zero
func RecordInboxZero() {
	aStore.mu.Lock()
	defer aStore.mu.Unlock()
	aStore.analytics.InboxZeroCount++
	aStore.analytics.CurrentStreak++
	if aStore.analytics.CurrentStreak > aStore.analytics.BestStreak {
		aStore.analytics.BestStreak = aStore.analytics.CurrentStreak
	}
}
