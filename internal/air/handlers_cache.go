package air

import (
	"database/sql"
	"fmt"
	"net/http"
	"time"

	"github.com/nylas/cli/internal/air/cache"
)

// CacheStatusResponse represents the cache status API response.
type CacheStatusResponse struct {
	Enabled           bool               `json:"enabled"`
	Online            bool               `json:"online"`
	Accounts          []CacheAccountInfo `json:"accounts"`
	TotalSizeBytes    int64              `json:"total_size_bytes"`
	PendingActions    int                `json:"pending_actions"`
	LastSync          *time.Time         `json:"last_sync,omitempty"`
	SyncInterval      int                `json:"sync_interval_minutes"`
	EncryptionEnabled bool               `json:"encryption_enabled"`
}

// CacheAccountInfo contains cache info for a single account.
type CacheAccountInfo struct {
	Email        string     `json:"email"`
	SizeBytes    int64      `json:"size_bytes"`
	EmailCount   int        `json:"email_count"`
	EventCount   int        `json:"event_count"`
	ContactCount int        `json:"contact_count"`
	LastSync     *time.Time `json:"last_sync,omitempty"`
}

// CacheSyncResponse represents the sync trigger response.
type CacheSyncResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
	Error   string `json:"error,omitempty"`
}

// CacheSearchResponse represents the search results response.
type CacheSearchResponse struct {
	Results []CacheSearchResult `json:"results"`
	Query   string              `json:"query"`
	Total   int                 `json:"total"`
}

// CacheSearchResult represents a single search result.
type CacheSearchResult struct {
	Type     string `json:"type"` // "email", "event", "contact"
	ID       string `json:"id"`
	Title    string `json:"title"`
	Subtitle string `json:"subtitle"`
	Date     int64  `json:"date"` // Unix timestamp
}

// CacheSettingsResponse represents cache settings.
type CacheSettingsResponse struct {
	Enabled             bool   `json:"cache_enabled"`
	MaxSizeMB           int    `json:"cache_max_size_mb"`
	TTLDays             int    `json:"cache_ttl_days"`
	SyncIntervalMinutes int    `json:"sync_interval_minutes"`
	OfflineQueueEnabled bool   `json:"offline_queue_enabled"`
	EncryptionEnabled   bool   `json:"encryption_enabled"`
	Theme               string `json:"theme"`
	DefaultView         string `json:"default_view"`
	CompactMode         bool   `json:"compact_mode"`
	PreviewPosition     string `json:"preview_position"`
}

// handleCacheStatus returns the current cache status.
func (s *Server) handleCacheStatus(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}
	if s.handleDemoMode(w, CacheStatusResponse{
		Enabled: true,
		Online:  true,
		Accounts: []CacheAccountInfo{
			{
				Email:        "demo@example.com",
				SizeBytes:    10 * 1024 * 1024, // 10MB
				EmailCount:   150,
				EventCount:   25,
				ContactCount: 50,
			},
		},
		TotalSizeBytes:    10 * 1024 * 1024,
		PendingActions:    0,
		SyncInterval:      5,
		EncryptionEnabled: false,
	}) {
		return
	}

	response := CacheStatusResponse{
		Enabled:           s.cacheSettings != nil && s.cacheSettings.IsCacheEnabled(),
		Online:            s.IsOnline(),
		SyncInterval:      5,
		EncryptionEnabled: s.cacheSettings != nil && s.cacheSettings.IsEncryptionEnabled(),
	}

	if s.cacheSettings != nil {
		response.SyncInterval = s.cacheSettings.Get().SyncIntervalMinutes
	}

	// Get stats for each account
	_ = s.withCacheManager(func(manager cacheRuntimeManager) error {
		accounts, err := manager.ListCachedAccounts()
		if err != nil {
			return err
		}
		for _, email := range accounts {
			stats, err := manager.GetStats(email)
			if err != nil {
				continue
			}

			info := CacheAccountInfo{
				Email:        email,
				SizeBytes:    stats.SizeBytes,
				EmailCount:   stats.EmailCount,
				EventCount:   stats.EventCount,
				ContactCount: stats.ContactCount,
			}
			if !stats.LastSync.IsZero() {
				info.LastSync = &stats.LastSync
			}

			response.Accounts = append(response.Accounts, info)
			response.TotalSizeBytes += stats.SizeBytes

			// Track latest sync time
			if response.LastSync == nil || (info.LastSync != nil && info.LastSync.After(*response.LastSync)) {
				response.LastSync = info.LastSync
			}
		}
		return nil
	})

	// Count pending actions across all queues
	for _, email := range s.offlineQueueEmails() {
		_ = s.withOfflineQueue(email, func(queue *cache.OfflineQueue) error {
			count, _ := queue.Count()
			response.PendingActions += count
			return nil
		})
	}

	writeJSON(w, http.StatusOK, response)
}

// handleCacheSync triggers a manual sync.
func (s *Server) handleCacheSync(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodPost) {
		return
	}
	if s.handleDemoMode(w, CacheSyncResponse{Success: true, Message: "Demo mode: sync simulated"}) {
		return
	}

	if !s.hasCacheRuntime() {
		writeJSON(w, http.StatusOK, CacheSyncResponse{
			Success: false,
			Error:   "Cache not initialized",
		})
		return
	}

	if !s.IsOnline() {
		writeJSON(w, http.StatusOK, CacheSyncResponse{
			Success: false,
			Error:   "Offline mode: cannot sync",
		})
		return
	}

	// Get email from query param (optional - sync all if not specified)
	email := r.URL.Query().Get("email")

	// Get all grants
	grants, err := s.grantStore.ListGrants()
	if err != nil {
		writeJSON(w, http.StatusOK, CacheSyncResponse{
			Success: false,
			Error:   "Failed to get accounts",
		})
		return
	}

	// Sync accounts
	synced := 0
	for _, grant := range grants {
		if !grant.Provider.IsSupportedByAir() {
			continue
		}
		if email != "" && grant.Email != email {
			continue
		}

		s.syncAccount(grant.Email, grant.ID)
		synced++
	}

	writeJSON(w, http.StatusOK, CacheSyncResponse{
		Success: true,
		Message: fmt.Sprintf("Synced %d account(s)", synced),
	})
}

// handleCacheClear clears the cache.
func (s *Server) handleCacheClear(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodPost) {
		return
	}
	if s.handleDemoMode(w, CacheSyncResponse{Success: true, Message: "Demo mode: cache clear simulated"}) {
		return
	}

	if !s.hasCacheRuntime() {
		writeJSON(w, http.StatusOK, CacheSyncResponse{
			Success: false,
			Error:   "Cache not initialized",
		})
		return
	}

	// Get email from query param (optional - clear all if not specified)
	email := r.URL.Query().Get("email")

	if email != "" {
		// Clear single account
		if err := s.withCacheManager(func(manager cacheRuntimeManager) error {
			return manager.ClearCache(email)
		}); err != nil {
			writeJSON(w, http.StatusOK, CacheSyncResponse{
				Success: false,
				Error:   "Failed to clear cache: " + err.Error(),
			})
			return
		}
		s.offlineQueuesMu.Lock()
		delete(s.offlineQueues, email)
		s.offlineQueuesMu.Unlock()
	} else {
		// Clear all accounts
		if err := s.withCacheManager(func(manager cacheRuntimeManager) error {
			return manager.ClearAllCaches()
		}); err != nil {
			writeJSON(w, http.StatusOK, CacheSyncResponse{
				Success: false,
				Error:   "Failed to clear cache: " + err.Error(),
			})
			return
		}
		s.clearOfflineQueues()
	}

	writeJSON(w, http.StatusOK, CacheSyncResponse{
		Success: true,
		Message: "Cache cleared successfully",
	})
}

// handleCacheSearch performs a unified search across cached data.
func (s *Server) handleCacheSearch(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}

	query := r.URL.Query().Get("q")
	if query == "" {
		writeJSON(w, http.StatusOK, CacheSearchResponse{
			Results: []CacheSearchResult{},
			Query:   "",
			Total:   0,
		})
		return
	}

	if s.handleDemoMode(w, CacheSearchResponse{
		Results: []CacheSearchResult{
			{Type: "email", ID: "demo-1", Title: "Meeting Notes", Subtitle: "From: alice@example.com", Date: time.Now().Unix()},
			{Type: "event", ID: "demo-2", Title: "Team Standup", Subtitle: "Tomorrow 10:00 AM", Date: time.Now().Add(24 * time.Hour).Unix()},
			{Type: "contact", ID: "demo-3", Title: "Bob Smith", Subtitle: "bob@example.com", Date: time.Now().Unix()},
		},
		Query: query,
		Total: 3,
	}) {
		return
	}

	if !s.hasCacheRuntime() {
		writeJSON(w, http.StatusOK, CacheSearchResponse{
			Results: []CacheSearchResult{},
			Query:   query,
			Total:   0,
		})
		return
	}

	// Get current user email
	email := s.getCurrentUserEmail()
	if email == "" {
		writeJSON(w, http.StatusOK, CacheSearchResponse{
			Results: []CacheSearchResult{},
			Query:   query,
			Total:   0,
		})
		return
	}

	var results []*cache.UnifiedSearchResult
	if err := s.withAccountDB(email, func(db *sql.DB) error {
		var err error
		results, err = cache.UnifiedSearch(db, query, 20)
		return err
	}); err != nil {
		writeJSON(w, http.StatusOK, CacheSearchResponse{
			Results: []CacheSearchResult{},
			Query:   query,
			Total:   0,
		})
		return
	}

	// Convert to response format
	response := CacheSearchResponse{
		Query:   query,
		Total:   len(results),
		Results: make([]CacheSearchResult, 0, len(results)),
	}

	for _, r := range results {
		response.Results = append(response.Results, CacheSearchResult{
			Type:     r.Type,
			ID:       r.ID,
			Title:    r.Title,
			Subtitle: r.Subtitle,
			Date:     r.Date.Unix(),
		})
	}

	writeJSON(w, http.StatusOK, response)
}

// handleCacheSettings handles cache settings get/update.
func (s *Server) handleCacheSettings(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.getCacheSettings(w, r)
	case http.MethodPut:
		s.updateCacheSettings(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// getCacheSettings returns current cache settings.
func (s *Server) getCacheSettings(w http.ResponseWriter, _ *http.Request) {
	if s.demoMode || s.cacheSettings == nil {
		writeJSON(w, http.StatusOK, CacheSettingsResponse{
			Enabled:             true,
			MaxSizeMB:           500,
			TTLDays:             30,
			SyncIntervalMinutes: 5,
			OfflineQueueEnabled: true,
			EncryptionEnabled:   false,
			Theme:               "dark",
			DefaultView:         "email",
			CompactMode:         false,
			PreviewPosition:     "right",
		})
		return
	}

	settings := s.cacheSettings.Get()
	writeJSON(w, http.StatusOK, CacheSettingsResponse{
		Enabled:             settings.Enabled,
		MaxSizeMB:           settings.MaxSizeMB,
		TTLDays:             settings.TTLDays,
		SyncIntervalMinutes: settings.SyncIntervalMinutes,
		OfflineQueueEnabled: settings.OfflineQueueEnabled,
		EncryptionEnabled:   settings.EncryptionEnabled,
		Theme:               settings.Theme,
		DefaultView:         settings.DefaultView,
		CompactMode:         settings.CompactMode,
		PreviewPosition:     settings.PreviewPosition,
	})
}

// updateCacheSettings updates cache settings.
func (s *Server) updateCacheSettings(w http.ResponseWriter, r *http.Request) {
	if s.handleDemoMode(w, map[string]any{"success": true, "message": "Demo mode: settings update simulated"}) {
		return
	}

	if s.cacheSettings == nil {
		writeJSON(w, http.StatusOK, map[string]any{
			"success": false,
			"error":   "Cache not initialized",
		})
		return
	}

	var req CacheSettingsResponse
	if !parseJSONBody(w, r, &req) {
		return
	}

	current := s.cacheSettings.Get()

	// Update settings
	err := s.cacheSettings.Update(func(s *cache.Settings) {
		s.Enabled = req.Enabled
		s.MaxSizeMB = req.MaxSizeMB
		s.TTLDays = req.TTLDays
		s.SyncIntervalMinutes = req.SyncIntervalMinutes
		s.OfflineQueueEnabled = req.OfflineQueueEnabled
		s.EncryptionEnabled = req.EncryptionEnabled
		s.Theme = req.Theme
		s.DefaultView = req.DefaultView
		s.CompactMode = req.CompactMode
		s.PreviewPosition = req.PreviewPosition
	})

	if err != nil {
		writeJSON(w, http.StatusOK, map[string]any{
			"success": false,
			"error":   "Failed to save settings: " + err.Error(),
		})
		return
	}

	if current.Enabled != req.Enabled ||
		current.OfflineQueueEnabled != req.OfflineQueueEnabled ||
		current.EncryptionEnabled != req.EncryptionEnabled {
		if err := s.reconfigureCacheRuntime(); err != nil {
			_ = s.cacheSettings.Update(func(settings *cache.Settings) {
				settings.Enabled = current.Enabled
				settings.MaxSizeMB = current.MaxSizeMB
				settings.TTLDays = current.TTLDays
				settings.SyncIntervalMinutes = current.SyncIntervalMinutes
				settings.OfflineQueueEnabled = current.OfflineQueueEnabled
				settings.EncryptionEnabled = current.EncryptionEnabled
				settings.Theme = current.Theme
				settings.DefaultView = current.DefaultView
				settings.CompactMode = current.CompactMode
				settings.PreviewPosition = current.PreviewPosition
			})
			_ = s.reconfigureCacheRuntime()
			writeJSON(w, http.StatusOK, map[string]any{
				"success": false,
				"error":   "Failed to apply runtime settings: " + err.Error(),
			})
			return
		}
	}

	if current.SyncIntervalMinutes != req.SyncIntervalMinutes &&
		current.Enabled == req.Enabled &&
		current.OfflineQueueEnabled == req.OfflineQueueEnabled &&
		current.EncryptionEnabled == req.EncryptionEnabled {
		s.restartBackgroundSync()
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"success": true,
		"message": "Settings updated successfully",
	})
}

// getCurrentUserEmail returns the current user's email address.
func (s *Server) getCurrentUserEmail() string {
	grant, err := s.resolveDefaultGrantInfo()
	if err != nil {
		return ""
	}

	return grant.Email
}
