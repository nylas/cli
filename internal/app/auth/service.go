// Package auth provides authentication-related business logic.
package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"

	"github.com/nylas/cli/internal/domain"
	"github.com/nylas/cli/internal/ports"
)

// Service handles authentication operations.
type Service struct {
	client     ports.NylasClient
	grantStore ports.GrantStore
	config     ports.ConfigStore
	server     ports.OAuthServer
	browser    ports.Browser
}

// NewService creates a new auth service.
func NewService(
	client ports.NylasClient,
	grantStore ports.GrantStore,
	config ports.ConfigStore,
	server ports.OAuthServer,
	browser ports.Browser,
) *Service {
	return &Service{
		client:     client,
		grantStore: grantStore,
		config:     config,
		server:     server,
		browser:    browser,
	}
}

// Login performs OAuth login with the specified provider.
func (s *Service) Login(ctx context.Context, provider domain.Provider) (*domain.Grant, error) {
	// Start callback server
	if err := s.server.Start(); err != nil {
		return nil, err
	}
	defer func() { _ = s.server.Stop() }()

	state, err := generateOAuthState()
	if err != nil {
		return nil, err
	}
	codeVerifier, codeChallenge, err := generatePKCEPair()
	if err != nil {
		return nil, err
	}

	redirectURI := s.server.GetRedirectURI()
	callbackCh := make(chan oauthCallbackResult, 1)
	waitCtx, waitCancel := context.WithCancel(ctx)
	defer waitCancel()

	go func() {
		code, waitErr := s.server.WaitForCallback(waitCtx, state)
		callbackCh <- oauthCallbackResult{code: code, err: waitErr}
	}()

	// Build auth URL and open browser
	authURL := s.client.BuildAuthURL(provider, redirectURI, state, codeChallenge)
	if err := s.browser.Open(authURL); err != nil {
		return nil, err
	}

	// Wait for callback
	callback := <-callbackCh
	if callback.err != nil {
		return nil, callback.err
	}

	// Exchange code for tokens
	grant, err := s.client.ExchangeCode(ctx, callback.code, redirectURI, codeVerifier)
	if err != nil {
		return nil, err
	}

	// Save grant info
	grantInfo := domain.GrantInfo{
		ID:       grant.ID,
		Email:    grant.Email,
		Provider: grant.Provider,
	}
	if err := s.grantStore.SaveGrant(grantInfo); err != nil {
		return nil, err
	}

	// Set as default if no default exists. PersistDefaultGrant writes both
	// the grant cache and config.yaml atomically so every reader (CLI, Air,
	// UI, TUI) observes the same value.
	if _, err := s.grantStore.GetDefaultGrant(); err == domain.ErrNoDefaultGrant {
		_ = PersistDefaultGrant(s.config, s.grantStore, grant.ID)
	}

	return grant, nil
}

// Logout revokes the current grant.
func (s *Service) Logout(ctx context.Context) error {
	grantID, err := s.grantStore.GetDefaultGrant()
	if err != nil {
		return err
	}

	// Revoke on Nylas
	if err := s.client.RevokeGrant(ctx, grantID); err != nil && err != domain.ErrGrantNotFound {
		return err
	}

	// Remove from local storage
	if err := s.grantStore.DeleteGrant(grantID); err != nil {
		return err
	}

	// Auto-switch to another grant if available
	s.autoSwitchDefault()

	return nil
}

// LogoutGrant revokes a specific grant.
func (s *Service) LogoutGrant(ctx context.Context, grantID string) error {
	// Check if this is the default grant
	defaultID, _ := s.grantStore.GetDefaultGrant()
	isDefault := grantID == defaultID

	// Revoke on Nylas
	if err := s.client.RevokeGrant(ctx, grantID); err != nil && err != domain.ErrGrantNotFound {
		return err
	}

	// Remove from local storage
	if err := s.grantStore.DeleteGrant(grantID); err != nil {
		return err
	}

	// Auto-switch to another grant if we deleted the default
	if isDefault {
		s.autoSwitchDefault()
	} else {
		s.syncConfigWithGrantStore()
	}

	return nil
}

// RemoveLocalGrant removes a grant from local storage without revoking it on Nylas.
func (s *Service) RemoveLocalGrant(grantID string) error {
	defaultID, _ := s.grantStore.GetDefaultGrant()
	isDefault := grantID == defaultID

	if err := s.grantStore.DeleteGrant(grantID); err != nil {
		return err
	}

	if isDefault {
		s.autoSwitchDefault()
	} else {
		s.syncConfigWithGrantStore()
	}

	return nil
}

// autoSwitchDefault sets a new default grant from remaining grants. Both
// branches route through PersistDefaultGrant so config.yaml stays in sync
// with the grant cache.
func (s *Service) autoSwitchDefault() {
	grants, err := s.grantStore.ListGrants()
	if err != nil {
		return
	}
	if len(grants) == 0 {
		// No remaining grants — clear the cache file and the mirrored default.
		_ = s.grantStore.ClearGrants()
		_ = PersistDefaultGrant(s.config, s.grantStore, "")
		return
	}
	_ = PersistDefaultGrant(s.config, s.grantStore, grants[0].ID)
}

// syncConfigWithGrantStore mirrors the grant cache's current default into
// config.yaml. Used as a defensive sync after operations that mutate the
// grant cache without changing the default (e.g. removing a non-default
// grant) — the cached value is read and re-persisted via PersistDefaultGrant
// so both stores stay in lockstep.
func (s *Service) syncConfigWithGrantStore() {
	defaultGrant, err := s.grantStore.GetDefaultGrant()
	if err == domain.ErrNoDefaultGrant {
		defaultGrant = ""
	} else if err != nil {
		return
	}
	_ = PersistDefaultGrant(s.config, s.grantStore, defaultGrant)
}

func generateOAuthState() (string, error) {
	return generateOAuthToken(32)
}

func generatePKCEPair() (string, string, error) {
	verifier, err := generateOAuthToken(32)
	if err != nil {
		return "", "", err
	}

	hash := sha256.Sum256([]byte(verifier))
	challenge := base64.RawURLEncoding.EncodeToString(hash[:])

	return verifier, challenge, nil
}

func generateOAuthToken(size int) (string, error) {
	token := make([]byte, size)
	if _, err := rand.Read(token); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(token), nil
}

type oauthCallbackResult struct {
	code string
	err  error
}
