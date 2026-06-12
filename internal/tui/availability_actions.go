package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/domain"
)

func (v *AvailabilityView) fetchAvailability() {
	if len(v.participants) == 0 {
		v.timeline.SetText("[gray]Add participants to check availability[-]")
		v.slotsList.Clear()
		v.slots = nil
		return
	}

	v.timeline.SetText("[gray]Loading availability...[-]")
	v.slotsList.Clear()

	// Snapshot view state on the event loop; the goroutine below must not
	// read fields that the event loop may mutate concurrently.
	participants := make([]domain.AvailabilityParticipant, len(v.participants))
	for i, email := range v.participants {
		participants[i] = domain.AvailabilityParticipant{
			Email: email,
		}
	}

	req := &domain.AvailabilityRequest{
		StartTime:       v.startDate.Unix(),
		EndTime:         v.endDate.Add(24 * time.Hour).Unix(), // Include full end day
		DurationMinutes: v.duration,
		Participants:    participants,
		IntervalMinutes: 15,
	}

	// Also fetch free/busy for timeline visualization
	freeBusyReq := &domain.FreeBusyRequest{
		StartTime: v.startDate.Unix(),
		EndTime:   v.endDate.Add(24 * time.Hour).Unix(),
		Emails:    append([]string(nil), v.participants...),
	}
	grantID := v.app.config.GrantID

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		resp, err := v.app.config.Client.GetAvailability(ctx, req)
		if err != nil {
			v.app.QueueUpdateDraw(func() {
				if !v.app.grantStillCurrent(grantID) {
					return // grant switched while fetch was in flight; drop stale result
				}
				v.timeline.SetText(fmt.Sprintf("[red]Failed to load availability: %v[-]", err))
			})
			return
		}

		freeBusyResp, _ := v.app.config.Client.GetFreeBusy(ctx, grantID, freeBusyReq)

		v.app.QueueUpdateDraw(func() {
			if !v.app.grantStillCurrent(grantID) {
				return // grant switched while fetch was in flight; drop stale data
			}
			v.slots = resp.Data.TimeSlots
			if freeBusyResp != nil {
				v.freeBusy = freeBusyResp.Data
			}
			v.renderTimeline()
			v.renderSlots()
		})
	}()
}

func (v *AvailabilityView) renderTimeline() {
	styles := v.app.styles
	var content strings.Builder

	if len(v.freeBusy) == 0 {
		content.WriteString("[gray]No free/busy data available[-]\n")
		v.timeline.SetText(content.String())
		return
	}

	// Display busy times for each participant
	titleHex := styles.Hex(styles.TitleFg)
	infoHex := styles.Hex(styles.InfoColor)
	borderHex := styles.Hex(styles.BorderColor)
	for _, fb := range v.freeBusy {
		fmt.Fprintf(&content, "[%s]%s[-]\n", titleHex, fb.Email)

		if len(fb.TimeSlots) == 0 {
			content.WriteString("  [green]All free[-]\n")
		} else {
			// Group by day
			slotsByDay := make(map[string][]domain.TimeSlot)
			for _, slot := range fb.TimeSlots {
				day := time.Unix(slot.StartTime, 0).Format("Jan 2")
				slotsByDay[day] = append(slotsByDay[day], slot)
			}

			for day, slots := range slotsByDay {
				fmt.Fprintf(&content, "  [%s]%s:[-]", infoHex, day)
				for _, slot := range slots {
					start := time.Unix(slot.StartTime, 0).Local()
					end := time.Unix(slot.EndTime, 0).Local()
					fmt.Fprintf(&content, " [red]%s-%s[-]", start.Format("3:04PM"), end.Format("3:04PM"))
				}
				content.WriteString("\n")
			}
		}
		content.WriteString("\n")
	}

	// Add legend
	fmt.Fprintf(&content, "[%s]Legend: [-][red]Busy[-] [green]Free[-]\n", borderHex)

	v.timeline.SetText(content.String())
}

func (v *AvailabilityView) renderSlots() {
	v.slotsList.Clear()

	if len(v.slots) == 0 {
		v.slotsList.AddItem("[No available slots found]", "", 0, nil)
		return
	}

	// Group slots by day
	slotsByDay := make(map[string][]domain.AvailableSlot)
	for _, slot := range v.slots {
		day := time.Unix(slot.StartTime, 0).Local().Format(common.DisplayDateFormat)
		slotsByDay[day] = append(slotsByDay[day], slot)
	}

	// Sort days and display
	count := 0
	for _, slot := range v.slots {
		if count >= 20 {
			// Limit displayed slots
			v.slotsList.AddItem(fmt.Sprintf("... and %d more slots", len(v.slots)-20), "", 0, nil)
			break
		}

		start := time.Unix(slot.StartTime, 0).Local()
		end := time.Unix(slot.EndTime, 0).Local()

		mainText := fmt.Sprintf("%s %s - %s",
			start.Format("Mon, Jan 2"),
			start.Format("3:04 PM"),
			end.Format("3:04 PM"))

		secondaryText := fmt.Sprintf("%d min", v.duration)
		if len(slot.Emails) > 0 {
			secondaryText += " | " + strings.Join(slot.Emails, ", ")
		}

		idx := count
		v.slotsList.AddItem(mainText, secondaryText, 0, func() {
			if idx < len(v.slots) {
				v.createEventFromSlot(v.slots[idx])
			}
		})
		count++
	}
}

func (v *AvailabilityView) createEventFromSlot(slot domain.AvailableSlot) {
	start := time.Unix(slot.StartTime, 0).Local()
	end := time.Unix(slot.EndTime, 0).Local()

	// Create a new event with the selected time slot
	event := &domain.Event{
		When: domain.EventWhen{
			StartTime: slot.StartTime,
			EndTime:   slot.EndTime,
		},
	}

	// Add participants
	for _, email := range v.participants {
		event.Participants = append(event.Participants, domain.Participant{
			Person: domain.Person{Email: email},
			Status: "noreply",
		})
	}

	v.app.ShowConfirmDialog("Create Event",
		fmt.Sprintf("Create meeting on %s %s - %s?", start.Format("Mon, Jan 2"), start.Format("3:04 PM"), end.Format("3:04 PM")),
		func() {
			// Show event form with pre-filled data
			if v.selectedCalendarID == "" {
				v.app.Flash(FlashError, "No calendar selected")
				return
			}

			form := NewEventForm(v.app, v.selectedCalendarID, event,
				func(e *domain.Event) {
					v.app.content.Pop()
					v.app.Flash(FlashInfo, "Event created successfully")
				},
				func() {
					v.app.content.Pop()
				})
			v.app.content.Push("event-form", form)
			v.app.SetFocus(form)
		})
}

func (v *AvailabilityView) HandleKey(event *tcell.EventKey) *tcell.EventKey {
	// Global key handling
	switch event.Key() {
	case tcell.KeyRune:
		switch event.Rune() {
		case 'r':
			v.fetchAvailability()
			return nil
		}
	}
	return event
}
