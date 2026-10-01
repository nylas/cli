package domain

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// DefaultOAuthClientID is the static public client the Nylas authorization
// server registers for the CLI. It is an identifier, not a secret: the client
// is public (no client secret) and proves possession with PKCE instead.
const DefaultOAuthClientID = "b3a94d82-fc7d-4a22-803e-e603ae0f735c"

// ErrOAuthInvalidClientID is returned for a client id override that is not a
// plausible OAuth client identifier.
var ErrOAuthInvalidClientID = errors.New("invalid OAuth client id")

// oauthClientIDPattern allow-lists what a client id override may contain. The
// production id is a UUID; a local server may issue something else, so this
// is deliberately broader than a UUID check while still refusing anything
// that would need escaping in a form body or a log line.
var oauthClientIDPattern = regexp.MustCompile(`^[A-Za-z0-9._\-]{1,128}$`)

// ValidateOAuthClientID reports whether id is an acceptable client id.
func ValidateOAuthClientID(id string) error {
	if !oauthClientIDPattern.MatchString(id) {
		return fmt.Errorf("%w: must be 1-128 characters of A-Z, a-z, 0-9, '.', '_' or '-'", ErrOAuthInvalidClientID)
	}
	return nil
}

// ErrOAuthRedirectURI is returned when the callback server would advertise a
// redirect URI the authorization server has not registered for the CLI.
var ErrOAuthRedirectURI = errors.New("unregistered OAuth redirect URI")

// ValidateOAuthLoopbackRedirectURI checks a redirect URI against the two the
// server registers for the static client: http://127.0.0.1/callback and
// http://localhost/callback, each with any port (RFC 8252 section 7.3).
//
// Checked before the browser opens so a mismatch fails here, with a reason,
// rather than on a server error page after the user has already signed in.
func ValidateOAuthLoopbackRedirectURI(raw string) error {
	parsed, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrOAuthRedirectURI, err)
	}
	if parsed.Scheme != "http" {
		return fmt.Errorf("%w: scheme must be http, got %q", ErrOAuthRedirectURI, parsed.Scheme)
	}
	host := parsed.Hostname()
	if host != "127.0.0.1" && host != "localhost" {
		return fmt.Errorf("%w: host must be 127.0.0.1 or localhost, got %q", ErrOAuthRedirectURI, host)
	}
	if parsed.Port() == "" {
		return fmt.Errorf("%w: a port is required", ErrOAuthRedirectURI)
	}
	if parsed.Path != "/callback" || parsed.RawQuery != "" || parsed.Fragment != "" || parsed.User != nil {
		return fmt.Errorf("%w: must be exactly /callback with no query, fragment or userinfo", ErrOAuthRedirectURI)
	}
	if port, err := strconv.Atoi(parsed.Port()); err != nil || port < 1 || port > 65535 {
		return fmt.Errorf("%w: invalid port %q", ErrOAuthRedirectURI, parsed.Port())
	}
	return nil
}

// OAuthTokenGrant is one entry of an access token's grants claim: a grant
// the token may act on, and the application that grant belongs to.
type OAuthTokenGrant struct {
	ID            string `json:"id"`
	ApplicationID string `json:"application_id"`
}

// OAuthAccessTokenClaims are the claims the Nylas authorization server puts
// in an access token.
//
// These are DECODED, NOT VERIFIED. The CLI holds no key to check the
// signature and has no reason to: the token came from its own keyring, and
// the resource server is what verifies it. Use them to describe a session
// and to route a request, never to decide whether something is allowed.
type OAuthAccessTokenClaims struct {
	Subject   string            `json:"sub"`
	Audience  []string          `json:"-"`
	Scope     string            `json:"scope"`
	Org       string            `json:"org"`
	ClientID  string            `json:"client_id"`
	Grants    []OAuthTokenGrant `json:"grants"`
	ExpiresAt time.Time         `json:"-"`
}

// Scopes returns the space-delimited scope claim as a list.
func (c *OAuthAccessTokenClaims) Scopes() []string {
	return strings.Fields(c.Scope)
}

// HasGrant reports whether the token lists grantID in its grants claim.
func (c *OAuthAccessTokenClaims) HasGrant(grantID string) bool {
	if grantID == "" {
		return false
	}
	for _, grant := range c.Grants {
		if grant.ID == grantID {
			return true
		}
	}
	return false
}

// ErrOAuthTokenNotJWT is returned when an access token cannot be decoded as a
// JWT. The message never carries the token.
var ErrOAuthTokenNotJWT = errors.New("access token is not a decodable JWT")

// DecodeOAuthAccessToken decodes the payload of a JWT access token WITHOUT
// verifying its signature. See OAuthAccessTokenClaims for what that permits.
func DecodeOAuthAccessToken(token string) (*OAuthAccessTokenClaims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 || parts[1] == "" {
		return nil, ErrOAuthTokenNotJWT
	}
	payload, err := base64.RawURLEncoding.DecodeString(strings.TrimRight(parts[1], "="))
	if err != nil {
		return nil, fmt.Errorf("%w: payload is not base64url", ErrOAuthTokenNotJWT)
	}

	var raw struct {
		OAuthAccessTokenClaims
		Audience json.RawMessage `json:"aud"`
		Expiry   json.Number     `json:"exp"`
	}
	decoder := json.NewDecoder(strings.NewReader(string(payload)))
	decoder.UseNumber()
	if err := decoder.Decode(&raw); err != nil {
		return nil, fmt.Errorf("%w: payload is not a JSON object", ErrOAuthTokenNotJWT)
	}

	claims := raw.OAuthAccessTokenClaims
	audience, err := decodeAudience(raw.Audience)
	if err != nil {
		return nil, err
	}
	claims.Audience = audience
	if raw.Expiry != "" {
		seconds, err := raw.Expiry.Int64()
		if err != nil {
			return nil, fmt.Errorf("%w: exp is not an integer", ErrOAuthTokenNotJWT)
		}
		claims.ExpiresAt = time.Unix(seconds, 0).UTC()
	}
	return &claims, nil
}

// decodeAudience accepts both forms RFC 7519 allows for aud: a single string
// or an array of strings.
func decodeAudience(raw json.RawMessage) ([]string, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return nil, nil
	}
	var single string
	if err := json.Unmarshal(raw, &single); err == nil {
		return []string{single}, nil
	}
	var list []string
	if err := json.Unmarshal(raw, &list); err != nil {
		return nil, fmt.Errorf("%w: aud is neither a string nor a list of strings", ErrOAuthTokenNotJWT)
	}
	return list, nil
}
