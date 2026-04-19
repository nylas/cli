//go:build integration

package integration

import (
	"context"
	"errors"
	"testing"
)

func TestLookupRecentMessageID_RetriesUntilSuccess(t *testing.T) {
	attempts := 0
	beforeCalls := 0

	messageID, err := lookupRecentMessageID(context.Background(), func() {
		beforeCalls++
	}, func(ctx context.Context) (string, error) {
		attempts++
		if attempts < 3 {
			return "", errors.New("temporary failure")
		}
		return "msg-123", nil
	})

	if err != nil {
		t.Fatalf("lookupRecentMessageID() error = %v", err)
	}
	if messageID != "msg-123" {
		t.Fatalf("lookupRecentMessageID() = %q, want %q", messageID, "msg-123")
	}
	if attempts != 3 {
		t.Fatalf("lookupRecentMessageID() attempts = %d, want 3", attempts)
	}
	if beforeCalls != 3 {
		t.Fatalf("lookupRecentMessageID() beforeAttempt calls = %d, want 3", beforeCalls)
	}
}
