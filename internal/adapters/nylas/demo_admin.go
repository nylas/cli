package nylas

import (
	"context"
	"time"

	"github.com/nylas/cli/internal/domain"
)

// Scheduler Demo Implementations

func (d *DemoClient) ListSchedulerConfigurations(ctx context.Context) ([]domain.SchedulerConfiguration, error) {
	return []domain.SchedulerConfiguration{
		{ID: "config-demo-1", Name: "30 Minute Meeting", Slug: "30min-demo"},
		{ID: "config-demo-2", Name: "1 Hour Meeting", Slug: "1hour-demo"},
	}, nil
}

func (d *DemoClient) GetSchedulerConfiguration(ctx context.Context, configID string) (*domain.SchedulerConfiguration, error) {
	return &domain.SchedulerConfiguration{
		ID:   configID,
		Name: "30 Minute Meeting",
		Slug: "30min-demo",
	}, nil
}

func (d *DemoClient) CreateSchedulerConfiguration(ctx context.Context, req *domain.CreateSchedulerConfigurationRequest) (*domain.SchedulerConfiguration, error) {
	return &domain.SchedulerConfiguration{
		ID:   "config-demo-new",
		Name: req.Name,
		Slug: req.Slug,
	}, nil
}

func (d *DemoClient) UpdateSchedulerConfiguration(ctx context.Context, configID string, req *domain.UpdateSchedulerConfigurationRequest) (*domain.SchedulerConfiguration, error) {
	name := "Updated Configuration"
	if req.Name != nil {
		name = *req.Name
	}
	return &domain.SchedulerConfiguration{
		ID:   configID,
		Name: name,
	}, nil
}

func (d *DemoClient) DeleteSchedulerConfiguration(ctx context.Context, configID string) error {
	return nil
}

func (d *DemoClient) CreateSchedulerSession(ctx context.Context, req *domain.CreateSchedulerSessionRequest) (*domain.SchedulerSession, error) {
	return &domain.SchedulerSession{
		SessionID:       "session-demo-123",
		ConfigurationID: req.ConfigurationID,
		BookingURL:      "https://schedule.nylas.com/demo-session",
	}, nil
}

func (d *DemoClient) GetSchedulerSession(ctx context.Context, sessionID string) (*domain.SchedulerSession, error) {
	return &domain.SchedulerSession{
		SessionID:       sessionID,
		ConfigurationID: "config-demo-1",
		BookingURL:      "https://schedule.nylas.com/demo-session",
	}, nil
}

func (d *DemoClient) ListBookings(ctx context.Context, configID string) ([]domain.Booking, error) {
	return []domain.Booking{
		{
			BookingID: "booking-demo-1",
			Title:     "Demo Meeting",
			Status:    "confirmed",
		},
	}, nil
}

func (d *DemoClient) GetBooking(ctx context.Context, bookingID string) (*domain.Booking, error) {
	return &domain.Booking{
		BookingID: bookingID,
		Title:     "Demo Meeting",
		Status:    "confirmed",
	}, nil
}

func (d *DemoClient) ConfirmBooking(ctx context.Context, bookingID string, req *domain.ConfirmBookingRequest) (*domain.Booking, error) {
	return &domain.Booking{
		BookingID: bookingID,
		Status:    req.Status,
	}, nil
}

func (d *DemoClient) RescheduleBooking(ctx context.Context, bookingID string, req *domain.RescheduleBookingRequest) (*domain.Booking, error) {
	return &domain.Booking{
		BookingID: bookingID,
		Status:    "confirmed",
		StartTime: time.Unix(req.StartTime, 0),
		EndTime:   time.Unix(req.EndTime, 0),
	}, nil
}

func (d *DemoClient) CancelBooking(ctx context.Context, bookingID string, reason string) error {
	return nil
}

func (d *DemoClient) ListSchedulerPages(ctx context.Context) ([]domain.SchedulerPage, error) {
	return []domain.SchedulerPage{
		{ID: "page-demo-1", Name: "Demo Booking Page", Slug: "book-demo"},
	}, nil
}

func (d *DemoClient) GetSchedulerPage(ctx context.Context, pageID string) (*domain.SchedulerPage, error) {
	return &domain.SchedulerPage{
		ID:   pageID,
		Name: "Demo Booking Page",
		Slug: "book-demo",
	}, nil
}

func (d *DemoClient) CreateSchedulerPage(ctx context.Context, req *domain.CreateSchedulerPageRequest) (*domain.SchedulerPage, error) {
	return &domain.SchedulerPage{
		ID:   "page-demo-new",
		Name: req.Name,
		Slug: req.Slug,
	}, nil
}

func (d *DemoClient) UpdateSchedulerPage(ctx context.Context, pageID string, req *domain.UpdateSchedulerPageRequest) (*domain.SchedulerPage, error) {
	name := "Updated Page"
	if req.Name != nil {
		name = *req.Name
	}
	return &domain.SchedulerPage{
		ID:   pageID,
		Name: name,
	}, nil
}

func (d *DemoClient) DeleteSchedulerPage(ctx context.Context, pageID string) error {
	return nil
}

// Admin Demo Implementations

func (d *DemoClient) ListApplications(ctx context.Context) ([]domain.Application, error) {
	return []domain.Application{
		{ID: "app-demo-1", ApplicationID: "app-id-demo-1", Region: "us"},
	}, nil
}

func (d *DemoClient) GetApplication(ctx context.Context, appID string) (*domain.Application, error) {
	return &domain.Application{
		ID:            appID,
		ApplicationID: appID,
		Region:        "us",
	}, nil
}

func (d *DemoClient) CreateApplication(ctx context.Context, req *domain.CreateApplicationRequest) (*domain.Application, error) {
	return &domain.Application{
		ID:            "app-demo-new",
		ApplicationID: "app-id-demo-new",
		Region:        req.Region,
	}, nil
}

func (d *DemoClient) UpdateApplication(ctx context.Context, appID string, req *domain.UpdateApplicationRequest) (*domain.Application, error) {
	return &domain.Application{
		ID:            appID,
		ApplicationID: appID,
		Region:        "us",
	}, nil
}

func (d *DemoClient) DeleteApplication(ctx context.Context, appID string) error {
	return nil
}

func (d *DemoClient) ListConnectors(ctx context.Context) ([]domain.Connector, error) {
	return []domain.Connector{
		{ID: "conn-demo-1", Name: "Google Demo Connector", Provider: "google"},
		{ID: "conn-demo-2", Name: "Microsoft Demo Connector", Provider: "microsoft"},
	}, nil
}

func (d *DemoClient) GetConnector(ctx context.Context, connectorID string) (*domain.Connector, error) {
	return &domain.Connector{
		ID:       connectorID,
		Name:     "Google Demo Connector",
		Provider: "google",
	}, nil
}

func (d *DemoClient) CreateConnector(ctx context.Context, req *domain.CreateConnectorRequest) (*domain.Connector, error) {
	return &domain.Connector{
		ID:       "conn-demo-new",
		Name:     req.Name,
		Provider: req.Provider,
	}, nil
}

func (d *DemoClient) UpdateConnector(ctx context.Context, connectorID string, req *domain.UpdateConnectorRequest) (*domain.Connector, error) {
	name := "Updated Connector"
	if req.Name != nil {
		name = *req.Name
	}
	return &domain.Connector{
		ID:       connectorID,
		Name:     name,
		Provider: "google",
	}, nil
}

func (d *DemoClient) DeleteConnector(ctx context.Context, connectorID string) error {
	return nil
}

func (d *DemoClient) ListCredentials(ctx context.Context, connectorID string) ([]domain.ConnectorCredential, error) {
	return []domain.ConnectorCredential{
		{ID: "cred-demo-1", Name: "OAuth Demo Credential", CredentialType: "oauth"},
	}, nil
}

func (d *DemoClient) GetCredential(ctx context.Context, credentialID string) (*domain.ConnectorCredential, error) {
	return &domain.ConnectorCredential{
		ID:             credentialID,
		Name:           "OAuth Demo Credential",
		CredentialType: "oauth",
	}, nil
}

func (d *DemoClient) CreateCredential(ctx context.Context, connectorID string, req *domain.CreateCredentialRequest) (*domain.ConnectorCredential, error) {
	return &domain.ConnectorCredential{
		ID:             "cred-demo-new",
		Name:           req.Name,
		CredentialType: req.CredentialType,
	}, nil
}

func (d *DemoClient) UpdateCredential(ctx context.Context, credentialID string, req *domain.UpdateCredentialRequest) (*domain.ConnectorCredential, error) {
	name := "Updated Credential"
	if req.Name != nil {
		name = *req.Name
	}
	return &domain.ConnectorCredential{
		ID:             credentialID,
		Name:           name,
		CredentialType: "oauth",
	}, nil
}

func (d *DemoClient) DeleteCredential(ctx context.Context, credentialID string) error {
	return nil
}
