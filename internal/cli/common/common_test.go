package common

import (
	"bytes"
	"context"
	"errors"
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

func TestLogger_Init(t *testing.T) {
	ResetLogger()

	InitLogger(false, false)
	assert.NotNil(t, GetLogger())
	assert.False(t, IsDebug())
	assert.False(t, IsQuiet())
}

func TestLogger_DebugMode(t *testing.T) {
	ResetLogger()

	InitLogger(true, false)
	assert.True(t, IsDebug())
	assert.False(t, IsQuiet())
}

func TestLogger_QuietMode(t *testing.T) {
	ResetLogger()

	InitLogger(false, true)
	assert.False(t, IsDebug())
	assert.True(t, IsQuiet())
}

func TestLogger_Functions(t *testing.T) {
	ResetLogger()
	InitLogger(true, false)

	// These should not panic
	Debug("debug message", "key", "value")
	Info("info message")
	Warn("warning message")
	Error("error message")
	DebugHTTP("GET", "https://api.nylas.com", 200, "100ms")
	DebugAPI("GetMessages", "grant_id", "test")
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
	ResetLogger()
	InitLogger(false, true) // quiet mode

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
	ResetLogger()
	InitLogger(false, true) // quiet mode to avoid output

	var buf bytes.Buffer
	spinner := NewSpinner("Loading...").SetWriter(&buf)

	spinner.Start()
	time.Sleep(100 * time.Millisecond)
	spinner.Stop()

	// In quiet mode, should not produce output
	assert.Empty(t, buf.String())
}

func TestSpinner_StopWithMessage(t *testing.T) {
	ResetLogger()
	InitLogger(false, false)

	var buf bytes.Buffer
	spinner := NewSpinner("Loading...").SetWriter(&buf)

	spinner.Start()
	time.Sleep(50 * time.Millisecond)
	spinner.StopWithSuccess("Done!")

	assert.Contains(t, buf.String(), "Done!")
}

func TestProgressBar_Increment(t *testing.T) {
	ResetLogger()
	InitLogger(false, true) // quiet mode

	var buf bytes.Buffer
	bar := NewProgressBar(10, "Processing").SetWriter(&buf)

	for i := 0; i < 10; i++ {
		bar.Increment()
	}

	// In quiet mode, should not produce output
	assert.Empty(t, buf.String())
}

func TestProgressBar_Set(t *testing.T) {
	ResetLogger()
	InitLogger(false, true)

	bar := NewProgressBar(100, "Processing")
	bar.Set(50)
	bar.Finish()
}

func TestCounter(t *testing.T) {
	ResetLogger()
	InitLogger(false, true)

	counter := NewCounter("Items")
	counter.Increment()
	counter.Increment()
	counter.Increment()

	assert.Equal(t, 3, counter.Count())
	counter.Finish()
}

// =============================================================================
// Format Tests
// =============================================================================

func TestParseFormat(t *testing.T) {
	tests := []struct {
		input    string
		expected OutputFormat
		hasError bool
	}{
		{"table", FormatTable, false},
		{"TABLE", FormatTable, false},
		{"", FormatTable, false},
		{"json", FormatJSON, false},
		{"JSON", FormatJSON, false},
		{"csv", FormatCSV, false},
		{"yaml", FormatYAML, false},
		{"yml", FormatYAML, false},
		{"invalid", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			format, err := ParseFormat(tt.input)
			if tt.hasError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, format)
			}
		})
	}
}

func TestFormatter_JSON(t *testing.T) {
	var buf bytes.Buffer
	formatter := NewFormatter(FormatJSON).SetWriter(&buf)

	data := map[string]string{"key": "value"}
	err := formatter.Format(data)

	require.NoError(t, err)
	assert.Contains(t, buf.String(), `"key"`)
	assert.Contains(t, buf.String(), `"value"`)
}

func TestFormatter_YAML(t *testing.T) {
	var buf bytes.Buffer
	formatter := NewFormatter(FormatYAML).SetWriter(&buf)

	data := map[string]string{"key": "value"}
	err := formatter.Format(data)

	require.NoError(t, err)
	assert.Contains(t, buf.String(), "key:")
	assert.Contains(t, buf.String(), "value")
}

func TestFormatter_CSV(t *testing.T) {
	var buf bytes.Buffer
	formatter := NewFormatter(FormatCSV).SetWriter(&buf)

	type Item struct {
		Name  string `json:"name"`
		Value int    `json:"value"`
	}

	data := []Item{
		{Name: "item1", Value: 1},
		{Name: "item2", Value: 2},
	}

	err := formatter.Format(data)

	require.NoError(t, err)
	assert.Contains(t, buf.String(), "name")
	assert.Contains(t, buf.String(), "item1")
	assert.Contains(t, buf.String(), "item2")
}

func TestTable(t *testing.T) {
	ResetLogger()
	InitLogger(false, false)

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
	ResetLogger()
	InitLogger(false, false)

	var buf bytes.Buffer
	table := NewTable("NAME", "COUNT").SetWriter(&buf)
	table.AlignRight(1)
	table.AddRow("Items", "100")
	table.Render()

	assert.Contains(t, buf.String(), "100")
}

func TestConfirm(t *testing.T) {
	ResetLogger()
	InitLogger(false, true) // quiet mode

	// In quiet mode, should return default
	assert.True(t, Confirm("Continue?", true))
	assert.False(t, Confirm("Continue?", false))
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
	ResetLogger()
	InitLogger(false, false)

	err := domain.ErrNotConfigured
	formatted := FormatError(err)

	assert.Contains(t, formatted, "Error:")
	assert.Contains(t, formatted, "Hint:")
}

func TestFormatError_DebugMode(t *testing.T) {
	ResetLogger()
	InitLogger(true, false)

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
