package dashboard

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/spf13/cobra"

	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/domain"
	"github.com/nylas/cli/internal/ports"
)

const (
	defaultDomainListLimit   = 100
	maxDomainResolutionPages = 100
)

var (
	domainLabelPattern = regexp.MustCompile(`^[A-Za-z0-9-]+$`)
	domainInfoTypes    = []string{"ownership", "mx", "spf", "feedback", "dkim", "dmarc", "arc"}
)

func newDomainsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "domains",
		Short: "Manage inbox and Agent Account domains",
		Long: `Manage domains used by Nylas Inbox and Agent Accounts.

Domains are registered in your active Dashboard organization. After creating a
custom domain, configure the DNS records printed by the dns command, then run
verification.`,
	}

	cmd.AddCommand(newDomainsListCmd())
	cmd.AddCommand(newDomainsCheckCmd())
	cmd.AddCommand(newDomainsCreateCmd())
	cmd.AddCommand(newDomainsShowCmd())
	cmd.AddCommand(newDomainsDNSCmd())
	cmd.AddCommand(newDomainsVerifyCmd())
	cmd.AddCommand(newDomainsUpdateCmd())
	cmd.AddCommand(newDomainsDeleteCmd())

	return cmd
}

type domainRow struct {
	ID        string `json:"id"`
	Domain    string `json:"domain"`
	Name      string `json:"name"`
	Region    string `json:"region"`
	Branded   bool   `json:"branded"`
	Ownership string `json:"ownership"`
	MX        string `json:"mx"`
	SPF       string `json:"spf"`
	DKIM      string `json:"dkim"`
	DMARC     string `json:"dmarc"`
	ARC       string `json:"arc"`
	Feedback  string `json:"feedback"`
}

type domainAvailabilityRow struct {
	Domain        string `json:"domain"`
	Available     bool   `json:"available"`
	ConflictsWith string `json:"conflicts_with,omitempty"`
}

type domainListResult struct {
	Domains    []domainRow `json:"domains"`
	NextCursor string      `json:"next_cursor,omitempty"`
}

type domainDNSRow struct {
	Type   string `json:"type"`
	Host   string `json:"host"`
	Record string `json:"record"`
	Value  string `json:"value"`
	Status string `json:"status"`
}

type domainVerificationRow struct {
	Type    string `json:"type"`
	Status  string `json:"status"`
	Message string `json:"message"`
	Host    string `json:"host,omitempty"`
	Record  string `json:"record,omitempty"`
	Value   string `json:"value,omitempty"`
}

var domainColumns = []ports.Column{
	{Header: "ID", Field: "ID"},
	{Header: "DOMAIN", Field: "Domain"},
	{Header: "NAME", Field: "Name"},
	{Header: "REGION", Field: "Region"},
	{Header: "OWN", Field: "Ownership"},
	{Header: "MX", Field: "MX"},
	{Header: "SPF", Field: "SPF"},
	{Header: "DKIM", Field: "DKIM"},
}

var domainDNSColumns = []ports.Column{
	{Header: "TYPE", Field: "Type"},
	{Header: "HOST", Field: "Host"},
	{Header: "RECORD", Field: "Record"},
	{Header: "VALUE", Field: "Value", Width: -1},
	{Header: "STATUS", Field: "Status"},
}

var domainVerificationColumns = []ports.Column{
	{Header: "TYPE", Field: "Type"},
	{Header: "STATUS", Field: "Status"},
	{Header: "MESSAGE", Field: "Message", Width: -1},
}

func newDomainsListCmd() *cobra.Command {
	var (
		limit     int
		pageToken string
	)

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List domains",
		RunE: func(cmd *cobra.Command, args []string) error {
			limit = common.NormalizePageSize(limit)

			domainSvc, err := createDomainServiceFn()
			if err != nil {
				return wrapDashboardError(err)
			}

			ctx, cancel := common.CreateContext()
			defer cancel()

			page, err := domainSvc.ListDomains(ctx, limit, pageToken)
			if err != nil {
				return wrapDashboardError(err)
			}

			return writeDomainListResult(cmd, page)
		},
	}

	cmd.Flags().IntVar(&limit, "limit", defaultDomainListLimit, "Maximum domains to return")
	cmd.Flags().StringVar(&pageToken, "page-token", "", "Page token for pagination")

	return cmd
}

func newDomainsCheckCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "check [domain]",
		Short: "Check whether a domain is already registered in your organization",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			domainAddress := normalizeDomainAddress(args[0])
			if err := validateDomainAddress(domainAddress); err != nil {
				return err
			}

			domainSvc, err := createDomainServiceFn()
			if err != nil {
				return wrapDashboardError(err)
			}

			ctx, cancel := common.CreateContext()
			defer cancel()

			result, err := domainSvc.CheckAvailability(ctx, domainAddress)
			if err != nil {
				return wrapDashboardError(err)
			}

			row := domainAvailabilityRow{
				Domain:    result.DomainAddress,
				Available: result.Available,
			}
			if result.ConflictsWith != nil {
				row.ConflictsWith = *result.ConflictsWith
			}
			return common.GetOutputWriter(cmd).Write(row)
		},
	}

	return cmd
}

func newDomainsCreateCmd() *cobra.Command {
	var (
		domainFlag string
		name       string
		region     string
	)

	cmd := &cobra.Command{
		Use:     "create [domain]",
		Aliases: []string{"register"},
		Short:   "Register a domain",
		Example: `  nylas dashboard domains create example.com --region us
  nylas dashboard domains register mail.example.com --name "Mail" --region eu`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			domainAddress, err := resolveDomainAddress(args, domainFlag)
			if err != nil {
				return err
			}
			if err := validateRegion(region); err != nil {
				return err
			}
			if name == "" {
				name = domainAddress
			}

			domainSvc, err := createDomainServiceFn()
			if err != nil {
				return wrapDashboardError(err)
			}

			ctx, cancel := common.CreateContext()
			defer cancel()

			created, err := domainSvc.CreateDomain(ctx, domain.DashboardCreateInboxDomainInput{
				Name:          name,
				DomainAddress: domainAddress,
				Region:        region,
			})
			if err != nil {
				return wrapDashboardError(err)
			}
			if err := validateCreatedDomain(created); err != nil {
				return err
			}

			if common.IsStructuredOutput(cmd) {
				return common.GetOutputWriter(cmd).Write(toDomainRow(*created))
			}

			common.PrintSuccess("Domain registered")
			if err := common.GetOutputWriter(cmd).Write(toDomainRow(*created)); err != nil {
				return err
			}
			fmt.Printf("\nNext: nylas dashboard domains dns %s --region %s\n", created.ID, created.Region)
			return nil
		},
	}

	cmd.Flags().StringVar(&domainFlag, "domain", "", "Domain address to register")
	cmd.Flags().StringVarP(&name, "name", "n", "", "Display name (default: domain)")
	cmd.Flags().StringVarP(&region, "region", "r", "", "Region (required: us or eu)")

	return cmd
}

func newDomainsShowCmd() *cobra.Command {
	var region string

	cmd := &cobra.Command{
		Use:   "show [domain-id-or-address]",
		Short: "Show a domain",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			domainSvc, err := createDomainServiceFn()
			if err != nil {
				return wrapDashboardError(err)
			}

			ctx, cancel := common.CreateContext()
			defer cancel()

			ref, err := resolveExistingDomainRef(ctx, domainSvc, args[0], region)
			if err != nil {
				return wrapDashboardError(err)
			}

			found, err := domainSvc.GetDomain(ctx, ref.IDOrAddress, ref.Region)
			if err != nil {
				return wrapDashboardError(err)
			}
			if found == nil || found.ID == "" {
				return dashboardError("domain not found", "Check the domain ID/address and region")
			}
			return common.GetOutputWriter(cmd).Write(toDomainRow(*found))
		},
	}

	cmd.Flags().StringVarP(&region, "region", "r", "", "Region (us or eu; inferred for existing domains when omitted)")

	return cmd
}

func newDomainsDNSCmd() *cobra.Command {
	var (
		region string
		types  []string
	)

	cmd := &cobra.Command{
		Use:   "dns [domain-id-or-address]",
		Short: "Show DNS records required for verification",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			types = normalizeVerificationTypes(types)
			if err := validateVerificationTypes(types); err != nil {
				return err
			}

			domainSvc, err := createDomainServiceFn()
			if err != nil {
				return wrapDashboardError(err)
			}

			ctx, cancel := common.CreateContext()
			defer cancel()

			ref, err := resolveExistingDomainRef(ctx, domainSvc, args[0], region)
			if err != nil {
				return wrapDashboardError(err)
			}

			rows := make([]domainDNSRow, 0, len(types))
			for _, typ := range types {
				info, err := domainSvc.GetDomainInfo(ctx, ref.IDOrAddress, ref.Region, typ)
				if err != nil {
					return wrapDashboardError(err)
				}
				rows = append(rows, toDNSRow(typ, info))
			}
			return common.WriteListWithColumns(cmd, rows, domainDNSColumns)
		},
	}

	cmd.Flags().StringVarP(&region, "region", "r", "", "Region (us or eu; inferred for existing domains when omitted)")
	cmd.Flags().StringSliceVar(&types, "type", nil, "Verification type to show (repeatable: ownership,mx,spf,feedback,dkim,dmarc,arc)")

	return cmd
}

func newDomainsVerifyCmd() *cobra.Command {
	var (
		region string
		types  []string
		all    bool
	)

	cmd := &cobra.Command{
		Use:   "verify [domain-id-or-address]",
		Short: "Verify domain DNS records",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if all {
				types = domainInfoTypes
			}
			if len(types) == 0 {
				return dashboardError("verification type is required", "Pass --type ownership or --all")
			}
			types = normalizeVerificationTypes(types)
			if err := validateVerificationTypes(types); err != nil {
				return err
			}

			domainSvc, err := createDomainServiceFn()
			if err != nil {
				return wrapDashboardError(err)
			}

			ctx, cancel := common.CreateContext()
			defer cancel()

			ref, err := resolveExistingDomainRef(ctx, domainSvc, args[0], region)
			if err != nil {
				return wrapDashboardError(err)
			}

			rows := make([]domainVerificationRow, 0, len(types))
			for _, typ := range types {
				result, err := domainSvc.VerifyDomain(ctx, ref.IDOrAddress, ref.Region, domain.DashboardVerifyInboxDomainInput{Type: typ})
				if err != nil {
					return wrapDashboardError(err)
				}
				rows = append(rows, toVerificationRow(typ, result))
			}
			return common.WriteListWithColumns(cmd, rows, domainVerificationColumns)
		},
	}

	cmd.Flags().StringVarP(&region, "region", "r", "", "Region (us or eu; inferred for existing domains when omitted)")
	cmd.Flags().StringSliceVar(&types, "type", nil, "Verification type to run (repeatable: ownership,mx,spf,feedback,dkim,dmarc,arc)")
	cmd.Flags().BoolVar(&all, "all", false, "Verify all supported record types")

	return cmd
}

func newDomainsUpdateCmd() *cobra.Command {
	var (
		region string
		name   string
	)

	cmd := &cobra.Command{
		Use:   "update [domain-id-or-address]",
		Short: "Update a domain",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if strings.TrimSpace(name) == "" {
				return dashboardError("domain name is required", "Pass --name")
			}

			domainSvc, err := createDomainServiceFn()
			if err != nil {
				return wrapDashboardError(err)
			}

			ctx, cancel := common.CreateContext()
			defer cancel()

			ref, err := resolveExistingDomainRef(ctx, domainSvc, args[0], region)
			if err != nil {
				return wrapDashboardError(err)
			}

			updated, err := domainSvc.UpdateDomain(ctx, ref.IDOrAddress, ref.Region, domain.DashboardUpdateInboxDomainInput{Name: name})
			if err != nil {
				return wrapDashboardError(err)
			}
			if err := validateUpdatedDomain(updated); err != nil {
				return err
			}
			return common.GetOutputWriter(cmd).Write(toDomainRow(*updated))
		},
	}

	cmd.Flags().StringVarP(&region, "region", "r", "", "Region (us or eu; inferred for existing domains when omitted)")
	cmd.Flags().StringVarP(&name, "name", "n", "", "New display name")

	return cmd
}

func newDomainsDeleteCmd() *cobra.Command {
	var (
		region string
		yes    bool
	)

	cmd := &cobra.Command{
		Use:   "delete [domain-id-or-address]",
		Short: "Delete a domain",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			domainSvc, err := createDomainServiceFn()
			if err != nil {
				return wrapDashboardError(err)
			}

			ctx, cancel := common.CreateContext()
			defer cancel()

			ref, err := resolveExistingDomainRef(ctx, domainSvc, args[0], region)
			if err != nil {
				return wrapDashboardError(err)
			}
			if !yes && !common.Confirm(fmt.Sprintf("Delete domain %s in %s?", ref.Display, ref.Region), false) {
				return nil
			}

			deleted, err := domainSvc.DeleteDomain(ctx, ref.IDOrAddress, ref.Region)
			if err != nil {
				return wrapDashboardError(err)
			}

			return writeDomainDeleteResult(cmd, deleted)
		},
	}

	cmd.Flags().StringVarP(&region, "region", "r", "", "Region (us or eu; inferred for existing domains when omitted)")
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "Skip confirmation")

	return cmd
}

type resolvedDomainRef struct {
	IDOrAddress string
	Region      string
	Display     string
}

func resolveExistingDomainRef(ctx context.Context, domainSvc domainService, domainIDOrAddress, region string) (resolvedDomainRef, error) {
	domainIDOrAddress = strings.TrimSpace(domainIDOrAddress)
	if region != "" {
		if err := validateRegion(region); err != nil {
			return resolvedDomainRef{}, err
		}
		domainIDOrAddress = normalizeDomainLookupValue(domainIDOrAddress)
		return resolvedDomainRef{
			IDOrAddress: domainIDOrAddress,
			Region:      region,
			Display:     domainIDOrAddress,
		}, nil
	}

	domains, err := listDomainsForResolution(ctx, domainSvc)
	if err != nil {
		return resolvedDomainRef{}, err
	}

	normalizedInput := normalizeDomainAddress(domainIDOrAddress)
	var matches []domain.DashboardInboxDomain
	for _, item := range domains {
		if item.ID == domainIDOrAddress || normalizeDomainAddress(item.DomainAddress) == normalizedInput {
			matches = append(matches, item)
		}
	}

	if len(matches) == 0 {
		return resolvedDomainRef{}, dashboardError(
			"domain not found",
			"Pass --region us or --region eu, or run 'nylas dashboard domains list' to see registered domains",
		)
	}
	if len(matches) > 1 {
		return resolvedDomainRef{}, dashboardError(
			"domain matches multiple regions",
			"Pass --region us or --region eu",
		)
	}

	match := matches[0]
	return resolvedDomainRef{
		IDOrAddress: match.ID,
		Region:      match.Region,
		Display:     match.DomainAddress,
	}, nil
}

func listDomainsForResolution(ctx context.Context, domainSvc domainService) ([]domain.DashboardInboxDomain, error) {
	const pageSize = common.MaxAPILimit

	var all []domain.DashboardInboxDomain
	pageToken := ""
	seenCursors := map[string]bool{"": true}
	for pageCount := 0; pageCount < maxDomainResolutionPages; pageCount++ {
		page, err := domainSvc.ListDomains(ctx, pageSize, pageToken)
		if err != nil {
			return nil, err
		}
		all = append(all, page.Domains...)
		if page.NextCursor == "" || len(page.Domains) == 0 {
			return all, nil
		}
		if seenCursors[page.NextCursor] {
			return all, nil
		}
		seenCursors[page.NextCursor] = true
		pageToken = page.NextCursor
	}
	return nil, fmt.Errorf("too many domain pages while resolving region; pass --region us or --region eu")
}

func writeDomainDeleteResult(cmd *cobra.Command, deleted bool) error {
	if !deleted {
		return dashboardError("domain was not deleted", "Check the domain ID and region")
	}
	if common.IsStructuredOutput(cmd) {
		return common.GetOutputWriter(cmd).Write(struct {
			Success bool `json:"success"`
		}{Success: true})
	}
	common.PrintSuccess("Domain deleted")
	return nil
}

func writeDomainListResult(cmd *cobra.Command, page domain.DashboardInboxDomainPage) error {
	rows := toDomainRows(page.Domains)
	format, _ := cmd.Flags().GetString("format")
	if common.IsJSON(cmd) || format == "yaml" {
		if rows == nil {
			rows = []domainRow{}
		}
		return common.GetOutputWriter(cmd).Write(domainListResult{
			Domains:    rows,
			NextCursor: page.NextCursor,
		})
	}

	if len(rows) == 0 {
		quiet, _ := cmd.Flags().GetBool("quiet")
		if quiet {
			return nil
		}
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "No domains found.")
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "\nRegister one with: nylas dashboard domains create example.com --region us")
		return nil
	}

	if err := common.WriteListWithColumns(cmd, rows, domainColumns); err != nil {
		return err
	}

	quiet, _ := cmd.Flags().GetBool("quiet")
	if page.NextCursor != "" && !quiet {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "\nNext: nylas dashboard domains list --page-token %s\n", page.NextCursor)
	}
	return nil
}

func validateCreatedDomain(created *domain.DashboardInboxDomain) error {
	if created == nil || created.ID == "" || created.DomainAddress == "" || created.Region == "" {
		return dashboardError("domain was not created", "Dashboard returned an incomplete domain response")
	}
	return nil
}

func validateUpdatedDomain(updated *domain.DashboardInboxDomain) error {
	if updated == nil || updated.ID == "" {
		return dashboardError("domain was not updated", "Dashboard returned an incomplete domain response")
	}
	return nil
}

func resolveDomainAddress(args []string, flagValue string) (string, error) {
	domainAddress := strings.TrimSpace(flagValue)
	if len(args) > 0 {
		argAddress := strings.TrimSpace(args[0])
		if domainAddress != "" && normalizeDomainAddress(domainAddress) != normalizeDomainAddress(argAddress) {
			return "", dashboardError("domain specified twice", "Use either positional [domain] or --domain")
		}
		domainAddress = argAddress
	}
	domainAddress = normalizeDomainAddress(domainAddress)
	if err := validateDomainAddress(domainAddress); err != nil {
		return "", err
	}
	return domainAddress, nil
}

func normalizeDomainAddress(address string) string {
	return strings.TrimSuffix(strings.ToLower(strings.TrimSpace(address)), ".")
}

func normalizeDomainLookupValue(value string) string {
	value = strings.TrimSpace(value)
	if strings.Contains(value, ".") {
		return normalizeDomainAddress(value)
	}
	return value
}

func validateDomainAddress(address string) error {
	if len(address) < 3 || len(address) > 253 {
		return dashboardError("invalid domain", "Domain must be between 3 and 253 characters")
	}
	address = strings.TrimSuffix(address, ".")
	labels := strings.Split(address, ".")
	if len(labels) < 2 {
		return dashboardError("invalid domain", "Domain must include at least one dot, for example example.com")
	}
	for _, label := range labels {
		if len(label) == 0 || len(label) > 63 {
			return dashboardError("invalid domain", "Each domain label must be 1-63 characters")
		}
		if !domainLabelPattern.MatchString(label) {
			return dashboardError("invalid domain", "Domain labels may contain only letters, numbers, and hyphens")
		}
		if strings.HasPrefix(label, "-") || strings.HasSuffix(label, "-") {
			return dashboardError("invalid domain", "Domain labels cannot start or end with a hyphen")
		}
	}
	return nil
}

func validateRegion(region string) error {
	if region != string(domain.DashboardInboxRegionUS) && region != string(domain.DashboardInboxRegionEU) {
		return dashboardError("invalid region", "Use --region us or --region eu")
	}
	return nil
}

func normalizeVerificationTypes(types []string) []string {
	if len(types) == 0 {
		return domainInfoTypes
	}
	out := make([]string, 0, len(types))
	seen := make(map[string]bool, len(types))
	for _, typ := range types {
		typ = strings.ToLower(strings.TrimSpace(typ))
		if typ == "" || seen[typ] {
			continue
		}
		seen[typ] = true
		out = append(out, typ)
	}
	return out
}

func validateVerificationTypes(types []string) error {
	allowed := make(map[string]bool, len(domainInfoTypes))
	for _, typ := range domainInfoTypes {
		allowed[typ] = true
	}
	for _, typ := range types {
		if !allowed[typ] {
			return dashboardError("invalid verification type", "Use one of: "+strings.Join(domainInfoTypes, ", "))
		}
	}
	return nil
}

func toDomainRows(domains []domain.DashboardInboxDomain) []domainRow {
	rows := make([]domainRow, len(domains))
	for i, item := range domains {
		rows[i] = toDomainRow(item)
	}
	return rows
}

func toDomainRow(item domain.DashboardInboxDomain) domainRow {
	return domainRow{
		ID:        item.ID,
		Domain:    item.DomainAddress,
		Name:      item.Name,
		Region:    item.Region,
		Branded:   item.Branded,
		Ownership: yesNo(item.VerifiedOwnership),
		MX:        yesNo(item.VerifiedMX),
		SPF:       yesNo(item.VerifiedSPF),
		DKIM:      yesNo(item.VerifiedDKIM),
		DMARC:     yesNo(item.VerifiedDMARC),
		ARC:       yesNo(item.VerifiedARC),
		Feedback:  yesNo(item.VerifiedFeedback),
	}
}

func toDNSRow(typ string, result *domain.DashboardDomainVerificationResult) domainDNSRow {
	row := domainDNSRow{Type: typ}
	if result == nil {
		return row
	}
	row.Status = result.Status
	if result.Attempt == nil {
		row.Value = result.Message
		return row
	}
	row.Host = result.Attempt.Options.Host
	row.Record = result.Attempt.Options.Type
	row.Value = result.Attempt.Options.Value
	return row
}

func toVerificationRow(typ string, result *domain.DashboardDomainVerificationResult) domainVerificationRow {
	row := domainVerificationRow{Type: typ}
	if result == nil {
		return row
	}
	row.Status = result.Status
	row.Message = result.Message
	if result.Attempt != nil {
		row.Host = result.Attempt.Options.Host
		row.Record = result.Attempt.Options.Type
		row.Value = result.Attempt.Options.Value
	}
	return row
}

func yesNo(ok bool) string {
	if ok {
		return "yes"
	}
	return "no"
}
