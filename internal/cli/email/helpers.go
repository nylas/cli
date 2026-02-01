package email

import (
	"fmt"
	"strings"

	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/domain"
)

// printMessage prints a message in a formatted way.
func printMessage(msg domain.Message, showBody bool) {
	// Status indicators
	status := ""
	if msg.Unread {
		status += common.Cyan.Sprint("●") + " "
	}
	if msg.Starred {
		status += common.Yellow.Sprint("★") + " "
	}

	// Print header
	fmt.Println(strings.Repeat("─", 60))
	_, _ = common.BoldWhite.Printf("Subject: %s\n", msg.Subject)
	fmt.Printf("From:    %s\n", common.FormatParticipants(msg.From))
	if len(msg.To) > 0 {
		fmt.Printf("To:      %s\n", common.FormatParticipants(msg.To))
	}
	fmt.Printf("Date:    %s (%s)\n", msg.Date.Format(common.DisplayDateTime), common.FormatTimeAgo(msg.Date))
	if status != "" {
		fmt.Printf("Status:  %s\n", status)
	}
	if len(msg.Attachments) > 0 {
		fmt.Printf("Attachments: %d files\n", len(msg.Attachments))
		for _, a := range msg.Attachments {
			_, _ = common.Dim.Printf("  - %s (%s)\n", a.Filename, common.FormatSize(a.Size))
		}
	}

	if showBody {
		fmt.Println(strings.Repeat("─", 60))
		body := msg.Body
		if body == "" {
			body = msg.Snippet
		}
		// Strip HTML tags for terminal display
		body = common.StripHTML(body)
		fmt.Println(body)
	}
	fmt.Println()
}

// printMessageRaw prints a message with raw body (no HTML processing).
func printMessageRaw(msg domain.Message) {
	// Print header
	fmt.Println(strings.Repeat("─", 60))
	_, _ = common.BoldWhite.Printf("Subject: %s\n", msg.Subject)
	fmt.Printf("From:    %s\n", common.FormatParticipants(msg.From))
	if len(msg.To) > 0 {
		fmt.Printf("To:      %s\n", common.FormatParticipants(msg.To))
	}
	fmt.Printf("Date:    %s (%s)\n", msg.Date.Format(common.DisplayDateTime), common.FormatTimeAgo(msg.Date))
	fmt.Printf("ID:      %s\n", msg.ID)
	fmt.Println(strings.Repeat("─", 60))

	// Print raw body without any processing
	body := msg.Body
	if body == "" {
		body = msg.Snippet
	}
	fmt.Println(body)
	fmt.Println()
}

// printMessageMIMEWithProvider prints the raw RFC822/MIME format with provider-aware error messages.
func printMessageMIMEWithProvider(msg domain.Message, provider domain.Provider) {
	_ = provider // Provider info available for future use if needed

	if msg.RawMIME == "" {
		fmt.Println(strings.Repeat("─", 60))
		_, _ = common.Yellow.Println("No raw MIME data available")
		fmt.Println(strings.Repeat("─", 60))
		fmt.Println("This message does not have MIME data available.")
		fmt.Println()
		_, _ = common.Cyan.Println("Tip: Use --headers to view email headers:")
		fmt.Println("  nylas email read <message-id> --headers")
		return
	}

	// Print header
	fmt.Println(strings.Repeat("─", 60))
	_, _ = common.BoldWhite.Println("RAW RFC822/MIME FORMAT")
	_, _ = common.Dim.Printf("Message ID: %s\n", msg.ID)
	fmt.Println(strings.Repeat("─", 60))
	fmt.Println()

	// Print raw MIME content
	fmt.Print(msg.RawMIME)

	// Ensure trailing newline
	if !strings.HasSuffix(msg.RawMIME, "\n") {
		fmt.Println()
	}
	fmt.Println()
}

// printMessageHeaders prints the email headers in a formatted way.
func printMessageHeaders(msg domain.Message) {
	fmt.Println(strings.Repeat("─", 60))
	_, _ = common.BoldWhite.Println("EMAIL HEADERS")
	_, _ = common.Dim.Printf("Message ID: %s\n", msg.ID)
	fmt.Println(strings.Repeat("─", 60))

	if len(msg.Headers) == 0 {
		_, _ = common.Yellow.Println("No headers available for this message.")
		return
	}

	// Print headers in name: value format
	for _, h := range msg.Headers {
		_, _ = common.Cyan.Printf("%s: ", h.Name)
		fmt.Println(h.Value)
	}
	fmt.Println()
}

// printMessageSummary prints a single-line message summary.
func printMessageSummary(msg domain.Message, index int) {
	printMessageSummaryWithID(msg, index, false)
}

// printMessageSummaryWithID prints a single-line message summary, optionally with ID.
func printMessageSummaryWithID(msg domain.Message, index int, showID bool) {
	status := " "
	if msg.Unread {
		status = common.Cyan.Sprint("●")
	}

	star := " "
	if msg.Starred {
		star = common.Yellow.Sprint("★")
	}

	from := common.FormatParticipants(msg.From)
	if len(from) > 20 {
		from = from[:17] + "..."
	}

	subject := msg.Subject
	if len(subject) > 40 {
		subject = subject[:37] + "..."
	}

	dateStr := common.FormatTimeAgo(msg.Date)
	if len(dateStr) > 12 {
		dateStr = msg.Date.Format("Jan 2")
	}

	if showID {
		// Show full ID on its own line for easy copying
		fmt.Printf("%s %s %-20s %-40s %s\n", status, star, from, subject, common.Dim.Sprint(dateStr))
		_, _ = common.Dim.Printf("      ID: %s\n", msg.ID)
	} else {
		fmt.Printf("%s %s %-20s %-40s %s\n", status, star, from, subject, common.Dim.Sprint(dateStr))
	}
}

// printSuccess prints a success message in green.
// Delegates to common.PrintSuccess for consistent success formatting.
func printSuccess(format string, args ...any) {
	common.PrintSuccess(format, args...)
}
