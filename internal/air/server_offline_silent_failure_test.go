package air

import (
	"context"
	"errors"
	"testing"

	"github.com/nylas/cli/internal/domain"
	"github.com/stretchr/testify/assert"
)

// TestProcessOfflineQueue_ActionDroppedAfter3Attempts_LogsFailure pins
// the worst silent-failure mode in the offline replay loop:
// server_offline.go:45-46 silently does
//
//	if action.Attempts >= 3 {
//	    _ = s.removeOfflineAction(email, action.ID)
//	}
//
// when an action has failed three times. The action is permanently
// dropped — the user took an action offline, expected it to sync, and
// will never know it didn't. There is no slog record, no metric, no
// trace at all. Support cannot correlate "my offline archive
// disappeared" reports to log lines because no log line was produced.
//
// EXPECTED FAILURE today: after four processOfflineQueue ticks the
// action is gone from the queue but the slog buffer contains nothing
// referencing the dropped emailID or the upstream-error sentinel.
// After the fix an Error-level log entry should fire with the action's
// resource_id and the last error captured during the attempts.
func TestProcessOfflineQueue_ActionDroppedAfter3Attempts_LogsFailure(t *testing.T) {
	// No t.Parallel — captureSlog mutates process-global slog default.
	server, client, accountEmail := newCachedTestServer(t)

	const sentinelEmailID = "drop-canary-3strike-XYZ"
	const upstreamErrSentinel = "TRANSIENT-NYLAS-ERR-CANARY-555"
	client.UpdateMessageFunc = func(_ context.Context, _, _ string, _ *domain.UpdateMessageRequest) (*domain.Message, error) {
		return nil, errors.New(upstreamErrSentinel)
	}

	unread := false
	if err := server.enqueueMessageUpdate("grant-123", accountEmail, sentinelEmailID, &domain.UpdateMessageRequest{
		Unread: &unread,
	}); err != nil {
		t.Fatalf("enqueue: %v", err)
	}

	logs := captureSlog(t)

	// Four ticks: the peek returns the action.Attempts value AFTER the
	// previous markFailed wrote attempts+=1, so the sequence is
	//   tick 1: peek attempts=0 → fail → markFailed → attempts=1
	//   tick 2: peek attempts=1 → fail → markFailed → attempts=2
	//   tick 3: peek attempts=2 → fail → markFailed → attempts=3
	//   tick 4: peek attempts=3 → fail → 3>=3 → SILENT removeOfflineAction
	for range 4 {
		server.processOfflineQueue(accountEmail)
	}

	// Sanity-check: action really was dropped.
	db, err := server.cacheManager.GetDB(accountEmail)
	if err != nil {
		t.Fatalf("get db: %v", err)
	}
	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM offline_queue").Scan(&count); err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 0 {
		t.Fatalf("queue not drained: count=%d, want 0 (3-strike drop expected)", count)
	}

	got := logs.String()
	assert.Contains(t, got, sentinelEmailID,
		"slog must record the dropped action's resource_id for support diagnosis "+
			"(server_offline.go:46 currently silent); got %q", got)
	assert.Contains(t, got, upstreamErrSentinel,
		"slog must record the upstream error context that drove the action "+
			"to be permanently dropped; got %q", got)
}

// TestProcessOfflineQueue_PermanentResolveFailure_LogsFailure pins the
// twin silent-drop site at server_offline.go:35-36: when the grant
// referenced by a queued action can no longer be resolved (e.g., the
// user disconnected the account while an offline action was sitting in
// the queue), the same 3-strike silent removal applies. Today this
// also fails closed with no observability.
func TestProcessOfflineQueue_PermanentGrantResolveFailure_LogsFailure(t *testing.T) {
	// No t.Parallel — captureSlog mutates process-global slog default.
	server, _, accountEmail := newCachedTestServer(t)

	const sentinelEmailID = "drop-canary-grant-resolve-XYZ"

	unread := false
	if err := server.enqueueMessageUpdate("grant-DISCONNECTED-123", accountEmail, sentinelEmailID, &domain.UpdateMessageRequest{
		Unread: &unread,
	}); err != nil {
		t.Fatalf("enqueue: %v", err)
	}

	// Remove the grant after enqueue so resolveQueuedActionGrantID
	// returns "queued grant <id> unavailable" on every tick.
	store := server.grantStore.(*testGrantStore)
	store.grants = nil

	logs := captureSlog(t)

	for range 4 {
		server.processOfflineQueue(accountEmail)
	}

	got := logs.String()
	assert.Contains(t, got, sentinelEmailID,
		"slog must record the dropped action's resource_id when the grant "+
			"can no longer be resolved (server_offline.go:36 currently silent); got %q", got)
}
