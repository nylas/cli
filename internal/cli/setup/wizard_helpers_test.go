package setup

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/nylas/cli/internal/domain"
)

// stubRegistrar is a test double for the dashboard domain service.
type stubRegistrar struct {
	created  *domain.DashboardInboxDomain
	err      error
	calls    int
	gotInput domain.DashboardCreateInboxDomainInput
}

func (s *stubRegistrar) CreateDomain(
	_ context.Context,
	input domain.DashboardCreateInboxDomainInput,
) (*domain.DashboardInboxDomain, error) {
	s.calls++
	s.gotInput = input
	return s.created, s.err
}

// withAgentDomainStubs swaps the prompt + service seams for the duration of a test.
func withAgentDomainStubs(t *testing.T, confirm bool, label string, reg *stubRegistrar) {
	t.Helper()
	origConfirm, origInput, origSvc := confirmPromptFn, inputPromptFn, createDomainServiceFn
	t.Cleanup(func() {
		confirmPromptFn, inputPromptFn, createDomainServiceFn = origConfirm, origInput, origSvc
	})
	confirmPromptFn = func(string, bool) (bool, error) { return confirm, nil }
	inputPromptFn = func(string, string) (string, error) { return label, nil }
	createDomainServiceFn = func() (domainRegistrar, error) { return reg, nil }
}

func TestStepAgentDomain_RegistersDomain(t *testing.T) {
	// Why: a successful registration must record the domain on status so the
	// completion summary can show a ready-to-run create command for it.
	reg := &stubRegistrar{created: &domain.DashboardInboxDomain{
		ID:            "dom_1",
		DomainAddress: "acme.nylas.email",
		Region:        "us",
	}}
	withAgentDomainStubs(t, true, "ACME", reg) // mixed case → normalized to lowercase

	status := SetupStatus{HasDashboardAuth: true, ActiveAppRegion: "us"}
	_ = captureStdout(t, func() { stepAgentDomain(&status) })

	if reg.calls != 1 {
		t.Fatalf("expected CreateDomain called once, got %d", reg.calls)
	}
	if status.AgentDomain != "acme.nylas.email" {
		t.Fatalf("expected AgentDomain set to acme.nylas.email, got %q", status.AgentDomain)
	}
	// The lowercased label + active region must flow into the request unchanged.
	if reg.gotInput.DomainAddress != "acme.nylas.email" || reg.gotInput.Region != "us" {
		t.Fatalf("unexpected create input: %+v", reg.gotInput)
	}
}

func TestStepAgentDomain_RegionPrompt(t *testing.T) {
	// Why: when the active app region is unknown the user is prompted. A prompt
	// error must fail loud and skip; a chosen region must flow into the request.
	t.Run("prompt error skips and surfaces message", func(t *testing.T) {
		reg := &stubRegistrar{}
		withAgentDomainStubs(t, true, "acme", reg)
		origSel := selectRegionFn
		t.Cleanup(func() { selectRegionFn = origSel })
		selectRegionFn = func() (string, error) { return "", errors.New("no tty") }

		// Note: the "Skipped" message uses the color package's cached writer, which
		// captureStdout can't intercept, so we assert behavior (no API call, no
		// domain recorded) rather than the message text.
		status := SetupStatus{HasDashboardAuth: true} // empty region → prompt
		_ = captureStdout(t, func() { stepAgentDomain(&status) })

		if reg.calls != 0 {
			t.Fatalf("expected no API call on region error, got %d", reg.calls)
		}
		if status.AgentDomain != "" {
			t.Fatalf("expected no domain registered, got %q", status.AgentDomain)
		}
	})

	t.Run("chosen region flows into request", func(t *testing.T) {
		reg := &stubRegistrar{created: &domain.DashboardInboxDomain{
			ID:            "dom_1",
			DomainAddress: "acme.nylas.email",
			Region:        "eu",
		}}
		withAgentDomainStubs(t, true, "acme", reg)
		origSel := selectRegionFn
		t.Cleanup(func() { selectRegionFn = origSel })
		selectRegionFn = func() (string, error) { return "eu", nil }

		status := SetupStatus{HasDashboardAuth: true}
		_ = captureStdout(t, func() { stepAgentDomain(&status) })

		if status.AgentDomain != "acme.nylas.email" {
			t.Fatalf("expected AgentDomain set, got %q", status.AgentDomain)
		}
		if reg.gotInput.Region != "eu" {
			t.Fatalf("expected region eu in request, got %q", reg.gotInput.Region)
		}
	})
}

func TestStepAgentDomain_DoesNotRegister(t *testing.T) {
	tests := []struct {
		name     string
		confirm  bool
		label    string
		reg      *stubRegistrar
		wantCall bool
	}{
		{
			name:    "user declines",
			confirm: false,
			label:   "acme",
			reg:     &stubRegistrar{},
		},
		{
			name:    "invalid subdomain never calls the API",
			confirm: true,
			label:   "-bad-",
			reg:     &stubRegistrar{},
		},
		{
			name:     "create error",
			confirm:  true,
			label:    "acme",
			reg:      &stubRegistrar{err: errors.New("boom")},
			wantCall: true,
		},
		{
			name:     "region mismatch in response is rejected",
			confirm:  true,
			label:    "acme",
			reg:      &stubRegistrar{created: &domain.DashboardInboxDomain{ID: "dom_1", DomainAddress: "acme.nylas.email", Region: "eu"}},
			wantCall: true,
		},
		{
			name:     "missing ID in response is rejected",
			confirm:  true,
			label:    "acme",
			reg:      &stubRegistrar{created: &domain.DashboardInboxDomain{DomainAddress: "acme.nylas.email", Region: "us"}},
			wantCall: true,
		},
		{
			name:     "address mismatch in response is rejected",
			confirm:  true,
			label:    "acme",
			reg:      &stubRegistrar{created: &domain.DashboardInboxDomain{ID: "dom_1", DomainAddress: "other.nylas.email", Region: "us"}},
			wantCall: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			withAgentDomainStubs(t, tt.confirm, tt.label, tt.reg)
			status := SetupStatus{HasDashboardAuth: true, ActiveAppRegion: "us"}
			_ = captureStdout(t, func() { stepAgentDomain(&status) })

			if status.AgentDomain != "" {
				t.Fatalf("expected no domain registered, got %q", status.AgentDomain)
			}
			if (tt.reg.calls > 0) != tt.wantCall {
				t.Fatalf("CreateDomain call mismatch: called=%v want=%v", tt.reg.calls > 0, tt.wantCall)
			}
		})
	}
}

// captureStdout redirects os.Stdout while fn runs and returns what was written.
// printComplete writes plain text via fmt, so the asserted lines are captured.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	orig := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	os.Stdout = w
	defer func() { os.Stdout = orig }()

	fn()
	_ = w.Close()

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		t.Fatalf("read captured stdout: %v", err)
	}
	return buf.String()
}

func TestPrintComplete_AgentDomainBranch(t *testing.T) {
	// Why: when Step 5 registers a domain, the final summary must show a
	// ready-to-run create command for that exact domain and drop the generic
	// "register a domain" instructions; without a domain it must show the
	// fallback instructions instead.
	tests := []struct {
		name            string
		status          SetupStatus
		wantContains    []string
		wantNotContains []string
	}{
		{
			name:   "registered domain shows ready-to-run create command",
			status: SetupStatus{AgentDomain: "acme.nylas.email"},
			wantContains: []string{
				"Create an Agent Account on your domain:",
				"nylas agent account create user@acme.nylas.email",
			},
			wantNotContains: []string{
				"Register a free Agent Account email domain:",
				"<subdomain>.nylas.email",
			},
		},
		{
			name:   "no domain shows manual registration instructions",
			status: SetupStatus{ActiveAppRegion: "us"},
			wantContains: []string{
				"Register a free Agent Account email domain:",
				"nylas dashboard domains create <subdomain>.nylas.email --region us",
				"nylas agent account create user@<subdomain>.nylas.email",
			},
			wantNotContains: []string{
				"Create an Agent Account on your domain:",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out := captureStdout(t, func() { printComplete(tt.status) })
			for _, s := range tt.wantContains {
				if !strings.Contains(out, s) {
					t.Errorf("output missing %q\n---\n%s", s, out)
				}
			}
			for _, s := range tt.wantNotContains {
				if strings.Contains(out, s) {
					t.Errorf("output unexpectedly contains %q\n---\n%s", s, out)
				}
			}
		})
	}
}

func TestEnsureSetupCallbackURI_AllowsManualFallbackWhenProvisioningFails(t *testing.T) {
	originalProvisioner := setupCallbackProvisioner
	t.Cleanup(func() {
		setupCallbackProvisioner = originalProvisioner
	})

	setupCallbackProvisioner = func(apiKey, clientID, region string, callbackPort int) (*CallbackURIProvisionResult, error) {
		return &CallbackURIProvisionResult{
			RequiredURI: "http://localhost:9007/callback",
		}, errors.New("admin api unavailable")
	}

	if err := ensureSetupCallbackURI("nyl_test", "client-123", "us"); err != nil {
		t.Fatalf("expected setup callback URI failure to degrade gracefully, got %v", err)
	}
}

func TestEnsureSetupCallbackURI_RequiresClientID(t *testing.T) {
	if err := ensureSetupCallbackURI("nyl_test", "", "us"); err == nil {
		t.Fatal("expected empty client ID to fail")
	}
}

func TestAgentSubdomainPattern(t *testing.T) {
	// Why: the chosen label becomes part of <label>.nylas.email registered against
	// the dashboard. Rejecting bad input locally avoids a wasted round-trip and a
	// confusing server error during first-run setup.
	tests := []struct {
		label string
		want  bool
	}{
		{"acme", true},
		{"acme-bots", true},
		{"a1", true},
		{"a", true},
		{"", false},
		{"-acme", false},
		{"acme-", false},
		{"ACME", false}, // callers lowercase first; uppercase must not slip through
		{"acme.bots", false},
		{"acme bots", false},
		{"acme_bots", false},
	}
	for _, tt := range tests {
		if got := agentSubdomainPattern.MatchString(tt.label); got != tt.want {
			t.Errorf("agentSubdomainPattern.MatchString(%q) = %v, want %v", tt.label, got, tt.want)
		}
	}
}

func TestStepAgentDomain_SkipsWithoutDashboardAuth(t *testing.T) {
	// Why: registering a managed .nylas.email domain needs a dashboard session
	// (user/org tokens), not just an API key. The API-key-only path must skip the
	// step entirely — never prompt, never call the dashboard, never claim a domain.
	status := SetupStatus{HasDashboardAuth: false}
	stepAgentDomain(&status)
	if status.AgentDomain != "" {
		t.Fatalf("expected no domain registered without dashboard auth, got %q", status.AgentDomain)
	}
}

func TestDomainRegistrationCommands(t *testing.T) {
	tests := []struct {
		name   string
		status SetupStatus
		want   []string
	}{
		{
			name:   "uses US active app region",
			status: SetupStatus{ActiveAppRegion: "us"},
			want: []string{
				"nylas dashboard domains create <subdomain>.nylas.email --region us",
			},
		},
		{
			name:   "uses EU active app region",
			status: SetupStatus{ActiveAppRegion: "eu"},
			want: []string{
				"nylas dashboard domains create <subdomain>.nylas.email --region eu",
			},
		},
		{
			name:   "falls back to copy-pasteable commands when active app region is unknown",
			status: SetupStatus{},
			want: []string{
				"nylas dashboard domains create <subdomain>.nylas.email --region us",
				"nylas dashboard domains create <subdomain>.nylas.email --region eu",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := domainRegistrationCommands(tt.status)
			if len(got) != len(tt.want) {
				t.Fatalf("expected %d commands, got %d: %v", len(tt.want), len(got), got)
			}
			for i := range tt.want {
				if got[i] != tt.want[i] {
					t.Fatalf("expected command %d to be %q, got %q", i, tt.want[i], got[i])
				}
			}
		})
	}
}
