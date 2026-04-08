package nylas

import (
	"context"
	"fmt"
	"time"

	"github.com/nylas/cli/internal/domain"
)

func (d *DemoClient) ListPubSubChannels(ctx context.Context) (*domain.PubSubChannelListResponse, error) {
	return &domain.PubSubChannelListResponse{
		Data: []domain.PubSubChannel{
			{
				ID:           "pubsub-001",
				Description:  "High-volume message events",
				TriggerTypes: []string{domain.TriggerMessageCreated, domain.TriggerMessageUpdated},
				Topic:        "projects/demo/topics/messages",
				Status:       "active",
				CreatedAt:    time.Now().Add(-14 * 24 * time.Hour),
			},
			{
				ID:           "pubsub-002",
				Description:  "Calendar events",
				TriggerTypes: []string{domain.TriggerEventCreated, domain.TriggerEventUpdated},
				Topic:        "projects/demo/topics/calendar",
				Status:       "active",
				CreatedAt:    time.Now().Add(-7 * 24 * time.Hour),
			},
		},
	}, nil
}

func (d *DemoClient) GetPubSubChannel(ctx context.Context, channelID string) (*domain.PubSubChannel, error) {
	channels, _ := d.ListPubSubChannels(ctx)
	for _, channel := range channels.Data {
		if channel.ID == channelID {
			return &channel, nil
		}
	}
	return nil, fmt.Errorf("%w: %s", domain.ErrPubSubChannelNotFound, channelID)
}

func (d *DemoClient) CreatePubSubChannel(
	ctx context.Context,
	req *domain.CreatePubSubChannelRequest,
) (*domain.PubSubChannel, error) {
	return &domain.PubSubChannel{
		ID:                         "pubsub-new",
		Description:                req.Description,
		TriggerTypes:               req.TriggerTypes,
		Topic:                      req.Topic,
		EncryptionKey:              req.EncryptionKey,
		NotificationEmailAddresses: req.NotificationEmailAddresses,
		Status:                     "active",
	}, nil
}

func (d *DemoClient) UpdatePubSubChannel(
	ctx context.Context,
	channelID string,
	req *domain.UpdatePubSubChannelRequest,
) (*domain.PubSubChannel, error) {
	return &domain.PubSubChannel{
		ID:                         channelID,
		Description:                req.Description,
		TriggerTypes:               req.TriggerTypes,
		Topic:                      req.Topic,
		EncryptionKey:              req.EncryptionKey,
		NotificationEmailAddresses: req.NotificationEmailAddresses,
		Status:                     req.Status,
	}, nil
}

func (d *DemoClient) DeletePubSubChannel(ctx context.Context, channelID string) error {
	return nil
}
