package setup

import (
	"bufio"
	"fmt"
	"os"
	"strings"

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
	fmt.Println()
	fmt.Println("Select default account:")
	for i, grant := range grants {
		fmt.Printf("  [%d] %s (%s)\n", i+1, grant.Email, grant.Provider.DisplayName())
	}
	fmt.Println()

	choice, err := readLine(fmt.Sprintf("Choose [1-%d]: ", len(grants)))
	if err != nil {
		return grants[0].ID, nil
	}

	var selected int
	if _, err := fmt.Sscanf(choice, "%d", &selected); err != nil || selected < 1 || selected > len(grants) {
		_, _ = common.Yellow.Printf("Invalid selection, defaulting to %s\n", grants[0].Email)
		_ = grantStore.SetDefaultGrant(grants[0].ID)
		return grants[0].ID, nil
	}

	chosen := grants[selected-1]
	_ = grantStore.SetDefaultGrant(chosen.ID)
	return chosen.ID, nil
}

// readLine prompts for a line of text input.
func readLine(prompt string) (string, error) {
	fmt.Print(prompt)
	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("failed to read input: %w", err)
	}
	return strings.TrimSpace(input), nil
}
