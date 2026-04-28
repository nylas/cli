package setup

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/nylas/cli/internal/adapters/grantcache"
	"github.com/nylas/cli/internal/domain"
)

type fakeGrantLister struct {
	grants []domain.Grant
	params *domain.GrantsQueryParams
}

func (f *fakeGrantLister) ListAllGrants(_ context.Context, params *domain.GrantsQueryParams) ([]domain.Grant, error) {
	f.params = params
	return f.grants, nil
}

func TestSyncGrantsWithClientFetchesAllGrantsWithoutLimit(t *testing.T) {
	grantStore := grantcache.New(filepath.Join(t.TempDir(), "grants.json"))
	client := &fakeGrantLister{grants: makeValidSetupGrants(30)}

	result, err := syncGrantsWithClient(context.Background(), grantStore, client)
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
