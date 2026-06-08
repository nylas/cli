//go:build integration

package integration

import (
	"os"
	"strings"
	"testing"
)

func TestCLI_EmailReplyHelp(t *testing.T) {
	if testBinary == "" {
		t.Skip("CLI binary not found")
	}

	stdout, stderr, err := runCLI("email", "reply", "--help")
	if err != nil {
		t.Fatalf("email reply --help failed: %v\nstderr: %s", err, stderr)
	}

	for _, want := range []string{"--all", "--body", "--interactive", "reply_to_message_id"} {
		if !strings.Contains(stdout, want) {
			t.Errorf("expected %q in reply help, got: %s", want, stdout)
		}
	}

	t.Logf("email reply help output:\n%s", stdout)
}

func TestCLI_EmailReply_RequiresBody(t *testing.T) {
	if testBinary == "" {
		t.Skip("CLI binary not found")
	}

	// Body validation happens before any client/network call, so this needs no
	// credentials and must fail fast regardless of the message ID.
	_, stderr, err := runCLI("email", "reply", "some-message-id", "--yes")
	if err == nil {
		t.Fatal("expected error when replying without a body, but command succeeded")
	}
	if !strings.Contains(stderr, "reply body is required") {
		t.Errorf("expected 'reply body is required' error, got: %s", stderr)
	}
}

func TestCLI_EmailReply_RoundTrip(t *testing.T) {
	if os.Getenv("NYLAS_TEST_SEND_EMAIL") != "true" {
		t.Skip("Skipping reply round-trip - set NYLAS_TEST_SEND_EMAIL=true to enable")
	}
	skipIfMissingCreds(t)

	// The original message is sent from the grant, so the grant is its sender.
	// Replying excludes your own address by design, so the thread needs a
	// non-self recipient and the reply must use --all to reach them.
	target := getSendTargetEmail(t)
	if strings.EqualFold(strings.TrimSpace(target), strings.TrimSpace(getGrantEmail(t))) {
		t.Skip("reply round-trip needs a send target different from the grant's own address (set NYLAS_TEST_EMAIL)")
	}

	sendStdout, sendStderr, sendErr := runCLIWithRateLimit(t,
		"email", "send",
		"--to", target,
		"--subject", "CLI Reply Round Trip",
		"--body", "Original message for the reply integration test.",
		"--yes",
		testGrantID,
	)
	if sendErr != nil {
		t.Fatalf("email send failed: %v\nstderr: %s", sendErr, sendStderr)
	}

	messageID := extractMessageID(sendStdout)
	if messageID == "" {
		t.Fatalf("failed to extract message ID from send output: %s", sendStdout)
	}
	t.Cleanup(func() {
		_, _, _ = runCLI("email", "delete", messageID, "--yes", testGrantID)
	})

	replyStdout, replyStderr, replyErr := runCLIWithRateLimit(t,
		"email", "reply", messageID, testGrantID,
		"--all",
		"--body", "This is the threaded reply.",
		"--yes",
	)
	if replyErr != nil {
		t.Fatalf("email reply failed: %v\nstderr: %s", replyErr, replyStderr)
	}

	if !strings.Contains(replyStdout, "Reply sent successfully! Message ID:") {
		t.Errorf("expected reply success confirmation in output, got: %s", replyStdout)
	}

	if replyID := extractMessageID(replyStdout); replyID != "" {
		t.Cleanup(func() {
			_, _, _ = runCLI("email", "delete", replyID, "--yes", testGrantID)
		})
	}

	t.Logf("email reply output:\n%s", replyStdout)
}
