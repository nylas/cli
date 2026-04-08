package nylas

import (
	"context"

	"github.com/nylas/cli/internal/domain"
)

func (m *MockClient) ListPubSubChannels(ctx context.Context) (*domain.PubSubChannelListResponse, error) {
	m.ListPubSubChannelsCalled = true
	if m.ListPubSubChannelsFunc != nil {
		return m.ListPubSubChannelsFunc(ctx)
	}

	return &domain.PubSubChannelListResponse{
		Data: []domain.PubSubChannel{
			{
				ID:           "pubsub-1",
				Description:  "Message notifications",
				TriggerTypes: []string{domain.TriggerMessageCreated},
				Topic:        "projects/demo/topics/notifications",
				Status:       "active",
			},
		},
	}, nil
}

func (m *MockClient) GetPubSubChannel(ctx context.Context, channelID string) (*domain.PubSubChannel, error) {
	m.GetPubSubChannelCalled = true
	m.LastPubSubChannelID = channelID
	if m.GetPubSubChannelFunc != nil {
		return m.GetPubSubChannelFunc(ctx, channelID)
	}

	return &domain.PubSubChannel{
		ID:           channelID,
		Description:  "Message notifications",
		TriggerTypes: []string{domain.TriggerMessageCreated},
		Topic:        "projects/demo/topics/notifications",
		Status:       "active",
	}, nil
}

func (m *MockClient) CreatePubSubChannel(
	ctx context.Context,
	req *domain.CreatePubSubChannelRequest,
) (*domain.PubSubChannel, error) {
	m.CreatePubSubChannelCalled = true
	if m.CreatePubSubChannelFunc != nil {
		return m.CreatePubSubChannelFunc(ctx, req)
	}

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

func (m *MockClient) UpdatePubSubChannel(
	ctx context.Context,
	channelID string,
	req *domain.UpdatePubSubChannelRequest,
) (*domain.PubSubChannel, error) {
	m.UpdatePubSubChannelCalled = true
	m.LastPubSubChannelID = channelID
	if m.UpdatePubSubChannelFunc != nil {
		return m.UpdatePubSubChannelFunc(ctx, channelID, req)
	}

	channel := &domain.PubSubChannel{
		ID:     channelID,
		Status: "active",
	}
	if req.Description != "" {
		channel.Description = req.Description
	}
	if len(req.TriggerTypes) > 0 {
		channel.TriggerTypes = req.TriggerTypes
	}
	if req.Topic != "" {
		channel.Topic = req.Topic
	}
	if req.EncryptionKey != "" {
		channel.EncryptionKey = req.EncryptionKey
	}
	if len(req.NotificationEmailAddresses) > 0 {
		channel.NotificationEmailAddresses = req.NotificationEmailAddresses
	}
	if req.Status != "" {
		channel.Status = req.Status
	}
	return channel, nil
}

func (m *MockClient) DeletePubSubChannel(ctx context.Context, channelID string) error {
	m.DeletePubSubChannelCalled = true
	m.LastPubSubChannelID = channelID
	if m.DeletePubSubChannelFunc != nil {
		return m.DeletePubSubChannelFunc(ctx, channelID)
	}
	return nil
}
