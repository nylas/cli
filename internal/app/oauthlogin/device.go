package oauthlogin

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/nylas/cli/internal/domain"
	"github.com/nylas/cli/internal/ports"
)

func waitDevicePoll(ctx context.Context, interval time.Duration) error {
	timer := time.NewTimer(interval)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func (s *Service) loginDevice(ctx context.Context, metadata *domain.OAuthServerMetadata, opts LoginOptions, scopes, dropped []string) (*LoginResult, error) {
	// Device requests carry a regional resource rather than a browser-only
	// signup hint, so an explicitly configured signup region stays fixed.
	if opts.Resource == "" && opts.Region != "" {
		resource, err := domain.MCPResourceForRegion(opts.Region)
		if err != nil {
			return nil, err
		}
		opts.Resource = resource
	}
	if s.clientID != domain.DefaultOAuthClientID {
		return nil, errors.New("device sign-in requires the built-in CLI client")
	}
	client, ok := s.client.(ports.OAuthDeviceAuthServerClient)
	if !ok || metadata.DeviceAuthorizationEndpoint == "" {
		return nil, errors.New("this server does not offer device sign-in")
	}
	if opts.OnDeviceAuthorization == nil {
		return nil, errors.New("device sign-in output is not configured")
	}
	// Start the deadline before the request: response latency never extends
	// the server's advertised device-code lifetime.
	started := s.now()
	authorization, err := client.StartDeviceAuthorization(ctx, s.clientID, scopes, opts.Resource)
	if err != nil {
		return nil, err
	}
	if authorization == nil {
		return nil, domain.ErrOAuthMetadata
	}
	if err := authorization.ValidateFor(metadata.Issuer); err != nil {
		return nil, err
	}
	deadline := started.Add(time.Duration(authorization.ExpiresIn) * time.Second)
	pollCtx, cancel := context.WithTimeout(ctx, deadline.Sub(s.now()))
	defer cancel()
	if err := opts.OnDeviceAuthorization(domain.OAuthDevicePrompt{
		UserCode: authorization.UserCode, VerificationURI: authorization.VerificationURI,
		VerificationURIComplete: authorization.VerificationURIComplete,
	}); err != nil {
		return nil, err
	}
	interval := time.Duration(authorization.Interval) * time.Second
	if interval == 0 {
		interval = 5 * time.Second
	}
	wait := s.waitDevice
	if wait == nil {
		wait = waitDevicePoll
	}
	for {
		if err := wait(pollCtx, interval); err != nil {
			if ctx.Err() != nil {
				return nil, ctx.Err()
			}
			return nil, errors.New("device sign-in expired; run `nylas oauth login --device` again")
		}
		if !s.now().Before(deadline) {
			return nil, errors.New("device sign-in expired; run `nylas oauth login --device` again")
		}
		tokens, err := client.PollDeviceTokens(pollCtx, s.clientID, authorization.DeviceCode, opts.Resource)
		if err != nil {
			if ctx.Err() != nil {
				return nil, ctx.Err()
			}
			if pollCtx.Err() != nil {
				return nil, errors.New("device sign-in expired; run `nylas oauth login --device` again")
			}
			var oauthErr *domain.OAuthError
			if errors.As(err, &oauthErr) {
				switch oauthErr.Code {
				case "authorization_pending":
					continue
				case "slow_down":
					interval += 5 * time.Second
					continue
				case "expired_token":
					return nil, errors.New("device sign-in expired; run `nylas oauth login --device` again")
				case "access_denied":
					return nil, errors.New("device sign-in was denied")
				}
			}
			if errors.Is(err, domain.ErrNetworkError) {
				interval = max(interval, min(interval*2, 60*time.Second))
				continue
			}
			return nil, errors.New("device sign-in failed; start a new sign-in when ready")
		}
		if tokens == nil || tokens.AccessToken == "" {
			return nil, errors.New("device sign-in returned no access token")
		}
		storeCtx, cancelStore := context.WithTimeout(context.WithoutCancel(ctx), refreshTimeout)
		err = s.withSessionLock(storeCtx, func() error {
			s.revokeReplacedSession(storeCtx)
			if err := s.clearSession(); err != nil {
				return err
			}
			if err := s.saveTokens(metadata.Issuer, opts.Resource, tokens); err != nil {
				return errors.Join(err, s.clearSession())
			}
			return nil
		})
		cancelStore()
		if err != nil {
			return nil, fmt.Errorf("could not store device sign-in: %w", err)
		}
		return &LoginResult{Issuer: metadata.Issuer, ClientID: s.clientID, Scope: tokens.Scope,
			ExpiresAt: tokens.ExpiresAt, HasRefresh: tokens.RefreshToken != "", Resource: opts.Resource,
			DroppedScopes: dropped}, nil
	}
}
