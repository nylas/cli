package rpcserver

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/nylas/cli/internal/domain"
	"github.com/nylas/cli/internal/ports"
)

type auditListParams struct {
	Limit int `json:"limit,omitempty"`
}

type auditEntriesResult struct {
	Entries []domain.AuditEntry `json:"entries"`
}

type auditStatsResult struct {
	FileCount      int                `json:"file_count"`
	TotalSizeBytes int64              `json:"total_size_bytes"`
	OldestEntry    *domain.AuditEntry `json:"oldest_entry"`
}

type auditPathResult struct {
	Path string `json:"path"`
}

type auditOKResult struct {
	OK bool `json:"ok"`
}

type auditClearedResult struct {
	Cleared bool `json:"cleared"`
}

func RegisterAuditHandlers(d *Dispatcher, svc ports.AuditStore) {
	d.Register("audit.list", func(ctx context.Context, params json.RawMessage) (any, error) {
		var p auditListParams
		if err := decodeParams(params, &p); err != nil {
			return nil, err
		}

		entries, err := svc.List(ctx, p.Limit)
		if err != nil {
			return nil, fmt.Errorf("audit.list: %w", err)
		}
		return auditEntriesResult{Entries: entries}, nil
	})

	d.Register("audit.query", func(ctx context.Context, params json.RawMessage) (any, error) {
		var p domain.AuditQueryOptions
		if err := decodeParams(params, &p); err != nil {
			return nil, err
		}

		entries, err := svc.Query(ctx, &p)
		if err != nil {
			return nil, fmt.Errorf("audit.query: %w", err)
		}
		return auditEntriesResult{Entries: entries}, nil
	})

	d.Register("audit.summary", func(ctx context.Context, params json.RawMessage) (any, error) {
		var p struct {
			Days int `json:"days,omitempty"`
		}
		if err := decodeParams(params, &p); err != nil {
			return nil, err
		}

		summary, err := svc.Summary(ctx, p.Days)
		if err != nil {
			return nil, fmt.Errorf("audit.summary: %w", err)
		}
		return summary, nil
	})

	d.Register("audit.stats", func(_ context.Context, params json.RawMessage) (any, error) {
		var p struct{}
		if err := decodeParams(params, &p); err != nil {
			return nil, err
		}

		fileCount, totalSizeBytes, oldestEntry, err := svc.Stats()
		if err != nil {
			return nil, fmt.Errorf("audit.stats: %w", err)
		}
		return auditStatsResult{
			FileCount:      fileCount,
			TotalSizeBytes: totalSizeBytes,
			OldestEntry:    oldestEntry,
		}, nil
	})

	d.Register("audit.config.read", func(_ context.Context, params json.RawMessage) (any, error) {
		var p struct{}
		if err := decodeParams(params, &p); err != nil {
			return nil, err
		}

		cfg, err := svc.GetConfig()
		if err != nil {
			return nil, fmt.Errorf("audit.config.read: %w", err)
		}
		return cfg, nil
	})

	d.Register("audit.config.save", func(_ context.Context, params json.RawMessage) (any, error) {
		var p domain.AuditConfig
		if err := decodeParams(params, &p); err != nil {
			return nil, err
		}

		if err := svc.SaveConfig(&p); err != nil {
			return nil, fmt.Errorf("audit.config.save: %w", err)
		}
		return auditOKResult{OK: true}, nil
	})

	d.Register("audit.path", func(_ context.Context, params json.RawMessage) (any, error) {
		var p struct{}
		if err := decodeParams(params, &p); err != nil {
			return nil, err
		}

		return auditPathResult{Path: svc.Path()}, nil
	})

	d.Register("audit.clear", func(ctx context.Context, params json.RawMessage) (any, error) {
		var p struct{}
		if err := decodeParams(params, &p); err != nil {
			return nil, err
		}

		if err := svc.Clear(ctx); err != nil {
			return nil, fmt.Errorf("audit.clear: %w", err)
		}
		return auditClearedResult{Cleared: true}, nil
	})

	d.Register("audit.cleanup", func(ctx context.Context, params json.RawMessage) (any, error) {
		var p struct{}
		if err := decodeParams(params, &p); err != nil {
			return nil, err
		}

		if err := svc.Cleanup(ctx); err != nil {
			return nil, fmt.Errorf("audit.cleanup: %w", err)
		}
		return auditOKResult{OK: true}, nil
	})
}
