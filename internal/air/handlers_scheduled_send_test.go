package air

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// TestCreateScheduledMessage_RejectsFarFuture pins the post-fix invariant:
// a send_at more than ~1 year in the future is refused. Without it, a
// client bug or hostile request could keep a message in the queue
// indefinitely (or trip Nylas API limits silently).
func TestCreateScheduledMessage_RejectsFarFuture(t *testing.T) {
	server := &Server{demoMode: true}

	body, _ := json.Marshal(ScheduledSendRequest{
		SendAt:  time.Now().Add(10 * 365 * 24 * time.Hour).Unix(),
		To:      []EmailParticipantResponse{{Email: "a@example.com"}},
		Subject: "ping",
		Body:    "hi",
	})
	r := httptest.NewRequest(http.MethodPost, "/api/scheduled-send", bytes.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.createScheduledMessage(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
	if !bytes.Contains(w.Body.Bytes(), []byte("within one year")) {
		t.Fatalf("expected 'within one year' message, got %s", w.Body.String())
	}
}

func TestCreateScheduledMessage_AcceptsNearFuture(t *testing.T) {
	server := &Server{demoMode: true}

	body, _ := json.Marshal(ScheduledSendRequest{
		SendAt:  time.Now().Add(2 * time.Hour).Unix(),
		To:      []EmailParticipantResponse{{Email: "a@example.com"}},
		Subject: "ping",
		Body:    "hi",
	})
	r := httptest.NewRequest(http.MethodPost, "/api/scheduled-send", bytes.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.createScheduledMessage(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}
}

func TestCreateScheduledMessage_RejectsTooSoon(t *testing.T) {
	server := &Server{demoMode: true}

	body, _ := json.Marshal(ScheduledSendRequest{
		SendAt:  time.Now().Unix(),
		To:      []EmailParticipantResponse{{Email: "a@example.com"}},
		Subject: "ping",
		Body:    "hi",
	})
	r := httptest.NewRequest(http.MethodPost, "/api/scheduled-send", bytes.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.createScheduledMessage(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for past time, got %d", w.Code)
	}
}
