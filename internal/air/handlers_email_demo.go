package air

import (
	"strings"
	"time"
)

// demoEmails returns demo email data spread across multiple folders so the
// sidebar (Inbox / Sent / Drafts / Archive / Trash) actually shows different
// content per folder. Includes one calendar-invite (.ics) email so the
// calendar-invite card UI has something to render.
func demoEmails() []EmailResponse {
	now := time.Now()
	return []EmailResponse{
		// Inbox
		{
			ID:      "demo-email-001",
			Subject: "Q4 Product Roadmap Review",
			Snippet: "Hi team, I've attached the updated roadmap for Q4...",
			Body:    "<p>Hi team,</p><p>I've attached the updated roadmap for Q4. Please review the timeline changes and let me know if you have any concerns.</p>",
			From:    []EmailParticipantResponse{{Name: "Sarah Chen", Email: "sarah.chen@company.com"}},
			To:      []EmailParticipantResponse{{Name: "Team", Email: "team@company.com"}},
			Date:    now.Add(-2 * time.Minute).Unix(),
			Unread:  true,
			Starred: true,
			Folders: []string{"inbox"},
			Attachments: []AttachmentResponse{
				{ID: "att-001", Filename: "Q4_Roadmap_v2.pdf", ContentType: "application/pdf", Size: 2516582},
			},
		},
		{
			ID:      "demo-email-002",
			Subject: "[nylas/cli] PR #142: Add focus time feature",
			Snippet: "mergify[bot] merged 1 commit into main...",
			From:    []EmailParticipantResponse{{Name: "GitHub", Email: "notifications@github.com"}},
			To:      []EmailParticipantResponse{{Name: "You", Email: "you@example.com"}},
			Date:    now.Add(-15 * time.Minute).Unix(),
			Unread:  true,
			Starred: false,
			Folders: []string{"inbox"},
		},
		{
			ID:      "demo-email-003",
			Subject: "Re: Meeting Tomorrow",
			Snippet: "That works for me. I'll send a calendar invite...",
			From:    []EmailParticipantResponse{{Name: "Alex Johnson", Email: "demo@example.com"}},
			To:      []EmailParticipantResponse{{Name: "You", Email: "you@example.com"}},
			Date:    now.Add(-1 * time.Hour).Unix(),
			Unread:  false,
			Starred: false,
			Folders: []string{"inbox"},
		},
		{
			ID:      "demo-email-004",
			Subject: "Your December invoice is ready",
			Snippet: "Your invoice for December 2024 is now available...",
			From:    []EmailParticipantResponse{{Name: "Stripe", Email: "billing@stripe.com"}},
			To:      []EmailParticipantResponse{{Name: "You", Email: "you@example.com"}},
			Date:    now.Add(-3 * time.Hour).Unix(),
			Unread:  false,
			Starred: true,
			Folders: []string{"inbox"},
		},
		{
			ID:      "demo-email-005",
			Subject: "This week in design: AI tools reshaping...",
			Snippet: "The latest trends, tools, and inspiration...",
			From:    []EmailParticipantResponse{{Name: "Design Weekly", Email: "newsletter@designweekly.com"}},
			To:      []EmailParticipantResponse{{Name: "You", Email: "you@example.com"}},
			Date:    now.Add(-5 * time.Hour).Unix(),
			Unread:  false,
			Starred: false,
			Folders: []string{"inbox"},
		},
		// Calendar invite (with .ics attachment)
		{
			ID:      "demo-email-invite-001",
			Subject: "Event Invitation: Quarterly Sync",
			Snippet: "You have received a calendar invitation: Quarterly Sync",
			Body:    "<p>You have received a calendar invitation: <strong>Quarterly Sync</strong></p><p>Please let me know if this time works.</p>",
			From:    []EmailParticipantResponse{{Name: "Priya Patel", Email: "priya@partner.example"}},
			To:      []EmailParticipantResponse{{Name: "You", Email: "you@example.com"}},
			Date:    now.Add(-30 * time.Minute).Unix(),
			Unread:  true,
			Starred: false,
			Folders: []string{"inbox"},
			Attachments: []AttachmentResponse{
				{
					ID:          "att-invite-001",
					Filename:    "invite.ics",
					ContentType: "text/calendar",
					Size:        1024,
				},
			},
		},
		// Sent — explicitly more than one so we can prove the filter works.
		{
			ID:      "demo-email-sent-001",
			Subject: "Re: Q4 Product Roadmap Review",
			Snippet: "Thanks Sarah, here are my comments on the roadmap...",
			Body:    "<p>Thanks Sarah,</p><p>Here are my comments on the roadmap. Looks good overall — happy to discuss the Q4 priorities live.</p>",
			From:    []EmailParticipantResponse{{Name: "You", Email: "you@example.com"}},
			To:      []EmailParticipantResponse{{Name: "Sarah Chen", Email: "sarah.chen@company.com"}},
			Date:    now.Add(-1 * time.Hour).Unix(),
			Folders: []string{"sent"},
		},
		{
			ID:      "demo-email-sent-002",
			Subject: "Pricing follow-up",
			Snippet: "Hi Mike, sending the updated pricing sheet...",
			Body:    "<p>Hi Mike,</p><p>Sending the updated pricing sheet as discussed. Let me know if you need any changes.</p>",
			From:    []EmailParticipantResponse{{Name: "You", Email: "you@example.com"}},
			To:      []EmailParticipantResponse{{Name: "Mike Johnson", Email: "mike@customer.example"}},
			Date:    now.Add(-3 * time.Hour).Unix(),
			Folders: []string{"sent"},
		},
		{
			ID:      "demo-email-sent-003",
			Subject: "Welcome to the team!",
			Snippet: "Excited to have you joining us next Monday...",
			Body:    "<p>Excited to have you joining us next Monday! Here's the on-boarding checklist.</p>",
			From:    []EmailParticipantResponse{{Name: "You", Email: "you@example.com"}},
			To:      []EmailParticipantResponse{{Name: "Jamie Lee", Email: "jamie@newhire.example"}},
			Date:    now.Add(-1 * 24 * time.Hour).Unix(),
			Folders: []string{"sent"},
		},
		// Drafts
		{
			ID:      "demo-email-draft-001",
			Subject: "Draft: Proposal for Acme",
			Snippet: "Hi Acme team, here's the rough proposal...",
			Body:    "<p>Hi Acme team,</p><p>Here's the rough proposal — still working through the timeline section.</p>",
			From:    []EmailParticipantResponse{{Name: "You", Email: "you@example.com"}},
			To:      []EmailParticipantResponse{{Name: "Acme Procurement", Email: "buyers@acme.example"}},
			Date:    now.Add(-4 * time.Hour).Unix(),
			Folders: []string{"drafts"},
		},
		// Archive
		{
			ID:      "demo-email-archive-001",
			Subject: "Confirmation: Subscription renewed",
			Snippet: "Your annual subscription has been renewed...",
			Body:    "<p>Your annual subscription has been renewed for another year.</p>",
			From:    []EmailParticipantResponse{{Name: "Acme Billing", Email: "billing@acme.example"}},
			To:      []EmailParticipantResponse{{Name: "You", Email: "you@example.com"}},
			Date:    now.Add(-30 * 24 * time.Hour).Unix(),
			Folders: []string{"archive"},
		},
		// Trash
		{
			ID:      "demo-email-trash-001",
			Subject: "URGENT: Winning offer (don't miss out)",
			Snippet: "You've been pre-selected for an exclusive offer...",
			Body:    "<p>You've been pre-selected.</p>",
			From:    []EmailParticipantResponse{{Name: "Promo Bot", Email: "deals@spammy.example"}},
			To:      []EmailParticipantResponse{{Name: "You", Email: "you@example.com"}},
			Date:    now.Add(-7 * 24 * time.Hour).Unix(),
			Folders: []string{"trash"},
		},
	}
}

// filterDemoEmails applies folder/unread/starred filters to a demo email
// list. Folder matching is case-insensitive against email.Folders entries
// and against well-known aliases (e.g., "SENT" → "sent", "Sent Items"
// → "sent"). Empty folder string means "no folder filter."
func filterDemoEmails(emails []EmailResponse, folder string, onlyUnread, onlyStarred bool) []EmailResponse {
	target := normalizeDemoFolder(folder)
	out := make([]EmailResponse, 0, len(emails))
	for _, e := range emails {
		if onlyUnread && !e.Unread {
			continue
		}
		if onlyStarred && !e.Starred {
			continue
		}
		if target != "" && !demoEmailIsInFolder(e, target) {
			continue
		}
		out = append(out, e)
	}
	return out
}

// normalizeDemoFolder turns a UI-supplied folder identifier (e.g., the
// system-folder ID "SENT", the Microsoft display name "Sent Items", or the
// canonical "sent") into the lowercase canonical name used in demoEmails.
//
// "all" and "all mail" route to the dedicated "all" target rather than
// collapsing into "archive": Gmail's "All Mail" view shows every
// message regardless of folder, and demoEmailIsInFolder's `target ==
// "all"` branch matches that semantic. Aliasing them to "archive"
// instead would surface only the single demo email tagged with the
// archive folder, which is wrong end-user behavior.
func normalizeDemoFolder(folder string) string {
	f := strings.ToLower(strings.TrimSpace(folder))
	switch f {
	case "":
		return ""
	case "inbox":
		return "inbox"
	case "sent", "sent items", "sent mail":
		return "sent"
	case "drafts", "draft":
		return "drafts"
	case "archive":
		return "archive"
	case "all", "all mail":
		return "all"
	case "trash", "deleted items", "deleted":
		return "trash"
	case "spam", "junk", "junk email":
		return "spam"
	case "starred":
		return "starred"
	default:
		return f
	}
}

// demoEmailIsInFolder reports whether the email is in `target` (already
// canonicalised). Special-cases "starred" since starring is a flag, not a
// folder. "all" never filters anything out.
func demoEmailIsInFolder(e EmailResponse, target string) bool {
	if target == "starred" {
		return e.Starred
	}
	if target == "all" {
		return true
	}
	for _, f := range e.Folders {
		if strings.EqualFold(f, target) {
			return true
		}
	}
	return false
}
