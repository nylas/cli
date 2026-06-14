package nylas

import (
	"context"
	"errors"
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

// Group Event Demo Implementations

func (d *DemoClient) ListGroupEvents(ctx context.Context, grantID, configID, calendarID string, startTime, endTime int64) ([]domain.GroupEvent, error) {
	return []domain.GroupEvent{
		{ID: "ge-001", Title: "Annual Philosophy Club Meeting", CalendarID: "primary", Capacity: 50, Location: "Library, Cave Room"},
		{ID: "ge-002", Title: "Intro to Stoicism Workshop", CalendarID: "primary", Capacity: 20},
	}, nil
}

func (d *DemoClient) CreateGroupEvent(ctx context.Context, grantID, configID string, req *domain.CreateGroupEventRequest) ([]domain.GroupEvent, error) {
	return []domain.GroupEvent{
		{ID: "ge-new", Title: req.Title, CalendarID: req.CalendarID, Capacity: req.Capacity, Participants: req.Participants, When: req.When},
	}, nil
}

func (d *DemoClient) UpdateGroupEvent(ctx context.Context, grantID, configID, eventID string, req *domain.UpdateGroupEventRequest) ([]domain.GroupEvent, error) {
	return []domain.GroupEvent{
		{ID: eventID, Title: req.Title, CalendarID: req.CalendarID, Capacity: req.Capacity},
	}, nil
}

func (d *DemoClient) DeleteGroupEvent(ctx context.Context, grantID, configID, eventID string) error {
	return nil
}

func (d *DemoClient) ImportGroupEvents(ctx context.Context, configID string, items []domain.ImportGroupEventItem) ([]domain.GroupEvent, error) {
	events := make([]domain.GroupEvent, 0, len(items))
	for _, it := range items {
		events = append(events, domain.GroupEvent{ID: "ge-import", CalendarID: it.CalendarID, Capacity: it.Capacity})
	}
	return events, nil
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

func (d *DemoClient) ListCallbackURIs(ctx context.Context) ([]domain.CallbackURI, error) {
	return []domain.CallbackURI{
		{ID: "cb-demo-1", URL: "http://localhost:9007/callback", Platform: "web"},
	}, nil
}

func (d *DemoClient) GetCallbackURI(ctx context.Context, uriID string) (*domain.CallbackURI, error) {
	return &domain.CallbackURI{
		ID:       uriID,
		URL:      "http://localhost:9007/callback",
		Platform: "web",
	}, nil
}

func (d *DemoClient) CreateCallbackURI(ctx context.Context, req *domain.CreateCallbackURIRequest) (*domain.CallbackURI, error) {
	return &domain.CallbackURI{
		ID:       "cb-demo-new",
		URL:      req.URL,
		Platform: req.Platform,
	}, nil
}

func (d *DemoClient) UpdateCallbackURI(ctx context.Context, uriID string, req *domain.UpdateCallbackURIRequest) (*domain.CallbackURI, error) {
	uri := &domain.CallbackURI{
		ID:       uriID,
		URL:      "http://localhost:9007/callback",
		Platform: "web",
	}
	if req.URL != nil {
		uri.URL = *req.URL
	}
	if req.Platform != nil {
		uri.Platform = *req.Platform
	}
	return uri, nil
}

func (d *DemoClient) DeleteCallbackURI(ctx context.Context, uriID string) error {
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

func (d *DemoClient) ListWorkspaces(ctx context.Context) ([]domain.Workspace, error) {
	return []domain.Workspace{
		{ID: "workspace-demo-1", Name: "Demo Agent Workspace", PolicyID: "policy-demo-1"},
	}, nil
}

func (d *DemoClient) GetWorkspace(ctx context.Context, workspaceID string) (*domain.Workspace, error) {
	return &domain.Workspace{
		ID:       workspaceID,
		Name:     "Demo Agent Workspace",
		PolicyID: "policy-demo-1",
		RulesIDs: []string{"rule-demo-1"},
	}, nil
}

func (d *DemoClient) CreateWorkspace(ctx context.Context, req *domain.CreateWorkspaceRequest) (*domain.Workspace, error) {
	return &domain.Workspace{
		ID:       "workspace-demo-new",
		Name:     req.Name,
		PolicyID: req.PolicyID,
		RulesIDs: req.RulesIDs,
	}, nil
}

func (d *DemoClient) AssignWorkspaceGrants(ctx context.Context, workspaceID string, req *domain.WorkspaceAssignRequest) (*domain.WorkspaceAssignResult, error) {
	if req == nil || (len(req.AssignGrants) == 0 && len(req.RemoveGrants) == 0) {
		return nil, errors.New("at least one grant must be assigned or removed")
	}
	return &domain.WorkspaceAssignResult{
		WorkspaceID:    workspaceID,
		GrantsAssigned: append([]string(nil), req.AssignGrants...),
		GrantsRemoved:  append([]string(nil), req.RemoveGrants...),
	}, nil
}

func (d *DemoClient) UpdateWorkspace(ctx context.Context, workspaceID string, req *domain.UpdateWorkspaceRequest) (*domain.Workspace, error) {
	workspace := &domain.Workspace{ID: workspaceID, Name: "Demo Agent Workspace"}
	if req.PolicyID != nil {
		workspace.PolicyID = *req.PolicyID
	}
	if req.RulesIDs != nil {
		workspace.RulesIDs = append([]string(nil), (*req.RulesIDs)...)
	}
	return workspace, nil
}

func (d *DemoClient) DeleteWorkspace(ctx context.Context, workspaceID string) error {
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
