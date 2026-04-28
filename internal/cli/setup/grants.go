package setup

import (
	"context"
	"fmt"

	nylasadapter "github.com/nylas/cli/internal/adapters/nylas"
	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/domain"
	"github.com/nylas/cli/internal/ports"
)

type grantLister interface {
	ListAllGrants(ctx context.Context, params *domain.GrantsQueryParams) ([]domain.Grant, error)
}

// MaxSyncedGrants caps the number of grants SyncGrants will persist to the
// local keyring. SyncGrants fetches one extra grant as a lookahead so it can
// distinguish exactly-at-cap accounts from truncated accounts.
// The keyring stores grant metadata as a single JSON blob and rewrites it on
// every SaveGrant call, so unbounded growth is undesirable on platforms with
// per-item size limits (notably macOS Keychain).
const MaxSyncedGrants = 25

// SyncResult holds the result of a grant sync operation.
type SyncResult struct {
	ValidGrants    []domain.Grant
	DefaultGrantID string
	// Truncated reports whether the user has more grants than were
	// synced.
	Truncated bool
}

// SyncGrants fetches grants from the Nylas API and saves them to the local keyring.
// It returns the list of valid grants and the chosen default grant ID.
// The caller is responsible for setting the default if multiple grants exist
// (use PromptDefaultGrant for interactive selection).
//
// At most MaxSyncedGrants grants are stored; if more exist on the server,
// SyncResult.Truncated is set so callers can surface a hint.
func SyncGrants(grantStore ports.GrantStore, apiKey, clientID, region string) (*SyncResult, error) {
	client := nylasadapter.NewHTTPClient()
	client.SetRegion(region)
	client.SetCredentials(clientID, "", apiKey)

	ctx, cancel := common.CreateContext()
	defer cancel()

	return syncGrantsWithClient(ctx, grantStore, client)
}

func syncGrantsWithClient(ctx context.Context, grantStore ports.GrantStore, client grantLister) (*SyncResult, error) {
	grants, err := client.ListAllGrants(ctx, &domain.GrantsQueryParams{Limit: MaxSyncedGrants + 1})
	if err != nil {
		return nil, fmt.Errorf("could not fetch grants: %w", err)
	}
	truncated := len(grants) > MaxSyncedGrants
	if truncated {
		grants = grants[:MaxSyncedGrants]
	}

	var validGrants []domain.Grant
	for _, grant := range grants {
		if !grant.IsValid() {
			continue
		}

		grantInfo := domain.GrantInfo{
			ID:       grant.ID,
			Email:    grant.Email,
			Provider: grant.Provider,
		}

		if saveErr := grantStore.SaveGrant(grantInfo); saveErr != nil {
			continue
		}

		validGrants = append(validGrants, grant)
		_, _ = common.Green.Printf("  ✓ Added %s (%s)\n", grant.Email, grant.Provider.DisplayName())
	}

	result := &SyncResult{
		ValidGrants: validGrants,
		Truncated:   truncated,
	}

	// Auto-set default if there's exactly one valid grant.
	if len(validGrants) == 1 {
		result.DefaultGrantID = validGrants[0].ID
		_ = grantStore.SetDefaultGrant(result.DefaultGrantID)
	}

	return result, nil
}

// PromptDefaultGrant presents an interactive menu for the user to select a default grant.
func PromptDefaultGrant(grantStore ports.GrantStore, grants []domain.Grant) (string, error) {
	opts := make([]common.SelectOption[string], len(grants))
	for i, grant := range grants {
		opts[i] = common.SelectOption[string]{
			Label: fmt.Sprintf("%s (%s)", grant.Email, grant.Provider.DisplayName()),
			Value: grant.ID,
		}
	}

	chosen, err := common.Select("Select default account", opts)
	if err != nil {
		chosen = grants[0].ID
	}

	_ = grantStore.SetDefaultGrant(chosen)
	return chosen, nil
}
