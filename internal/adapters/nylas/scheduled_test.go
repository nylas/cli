package nylas_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/nylas/cli/internal/adapters/nylas"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHTTPClient_ListScheduledMessages(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/grants/grant-123/messages/schedules", r.URL.Path)
		assert.Equal(t, "GET", r.Method)

		response := map[string]interface{}{
			"data": []map[string]interface{}{
				{
					"schedule_id": "schedule-001",
					"status": map[string]string{
						"code":        "scheduled",
						"description": "schedule send set",
					},
					"close_time": 1700000000,
				},
				{
					"schedule_id": "schedule-002",
					"status": map[string]string{
						"code":        "pending",
						"description": "sending soon",
					},
					"close_time": 1700100000,
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode( // Test helper, encode error not actionable
			response)
	}))
	defer server.Close()

	client := nylas.NewHTTPClient()
	client.SetCredentials("client-id", "secret", "api-key")
	client.SetBaseURL(server.URL)

	ctx := context.Background()
	scheduled, err := client.ListScheduledMessages(ctx, "grant-123")

	require.NoError(t, err)
	assert.Len(t, scheduled, 2)
	assert.Equal(t, "schedule-001", scheduled[0].ScheduleID)
	assert.Equal(t, "scheduled", scheduled[0].Status)
	assert.Equal(t, int64(1700000000), scheduled[0].CloseTime)
	assert.Equal(t, "schedule-002", scheduled[1].ScheduleID)
	assert.Equal(t, "pending", scheduled[1].Status)
}

func TestHTTPClient_GetScheduledMessage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/grants/grant-123/messages/schedules/schedule-456", r.URL.Path)
		assert.Equal(t, "GET", r.Method)

		response := map[string]interface{}{
			"data": map[string]interface{}{
				"schedule_id": "schedule-456",
				"status": map[string]string{
					"code":        "scheduled",
					"description": "message scheduled",
				},
				"close_time": 1700200000,
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode( // Test helper, encode error not actionable
			response)
	}))
	defer server.Close()

	client := nylas.NewHTTPClient()
	client.SetCredentials("client-id", "secret", "api-key")
	client.SetBaseURL(server.URL)

	ctx := context.Background()
	scheduled, err := client.GetScheduledMessage(ctx, "grant-123", "schedule-456")

	require.NoError(t, err)
	assert.Equal(t, "schedule-456", scheduled.ScheduleID)
	assert.Equal(t, "scheduled", scheduled.Status)
	assert.Equal(t, int64(1700200000), scheduled.CloseTime)
}

func TestHTTPClient_CancelScheduledMessage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/grants/grant-123/messages/schedules/schedule-789", r.URL.Path)
		assert.Equal(t, "DELETE", r.Method)

		w.WriteHeader(http.StatusOK)
		response := map[string]interface{}{
			"data": map[string]interface{}{
				"message": "Scheduled message cancelled",
			},
		}
		_ = json.NewEncoder(w).Encode( // Test helper, encode error not actionable
			response)
	}))
	defer server.Close()

	client := nylas.NewHTTPClient()
	client.SetCredentials("client-id", "secret", "api-key")
	client.SetBaseURL(server.URL)

	ctx := context.Background()
	err := client.CancelScheduledMessage(ctx, "grant-123", "schedule-789")

	require.NoError(t, err)
}

func TestHTTPClient_CancelScheduledMessage_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode( // Test helper, encode error not actionable
			map[string]interface{}{
				"error": map[string]string{
					"message": "Schedule not found",
				},
			})
	}))
	defer server.Close()

	client := nylas.NewHTTPClient()
	client.SetCredentials("client-id", "secret", "api-key")
	client.SetBaseURL(server.URL)

	ctx := context.Background()
	err := client.CancelScheduledMessage(ctx, "grant-123", "nonexistent")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestMockClient_ScheduledMessages(t *testing.T) {
	ctx := context.Background()
	mock := nylas.NewMockClient()

	// Test ListScheduledMessages
	scheduled, err := mock.ListScheduledMessages(ctx, "grant-123")
	require.NoError(t, err)
	assert.Len(t, scheduled, 2)
	assert.Equal(t, "grant-123", mock.LastGrantID)
	assert.Equal(t, "schedule-1", scheduled[0].ScheduleID)

	// Test GetScheduledMessage
	msg, err := mock.GetScheduledMessage(ctx, "grant-456", "schedule-xyz")
	require.NoError(t, err)
	assert.Equal(t, "schedule-xyz", msg.ScheduleID)
	assert.Equal(t, "grant-456", mock.LastGrantID)

	// Test CancelScheduledMessage
	err = mock.CancelScheduledMessage(ctx, "grant-789", "schedule-abc")
	require.NoError(t, err)
	assert.Equal(t, "grant-789", mock.LastGrantID)
}

func TestDemoClient_ScheduledMessages(t *testing.T) {
	ctx := context.Background()
	demo := nylas.NewDemoClient()

	// Test ListScheduledMessages
	scheduled, err := demo.ListScheduledMessages(ctx, "demo-grant")
	require.NoError(t, err)
	assert.Len(t, scheduled, 2)
	assert.Equal(t, "schedule-001", scheduled[0].ScheduleID)
	assert.Equal(t, "scheduled", scheduled[0].Status)

	// Test GetScheduledMessage
	msg, err := demo.GetScheduledMessage(ctx, "demo-grant", "schedule-test")
	require.NoError(t, err)
	assert.Equal(t, "schedule-test", msg.ScheduleID)
	assert.Equal(t, "scheduled", msg.Status)

	// Test CancelScheduledMessage
	err = demo.CancelScheduledMessage(ctx, "demo-grant", "schedule-test")
	require.NoError(t, err)
}

func TestHTTPClient_ListScheduledMessages_Empty(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := map[string]interface{}{
			"data": []interface{}{},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode( // Test helper, encode error not actionable
			response)
	}))
	defer server.Close()

	client := nylas.NewHTTPClient()
	client.SetCredentials("client-id", "secret", "api-key")
	client.SetBaseURL(server.URL)

	ctx := context.Background()
	scheduled, err := client.ListScheduledMessages(ctx, "grant-123")

	require.NoError(t, err)
	assert.Empty(t, scheduled)
}
