package email

import (
	"context"
	"fmt"
	"strings"

	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/domain"
	"github.com/nylas/cli/internal/ports"
)

func isManagedTransactionalGrant(grant *domain.Grant) bool {
	return grant != nil && grant.Provider == domain.ProviderNylas
}

func validateSendSignatureSupport(signatureID string, sign, encrypt bool, grant *domain.Grant) error {
	if signatureID == "" {
		return nil
	}
	if sign || encrypt {
		return common.NewUserError(
			"`--signature-id` is not supported with GPG signing or encryption",
			"Use standard JSON send mode without --sign/--encrypt, or add the signature HTML directly to the message body",
		)
	}
	if isManagedTransactionalGrant(grant) {
		return common.NewUserError(
			"`--signature-id` is not supported for managed transactional sends",
			"Managed Nylas grants use the domain-based transactional send endpoint, which does not accept signature_id",
		)
	}
	return nil
}

func validateManagedSecureSendSupport(sign, encrypt bool, grant *domain.Grant) error {
	if !sign && !encrypt {
		return nil
	}
	if !isManagedTransactionalGrant(grant) {
		return nil
	}
	return common.NewUserError(
		"`--sign` and `--encrypt` are not supported for managed transactional sends",
		"Managed Nylas grants use the domain-based transactional send endpoint, which does not support raw MIME GPG send",
	)
}

func validateSignatureSelection(
	ctx context.Context,
	client ports.NylasClient,
	grantID, signatureID string,
	grant *domain.Grant,
) ([]domain.Signature, error) {
	if signatureID == "" {
		return nil, nil
	}
	if grant == nil {
		var err error
		grant, err = client.GetGrant(ctx, grantID)
		if err != nil {
			return nil, common.WrapGetError("grant", err)
		}
	}
	if err := validateSendSignatureSupport(signatureID, false, false, grant); err != nil {
		return nil, err
	}

	signatures, err := client.GetSignatures(ctx, grantID)
	if err != nil {
		return nil, common.WrapGetError("signatures", err)
	}
	for _, signature := range signatures {
		if signature.ID == signatureID {
			return signatures, nil
		}
	}

	return nil, common.NewUserError(
		fmt.Sprintf("signature %q was not found for this grant", signatureID),
		"List available signatures with: nylas email signatures list [grant-id]",
	)
}

func validateDraftSendSignatureSelection(
	ctx context.Context,
	client ports.NylasClient,
	grantID string,
	draft *domain.Draft,
	signatureID string,
) error {
	signatures, err := validateSignatureSelection(ctx, client, grantID, signatureID, nil)
	if err != nil || draft == nil || signatureID == "" {
		return err
	}

	if existing := findStoredSignatureInBody(draft.Body, signatures); existing != nil {
		return common.NewUserError(
			"`--signature-id` cannot be used when the draft body already contains a stored signature",
			fmt.Sprintf(
				"Draft %q already contains stored signature %q. Send it without --signature-id, or update the draft body before switching signatures.",
				draft.ID,
				existing.Name,
			),
		)
	}

	return nil
}

func findStoredSignatureInBody(body string, signatures []domain.Signature) *domain.Signature {
	if strings.TrimSpace(body) == "" {
		return nil
	}

	normalizedBodyText := normalizeSignatureText(body)
	for i := range signatures {
		signature := &signatures[i]
		if strings.TrimSpace(signature.Body) == "" {
			continue
		}
		if strings.Contains(body, signature.Body) {
			return signature
		}

		normalizedSignatureText := normalizeSignatureText(signature.Body)
		if normalizedSignatureText != "" && strings.Contains(normalizedBodyText, normalizedSignatureText) {
			return signature
		}
	}

	return nil
}

func normalizeSignatureText(body string) string {
	text := common.StripHTML(body)
	return strings.Join(strings.Fields(strings.ToLower(text)), " ")
}

func sendDraftRequest(signatureID string) *domain.SendDraftRequest {
	if signatureID == "" {
		return nil
	}
	return &domain.SendDraftRequest{SignatureID: signatureID}
}

func signaturePreview(body string) string {
	if body == "" {
		return ""
	}
	return common.Truncate(common.StripHTML(body), 80)
}

func printSignature(signature *domain.Signature) {
	fmt.Println("════════════════════════════════════════════════════════════")
	_, _ = common.BoldWhite.Printf("Signature: %s\n", signature.Name)
	fmt.Println("════════════════════════════════════════════════════════════")
	fmt.Printf("ID:        %s\n", signature.ID)
	if !signature.CreatedAt.IsZero() {
		fmt.Printf("Created:   %s\n", signature.CreatedAt.Format(common.DisplayDateTime))
	}
	if !signature.UpdatedAt.IsZero() {
		fmt.Printf("Updated:   %s\n", signature.UpdatedAt.Format(common.DisplayDateTime))
	}
	if preview := signaturePreview(signature.Body); preview != "" {
		fmt.Printf("Preview:   %s\n", preview)
	}
	if signature.Body != "" {
		fmt.Println("\nBody:")
		fmt.Println("────────────────────────────────────────────────────────────")
		fmt.Println(signature.Body)
	}
}
