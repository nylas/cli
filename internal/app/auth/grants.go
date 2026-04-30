package auth

import (
	"context"
	"errors"

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

// CachedGrantCount returns the number of locally cached grants.
func (s *GrantService) CachedGrantCount() int {
	grants, err := s.grantStore.ListGrants()
	if err != nil {
		return 0
	}
	return len(grants)
}

// ListGrants returns all grants with their status.
func (s *GrantService) ListGrants(ctx context.Context) ([]domain.GrantStatus, error) {
	liveGrants, err := s.client.ListGrants(ctx)
	if err != nil {
		return nil, err
	}

	defaultGrant := s.defaultGrantID()
	defaultStillExists := defaultGrant == ""
	cacheGrants := make([]domain.GrantInfo, 0, len(liveGrants))
	result := make([]domain.GrantStatus, 0, len(liveGrants))

	for _, grant := range liveGrants {
		status := grant.GrantStatus
		if grant.IsValid() {
			status = "valid"
		}

		result = append(result, domain.GrantStatus{
			ID:        grant.ID,
			Email:     grant.Email,
			Provider:  grant.Provider,
			Status:    status,
			IsDefault: grant.ID == defaultGrant,
		})
		if grant.ID == defaultGrant {
			defaultStillExists = true
		}
		cacheGrants = append(cacheGrants, domain.GrantInfo{
			ID:       grant.ID,
			Email:    grant.Email,
			Provider: grant.Provider,
		})
	}
	_ = s.grantStore.ReplaceGrants(cacheGrants)
	if !defaultStillExists {
		_ = PersistDefaultGrant(s.config, s.grantStore, "")
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
	ctx, cancel := context.WithTimeout(context.Background(), domain.TimeoutAPI)
	defer cancel()

	grant, err := s.client.GetGrant(ctx, grantID)
	if err != nil {
		return err
	}
	if err := s.grantStore.SaveGrant(domain.GrantInfo{
		ID:       grant.ID,
		Email:    grant.Email,
		Provider: grant.Provider,
	}); err != nil {
		return err
	}
	return s.setDefaultGrant(grantID)
}

// SwitchGrantByEmail switches the default grant by email.
func (s *GrantService) SwitchGrantByEmail(email string) error {
	ctx, cancel := context.WithTimeout(context.Background(), domain.TimeoutAPI)
	defer cancel()

	grants, err := s.client.ListGrants(ctx)
	if err != nil {
		return err
	}
	for _, grant := range grants {
		if grant.Email == email {
			if err := s.grantStore.SaveGrant(domain.GrantInfo{
				ID:       grant.ID,
				Email:    grant.Email,
				Provider: grant.Provider,
			}); err != nil {
				return err
			}
			return s.setDefaultGrant(grant.ID)
		}
	}
	return domain.ErrGrantNotFound
}

// PersistDefaultGrant updates both the config file and grant store so all
// callers observe the same default grant value.
func PersistDefaultGrant(config ports.ConfigStore, grantStore ports.GrantStore, grantID string) error {
	var (
		cfgToSave    *domain.Config
		rollbackCfg  *domain.Config
		configLoaded bool
	)

	if config != nil {
		cfg, err := config.Load()
		if err == nil && cfg != nil {
			configLoaded = true
			rollbackCfg = cloneConfig(cfg)
			cfgToSave = cloneConfig(cfg)
		} else {
			cfgToSave = domain.DefaultConfig()
		}

		cfgToSave.DefaultGrant = grantID
		// Drop the legacy in-memory grants slice — grant metadata lives in
		// the grant cache now, and config.Grants is a transient field that
		// shouldn't be re-persisted from older config snapshots.
		cfgToSave.Grants = nil
		if err := config.Save(cfgToSave); err != nil {
			return err
		}
	}

	if err := grantStore.SetDefaultGrant(grantID); err != nil {
		if config == nil {
			return err
		}

		restoreCfg := rollbackCfg
		if !configLoaded || restoreCfg == nil {
			restoreCfg = domain.DefaultConfig()
		}
		if rollbackErr := config.Save(restoreCfg); rollbackErr != nil {
			return errors.Join(err, rollbackErr)
		}
		return err
	}

	return nil
}

// setDefaultGrant updates the default grant in both local cache and config file.
func (s *GrantService) setDefaultGrant(grantID string) error {
	return PersistDefaultGrant(s.config, s.grantStore, grantID)
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
		return s.setDefaultGrant(grantID)
	}

	// Auto-set as default if no default exists
	if _, err := s.grantStore.GetDefaultGrant(); err == domain.ErrNoDefaultGrant {
		return s.setDefaultGrant(grantID)
	}

	return nil
}

// FetchGrantFromNylas fetches grant details directly from Nylas API.
func (s *GrantService) FetchGrantFromNylas(ctx context.Context, grantID string) (*domain.Grant, error) {
	return s.client.GetGrant(ctx, grantID)
}

func (s *GrantService) defaultGrantID() string {
	defaultGrant, err := s.grantStore.GetDefaultGrant()
	if err == nil {
		return defaultGrant
	}
	if !errors.Is(err, domain.ErrNoDefaultGrant) {
		return ""
	}
	if s.config == nil {
		return ""
	}
	cfg, err := s.config.Load()
	if err != nil || cfg == nil {
		return ""
	}
	return cfg.DefaultGrant
}
