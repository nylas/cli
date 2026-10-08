package domain

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"net"
	"net/url"
	"strings"
	"time"
)

// OAuth authorization server errors.
var (
	ErrOAuthNotLoggedIn    = errors.New("not logged in to the Nylas authorization server")
	ErrOAuthNoRefreshToken = errors.New("no refresh token stored")
	ErrOAuthMetadata       = errors.New("invalid authorization server metadata")
	// ErrOAuthIssuerMismatch: the stored session belongs to a different
	// authorization server than the one configured now, so its tokens must
	// not be sent there.
	ErrOAuthIssuerMismatch = errors.New("the stored OAuth session was issued by a different authorization server")
)

// Identity scopes. Data scopes (email.read, grants.read, ...) are in
// mcp_auth.go; the server only offers the ones a deployment has enabled.
const (
	OAuthScopeOpenID        = "openid"
	OAuthScopeEmail         = "email"
	OAuthScopeOfflineAccess = "offline_access"
)

// OAuthScopeDashboardSession is what the dashboard-session exchange
// (POST /auth/cli/oauth/exchange) requires. Its consent line tells the person
// the CLI will act as them in the dashboard with their full role.
//
// It is FIRST-PARTY ONLY: the server accepts it only from a built-in client
// such as the CLI, and never lists it in scopes_supported, so it is never
// dropped as "not offered" (see IsFirstPartyOAuthScope).
const OAuthScopeDashboardSession = "dashboard.session"

// IsFirstPartyOAuthScope reports a scope the server accepts from the CLI
// without advertising it in discovery.
func IsFirstPartyOAuthScope(scope string) bool {
	return scope == OAuthScopeDashboardSession
}

// DefaultOAuthScopes is what `nylas oauth login` requests: identity, a
// refresh token (offline_access is the only way the server issues one), and
// the dashboard session the dashboard commands are signed in with.
func DefaultOAuthScopes() []string {
	return []string{OAuthScopeOpenID, OAuthScopeEmail, OAuthScopeOfflineAccess, OAuthScopeDashboardSession}
}

// WithDashboardSessionScope returns scopes with dashboard.session added when
// it is missing, leaving the caller's slice untouched.
func WithDashboardSessionScope(scopes []string) []string {
	for _, scope := range scopes {
		if scope == OAuthScopeDashboardSession {
			return scopes
		}
	}
	return append(append([]string{}, scopes...), OAuthScopeDashboardSession)
}

// OAuthServerMetadata is the subset of the RFC 8414 authorization server
// metadata document the CLI uses.
type OAuthServerMetadata struct {
	Issuer                        string   `json:"issuer"`
	AuthorizationEndpoint         string   `json:"authorization_endpoint"`
	TokenEndpoint                 string   `json:"token_endpoint"`
	UserInfoEndpoint              string   `json:"userinfo_endpoint"`
	RevocationEndpoint            string   `json:"revocation_endpoint"`
	JWKSURI                       string   `json:"jwks_uri"`
	ScopesSupported               []string `json:"scopes_supported"`
	GrantTypesSupported           []string `json:"grant_types_supported"`
	ResponseTypesSupported        []string `json:"response_types_supported"`
	CodeChallengeMethodsSupported []string `json:"code_challenge_methods_supported"`
	TokenEndpointAuthMethods      []string `json:"token_endpoint_auth_methods_supported"`
}

// Validate reports whether the document carries what an authorization code
// flow needs. Discovery pointed at the wrong host usually returns valid JSON
// with none of these fields, and failing here names the problem.
func (m *OAuthServerMetadata) Validate() error {
	missing := []string{}
	for name, value := range map[string]string{
		"issuer":                 m.Issuer,
		"authorization_endpoint": m.AuthorizationEndpoint,
		"token_endpoint":         m.TokenEndpoint,
	} {
		if value == "" {
			missing = append(missing, name)
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("%w: missing %s", ErrOAuthMetadata, strings.Join(missing, ", "))
	}
	if len(m.CodeChallengeMethodsSupported) > 0 && !contains(m.CodeChallengeMethodsSupported, "S256") {
		return fmt.Errorf("%w: server does not support the S256 code challenge method", ErrOAuthMetadata)
	}
	return nil
}

// ValidateFor checks the document against the server it was fetched from.
// It is what stands between a tampered or misdirected discovery response and
// the code, PKCE verifier and refresh tokens the CLI then sends:
//   - the base URL is https, except on a loopback address;
//   - the issuer IS the base URL (RFC 8414 section 3.3). The one exception is
//     a loopback base URL, where a local dev server may advertise a tunnel as
//     its issuer; that issuer must still be https or loopback;
//   - every endpoint is on the issuer's own scheme and host.
func (m *OAuthServerMetadata) ValidateFor(baseURL string) error {
	if err := m.Validate(); err != nil {
		return err
	}
	base, err := url.Parse(baseURL)
	if err != nil || base.Host == "" {
		return fmt.Errorf("%w: %q is not an absolute URL", ErrOAuthMetadata, baseURL)
	}
	if !isHTTPSOrLoopback(base) {
		return fmt.Errorf("%w: %s must use https (http is only allowed on a loopback address)", ErrOAuthMetadata, baseURL)
	}
	issuer, err := url.Parse(NormalizeOAuthIssuer(m.Issuer))
	if err != nil || issuer.Host == "" {
		return fmt.Errorf("%w: issuer %q is not an absolute URL", ErrOAuthMetadata, m.Issuer)
	}
	if IsLoopbackHost(base.Hostname()) {
		if !isHTTPSOrLoopback(issuer) {
			return fmt.Errorf("%w: issuer %q must use https", ErrOAuthMetadata, m.Issuer)
		}
	} else if NormalizeOAuthIssuer(m.Issuer) != NormalizeOAuthIssuer(baseURL) {
		return fmt.Errorf("%w: issuer %q does not match %q", ErrOAuthMetadata, m.Issuer, baseURL)
	}
	for name, endpoint := range map[string]string{
		"authorization_endpoint": m.AuthorizationEndpoint,
		"token_endpoint":         m.TokenEndpoint,
		"userinfo_endpoint":      m.UserInfoEndpoint,
		"revocation_endpoint":    m.RevocationEndpoint,
		"jwks_uri":               m.JWKSURI,
	} {
		if endpoint == "" {
			continue
		}
		parsed, err := url.Parse(endpoint)
		if err != nil || parsed.Scheme != issuer.Scheme || !strings.EqualFold(parsed.Host, issuer.Host) {
			return fmt.Errorf("%w: %s %q is not on %s", ErrOAuthMetadata, name, endpoint, issuer.Scheme+"://"+issuer.Host)
		}
	}
	return nil
}

// NormalizeOAuthIssuer makes two spellings of one issuer compare equal: a
// trailing slash is not a different server.
func NormalizeOAuthIssuer(issuer string) string {
	return strings.TrimRight(strings.TrimSpace(issuer), "/")
}

// isHTTPSOrLoopback reports whether u is https, or plain http to this machine.
func isHTTPSOrLoopback(u *url.URL) bool {
	return u.Scheme == "https" || (u.Scheme == "http" && IsLoopbackHost(u.Hostname()))
}

// IsLoopbackHost reports whether host is localhost or a loopback IP.
func IsLoopbackHost(host string) bool {
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

// OAuthError is an RFC 6749 section 5.2 error response.
type OAuthError struct {
	Code        string `json:"error"`
	Description string `json:"error_description"`
	StatusCode  int    `json:"-"`
}

func (e *OAuthError) Error() string {
	if e.Description == "" {
		return fmt.Sprintf("oauth error %q (HTTP %d)", e.Code, e.StatusCode)
	}
	return fmt.Sprintf("oauth error %q: %s (HTTP %d)", e.Code, e.Description, e.StatusCode)
}

// OAuthAuthorizationParams are the query parameters of an authorization request.
type OAuthAuthorizationParams struct {
	ClientID      string
	RedirectURI   string
	Scopes        []string
	State         string
	CodeChallenge string
	Nonce         string
	// Resource is the RFC 8707 resource indicator, the server the token is
	// for. Empty requests a token with no resource-server audience.
	Resource string
	// Region ("us" or "eu") is where the organization of an account created
	// during this sign-in is placed. A hint only: it never changes an existing
	// account. Empty leaves it to the server.
	Region string
}

// OAuthCodeExchange carries an authorization code back to the token endpoint.
// ClientSecret stays empty for a public client, which is what the CLI is.
type OAuthCodeExchange struct {
	ClientID     string
	ClientSecret string
	Code         string
	RedirectURI  string
	CodeVerifier string
	// Resource repeats the authorization request's resource indicator, as
	// RFC 8707 requires of the token request.
	Resource string
}

// OAuthTokens is a token endpoint response.
type OAuthTokens struct {
	AccessToken  string    `json:"access_token"`
	TokenType    string    `json:"token_type"`
	ExpiresIn    int       `json:"expires_in"`
	RefreshToken string    `json:"refresh_token,omitempty"`
	IDToken      string    `json:"id_token,omitempty"`
	Scope        string    `json:"scope,omitempty"`
	ExpiresAt    time.Time `json:"-"`
}

// oauthExpiryLeeway refreshes slightly early so a token cannot expire in
// flight between the check and the request that uses it.
const oauthExpiryLeeway = 30 * time.Second

// IsExpired reports whether the access token is expired at now. A token set
// with no known expiry is treated as expired so the caller refreshes rather
// than sending a credential the server will reject.
func (t *OAuthTokens) IsExpired(now time.Time) bool {
	return t.ExpiresWithin(now, 0)
}

// ExpiresWithin reports whether the access token has less than minValid (and
// never less than the refresh leeway) left at now.
func (t *OAuthTokens) ExpiresWithin(now time.Time, minValid time.Duration) bool {
	if t.ExpiresAt.IsZero() {
		return true
	}
	return !now.Add(max(minValid, oauthExpiryLeeway)).Before(t.ExpiresAt)
}

// OAuthUserInfo holds the OIDC claims returned by the userinfo endpoint.
type OAuthUserInfo struct {
	Subject       string `json:"sub"`
	Email         string `json:"email,omitempty"`
	EmailVerified bool   `json:"email_verified,omitempty"`
	Org           string `json:"org,omitempty"`
}

// PKCE is an RFC 7636 verifier and its S256 challenge.
type PKCE struct {
	Verifier  string
	Challenge string
}

// NewPKCE generates an RFC 7636 S256 pair.
//
// Deliberately not the same as auth.generatePKCEPair, which computes
// base64std(hex(sha256(v))) for Nylas hosted auth. The authorization server
// enforces /^[A-Za-z0-9\-_]{43}$/ on the challenge, so only the RFC form passes.
func NewPKCE() (*PKCE, error) {
	verifier, err := randomURLSafe(32)
	if err != nil {
		return nil, fmt.Errorf("failed to generate PKCE verifier: %w", err)
	}
	sum := sha256.Sum256([]byte(verifier))
	return &PKCE{
		Verifier:  verifier,
		Challenge: base64.RawURLEncoding.EncodeToString(sum[:]),
	}, nil
}

// NewOAuthState generates an opaque CSRF state value for an authorization request.
func NewOAuthState() (string, error) {
	state, err := randomURLSafe(32)
	if err != nil {
		return "", fmt.Errorf("failed to generate OAuth state: %w", err)
	}
	return state, nil
}

func randomURLSafe(size int) (string, error) {
	buf := make([]byte, size)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}
