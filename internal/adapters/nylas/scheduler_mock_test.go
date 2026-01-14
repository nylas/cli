package nylas_test

import (
	"context"
	"testing"

	"github.com/nylas/cli/internal/adapters/nylas"
	"github.com/nylas/cli/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMockClient_SchedulerOperations(t *testing.T) {
	ctx := context.Background()
	mock := nylas.NewMockClient()

	// Test ListSchedulerConfigurations
	configs, err := mock.ListSchedulerConfigurations(ctx)
	require.NoError(t, err)
	assert.NotEmpty(t, configs)

	// Test GetSchedulerConfiguration
	config, err := mock.GetSchedulerConfiguration(ctx, "config-123")
	require.NoError(t, err)
	assert.Equal(t, "config-123", config.ID)

	// Test CreateSchedulerConfiguration
	createReq := &domain.CreateSchedulerConfigurationRequest{Name: "Test Config"}
	created, err := mock.CreateSchedulerConfiguration(ctx, createReq)
	require.NoError(t, err)
	assert.NotEmpty(t, created.ID)

	// Test UpdateSchedulerConfiguration
	updateReq := &domain.UpdateSchedulerConfigurationRequest{Name: strPtr("Updated")}
	updated, err := mock.UpdateSchedulerConfiguration(ctx, "config-456", updateReq)
	require.NoError(t, err)
	assert.Equal(t, "Updated", updated.Name)

	// Test DeleteSchedulerConfiguration
	err = mock.DeleteSchedulerConfiguration(ctx, "config-789")
	require.NoError(t, err)

	// Test CreateSchedulerSession
	sessionReq := &domain.CreateSchedulerSessionRequest{ConfigurationID: "config-123"}
	session, err := mock.CreateSchedulerSession(ctx, sessionReq)
	require.NoError(t, err)
	assert.NotEmpty(t, session.SessionID)

	// Test GetSchedulerSession
	getSession, err := mock.GetSchedulerSession(ctx, "session-123")
	require.NoError(t, err)
	assert.Equal(t, "session-123", getSession.SessionID)

	// Test GetBooking
	booking, err := mock.GetBooking(ctx, "booking-123")
	require.NoError(t, err)
	assert.Equal(t, "booking-123", booking.BookingID)

	// Test ListBookings
	bookings, err := mock.ListBookings(ctx, "config-123")
	require.NoError(t, err)
	assert.NotEmpty(t, bookings)

	// Test ConfirmBooking
	confirmReq := &domain.ConfirmBookingRequest{}
	confirmed, err := mock.ConfirmBooking(ctx, "booking-123", confirmReq)
	require.NoError(t, err)
	assert.Equal(t, "confirmed", confirmed.Status)

	// Test RescheduleBooking
	rescheduleReq := &domain.RescheduleBookingRequest{
		StartTime: 1704067200,
		EndTime:   1704070800,
	}
	rescheduled, err := mock.RescheduleBooking(ctx, "booking-456", rescheduleReq)
	require.NoError(t, err)
	assert.NotEmpty(t, rescheduled.BookingID)

	// Test CancelBooking
	err = mock.CancelBooking(ctx, "booking-789", "User cancelled")
	require.NoError(t, err)

	// Test GetSchedulerPage
	page, err := mock.GetSchedulerPage(ctx, "page-123")
	require.NoError(t, err)
	assert.Equal(t, "page-123", page.ID)

	// Test ListSchedulerPages
	pages, err := mock.ListSchedulerPages(ctx)
	require.NoError(t, err)
	assert.NotEmpty(t, pages)

	// Test CreateSchedulerPage
	pageReq := &domain.CreateSchedulerPageRequest{Name: "Test Page"}
	newPage, err := mock.CreateSchedulerPage(ctx, pageReq)
	require.NoError(t, err)
	assert.NotEmpty(t, newPage.ID)

	// Test UpdateSchedulerPage
	updatePageReq := &domain.UpdateSchedulerPageRequest{Name: strPtr("Updated Page")}
	updatedPage, err := mock.UpdateSchedulerPage(ctx, "page-456", updatePageReq)
	require.NoError(t, err)
	assert.Equal(t, "Updated Page", updatedPage.Name)

	// Test DeleteSchedulerPage
	err = mock.DeleteSchedulerPage(ctx, "page-789")
	require.NoError(t, err)
}

// Demo Client Tests

func TestDemoClient_SchedulerOperations(t *testing.T) {
	ctx := context.Background()
	demo := nylas.NewDemoClient()

	// Test ListSchedulerConfigurations
	configs, err := demo.ListSchedulerConfigurations(ctx)
	require.NoError(t, err)
	assert.NotEmpty(t, configs)

	// Test GetSchedulerConfiguration
	config, err := demo.GetSchedulerConfiguration(ctx, "demo-config")
	require.NoError(t, err)
	assert.NotEmpty(t, config.ID)

	// Test CreateSchedulerConfiguration
	createReq := &domain.CreateSchedulerConfigurationRequest{Name: "Demo Config"}
	created, err := demo.CreateSchedulerConfiguration(ctx, createReq)
	require.NoError(t, err)
	assert.Equal(t, "Demo Config", created.Name)

	// Test UpdateSchedulerConfiguration
	updateReq := &domain.UpdateSchedulerConfigurationRequest{Name: strPtr("Demo Updated")}
	updated, err := demo.UpdateSchedulerConfiguration(ctx, "demo-config", updateReq)
	require.NoError(t, err)
	assert.Equal(t, "Demo Updated", updated.Name)

	// Test DeleteSchedulerConfiguration
	err = demo.DeleteSchedulerConfiguration(ctx, "demo-config")
	require.NoError(t, err)

	// Test sessions, bookings, and pages
	session, err := demo.CreateSchedulerSession(ctx, &domain.CreateSchedulerSessionRequest{ConfigurationID: "demo-config"})
	require.NoError(t, err)
	assert.NotEmpty(t, session.SessionID)

	getSession, err := demo.GetSchedulerSession(ctx, "demo-session")
	require.NoError(t, err)
	assert.NotEmpty(t, getSession.SessionID)

	bookings, err := demo.ListBookings(ctx, "demo-config")
	require.NoError(t, err)
	assert.NotEmpty(t, bookings)

	pages, err := demo.ListSchedulerPages(ctx)
	require.NoError(t, err)
	assert.NotEmpty(t, pages)
}

// Helper function
func strPtr(s string) *string {
	return &s
}
