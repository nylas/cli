//go:build integration
// +build integration

package nylas_test

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/nylas/cli/internal/domain"
	"github.com/stretchr/testify/require"
)

func TestIntegration_SendMessageWithTracking(t *testing.T) {
	if os.Getenv("NYLAS_TEST_SEND_EMAIL") != "true" {
		t.Skip("Skipping send email test - set NYLAS_TEST_SEND_EMAIL=true to enable")
	}

	client, grantID := getTestClient(t)
	ctx, cancel := createLongTestContext()
	defer cancel()

	testEmail := os.Getenv("NYLAS_TEST_EMAIL")
	if testEmail == "" {
		t.Skip("NYLAS_TEST_EMAIL not set")
	}

	timestamp := time.Now().Format("2006-01-02 15:04:05")
	req := &domain.SendMessageRequest{
		Subject: fmt.Sprintf("Tracking Test - %s", timestamp),
		Body:    "<html><body><p>Test email with tracking enabled</p></body></html>",
		To:      []domain.EmailParticipant{{Email: testEmail, Name: "Test Recipient"}},
		TrackingOpts: &domain.TrackingOptions{
			Opens: true,
			Links: true,
			Label: "integration-test",
		},
	}

	msg, err := client.SendMessage(ctx, grantID, req)
	skipIfTrialAccountLimitation(t, err)
	require.NoError(t, err)
	require.NotEmpty(t, msg.ID)

	t.Logf("Sent tracked email: %s", msg.ID)
	t.Logf("Subject: %s", msg.Subject)
	t.Logf("Tracking: opens=%v, links=%v, label=%s", true, true, "integration-test")
}

func TestIntegration_SendMessageWithMetadata(t *testing.T) {
	if os.Getenv("NYLAS_TEST_SEND_EMAIL") != "true" {
		t.Skip("Skipping send email test - set NYLAS_TEST_SEND_EMAIL=true to enable")
	}

	client, grantID := getTestClient(t)
	ctx, cancel := createLongTestContext()
	defer cancel()

	testEmail := os.Getenv("NYLAS_TEST_EMAIL")
	if testEmail == "" {
		t.Skip("NYLAS_TEST_EMAIL not set")
	}

	timestamp := time.Now().Format("2006-01-02 15:04:05")
	req := &domain.SendMessageRequest{
		Subject: fmt.Sprintf("Metadata Test - %s", timestamp),
		Body:    "<html><body><p>Test email with custom metadata</p></body></html>",
		To:      []domain.EmailParticipant{{Email: testEmail, Name: "Test Recipient"}},
		Metadata: map[string]string{
			"campaign_id": "test-campaign-001",
			"customer_id": "cust-12345",
			"test_run":    "integration",
		},
	}

	msg, err := client.SendMessage(ctx, grantID, req)
	require.NoError(t, err)
	require.NotEmpty(t, msg.ID)

	t.Logf("Sent email with metadata: %s", msg.ID)
	t.Logf("Subject: %s", msg.Subject)
	t.Logf("Metadata: campaign_id=test-campaign-001, customer_id=cust-12345")
}
