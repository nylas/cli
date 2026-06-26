//go:build integration
// +build integration

package integration

import "testing"

// TestCLI_RPC_ExtSmoke proves the newly added email/calendar/contacts/auth handlers
// (RegisterEmailExtHandlers / RegisterCalendarExtHandlers / RegisterContactExtHandlers)
// are wired into the running `nylas rpc serve` binary and reachable end-to-end over the
// WebSocket transport. It is intentionally READ-ONLY — no state is mutated, so there is
// nothing to clean up. Per-method live behavior of the underlying client methods is
// already covered by the CLI integration suite; this only guards the RPC wiring.
func TestCLI_RPC_ExtSmoke(t *testing.T) {
	skipIfMissingCreds(t)

	addr, tok := startRPCServer(t, nil)
	conn := dialRPC(t, addr, tok)
	id := 1

	t.Run("email.folder.list", func(t *testing.T) {
		acquireRateLimit(t)
		res := rpcCall(t, conn, id, "email.folder.list", map[string]any{
			"grant_id": testGrantID,
		})
		id++
		if res.IsError {
			t.Fatalf("email.folder.list returned RPC error %d %q", res.ErrCode, res.ErrMsg)
		}
		if _, ok := res.Result["folders"]; !ok {
			t.Fatal("email.folder.list result missing folders key")
		}
	})

	t.Run("email.signature.list", func(t *testing.T) {
		acquireRateLimit(t)
		res := rpcCall(t, conn, id, "email.signature.list", map[string]any{
			"grant_id": testGrantID,
		})
		id++
		if res.IsError {
			t.Fatalf("email.signature.list returned RPC error %d %q", res.ErrCode, res.ErrMsg)
		}
		if _, ok := res.Result["signatures"]; !ok {
			t.Fatal("email.signature.list result missing signatures key")
		}
	})

	t.Run("calendar.get primary", func(t *testing.T) {
		acquireRateLimit(t)
		res := rpcCall(t, conn, id, "calendar.get", map[string]any{
			"grant_id":    testGrantID,
			"calendar_id": "primary",
		})
		id++
		if res.IsError {
			t.Fatalf("calendar.get returned RPC error %d %q", res.ErrCode, res.ErrMsg)
		}
		if _, ok := res.Result["id"]; !ok {
			t.Fatal("calendar.get result missing id key")
		}
	})

	t.Run("contact.group.list", func(t *testing.T) {
		acquireRateLimit(t)
		res := rpcCall(t, conn, id, "contact.group.list", map[string]any{
			"grant_id": testGrantID,
		})
		id++
		if res.IsError {
			t.Fatalf("contact.group.list returned RPC error %d %q", res.ErrCode, res.ErrMsg)
		}
		// Some providers (e.g. Google) return groups; the key must always be present.
		if _, ok := res.Result["groups"]; !ok {
			t.Fatal("contact.group.list result missing groups key")
		}
	})
}
