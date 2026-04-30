package air

import (
	"encoding/json"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/nylas/cli/internal/httputil"
)

// Bundle represents an email bundle/category
type Bundle struct {
	ID          string       `json:"id"`
	Name        string       `json:"name"`
	Icon        string       `json:"icon"`
	Description string       `json:"description"`
	Rules       []BundleRule `json:"rules"`
	Collapsed   bool         `json:"collapsed"`
	Count       int          `json:"count"`
	UnreadCount int          `json:"unreadCount"`
	LastUpdated time.Time    `json:"lastUpdated"`
}

// BundleRule defines matching criteria for emails
type BundleRule struct {
	Field    string `json:"field"`    // from, to, subject, domain
	Operator string `json:"operator"` // contains, equals, matches, startsWith
	Value    string `json:"value"`
	Priority int    `json:"priority"`
}

// BundledEmail represents an email assigned to a bundle
type BundledEmail struct {
	EmailID  string `json:"emailId"`
	BundleID string `json:"bundleId"`
	Score    int    `json:"score"` // Match confidence
}

// BundleStore manages bundles in memory
type BundleStore struct {
	bundles map[string]*Bundle
	emails  map[string]string // emailID -> bundleID
	mu      sync.RWMutex
}

// NewBundleStore creates a new bundle store with defaults
func NewBundleStore() *BundleStore {
	store := &BundleStore{
		bundles: make(map[string]*Bundle),
		emails:  make(map[string]string),
	}
	store.initDefaultBundles()
	return store
}

// initDefaultBundles sets up default smart bundles
func (bs *BundleStore) initDefaultBundles() {
	defaults := []Bundle{
		{
			ID: "newsletters", Name: "Newsletters", Icon: "📰",
			Description: "Newsletter subscriptions",
			Rules: []BundleRule{
				{Field: "from", Operator: "contains", Value: "newsletter"},
				{Field: "from", Operator: "contains", Value: "digest"},
				{Field: "subject", Operator: "contains", Value: "unsubscribe"},
			},
			Collapsed: true,
		},
		{
			ID: "receipts", Name: "Receipts & Orders", Icon: "🧾",
			Description: "Purchase confirmations and receipts",
			Rules: []BundleRule{
				{Field: "subject", Operator: "contains", Value: "receipt"},
				{Field: "subject", Operator: "contains", Value: "order confirm"},
				{Field: "subject", Operator: "contains", Value: "your order"},
				{Field: "from", Operator: "contains", Value: "noreply"},
			},
			Collapsed: true,
		},
		{
			ID: "social", Name: "Social", Icon: "👥",
			Description: "Social media notifications",
			Rules: []BundleRule{
				{Field: "domain", Operator: "equals", Value: "twitter.com"},
				{Field: "domain", Operator: "equals", Value: "facebook.com"},
				{Field: "domain", Operator: "equals", Value: "linkedin.com"},
				{Field: "domain", Operator: "equals", Value: "instagram.com"},
			},
			Collapsed: true,
		},
		{
			ID: "updates", Name: "Updates", Icon: "🔔",
			Description: "Service updates and notifications",
			Rules: []BundleRule{
				{Field: "from", Operator: "contains", Value: "notifications"},
				{Field: "from", Operator: "contains", Value: "updates"},
				{Field: "subject", Operator: "contains", Value: "update"},
			},
			Collapsed: true,
		},
		{
			ID: "promotions", Name: "Promotions", Icon: "🏷️",
			Description: "Deals and promotional emails",
			Rules: []BundleRule{
				{Field: "subject", Operator: "contains", Value: "% off"},
				{Field: "subject", Operator: "contains", Value: "sale"},
				{Field: "subject", Operator: "contains", Value: "discount"},
				{Field: "subject", Operator: "contains", Value: "deal"},
			},
			Collapsed: true,
		},
		{
			ID: "finance", Name: "Finance", Icon: "💰",
			Description: "Banking and financial emails",
			Rules: []BundleRule{
				{Field: "domain", Operator: "contains", Value: "bank"},
				{Field: "subject", Operator: "contains", Value: "statement"},
				{Field: "subject", Operator: "contains", Value: "transaction"},
				{Field: "subject", Operator: "contains", Value: "payment"},
			},
			Collapsed: true,
		},
		{
			ID: "travel", Name: "Travel", Icon: "✈️",
			Description: "Flight, hotel, and travel bookings",
			Rules: []BundleRule{
				{Field: "subject", Operator: "contains", Value: "booking"},
				{Field: "subject", Operator: "contains", Value: "itinerary"},
				{Field: "subject", Operator: "contains", Value: "flight"},
				{Field: "subject", Operator: "contains", Value: "reservation"},
			},
			Collapsed: true,
		},
	}

	for i := range defaults {
		// Create a heap-allocated copy to avoid pointer aliasing
		bundle := new(Bundle)
		*bundle = defaults[i]
		bundle.LastUpdated = time.Now()
		bs.bundles[bundle.ID] = bundle
	}
}

// Global bundle store
var bundleStore = NewBundleStore()

// handleGetBundles returns all bundles
func (s *Server) handleGetBundles(w http.ResponseWriter, r *http.Request) {
	bundleStore.mu.RLock()
	defer bundleStore.mu.RUnlock()

	bundles := make([]*Bundle, 0, len(bundleStore.bundles))
	for _, b := range bundleStore.bundles {
		bundles = append(bundles, b)
	}

	httputil.WriteJSON(w, http.StatusOK, bundles)
}

// handleBundleCategorize assigns an email to a bundle
func (s *Server) handleBundleCategorize(w http.ResponseWriter, r *http.Request) {
	var req struct {
		From    string `json:"from"`
		Subject string `json:"subject"`
		EmailID string `json:"emailId"`
	}

	if err := json.NewDecoder(limitedBody(w, r)).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	bundleID := categorizeEmail(req.From, req.Subject)

	if req.EmailID != "" && bundleID != "" {
		func() {
			bundleStore.mu.Lock()
			defer bundleStore.mu.Unlock()
			bundleStore.emails[req.EmailID] = bundleID
			if b, ok := bundleStore.bundles[bundleID]; ok {
				b.Count++
				b.LastUpdated = time.Now()
			}
		}()
	}

	httputil.WriteJSON(w, http.StatusOK, map[string]string{"bundleId": bundleID})
}

// categorizeEmail determines which bundle an email belongs to
func categorizeEmail(from, subject string) string {
	bundleStore.mu.RLock()
	defer bundleStore.mu.RUnlock()

	fromLower := strings.ToLower(from)
	subjectLower := strings.ToLower(subject)

	// Extract domain from email
	domain := extractDomain(from)

	bestMatch := ""
	bestScore := 0

	for _, bundle := range bundleStore.bundles {
		score := 0
		for _, rule := range bundle.Rules {
			if matchRule(rule, fromLower, subjectLower, domain) {
				score += rule.Priority + 1
			}
		}
		if score > bestScore {
			bestScore = score
			bestMatch = bundle.ID
		}
	}

	return bestMatch
}

// matchRule checks if a rule matches the email
func matchRule(rule BundleRule, from, subject, domain string) bool {
	var fieldValue string
	switch rule.Field {
	case "from":
		fieldValue = from
	case "subject":
		fieldValue = subject
	case "domain":
		fieldValue = domain
	default:
		return false
	}

	valueLower := strings.ToLower(rule.Value)

	switch rule.Operator {
	case "contains":
		return strings.Contains(fieldValue, valueLower)
	case "equals":
		return fieldValue == valueLower
	case "startsWith":
		return strings.HasPrefix(fieldValue, valueLower)
	case "matches":
		// Compile-and-match through a cached compile so a pathological
		// pattern can't recompile on every email. Bad patterns are
		// rejected at write-time by validateBundleRule, but old/persisted
		// rules go through this path too — fall through quietly to keep
		// the matcher panic-free.
		re, err := compiledBundleRegex(rule.Value)
		if err != nil || re == nil {
			return false
		}
		return re.MatchString(fieldValue)
	default:
		return false
	}
}

// bundleRegexCache caches compiled "matches" patterns. Bundle rules are
// re-applied to every message arrival, so re-running regexp.Compile per
// match would be both slow and an attack surface (catastrophic backtracking
// patterns get compiled and burned per call).
var (
	bundleRegexCache   = make(map[string]*regexp.Regexp)
	bundleRegexCacheMu sync.RWMutex
)

func compiledBundleRegex(pattern string) (*regexp.Regexp, error) {
	bundleRegexCacheMu.RLock()
	if re, ok := bundleRegexCache[pattern]; ok {
		bundleRegexCacheMu.RUnlock()
		return re, nil
	}
	bundleRegexCacheMu.RUnlock()

	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, err
	}
	bundleRegexCacheMu.Lock()
	bundleRegexCache[pattern] = re
	bundleRegexCacheMu.Unlock()
	return re, nil
}

// validateBundleRule rejects rules whose regex fails to compile so we
// surface the error at PUT time instead of silently dropping every match.
func validateBundleRule(rule BundleRule) error {
	if rule.Operator != "matches" || rule.Value == "" {
		return nil
	}
	if _, err := compiledBundleRegex(rule.Value); err != nil {
		return err
	}
	return nil
}

// extractDomain extracts domain from email address
func extractDomain(email string) string {
	atIdx := strings.LastIndex(email, "@")
	if atIdx == -1 || atIdx >= len(email)-1 {
		return ""
	}

	domain := email[atIdx+1:]
	// Remove trailing > if present
	domain = strings.TrimSuffix(domain, ">")
	return strings.ToLower(domain)
}

// handleUpdateBundle updates a bundle configuration
func (s *Server) handleUpdateBundle(w http.ResponseWriter, r *http.Request) {
	var bundle Bundle
	if err := json.NewDecoder(limitedBody(w, r)).Decode(&bundle); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	for i, rule := range bundle.Rules {
		if err := validateBundleRule(rule); err != nil {
			http.Error(w, "Invalid regex in rule "+strconv.Itoa(i)+": "+err.Error(), http.StatusBadRequest)
			return
		}
	}

	bundleStore.mu.Lock()
	defer bundleStore.mu.Unlock()

	if existing, ok := bundleStore.bundles[bundle.ID]; ok {
		// Only update provided fields
		if bundle.Name != "" {
			existing.Name = bundle.Name
		}
		if bundle.Icon != "" {
			existing.Icon = bundle.Icon
		}
		// Only update rules if explicitly provided
		if len(bundle.Rules) > 0 {
			existing.Rules = bundle.Rules
		}
		existing.Collapsed = bundle.Collapsed
		existing.LastUpdated = time.Now()
	}

	httputil.WriteJSON(w, http.StatusOK, bundle)
}

// handleGetBundleEmails returns emails for a specific bundle
func (s *Server) handleGetBundleEmails(w http.ResponseWriter, r *http.Request) {
	bundleID := r.URL.Query().Get("bundleId")
	if bundleID == "" {
		http.Error(w, "bundleId required", http.StatusBadRequest)
		return
	}

	bundleStore.mu.RLock()
	defer bundleStore.mu.RUnlock()

	emailIDs := []string{}
	for emailID, bID := range bundleStore.emails {
		if bID == bundleID {
			emailIDs = append(emailIDs, emailID)
		}
	}

	httputil.WriteJSON(w, http.StatusOK, emailIDs)
}
