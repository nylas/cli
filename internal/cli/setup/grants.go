package setup

import (
	"fmt"

	nylasadapter "github.com/nylas/cli/internal/adapters/nylas"
	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/domain"
	"github.com/nylas/cli/internal/ports"
)

// SyncResult holds the result of a grant sync operation.
type SyncResult struct {
	ValidGrants    []domain.Grant
	DefaultGrantID string
}

// SyncGrants fetches grants from the Nylas API and saves them to the local keyring.
// It returns the list of valid grants and the chosen default grant ID.
// The caller is responsible for setting the default if multiple grants exist
// (use PromptDefaultGrant for interactive selection).
func SyncGrants(grantStore ports.GrantStore, apiKey, clientID, region string) (*SyncResult, error) {
	client := nylasadapter.NewHTTPClient()
	client.SetRegion(region)
	client.SetCredentials(clientID, "", apiKey)

	ctx, cancel := common.CreateContext()
	defer cancel()

	grants, err := client.ListGrants(ctx)
	if err != nil {
		return nil, fmt.Errorf("could not fetch grants: %w", err)
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

	result := &SyncResult{ValidGrants: validGrants}

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
