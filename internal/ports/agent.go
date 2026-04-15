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
	// appPassword is optional and enables IMAP/SMTP client access when set.
	// policyID is optional and attaches the created account to an existing policy.
	CreateAgentAccount(ctx context.Context, email, appPassword, policyID string) (*domain.AgentAccount, error)

	// DeleteAgentAccount deletes an agent account by revoking its grant.
	DeleteAgentAccount(ctx context.Context, grantID string) error
}
