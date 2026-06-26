//go:build integration
// +build integration

package integration

import (
	"encoding/json"
	"testing"
	"time"
)

func TestCLI_RPC_Writes(t *testing.T) {
	skipIfMissingCreds(t)

	addr, token := startRPCServer(t, nil)
	conn := dialRPC(t, addr, token)
	id := 0
	nextID := func() int {
		id++
		return id
	}
	call := func(t *testing.T, method string, params map[string]any) rpcResult {
		t.Helper()
		acquireRateLimit(t)
		return rpcCall(t, conn, nextID(), method, params)
	}
	requireNoError := func(t *testing.T, method string, result rpcResult) {
		t.Helper()
		if result.IsError {
			t.Fatalf("%s returned RPC error %d %q", method, result.ErrCode, result.ErrMsg)
		}
	}
	requireInvalidParams := func(t *testing.T, method string, result rpcResult) {
		t.Helper()
		if !result.IsError || result.ErrCode != -32602 {
			t.Fatalf("%s error = (%v, %d, %q), want code -32602", method, result.IsError, result.ErrCode, result.ErrMsg)
		}
	}
	requireDeleted := func(t *testing.T, method string, result rpcResult) {
		t.Helper()
		requireNoError(t, method, result)
		var deleted bool
		if raw, ok := result.Result["deleted"]; ok {
			_ = json.Unmarshal(raw, &deleted)
		}
		if !deleted {
			t.Fatalf("%s deleted = false, result = %v", method, result.Result)
		}
	}
	skipCreateError := func(t *testing.T, method string, result rpcResult) {
		t.Helper()
		if result.IsError && result.ErrCode == -32603 {
			t.Skipf("%s not supported by this provider/account: RPC error %d %q", method, result.ErrCode, result.ErrMsg)
		}
	}

	t.Run("draft round-trip", func(t *testing.T) {
		requireInvalidParams(t, "draft.update", call(t, "draft.update", map[string]any{
			"grant_id": testGrantID,
		}))

		create := call(t, "draft.create", map[string]any{
			"grant_id": testGrantID,
			"subject":  "RPC-IT draft (delete me)",
			"body":     "x",
		})
		skipCreateError(t, "draft.create", create)
		requireNoError(t, "draft.create", create)
		draftID := rpcID(t, create.Result)
		if draftID == "" {
			t.Fatal("draft.create result missing id")
		}

		deleted := false
		t.Cleanup(func() {
			if deleted {
				return
			}
			requireDeleted(t, "draft.delete cleanup", call(t, "draft.delete", map[string]any{
				"grant_id": testGrantID,
				"draft_id": draftID,
			}))
		})

		requireNoError(t, "draft.update", call(t, "draft.update", map[string]any{
			"grant_id": testGrantID,
			"draft_id": draftID,
			"subject":  "RPC-IT draft updated",
			"body":     "y",
		}))
		requireDeleted(t, "draft.delete", call(t, "draft.delete", map[string]any{
			"grant_id": testGrantID,
			"draft_id": draftID,
		}))
		deleted = true
	})

	t.Run("contact round-trip", func(t *testing.T) {
		requireInvalidParams(t, "contact.update", call(t, "contact.update", map[string]any{
			"grant_id": testGrantID,
		}))

		create := call(t, "contact.create", map[string]any{
			"grant_id":   testGrantID,
			"given_name": "RPCIT",
			"surname":    "DeleteMe",
		})
		skipCreateError(t, "contact.create", create)
		requireNoError(t, "contact.create", create)
		contactID := rpcID(t, create.Result)
		if contactID == "" {
			t.Fatal("contact.create result missing id")
		}

		deleted := false
		t.Cleanup(func() {
			if deleted {
				return
			}
			requireDeleted(t, "contact.delete cleanup", call(t, "contact.delete", map[string]any{
				"grant_id":   testGrantID,
				"contact_id": contactID,
			}))
		})

		requireNoError(t, "contact.update", call(t, "contact.update", map[string]any{
			"grant_id":   testGrantID,
			"contact_id": contactID,
			"given_name": "RPCITUpdated",
		}))
		requireDeleted(t, "contact.delete", call(t, "contact.delete", map[string]any{
			"grant_id":   testGrantID,
			"contact_id": contactID,
		}))
		deleted = true
	})

	t.Run("event round-trip", func(t *testing.T) {
		requireInvalidParams(t, "event.update", call(t, "event.update", map[string]any{
			"grant_id":    testGrantID,
			"calendar_id": "primary",
		}))

		start := time.Now().Add(30 * 24 * time.Hour).Unix()
		create := call(t, "event.create", map[string]any{
			"grant_id":    testGrantID,
			"calendar_id": "primary",
			"title":       "RPC-IT event (delete me)",
			"when": map[string]any{
				"object":     "timespan",
				"start_time": start,
				"end_time":   start + int64(time.Hour/time.Second),
			},
		})
		skipCreateError(t, "event.create", create)
		requireNoError(t, "event.create", create)
		eventID := rpcID(t, create.Result)
		if eventID == "" {
			t.Fatal("event.create result missing id")
		}

		deleted := false
		t.Cleanup(func() {
			if deleted {
				return
			}
			requireDeleted(t, "event.delete cleanup", call(t, "event.delete", map[string]any{
				"grant_id":    testGrantID,
				"calendar_id": "primary",
				"event_id":    eventID,
			}))
		})

		requireNoError(t, "event.update", call(t, "event.update", map[string]any{
			"grant_id":    testGrantID,
			"calendar_id": "primary",
			"event_id":    eventID,
			"title":       "RPC-IT event updated",
		}))
		requireDeleted(t, "event.delete", call(t, "event.delete", map[string]any{
			"grant_id":    testGrantID,
			"calendar_id": "primary",
			"event_id":    eventID,
		}))
		deleted = true
	})
}
