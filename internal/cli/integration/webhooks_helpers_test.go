//go:build integration

package integration

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestWaitForWebhookChallengeStability_SucceedsAfterConsecutiveResponses(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "codex-webhook-ready")
	}))
	defer server.Close()

	err := waitForWebhookChallengeStability(server.URL, 2, 5*time.Millisecond, 100*time.Millisecond)
	if err != nil {
		t.Fatalf("waitForWebhookChallengeStability() error = %v", err)
	}
}

func TestWaitForWebhookChallengeStability_ReturnsErrorWhenNeverStable(t *testing.T) {
	var calls int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		call := atomic.AddInt32(&calls, 1)
		if call%2 == 0 {
			fmt.Fprint(w, "codex-webhook-ready")
			return
		}
		http.Error(w, "not ready", http.StatusServiceUnavailable)
	}))
	defer server.Close()

	err := waitForWebhookChallengeStability(server.URL, 2, 5*time.Millisecond, 60*time.Millisecond)
	if err == nil {
		t.Fatal("waitForWebhookChallengeStability() error = nil, want timeout")
	}
}
