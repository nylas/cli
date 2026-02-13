package chat

import "sync"

// ActiveSession tracks the current conversation for a browser session.
type ActiveSession struct {
	conversationID string
	mu             sync.RWMutex
}

// NewActiveSession creates a new ActiveSession.
func NewActiveSession() *ActiveSession {
	return &ActiveSession{}
}

// Get returns the current conversation ID.
func (s *ActiveSession) Get() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.conversationID
}

// Set updates the current conversation ID.
func (s *ActiveSession) Set(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.conversationID = id
}
