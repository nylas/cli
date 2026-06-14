//go:build integration

package integration

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/nylas/cli/internal/domain"
)

func TestCLI_EmailCleanHelp(t *testing.T) {
	if testBinary == "" {
		t.Skip("CLI binary not found")
	}
	stdout, stderr, err := runCLI("email", "clean", "--help")
	if err != nil {
		t.Fatalf("email clean --help failed: %v\nstderr: %s", err, stderr)
	}
	if !strings.Contains(stdout, "conversation") {
		t.Errorf("expected email clean help to mention 'conversation', got: %s", stdout)
	}
}

func TestCLI_EmailMoveHelp(t *testing.T) {
	if testBinary == "" {
		t.Skip("CLI binary not found")
	}
	stdout, stderr, err := runCLI("email", "move", "--help")
	if err != nil {
		t.Fatalf("email move --help failed: %v\nstderr: %s", err, stderr)
	}
	for _, want := range []string{"--folder", "--archive"} {
		if !strings.Contains(stdout, want) {
			t.Errorf("expected email move help to contain %q, got: %s", want, stdout)
		}
	}
}

// TestCLI_EmailMoveGuards exercises the mutually-exclusive / required-destination
// guards through the real binary (these run before any API call).
func TestCLI_EmailMoveGuards(t *testing.T) {
	if testBinary == "" {
		t.Skip("CLI binary not found")
	}

	_, _, err := runCLI("email", "move", "msg-1", "--folder", "F1", "--archive")
	if err == nil {
		t.Error("expected error when both --folder and --archive are given")
	}

	_, _, err = runCLI("email", "move", "msg-1")
	if err == nil {
		t.Error("expected error when neither --folder nor --archive is given")
	}
}

// TestEmailClean_Integration cleans a real message (read-only — clean returns
// parsed text without modifying the message).
func TestEmailClean_Integration(t *testing.T) {
	skipIfMissingCreds(t)
	client := getTestClient()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	acquireRateLimit(t)
	messages, err := client.GetMessages(ctx, testGrantID, 1)
	if err != nil {
		t.Fatalf("GetMessages() error = %v", err)
	}
	if len(messages) == 0 {
		t.Skip("no messages available to clean")
	}

	acquireRateLimit(t)
	cleaned, err := client.CleanMessages(ctx, testGrantID, &domain.CleanMessagesRequest{
		MessageIDs: []string{messages[0].ID},
	})
	if err != nil {
		if isUnavailableErr(err) {
			t.Skipf("clean messages not available for this account: %v", err)
		}
		t.Fatalf("CleanMessages() error = %v", err)
	}
	if len(cleaned) != 1 {
		t.Fatalf("expected 1 cleaned message, got %d", len(cleaned))
	}
	// The cleaned message must correspond to the one we asked for.
	if cleaned[0].ID != messages[0].ID {
		t.Errorf("cleaned message ID = %q, want %q", cleaned[0].ID, messages[0].ID)
	}
}
