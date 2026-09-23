package oauthas

import (
	"context"
	"time"

	"github.com/nylas/cli/internal/domain"
)

// MockClient is a configurable ports.OAuthAuthServerClient for tests.
// Unset function fields fall back to a working default so a test only has to
// state the behaviour it cares about.
type MockClient struct {
	MetadataFunc         func(ctx context.Context) (*domain.OAuthServerMetadata, error)
	AuthorizationURLFunc func(ctx context.Context, params domain.OAuthAuthorizationParams) (string, error)
	ExchangeCodeFunc     func(ctx context.Context, params domain.OAuthCodeExchange) (*domain.OAuthTokens, error)
	RefreshFunc          func(ctx context.Context, clientID, refreshToken string) (*domain.OAuthTokens, error)
	RevokeFunc           func(ctx context.Context, clientID, token string) error
	UserInfoFunc         func(ctx context.Context, accessToken string) (*domain.OAuthUserInfo, error)

	// Now backs the ExpiresAt the default token responses carry, mirroring
	// what the real client derives from expires_in. Tests with a fixed clock
	// set this to the same clock as the service under test.
	Now func() time.Time

	// Recorded calls.
	AuthorizationCalls []domain.OAuthAuthorizationParams
	ExchangeCalls      []domain.OAuthCodeExchange
	RefreshCalls       []string
	RevokeCalls        []string
}

// MockIssuer is the issuer MockClient advertises by default.
const MockIssuer = "https://auth.example.test"

func (m *MockClient) Metadata(ctx context.Context) (*domain.OAuthServerMetadata, error) {
	if m.MetadataFunc != nil {
		return m.MetadataFunc(ctx)
	}
	return &domain.OAuthServerMetadata{
		Issuer:                        MockIssuer,
		AuthorizationEndpoint:         MockIssuer + "/oauth/authorize",
		TokenEndpoint:                 MockIssuer + "/oauth/token",
		UserInfoEndpoint:              MockIssuer + "/oauth/userinfo",
		RevocationEndpoint:            MockIssuer + "/oauth/revoke",
		CodeChallengeMethodsSupported: []string{"S256"},
	}, nil
}

func (m *MockClient) AuthorizationURL(ctx context.Context, params domain.OAuthAuthorizationParams) (string, error) {
	m.AuthorizationCalls = append(m.AuthorizationCalls, params)
	if m.AuthorizationURLFunc != nil {
		return m.AuthorizationURLFunc(ctx, params)
	}
	return MockIssuer + "/oauth/authorize?state=" + params.State, nil
}

func (m *MockClient) ExchangeCode(ctx context.Context, params domain.OAuthCodeExchange) (*domain.OAuthTokens, error) {
	m.ExchangeCalls = append(m.ExchangeCalls, params)
	if m.ExchangeCodeFunc != nil {
		return m.ExchangeCodeFunc(ctx, params)
	}
	return &domain.OAuthTokens{
		AccessToken:  "mock-access-token",
		TokenType:    "Bearer",
		ExpiresIn:    3600,
		ExpiresAt:    m.expiry(3600),
		RefreshToken: "mock-refresh-token",
		IDToken:      "mock-id-token",
		Scope:        "openid email offline_access",
	}, nil
}

func (m *MockClient) Refresh(ctx context.Context, clientID, refreshToken string) (*domain.OAuthTokens, error) {
	m.RefreshCalls = append(m.RefreshCalls, refreshToken)
	if m.RefreshFunc != nil {
		return m.RefreshFunc(ctx, clientID, refreshToken)
	}
	return &domain.OAuthTokens{
		AccessToken:  "mock-access-token-2",
		TokenType:    "Bearer",
		ExpiresIn:    3600,
		ExpiresAt:    m.expiry(3600),
		RefreshToken: "mock-refresh-token-2",
		Scope:        "openid email offline_access",
	}, nil
}

func (m *MockClient) expiry(seconds int) time.Time {
	now := time.Now
	if m.Now != nil {
		now = m.Now
	}
	return now().Add(time.Duration(seconds) * time.Second)
}

func (m *MockClient) Revoke(ctx context.Context, clientID, token string) error {
	m.RevokeCalls = append(m.RevokeCalls, token)
	if m.RevokeFunc != nil {
		return m.RevokeFunc(ctx, clientID, token)
	}
	return nil
}

func (m *MockClient) UserInfo(ctx context.Context, accessToken string) (*domain.OAuthUserInfo, error) {
	if m.UserInfoFunc != nil {
		return m.UserInfoFunc(ctx, accessToken)
	}
	return &domain.OAuthUserInfo{Subject: "user-1", Email: "dev@example.test", EmailVerified: true}, nil
}
