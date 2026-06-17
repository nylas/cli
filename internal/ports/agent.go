package ports

import (
	"context"

	"github.com/nylas/cli/internal/domain"
)

// AgentClient defines the interface for Nylas-managed agent account operations.
type AgentClient interface {
	// ListAgentAccounts retrieves all agent accounts (provider=nylas).
	ListAgentAccounts(ctx context.Context) ([]domain.AgentAccount, error)

	// GetAgentAccount retrieves a specific agent account grant by ID.
	GetAgentAccount(ctx context.Context, grantID string) (*domain.AgentAccount, error)

	// CreateAgentAccount creates a new agent account with the given email address.
	// name is an optional top-level display name for the grant (1-256 chars when set).
	// appPassword is optional and enables IMAP/SMTP client access when set.
	// workspaceID assigns the account to an existing workspace when set.
	CreateAgentAccount(ctx context.Context, email, name, appPassword, workspaceID string) (*domain.AgentAccount, error)

	// UpdateAgentAccount updates mutable settings on an existing agent account.
	// email is required by the current grant update API for provider=nylas grants.
	// name sets the top-level display name; callers should pass the existing name
	// to preserve it, since the grant update replaces the full record.
	// appPassword rotates or adds IMAP/SMTP credentials when set.
	UpdateAgentAccount(ctx context.Context, grantID, email, name, appPassword string) (*domain.AgentAccount, error)

	// DeleteAgentAccount deletes an agent account by revoking its grant.
	DeleteAgentAccount(ctx context.Context, grantID string) error
}
