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

func sendMessageForGrant(
	ctx context.Context,
	client ports.NylasClient,
	grantID string,
	grant *domain.Grant,
	req *domain.SendMessageRequest,
) (*domain.Message, error) {
	if isManagedTransactionalGrant(grant) {
		emailDomain := common.ExtractDomain(grant.Email)
		if emailDomain == "" {
			return nil, common.NewUserError(
				"could not extract domain from grant email",
				"Ensure the grant has a valid email address",
			)
		}
		req.From = []domain.EmailParticipant{{Email: grant.Email}}
		return client.SendTransactionalMessage(ctx, emailDomain, req)
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
