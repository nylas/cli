// Package domain contains the core business logic and domain models.
package domain

import "errors"

// Sentinel errors for the application.
var (
	// Auth errors
	ErrNotConfigured   = errors.New("nylas not configured")
	ErrAuthFailed      = errors.New("authentication failed")
	ErrAuthTimeout     = errors.New("authentication timed out")
	ErrInvalidProvider = errors.New("invalid provider")
	ErrGrantNotFound   = errors.New("grant not found")
	ErrNoDefaultGrant  = errors.New("no default grant set")
	ErrInvalidGrant    = errors.New("invalid or expired grant")
	ErrTokenExpired    = errors.New("token expired")
	ErrAPIError        = errors.New("nylas API error")
	ErrNetworkError    = errors.New("network error")
	ErrInvalidInput    = errors.New("invalid input")

	// Secret store errors
	ErrSecretNotFound    = errors.New("secret not found")
	ErrSecretStoreFailed = errors.New("secret store operation failed")

	// Config errors
	ErrConfigNotFound = errors.New("config not found")
	ErrConfigInvalid  = errors.New("config invalid")

	// OTP errors
	ErrOTPNotFound     = errors.New("no OTP found in recent messages")
	ErrAccountNotFound = errors.New("account not found")
	ErrNoMessages      = errors.New("no messages found")

	// Slack errors
	ErrSlackNotConfigured    = errors.New("slack not configured")
	ErrSlackAuthFailed       = errors.New("slack authentication failed")
	ErrSlackRateLimited      = errors.New("slack rate limited")
	ErrSlackChannelNotFound  = errors.New("slack channel not found")
	ErrSlackMessageNotFound  = errors.New("slack message not found")
	ErrSlackPermissionDenied = errors.New("slack permission denied")

	// Resource not found errors - use these instead of creating ad-hoc errors.
	// Wrap with additional context: fmt.Errorf("%w: %s", domain.ErrContactNotFound, id)
	ErrContactNotFound     = errors.New("contact not found")
	ErrEventNotFound       = errors.New("event not found")
	ErrCalendarNotFound    = errors.New("calendar not found")
	ErrMessageNotFound     = errors.New("message not found")
	ErrFolderNotFound      = errors.New("folder not found")
	ErrDraftNotFound       = errors.New("draft not found")
	ErrThreadNotFound      = errors.New("thread not found")
	ErrAttachmentNotFound  = errors.New("attachment not found")
	ErrWebhookNotFound     = errors.New("webhook not found")
	ErrNotetakerNotFound   = errors.New("notetaker not found")
	ErrTemplateNotFound    = errors.New("template not found")
	ErrApplicationNotFound = errors.New("application not found")
	ErrConnectorNotFound   = errors.New("connector not found")
	ErrCredentialNotFound  = errors.New("credential not found")

	// Dashboard auth errors
	ErrDashboardNotLoggedIn    = errors.New("not logged in to Nylas Dashboard")
	ErrDashboardSessionExpired = errors.New("dashboard session expired")
	ErrDashboardLoginFailed    = errors.New("dashboard login failed")
	ErrDashboardMFARequired    = errors.New("MFA required")
	ErrDashboardSSOFailed      = errors.New("SSO authentication failed")
	ErrDashboardDPoP           = errors.New("DPoP proof generation failed")

	// Scheduler errors
	ErrBookingNotFound       = errors.New("booking not found")
	ErrSessionNotFound       = errors.New("session not found")
	ErrConfigurationNotFound = errors.New("configuration not found")
	ErrPageNotFound          = errors.New("page not found")
)
