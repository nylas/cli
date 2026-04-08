package air

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewBundleStore(t *testing.T) {
	store := NewBundleStore()

	if store == nil {
		t.Fatal("expected non-nil store")
		return
	}

	if len(store.bundles) == 0 {
		t.Error("expected default bundles to be initialized")
	}

	// Check for expected default bundles
	expectedBundles := []string{"newsletters", "receipts", "social", "updates", "promotions", "finance", "travel"}
	for _, id := range expectedBundles {
		if _, ok := store.bundles[id]; !ok {
			t.Errorf("expected bundle %q to exist", id)
		}
	}
}

func TestCategorizeEmail(t *testing.T) {
	tests := []struct {
		name     string
		from     string
		subject  string
		expected string
	}{
		{
			name:     "newsletter detection",
			from:     "newsletter@example.com",
			subject:  "Weekly digest",
			expected: "newsletters",
		},
		{
			name:     "receipt detection",
			from:     "noreply@store.com",
			subject:  "Your order confirmation #12345",
			expected: "receipts",
		},
		{
			name:     "social - twitter",
			from:     "notify@twitter.com",
			subject:  "You have new followers",
			expected: "social",
		},
		{
			name:     "social - linkedin",
			from:     "messages@linkedin.com",
			subject:  "New connection request",
			expected: "social",
		},
		{
			name:     "promotion detection",
			from:     "deals@shop.com",
			subject:  "50% off sale ends today!",
			expected: "promotions",
		},
		{
			name:     "finance detection",
			from:     "alerts@mybank.com",
			subject:  "Your monthly statement is ready",
			expected: "finance",
		},
		{
			name:     "travel detection",
			from:     "bookings@airline.com",
			subject:  "Your flight itinerary",
			expected: "travel",
		},
		{
			name:     "no match - personal email",
			from:     "friend@gmail.com",
			subject:  "Hey, how are you?",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := categorizeEmail(tt.from, tt.subject)
			if result != tt.expected {
				t.Errorf("categorizeEmail(%q, %q) = %q, want %q",
					tt.from, tt.subject, result, tt.expected)
			}
		})
	}
}

func TestExtractDomain(t *testing.T) {
	tests := []struct {
		email    string
		expected string
	}{
		{"user@example.com", "example.com"},
		{"test@twitter.com", "twitter.com"},
		{"<no-reply@linkedin.com>", "linkedin.com"},
		{"invalid-email", ""},
		{"@nodomain.com", "nodomain.com"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.email, func(t *testing.T) {
			result := extractDomain(tt.email)
			if result != tt.expected {
				t.Errorf("extractDomain(%q) = %q, want %q",
					tt.email, result, tt.expected)
			}
		})
	}
}

func TestMatchRule(t *testing.T) {
	tests := []struct {
		name     string
		rule     BundleRule
		from     string
		subject  string
		domain   string
		expected bool
	}{
		{
			name:     "contains match",
			rule:     BundleRule{Field: "from", Operator: "contains", Value: "newsletter"},
			from:     "newsletter@test.com",
			subject:  "",
			domain:   "",
			expected: true,
		},
		{
			name:     "contains no match",
			rule:     BundleRule{Field: "from", Operator: "contains", Value: "newsletter"},
			from:     "user@test.com",
			subject:  "",
			domain:   "",
			expected: false,
		},
		{
			name:     "equals match",
			rule:     BundleRule{Field: "domain", Operator: "equals", Value: "twitter.com"},
			from:     "",
			subject:  "",
			domain:   "twitter.com",
			expected: true,
		},
		{
			name:     "equals no match",
			rule:     BundleRule{Field: "domain", Operator: "equals", Value: "twitter.com"},
			from:     "",
			subject:  "",
			domain:   "facebook.com",
			expected: false,
		},
		{
			name:     "startsWith match",
			rule:     BundleRule{Field: "subject", Operator: "startsWith", Value: "re:"},
			from:     "",
			subject:  "re: hello",
			domain:   "",
			expected: true,
		},
		{
			name:     "subject contains",
			rule:     BundleRule{Field: "subject", Operator: "contains", Value: "receipt"},
			from:     "",
			subject:  "Your receipt for order #123",
			domain:   "",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := matchRule(tt.rule, tt.from, tt.subject, tt.domain)
			if result != tt.expected {
				t.Errorf("matchRule() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestHandleGetBundles(t *testing.T) {
	server := &Server{}

	req := httptest.NewRequest(http.MethodGet, "/api/bundles", nil)
	w := httptest.NewRecorder()

	server.handleGetBundles(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var bundles []*Bundle
	if err := json.NewDecoder(w.Body).Decode(&bundles); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(bundles) == 0 {
		t.Error("expected at least one bundle")
	}
}

func TestHandleBundleCategorize(t *testing.T) {
	server := &Server{}

	body := map[string]string{
		"from":    "newsletter@example.com",
		"subject": "Weekly digest",
		"emailId": "test-email-123",
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/api/bundles/categorize", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.handleBundleCategorize(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var result map[string]string
	if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if result["bundleId"] != "newsletters" {
		t.Errorf("expected bundleId 'newsletters', got %q", result["bundleId"])
	}
}

func TestHandleBundleCategorizeInvalidBody(t *testing.T) {
	server := &Server{}

	req := httptest.NewRequest(http.MethodPost, "/api/bundles/categorize", bytes.NewReader([]byte("invalid")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.handleBundleCategorize(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}

func TestHandleUpdateBundle(t *testing.T) {
	server := &Server{}

	bundle := Bundle{
		ID:        "newsletters",
		Name:      "Updated Newsletters",
		Icon:      "📰",
		Collapsed: false,
	}
	bodyBytes, _ := json.Marshal(bundle)

	req := httptest.NewRequest(http.MethodPut, "/api/bundles", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.handleUpdateBundle(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestHandleGetBundleEmails(t *testing.T) {
	server := &Server{}

	req := httptest.NewRequest(http.MethodGet, "/api/bundles/emails?bundleId=newsletters", nil)
	w := httptest.NewRecorder()

	server.handleGetBundleEmails(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var emailIds []string
	if err := json.NewDecoder(w.Body).Decode(&emailIds); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
}

func TestHandleGetBundleEmailsMissingParam(t *testing.T) {
	server := &Server{}

	req := httptest.NewRequest(http.MethodGet, "/api/bundles/emails", nil)
	w := httptest.NewRecorder()

	server.handleGetBundleEmails(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}
