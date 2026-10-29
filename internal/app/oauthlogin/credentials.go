package oauthlogin

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/nylas/cli/internal/domain"
	"github.com/nylas/cli/internal/ports"
)

// An MCP-only credential (`nylas oauth login --mcp-only`) is stored beside
// the default session, never in place of it. The default session is the
// dashboard-capable login every other command uses; an MCP-only credential
// may be narrowed by an IAM principal, and a narrowed token must never be the
// one the dashboard commands exchange for a full-role session.
//
// Each credential is filed under an id derived from what makes it a
// different authorization: the server, the audience, the organization, the
// user and the principal. Logging in again with all five the same replaces
// that credential (and revokes the one it replaces); anything else adds a
// new one, so one login can never overwrite or silently broaden another.

// ErrMCPCredentialNotFound is returned for a --credential id nothing is stored under.
var ErrMCPCredentialNotFound = errors.New("no stored MCP credential has that id")

// ErrMCPCredentialAmbiguous is returned when several MCP-only credentials are
// stored, no default session is, and no --credential picks one.
var ErrMCPCredentialAmbiguous = errors.New("several MCP credentials are stored")

// ErrPrincipalNeedsMCPOnly is returned when a dashboard-capable login comes
// back narrowed by a principal. Such a token is not stored as the default
// session, where the dashboard commands would try to exchange it.
var ErrPrincipalNeedsMCPOnly = errors.New("this sign-in chose an IAM principal, and a principal credential is MCP-only")

// MCPCredentialRecord describes one stored MCP-only credential. It carries no
// token: it is what `nylas oauth status` lists and what selection matches.
type MCPCredentialRecord struct {
	ID        string                      `json:"id"`
	ServerURL string                      `json:"server_url"`
	Audience  string                      `json:"audience"`
	Org       string                      `json:"org"`
	Subject   string                      `json:"sub"`
	Principal *domain.OAuthTokenPrincipal `json:"principal,omitempty"`
}

var credentialIDPattern = regexp.MustCompile(`^[0-9a-f]{12}$`)

// credentialID is a short, stable name for one authorization. It is part of
// keyring key names, so it is hex and nothing else.
func credentialID(serverURL, audience, org, subject string, principal *domain.OAuthTokenPrincipal) string {
	parts := []string{domain.NormalizeOAuthIssuer(serverURL), audience, org, subject, "", ""}
	if principal != nil {
		parts[4], parts[5] = principal.ID, principal.Region
	}
	sum := sha256.Sum256([]byte(strings.Join(parts, "\x00")))
	return hex.EncodeToString(sum[:])[:12]
}

// key is base for this service's credential: base itself for the default
// session, "oauth_mcp_<id>_<field>" for an MCP-only credential.
func (s *Service) key(base string) string {
	if s.credentialID == "" {
		return base
	}
	return "oauth_mcp_" + s.credentialID + "_" + strings.TrimPrefix(base, "oauth_")
}

// forCredential returns a service bound to one MCP-only credential's keys.
// It shares the client, the store and the session lock.
func (s *Service) forCredential(id string) *Service {
	bound := *s
	bound.credentialID = id
	return &bound
}

// CredentialID is the MCP-only credential this service is bound to, empty
// for the default session.
func (s *Service) CredentialID() string {
	return s.credentialID
}

// ListMCPCredentials returns the stored MCP-only credentials.
func (s *Service) ListMCPCredentials() ([]MCPCredentialRecord, error) {
	return s.readIndex()
}

// ForMCPCredential returns a service bound to the MCP-only credential id.
func (s *Service) ForMCPCredential(id string) (*Service, error) {
	records, err := s.readIndex()
	if err != nil {
		return nil, err
	}
	for _, record := range records {
		if record.ID == id {
			return s.forCredential(id), nil
		}
	}
	return nil, fmt.Errorf("%w: %q (run `nylas oauth status` to list them)", ErrMCPCredentialNotFound, id)
}

// SelectMCPCredential picks the credential `nylas mcp serve --auth oauth`
// uses. An id picks that MCP-only credential. Without one, the default
// session is used when there is one, as it always has been; otherwise the
// only MCP-only credential, and with several the caller has to choose.
func (s *Service) SelectMCPCredential(id string) (*Service, error) {
	if id != "" {
		return s.ForMCPCredential(id)
	}
	if _, err := s.loadSession(); !errors.Is(err, domain.ErrOAuthNotLoggedIn) {
		return s, err
	}
	records, err := s.readIndex()
	if err != nil {
		return nil, err
	}
	switch len(records) {
	case 0:
		return nil, domain.ErrOAuthNotLoggedIn
	case 1:
		return s.forCredential(records[0].ID), nil
	default:
		ids := make([]string, 0, len(records))
		for _, record := range records {
			ids = append(ids, record.ID)
		}
		return nil, fmt.Errorf("%w (%s): pick one with --credential", ErrMCPCredentialAmbiguous, strings.Join(ids, ", "))
	}
}

// mcpOnlyRecord checks what an MCP-only login returned and names the
// credential it becomes. The claims are decoded, not verified; they decide
// where the token is filed and sent, and the MCP server checks the rest.
func mcpOnlyRecord(serverURL, resource string, tokens *domain.OAuthTokens) (*MCPCredentialRecord, error) {
	claims, err := domain.DecodeOAuthAccessToken(tokens.AccessToken)
	if err != nil {
		return nil, fmt.Errorf("an MCP-only sign-in must return a JWT access token: %w", err)
	}
	if slices.Contains(claims.Scopes(), domain.OAuthScopeDashboardSession) {
		return nil, errors.New("the server granted dashboard.session to an MCP-only sign-in")
	}
	audience, err := domain.MCPEndpointFromAudience(claims.Audience)
	if err != nil {
		return nil, err
	}
	if audience != resource {
		return nil, fmt.Errorf("%w: the token is for %s, not %s", domain.ErrMCPResource, audience, resource)
	}
	if claims.Org == "" || claims.Subject == "" {
		return nil, errors.New("the MCP access token names no organization or user")
	}
	if claims.Principal != nil {
		if err := claims.Principal.Validate(); err != nil {
			return nil, err
		}
		// The server binds a principal to its own region's MCP server.
		if regional, err := domain.MCPResourceForRegion(claims.Principal.Region); err != nil || regional != audience {
			return nil, fmt.Errorf("%w: a %s principal on a token for %s", domain.ErrOAuthPrincipalClaim, claims.Principal.Region, audience)
		}
	}
	return &MCPCredentialRecord{
		ID:        credentialID(serverURL, audience, claims.Org, claims.Subject, claims.Principal),
		ServerURL: serverURL,
		Audience:  audience,
		Org:       claims.Org,
		Subject:   claims.Subject,
		Principal: claims.Principal,
	}, nil
}

// storeLogin keeps a login's tokens: in its own credential for an MCP-only
// login, and as the default session otherwise. A token it refuses to keep is
// revoked, so a refused login leaves no live token family on the server.
func (s *Service) storeLogin(ctx context.Context, issuer string, opts LoginOptions, tokens *domain.OAuthTokens) (*MCPCredentialRecord, error) {
	if opts.MCPOnly {
		record, err := mcpOnlyRecord(s.client.ServerURL(), opts.Resource, tokens)
		if err != nil {
			s.revokeTokens(ctx, tokens)
			return nil, err
		}
		return record, s.withSessionLock(ctx, func() error {
			return s.storeMCPCredential(ctx, issuer, opts.Resource, record, tokens)
		})
	}

	if claims, err := domain.DecodeOAuthAccessToken(tokens.AccessToken); err == nil && claims.Principal != nil {
		s.revokeTokens(ctx, tokens)
		return nil, fmt.Errorf("%w: run `nylas oauth login --mcp-only` to keep it", ErrPrincipalNeedsMCPOnly)
	}
	return nil, s.withSessionLock(ctx, func() error {
		s.revokeReplacedSession(ctx)
		// Clear whatever session is stored first: a login against a different
		// server must not leave that server's old tokens filed under the new
		// server's name, where checkIssuer would then accept them.
		if err := s.clearSession(); err != nil {
			return err
		}
		if err := s.saveTokens(issuer, opts.Resource, tokens); err != nil {
			return errors.Join(err, s.clearSession())
		}
		return nil
	})
}

// storeMCPCredential replaces the credential record names and indexes it.
// The caller holds the session lock.
func (s *Service) storeMCPCredential(ctx context.Context, issuer, resource string, record *MCPCredentialRecord, tokens *domain.OAuthTokens) error {
	bound := s.forCredential(record.ID)
	bound.revokeReplacedSession(ctx)
	if err := bound.clearSession(); err != nil {
		return err
	}
	if err := bound.saveTokens(issuer, resource, tokens); err != nil {
		return errors.Join(err, bound.clearSession())
	}
	records, err := s.readIndex()
	if err != nil {
		return errors.Join(err, bound.clearSession())
	}
	records = slices.DeleteFunc(records, func(r MCPCredentialRecord) bool { return r.ID == record.ID })
	if err := s.writeIndex(append(records, *record)); err != nil {
		return errors.Join(err, bound.clearSession())
	}
	return nil
}

// revokeTokens revokes a token set the CLI is not keeping. Best effort.
func (s *Service) revokeTokens(ctx context.Context, tokens *domain.OAuthTokens) {
	token := tokens.RefreshToken
	if token == "" {
		token = tokens.AccessToken
	}
	_ = s.client.Revoke(ctx, s.clientID, token)
}

// LogoutMCPCredential revokes and removes one MCP-only credential. The
// default session and the dashboard session are left alone.
func (s *Service) LogoutMCPCredential(ctx context.Context, id string) error {
	bound, err := s.ForMCPCredential(id)
	if err != nil {
		return err
	}
	return s.withSessionLock(ctx, func() error {
		logoutErr := bound.logoutLocked(ctx)
		records, err := s.readIndex()
		if err != nil {
			return errors.Join(logoutErr, err)
		}
		records = slices.DeleteFunc(records, func(r MCPCredentialRecord) bool { return r.ID == id })
		return errors.Join(logoutErr, s.writeIndex(records))
	})
}

// readIndex reads the MCP credential index. An entry that is not in the
// shape this package writes is dropped rather than turned into key names.
func (s *Service) readIndex() ([]MCPCredentialRecord, error) {
	raw, err := s.getSecret(ports.KeyOAuthMCPCredentials)
	if err != nil || raw == "" {
		return nil, err
	}
	var records []MCPCredentialRecord
	if err := json.Unmarshal([]byte(raw), &records); err != nil {
		return nil, fmt.Errorf("failed to read the MCP credential index: %w", err)
	}
	return slices.DeleteFunc(records, func(r MCPCredentialRecord) bool {
		return !credentialIDPattern.MatchString(r.ID)
	}), nil
}

func (s *Service) writeIndex(records []MCPCredentialRecord) error {
	if len(records) == 0 {
		return s.deleteSecret(ports.KeyOAuthMCPCredentials)
	}
	raw, err := json.Marshal(records)
	if err != nil {
		return err
	}
	if err := s.secrets.Set(ports.KeyOAuthMCPCredentials, string(raw)); err != nil {
		return fmt.Errorf("failed to store the MCP credential index: %w", err)
	}
	return nil
}
