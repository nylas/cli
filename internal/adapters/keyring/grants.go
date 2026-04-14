package keyring

import (
	"encoding/json"

	"github.com/nylas/cli/internal/domain"
	"github.com/nylas/cli/internal/ports"
)

const (
	grantsKey       = "grants"
	defaultGrantKey = "default_grant"
)

// GrantStore implements ports.GrantStore using a SecretStore backend.
type GrantStore struct {
	secrets ports.SecretStore
}

// NewGrantStore creates a new GrantStore.
func NewGrantStore(secrets ports.SecretStore) *GrantStore {
	return &GrantStore{secrets: secrets}
}

// SaveGrant saves grant info to storage.
func (g *GrantStore) SaveGrant(info domain.GrantInfo) error {
	if info.ID == "" || info.Email == "" {
		return domain.ErrInvalidInput
	}

	grants, err := g.ListGrants()
	if err != nil && err != domain.ErrSecretNotFound {
		return err
	}

	// Check if grant already exists and update it
	found := false
	for i, grant := range grants {
		if grant.ID == info.ID {
			grants[i] = info
			found = true
			break
		}
	}
	if !found {
		grants = append(grants, info)
	}

	return g.saveGrants(grants)
}

// GetGrant retrieves grant info by ID.
func (g *GrantStore) GetGrant(grantID string) (*domain.GrantInfo, error) {
	grants, err := g.ListGrants()
	if err != nil {
		return nil, err
	}

	for _, grant := range grants {
		if grant.ID == grantID {
			return &grant, nil
		}
	}
	return nil, domain.ErrGrantNotFound
}

// GetGrantByEmail retrieves grant info by email.
func (g *GrantStore) GetGrantByEmail(email string) (*domain.GrantInfo, error) {
	grants, err := g.ListGrants()
	if err != nil {
		return nil, err
	}

	for _, grant := range grants {
		if grant.Email == email {
			return &grant, nil
		}
	}
	return nil, domain.ErrGrantNotFound
}

// ListGrants returns all stored grants.
func (g *GrantStore) ListGrants() ([]domain.GrantInfo, error) {
	data, err := g.secrets.Get(grantsKey)
	if err != nil {
		if err == domain.ErrSecretNotFound {
			if err := g.repairDefaultGrant(nil); err != nil {
				return nil, err
			}
			return []domain.GrantInfo{}, nil
		}
		return nil, err
	}

	var grants []domain.GrantInfo
	if err := json.Unmarshal([]byte(data), &grants); err != nil {
		return nil, err
	}

	sanitized, changed := sanitizeGrants(grants)
	if changed {
		if err := g.saveGrants(sanitized); err != nil {
			return nil, err
		}
	}
	if err := g.repairDefaultGrant(sanitized); err != nil {
		return nil, err
	}

	return sanitized, nil
}

// DeleteGrant removes a grant from storage.
func (g *GrantStore) DeleteGrant(grantID string) error {
	grants, err := g.ListGrants()
	if err != nil {
		return err
	}

	newGrants := make([]domain.GrantInfo, 0, len(grants))
	for _, grant := range grants {
		if grant.ID != grantID {
			newGrants = append(newGrants, grant)
		}
	}

	// Also delete the grant's token if it exists
	_ = g.secrets.Delete(ports.GrantTokenKey(grantID))

	if err := g.saveGrants(newGrants); err != nil {
		return err
	}

	return g.repairDefaultGrant(newGrants)
}

// SetDefaultGrant sets the default grant ID.
func (g *GrantStore) SetDefaultGrant(grantID string) error {
	return g.secrets.Set(defaultGrantKey, grantID)
}

// GetDefaultGrant returns the default grant ID.
func (g *GrantStore) GetDefaultGrant() (string, error) {
	_, err := g.ListGrants()
	if err != nil {
		return "", err
	}

	grantID, err := g.secrets.Get(defaultGrantKey)
	if err != nil {
		if err == domain.ErrSecretNotFound {
			return "", domain.ErrNoDefaultGrant
		}
		return "", err
	}

	return grantID, nil
}

// ClearGrants removes all grants from storage.
func (g *GrantStore) ClearGrants() error {
	_ = g.secrets.Delete(grantsKey)
	_ = g.secrets.Delete(defaultGrantKey)
	return nil
}

func (g *GrantStore) saveGrants(grants []domain.GrantInfo) error {
	data, err := json.Marshal(grants)
	if err != nil {
		return err
	}
	return g.secrets.Set(grantsKey, string(data))
}

func sanitizeGrants(grants []domain.GrantInfo) ([]domain.GrantInfo, bool) {
	sanitized := make([]domain.GrantInfo, 0, len(grants))
	changed := false
	for _, grant := range grants {
		if grant.ID == "" || grant.Email == "" {
			changed = true
			continue
		}
		sanitized = append(sanitized, grant)
	}
	return sanitized, changed
}

func (g *GrantStore) repairDefaultGrant(grants []domain.GrantInfo) error {
	defaultID, err := g.secrets.Get(defaultGrantKey)
	if err != nil {
		if err != domain.ErrSecretNotFound {
			return err
		}
		if len(grants) == 0 {
			return nil
		}
		return g.secrets.Set(defaultGrantKey, grants[0].ID)
	}

	for _, grant := range grants {
		if grant.ID == defaultID {
			return nil
		}
	}

	if len(grants) == 0 {
		_ = g.secrets.Delete(defaultGrantKey)
		return nil
	}

	return g.secrets.Set(defaultGrantKey, grants[0].ID)
}
