//go:build integration
// +build integration

package nylas_test

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/nylas/cli/internal/domain"
	"github.com/stretchr/testify/require"
)

func TestIntegration_GetSignatures(t *testing.T) {
	client, grantID := getTestClient(t)
	ctx, cancel := createLongTestContext()
	defer cancel()

	signatures, err := client.GetSignatures(ctx, grantID)
	if err != nil {
		skipIfProviderNotSupported(t, err)
	}
	require.NoError(t, err)
	t.Logf("Listed %d signatures", len(signatures))
}

func TestIntegration_SignatureLifecycle(t *testing.T) {
	if os.Getenv("NYLAS_TEST_DELETE") != "true" {
		t.Skip("Skipping destructive signature lifecycle test - set NYLAS_TEST_DELETE=true to enable")
	}

	client, grantID := getTestClient(t)
	ctx, cancel := createLongTestContext()
	defer cancel()

	signatures, err := client.GetSignatures(ctx, grantID)
	if err != nil {
		skipIfProviderNotSupported(t, err)
	}
	require.NoError(t, err)
	if len(signatures) >= 10 {
		t.Skip("Grant already has 10 signatures; skipping lifecycle test to avoid hard limit failures")
	}

	marker := fmt.Sprintf("Integration Signature %d", time.Now().UnixNano())
	created, err := client.CreateSignature(ctx, grantID, &domain.CreateSignatureRequest{
		Name: "Integration Work",
		Body: fmt.Sprintf("<p>%s</p>", marker),
	})
	require.NoError(t, err)
	require.NotNil(t, created)
	t.Logf("Created signature %s", created.ID)
	defer func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cleanupCancel()
		_ = client.DeleteSignature(cleanupCtx, grantID, created.ID)
	}()

	fetched, err := client.GetSignature(ctx, grantID, created.ID)
	require.NoError(t, err)
	require.NotNil(t, fetched)
	require.Equal(t, created.ID, fetched.ID)

	updatedMarker := marker + " Updated"
	updatedName := "Integration Updated"
	updatedBody := fmt.Sprintf("<p>%s</p>", updatedMarker)
	updated, err := client.UpdateSignature(ctx, grantID, created.ID, &domain.UpdateSignatureRequest{
		Name: &updatedName,
		Body: stringPtr(updatedBody),
	})
	require.NoError(t, err)
	require.NotNil(t, updated)
	require.Equal(t, updatedName, updated.Name)
	require.Contains(t, updated.Body, updatedMarker)

	draftRecipient := os.Getenv("NYLAS_TEST_EMAIL")
	if draftRecipient == "" {
		draftRecipient = "test@example.com"
	}

	draft, err := client.CreateDraft(ctx, grantID, &domain.CreateDraftRequest{
		Subject:     fmt.Sprintf("Signature Draft %d", time.Now().Unix()),
		Body:        "<p>Base draft body</p>",
		To:          []domain.EmailParticipant{{Email: draftRecipient}},
		SignatureID: created.ID,
	})
	require.NoError(t, err)
	require.NotNil(t, draft)
	t.Logf("Created signature-backed draft %s", draft.ID)
	defer func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cleanupCancel()
		_ = client.DeleteDraft(cleanupCtx, grantID, draft.ID)
	}()

	storedDraft, err := client.GetDraft(ctx, grantID, draft.ID)
	require.NoError(t, err)
	require.NotNil(t, storedDraft)
	require.True(t,
		strings.Contains(storedDraft.Body, marker) || strings.Contains(storedDraft.Body, updatedMarker),
		"expected draft body to contain signature marker, got %q", storedDraft.Body,
	)
}

func TestIntegration_SendMessageWithSignature(t *testing.T) {
	if os.Getenv("NYLAS_TEST_SEND_EMAIL") != "true" {
		t.Skip("Skipping send email test - set NYLAS_TEST_SEND_EMAIL=true to enable")
	}
	if os.Getenv("NYLAS_TEST_DELETE") != "true" {
		t.Skip("Skipping signature send test cleanup - set NYLAS_TEST_DELETE=true to enable")
	}

	client, grantID := getTestClient(t)
	ctx, cancel := createLongTestContext()
	defer cancel()

	testEmail := os.Getenv("NYLAS_TEST_EMAIL")
	if testEmail == "" {
		t.Skip("NYLAS_TEST_EMAIL not set")
	}

	signatures, err := client.GetSignatures(ctx, grantID)
	if err != nil {
		skipIfProviderNotSupported(t, err)
	}
	require.NoError(t, err)
	if len(signatures) >= 10 {
		t.Skip("Grant already has 10 signatures; skipping send test to avoid hard limit failures")
	}

	marker := fmt.Sprintf("Send Signature %d", time.Now().UnixNano())
	signature, err := client.CreateSignature(ctx, grantID, &domain.CreateSignatureRequest{
		Name: "Integration Send",
		Body: fmt.Sprintf("<p>%s</p>", marker),
	})
	require.NoError(t, err)
	defer func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cleanupCancel()
		_ = client.DeleteSignature(cleanupCtx, grantID, signature.ID)
	}()

	msg, err := client.SendMessage(ctx, grantID, &domain.SendMessageRequest{
		Subject:     fmt.Sprintf("Signature Send %d", time.Now().Unix()),
		Body:        "<p>Body</p>",
		To:          []domain.EmailParticipant{{Email: testEmail}},
		SignatureID: signature.ID,
	})
	require.NoError(t, err)
	require.NotEmpty(t, msg.ID)
}

func TestIntegration_SendDraftWithSignature(t *testing.T) {
	if os.Getenv("NYLAS_TEST_SEND_EMAIL") != "true" {
		t.Skip("Skipping send email test - set NYLAS_TEST_SEND_EMAIL=true to enable")
	}
	if os.Getenv("NYLAS_TEST_DELETE") != "true" {
		t.Skip("Skipping signature send test cleanup - set NYLAS_TEST_DELETE=true to enable")
	}

	client, grantID := getTestClient(t)
	ctx, cancel := createLongTestContext()
	defer cancel()

	testEmail := os.Getenv("NYLAS_TEST_EMAIL")
	if testEmail == "" {
		t.Skip("NYLAS_TEST_EMAIL not set")
	}

	signatures, err := client.GetSignatures(ctx, grantID)
	if err != nil {
		skipIfProviderNotSupported(t, err)
	}
	require.NoError(t, err)
	if len(signatures) >= 10 {
		t.Skip("Grant already has 10 signatures; skipping send-draft test to avoid hard limit failures")
	}

	signature, err := client.CreateSignature(ctx, grantID, &domain.CreateSignatureRequest{
		Name: "Integration Draft Send",
		Body: fmt.Sprintf("<p>Draft send signature %d</p>", time.Now().UnixNano()),
	})
	require.NoError(t, err)
	defer func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cleanupCancel()
		_ = client.DeleteSignature(cleanupCtx, grantID, signature.ID)
	}()

	draft, err := client.CreateDraft(ctx, grantID, &domain.CreateDraftRequest{
		Subject: fmt.Sprintf("Send Draft Signature %d", time.Now().Unix()),
		Body:    "<p>Draft Body</p>",
		To:      []domain.EmailParticipant{{Email: testEmail}},
	})
	require.NoError(t, err)
	defer func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cleanupCancel()
		_ = client.DeleteDraft(cleanupCtx, grantID, draft.ID)
	}()

	msg, err := client.SendDraft(ctx, grantID, draft.ID, &domain.SendDraftRequest{SignatureID: signature.ID})
	require.NoError(t, err)
	require.NotEmpty(t, msg.ID)
}

func stringPtr(value string) *string {
	return &value
}
