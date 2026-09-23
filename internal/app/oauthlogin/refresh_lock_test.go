//go:build !integration

package oauthlogin

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/nylas/cli/internal/adapters/filelock"
	"github.com/nylas/cli/internal/adapters/keyring"
	"github.com/nylas/cli/internal/adapters/oauth"
	"github.com/nylas/cli/internal/adapters/oauthas"
	"github.com/nylas/cli/internal/domain"
	"github.com/nylas/cli/internal/ports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// rotatingTokenServer behaves like the Nylas authorization server's refresh
// grant: every use rotates the refresh token, and presenting one that was
// already consumed revokes the whole family, after which nothing refreshes.
type rotatingTokenServer struct {
	mu       sync.Mutex
	current  string
	serial   int
	requests int
	burned   bool
}

func (s *rotatingTokenServer) handler(issuer func() string) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/.well-known/oauth-authorization-server", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"issuer":                           issuer(),
			"authorization_endpoint":           issuer() + "/oauth/authorize",
			"token_endpoint":                   issuer() + "/oauth/token",
			"code_challenge_methods_supported": []string{"S256"},
		})
	})
	mux.HandleFunc("/oauth/token", func(w http.ResponseWriter, r *http.Request) {
		// Widen the window a racing refresher would need to slip through.
		time.Sleep(20 * time.Millisecond)

		s.mu.Lock()
		defer s.mu.Unlock()
		s.requests++

		w.Header().Set("Content-Type", "application/json")
		presented := r.FormValue("refresh_token")
		if s.burned || presented != s.current {
			s.burned = true
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"error":"invalid_grant","error_description":"refresh token reuse detected"}`))
			return
		}

		s.serial++
		s.current = fmt.Sprintf("rt-%d", s.serial)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"access_token":  fmt.Sprintf("at-%d", s.serial),
			"token_type":    "Bearer",
			"expires_in":    900,
			"refresh_token": s.current,
			"scope":         "openid email offline_access",
		})
	})
	return mux
}

func (s *rotatingTokenServer) snapshot() (requests int, burned bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.requests, s.burned
}

// seedExpiredSession stores a session whose access token has expired, which
// is the state every `nylas mcp serve` process meets at the same moment.
func seedExpiredSession(t *testing.T, secrets ports.SecretStore, issuer string) {
	t.Helper()
	require.NoError(t, secrets.Set(ports.KeyOAuthIssuer, issuer))
	require.NoError(t, secrets.Set(ports.KeyOAuthRefreshToken, "rt-0"))
	require.NoError(t, secrets.Set(ports.KeyOAuthExpiresAt, time.Now().Add(-time.Minute).UTC().Format(time.RFC3339)))
	require.NoError(t, secrets.Set(ports.KeyOAuthAccessToken, "at-0"))
}

func TestAccessToken_ConcurrentRefreshersSpendTheRefreshTokenOnce(t *testing.T) {
	// Several `nylas mcp serve` processes share one keyring session. Each one
	// here is a separate Service with its OWN lock handle on the same file and
	// its OWN HTTP client — the same isolation separate processes have (flock
	// excludes separate open file descriptions, not only separate PIDs;
	// filelock's own tests prove it across real processes).
	const refreshers = 12

	tokenServer := &rotatingTokenServer{current: "rt-0"}
	var httpServer *httptest.Server
	httpServer = httptest.NewServer(tokenServer.handler(func() string { return httpServer.URL }))
	t.Cleanup(httpServer.Close)

	secrets := keyring.NewMockSecretStore()
	seedExpiredSession(t, secrets, httpServer.URL)
	lockPath := filepath.Join(t.TempDir(), "oauth-session.lock")

	start := make(chan struct{})
	results := make([]string, refreshers)
	errs := make([]error, refreshers)
	var wg sync.WaitGroup
	for i := range refreshers {
		svc := NewService(
			domain.DefaultOAuthClientID,
			oauthas.NewClient(httpServer.URL),
			oauth.NewMockServer("unused"),
			&mockBrowser{},
			secrets,
			filelock.New(lockPath),
		)
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			results[i], errs[i] = svc.AccessToken(context.Background())
		}()
	}
	close(start)
	wg.Wait()

	requests, burned := tokenServer.snapshot()
	assert.Equal(t, 1, requests, "exactly one process may spend the refresh token")
	assert.False(t, burned, "a replayed refresh token revokes the family and signs everyone out")
	for i := range refreshers {
		require.NoError(t, errs[i], "refresher %d", i)
		assert.Equal(t, "at-1", results[i], "refresher %d must end with the one rotated token", i)
	}
	assert.Equal(t, "rt-1", secrets.GetAll()[ports.KeyOAuthRefreshToken])
}

func TestAccessToken_WithoutTheLockRefreshersReplay(t *testing.T) {
	// The control for the test above: the same race with the lock bypassed
	// really does burn the family, so the passing test is proving the lock
	// and not a lucky schedule.
	tokenServer := &rotatingTokenServer{current: "rt-0"}
	var httpServer *httptest.Server
	httpServer = httptest.NewServer(tokenServer.handler(func() string { return httpServer.URL }))
	t.Cleanup(httpServer.Close)

	secrets := keyring.NewMockSecretStore()
	seedExpiredSession(t, secrets, httpServer.URL)

	start := make(chan struct{})
	var wg sync.WaitGroup
	for range 4 {
		svc := NewService(domain.DefaultOAuthClientID, oauthas.NewClient(httpServer.URL),
			oauth.NewMockServer("unused"), &mockBrowser{}, secrets, noopLock{})
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			_, _ = svc.AccessToken(context.Background())
		}()
	}
	close(start)
	wg.Wait()

	requests, burned := tokenServer.snapshot()
	assert.Greater(t, requests, 1)
	assert.True(t, burned)
}

// noopLock grants every caller at once, which is what "no lock" means.
type noopLock struct{}

func (noopLock) Lock(context.Context) (func() error, error) { return func() error { return nil }, nil }

func TestAccessToken_UsesTokenAnotherProcessRefreshedWhileWaiting(t *testing.T) {
	// The waiter reads a stale, expired session, blocks on the lock, and by
	// the time it holds it another process has stored rotated tokens.
	f := newFixture(t)
	require.NoError(t, f.secrets.Set(ports.KeyOAuthIssuer, oauthas.MockIssuer))
	require.NoError(t, f.secrets.Set(ports.KeyOAuthRefreshToken, "rt-old"))
	require.NoError(t, f.secrets.Set(ports.KeyOAuthExpiresAt, f.clock.Add(-time.Minute).Format(time.RFC3339)))
	require.NoError(t, f.secrets.Set(ports.KeyOAuthAccessToken, "at-old"))

	holder, err := f.lock.Lock(context.Background())
	require.NoError(t, err)

	done := make(chan struct{})
	var token string
	var tokenErr error
	go func() {
		defer close(done)
		token, tokenErr = f.service.AccessToken(context.Background())
	}()

	// "Another process" refreshes and releases the lock.
	time.Sleep(50 * time.Millisecond)
	require.NoError(t, f.secrets.Set(ports.KeyOAuthRefreshToken, "rt-new"))
	require.NoError(t, f.secrets.Set(ports.KeyOAuthExpiresAt, f.clock.Add(15*time.Minute).Format(time.RFC3339)))
	require.NoError(t, f.secrets.Set(ports.KeyOAuthAccessToken, "at-new"))
	require.NoError(t, holder())
	<-done

	require.NoError(t, tokenErr)
	assert.Equal(t, "at-new", token)
	assert.Empty(t, f.client.RefreshCalls, "rt-old was already spent by the other process")
}

func TestAccessToken_DoesNotRefreshWhenTheLockCannotBeTaken(t *testing.T) {
	f := newFixture(t)
	_, err := f.service.Login(context.Background(), nil)
	require.NoError(t, err)
	f.advance(2 * time.Hour)

	holder, err := f.lock.Lock(context.Background())
	require.NoError(t, err)
	t.Cleanup(func() { _ = holder() })

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	_, err = f.service.AccessToken(ctx)

	require.ErrorIs(t, err, context.DeadlineExceeded)
	assert.Empty(t, f.client.RefreshCalls, "refreshing without the lock is the race it exists to stop")
}

func TestAccessToken_FailsClosedWithoutALock(t *testing.T) {
	f := newFixture(t)
	_, err := f.service.Login(context.Background(), nil)
	require.NoError(t, err)
	f.advance(2 * time.Hour)
	f.service.lock = nil

	_, err = f.service.AccessToken(context.Background())

	require.Error(t, err)
	assert.Empty(t, f.client.RefreshCalls)
}

func TestLogout_WaitsForAnInFlightRefresh(t *testing.T) {
	// A refresh that lands after logout would write a live session back into
	// a keyring the user just asked to be cleared.
	f := newFixture(t)
	_, err := f.service.Login(context.Background(), nil)
	require.NoError(t, err)

	holder, err := f.lock.Lock(context.Background())
	require.NoError(t, err)

	done := make(chan error, 1)
	go func() { done <- f.service.Logout(context.Background()) }()

	select {
	case <-done:
		t.Fatal("logout must wait for the session lock")
	case <-time.After(60 * time.Millisecond):
	}
	require.NoError(t, holder())
	require.NoError(t, <-done)
	assert.NotContains(t, f.secrets.GetAll(), ports.KeyOAuthAccessToken)
}

func TestAccessToken_RefreshFailureSurfacesTheOAuthError(t *testing.T) {
	f := newFixture(t)
	_, err := f.service.Login(context.Background(), nil)
	require.NoError(t, err)
	f.advance(2 * time.Hour)
	f.client.RefreshFunc = func(context.Context, string, string) (*domain.OAuthTokens, error) {
		return nil, &domain.OAuthError{Code: "invalid_grant", StatusCode: http.StatusBadRequest}
	}

	_, err = f.service.AccessToken(context.Background())

	var oauthErr *domain.OAuthError
	require.True(t, errors.As(err, &oauthErr))
	assert.Equal(t, "invalid_grant", oauthErr.Code)
}
