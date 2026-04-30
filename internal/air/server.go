package air

import (
	"embed"
	"html/template"
	"sync"

	"github.com/nylas/cli/internal/air/cache"
	authapp "github.com/nylas/cli/internal/app/auth"
	"github.com/nylas/cli/internal/ports"
)

//go:embed static/css/*.css static/js/*.js static/icons/* static/sw.js static/favicon.svg
var staticFiles embed.FS

//go:embed templates/*.gohtml templates/partials/*.gohtml templates/pages/*.gohtml
var templateFiles embed.FS

// Server represents the Air web UI server.
type Server struct {
	addr        string
	demoMode    bool
	configSvc   *authapp.ConfigService
	configStore ports.ConfigStore
	secretStore ports.SecretStore
	grantStore  ports.GrantStore
	nylasClient ports.NylasClient
	templates   *template.Template
	hasAPIKey   bool // True if API key is configured (from env vars or keyring)

	// Cache components
	cacheManager    cacheRuntimeManager
	cacheSettings   *cache.Settings
	photoStore      *cache.PhotoStore              // Contact photo cache
	offlineQueues   map[string]*cache.OfflineQueue // Per-email offline queues
	offlineQueuesMu sync.RWMutex                   // Protects offlineQueues
	runtimeMu       sync.RWMutex                   // Protects runtime cache and photo store swaps
	syncMu          sync.Mutex                     // Protects background sync lifecycle
	syncStopCh      chan struct{}                  // Channel to stop background sync
	syncWg          sync.WaitGroup                 // Wait group for sync goroutines
	bgWg            sync.WaitGroup                 // Wait group for fire-and-forget background tasks (cache prune, etc.)
	syncRunning     bool                           // Tracks whether background sync workers are running
	isOnline        bool                           // Online status
	onlineMu        sync.RWMutex                   // Protects isOnline

	// Productivity features (Phase 6)
	splitInboxConfig *SplitInboxConfig        // Split inbox configuration
	splitInboxMu     sync.RWMutex             // Protects splitInboxConfig
	snoozedEmails    map[string]SnoozedEmail  // Snoozed emails by email ID
	snoozeMu         sync.RWMutex             // Protects snoozedEmails
	undoSendConfig   *UndoSendConfig          // Undo send configuration
	undoSendMu       sync.RWMutex             // Protects undoSendConfig
	pendingSends     map[string]PendingSend   // Pending sends in grace period
	pendingSendMu    sync.RWMutex             // Protects pendingSends
	emailTemplates   map[string]EmailTemplate // Email templates
	templatesMu      sync.RWMutex             // Protects emailTemplates
}
