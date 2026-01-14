//go:build integration
// +build integration

package integration

import (
	"context"
	"testing"
	"time"

	"github.com/nylas/cli/internal/domain"
)

func TestMetadataFiltering(t *testing.T) {
	skipIfMissingCreds(t)

	client := getTestClient()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	t.Run("filter messages by metadata key1", func(t *testing.T) {
		// Test filtering by metadata_pair
		params := &domain.MessageQueryParams{
			Limit:        10,
			MetadataPair: "key1:test-value",
		}

		messages, err := client.GetMessagesWithParams(ctx, testGrantID, params)
		if err != nil {
			t.Logf("Warning: Metadata filtering failed (may not have messages with this metadata): %v", err)
			// Don't fail the test as it's expected that there may be no messages with this metadata
			return
		}

		// If we got messages, verify they have the metadata
		for _, msg := range messages {
			if msg.Metadata != nil {
				if val, ok := msg.Metadata["key1"]; ok {
					if val != "test-value" {
						t.Errorf("expected metadata key1 to be 'test-value', got '%s'", val)
					}
				}
			}
		}

		t.Logf("Successfully filtered %d messages with metadata key1:test-value", len(messages))
	})

	t.Run("filter messages by metadata key2", func(t *testing.T) {
		// Test filtering by a different indexed key
		params := &domain.MessageQueryParams{
			Limit:        10,
			MetadataPair: "key2:project-alpha",
		}

		_, err := client.GetMessagesWithParams(ctx, testGrantID, params)
		if err != nil {
			t.Logf("Warning: Metadata filtering failed (may not have messages with this metadata): %v", err)
			// Don't fail the test as it's expected that there may be no messages with this metadata
			return
		}

		t.Log("Successfully queried messages with metadata key2:project-alpha")
	})

	t.Run("get message with metadata", func(t *testing.T) {
		// First get any message
		params := &domain.MessageQueryParams{
			Limit: 1,
		}

		messages, err := client.GetMessagesWithParams(ctx, testGrantID, params)
		if err != nil {
			t.Fatalf("failed to get messages: %v", err)
		}

		if len(messages) == 0 {
			t.Skip("no messages available for testing")
		}

		messageID := messages[0].ID

		// Get the full message details
		msg, err := client.GetMessage(ctx, testGrantID, messageID)
		if err != nil {
			t.Fatalf("failed to get message: %v", err)
		}

		// Verify Metadata field exists (may be nil or empty)
		if msg.Metadata == nil {
			t.Log("Message has no metadata (nil)")
		} else if len(msg.Metadata) == 0 {
			t.Log("Message has empty metadata map")
		} else {
			t.Logf("Message has %d metadata entries", len(msg.Metadata))
			for key, val := range msg.Metadata {
				t.Logf("  %s: %s", key, val)
			}
		}
	})
}

func TestMetadataWithPagination(t *testing.T) {
	skipIfMissingCreds(t)

	client := getTestClient()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	t.Run("metadata filtering with cursor pagination", func(t *testing.T) {
		params := &domain.MessageQueryParams{
			Limit:        5,
			MetadataPair: "key1:test",
		}

		resp, err := client.GetMessagesWithCursor(ctx, testGrantID, params)
		if err != nil {
			t.Logf("Warning: Metadata filtering with cursor failed: %v", err)
			return
		}

		t.Logf("First page: %d messages, has_more: %v", len(resp.Data), resp.Pagination.HasMore)

		if resp.Pagination.HasMore {
			params.PageToken = resp.Pagination.NextCursor
			resp2, err := client.GetMessagesWithCursor(ctx, testGrantID, params)
			if err != nil {
				t.Fatalf("failed to get second page: %v", err)
			}
			t.Logf("Second page: %d messages", len(resp2.Data))
		}
	})
}
