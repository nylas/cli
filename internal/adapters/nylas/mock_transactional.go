package nylas

import (
	"context"

	"github.com/nylas/cli/internal/domain"
)

// SendTransactionalMessageFunc allows customization of SendTransactionalMessage behavior in tests.
var SendTransactionalMessageFunc func(ctx context.Context, domainName string, req *domain.SendMessageRequest) (*domain.Message, error)

// SendTransactionalMessage sends an email via the domain-based transactional endpoint.
func (m *MockClient) SendTransactionalMessage(ctx context.Context, domainName string, req *domain.SendMessageRequest) (*domain.Message, error) {
	if SendTransactionalMessageFunc != nil {
		return SendTransactionalMessageFunc(ctx, domainName, req)
	}
	return &domain.Message{
		ID:      "sent-transactional-message-id",
		Subject: req.Subject,
		To:      req.To,
		Body:    req.Body,
	}, nil
}
