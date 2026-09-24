package dashboard

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/nylas/cli/internal/domain"
	"github.com/nylas/cli/internal/ports"
)

// OAuthAccessTokens hands out a currently valid OAuth access token, refreshing
// it when needed. oauthlogin.Service implements it.
type OAuthAccessTokens interface {
	AccessToken(ctx context.Context) (string, error)
}

const sessionOriginOAuth = "oauth"

// A session is re-exchanged this long before it expires, so a command never
// starts with one that lapses mid-request.
const oauthSessionRenewBefore = time.Minute

var oauthSessionStateKeys = append(
	append([]string{}, dashboardSessionStateKeys...),
	ports.KeyDashboardSessionOrigin,
	ports.KeyDashboardSessionExpiresAt,
)

// SessionRenewer keeps a dashboard session that came from `nylas oauth login`
// current. The server refuses to refresh such a session, so it is exchanged
// again from a fresh OAuth access token instead.
type SessionRenewer struct {
	account ports.DashboardAccountClient
	secrets ports.SecretStore
	tokens  OAuthAccessTokens
	now     func() time.Time
}

// NewSessionRenewer creates a renewer. tokens may be nil, in which case an
// expired OAuth session can only be replaced by logging in again.
func NewSessionRenewer(account ports.DashboardAccountClient, secrets ports.SecretStore, tokens OAuthAccessTokens) *SessionRenewer {
	return &SessionRenewer{account: account, secrets: secrets, tokens: tokens, now: time.Now}
}

// Login exchanges the OAuth session for a dashboard session and stores it,
// clearing the active app selection like any other login.
func (r *SessionRenewer) Login(ctx context.Context) (*domain.DashboardOAuthExchangeResponse, error) {
	return r.exchange(ctx, true)
}

// EnsureFresh re-exchanges the stored session when it came from OAuth and is
// about to expire. Sessions from `nylas dashboard login` are left alone.
func (r *SessionRenewer) EnsureFresh(ctx context.Context) error {
	if r == nil || !isOAuthSession(r.secrets) {
		return nil
	}
	raw, _ := r.secrets.Get(ports.KeyDashboardSessionExpiresAt)
	expiresAt, err := time.Parse(time.RFC3339, raw)
	if err == nil && r.now().Add(oauthSessionRenewBefore).Before(expiresAt) {
		return nil
	}
	_, err = r.exchange(ctx, false)
	return err
}

func (r *SessionRenewer) exchange(ctx context.Context, resetAppSelection bool) (*domain.DashboardOAuthExchangeResponse, error) {
	if r.tokens == nil {
		return nil, fmt.Errorf("%w: run `nylas oauth login` again", domain.ErrDashboardSessionExpired)
	}
	accessToken, err := r.tokens.AccessToken(ctx)
	if err != nil {
		return nil, err
	}
	resp, err := r.account.ExchangeOAuthToken(ctx, accessToken)
	if err != nil {
		return nil, err
	}

	updates := map[string]*string{
		ports.KeyDashboardUserToken:        stringPtrOrNil(resp.UserToken),
		ports.KeyDashboardOrgToken:         stringPtrOrNil(resp.OrgToken),
		ports.KeyDashboardUserPublicID:     stringPtrOrNil(resp.User.PublicID),
		ports.KeyDashboardOrgPublicID:      stringPtrOrNil(resp.OrgPublicID),
		ports.KeyDashboardSessionOrigin:    stringPtrOrNil(sessionOriginOAuth),
		ports.KeyDashboardSessionExpiresAt: stringPtrOrNil(resp.ExpiresAt.UTC().Format(time.RFC3339)),
	}
	if resetAppSelection {
		updates[ports.KeyDashboardAppID] = nil
		updates[ports.KeyDashboardAppRegion] = nil
	}
	if err := NewAuthService(r.account, r.secrets).replaceSecretValues(oauthSessionStateKeys, updates); err != nil {
		return nil, fmt.Errorf("failed to store dashboard session: %w", err)
	}
	return resp, nil
}

// ClearIfOAuth ends the dashboard session when it came from OAuth, so
// `nylas oauth logout` does not leave dashboard access behind. A session from
// `nylas dashboard login` is not this command's to end.
func (r *SessionRenewer) ClearIfOAuth(ctx context.Context) error {
	if r == nil || !isOAuthSession(r.secrets) {
		return nil
	}
	return NewAuthService(r.account, r.secrets).Logout(ctx)
}

func isOAuthSession(secrets ports.SecretStore) bool {
	origin, err := secrets.Get(ports.KeyDashboardSessionOrigin)
	return err == nil && origin == sessionOriginOAuth
}

// errOAuthSessionNotRefreshable is returned by refreshTokens for an OAuth
// session when no renewer is wired, instead of calling a refresh the server
// will refuse.
var errOAuthSessionNotRefreshable = errors.New("this dashboard session came from `nylas oauth login` and cannot be refreshed; run it again")
