package air

import (
	"encoding/json"
	"net/http"
	"slices"
	"sync"
	"time"

	"github.com/nylas/cli/internal/httputil"
)

// FocusModeState represents the current focus mode state
type FocusModeState struct {
	IsActive      bool      `json:"isActive"`
	StartedAt     time.Time `json:"startedAt,omitzero"`
	EndsAt        time.Time `json:"endsAt,omitzero"`
	Duration      int       `json:"duration"` // minutes
	PomodoroMode  bool      `json:"pomodoroMode"`
	SessionCount  int       `json:"sessionCount"`
	BreakDuration int       `json:"breakDuration"` // minutes
	InBreak       bool      `json:"inBreak"`
}

// FocusModeSettings represents focus mode preferences
type FocusModeSettings struct {
	DefaultDuration    int      `json:"defaultDuration"`    // minutes
	PomodoroWork       int      `json:"pomodoroWork"`       // minutes
	PomodoroBreak      int      `json:"pomodoroBreak"`      // minutes
	PomodoroLongBreak  int      `json:"pomodoroLongBreak"`  // minutes
	SessionsBeforeLong int      `json:"sessionsBeforeLong"` // sessions before long break
	HideNotifications  bool     `json:"hideNotifications"`
	HideEmailList      bool     `json:"hideEmailList"`
	MuteSound          bool     `json:"muteSound"`
	AllowedSenders     []string `json:"allowedSenders"` // VIP list that can interrupt
	AutoReplyEnabled   bool     `json:"autoReplyEnabled"`
	AutoReplyMessage   string   `json:"autoReplyMessage"`
}

// focusModeStore holds focus mode state
type focusModeStore struct {
	state    *FocusModeState
	settings *FocusModeSettings
	mu       sync.RWMutex
}

var fmStore = &focusModeStore{
	state: &FocusModeState{
		IsActive:     false,
		Duration:     25,
		PomodoroMode: false,
	},
	settings: &FocusModeSettings{
		DefaultDuration:    25,
		PomodoroWork:       25,
		PomodoroBreak:      5,
		PomodoroLongBreak:  15,
		SessionsBeforeLong: 4,
		HideNotifications:  true,
		HideEmailList:      true,
		MuteSound:          false,
		AllowedSenders:     []string{},
		AutoReplyEnabled:   false,
		AutoReplyMessage:   "I'm currently in focus mode and will respond later.",
	},
}

// handleFocusModeRoute dispatches focus mode requests by method
func (s *Server) handleFocusModeRoute(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.handleGetFocusModeState(w, r)
	case http.MethodPost:
		s.handleStartFocusMode(w, r)
	case http.MethodDelete:
		s.handleStopFocusMode(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleFocusModeSettings dispatches focus mode settings requests by method
func (s *Server) handleFocusModeSettings(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.handleGetFocusModeSettings(w, r)
	case http.MethodPut, http.MethodPost:
		s.handleUpdateFocusModeSettings(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleGetFocusModeState returns current focus mode state
func (s *Server) handleGetFocusModeState(w http.ResponseWriter, r *http.Request) {
	fmStore.mu.RLock()
	defer fmStore.mu.RUnlock()

	// Check if session has ended
	state := *fmStore.state
	if state.IsActive && !state.EndsAt.IsZero() && time.Now().After(state.EndsAt) {
		state.IsActive = false
	}

	// Calculate remaining time
	response := map[string]any{
		"state": state,
	}

	if state.IsActive && !state.EndsAt.IsZero() {
		remaining := time.Until(state.EndsAt)
		if remaining > 0 {
			response["remainingMinutes"] = int(remaining.Minutes())
			response["remainingSeconds"] = int(remaining.Seconds()) % 60
		}
	}

	httputil.WriteJSON(w, http.StatusOK, response)
}

// handleStartFocusMode starts a focus mode session
func (s *Server) handleStartFocusMode(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Duration     int  `json:"duration,omitempty"`
		PomodoroMode bool `json:"pomodoroMode"`
	}

	if err := json.NewDecoder(limitedBody(w, r)).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	fmStore.mu.Lock()
	defer fmStore.mu.Unlock()

	duration := req.Duration
	if duration <= 0 {
		if req.PomodoroMode {
			duration = fmStore.settings.PomodoroWork
		} else {
			duration = fmStore.settings.DefaultDuration
		}
	}

	now := time.Now()
	fmStore.state = &FocusModeState{
		IsActive:      true,
		StartedAt:     now,
		EndsAt:        now.Add(time.Duration(duration) * time.Minute),
		Duration:      duration,
		PomodoroMode:  req.PomodoroMode,
		SessionCount:  fmStore.state.SessionCount,
		BreakDuration: fmStore.settings.PomodoroBreak,
		InBreak:       false,
	}

	httputil.WriteJSON(w, http.StatusOK, fmStore.state)
}

// handleStopFocusMode stops the current focus mode session
func (s *Server) handleStopFocusMode(w http.ResponseWriter, r *http.Request) {
	fmStore.mu.Lock()
	defer fmStore.mu.Unlock()

	if fmStore.state.IsActive && !fmStore.state.InBreak {
		fmStore.state.SessionCount++
	}
	fmStore.state.IsActive = false
	fmStore.state.InBreak = false

	httputil.WriteJSON(w, http.StatusOK, map[string]any{
		"status":       "stopped",
		"sessionCount": fmStore.state.SessionCount,
	})
}

// handleStartBreak starts a break in pomodoro mode
func (s *Server) handleStartBreak(w http.ResponseWriter, r *http.Request) {
	fmStore.mu.Lock()
	defer fmStore.mu.Unlock()

	if !fmStore.state.PomodoroMode {
		http.Error(w, "Not in pomodoro mode", http.StatusBadRequest)
		return
	}

	// Determine break duration
	breakDuration := fmStore.settings.PomodoroBreak
	if fmStore.state.SessionCount > 0 && fmStore.state.SessionCount%fmStore.settings.SessionsBeforeLong == 0 {
		breakDuration = fmStore.settings.PomodoroLongBreak
	}

	now := time.Now()
	fmStore.state.InBreak = true
	fmStore.state.StartedAt = now
	fmStore.state.EndsAt = now.Add(time.Duration(breakDuration) * time.Minute)
	fmStore.state.BreakDuration = breakDuration

	httputil.WriteJSON(w, http.StatusOK, fmStore.state)
}

// handleGetFocusModeSettings returns focus mode settings
func (s *Server) handleGetFocusModeSettings(w http.ResponseWriter, r *http.Request) {
	fmStore.mu.RLock()
	defer fmStore.mu.RUnlock()

	httputil.WriteJSON(w, http.StatusOK, fmStore.settings)
}

// handleUpdateFocusModeSettings updates focus mode settings
func (s *Server) handleUpdateFocusModeSettings(w http.ResponseWriter, r *http.Request) {
	var settings FocusModeSettings
	if err := json.NewDecoder(limitedBody(w, r)).Decode(&settings); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	func() {
		fmStore.mu.Lock()
		defer fmStore.mu.Unlock()
		fmStore.settings = &settings
	}()

	httputil.WriteJSON(w, http.StatusOK, map[string]string{"status": "updated"})
}

// IsFocusModeActive reports whether a focus session is currently running.
//
// A session is "active" when IsActive=true AND we haven't passed EndsAt.
// The handler that started the session never schedules a goroutine to flip
// IsActive when the timer elapses, so callers (notification gating, auto-
// reply) have to perform the expiry check at call time. Without this the
// session would appear to last forever after the user closes the tab.
func IsFocusModeActive() bool {
	fmStore.mu.RLock()
	defer fmStore.mu.RUnlock()
	if !fmStore.state.IsActive {
		return false
	}
	if !fmStore.state.EndsAt.IsZero() && !time.Now().Before(fmStore.state.EndsAt) {
		return false
	}
	return true
}

// ShouldAllowNotification reports whether a notification should be surfaced.
//
// Honours the active expiry window — once EndsAt has passed, focus mode is
// effectively over even if no client GET refreshed the persisted state.
func ShouldAllowNotification(senderEmail string) bool {
	fmStore.mu.RLock()
	defer fmStore.mu.RUnlock()

	active := fmStore.state.IsActive
	if active && !fmStore.state.EndsAt.IsZero() && !time.Now().Before(fmStore.state.EndsAt) {
		active = false
	}
	if !active || !fmStore.settings.HideNotifications {
		return true
	}

	return slices.Contains(fmStore.settings.AllowedSenders, senderEmail)
}
