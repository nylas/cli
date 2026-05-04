package air

import (
	"context"
	"errors"
	"log/slog"
	"net"

	"github.com/nylas/cli/internal/air/cache"
	"github.com/nylas/cli/internal/domain"
)

// shouldQueueEmailAction reports whether an upstream API failure should
// route through the offline queue. The queue must be enabled (cache is
// configured AND queueing is opted in), and the failure has to look like
// a transient network/timeout problem — application errors (4xx) flow
// straight back to the caller.
func (s *Server) shouldQueueEmailAction(err error) bool {
	if !s.offlineQueueEnabled() {
		return false
	}
	if !s.IsOnline() {
		return true
	}
	var netErr net.Error
	return errors.As(err, &netErr) || errors.Is(err, context.DeadlineExceeded)
}

func (s *Server) enqueueMessageUpdate(grantID, accountEmail, emailID string, updateReq *domain.UpdateMessageRequest) error {
	if accountEmail == "" || !s.offlineQueueEnabled() {
		return errors.New("offline queue unavailable")
	}

	return s.withOfflineQueue(accountEmail, func(queue *cache.OfflineQueue) error {
		return queue.Enqueue(cache.ActionUpdateMessage, emailID, cache.UpdateMessagePayload{
			GrantID: grantID,
			EmailID: emailID,
			Unread:  updateReq.Unread,
			Starred: updateReq.Starred,
			Folders: updateReq.Folders,
		})
	})
}

func (s *Server) enqueueMessageDelete(grantID, accountEmail, emailID string) error {
	if accountEmail == "" || !s.offlineQueueEnabled() {
		return errors.New("offline queue unavailable")
	}

	return s.withOfflineQueue(accountEmail, func(queue *cache.OfflineQueue) error {
		return queue.Enqueue(cache.ActionDelete, emailID, cache.DeleteMessagePayload{
			GrantID: grantID,
			EmailID: emailID,
		})
	})
}

// updateCachedEmail mirrors a remote update into the local cache.
// Cache write failures are logged but never bubbled — the live update
// already succeeded. folders nil = leave alone; non-nil = set.
func (s *Server) updateCachedEmail(accountEmail, emailID string, unread, starred *bool, folders []string) {
	if accountEmail == "" || !s.cacheAvailable() {
		return
	}

	if err := s.withEmailStore(accountEmail, func(store *cache.EmailStore) error {
		return store.UpdateMessage(emailID, unread, starred, folders)
	}); err != nil {
		slog.Warn("cache update failed", "emailID", emailID, "account", redactEmail(accountEmail), "err", err)
	}
}

func (s *Server) deleteCachedEmail(accountEmail, emailID string) {
	if accountEmail == "" || !s.cacheAvailable() {
		return
	}

	if err := s.withEmailStore(accountEmail, func(store *cache.EmailStore) error {
		return store.Delete(emailID)
	}); err != nil {
		slog.Warn("cache delete failed", "emailID", emailID, "account", redactEmail(accountEmail), "err", err)
	}
}
