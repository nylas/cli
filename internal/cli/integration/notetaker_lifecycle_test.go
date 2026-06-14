//go:build integration

package integration

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/nylas/cli/internal/domain"
)

func TestCLI_NotetakerUpdateHelp(t *testing.T) {
	if testBinary == "" {
		t.Skip("CLI binary not found")
	}
	stdout, stderr, err := runCLI("notetaker", "update", "--help")
	if err != nil {
		t.Fatalf("notetaker update --help failed: %v\nstderr: %s", err, stderr)
	}
	for _, want := range []string{"--join-time", "--bot-name", "--transcription"} {
		if !strings.Contains(stdout, want) {
			t.Errorf("expected notetaker update help to contain %q, got: %s", want, stdout)
		}
	}
}

func TestCLI_NotetakerLeaveHelp(t *testing.T) {
	if testBinary == "" {
		t.Skip("CLI binary not found")
	}
	stdout, stderr, err := runCLI("notetaker", "leave", "--help")
	if err != nil {
		t.Fatalf("notetaker leave --help failed: %v\nstderr: %s", err, stderr)
	}
	// Help must explain the leave-vs-delete distinction.
	if !strings.Contains(stdout, "delete") {
		t.Errorf("expected notetaker leave help to mention 'delete', got: %s", stdout)
	}
}

// TestNotetakerUpdate_Integration creates a scheduled notetaker, updates it,
// then deletes it. Skips gracefully when notetaker is not enabled.
func TestNotetakerUpdate_Integration(t *testing.T) {
	skipIfMissingCreds(t)
	client := getTestClient()

	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	acquireRateLimit(t)
	created, err := client.CreateNotetaker(ctx, testGrantID, &domain.CreateNotetakerRequest{
		MeetingLink: "https://zoom.us/j/123456789",
		JoinTime:    time.Now().Add(2 * time.Hour).Unix(),
	})
	if err != nil {
		if isUnavailableErr(err) {
			t.Skipf("notetaker not available for this account: %v", err)
		}
		t.Fatalf("CreateNotetaker() error = %v", err)
	}
	// Always clean up the bot we created.
	defer func() {
		acquireRateLimit(t)
		_ = client.DeleteNotetaker(context.Background(), testGrantID, created.ID)
	}()

	newName := "Integration Recorder"
	acquireRateLimit(t)
	updated, err := client.UpdateNotetaker(ctx, testGrantID, created.ID, &domain.UpdateNotetakerRequest{
		Name: newName,
	})
	if err != nil {
		if isUnavailableErr(err) {
			t.Skipf("notetaker update not available for this account: %v", err)
		}
		t.Fatalf("UpdateNotetaker() error = %v", err)
	}
	if updated.ID != created.ID {
		t.Errorf("UpdateNotetaker returned ID %q, want %q", updated.ID, created.ID)
	}
}
