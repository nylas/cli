package ui

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	nylasadapter "github.com/nylas/cli/internal/adapters/nylas"
	authapp "github.com/nylas/cli/internal/app/auth"
	"github.com/nylas/cli/internal/cli/common"
	setupcli "github.com/nylas/cli/internal/cli/setup"
	"github.com/nylas/cli/internal/domain"
)

type ConfigStatusResponse struct {
	Configured   bool   `json:"configured"`
	Region       string `json:"region"`
	ClientID     string `json:"client_id,omitempty"`
	HasAPIKey    bool   `json:"has_api_key"`
	GrantCount   int    `json:"grant_count"`
	DefaultGrant string `json:"default_grant,omitempty"`
}

func (s *Server) handleConfigStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Demo mode: return sample configured status
	if s.demoMode {
		writeJSON(w, http.StatusOK, ConfigStatusResponse{
			Configured:   true,
			Region:       "us",
			ClientID:     "demo-client-id",
			HasAPIKey:    true,
			GrantCount:   3,
			DefaultGrant: demoDefaultGrant(),
		})
		return
	}

	status, err := s.configSvc.GetStatus()
	if err != nil {
		writeJSON(w, http.StatusOK, ConfigStatusResponse{Configured: false})
		return
	}

	resp := ConfigStatusResponse{
		Configured:   status.IsConfigured,
		Region:       status.Region,
		ClientID:     status.ClientID,
		HasAPIKey:    status.HasAPIKey,
		GrantCount:   status.GrantCount,
		DefaultGrant: status.DefaultGrant,
	}

	writeJSON(w, http.StatusOK, resp)
}

// SetupRequest represents the setup API request.
type SetupRequest struct {
	APIKey   string `json:"api_key"`
	Region   string `json:"region"`
	ClientID string `json:"client_id,omitempty"`
}

// SetupResponse represents the setup API response.
type SetupResponse struct {
	Success      bool          `json:"success"`
	Message      string        `json:"message"`
	Warning      string        `json:"warning,omitempty"`
	Region       string        `json:"region,omitempty"`
	ClientID     string        `json:"client_id,omitempty"`
	Applications []Application `json:"applications,omitempty"`
	Grants       []Grant       `json:"grants,omitempty"`
	Error        string        `json:"error,omitempty"`
}

// Application represents a Nylas application.
type Application struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Environment string `json:"environment"`
}

// Grant represents an authenticated account.
type Grant struct {
	ID       string `json:"id"`
	Email    string `json:"email"`
	Provider string `json:"provider"`
}

type setupClient interface {
	ListApplications(ctx context.Context) ([]domain.Application, error)
	ListGrants(ctx context.Context) ([]domain.Grant, error)
}

var newSetupClient = func(region, clientID, apiKey string) setupClient {
	client := nylasadapter.NewHTTPClient()
	client.SetRegion(region)
	client.SetCredentials(clientID, "", apiKey)
	return client
}

var ensureSetupCallbackURI = setupcli.EnsureOAuthCallbackURI

// grantFromDomain converts a domain.GrantInfo to a Grant for API responses.
func grantFromDomain(g domain.GrantInfo) Grant {
	return Grant{
		ID:       g.ID,
		Email:    g.Email,
		Provider: string(g.Provider),
	}
}

func (s *Server) handleConfigSetup(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Demo mode: simulate successful setup
	if s.demoMode {
		writeJSON(w, http.StatusOK, SetupResponse{
			Success:  true,
			Message:  "Demo mode - configuration simulated",
			Region:   "us",
			ClientID: "demo-client-id",
			Applications: []Application{
				{ID: "demo-app", Name: "Demo Application", Environment: "production"},
			},
			Grants: demoGrants(),
		})
		return
	}

	var req SetupRequest
	if err := json.NewDecoder(limitedBody(w, r)).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, SetupResponse{
			Success: false,
			Error:   "Invalid request body",
		})
		return
	}

	if req.APIKey == "" {
		writeJSON(w, http.StatusBadRequest, SetupResponse{
			Success: false,
			Error:   "API key is required",
		})
		return
	}

	if req.Region == "" {
		req.Region = "us"
	}

	// Create Nylas client to detect applications
	client := newSetupClient(req.Region, "", req.APIKey)

	ctx, cancel := common.CreateContext()
	defer cancel()

	// List applications to get Client ID
	apps, err := client.ListApplications(ctx)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, SetupResponse{
			Success: false,
			Error:   "Invalid API key or could not connect to Nylas: " + err.Error(),
		})
		return
	}

	if len(apps) == 0 {
		writeJSON(w, http.StatusBadRequest, SetupResponse{
			Success: false,
			Error:   "No applications found for this API key",
		})
		return
	}

	appList := buildApplicationList(apps)
	selectedApp, err := resolveSetupApplication(apps, req.ClientID)
	if err != nil {
		writeJSON(w, http.StatusConflict, SetupResponse{
			Success:      false,
			Error:        err.Error(),
			Applications: appList,
		})
		return
	}

	clientID := setupcli.AppClientID(*selectedApp)
	orgID := selectedApp.OrganizationID

	cfg, err := s.configStore.Load()
	if err != nil || cfg == nil {
		cfg = domain.DefaultConfig()
	}

	warning := ""
	callbackResult, callbackErr := ensureSetupCallbackURI(req.APIKey, clientID, req.Region, cfg.CallbackPort)
	if callbackErr != nil {
		warning = fmt.Sprintf("Add callback URI manually in the dashboard: %s", callbackResult.RequiredURI)
	}

	// Save configuration
	if err := s.configSvc.SetupConfig(req.Region, clientID, "", req.APIKey, orgID); err != nil {
		writeJSON(w, http.StatusInternalServerError, SetupResponse{
			Success: false,
			Error:   "Failed to save configuration: " + err.Error(),
		})
		return
	}

	// Update client with credentials for grant lookup
	client = newSetupClient(req.Region, clientID, req.APIKey)

	// Fetch existing grants
	grants, _ := client.ListGrants(ctx)

	// Save grants locally
	var grantList []Grant
	defaultAssigned := false
	for _, grant := range grants {
		if !grant.IsValid() {
			continue
		}

		grantInfo := domain.GrantInfo{
			ID:       grant.ID,
			Email:    grant.Email,
			Provider: grant.Provider,
		}

		_ = s.grantStore.SaveGrant(grantInfo)

		if !defaultAssigned {
			if err := authapp.PersistDefaultGrant(s.configStore, s.grantStore, grant.ID); err != nil {
				writeJSON(w, http.StatusInternalServerError, SetupResponse{
					Success: false,
					Error:   "Failed to set default grant: " + err.Error(),
				})
				return
			}
			defaultAssigned = true
		}

		grantList = append(grantList, grantFromDomain(grantInfo))
	}

	writeJSON(w, http.StatusOK, SetupResponse{
		Success:      true,
		Message:      "Configuration saved successfully",
		Warning:      warning,
		Region:       req.Region,
		ClientID:     clientID,
		Applications: appList,
		Grants:       grantList,
	})
}

func buildApplicationList(apps []domain.Application) []Application {
	appList := make([]Application, 0, len(apps))
	for _, app := range apps {
		id := setupcli.AppClientID(app)
		appList = append(appList, Application{
			ID:          id,
			Name:        id,
			Environment: app.Environment,
		})
	}
	return appList
}

func resolveSetupApplication(apps []domain.Application, requestedClientID string) (*domain.Application, error) {
	if len(apps) == 0 {
		return nil, fmt.Errorf("no applications found for this API key")
	}
	if requestedClientID == "" {
		if len(apps) > 1 {
			return nil, fmt.Errorf("multiple applications found for this API key; provide client_id to select one")
		}
		return &apps[0], nil
	}

	for i := range apps {
		if setupcli.AppClientID(apps[i]) == requestedClientID {
			return &apps[i], nil
		}
	}

	return nil, fmt.Errorf("client_id %q was not found for this API key", requestedClientID)
}

// GrantsResponse represents the grants list API response.
type GrantsResponse struct {
	Grants       []Grant `json:"grants"`
	DefaultGrant string  `json:"default_grant"`
}
