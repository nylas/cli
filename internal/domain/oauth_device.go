package domain

import (
	"fmt"
	"net/url"
	"regexp"
)

const OAuthDeviceGrant = "urn:ietf:params:oauth:grant-type:device_code"

// OAuthDeviceAuthorization is held in memory only. DeviceCode is a bearer
// credential and must never reach terminal output, diagnostic logs or storage.
type OAuthDeviceAuthorization struct {
	DeviceCode              string `json:"device_code"`
	UserCode                string `json:"user_code"`
	VerificationURI         string `json:"verification_uri"`
	VerificationURIComplete string `json:"verification_uri_complete"`
	ExpiresIn               int    `json:"expires_in"`
	Interval                int    `json:"interval"`
}

// OAuthDevicePrompt contains only the values intended for the user's terminal.
type OAuthDevicePrompt struct {
	UserCode                string
	VerificationURI         string
	VerificationURIComplete string
}

var deviceCodePattern = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)

var deviceUserCode = regexp.MustCompile(`^[A-Z0-9]{4}-?[A-Z0-9]{4}$`)

// ValidateFor rejects malformed values and links outside the discovered issuer.
func (d *OAuthDeviceAuthorization) ValidateFor(issuer string) error {
	if len(d.DeviceCode) < 16 || len(d.DeviceCode) > 2048 || !deviceCodePattern.MatchString(d.DeviceCode) || !deviceUserCode.MatchString(d.UserCode) || d.ExpiresIn < 1 || d.ExpiresIn > 600 || d.Interval < 0 || d.Interval > 600 {
		return fmt.Errorf("%w: invalid device authorization", ErrOAuthMetadata)
	}
	base, err := url.Parse(issuer)
	if err != nil || base.Host == "" {
		return ErrOAuthMetadata
	}
	for _, raw := range []string{d.VerificationURI, d.VerificationURIComplete} {
		if raw == "" {
			continue
		}
		u, err := url.Parse(raw)
		if err != nil || len(raw) > 4096 || u.Scheme != base.Scheme || u.Host != base.Host || u.User != nil || u.Fragment != "" {
			return fmt.Errorf("%w: invalid device verification link", ErrOAuthMetadata)
		}
	}
	if d.VerificationURI == "" {
		return fmt.Errorf("%w: missing device verification link", ErrOAuthMetadata)
	}
	return nil
}
