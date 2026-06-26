package rpcserver

import (
	"context"
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
