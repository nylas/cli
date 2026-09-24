package mcp

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/nylas/cli/internal/domain"
)

// OAuthLoginCommand is what a user runs to get a session the hosted MCP
// server accepts. Every OAuth failure the proxy reports names it.
const OAuthLoginCommand = "nylas oauth login --for mcp"

// maxErrorBody bounds how much of an error response is read and echoed.
const maxErrorBody = 4 << 10

// apiKeyCredentials is the credential source behind NewProxy: the same API
// key for every request, and nothing to renew.
type apiKeyCredentials struct {
	apiKey string
}

func (a apiKeyCredentials) Credential(context.Context) (*domain.MCPCredential, error) {
	return &domain.MCPCredential{Token: a.apiKey}, nil
}

func (apiKeyCredentials) Renew(context.Context, *domain.MCPCredential) (*domain.MCPCredential, error) {
	return nil, domain.ErrMCPCredentialNotRenewable
}

// credential asks the source for this request's credential.
func (p *Proxy) credential(ctx context.Context) (*domain.MCPCredential, error) {
	if p.creds == nil {
		return nil, errors.New("MCP proxy has no credential source")
	}
	cred, err := p.creds.Credential(ctx)
	if err != nil {
		return nil, p.loginError("no usable OAuth session", err)
	}
	if cred == nil || cred.Token == "" {
		return nil, p.loginError("no usable OAuth session", errors.New("credential source returned no token"))
	}
	return cred, nil
}

// renew asks the source for a replacement after a 401.
func (p *Proxy) renew(ctx context.Context, rejected *domain.MCPCredential) (*domain.MCPCredential, error) {
	next, err := p.creds.Renew(ctx, rejected)
	if err != nil {
		if errors.Is(err, domain.ErrMCPCredentialNotRenewable) {
			return nil, err
		}
		return nil, p.loginError("the Nylas MCP server rejected the OAuth token and it could not be refreshed", err)
	}
	if next == nil || next.Token == "" {
		return nil, p.loginError("the Nylas MCP server rejected the OAuth token", errors.New("refresh returned no token"))
	}
	return next, nil
}

// loginError says what went wrong and what to run. An API key proxy has no
// login command to offer, so its errors pass through unchanged.
func (p *Proxy) loginError(what string, cause error) error {
	if !p.oauth {
		return cause
	}
	return fmt.Errorf("%s: %w. Run `%s` to sign in again", what, cause, OAuthLoginCommand)
}

// statusError explains a non-200 answer. The two OAuth refusals get a
// sentence the user can act on; everything else keeps the status and body.
func (p *Proxy) statusError(resp *http.Response, body []byte, cred *domain.MCPCredential) error {
	challenge := domain.ParseBearerChallenge(resp.Header.Get("WWW-Authenticate"))

	if resp.StatusCode == http.StatusForbidden && challenge != nil && challenge.Error == "insufficient_scope" {
		missing := "a scope"
		if scopes := challenge.Scopes(); len(scopes) > 0 {
			missing = "scope " + strings.Join(scopes, ", ")
		}
		return fmt.Errorf("the Nylas MCP server requires %s, which this login was not granted. Run `%s` to sign in with the scopes the MCP tools use",
			missing, OAuthLoginCommand)
	}

	if resp.StatusCode == http.StatusUnauthorized && p.oauth && cred != nil {
		return fmt.Errorf("the Nylas MCP server rejected the OAuth token (HTTP 401). Run `%s` to sign in again",
			OAuthLoginCommand)
	}

	return fmt.Errorf("server returned %d: %s", resp.StatusCode, string(body))
}

func drainAndClose(resp *http.Response) {
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, maxErrorBody))
	_ = resp.Body.Close()
}
