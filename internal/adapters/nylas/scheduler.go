package nylas

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/nylas/cli/internal/domain"
)

// Scheduler Configurations

// ListSchedulerConfigurations retrieves all scheduler configurations.
func (c *HTTPClient) ListSchedulerConfigurations(ctx context.Context) ([]domain.SchedulerConfiguration, error) {
	queryURL := fmt.Sprintf("%s/v3/scheduling/configurations", c.baseURL)

	resp, err := c.doJSONRequest(ctx, "GET", queryURL, nil)
	if err != nil {
		return nil, err
	}

	var result struct {
		Data []domain.SchedulerConfiguration `json:"data"`
	}
	if err := c.decodeJSONResponse(resp, &result); err != nil {
		return nil, err
	}
	return result.Data, nil
}

// GetSchedulerConfiguration retrieves a specific scheduler configuration.
func (c *HTTPClient) GetSchedulerConfiguration(ctx context.Context, configID string) (*domain.SchedulerConfiguration, error) {
	if err := validateRequired("configuration ID", configID); err != nil {
		return nil, err
	}

	queryURL := fmt.Sprintf("%s/v3/scheduling/configurations/%s", c.baseURL, configID)

	var result struct {
		Data domain.SchedulerConfiguration `json:"data"`
	}
	if err := c.doGetWithNotFound(ctx, queryURL, &result, fmt.Errorf("%w: configuration not found", domain.ErrAPIError)); err != nil {
		return nil, err
	}
	return &result.Data, nil
}

// CreateSchedulerConfiguration creates a new scheduler configuration.
func (c *HTTPClient) CreateSchedulerConfiguration(ctx context.Context, req *domain.CreateSchedulerConfigurationRequest) (*domain.SchedulerConfiguration, error) {
	queryURL := fmt.Sprintf("%s/v3/scheduling/configurations", c.baseURL)

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

// UpdateSchedulerConfiguration updates an existing scheduler configuration.
func (c *HTTPClient) UpdateSchedulerConfiguration(ctx context.Context, configID string, req *domain.UpdateSchedulerConfigurationRequest) (*domain.SchedulerConfiguration, error) {
	if err := validateRequired("configuration ID", configID); err != nil {
		return nil, err
	}

	queryURL := fmt.Sprintf("%s/v3/scheduling/configurations/%s", c.baseURL, configID)

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

// DeleteSchedulerConfiguration deletes a scheduler configuration.
func (c *HTTPClient) DeleteSchedulerConfiguration(ctx context.Context, configID string) error {
	if err := validateRequired("configuration ID", configID); err != nil {
		return err
	}
	queryURL := fmt.Sprintf("%s/v3/scheduling/configurations/%s", c.baseURL, configID)
	return c.doDelete(ctx, queryURL)
}

// Scheduler Sessions

// CreateSchedulerSession creates a new scheduler session.
func (c *HTTPClient) CreateSchedulerSession(ctx context.Context, req *domain.CreateSchedulerSessionRequest) (*domain.SchedulerSession, error) {
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

	queryURL := fmt.Sprintf("%s/v3/scheduling/sessions/%s", c.baseURL, sessionID)

	var result struct {
		Data domain.SchedulerSession `json:"data"`
	}
	if err := c.doGetWithNotFound(ctx, queryURL, &result, fmt.Errorf("%w: session not found", domain.ErrAPIError)); err != nil {
		return nil, err
	}
	return &result.Data, nil
}

// Bookings

// ListBookings retrieves all bookings for a configuration.
func (c *HTTPClient) ListBookings(ctx context.Context, configID string) ([]domain.Booking, error) {
	baseURL := fmt.Sprintf("%s/v3/scheduling/bookings", c.baseURL)
	queryURL := NewQueryBuilder().Add("configuration_id", configID).BuildURL(baseURL)

	resp, err := c.doJSONRequest(ctx, "GET", queryURL, nil)
	if err != nil {
		return nil, err
	}

	var result struct {
		Data []domain.Booking `json:"data"`
	}
	if err := c.decodeJSONResponse(resp, &result); err != nil {
		return nil, err
	}
	return result.Data, nil
}

// GetBooking retrieves a specific booking.
func (c *HTTPClient) GetBooking(ctx context.Context, bookingID string) (*domain.Booking, error) {
	if err := validateRequired("booking ID", bookingID); err != nil {
		return nil, err
	}

	queryURL := fmt.Sprintf("%s/v3/scheduling/bookings/%s", c.baseURL, bookingID)

	var result struct {
		Data domain.Booking `json:"data"`
	}
	if err := c.doGetWithNotFound(ctx, queryURL, &result, fmt.Errorf("%w: booking not found", domain.ErrAPIError)); err != nil {
		return nil, err
	}
	return &result.Data, nil
}

// ConfirmBooking confirms a booking.
func (c *HTTPClient) ConfirmBooking(ctx context.Context, bookingID string, req *domain.ConfirmBookingRequest) (*domain.Booking, error) {
	if err := validateRequired("booking ID", bookingID); err != nil {
		return nil, err
	}

	queryURL := fmt.Sprintf("%s/v3/scheduling/bookings/%s", c.baseURL, bookingID)

	resp, err := c.doJSONRequest(ctx, "PUT", queryURL, req)
	if err != nil {
		return nil, err
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
func (c *HTTPClient) RescheduleBooking(ctx context.Context, bookingID string, req *domain.RescheduleBookingRequest) (*domain.Booking, error) {
	if err := validateRequired("booking ID", bookingID); err != nil {
		return nil, err
	}

	queryURL := fmt.Sprintf("%s/v3/scheduling/bookings/%s/reschedule", c.baseURL, url.PathEscape(bookingID))

	resp, err := c.doJSONRequest(ctx, "PATCH", queryURL, req)
	if err != nil {
		return nil, err
	}

	var result struct {
		Data domain.Booking `json:"data"`
	}
	if err := c.decodeJSONResponse(resp, &result); err != nil {
		return nil, err
	}
	return &result.Data, nil
}

// CancelBooking cancels a booking.
func (c *HTTPClient) CancelBooking(ctx context.Context, bookingID string, reason string) error {
	if err := validateRequired("booking ID", bookingID); err != nil {
		return err
	}

	baseURL := fmt.Sprintf("%s/v3/scheduling/bookings/%s", c.baseURL, url.PathEscape(bookingID))
	queryURL := NewQueryBuilder().Add("reason", reason).BuildURL(baseURL)

	resp, err := c.doJSONRequest(ctx, "DELETE", queryURL, nil, http.StatusOK, http.StatusNoContent)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	return nil
}

// Scheduler Pages

// ListSchedulerPages retrieves all scheduler pages.
func (c *HTTPClient) ListSchedulerPages(ctx context.Context) ([]domain.SchedulerPage, error) {
	queryURL := fmt.Sprintf("%s/v3/scheduling/pages", c.baseURL)

	resp, err := c.doJSONRequest(ctx, "GET", queryURL, nil)
	if err != nil {
		return nil, err
	}

	var result struct {
		Data []domain.SchedulerPage `json:"data"`
	}
	if err := c.decodeJSONResponse(resp, &result); err != nil {
		return nil, err
	}
	return result.Data, nil
}

// GetSchedulerPage retrieves a specific scheduler page.
func (c *HTTPClient) GetSchedulerPage(ctx context.Context, pageID string) (*domain.SchedulerPage, error) {
	if err := validateRequired("page ID", pageID); err != nil {
		return nil, err
	}

	queryURL := fmt.Sprintf("%s/v3/scheduling/pages/%s", c.baseURL, pageID)

	var result struct {
		Data domain.SchedulerPage `json:"data"`
	}
	if err := c.doGetWithNotFound(ctx, queryURL, &result, fmt.Errorf("%w: page not found", domain.ErrAPIError)); err != nil {
		return nil, err
	}
	return &result.Data, nil
}

// CreateSchedulerPage creates a new scheduler page.
func (c *HTTPClient) CreateSchedulerPage(ctx context.Context, req *domain.CreateSchedulerPageRequest) (*domain.SchedulerPage, error) {
	queryURL := fmt.Sprintf("%s/v3/scheduling/pages", c.baseURL)

	resp, err := c.doJSONRequest(ctx, "POST", queryURL, req)
	if err != nil {
		return nil, err
	}

	var result struct {
		Data domain.SchedulerPage `json:"data"`
	}
	if err := c.decodeJSONResponse(resp, &result); err != nil {
		return nil, err
	}
	return &result.Data, nil
}

// UpdateSchedulerPage updates an existing scheduler page.
func (c *HTTPClient) UpdateSchedulerPage(ctx context.Context, pageID string, req *domain.UpdateSchedulerPageRequest) (*domain.SchedulerPage, error) {
	if err := validateRequired("page ID", pageID); err != nil {
		return nil, err
	}

	queryURL := fmt.Sprintf("%s/v3/scheduling/pages/%s", c.baseURL, pageID)

	resp, err := c.doJSONRequest(ctx, "PUT", queryURL, req)
	if err != nil {
		return nil, err
	}

	var result struct {
		Data domain.SchedulerPage `json:"data"`
	}
	if err := c.decodeJSONResponse(resp, &result); err != nil {
		return nil, err
	}
	return &result.Data, nil
}

// DeleteSchedulerPage deletes a scheduler page.
func (c *HTTPClient) DeleteSchedulerPage(ctx context.Context, pageID string) error {
	if err := validateRequired("page ID", pageID); err != nil {
		return err
	}
	queryURL := fmt.Sprintf("%s/v3/scheduling/pages/%s", c.baseURL, pageID)
	return c.doDelete(ctx, queryURL)
}
