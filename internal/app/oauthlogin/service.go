// Package oauthlogin drives the OAuth 2.1 authorization code flow against
// the Nylas authorization server hosted by dashboard-account.
package oauthlogin

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/nylas/cli/internal/domain"
	"github.com/nylas/cli/internal/ports"
)

// refreshTimeout bounds a refresh that no longer follows the caller's
// context (see refreshLocked).
const refreshTimeout = 30 * time.Second

// Service performs authorization server logins and owns the stored session.
type Service struct {
	clientID string
	client   ports.OAuthAuthServerClient
	server   ports.OAuthServer
	browser  ports.Browser
	secrets  ports.SecretStore
	lock     ports.CrossProcessLock
	now      func() time.Time
}

// NewService creates a login service for the given public client id —
// domain.DefaultOAuthClientID unless a local or dev server needs another.
//
// The client is registered statically on the server, with the redirect URIs
// http://127.0.0.1/callback and http://localhost/callback and a free port
// (RFC 8252 section 7.3). The server callback must advertise one of those.
//
// lock serialises every write to the stored session across processes. It is
// required: several `nylas mcp serve` processes share one keyring session,
// and two of them refreshing at once would replay a rotated refresh token,
// which makes the server revoke the whole family and signs the user out.
func NewService(
	clientID string,
	client ports.OAuthAuthServerClient,
	server ports.OAuthServer,
	browser ports.Browser,
	secrets ports.SecretStore,
	lock ports.CrossProcessLock,
) *Service {
	return &Service{
		clientID: clientID,
		client:   client,
		server:   server,
		browser:  browser,
		secrets:  secrets,
		lock:     lock,
		now:      time.Now,
	}
}

// withSessionLock runs fn while holding the cross-process session lock.
func (s *Service) withSessionLock(ctx context.Context, fn func() error) error {
	if s.lock == nil {
		// Fail closed: refreshing without the lock is exactly the race that
		// burns the refresh token family.
		return errors.New("oauth session lock is not configured")
	}
	unlock, err := s.lock.Lock(ctx)
	if err != nil {
		return fmt.Errorf("failed to acquire the OAuth session lock: %w", err)
	}
	fnErr := fn()
	if err := unlock(); err != nil && fnErr == nil {
		return fmt.Errorf("failed to release the OAuth session lock: %w", err)
	}
	return fnErr
}

// LoginResult summarises a completed login.
type LoginResult struct {
	Issuer     string
	ClientID   string
	Scope      string
	ExpiresAt  time.Time
	HasRefresh bool
	Resource   string
	// DroppedScopes are requested scopes the server does not offer, left out
	// of the request because LoginOptions.DropUnsupportedScopes was set.
	DroppedScopes []string
}

// LoginOptions shape an authorization request.
type LoginOptions struct {
	// Scopes to request; domain.DefaultOAuthScopes when empty.
	Scopes []string

	// Resource is the RFC 8707 resource indicator (a hosted MCP server), sent
	// on the authorization request, the code exchange, and every refresh of
	// the session. Empty requests no resource-server audience.
	Resource string

	// Region ("us" or "eu") places the organization of an account created
	// during sign-in. Empty leaves it to the server.
	Region string

	// DropUnsupportedScopes leaves out scopes the server's discovery document
	// does not list, instead of letting the whole request fail with
	// invalid_scope. The server only offers data scopes a deployment enables.
	DropUnsupportedScopes bool
}

// Login runs the browser authorization code flow and stores the tokens.
func (s *Service) Login(ctx context.Context, opts LoginOptions) (*LoginResult, error) {
	scopes := opts.Scopes
	if len(scopes) == 0 {
		scopes = domain.DefaultOAuthScopes()
	}
	if opts.Resource != "" {
		if err := domain.ValidateMCPResource(opts.Resource); err != nil {
			return nil, err
		}
	}

	metadata, err := s.client.Metadata(ctx)
	if err != nil {
		return nil, err
	}

	var dropped []string
	if opts.DropUnsupportedScopes {
		scopes, dropped = supportedScopes(scopes, metadata.ScopesSupported)
		if len(scopes) == 0 {
			return nil, fmt.Errorf("the authorization server offers none of the requested scopes %v", dropped)
		}
	}

	clientID := s.clientID
	if err := domain.ValidateOAuthClientID(clientID); err != nil {
		return nil, err
	}

	// The server accepts a first-party scope only from the built-in CLI
	// client and refuses the whole request otherwise, so a NYLAS_OAUTH_CLIENT_ID
	// override leaves it out. The dashboard exchange would refuse that
	// client's token anyway.
	if clientID != domain.DefaultOAuthClientID {
		var firstParty []string
		scopes, firstParty = withoutFirstPartyScopes(scopes)
		dropped = append(dropped, firstParty...)
		if len(scopes) == 0 {
			return nil, fmt.Errorf("client %s may not request any of the scopes %v", clientID, firstParty)
		}
	}

	if err := s.server.Start(); err != nil {
		return nil, err
	}
	defer func() { _ = s.server.Stop() }()

	state, err := domain.NewOAuthState()
	if err != nil {
		return nil, err
	}
	nonce, err := domain.NewOAuthState()
	if err != nil {
		return nil, err
	}
	pkce, err := domain.NewPKCE()
	if err != nil {
		return nil, err
	}

	// Fail closed before the browser opens: an unregistered spelling would
	// only surface as a server error page after the user has signed in.
	redirectURI := s.server.GetRedirectURI()
	if err := domain.ValidateOAuthLoopbackRedirectURI(redirectURI); err != nil {
		return nil, err
	}
	authURL, err := s.client.AuthorizationURL(ctx, domain.OAuthAuthorizationParams{
		ClientID:      clientID,
		RedirectURI:   redirectURI,
		Scopes:        scopes,
		State:         state,
		CodeChallenge: pkce.Challenge,
		Nonce:         nonce,
		Resource:      opts.Resource,
		Region:        opts.Region,
	})
	if err != nil {
		return nil, err
	}

	// Listen before the browser is opened: a fast redirect would otherwise
	// arrive before anything is waiting for it.
	type callbackResult struct {
		code string
		err  error
	}
	// Before the browser opens: a redirect that arrives first must be
	// checked against this login's state, not an unset one.
	s.server.SetExpectedState(state)
	callbackCh := make(chan callbackResult, 1)
	waitCtx, cancelWait := context.WithCancel(ctx)
	defer cancelWait()
	go func() {
		code, waitErr := s.server.WaitForCallback(waitCtx, state)
		callbackCh <- callbackResult{code: code, err: waitErr}
	}()

	if err := s.browser.Open(authURL); err != nil {
		return nil, fmt.Errorf("failed to open browser: %w", err)
	}

	callback := <-callbackCh
	if callback.err != nil {
		return nil, callback.err
	}

	// The consent screen may have used most of ctx. The code is single-use,
	// so the exchange and the write that keeps its answer get their own
	// deadline rather than whatever the user left of it.
	exchangeCtx, cancelExchange := context.WithTimeout(context.WithoutCancel(ctx), refreshTimeout)
	defer cancelExchange()
	tokens, err := s.client.ExchangeCode(exchangeCtx, domain.OAuthCodeExchange{
		ClientID:     clientID,
		Code:         callback.code,
		RedirectURI:  redirectURI,
		CodeVerifier: pkce.Verifier,
		Resource:     opts.Resource,
	})
	if err != nil {
		return nil, err
	}

	if err := s.withSessionLock(exchangeCtx, func() error {
		s.revokeReplacedSession(exchangeCtx)
		// Clear whatever session is stored first: a login against a different
		// server must not leave that server's old tokens filed under the new
		// server's name, where checkIssuer would then accept them.
		if err := s.clearSession(); err != nil {
			return err
		}
		if err := s.saveTokens(metadata.Issuer, opts.Resource, tokens); err != nil {
			return errors.Join(err, s.clearSession())
		}
		return nil
	}); err != nil {
		return nil, err
	}

	return &LoginResult{
		Issuer:        metadata.Issuer,
		ClientID:      clientID,
		Scope:         tokens.Scope,
		ExpiresAt:     tokens.ExpiresAt,
		HasRefresh:    tokens.RefreshToken != "",
		Resource:      opts.Resource,
		DroppedScopes: dropped,
	}, nil
}

// revokeReplacedSession revokes the refresh token of the session a login is
// about to replace, so each `oauth login` or `orgs switch` does not leave a
// live token family behind on the server. Best effort, and only for a session
// this server issued: sending another server's token here would hand it over.
// The caller holds the session lock.
func (s *Service) revokeReplacedSession(ctx context.Context) {
	old, err := s.loadSession()
	if err != nil || old.Tokens.RefreshToken == "" || s.checkIssuer(old) != nil {
		return
	}
	_ = s.client.Revoke(ctx, old.ClientID, old.Tokens.RefreshToken)
}

// Relogin runs the browser flow again for the stored session's scopes and
// resource, so a session logged in `--for mcp` stays one the MCP server
// accepts. It is how the organization of a session changes: the server binds
// a session to the organization picked on its consent screen, and that choice
// is only offered by signing in again.
func (s *Service) Relogin(ctx context.Context) (*LoginResult, error) {
	session, err := s.loadSessionForServer()
	if err != nil {
		return nil, err
	}
	// A session from before dashboard.session existed lacks it, and the
	// dashboard commands that relogin (switch-org) need it.
	return s.Login(ctx, LoginOptions{
		Scopes:   domain.WithDashboardSessionScope(strings.Fields(session.Tokens.Scope)),
		Resource: session.Resource,
	})
}

// withoutFirstPartyScopes splits scopes into the ones any client may request
// and the first-party ones only the built-in CLI client may.
func withoutFirstPartyScopes(scopes []string) (kept, firstParty []string) {
	for _, scope := range scopes {
		if domain.IsFirstPartyOAuthScope(scope) {
			firstParty = append(firstParty, scope)
		} else {
			kept = append(kept, scope)
		}
	}
	return kept, firstParty
}

// supportedScopes splits requested into what the server advertises and what
// it does not. A server that advertises nothing is taken at its word that it
// has no list, and everything is kept. A first-party scope is always kept: the
// server accepts it from the CLI and deliberately never advertises it.
func supportedScopes(requested, supported []string) (kept, dropped []string) {
	if len(supported) == 0 {
		return requested, nil
	}
	for _, scope := range requested {
		if domain.IsFirstPartyOAuthScope(scope) || slices.Contains(supported, scope) {
			kept = append(kept, scope)
		} else {
			dropped = append(dropped, scope)
		}
	}
	return kept, dropped
}

// AccessToken returns a usable access token, refreshing it when it has expired.
func (s *Service) AccessToken(ctx context.Context) (string, error) {
	return s.AccessTokenValidFor(ctx, 0)
}

// AccessTokenValidFor returns an access token with at least minValid left,
// refreshing it early when it has less. A caller whose own session ends with
// the token (the dashboard exchange) uses it so a renewal gets a new expiry.
func (s *Service) AccessTokenValidFor(ctx context.Context, minValid time.Duration) (string, error) {
	session, err := s.loadSessionForServer()
	if err != nil {
		return "", err
	}
	if !session.Tokens.ExpiresWithin(s.now(), minValid) {
		return session.Tokens.AccessToken, nil
	}
	return s.refreshLocked(ctx, "", minValid)
}

// RefreshAccessToken is for a caller whose request was just refused with
// 401: rejected is the access token the server would not accept. It goes
// through the same cross-process lock as AccessToken, so if another process
// has already replaced that token this returns the replacement instead of
// spending the refresh token a second time.
func (s *Service) RefreshAccessToken(ctx context.Context, rejected string) (string, error) {
	return s.refreshLocked(ctx, rejected, 0)
}

// refreshLocked is the only path that spends a refresh token.
//
// It holds the cross-process lock across read, refresh and write, and
// re-reads the session once the lock is held: whoever waited behind another
// refresher finds the rotated tokens already stored and uses them rather
// than replaying the refresh token that process just consumed. rejected is
// an access token the caller already knows is bad (empty when the token only
// expired), so a stored copy of it is not mistaken for a fresh one.
func (s *Service) refreshLocked(ctx context.Context, rejected string, minValid time.Duration) (string, error) {
	var accessToken string
	err := s.withSessionLock(ctx, func() error {
		session, err := s.loadSessionForServer()
		if err != nil {
			return err
		}
		stored := session.Tokens.AccessToken
		if !session.Tokens.ExpiresWithin(s.now(), minValid) && stored != rejected {
			accessToken = stored
			return nil
		}
		if session.Tokens.RefreshToken == "" {
			return fmt.Errorf("%w: run `nylas oauth login` again", domain.ErrOAuthNoRefreshToken)
		}

		// Once the request is sent the server rotates the refresh token, so
		// the caller giving up (Ctrl-C, an MCP request timing out) must not
		// abandon the response: the stored token would be spent and the new
		// one lost. The refresh runs to completion on its own deadline.
		refreshCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), refreshTimeout)
		defer cancel()
		tokens, err := s.client.Refresh(refreshCtx, session.ClientID, session.Tokens.RefreshToken, session.Resource)
		if err != nil {
			return err
		}

		// The server rotates the refresh token on every use and burns the
		// whole family if an old one reappears. Persist exactly what came
		// back — never carry the previous refresh token forward to fill an
		// empty field.
		if err := s.saveTokens(session.Issuer, session.Resource, tokens); err != nil {
			return s.dropSpentRefreshToken(session.Tokens.RefreshToken, err)
		}
		accessToken = tokens.AccessToken
		return nil
	})
	if err != nil {
		return "", err
	}
	return accessToken, nil
}

// dropSpentRefreshToken handles a refresh whose answer could not be stored.
// The server has already rotated spent, so if it is still what the store
// holds it is deleted: the next command then asks for a login instead of
// replaying it. A new refresh token that did get stored is kept, because the
// write order in saveTokens makes the rest of the session recover from it.
func (s *Service) dropSpentRefreshToken(spent string, saveErr error) error {
	stored, err := s.getSecret(ports.KeyOAuthRefreshToken)
	if err != nil {
		return errors.Join(saveErr, err)
	}
	if stored != spent {
		return saveErr
	}
	if err := s.deleteSecret(ports.KeyOAuthRefreshToken); err != nil {
		return errors.Join(saveErr, err)
	}
	return fmt.Errorf("%w; run `nylas oauth login` again", saveErr)
}

// Status returns the stored session without contacting the server.
func (s *Service) Status() (*Session, error) {
	return s.loadSession()
}

// UserInfo returns the OIDC claims for the current session.
func (s *Service) UserInfo(ctx context.Context) (*domain.OAuthUserInfo, error) {
	accessToken, err := s.AccessToken(ctx)
	if err != nil {
		return nil, err
	}
	return s.client.UserInfo(ctx, accessToken)
}

// Logout revokes the session's refresh token and clears local state.
//
// Revocation is best effort: an unreachable server must not leave tokens on
// disk, since the local copy is the thing the user asked to be rid of.
func (s *Service) Logout(ctx context.Context) error {
	// Under the lock, so a refresh in another process cannot write a rotated
	// session back after this one has cleared it.
	return s.withSessionLock(ctx, func() error {
		session, err := s.loadSession()
		if err != nil {
			if errors.Is(err, domain.ErrOAuthNotLoggedIn) {
				return s.clearSession()
			}
			return err
		}

		// A session from another server is cleared but not revoked: sending
		// its token here would hand it to a server that never issued it.
		if err := s.checkIssuer(session); err != nil {
			if clearErr := s.clearSession(); clearErr != nil {
				return clearErr
			}
			return fmt.Errorf("%w; the local session was cleared but not revoked", err)
		}

		token := session.Tokens.RefreshToken
		if token == "" {
			token = session.Tokens.AccessToken
		}
		revokeErr := s.client.Revoke(ctx, session.ClientID, token)

		if err := s.clearSession(); err != nil {
			return err
		}
		return revokeErr
	})
}

// loadSessionForServer loads the session only if it was issued by the server
// this service talks to. Every path that sends a stored token anywhere goes
// through it: the refresh token to the token endpoint, the access token to
// userinfo, to the dashboard exchange and to the MCP server.
func (s *Service) loadSessionForServer() (*Session, error) {
	session, err := s.loadSession()
	if err != nil {
		return nil, err
	}
	if err := s.checkIssuer(session); err != nil {
		return nil, err
	}
	return session, nil
}

func (s *Service) checkIssuer(session *Session) error {
	stored := session.ServerURL
	if stored == "" {
		stored = session.Issuer
	}
	stored = domain.NormalizeOAuthIssuer(stored)
	current := domain.NormalizeOAuthIssuer(s.client.ServerURL())
	if stored == current {
		return nil
	}
	return fmt.Errorf(
		"%w: it came from %q but the configured server is %q; run `nylas oauth login` against %q, or point NYLAS_DASHBOARD_ACCOUNT_URL back at %q",
		domain.ErrOAuthIssuerMismatch, stored, current, current, stored,
	)
}
