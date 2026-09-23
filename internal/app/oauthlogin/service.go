// Package oauthlogin drives the OAuth 2.1 authorization code flow against
// the Nylas authorization server hosted by dashboard-account.
package oauthlogin

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/nylas/cli/internal/domain"
	"github.com/nylas/cli/internal/ports"
)

// Service performs authorization server logins and owns the stored session.
type Service struct {
	clientID string
	client   ports.OAuthAuthServerClient
	server   ports.OAuthServer
	browser  ports.Browser
	secrets  ports.SecretStore
	now      func() time.Time
}

// NewService creates a login service for the given public client id —
// domain.DefaultOAuthClientID unless a local or dev server needs another.
//
// The client is registered statically on the server, with the redirect URIs
// http://127.0.0.1/callback and http://localhost/callback and a free port
// (RFC 8252 section 7.3). The server callback must advertise one of those.
func NewService(
	clientID string,
	client ports.OAuthAuthServerClient,
	server ports.OAuthServer,
	browser ports.Browser,
	secrets ports.SecretStore,
) *Service {
	return &Service{
		clientID: clientID,
		client:   client,
		server:   server,
		browser:  browser,
		secrets:  secrets,
		now:      time.Now,
	}
}

// LoginResult summarises a completed login.
type LoginResult struct {
	Issuer     string
	ClientID   string
	Scope      string
	ExpiresAt  time.Time
	HasRefresh bool
}

// Login runs the browser authorization code flow and stores the tokens.
func (s *Service) Login(ctx context.Context, scopes []string) (*LoginResult, error) {
	if len(scopes) == 0 {
		scopes = domain.DefaultOAuthScopes()
	}

	metadata, err := s.client.Metadata(ctx)
	if err != nil {
		return nil, err
	}

	clientID := s.clientID
	if err := domain.ValidateOAuthClientID(clientID); err != nil {
		return nil, err
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

	tokens, err := s.client.ExchangeCode(ctx, domain.OAuthCodeExchange{
		ClientID:     clientID,
		Code:         callback.code,
		RedirectURI:  redirectURI,
		CodeVerifier: pkce.Verifier,
	})
	if err != nil {
		return nil, err
	}

	if err := s.saveTokens(metadata.Issuer, tokens); err != nil {
		return nil, err
	}

	return &LoginResult{
		Issuer:     metadata.Issuer,
		ClientID:   clientID,
		Scope:      tokens.Scope,
		ExpiresAt:  tokens.ExpiresAt,
		HasRefresh: tokens.RefreshToken != "",
	}, nil
}

// AccessToken returns a usable access token, refreshing it when it has expired.
func (s *Service) AccessToken(ctx context.Context) (string, error) {
	session, err := s.loadSession()
	if err != nil {
		return "", err
	}
	if !session.Tokens.IsExpired(s.now()) {
		return session.Tokens.AccessToken, nil
	}
	if session.Tokens.RefreshToken == "" {
		return "", fmt.Errorf("%w: run `nylas oauth login` again", domain.ErrOAuthNoRefreshToken)
	}

	tokens, err := s.client.Refresh(ctx, session.ClientID, session.Tokens.RefreshToken)
	if err != nil {
		return "", err
	}

	// The server rotates the refresh token on every use and burns the whole
	// family if an old one reappears. Persist exactly what came back — never
	// carry the previous refresh token forward to fill an empty field.
	if err := s.saveTokens(session.Issuer, tokens); err != nil {
		return "", err
	}
	return tokens.AccessToken, nil
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
	session, err := s.loadSession()
	if err != nil {
		if errors.Is(err, domain.ErrOAuthNotLoggedIn) {
			return s.clearSession()
		}
		return err
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
}
