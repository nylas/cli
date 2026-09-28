package oauthlogin

import (
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
// safe to use:
//
//   - The server and resource first. A login against another server that
//     fails after them leaves the old tokens under the new server's name, and
//     loadSessionForServer refuses them rather than sending them anywhere.
//   - The refresh token next. The server has already rotated it, so the old
//     one is spent; losing the new one is what signs the user out.
//   - The access token, then its expiry last. A write that stops in between
//     leaves an expiry that reads as past, so the next command refreshes with
//     the refresh token just stored instead of trusting a stale access token.
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

// ClearSession removes every stored OAuth session key without contacting the
// server. `nylas oauth logout` also revokes; this is for a local reset.
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
