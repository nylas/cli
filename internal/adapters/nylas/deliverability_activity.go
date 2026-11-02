package nylas

import (
	"context"
	"net/url"

	"github.com/nylas/cli/internal/domain"
)

// ListSendEvents returns one page of GET /v3/activity/send.
func (c *HTTPClient) ListSendEvents(ctx context.Context, params *domain.SendEventListParams) (*domain.SendEventListResponse, error) {
	if params == nil {
		params = &domain.SendEventListParams{}
	}
	// event_type repeats as separate keys; the API rejects a comma-joined value.
	queryURL := activityWindowQuery(params.Start, params.End, params.Order, params.Limit, params.PageToken).
		AddSlice("event_type", params.EventTypes).
		Add("bounce_class", params.BounceClass).
		Add("sending_domain", params.SendingDomain).
		Add("from", params.From).
		Add("grant_id", params.GrantID).
		Add("recipient", params.Recipient).
		Add("recipient_domain", params.RecipientDomain).
		Add("message_id", params.MessageID).
		Add("thread_id", params.ThreadID).
		BuildURL(c.baseURL + "/v3/activity/send")

	var result domain.SendEventListResponse
	if err := c.doGet(ctx, queryURL, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// GetSendEvent returns one send event by its 64-hex ID.
func (c *HTTPClient) GetSendEvent(ctx context.Context, eventID string) (*domain.SendEvent, error) {
	if err := validateRequired("event ID", eventID); err != nil {
		return nil, err
	}
	var result struct {
		Data domain.SendEvent `json:"data"`
	}
	if err := c.doGet(ctx, c.baseURL+"/v3/activity/send/"+url.PathEscape(eventID), &result); err != nil {
		return nil, err
	}
	return &result.Data, nil
}

// GetSendMessageActivity returns every recipient's send timeline for one message.
func (c *HTTPClient) GetSendMessageActivity(ctx context.Context, messageID string) (*domain.SendMessageActivity, error) {
	if err := validateRequired("message ID", messageID); err != nil {
		return nil, err
	}
	var result struct {
		Data domain.SendMessageActivity `json:"data"`
	}
	if err := c.doGet(ctx, c.baseURL+"/v3/activity/send/messages/"+url.PathEscape(messageID), &result); err != nil {
		return nil, err
	}
	return &result.Data, nil
}

// ListTrackingEvents returns one page of GET /v3/activity/tracking.
func (c *HTTPClient) ListTrackingEvents(ctx context.Context, params *domain.TrackingEventListParams) (*domain.TrackingEventListResponse, error) {
	if params == nil {
		params = &domain.TrackingEventListParams{}
	}
	queryURL := activityWindowQuery(params.Start, params.End, params.Order, params.Limit, params.PageToken).
		AddSlice("event_type", params.EventTypes).
		Add("sending_domain", params.SendingDomain).
		Add("from", params.From).
		Add("grant_id", params.GrantID).
		Add("message_id", params.MessageID).
		Add("thread_id", params.ThreadID).
		Add("label", params.Label).
		BuildURL(c.baseURL + "/v3/activity/tracking")

	var result domain.TrackingEventListResponse
	if err := c.doGet(ctx, queryURL, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// GetTrackingEvent returns one tracking event by its 64-hex ID.
func (c *HTTPClient) GetTrackingEvent(ctx context.Context, eventID string) (*domain.TrackingEvent, error) {
	if err := validateRequired("event ID", eventID); err != nil {
		return nil, err
	}
	var result struct {
		Data domain.TrackingEvent `json:"data"`
	}
	if err := c.doGet(ctx, c.baseURL+"/v3/activity/tracking/"+url.PathEscape(eventID), &result); err != nil {
		return nil, err
	}
	return &result.Data, nil
}

// GetTrackingMessageActivity returns open and click totals and recent interactions for one message.
func (c *HTTPClient) GetTrackingMessageActivity(ctx context.Context, messageID string) (*domain.TrackingMessageActivity, error) {
	if err := validateRequired("message ID", messageID); err != nil {
		return nil, err
	}
	var result struct {
		Data domain.TrackingMessageActivity `json:"data"`
	}
	if err := c.doGet(ctx, c.baseURL+"/v3/activity/tracking/messages/"+url.PathEscape(messageID), &result); err != nil {
		return nil, err
	}
	return &result.Data, nil
}

// activityWindowQuery adds the window and paging params shared by both feeds; zero means server default.
func activityWindowQuery(start, end int64, order string, limit int, pageToken string) *QueryBuilder {
	return NewQueryBuilder().
		AddInt64("start", start).
		AddInt64("end", end).
		Add("order", order).
		AddInt("limit", limit).
		Add("page_token", pageToken)
}
