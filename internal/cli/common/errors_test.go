//go:build !integration

package common

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/nylas/cli/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCLIError_Error tests the Error() method of CLIError.
func TestCLIError_Error(t *testing.T) {
	tests := []struct {
		name     string
		err      *CLIError
		expected string
	}{
		{
			name:     "returns message",
			err:      &CLIError{Message: "test error message"},
			expected: "test error message",
		},
		{
			name:     "empty message",
			err:      &CLIError{Message: ""},
			expected: "",
		},
		{
			name:     "message with details",
			err:      &CLIError{Message: "error", Code: "E001", Suggestion: "try this"},
			expected: "error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.err.Error())
		})
	}
}

// TestCLIError_Unwrap tests error unwrapping behavior.
func TestCLIError_Unwrap(t *testing.T) {
	originalErr := errors.New("original error")
	cliErr := &CLIError{
		Err:     originalErr,
		Message: "wrapped error",
	}

	assert.Equal(t, originalErr, cliErr.Unwrap())
	assert.True(t, errors.Is(cliErr, originalErr))
}

// TestWrapError_NilError tests that nil errors return nil.
func TestWrapError_NilError(t *testing.T) {
	result := WrapError(nil)
	assert.Nil(t, result)
}

// TestWrapError_ExistingCLIError tests that CLIErrors are returned as-is.
func TestWrapError_ExistingCLIError(t *testing.T) {
	original := &CLIError{
		Message:    "original message",
		Suggestion: "original suggestion",
		Code:       "E999",
	}

	result := WrapError(original)

	assert.Same(t, original, result)
	assert.Equal(t, "original message", result.Message)
	assert.Equal(t, "original suggestion", result.Suggestion)
	assert.Equal(t, "E999", result.Code)
}

// TestWrapError_DomainErrors_Extended tests additional domain error handling.
func TestWrapError_DomainErrors_Extended(t *testing.T) {
	tests := []struct {
		name            string
		err             error
		expectedMessage string
		expectedCode    string
		hasSuggestion   bool
	}{
		{
			name:            "ErrSecretStoreFailed",
			err:             domain.ErrSecretStoreFailed,
			expectedMessage: "Failed to access secret store",
			expectedCode:    ErrCodePermissionDenied,
			hasSuggestion:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := WrapError(tt.err)

			require.NotNil(t, result)
			assert.Equal(t, tt.expectedMessage, result.Message)
			assert.Equal(t, tt.expectedCode, result.Code)
			if tt.hasSuggestion {
				assert.NotEmpty(t, result.Suggestion)
			}
			assert.True(t, errors.Is(result, tt.err))
		})
	}
}

func TestWrapError_InsufficientScopes(t *testing.T) {
	t.Run("with message", func(t *testing.T) {
		apiErr := &domain.APIError{
			StatusCode: 403,
			Type:       "insufficient_scopes",
			Message:    "Grant lacks required scope: gmail.readonly",
		}
		wrapped := fmt.Errorf("failed to fetch threads: %w", apiErr)

		result := WrapError(wrapped)

		require.NotNil(t, result)
		assert.Equal(t, ErrCodePermissionDenied, result.Code)
		assert.Equal(t, "Grant lacks required scope: gmail.readonly", result.Message)
		if assert.NotEmpty(t, result.Suggestions) {
			joined := strings.Join(result.Suggestions, " | ")
			assert.Contains(t, joined, "nylas auth show")
			assert.Contains(t, joined, "nylas auth login")
		}
	})

	t.Run("empty message", func(t *testing.T) {
		apiErr := &domain.APIError{
			StatusCode: 403,
			Type:       "insufficient_scopes",
		}
		result := WrapError(apiErr)

		require.NotNil(t, result)
		assert.Equal(t, ErrCodePermissionDenied, result.Code)
		assert.Equal(t, "Grant lacks required scopes for this operation", result.Message)
		if assert.NotEmpty(t, result.Suggestions) {
			joined := strings.Join(result.Suggestions, " | ")
			assert.Contains(t, joined, "nylas auth show")
		}
	})
}

func TestWrapError_GenericForbiddenFallsThrough(t *testing.T) {
	// 403 without insufficient_scopes type should fall through to default
	// wrapper, not get the scope-specific suggestion.
	apiErr := &domain.APIError{StatusCode: 403, Message: "Access denied"}
	result := WrapError(apiErr)

	require.NotNil(t, result)
	assert.NotEqual(t, ErrCodePermissionDenied, result.Code, "generic 403 should not be classified as scope error")
	for _, s := range result.Suggestions {
		assert.NotContains(t, s, "auth show", "generic 403 should not get scope-specific suggestion")
	}
}

func TestWrapError_APIErrorStatusClassification(t *testing.T) {
	tests := []struct {
		name        string
		err         *domain.APIError
		wantMessage string
		wantCode    string
		wantSuggest string
	}{
		{
			name:        "type only rate limit",
			err:         &domain.APIError{StatusCode: 429, Type: "api_error"},
			wantMessage: "Rate limit exceeded",
			wantCode:    ErrCodeRateLimited,
			wantSuggest: "reduce the frequency",
		},
		{
			name:        "type only server error",
			err:         &domain.APIError{StatusCode: 500, Type: "api_error"},
			wantMessage: "Nylas API server error",
			wantCode:    ErrCodeServerError,
			wantSuggest: "temporary issue",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := WrapError(tt.err)

			require.NotNil(t, result)
			assert.Equal(t, tt.wantMessage, result.Message)
			assert.Equal(t, tt.wantCode, result.Code)
			assert.Contains(t, result.Suggestion, tt.wantSuggest)
			assert.True(t, errors.Is(result, domain.ErrAPIError))
		})
	}
}

func TestWrapError_SecretStorePassphraseRequirement(t *testing.T) {
	err := fmt.Errorf("%w: %s must be set to unlock the encrypted file store", domain.ErrSecretStoreFailed, "NYLAS_FILE_STORE_PASSPHRASE")

	result := WrapError(err)

	require.NotNil(t, result)
	assert.Equal(t, "Failed to access encrypted file secret store", result.Message)
	assert.Equal(t, ErrCodePermissionDenied, result.Code)
	assert.Empty(t, result.Suggestion)
	assert.Equal(t, []string{
		"Set NYLAS_FILE_STORE_PASSPHRASE before using the file-based secret store",
		"Unset NYLAS_DISABLE_KEYRING to use the system keyring instead",
	}, result.Suggestions)
	assert.True(t, errors.Is(result, domain.ErrSecretStoreFailed))
}

// TestWrapError_HTTPStatusPatterns tests HTTP status code patterns.
func TestWrapError_HTTPStatusPatterns(t *testing.T) {
	tests := []struct {
		name            string
		errMessage      string
		expectedMessage string
		expectedCode    string
	}{
		{
			name:            "429 status",
			errMessage:      "got status 429",
			expectedMessage: "Rate limit exceeded",
			expectedCode:    ErrCodeRateLimited,
		},
		{
			name:            "500 server error",
			errMessage:      "got status 500",
			expectedMessage: "Nylas API server error",
			expectedCode:    ErrCodeServerError,
		},
		{
			name:            "502 bad gateway",
			errMessage:      "got status 502",
			expectedMessage: "Nylas API server error",
			expectedCode:    ErrCodeServerError,
		},
		{
			name:            "503 service unavailable",
			errMessage:      "got status 503",
			expectedMessage: "Nylas API server error",
			expectedCode:    ErrCodeServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := errors.New(tt.errMessage)
			result := WrapError(err)

			require.NotNil(t, result)
			assert.Equal(t, tt.expectedMessage, result.Message)
			assert.Equal(t, tt.expectedCode, result.Code)
			assert.NotEmpty(t, result.Suggestion)
		})
	}
}

// TestWrapError_NetworkPatterns tests network-related error patterns.
func TestWrapError_NetworkPatterns(t *testing.T) {
	tests := []struct {
		name            string
		errMessage      string
		expectedMessage string
		expectedCode    string
	}{
		{
			name:            "connection refused",
			errMessage:      "dial tcp: connection refused",
			expectedMessage: "Unable to connect to Nylas API",
			expectedCode:    ErrCodeNetworkError,
		},
		{
			name:            "no such host",
			errMessage:      "lookup api.nylas.com: no such host",
			expectedMessage: "Unable to connect to Nylas API",
			expectedCode:    ErrCodeNetworkError,
		},
		{
			name:            "timeout",
			errMessage:      "context deadline exceeded: timeout",
			expectedMessage: "Request timed out",
			expectedCode:    ErrCodeNetworkError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := errors.New(tt.errMessage)
			result := WrapError(err)

			require.NotNil(t, result)
			assert.Equal(t, tt.expectedMessage, result.Message)
			assert.Equal(t, tt.expectedCode, result.Code)
			assert.NotEmpty(t, result.Suggestion)
		})
	}
}

// TestWrapError_UnknownError tests handling of unrecognized errors.
func TestWrapError_UnknownError(t *testing.T) {
	unknownErr := errors.New("some unknown error")
	result := WrapError(unknownErr)

	require.NotNil(t, result)
	assert.Equal(t, "some unknown error", result.Message)
	assert.Empty(t, result.Code)
	assert.Empty(t, result.Suggestion)
	assert.Equal(t, unknownErr, result.Err)
}

// TestFormatError_NilError tests nil error formatting.
func TestFormatError_NilError(t *testing.T) {
	result := FormatError(nil)
	assert.Empty(t, result)
}

// TestFormatError_BasicError tests basic error formatting.
func TestFormatError_BasicError(t *testing.T) {
	err := errors.New("test error")
	result := FormatError(err)

	assert.Contains(t, result, "Error:")
	assert.Contains(t, result, "test error")
}

// TestFormatError_WithCodeAndSuggestion tests complete error formatting.
func TestFormatError_WithCodeAndSuggestion(t *testing.T) {
	cliErr := &CLIError{
		Message:    "Test error message",
		Code:       "E001",
		Suggestion: "Try this fix",
	}

	result := FormatError(cliErr)

	assert.Contains(t, result, "Error:")
	assert.Contains(t, result, "Test error message")
	assert.Contains(t, result, "Code: E001")
	assert.Contains(t, result, "Suggestion:")
	assert.Contains(t, result, "• Try this fix")
}

func TestFormatError_WithMultipleSuggestions(t *testing.T) {
	cliErr := &CLIError{
		Message: "Secret store locked",
		Code:    ErrCodePermissionDenied,
		Suggestions: []string{
			"Set NYLAS_FILE_STORE_PASSPHRASE",
			"Unset NYLAS_DISABLE_KEYRING",
		},
	}

	result := FormatError(cliErr)

	assert.Contains(t, result, "Suggestions:")
	assert.Contains(t, result, "• Set NYLAS_FILE_STORE_PASSPHRASE")
	assert.Contains(t, result, "• Unset NYLAS_DISABLE_KEYRING")
}

// TestErrorCodeConstants tests that all error codes are unique.
func TestErrorCodeConstants(t *testing.T) {
	codes := map[string]bool{
		ErrCodeNotConfigured:    true,
		ErrCodeAuthFailed:       true,
		ErrCodeNetworkError:     true,
		ErrCodeNotFound:         true,
		ErrCodePermissionDenied: true,
		ErrCodeInvalidInput:     true,
		ErrCodeRateLimited:      true,
		ErrCodeServerError:      true,
	}

	assert.Len(t, codes, 8, "all error codes should be unique")
}

// TestWrapError_WrappedDomainError tests detection of wrapped domain errors.
func TestWrapError_WrappedDomainError(t *testing.T) {
	wrappedErr := errors.Join(errors.New("outer"), domain.ErrNotConfigured)

	result := WrapError(wrappedErr)

	require.NotNil(t, result)
	assert.Equal(t, "Nylas CLI is not configured", result.Message)
	assert.Equal(t, ErrCodeNotConfigured, result.Code)
}

// TestCLIError_ErrorChain tests error chain traversal.
func TestCLIError_ErrorChain(t *testing.T) {
	rootCause := errors.New("root cause")
	cliErr := &CLIError{
		Err:     rootCause,
		Message: "CLI error",
	}
	wrapped := errors.Join(errors.New("context"), cliErr)

	var foundCLIErr *CLIError
	assert.True(t, errors.As(wrapped, &foundCLIErr))
	assert.Equal(t, "CLI error", foundCLIErr.Message)

	assert.True(t, errors.Is(cliErr, rootCause))
}

// TestFormatError_MultilineOutput tests multiline error output.
func TestFormatError_MultilineOutput(t *testing.T) {
	cliErr := &CLIError{
		Message:    "Multi-word error message here",
		Code:       "E001",
		Suggestion: "A suggestion with multiple words",
	}

	result := FormatError(cliErr)
	lines := strings.Split(strings.TrimSpace(result), "\n")

	assert.GreaterOrEqual(t, len(lines), 3)
}

// TestNewUserError_Extended tests NewUserError with various inputs.
func TestNewUserError_Extended(t *testing.T) {
	tests := []struct {
		name       string
		message    string
		suggestion string
	}{
		{
			name:       "with suggestion",
			message:    "Something went wrong",
			suggestion: "Try doing X",
		},
		{
			name:       "empty suggestion",
			message:    "Error occurred",
			suggestion: "",
		},
		{
			name:       "both empty",
			message:    "",
			suggestion: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := NewUserError(tt.message, tt.suggestion)

			require.NotNil(t, err)
			var cliErr *CLIError
			require.True(t, errors.As(err, &cliErr))
			assert.Equal(t, tt.message, cliErr.Message)
			assert.Equal(t, tt.suggestion, cliErr.Suggestion)
			assert.Empty(t, cliErr.Code)
		})
	}
}

// TestNewInputError_Extended tests NewInputError with various inputs.
func TestNewInputError_Extended(t *testing.T) {
	tests := []struct {
		name    string
		message string
	}{
		{
			name:    "basic input error",
			message: "Invalid input: field is required",
		},
		{
			name:    "empty message",
			message: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := NewInputError(tt.message)

			require.NotNil(t, err)
			var cliErr *CLIError
			require.True(t, errors.As(err, &cliErr))
			assert.Equal(t, tt.message, cliErr.Message)
			assert.Equal(t, ErrCodeInvalidInput, cliErr.Code)
			assert.Empty(t, cliErr.Suggestion)
		})
	}
}
