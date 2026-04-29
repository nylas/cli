package email

import (
	"context"

	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/domain"
	"github.com/nylas/cli/internal/ports"
)

func getGrantForSend(ctx context.Context, client ports.NylasClient, grantID string) (*domain.Grant, error) {
	grant, err := client.GetGrant(ctx, grantID)
	if err != nil {
		return nil, common.WrapGetError("grant", err)
	}
	return grant, nil
}

// sendMessageForGrant sends via the per-grant /v3/grants/{id}/messages/send
// endpoint for every provider. For Nylas-managed grants the API requires an
// explicit From, so we populate it from the grant when the caller didn't.
//
// The per-grant endpoint is the only one that archives the message to the
// sender's Sent folder; the domain-based transactional endpoint is a relay
// (and injects a developer-account banner), so it is *not* used here.
func sendMessageForGrant(
	ctx context.Context,
	client ports.NylasClient,
	grantID string,
	grant *domain.Grant,
	req *domain.SendMessageRequest,
) (*domain.Message, error) {
	if isManagedTransactionalGrant(grant) && len(req.From) == 0 && grant.Email != "" {
		req.From = []domain.EmailParticipant{{Email: grant.Email}}
	}
	return client.SendMessage(ctx, grantID, req)
}

func shouldUseInteractiveSendMode(
	interactive bool,
	to []string,
	subject, body string,
	templateOpts hostedTemplateSendOptions,
) bool {
	if interactive {
		return true
	}
	if templateOpts.TemplateID != "" {
		return len(to) == 0 && !templateOpts.RenderOnly
	}
	return len(to) == 0 && subject == "" && body == ""
}
