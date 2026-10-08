package dashboard

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/nylas/cli/internal/domain"
	"github.com/nylas/cli/internal/ports"
)

// OAuthAccessTokens hands out an OAuth access token that stays valid for at
// least minValid, refreshing it when needed. oauthlogin.Service implements it.
type OAuthAccessTokens interface {
	AccessTokenValidFor(ctx context.Context, minValid time.Duration) (string, error)
}

const sessionOriginOAuth = "oauth"

// A session is re-exchanged this long before it expires, so a command never
// starts with one that lapses mid-request. The exchanged session ends when
// the OAuth access token does, so the token is asked for with at least this
// much life left too; otherwise the re-exchange would return the same expiry.
const oauthSessionRenewBefore = time.Minute

// ErrDashboardLoginSessionKept is returned by SessionRenewer.Login when a
// session from `nylas dashboard login` is stored. It is left in place: its
// app selection and organization are the user's, and replacing it would leave
// it live on the server.
var ErrDashboardLoginSessionKept = errors.New("the dashboard commands stay signed in with `nylas dashboard login`")

var oauthSessionStateKeys = append(
	append([]string{}, dashboardSessionStateKeys...),
	ports.KeyDashboardSessionOrigin,
	ports.KeyDashboardSessionExpiresAt,
	ports.KeyDashboardSessionServer,
)

// SessionRenewer keeps a dashboard session that came from `nylas oauth login`
// current. The server refuses to refresh such a session, so it is exchanged
// again from a fresh OAuth access token instead.
type SessionRenewer struct {
	account ports.DashboardAccountClient
	secrets ports.SecretStore
	tokens  OAuthAccessTokens
	lock    ports.CrossProcessLock
	server  string
	now     func() time.Time
}

// NewSessionRenewer creates a renewer. tokens may be nil, in which case an
// expired OAuth session can only be replaced by logging in again.
//
// lock serialises every write to the dashboard session across processes. It
// is required: the session is several keys, and two processes renewing at
// once would interleave them, pairing one exchange's user token with
// another's org token. The lock is taken before the OAuth session's own lock
// (tokens may refresh under it), never after.
func NewSessionRenewer(account ports.DashboardAccountClient, secrets ports.SecretStore, tokens OAuthAccessTokens, lock ports.CrossProcessLock) *SessionRenewer {
	return &SessionRenewer{account: account, secrets: secrets, tokens: tokens, lock: lock, now: time.Now}
}

// WithServer records server on every session this renewer stores; see
// AuthService.WithServer.
func (r *SessionRenewer) WithServer(server string) *SessionRenewer {
	r.server = server
	return r
}

func (r *SessionRenewer) withLock(ctx context.Context, fn func() error) error {
	if r.lock == nil {
		return errors.New("dashboard session lock is not configured")
	}
	unlock, err := r.lock.Lock(ctx)
	if err != nil {
		return fmt.Errorf("failed to acquire the dashboard session lock: %w", err)
	}
	fnErr := fn()
	if err := unlock(); err != nil && fnErr == nil {
		return fmt.Errorf("failed to release the dashboard session lock: %w", err)
	}
	return fnErr
}

// Login exchanges the OAuth session for a dashboard session and stores it,
// clearing the active app selection like any other login. A session from
// `nylas dashboard login` is kept, and ErrDashboardLoginSessionKept returned.
func (r *SessionRenewer) Login(ctx context.Context) (*domain.DashboardOAuthExchangeResponse, error) {
	var resp *domain.DashboardOAuthExchangeResponse
	err := r.withLock(ctx, func() error {
		kept, err := r.hasDashboardLoginSession()
		if err != nil {
			return err
		}
		if kept {
			return ErrDashboardLoginSessionKept
		}
		resp, err = r.exchange(ctx, true)
		if err != nil && isOAuthSession(r.secrets) {
			// The OAuth session this one was exchanged from has just been
			// replaced. Keeping it would leave every dashboard command acting
			// as the previous user or organization until it expires.
			clearErr := NewAuthService(r.account, r.secrets).WithServer(r.server).Logout(ctx)
			return errors.Join(err, clearErr)
		}
		return err
	})
	return resp, err
}

// EnsureFresh re-exchanges the stored session when it came from OAuth and is
// about to expire. Sessions from `nylas dashboard login` are left alone.
func (r *SessionRenewer) EnsureFresh(ctx context.Context) error {
	if r == nil || !isOAuthSession(r.secrets) || !r.needsRenewal() {
		return nil
	}
	return r.withLock(ctx, func() error {
		// Another process may have renewed while this one waited.
		if !isOAuthSession(r.secrets) || !r.needsRenewal() {
			return nil
		}
		_, err := r.exchange(ctx, false)
		return err
	})
}

// renewRejected replaces a session the server just refused. rejected is the
// user token it would not accept: if another process has already stored a
// replacement, that is returned instead of exchanging again. The replacement
// is read through loadDashboardTokens, so one stored for other servers is
// refused like on every other load path rather than sent here.
func (r *SessionRenewer) renewRejected(ctx context.Context, rejected string) (userToken, orgToken string, err error) {
	err = r.withLock(ctx, func() error {
		if isOAuthSession(r.secrets) && !r.needsRenewal() {
			u, o, loadErr := loadDashboardTokens(r.secrets, r.server)
			if loadErr == nil && u != rejected {
				userToken, orgToken = u, o
				return nil
			}
			if loadErr != nil && !errors.Is(loadErr, domain.ErrDashboardNotLoggedIn) {
				return loadErr
			}
		}
		resp, err := r.exchange(ctx, false)
		if err != nil {
			return err
		}
		userToken, orgToken = resp.UserToken, resp.OrgToken
		return nil
	})
	return userToken, orgToken, err
}

func (r *SessionRenewer) needsRenewal() bool {
	raw, _ := r.secrets.Get(ports.KeyDashboardSessionExpiresAt)
	expiresAt, err := time.Parse(time.RFC3339, raw)
	return err != nil || !r.now().Add(oauthSessionRenewBefore).Before(expiresAt)
}

// exchange must be called with the lock held.
func (r *SessionRenewer) exchange(ctx context.Context, resetAppSelection bool) (*domain.DashboardOAuthExchangeResponse, error) {
	if r.tokens == nil {
		return nil, fmt.Errorf("%w: run `nylas oauth login` again", domain.ErrDashboardSessionExpired)
	}
	accessToken, err := r.tokens.AccessTokenValidFor(ctx, oauthSessionRenewBefore)
	if err != nil {
		return nil, err
	}
	// The server refuses a token without dashboard.session with a bare
	// INVALID_TOKEN. Reading the unverified claim here only turns that into
	// an actionable message; the server still decides.
	if claims, err := domain.DecodeOAuthAccessToken(accessToken); err == nil &&
		!slices.Contains(claims.Scopes(), domain.OAuthScopeDashboardSession) {
		return nil, fmt.Errorf("%w: run `nylas oauth login` again and allow dashboard access", domain.ErrDashboardSessionNotConsented)
	}
	resp, err := r.account.ExchangeOAuthToken(ctx, accessToken)
	if err != nil {
		return nil, err
	}
	if resp == nil || resp.UserToken == "" || resp.ExpiresAt.IsZero() {
		return nil, errors.New("the dashboard session exchange returned no session")
	}

	updates := map[string]*string{
		ports.KeyDashboardUserToken:        stringPtrOrNil(resp.UserToken),
		ports.KeyDashboardOrgToken:         stringPtrOrNil(resp.OrgToken),
		ports.KeyDashboardUserPublicID:     stringPtrOrNil(resp.User.PublicID),
		ports.KeyDashboardOrgPublicID:      stringPtrOrNil(resp.OrgPublicID),
		ports.KeyDashboardSessionOrigin:    stringPtrOrNil(sessionOriginOAuth),
		ports.KeyDashboardSessionExpiresAt: stringPtrOrNil(resp.ExpiresAt.UTC().Format(time.RFC3339)),
		ports.KeyDashboardSessionServer:    stringPtrOrNil(r.server),
	}
	if resetAppSelection {
		updates[ports.KeyDashboardAppID] = nil
		updates[ports.KeyDashboardAppRegion] = nil
	}
	if err := NewAuthService(r.account, r.secrets).WithServer(r.server).replaceSecretValues(oauthSessionStateKeys, updates); err != nil {
		return nil, fmt.Errorf("failed to store dashboard session: %w", err)
	}
	return resp, nil
}

// ClearIfOAuth ends the dashboard session when it came from OAuth, so
// `nylas oauth logout` does not leave dashboard access behind. A session from
// `nylas dashboard login` is not this command's to end.
func (r *SessionRenewer) ClearIfOAuth(ctx context.Context) error {
	clearErr, _ := r.ClearIfOAuthThen(ctx, func() error { return nil })
	return clearErr
}

// ClearIfOAuthThen is ClearIfOAuth followed by then, both under the lock.
// `nylas oauth logout` passes the OAuth logout as then: the dashboard session
// has to go first, while the OAuth session's server is still stored to tell
// where an unrecorded session came from, and holding the lock across both
// stops a concurrent login from exchanging a new dashboard session in between.
func (r *SessionRenewer) ClearIfOAuthThen(ctx context.Context, then func() error) (clearErr, thenErr error) {
	if r == nil {
		return nil, then()
	}
	ran := false
	lockErr := r.withLock(ctx, func() error {
		ran = true
		// A keyring that cannot say where the session came from is reported,
		// not read as "not from OAuth": that would leave the session behind
		// while the logout reports success.
		switch fromOAuth, err := oauthSessionOrigin(r.secrets); {
		case err != nil:
			clearErr = err
		case fromOAuth:
			clearErr = NewAuthService(r.account, r.secrets).WithServer(r.server).Logout(ctx)
		}
		thenErr = then()
		return nil
	})
	if !ran {
		return lockErr, then()
	}
	// The lock could not be released after both ran. Report it, but do not
	// run then again: a concurrent login may have stored a session since.
	return errors.Join(clearErr, lockErr), thenErr
}

func isOAuthSession(secrets ports.SecretStore) bool {
	fromOAuth, err := oauthSessionOrigin(secrets)
	return err == nil && fromOAuth
}

// oauthSessionOrigin reports whether the stored session was exchanged from
// `nylas oauth login`, telling a failed read apart from no marker.
func oauthSessionOrigin(secrets ports.SecretStore) (bool, error) {
	origin, err := secrets.Get(ports.KeyDashboardSessionOrigin)
	if errors.Is(err, domain.ErrSecretNotFound) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("failed to read the dashboard session origin: %w", err)
	}
	return origin == sessionOriginOAuth, nil
}

// hasDashboardLoginSession reports a stored session that did not come from
// `nylas oauth login` and is usable against this renewer's server. One for
// other servers would be refused anyway, so it does not block a login. A
// keyring that cannot be read is reported rather than taken for "no session",
// which would let the exchange overwrite one.
func (r *SessionRenewer) hasDashboardLoginSession() (bool, error) {
	if isOAuthSession(r.secrets) {
		return false, nil
	}
	_, _, err := loadDashboardTokens(r.secrets, r.server)
	switch {
	case err == nil:
		return true, nil
	case errors.Is(err, domain.ErrDashboardNotLoggedIn), errors.Is(err, domain.ErrDashboardServerMismatch):
		return false, nil
	default:
		return false, err
	}
}

// errOAuthSessionNotRefreshable is returned by refreshTokens for an OAuth
// session when no renewer is wired, instead of calling a refresh the server
// will refuse.
var errOAuthSessionNotRefreshable = errors.New("this dashboard session came from `nylas oauth login` and cannot be refreshed; run it again")
