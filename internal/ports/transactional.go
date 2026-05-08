package ports

import (
	"context"

	"github.com/nylas/cli/internal/domain"
)

// TransactionalClient defines the interface for domain-based transactional email operations.
// Normal Agent Account mailbox sends use per-grant message send so they can
// preserve mailbox behavior such as Sent-folder archiving.
type TransactionalClient interface {
	// SendTransactionalMessage sends an email via the domain-based transactional endpoint.
	// POST /v3/domains/{domain}/messages/send
	SendTransactionalMessage(ctx context.Context, domainName string, req *domain.SendMessageRequest) (*domain.Message, error)
}
