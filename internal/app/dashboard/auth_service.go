// Package dashboard provides the application-layer orchestration for
// dashboard authentication and application management.
package dashboard

import (
	"context"
	"errors"
	"fmt"

	"github.com/nylas/cli/internal/domain"
	"github.com/nylas/cli/internal/ports"
)

// AuthService orchestrates dashboard auth flows and manages token lifecycle.
type AuthService struct {
	account ports.DashboardAccountClient
	secrets ports.SecretStore
}

var (
	dashboardSessionStateKeys = []string{
		ports.KeyDashboardUserToken,
		ports.KeyDashboardOrgToken,
		ports.KeyDashboardUserPublicID,
		ports.KeyDashboardOrgPublicID,
		ports.KeyDashboardAppID,
		ports.KeyDashboardAppRegion,
	}
	dashboardRefreshStateKeys = []string{
		ports.KeyDashboardUserToken,
		ports.KeyDashboardOrgToken,
	}
	dashboardSwitchOrgStateKeys = []string{
		ports.KeyDashboardOrgToken,
		ports.KeyDashboardOrgPublicID,
		ports.KeyDashboardAppID,
		ports.KeyDashboardAppRegion,
	}
)

type secretSnapshot struct {
	value   string
	present bool
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

	_, _, err = s.refreshTokens(ctx, userToken, orgToken)
	return err
}

// Logout invalidates the session and clears local tokens.
func (s *AuthService) Logout(ctx context.Context) error {
	userToken, orgToken, _ := s.loadTokens()

	// Best effort: call the server to invalidate tokens
	if userToken != "" {
		_ = s.account.Logout(ctx, userToken, orgToken)
	}

	// Always clear local state
	return s.clearTokens()
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

	session, err := s.account.GetCurrentSession(ctx, userToken, orgToken)
	if !errors.Is(err, domain.ErrDashboardSessionExpired) {
		return session, err
	}

	userToken, orgToken, err = s.refreshTokens(ctx, userToken, orgToken)
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
	if errors.Is(err, domain.ErrDashboardSessionExpired) {
		userToken, orgToken, err = s.refreshTokens(ctx, userToken, orgToken)
		if err != nil {
			return nil, err
		}
		resp, err = s.account.SwitchOrg(ctx, orgPublicID, userToken, orgToken)
	}
	if err != nil {
		return nil, err
	}

	nextOrgID := orgPublicID
	if resp.Org.PublicID != "" {
		nextOrgID = resp.Org.PublicID
	}
	if err := s.replaceSecretValues(dashboardSwitchOrgStateKeys, map[string]*string{
		ports.KeyDashboardOrgToken:    stringPtrOrNil(resp.OrgToken),
		ports.KeyDashboardOrgPublicID: stringPtrOrNil(nextOrgID),
		ports.KeyDashboardAppID:       nil,
		ports.KeyDashboardAppRegion:   nil,
	}); err != nil {
		return nil, fmt.Errorf("failed to persist organization switch: %w", err)
	}

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
	orgPublicID := ""
	if len(resp.Organizations) == 1 {
		orgPublicID = resp.Organizations[0].PublicID
	}

	return s.replaceSecretValues(dashboardSessionStateKeys, map[string]*string{
		ports.KeyDashboardUserToken:    stringPtrOrNil(resp.UserToken),
		ports.KeyDashboardOrgToken:     stringPtrOrNil(resp.OrgToken),
		ports.KeyDashboardUserPublicID: stringPtrOrNil(resp.User.PublicID),
		ports.KeyDashboardOrgPublicID:  stringPtrOrNil(orgPublicID),
		ports.KeyDashboardAppID:        nil,
		ports.KeyDashboardAppRegion:    nil,
	})
}

// SetActiveOrg updates the active organization.
func (s *AuthService) SetActiveOrg(orgPublicID string) error {
	return s.secrets.Set(ports.KeyDashboardOrgPublicID, orgPublicID)
}

// clearTokens removes all dashboard auth data from the keyring,
// including the active app selection to prevent stale state after re-login.
func (s *AuthService) clearTokens() error {
	var errs []error
	for _, key := range dashboardSessionStateKeys {
		if err := s.secrets.Delete(key); err != nil {
			errs = append(errs, fmt.Errorf("failed to clear %s: %w", key, err))
		}
	}
	return errors.Join(errs...)
}

// loadTokens retrieves the stored tokens.
func (s *AuthService) loadTokens() (userToken, orgToken string, err error) {
	return loadDashboardTokens(s.secrets)
}

func (s *AuthService) refreshTokens(ctx context.Context, userToken, orgToken string) (string, string, error) {
	resp, err := s.account.Refresh(ctx, userToken, orgToken)
	if err != nil {
		return "", "", err
	}

	updates := map[string]*string{
		ports.KeyDashboardUserToken: stringPtrOrNil(resp.UserToken),
	}
	if resp.OrgToken != "" {
		updates[ports.KeyDashboardOrgToken] = stringPtrOrNil(resp.OrgToken)
	}
	if err := s.replaceSecretValues(dashboardRefreshStateKeys, updates); err != nil {
		return "", "", fmt.Errorf("failed to store refreshed credentials: %w", err)
	}
	userToken = resp.UserToken
	if resp.OrgToken != "" {
		orgToken = resp.OrgToken
	}

	return userToken, orgToken, nil
}

func (s *AuthService) replaceSecretValues(keys []string, updates map[string]*string) error {
	snapshot, err := s.snapshotSecretValues(keys)
	if err != nil {
		return err
	}
	if err := s.applySecretValues(keys, updates); err != nil {
		if rollbackErr := s.restoreSecretValues(keys, snapshot); rollbackErr != nil {
			return errors.Join(err, fmt.Errorf("failed to rollback dashboard session state: %w", rollbackErr))
		}
		return err
	}
	return nil
}

func (s *AuthService) snapshotSecretValues(keys []string) (map[string]secretSnapshot, error) {
	snapshot := make(map[string]secretSnapshot, len(keys))
	for _, key := range keys {
		value, err := s.secrets.Get(key)
		switch {
		case err == nil:
			snapshot[key] = secretSnapshot{value: value, present: true}
		case errors.Is(err, domain.ErrSecretNotFound):
			snapshot[key] = secretSnapshot{}
		default:
			return nil, fmt.Errorf("failed to read %s: %w", key, err)
		}
	}
	return snapshot, nil
}

func (s *AuthService) applySecretValues(keys []string, updates map[string]*string) error {
	for _, key := range keys {
		value, ok := updates[key]
		if !ok {
			continue
		}
		if value == nil {
			if err := s.secrets.Delete(key); err != nil {
				return fmt.Errorf("failed to clear %s: %w", key, err)
			}
			continue
		}
		if err := s.secrets.Set(key, *value); err != nil {
			return fmt.Errorf("failed to store %s: %w", key, err)
		}
	}
	return nil
}

func (s *AuthService) restoreSecretValues(keys []string, snapshot map[string]secretSnapshot) error {
	var errs []error
	for _, key := range keys {
		prev := snapshot[key]
		var err error
		if prev.present {
			err = s.secrets.Set(key, prev.value)
		} else {
			err = s.secrets.Delete(key)
		}
		if err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", key, err))
		}
	}
	return errors.Join(errs...)
}

func stringPtrOrNil(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}
