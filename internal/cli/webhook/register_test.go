package webhook

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/nylas/cli/internal/domain"
)

// fakeWebhookClient is a minimal ports.WebhookClient for register tests. Only
// List/Create/Delete carry behaviour; the rest satisfy the interface.
type fakeWebhookClient struct {
	existing   []domain.Webhook
	created    *domain.CreateWebhookRequest
	createResp *domain.Webhook
	createErr  error
	deleted    []string

	// verifyFailures makes CreateWebhook return the Nylas verify error (70005)
	// this many times before succeeding — simulating tunnel-propagation delay.
	verifyFailures int
	createCalls    int
}

func (f *fakeWebhookClient) ListWebhooks(_ context.Context) ([]domain.Webhook, error) {
	return f.existing, nil
}
func (f *fakeWebhookClient) GetWebhook(_ context.Context, _ string) (*domain.Webhook, error) {
	return nil, nil
}
func (f *fakeWebhookClient) CreateWebhook(_ context.Context, req *domain.CreateWebhookRequest) (*domain.Webhook, error) {
	f.created = req
	f.createCalls++
	if f.createCalls <= f.verifyFailures {
		// Mirrors Nylas: numeric code in Type, symbolic code in Message.
		return nil, &domain.APIError{StatusCode: 400, Type: "70005", Message: "unable.verify.webhook_url : unable to verify webhook URL"}
	}
	if f.createErr != nil {
		return nil, f.createErr
	}
	return f.createResp, nil
}
func (f *fakeWebhookClient) UpdateWebhook(_ context.Context, _ string, _ *domain.UpdateWebhookRequest) (*domain.Webhook, error) {
	return nil, nil
}
func (f *fakeWebhookClient) DeleteWebhook(_ context.Context, id string) error {
	f.deleted = append(f.deleted, id)
	return nil
}
func (f *fakeWebhookClient) RotateWebhookSecret(_ context.Context, _ string) (*domain.RotateWebhookSecretResponse, error) {
	return nil, nil
}
func (f *fakeWebhookClient) SendWebhookTestEvent(_ context.Context, _ string) error { return nil }
func (f *fakeWebhookClient) GetWebhookMockPayload(_ context.Context, _ string) (map[string]any, error) {
	return nil, nil
}

func TestResolveRegisterTriggers(t *testing.T) {
	t.Run("non-interactive with no triggers errors", func(t *testing.T) {
		_, err := resolveRegisterTriggers(nil, false, &mockPrompter{})
		if err == nil {
			t.Fatal("expected error when --triggers omitted non-interactively")
		}
	})

	t.Run("interactive prompts and validates the answer", func(t *testing.T) {
		p := &mockPrompter{asks: []askResp{{value: "message.created,event.created"}}}
		got, err := resolveRegisterTriggers(nil, true, p)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		want := []string{domain.TriggerMessageCreated, domain.TriggerEventCreated}
		if len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
			t.Fatalf("triggers = %v, want %v", got, want)
		}
		if p.tAsks != 1 {
			t.Errorf("expected exactly one Ask call, got %d", p.tAsks)
		}
	})

	t.Run("flag-provided triggers are validated", func(t *testing.T) {
		got, err := resolveRegisterTriggers([]string{"message.created"}, false, &mockPrompter{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(got) != 1 || got[0] != domain.TriggerMessageCreated {
			t.Fatalf("triggers = %v", got)
		}
	})

	t.Run("invalid trigger rejected", func(t *testing.T) {
		_, err := resolveRegisterTriggers([]string{"not.a.real.trigger"}, false, &mockPrompter{})
		if err == nil {
			t.Fatal("expected invalid trigger to error")
		}
	})
}

func TestRegisterWebhook_SweepsStaleThenCreates(t *testing.T) {
	client := &fakeWebhookClient{
		existing: []domain.Webhook{
			{ID: "stale-1", Description: autoWebhookDescription},
			{ID: "user-owned", Description: "my real webhook"},
			{ID: "stale-2", Description: autoWebhookDescription},
		},
		createResp: &domain.Webhook{ID: "new-id", WebhookSecret: "sekret"},
	}

	secret, reg, err := registerWebhook(context.Background(), client,
		"https://x.trycloudflare.com/webhook", []string{domain.TriggerMessageCreated})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Only auto-tagged webhooks are swept; the user's own webhook is untouched.
	if len(client.deleted) != 2 || client.deleted[0] != "stale-1" || client.deleted[1] != "stale-2" {
		t.Errorf("swept = %v, want [stale-1 stale-2]", client.deleted)
	}
	if secret != "sekret" {
		t.Errorf("secret = %q, want sekret", secret)
	}
	if reg == nil || reg.webhookID != "new-id" {
		t.Fatalf("registration = %+v, want webhookID new-id", reg)
	}
	if client.created == nil || client.created.Description != autoWebhookDescription {
		t.Errorf("created webhook missing auto description: %+v", client.created)
	}
	if client.created.WebhookURL != "https://x.trycloudflare.com/webhook" {
		t.Errorf("created URL = %q", client.created.WebhookURL)
	}
}

func TestRegisterWebhook_RetriesVerifyErrorThenSucceeds(t *testing.T) {
	// Shrink the retry cadence so the test doesn't wait the production interval.
	origInterval, origTimeout := registerRetryInterval, registerVerifyTimeout
	registerRetryInterval, registerVerifyTimeout = time.Millisecond, 5*time.Second
	t.Cleanup(func() { registerRetryInterval, registerVerifyTimeout = origInterval, origTimeout })

	client := &fakeWebhookClient{
		verifyFailures: 2,
		createResp:     &domain.Webhook{ID: "new-id", WebhookSecret: "sekret"},
	}
	secret, reg, err := registerWebhook(context.Background(), client, "https://x/webhook",
		[]string{domain.TriggerMessageCreated})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if client.createCalls != 3 {
		t.Errorf("createCalls = %d, want 3 (2 verify failures + 1 success)", client.createCalls)
	}
	if secret != "sekret" || reg == nil || reg.webhookID != "new-id" {
		t.Errorf("registration after retry = (%q, %+v)", secret, reg)
	}
}

func TestRegisterWebhook_GivesUpAfterTimeout(t *testing.T) {
	origInterval, origTimeout := registerRetryInterval, registerVerifyTimeout
	registerRetryInterval, registerVerifyTimeout = time.Millisecond, 20*time.Millisecond
	t.Cleanup(func() { registerRetryInterval, registerVerifyTimeout = origInterval, origTimeout })

	// Always fails verification — should exhaust the budget and return a clear
	// "could not reach" error rather than spinning forever.
	client := &fakeWebhookClient{verifyFailures: 1 << 30}
	_, _, err := registerWebhook(context.Background(), client, "https://x/webhook",
		[]string{domain.TriggerMessageCreated})
	if err == nil {
		t.Fatal("expected timeout error when verification never succeeds")
	}
	if !errorMessageContains(err, "could not reach the tunnel URL") {
		t.Errorf("error = %v, want timeout message", err)
	}
}

func TestEnsureCloudflaredInstalled(t *testing.T) {
	origInstalled, origBrew, origInstall := cloudflaredInstalled, cloudflaredViaBrew, installCloudflaredFn
	t.Cleanup(func() {
		cloudflaredInstalled, cloudflaredViaBrew, installCloudflaredFn = origInstalled, origBrew, origInstall
	})

	t.Run("already installed is a no-op", func(t *testing.T) {
		cloudflaredInstalled = func() bool { return true }
		p := &mockPrompter{}
		if err := ensureCloudflaredInstalled(true, p); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if p.tConfirms != 0 {
			t.Errorf("should not prompt when already installed")
		}
	})

	t.Run("non-interactive errors without prompting", func(t *testing.T) {
		cloudflaredInstalled = func() bool { return false }
		p := &mockPrompter{}
		err := ensureCloudflaredInstalled(false, p)
		if err == nil || !errorMessageContains(err, "cloudflared is not installed") {
			t.Fatalf("want not-installed error, got %v", err)
		}
		if p.tConfirms != 0 {
			t.Errorf("should not prompt in non-interactive mode")
		}
	})

	t.Run("interactive brew install succeeds", func(t *testing.T) {
		calls := 0
		// First probe: not installed. After install: installed.
		cloudflaredInstalled = func() bool { calls++; return calls > 1 }
		cloudflaredViaBrew = func() bool { return true }
		installed := false
		installCloudflaredFn = func() error { installed = true; return nil }
		p := &mockPrompter{confirms: []confirmResp{{value: true}}}
		if err := ensureCloudflaredInstalled(true, p); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !installed {
			t.Error("expected brew install to run")
		}
	})

	t.Run("interactive decline errors", func(t *testing.T) {
		cloudflaredInstalled = func() bool { return false }
		cloudflaredViaBrew = func() bool { return true }
		p := &mockPrompter{confirms: []confirmResp{{value: false}}}
		err := ensureCloudflaredInstalled(true, p)
		if err == nil || !errorMessageContains(err, "cloudflared is not installed") {
			t.Fatalf("want not-installed error after decline, got %v", err)
		}
	})
}

func TestRegisterWebhook_CreateErrorPropagates(t *testing.T) {
	client := &fakeWebhookClient{createErr: errors.New("boom")}
	_, _, err := registerWebhook(context.Background(), client, "https://x/webhook",
		[]string{domain.TriggerMessageCreated})
	if err == nil {
		t.Fatal("expected create error to propagate")
	}
}

func TestRegisterWebhook_EmptySecretIsRejectedAndCleanedUp(t *testing.T) {
	// Nylas returns a webhook but no secret — verification would be silently
	// off, so registerWebhook must fail AND delete the half-created webhook.
	client := &fakeWebhookClient{createResp: &domain.Webhook{ID: "no-secret-id", WebhookSecret: ""}}
	_, reg, err := registerWebhook(context.Background(), client, "https://x/webhook",
		[]string{domain.TriggerMessageCreated})
	if err == nil || !errorMessageContains(err, "without a signing secret") {
		t.Fatalf("want empty-secret error, got %v", err)
	}
	if reg != nil {
		t.Error("no registration handle should be returned on failure")
	}
	if len(client.deleted) != 1 || client.deleted[0] != "no-secret-id" {
		t.Errorf("half-created webhook not cleaned up: deleted=%v", client.deleted)
	}
}

func TestIsWebhookVerifyError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"typed code 70005", &domain.APIError{Type: "70005", Message: "unable.verify.webhook_url : unable to verify webhook URL"}, true},
		{"message mentions verify.webhook_url", &domain.APIError{Type: "400", Message: "unable.verify.webhook_url"}, true},
		{"unrelated api error", &domain.APIError{Type: "auth.unauthorized", Message: "bad key"}, false},
		// The reason we dropped the err.Error() fallback: a request ID that
		// happens to contain 70005 must NOT be treated as the verify error.
		{"request id contains 70005", &domain.APIError{Type: "auth.unauthorized", Message: "bad key", RequestID: "170005-abc"}, false},
		{"non-api error", errors.New("dial tcp: no such host"), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isWebhookVerifyError(tt.err); got != tt.want {
				t.Errorf("isWebhookVerifyError(%v) = %v, want %v", tt.err, got, tt.want)
			}
		})
	}
}

func TestRegisterTeardownDeletesWebhook(t *testing.T) {
	client := &fakeWebhookClient{}
	reg := &autoRegistration{client: client, webhookID: "del-me"}
	reg.teardown()
	if len(client.deleted) != 1 || client.deleted[0] != "del-me" {
		t.Errorf("deleted = %v, want [del-me]", client.deleted)
	}
}

// TestRunServer_RegisterFlagConflicts locks in the mutually-exclusive flag
// gates so --register can never be combined with manual-secret options (which
// would be silently ignored) or with --no-tunnel (which leaves no URL to
// register).
func TestRunServer_RegisterFlagConflicts(t *testing.T) {
	tests := []struct {
		name          string
		secret        string
		allowUnsigned bool
		noTunnel      bool
		wantContains  string
	}{
		{name: "secret", secret: "s", wantContains: "--secret cannot be combined"},
		{name: "allow-unsigned", allowUnsigned: true, wantContains: "--allow-unsigned cannot be combined"},
		{name: "no-tunnel", noTunnel: true, wantContains: "requires a public tunnel"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := runServer(0, "/webhook", "", tt.secret, tt.allowUnsigned, tt.noTunnel,
				true /* register */, []string{"message.created"}, false, true /* quiet */)
			if err == nil {
				t.Fatal("expected conflict error, got nil")
			}
			if !errorMessageContains(err, tt.wantContains) {
				t.Errorf("error = %v, want substring %q", err, tt.wantContains)
			}
		})
	}
}
