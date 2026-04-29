//go:build integration
// +build integration

package integration

import (
	"bytes"
	"context"
	"os/exec"
	"strings"
	"testing"
	"time"
)

// runCLI executes a CLI command and returns stdout, stderr, and error.
// NOTE: This does NOT apply rate limiting. For tests that make API calls,
// either call acquireRateLimit(t) before this, or use runCLIWithRateLimit.
func runCLI(args ...string) (string, string, error) {
	return runCLIWithTimeout(2*time.Minute, args...)
}

// runCLIWithTimeout executes a CLI command with a specified timeout.
// Use this for commands that might take a long time (e.g., AI/LLM calls).
func runCLIWithTimeout(timeout time.Duration, args ...string) (string, string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, testBinary, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	cmd.Env = cliTestEnv(nil)

	err := cmd.Run()
	return stdout.String(), stderr.String(), err
}

func runCLIWithOverrides(timeout time.Duration, envOverrides map[string]string, args ...string) (string, string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, testBinary, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	cmd.Env = cliTestEnv(envOverrides)

	err := cmd.Run()
	return stdout.String(), stderr.String(), err
}

func runCLIWithOverridesAndRateLimit(t *testing.T, timeout time.Duration, envOverrides map[string]string, args ...string) (string, string, error) {
	t.Helper()
	acquireRateLimit(t)
	return runCLIWithOverrides(timeout, envOverrides, args...)
}

// runCLIWithRateLimit executes a CLI command with rate limiting.
// Use this for commands that make API calls when running tests with t.Parallel().
// For offline commands (timezone, ai config, help), use runCLI directly.
func runCLIWithRateLimit(t *testing.T, args ...string) (string, string, error) {
	t.Helper()
	acquireRateLimit(t)
	return runCLI(args...)
}

// runCLIWithInput executes a CLI command with stdin input.
// NOTE: This does NOT apply rate limiting. For tests that make API calls,
// either call acquireRateLimit(t) before this, or use runCLIWithInputAndRateLimit.
func runCLIWithInput(input string, args ...string) (string, string, error) {
	cmd := exec.Command(testBinary, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	cmd.Stdin = strings.NewReader(input)

	// Build environment with all necessary variables
	cmd.Env = cliTestEnv(nil)

	err := cmd.Run()
	return stdout.String(), stderr.String(), err
}

// runCLIWithInputAndRateLimit executes a CLI command with stdin input and rate limiting.
// Use this for commands that make API calls when running tests with t.Parallel().
func runCLIWithInputAndRateLimit(t *testing.T, input string, args ...string) (string, string, error) {
	t.Helper()
	acquireRateLimit(t)
	return runCLIWithInput(input, args...)
}
