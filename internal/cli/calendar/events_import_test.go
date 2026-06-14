package calendar

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEventsImportCommand_Structure(t *testing.T) {
	cmd := newEventsImportCmd()

	assert.Equal(t, "import [grant-id]", cmd.Use)
	for _, flag := range []string{"calendar", "start", "end", "limit"} {
		assert.NotNil(t, cmd.Flags().Lookup(flag), "missing --%s flag", flag)
	}
}

func TestEventsCmd_RegistersImport(t *testing.T) {
	cmd := newEventsCmd()
	names := make(map[string]bool)
	for _, sub := range cmd.Commands() {
		names[sub.Name()] = true
	}
	assert.True(t, names["import"], "events command must register the import subcommand")
}

func TestParseImportTime(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    int64
		wantErr bool
	}{
		{name: "empty defers to API default", input: "", want: 0},
		{name: "unix timestamp passes through", input: "1735689600", want: 1735689600},
		{name: "date only", input: "2025-01-01", want: time.Date(2025, 1, 1, 0, 0, 0, 0, time.Local).Unix()},
		{name: "invalid", input: "not-a-date", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseImportTime("start", tt.input)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}
