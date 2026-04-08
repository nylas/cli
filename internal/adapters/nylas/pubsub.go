package nylas

import (
	"context"
	"fmt"
	"net/url"

	"github.com/nylas/cli/internal/domain"
	"github.com/nylas/cli/internal/util"
)

type pubSubChannelResponse struct {
	ID                         string   `json:"id"`
	Description                string   `json:"description"`
	TriggerTypes               []string `json:"trigger_types"`
	Topic                      string   `json:"topic"`
	EncryptionKey              string   `json:"encryption_key"`
	Status                     string   `json:"status"`
	NotificationEmailAddresses []string `json:"notification_email_addresses"`
	CreatedAt                  int64    `json:"created_at"`
	UpdatedAt                  int64    `json:"updated_at"`
	Object                     string   `json:"object"`
}

func (c *HTTPClient) ListPubSubChannels(ctx context.Context) (*domain.PubSubChannelListResponse, error) {
	queryURL := fmt.Sprintf("%s/v3/channels/pubsub", c.baseURL)

	var result struct {
		Data       []pubSubChannelResponse `json:"data"`
		NextCursor string                  `json:"next_cursor,omitempty"`
		RequestID  string                  `json:"request_id,omitempty"`
	}
	if err := c.doGet(ctx, queryURL, &result); err != nil {
		return nil, err
	}

	return &domain.PubSubChannelListResponse{
		Data:       util.Map(result.Data, convertPubSubChannel),
		NextCursor: result.NextCursor,
		RequestID:  result.RequestID,
	}, nil
}

func (c *HTTPClient) GetPubSubChannel(ctx context.Context, channelID string) (*domain.PubSubChannel, error) {
	if err := validateRequired("channel ID", channelID); err != nil {
		return nil, err
	}

	queryURL := fmt.Sprintf("%s/v3/channels/pubsub/%s", c.baseURL, url.PathEscape(channelID))

	var result struct {
		Data pubSubChannelResponse `json:"data"`
	}
	if err := c.doGetWithNotFound(ctx, queryURL, &result, domain.ErrPubSubChannelNotFound); err != nil {
		return nil, err
	}

	channel := convertPubSubChannel(result.Data)
	return &channel, nil
}

func (c *HTTPClient) CreatePubSubChannel(ctx context.Context, req *domain.CreatePubSubChannelRequest) (*domain.PubSubChannel, error) {
	if req == nil {
		return nil, fmt.Errorf("create pub/sub channel request is required")
	}

	queryURL := fmt.Sprintf("%s/v3/channels/pubsub", c.baseURL)

	resp, err := c.doJSONRequest(ctx, "POST", queryURL, req)
	if err != nil {
		return nil, err
	}

	var result struct {
		Data pubSubChannelResponse `json:"data"`
	}
	if err := c.decodeJSONResponse(resp, &result); err != nil {
		return nil, err
	}

	channel := convertPubSubChannel(result.Data)
	return &channel, nil
}

func (c *HTTPClient) UpdatePubSubChannel(
	ctx context.Context,
	channelID string,
	req *domain.UpdatePubSubChannelRequest,
) (*domain.PubSubChannel, error) {
	if err := validateRequired("channel ID", channelID); err != nil {
		return nil, err
	}
	if req == nil {
		return nil, fmt.Errorf("update pub/sub channel request is required")
	}

	queryURL := fmt.Sprintf("%s/v3/channels/pubsub/%s", c.baseURL, url.PathEscape(channelID))

	resp, err := c.doJSONRequest(ctx, "PUT", queryURL, req)
	if err != nil {
		return nil, err
	}

	var result struct {
		Data pubSubChannelResponse `json:"data"`
	}
	if err := c.decodeJSONResponse(resp, &result); err != nil {
		return nil, err
	}

	channel := convertPubSubChannel(result.Data)
	return &channel, nil
}

func (c *HTTPClient) DeletePubSubChannel(ctx context.Context, channelID string) error {
	if err := validateRequired("channel ID", channelID); err != nil {
		return err
	}

	queryURL := fmt.Sprintf("%s/v3/channels/pubsub/%s", c.baseURL, url.PathEscape(channelID))
	return c.doDelete(ctx, queryURL)
}

func convertPubSubChannel(channel pubSubChannelResponse) domain.PubSubChannel {
	return domain.PubSubChannel{
		ID:                         channel.ID,
		Description:                channel.Description,
		TriggerTypes:               channel.TriggerTypes,
		Topic:                      channel.Topic,
		EncryptionKey:              channel.EncryptionKey,
		Status:                     channel.Status,
		NotificationEmailAddresses: channel.NotificationEmailAddresses,
		CreatedAt:                  unixToTime(channel.CreatedAt),
		UpdatedAt:                  unixToTime(channel.UpdatedAt),
		Object:                     channel.Object,
	}
}
