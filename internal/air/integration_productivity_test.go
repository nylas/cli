//go:build integration
// +build integration

package air

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestIntegration_FocusMode tests focus mode endpoints
func TestIntegration_FocusMode(t *testing.T) {
	server := testServer(t)

	// Test GET focus mode state
	req := httptest.NewRequest(http.MethodGet, "/api/focus", nil)
	w := httptest.NewRecorder()
	server.handleFocusModeRoute(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var stateResp map[string]any
	if err := json.NewDecoder(w.Body).Decode(&stateResp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	// Verify we got a state back
	if _, ok := stateResp["state"]; !ok {
		t.Error("expected 'state' key in response")
	}

	// Test POST to start focus mode
	startReq := httptest.NewRequest(http.MethodPost, "/api/focus",
		strings.NewReader(`{"duration": 25, "pomodoroMode": true}`))
	startReq.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	server.handleFocusModeRoute(w, startReq)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200 starting focus, got %d: %s", w.Code, w.Body.String())
	}

	var startResp FocusModeState
	if err := json.NewDecoder(w.Body).Decode(&startResp); err != nil {
		t.Fatalf("failed to decode start response: %v", err)
	}

	if !startResp.IsActive {
		t.Error("expected focus mode to be active after starting")
	}
	if startResp.Duration != 25 {
		t.Errorf("expected duration 25, got %d", startResp.Duration)
	}
	if !startResp.PomodoroMode {
		t.Error("expected pomodoro mode to be enabled")
	}

	// Test DELETE to stop focus mode
	stopReq := httptest.NewRequest(http.MethodDelete, "/api/focus", nil)
	w = httptest.NewRecorder()
	server.handleFocusModeRoute(w, stopReq)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200 stopping focus, got %d: %s", w.Code, w.Body.String())
	}
}

// TestIntegration_FocusModeSettings tests focus mode settings
func TestIntegration_FocusModeSettings(t *testing.T) {
	server := testServer(t)

	// Test GET settings
	req := httptest.NewRequest(http.MethodGet, "/api/focus/settings", nil)
	w := httptest.NewRecorder()
	server.handleFocusModeSettings(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var settings FocusModeSettings
	if err := json.NewDecoder(w.Body).Decode(&settings); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	// Verify default settings
	if settings.DefaultDuration <= 0 {
		t.Error("expected positive default duration")
	}
	if settings.PomodoroWork <= 0 {
		t.Error("expected positive pomodoro work time")
	}
}

// TestIntegration_ReplyLater tests reply later endpoints
func TestIntegration_ReplyLater(t *testing.T) {
	server := testServer(t)

	// Test GET empty list
	req := httptest.NewRequest(http.MethodGet, "/api/reply-later", nil)
	w := httptest.NewRecorder()
	server.handleReplyLaterRoute(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var items []*ReplyLaterItem
	if err := json.NewDecoder(w.Body).Decode(&items); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	// Test POST to add item
	addReq := httptest.NewRequest(http.MethodPost, "/api/reply-later",
		strings.NewReader(`{"emailId": "test-email-123", "subject": "Test Subject", "from": "test@example.com", "priority": 1}`))
	addReq.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	server.handleReplyLaterRoute(w, addReq)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200 adding item, got %d: %s", w.Code, w.Body.String())
	}

	var item ReplyLaterItem
	if err := json.NewDecoder(w.Body).Decode(&item); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if item.EmailID != "test-email-123" {
		t.Errorf("expected email ID 'test-email-123', got '%s'", item.EmailID)
	}
	if item.Priority != 1 {
		t.Errorf("expected priority 1, got %d", item.Priority)
	}
}

// TestIntegration_Analytics tests analytics endpoints
func TestIntegration_Analytics(t *testing.T) {
	server := testServer(t)

	// Test dashboard endpoint
	req := httptest.NewRequest(http.MethodGet, "/api/analytics/dashboard", nil)
	w := httptest.NewRecorder()
	server.handleGetAnalyticsDashboard(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var analytics EmailAnalytics
	if err := json.NewDecoder(w.Body).Decode(&analytics); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	// Verify analytics data
	if analytics.TotalReceived < 0 {
		t.Error("expected non-negative total received")
	}

	// Test trends endpoint
	req = httptest.NewRequest(http.MethodGet, "/api/analytics/trends?period=week", nil)
	w = httptest.NewRecorder()
	server.handleGetAnalyticsTrends(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200 for trends, got %d: %s", w.Code, w.Body.String())
	}

	// Test productivity stats
	req = httptest.NewRequest(http.MethodGet, "/api/analytics/productivity", nil)
	w = httptest.NewRecorder()
	server.handleGetProductivityStats(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200 for productivity, got %d: %s", w.Code, w.Body.String())
	}
}

// TestIntegration_Notetaker tests notetaker endpoints
func TestIntegration_Notetaker(t *testing.T) {
	server := testServer(t)

	// Test GET empty list
	req := httptest.NewRequest(http.MethodGet, "/api/notetakers", nil)
	w := httptest.NewRecorder()
	server.handleNotetakersRoute(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	// Test POST to create notetaker
	createReq := httptest.NewRequest(http.MethodPost, "/api/notetakers",
		strings.NewReader(`{"meetingLink": "https://meet.google.com/abc-defg-hij"}`))
	createReq.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	server.handleNotetakersRoute(w, createReq)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200 creating notetaker, got %d: %s", w.Code, w.Body.String())
	}

	var nt NotetakerResponse
	if err := json.NewDecoder(w.Body).Decode(&nt); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if nt.ID == "" {
		t.Error("expected notetaker ID")
	}
	if nt.Provider != "google_meet" {
		t.Errorf("expected provider 'google_meet', got '%s'", nt.Provider)
	}
	// Accept both "scheduled" and "connecting" as valid initial states
	// The Nylas API may return "connecting" initially before transitioning to "scheduled"
	if nt.State != "scheduled" && nt.State != "connecting" {
		t.Errorf("expected state 'scheduled' or 'connecting', got '%s'", nt.State)
	}
}

// TestIntegration_Screener tests screener endpoints
func TestIntegration_Screener(t *testing.T) {
	server := testServer(t)

	// Test GET pending senders
	req := httptest.NewRequest(http.MethodGet, "/api/screener", nil)
	w := httptest.NewRecorder()
	server.handleGetScreenedSenders(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	// Test adding to screener
	addReq := httptest.NewRequest(http.MethodPost, "/api/screener/add",
		strings.NewReader(`{"email": "new@sender.com", "name": "New Sender"}`))
	addReq.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	server.handleAddToScreener(w, addReq)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200 adding to screener, got %d: %s", w.Code, w.Body.String())
	}

	// Test allowing sender
	allowReq := httptest.NewRequest(http.MethodPost, "/api/screener/allow",
		strings.NewReader(`{"email": "new@sender.com", "destination": "inbox"}`))
	allowReq.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	server.handleScreenerAllow(w, allowReq)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200 allowing sender, got %d: %s", w.Code, w.Body.String())
	}
}

// TestIntegration_AIConfig tests AI configuration endpoints
func TestIntegration_AIConfig(t *testing.T) {
	server := testServer(t)

	// Test GET config
	req := httptest.NewRequest(http.MethodGet, "/api/ai/config", nil)
	w := httptest.NewRecorder()
	server.handleAIConfigRoute(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var config AIConfig
	if err := json.NewDecoder(w.Body).Decode(&config); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	// Verify config
	if config.Provider == "" {
		t.Error("expected provider to be set")
	}
	if config.Model == "" {
		t.Error("expected model to be set")
	}

	// Test GET providers
	req = httptest.NewRequest(http.MethodGet, "/api/ai/providers", nil)
	w = httptest.NewRecorder()
	server.handleGetAIProviders(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200 for providers, got %d: %s", w.Code, w.Body.String())
	}

	var providers []map[string]any
	if err := json.NewDecoder(w.Body).Decode(&providers); err != nil {
		t.Fatalf("failed to decode providers: %v", err)
	}

	if len(providers) < 1 {
		t.Error("expected at least one AI provider")
	}
}

// TestIntegration_ReadReceipts tests read receipt endpoints
func TestIntegration_ReadReceipts(t *testing.T) {
	server := testServer(t)

	// Test GET receipts
	req := httptest.NewRequest(http.MethodGet, "/api/receipts", nil)
	w := httptest.NewRecorder()
	server.handleGetReadReceipts(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	// Test GET settings
	req = httptest.NewRequest(http.MethodGet, "/api/receipts/settings", nil)
	w = httptest.NewRecorder()
	server.handleReadReceiptSettings(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200 for settings, got %d: %s", w.Code, w.Body.String())
	}

	var settings ReadReceiptSettings
	if err := json.NewDecoder(w.Body).Decode(&settings); err != nil {
		t.Fatalf("failed to decode settings: %v", err)
	}

	// Test tracking pixel endpoint
	req = httptest.NewRequest(http.MethodGet, "/api/track/open?id=test123", nil)
	w = httptest.NewRecorder()
	server.handleTrackOpen(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200 for tracking, got %d", w.Code)
	}

	// Verify it returns a GIF
	contentType := w.Header().Get("Content-Type")
	if contentType != "image/gif" {
		t.Errorf("expected Content-Type 'image/gif', got '%s'", contentType)
	}
}
