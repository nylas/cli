package dpop

import (
	"encoding/base64"
	"testing"

	"github.com/nylas/cli/internal/domain"
	"github.com/nylas/cli/internal/ports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew_MalformedStoredKeyIsReportedNotReplaced(t *testing.T) {
	t.Parallel()
	// The server binds sessions to the stored key. One that does not decode
	// may be a damaged read; replacing it would orphan every bound session,
	// so the user is told instead.
	tests := []struct {
		name   string
		stored string
	}{
		{name: "not base64", stored: "%%% not base64 %%%"},
		{name: "wrong length", stored: base64.StdEncoding.EncodeToString([]byte("too short"))},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			store := newMockSecretStore()
			store.data[ports.KeyDashboardDPoPKey] = tt.stored

			svc, err := New(store)

			require.ErrorIs(t, err, domain.ErrDashboardDPoPKeyInvalid)
			assert.Nil(t, svc)
			assert.Equal(t, tt.stored, store.data[ports.KeyDashboardDPoPKey])
		})
	}
}
