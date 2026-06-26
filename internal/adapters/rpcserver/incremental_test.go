package rpcserver

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"
)

func TestIntervalController_Current(t *testing.T) {
	fast := 5 * time.Second
	idle := 30 * time.Second
	ctrl := NewIntervalController(fast, idle)
	if got := ctrl.Current(); got != idle {
		t.Fatalf("Current() = %v, want default idle %v", got, idle)
	}

	tests := []struct {
		name    string
		focused bool
		want    time.Duration
	}{
		{name: "fast when focused", focused: true, want: fast},
		{name: "idle after focus clears", focused: false, want: idle},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl.SetFocused(tt.focused)
			if got := ctrl.Current(); got != tt.want {
				t.Fatalf("Current() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIntervalController_CurrentConcurrentAccess(t *testing.T) {
	ctrl := NewIntervalController(time.Millisecond, time.Second)
	start := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		<-start
		for i := range 10_000 {
			ctrl.SetFocused(i%2 == 0)
		}
	}()

	go func() {
		defer wg.Done()
		<-start
		for range 10_000 {
			_ = ctrl.Current()
		}
	}()

	close(start)
	wg.Wait()
}

func TestRunAdaptive_PollsReportsErrorsAndReturnsContextError(t *testing.T) {
	ctrl := NewIntervalController(time.Millisecond, time.Millisecond)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	pollErr := errors.New("poll failed")
	var calls int
	var gotErrs []error

	err := RunAdaptive(ctx, ctrl, func(err error) {
		gotErrs = append(gotErrs, err)
	}, func(ctx context.Context) error {
		calls++
		if calls == 1 {
			return pollErr
		}
		cancel()
		return nil
	})

	if !errors.Is(err, context.Canceled) {
		t.Fatalf("RunAdaptive() error = %v, want %v", err, context.Canceled)
	}
	if calls < 2 {
		t.Fatalf("pollOnce calls = %d, want at least 2", calls)
	}
	if len(gotErrs) != 1 || !errors.Is(gotErrs[0], pollErr) {
		t.Fatalf("onError calls = %v, want %v", gotErrs, pollErr)
	}
}

func TestIntervalController_SetIntervals(t *testing.T) {
	ctrl := NewIntervalController(5*time.Second, 30*time.Second)

	// Non-positive values leave the corresponding bound unchanged.
	ctrl.SetIntervals(2*time.Second, 0)
	if fast, idle := ctrl.Intervals(); fast != 2*time.Second || idle != 30*time.Second {
		t.Fatalf("Intervals() = (%v, %v), want (2s, 30s)", fast, idle)
	}

	ctrl.SetIntervals(0, time.Minute)
	if fast, idle := ctrl.Intervals(); fast != 2*time.Second || idle != time.Minute {
		t.Fatalf("Intervals() = (%v, %v), want (2s, 1m)", fast, idle)
	}

	// The live value follows focus state after an update.
	ctrl.SetFocused(true)
	if got := ctrl.Current(); got != 2*time.Second {
		t.Fatalf("Current() = %v, want 2s", got)
	}
}

func TestRegisterPollConfigHandler(t *testing.T) {
	ctrl := NewIntervalController(5*time.Second, 30*time.Second)
	contactCtrl := NewIntervalController(60*time.Second, 60*time.Second)
	d := NewDispatcher()
	RegisterPollConfigHandler(d, ctrl, contactCtrl)

	t.Run("updates provided fields and reports effective values", func(t *testing.T) {
		got := d.Dispatch(context.Background(),
			[]byte(`{"jsonrpc":"2.0","id":1,"method":"client.pollConfig","params":{"fast":"2s","contacts":"90s"}}`))

		var resp struct {
			Result pollConfigResult `json:"result"`
			Error  *RPCError        `json:"error"`
		}
		if err := json.Unmarshal(got, &resp); err != nil {
			t.Fatalf("unmarshal response: %v", err)
		}
		if resp.Error != nil {
			t.Fatalf("unexpected error: %+v", resp.Error)
		}
		// fast changed, idle left untouched, contacts changed.
		if resp.Result.Fast != "2s" || resp.Result.Idle != "30s" || resp.Result.Contacts != "1m30s" {
			t.Fatalf("result = %+v, want {2s 30s 1m30s}", resp.Result)
		}
		if fast, _ := ctrl.Intervals(); fast != 2*time.Second {
			t.Fatalf("controller fast = %v, want 2s", fast)
		}
		if c, _ := contactCtrl.Intervals(); c != 90*time.Second {
			t.Fatalf("contact interval = %v, want 90s", c)
		}
	})

	t.Run("rejects an invalid duration", func(t *testing.T) {
		got := d.Dispatch(context.Background(),
			[]byte(`{"jsonrpc":"2.0","id":2,"method":"client.pollConfig","params":{"idle":"nope"}}`))

		var resp struct {
			Error *RPCError `json:"error"`
		}
		if err := json.Unmarshal(got, &resp); err != nil {
			t.Fatalf("unmarshal response: %v", err)
		}
		if resp.Error == nil || resp.Error.Code != InvalidParams {
			t.Fatalf("error = %+v, want InvalidParams", resp.Error)
		}
	})

	t.Run("rejects a non-positive duration", func(t *testing.T) {
		got := d.Dispatch(context.Background(),
			[]byte(`{"jsonrpc":"2.0","id":3,"method":"client.pollConfig","params":{"fast":"0s"}}`))

		var resp struct {
			Error *RPCError `json:"error"`
		}
		if err := json.Unmarshal(got, &resp); err != nil {
			t.Fatalf("unmarshal response: %v", err)
		}
		if resp.Error == nil || resp.Error.Code != InvalidParams {
			t.Fatalf("error = %+v, want InvalidParams", resp.Error)
		}
	})
}

func TestRegisterFocusHandler(t *testing.T) {
	fast := time.Millisecond
	idle := time.Second
	ctrl := NewIntervalController(fast, idle)
	d := NewDispatcher()
	RegisterFocusHandler(d, ctrl)

	got := d.Dispatch(context.Background(), []byte(`{"jsonrpc":"2.0","method":"client.focus","params":{"focused":true}}`))
	if got != nil {
		t.Fatalf("Dispatch() = %s, want nil", got)
	}
	if current := ctrl.Current(); current != fast {
		t.Fatalf("Current() = %v, want %v", current, fast)
	}
}
