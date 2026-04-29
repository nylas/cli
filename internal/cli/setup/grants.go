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

// SyncResult holds the result of a grant sync operation.
type SyncResult struct {
	ValidGrants    []domain.Grant
	DefaultGrantID string
}

// SyncGrants fetches grants from the Nylas API and saves them to the local grant cache.
// It returns the list of valid grants and the chosen default grant ID.
// The caller is responsible for setting the default if multiple grants exist
// (use PromptDefaultGrant for interactive selection).
func SyncGrants(grantStore ports.GrantStore, apiKey, clientID, region string) (*SyncResult, error) {
	client := nylasadapter.NewHTTPClient()
	client.SetRegion(region)
	client.SetCredentials(clientID, "", apiKey)

	ctx, cancel := common.CreateContext()
	defer cancel()

	return syncGrantsWithClient(ctx, grantStore, client)
}

func syncGrantsWithClient(ctx context.Context, grantStore ports.GrantStore, client grantLister) (*SyncResult, error) {
	grants, err := client.ListAllGrants(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("could not fetch grants: %w", err)
	}

	var validGrants []domain.Grant
	grantInfos := make([]domain.GrantInfo, 0, len(grants))
	for _, grant := range grants {
		if !grant.IsValid() {
			continue
		}

		grantInfo := domain.GrantInfo{
			ID:       grant.ID,
			Email:    grant.Email,
			Provider: grant.Provider,
		}

		grantInfos = append(grantInfos, grantInfo)
		validGrants = append(validGrants, grant)
		_, _ = common.Green.Printf("  ✓ Added %s (%s)\n", grant.Email, grant.Provider.DisplayName())
	}

	if err := grantStore.ReplaceGrants(grantInfos); err != nil {
		return nil, fmt.Errorf("could not save grants: %w", err)
	}

	result := &SyncResult{
		ValidGrants: validGrants,
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
