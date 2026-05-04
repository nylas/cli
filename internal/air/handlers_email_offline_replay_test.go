package air

import (
	"context"
	"testing"

	"github.com/nylas/cli/internal/domain"
)

// TestProcessOfflineQueues_PreservesEmptyFoldersIntent locks down the
// payload-shape contract that motivates the nil-vs-empty distinction on
// `[]string` Folders. The Gmail-archive intent is encoded as a non-nil
// empty slice ([]string{}) — distinct from "leave folders alone" (nil).
// A regression that re-introduced `omitempty` on cache.UpdateMessagePayload
// would silently drop the empty slice on the queue's JSON round-trip,
// replaying as nil and reverting every offline-archived message: UI says
// archived, server unchanged.
//
// This test enqueues an empty-Folders update, drains the queue, and
// asserts the captured request still has Folders as a non-nil empty
// slice. Should pass today — the find that motivated it is the absence
// of coverage, not a live bug.
func TestProcessOfflineQueues_PreservesEmptyFoldersIntent(t *testing.T) {
	t.Parallel()

	server, client, accountEmail := newCachedTestServer(t)
	server.SetOnline(false)

	var captured *domain.UpdateMessageRequest
	client.UpdateMessageFunc = func(_ context.Context, _, _ string, req *domain.UpdateMessageRequest) (*domain.Message, error) {
		// Snapshot the request so SetOnline(true)'s replay write is
		// observable after the fact. Mock writes to LastGrantID etc.
		// are not enough — those don't capture Folders.
		captured = req
		return &domain.Message{ID: "email-archive"}, nil
	}

	if err := server.enqueueMessageUpdate("grant-123", accountEmail, "email-archive", &domain.UpdateMessageRequest{
		Folders: []string{},
	}); err != nil {
		t.Fatalf("enqueue update: %v", err)
	}

	server.SetOnline(true) // triggers processOfflineQueues synchronously

	if !client.UpdateMessageCalled {
		t.Fatal("expected UpdateMessage to be replayed when going back online")
	}
	if captured == nil {
		t.Fatal("UpdateMessage was called but request was not captured")
	}
	if captured.Folders == nil {
		t.Fatal("replayed Folders is nil; want non-nil empty slice (Gmail archive intent)")
	}
	if len(captured.Folders) != 0 {
		t.Errorf("replayed Folders = %v, want empty slice (Gmail archive intent)",
			captured.Folders)
	}
}

// TestProcessOfflineQueues_PreservesNilFoldersIntent is the symmetric
// lock-down: nil Folders ("leave folders alone", e.g. mark-as-read
// without touching folders) must replay as nil, not as []string{}.
// Confusing the two on either side of the queue corrupts the user's
// intent: a "mark unread" enqueued offline must not also clear the
// folder list when it eventually replays.
func TestProcessOfflineQueues_PreservesNilFoldersIntent(t *testing.T) {
	t.Parallel()

	server, client, accountEmail := newCachedTestServer(t)
	server.SetOnline(false)

	var captured *domain.UpdateMessageRequest
	client.UpdateMessageFunc = func(_ context.Context, _, _ string, req *domain.UpdateMessageRequest) (*domain.Message, error) {
		captured = req
		return &domain.Message{ID: "email-mark-read"}, nil
	}

	unread := false
	if err := server.enqueueMessageUpdate("grant-123", accountEmail, "email-mark-read", &domain.UpdateMessageRequest{
		Unread: &unread,
		// Folders intentionally omitted — caller's intent is "do not
		// touch folders," and the queue must not invent one on replay.
	}); err != nil {
		t.Fatalf("enqueue update: %v", err)
	}

	server.SetOnline(true)

	if !client.UpdateMessageCalled {
		t.Fatal("expected UpdateMessage to be replayed when going back online")
	}
	if captured == nil {
		t.Fatal("UpdateMessage was called but request was not captured")
	}
	if captured.Folders != nil {
		t.Errorf("replayed Folders = %v (non-nil), want nil (leave-alone intent)",
			captured.Folders)
	}
}
