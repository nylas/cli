//go:build !integration

package common

import (
	"bytes"
	"context"
	"errors"
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

func TestFetchAllPages_SinglePage(t *testing.T) {
	ResetLogger()
	InitLogger(false, true) // Quiet mode to avoid progress output

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
	ResetLogger()
	InitLogger(false, true)

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
	ResetLogger()
	InitLogger(false, true)

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
	ResetLogger()
	InitLogger(false, true)

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
	ResetLogger()
	InitLogger(false, true)

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
	ResetLogger()
	InitLogger(false, true)

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
			NextCursor: "more",
		}, nil
	}

	results, err := FetchAllPages(ctx, config, fetcher)

	assert.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled)
	assert.Equal(t, 2, len(results)) // Should have results from completed pages
}

func TestFetchAllPages_EmptyFirstPage(t *testing.T) {
	ResetLogger()
	InitLogger(false, true)

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

func TestFetchAllWithProgress(t *testing.T) {
	ResetLogger()
	InitLogger(false, true)

	fetcher := func(ctx context.Context, cursor string) (PageResult[string], error) {
		return PageResult[string]{
			Data:       []string{"a", "b", "c"},
			NextCursor: "",
		}, nil
	}

	results, err := FetchAllWithProgress(context.Background(), fetcher, 0)

	require.NoError(t, err)
	assert.Equal(t, 3, len(results))
}

func TestFetchAllWithProgress_WithMaxItems(t *testing.T) {
	ResetLogger()
	InitLogger(false, true)

	fetcher := func(ctx context.Context, cursor string) (PageResult[string], error) {
		return PageResult[string]{
			Data:       []string{"a", "b", "c", "d", "e"},
			NextCursor: "more",
		}, nil
	}

	results, err := FetchAllWithProgress(context.Background(), fetcher, 2)

	require.NoError(t, err)
	assert.Equal(t, 2, len(results))
}

func TestPaginatedDisplay_Operations(t *testing.T) {
	ResetLogger()
	InitLogger(false, false) // Not quiet for display output

	t.Run("display page", func(t *testing.T) {
		var buf bytes.Buffer
		display := NewPaginatedDisplay(10).SetWriter(&buf)

		display.DisplayPage(5, true)

		assert.Equal(t, 1, display.CurrentPage)
		assert.Equal(t, 5, display.TotalFetched)
		assert.Contains(t, buf.String(), "Page 1")
	})

	t.Run("display multiple pages", func(t *testing.T) {
		var buf bytes.Buffer
		display := NewPaginatedDisplay(10).SetWriter(&buf)

		display.DisplayPage(10, true)
		display.DisplayPage(10, true)
		display.DisplayPage(5, false) // Last page

		assert.Equal(t, 3, display.CurrentPage)
		assert.Equal(t, 25, display.TotalFetched)
	})

	t.Run("display summary", func(t *testing.T) {
		var buf bytes.Buffer
		display := NewPaginatedDisplay(10).SetWriter(&buf)

		display.DisplayPage(10, true)
		display.DisplayPage(5, false)
		display.DisplaySummary()

		output := buf.String()
		assert.Contains(t, output, "15 items")
		assert.Contains(t, output, "2 pages")
	})

	t.Run("no summary for single page", func(t *testing.T) {
		var buf bytes.Buffer
		display := NewPaginatedDisplay(10).SetWriter(&buf)

		display.DisplayPage(5, false) // Single page, no more
		buf.Reset()
		display.DisplaySummary()

		// Summary should not be shown for single page
		assert.Empty(t, buf.String())
	})
}

func TestPaginatedDisplay_QuietMode(t *testing.T) {
	ResetLogger()
	InitLogger(false, true) // Quiet mode

	var buf bytes.Buffer
	display := NewPaginatedDisplay(10).SetWriter(&buf)

	display.DisplayPage(10, true)
	display.DisplayPage(5, false)
	display.DisplaySummary()

	// In quiet mode, should not produce output
	assert.Empty(t, buf.String())
}

func TestFetchAllPages_WithProgress(t *testing.T) {
	ResetLogger()
	InitLogger(false, false) // Not quiet

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
			NextCursor: "more",
		}, nil
	}

	results, err := FetchAllPages(context.Background(), config, fetcher)

	require.NoError(t, err)
	assert.Equal(t, 5, len(results))
	assert.Equal(t, 3, fetcherCalls)
}

func TestFetchAllPages_ContextDeadline(t *testing.T) {
	ResetLogger()
	InitLogger(false, true)

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
