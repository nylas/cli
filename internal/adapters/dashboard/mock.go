package dashboard

import (
	"context"

	"github.com/nylas/cli/internal/domain"
)

// MockAccountClient is a test mock for ports.DashboardAccountClient.
type MockAccountClient struct {
	RegisterFn               func(ctx context.Context, email, password string, privacyPolicyAccepted bool) (*domain.DashboardRegisterResponse, error)
	VerifyEmailCodeFn        func(ctx context.Context, email, code, region string) (*domain.DashboardAuthResponse, error)
	ResendVerificationCodeFn func(ctx context.Context, email string) error
	LoginFn                  func(ctx context.Context, email, password, orgPublicID string) (*domain.DashboardAuthResponse, *domain.DashboardMFARequired, error)
	LoginMFAFn               func(ctx context.Context, userPublicID, code, orgPublicID string) (*domain.DashboardAuthResponse, error)
	RefreshFn                func(ctx context.Context, userToken, orgToken string) (*domain.DashboardRefreshResponse, error)
	LogoutFn                 func(ctx context.Context, userToken, orgToken string) error
	SSOStartFn               func(ctx context.Context, loginType, mode string, privacyPolicyAccepted bool) (*domain.DashboardSSOStartResponse, error)
	SSOPollFn                func(ctx context.Context, flowID, orgPublicID string) (*domain.DashboardSSOPollResponse, error)
}

func (m *MockAccountClient) Register(ctx context.Context, email, password string, privacyPolicyAccepted bool) (*domain.DashboardRegisterResponse, error) {
	return m.RegisterFn(ctx, email, password, privacyPolicyAccepted)
}
func (m *MockAccountClient) VerifyEmailCode(ctx context.Context, email, code, region string) (*domain.DashboardAuthResponse, error) {
	return m.VerifyEmailCodeFn(ctx, email, code, region)
}
func (m *MockAccountClient) ResendVerificationCode(ctx context.Context, email string) error {
	return m.ResendVerificationCodeFn(ctx, email)
}
func (m *MockAccountClient) Login(ctx context.Context, email, password, orgPublicID string) (*domain.DashboardAuthResponse, *domain.DashboardMFARequired, error) {
	return m.LoginFn(ctx, email, password, orgPublicID)
}
func (m *MockAccountClient) LoginMFA(ctx context.Context, userPublicID, code, orgPublicID string) (*domain.DashboardAuthResponse, error) {
	return m.LoginMFAFn(ctx, userPublicID, code, orgPublicID)
}
func (m *MockAccountClient) Refresh(ctx context.Context, userToken, orgToken string) (*domain.DashboardRefreshResponse, error) {
	return m.RefreshFn(ctx, userToken, orgToken)
}
func (m *MockAccountClient) Logout(ctx context.Context, userToken, orgToken string) error {
	return m.LogoutFn(ctx, userToken, orgToken)
}
func (m *MockAccountClient) SSOStart(ctx context.Context, loginType, mode string, privacyPolicyAccepted bool) (*domain.DashboardSSOStartResponse, error) {
	return m.SSOStartFn(ctx, loginType, mode, privacyPolicyAccepted)
}
func (m *MockAccountClient) SSOPoll(ctx context.Context, flowID, orgPublicID string) (*domain.DashboardSSOPollResponse, error) {
	return m.SSOPollFn(ctx, flowID, orgPublicID)
}

// MockGatewayClient is a test mock for ports.DashboardGatewayClient.
type MockGatewayClient struct {
	ListApplicationsFn  func(ctx context.Context, orgPublicID, region, userToken, orgToken string) ([]domain.GatewayApplication, error)
	CreateApplicationFn func(ctx context.Context, orgPublicID, region, name, userToken, orgToken string) (*domain.GatewayCreatedApplication, error)
}

func (m *MockGatewayClient) ListApplications(ctx context.Context, orgPublicID, region, userToken, orgToken string) ([]domain.GatewayApplication, error) {
	return m.ListApplicationsFn(ctx, orgPublicID, region, userToken, orgToken)
}
func (m *MockGatewayClient) CreateApplication(ctx context.Context, orgPublicID, region, name, userToken, orgToken string) (*domain.GatewayCreatedApplication, error) {
	return m.CreateApplicationFn(ctx, orgPublicID, region, name, userToken, orgToken)
}
