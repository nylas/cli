package nylas

import (
	"context"
	"fmt"
	"net/url"

	"github.com/nylas/cli/internal/domain"
)

type managedGrantResponse struct {
	ID           string               `json:"id"`
	Email        string               `json:"email"`
	Provider     domain.Provider      `json:"provider"`
	GrantStatus  string               `json:"grant_status"`
	CreatedAt    domain.UnixTime      `json:"created_at"`
	UpdatedAt    domain.UnixTime      `json:"updated_at"`
	CredentialID string               `json:"credential_id,omitempty"`
	Blocked      bool                 `json:"blocked,omitempty"`
	Settings     agentSettingsPayload `json:"settings,omitempty"`
}

type agentSettingsPayload struct {
	PolicyID string `json:"policy_id,omitempty"`
}

// maxManagedGrantPages caps pagination so a misbehaving server can't
// trap us in an unbounded loop of fresh-but-empty cursors.
const maxManagedGrantPages = 1000

// listManagedGrants returns every grant whose provider matches `provider`.
//
// We deliberately do NOT pass `provider=<name>` as a server-side filter:
// the server-side filtered listing has been observed to lag freshly-
// created managed grants by tens of seconds (>70s in the worst case),
// while the unfiltered listing surfaces new grants almost immediately.
// Trade ~4x more page bytes (typical tenants have <100 grants) for
// freshness and predictability. We filter to `provider` client-side.
func (c *HTTPClient) listManagedGrants(ctx context.Context, provider domain.Provider) ([]managedGrantResponse, error) {
	baseURL := fmt.Sprintf("%s/v3/grants", c.baseURL)
	pageToken := ""
	grants := make([]managedGrantResponse, 0)

	seenCursors := make(map[string]struct{})
	for {
		queryURL := NewQueryBuilder().
			Add("page_token", pageToken).
			BuildURL(baseURL)

		var result struct {
			Data       []managedGrantResponse `json:"data"`
			NextCursor string                 `json:"next_cursor,omitempty"`
		}
		if err := c.doGet(ctx, queryURL, &result); err != nil {
			return nil, err
		}

		for _, grant := range result.Data {
			if grant.Provider == provider {
				grants = append(grants, grant)
			}
		}

		if result.NextCursor == "" {
			break
		}
		if result.NextCursor == pageToken {
			return nil, fmt.Errorf("failed to paginate managed grants: repeated cursor %q", result.NextCursor)
		}
		if _, seen := seenCursors[result.NextCursor]; seen {
			return nil, fmt.Errorf("failed to paginate managed grants: cursor cycle detected near %q", result.NextCursor)
		}
		seenCursors[result.NextCursor] = struct{}{}
		if len(seenCursors) > maxManagedGrantPages {
			return nil, fmt.Errorf("failed to paginate managed grants: exceeded max page count (%d)", maxManagedGrantPages)
		}
		pageToken = result.NextCursor
	}

	return grants, nil
}

func (c *HTTPClient) getManagedGrant(ctx context.Context, grantID string) (*managedGrantResponse, error) {
	queryURL := fmt.Sprintf("%s/v3/grants/%s", c.baseURL, url.PathEscape(grantID))

	var result struct {
		Data managedGrantResponse `json:"data"`
	}
	if err := c.doGetWithNotFound(ctx, queryURL, &result, domain.ErrGrantNotFound); err != nil {
		return nil, err
	}

	return &result.Data, nil
}

func (c *HTTPClient) deleteManagedGrant(ctx context.Context, grantID string, expectedProvider domain.Provider) error {
	grant, err := c.getManagedGrant(ctx, grantID)
	if err != nil {
		return err
	}
	if grant == nil {
		return domain.ErrGrantNotFound
	}
	if grant.Provider != expectedProvider {
		return fmt.Errorf("%w: grant is not a %s managed grant (provider=%s)", domain.ErrInvalidGrant, expectedProvider, grant.Provider)
	}

	return c.RevokeGrant(ctx, grantID)
}

func convertManagedGrantToAgentAccount(grant managedGrantResponse) domain.AgentAccount {
	return domain.AgentAccount{
		ID:           grant.ID,
		Provider:     grant.Provider,
		Email:        grant.Email,
		GrantStatus:  grant.GrantStatus,
		CreatedAt:    grant.CreatedAt,
		UpdatedAt:    grant.UpdatedAt,
		CredentialID: grant.CredentialID,
		Blocked:      grant.Blocked,
		Settings: domain.AgentAccountSettings{
			PolicyID: grant.Settings.PolicyID,
		},
	}
}
