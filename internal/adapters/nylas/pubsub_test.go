package nylas_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/nylas/cli/internal/adapters/nylas"
	"github.com/nylas/cli/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHTTPClient_ListPubSubChannels(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/channels/pubsub", r.URL.Path)
		assert.Equal(t, http.MethodGet, r.Method)

		response := map[string]any{
			"data": []map[string]any{
				{
					"id":            "pubsub-1",
					"description":   "Message notifications",
					"trigger_types": []string{"message.created"},
					"topic":         "projects/demo/topics/messages",
					"status":        "active",
				},
			},
			"next_cursor": "cursor-123",
			"request_id":  "req-123",
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := nylas.NewHTTPClient()
	client.SetCredentials("client-id", "secret", "api-key")
	client.SetBaseURL(server.URL)

	resp, err := client.ListPubSubChannels(context.Background())

	require.NoError(t, err)
	require.NotNil(t, resp)
	require.Len(t, resp.Data, 1)
	assert.Equal(t, "pubsub-1", resp.Data[0].ID)
	assert.Equal(t, "projects/demo/topics/messages", resp.Data[0].Topic)
	assert.Equal(t, "cursor-123", resp.NextCursor)
	assert.Equal(t, "req-123", resp.RequestID)
}

func TestHTTPClient_GetPubSubChannel(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/channels/pubsub/pubsub-1", r.URL.Path)
		assert.Equal(t, http.MethodGet, r.Method)

		response := map[string]any{
			"data": map[string]any{
				"id":                           "pubsub-1",
				"description":                  "Message notifications",
				"trigger_types":                []string{"message.created", "message.updated"},
				"topic":                        "projects/demo/topics/messages",
				"notification_email_addresses": []string{"admin@example.com"},
				"status":                       "active",
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := nylas.NewHTTPClient()
	client.SetCredentials("client-id", "secret", "api-key")
	client.SetBaseURL(server.URL)

	channel, err := client.GetPubSubChannel(context.Background(), "pubsub-1")

	require.NoError(t, err)
	require.NotNil(t, channel)
	assert.Equal(t, "pubsub-1", channel.ID)
	assert.Equal(t, "active", channel.Status)
	assert.Len(t, channel.NotificationEmailAddresses, 1)
}

func TestHTTPClient_GetPubSubChannel_EmptyID(t *testing.T) {
	t.Parallel()

	client := nylas.NewHTTPClient()
	client.SetCredentials("client-id", "secret", "api-key")

	channel, err := client.GetPubSubChannel(context.Background(), "")

	require.Error(t, err)
	assert.Nil(t, channel)
	assert.Contains(t, err.Error(), "channel ID")
}

func TestHTTPClient_CreatePubSubChannel(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/channels/pubsub", r.URL.Path)
		assert.Equal(t, http.MethodPost, r.Method)

		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		assert.Equal(t, "projects/demo/topics/messages", body["topic"])

		response := map[string]any{
			"data": map[string]any{
				"id":            "pubsub-new",
				"description":   body["description"],
				"trigger_types": body["trigger_types"],
				"topic":         body["topic"],
				"status":        "active",
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := nylas.NewHTTPClient()
	client.SetCredentials("client-id", "secret", "api-key")
	client.SetBaseURL(server.URL)

	channel, err := client.CreatePubSubChannel(context.Background(), &domain.CreatePubSubChannelRequest{
		Description:  "Message notifications",
		TriggerTypes: []string{"message.created"},
		Topic:        "projects/demo/topics/messages",
	})

	require.NoError(t, err)
	require.NotNil(t, channel)
	assert.Equal(t, "pubsub-new", channel.ID)
	assert.Equal(t, "projects/demo/topics/messages", channel.Topic)
}

func TestHTTPClient_UpdatePubSubChannel(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/channels/pubsub/pubsub-1", r.URL.Path)
		assert.Equal(t, http.MethodPut, r.Method)

		response := map[string]any{
			"data": map[string]any{
				"id":            "pubsub-1",
				"description":   "Updated channel",
				"trigger_types": []string{"event.created"},
				"topic":         "projects/demo/topics/calendar",
				"status":        "inactive",
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := nylas.NewHTTPClient()
	client.SetCredentials("client-id", "secret", "api-key")
	client.SetBaseURL(server.URL)

	channel, err := client.UpdatePubSubChannel(context.Background(), "pubsub-1", &domain.UpdatePubSubChannelRequest{
		Description: "Updated channel",
		Status:      "inactive",
	})

	require.NoError(t, err)
	require.NotNil(t, channel)
	assert.Equal(t, "inactive", channel.Status)
	assert.Equal(t, "Updated channel", channel.Description)
}

func TestHTTPClient_UpdatePubSubChannel_EmptyID(t *testing.T) {
	t.Parallel()

	client := nylas.NewHTTPClient()
	client.SetCredentials("client-id", "secret", "api-key")

	channel, err := client.UpdatePubSubChannel(context.Background(), "", &domain.UpdatePubSubChannelRequest{
		Description: "Updated channel",
	})

	require.Error(t, err)
	assert.Nil(t, channel)
	assert.Contains(t, err.Error(), "channel ID")
}

func TestHTTPClient_DeletePubSubChannel(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/channels/pubsub/pubsub-1", r.URL.Path)
		assert.Equal(t, http.MethodDelete, r.Method)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client := nylas.NewHTTPClient()
	client.SetCredentials("client-id", "secret", "api-key")
	client.SetBaseURL(server.URL)

	err := client.DeletePubSubChannel(context.Background(), "pubsub-1")
	require.NoError(t, err)
}

func TestHTTPClient_DeletePubSubChannel_EmptyID(t *testing.T) {
	t.Parallel()

	client := nylas.NewHTTPClient()
	client.SetCredentials("client-id", "secret", "api-key")

	err := client.DeletePubSubChannel(context.Background(), "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "channel ID")
}

func TestHTTPClient_CreatePubSubChannel_NilRequest(t *testing.T) {
	t.Parallel()

	client := nylas.NewHTTPClient()
	client.SetCredentials("client-id", "secret", "api-key")

	channel, err := client.CreatePubSubChannel(context.Background(), nil)

	require.Error(t, err)
	assert.Nil(t, channel)
	assert.Contains(t, err.Error(), "request is required")
}

func TestHTTPClient_UpdatePubSubChannel_NilRequest(t *testing.T) {
	t.Parallel()

	client := nylas.NewHTTPClient()
	client.SetCredentials("client-id", "secret", "api-key")

	channel, err := client.UpdatePubSubChannel(context.Background(), "pubsub-1", nil)

	require.Error(t, err)
	assert.Nil(t, channel)
	assert.Contains(t, err.Error(), "request is required")
}

func TestDemoClient_GetPubSubChannel_NotFound(t *testing.T) {
	t.Parallel()

	client := nylas.NewDemoClient()

	channel, err := client.GetPubSubChannel(context.Background(), "missing-channel")

	require.Error(t, err)
	assert.Nil(t, channel)
	assert.ErrorIs(t, err, domain.ErrPubSubChannelNotFound)
}
