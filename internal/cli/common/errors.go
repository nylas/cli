package common

import (
	"errors"
	"fmt"
	"strings"

	"github.com/fatih/color"
	"github.com/nylas/cli/internal/domain"
)

// CLIError wraps an error with additional context for CLI display.
type CLIError struct {
	Err        error
	Message    string
	Suggestion string
	Code       string
}

func (e *CLIError) Error() string {
	return e.Message
}

func (e *CLIError) Unwrap() error {
	return e.Err
}

// ErrorCode constants for common errors.
const (
	ErrCodeNotConfigured    = "E001"
	ErrCodeAuthFailed       = "E002"
	ErrCodeNetworkError     = "E003"
	ErrCodeNotFound         = "E004"
	ErrCodePermissionDenied = "E005"
	ErrCodeInvalidInput     = "E006"
	ErrCodeRateLimited      = "E007"
	ErrCodeServerError      = "E008"
)

// WrapError wraps an error with CLI-friendly context.
func WrapError(err error) *CLIError {
	if err == nil {
		return nil
	}

	// Check for existing CLIError
	var cliErr *CLIError
	if errors.As(err, &cliErr) {
		return cliErr
	}

	// Map domain errors to CLI errors
	switch {
	case errors.Is(err, domain.ErrNotConfigured):
		return &CLIError{
			Err:        err,
			Message:    "Nylas CLI is not configured",
			Suggestion: "Run 'nylas auth config' to set up your API credentials",
			Code:       ErrCodeNotConfigured,
		}

	case errors.Is(err, domain.ErrAuthFailed):
		return &CLIError{
			Err:        err,
			Message:    "Authentication failed",
			Suggestion: "Check your API key with 'nylas auth status' or reconfigure with 'nylas auth config'",
			Code:       ErrCodeAuthFailed,
		}

	case errors.Is(err, domain.ErrGrantNotFound):
		return &CLIError{
			Err:        err,
			Message:    "Grant not found",
			Suggestion: "Run 'nylas auth list' to see available grants, or 'nylas auth login' to add a new one",
			Code:       ErrCodeNotFound,
		}

	case errors.Is(err, domain.ErrNoDefaultGrant):
		return &CLIError{
			Err:        err,
			Message:    "No default grant set",
			Suggestion: "Run 'nylas auth list' to see grants, then 'nylas auth switch <grant-id>' to set a default",
			Code:       ErrCodeNotConfigured,
		}

	case errors.Is(err, domain.ErrSecretNotFound):
		return &CLIError{
			Err:        err,
			Message:    "Credentials not found in secret store",
			Suggestion: "Run 'nylas auth config' to set up your API credentials",
			Code:       ErrCodeNotConfigured,
		}

	case errors.Is(err, domain.ErrSecretStoreFailed):
		return &CLIError{
			Err:        err,
			Message:    "Failed to access secret store",
			Suggestion: "Check that your system keyring is accessible, or try running with elevated permissions",
			Code:       ErrCodePermissionDenied,
		}

	case errors.Is(err, domain.ErrNetworkError):
		return &CLIError{
			Err:        err,
			Message:    "Network error",
			Suggestion: "Check your internet connection and try again",
			Code:       ErrCodeNetworkError,
		}

	case errors.Is(err, domain.ErrTokenExpired):
		return &CLIError{
			Err:        err,
			Message:    "Authentication token has expired",
			Suggestion: "Run 'nylas auth login' to re-authenticate",
			Code:       ErrCodeAuthFailed,
		}

	case errors.Is(err, domain.ErrInvalidProvider):
		return &CLIError{
			Err:        err,
			Message:    "Invalid email provider",
			Suggestion: "Supported providers are 'google' and 'microsoft'",
			Code:       ErrCodeInvalidInput,
		}
	}

	// Check for common error patterns in the error message
	errMsg := err.Error()

	if strings.Contains(errMsg, "Invalid API Key") {
		return &CLIError{
			Err:        err,
			Message:    "Invalid API key",
			Suggestion: "Run 'nylas auth config' to update your API key",
			Code:       ErrCodeAuthFailed,
		}
	}

	if strings.Contains(errMsg, "rate limit") || strings.Contains(errMsg, "429") {
		return &CLIError{
			Err:        err,
			Message:    "Rate limit exceeded",
			Suggestion: "Wait a moment and try again, or reduce the frequency of requests",
			Code:       ErrCodeRateLimited,
		}
	}

	if strings.Contains(errMsg, "connection refused") || strings.Contains(errMsg, "no such host") {
		return &CLIError{
			Err:        err,
			Message:    "Unable to connect to Nylas API",
			Suggestion: "Check your internet connection and firewall settings",
			Code:       ErrCodeNetworkError,
		}
	}

	if strings.Contains(errMsg, "timeout") {
		return &CLIError{
			Err:        err,
			Message:    "Request timed out",
			Suggestion: "The server is taking too long to respond. Try again later",
			Code:       ErrCodeNetworkError,
		}
	}

	if strings.Contains(errMsg, "500") || strings.Contains(errMsg, "502") || strings.Contains(errMsg, "503") {
		return &CLIError{
			Err:        err,
			Message:    "Nylas API server error",
			Suggestion: "This is a temporary issue. Please try again in a few minutes",
			Code:       ErrCodeServerError,
		}
	}

	// Default wrapper
	return &CLIError{
		Err:     err,
		Message: err.Error(),
	}
}

// FormatError formats an error for CLI display.
func FormatError(err error) string {
	cliErr := WrapError(err)
	if cliErr == nil {
		return ""
	}

	var sb strings.Builder

	// Error message
	_, _ = Red.Fprintf(&sb, "Error: %s\n", cliErr.Message)

	// Error code (if available)
	if cliErr.Code != "" {
		_, _ = Dim.Fprintf(&sb, "  Code: %s\n", cliErr.Code)
	}

	// Suggestion (if available)
	if cliErr.Suggestion != "" {
		_, _ = Yellow.Fprintf(&sb, "  Hint: %s\n", cliErr.Suggestion)
	}

	// Original error in debug mode
	if IsDebug() && cliErr.Err != nil && cliErr.Err.Error() != cliErr.Message {
		_, _ = Dim.Fprintf(&sb, "  Details: %s\n", cliErr.Err.Error())
	}

	return sb.String()
}

// PrintFormattedError prints a formatted error to stderr.
func PrintFormattedError(err error) {
	_, _ = fmt.Fprint(color.Error, FormatError(err))
}

// NewUserError creates a user-facing error with a suggestion.
func NewUserError(message, suggestion string) error {
	return &CLIError{
		Message:    message,
		Suggestion: suggestion,
	}
}

// NewInputError creates an input validation error.
func NewInputError(message string) error {
	return &CLIError{
		Message: message,
		Code:    ErrCodeInvalidInput,
	}
}

// WrapGetError wraps an error from a GET operation.
func WrapGetError(resource string, err error) error {
	return fmt.Errorf("failed to get %s: %w", resource, err)
}

// WrapFetchError wraps an error from a fetch/list operation.
func WrapFetchError(resource string, err error) error {
	return fmt.Errorf("failed to fetch %s: %w", resource, err)
}

// WrapCreateError wraps an error from a create operation.
func WrapCreateError(resource string, err error) error {
	return fmt.Errorf("failed to create %s: %w", resource, err)
}

// WrapUpdateError wraps an error from an update operation.
func WrapUpdateError(resource string, err error) error {
	return fmt.Errorf("failed to update %s: %w", resource, err)
}

// WrapDeleteError wraps an error from a delete operation.
func WrapDeleteError(resource string, err error) error {
	return fmt.Errorf("failed to delete %s: %w", resource, err)
}

// WrapSendError wraps an error from a send operation.
func WrapSendError(resource string, err error) error {
	return fmt.Errorf("failed to send %s: %w", resource, err)
}

// WrapListError wraps an error from a list operation.
func WrapListError(resource string, err error) error {
	return fmt.Errorf("failed to list %s: %w", resource, err)
}

// WrapLoadError wraps an error from a load operation.
func WrapLoadError(resource string, err error) error {
	return fmt.Errorf("failed to load %s: %w", resource, err)
}

// WrapSaveError wraps an error from a save operation.
func WrapSaveError(resource string, err error) error {
	return fmt.Errorf("failed to save %s: %w", resource, err)
}

// WrapMarshalError wraps an error from a marshal/encode operation.
func WrapMarshalError(resource string, err error) error {
	return fmt.Errorf("failed to marshal %s: %w", resource, err)
}

// WrapDecodeError wraps an error from a decode operation.
func WrapDecodeError(resource string, err error) error {
	return fmt.Errorf("failed to decode %s: %w", resource, err)
}

// WrapWriteError wraps an error from a write operation.
func WrapWriteError(resource string, err error) error {
	return fmt.Errorf("failed to write %s: %w", resource, err)
}

// WrapDownloadError wraps an error from a download operation.
func WrapDownloadError(resource string, err error) error {
	return fmt.Errorf("failed to download %s: %w", resource, err)
}

// WrapCancelError wraps an error from a cancel operation.
func WrapCancelError(resource string, err error) error {
	return fmt.Errorf("failed to cancel %s: %w", resource, err)
}

// WrapGenerateError wraps an error from a generate operation.
func WrapGenerateError(resource string, err error) error {
	return fmt.Errorf("failed to generate %s: %w", resource, err)
}

// WrapSearchError wraps an error from a search operation.
func WrapSearchError(resource string, err error) error {
	return fmt.Errorf("failed to search %s: %w", resource, err)
}

// WrapDateParseError wraps a date parsing error with context.
func WrapDateParseError(flagName string, err error) error {
	return &CLIError{
		Err:     err,
		Message: fmt.Sprintf("invalid '%s' date format", flagName),
		Code:    ErrCodeInvalidInput,
	}
}

// NewMutuallyExclusiveError creates an error for mutually exclusive flags.
func NewMutuallyExclusiveError(flag1, flag2 string) error {
	return &CLIError{
		Message: fmt.Sprintf("cannot specify both --%s and --%s", flag1, flag2),
		Code:    ErrCodeInvalidInput,
	}
}

// WrapRecipientError wraps a recipient parsing error.
func WrapRecipientError(recipientType string, err error) error {
	return &CLIError{
		Err:     err,
		Message: fmt.Sprintf("invalid '%s' recipient format", recipientType),
		Code:    ErrCodeInvalidInput,
	}
}
