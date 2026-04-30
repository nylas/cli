package tui

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/nylas/cli/internal/adapters/config"
	"github.com/nylas/cli/internal/adapters/grantcache"
	"github.com/nylas/cli/internal/adapters/nylas"
	"github.com/nylas/cli/internal/domain"
)

func TestAppSwitchGrantPersistsDefaultToConfigStore(t *testing.T) {
	dir := t.TempDir()
	grantStore := grantcache.New(filepath.Join(dir, "grants.json"))
	configStore := config.NewFileStore(filepath.Join(dir, "config.yaml"))

	if err := grantStore.SaveGrant(domain.GrantInfo{ID: "grant-old", Email: "old@example.com", Provider: domain.ProviderGoogle}); err != nil {
		t.Fatalf("SaveGrant old failed: %v", err)
	}
	if err := grantStore.SaveGrant(domain.GrantInfo{ID: "grant-new", Email: "new@example.com", Provider: domain.ProviderMicrosoft}); err != nil {
		t.Fatalf("SaveGrant new failed: %v", err)
	}
	if err := grantStore.SetDefaultGrant("grant-old"); err != nil {
		t.Fatalf("SetDefaultGrant failed: %v", err)
	}
	if err := configStore.Save(&domain.Config{Region: "us", DefaultGrant: "grant-old"}); err != nil {
		t.Fatalf("config Save failed: %v", err)
	}

	app := NewApp(Config{
		Client:          nylas.NewMockClient(),
		GrantStore:      grantStore,
		ConfigStore:     configStore,
		GrantID:         "grant-old",
		Email:           "old@example.com",
		Provider:        string(domain.ProviderGoogle),
		RefreshInterval: time.Second,
		Theme:           ThemeK9s,
	})

	if err := app.SwitchGrant("grant-new", "new@example.com", string(domain.ProviderMicrosoft)); err != nil {
		t.Fatalf("SwitchGrant failed: %v", err)
	}

	defaultGrant, err := grantStore.GetDefaultGrant()
	if err != nil {
		t.Fatalf("GetDefaultGrant failed: %v", err)
	}
	if defaultGrant != "grant-new" {
		t.Fatalf("grants.json default = %q, want grant-new", defaultGrant)
	}

	cfg, err := configStore.Load()
	if err != nil {
		t.Fatalf("config Load failed: %v", err)
	}
	if cfg.DefaultGrant != "grant-new" {
		t.Fatalf("config.yaml default = %q, want grant-new", cfg.DefaultGrant)
	}
	if app.config.GrantID != "grant-new" {
		t.Fatalf("app grant ID = %q, want grant-new", app.config.GrantID)
	}
}
