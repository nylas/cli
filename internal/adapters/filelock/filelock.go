// Package filelock implements ports.CrossProcessLock with an advisory lock on
// a file: flock(2) on Unix, LockFileEx on Windows.
//
// Both are released by the kernel when the holding process exits, including
// when it crashes, so a killed `nylas mcp serve` cannot wedge the others.
package filelock

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// defaultPollInterval is how often a waiter retries. Acquisition is
// non-blocking plus polling, rather than a blocking syscall, because a
// blocking flock cannot be cancelled by a context.
const defaultPollInterval = 20 * time.Millisecond

// Lock is a cross-process lock backed by the file at path.
type Lock struct {
	path string
	poll time.Duration
}

// New returns a lock on path. The file and its directory are created on
// first use; the file's content is never read or written.
func New(path string) *Lock {
	return &Lock{path: path, poll: defaultPollInterval}
}

// Lock acquires the lock, waiting until ctx is done.
func (l *Lock) Lock(ctx context.Context) (func() error, error) {
	if err := os.MkdirAll(filepath.Dir(l.path), 0o700); err != nil {
		return nil, fmt.Errorf("failed to create lock directory: %w", err)
	}
	// #nosec G304 -- the path is built by the caller from the CLI's own config dir.
	file, err := os.OpenFile(l.path, os.O_RDWR|os.O_CREATE, 0o600)
	if err != nil {
		return nil, fmt.Errorf("failed to open lock file: %w", err)
	}

	for {
		acquired, err := tryLock(file)
		if err != nil {
			_ = file.Close()
			return nil, fmt.Errorf("failed to lock %s: %w", l.path, err)
		}
		if acquired {
			return unlocker(file), nil
		}

		timer := time.NewTimer(l.poll)
		select {
		case <-ctx.Done():
			timer.Stop()
			_ = file.Close()
			return nil, fmt.Errorf("waiting for lock %s: %w", l.path, ctx.Err())
		case <-timer.C:
		}
	}
}

func unlocker(file *os.File) func() error {
	released := false
	return func() error {
		if released {
			return errors.New("lock already released")
		}
		released = true
		unlockErr := unlockFile(file)
		closeErr := file.Close()
		return errors.Join(unlockErr, closeErr)
	}
}
