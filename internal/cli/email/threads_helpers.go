package email

import (
	"fmt"

	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/domain"
)

// DisplayThreadListItem formats and prints a single thread in list view
func DisplayThreadListItem(t domain.Thread, showID bool) {
	status := " "
	if t.Unread {
		status = common.Cyan.Sprint("●")
	}

	star := " "
	if t.Starred {
		star = common.Yellow.Sprint("★")
	}

	attach := " "
	if t.HasAttachments {
		attach = "📎"
	}

	// Format participants
	participants := common.Truncate(common.FormatParticipants(t.Participants), 25)

	subj := common.Truncate(t.Subject, 35)

	msgCount := fmt.Sprintf("(%d)", len(t.MessageIDs))
	dateStr := common.FormatTimeAgo(t.LatestMessageRecvDate)

	fmt.Printf("%s %s %s %-25s %-35s %-5s %s\n",
		status, star, attach, participants, subj, msgCount, common.Dim.Sprint(dateStr))

	if showID {
		_, _ = common.Dim.Printf("      ID: %s\n", t.ID)
	}
}
