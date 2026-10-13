package domain

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDeviceAuthorization_RejectsUnsafeValues(t *testing.T) {
	valid := OAuthDeviceAuthorization{DeviceCode: strings.Repeat("a", 43), UserCode: "BCDF-GHJK", VerificationURI: "https://auth.example.test/device", ExpiresIn: 600, Interval: 5}
	require.NoError(t, valid.ValidateFor("https://auth.example.test"))
	for _, change := range []func(*OAuthDeviceAuthorization){
		func(d *OAuthDeviceAuthorization) { d.DeviceCode = strings.Repeat("a", 2049) },
		func(d *OAuthDeviceAuthorization) { d.DeviceCode = "invalid\n" },
		func(d *OAuthDeviceAuthorization) { d.UserCode = "code\x1b[31m" },
		func(d *OAuthDeviceAuthorization) { d.VerificationURI = "https://evil.example.test/device" },
		func(d *OAuthDeviceAuthorization) {
			d.VerificationURIComplete = "https://user:secret@auth.example.test/device"
		},
		func(d *OAuthDeviceAuthorization) { d.ExpiresIn = 601 },
		func(d *OAuthDeviceAuthorization) { d.Interval = -1 },
	} {
		d := valid
		change(&d)
		require.ErrorIs(t, d.ValidateFor("https://auth.example.test"), ErrOAuthMetadata)
	}
}
