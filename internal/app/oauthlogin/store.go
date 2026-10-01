package oauthlogin

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/nylas/cli/internal/domain"
	"github.com/nylas/cli/internal/ports"
)

// legacyKeyOAuthClientID held a dynamically registered client id before the
// CLI moved to a static client. Nothing reads it; it is only cleared, so a
// keyring written by an older build does not keep a stale entry forever.
const legacyKeyOAuthClientID = "oauth_client_id"

// sessionKeys is every secret this package owns. Clearing the set is what
// logout means, so a new key must be added here or it outlives the session.
var sessionKeys = []string{
	ports.KeyOAuthIssuer,
	ports.KeyOAuthServerURL,
	ports.KeyOAuthResource,
	legacyKeyOAuthClientID,
	ports.KeyOAuthAccessToken,
	ports.KeyOAuthRefreshToken,
	// Not written any more (see saveTokens); listed so an older build's copy
	// is cleared.
	ports.KeyOAuthIDToken,
	ports.KeyOAuthExpiresAt,
	ports.KeyOAuthScope,
}

// Session is a stored authorization server login. ClientID is the client the
// service is configured with, not a stored value.
type Session struct {
	Issuer string
	// ServerURL is the authorization server URL the session came from. Empty
	// for a session stored before it was recorded; the issuer stands in.
	ServerURL string
	ClientID  string
	// Resource is the RFC 8707 resource indicator the session was logged in
	// for; every refresh repeats it so the audience does not change.
	Resource string
	Tokens   domain.OAuthTokens
}

func (s *Service) loadSession() (*Session, error) {
	accessToken, err := s.getSecret(ports.KeyOAuthAccessToken)
	if err != nil {
		return nil, err
	}
	if accessToken == "" {
		return nil, domain.ErrOAuthNotLoggedIn
	}

	session := &Session{
		ClientID: s.clientID,
		Tokens:   domain.OAuthTokens{AccessToken: accessToken, TokenType: "Bearer"},
	}
	for key, target := range map[string]*string{
		ports.KeyOAuthIssuer:       &session.Issuer,
		ports.KeyOAuthServerURL:    &session.ServerURL,
		ports.KeyOAuthResource:     &session.Resource,
		ports.KeyOAuthRefreshToken: &session.Tokens.RefreshToken,
		ports.KeyOAuthScope:        &session.Tokens.Scope,
	} {
		value, err := s.getSecret(key)
		if err != nil {
			return nil, err
		}
		*target = value
	}

	expiresAt, err := s.getSecret(ports.KeyOAuthExpiresAt)
	if err != nil {
		return nil, err
	}
	if expiresAt != "" {
		parsed, err := time.Parse(time.RFC3339, expiresAt)
		if err != nil {
			// A zero ExpiresAt reads as expired, so an unparseable value
			// triggers a refresh rather than failing the command.
			parsed = time.Time{}
		}
		session.Tokens.ExpiresAt = parsed
	}

	return session, nil
}

// saveTokens persists a token set, in the order that keeps a partial write
// safe to use for a refresh (same server, same session):
//
//   - The server and resource first, then the refresh token. The server has
//     already rotated it, so the old one is spent; losing the new one is what
//     signs the user out.
//   - The access token, then its expiry last. A write that stops in between
//     leaves an expiry that reads as past, so the next command refreshes with
//     the refresh token just stored instead of trusting a stale access token.
//
// A login against a different server is not safe against a partial write:
// the server and resource fields alone don't distinguish "new session, write
// interrupted" from "old session, still valid", so callers logging in must
// clear the previous session first (see Service.Login).
//
// The ID token is not stored. Nothing verifies or reads it, and it is the
// largest value in the set, which counts against the keychain's item limits.
func (s *Service) saveTokens(issuer, resource string, tokens *domain.OAuthTokens) error {
	expiresAt := ""
	if !tokens.ExpiresAt.IsZero() {
		expiresAt = tokens.ExpiresAt.UTC().Format(time.RFC3339)
	}

	ordered := []struct {
		key   string
		value string
	}{
		{ports.KeyOAuthIssuer, issuer},
		{ports.KeyOAuthServerURL, s.client.ServerURL()},
		{ports.KeyOAuthResource, resource},
		{ports.KeyOAuthScope, tokens.Scope},
		{legacyKeyOAuthClientID, ""},
		{ports.KeyOAuthIDToken, ""},
		{ports.KeyOAuthRefreshToken, tokens.RefreshToken},
		{ports.KeyOAuthAccessToken, tokens.AccessToken},
		{ports.KeyOAuthExpiresAt, expiresAt},
	}

	for _, entry := range ordered {
		if entry.value == "" {
			if err := s.deleteSecret(entry.key); err != nil {
				return err
			}
			continue
		}
		if err := s.secrets.Set(entry.key, entry.value); err != nil {
			return fmt.Errorf("failed to store %s: %w", entry.key, err)
		}
	}
	return nil
}

func (s *Service) clearSession() error {
	return ClearSession(s.secrets)
}

// ClearSessionLocked is ClearSession under the cross-process session lock, so
// a refresh in flight in another process (`nylas mcp serve`) cannot write a
// rotated session back after the clear.
func ClearSessionLocked(ctx context.Context, secrets ports.SecretStore, lock ports.CrossProcessLock) error {
	s := &Service{secrets: secrets, lock: lock}
	return s.withSessionLock(ctx, s.clearSession)
}

// ClearSession removes every stored OAuth session key without contacting the
// server. The caller must hold the session lock; see ClearSessionLocked.
func ClearSession(secrets ports.SecretStore) error {
	var errs []error
	for _, key := range sessionKeys {
		if err := secrets.Delete(key); err != nil && !errors.Is(err, domain.ErrSecretNotFound) {
			errs = append(errs, fmt.Errorf("failed to clear %s: %w", key, err))
		}
	}
	return errors.Join(errs...)
}

func (s *Service) getSecret(key string) (string, error) {
	value, err := s.secrets.Get(key)
	if err != nil {
		if errors.Is(err, domain.ErrSecretNotFound) {
			return "", nil
		}
		return "", fmt.Errorf("failed to read %s: %w", key, err)
	}
	return value, nil
}

func (s *Service) deleteSecret(key string) error {
	if err := s.secrets.Delete(key); err != nil && !errors.Is(err, domain.ErrSecretNotFound) {
		return fmt.Errorf("failed to clear %s: %w", key, err)
	}
	return nil
}
