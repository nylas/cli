package ports

import (
	"context"

	"github.com/nylas/cli/internal/domain"
)

// PubSubClient defines the interface for Pub/Sub notification channels.
type PubSubClient interface {
	ListPubSubChannels(ctx context.Context) (*domain.PubSubChannelListResponse, error)
	GetPubSubChannel(ctx context.Context, channelID string) (*domain.PubSubChannel, error)
	CreatePubSubChannel(ctx context.Context, req *domain.CreatePubSubChannelRequest) (*domain.PubSubChannel, error)
	UpdatePubSubChannel(ctx context.Context, channelID string, req *domain.UpdatePubSubChannelRequest) (*domain.PubSubChannel, error)
	DeletePubSubChannel(ctx context.Context, channelID string) error
}
