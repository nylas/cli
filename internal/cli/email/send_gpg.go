package email

import (
	"context"
	"fmt"
	"strings"

	"github.com/nylas/cli/internal/adapters/config"
	"github.com/nylas/cli/internal/adapters/gpg"
	"github.com/nylas/cli/internal/adapters/mime"
	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/domain"
	"github.com/nylas/cli/internal/ports"
)

// handleListGPGKeys lists available GPG signing keys.
func handleListGPGKeys(ctx context.Context) error {
	gpgSvc := gpg.NewService()

	// Check GPG is available
	if err := gpgSvc.CheckGPGAvailable(ctx); err != nil {
		return err
	}

	// List keys
	keys, err := gpgSvc.ListSigningKeys(ctx)
	if err != nil {
		return err
	}

	if len(keys) == 0 {
		fmt.Println("No GPG signing keys found.")
		fmt.Println("\nTo generate a new GPG key, run: gpg --gen-key")
		return nil
	}

	fmt.Printf("Available GPG signing keys (%d):\n\n", len(keys))
	for i, key := range keys {
		fmt.Printf("%d. Key ID: %s\n", i+1, key.KeyID)
		if key.Fingerprint != "" {
			fmt.Printf("   Fingerprint: %s\n", key.Fingerprint)
		}
		for _, uid := range key.UIDs {
			fmt.Printf("   UID: %s\n", uid)
		}
		if key.Expires != nil {
			fmt.Printf("   Expires: %s\n", key.Expires.Format("2006-01-02"))
		} else {
			fmt.Printf("   Expires: Never\n")
		}
		fmt.Println()
	}

	// Show default keys if configured
	fmt.Println("Default signing keys:")

	// Check Nylas config
	configStore := config.NewDefaultFileStore()
	cfg, err := configStore.Load()
	if err == nil && cfg != nil && cfg.GPG != nil && cfg.GPG.DefaultKey != "" {
		fmt.Printf("  From Nylas config: %s\n", cfg.GPG.DefaultKey)
	}

	// Check git config
	defaultKey, err := gpgSvc.GetDefaultSigningKey(ctx)
	if err == nil && defaultKey != nil {
		fmt.Printf("  From git config: %s\n", defaultKey.KeyID)
	}

	return nil
}

// sendSignedEmail signs an email with GPG and sends it as raw MIME.
func sendSignedEmail(ctx context.Context, client ports.NylasClient, grantID string, req *domain.SendMessageRequest, gpgKeyID string, toContacts []domain.EmailParticipant, subject, body string) (*domain.Message, error) {
	gpgSvc := gpg.NewService()

	// Step 1: Check GPG is available
	spinner := common.NewSpinner("Checking GPG...")
	spinner.Start()
	if err := gpgSvc.CheckGPGAvailable(ctx); err != nil {
		spinner.Stop()
		return nil, err
	}
	spinner.Stop()

	// Step 2: Get signing key/identity (Priority: CLI flag > Config > From email > Git config)
	spinner = common.NewSpinner("Getting GPG signing key...")
	spinner.Start()
	var keyID string
	var signingIdentity string

	if gpgKeyID != "" {
		// Priority 1: Explicit key ID provided via --gpg-key flag
		keyID = gpgKeyID
		signingIdentity = gpgKeyID
	} else {
		// Priority 2: Check Nylas config for default key
		configStore := config.NewDefaultFileStore()
		cfg, err := configStore.Load()
		if err == nil && cfg != nil && cfg.GPG != nil && cfg.GPG.DefaultKey != "" {
			keyID = cfg.GPG.DefaultKey
			signingIdentity = keyID
		} else if len(req.From) > 0 && req.From[0].Email != "" {
			// Priority 3: Use From email address to find key
			// IMPORTANT: We must use the actual key ID for --local-user, not the email.
			// GPG's --sender option only works correctly when --local-user is a key ID.
			fromEmail := req.From[0].Email
			signingIdentity = fromEmail

			// Look up the actual key ID for this email
			key, err := gpgSvc.FindKeyByEmail(ctx, fromEmail)
			if err != nil {
				spinner.Stop()
				return nil, fmt.Errorf("no GPG key found for %s: %w", fromEmail, err)
			}
			keyID = key.KeyID
		} else {
			// Priority 4: Fallback to default key from git config
			key, err := gpgSvc.GetDefaultSigningKey(ctx)
			if err != nil {
				spinner.Stop()
				return nil, err
			}
			keyID = key.KeyID
			signingIdentity = keyID
		}
	}
	spinner.Stop()

	// Step 3: Build MIME content to sign
	spinner = common.NewSpinner(fmt.Sprintf("Signing email with GPG identity: %s...", signingIdentity))
	spinner.Start()

	// Determine content type
	contentType := "text/plain"
	if strings.Contains(strings.ToLower(body), "<html") {
		contentType = "text/html"
	}

	// Prepare the MIME content part to be signed (includes headers)
	// PGP/MIME requires signing the entire MIME part, not just the body text
	mimeBuilder := mime.NewBuilder()
	dataToSign, err := mimeBuilder.PrepareContentToSign(body, contentType, req.Attachments)
	if err != nil {
		spinner.Stop()
		return nil, fmt.Errorf("failed to prepare content for signing: %w", err)
	}

	// Extract sender email for the Signer's User ID subpacket
	// This ensures the correct email appears in the signature when
	// the key has multiple UIDs
	var senderEmail string
	if len(req.From) > 0 && req.From[0].Email != "" {
		senderEmail = req.From[0].Email
	}

	// Sign the MIME content part with sender email for proper UID embedding
	signResult, err := gpgSvc.SignData(ctx, keyID, dataToSign, senderEmail)
	if err != nil {
		spinner.Stop()
		return nil, err
	}
	spinner.Stop()

	// Step 4: Build PGP/MIME message
	spinner = common.NewSpinner("Building PGP/MIME message...")
	spinner.Start()

	// Use the same MIME builder instance to ensure consistency
	mimeReq := &mime.SignedMessageRequest{
		From:            req.From,
		To:              toContacts,
		Cc:              req.Cc,
		Bcc:             req.Bcc,
		ReplyTo:         req.ReplyTo,
		Subject:         subject,
		Body:            body,
		ContentType:     contentType,
		Signature:       signResult.Signature,
		HashAlgo:        signResult.HashAlgo,
		PreparedContent: dataToSign, // Use the exact content that was signed
		Attachments:     req.Attachments,
	}

	rawMIME, err := mimeBuilder.BuildSignedMessage(mimeReq)
	if err != nil {
		spinner.Stop()
		return nil, fmt.Errorf("failed to build PGP/MIME message: %w", err)
	}
	spinner.Stop()

	// Step 5: Send raw MIME message
	spinner = common.NewSpinner("Sending signed email...")
	spinner.Start()

	msg, err := client.SendRawMessage(ctx, grantID, rawMIME)
	spinner.Stop()

	if err != nil {
		return nil, err
	}

	return msg, nil
}
