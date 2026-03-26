// Package dashboard implements clients for the Nylas Dashboard account
// and API gateway services.
package dashboard

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/nylas/cli/internal/domain"
	"github.com/nylas/cli/internal/ports"
)

// AccountClient implements ports.DashboardAccountClient for the
// dashboard-account CLI auth endpoints.
type AccountClient struct {
	baseURL    string
	httpClient *http.Client
	dpop       ports.DPoP
}

// NewAccountClient creates a new dashboard account client.
func NewAccountClient(baseURL string, dpop ports.DPoP) *AccountClient {
	return &AccountClient{
		baseURL:    baseURL,
		httpClient: newNonRedirectClient(),
		dpop:       dpop,
	}
}

// Register creates a new dashboard account and triggers email verification.
func (c *AccountClient) Register(ctx context.Context, email, password string, privacyPolicyAccepted bool) (*domain.DashboardRegisterResponse, error) {
	body := map[string]any{
		"email":                 email,
		"password":              password,
		"privacyPolicyAccepted": privacyPolicyAccepted,
	}

	var result domain.DashboardRegisterResponse
	if err := c.doPost(ctx, "/auth/cli/register", body, nil, "", &result); err != nil {
		return nil, fmt.Errorf("registration failed: %w", err)
	}
	return &result, nil
}

// VerifyEmailCode verifies the email verification code after registration.
func (c *AccountClient) VerifyEmailCode(ctx context.Context, email, code, region string) (*domain.DashboardAuthResponse, error) {
	body := map[string]any{
		"email":  email,
		"code":   code,
		"region": region,
	}

	var result domain.DashboardAuthResponse
	if err := c.doPost(ctx, "/auth/cli/verify-email-code", body, nil, "", &result); err != nil {
		return nil, fmt.Errorf("verification code invalid or expired: %w", err)
	}
	return &result, nil
}

// ResendVerificationCode resends the email verification code.
func (c *AccountClient) ResendVerificationCode(ctx context.Context, email string) error {
	body := map[string]any{"email": email}
	return c.doPost(ctx, "/auth/cli/resend-verification-code", body, nil, "", nil)
}

// Login authenticates with email and password.
func (c *AccountClient) Login(ctx context.Context, email, password, orgPublicID string) (*domain.DashboardAuthResponse, *domain.DashboardMFARequired, error) {
	body := map[string]any{
		"email":    email,
		"password": password,
	}
	if orgPublicID != "" {
		body["orgPublicId"] = orgPublicID
	}

	raw, err := c.doPostRaw(ctx, "/auth/cli/login", body, nil, "")
	if err != nil {
		return nil, nil, fmt.Errorf("%w", domain.ErrDashboardLoginFailed)
	}

	// Check if response contains userToken (success) or totpFactor (MFA required)
	var probe struct {
		UserToken  string `json:"userToken"`
		TOTPFactor any    `json:"totpFactor"`
	}
	if err := json.Unmarshal(raw, &probe); err != nil {
		return nil, nil, fmt.Errorf("failed to parse login response: %w", err)
	}

	if probe.UserToken != "" {
		var auth domain.DashboardAuthResponse
		if err := json.Unmarshal(raw, &auth); err != nil {
			return nil, nil, fmt.Errorf("failed to parse auth response: %w", err)
		}
		return &auth, nil, nil
	}

	if probe.TOTPFactor != nil {
		var mfa domain.DashboardMFARequired
		if err := json.Unmarshal(raw, &mfa); err != nil {
			return nil, nil, fmt.Errorf("failed to parse MFA response: %w", err)
		}
		return nil, &mfa, nil
	}

	return nil, nil, fmt.Errorf("%w", domain.ErrDashboardLoginFailed)
}

// LoginMFA completes MFA authentication with a TOTP code.
func (c *AccountClient) LoginMFA(ctx context.Context, userPublicID, code, orgPublicID string) (*domain.DashboardAuthResponse, error) {
	body := map[string]any{
		"userPublicId": userPublicID,
		"code":         code,
	}
	if orgPublicID != "" {
		body["orgPublicId"] = orgPublicID
	}

	var result domain.DashboardAuthResponse
	if err := c.doPost(ctx, "/auth/cli/login/mfa", body, nil, "", &result); err != nil {
		return nil, fmt.Errorf("%w", domain.ErrDashboardLoginFailed)
	}
	return &result, nil
}

// Refresh refreshes the session tokens.
func (c *AccountClient) Refresh(ctx context.Context, userToken, orgToken string) (*domain.DashboardRefreshResponse, error) {
	headers := bearerHeaders(userToken, orgToken)
	var result domain.DashboardRefreshResponse
	if err := c.doPost(ctx, "/auth/cli/refresh", nil, headers, userToken, &result); err != nil {
		return nil, fmt.Errorf("%w", domain.ErrDashboardSessionExpired)
	}
	return &result, nil
}

// Logout invalidates the session tokens.
func (c *AccountClient) Logout(ctx context.Context, userToken, orgToken string) error {
	headers := bearerHeaders(userToken, orgToken)
	return c.doPost(ctx, "/auth/cli/logout", nil, headers, userToken, nil)
}

// SSOStart initiates an SSO device authorization flow.
func (c *AccountClient) SSOStart(ctx context.Context, loginType, mode string, privacyPolicyAccepted bool) (*domain.DashboardSSOStartResponse, error) {
	body := map[string]any{
		"loginType": loginType,
		"mode":      mode,
	}
	if mode == "register" {
		body["privacyPolicyAccepted"] = privacyPolicyAccepted
	}

	var result domain.DashboardSSOStartResponse
	if err := c.doPost(ctx, "/auth/cli/sso/start", body, nil, "", &result); err != nil {
		return nil, fmt.Errorf("%w: %w", domain.ErrDashboardSSOFailed, err)
	}
	return &result, nil
}

// SSOPoll polls the SSO device flow for completion.
func (c *AccountClient) SSOPoll(ctx context.Context, flowID, orgPublicID string) (*domain.DashboardSSOPollResponse, error) {
	body := map[string]any{
		"flowId": flowID,
	}
	if orgPublicID != "" {
		body["orgPublicId"] = orgPublicID
	}

	raw, err := c.doPostRaw(ctx, "/auth/cli/sso/poll", body, nil, "")
	if err != nil {
		return nil, fmt.Errorf("%w: %w", domain.ErrDashboardSSOFailed, err)
	}

	var result domain.DashboardSSOPollResponse
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, fmt.Errorf("failed to parse SSO poll response: %w", err)
	}

	switch result.Status {
	case domain.SSOStatusComplete:
		var auth domain.DashboardAuthResponse
		if err := json.Unmarshal(raw, &auth); err != nil {
			return nil, fmt.Errorf("failed to parse SSO auth: %w", err)
		}
		result.Auth = &auth

	case domain.SSOStatusMFARequired:
		var mfa domain.DashboardMFARequired
		if err := json.Unmarshal(raw, &mfa); err != nil {
			return nil, fmt.Errorf("failed to parse SSO MFA: %w", err)
		}
		result.MFA = &mfa
	}

	return &result, nil
}

// GetCurrentSession returns the current session info including the active org.
func (c *AccountClient) GetCurrentSession(ctx context.Context, userToken, orgToken string) (*domain.DashboardSessionResponse, error) {
	headers := bearerHeaders(userToken, orgToken)
	var result domain.DashboardSessionResponse
	if err := c.doGet(ctx, "/sessions/current", headers, userToken, &result); err != nil {
		return nil, fmt.Errorf("failed to get current session: %w", err)
	}
	return &result, nil
}

// SwitchOrg switches the session to a different organization.
func (c *AccountClient) SwitchOrg(ctx context.Context, orgPublicID, userToken, orgToken string) (*domain.DashboardSwitchOrgResponse, error) {
	body := map[string]any{
		"orgPublicId": orgPublicID,
	}
	headers := bearerHeaders(userToken, orgToken)
	var result domain.DashboardSwitchOrgResponse
	if err := c.doPost(ctx, "/sessions/switch-org", body, headers, userToken, &result); err != nil {
		return nil, fmt.Errorf("failed to switch organization: %w", err)
	}
	return &result, nil
}

// bearerHeaders creates the Authorization and X-Nylas-Org headers.
func bearerHeaders(userToken, orgToken string) map[string]string {
	h := map[string]string{
		"Authorization": "Bearer " + userToken,
	}
	if orgToken != "" {
		h["X-Nylas-Org"] = orgToken
	}
	return h
}
