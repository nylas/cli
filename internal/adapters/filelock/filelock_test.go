//go:build !integration

package filelock

import (
	"bufio"
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const helperEnv = "NYLAS_FILELOCK_HELPER_PATH"

// TestMain lets the test binary double as a separate process that holds the
// lock, because the property that matters is exclusion between processes.
func TestMain(m *testing.M) {
	if path := os.Getenv(helperEnv); path != "" {
		holdLockUntilStdinCloses(path)
		return
	}
	os.Exit(m.Run())
}

func holdLockUntilStdinCloses(path string) {
	unlock, err := New(path).Lock(context.Background())
	if err != nil {
		_, _ = os.Stdout.WriteString("error: " + err.Error() + "\n")
		os.Exit(2)
	}
	_, _ = os.Stdout.WriteString("locked\n")
	_, _ = bufio.NewReader(os.Stdin).ReadString('\n')
	_ = unlock()
	os.Exit(0)
}

// startHolder runs a child process that holds the lock until its stdin closes.
func startHolder(t *testing.T, path string) (*exec.Cmd, func()) {
	t.Helper()

	// #nosec G204 -- re-executes this test binary.
	cmd := exec.Command(os.Args[0], "-test.run=^$")
	cmd.Env = append(os.Environ(), helperEnv+"="+path)
	stdin, err := cmd.StdinPipe()
	require.NoError(t, err)
	stdout, err := cmd.StdoutPipe()
	require.NoError(t, err)
	require.NoError(t, cmd.Start())

	line, err := bufio.NewReader(stdout).ReadString('\n')
	require.NoError(t, err)
	require.Equal(t, "locked\n", line)

	release := func() {
		_ = stdin.Close()
		_ = cmd.Wait()
	}
	t.Cleanup(func() {
		_ = stdin.Close()
		if cmd.ProcessState == nil {
			_ = cmd.Process.Kill()
			_ = cmd.Wait()
		}
	})
	return cmd, release
}

func tryLockFor(path string, d time.Duration) (func() error, error) {
	ctx, cancel := context.WithTimeout(context.Background(), d)
	defer cancel()
	return New(path).Lock(ctx)
}

func TestLock_ExcludesAnotherProcess(t *testing.T) {
	path := filepath.Join(t.TempDir(), "refresh.lock")
	_, release := startHolder(t, path)

	_, err := tryLockFor(path, 150*time.Millisecond)
	require.ErrorIs(t, err, context.DeadlineExceeded, "the lock is held by the other process")

	release()

	unlock, err := tryLockFor(path, 2*time.Second)
	require.NoError(t, err, "released by the other process")
	require.NoError(t, unlock())
}

func TestLock_IsReleasedWhenTheHolderIsKilled(t *testing.T) {
	// A crashed `nylas mcp serve` must not sign every other process out by
	// wedging the refresh lock forever.
	path := filepath.Join(t.TempDir(), "refresh.lock")
	cmd, _ := startHolder(t, path)

	require.NoError(t, cmd.Process.Kill())
	_ = cmd.Wait()

	unlock, err := tryLockFor(path, 2*time.Second)
	require.NoError(t, err)
	require.NoError(t, unlock())
}

func TestLock_ExcludesWithinOneProcess(t *testing.T) {
	path := filepath.Join(t.TempDir(), "refresh.lock")

	var inside, maxInside atomic.Int32
	var wg sync.WaitGroup
	for range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			unlock, err := New(path).Lock(context.Background())
			if !assert.NoError(t, err) {
				return
			}
			now := inside.Add(1)
			for {
				prev := maxInside.Load()
				if now <= prev || maxInside.CompareAndSwap(prev, now) {
					break
				}
			}
			time.Sleep(5 * time.Millisecond)
			inside.Add(-1)
			assert.NoError(t, unlock())
		}()
	}
	wg.Wait()

	assert.Equal(t, int32(1), maxInside.Load(), "never more than one holder at a time")
}

func TestLock_CreatesDirectoryAndPrivateFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "dir", "refresh.lock")

	unlock, err := New(path).Lock(context.Background())
	require.NoError(t, err)
	t.Cleanup(func() { _ = unlock() })

	info, err := os.Stat(path)
	require.NoError(t, err)
	if filepath.Separator == '/' {
		assert.Equal(t, os.FileMode(0o600), info.Mode().Perm())
	}
}

func TestLock_UnlockTwiceIsAnError(t *testing.T) {
	unlock, err := New(filepath.Join(t.TempDir(), "refresh.lock")).Lock(context.Background())
	require.NoError(t, err)

	require.NoError(t, unlock())
	assert.Error(t, unlock())
}

func TestLock_HonoursAlreadyCancelledContext(t *testing.T) {
	path := filepath.Join(t.TempDir(), "refresh.lock")
	holder, err := New(path).Lock(context.Background())
	require.NoError(t, err)
	t.Cleanup(func() { _ = holder() })

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = New(path).Lock(ctx)

	require.True(t, errors.Is(err, context.Canceled))
}
