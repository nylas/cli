package ports

import (
	"context"

	"github.com/nylas/cli/internal/domain"
)

// TransactionalClient defines the interface for domain-based transactional email operations.
// This is used for managed Nylas grants which use domain-based endpoints instead of grant-based.
type TransactionalClient interface {
	// SendTransactionalMessage sends an email via the domain-based transactional endpoint.
	// Used for managed Nylas grants: POST /v3/domains/{domain}/messages/send
	SendTransactionalMessage(ctx context.Context, domainName string, req *domain.SendMessageRequest) (*domain.Message, error)
}
