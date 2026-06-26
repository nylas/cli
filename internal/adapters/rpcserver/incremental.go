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

// SetIntervals updates the focused/idle durations live. Non-positive values are
// ignored, so a caller can change one bound without touching the other.
func (c *IntervalController) SetIntervals(fast, idle time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if fast > 0 {
		c.fast = fast
	}
	if idle > 0 {
		c.idle = idle
	}
}

// Intervals returns the current focused and idle durations.
func (c *IntervalController) Intervals() (fast, idle time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.fast, c.idle
}

type incrementalState struct {
	cursor      int64
	boundaryIDs map[string]struct{}
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

type pollConfigParams struct {
	Fast     string `json:"fast,omitempty"`
	Idle     string `json:"idle,omitempty"`
	Contacts string `json:"contacts,omitempty"`
}

type pollConfigResult struct {
	Fast     string `json:"fast"`
	Idle     string `json:"idle"`
	Contacts string `json:"contacts"`
}

// RegisterPollConfigHandler exposes client.pollConfig, which reads and optionally
// updates the live polling intervals. Durations use Go syntax (e.g. "2s", "1m");
// omitted or empty fields are left unchanged. The result always reports the
// effective values. ctrl drives messages/threads/events (focused/idle), and
// contactCtrl drives contacts (a single interval, so focused == idle).
func RegisterPollConfigHandler(d *Dispatcher, ctrl, contactCtrl *IntervalController) {
	d.Register("client.pollConfig", func(_ context.Context, params json.RawMessage) (any, error) {
		var p pollConfigParams
		if err := decodeParams(params, &p); err != nil {
			return nil, err
		}

		fast, err := parsePollInterval("fast", p.Fast)
		if err != nil {
			return nil, err
		}
		idle, err := parsePollInterval("idle", p.Idle)
		if err != nil {
			return nil, err
		}
		contacts, err := parsePollInterval("contacts", p.Contacts)
		if err != nil {
			return nil, err
		}

		ctrl.SetIntervals(fast, idle)
		contactCtrl.SetIntervals(contacts, contacts)

		effFast, effIdle := ctrl.Intervals()
		effContacts, _ := contactCtrl.Intervals()
		return pollConfigResult{
			Fast:     effFast.String(),
			Idle:     effIdle.String(),
			Contacts: effContacts.String(),
		}, nil
	})
}

// parsePollInterval returns 0 for an empty value (meaning "leave unchanged").
// A non-empty value must parse as a positive Go duration.
func parsePollInterval(field, value string) (time.Duration, error) {
	if value == "" {
		return 0, nil
	}
	d, err := time.ParseDuration(value)
	if err != nil {
		return 0, NewRPCError(InvalidParams, field+` must be a duration (e.g. "2s", "1m")`, err.Error())
	}
	if d <= 0 {
		return 0, NewRPCError(InvalidParams, field+" must be positive", nil)
	}
	return d, nil
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
