package ports

import (
	"context"

	"github.com/nylas/cli/internal/domain"
)

// CalendarClient defines the interface for calendar, event, and availability operations.
type CalendarClient interface {
	// ================================
	// CALENDAR OPERATIONS
	// ================================

	// GetCalendars retrieves all calendars.
	GetCalendars(ctx context.Context, grantID string) ([]domain.Calendar, error)

	// GetCalendar retrieves a specific calendar.
	GetCalendar(ctx context.Context, grantID, calendarID string) (*domain.Calendar, error)

	// CreateCalendar creates a new calendar.
	CreateCalendar(ctx context.Context, grantID string, req *domain.CreateCalendarRequest) (*domain.Calendar, error)

	// UpdateCalendar updates an existing calendar.
	UpdateCalendar(ctx context.Context, grantID, calendarID string, req *domain.UpdateCalendarRequest) (*domain.Calendar, error)

	// DeleteCalendar deletes a calendar.
	DeleteCalendar(ctx context.Context, grantID, calendarID string) error

	// ================================
	// EVENT OPERATIONS
	// ================================

	// GetEvents retrieves events with query parameters.
	GetEvents(ctx context.Context, grantID, calendarID string, params *domain.EventQueryParams) ([]domain.Event, error)

	// GetEventsWithCursor retrieves events with cursor-based pagination.
	GetEventsWithCursor(ctx context.Context, grantID, calendarID string, params *domain.EventQueryParams) (*domain.EventListResponse, error)

	// GetEvent retrieves a specific event.
	GetEvent(ctx context.Context, grantID, calendarID, eventID string) (*domain.Event, error)

	// ImportEvents bulk-reads events from a calendar over a time window
	// (GET /v3/grants/{id}/events/import), including expanded recurring
	// instances. Intended for migration/export. CalendarID is required.
	ImportEvents(ctx context.Context, grantID string, params *domain.EventQueryParams) ([]domain.Event, error)

	// CreateEvent creates a new event.
	CreateEvent(ctx context.Context, grantID, calendarID string, req *domain.CreateEventRequest) (*domain.Event, error)

	// UpdateEvent updates an existing event.
	UpdateEvent(ctx context.Context, grantID, calendarID, eventID string, req *domain.UpdateEventRequest) (*domain.Event, error)

	// DeleteEvent deletes an event.
	DeleteEvent(ctx context.Context, grantID, calendarID, eventID string) error

	// SendRSVP sends an RSVP response to an event.
	SendRSVP(ctx context.Context, grantID, calendarID, eventID string, req *domain.SendRSVPRequest) error

	// ================================
	// AVAILABILITY OPERATIONS
	// ================================

	// GetFreeBusy retrieves free/busy information.
	GetFreeBusy(ctx context.Context, grantID string, req *domain.FreeBusyRequest) (*domain.FreeBusyResponse, error)

	// GetAvailability retrieves availability across multiple accounts.
	GetAvailability(ctx context.Context, req *domain.AvailabilityRequest) (*domain.AvailabilityResponse, error)

	// ListRoomResources retrieves bookable room and equipment resources for a grant.
	ListRoomResources(ctx context.Context, grantID string) ([]domain.RoomResource, error)

	// ================================
	// VIRTUAL CALENDAR OPERATIONS
	// ================================

	// CreateVirtualCalendarGrant creates a virtual calendar grant.
	CreateVirtualCalendarGrant(ctx context.Context, email string) (*domain.VirtualCalendarGrant, error)

	// ListVirtualCalendarGrants retrieves all virtual calendar grants.
	ListVirtualCalendarGrants(ctx context.Context) ([]domain.VirtualCalendarGrant, error)

	// GetVirtualCalendarGrant retrieves a specific virtual calendar grant.
	GetVirtualCalendarGrant(ctx context.Context, grantID string) (*domain.VirtualCalendarGrant, error)

	// DeleteVirtualCalendarGrant deletes a virtual calendar grant.
	DeleteVirtualCalendarGrant(ctx context.Context, grantID string) error

	// ================================
	// RECURRING EVENT OPERATIONS
	// ================================

	// GetRecurringEventInstances retrieves instances of a recurring event.
	GetRecurringEventInstances(ctx context.Context, grantID, calendarID, masterEventID string, params *domain.EventQueryParams) ([]domain.Event, error)

	// UpdateRecurringEventInstance updates a specific instance of a recurring event.
	UpdateRecurringEventInstance(ctx context.Context, grantID, calendarID, eventID string, req *domain.UpdateEventRequest) (*domain.Event, error)

	// DeleteRecurringEventInstance deletes a specific instance of a recurring event.
	DeleteRecurringEventInstance(ctx context.Context, grantID, calendarID, eventID string) error
}
