package ports

import (
	"context"

	"github.com/nylas/cli/internal/domain"
)

// AuditRequestHook is called after API requests to track request info.
// Set by the cli package during initialization.
var AuditRequestHook func(requestID string, httpStatus int)

// AuditStore defines the interface for audit log storage and retrieval.
type AuditStore interface {
	// GetConfig returns the current audit configuration.
	GetConfig() (*domain.AuditConfig, error)

	// SaveConfig saves the audit configuration.
	SaveConfig(cfg *domain.AuditConfig) error

	// Log records an audit entry.
	Log(entry *domain.AuditEntry) error

	// List returns recent audit entries with optional limit.
	List(ctx context.Context, limit int) ([]domain.AuditEntry, error)

	// Query returns audit entries matching the given options.
	Query(ctx context.Context, opts *domain.AuditQueryOptions) ([]domain.AuditEntry, error)

	// Summary returns aggregate statistics for the given number of days.
	Summary(ctx context.Context, days int) (*domain.AuditSummary, error)

	// Clear removes all audit logs.
	Clear(ctx context.Context) error

	// Path returns the audit log directory path.
	Path() string

	// Stats returns storage statistics (file count, total size).
	Stats() (fileCount int, totalSizeBytes int64, oldestEntry *domain.AuditEntry, err error)

	// Cleanup removes old log files based on retention settings.
	Cleanup(ctx context.Context) error
}
