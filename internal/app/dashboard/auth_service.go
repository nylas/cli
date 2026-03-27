// Package dashboard provides the application-layer orchestration for
// dashboard authentication and application management.
package dashboard

import (
	"context"
	"fmt"

	"github.com/nylas/cli/internal/domain"
	"github.com/nylas/cli/internal/ports"
)

// AuthService orchestrates dashboard auth flows and manages token lifecycle.
type AuthService struct {
	account ports.DashboardAccountClient
	secrets ports.SecretStore
}

// NewAuthService creates a new dashboard auth service.
func NewAuthService(account ports.DashboardAccountClient, secrets ports.SecretStore) *AuthService {
	return &AuthService{
		account: account,
		secrets: secrets,
	}
}

// Register creates a new dashboard account and triggers email verification.
func (s *AuthService) Register(ctx context.Context, email, password string, privacyPolicyAccepted bool) (*domain.DashboardRegisterResponse, error) {
	return s.account.Register(ctx, email, password, privacyPolicyAccepted)
}

// VerifyEmailCode verifies the email code and stores resulting tokens.
func (s *AuthService) VerifyEmailCode(ctx context.Context, email, code, region string) (*domain.DashboardAuthResponse, error) {
	resp, err := s.account.VerifyEmailCode(ctx, email, code, region)
	if err != nil {
		return nil, err
	}

	if err := s.storeTokens(resp); err != nil {
		return nil, fmt.Errorf("failed to store credentials: %w", err)
	}
	return resp, nil
}

// ResendVerificationCode resends the email verification code.
func (s *AuthService) ResendVerificationCode(ctx context.Context, email string) error {
	return s.account.ResendVerificationCode(ctx, email)
}

// Login authenticates with email and password.
// Returns (auth, nil) on success, (nil, mfa) when MFA is required.
func (s *AuthService) Login(ctx context.Context, email, password, orgPublicID string) (*domain.DashboardAuthResponse, *domain.DashboardMFARequired, error) {
	auth, mfa, err := s.account.Login(ctx, email, password, orgPublicID)
	if err != nil {
		return nil, nil, err
	}

	if auth != nil {
		if err := s.storeTokens(auth); err != nil {
			return nil, nil, fmt.Errorf("failed to store credentials: %w", err)
		}
		return auth, nil, nil
	}

	return nil, mfa, nil
}

// CompleteMFA finishes MFA authentication and stores tokens.
func (s *AuthService) CompleteMFA(ctx context.Context, userPublicID, code, orgPublicID string) (*domain.DashboardAuthResponse, error) {
	resp, err := s.account.LoginMFA(ctx, userPublicID, code, orgPublicID)
	if err != nil {
		return nil, err
	}

	if err := s.storeTokens(resp); err != nil {
		return nil, fmt.Errorf("failed to store credentials: %w", err)
	}
	return resp, nil
}

// Refresh refreshes the session tokens using the stored tokens.
func (s *AuthService) Refresh(ctx context.Context) error {
	userToken, orgToken, err := s.loadTokens()
	if err != nil {
		return err
	}

	resp, err := s.account.Refresh(ctx, userToken, orgToken)
	if err != nil {
		return err
	}

	if err := s.secrets.Set(ports.KeyDashboardUserToken, resp.UserToken); err != nil {
		return fmt.Errorf("failed to store refreshed user token: %w", err)
	}
	if resp.OrgToken != "" {
		if err := s.secrets.Set(ports.KeyDashboardOrgToken, resp.OrgToken); err != nil {
			return fmt.Errorf("failed to store refreshed org token: %w", err)
		}
	}

	return nil
}

// Logout invalidates the session and clears local tokens.
func (s *AuthService) Logout(ctx context.Context) error {
	userToken, orgToken, _ := s.loadTokens()

	// Best effort: call the server to invalidate tokens
	if userToken != "" {
		_ = s.account.Logout(ctx, userToken, orgToken)
	}

	// Always clear local state
	s.clearTokens()
	return nil
}

// SSOStart initiates an SSO device authorization flow.
func (s *AuthService) SSOStart(ctx context.Context, loginType, mode string, privacyPolicyAccepted bool) (*domain.DashboardSSOStartResponse, error) {
	return s.account.SSOStart(ctx, loginType, mode, privacyPolicyAccepted)
}

// SSOPoll polls the SSO device flow. On completion, stores tokens.
func (s *AuthService) SSOPoll(ctx context.Context, flowID, orgPublicID string) (*domain.DashboardSSOPollResponse, error) {
	resp, err := s.account.SSOPoll(ctx, flowID, orgPublicID)
	if err != nil {
		return nil, err
	}

	if resp.Status == domain.SSOStatusComplete && resp.Auth != nil {
		if err := s.storeTokens(resp.Auth); err != nil {
			return nil, fmt.Errorf("failed to store credentials: %w", err)
		}
	}

	return resp, nil
}

// IsLoggedIn returns true if dashboard tokens exist in the keyring.
func (s *AuthService) IsLoggedIn() bool {
	token, err := s.secrets.Get(ports.KeyDashboardUserToken)
	return err == nil && token != ""
}

// Status represents the current dashboard authentication status.
type Status struct {
	LoggedIn    bool
	UserID      string
	OrgID       string
	HasOrgToken bool
}

// GetStatus returns the current dashboard auth status.
func (s *AuthService) GetStatus() Status {
	st := Status{}
	userToken, _ := s.secrets.Get(ports.KeyDashboardUserToken)
	st.LoggedIn = userToken != ""
	st.UserID, _ = s.secrets.Get(ports.KeyDashboardUserPublicID)
	st.OrgID, _ = s.secrets.Get(ports.KeyDashboardOrgPublicID)
	orgToken, _ := s.secrets.Get(ports.KeyDashboardOrgToken)
	st.HasOrgToken = orgToken != ""
	return st
}

// GetCurrentSession returns the current session info, including the active org and all orgs.
func (s *AuthService) GetCurrentSession(ctx context.Context) (*domain.DashboardSessionResponse, error) {
	userToken, orgToken, err := s.loadTokens()
	if err != nil {
		return nil, err
	}
	return s.account.GetCurrentSession(ctx, userToken, orgToken)
}

// SwitchOrg switches the active organization and stores the new org token.
func (s *AuthService) SwitchOrg(ctx context.Context, orgPublicID string) (*domain.DashboardSwitchOrgResponse, error) {
	userToken, orgToken, err := s.loadTokens()
	if err != nil {
		return nil, err
	}

	resp, err := s.account.SwitchOrg(ctx, orgPublicID, userToken, orgToken)
	if err != nil {
		return nil, err
	}

	// Store the new org token and org ID
	if resp.OrgToken != "" {
		if err := s.secrets.Set(ports.KeyDashboardOrgToken, resp.OrgToken); err != nil {
			return nil, fmt.Errorf("failed to store org token: %w", err)
		}
	}
	if resp.Org.PublicID != "" {
		if err := s.secrets.Set(ports.KeyDashboardOrgPublicID, resp.Org.PublicID); err != nil {
			return nil, fmt.Errorf("failed to store org ID: %w", err)
		}
	}

	// Clear active app since it belongs to the previous org
	_ = s.secrets.Delete(ports.KeyDashboardAppID)
	_ = s.secrets.Delete(ports.KeyDashboardAppRegion)

	return resp, nil
}

// SyncSessionOrg fetches the current session from the server and stores the
// actual active org. Call this after login to ensure the stored org matches
// the server-side default rather than guessing from the organizations list.
func (s *AuthService) SyncSessionOrg(ctx context.Context) error {
	session, err := s.GetCurrentSession(ctx)
	if err != nil {
		return fmt.Errorf("failed to fetch current dashboard session: %w", err)
	}
	if session.CurrentOrg != "" {
		if err := s.secrets.Set(ports.KeyDashboardOrgPublicID, session.CurrentOrg); err != nil {
			return fmt.Errorf("failed to store active organization: %w", err)
		}
	}
	return nil
}

// storeTokens persists auth tokens and user/org identifiers.
func (s *AuthService) storeTokens(resp *domain.DashboardAuthResponse) error {
	if err := s.secrets.Set(ports.KeyDashboardUserToken, resp.UserToken); err != nil {
		return err
	}
	if resp.OrgToken != "" {
		if err := s.secrets.Set(ports.KeyDashboardOrgToken, resp.OrgToken); err != nil {
			return err
		}
	}
	if resp.User.PublicID != "" {
		if err := s.secrets.Set(ports.KeyDashboardUserPublicID, resp.User.PublicID); err != nil {
			return err
		}
	}
	if len(resp.Organizations) == 1 {
		if err := s.secrets.Set(ports.KeyDashboardOrgPublicID, resp.Organizations[0].PublicID); err != nil {
			return err
		}
	}
	return nil
}

// SetActiveOrg updates the active organization.
func (s *AuthService) SetActiveOrg(orgPublicID string) error {
	return s.secrets.Set(ports.KeyDashboardOrgPublicID, orgPublicID)
}

// clearTokens removes all dashboard auth data from the keyring,
// including the active app selection to prevent stale state after re-login.
func (s *AuthService) clearTokens() {
	_ = s.secrets.Delete(ports.KeyDashboardUserToken)
	_ = s.secrets.Delete(ports.KeyDashboardOrgToken)
	_ = s.secrets.Delete(ports.KeyDashboardUserPublicID)
	_ = s.secrets.Delete(ports.KeyDashboardOrgPublicID)
	_ = s.secrets.Delete(ports.KeyDashboardAppID)
	_ = s.secrets.Delete(ports.KeyDashboardAppRegion)
}

// loadTokens retrieves the stored tokens.
func (s *AuthService) loadTokens() (userToken, orgToken string, err error) {
	return loadDashboardTokens(s.secrets)
}
