package webhook

import (
	"errors"
	"io"
	"testing"
)

// mockPrompter is a scripted preflightPrompter for unit tests. Each Confirm
// or Password call pops the next response off the corresponding queue.
type mockPrompter struct {
	confirms   []confirmResp
	passwords  []passwordResp
	tConfirms  int
	tPasswords int
}

type confirmResp struct {
	value bool
	err   error
}

type passwordResp struct {
	value string
	err   error
}

func (m *mockPrompter) Confirm(message string, defaultYes bool) (bool, error) {
	if m.tConfirms >= len(m.confirms) {
		return defaultYes, nil
	}
	r := m.confirms[m.tConfirms]
	m.tConfirms++
	return r.value, r.err
}

func (m *mockPrompter) Password(message string) (string, error) {
	if m.tPasswords >= len(m.passwords) {
		return "", nil
	}
	r := m.passwords[m.tPasswords]
	m.tPasswords++
	return r.value, r.err
}

// TestPreflightTunnelChoice_BypassedInScriptedModes confirms the preflight
// returns immediately (no prompting, no tunnel change) whenever the caller
// has already made an explicit choice. interactive=true is used here so the
// test verifies the *flag-based* short-circuits (--tunnel/--no-tunnel/
// --quiet/--json) — the non-interactive bypass is covered separately below.
func TestPreflightTunnelChoice_BypassedInScriptedModes(t *testing.T) {
	cases := []struct {
		name       string
		tunnelType string
		noTunnel   bool
		quiet      bool
		jsonOutput bool
	}{
		{name: "tunnel set", tunnelType: "cloudflared"},
		{name: "no-tunnel set", noTunnel: true},
		{name: "quiet mode", quiet: true},
		{name: "json mode", jsonOutput: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := &mockPrompter{}
			gotTunnel, gotSecret, gotAllow, exit, err := preflightTunnelChoice(
				p, true /* interactive */, tc.tunnelType, tc.noTunnel, tc.quiet, tc.jsonOutput, "", false,
			)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if exit {
				t.Fatalf("expected exit=false, got true")
			}
			if gotTunnel != tc.tunnelType {
				t.Errorf("tunnel: want %q, got %q", tc.tunnelType, gotTunnel)
			}
			if gotSecret != "" || gotAllow {
				t.Errorf("scripted mode must not modify secret/allowUnsigned")
			}
			if p.tConfirms != 0 || p.tPasswords != 0 {
				t.Errorf("scripted mode must not prompt the user (got %d confirms, %d passwords)",
					p.tConfirms, p.tPasswords)
			}
		})
	}
}

// TestPreflightTunnelChoice_NonInteractiveBypass confirms that when stdin is
// not a TTY (interactive=false), the preflight returns without prompting,
// regardless of which flags are set. This is the contract that scripts and
// CI runs depend on.
func TestPreflightTunnelChoice_NonInteractiveBypass(t *testing.T) {
	p := &mockPrompter{}
	gotTunnel, gotSecret, gotAllow, exit, err := preflightTunnelChoice(
		p, false /* interactive */, "", false, false, false, "", false,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if exit {
		t.Fatalf("non-interactive bypass must not request exit")
	}
	if gotTunnel != "" || gotSecret != "" || gotAllow {
		t.Fatalf("non-interactive bypass must not modify any value: tunnel=%q secret=%q allow=%v",
			gotTunnel, gotSecret, gotAllow)
	}
	if p.tConfirms != 0 || p.tPasswords != 0 {
		t.Fatalf("non-interactive bypass must not prompt (got %d confirms, %d passwords)",
			p.tConfirms, p.tPasswords)
	}
}

// stubCloudflaredInstalled overrides the package-level cloudflared probe so
// these tests can exercise the secret/unsigned prompts without depending on
// a real cloudflared binary on the test host.
func stubCloudflaredInstalled(t *testing.T, installed bool) {
	t.Helper()
	prev := cloudflaredInstalled
	cloudflaredInstalled = func() bool { return installed }
	t.Cleanup(func() { cloudflaredInstalled = prev })
}

// TestPreflightTunnelChoice_EOFOnSecretDoesNotEnableUnsigned verifies that
// pressing Ctrl-D at the secret prompt aborts the preflight rather than
// silently flipping the user into --allow-unsigned. This is the security
// gate that makes the empty-input → unsigned shortcut safe: cancellation
// must NEVER be misread as consent.
func TestPreflightTunnelChoice_EOFOnSecretDoesNotEnableUnsigned(t *testing.T) {
	stubCloudflaredInstalled(t, true)

	p := &mockPrompter{
		confirms: []confirmResp{
			{value: true, err: nil}, // "Enable cloudflared tunnel?"
		},
		passwords: []passwordResp{
			{value: "", err: io.EOF}, // user hits Ctrl-D at the secret prompt
		},
	}

	tunnelType, secret, allow, exit, err := preflightTunnelChoice(p, true /* interactive */, "", false, false, false, "", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !exit {
		t.Fatalf("EOF on the secret prompt must produce exit=true; got tunnel=%q secret=%q allow=%v",
			tunnelType, secret, allow)
	}
	if allow {
		t.Fatalf("EOF must NOT silently enable allowUnsigned")
	}
	if secret != "" {
		t.Fatalf("EOF must not return a secret; got %q", secret)
	}
}

// TestPreflightTunnelChoice_EmptySecretRequiresExplicitUnsignedConfirm
// verifies the second-prompt gate: an empty secret entry only flips into
// allowUnsigned when the user explicitly confirms the insecure choice at
// a second confirm prompt. Saying "no" at that gate exits cleanly.
func TestPreflightTunnelChoice_EmptySecretRequiresExplicitUnsignedConfirm(t *testing.T) {
	stubCloudflaredInstalled(t, true)

	p := &mockPrompter{
		confirms: []confirmResp{
			{value: true, err: nil},  // "Enable cloudflared tunnel?"
			{value: false, err: nil}, // "Accept unsigned events on the public tunnel?"
		},
		passwords: []passwordResp{
			{value: "", err: nil}, // empty input — but NOT EOF
		},
	}

	_, _, allow, exit, err := preflightTunnelChoice(p, true /* interactive */, "", false, false, false, "", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !exit {
		t.Fatalf("declining the unsigned-confirm gate must exit, not proceed")
	}
	if allow {
		t.Fatalf("declining the unsigned-confirm gate must NOT enable allowUnsigned")
	}
}

// TestPreflightTunnelChoice_EmptySecretWithExplicitConfirmEnablesUnsigned
// is the positive twin of the test above: when the user explicitly answers
// "yes" at the unsigned-confirm gate, allowUnsigned must be set.
func TestPreflightTunnelChoice_EmptySecretWithExplicitConfirmEnablesUnsigned(t *testing.T) {
	stubCloudflaredInstalled(t, true)

	p := &mockPrompter{
		confirms: []confirmResp{
			{value: true, err: nil}, // "Enable cloudflared tunnel?"
			{value: true, err: nil}, // "Accept unsigned events on the public tunnel?"
		},
		passwords: []passwordResp{
			{value: "", err: nil},
		},
	}

	got, _, allow, exit, err := preflightTunnelChoice(p, true /* interactive */, "", false, false, false, "", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if exit {
		t.Fatalf("explicit yes at unsigned-confirm gate must NOT request exit")
	}
	if !allow {
		t.Fatalf("explicit yes at unsigned-confirm gate must enable allowUnsigned")
	}
	if got != "cloudflared" {
		t.Fatalf("tunnel type: want cloudflared, got %q", got)
	}
}

// TestPreflightTunnelChoice_TunnelMutexErrorAtRunServer is a smoke test for
// the runServer-level mutual-exclusion check (--tunnel + --no-tunnel
// rejected). Kept here next to the preflight tests so the security gate
// is visible to anyone reading the file.
func TestPreflightTunnelChoice_TunnelMutexErrorAtRunServer(t *testing.T) {
	err := runServer(0, "/webhook", "cloudflared", "", false, true /* noTunnel */, false, true /* quiet */)
	if err == nil {
		t.Fatal("expected --tunnel + --no-tunnel to error, got nil")
	}
	if !errorMessageContains(err, "cannot be combined") {
		t.Errorf("error message should mention mutual exclusion, got: %v", err)
	}
}

func TestWebhookServerConfigEnablesReplayWindowForSignedEvents(t *testing.T) {
	config := newWebhookServerConfig(3000, "/webhook", "cloudflared", "secret")
	if config.MaxEventAge != defaultSignedWebhookMaxEventAge {
		t.Fatalf("MaxEventAge = %v, want %v", config.MaxEventAge, defaultSignedWebhookMaxEventAge)
	}
}

func TestWebhookServerConfigLeavesReplayWindowDisabledWithoutSecret(t *testing.T) {
	config := newWebhookServerConfig(3000, "/webhook", "cloudflared", "")
	if config.MaxEventAge != 0 {
		t.Fatalf("MaxEventAge = %v, want disabled", config.MaxEventAge)
	}
}

func errorMessageContains(err error, substr string) bool {
	if err == nil {
		return false
	}
	type causer interface{ Unwrap() error }
	for cur := err; cur != nil; {
		if msg := cur.Error(); msg != "" {
			if containsSubstring(msg, substr) {
				return true
			}
		}
		c, ok := cur.(causer)
		if !ok {
			break
		}
		cur = c.Unwrap()
	}
	return false
}

func containsSubstring(s, substr string) bool {
	if substr == "" {
		return true
	}
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// Ensure the file-level errors package import gets exercised by at least one
// test (errors.Is) — keeps `go vet` from complaining about an unused import
// if future tests stop using errors.* directly.
var _ = errors.Is
