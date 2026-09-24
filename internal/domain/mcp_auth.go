package domain

import (
	"errors"
	"fmt"
	"slices"
	"strings"
)

// The hosted Nylas MCP servers. Each is also the RFC 8707 resource indicator
// the authorization server accepts, and the audience of a token issued for
// it; the server answers any other resource with invalid_target.
const (
	MCPResourceUS = "https://mcp.us.nylas.com"
	MCPResourceEU = "https://mcp.eu.nylas.com"
)

// ErrMCPResource is returned for a resource or audience that is not one of
// the hosted Nylas MCP servers.
var ErrMCPResource = errors.New("not a Nylas MCP server")

// MCPResourceForRegion returns the MCP server for a configured region. An
// empty region is US, matching the CLI's region default.
func MCPResourceForRegion(region string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(region)) {
	case "", "us":
		return MCPResourceUS, nil
	case "eu":
		return MCPResourceEU, nil
	default:
		return "", fmt.Errorf("%w: unknown region %q (want us or eu)", ErrMCPResource, region)
	}
}

// ValidateMCPResource reports whether resource is a hosted MCP server.
func ValidateMCPResource(resource string) error {
	if resource != MCPResourceUS && resource != MCPResourceEU {
		return fmt.Errorf("%w: %q", ErrMCPResource, resource)
	}
	return nil
}

// MCPEndpointFromAudience picks the MCP server a token was issued for. The
// audience must name exactly one of the two hosted servers: sending a token
// to a host it does not name would either be refused or, worse, hand a
// credential to a server that was never meant to see it.
func MCPEndpointFromAudience(audience []string) (string, error) {
	var found []string
	for _, aud := range audience {
		if ValidateMCPResource(aud) == nil && !slices.Contains(found, aud) {
			found = append(found, aud)
		}
	}
	switch len(found) {
	case 1:
		return found[0], nil
	case 0:
		return "", fmt.Errorf("%w: token audience %v names no Nylas MCP server", ErrMCPResource, audience)
	default:
		return "", fmt.Errorf("%w: token audience %v names more than one Nylas MCP server", ErrMCPResource, audience)
	}
}

// Data scopes the authorization server defines. It only offers the ones a
// deployment has enabled, which discovery lists in scopes_supported.
const (
	OAuthScopeEmailRead     = "email.read"
	OAuthScopeEmailSend     = "email.send"
	OAuthScopeCalendarRead  = "calendar.read"
	OAuthScopeCalendarWrite = "calendar.write"
	OAuthScopeContactsRead  = "contacts.read"
	OAuthScopeNotetakerRead = "notetaker.read"
	OAuthScopeGrantsRead    = "grants.read"
	OAuthScopeGrantsWrite   = "grants.write"
)

// MCPOAuthScopes is what `nylas oauth login --for mcp` requests: the data
// scopes the hosted MCP tools use, plus offline_access so a long-running
// `nylas mcp serve` can refresh. grants.read, not grants.write — the tools
// look grants up and never create or delete one.
func MCPOAuthScopes() []string {
	return []string{
		OAuthScopeEmailRead,
		OAuthScopeEmailSend,
		OAuthScopeCalendarRead,
		OAuthScopeCalendarWrite,
		OAuthScopeContactsRead,
		OAuthScopeNotetakerRead,
		OAuthScopeGrantsRead,
		OAuthScopeOfflineAccess,
	}
}

// ErrMCPCredentialNotRenewable is returned by a credential source that has
// nothing to renew — an API key is what it is.
var ErrMCPCredentialNotRenewable = errors.New("credential cannot be renewed")

// MCPCredential is what the MCP proxy authenticates one request with.
type MCPCredential struct {
	// Token is sent as "Authorization: Bearer <Token>".
	Token string

	// Endpoint is the MCP server to send the request to. Empty means the
	// proxy's regional default, which only an API key uses.
	Endpoint string

	// GrantScoped restricts the X-Nylas-Grant-Id hint to Grants. An OAuth
	// token names the grants it may act on; an API key does not.
	GrantScoped bool
	Grants      []OAuthTokenGrant
}

// AllowsGrantHint reports whether grantID may be offered to the server as
// the default grant for this credential.
func (c *MCPCredential) AllowsGrantHint(grantID string) bool {
	if grantID == "" {
		return false
	}
	if !c.GrantScoped {
		return true
	}
	for _, grant := range c.Grants {
		if grant.ID == grantID {
			return true
		}
	}
	return false
}

// BearerChallenge is a parsed RFC 6750 WWW-Authenticate Bearer challenge.
type BearerChallenge struct {
	Error            string
	ErrorDescription string
	Scope            string
	ResourceMetadata string
}

// Scopes returns the challenge's space-delimited scope parameter as a list.
func (c *BearerChallenge) Scopes() []string {
	return strings.Fields(c.Scope)
}

// ParseBearerChallenge extracts the Bearer challenge from a WWW-Authenticate
// header value. It returns nil when the header carries no Bearer challenge.
func ParseBearerChallenge(header string) *BearerChallenge {
	rest, ok := cutScheme(header, "Bearer")
	if !ok {
		return nil
	}

	challenge := &BearerChallenge{}
	for _, param := range splitAuthParams(rest) {
		name, value, found := strings.Cut(param, "=")
		if !found {
			// A bare token after the params is the next challenge's scheme.
			break
		}
		name = strings.ToLower(strings.TrimSpace(name))
		value = unquoteAuthParam(strings.TrimSpace(value))
		switch name {
		case "error":
			challenge.Error = value
		case "error_description":
			challenge.ErrorDescription = value
		case "scope":
			challenge.Scope = value
		case "resource_metadata":
			challenge.ResourceMetadata = value
		}
	}
	return challenge
}

// cutScheme finds scheme as an auth-scheme token in header and returns what
// follows it.
func cutScheme(header, scheme string) (string, bool) {
	lower := strings.ToLower(header)
	want := strings.ToLower(scheme)
	for i := 0; i+len(want) <= len(lower); i++ {
		if lower[i:i+len(want)] != want {
			continue
		}
		atStart := i == 0 || lower[i-1] == ' ' || lower[i-1] == ','
		end := i + len(want)
		atEnd := end == len(lower) || lower[end] == ' '
		if atStart && atEnd {
			return header[end:], true
		}
	}
	return "", false
}

// splitAuthParams splits comma-separated auth-params, respecting quotes.
func splitAuthParams(s string) []string {
	var params []string
	var current strings.Builder
	inQuotes, escaped := false, false
	for _, r := range s {
		switch {
		case escaped:
			escaped = false
		case r == '\\' && inQuotes:
			escaped = true
		case r == '"':
			inQuotes = !inQuotes
		case r == ',' && !inQuotes:
			if p := strings.TrimSpace(current.String()); p != "" {
				params = append(params, p)
			}
			current.Reset()
			continue
		}
		current.WriteRune(r)
	}
	if p := strings.TrimSpace(current.String()); p != "" {
		params = append(params, p)
	}
	return params
}

func unquoteAuthParam(value string) string {
	if len(value) < 2 || value[0] != '"' || value[len(value)-1] != '"' {
		return value
	}
	inner := value[1 : len(value)-1]
	var out strings.Builder
	escaped := false
	for _, r := range inner {
		if !escaped && r == '\\' {
			escaped = true
			continue
		}
		escaped = false
		out.WriteRune(r)
	}
	return out.String()
}
