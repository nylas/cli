// Package studio serves Agent Studio, the local web UI for managing agent
// accounts, workspaces, policies, rules, and lists started by
// `nylas agent studio`.
package studio

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/nylas/cli/internal/ports"
	"github.com/nylas/cli/internal/webguard"
)

const requestTimeout = 30 * time.Second

// Server hosts the Agent Studio page and its JSON API.
type Server struct {
	addr        string
	nylasClient ports.NylasClient
	httpServer  *http.Server

	testEmailMu   sync.Mutex
	testEmailLast map[string]time.Time
}

// NewServer creates an Agent Studio server bound to addr.
func NewServer(addr string, client ports.NylasClient) *Server {
	return &Server{
		addr:        addr,
		nylasClient: client,
	}
}

// Start runs the server until the context is canceled.
func (s *Server) Start(ctx context.Context) error {
	mux := http.NewServeMux()
	s.registerRoutes(mux)

	handler := webguard.HostValidationMiddleware(
		webguard.OriginProtectionMiddleware(
			webguard.SecurityHeadersMiddleware(mux)))

	s.httpServer = &http.Server{
		Addr:              s.addr,
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	select {
	case err := <-errCh:
		return fmt.Errorf("studio server: %w", err)
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return s.httpServer.Shutdown(shutdownCtx)
	}
}

func (s *Server) registerRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/", s.handleIndex)
	mux.HandleFunc("/static/", s.handleStatic)
	mux.HandleFunc("/api/board", s.handleBoard)
	mux.HandleFunc("/api/workspaces", s.routeWorkspaces)
	mux.HandleFunc("/api/workspaces/", s.routeWorkspaces)
	mux.HandleFunc("/api/accounts", s.routeAccounts)
	mux.HandleFunc("/api/accounts/", s.routeAccounts)
	mux.HandleFunc("/api/policies", s.routePolicies)
	mux.HandleFunc("/api/policies/", s.routePolicies)
	mux.HandleFunc("/api/rules", s.routeRules)
	mux.HandleFunc("/api/rules/", s.routeRules)
	mux.HandleFunc("/api/lists", s.routeLists)
	mux.HandleFunc("/api/lists/", s.dispatchLists)
	mux.HandleFunc("/api/actions/test-email", s.handleTestEmail)
}

// dispatchLists splits /api/lists/{id} from /api/lists/{id}/items.
func (s *Server) dispatchLists(w http.ResponseWriter, r *http.Request) {
	if strings.HasSuffix(r.URL.Path, "/items") {
		s.routeListItems(w, r)
		return
	}
	s.routeLists(w, r)
}

func (s *Server) withTimeout(r *http.Request) (context.Context, context.CancelFunc) {
	return context.WithTimeout(r.Context(), requestTimeout)
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		slog.Error("studio: encode response", "err", err)
	}
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

// writeUpstreamError logs the raw upstream error and sends only the generic
// message to the browser, so grant IDs and API fragments never leak to the UI.
func writeUpstreamError(w http.ResponseWriter, status int, msg string, err error) {
	slog.Error(msg, "err", err)
	writeError(w, status, msg)
}

func requireMethod(w http.ResponseWriter, r *http.Request, method string) bool {
	if r.Method != method {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return false
	}
	return true
}
