package air

import (
	"context"
	"errors"
	"fmt"

	"github.com/nylas/cli/internal/air/cache"
	"github.com/nylas/cli/internal/domain"
)

// processOfflineQueues processes all pending offline actions.
func (s *Server) processOfflineQueues() {
	for _, email := range s.offlineQueueEmails() {
		s.processOfflineQueue(email)
	}
}

// processOfflineQueue processes a single account's offline queue.
func (s *Server) processOfflineQueue(email string) {
	if s.nylasClient == nil || !s.IsOnline() {
		return
	}

	ctx := context.Background()

	for {
		action, err := s.peekOfflineAction(email)
		if err != nil || action == nil {
			return
		}

		grantID, err := s.resolveQueuedActionGrantID(email, action)
		if err != nil {
			if action.Attempts >= 3 {
				_ = s.removeOfflineAction(email, action.ID)
			} else {
				_ = s.markOfflineActionFailed(email, action.ID, err)
			}
			return
		}

		err = s.processOfflineAction(ctx, grantID, action)
		if err != nil {
			if action.Attempts >= 3 {
				_ = s.removeOfflineAction(email, action.ID)
			} else {
				_ = s.markOfflineActionFailed(email, action.ID, err)
			}
			return
		}

		if err := s.removeOfflineAction(email, action.ID); err != nil {
			return
		}
	}
}

func (s *Server) peekOfflineAction(email string) (*cache.QueuedAction, error) {
	var action *cache.QueuedAction
	err := s.withOfflineQueue(email, func(queue *cache.OfflineQueue) error {
		var err error
		action, err = queue.Peek()
		return err
	})
	return action, err
}

func (s *Server) markOfflineActionFailed(email string, actionID int64, markErr error) error {
	return s.withOfflineQueue(email, func(queue *cache.OfflineQueue) error {
		return queue.MarkFailed(actionID, markErr)
	})
}

func (s *Server) removeOfflineAction(email string, actionID int64) error {
	return s.withOfflineQueue(email, func(queue *cache.OfflineQueue) error {
		return queue.Remove(actionID)
	})
}

func (s *Server) resolveQueuedActionGrantID(accountEmail string, action *cache.QueuedAction) (string, error) {
	if action == nil {
		return "", errors.New("queued action is nil")
	}

	if grantID := queuedActionGrantID(action); grantID != "" {
		grant, err := s.grantStore.GetGrant(grantID)
		if err != nil || grant == nil {
			return "", fmt.Errorf("queued grant %s unavailable", grantID)
		}
		return grantID, nil
	}

	grants, err := s.grantStore.ListGrants()
	if err != nil {
		return "", err
	}
	for _, grant := range grants {
		if grant.Email == accountEmail {
			return grant.ID, nil
		}
	}

	return "", fmt.Errorf("no grant found for account %s", accountEmail)
}

func queuedActionGrantID(action *cache.QueuedAction) string {
	switch action.Type {
	case cache.ActionUpdateMessage:
		var payload cache.UpdateMessagePayload
		if action.GetActionData(&payload) == nil {
			return payload.GrantID
		}
	case cache.ActionMarkRead, cache.ActionMarkUnread:
		var payload cache.MarkReadPayload
		if action.GetActionData(&payload) == nil {
			return payload.GrantID
		}
	case cache.ActionStar, cache.ActionUnstar:
		var payload cache.StarPayload
		if action.GetActionData(&payload) == nil {
			return payload.GrantID
		}
	case cache.ActionMove:
		var payload cache.MovePayload
		if action.GetActionData(&payload) == nil {
			return payload.GrantID
		}
	case cache.ActionDelete:
		var payload cache.DeleteMessagePayload
		if action.GetActionData(&payload) == nil {
			return payload.GrantID
		}
	}

	return ""
}

// processOfflineAction processes a single offline action.
func (s *Server) processOfflineAction(ctx context.Context, grantID string, action *cache.QueuedAction) error {
	switch action.Type {
	case cache.ActionUpdateMessage:
		var payload cache.UpdateMessagePayload
		if err := action.GetActionData(&payload); err != nil {
			return err
		}
		_, err := s.nylasClient.UpdateMessage(ctx, grantID, payload.EmailID, &domain.UpdateMessageRequest{
			Unread:  payload.Unread,
			Starred: payload.Starred,
			Folders: payload.Folders,
		})
		return err

	case cache.ActionMarkRead, cache.ActionMarkUnread:
		var payload cache.MarkReadPayload
		if err := action.GetActionData(&payload); err != nil {
			return err
		}
		_, err := s.nylasClient.UpdateMessage(ctx, grantID, payload.EmailID, &domain.UpdateMessageRequest{
			Unread: &payload.Unread,
		})
		return err

	case cache.ActionStar, cache.ActionUnstar:
		var payload cache.StarPayload
		if err := action.GetActionData(&payload); err != nil {
			return err
		}
		_, err := s.nylasClient.UpdateMessage(ctx, grantID, payload.EmailID, &domain.UpdateMessageRequest{
			Starred: &payload.Starred,
		})
		return err

	case cache.ActionDelete:
		var payload cache.DeleteMessagePayload
		if err := action.GetActionData(&payload); err == nil && payload.EmailID != "" {
			return s.nylasClient.DeleteMessage(ctx, grantID, payload.EmailID)
		}
		return s.nylasClient.DeleteMessage(ctx, grantID, action.ResourceID)

	case cache.ActionMove:
		var payload cache.MovePayload
		if err := action.GetActionData(&payload); err != nil {
			return err
		}
		_, err := s.nylasClient.UpdateMessage(ctx, grantID, payload.EmailID, &domain.UpdateMessageRequest{
			Folders: []string{payload.FolderID},
		})
		return err

	default:
		return fmt.Errorf("unknown action type: %s", action.Type)
	}
}
