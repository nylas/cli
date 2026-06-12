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
	apiErr := &domain.APIError{StatusCode: 403, Message: "Access denied"}
	result := WrapError(apiErr)

	require.NotNil(t, result)
	assert.Equal(t, ErrCodePermissionDenied, result.Code)
	assert.Equal(t, "Permission denied", result.Message)
	for _, s := range result.Suggestions {
		assert.NotContains(t, s, "auth show", "generic 403 should not get scope-specific suggestion")
	}
}

func TestWrapError_PlanLimitForbidden(t *testing.T) {
	// The inbox service returns a 403 (forbidden_access) for billing-plan
	// capacity limits on rules and lists. These must NOT be reported as an
	// API-key permission problem — the user needs to remove items or upgrade.
	cases := []struct {
		name    string
		message string
	}{
		{"rules cap reached", "Maximum number of rules (5) reached for this plan"},
		{"lists cap reached", "Maximum number of lists (50) reached for this plan"},
		{"rules not allowed", "Rules are not allowed for this plan"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			apiErr := &domain.APIError{
				StatusCode: 403,
				Type:       "forbidden_access",
				Message:    tc.message,
				RequestID:  "req-plan-1",
			}
			result := WrapError(fmt.Errorf("failed to create rule: %w", apiErr))

			require.NotNil(t, result)
			// Surface the server's real reason, not "Permission denied".
			assert.Equal(t, tc.message, result.Message)
			assert.Equal(t, "req-plan-1", result.RequestID)
			joined := strings.Join(append(result.Suggestions, result.Suggestion), " | ")
			assert.Contains(t, strings.ToLower(joined), "plan",
				"plan-limit 403 must suggest a plan/limit action")
			assert.NotContains(t, strings.ToLower(joined), "api key",
				"plan-limit 403 must not blame the API key")
		})
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
			name:        "rate limit 429",
			err:         &domain.APIError{StatusCode: 429, Type: "api_error"},
			wantMessage: "Rate limit exceeded",
			wantCode:    ErrCodeRateLimited,
			wantSuggest: "reduce the frequency",
		},
		{
			name:        "server error 500",
			err:         &domain.APIError{StatusCode: 500, Type: "api_error"},
			wantMessage: "Nylas API server error",
			wantCode:    ErrCodeServerError,
			wantSuggest: "temporary issue",
		},
		{
			name:        "unauthorized 401",
			err:         &domain.APIError{StatusCode: 401, Type: "unauthorized", Message: "Invalid API Key"},
			wantMessage: "Authentication failed",
			wantCode:    ErrCodeAuthFailed,
			wantSuggest: "nylas auth status",
		},
		{
			name:        "forbidden 403",
			err:         &domain.APIError{StatusCode: 403, Message: "Access denied"},
			wantMessage: "Permission denied",
			wantCode:    ErrCodePermissionDenied,
			wantSuggest: "required permissions",
		},
		{
			name:        "not found 404",
			err:         &domain.APIError{StatusCode: 404, Message: "Not found"},
			wantMessage: "Resource not found",
			wantCode:    ErrCodeNotFound,
			wantSuggest: "list",
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

func TestWrapError_RequestIDDoesNotFalseMatchStatusCode(t *testing.T) {
	tests := []struct {
		name     string
		err      *domain.APIError
		wantCode string
	}{
		{
			name:     "request ID containing 502 on a 401 error",
			err:      &domain.APIError{StatusCode: 401, Message: "bad token", RequestID: "req-5023-abc"},
			wantCode: ErrCodeAuthFailed,
		},
		{
			name:     "request ID containing 429 on a 404 error",
			err:      &domain.APIError{StatusCode: 404, Message: "not found", RequestID: "req-4291-def"},
			wantCode: ErrCodeNotFound,
		},
		{
			name:     "request ID containing 500 on a 403 error",
			err:      &domain.APIError{StatusCode: 403, Message: "forbidden", RequestID: "req-5001-xyz"},
			wantCode: ErrCodePermissionDenied,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wrapped := fmt.Errorf("API call failed: %w", tt.err)
			result := WrapError(wrapped)

			require.NotNil(t, result)
			assert.Equal(t, tt.wantCode, result.Code)
			assert.Equal(t, tt.err.RequestID, result.RequestID)
		})
	}
}

func TestWrapError_PropagatesRequestID(t *testing.T) {
	tests := []struct {
		name      string
		err       *domain.APIError
		wantReqID string
	}{
		{
			name:      "server error with request ID",
			err:       &domain.APIError{StatusCode: 500, RequestID: "req-abc-123"},
			wantReqID: "req-abc-123",
		},
		{
			name:      "rate limit with request ID",
			err:       &domain.APIError{StatusCode: 429, RequestID: "req-def-456"},
			wantReqID: "req-def-456",
		},
		{
			name:      "insufficient scopes with request ID",
			err:       &domain.APIError{StatusCode: 403, Type: "insufficient_scopes", RequestID: "req-scope-789"},
			wantReqID: "req-scope-789",
		},
		{
			name:      "unclassified status with request ID",
			err:       &domain.APIError{StatusCode: 401, Type: "unauthorized", Message: "bad token", RequestID: "req-fallback-999"},
			wantReqID: "req-fallback-999",
		},
		{
			name:      "empty request ID stays empty",
			err:       &domain.APIError{StatusCode: 500},
			wantReqID: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := WrapError(tt.err)

			require.NotNil(t, result)
			assert.Equal(t, tt.wantReqID, result.RequestID)
		})
	}
}

func TestWrapError_StripsInlineRequestIDFromMessage(t *testing.T) {
	apiErr := &domain.APIError{
		StatusCode: 400,
		Type:       "invalid_request",
		Message:    "Missing required field",
		RequestID:  "req-strip-test",
	}
	wrapped := fmt.Errorf("failed to create event: %w", apiErr)

	result := WrapError(wrapped)

	require.NotNil(t, result)
	assert.Equal(t, "req-strip-test", result.RequestID)
	assert.NotContains(t, result.Message, "[request_id:")
	assert.Contains(t, result.Message, "Missing required field")
}

func TestFormatError_RendersRequestID(t *testing.T) {
	cliErr := &CLIError{
		Message:    "Nylas API server error",
		Code:       ErrCodeServerError,
		Suggestion: "Try again later",
		RequestID:  "1120765200-c4c8e151-3414-4448-b884-1498872b0912",
	}

	result := FormatError(cliErr)

	assert.Contains(t, result, "Request ID: 1120765200-c4c8e151-3414-4448-b884-1498872b0912")
}

func TestFormatError_OmitsEmptyRequestID(t *testing.T) {
	cliErr := &CLIError{
		Message: "Some error",
		Code:    ErrCodeServerError,
	}

	result := FormatError(cliErr)

	assert.NotContains(t, result, "Request ID")
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
