package air

import (
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"time"

	"github.com/nylas/cli/internal/adapters/config"
	"github.com/nylas/cli/internal/adapters/keyring"
	"github.com/nylas/cli/internal/adapters/nylas"
	"github.com/nylas/cli/internal/air/cache"
	authapp "github.com/nylas/cli/internal/app/auth"
	"github.com/nylas/cli/internal/ports"
)

// NewServer creates a new Air server.
func NewServer(addr string) *Server {
	configStore := config.NewDefaultFileStore()
	secretStore, _ := keyring.NewSecretStore(config.DefaultConfigDir())
	grantStore := keyring.NewGrantStore(secretStore)
	configSvc := authapp.NewConfigService(configStore, secretStore)

	// Create Nylas client for API calls (keyring only - env vars are for integration tests)
	var nylasClient ports.NylasClient
	var hasAPIKey bool
	cfg, err := configStore.Load()
	if err == nil {
		apiKey, _ := secretStore.Get(ports.KeyAPIKey)
		clientID, _ := secretStore.Get(ports.KeyClientID)
		clientSecret, _ := secretStore.Get(ports.KeyClientSecret)

		hasAPIKey = apiKey != ""
		if hasAPIKey {
			client := nylas.NewHTTPClient()
			client.SetRegion(cfg.Region)
			client.SetCredentials(clientID, clientSecret, apiKey)
			nylasClient = client
		}
	}

	// Load templates
	tmpl, err := loadTemplates()
	if err != nil {
		// Log error and fall back to nil
		fmt.Fprintf(os.Stderr, "Warning: Failed to load templates: %v\n", err)
		tmpl = nil
	}

	// Load cache settings; runtime cache components are initialized at server start.
	cacheCfg := cache.DefaultConfig()
	cacheSettings, _ := cache.LoadSettings(cacheCfg.BasePath)

	return &Server{
		addr:          addr,
		demoMode:      false,
		configSvc:     configSvc,
		configStore:   configStore,
		secretStore:   secretStore,
		grantStore:    grantStore,
		nylasClient:   nylasClient,
		templates:     tmpl,
		hasAPIKey:     hasAPIKey,
		cacheSettings: cacheSettings,
		offlineQueues: make(map[string]*cache.OfflineQueue),
		syncStopCh:    make(chan struct{}),
		isOnline:      true,
	}
}

// initCacheRuntime initializes runtime cache components for the server.
// This is intentionally deferred until Start() so NewServer remains lightweight.
func (s *Server) initCacheRuntime() {
	if s.demoMode || s.cacheManager != nil {
		return
	}

	cacheCfg := cache.DefaultConfig()

	// If settings weren't loaded during construction, best-effort load them now.
	if s.cacheSettings == nil {
		settings, err := cache.LoadSettings(cacheCfg.BasePath)
		if err != nil {
			return
		}
		s.cacheSettings = settings
	}

	// Respect cache enablement from settings.
	if !s.cacheSettings.IsCacheEnabled() {
		return
	}

	cacheCfg = s.cacheSettings.ToConfig(cacheCfg.BasePath)

	cacheManager, err := cache.NewManager(cacheCfg)
	if err != nil {
		return
	}
	s.cacheManager = cacheManager

	photoDB, err := cache.OpenSharedDB(cacheCfg.BasePath, "photos.db")
	if err != nil {
		return
	}

	photoStore, err := cache.NewPhotoStore(photoDB, cacheCfg.BasePath, cache.DefaultPhotoTTL)
	if err != nil {
		return
	}
	s.photoStore = photoStore

	// Prune expired photos asynchronously after startup.
	go func() {
		if pruned, err := photoStore.Prune(); err == nil && pruned > 0 {
			fmt.Fprintf(os.Stderr, "Pruned %d expired photos from cache\n", pruned)
		}
	}()
}

// NewDemoServer creates an Air server in demo mode with sample data.
func NewDemoServer(addr string) *Server {
	tmpl, err := loadTemplates()
	if err != nil {
		tmpl = nil
	}

	return &Server{
		addr:      addr,
		demoMode:  true,
		templates: tmpl,
	}
}

// Start starts the HTTP server.
func (s *Server) Start() error {
	mux := http.NewServeMux()

	// API routes - Config & Grants
	mux.HandleFunc("/api/config", s.handleConfigStatus)
	mux.HandleFunc("/api/grants", s.handleListGrants)
	mux.HandleFunc("/api/grants/default", s.handleSetDefaultGrant)

	// API routes - Email (Phase 3)
	mux.HandleFunc("/api/folders", s.handleListFolders)
	mux.HandleFunc("/api/emails", s.handleListEmails)
	mux.HandleFunc("/api/emails/", s.handleEmailByID) // Handles /api/emails/:id and actions

	// API routes - Compose & Send (Phase 3)
	mux.HandleFunc("/api/drafts", s.handleDrafts)     // POST to create, GET to list
	mux.HandleFunc("/api/drafts/", s.handleDraftByID) // GET, PUT, DELETE, POST .../send
	mux.HandleFunc("/api/send", s.handleSendMessage)  // POST to send directly

	// API routes - Calendar (Phase 4)
	mux.HandleFunc("/api/calendars", s.handleListCalendars)    // GET calendars
	mux.HandleFunc("/api/events", s.handleEventsRoute)         // GET list, POST create
	mux.HandleFunc("/api/events/conflicts", s.handleConflicts) // GET conflicts for time range
	mux.HandleFunc("/api/events/", s.handleEventByID)          // GET, PUT, DELETE by ID
	mux.HandleFunc("/api/availability", s.handleAvailability)  // GET/POST find available times
	mux.HandleFunc("/api/freebusy", s.handleFreeBusy)          // GET/POST free/busy info

	// API routes - Contacts (Phase 5)
	mux.HandleFunc("/api/contacts", s.handleContactsRoute)        // GET list, POST create
	mux.HandleFunc("/api/contacts/search", s.handleContactSearch) // GET search contacts
	mux.HandleFunc("/api/contacts/", s.handleContactByID)         // GET, PUT, DELETE by ID
	mux.HandleFunc("/api/contact-groups", s.handleContactGroups)  // GET groups

	// API routes - Productivity (Phase 6)
	mux.HandleFunc("/api/inbox/split", s.handleSplitInbox)           // GET/PUT split inbox config
	mux.HandleFunc("/api/inbox/categorize", s.handleCategorizeEmail) // POST categorize email
	mux.HandleFunc("/api/inbox/vip", s.handleVIPSenders)             // GET/POST/DELETE VIP senders
	mux.HandleFunc("/api/snooze", s.handleSnooze)                    // GET/POST/DELETE snooze
	mux.HandleFunc("/api/scheduled", s.handleScheduledSend)          // GET/POST/DELETE scheduled send
	mux.HandleFunc("/api/undo-send", s.handleUndoSend)               // GET/PUT/POST undo send
	mux.HandleFunc("/api/pending-sends", s.handlePendingSends)       // GET pending sends
	mux.HandleFunc("/api/templates", s.handleTemplates)              // GET/POST email templates
	mux.HandleFunc("/api/templates/", s.handleTemplateByID)          // GET/PUT/DELETE/expand template

	// API routes - Cache (Phase 8)
	mux.HandleFunc("/api/cache/status", s.handleCacheStatus)     // GET cache status
	mux.HandleFunc("/api/cache/sync", s.handleCacheSync)         // POST trigger sync
	mux.HandleFunc("/api/cache/clear", s.handleCacheClear)       // POST clear cache
	mux.HandleFunc("/api/cache/search", s.handleCacheSearch)     // GET search cached data
	mux.HandleFunc("/api/cache/settings", s.handleCacheSettings) // GET/PUT cache settings

	// API routes - AI (Claude Code integration)
	mux.HandleFunc("/api/ai/summarize", s.handleAISummarize)              // POST summarize email
	mux.HandleFunc("/api/ai/smart-replies", s.handleAISmartReplies)       // POST smart reply suggestions
	mux.HandleFunc("/api/ai/enhanced-summary", s.handleAIEnhancedSummary) // POST enhanced summary with action items
	mux.HandleFunc("/api/ai/auto-label", s.handleAIAutoLabel)             // POST auto-label email
	mux.HandleFunc("/api/ai/thread-summary", s.handleAIThreadSummary)     // POST summarize email thread
	mux.HandleFunc("/api/ai/complete", s.handleAIComplete)                // POST smart compose autocomplete
	mux.HandleFunc("/api/ai/nl-search", s.handleNLSearch)                 // POST natural language search

	// API routes - Bundles (smart email categorization)
	mux.HandleFunc("/api/bundles", s.handleGetBundles)                  // GET list bundles
	mux.HandleFunc("/api/bundles/categorize", s.handleBundleCategorize) // POST categorize into bundle
	mux.HandleFunc("/api/bundles/emails", s.handleGetBundleEmails)      // GET emails in bundle

	// API routes - Notetaker (meeting recordings)
	mux.HandleFunc("/api/notetakers", s.handleNotetakersRoute)         // GET list, POST create
	mux.HandleFunc("/api/notetakers/", s.handleNotetakerByID)          // GET, DELETE by ID
	mux.HandleFunc("/api/notetakers/media", s.handleGetNotetakerMedia) // GET media for notetaker

	// API routes - Screener (sender approval)
	mux.HandleFunc("/api/screener", s.handleGetScreenedSenders)  // GET pending senders
	mux.HandleFunc("/api/screener/add", s.handleAddToScreener)   // POST add to screener
	mux.HandleFunc("/api/screener/allow", s.handleScreenerAllow) // POST allow sender
	mux.HandleFunc("/api/screener/block", s.handleScreenerBlock) // POST block sender

	// API routes - AI Configuration
	mux.HandleFunc("/api/ai/config", s.handleAIConfigRoute)     // GET/PUT AI config
	mux.HandleFunc("/api/ai/test", s.handleTestAIConnection)    // POST test connection
	mux.HandleFunc("/api/ai/usage", s.handleGetAIUsage)         // GET usage stats
	mux.HandleFunc("/api/ai/providers", s.handleGetAIProviders) // GET available providers

	// API routes - Read Receipts
	mux.HandleFunc("/api/receipts", s.handleGetReadReceipts)              // GET receipts
	mux.HandleFunc("/api/receipts/settings", s.handleReadReceiptSettings) // GET/PUT settings
	mux.HandleFunc("/api/track/open", s.handleTrackOpen)                  // GET tracking pixel

	// API routes - Reply Later
	mux.HandleFunc("/api/reply-later", s.handleReplyLaterRoute)             // GET list, POST add
	mux.HandleFunc("/api/reply-later/update", s.handleUpdateReplyLater)     // PUT update
	mux.HandleFunc("/api/reply-later/remove", s.handleRemoveFromReplyLater) // DELETE remove

	// API routes - Focus Mode
	mux.HandleFunc("/api/focus", s.handleFocusModeRoute)             // GET state, POST start, DELETE stop
	mux.HandleFunc("/api/focus/break", s.handleStartBreak)           // POST start break
	mux.HandleFunc("/api/focus/settings", s.handleFocusModeSettings) // GET/PUT settings

	// API routes - Analytics
	mux.HandleFunc("/api/analytics/dashboard", s.handleGetAnalyticsDashboard)    // GET dashboard
	mux.HandleFunc("/api/analytics/trends", s.handleGetAnalyticsTrends)          // GET trends
	mux.HandleFunc("/api/analytics/focus-time", s.handleGetFocusTimeSuggestions) // GET suggestions
	mux.HandleFunc("/api/analytics/productivity", s.handleGetProductivityStats)  // GET productivity

	// Static files (CSS, JS, icons)
	staticFS, _ := fs.Sub(staticFiles, "static")
	fileServer := http.FileServer(http.FS(staticFS))

	// Wrap JS files with no-cache headers to prevent stale caching
	noCacheJS := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
		w.Header().Set("Pragma", "no-cache")
		w.Header().Set("Expires", "0")
		fileServer.ServeHTTP(w, r)
	})

	// Serve static files for specific paths
	mux.Handle("/css/", fileServer)
	mux.Handle("/js/", noCacheJS)
	mux.Handle("/icons/", fileServer)

	// Service worker (must be served from root for proper scope)
	mux.HandleFunc("/sw.js", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/javascript")
		w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
		data, err := staticFiles.ReadFile("static/sw.js")
		if err != nil {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write(data)
	})

	// Favicon
	mux.HandleFunc("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/svg+xml")
		data, err := staticFiles.ReadFile("static/favicon.svg")
		if err != nil {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write(data)
	})
	mux.HandleFunc("/favicon.svg", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/svg+xml")
		data, err := staticFiles.ReadFile("static/favicon.svg")
		if err != nil {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write(data)
	})

	// Template-rendered index page
	mux.HandleFunc("/", s.handleIndex)

	// Initialize cache runtime components after routes are wired.
	s.initCacheRuntime()

	// Start background sync if cache is available and enabled.
	if !s.demoMode && s.cacheManager != nil && s.cacheSettings != nil && s.cacheSettings.IsCacheEnabled() {
		s.startBackgroundSync()
	}

	// Apply middleware chain for performance and security
	// Order matters: CORS → Security → Compression → Cache → Monitoring → MethodOverride → Handler
	handler := CORSMiddleware(
		SecurityHeadersMiddleware(
			CompressionMiddleware(
				CacheMiddleware(
					PerformanceMonitoringMiddleware(
						MethodOverrideMiddleware(mux))))))

	server := &http.Server{
		Addr:              s.addr,
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       120 * time.Second,
		MaxHeaderBytes:    1 << 20, // 1 MB
	}

	return server.ListenAndServe()
}

// Stop gracefully stops the server and background processes.
func (s *Server) Stop() error {
	// Signal background sync to stop
	close(s.syncStopCh)

	// Wait for sync goroutines to finish
	s.syncWg.Wait()

	// Close cache manager
	if s.cacheManager != nil {
		return s.cacheManager.Close()
	}

	return nil
}

// IsOnline returns whether the server has network connectivity.
func (s *Server) IsOnline() bool {
	s.onlineMu.RLock()
	defer s.onlineMu.RUnlock()
	return s.isOnline
}

// SetOnline updates the online status.
func (s *Server) SetOnline(online bool) {
	s.onlineMu.Lock()
	s.isOnline = online
	s.onlineMu.Unlock()

	// If coming back online, process offline queue
	if online && s.cacheManager != nil {
		s.processOfflineQueues()
	}
}
