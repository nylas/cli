package rpcserver

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const shutdownTimeout = 5 * time.Second

type Config struct {
	Addr           string
	Token          string
	AllowedOrigins []string
}

type Server struct {
	dispatcher *Dispatcher
	cfg        Config
	upgrader   websocket.Upgrader
	baseCtx    context.Context

	mu    sync.Mutex
	conns map[*websocket.Conn]*clientConn
}

type clientConn struct {
	conn    *websocket.Conn
	writeMu sync.Mutex
}

func NewServer(cfg Config, d *Dispatcher) *Server {
	return &Server{
		dispatcher: d,
		cfg:        cfg,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return ValidateOrigin(r.Header.Get("Origin"), cfg.AllowedOrigins)
			},
		},
		conns: make(map[*websocket.Conn]*clientConn),
	}
}

// Broadcast writes a JSON-RPC notification to every connected client.
func (s *Server) Broadcast(method string, params any) error {
	msg, err := NewNotification(method, params)
	if err != nil {
		return fmt.Errorf("create notification: %w", err)
	}

	s.mu.Lock()
	conns := make([]*clientConn, 0, len(s.conns))
	for _, c := range s.conns {
		conns = append(conns, c)
	}
	s.mu.Unlock()

	for _, c := range conns {
		c.writeMu.Lock()
		err := c.conn.WriteMessage(websocket.TextMessage, msg)
		c.writeMu.Unlock()
		if err != nil {
			s.unregister(c)
		}
	}

	return nil
}

// Serve starts the WebSocket server and blocks until ctx is cancelled or the server fails.
func (s *Server) Serve(ctx context.Context) error {
	s.mu.Lock()
	s.baseCtx = ctx
	s.mu.Unlock()

	httpServer := &http.Server{
		Addr:              s.cfg.Addr,
		Handler:           s.handler(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		err := httpServer.ListenAndServe()
		if errors.Is(err, http.ErrServerClosed) {
			errCh <- nil
			return
		}
		errCh <- err
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()
		if err := httpServer.Shutdown(shutdownCtx); err != nil {
			s.closeConns()
			return fmt.Errorf("shutdown rpc server: %w", err)
		}
		s.closeConns()
		return <-errCh
	}
}

func (s *Server) handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/ws", s.handleWebSocket)
	return mux
}

func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	if !s.authorized(r) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	if !ValidateOrigin(r.Header.Get("Origin"), s.cfg.AllowedOrigins) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}

	c := &clientConn{conn: conn}
	s.register(c)
	defer s.unregister(c)

	s.mu.Lock()
	baseCtx := s.baseCtx
	s.mu.Unlock()
	if baseCtx == nil {
		baseCtx = context.Background()
	}
	connCtx, cancel := context.WithCancel(baseCtx)
	defer cancel()

	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			return
		}
		resp := s.dispatcher.Dispatch(connCtx, msg)
		if resp == nil {
			continue
		}

		c.writeMu.Lock()
		err = conn.WriteMessage(websocket.TextMessage, resp)
		c.writeMu.Unlock()
		if err != nil {
			return
		}
	}
}

func (s *Server) authorized(r *http.Request) bool {
	if token := bearerToken(r.Header.Get("Authorization")); ValidateToken(s.cfg.Token, token) {
		return true
	}
	return ValidateToken(s.cfg.Token, r.URL.Query().Get("token"))
}

func bearerToken(header string) string {
	token, ok := strings.CutPrefix(header, "Bearer ")
	if !ok {
		return ""
	}
	return token
}

func (s *Server) register(c *clientConn) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.conns[c.conn] = c
}

func (s *Server) unregister(c *clientConn) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.conns[c.conn]; ok {
		delete(s.conns, c.conn)
		_ = c.conn.Close()
	}
}

func (s *Server) closeConns() {
	s.mu.Lock()
	conns := make([]*clientConn, 0, len(s.conns))
	for _, c := range s.conns {
		conns = append(conns, c)
	}
	s.mu.Unlock()

	for _, c := range conns {
		s.unregister(c)
	}
}
