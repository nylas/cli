package nylas

import (
	"context"
	"fmt"
	"time"

	"github.com/nylas/cli/internal/domain"
)

func (m *MockClient) ListSchedulerConfigurations(ctx context.Context) ([]domain.SchedulerConfiguration, error) {
	return []domain.SchedulerConfiguration{
		{ID: "config-1", Name: "30 Minute Meeting", Slug: "30min"},
		{ID: "config-2", Name: "1 Hour Meeting", Slug: "1hour"},
	}, nil
}

func (m *MockClient) GetSchedulerConfiguration(ctx context.Context, configID string) (*domain.SchedulerConfiguration, error) {
	return &domain.SchedulerConfiguration{
		ID:   configID,
		Name: "30 Minute Meeting",
		Slug: "30min",
	}, nil
}

func (m *MockClient) CreateSchedulerConfiguration(ctx context.Context, req *domain.CreateSchedulerConfigurationRequest) (*domain.SchedulerConfiguration, error) {
	return &domain.SchedulerConfiguration{
		ID:   "new-config",
		Name: req.Name,
		Slug: req.Slug,
	}, nil
}

func (m *MockClient) UpdateSchedulerConfiguration(ctx context.Context, configID string, req *domain.UpdateSchedulerConfigurationRequest) (*domain.SchedulerConfiguration, error) {
	name := "Updated Configuration"
	if req.Name != nil {
		name = *req.Name
	}
	return &domain.SchedulerConfiguration{
		ID:   configID,
		Name: name,
	}, nil
}

func (m *MockClient) DeleteSchedulerConfiguration(ctx context.Context, configID string) error {
	return nil
}

func (m *MockClient) CreateSchedulerSession(ctx context.Context, req *domain.CreateSchedulerSessionRequest) (*domain.SchedulerSession, error) {
	return &domain.SchedulerSession{
		SessionID:       "session-123",
		ConfigurationID: req.ConfigurationID,
		BookingURL:      fmt.Sprintf("https://schedule.nylas.com/%s", req.Slug),
	}, nil
}

func (m *MockClient) GetSchedulerSession(ctx context.Context, sessionID string) (*domain.SchedulerSession, error) {
	return &domain.SchedulerSession{
		SessionID:       sessionID,
		ConfigurationID: "config-1",
		BookingURL:      "https://schedule.nylas.com/session-123",
	}, nil
}

func (m *MockClient) ListBookings(ctx context.Context, configID string) ([]domain.Booking, error) {
	return []domain.Booking{
		{
			BookingID: "booking-1",
			Title:     "Meeting with John",
			Status:    "confirmed",
		},
	}, nil
}

func (m *MockClient) GetBooking(ctx context.Context, bookingID string) (*domain.Booking, error) {
	return &domain.Booking{
		BookingID: bookingID,
		Title:     "Meeting with John",
		Status:    "confirmed",
	}, nil
}

func (m *MockClient) ConfirmBooking(ctx context.Context, bookingID string, req *domain.ConfirmBookingRequest) (*domain.Booking, error) {
	return &domain.Booking{
		BookingID: bookingID,
		Status:    "confirmed",
	}, nil
}

func (m *MockClient) RescheduleBooking(ctx context.Context, bookingID string, req *domain.RescheduleBookingRequest) (*domain.Booking, error) {
	return &domain.Booking{
		BookingID: bookingID,
		Status:    "confirmed",
		StartTime: time.Unix(req.StartTime, 0),
		EndTime:   time.Unix(req.EndTime, 0),
	}, nil
}

func (m *MockClient) CancelBooking(ctx context.Context, bookingID string, reason string) error {
	return nil
}

func (m *MockClient) ListSchedulerPages(ctx context.Context) ([]domain.SchedulerPage, error) {
	return []domain.SchedulerPage{
		{ID: "page-1", Name: "Booking Page", Slug: "book-me"},
	}, nil
}

func (m *MockClient) GetSchedulerPage(ctx context.Context, pageID string) (*domain.SchedulerPage, error) {
	return &domain.SchedulerPage{
		ID:   pageID,
		Name: "Booking Page",
		Slug: "book-me",
	}, nil
}

func (m *MockClient) CreateSchedulerPage(ctx context.Context, req *domain.CreateSchedulerPageRequest) (*domain.SchedulerPage, error) {
	return &domain.SchedulerPage{
		ID:   "new-page",
		Name: req.Name,
		Slug: req.Slug,
	}, nil
}

func (m *MockClient) UpdateSchedulerPage(ctx context.Context, pageID string, req *domain.UpdateSchedulerPageRequest) (*domain.SchedulerPage, error) {
	name := "Updated Page"
	if req.Name != nil {
		name = *req.Name
	}
	return &domain.SchedulerPage{
		ID:   pageID,
		Name: name,
	}, nil
}

func (m *MockClient) DeleteSchedulerPage(ctx context.Context, pageID string) error {
	return nil
}

// Admin Mock Implementations
