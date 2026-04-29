package air

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/nylas/cli/internal/httputil"
)

// ReadReceipt represents a read receipt for a sent email
type ReadReceipt struct {
	EmailID   string    `json:"emailId"`
	Recipient string    `json:"recipient"`
	OpenedAt  time.Time `json:"openedAt,omitzero"`
	OpenCount int       `json:"openCount"`
	Device    string    `json:"device,omitempty"`
	Location  string    `json:"location,omitempty"`
	UserAgent string    `json:"userAgent,omitempty"`
	IsOpened  bool      `json:"isOpened"`
}

// ReadReceiptSettings represents user settings for read receipts
type ReadReceiptSettings struct {
	Enabled          bool `json:"enabled"`
	TrackOpens       bool `json:"trackOpens"`
	TrackClicks      bool `json:"trackClicks"`
	ShowNotification bool `json:"showNotification"`
	BlockTracking    bool `json:"blockTracking"` // Block tracking pixels in received emails
}

// readReceiptStore holds read receipts
type readReceiptStore struct {
	receipts map[string]*ReadReceipt // emailID -> receipt
	settings *ReadReceiptSettings
	mu       sync.RWMutex
}

var rrStore = &readReceiptStore{
	receipts: make(map[string]*ReadReceipt),
	settings: &ReadReceiptSettings{
		Enabled:          true,
		TrackOpens:       true,
		TrackClicks:      false,
		ShowNotification: true,
		BlockTracking:    true,
	},
}

// handleGetReadReceipts returns read receipts for sent emails
func (s *Server) handleGetReadReceipts(w http.ResponseWriter, r *http.Request) {
	emailID := r.URL.Query().Get("emailId")

	rrStore.mu.RLock()
	defer rrStore.mu.RUnlock()

	if emailID != "" {
		if receipt, ok := rrStore.receipts[emailID]; ok {
			httputil.WriteJSON(w, http.StatusOK, receipt)
			return
		}
		http.Error(w, "Receipt not found", http.StatusNotFound)
		return
	}

	receipts := make([]*ReadReceipt, 0, len(rrStore.receipts))
	for _, r := range rrStore.receipts {
		receipts = append(receipts, r)
	}

	httputil.WriteJSON(w, http.StatusOK, receipts)
}

// handleTrackOpen records an email open (tracking pixel endpoint)
func (s *Server) handleTrackOpen(w http.ResponseWriter, r *http.Request) {
	emailID := r.URL.Query().Get("id")
	if emailID == "" {
		// Return transparent pixel anyway
		w.Header().Set("Content-Type", "image/gif")
		w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
		_, _ = w.Write(transparentPixel)
		return
	}

	rrStore.mu.Lock()
	if receipt, ok := rrStore.receipts[emailID]; ok {
		receipt.OpenCount++
		if !receipt.IsOpened {
			receipt.IsOpened = true
			receipt.OpenedAt = time.Now()
		}
		receipt.UserAgent = r.UserAgent()
		// Could extract device/location from User-Agent and IP
	}
	rrStore.mu.Unlock()

	// Return transparent 1x1 GIF
	w.Header().Set("Content-Type", "image/gif")
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	_, _ = w.Write(transparentPixel)
}

// transparentPixel is a 1x1 transparent GIF
var transparentPixel = []byte{
	0x47, 0x49, 0x46, 0x38, 0x39, 0x61, 0x01, 0x00, 0x01, 0x00,
	0x80, 0x00, 0x00, 0xff, 0xff, 0xff, 0x00, 0x00, 0x00, 0x21,
	0xf9, 0x04, 0x01, 0x00, 0x00, 0x00, 0x00, 0x2c, 0x00, 0x00,
	0x00, 0x00, 0x01, 0x00, 0x01, 0x00, 0x00, 0x02, 0x02, 0x44,
	0x01, 0x00, 0x3b,
}

// handleReadReceiptSettings dispatches settings requests by method
func (s *Server) handleReadReceiptSettings(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.handleGetReadReceiptSettings(w, r)
	case http.MethodPut, http.MethodPost:
		s.handleUpdateReadReceiptSettings(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleGetReadReceiptSettings returns settings
func (s *Server) handleGetReadReceiptSettings(w http.ResponseWriter, r *http.Request) {
	rrStore.mu.RLock()
	defer rrStore.mu.RUnlock()

	httputil.WriteJSON(w, http.StatusOK, rrStore.settings)
}

// handleUpdateReadReceiptSettings updates settings
func (s *Server) handleUpdateReadReceiptSettings(w http.ResponseWriter, r *http.Request) {
	var settings ReadReceiptSettings
	if err := json.NewDecoder(limitedBody(w, r)).Decode(&settings); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	rrStore.mu.Lock()
	rrStore.settings = &settings
	rrStore.mu.Unlock()

	httputil.WriteJSON(w, http.StatusOK, map[string]string{"status": "updated"})
}

// RegisterEmailForTracking registers an email for read tracking
func RegisterEmailForTracking(emailID, recipient string) {
	rrStore.mu.Lock()
	defer rrStore.mu.Unlock()

	rrStore.receipts[emailID] = &ReadReceipt{
		EmailID:   emailID,
		Recipient: recipient,
		OpenCount: 0,
		IsOpened:  false,
	}
}

// GetTrackingPixelURL returns the tracking pixel URL for an email
func GetTrackingPixelURL(emailID string) string {
	return "/api/track/open?id=" + emailID
}
