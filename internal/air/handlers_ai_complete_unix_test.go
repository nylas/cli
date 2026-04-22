//go:build !windows

package air

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

func TestRunCommandOutput_KillsChildProcessGroupOnCancel(t *testing.T) {
	t.Parallel()

	pidFile := t.TempDir() + "/child.pid"
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cmd := exec.Command("sh", "-c", `sleep 30 & echo $! > "$CHILD_PID_FILE"; wait`)
	cmd.Env = append(os.Environ(), "CHILD_PID_FILE="+pidFile)

	doneCh := make(chan error, 1)
	go func() {
		_, err := runCommandOutput(ctx, cmd)
		doneCh <- err
	}()

	childPID := waitForChildPID(t, pidFile)
	cancel()

	select {
	case err := <-doneCh:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("expected context cancellation, got %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for command cancellation")
	}

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if err := syscall.Kill(childPID, 0); errors.Is(err, syscall.ESRCH) {
			return
		}
		time.Sleep(25 * time.Millisecond)
	}

	t.Fatalf("expected child process %d to be terminated with its parent process group", childPID)
}

func TestRunCommandOutput_DoesNotStartWhenContextAlreadyCanceled(t *testing.T) {
	t.Parallel()

	sentinel := t.TempDir() + "/started"
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	cmd := exec.Command("sh", "-c", `touch "$SENTINEL"`)
	cmd.Env = append(os.Environ(), "SENTINEL="+sentinel)

	_, err := runCommandOutput(ctx, cmd)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context cancellation, got %v", err)
	}

	if _, statErr := os.Stat(sentinel); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("expected command not to start, stat error=%v", statErr)
	}
}

func waitForChildPID(t *testing.T, pidFile string) int {
	t.Helper()

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		data, err := os.ReadFile(pidFile)
		if err == nil && len(strings.TrimSpace(string(data))) > 0 {
			pid, convErr := strconv.Atoi(strings.TrimSpace(string(data)))
			if convErr != nil {
				t.Fatalf("parse child pid: %v", convErr)
			}
			return pid
		}
		time.Sleep(25 * time.Millisecond)
	}

	t.Fatalf("timed out waiting for child pid file %s", pidFile)
	return 0
}
