package ports

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIAMPolicy_HasMemberInRole(t *testing.T) {
	policy := &IAMPolicy{
		Bindings: []*IAMBinding{
			{Role: "roles/owner", Members: []string{"user:alice@example.com"}},
			{Role: "roles/editor", Members: []string{"user:bob@example.com"}},
		},
	}

	assert.True(t, policy.HasMemberInRole("roles/owner", "user:alice@example.com"))
	assert.False(t, policy.HasMemberInRole("roles/owner", "user:bob@example.com"))
	assert.True(t, policy.HasMemberInRole("roles/editor", "user:bob@example.com"))
	assert.False(t, policy.HasMemberInRole("roles/viewer", "user:alice@example.com"))
}

func TestIAMPolicy_AddBinding(t *testing.T) {
	t.Run("add to existing role", func(t *testing.T) {
		policy := &IAMPolicy{
			Bindings: []*IAMBinding{
				{Role: "roles/owner", Members: []string{"user:alice@example.com"}},
			},
		}

		policy.AddBinding("roles/owner", "user:bob@example.com")
		assert.True(t, policy.HasMemberInRole("roles/owner", "user:bob@example.com"))
		assert.Len(t, policy.Bindings, 1) // Same binding, just extended
	})

	t.Run("add new role", func(t *testing.T) {
		policy := &IAMPolicy{}

		policy.AddBinding("roles/editor", "user:alice@example.com")
		assert.True(t, policy.HasMemberInRole("roles/editor", "user:alice@example.com"))
		assert.Len(t, policy.Bindings, 1)
	})

	t.Run("idempotent add", func(t *testing.T) {
		policy := &IAMPolicy{
			Bindings: []*IAMBinding{
				{Role: "roles/owner", Members: []string{"user:alice@example.com"}},
			},
		}

		policy.AddBinding("roles/owner", "user:alice@example.com")
		assert.Len(t, policy.Bindings[0].Members, 1) // No duplicate
	})
}
