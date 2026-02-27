package mcp

import (
	"context"

	"github.com/nylas/cli/internal/domain"
)

// ============================================================================
// NotetakerClient
// ============================================================================

func (m *mockNylasClient) ListNotetakers(ctx context.Context, grantID string, params *domain.NotetakerQueryParams) ([]domain.Notetaker, error) {
	return nil, nil
}
func (m *mockNylasClient) GetNotetaker(ctx context.Context, grantID, notetakerID string) (*domain.Notetaker, error) {
	return nil, nil
}
func (m *mockNylasClient) CreateNotetaker(ctx context.Context, grantID string, req *domain.CreateNotetakerRequest) (*domain.Notetaker, error) {
	return nil, nil
}
func (m *mockNylasClient) DeleteNotetaker(ctx context.Context, grantID, notetakerID string) error {
	return nil
}
func (m *mockNylasClient) GetNotetakerMedia(ctx context.Context, grantID, notetakerID string) (*domain.MediaData, error) {
	return nil, nil
}

// ============================================================================
// InboundClient
// ============================================================================

func (m *mockNylasClient) ListInboundInboxes(ctx context.Context) ([]domain.InboundInbox, error) {
	return nil, nil
}
func (m *mockNylasClient) GetInboundInbox(ctx context.Context, grantID string) (*domain.InboundInbox, error) {
	return nil, nil
}
func (m *mockNylasClient) CreateInboundInbox(ctx context.Context, email string) (*domain.InboundInbox, error) {
	return nil, nil
}
func (m *mockNylasClient) DeleteInboundInbox(ctx context.Context, grantID string) error { return nil }
func (m *mockNylasClient) GetInboundMessages(ctx context.Context, grantID string, params *domain.MessageQueryParams) ([]domain.InboundMessage, error) {
	return nil, nil
}

// ============================================================================
// SchedulerClient
// ============================================================================

func (m *mockNylasClient) ListSchedulerConfigurations(ctx context.Context) ([]domain.SchedulerConfiguration, error) {
	return nil, nil
}
func (m *mockNylasClient) GetSchedulerConfiguration(ctx context.Context, configID string) (*domain.SchedulerConfiguration, error) {
	return nil, nil
}
func (m *mockNylasClient) CreateSchedulerConfiguration(ctx context.Context, req *domain.CreateSchedulerConfigurationRequest) (*domain.SchedulerConfiguration, error) {
	return nil, nil
}
func (m *mockNylasClient) UpdateSchedulerConfiguration(ctx context.Context, configID string, req *domain.UpdateSchedulerConfigurationRequest) (*domain.SchedulerConfiguration, error) {
	return nil, nil
}
func (m *mockNylasClient) DeleteSchedulerConfiguration(ctx context.Context, configID string) error {
	return nil
}
func (m *mockNylasClient) CreateSchedulerSession(ctx context.Context, req *domain.CreateSchedulerSessionRequest) (*domain.SchedulerSession, error) {
	return nil, nil
}
func (m *mockNylasClient) GetSchedulerSession(ctx context.Context, sessionID string) (*domain.SchedulerSession, error) {
	return nil, nil
}
func (m *mockNylasClient) ListBookings(ctx context.Context, configID string) ([]domain.Booking, error) {
	return nil, nil
}
func (m *mockNylasClient) GetBooking(ctx context.Context, bookingID string) (*domain.Booking, error) {
	return nil, nil
}
func (m *mockNylasClient) ConfirmBooking(ctx context.Context, bookingID string, req *domain.ConfirmBookingRequest) (*domain.Booking, error) {
	return nil, nil
}
func (m *mockNylasClient) RescheduleBooking(ctx context.Context, bookingID string, req *domain.RescheduleBookingRequest) (*domain.Booking, error) {
	return nil, nil
}
func (m *mockNylasClient) CancelBooking(ctx context.Context, bookingID string, reason string) error {
	return nil
}
func (m *mockNylasClient) ListSchedulerPages(ctx context.Context) ([]domain.SchedulerPage, error) {
	return nil, nil
}
func (m *mockNylasClient) GetSchedulerPage(ctx context.Context, pageID string) (*domain.SchedulerPage, error) {
	return nil, nil
}
func (m *mockNylasClient) CreateSchedulerPage(ctx context.Context, req *domain.CreateSchedulerPageRequest) (*domain.SchedulerPage, error) {
	return nil, nil
}
func (m *mockNylasClient) UpdateSchedulerPage(ctx context.Context, pageID string, req *domain.UpdateSchedulerPageRequest) (*domain.SchedulerPage, error) {
	return nil, nil
}
func (m *mockNylasClient) DeleteSchedulerPage(ctx context.Context, pageID string) error { return nil }

// ============================================================================
// AdminClient
// ============================================================================

func (m *mockNylasClient) ListApplications(ctx context.Context) ([]domain.Application, error) {
	return nil, nil
}
func (m *mockNylasClient) GetApplication(ctx context.Context, appID string) (*domain.Application, error) {
	return nil, nil
}
func (m *mockNylasClient) CreateApplication(ctx context.Context, req *domain.CreateApplicationRequest) (*domain.Application, error) {
	return nil, nil
}
func (m *mockNylasClient) UpdateApplication(ctx context.Context, appID string, req *domain.UpdateApplicationRequest) (*domain.Application, error) {
	return nil, nil
}
func (m *mockNylasClient) DeleteApplication(ctx context.Context, appID string) error { return nil }
func (m *mockNylasClient) ListConnectors(ctx context.Context) ([]domain.Connector, error) {
	return nil, nil
}
func (m *mockNylasClient) GetConnector(ctx context.Context, connectorID string) (*domain.Connector, error) {
	return nil, nil
}
func (m *mockNylasClient) CreateConnector(ctx context.Context, req *domain.CreateConnectorRequest) (*domain.Connector, error) {
	return nil, nil
}
func (m *mockNylasClient) UpdateConnector(ctx context.Context, connectorID string, req *domain.UpdateConnectorRequest) (*domain.Connector, error) {
	return nil, nil
}
func (m *mockNylasClient) DeleteConnector(ctx context.Context, connectorID string) error { return nil }
func (m *mockNylasClient) ListCredentials(ctx context.Context, connectorID string) ([]domain.ConnectorCredential, error) {
	return nil, nil
}
func (m *mockNylasClient) GetCredential(ctx context.Context, credentialID string) (*domain.ConnectorCredential, error) {
	return nil, nil
}
func (m *mockNylasClient) CreateCredential(ctx context.Context, connectorID string, req *domain.CreateCredentialRequest) (*domain.ConnectorCredential, error) {
	return nil, nil
}
func (m *mockNylasClient) UpdateCredential(ctx context.Context, credentialID string, req *domain.UpdateCredentialRequest) (*domain.ConnectorCredential, error) {
	return nil, nil
}
func (m *mockNylasClient) DeleteCredential(ctx context.Context, credentialID string) error {
	return nil
}
func (m *mockNylasClient) ListAllGrants(ctx context.Context, params *domain.GrantsQueryParams) ([]domain.Grant, error) {
	return nil, nil
}
func (m *mockNylasClient) GetGrantStats(ctx context.Context) (*domain.GrantStats, error) {
	return nil, nil
}

// ============================================================================
// TransactionalClient
// ============================================================================

func (m *mockNylasClient) SendTransactionalMessage(ctx context.Context, domainName string, req *domain.SendMessageRequest) (*domain.Message, error) {
	return nil, nil
}

// ============================================================================
// Configuration methods
// ============================================================================

func (m *mockNylasClient) SetRegion(region string)                              {}
func (m *mockNylasClient) SetCredentials(clientID, clientSecret, apiKey string) {}
