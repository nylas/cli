package chat

import (
	"embed"
	"html/template"
	"io/fs"
	"net/http"
	"sync"
	"time"

	"github.com/nylas/cli/internal/ports"
)

//go:embed static/css/*.css static/js/*.js
var staticFiles embed.FS

//go:embed templates/*.gohtml
var templateFiles embed.FS

// Server is the chat web UI HTTP server.
type Server struct {
	addr      string
	agent     *Agent
	agents    []Agent
	agentMu   sync.RWMutex // protects agent switching
	nylas     ports.NylasClient
	slack     ports.SlackClient // nil if not configured
	hasSlack  bool
	grantID   string
	memory    *MemoryStore
	executor  *ToolExecutor
	context   *ContextBuilder
	session   *ActiveSession
	approvals *ApprovalStore
	tmpl      *template.Template
}

// ActiveAgent returns the current agent (thread-safe).
func (s *Server) ActiveAgent() *Agent {
	s.agentMu.RLock()
	defer s.agentMu.RUnlock()
	return s.agent
}

// SetAgent switches the active agent by type. Returns false if not found.
func (s *Server) SetAgent(agentType AgentType) bool {
	agent := FindAgent(s.agents, agentType)
	if agent == nil {
		return false
	}
	s.agentMu.Lock()
	s.agent = agent
	s.context = NewContextBuilder(agent, s.memory, s.grantID, s.hasSlack)
	s.agentMu.Unlock()
	return true
}

// NewServer creates a new chat Server.
func NewServer(addr string, agent *Agent, agents []Agent, nylas ports.NylasClient, grantID string, memory *MemoryStore, slack ports.SlackClient) *Server {
	hasSlack := slack != nil
	executor := NewToolExecutor(nylas, grantID, slack)
	ctx := NewContextBuilder(agent, memory, grantID, hasSlack)

	tmpl, _ := template.New("").ParseFS(templateFiles, "templates/*.gohtml")

	return &Server{
		addr:      addr,
		agent:     agent,
		agents:    agents,
		nylas:     nylas,
		slack:     slack,
		hasSlack:  hasSlack,
		grantID:   grantID,
		memory:    memory,
		executor:  executor,
		context:   ctx,
		session:   NewActiveSession(),
		approvals: NewApprovalStore(),
		tmpl:      tmpl,
	}
}

// Start starts the HTTP server and blocks until interrupted.
func (s *Server) Start() error {
	mux := http.NewServeMux()

	// API routes
	mux.HandleFunc("/api/chat", s.handleChat)
	mux.HandleFunc("/api/chat/approve", s.handleApprove)
	mux.HandleFunc("/api/chat/reject", s.handleReject)
	mux.HandleFunc("/api/command", s.handleCommand)
	mux.HandleFunc("/api/conversations", s.handleConversations)
	mux.HandleFunc("/api/conversations/", s.handleConversationByID)
	mux.HandleFunc("/api/config", s.handleConfig)
	mux.HandleFunc("/api/health", s.handleHealth)

	// Static files
	staticFS, _ := fs.Sub(staticFiles, "static")
	fileServer := http.FileServer(http.FS(staticFS))

	noCacheHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
		fileServer.ServeHTTP(w, r)
	})

	mux.Handle("/css/", noCacheHandler)
	mux.Handle("/js/", noCacheHandler)

	// Index page
	mux.HandleFunc("/", s.handleIndex)

	server := &http.Server{
		Addr:              s.addr,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
		WriteTimeout:      360 * time.Second, // long for SSE streaming + approval gating
		IdleTimeout:       120 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}

	return server.ListenAndServe()
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	if s.tmpl == nil {
		http.Error(w, "templates not loaded", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = s.tmpl.ExecuteTemplate(w, "index.gohtml", nil)
}
