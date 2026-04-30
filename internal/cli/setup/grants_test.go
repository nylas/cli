package setup

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/nylas/cli/internal/adapters/config"
	"github.com/nylas/cli/internal/adapters/grantcache"
	"github.com/nylas/cli/internal/domain"
)

var errPersistDefaultGrant = errors.New("persist default grant failed")

type fakeGrantLister struct {
	grants []domain.Grant
	params *domain.GrantsQueryParams
}

func (f *fakeGrantLister) ListAllGrants(_ context.Context, params *domain.GrantsQueryParams) ([]domain.Grant, error) {
	f.params = params
	return f.grants, nil
}

func TestSyncGrantsWithClientFetchesAllGrantsWithoutLimit(t *testing.T) {
	dir := t.TempDir()
	grantStore := grantcache.New(filepath.Join(dir, "grants.json"))
	configStore := config.NewFileStore(filepath.Join(dir, "config.yaml"))
	client := &fakeGrantLister{grants: makeValidSetupGrants(30)}

	result, err := syncGrantsWithClient(context.Background(), configStore, grantStore, client)
	if err != nil {
		t.Fatalf("syncGrantsWithClient failed: %v", err)
	}
	if client.params != nil {
		t.Fatalf("ListAllGrants params = %#v, want nil", client.params)
	}
	if len(result.ValidGrants) != 30 {
		t.Fatalf("ValidGrants len = %d, want %d", len(result.ValidGrants), 30)
	}
	stored, err := grantStore.ListGrants()
	if err != nil {
		t.Fatalf("ListGrants failed: %v", err)
	}
	if len(stored) != 30 {
		t.Fatalf("stored grants len = %d, want %d", len(stored), 30)
	}
}

// TestSyncGrantsWithClientPersistsSingleGrantToBothStores guards the contract
// that a single-grant sync writes the default to grants.json AND mirrors it
// into config.yaml via PersistDefaultGrant. Regressions in either store would
// break the TUI/Air/CLI consistency we standardized on.
func TestSyncGrantsWithClientPersistsSingleGrantToBothStores(t *testing.T) {
	dir := t.TempDir()
	grantStore := grantcache.New(filepath.Join(dir, "grants.json"))
	configStore := config.NewFileStore(filepath.Join(dir, "config.yaml"))
	client := &fakeGrantLister{grants: makeValidSetupGrants(1)}

	result, err := syncGrantsWithClient(context.Background(), configStore, grantStore, client)
	if err != nil {
		t.Fatalf("syncGrantsWithClient failed: %v", err)
	}
	if result.DefaultGrantID != "grant-00" {
		t.Fatalf("DefaultGrantID = %q, want grant-00", result.DefaultGrantID)
	}

	gotGrantsJSON, err := grantStore.GetDefaultGrant()
	if err != nil {
		t.Fatalf("GetDefaultGrant failed: %v", err)
	}
	if gotGrantsJSON != "grant-00" {
		t.Fatalf("grants.json default = %q, want grant-00", gotGrantsJSON)
	}

	cfg, err := configStore.Load()
	if err != nil {
		t.Fatalf("config Load failed: %v", err)
	}
	if cfg.DefaultGrant != "grant-00" {
		t.Fatalf("config.yaml DefaultGrant = %q, want grant-00", cfg.DefaultGrant)
	}
}

func TestSyncGrantsWithClientReturnsErrorWhenSingleGrantDefaultCannotPersist(t *testing.T) {
	dir := t.TempDir()
	grantStore := grantcache.New(filepath.Join(dir, "grants.json"))
	configStore := &failingSetupConfigStore{err: errPersistDefaultGrant}
	client := &fakeGrantLister{grants: makeValidSetupGrants(1)}

	result, err := syncGrantsWithClient(context.Background(), configStore, grantStore, client)
	if !errors.Is(err, errPersistDefaultGrant) {
		t.Fatalf("syncGrantsWithClient err = %v, want %v", err, errPersistDefaultGrant)
	}
	if result != nil {
		t.Fatalf("syncGrantsWithClient result = %#v, want nil on persist failure", result)
	}

	if _, err := grantStore.GetDefaultGrant(); err != domain.ErrNoDefaultGrant {
		t.Fatalf("GetDefaultGrant err = %v, want ErrNoDefaultGrant", err)
	}
}

// TestSyncGrantsWithClientSkipsDefaultForMultipleGrants ensures that with more
// than one valid grant, neither store has a default set — the caller is
// expected to disambiguate via PromptDefaultGrant.
func TestSyncGrantsWithClientSkipsDefaultForMultipleGrants(t *testing.T) {
	dir := t.TempDir()
	grantStore := grantcache.New(filepath.Join(dir, "grants.json"))
	configStore := config.NewFileStore(filepath.Join(dir, "config.yaml"))
	client := &fakeGrantLister{grants: makeValidSetupGrants(3)}

	result, err := syncGrantsWithClient(context.Background(), configStore, grantStore, client)
	if err != nil {
		t.Fatalf("syncGrantsWithClient failed: %v", err)
	}
	if result.DefaultGrantID != "" {
		t.Fatalf("DefaultGrantID = %q, want empty", result.DefaultGrantID)
	}

	if _, err := grantStore.GetDefaultGrant(); err != domain.ErrNoDefaultGrant {
		t.Fatalf("GetDefaultGrant err = %v, want ErrNoDefaultGrant", err)
	}
}

func makeValidSetupGrants(n int) []domain.Grant {
	grants := make([]domain.Grant, n)
	for i := range grants {
		grants[i] = domain.Grant{
			ID:          fmt.Sprintf("grant-%02d", i),
			Email:       fmt.Sprintf("user%02d@example.com", i),
			Provider:    domain.ProviderGoogle,
			GrantStatus: "valid",
		}
	}
	return grants
}

type failingSetupConfigStore struct {
	err error
}

func (f *failingSetupConfigStore) Load() (*domain.Config, error) {
	return domain.DefaultConfig(), nil
}

func (f *failingSetupConfigStore) Save(*domain.Config) error {
	return f.err
}

func (f *failingSetupConfigStore) Path() string {
	return "/tmp/config.yaml"
}

func (f *failingSetupConfigStore) Exists() bool {
	return true
}
