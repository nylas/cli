//go:build integration

package oauth

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/nylas/cli/internal/domain"
)

func TestIntegration_CallbackServer_InvalidStateFailsImmediately(t *testing.T) {
	server := NewCallbackServer(0)
	if err := server.Start(); err != nil {
		t.Fatalf("failed to start callback server: %v", err)
	}
	defer func() { _ = server.Stop() }()

	port := server.listener.Addr().(*net.TCPAddr).Port
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	resultCh := make(chan error, 1)
	go func() {
		_, err := server.WaitForCallback(ctx, "expected-state")
		resultCh <- err
	}()

	deadline := time.Now().Add(200 * time.Millisecond)
	for time.Now().Before(deadline) {
		if server.matchesExpectedState("expected-state") {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}

	callbackURL := fmt.Sprintf("http://127.0.0.1:%d/callback?code=test-code-123&state=wrong-state", port)
	var resp *http.Response
	var err error
	requestDeadline := time.Now().Add(time.Second)
	for time.Now().Before(requestDeadline) {
		resp, err = http.Get(callbackURL)
		if err == nil {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if err != nil {
		t.Fatalf("failed to send callback request: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}

	select {
	case err := <-resultCh:
		if !errors.Is(err, domain.ErrAuthFailed) {
			t.Fatalf("error = %v, want %v", err, domain.ErrAuthFailed)
		}
		if err == nil || err.Error() == domain.ErrAuthFailed.Error() {
			t.Fatalf("error = %v, want invalid state details", err)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("WaitForCallback did not fail after invalid state callback")
	}
}
