//go:build !integration

package oauth

import (
	"testing"

	"github.com/nylas/cli/internal/app/oauthlogin"
	"github.com/nylas/cli/internal/cli/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoginCmd_DeviceSelection(t *testing.T) {
	for _, tc := range []struct {
		name  string
		flags []string
		ssh   string
		want  bool
	}{
		{"device", []string{"--device"}, "", true},
		{"no-browser", []string{"--no-browser"}, "", true},
		{"SSH", nil, "remote", true},
		{"local-no-DISPLAY", nil, "", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("SSH_CONNECTION", tc.ssh)
			t.Setenv("SSH_CLIENT", "")
			t.Setenv("SSH_TTY", "")
			t.Setenv("DISPLAY", "")
			svc := &fakeService{loginResult: &oauthlogin.LoginResult{Issuer: "https://auth.example.test", ClientID: "client"}}
			withService(t, svc)
			out, _, err := testutil.ExecuteSubCommand(newLoginCmd(), tc.flags...)
			require.NoError(t, err)
			require.NotNil(t, svc.loginOpts)
			assert.Equal(t, tc.want, svc.loginOpts.Device)
			if tc.want {
				assert.Contains(t, out, "Using device sign-in")
			} else {
				assert.Contains(t, out, "Opening your browser")
			}
		})
	}
}
