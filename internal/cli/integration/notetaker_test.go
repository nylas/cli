//go:build integration
// +build integration

package integration

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/nylas/cli/internal/domain"
)

func TestNotetaker_Integration(t *testing.T) {
	skipIfMissingCreds(t)

	client := getTestClient()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	t.Run("ListNotetakers", func(t *testing.T) {
		notetakers, err := client.ListNotetakers(ctx, testGrantID, nil)
		if err != nil {
			t.Fatalf("ListNotetakers() error = %v", err)
		}
		// Should return list (might be empty)
		_ = notetakers
	})

	t.Run("CreateAndDeleteNotetaker", func(t *testing.T) {
		// Create notetaker
		req := &domain.CreateNotetakerRequest{
			MeetingLink: "https://zoom.us/j/123456789",
			JoinTime:    time.Now().Add(1 * time.Hour).Unix(),
		}

		notetaker, err := client.CreateNotetaker(ctx, testGrantID, req)
		if err != nil {
			// Some test accounts may not have notetaker access
			if strings.Contains(err.Error(), "not found") ||
				strings.Contains(err.Error(), "forbidden") ||
				strings.Contains(err.Error(), "not available") {
				t.Skip("Notetaker not available for this account")
			}
			t.Fatalf("CreateNotetaker() error = %v", err)
		}

		if notetaker.ID == "" {
			t.Error("Created notetaker has empty ID")
		}

		// Get notetaker
		retrieved, err := client.GetNotetaker(ctx, testGrantID, notetaker.ID)
		if err != nil {
			t.Errorf("GetNotetaker() error = %v", err)
		}
		if retrieved.ID != notetaker.ID {
			t.Errorf("GetNotetaker() ID = %q, want %q", retrieved.ID, notetaker.ID)
		}

		// Delete notetaker
		if err := client.DeleteNotetaker(ctx, testGrantID, notetaker.ID); err != nil {
			// Note: Delete may not be supported (405) - log but don't fail
			if strings.Contains(err.Error(), "status 405") || strings.Contains(err.Error(), "Method Not Allowed") {
				t.Logf("DeleteNotetaker() not supported by API (405): %v", err)
				t.Skip("Delete operation not supported by API")
			}
			t.Errorf("DeleteNotetaker() error = %v", err)
		}

		// Verify deletion
		_, err = client.GetNotetaker(ctx, testGrantID, notetaker.ID)
		if err == nil {
			t.Error("GetNotetaker() after delete should return error")
		}
	})
}

func TestNotetaker_ValidationErrors(t *testing.T) {
	skipIfMissingCreds(t)

	client := getTestClient()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	t.Run("CreateNotetaker_MissingRequired", func(t *testing.T) {
		// Missing required fields
		req := &domain.CreateNotetakerRequest{}

		_, err := client.CreateNotetaker(ctx, testGrantID, req)
		if err == nil {
			t.Error("CreateNotetaker() with missing fields should return error")
		}
	})

	t.Run("GetNotetaker_InvalidID", func(t *testing.T) {
		_, err := client.GetNotetaker(ctx, testGrantID, "invalid-notetaker-id")
		if err == nil {
			t.Error("GetNotetaker() with invalid ID should return error")
		}
	})

	t.Run("DeleteNotetaker_InvalidID", func(t *testing.T) {
		err := client.DeleteNotetaker(ctx, testGrantID, "invalid-notetaker-id")
		if err == nil {
			t.Error("DeleteNotetaker() with invalid ID should return error")
		}
	})
}
