//go:build integration
// +build integration

package integration

import (
	"encoding/json"
	"testing"
)

func TestCLI_RPC_Reads(t *testing.T) {
	skipIfMissingCreds(t)

	addr, tok := startRPCServer(t, nil)
	conn := dialRPC(t, addr, tok)
	id := 1

	t.Run("email.list", func(t *testing.T) {
		acquireRateLimit(t)
		res := rpcCall(t, conn, id, "email.list", map[string]any{
			"grant_id": testGrantID,
			"limit":    2,
		})
		id++
		if res.IsError {
			t.Fatalf("email.list returned RPC error %d %q", res.ErrCode, res.ErrMsg)
		}
		if _, ok := res.Result["messages"]; !ok {
			t.Fatal("email.list result missing messages key")
		}
		var nextCursor string
		if raw, ok := res.Result["next_cursor"]; ok && string(raw) != "null" {
			if err := json.Unmarshal(raw, &nextCursor); err != nil {
				t.Fatalf("unmarshal next_cursor: %v", err)
			}
		}
	})

	t.Run("email.get", func(t *testing.T) {
		acquireRateLimit(t)
		listRes := rpcCall(t, conn, id, "email.list", map[string]any{
			"grant_id": testGrantID,
			"limit":    2,
		})
		id++
		if listRes.IsError {
			t.Fatalf("email.list returned RPC error %d %q", listRes.ErrCode, listRes.ErrMsg)
		}

		var messages []struct {
			ID string `json:"id"`
		}
		if err := json.Unmarshal(listRes.Result["messages"], &messages); err != nil {
			t.Fatalf("unmarshal messages: %v", err)
		}
		if len(messages) == 0 {
			t.Skip("no messages available")
		}

		acquireRateLimit(t)
		res := rpcCall(t, conn, id, "email.get", map[string]any{
			"grant_id":   testGrantID,
			"message_id": messages[0].ID,
		})
		id++
		if res.IsError {
			t.Fatalf("email.get returned RPC error %d %q", res.ErrCode, res.ErrMsg)
		}
		if _, ok := res.Result["id"]; !ok {
			t.Fatal("email.get result missing id key")
		}
	})

	t.Run("thread.list", func(t *testing.T) {
		acquireRateLimit(t)
		res := rpcCall(t, conn, id, "thread.list", map[string]any{
			"grant_id": testGrantID,
			"limit":    2,
		})
		id++
		if res.IsError {
			t.Fatalf("thread.list returned RPC error %d %q", res.ErrCode, res.ErrMsg)
		}
		if _, ok := res.Result["threads"]; !ok {
			t.Fatal("thread.list result missing threads key")
		}
	})

	t.Run("calendar.list", func(t *testing.T) {
		acquireRateLimit(t)
		res := rpcCall(t, conn, id, "calendar.list", map[string]any{
			"grant_id": testGrantID,
		})
		id++
		if res.IsError {
			t.Fatalf("calendar.list returned RPC error %d %q", res.ErrCode, res.ErrMsg)
		}
		if _, ok := res.Result["calendars"]; !ok {
			t.Fatal("calendar.list result missing calendars key")
		}
	})

	t.Run("event.list", func(t *testing.T) {
		acquireRateLimit(t)
		res := rpcCall(t, conn, id, "event.list", map[string]any{
			"grant_id":    testGrantID,
			"calendar_id": "primary",
			"limit":       2,
		})
		id++
		if res.IsError {
			t.Fatalf("event.list returned RPC error %d %q", res.ErrCode, res.ErrMsg)
		}
		if _, ok := res.Result["events"]; !ok {
			t.Fatal("event.list result missing events key")
		}
	})

	t.Run("contact.list", func(t *testing.T) {
		acquireRateLimit(t)
		res := rpcCall(t, conn, id, "contact.list", map[string]any{
			"grant_id": testGrantID,
			"limit":    2,
		})
		id++
		if res.IsError {
			t.Fatalf("contact.list returned RPC error %d %q", res.ErrCode, res.ErrMsg)
		}
		if _, ok := res.Result["contacts"]; !ok {
			t.Fatal("contact.list result missing contacts key")
		}
	})

	t.Run("agentAccount.list", func(t *testing.T) {
		acquireRateLimit(t)
		res := rpcCall(t, conn, id, "agentAccount.list", map[string]any{})
		id++
		if res.IsError {
			t.Fatalf("agentAccount.list returned RPC error %d %q", res.ErrCode, res.ErrMsg)
		}
		if _, ok := res.Result["accounts"]; !ok {
			t.Fatal("agentAccount.list result missing accounts key")
		}
	})

	t.Run("grant.list", func(t *testing.T) {
		acquireRateLimit(t)
		res := rpcCall(t, conn, id, "grant.list", map[string]any{})
		id++
		if res.IsError {
			t.Fatalf("grant.list returned RPC error %d %q", res.ErrCode, res.ErrMsg)
		}
		if _, ok := res.Result["grants"]; !ok {
			t.Fatal("grant.list result missing grants key")
		}
	})

	t.Run("config.read", func(t *testing.T) {
		acquireRateLimit(t)
		res := rpcCall(t, conn, id, "config.read", map[string]any{})
		id++
		if res.IsError {
			t.Fatalf("config.read returned RPC error %d %q", res.ErrCode, res.ErrMsg)
		}
		for _, key := range []string{"region", "ai_configured"} {
			if _, ok := res.Result[key]; !ok {
				t.Fatalf("config.read result missing %s key", key)
			}
		}
		for _, key := range []string{"api_key", "client_secret", "grants", "ai", "gpg", "dashboard"} {
			if _, ok := res.Result[key]; ok {
				t.Fatalf("config.read result unexpectedly included %s key", key)
			}
		}
	})

	t.Run("email.list pagination", func(t *testing.T) {
		acquireRateLimit(t)
		firstRes := rpcCall(t, conn, id, "email.list", map[string]any{
			"grant_id": testGrantID,
			"limit":    1,
		})
		id++
		if firstRes.IsError {
			t.Fatalf("email.list returned RPC error %d %q", firstRes.ErrCode, firstRes.ErrMsg)
		}

		var nextCursor string
		if raw, ok := firstRes.Result["next_cursor"]; ok && string(raw) != "null" {
			if err := json.Unmarshal(raw, &nextCursor); err != nil {
				t.Fatalf("unmarshal next_cursor: %v", err)
			}
		}
		if nextCursor == "" {
			t.Skip("no next_cursor available")
		}

		acquireRateLimit(t)
		res := rpcCall(t, conn, id, "email.list", map[string]any{
			"grant_id":   testGrantID,
			"limit":      1,
			"page_token": nextCursor,
		})
		id++
		if res.IsError {
			t.Fatalf("email.list page 2 returned RPC error %d %q", res.ErrCode, res.ErrMsg)
		}
		if _, ok := res.Result["messages"]; !ok {
			t.Fatal("email.list page 2 result missing messages key")
		}
	})

	t.Run("email.get not found", func(t *testing.T) {
		acquireRateLimit(t)
		res := rpcCall(t, conn, id, "email.get", map[string]any{
			"grant_id":   testGrantID,
			"message_id": "definitely-not-a-real-id",
		})
		id++
		if !res.IsError {
			t.Fatal("email.get not found returned success")
		}
	})
}
