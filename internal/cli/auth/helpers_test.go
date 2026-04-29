package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/nylas/cli/internal/adapters/config"
	"github.com/nylas/cli/internal/domain"
)

func TestCreateGrantServiceUsesConfiguredAPIBaseURL(t *testing.T) {
	restoreAuthHelperEnv(t)

	tempDir := t.TempDir()
	configHome := filepath.Join(tempDir, "config")
	cacheHome := filepath.Join(tempDir, "cache")
	t.Setenv("XDG_CONFIG_HOME", configHome)
	t.Setenv("XDG_CACHE_HOME", cacheHome)
	t.Setenv("HOME", tempDir)
	t.Setenv("NYLAS_DISABLE_KEYRING", "true")
	t.Setenv("NYLAS_FILE_STORE_PASSPHRASE", "test-passphrase")
	t.Setenv("NYLAS_API_KEY", "test-api-key")
	t.Setenv("NYLAS_CLIENT_ID", "")
	t.Setenv("NYLAS_CLIENT_SECRET", "")

	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		if r.URL.Path != "/v3/grants" {
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-api-key" {
			t.Fatalf("Authorization header = %q, want bearer API key", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[{"id":"grant-1","email":"user@example.com","provider":"google","grant_status":"valid"}]}`))
	}))
	defer server.Close()

	configStore := config.NewDefaultFileStore()
	if err := configStore.Save(&domain.Config{
		Region: "eu",
		API:    &domain.APIConfig{BaseURL: server.URL},
	}); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	grantSvc, _, err := createGrantService()
	if err != nil {
		t.Fatalf("createGrantService failed: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	grants, err := grantSvc.ListGrants(ctx)
	if err != nil {
		t.Fatalf("ListGrants failed: %v", err)
	}
	if requests != 1 {
		t.Fatalf("configured API base URL server received %d requests, want 1", requests)
	}
	if len(grants) != 1 || grants[0].ID != "grant-1" {
		t.Fatalf("unexpected grants: %+v", grants)
	}
}

func restoreAuthHelperEnv(t *testing.T) {
	t.Helper()
	for _, key := range []string{
		"XDG_CONFIG_HOME",
		"XDG_CACHE_HOME",
		"HOME",
		"NYLAS_DISABLE_KEYRING",
		"NYLAS_FILE_STORE_PASSPHRASE",
		"NYLAS_API_KEY",
		"NYLAS_CLIENT_ID",
		"NYLAS_CLIENT_SECRET",
	} {
		value, ok := os.LookupEnv(key)
		t.Cleanup(func() {
			if ok {
				_ = os.Setenv(key, value)
			} else {
				_ = os.Unsetenv(key)
			}
		})
	}
}
