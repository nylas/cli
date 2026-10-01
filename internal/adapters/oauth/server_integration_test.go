//go:build integration

package oauth

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"testing"
	"time"
)

// TestIntegration_CallbackServer_InvalidStateIsRefusedAndTheLoginContinues
// drives the real listener: a request with the wrong state is refused with
// 400 and does not end the login, and the real redirect after it still does.
func TestIntegration_CallbackServer_InvalidStateIsRefusedAndTheLoginContinues(t *testing.T) {
	server := NewCallbackServer(0)
	if err := server.Start(); err != nil {
		t.Fatalf("failed to start callback server: %v", err)
	}
	defer func() { _ = server.Stop() }()
	server.SetExpectedState("expected-state")

	port := server.listener.Addr().(*net.TCPAddr).Port
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	type result struct {
		code string
		err  error
	}
	resultCh := make(chan result, 1)
	go func() {
		code, err := server.WaitForCallback(ctx, "expected-state")
		resultCh <- result{code, err}
	}()

	get := func(query string) int {
		t.Helper()
		url := fmt.Sprintf("http://127.0.0.1:%d/callback?%s", port, query)
		var resp *http.Response
		var err error
		for deadline := time.Now().Add(time.Second); time.Now().Before(deadline); time.Sleep(10 * time.Millisecond) {
			if resp, err = http.Get(url); err == nil {
				break
			}
		}
		if err != nil {
			t.Fatalf("failed to send callback request: %v", err)
		}
		_ = resp.Body.Close()
		return resp.StatusCode
	}

	if status := get("code=stolen&state=wrong-state"); status != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", status, http.StatusBadRequest)
	}
	select {
	case r := <-resultCh:
		t.Fatalf("a wrong-state request ended the login: %+v", r)
	case <-time.After(200 * time.Millisecond):
	}

	if status := get("code=real-code&state=expected-state"); status != http.StatusOK {
		t.Fatalf("status = %d, want %d", status, http.StatusOK)
	}
	select {
	case r := <-resultCh:
		if r.err != nil || r.code != "real-code" {
			t.Fatalf("WaitForCallback() = %q, %v; want real-code, nil", r.code, r.err)
		}
	case <-time.After(time.Second):
		t.Fatal("WaitForCallback did not return after the real redirect")
	}
}
