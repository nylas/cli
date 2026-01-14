//go:build integration
// +build integration

package nylas_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIntegration_ListGrants(t *testing.T) {
	client, _ := getTestClient(t)
	ctx, cancel := createTestContext()
	defer cancel()

	grants, err := client.ListGrants(ctx)
	require.NoError(t, err, "ListGrants should not return error")
	require.NotEmpty(t, grants, "Should have at least one grant")

	t.Logf("Found %d grants", len(grants))
	for _, g := range grants {
		t.Logf("  Grant ID: %s, Email: %s, Provider: %s, Status: %s",
			g.ID, g.Email, g.Provider, g.GrantStatus)

		// Validate grant fields
		assert.NotEmpty(t, g.ID, "Grant should have ID")
		assert.NotEmpty(t, g.Email, "Grant should have email")
		assert.NotEmpty(t, g.Provider, "Grant should have provider")
	}
}

func TestIntegration_ListGrants_ValidatesProvider(t *testing.T) {
	client, _ := getTestClient(t)
	ctx, cancel := createTestContext()
	defer cancel()

	grants, err := client.ListGrants(ctx)
	require.NoError(t, err)
	require.NotEmpty(t, grants)

	validProviders := map[string]bool{
		"google":           true,
		"microsoft":        true,
		"imap":             true,
		"yahoo":            true,
		"icloud":           true,
		"ews":              true,
		"inbox":            true, // Nylas Native Auth
		"virtual":          true,
		"virtual-calendar": true, // Virtual calendar provider
	}

	for _, g := range grants {
		_, isValid := validProviders[strings.ToLower(string(g.Provider))]
		assert.True(t, isValid, "Provider %s should be a valid Nylas provider", g.Provider)
	}
}

// =============================================================================
// Message Tests - Basic Operations
// =============================================================================
