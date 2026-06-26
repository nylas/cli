//go:build integration
// +build integration

package integration

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"
)

func TestCLI_RPC_Notification_MessageReceived(t *testing.T) {
	skipIfMissingCreds(t)

	recipient := strings.TrimSpace(getTestEmail())
	if recipient == "" {
		t.Skip("no test email configured")
	}

	addr, tok := startRPCServer(t, map[string]string{"NYLAS_GRANT_ID": testGrantID})
	conn := dialRPC(t, addr, tok)

	marker := fmt.Sprintf("RPC-IT-%d-%d", os.Getpid(), time.Now().UnixNano())
	subject := marker + " notification test"

	time.Sleep(2 * time.Second)
	acquireRateLimit(t)

	stdout, stderr, err := runCLIWithOverrides(2*time.Minute, map[string]string{"NYLAS_GRANT_ID": testGrantID},
		"email", "send",
		"-t", recipient,
		"-s", subject,
		"-b", "rpc integration notification test",
		"-y",
	)
	if err != nil {
		t.Fatalf("email send failed: %v\nstdout: %s\nstderr: %s", err, stdout, stderr)
	}

	params, ok := waitForNotification(t, conn, "message.received", 90*time.Second, func(p json.RawMessage) bool {
		var msg struct {
			Subject string `json:"subject"`
		}
		if err := json.Unmarshal(p, &msg); err != nil {
			return false
		}
		return strings.Contains(msg.Subject, marker)
	})
	if !ok {
		t.Fatalf("timed out waiting for message.received notification with marker %q", marker)
	}
	if len(params) == 0 {
		t.Fatal("message.received notification returned empty params")
	}
}
