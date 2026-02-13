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

		cursor = page.NextCursor
	}

	if counter != nil {
		counter.Finish()
	}

	return results, nil
}

// FetchAllWithProgress fetches all pages and shows a progress indicator.
func FetchAllWithProgress[T any](ctx context.Context, fetcher PageFetcher[T], maxItems int) ([]T, error) {
	config := DefaultPaginationConfig()
	config.MaxItems = maxItems
	return FetchAllPages(ctx, config, fetcher)
}

// PaginatedDisplay handles displaying paginated results with optional streaming.
type PaginatedDisplay struct {
	PageSize     int
	CurrentPage  int
	TotalFetched int
	Writer       io.Writer
}

// NewPaginatedDisplay creates a new paginated display helper.
func NewPaginatedDisplay(pageSize int) *PaginatedDisplay {
	return &PaginatedDisplay{
		PageSize:    pageSize,
		CurrentPage: 0,
		Writer:      os.Stdout,
	}
}

// SetWriter sets the output writer.
func (p *PaginatedDisplay) SetWriter(w io.Writer) *PaginatedDisplay {
	p.Writer = w
	return p
}

// DisplayPage shows a summary after displaying items.
func (p *PaginatedDisplay) DisplayPage(itemsDisplayed int, hasMore bool) {
	p.CurrentPage++
	p.TotalFetched += itemsDisplayed

	if !IsQuiet() && hasMore {
		_, _ = fmt.Fprintf(p.Writer, "\n--- Page %d (%d items, %d total) ---\n",
			p.CurrentPage, itemsDisplayed, p.TotalFetched)
	}
}

// DisplaySummary shows a final summary.
func (p *PaginatedDisplay) DisplaySummary() {
	if !IsQuiet() && p.CurrentPage > 1 {
		_, _ = fmt.Fprintf(p.Writer, "\nFetched %d items across %d pages\n",
			p.TotalFetched, p.CurrentPage)
	}
}
