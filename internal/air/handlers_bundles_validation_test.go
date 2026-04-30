package air

import (
	"testing"
)

// TestValidateBundleRule_RejectsBadRegex pins the new contract: a bundle
// PUT with an unparseable "matches" regex is rejected at the boundary
// instead of every email match silently failing forever afterwards.
func TestValidateBundleRule_RejectsBadRegex(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		rule    BundleRule
		wantErr bool
	}{
		{
			"valid contains",
			BundleRule{Field: "from", Operator: "contains", Value: "@example.com"},
			false,
		},
		{
			"valid matches",
			BundleRule{Field: "subject", Operator: "matches", Value: `^\[urgent\]`},
			false,
		},
		{
			"empty matches value (skipped)",
			BundleRule{Field: "subject", Operator: "matches", Value: ""},
			false,
		},
		{
			"unbalanced parens",
			BundleRule{Field: "subject", Operator: "matches", Value: "(unclosed"},
			true,
		},
		{
			"invalid char class",
			BundleRule{Field: "subject", Operator: "matches", Value: "[unclosed"},
			true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := validateBundleRule(tc.rule)
			if tc.wantErr && err == nil {
				t.Error("expected error, got nil")
			}
			if !tc.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestMatchRule_RegexCacheReused(t *testing.T) {
	rule := BundleRule{Field: "subject", Operator: "matches", Value: "^urgent"}

	if !matchRule(rule, "from", "urgent: please reply", "domain") {
		t.Error("expected match")
	}
	if matchRule(rule, "from", "no rush", "domain") {
		t.Error("expected no match")
	}

	// Now compile must come from cache.
	bundleRegexCacheMu.RLock()
	_, ok := bundleRegexCache[rule.Value]
	bundleRegexCacheMu.RUnlock()
	if !ok {
		t.Error("expected pattern to be cached after first match")
	}
}

func TestMatchRule_BadRegexReturnsFalse(t *testing.T) {
	rule := BundleRule{Field: "subject", Operator: "matches", Value: "(unclosed"}
	if matchRule(rule, "from", "anything", "domain") {
		t.Error("matchRule should return false for uncompilable regex")
	}
}
