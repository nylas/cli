package auth

import (
	"context"

	"github.com/nylas/cli/internal/domain"
	"github.com/nylas/cli/internal/ports"
)

// GrantService handles grant operations.
type GrantService struct {
	client     ports.NylasClient
	grantStore ports.GrantStore
	config     ports.ConfigStore
}

// NewGrantService creates a new grant service.
func NewGrantService(client ports.NylasClient, grantStore ports.GrantStore, config ports.ConfigStore) *GrantService {
	return &GrantService{
		client:     client,
		grantStore: grantStore,
		config:     config,
	}
}

// ListGrants returns all grants with their status.
func (s *GrantService) ListGrants(ctx context.Context) ([]domain.GrantStatus, error) {
	localGrants, err := s.grantStore.ListGrants()
	if err != nil {
		return nil, err
	}

	defaultGrant, _ := s.grantStore.GetDefaultGrant()

	var result []domain.GrantStatus
	for _, g := range localGrants {
		var status string
		var errMsg string
		provider := g.Provider // Default to local storage

		// Verify grant is still valid on Nylas
		grant, err := s.client.GetGrant(ctx, g.ID)
		if err == nil && grant != nil {
			if grant.IsValid() {
				status = "valid"
			} else {
				status = grant.GrantStatus
			}
			// Use provider from API (authoritative source)
			if grant.Provider != "" {
				provider = grant.Provider
				// Update local storage if provider changed
				if g.Provider != grant.Provider {
					g.Provider = grant.Provider
					_ = s.grantStore.SaveGrant(g)
				}
			}
		} else if err == domain.ErrGrantNotFound {
			status = "revoked"
		} else {
			status = "error"
			if err != nil {
				errMsg = err.Error()
			}
		}

		result = append(result, domain.GrantStatus{
			ID:        g.ID,
			Email:     g.Email,
			Provider:  provider,
			Status:    status,
			IsDefault: g.ID == defaultGrant,
			Error:     errMsg,
		})
	}

	return result, nil
}

// GetCurrentGrant returns the current (default) grant info.
func (s *GrantService) GetCurrentGrant(ctx context.Context) (*domain.GrantStatus, error) {
	grantID, err := s.grantStore.GetDefaultGrant()
	if err != nil {
		return nil, err
	}

	info, err := s.grantStore.GetGrant(grantID)
	if err != nil {
		return nil, err
	}

	// Verify on Nylas and get current provider info
	status := "unknown"
	provider := info.Provider // Default to local storage
	grant, err := s.client.GetGrant(ctx, grantID)
	if err == nil && grant != nil {
		if grant.IsValid() {
			status = "valid"
		} else {
			status = grant.GrantStatus
		}
		// Use provider from API (authoritative source)
		if grant.Provider != "" {
			provider = grant.Provider
			// Update local storage if provider changed
			if info.Provider != grant.Provider {
				info.Provider = grant.Provider
				_ = s.grantStore.SaveGrant(*info)
			}
		}
	} else if err == domain.ErrGrantNotFound {
		status = "revoked"
	}

	return &domain.GrantStatus{
		ID:        info.ID,
		Email:     info.Email,
		Provider:  provider,
		Status:    status,
		IsDefault: true,
	}, nil
}

// SwitchGrant switches the default grant to the specified ID.
func (s *GrantService) SwitchGrant(grantID string) error {
	// Verify grant exists
	if _, err := s.grantStore.GetGrant(grantID); err != nil {
		return err
	}
	return s.grantStore.SetDefaultGrant(grantID)
}

// SwitchGrantByEmail switches the default grant by email.
func (s *GrantService) SwitchGrantByEmail(email string) error {
	info, err := s.grantStore.GetGrantByEmail(email)
	if err != nil {
		return err
	}
	return s.grantStore.SetDefaultGrant(info.ID)
}

// ValidateGrant checks if a grant is valid.
func (s *GrantService) ValidateGrant(ctx context.Context, grantID string) (bool, error) {
	grant, err := s.client.GetGrant(ctx, grantID)
	if err != nil {
		return false, err
	}
	return grant.IsValid(), nil
}

// GetDefaultGrantID returns the default grant ID.
func (s *GrantService) GetDefaultGrantID() (string, error) {
	return s.grantStore.GetDefaultGrant()
}

// AddGrant manually adds a grant to local storage.
func (s *GrantService) AddGrant(grantID, email string, provider domain.Provider, setDefault bool) error {
	grantInfo := domain.GrantInfo{
		ID:       grantID,
		Email:    email,
		Provider: provider,
	}

	if err := s.grantStore.SaveGrant(grantInfo); err != nil {
		return err
	}

	// Set as default if requested or if this is the first grant
	if setDefault {
		return s.grantStore.SetDefaultGrant(grantID)
	}

	// Auto-set as default if no default exists
	if _, err := s.grantStore.GetDefaultGrant(); err == domain.ErrNoDefaultGrant {
		return s.grantStore.SetDefaultGrant(grantID)
	}

	return nil
}

// FetchGrantFromNylas fetches grant details directly from Nylas API.
func (s *GrantService) FetchGrantFromNylas(ctx context.Context, grantID string) (*domain.Grant, error) {
	return s.client.GetGrant(ctx, grantID)
}
