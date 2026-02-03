package nylas

import (
	"context"

	"github.com/nylas/cli/internal/domain"
)

// SendTransactionalMessage returns a mock sent message for demo mode.
func (d *DemoClient) SendTransactionalMessage(ctx context.Context, domainName string, req *domain.SendMessageRequest) (*domain.Message, error) {
	return &domain.Message{
		ID:      "demo-transactional-message-id",
		Subject: req.Subject,
		To:      req.To,
		Body:    req.Body,
	}, nil
}
