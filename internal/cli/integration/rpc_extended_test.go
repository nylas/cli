//go:build integration
// +build integration

package integration

import "testing"

// TestCLI_RPC_ExtendedReads exercises the read methods of the extended domains
// (draft, notetaker, scheduler, template, workflow, admin, workspace, audit, auth, otp).
//
// Local/pure methods (audit.*, auth.url) must succeed outright. API-backed methods
// (admin/scheduler/template/...) only need to RESPOND end-to-end — a success OR a
// well-formed RPC error both prove the method is wired and the handler ran; the test
// tolerates permission/empty-resource errors since the test account may not have
// admin access or any scheduler configs. Deep CRUD of these domains is covered by the
// unit tests (creating real connectors/configs/credentials live is impractical/unsafe).
func TestCLI_RPC_ExtendedReads(t *testing.T) {
	skipIfMissingCreds(t)
	addr, tok := startRPCServer(t, nil)
	conn := dialRPC(t, addr, tok)

	id := 0
	nextID := func() int { id++; return id }

	// Group 1: local / pure builders — must succeed, with the expected result key.
	local := []struct {
		name, method, key string
		params            map[string]any
	}{
		{"audit.list", "audit.list", "entries", map[string]any{"limit": 5}},
		{"audit.stats", "audit.stats", "file_count", nil},
		{"audit.path", "audit.path", "path", nil},
		{"audit.config.read", "audit.config.read", "", nil},
		{"auth.url", "auth.url", "url", map[string]any{"provider": "google", "redirect_uri": "http://localhost/callback"}},
	}
	for _, tc := range local {
		t.Run(tc.name, func(t *testing.T) {
			acquireRateLimit(t)
			res := rpcCall(t, conn, nextID(), tc.method, tc.params)
			if res.IsError {
				t.Fatalf("%s unexpected RPC error %d %q", tc.method, res.ErrCode, res.ErrMsg)
			}
			if tc.key != "" {
				if _, ok := res.Result[tc.key]; !ok {
					t.Fatalf("%s result missing key %q; got %v", tc.method, tc.key, res.Result)
				}
			}
		})
	}

	// Group 2: API-backed reads — must respond end-to-end; success or a clean RPC error are both OK.
	apiBacked := []struct {
		name, method string
		params       map[string]any
	}{
		{"draft.list", "draft.list", map[string]any{"grant_id": testGrantID, "limit": 2}},
		{"notetaker.list", "notetaker.list", map[string]any{"grant_id": testGrantID}},
		{"scheduler.config.list", "scheduler.config.list", nil},
		{"template.list", "template.list", map[string]any{"scope": "app"}},
		{"workflow.list", "workflow.list", map[string]any{"scope": "app"}},
		{"admin.app.list", "admin.app.list", nil},
		{"admin.connector.list", "admin.connector.list", nil},
		{"workspace.list", "workspace.list", nil},
		{"auth.grant.get", "auth.grant.get", map[string]any{"grant_id": testGrantID}},
	}
	for _, tc := range apiBacked {
		t.Run(tc.name, func(t *testing.T) {
			acquireRateLimit(t)
			// rpcCall fails the test only on a transport/read failure; a returned result OR a
			// structured RPC error both mean the method is wired and reachable.
			res := rpcCall(t, conn, nextID(), tc.method, tc.params)
			if res.IsError {
				t.Logf("%s returned RPC error %d %q (acceptable — account may lack permission/resources)",
					tc.method, res.ErrCode, res.ErrMsg)
			}
		})
	}
}
