package common

import (
	"context"
	"fmt"
	"io"
	"os"
)

// MaxAPILimit is the maximum number of items the Nylas API returns per request.
const MaxAPILimit = 200

// PageResult represents a paginated API response.
type PageResult[T any] struct {
	Data       []T    // The items in this page
	NextCursor string // Cursor for the next page, empty if no more pages
	RequestID  string // Request ID for debugging
}

// HasMore returns true if there are more pages to fetch.
func (p PageResult[T]) HasMore() bool {
	return p.NextCursor != ""
}

// PageFetcher is a function that fetches a single page of results.
type PageFetcher[T any] func(ctx context.Context, cursor string) (PageResult[T], error)

// PaginationConfig configures pagination behavior.
type PaginationConfig struct {
	PageSize     int       // Items per page
	MaxItems     int       // Maximum total items (0 = unlimited)
	MaxPages     int       // Maximum pages to fetch (0 = unlimited)
	ShowProgress bool      // Show progress indicator
	Writer       io.Writer // Output writer for progress
}

// NormalizePageSize clamps a requested page size to the API maximum.
// Zero or negative values fall back to the maximum API page size.
func NormalizePageSize(limit int) int {
	pageSize := min(limit, MaxAPILimit)
	if pageSize <= 0 {
		return MaxAPILimit
	}
	return pageSize
}

// DefaultPaginationConfig returns default pagination settings.
func DefaultPaginationConfig() PaginationConfig {
	return PaginationConfig{
		PageSize:     50,
		MaxItems:     0,
		MaxPages:     0,
		ShowProgress: true,
		Writer:       os.Stderr,
	}
}

// FetchAllPages fetches all pages using the provided fetcher function.
func FetchAllPages[T any](ctx context.Context, config PaginationConfig, fetcher PageFetcher[T]) ([]T, error) {
	var results []T
	cursor := ""
	pageCount := 0

	var counter *Counter
	if config.ShowProgress && !IsQuiet() {
		counter = NewCounter("Fetching items")
	}

	for {
		// Check context cancellation
		select {
		case <-ctx.Done():
			return results, ctx.Err()
		default:
		}

		// Fetch the next page
		page, err := fetcher(ctx, cursor)
		if err != nil {
			if counter != nil {
				counter.Finish()
			}
			return results, fmt.Errorf("failed to fetch page %d: %w", pageCount+1, err)
		}

		// Append results
		results = append(results, page.Data...)
		pageCount++

		// Update progress
		if counter != nil {
			for range page.Data {
				counter.Increment()
			}
		}

		// Check if we've reached the limit
		if config.MaxItems > 0 && len(results) >= config.MaxItems {
			results = results[:config.MaxItems]
			break
		}

		if config.MaxPages > 0 && pageCount >= config.MaxPages {
			break
		}

		// Check if there are more pages
		if !page.HasMore() {
			break
		}

		// Guard against server-side pagination bugs that would loop forever:
		// an empty page that claims more results, or a cursor that doesn't advance.
		if len(page.Data) == 0 || page.NextCursor == cursor {
			break
		}

		cursor = page.NextCursor
	}

	if counter != nil {
		counter.Finish()
	}

	return results, nil
}

// FetchCursorPages fetches one or more cursor-based pages using the shared
// pagination config. maxItems uses the same semantics as FetchAllPages:
// 0 means unlimited, values >0 cap the total returned items.
func FetchCursorPages[T any](ctx context.Context, limit int, maxItems int, fetcher PageFetcher[T]) ([]T, error) {
	config := DefaultPaginationConfig()
	config.PageSize = NormalizePageSize(limit)
	config.MaxItems = maxItems
	return FetchAllPages(ctx, config, fetcher)
}

// PaginationMode distinguishes between the three pagination behaviors.
type PaginationMode int

const (
	// PaginateSinglePage fetches one page only.
	PaginateSinglePage PaginationMode = iota
	// PaginateWithCap fetches multiple pages up to MaxItems.
	PaginateWithCap
	// PaginateAll fetches every page with no cap.
	PaginateAll
)

// PaginationLimits holds the resolved pagination parameters after applying
// auto-pagination logic. Use SetupPagination to compute these from user flags.
type PaginationLimits struct {
	Limit    int            // Per-page limit to pass to the API
	MaxItems int            // Total items to fetch (only meaningful when Mode == PaginateWithCap)
	Mode     PaginationMode // Which pagination behavior to use
}

// SetupPagination resolves pagination parameters from user-provided flags.
// When limit exceeds MaxAPILimit, it enables auto-pagination.
// When fetchAll is true, it fetches all items up to maxItems (0 = unlimited).
//
// This eliminates duplicate logic across list commands (contacts, email, etc.).
func SetupPagination(limit int, fetchAll bool, maxItems int) PaginationLimits {
	if fetchAll {
		if maxItems > 0 {
			return PaginationLimits{
				Limit:    MaxAPILimit,
				MaxItems: maxItems,
				Mode:     PaginateWithCap,
			}
		}
		return PaginationLimits{
			Limit: MaxAPILimit,
			Mode:  PaginateAll,
		}
	}
	if limit > MaxAPILimit {
		return PaginationLimits{
			Limit:    MaxAPILimit,
			MaxItems: limit,
			Mode:     PaginateWithCap,
		}
	}
	return PaginationLimits{
		Limit: limit,
		Mode:  PaginateSinglePage,
	}
}
