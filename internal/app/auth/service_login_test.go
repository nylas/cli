package auth

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"testing"

	"github.com/nylas/cli/internal/adapters/nylas"
	"github.com/nylas/cli/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestService_Login(t *testing.T) {
	t.Run("successful login sets grant as default", func(t *testing.T) {
		client := nylas.NewMockClient()
		var capturedState string
		var capturedChallenge string
		client.BuildAuthURLFunc = func(provider domain.Provider, redirectURI, state, codeChallenge string) string {
			capturedState = state
			capturedChallenge = codeChallenge
			return "https://mock.nylas.com/auth?state=" + state
		}
		client.ExchangeCodeFunc = func(ctx context.Context, code, redirectURI, codeVerifier string) (*domain.Grant, error) {
			assert.Equal(t, "auth-code-123", code)
			assert.Equal(t, capturedChallenge, pkceChallenge(codeVerifier))
			return &domain.Grant{
				ID:       "grant-123",
				Email:    "user@example.com",
				Provider: domain.ProviderGoogle,
			}, nil
		}

		grantStore := newMockGrantStore()
		configStore := newMockConfigStore()
		server := &mockOAuthServer{
			redirectURI: "http://localhost:8080/callback",
			code:        "auth-code-123",
		}
		var stateWhenBrowserOpened string
		browser := &mockBrowser{onOpen: func() { stateWhenBrowserOpened = server.setState }}

		svc := NewService(client, grantStore, configStore, server, browser)

		grant, err := svc.Login(context.Background(), domain.ProviderGoogle)

		require.NoError(t, err)
		assert.NotNil(t, grant)
		assert.Equal(t, "grant-123", grant.ID)
		assert.Equal(t, "user@example.com", grant.Email)

		// Verify server was started and stopped
		assert.True(t, server.startCalled)
		assert.True(t, server.stopCalled)

		assert.Equal(t, capturedState, stateWhenBrowserOpened,
			"a fast redirect must be checked against this login's state, not an unset one")

		// Verify browser and callback state/PKCE values were wired through.
		assert.Equal(t, "https://mock.nylas.com/auth?state="+capturedState, browser.openedURL)
		assert.Equal(t, capturedState, server.expectedState)
		assert.NotEmpty(t, capturedState)
		assert.NotEmpty(t, capturedChallenge)

		// Verify grant was saved
		savedGrant, err := grantStore.GetGrant("grant-123")
		require.NoError(t, err)
		assert.Equal(t, "grant-123", savedGrant.ID)

		// Verify grant was set as default
		defaultID, err := grantStore.GetDefaultGrant()
		require.NoError(t, err)
		assert.Equal(t, "grant-123", defaultID)
		assert.Equal(t, "grant-123", configStore.config.DefaultGrant)
		assert.Empty(t, configStore.config.Grants)
	})

	t.Run("default persistence failure returns error", func(t *testing.T) {
		client := nylas.NewMockClient()
		client.ExchangeCodeFunc = func(ctx context.Context, code, redirectURI, codeVerifier string) (*domain.Grant, error) {
			return &domain.Grant{
				ID:       "grant-123",
				Email:    "user@example.com",
				Provider: domain.ProviderGoogle,
			}, nil
		}

		grantStore := newMockGrantStore()
		configStore := &failingSaveConfigStore{config: &domain.Config{Region: "us"}}
		server := &mockOAuthServer{
			redirectURI: "http://localhost:8080/callback",
			code:        "auth-code-123",
		}
		browser := &mockBrowser{}

		svc := NewService(client, grantStore, configStore, server, browser)

		grant, err := svc.Login(context.Background(), domain.ProviderGoogle)

		require.Error(t, err)
		assert.Nil(t, grant)
		assert.ErrorIs(t, err, domain.ErrInvalidInput)
		_, defaultErr := grantStore.GetDefaultGrant()
		assert.ErrorIs(t, defaultErr, domain.ErrNoDefaultGrant)
	})

	t.Run("existing default is mirrored to config after login", func(t *testing.T) {
		client := nylas.NewMockClient()
		client.ExchangeCodeFunc = func(ctx context.Context, code, redirectURI, codeVerifier string) (*domain.Grant, error) {
			return &domain.Grant{
				ID:       "grant-2",
				Email:    "second@example.com",
				Provider: domain.ProviderMicrosoft,
			}, nil
		}

		grantStore := newMockGrantStore()
		require.NoError(t, grantStore.SaveGrant(domain.GrantInfo{ID: "grant-1", Email: "first@example.com", Provider: domain.ProviderGoogle}))
		require.NoError(t, grantStore.SetDefaultGrant("grant-1"))
		configStore := newMockConfigStore()
		configStore.config.DefaultGrant = "stale-config-default"
		server := &mockOAuthServer{
			redirectURI: "http://localhost:8080/callback",
			code:        "auth-code-123",
		}
		browser := &mockBrowser{}

		svc := NewService(client, grantStore, configStore, server, browser)

		grant, err := svc.Login(context.Background(), domain.ProviderMicrosoft)

		require.NoError(t, err)
		require.NotNil(t, grant)
		defaultID, err := grantStore.GetDefaultGrant()
		require.NoError(t, err)
		assert.Equal(t, "grant-1", defaultID)
		assert.Equal(t, "grant-1", configStore.config.DefaultGrant)
		assert.Empty(t, configStore.config.Grants)
	})

	t.Run("server start failure returns error", func(t *testing.T) {
		server := &mockOAuthServer{
			startErr: errors.New("server start failed"),
		}
		svc := NewService(nylas.NewMockClient(), newMockGrantStore(), newMockConfigStore(), server, &mockBrowser{})

		_, err := svc.Login(context.Background(), domain.ProviderGoogle)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "server start failed")
	})

	t.Run("browser open failure returns error", func(t *testing.T) {
		browser := &mockBrowser{
			openErr: errors.New("browser open failed"),
		}
		svc := NewService(
			nylas.NewMockClient(),
			newMockGrantStore(),
			newMockConfigStore(),
			&mockOAuthServer{redirectURI: "http://localhost:8080/callback"},
			browser,
		)

		_, err := svc.Login(context.Background(), domain.ProviderGoogle)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "browser open failed")
	})

	t.Run("callback wait failure returns error", func(t *testing.T) {
		server := &mockOAuthServer{
			redirectURI: "http://localhost:8080/callback",
			waitErr:     errors.New("callback wait failed"),
		}
		svc := NewService(
			nylas.NewMockClient(),
			newMockGrantStore(),
			newMockConfigStore(),
			server,
			&mockBrowser{},
		)

		_, err := svc.Login(context.Background(), domain.ProviderGoogle)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "callback wait failed")
	})

	t.Run("code exchange failure returns error", func(t *testing.T) {
		client := nylas.NewMockClient()
		client.ExchangeCodeFunc = func(ctx context.Context, code, redirectURI, codeVerifier string) (*domain.Grant, error) {
			return nil, errors.New("code exchange failed")
		}

		server := &mockOAuthServer{
			redirectURI: "http://localhost:8080/callback",
			code:        "auth-code",
		}
		svc := NewService(client, newMockGrantStore(), newMockConfigStore(), server, &mockBrowser{})

		_, err := svc.Login(context.Background(), domain.ProviderGoogle)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "code exchange failed")
	})
}

func TestService_LoginWithCredentials(t *testing.T) {
	t.Run("successful icloud login saves grant and sets default", func(t *testing.T) {
		client := nylas.NewMockClient()
		client.CreateCustomGrantFunc = func(ctx context.Context, provider string, settings map[string]any) (*domain.Grant, error) {
			assert.Equal(t, "icloud", provider)
			assert.Equal(t, "user@icloud.com", settings["username"])
			return &domain.Grant{
				ID:          "icloud-grant-001",
				Email:       "user@icloud.com",
				Provider:    domain.ProviderICloud,
				GrantStatus: "valid",
			}, nil
		}

		grantStore := newMockGrantStore()
		configStore := newMockConfigStore()

		svc := NewService(client, grantStore, configStore, &mockOAuthServer{}, &mockBrowser{})

		grant, err := svc.LoginWithCredentials(context.Background(), "icloud", map[string]any{
			"username": "user@icloud.com",
			"password": "app-specific-password",
		})

		require.NoError(t, err)
		assert.Equal(t, "icloud-grant-001", grant.ID)
		assert.Equal(t, "user@icloud.com", grant.Email)

		savedGrant, err := grantStore.GetGrant("icloud-grant-001")
		require.NoError(t, err)
		assert.Equal(t, domain.ProviderICloud, savedGrant.Provider)

		defaultID, err := grantStore.GetDefaultGrant()
		require.NoError(t, err)
		assert.Equal(t, "icloud-grant-001", defaultID)
	})

	t.Run("does not override existing default", func(t *testing.T) {
		client := nylas.NewMockClient()
		client.CreateCustomGrantFunc = func(ctx context.Context, provider string, settings map[string]any) (*domain.Grant, error) {
			return &domain.Grant{
				ID:       "yahoo-grant-001",
				Email:    "user@yahoo.com",
				Provider: domain.ProviderIMAP,
			}, nil
		}

		grantStore := newMockGrantStore()
		require.NoError(t, grantStore.SaveGrant(domain.GrantInfo{ID: "existing-grant", Email: "first@example.com"}))
		require.NoError(t, grantStore.SetDefaultGrant("existing-grant"))
		configStore := newMockConfigStore()

		svc := NewService(client, grantStore, configStore, &mockOAuthServer{}, &mockBrowser{})

		grant, err := svc.LoginWithCredentials(context.Background(), "imap", map[string]any{
			"imap_username": "user@yahoo.com",
			"imap_password": "secret",
			"imap_host":     "imap.mail.yahoo.com",
			"imap_port":     993,
			"type":          "yahoo",
		})

		require.NoError(t, err)
		assert.Equal(t, "yahoo-grant-001", grant.ID)

		defaultID, err := grantStore.GetDefaultGrant()
		require.NoError(t, err)
		assert.Equal(t, "existing-grant", defaultID)
	})

	t.Run("api error propagates", func(t *testing.T) {
		client := nylas.NewMockClient()
		client.CreateCustomGrantFunc = func(ctx context.Context, provider string, settings map[string]any) (*domain.Grant, error) {
			return nil, errors.New("Invalid IMAP credentials")
		}

		svc := NewService(client, newMockGrantStore(), newMockConfigStore(), &mockOAuthServer{}, &mockBrowser{})

		_, err := svc.LoginWithCredentials(context.Background(), "imap", map[string]any{
			"imap_username": "bad@example.com",
			"imap_password": "wrong",
			"imap_host":     "mail.example.com",
			"imap_port":     993,
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "Invalid IMAP credentials")
	})
}

func pkceChallenge(verifier string) string {
	hash := sha256.Sum256([]byte(verifier))
	hexHash := fmt.Sprintf("%x", hash)
	return base64.RawStdEncoding.EncodeToString([]byte(hexHash))
}
