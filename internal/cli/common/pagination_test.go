//go:build !integration

package common

import (
	"context"
	"errors"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPageResult_HasMore(t *testing.T) {
	tests := []struct {
		name       string
		nextCursor string
		expected   bool
	}{
		{"with cursor", "abc123", true},
		{"empty cursor", "", false},
		{"whitespace only", "   ", true}, // whitespace is still a valid cursor
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			page := PageResult[string]{
				Data:       []string{"item1"},
				NextCursor: tt.nextCursor,
			}
			assert.Equal(t, tt.expected, page.HasMore())
		})
	}
}

func TestDefaultPaginationConfig(t *testing.T) {
	config := DefaultPaginationConfig()

	assert.Equal(t, 50, config.PageSize)
	assert.Equal(t, 0, config.MaxItems)
	assert.Equal(t, 0, config.MaxPages)
	assert.True(t, config.ShowProgress)
	assert.NotNil(t, config.Writer)
}

func TestNormalizePageSize(t *testing.T) {
	tests := []struct {
		name     string
		limit    int
		expected int
	}{
		{"keeps small values", 50, 50},
		{"clamps to API max", 500, MaxAPILimit},
		{"zero falls back to API max", 0, MaxAPILimit},
		{"negative falls back to API max", -5, MaxAPILimit},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, NormalizePageSize(tt.limit))
		})
	}
}

func TestFetchAllPages_SinglePage(t *testing.T) {
	SetQuiet(true) // Quiet mode to avoid progress output

	config := DefaultPaginationConfig()
	config.ShowProgress = false

	fetcherCalls := 0
	fetcher := func(ctx context.Context, cursor string) (PageResult[string], error) {
		fetcherCalls++
		return PageResult[string]{
			Data:       []string{"item1", "item2", "item3"},
			NextCursor: "", // No more pages
		}, nil
	}

	results, err := FetchAllPages(context.Background(), config, fetcher)

	require.NoError(t, err)
	assert.Equal(t, 3, len(results))
	assert.Equal(t, 1, fetcherCalls)
	assert.Contains(t, results, "item1")
	assert.Contains(t, results, "item2")
	assert.Contains(t, results, "item3")
}

func TestFetchAllPages_MultiplePages(t *testing.T) {
	SetQuiet(true)

	config := DefaultPaginationConfig()
	config.ShowProgress = false

	fetcherCalls := 0
	fetcher := func(ctx context.Context, cursor string) (PageResult[string], error) {
		fetcherCalls++

		switch cursor {
		case "":
			return PageResult[string]{
				Data:       []string{"page1-item1", "page1-item2"},
				NextCursor: "cursor1",
			}, nil
		case "cursor1":
			return PageResult[string]{
				Data:       []string{"page2-item1", "page2-item2"},
				NextCursor: "cursor2",
			}, nil
		case "cursor2":
			return PageResult[string]{
				Data:       []string{"page3-item1"},
				NextCursor: "", // Last page
			}, nil
		default:
			return PageResult[string]{}, errors.New("unexpected cursor")
		}
	}

	results, err := FetchAllPages(context.Background(), config, fetcher)

	require.NoError(t, err)
	assert.Equal(t, 5, len(results))
	assert.Equal(t, 3, fetcherCalls)
}

func TestFetchAllPages_MaxItems(t *testing.T) {
	SetQuiet(true)

	config := DefaultPaginationConfig()
	config.ShowProgress = false
	config.MaxItems = 3 // Limit to 3 items

	fetcher := func(ctx context.Context, cursor string) (PageResult[string], error) {
		return PageResult[string]{
			Data:       []string{"item1", "item2", "item3", "item4", "item5"},
			NextCursor: "more",
		}, nil
	}

	results, err := FetchAllPages(context.Background(), config, fetcher)

	require.NoError(t, err)
	assert.Equal(t, 3, len(results))
}

func TestFetchAllPages_MaxPages(t *testing.T) {
	SetQuiet(true)

	config := DefaultPaginationConfig()
	config.ShowProgress = false
	config.MaxPages = 2 // Limit to 2 pages

	fetcherCalls := 0
	fetcher := func(ctx context.Context, cursor string) (PageResult[string], error) {
		fetcherCalls++
		return PageResult[string]{
			Data:       []string{"item"},
			NextCursor: "more", // Always has more
		}, nil
	}

	results, err := FetchAllPages(context.Background(), config, fetcher)

	require.NoError(t, err)
	assert.Equal(t, 2, fetcherCalls)
	assert.Equal(t, 2, len(results))
}

func TestFetchAllPages_FetcherError(t *testing.T) {
	SetQuiet(true)

	config := DefaultPaginationConfig()
	config.ShowProgress = false

	fetcherCalls := 0
	fetcher := func(ctx context.Context, cursor string) (PageResult[string], error) {
		fetcherCalls++
		if fetcherCalls == 2 {
			return PageResult[string]{}, errors.New("fetch failed")
		}
		return PageResult[string]{
			Data:       []string{"item"},
			NextCursor: "more",
		}, nil
	}

	results, err := FetchAllPages(context.Background(), config, fetcher)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to fetch page 2")
	assert.Equal(t, 1, len(results)) // Should have partial results
}

func TestFetchAllPages_ContextCancellation(t *testing.T) {
	SetQuiet(true)

	config := DefaultPaginationConfig()
	config.ShowProgress = false

	ctx, cancel := context.WithCancel(context.Background())

	fetcherCalls := 0
	fetcher := func(ctx context.Context, cursor string) (PageResult[string], error) {
		fetcherCalls++
		if fetcherCalls == 2 {
			cancel() // Cancel context on second call
		}
		return PageResult[string]{
			Data:       []string{"item"},
			NextCursor: "more-" + strconv.Itoa(fetcherCalls), // Advancing cursor
		}, nil
	}

	results, err := FetchAllPages(ctx, config, fetcher)

	assert.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled)
	assert.Equal(t, 2, len(results)) // Should have results from completed pages
}

func TestFetchAllPages_EmptyFirstPage(t *testing.T) {
	SetQuiet(true)

	config := DefaultPaginationConfig()
	config.ShowProgress = false

	fetcher := func(ctx context.Context, cursor string) (PageResult[string], error) {
		return PageResult[string]{
			Data:       []string{},
			NextCursor: "",
		}, nil
	}

	results, err := FetchAllPages(context.Background(), config, fetcher)

	require.NoError(t, err)
	assert.Empty(t, results)
}

func TestFetchCursorPages(t *testing.T) {
	SetQuiet(true)

	t.Run("caps results across multiple pages", func(t *testing.T) {
		fetcher := func(ctx context.Context, cursor string) (PageResult[string], error) {
			switch cursor {
			case "":
				return PageResult[string]{
					Data:       []string{"a", "b"},
					NextCursor: "next",
				}, nil
			case "next":
				return PageResult[string]{
					Data: []string{"c", "d"},
				}, nil
			default:
				return PageResult[string]{}, errors.New("unexpected cursor")
			}
		}

		results, err := FetchCursorPages(context.Background(), 500, 3, fetcher)

		require.NoError(t, err)
		assert.Equal(t, []string{"a", "b", "c"}, results)
	})
}

func TestFetchAllPages_WithProgress(t *testing.T) {
	SetQuiet(false) // Not quiet

	config := DefaultPaginationConfig()
	config.ShowProgress = true

	fetcherCalls := 0
	fetcher := func(ctx context.Context, cursor string) (PageResult[string], error) {
		fetcherCalls++
		if fetcherCalls > 2 {
			return PageResult[string]{
				Data:       []string{"item"},
				NextCursor: "",
			}, nil
		}
		return PageResult[string]{
			Data:       []string{"item1", "item2"},
			NextCursor: "more-" + strconv.Itoa(fetcherCalls), // Advancing cursor
		}, nil
	}

	results, err := FetchAllPages(context.Background(), config, fetcher)

	require.NoError(t, err)
	assert.Equal(t, 5, len(results))
	assert.Equal(t, 3, fetcherCalls)
}

func TestFetchAllPages_ContextDeadline(t *testing.T) {
	SetQuiet(true)

	config := DefaultPaginationConfig()
	config.ShowProgress = false

	// Create context with very short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()

	// Wait for context to expire
	time.Sleep(5 * time.Millisecond)

	fetcher := func(ctx context.Context, cursor string) (PageResult[string], error) {
		return PageResult[string]{
			Data:       []string{"item"},
			NextCursor: "more",
		}, nil
	}

	results, err := FetchAllPages(ctx, config, fetcher)

	assert.Error(t, err)
	assert.ErrorIs(t, err, context.DeadlineExceeded)
	assert.Empty(t, results)
}

func TestSetupPagination(t *testing.T) {
	tests := []struct {
		name         string
		limit        int
		fetchAll     bool
		maxItems     int
		wantLimit    int
		wantMaxItems int
		wantMode     PaginationMode
	}{
		{
			name:      "small limit, single page",
			limit:     50,
			fetchAll:  false,
			maxItems:  0,
			wantLimit: 50,
			wantMode:  PaginateSinglePage,
		},
		{
			name:      "limit at API max, single page",
			limit:     MaxAPILimit,
			fetchAll:  false,
			maxItems:  0,
			wantLimit: MaxAPILimit,
			wantMode:  PaginateSinglePage,
		},
		{
			name:         "limit exceeds API max, auto-paginate with cap",
			limit:        500,
			fetchAll:     false,
			maxItems:     0,
			wantLimit:    MaxAPILimit,
			wantMaxItems: 500,
			wantMode:     PaginateWithCap,
		},
		{
			name:      "fetchAll with no max fetches unlimited",
			limit:     50,
			fetchAll:  true,
			maxItems:  0,
			wantLimit: MaxAPILimit,
			wantMode:  PaginateAll,
		},
		{
			name:         "fetchAll with max items caps",
			limit:        50,
			fetchAll:     true,
			maxItems:     1000,
			wantLimit:    MaxAPILimit,
			wantMaxItems: 1000,
			wantMode:     PaginateWithCap,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SetupPagination(tt.limit, tt.fetchAll, tt.maxItems)
			assert.Equal(t, tt.wantLimit, result.Limit, "Limit mismatch")
			assert.Equal(t, tt.wantMaxItems, result.MaxItems, "MaxItems mismatch")
			assert.Equal(t, tt.wantMode, result.Mode, "Mode mismatch")
		})
	}
}

func TestPaginationMode(t *testing.T) {
	tests := []struct {
		name string
		mode PaginationMode
		desc string
	}{
		{"single page is 0", PaginateSinglePage, "default zero value"},
		{"with cap is 1", PaginateWithCap, "explicit cap"},
		{"all is 2", PaginateAll, "unlimited"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Verify the constants are distinct
			assert.NotEqual(t, PaginateSinglePage, PaginateWithCap)
			assert.NotEqual(t, PaginateWithCap, PaginateAll)
			assert.NotEqual(t, PaginateSinglePage, PaginateAll)
		})
	}
}

func TestFetchAllPages_StuckCursor(t *testing.T) {
	SetQuiet(true)

	config := DefaultPaginationConfig()
	config.ShowProgress = false

	fetcherCalls := 0
	fetcher := func(ctx context.Context, cursor string) (PageResult[string], error) {
		fetcherCalls++
		// Buggy server: always claims more results with a cursor that never advances.
		return PageResult[string]{
			Data:       []string{"item-" + strconv.Itoa(fetcherCalls)},
			NextCursor: "stuck-cursor",
		}, nil
	}

	results, err := FetchAllPages(context.Background(), config, fetcher)

	require.NoError(t, err)
	// Page 1 uses cursor "", page 2 uses "stuck-cursor" and returns the same
	// cursor again — pagination must stop instead of looping forever.
	assert.Equal(t, 2, fetcherCalls)
	assert.Len(t, results, 2)
}

func TestFetchAllPages_EmptyPageClaimingMore(t *testing.T) {
	SetQuiet(true)

	config := DefaultPaginationConfig()
	config.ShowProgress = false

	fetcherCalls := 0
	fetcher := func(ctx context.Context, cursor string) (PageResult[string], error) {
		fetcherCalls++
		// Buggy server: returns no items but keeps advancing the cursor,
		// claiming there is always more data.
		return PageResult[string]{
			Data:       nil,
			NextCursor: "cursor-" + strconv.Itoa(fetcherCalls),
		}, nil
	}

	results, err := FetchAllPages(context.Background(), config, fetcher)

	require.NoError(t, err)
	// An empty page that claims more results must terminate pagination.
	assert.Equal(t, 1, fetcherCalls)
	assert.Empty(t, results)
}
