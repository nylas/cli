package common

import (
	"bytes"
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/nylas/cli/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// Logger Tests
// =============================================================================

func TestQuietMode(t *testing.T) {
	SetQuiet(false)
	defer SetQuiet(false)

	assert.False(t, IsQuiet())
	SetQuiet(true)
	assert.True(t, IsQuiet())
}

// =============================================================================
// Retry Tests
// =============================================================================

func TestRetry_Success(t *testing.T) {
	config := DefaultRetryConfig()
	attempts := 0

	err := WithRetry(context.Background(), config, func() error {
		attempts++
		return nil
	})

	assert.NoError(t, err)
	assert.Equal(t, 1, attempts)
}

func TestRetry_SuccessAfterRetries(t *testing.T) {
	config := RetryConfig{
		MaxRetries:  3,
		BaseDelay:   10 * time.Millisecond,
		MaxDelay:    100 * time.Millisecond,
		Multiplier:  2.0,
		JitterRatio: 0,
	}
	attempts := 0

	err := WithRetry(context.Background(), config, func() error {
		attempts++
		if attempts < 3 {
			return &RetryableError{Err: errors.New("temporary error")}
		}
		return nil
	})

	assert.NoError(t, err)
	assert.Equal(t, 3, attempts)
}

func TestRetry_MaxRetriesExceeded(t *testing.T) {
	config := RetryConfig{
		MaxRetries:  2,
		BaseDelay:   10 * time.Millisecond,
		MaxDelay:    100 * time.Millisecond,
		Multiplier:  2.0,
		JitterRatio: 0,
	}
	attempts := 0

	err := WithRetry(context.Background(), config, func() error {
		attempts++
		return &RetryableError{Err: errors.New("persistent error")}
	})

	assert.Error(t, err)
	assert.Equal(t, 3, attempts) // Initial + 2 retries
}

func TestRetry_NonRetryableError(t *testing.T) {
	SetQuiet(true) // quiet mode

	config := DefaultRetryConfig()
	attempts := 0

	err := WithRetry(context.Background(), config, func() error {
		attempts++
		return context.Canceled
	})

	assert.Error(t, err)
	assert.Equal(t, 1, attempts) // No retries for non-retryable errors
}

func TestRetry_ContextCanceled(t *testing.T) {
	config := RetryConfig{
		MaxRetries: 5,
		BaseDelay:  100 * time.Millisecond,
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	err := WithRetry(ctx, config, func() error {
		return &RetryableError{Err: errors.New("error")}
	})

	assert.ErrorIs(t, err, context.Canceled)
}

func TestRetry_IsRetryableStatusCode(t *testing.T) {
	tests := []struct {
		code     int
		expected bool
	}{
		{200, false},
		{400, false},
		{401, false},
		{403, false},
		{404, false},
		{429, true},
		{500, true},
		{502, true},
		{503, true},
		{504, true},
	}

	for _, tt := range tests {
		t.Run(string(rune(tt.code)), func(t *testing.T) {
			assert.Equal(t, tt.expected, IsRetryableStatusCode(tt.code))
		})
	}
}

func TestRetry_NoRetryConfig(t *testing.T) {
	config := NoRetryConfig()
	attempts := 0

	err := WithRetry(context.Background(), config, func() error {
		attempts++
		return &RetryableError{Err: errors.New("error")}
	})

	assert.Error(t, err)
	assert.Equal(t, 1, attempts)
}

// =============================================================================
// Progress Tests
// =============================================================================

func TestSpinner_StartStop(t *testing.T) {
	SetQuiet(true) // quiet mode to avoid output

	var buf bytes.Buffer
	spinner := NewSpinner("Loading...").SetWriter(&buf)

	spinner.Start()
	time.Sleep(100 * time.Millisecond)
	spinner.Stop()

	// In quiet mode, should not produce output
	assert.Empty(t, buf.String())
}

func TestSpinner_StopWithMessage(t *testing.T) {
	SetQuiet(false)

	var buf bytes.Buffer
	spinner := NewSpinner("Loading...").SetWriter(&buf)

	spinner.Start()
	time.Sleep(50 * time.Millisecond)
	spinner.StopWithSuccess("Done!")

	assert.Contains(t, buf.String(), "Done!")
}

// =============================================================================
// Format Tests
// =============================================================================

func TestTable(t *testing.T) {
	SetQuiet(false)

	var buf bytes.Buffer
	table := NewTable("ID", "NAME", "STATUS").SetWriter(&buf)

	table.AddRow("1", "Item One", "Active")
	table.AddRow("2", "Item Two", "Inactive")
	table.Render()

	output := buf.String()
	assert.Contains(t, output, "ID")
	assert.Contains(t, output, "NAME")
	assert.Contains(t, output, "STATUS")
	assert.Contains(t, output, "Item One")
	assert.Contains(t, output, "Item Two")
	assert.Equal(t, 2, table.RowCount())
}

func TestTable_AlignRight(t *testing.T) {
	SetQuiet(false)

	var buf bytes.Buffer
	table := NewTable("NAME", "COUNT").SetWriter(&buf)
	table.AlignRight(1)
	table.AddRow("Items", "100")
	table.Render()

	assert.Contains(t, buf.String(), "100")
}

func TestConfirm(t *testing.T) {
	SetQuiet(true) // quiet mode

	// In quiet mode, should return default WITHOUT prompting.
	// Destructive commands rely on this: default-no confirms cancel in quiet
	// mode, so --force/--yes stays the only way to skip confirmation.
	assert.True(t, Confirm("Continue?", true))
	assert.False(t, Confirm("Continue?", false))
}

func TestConfirm_Interactive(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		defaultYes bool
		want       bool
	}{
		{"y accepts", "y\n", false, true},
		{"yes accepts", "yes\n", false, true},
		{"uppercase Y accepts", "Y\n", false, true},
		{"uppercase YES accepts", "YES\n", false, true},
		{"n rejects", "n\n", false, false},
		{"no rejects", "no\n", false, false},
		{"garbage rejects even with default yes", "maybe\n", true, false},
		{"empty input returns default no", "\n", false, false},
		{"empty input returns default yes", "\n", true, true},
		{"no input (EOF) returns default no", "", false, false},
		{"no input (EOF) returns default yes", "", true, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			SetQuiet(false) // NOT quiet: exercise the stdin-reading path

			r, w, err := os.Pipe()
			require.NoError(t, err)
			_, err = w.WriteString(tt.input)
			require.NoError(t, err)
			require.NoError(t, w.Close())

			oldStdin := os.Stdin
			os.Stdin = r
			t.Cleanup(func() {
				os.Stdin = oldStdin
				_ = r.Close()
			})

			assert.Equal(t, tt.want, Confirm("Proceed?", tt.defaultYes))
		})
	}
}

// =============================================================================
// Error Tests
// =============================================================================

func TestWrapError_Nil(t *testing.T) {
	assert.Nil(t, WrapError(nil))
}

func TestWrapError_DomainErrors(t *testing.T) {
	tests := []struct {
		err      error
		contains string
		code     string
	}{
		{domain.ErrNotConfigured, "not configured", ErrCodeNotConfigured},
		{domain.ErrAuthFailed, "Authentication failed", ErrCodeAuthFailed},
		{domain.ErrGrantNotFound, "Grant not found", ErrCodeNotFound},
		{domain.ErrNoDefaultGrant, "No default grant", ErrCodeNotConfigured},
		{domain.ErrSecretNotFound, "Credentials not found", ErrCodeNotConfigured},
		{domain.ErrNetworkError, "Network error", ErrCodeNetworkError},
		{domain.ErrTokenExpired, "expired", ErrCodeAuthFailed},
		{domain.ErrOTPNotFound, "No OTP", ErrCodeNotFound},
		{domain.ErrInvalidProvider, "Invalid email provider", ErrCodeInvalidInput},
	}

	for _, tt := range tests {
		t.Run(tt.err.Error(), func(t *testing.T) {
			cliErr := WrapError(tt.err)
			assert.Contains(t, cliErr.Message, tt.contains)
			assert.Equal(t, tt.code, cliErr.Code)
			assert.NotEmpty(t, cliErr.Suggestion)
		})
	}
}

func TestWrapError_PatternMatching(t *testing.T) {
	tests := []struct {
		errMsg   string
		contains string
		code     string
	}{
		{"Invalid API Key", "Invalid API key", ErrCodeAuthFailed},
		{"rate limit exceeded", "Rate limit", ErrCodeRateLimited},
		{"connection refused", "Unable to connect", ErrCodeNetworkError},
		{"request timeout", "timed out", ErrCodeNetworkError},
		{"500 Internal Server Error", "server error", ErrCodeServerError},
	}

	for _, tt := range tests {
		t.Run(tt.errMsg, func(t *testing.T) {
			cliErr := WrapError(errors.New(tt.errMsg))
			assert.Contains(t, strings.ToLower(cliErr.Message), strings.ToLower(tt.contains))
			assert.Equal(t, tt.code, cliErr.Code)
		})
	}
}

func TestWrapError_AlreadyCLIError(t *testing.T) {
	original := &CLIError{
		Message:    "Original error",
		Suggestion: "Original suggestion",
		Code:       "E999",
	}

	wrapped := WrapError(original)
	assert.Equal(t, original, wrapped)
}

func TestFormatError(t *testing.T) {
	SetQuiet(false)

	err := domain.ErrNotConfigured
	formatted := FormatError(err)

	assert.Contains(t, formatted, "Error:")
	assert.Contains(t, formatted, "Suggestion:")
}

func TestFormatError_DebugMode(t *testing.T) {
	SetQuiet(false)

	err := errors.New("detailed error message")
	formatted := FormatError(err)

	assert.Contains(t, formatted, "detailed error message")
}

func TestNewUserError(t *testing.T) {
	err := NewUserError("Something went wrong", "Try again")
	var cliErr *CLIError
	require.True(t, errors.As(err, &cliErr))
	assert.Equal(t, "Something went wrong", cliErr.Message)
	assert.Equal(t, "Try again", cliErr.Suggestion)
}

func TestNewInputError(t *testing.T) {
	err := NewInputError("Invalid value")
	var cliErr *CLIError
	require.True(t, errors.As(err, &cliErr))
	assert.Equal(t, "Invalid value", cliErr.Message)
	assert.Equal(t, ErrCodeInvalidInput, cliErr.Code)
}
