package nylas

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/nylas/cli/internal/domain"
)

// Scheduler Configurations

// ListSchedulerConfigurations retrieves all scheduler configurations for a grant.
func (c *HTTPClient) ListSchedulerConfigurations(ctx context.Context, grantID string) ([]domain.SchedulerConfiguration, error) {
	if err := validateRequired("grant ID", grantID); err != nil {
		return nil, err
	}

	// The configurations endpoint is cursor-paginated (limit + page_token /
	// next_cursor). Follow the cursor so accounts with more than the default page
	// of configurations aren't silently truncated.
	baseURL := fmt.Sprintf("%s/v3/grants/%s/scheduling/configurations", c.baseURL, url.PathEscape(grantID))
	const maxConfigPages = 1000

	// Non-nil so an empty result marshals to `[]`, not `null`.
	all := make([]domain.SchedulerConfiguration, 0)
	pageToken := ""
	for range maxConfigPages {
		queryURL := NewQueryBuilder().AddInt("limit", 200).Add("page_token", pageToken).BuildURL(baseURL)

		var result struct {
			Data       []domain.SchedulerConfiguration `json:"data"`
			NextCursor string                          `json:"next_cursor,omitempty"`
		}
		if err := c.doGet(ctx, queryURL, &result); err != nil {
			return nil, err
		}
		all = append(all, result.Data...)

		if result.NextCursor == "" {
			return all, nil
		}
		pageToken = result.NextCursor
	}
	return nil, fmt.Errorf("failed to paginate scheduler configurations: exceeded max page count (%d)", maxConfigPages)
}

// GetSchedulerConfiguration retrieves a specific scheduler configuration for a grant.
func (c *HTTPClient) GetSchedulerConfiguration(ctx context.Context, grantID, configID string) (*domain.SchedulerConfiguration, error) {
	if err := validateRequired("grant ID", grantID); err != nil {
		return nil, err
	}
	if err := validateRequired("configuration ID", configID); err != nil {
		return nil, err
	}

	queryURL := fmt.Sprintf("%s/v3/grants/%s/scheduling/configurations/%s", c.baseURL, url.PathEscape(grantID), url.PathEscape(configID))

	var result struct {
		Data domain.SchedulerConfiguration `json:"data"`
	}
	if err := c.doGetWithNotFound(ctx, queryURL, &result, domain.ErrConfigurationNotFound); err != nil {
		return nil, err
	}
	return &result.Data, nil
}

// CreateSchedulerConfiguration creates a new scheduler configuration for a grant.
func (c *HTTPClient) CreateSchedulerConfiguration(ctx context.Context, grantID string, req *domain.CreateSchedulerConfigurationRequest) (*domain.SchedulerConfiguration, error) {
	if err := validateRequired("grant ID", grantID); err != nil {
		return nil, err
	}

	queryURL := fmt.Sprintf("%s/v3/grants/%s/scheduling/configurations", c.baseURL, url.PathEscape(grantID))

	resp, err := c.doJSONRequest(ctx, "POST", queryURL, req)
	if err != nil {
		return nil, err
	}

	var result struct {
		Data domain.SchedulerConfiguration `json:"data"`
	}
	if err := c.decodeJSONResponse(resp, &result); err != nil {
		return nil, err
	}
	return &result.Data, nil
}

// UpdateSchedulerConfiguration updates an existing scheduler configuration for a grant.
func (c *HTTPClient) UpdateSchedulerConfiguration(ctx context.Context, grantID, configID string, req *domain.UpdateSchedulerConfigurationRequest) (*domain.SchedulerConfiguration, error) {
	if err := validateRequired("grant ID", grantID); err != nil {
		return nil, err
	}
	if err := validateRequired("configuration ID", configID); err != nil {
		return nil, err
	}

	queryURL := fmt.Sprintf("%s/v3/grants/%s/scheduling/configurations/%s", c.baseURL, url.PathEscape(grantID), url.PathEscape(configID))

	resp, err := c.doJSONRequest(ctx, "PUT", queryURL, req)
	if err != nil {
		return nil, err
	}

	var result struct {
		Data domain.SchedulerConfiguration `json:"data"`
	}
	if err := c.decodeJSONResponse(resp, &result); err != nil {
		return nil, err
	}
	return &result.Data, nil
}

// DeleteSchedulerConfiguration deletes a scheduler configuration for a grant.
func (c *HTTPClient) DeleteSchedulerConfiguration(ctx context.Context, grantID, configID string) error {
	if err := validateRequired("grant ID", grantID); err != nil {
		return err
	}
	if err := validateRequired("configuration ID", configID); err != nil {
		return err
	}
	queryURL := fmt.Sprintf("%s/v3/grants/%s/scheduling/configurations/%s", c.baseURL, url.PathEscape(grantID), url.PathEscape(configID))
	return c.doDelete(ctx, queryURL)
}

// Scheduler Sessions

// CreateSchedulerSession creates a new scheduler session.
func (c *HTTPClient) CreateSchedulerSession(ctx context.Context, req *domain.CreateSchedulerSessionRequest) (*domain.SchedulerSession, error) {
	// The v3 spec requires configuration_id or slug, and caps time_to_live.
	if err := req.Validate(); err != nil {
		return nil, err
	}
	queryURL := fmt.Sprintf("%s/v3/scheduling/sessions", c.baseURL)

	resp, err := c.doJSONRequest(ctx, "POST", queryURL, req)
	if err != nil {
		return nil, err
	}

	var result struct {
		Data domain.SchedulerSession `json:"data"`
	}
	if err := c.decodeJSONResponse(resp, &result); err != nil {
		return nil, err
	}
	return &result.Data, nil
}

// GetSchedulerSession retrieves a scheduler session.
func (c *HTTPClient) GetSchedulerSession(ctx context.Context, sessionID string) (*domain.SchedulerSession, error) {
	if err := validateRequired("session ID", sessionID); err != nil {
		return nil, err
	}

	queryURL := fmt.Sprintf("%s/v3/scheduling/sessions/%s", c.baseURL, url.PathEscape(sessionID))

	var result struct {
		Data domain.SchedulerSession `json:"data"`
	}
	if err := c.doGetWithNotFound(ctx, queryURL, &result, domain.ErrSessionNotFound); err != nil {
		return nil, err
	}
	return &result.Data, nil
}

// Bookings

// bookingSessionToken mints a short-lived Scheduler session for a configuration
// and returns its token. Booking endpoints (/v3/scheduling/bookings/{id}) are
// authorized by a SCHEDULER_SESSION_TOKEN, not the application API key, so every
// booking operation exchanges a configuration ID for a session token first.
//
// ponytail: one fresh session per booking call, no caching — these are one-shot
// CLI/RPC operations. Cache per configuration if a batch caller ever needs it.
func (c *HTTPClient) bookingSessionToken(ctx context.Context, configurationID string) (string, error) {
	if err := validateRequired("configuration ID", configurationID); err != nil {
		return "", err
	}
	session, err := c.CreateSchedulerSession(ctx, &domain.CreateSchedulerSessionRequest{
		ConfigurationID: configurationID,
		TimeToLive:      5, // minutes; enough for a single booking call
	})
	if err != nil {
		return "", fmt.Errorf("create scheduler session: %w", err)
	}
	if session.SessionID == "" {
		return "", fmt.Errorf("scheduler session response missing session_id")
	}
	return session.SessionID, nil
}

// getBookingWithToken reads a booking using an already-minted session token.
// Shared by GetBooking (mints then reads) and RescheduleBooking (reuses its
// token to read the updated booking back, since PATCH returns no booking body).
func (c *HTTPClient) getBookingWithToken(ctx context.Context, token, bookingID string) (*domain.Booking, error) {
	queryURL := fmt.Sprintf("%s/v3/scheduling/bookings/%s", c.baseURL, url.PathEscape(bookingID))

	var result struct {
		Data domain.Booking `json:"data"`
	}
	if err := c.doGetWithNotFoundAuth(ctx, queryURL, token, &result, domain.ErrBookingNotFound); err != nil {
		return nil, err
	}
	return &result.Data, nil
}

// GetBooking retrieves a specific booking.
func (c *HTTPClient) GetBooking(ctx context.Context, configurationID, bookingID string) (*domain.Booking, error) {
	if err := validateRequired("booking ID", bookingID); err != nil {
		return nil, err
	}
	token, err := c.bookingSessionToken(ctx, configurationID)
	if err != nil {
		return nil, err
	}
	return c.getBookingWithToken(ctx, token, bookingID)
}

// ConfirmBooking confirms a booking.
func (c *HTTPClient) ConfirmBooking(ctx context.Context, configurationID, bookingID string, req *domain.ConfirmBookingRequest) (*domain.Booking, error) {
	if err := validateRequired("booking ID", bookingID); err != nil {
		return nil, err
	}
	if req == nil {
		return nil, fmt.Errorf("confirm booking request is required")
	}
	// The Nylas v3 spec requires salt and a confirmed/cancelled status.
	if err := req.Validate(); err != nil {
		return nil, err
	}
	token, err := c.bookingSessionToken(ctx, configurationID)
	if err != nil {
		return nil, err
	}

	queryURL := fmt.Sprintf("%s/v3/scheduling/bookings/%s", c.baseURL, url.PathEscape(bookingID))

	resp, err := c.doJSONRequestWithToken(ctx, "PUT", queryURL, token, req)
	if err != nil {
		return nil, err
	}

	// Declining a pending booking (status "cancelled") returns the no-data
	// delete envelope rather than a booking body, so don't try to decode one.
	if req.Status == "cancelled" {
		_ = resp.Body.Close()
		return &domain.Booking{BookingID: bookingID, Status: "cancelled"}, nil
	}

	var result struct {
		Data domain.Booking `json:"data"`
	}
	if err := c.decodeJSONResponse(resp, &result); err != nil {
		return nil, err
	}
	return &result.Data, nil
}

// RescheduleBooking reschedules a booking.
func (c *HTTPClient) RescheduleBooking(ctx context.Context, configurationID, bookingID string, req *domain.RescheduleBookingRequest) (*domain.Booking, error) {
	if err := validateRequired("booking ID", bookingID); err != nil {
		return nil, err
	}
	if req == nil {
		return nil, fmt.Errorf("reschedule booking request is required")
	}
	token, err := c.bookingSessionToken(ctx, configurationID)
	if err != nil {
		return nil, err
	}

	// Nylas v3 reschedules via PATCH on the bare booking resource (no
	// /reschedule suffix); the new start/end times are in the request body.
	queryURL := fmt.Sprintf("%s/v3/scheduling/bookings/%s", c.baseURL, url.PathEscape(bookingID))

	// The PATCH response carries only a request_id (no booking body). Read the
	// updated booking back for the full record; only a 404 (read racing ahead of
	// propagation) may fall back to the requested times — any other failure is
	// surfaced so a possibly-diverged server state is never reported as success.
	resp, err := c.doJSONRequestWithToken(ctx, "PATCH", queryURL, token, req)
	if err != nil {
		return nil, err
	}
	_ = resp.Body.Close()

	start, end := time.Unix(req.StartTime, 0), time.Unix(req.EndTime, 0)
	booking, err := c.getBookingWithToken(ctx, token, bookingID)
	if err != nil {
		// Single construction site for the partial-success record: callers on
		// the ErrBookingReadBackFailed path reuse this booking as-is.
		fallback := &domain.Booking{BookingID: bookingID, StartTime: start, EndTime: end}
		if errors.Is(err, domain.ErrBookingNotFound) {
			return fallback, nil
		}
		return fallback, fmt.Errorf("%w: %w", domain.ErrBookingReadBackFailed, err)
	}
	// The Nylas booking data model carries no start/end times (they exist only
	// on the update request), so reflect the just-applied times onto the
	// read-back record instead of leaving them zero.
	booking.StartTime, booking.EndTime = start, end
	return booking, nil
}

// CancelBooking cancels a booking.
func (c *HTTPClient) CancelBooking(ctx context.Context, configurationID, bookingID string, reason string) error {
	if err := validateRequired("booking ID", bookingID); err != nil {
		return err
	}
	token, err := c.bookingSessionToken(ctx, configurationID)
	if err != nil {
		return err
	}

	queryURL := fmt.Sprintf("%s/v3/scheduling/bookings/%s", c.baseURL, url.PathEscape(bookingID))
	body := struct {
		CancellationReason string `json:"cancellation_reason,omitempty"`
	}{CancellationReason: reason}

	resp, err := c.doJSONRequestWithToken(ctx, "DELETE", queryURL, token, body, http.StatusOK, http.StatusNoContent)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	return nil
}
