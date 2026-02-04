package email

import (
	"context"
	"fmt"
	"strings"
	"time"

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

// sendSecureEmail sends an email with GPG signing and/or encryption.
func sendSecureEmail(ctx context.Context, client ports.NylasClient, grantID string, req *domain.SendMessageRequest, gpgKeyID, recipientKeyID string, toContacts []domain.EmailParticipant, subject, body string, doSign, doEncrypt bool) (*domain.Message, error) {
	gpgSvc := gpg.NewService()

	// Step 1: Check GPG is available
	spinner := common.NewSpinner("Checking GPG...")
	spinner.Start()
	if err := gpgSvc.CheckGPGAvailable(ctx); err != nil {
		spinner.Stop()
		return nil, err
	}
	spinner.Stop()

	// Step 2: Resolve keys for signing and/or encryption
	var signerKeyID string
	var signingIdentity string
	var recipientKeyIDs []string

	if doSign {
		spinner = common.NewSpinner("Getting GPG signing key...")
		spinner.Start()
		signerKeyID, signingIdentity = resolveSigningKey(ctx, gpgSvc, gpgKeyID, req)
		if signerKeyID == "" {
			spinner.Stop()
			return nil, fmt.Errorf("could not determine signing key")
		}
		spinner.Stop()
	}

	if doEncrypt {
		spinner = common.NewSpinner("Resolving recipient public keys...")
		spinner.Start()
		var err error
		recipientKeyIDs, err = resolveRecipientKeys(ctx, gpgSvc, recipientKeyID, toContacts, req.Cc, req.Bcc)
		if err != nil {
			spinner.Stop()
			return nil, err
		}
		spinner.Stop()
	}

	// Determine content type
	contentType := "text/plain"
	if strings.Contains(strings.ToLower(body), "<html") {
		contentType = "text/html"
	}

	mimeBuilder := mime.NewBuilder()
	var rawMIME []byte

	// Step 3: Build and send based on mode
	if doSign && doEncrypt {
		// Sign+Encrypt: Maximum security
		rawMIME = buildSignedEncryptedMessage(ctx, gpgSvc, mimeBuilder, req, toContacts, subject, body, contentType, signerKeyID, recipientKeyIDs)
	} else if doEncrypt {
		// Encrypt only
		rawMIME = buildEncryptedMessage(ctx, gpgSvc, mimeBuilder, req, toContacts, subject, body, contentType, recipientKeyIDs)
	} else if doSign {
		// Sign only (original behavior)
		rawMIME = buildSignedMessage(ctx, gpgSvc, mimeBuilder, req, toContacts, subject, body, contentType, signerKeyID, signingIdentity)
	}

	if rawMIME == nil {
		return nil, fmt.Errorf("failed to build secure message")
	}

	// Step 4: Send raw MIME message
	var sendingMsg string
	if doSign && doEncrypt {
		sendingMsg = "Sending signed and encrypted email..."
	} else if doEncrypt {
		sendingMsg = "Sending encrypted email..."
	} else {
		sendingMsg = "Sending signed email..."
	}

	spinner = common.NewSpinner(sendingMsg)
	spinner.Start()

	msg, err := client.SendRawMessage(ctx, grantID, rawMIME)
	spinner.Stop()

	if err != nil {
		return nil, err
	}

	return msg, nil
}

// resolveSigningKey determines the signing key to use.
func resolveSigningKey(ctx context.Context, gpgSvc gpg.Service, explicitKeyID string, req *domain.SendMessageRequest) (keyID, identity string) {
	if explicitKeyID != "" {
		return explicitKeyID, explicitKeyID
	}

	// Check Nylas config for default key
	configStore := config.NewDefaultFileStore()
	cfg, err := configStore.Load()
	if err == nil && cfg != nil && cfg.GPG != nil && cfg.GPG.DefaultKey != "" {
		return cfg.GPG.DefaultKey, cfg.GPG.DefaultKey
	}

	// Use From email address to find key
	if len(req.From) > 0 && req.From[0].Email != "" {
		fromEmail := req.From[0].Email
		key, err := gpgSvc.FindKeyByEmail(ctx, fromEmail)
		if err == nil {
			return key.KeyID, fromEmail
		}
	}

	// Fallback to default key from git config
	key, err := gpgSvc.GetDefaultSigningKey(ctx)
	if err == nil {
		return key.KeyID, key.KeyID
	}

	return "", ""
}

// resolveRecipientKeys determines the encryption keys for all recipients.
func resolveRecipientKeys(ctx context.Context, gpgSvc gpg.Service, explicitKeyID string, to, cc, bcc []domain.EmailParticipant) ([]string, error) {
	// If explicit key provided, use it
	if explicitKeyID != "" {
		return []string{explicitKeyID}, nil
	}

	// Collect all recipient emails
	var recipients []domain.EmailParticipant
	recipients = append(recipients, to...)
	recipients = append(recipients, cc...)
	recipients = append(recipients, bcc...)

	if len(recipients) == 0 {
		return nil, fmt.Errorf("no recipients specified for encryption")
	}

	// Find public keys for each recipient (auto-fetch from key servers if needed)
	keyIDs := make([]string, 0, len(recipients))
	seen := make(map[string]bool)

	for _, recipient := range recipients {
		if seen[recipient.Email] {
			continue
		}
		seen[recipient.Email] = true

		key, err := gpgSvc.FindPublicKeyByEmail(ctx, recipient.Email)
		if err != nil {
			return nil, fmt.Errorf("could not find public key for %s: %w\n\nTip: Ask the recipient to upload their key to keys.openpgp.org", recipient.Email, err)
		}

		// Check for expired key (key.Expires is checked in FindPublicKeyByEmail)
		// Additional check here for recently expired keys
		if key.Expires != nil && key.Expires.Before(time.Now()) {
			return nil, fmt.Errorf("public key for %s has expired on %s", recipient.Email, key.Expires.Format("2006-01-02"))
		}

		keyIDs = append(keyIDs, key.KeyID)
	}

	return keyIDs, nil
}

// buildSignedMessage builds a signed-only PGP/MIME message.
func buildSignedMessage(ctx context.Context, gpgSvc gpg.Service, mimeBuilder mime.Builder, req *domain.SendMessageRequest, toContacts []domain.EmailParticipant, subject, body, contentType, signerKeyID, signingIdentity string) []byte {
	spinner := common.NewSpinner(fmt.Sprintf("Signing email with GPG identity: %s...", signingIdentity))
	spinner.Start()
	defer spinner.Stop()

	// Prepare the MIME content part to be signed
	dataToSign, err := mimeBuilder.PrepareContentToSign(body, contentType, req.Attachments)
	if err != nil {
		return nil
	}

	// Extract sender email for the Signer's User ID subpacket
	var senderEmail string
	if len(req.From) > 0 && req.From[0].Email != "" {
		senderEmail = req.From[0].Email
	}

	// Sign the MIME content part
	signResult, err := gpgSvc.SignData(ctx, signerKeyID, dataToSign, senderEmail)
	if err != nil {
		return nil
	}

	// Build PGP/MIME signed message
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
		PreparedContent: dataToSign,
		Attachments:     req.Attachments,
	}

	rawMIME, err := mimeBuilder.BuildSignedMessage(mimeReq)
	if err != nil {
		return nil
	}

	return rawMIME
}

// buildEncryptedMessage builds an encrypted-only PGP/MIME message.
func buildEncryptedMessage(ctx context.Context, gpgSvc gpg.Service, mimeBuilder mime.Builder, req *domain.SendMessageRequest, toContacts []domain.EmailParticipant, subject, body, contentType string, recipientKeyIDs []string) []byte {
	spinner := common.NewSpinner("Encrypting email...")
	spinner.Start()
	defer spinner.Stop()

	// Prepare the content to encrypt
	dataToEncrypt, err := mimeBuilder.PrepareContentToEncrypt(body, contentType, req.Attachments)
	if err != nil {
		return nil
	}

	// Encrypt the content
	encryptResult, err := gpgSvc.EncryptData(ctx, recipientKeyIDs, dataToEncrypt)
	if err != nil {
		return nil
	}

	// Build PGP/MIME encrypted message
	mimeReq := &mime.EncryptedMessageRequest{
		From:       req.From,
		To:         toContacts,
		Cc:         req.Cc,
		Bcc:        req.Bcc,
		ReplyTo:    req.ReplyTo,
		Subject:    subject,
		Ciphertext: encryptResult.Ciphertext,
	}

	rawMIME, err := mimeBuilder.BuildEncryptedMessage(mimeReq)
	if err != nil {
		return nil
	}

	return rawMIME
}

// buildSignedEncryptedMessage builds a signed AND encrypted PGP/MIME message.
// Order: Sign first, then encrypt (per OpenPGP best practice).
func buildSignedEncryptedMessage(ctx context.Context, gpgSvc gpg.Service, mimeBuilder mime.Builder, req *domain.SendMessageRequest, toContacts []domain.EmailParticipant, subject, body, contentType, signerKeyID string, recipientKeyIDs []string) []byte {
	spinner := common.NewSpinner("Signing and encrypting email...")
	spinner.Start()
	defer spinner.Stop()

	// Prepare the content
	dataToProcess, err := mimeBuilder.PrepareContentToEncrypt(body, contentType, req.Attachments)
	if err != nil {
		return nil
	}

	// Extract sender email for signing
	var senderEmail string
	if len(req.From) > 0 && req.From[0].Email != "" {
		senderEmail = req.From[0].Email
	}

	// Sign AND encrypt in one GPG operation
	encryptResult, err := gpgSvc.SignAndEncryptData(ctx, signerKeyID, recipientKeyIDs, dataToProcess, senderEmail)
	if err != nil {
		return nil
	}

	// Build PGP/MIME encrypted message (signature is inside the encrypted payload)
	mimeReq := &mime.EncryptedMessageRequest{
		From:       req.From,
		To:         toContacts,
		Cc:         req.Cc,
		Bcc:        req.Bcc,
		ReplyTo:    req.ReplyTo,
		Subject:    subject,
		Ciphertext: encryptResult.Ciphertext,
	}

	rawMIME, err := mimeBuilder.BuildEncryptedMessage(mimeReq)
	if err != nil {
		return nil
	}

	return rawMIME
}

// sendSignedEmail signs an email with GPG and sends it as raw MIME.
// Deprecated: Use sendSecureEmail with doSign=true instead.
func sendSignedEmail(ctx context.Context, client ports.NylasClient, grantID string, req *domain.SendMessageRequest, gpgKeyID string, toContacts []domain.EmailParticipant, subject, body string) (*domain.Message, error) {
	return sendSecureEmail(ctx, client, grantID, req, gpgKeyID, "", toContacts, subject, body, true, false)
}
