package webhook

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/domain"
	"github.com/nylas/cli/internal/ports"
)

// Retry budget for the create-time verification race. A fresh cloudflared
// quick-tunnel hostname can take up to ~a minute to resolve on Nylas's side,
// so CreateWebhook is retried (only on the verify-failure error) until it
// propagates. Package vars so tests can shrink them.
var (
	registerVerifyTimeout = 90 * time.Second
	registerRetryInterval = 4 * time.Second
)

// autoWebhookDescription tags webhooks created by `webhooks server --register`
// so they can be identified and swept on a later start (see registerWebhook).
const autoWebhookDescription = "nylas-cli webhook server (auto-registered)"

// autoRegistration is the teardown handle for a webhook created by --register.
type autoRegistration struct {
	client    ports.WebhookClient
	webhookID string
}

// resolveRegisterTriggers returns the validated trigger list for --register,
// prompting interactively when none were passed on the command line. In
// non-interactive mode an empty list is a hard error — we won't guess what a
// scripted caller meant to subscribe to.
func resolveRegisterTriggers(triggers []string, interactive bool, p preflightPrompter) ([]string, error) {
	if len(triggers) == 0 {
		if !interactive {
			return nil, common.NewUserError(
				"--triggers is required with --register",
				"Pass --triggers message.created (comma-separated for multiple). Run 'nylas webhooks triggers' to list them.",
			)
		}
		entered, err := p.Ask("Trigger types to subscribe to (comma-separated)", domain.TriggerMessageCreated)
		if err != nil {
			return nil, err
		}
		triggers = []string{entered}
	}
	return parseAndValidateTriggers(triggers)
}

// ensureCloudflaredInstalled fails fast (before we prompt for triggers or touch
// the API) when cloudflared is missing. On an interactive macOS shell it offers
// the same brew auto-install the normal tunnel preflight does; otherwise it
// returns actionable install instructions.
func ensureCloudflaredInstalled(interactive bool, prompter preflightPrompter) error {
	if cloudflaredInstalled() {
		return nil
	}
	if interactive && cloudflaredViaBrew() {
		confirm, err := prompter.Confirm("cloudflared is not installed. Install it via brew now?", true)
		if err == nil && confirm {
			if ierr := installCloudflaredFn(); ierr == nil && cloudflaredInstalled() {
				return nil
			}
		}
	}
	return common.NewUserError(
		"cloudflared is not installed",
		"Install it with: brew install cloudflared (macOS) or see "+
			"https://developers.cloudflare.com/cloudflare-one/connections/connect-apps/install-and-setup/installation/",
	)
}

// isWebhookVerifyError reports whether err is Nylas rejecting the webhook URL
// because it couldn't reach/verify it yet (error 70005). This is the transient
// case while a fresh quick-tunnel hostname propagates through DNS — safe and
// expected to retry. Any other error (bad triggers, auth, quota) is terminal.
//
// It matches the typed APIError fields only (Nylas returns Type="70005" with
// the symbolic code in Message); matching the formatted err.Error() string
// would false-positive on an unrelated error whose request ID happened to
// contain "70005".
func isWebhookVerifyError(err error) bool {
	var apiErr *domain.APIError
	if !errors.As(err, &apiErr) {
		return false
	}
	if apiErr.Type == "70005" {
		return true
	}
	return strings.Contains(apiErr.Message, "verify.webhook_url") ||
		strings.Contains(apiErr.Message, "verify webhook URL")
}

// registerWebhook deletes any stale auto-registered webhooks left by a previous
// crash, then creates a fresh webhook pointing at publicURL. Nylas verifies the
// URL synchronously at create time, so creation is retried while that
// verification keeps failing (the tunnel hostname is still propagating). The
// returned secret is the one Nylas minted (held in memory only) plus a teardown
// handle.
func registerWebhook(ctx context.Context, client ports.WebhookClient, publicURL string, triggers []string) (string, *autoRegistration, error) {
	// Sweep stale auto webhooks from a prior hard-kill (Ctrl+C deletes cleanly;
	// `kill -9`/crash does not). NOTE: this also removes the auto webhook of a
	// *concurrent* --register session on the same Nylas app — acceptable for a
	// local dev tool; switch to a per-process tag if concurrent servers matter.
	if existing, err := client.ListWebhooks(ctx); err == nil {
		for _, wh := range existing {
			if wh.Description == autoWebhookDescription {
				_ = client.DeleteWebhook(ctx, wh.ID)
			}
		}
	}

	req := &domain.CreateWebhookRequest{
		WebhookURL:   publicURL,
		TriggerTypes: triggers,
		Description:  autoWebhookDescription,
	}

	// Bound the whole verify-retry sequence. CreateWebhook is given the deadline
	// context (not the parent ctx) so a single slow attempt can't exceed the
	// budget via the client's own per-request timeout.
	deadline, cancel := context.WithTimeout(ctx, registerVerifyTimeout)
	defer cancel()

	for {
		wh, err := client.CreateWebhook(deadline, req)
		if err == nil {
			// Nylas must return a signing secret — without it, verification
			// would be silently disabled while we report it as on. Treat a
			// missing secret as a failure and remove the half-created webhook.
			if wh == nil || wh.ID == "" {
				return "", nil, common.NewUserError("webhook create returned no webhook", "Retry --register.")
			}
			if wh.WebhookSecret == "" {
				// Detached context so the cleanup delete still runs even if the
				// parent ctx was already cancelled (e.g. Ctrl+C raced create).
				cleanupCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				delErr := client.DeleteWebhook(cleanupCtx, wh.ID)
				cancel()
				if delErr != nil {
					return "", nil, common.NewUserError(
						fmt.Sprintf("webhook %s was created without a signing secret and could not be removed: %v", wh.ID, delErr),
						"Events could not be verified. Delete it manually: nylas webhooks delete "+wh.ID,
					)
				}
				return "", nil, common.NewUserError(
					"webhook was created without a signing secret",
					"Nylas returned no secret, so events could not be verified — the webhook was removed. "+
						"Retry --register, or register manually with a known secret.",
				)
			}
			return wh.WebhookSecret, &autoRegistration{client: client, webhookID: wh.ID}, nil
		}

		// Parent context cancelled (e.g. Ctrl+C) — abort the registration
		// promptly instead of running out the retry budget.
		if ctx.Err() != nil {
			return "", nil, ctx.Err()
		}
		// Terminal error (bad triggers, auth, quota). A deadline timeout also
		// surfaces here as a non-verify error, so only fail fast while the
		// budget is still alive; otherwise fall through to the budget message.
		if !isWebhookVerifyError(err) && deadline.Err() == nil {
			return "", nil, common.WrapCreateError("webhook", err)
		}
		// Verification failed because the tunnel URL isn't reachable from Nylas
		// yet. Wait and retry until it propagates or we run out of budget.
		select {
		case <-deadline.Done():
			if ctx.Err() != nil {
				return "", nil, ctx.Err()
			}
			return "", nil, common.NewUserError(
				"Nylas could not reach the tunnel URL in time",
				"A fresh cloudflared URL can take up to a minute to resolve globally. "+
					"Re-run --register, or register the URL manually once it's reachable.",
			)
		case <-time.After(registerRetryInterval):
		}
	}
}

// teardown deletes the auto-registered webhook. It uses its own context so the
// delete still runs even though the server's context was cancelled on shutdown.
// A failed delete is surfaced (not swallowed) so the user can remove the now-
// orphaned webhook — it points at a dead tunnel.
func (r *autoRegistration) teardown() {
	if r == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := r.client.DeleteWebhook(ctx, r.webhookID); err != nil {
		fmt.Fprintf(os.Stderr, "warn: could not delete auto-registered webhook %s: %v\n", r.webhookID, err)
		fmt.Fprintf(os.Stderr, "      remove it manually: nylas webhooks delete %s\n", r.webhookID)
	}
}
