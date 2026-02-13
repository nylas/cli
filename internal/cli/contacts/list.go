package contacts

import (
	"context"
	"fmt"

	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/domain"
	"github.com/nylas/cli/internal/ports"
	"github.com/spf13/cobra"
)

func newListCmd() *cobra.Command {
	var (
		limit  int
		email  string
		source string
		showID bool
	)

	cmd := &cobra.Command{
		Use:     "list [grant-id]",
		Aliases: []string{"ls"},
		Short:   "List contacts",
		Long:    "List all contacts for the specified grant or default account.",
		Args:    cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Auto-paginate when limit exceeds API maximum
			maxItems := 0
			if limit > common.MaxAPILimit {
				maxItems = limit
				limit = common.MaxAPILimit
			}

			// Check if we should use structured output (JSON/YAML/quiet)
			if common.IsStructuredOutput(cmd) {
				_, err := common.WithClient(args, func(ctx context.Context, client ports.NylasClient, grantID string) (struct{}, error) {
					params := &domain.ContactQueryParams{
						Limit:  limit,
						Email:  email,
						Source: source,
					}

					contacts, err := fetchContacts(ctx, client, grantID, params, maxItems)
					if err != nil {
						return struct{}{}, common.WrapListError("contacts", err)
					}

					out := common.GetOutputWriter(cmd)
					return struct{}{}, out.Write(contacts)
				})
				return err
			}

			// Traditional table output
			_, err := common.WithClient(args, func(ctx context.Context, client ports.NylasClient, grantID string) (struct{}, error) {
				params := &domain.ContactQueryParams{
					Limit:  limit,
					Email:  email,
					Source: source,
				}

				contacts, err := fetchContacts(ctx, client, grantID, params, maxItems)
				if err != nil {
					return struct{}{}, common.WrapListError("contacts", err)
				}

				if len(contacts) == 0 {
					common.PrintEmptyState("contacts")
					return struct{}{}, nil
				}

				fmt.Printf("Found %d contact(s):\n\n", len(contacts))

				var table *common.Table
				if showID {
					table = common.NewTable("ID", "NAME", "EMAIL", "PHONE", "COMPANY")
				} else {
					table = common.NewTable("NAME", "EMAIL", "PHONE", "COMPANY")
				}
				for _, contact := range contacts {
					name := contact.DisplayName()
					emailAddr := contact.PrimaryEmail()
					phone := contact.PrimaryPhone()
					company := contact.CompanyName
					if contact.JobTitle != "" && company != "" {
						company = fmt.Sprintf("%s - %s", contact.JobTitle, company)
					} else if contact.JobTitle != "" {
						company = contact.JobTitle
					}

					if showID {
						table.AddRow(
							common.Dim.Sprint(contact.ID),
							common.Cyan.Sprint(name),
							emailAddr,
							phone,
							common.Dim.Sprint(company),
						)
					} else {
						table.AddRow(
							common.Cyan.Sprint(name),
							emailAddr,
							phone,
							common.Dim.Sprint(company),
						)
					}
				}
				table.Render()

				return struct{}{}, nil
			})
			return err
		},
	}

	cmd.Flags().IntVarP(&limit, "limit", "n", 50, "Maximum number of contacts to show (auto-paginates if >200)")
	cmd.Flags().StringVarP(&email, "email", "e", "", "Filter by email address")
	cmd.Flags().StringVarP(&source, "source", "s", "", "Filter by source (address_book, inbox, domain)")
	cmd.Flags().BoolVar(&showID, "id", false, "Show contact IDs")

	return cmd
}

// fetchContacts retrieves contacts, using pagination when maxItems > 0.
func fetchContacts(ctx context.Context, client ports.NylasClient, grantID string, params *domain.ContactQueryParams, maxItems int) ([]domain.Contact, error) {
	if maxItems > 0 {
		pageSize := min(params.Limit, common.MaxAPILimit)
		params.Limit = pageSize

		fetcher := func(ctx context.Context, cursor string) (common.PageResult[domain.Contact], error) {
			params.PageToken = cursor
			resp, err := client.GetContactsWithCursor(ctx, grantID, params)
			if err != nil {
				return common.PageResult[domain.Contact]{}, err
			}
			return common.PageResult[domain.Contact]{
				Data:       resp.Data,
				NextCursor: resp.Pagination.NextCursor,
			}, nil
		}

		config := common.DefaultPaginationConfig()
		config.PageSize = pageSize
		config.MaxItems = maxItems

		return common.FetchAllPages(ctx, config, fetcher)
	}

	return client.GetContacts(ctx, grantID, params)
}
