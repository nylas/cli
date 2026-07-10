package ports

import (
	"context"

	"github.com/nylas/cli/internal/domain"
)

// SchedulerClient defines the interface for scheduler operations.
type SchedulerClient interface {
	// ================================
	// CONFIGURATION OPERATIONS
	// ================================

	// Configuration operations are grant-scoped
	// (/v3/grants/{grant_id}/scheduling/configurations).

	// ListSchedulerConfigurations retrieves all scheduler configurations for a grant.
	ListSchedulerConfigurations(ctx context.Context, grantID string) ([]domain.SchedulerConfiguration, error)

	// GetSchedulerConfiguration retrieves a specific scheduler configuration for a grant.
	GetSchedulerConfiguration(ctx context.Context, grantID, configID string) (*domain.SchedulerConfiguration, error)

	// CreateSchedulerConfiguration creates a new scheduler configuration for a grant.
	CreateSchedulerConfiguration(ctx context.Context, grantID string, req *domain.CreateSchedulerConfigurationRequest) (*domain.SchedulerConfiguration, error)

	// UpdateSchedulerConfiguration updates an existing scheduler configuration for a grant.
	UpdateSchedulerConfiguration(ctx context.Context, grantID, configID string, req *domain.UpdateSchedulerConfigurationRequest) (*domain.SchedulerConfiguration, error)

	// DeleteSchedulerConfiguration deletes a scheduler configuration for a grant.
	DeleteSchedulerConfiguration(ctx context.Context, grantID, configID string) error

	// ================================
	// SESSION OPERATIONS
	// ================================

	// CreateSchedulerSession creates a new scheduler session.
	CreateSchedulerSession(ctx context.Context, req *domain.CreateSchedulerSessionRequest) (*domain.SchedulerSession, error)

	// GetSchedulerSession retrieves a specific scheduler session.
	GetSchedulerSession(ctx context.Context, sessionID string) (*domain.SchedulerSession, error)

	// ================================
	// BOOKING OPERATIONS
	// ================================

	// Booking operations authenticate with a Scheduler session token minted from
	// the configuration ID (booking endpoints reject the application API key), so
	// each takes the booking's configurationID.

	// GetBooking retrieves a specific booking.
	GetBooking(ctx context.Context, configurationID, bookingID string) (*domain.Booking, error)

	// ConfirmBooking confirms a booking.
	ConfirmBooking(ctx context.Context, configurationID, bookingID string, req *domain.ConfirmBookingRequest) (*domain.Booking, error)

	// RescheduleBooking reschedules an existing booking.
	RescheduleBooking(ctx context.Context, configurationID, bookingID string, req *domain.RescheduleBookingRequest) (*domain.Booking, error)

	// CancelBooking cancels a booking.
	CancelBooking(ctx context.Context, configurationID, bookingID string, reason string) error

	// ================================
	// GROUP EVENT OPERATIONS
	// ================================

	// ListGroupEvents retrieves the group events for a configuration within a
	// time window. calendarID, startTime, and endTime (Unix seconds) are all
	// required by the API.
	ListGroupEvents(ctx context.Context, grantID, configID, calendarID string, startTime, endTime int64) ([]domain.GroupEvent, error)

	// CreateGroupEvent creates a group event under a configuration. The API may
	// return more than one event (e.g. when recurrence is set).
	CreateGroupEvent(ctx context.Context, grantID, configID string, req *domain.CreateGroupEventRequest) ([]domain.GroupEvent, error)

	// UpdateGroupEvent updates a group event.
	UpdateGroupEvent(ctx context.Context, grantID, configID, eventID string, req *domain.UpdateGroupEventRequest) ([]domain.GroupEvent, error)

	// DeleteGroupEvent deletes a group event.
	DeleteGroupEvent(ctx context.Context, grantID, configID, eventID string) error

	// ImportGroupEvents imports existing provider events as group events under a
	// configuration. This endpoint is configuration-scoped (not grant-scoped).
	ImportGroupEvents(ctx context.Context, configID string, items []domain.ImportGroupEventItem) ([]domain.GroupEvent, error)
}
