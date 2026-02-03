package ports

import (
	"context"

	"github.com/nylas/cli/internal/domain"
)

// TransactionalClient defines the interface for domain-based transactional email operations.
// This is used for Nylas Inbox provider grants which use domain-based endpoints instead of grant-based.
type TransactionalClient interface {
	// SendTransactionalMessage sends an email via the domain-based transactional endpoint.
	// Used for Inbox provider grants: POST /v3/domains/{domain}/messages/send
	SendTransactionalMessage(ctx context.Context, domainName string, req *domain.SendMessageRequest) (*domain.Message, error)
}
