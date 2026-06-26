package rpcserver

import (
	"cmp"
	"context"
	"encoding/json"
	"slices"
	"sync"
	"time"
)

type IntervalController struct {
	mu      sync.Mutex
	fast    time.Duration
	idle    time.Duration
	focused bool
}

func NewIntervalController(fast, idle time.Duration) *IntervalController {
	return &IntervalController{fast: fast, idle: idle}
}

func (c *IntervalController) SetFocused(focused bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.focused = focused
}

func (c *IntervalController) Current() time.Duration {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.focused {
		return c.fast
	}
	return c.idle
}

type incrementalState struct {
	cursor      int64
	boundaryIDs map[string]struct{}
}

func runTicker(ctx context.Context, interval time.Duration, onError func(error), pollOnce func(context.Context) error) error {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if err := pollOnce(ctx); err != nil && onError != nil {
				onError(err)
			}
		}
	}
}

func RunAdaptive(ctx context.Context, ctrl *IntervalController, onError func(error), pollOnce func(context.Context) error) error {
	for {
		timer := time.NewTimer(ctrl.Current())
		select {
		case <-ctx.Done():
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
			return ctx.Err()
		case <-timer.C:
		}

		if err := pollOnce(ctx); err != nil && onError != nil {
			onError(err)
		}
	}
}

func RegisterFocusHandler(d *Dispatcher, ctrl *IntervalController) {
	d.Register("client.focus", func(ctx context.Context, params json.RawMessage) (any, error) {
		var p struct {
			Focused bool `json:"focused"`
		}
		if err := decodeParams(params, &p); err != nil {
			return nil, err
		}
		ctrl.SetFocused(p.Focused)
		return nil, nil
	})
}

func pollIncremental[T any](
	ctx context.Context,
	st *incrementalState,
	fetch func(context.Context, int64) ([]T, error),
	tsOf func(T) int64,
	idOf func(T) string,
	method string,
	payloadOf func(T) any,
	notify NotifyFunc,
) error {
	startCursor := st.cursor
	queryAfter := startCursor
	if queryAfter > 0 {
		// The API filter is exclusive, so query cursor-1 and dedupe by id at the boundary second.
		queryAfter--
	}

	rows, err := fetch(ctx, queryAfter)
	if err != nil {
		return err
	}

	maxCursor := startCursor
	if len(rows) > 0 {
		maxRow := slices.MaxFunc(rows, func(a, b T) int {
			return cmp.Compare(tsOf(a), tsOf(b))
		})
		maxCursor = max(maxCursor, tsOf(maxRow))
	}

	nextBoundaryIDs := make(map[string]struct{})
	if maxCursor == startCursor {
		for id := range st.boundaryIDs {
			nextBoundaryIDs[id] = struct{}{}
		}
	}
	for _, row := range rows {
		if tsOf(row) == maxCursor {
			nextBoundaryIDs[idOf(row)] = struct{}{}
		}
	}

	slices.SortStableFunc(rows, func(a, b T) int {
		if ts := cmp.Compare(tsOf(a), tsOf(b)); ts != 0 {
			return ts
		}
		return cmp.Compare(idOf(a), idOf(b))
	})

	emitted := make(map[string]struct{})
	for _, row := range rows {
		ts := tsOf(row)
		id := idOf(row)
		if ts < startCursor {
			continue
		}
		if _, ok := emitted[id]; ok {
			continue
		}
		if ts == startCursor {
			if _, ok := st.boundaryIDs[id]; ok {
				continue
			}
		}

		if err := notify(method, payloadOf(row)); err != nil {
			return err
		}
		emitted[id] = struct{}{}
	}
	st.cursor = maxCursor
	st.boundaryIDs = nextBoundaryIDs
	return nil
}
