package audit

import (
	"context"
	"sort"
	"time"

	"github.com/nylas/cli/internal/domain"
)

// Summary returns aggregate statistics for the given number of days.
func (s *FileStore) Summary(ctx context.Context, days int) (*domain.AuditSummary, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if days <= 0 {
		days = 7
	}

	endDate := time.Now()
	startDate := endDate.AddDate(0, 0, -days)

	// Query all entries in the date range
	opts := &domain.AuditQueryOptions{
		Since: startDate,
		Until: endDate,
		Limit: 10000, // High limit for summary
	}

	// Get all log files
	files, err := s.getLogFiles()
	if err != nil {
		return nil, err
	}

	// Sort files by date ascending
	sort.Strings(files)

	var allEntries []domain.AuditEntry

	// Read relevant files
	for _, file := range files {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		fileEntries, err := s.readLogFile(s.basePath + "/" + file)
		if err != nil {
			continue
		}

		for _, entry := range fileEntries {
			if s.matchesQuery(&entry, opts) {
				allEntries = append(allEntries, entry)
			}
		}
	}

	// Build summary
	summary := &domain.AuditSummary{
		StartDate:     startDate,
		EndDate:       endDate,
		Days:          days,
		TotalCommands: len(allEntries),
		CommandCounts: make(map[string]int),
		AccountCounts: make(map[string]int),
		InvokerCounts: make(map[string]int),
	}

	var totalDuration time.Duration
	apiCallCount := 0
	apiErrorCount := 0

	for _, entry := range allEntries {
		// Count success/error
		if entry.Status == domain.AuditStatusSuccess {
			summary.SuccessCount++
		} else {
			summary.ErrorCount++
		}

		// Count commands
		summary.CommandCounts[entry.Command]++

		// Count accounts
		if entry.GrantEmail != "" {
			summary.AccountCounts[entry.GrantEmail]++
		}

		// Count invoker sources
		if entry.InvokerSource != "" {
			summary.InvokerCounts[entry.InvokerSource]++
		}

		// API statistics
		if entry.RequestID != "" {
			apiCallCount++
			totalDuration += entry.Duration
			if entry.HTTPStatus >= 400 {
				apiErrorCount++
			}
		}
	}

	// Calculate percentages
	if summary.TotalCommands > 0 {
		summary.SuccessPercent = float64(summary.SuccessCount) / float64(summary.TotalCommands) * 100
	}

	// API statistics
	summary.TotalAPICalls = apiCallCount
	if apiCallCount > 0 {
		summary.AvgResponseTime = totalDuration / time.Duration(apiCallCount)
		summary.APIErrorRate = float64(apiErrorCount) / float64(apiCallCount) * 100
	}

	return summary, nil
}
