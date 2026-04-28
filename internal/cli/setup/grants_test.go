package setup

import (
	"context"
	"fmt"
	"testing"

	"github.com/nylas/cli/internal/adapters/keyring"
	"github.com/nylas/cli/internal/domain"
)

type fakeGrantLister struct {
	grants []domain.Grant
	limit  int
}

func (f *fakeGrantLister) ListAllGrants(_ context.Context, params *domain.GrantsQueryParams) ([]domain.Grant, error) {
	if params != nil {
		f.limit = params.Limit
	}
	return f.grants, nil
}

func TestSyncGrantsWithClientDoesNotMarkExactCapTruncated(t *testing.T) {
	grantStore := keyring.NewGrantStore(keyring.NewMockSecretStore())
	client := &fakeGrantLister{grants: makeValidSetupGrants(MaxSyncedGrants)}

	result, err := syncGrantsWithClient(context.Background(), grantStore, client)
	if err != nil {
		t.Fatalf("syncGrantsWithClient failed: %v", err)
	}
	if client.limit != MaxSyncedGrants+1 {
		t.Fatalf("ListAllGrants limit = %d, want %d", client.limit, MaxSyncedGrants+1)
	}
	if result.Truncated {
		t.Fatal("Truncated = true for exactly MaxSyncedGrants grants")
	}
	if len(result.ValidGrants) != MaxSyncedGrants {
		t.Fatalf("ValidGrants len = %d, want %d", len(result.ValidGrants), MaxSyncedGrants)
	}
}

func TestSyncGrantsWithClientCapsStoredGrantsWhenTruncated(t *testing.T) {
	grantStore := keyring.NewGrantStore(keyring.NewMockSecretStore())
	client := &fakeGrantLister{grants: makeValidSetupGrants(MaxSyncedGrants + 1)}

	result, err := syncGrantsWithClient(context.Background(), grantStore, client)
	if err != nil {
		t.Fatalf("syncGrantsWithClient failed: %v", err)
	}
	if !result.Truncated {
		t.Fatal("Truncated = false when API returned more than MaxSyncedGrants")
	}
	if len(result.ValidGrants) != MaxSyncedGrants {
		t.Fatalf("ValidGrants len = %d, want %d", len(result.ValidGrants), MaxSyncedGrants)
	}
	stored, err := grantStore.ListGrants()
	if err != nil {
		t.Fatalf("ListGrants failed: %v", err)
	}
	if len(stored) != MaxSyncedGrants {
		t.Fatalf("stored grants len = %d, want %d", len(stored), MaxSyncedGrants)
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
