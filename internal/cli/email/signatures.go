package email

import (
	"context"
	"fmt"

	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/domain"
	"github.com/nylas/cli/internal/ports"
	"github.com/spf13/cobra"
)

func newSignaturesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "signatures",
		Short: "Manage stored email signatures",
		Long:  "List, show, create, update, and delete stored email signatures for a grant.",
	}

	cmd.AddCommand(newSignaturesListCmd())
	cmd.AddCommand(newSignaturesShowCmd())
	cmd.AddCommand(newSignaturesCreateCmd())
	cmd.AddCommand(newSignaturesUpdateCmd())
	cmd.AddCommand(newSignaturesDeleteCmd())

	return cmd
}

func newSignaturesListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list [grant-id]",
		Short: "List stored signatures",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			_, err := common.WithClient(args, func(ctx context.Context, client ports.NylasClient, grantID string) (struct{}, error) {
				signatures, err := client.GetSignatures(ctx, grantID)
				if err != nil {
					return struct{}{}, common.WrapGetError("signatures", err)
				}
				if common.IsStructuredOutput(cmd) {
					out := common.GetOutputWriter(cmd)
					return struct{}{}, out.Write(signatures)
				}
				if len(signatures) == 0 {
					common.PrintEmptyState("signatures")
					return struct{}{}, nil
				}
				out := common.GetOutputWriter(cmd)
				return struct{}{}, out.WriteList(signatures, []ports.Column{
					{Header: "ID", Field: "ID", Width: 20},
					{Header: "Name", Field: "Name", Width: 24},
					{Header: "Updated", Field: "UpdatedAt", Width: 0},
				})
			})
			return err
		},
	}

	return cmd
}

func newSignaturesShowCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show <signature-id> [grant-id]",
		Short: "Show signature details",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			signatureID := args[0]
			remainingArgs := args[1:]

			_, err := common.WithClient(remainingArgs, func(ctx context.Context, client ports.NylasClient, grantID string) (struct{}, error) {
				signature, err := client.GetSignature(ctx, grantID, signatureID)
				if err != nil {
					return struct{}{}, common.WrapGetError("signature", err)
				}
				if common.IsStructuredOutput(cmd) {
					out := common.GetOutputWriter(cmd)
					return struct{}{}, out.Write(signature)
				}
				printSignature(signature)
				return struct{}{}, nil
			})
			return err
		},
	}

	return cmd
}

func newSignaturesCreateCmd() *cobra.Command {
	var name string
	var body string
	var bodyFile string

	cmd := &cobra.Command{
		Use:   "create [grant-id]",
		Short: "Create a stored signature",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := common.ValidateRequiredFlag("--name", name); err != nil {
				return err
			}
			signatureBody, err := common.ReadStringOrFile("body", body, bodyFile, true)
			if err != nil {
				return err
			}

			_, err = common.WithClient(args, func(ctx context.Context, client ports.NylasClient, grantID string) (struct{}, error) {
				signature, err := client.CreateSignature(ctx, grantID, &domain.CreateSignatureRequest{
					Name: name,
					Body: signatureBody,
				})
				if err != nil {
					return struct{}{}, common.WrapCreateError("signature", err)
				}
				if common.IsStructuredOutput(cmd) {
					out := common.GetOutputWriter(cmd)
					return struct{}{}, out.Write(signature)
				}
				printSuccess("Signature created successfully!")
				fmt.Printf("  ID:      %s\n", signature.ID)
				fmt.Printf("  Name:    %s\n", signature.Name)
				if preview := signaturePreview(signature.Body); preview != "" {
					fmt.Printf("  Preview: %s\n", preview)
				}
				return struct{}{}, nil
			})
			return err
		},
	}

	cmd.Flags().StringVarP(&name, "name", "n", "", "Signature name (required)")
	cmd.Flags().StringVarP(&body, "body", "b", "", "Signature HTML body")
	cmd.Flags().StringVar(&bodyFile, "body-file", "", "Path to a file containing the signature HTML body")

	return cmd
}

func newSignaturesUpdateCmd() *cobra.Command {
	var name string
	var body string
	var bodyFile string

	cmd := &cobra.Command{
		Use:   "update <signature-id> [grant-id]",
		Short: "Update a stored signature",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			signatureID := args[0]
			remainingArgs := args[1:]

			signatureBody, err := common.ReadStringOrFile("body", body, bodyFile, false)
			if err != nil {
				return err
			}
			if err := common.ValidateAtLeastOne("signature update field", name, signatureBody); err != nil {
				return err
			}

			_, err = common.WithClient(remainingArgs, func(ctx context.Context, client ports.NylasClient, grantID string) (struct{}, error) {
				signature, err := client.UpdateSignature(ctx, grantID, signatureID, &domain.UpdateSignatureRequest{
					Name: optionalString(name),
					Body: optionalString(signatureBody),
				})
				if err != nil {
					return struct{}{}, common.WrapUpdateError("signature", err)
				}
				if common.IsStructuredOutput(cmd) {
					out := common.GetOutputWriter(cmd)
					return struct{}{}, out.Write(signature)
				}
				printSuccess("Signature updated successfully!")
				fmt.Printf("  ID:      %s\n", signature.ID)
				fmt.Printf("  Name:    %s\n", signature.Name)
				if preview := signaturePreview(signature.Body); preview != "" {
					fmt.Printf("  Preview: %s\n", preview)
				}
				return struct{}{}, nil
			})
			return err
		},
	}

	cmd.Flags().StringVarP(&name, "name", "n", "", "Updated signature name")
	cmd.Flags().StringVarP(&body, "body", "b", "", "Updated signature HTML body")
	cmd.Flags().StringVar(&bodyFile, "body-file", "", "Path to a file containing the updated signature HTML body")

	return cmd
}

func newSignaturesDeleteCmd() *cobra.Command {
	var yes bool

	cmd := &cobra.Command{
		Use:   "delete <signature-id> [grant-id]",
		Short: "Delete a stored signature",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			signatureID := args[0]
			remainingArgs := args[1:]

			_, err := common.WithClient(remainingArgs, func(ctx context.Context, client ports.NylasClient, grantID string) (struct{}, error) {
				signature, err := client.GetSignature(ctx, grantID, signatureID)
				if err != nil {
					return struct{}{}, common.WrapGetError("signature", err)
				}
				if !yes {
					fmt.Printf("Delete signature %q (%s)? [y/N]: ", signature.Name, signature.ID)
					var confirm string
					_, _ = fmt.Scanln(&confirm)
					if confirm != "y" && confirm != "Y" && confirm != "yes" {
						fmt.Println("Cancelled.")
						return struct{}{}, nil
					}
				}
				if err := client.DeleteSignature(ctx, grantID, signatureID); err != nil {
					return struct{}{}, common.WrapDeleteError("signature", err)
				}
				printSuccess("Signature deleted successfully!")
				return struct{}{}, nil
			})
			return err
		},
	}

	common.AddYesFlag(cmd, &yes)

	return cmd
}

func optionalString(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}
