package ports

import (
	"context"

	"github.com/nylas/cli/internal/domain"
)

// DashboardAccountClient defines the interface for dashboard-account CLI auth endpoints.
type DashboardAccountClient interface {
	// Register creates a new dashboard account and triggers email verification.
	Register(ctx context.Context, email, password string, privacyPolicyAccepted bool) (*domain.DashboardRegisterResponse, error)

	// VerifyEmailCode verifies the email verification code after registration.
	VerifyEmailCode(ctx context.Context, email, code, region string) (*domain.DashboardAuthResponse, error)

	// ResendVerificationCode resends the email verification code.
	ResendVerificationCode(ctx context.Context, email string) error

	// Login authenticates with email and password.
	// Returns auth response on success, or MFA info if MFA is required.
	Login(ctx context.Context, email, password, orgPublicID string) (*domain.DashboardAuthResponse, *domain.DashboardMFARequired, error)

	// LoginMFA completes MFA authentication with a TOTP code.
	LoginMFA(ctx context.Context, userPublicID, code, orgPublicID string) (*domain.DashboardAuthResponse, error)

	// Refresh refreshes the session tokens.
	Refresh(ctx context.Context, userToken, orgToken string) (*domain.DashboardRefreshResponse, error)

	// Logout invalidates the session tokens.
	Logout(ctx context.Context, userToken, orgToken string) error

	// SSOStart initiates an SSO device authorization flow.
	SSOStart(ctx context.Context, loginType, mode string, privacyPolicyAccepted bool) (*domain.DashboardSSOStartResponse, error)

	// SSOPoll polls the SSO device flow for completion.
	SSOPoll(ctx context.Context, flowID, orgPublicID string) (*domain.DashboardSSOPollResponse, error)

	// GetCurrentSession returns the current session info including the active org.
	GetCurrentSession(ctx context.Context, userToken, orgToken string) (*domain.DashboardSessionResponse, error)

	// SwitchOrg switches the session to a different organization.
	SwitchOrg(ctx context.Context, orgPublicID, userToken, orgToken string) (*domain.DashboardSwitchOrgResponse, error)
}

// DashboardGatewayClient defines the interface for dashboard API gateway GraphQL operations.
type DashboardGatewayClient interface {
	// ListApplications retrieves applications from the dashboard API gateway.
	ListApplications(ctx context.Context, orgPublicID, region, userToken, orgToken string) ([]domain.GatewayApplication, error)

	// CreateApplication creates a new application via the dashboard API gateway.
	CreateApplication(ctx context.Context, orgPublicID, region, name, userToken, orgToken string) (*domain.GatewayCreatedApplication, error)

	// ListAPIKeys retrieves API keys for an application.
	ListAPIKeys(ctx context.Context, appID, region, userToken, orgToken string) ([]domain.GatewayAPIKey, error)

	// CreateAPIKey creates a new API key for an application.
	CreateAPIKey(ctx context.Context, appID, region, name string, expiresInDays int, userToken, orgToken string) (*domain.GatewayCreatedAPIKey, error)
}

// DPoP defines the interface for DPoP proof generation.
type DPoP interface {
	// GenerateProof creates a DPoP proof JWT for the given HTTP method and URL.
	// If accessToken is non-empty, the proof includes an ath (access token hash) claim.
	GenerateProof(method, url string, accessToken string) (string, error)

	// Thumbprint returns the JWK thumbprint (RFC 7638) of the DPoP public key.
	Thumbprint() string
}
