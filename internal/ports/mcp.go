package ports

import (
	"context"

	"github.com/nylas/cli/internal/domain"
)

// MCPCredentialSource supplies the credential for each request the MCP
// proxy forwards. It is asked before every request rather than once at
// start-up, because an OAuth access token lives fifteen minutes and a
// `nylas mcp serve` process lives as long as the assistant that started it.
type MCPCredentialSource interface {
	// Credential returns a credential valid for the next request,
	// refreshing an OAuth token that is at or near expiry.
	Credential(ctx context.Context) (*domain.MCPCredential, error)

	// Renew is called once after the server answered rejected with 401 and
	// a Bearer challenge. It returns domain.ErrMCPCredentialNotRenewable
	// when there is nothing to renew, as for an API key.
	Renew(ctx context.Context, rejected *domain.MCPCredential) (*domain.MCPCredential, error)
}
