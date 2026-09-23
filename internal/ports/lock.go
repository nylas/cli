package ports

import "context"

// CrossProcessLock is a mutual-exclusion lock that holds across OS processes
// on one machine, not only across goroutines.
type CrossProcessLock interface {
	// Lock blocks until the lock is held or ctx is done. The returned
	// function releases it and must be called exactly once.
	Lock(ctx context.Context) (unlock func() error, err error)
}
