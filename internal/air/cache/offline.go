package cache

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

// ActionType represents the type of queued action.
type ActionType string

const (
	ActionMarkRead      ActionType = "mark_read"
	ActionMarkUnread    ActionType = "mark_unread"
	ActionUpdateMessage ActionType = "update_message"
	ActionStar          ActionType = "star"
	ActionUnstar        ActionType = "unstar"
	ActionArchive       ActionType = "archive"
	ActionDelete        ActionType = "delete"
	ActionMove          ActionType = "move"
	ActionSend          ActionType = "send"
	ActionSaveDraft     ActionType = "save_draft"
	ActionDeleteDraft   ActionType = "delete_draft"
	ActionCreateEvent   ActionType = "create_event"
	ActionUpdateEvent   ActionType = "update_event"
	ActionDeleteEvent   ActionType = "delete_event"
	ActionCreateContact ActionType = "create_contact"
	ActionUpdateContact ActionType = "update_contact"
	ActionDeleteContact ActionType = "delete_contact"
)

// QueuedAction represents an action to be synced when online.
type QueuedAction struct {
	ID         int64      `json:"id"`
	Type       ActionType `json:"type"`
	ResourceID string     `json:"resource_id"`
	Payload    string     `json:"payload"` // JSON-encoded action data
	CreatedAt  time.Time  `json:"created_at"`
	Attempts   int        `json:"attempts"`
	LastError  string     `json:"last_error,omitempty"`
}

// OfflineQueue manages queued actions for offline support.
type OfflineQueue struct {
	db *sql.DB
}

// NewOfflineQueue creates a new offline queue.
func NewOfflineQueue(db *sql.DB) (*OfflineQueue, error) {
	// Create queue table if not exists
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS offline_queue (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			type TEXT NOT NULL,
			resource_id TEXT NOT NULL,
			payload TEXT,
			created_at INTEGER NOT NULL,
			attempts INTEGER DEFAULT 0,
			last_error TEXT
		)
	`)
	if err != nil {
		return nil, fmt.Errorf("create offline_queue table: %w", err)
	}

	// Create index
	_, _ = db.Exec("CREATE INDEX IF NOT EXISTS idx_offline_queue_created ON offline_queue(created_at)")

	return &OfflineQueue{db: db}, nil
}

// Enqueue adds an action to the queue.
func (q *OfflineQueue) Enqueue(actionType ActionType, resourceID string, payload any) error {
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal payload: %w", err)
	}

	_, err = q.db.Exec(`
		INSERT INTO offline_queue (type, resource_id, payload, created_at)
		VALUES (?, ?, ?, ?)
	`, string(actionType), resourceID, string(payloadJSON), time.Now().Unix())

	return err
}

// Dequeue retrieves and removes the oldest action.
func (q *OfflineQueue) Dequeue() (*QueuedAction, error) {
	tx, err := q.db.Begin()
	if err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	row := tx.QueryRow(`
		SELECT id, type, resource_id, payload, created_at, attempts, last_error
		FROM offline_queue
		ORDER BY created_at ASC
		LIMIT 1
	`)

	var action QueuedAction
	var createdAtUnix int64
	var lastError sql.NullString

	err = row.Scan(
		&action.ID, &action.Type, &action.ResourceID, &action.Payload,
		&createdAtUnix, &action.Attempts, &lastError,
	)
	if err == sql.ErrNoRows {
		_ = tx.Rollback()
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	action.CreatedAt = time.Unix(createdAtUnix, 0)
	action.LastError = lastError.String

	// Remove from queue
	_, err = tx.Exec("DELETE FROM offline_queue WHERE id = ?", action.ID)
	if err != nil {
		return nil, err
	}

	return &action, tx.Commit()
}

// Peek retrieves the oldest action without removing it.
func (q *OfflineQueue) Peek() (*QueuedAction, error) {
	row := q.db.QueryRow(`
		SELECT id, type, resource_id, payload, created_at, attempts, last_error
		FROM offline_queue
		ORDER BY created_at ASC
		LIMIT 1
	`)

	var action QueuedAction
	var createdAtUnix int64
	var lastError sql.NullString

	err := row.Scan(
		&action.ID, &action.Type, &action.ResourceID, &action.Payload,
		&createdAtUnix, &action.Attempts, &lastError,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	action.CreatedAt = time.Unix(createdAtUnix, 0)
	action.LastError = lastError.String

	return &action, nil
}

// List retrieves all queued actions.
func (q *OfflineQueue) List() ([]*QueuedAction, error) {
	rows, err := q.db.Query(`
		SELECT id, type, resource_id, payload, created_at, attempts, last_error
		FROM offline_queue
		ORDER BY created_at ASC
	`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var actions []*QueuedAction
	for rows.Next() {
		var action QueuedAction
		var createdAtUnix int64
		var lastError sql.NullString

		err := rows.Scan(
			&action.ID, &action.Type, &action.ResourceID, &action.Payload,
			&createdAtUnix, &action.Attempts, &lastError,
		)
		if err != nil {
			return nil, err
		}

		action.CreatedAt = time.Unix(createdAtUnix, 0)
		action.LastError = lastError.String
		actions = append(actions, &action)
	}

	return actions, rows.Err()
}

// Count returns the number of queued actions.
func (q *OfflineQueue) Count() (int, error) {
	var count int
	err := q.db.QueryRow("SELECT COUNT(*) FROM offline_queue").Scan(&count)
	return count, err
}

// MarkFailed increments the attempt count and records the error.
func (q *OfflineQueue) MarkFailed(id int64, err error) error {
	_, dbErr := q.db.Exec(`
		UPDATE offline_queue
		SET attempts = attempts + 1, last_error = ?
		WHERE id = ?
	`, err.Error(), id)
	return dbErr
}

// Remove deletes an action from the queue.
func (q *OfflineQueue) Remove(id int64) error {
	_, err := q.db.Exec("DELETE FROM offline_queue WHERE id = ?", id)
	return err
}

// Clear removes all queued actions.
func (q *OfflineQueue) Clear() error {
	_, err := q.db.Exec("DELETE FROM offline_queue")
	return err
}

// RemoveStale removes actions older than the given duration.
func (q *OfflineQueue) RemoveStale(maxAge time.Duration) (int, error) {
	cutoff := time.Now().Add(-maxAge).Unix()
	result, err := q.db.Exec("DELETE FROM offline_queue WHERE created_at < ?", cutoff)
	if err != nil {
		return 0, err
	}
	affected, _ := result.RowsAffected()
	return int(affected), nil
}

// RemoveByResourceID removes all actions for a specific resource.
func (q *OfflineQueue) RemoveByResourceID(resourceID string) error {
	_, err := q.db.Exec("DELETE FROM offline_queue WHERE resource_id = ?", resourceID)
	return err
}

// HasPendingActions returns true if there are queued actions.
func (q *OfflineQueue) HasPendingActions() (bool, error) {
	count, err := q.Count()
	return count > 0, err
}

// GetActionData unmarshals the payload into the given type.
func (a *QueuedAction) GetActionData(v any) error {
	return json.Unmarshal([]byte(a.Payload), v)
}

// Email action payloads

// MarkReadPayload is the payload for mark read/unread actions.
type MarkReadPayload struct {
	EmailID string `json:"email_id"`
	Unread  bool   `json:"unread"`
}

// UpdateMessagePayload is the payload for a generic message update.
type UpdateMessagePayload struct {
	EmailID string   `json:"email_id"`
	Unread  *bool    `json:"unread,omitempty"`
	Starred *bool    `json:"starred,omitempty"`
	Folders []string `json:"folders,omitempty"`
}

// StarPayload is the payload for star/unstar actions.
type StarPayload struct {
	EmailID string `json:"email_id"`
	Starred bool   `json:"starred"`
}

// MovePayload is the payload for move actions.
type MovePayload struct {
	EmailID  string `json:"email_id"`
	FolderID string `json:"folder_id"`
}

// SendEmailPayload is the payload for send email actions.
type SendEmailPayload struct {
	To      []string `json:"to"`
	CC      []string `json:"cc,omitempty"`
	BCC     []string `json:"bcc,omitempty"`
	Subject string   `json:"subject"`
	Body    string   `json:"body"`
	ReplyTo string   `json:"reply_to,omitempty"`
}

// DraftPayload is the payload for draft actions.
type DraftPayload struct {
	DraftID string   `json:"draft_id,omitempty"`
	To      []string `json:"to"`
	CC      []string `json:"cc,omitempty"`
	BCC     []string `json:"bcc,omitempty"`
	Subject string   `json:"subject"`
	Body    string   `json:"body"`
}
