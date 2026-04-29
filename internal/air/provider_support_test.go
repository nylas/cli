package air

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	configmock "github.com/nylas/cli/internal/adapters/config"
	keyringmock "github.com/nylas/cli/internal/adapters/keyring"
	"github.com/nylas/cli/internal/air/cache"
	authapp "github.com/nylas/cli/internal/app/auth"
	"github.com/nylas/cli/internal/domain"
)

func newPageDataTestServer(grants []domain.GrantInfo, defaultGrant string) *Server {
	return &Server{
		configSvc: authapp.NewConfigService(
			configmock.NewMockConfigStore(),
			keyringmock.NewMockSecretStore(),
		),
		grantStore: &testGrantStore{
			grants:       grants,
			defaultGrant: defaultGrant,
		},
		hasAPIKey: true,
	}
}

func TestBuildPageData_SelectsSupportedDefaultGrant(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                string
		defaultGrant        string
		wantDefaultGrantID  string
		wantUserEmail       string
		wantProvider        string
		wantSupportedGrants int
	}{
		{
			name:                "nylas default remains selected",
			defaultGrant:        "grant-nylas",
			wantDefaultGrantID:  "grant-nylas",
			wantUserEmail:       "nylas@example.com",
			wantProvider:        string(domain.ProviderNylas),
			wantSupportedGrants: 2,
		},
		{
			name:                "unsupported default falls back to first supported grant",
			defaultGrant:        "grant-imap",
			wantDefaultGrantID:  "grant-google",
			wantUserEmail:       "google@example.com",
			wantProvider:        string(domain.ProviderGoogle),
			wantSupportedGrants: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			server := newPageDataTestServer([]domain.GrantInfo{
				{ID: "grant-google", Email: "google@example.com", Provider: domain.ProviderGoogle},
				{ID: "grant-nylas", Email: "nylas@example.com", Provider: domain.ProviderNylas},
				{ID: "grant-imap", Email: "imap@example.com", Provider: domain.ProviderIMAP},
			}, tt.defaultGrant)

			data := server.buildPageData()

			if !data.Configured {
				t.Fatal("expected page data to be configured")
			}
			if data.DefaultGrantID != tt.wantDefaultGrantID {
				t.Fatalf("expected default grant %q, got %q", tt.wantDefaultGrantID, data.DefaultGrantID)
			}
			if data.UserEmail != tt.wantUserEmail {
				t.Fatalf("expected user email %q, got %q", tt.wantUserEmail, data.UserEmail)
			}
			if data.Provider != tt.wantProvider {
				t.Fatalf("expected provider %q, got %q", tt.wantProvider, data.Provider)
			}
			if len(data.Grants) != tt.wantSupportedGrants {
				t.Fatalf("expected %d supported grants, got %d", tt.wantSupportedGrants, len(data.Grants))
			}
			if data.AccountsCount != tt.wantSupportedGrants {
				t.Fatalf("expected %d accounts, got %d", tt.wantSupportedGrants, data.AccountsCount)
			}

			for _, grant := range data.Grants {
				if grant.Provider == string(domain.ProviderIMAP) {
					t.Fatal("did not expect imap grant in page data")
				}
			}
		})
	}
}

func TestHandleCacheSync_FiltersSupportedProviders(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		query     string
		wantCount int
	}{
		{
			name:      "syncs only default provider when no account requested",
			query:     "/api/cache/sync",
			wantCount: 1,
		},
		{
			name:      "syncs only requested supported provider",
			query:     "/api/cache/sync?email=nylas@example.com",
			wantCount: 1,
		},
		{
			name:      "does not sync requested unsupported provider",
			query:     "/api/cache/sync?email=virtual@example.com",
			wantCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			manager, err := cache.NewManager(cache.Config{BasePath: t.TempDir()})
			if err != nil {
				t.Fatalf("new cache manager: %v", err)
			}
			t.Cleanup(func() {
				_ = manager.Close()
			})

			server := &Server{
				cacheManager: manager,
				grantStore: &testGrantStore{
					grants: []domain.GrantInfo{
						{ID: "grant-google", Email: "google@example.com", Provider: domain.ProviderGoogle},
						{ID: "grant-nylas", Email: "nylas@example.com", Provider: domain.ProviderNylas},
						{ID: "grant-virtual", Email: "virtual@example.com", Provider: domain.ProviderVirtual},
						{ID: "grant-imap", Email: "imap@example.com", Provider: domain.ProviderIMAP},
					},
					defaultGrant: "grant-google",
				},
				isOnline: true,
			}

			req := httptest.NewRequest(http.MethodPost, tt.query, nil)
			w := httptest.NewRecorder()

			server.handleCacheSync(w, req)

			if w.Code != http.StatusOK {
				t.Fatalf("expected status 200, got %d", w.Code)
			}

			var resp CacheSyncResponse
			if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
				t.Fatalf("decode response: %v", err)
			}

			if !resp.Success {
				t.Fatalf("expected success response, got error %q", resp.Error)
			}

			wantMessage := fmt.Sprintf("Synced %d account(s)", tt.wantCount)
			if resp.Message != wantMessage {
				t.Fatalf("expected message %q, got %q", wantMessage, resp.Message)
			}
		})
	}
}
