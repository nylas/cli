package contacts

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newSyncCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sync",
		Short: "Show contact sync information",
		Long: `Display information about how contact synchronization works in Nylas API v3.

Contact Synchronization in Nylas API v3
=======================================

Architecture Change:
  - Nylas v3 eliminated the traditional data sync model
  - No more local data storage with Nylas
  - No more waiting for initial syncs to complete
  - No more delta sync monitoring

How It Works:
  - Requests are forwarded directly to email providers
  - Responses come straight from provider APIs
  - Contact IDs are provider-native IDs (no more Nylas-specific IDs)
  - Data is always current (no stale cached data)

Provider-Specific Behavior:
  - Google/Gmail: Real-time access via Google Contacts API
  - Microsoft/Outlook: Real-time access via Microsoft Graph
  - IMAP: Contact sync depends on provider support
  - Virtual calendars: No provider sync (Nylas-managed)

Performance:
  - First request may be slower (no local cache)
  - Subsequent requests benefit from provider caching
  - No sync delays or waiting periods
  - Instant access to new contacts

Best Practices:
  - Don't store Nylas-specific sync state
  - Poll for changes if you need notifications
  - Handle provider rate limits gracefully
  - Cache data locally if you need fast repeated access

Polling for Changes:
  - For Google: Nylas polls every 5 minutes (API limitation)
  - For Microsoft: Real-time updates via webhooks
  - Use webhooks for event notifications:
    - contact.created
    - contact.updated
    - contact.deleted

Troubleshooting:
  - If contacts seem outdated, it's a provider issue
  - Check grant status with: nylas auth show <grant-id>
  - Verify provider connection is active
  - Re-authenticate if needed

Migrating from v2:
  - Remove sync status checks from your code
  - Remove delta sync implementations
  - Update contact ID handling (use provider IDs)
  - Switch to webhook-based change notifications

For more information:
  - Nylas v3 docs: https://developer.nylas.com/docs/v3/contacts/
  - Migration guide: https://developer.nylas.com/docs/v2/upgrade-to-v3/
`,
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println(cmd.Long)
		},
	}

	return cmd
}
